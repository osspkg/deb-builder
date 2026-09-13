/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package commands

import (
	"errors"
	"fmt"
	iofs "io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"go.osspkg.com/archives/ar"
	"go.osspkg.com/console"
	"go.osspkg.com/ioutils/fs"

	"github.com/osspkg/pkg-build/pkg/archive"
	"github.com/osspkg/pkg-build/pkg/config"
	"github.com/osspkg/pkg-build/pkg/control"
	"github.com/osspkg/pkg-build/pkg/exec"
	"github.com/osspkg/pkg-build/pkg/packages"
	"github.com/osspkg/pkg-build/pkg/paths"
	"github.com/osspkg/pkg-build/pkg/utils"
)

// BuildOptions contains the inputs needed to build configured Debian packages.
type BuildOptions struct {
	Config     string
	BaseDir    string
	TempDir    string
	NoRevision bool
}

// Build is the CLI adapter for RunBuild.
func Build() console.CommandGetter {
	return console.NewCommand(func(setter console.CommandSetter) {
		setter.Setup("build", "Build deb package")
		setter.Flag(func(flag console.FlagsSetter) {
			flag.StringVar("config", config.FileName, "Config file")
			flag.StringVar("base-dir", utils.GetEnv("DEB_STORAGE_BASE_DIR", "./build"), "Deb package base storage")
			flag.StringVar("tmp-dir", utils.GetEnv("DEB_BUILD_DIR", "/tmp/deb-build"), "Deb package build dir")
			flag.Bool("no-revision", "Don`t build revision deb package")
		})
		setter.ExecFunc(func(_ []string, debConf, baseDir, tmpDir string, noRevision bool) {
			console.FatalIfErr(RunBuild(BuildOptions{
				Config:     debConf,
				BaseDir:    baseDir,
				TempDir:    tmpDir,
				NoRevision: noRevision,
			}), "build")
		})
	})
}

// RunBuild builds all packages described by options and returns operational
// errors to the caller. It deliberately does not terminate the process, so it
// can be used by tests and other application front ends.
func RunBuild(options BuildOptions) (retErr error) {
	configs, err := config.Detect(options.Config)
	if err != nil {
		return fmt.Errorf("deb config not found: %w", err)
	}

	storageRoot, err := paths.OpenPathRoot(options.BaseDir)
	if err != nil {
		return fmt.Errorf("open storage root: %w", err)
	}
	buildRoot, err := paths.OpenPathRoot(options.TempDir)
	if err != nil {
		return errors.Join(fmt.Errorf("open build root: %w", err), storageRoot.Close())
	}
	defer func() {
		retErr = errors.Join(retErr, buildRoot.Close(), storageRoot.Close())
	}()

	for _, conf := range configs {
		if err := buildConfig(conf, storageRoot, buildRoot, options.NoRevision); err != nil {
			return err
		}
	}
	return nil
}

func buildConfig(conf config.Config, storageRoot, buildRoot *os.Root, noRevision bool) (retErr error) {
	ownedDir, err := paths.NewOwnedBuildDir(buildRoot, conf.Package, conf.Version)
	if err != nil {
		return fmt.Errorf("creating build directory: %w", err)
	}
	defer func() {
		retErr = errors.Join(retErr, ownedDir.Remove())
	}()

	buildDir := ownedDir.Path()
	storeRel := filepath.Join(conf.Package[:1], conf.Package)
	if err := paths.ValidateRootRelativePath(storeRel); err != nil {
		return fmt.Errorf("validate storage path: %w", err)
	}
	if err := storageRoot.MkdirAll(storeRel, 0755); err != nil {
		return fmt.Errorf("creating storage directory: %w", err)
	}
	storeDir := filepath.Join(storageRoot.Name(), storeRel)

	if err := exec.BuildWithError(conf.Control.Build, conf.Version, conf.Architecture, func(arch string, replacer exec.Replacer) error {
		return buildPackage(conf, buildDir, storeDir, storeRel, storageRoot, arch, replacer, noRevision)
	}); err != nil {
		return err
	}
	return nil
}

func buildPackage(conf config.Config, buildDir, storeDir, storeRel string, storageRoot *os.Root, arch string, replacer exec.Replacer, noRevision bool) (retErr error) {
	if noRevision {
		oldDebFile, _, _ := packages.BuildName(storeDir, conf.Package, conf.Version, arch, true)
		oldPackageFile, err := paths.DirectChildPath(storeDir, oldDebFile)
		if err != nil {
			return fmt.Errorf("validate package output: %w", err)
		}
		oldPackageRel := filepath.Join(storeRel, oldPackageFile)
		if err := paths.ValidateRootRelativePath(oldPackageRel); err != nil {
			return fmt.Errorf("validate package output: %w", err)
		}
		if err := paths.RemovePackageFile(storageRoot, oldPackageRel); err != nil {
			return fmt.Errorf("remove old %s: %w", oldDebFile, err)
		}
	}

	reservation, revision, carch, err := packages.ReserveBuildName(storeDir, conf.Package, conf.Version, arch, noRevision)
	if err != nil {
		return fmt.Errorf("reserve package output: %w", err)
	}
	defer func() {
		retErr = errors.Join(retErr, reservation.Abort())
	}()

	debFile := reservation.Path()
	packageFile, err := paths.DirectChildPath(storeDir, debFile)
	if err != nil {
		return fmt.Errorf("validate package output: %w", err)
	}
	packageRel := filepath.Join(storeRel, packageFile)
	if err := paths.ValidateRootRelativePath(packageRel); err != nil {
		return fmt.Errorf("validate package output: %w", err)
	}

	dataFile, dataSize, md5sum, err := writeDataArchive(conf, buildDir, replacer)
	if err != nil {
		return err
	}

	cpkg := control.NewControlPkg()
	md5file, err := md5sum.Save(buildDir)
	if err != nil {
		return fmt.Errorf("create md5sums: %w", err)
	}
	cpkg.AddFile(md5file)

	ctrl := control.NewControl(conf)
	ctrl.DataSize(dataSize)
	ctrl.Arch(carch)
	ctrlFile, err := ctrl.Save(buildDir, revision)
	if err != nil {
		return fmt.Errorf("create control: %w", err)
	}
	cpkg.AddFile(ctrlFile)

	other := control.NewOther(conf)
	if err := other.WriteTo(buildDir); err != nil {
		return fmt.Errorf("prepare other control files: %w", err)
	}
	cpkg.AddFile(other.List()...)

	controlFile := filepath.Join(buildDir, "control.tar.gz")
	if err := writeControlArchive(controlFile, cpkg.List(), ctrlFile); err != nil {
		return err
	}

	deb, err := ar.Open(debFile, 0644)
	if err != nil {
		return fmt.Errorf("create %s: %w", debFile, err)
	}
	debClosed := false
	defer func() {
		if !debClosed {
			retErr = errors.Join(retErr, deb.Close())
		}
	}()
	if err := deb.Write("debian-binary", []byte("2.0\n"), 0644); err != nil {
		return fmt.Errorf("write debian-binary to %s: %w", debFile, err)
	}
	if err := deb.Import(controlFile, 0644); err != nil {
		return fmt.Errorf("write %s to %s: %w", controlFile, debFile, err)
	}
	if err := deb.Import(dataFile, 0644); err != nil {
		return fmt.Errorf("write %s to %s: %w", dataFile, debFile, err)
	}
	if err := deb.Close(); err != nil {
		debClosed = true
		return fmt.Errorf("close file %s: %w", debFile, err)
	}
	debClosed = true
	if err := reservation.Commit(); err != nil {
		return fmt.Errorf("commit package output %s: %w", debFile, err)
	}

	console.Infof("Result: %s", debFile)
	return nil
}

func writeDataArchive(conf config.Config, buildDir string, replacer exec.Replacer) (filename string, size int64, md5sum *control.Md5Sums, retErr error) {
	filename = filepath.Join(buildDir, "data.tar.gz")
	tg, err := archive.NewWriter(filename)
	if err != nil {
		return filename, 0, nil, fmt.Errorf("create data.tar.gz: %w", err)
	}
	closed := false
	defer func() {
		if !closed {
			retErr = errors.Join(retErr, tg.Close())
		}
	}()

	md5sum = control.NewMd5Sums()
	dataFiles := make([]string, 0, len(conf.Data))
	for dst := range conf.Data {
		dataFiles = append(dataFiles, dst)
	}
	sort.Strings(dataFiles)
	for _, dst := range dataFiles {
		src := replacer.Replace(conf.Data[dst])
		add := func(file, hash string, writeErr error) error {
			if writeErr != nil {
				return fmt.Errorf("write %s to data.tar.gz: %w", src, writeErr)
			}
			md5sum.Add(file, hash)
			console.Infof("Add: %s", dst)
			return nil
		}

		switch {
		case strings.HasPrefix(src, "+"):
			file, hash, writeErr := tg.WriteData(dst, []byte(src)[1:])
			if err := add(file, hash, writeErr); err != nil {
				return filename, 0, nil, err
			}
		case strings.HasPrefix(src, "c:"):
			file, hash, writeErr := tg.WriteData(dst, []byte(src)[2:])
			if err := add(file, hash, writeErr); err != nil {
				return filename, 0, nil, err
			}
		case strings.HasPrefix(src, "~"):
			fullpath, err := filepath.Abs(src[1:])
			if err != nil {
				return filename, 0, nil, fmt.Errorf("get full path for %s: %w", src[1:], err)
			}
			if err := writeWalkedFiles(tg, md5sum, fullpath, dst, src); err != nil {
				return filename, 0, nil, err
			}
		case strings.HasPrefix(src, "d:"):
			fullpath, err := filepath.Abs(src[2:])
			if err != nil {
				return filename, 0, nil, fmt.Errorf("get full path for %s: %w", src[2:], err)
			}
			if err := writeWalkedFiles(tg, md5sum, fullpath, dst, src); err != nil {
				return filename, 0, nil, err
			}
		case strings.HasPrefix(src, "e:"):
			rex, err := regexp.Compile(`(?Us)^` + src[2:] + `$`)
			if err != nil {
				return filename, 0, nil, fmt.Errorf("build regexp `%s`: %w", src[2:], err)
			}
			if err := writeMatchingFiles(tg, md5sum, fs.CurrentDir(), dst, src, rex); err != nil {
				return filename, 0, nil, err
			}
		default:
			file, hash, writeErr := tg.WriteFile(src, dst)
			if err := add(file, hash, writeErr); err != nil {
				return filename, 0, nil, err
			}
		}
	}

	size = tg.Size()
	if err := tg.Close(); err != nil {
		closed = true
		return filename, 0, nil, fmt.Errorf("close data.tar.gz: %w", err)
	}
	closed = true
	return filename, size, md5sum, nil
}

func writeWalkedFiles(tg *archive.TGZWriter, md5sum *control.Md5Sums, fullpath string, dst, source string) error {
	err := filepath.Walk(fullpath, func(filename string, info iofs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		walkedFile := strings.ReplaceAll(filename, fullpath, dst)
		file, hash, err := tg.WriteFile(filename, walkedFile)
		if err != nil {
			return fmt.Errorf("write %s to data.tar.gz: %w", source, err)
		}
		md5sum.Add(file, hash)
		console.Infof("Add: %s", walkedFile)
		return nil
	})
	if err != nil {
		return fmt.Errorf("write %s to data.tar.gz: %w", source, err)
	}
	return nil
}

func writeMatchingFiles(tg *archive.TGZWriter, md5sum *control.Md5Sums, fullpath, dst, source string, rex *regexp.Regexp) error {
	err := filepath.Walk(fullpath, func(filename string, info iofs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !rex.MatchString(strings.TrimPrefix(filename, fullpath)) {
			return nil
		}
		walkedFile := strings.ReplaceAll(filename, fullpath, dst)
		file, hash, err := tg.WriteFile(filename, walkedFile)
		if err != nil {
			return fmt.Errorf("write %s to data.tar.gz: %w", source, err)
		}
		md5sum.Add(file, hash)
		console.Infof("Add: %s", walkedFile)
		return nil
	})
	if err != nil {
		return fmt.Errorf("write %s to data.tar.gz: %w", source, err)
	}
	return nil
}

func writeControlArchive(filename string, files []string, controlFile string) (retErr error) {
	tg, err := archive.NewWriter(filename)
	if err != nil {
		return fmt.Errorf("create control.tar.gz: %w", err)
	}
	closed := false
	defer func() {
		if !closed {
			retErr = errors.Join(retErr, tg.Close())
		}
	}()
	for _, file := range files {
		destination := filepath.Base(file)
		if file == controlFile {
			destination = control.ControlFileName
		}
		if _, _, err := tg.WriteFile(file, destination); err != nil {
			return fmt.Errorf("write %s to control.tar.gz: %w", file, err)
		}
	}
	if err := tg.Close(); err != nil {
		closed = true
		return fmt.Errorf("close file control.tar.gz: %w", err)
	}
	closed = true
	return nil
}
