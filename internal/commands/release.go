/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package commands

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"go.osspkg.com/archives/ar"
	"go.osspkg.com/console"

	"github.com/osspkg/pkg-build/pkg/archive"
	"github.com/osspkg/pkg-build/pkg/buffer"
	"github.com/osspkg/pkg-build/pkg/control"
	"github.com/osspkg/pkg-build/pkg/hash"
	"github.com/osspkg/pkg-build/pkg/packages"
	"github.com/osspkg/pkg-build/pkg/pgp"
	"github.com/osspkg/pkg-build/pkg/utils"
)

const (
	PathComponent    = "%s/pool/%s/"
	PathDistribution = "%s/dists/%s/"
	PathBinary       = "%s/dists/%s/%s/binary-%s/"
)

var (
	defaultReleaseArchitectures = []string{"i386", "amd64", "arm", "arm64"}
	releaseArchitectureRegexp   = regexp.MustCompile(`^[a-z0-9][a-z0-9+.-]*$`)
	releasePathComponentRegexp  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9+._-]*$`)
)

func GenerateRelease() console.CommandGetter {
	return console.NewCommand(func(setter console.CommandSetter) {
		setter.Setup("release", "Generate deb repository release")
		setter.Flag(func(f console.FlagsSetter) {
			f.StringVar("release-dir", utils.GetEnv("DEB_STORAGE_BASE_DIR", "./release"), "Path to deb repository")
			f.StringVar("temp", utils.GetEnv("DEB_BUILD_DIR", "/tmp/deb-release"), "Temp path for build release")
			f.StringVar("private-key", utils.GetEnv("DEB_PGP_KEY", "./key.pgp"), "PGP private key")
			f.StringVar("passwd", utils.GetEnv("DEB_PGP_KEY_PASSWD", ""), "password for private key if exist")
			f.StringVar("origin", utils.GetEnv("DEB_RELEASE_ORIGIN", "Packages Origin"), "release info")
			f.StringVar("label", utils.GetEnv("DEB_RELEASE_LABEL", "Packages Label"), "release info")
			f.StringVar("dist", utils.GetEnv("DEB_DISTRIBUTION", "stable"), "release distribution")
			f.StringVar("comp", utils.GetEnv("DEB_COMPONENT", "main"), "release component")
		})
		setter.ExecFunc(func(_ []string, path, tmp, privKeyFile, passwd, origin, label, dist, comp string) {
			console.FatalIfErr(validateReleasePathComponent("distribution", dist), "validate distribution")
			console.FatalIfErr(validateReleasePathComponent("component", comp), "validate component")

			/**
			LOAD PGP
			*/
			pgpStore := pgp.New()
			console.FatalIfErr(pgpStore.SetKeyFromFile(privKeyFile, passwd), "read PGP private key")

			/**
			Packages
			*/

			pkgs := make([]*packages.PackegesModel, 0, 1000)
			pathcomp := fmt.Sprintf(PathComponent, path, comp)
			err := filepath.Walk(pathcomp, func(filename string, info fs.FileInfo, err error) (retErr error) {
				if err != nil {
					return err
				}
				if info.IsDir() {
					return nil
				}
				if !info.Mode().IsRegular() || filepath.Ext(info.Name()) != ".deb" {
					return nil
				}
				shortName := strings.Replace(filename, path+"/", "", 1)
				console.Infof("deb: %s", shortName)

				arch, err := ar.Open(filename, info.Mode().Perm())
				if err != nil {
					return fmt.Errorf("open deb: %w", err)
				}
				defer func() {
					if closeErr := arch.Close(); closeErr != nil {
						retErr = errors.Join(retErr, fmt.Errorf("close deb: %w", closeErr))
					}
				}()
				if err = arch.Export("control.tar.gz", tmp); err != nil {
					return fmt.Errorf("export control.tar.gz: %w", err)
				}

				tgz, err := archive.NewReader(tmp + "/control.tar.gz")
				if err != nil {
					return fmt.Errorf("open control.tar.gz: %w", err)
				}
				defer func() {
					if closeErr := tgz.Close(); closeErr != nil {
						retErr = errors.Join(retErr, fmt.Errorf("close control.tar.gz: %w", closeErr))
					}
				}()
				controlData, err := tgz.Read(control.ControlFileName)
				if err != nil {
					return fmt.Errorf("read control: %w", err)
				}

				pkgModel := &packages.PackegesModel{}
				if err = pkgModel.Decode(controlData); err != nil {
					return fmt.Errorf("decode control: %w", err)
				}
				if pkgModel.Package == "" || pkgModel.Version == "" || pkgModel.Architecture == "" {
					return fmt.Errorf("control is missing package, version, or architecture")
				}
				pkgModel.Filename = shortName
				pkgModel.Size = info.Size()

				mh, err := hash.CalcMultiHash(filename)
				if err != nil {
					return fmt.Errorf("calc multi hash: %w", err)
				}

				pkgModel.MD5sum = mh.MD5
				pkgModel.SHA1 = mh.SHA1
				pkgModel.SHA256 = mh.SHA256

				pkgs = append(pkgs, pkgModel)
				return nil
			})
			console.FatalIfErr(err, "list packages")

			sortPackages(pkgs)
			archs, err := releaseArchitectures(pkgs)
			console.FatalIfErr(err, "detect package architectures")

			for _, arch := range archs {
				dir := fmt.Sprintf(PathBinary, path, dist, comp, arch)
				console.FatalIfErr(os.MkdirAll(dir, 0755), "validate dirs")
			}

			/**
			Release
			*/

			pkgBuffer := make(map[string]*buffer.Buffer)
			for _, v := range archs {
				pkgBuffer[v] = buffer.New(v)
			}

			for _, pkg := range pkgs {
				pkgInfo, err0 := pkg.Encode()
				console.FatalIfErr(err0, "encode package")
				pkgInfo = append(pkgInfo, []byte("\n\n")...)

				if pkg.Architecture == "all" {
					for _, arch := range archs {
						pkgBuffer[arch].Write(pkgInfo)
					}
				} else {
					if pb, ok := pkgBuffer[pkg.Architecture]; ok {
						pb.Write(pkgInfo)
					}
				}
			}

			inRelease := []string{}
			for _, arch := range archs {
				dir := fmt.Sprintf(PathBinary, path, dist, comp, arch)
				inRelease = append(inRelease, dir+"Packages", dir+"Packages.gz")

				packagesData := pkgBuffer[arch].Bytes()
				err = writeAtomicFile(dir+"Packages", packagesData)
				console.FatalIfErr(err, "write amd64 Packages")
				compressed, err := gzipData(packagesData)
				console.FatalIfErr(err, "compress %s Packages", arch)
				err = writeAtomicFile(dir+"Packages.gz", compressed)
				console.FatalIfErr(err, "write amd64 Packages.gz")
			}

			for _, arch := range archs {
				releasePkg := packages.ReleaseModel{
					Component:    comp,
					Origin:       origin,
					Label:        label,
					Architecture: arch,
					Description:  "Packages for Ubuntu and Debian",
				}
				releaseInfo, err2 := releasePkg.Encode()
				console.FatalIfErr(err2, "encode release info")

				dir := fmt.Sprintf(PathBinary, path, dist, comp, arch)
				err = writeAtomicFile(dir+"Release", releaseInfo)
				console.FatalIfErr(err, "write %s Packages", arch)
			}

			/**
			InRelease
			*/

			inReleaseModel := &packages.InReleaseModel{
				Origin:        origin,
				Label:         label,
				Suite:         dist,
				Component:     comp,
				Codename:      dist,
				Date:          time.Now().UTC().Format(time.RFC1123),
				Architectures: strings.Join(archs, " "),
				Components:    comp,
				Description:   "Packages for Ubuntu and Debian",
				MD5Sum:        "",
				SHA1:          "",
				SHA256:        "",
			}

			for _, fileName := range inRelease {
				inrHash, err1 := hash.CalcMultiHash(fileName)
				console.FatalIfErr(err1, "calc multi hash: %s", fileName)
				shortName := strings.Replace(fileName, fmt.Sprintf(PathDistribution, path, dist), "", 1)
				stats, err3 := os.Stat(fileName)
				console.FatalIfErr(err3, "file stat: %s", fileName)

				inReleaseModel.MD5Sum += fmt.Sprintf("\n %s %d %s", inrHash.MD5, stats.Size(), shortName)
				inReleaseModel.SHA1 += fmt.Sprintf("\n %s %d %s", inrHash.SHA1, stats.Size(), shortName)
				inReleaseModel.SHA256 += fmt.Sprintf("\n %s %d %s", inrHash.SHA256, stats.Size(), shortName)
			}

			inReleaseInfo, err := inReleaseModel.Encode()
			console.FatalIfErr(err, "encode Release")
			err = writeAtomicFile(fmt.Sprintf(PathDistribution, path, dist)+"Release", inReleaseInfo)
			console.FatalIfErr(err, "write Release")

			in := bytes.NewBuffer(inReleaseInfo)
			out := &bytes.Buffer{}
			console.FatalIfErr(pgpStore.Sign(in, out), "sign Release")
			err = writeAtomicFile(fmt.Sprintf(PathDistribution, path, dist)+"InRelease", out.Bytes())
			console.FatalIfErr(err, "write InRelease")

			/**
			Detached Release signature
			*/

			releaseSignature, err := signDetachedRelease(inReleaseInfo, pgpStore)
			console.FatalIfErr(err, "sign Release.gpg")
			err = writeAtomicFile(fmt.Sprintf(PathDistribution, path, dist)+"Release.gpg", releaseSignature)
			console.FatalIfErr(err, "write Release.gpg")

			pubKey, err := pgpStore.PublicKey()
			console.FatalIfErr(err, "read public key")
			err = writeAtomicFile(path+"/key.gpg", pubKey)
			console.FatalIfErr(err, "write key.gpg")

			info := fmt.Sprintf(`
=========================== %s ===========================

curl -fsSL https://[yourdomain]/key.gpg | sudo gpg --dearmor -o /etc/apt/keyrings/[yourdomain].gpg
sudo chmod a+r /etc/apt/keyrings/[yourdomain].gpg
sudo tee /etc/apt/sources.list.d/[yourdomain].list <<'EOF'
deb [signed-by=/etc/apt/keyrings/[yourdomain].gpg] https://[yourdomain]/ %s %s
EOF
sudo apt update

`, strings.Join(archs, " "), dist, comp)

			console.Infof(info)

		})
	})
}

func sortPackages(pkgs []*packages.PackegesModel) {
	sort.Slice(pkgs, func(i, j int) bool {
		if pkgs[i].Package != pkgs[j].Package {
			return pkgs[i].Package > pkgs[j].Package
		}
		if pkgs[i].Version != pkgs[j].Version {
			return pkgs[i].Version > pkgs[j].Version
		}
		if pkgs[i].Architecture != pkgs[j].Architecture {
			return pkgs[i].Architecture > pkgs[j].Architecture
		}
		return pkgs[i].Filename > pkgs[j].Filename
	})
}

func releaseArchitectures(pkgs []*packages.PackegesModel) ([]string, error) {
	available := make(map[string]struct{}, len(defaultReleaseArchitectures)+len(pkgs))
	for _, arch := range defaultReleaseArchitectures {
		available[arch] = struct{}{}
	}

	for _, pkg := range pkgs {
		arch := pkg.Architecture
		if arch == "all" {
			continue
		}
		if !releaseArchitectureRegexp.MatchString(arch) {
			return nil, fmt.Errorf("invalid package architecture %q", arch)
		}
		available[arch] = struct{}{}
	}

	result := make([]string, 0, len(available))
	for arch := range available {
		result = append(result, arch)
	}
	sort.Strings(result)
	return result, nil
}

func validateReleasePathComponent(field, value string) error {
	if !releasePathComponentRegexp.MatchString(value) {
		return fmt.Errorf("invalid %s %q", field, value)
	}
	return nil
}

func gzipData(data []byte) ([]byte, error) {
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write(data); err != nil {
		_ = writer.Close()
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return compressed.Bytes(), nil
}

func writeAtomicFile(filename string, data []byte) (retErr error) {
	dir := filepath.Dir(filename)
	file, err := os.CreateTemp(dir, ".pkg-build-")
	if err != nil {
		return err
	}
	removeTemp := true
	closed := false
	defer func() {
		if !closed {
			retErr = errors.Join(retErr, file.Close())
		}
		if removeTemp {
			retErr = errors.Join(retErr, os.Remove(file.Name()))
		}
	}()

	if err = file.Chmod(0644); err != nil {
		return err
	}
	if _, err = file.Write(data); err != nil {
		return err
	}
	if err = file.Sync(); err != nil {
		return err
	}
	if err = file.Close(); err != nil {
		closed = true
		return err
	}
	closed = true
	if err = os.Rename(file.Name(), filename); err != nil {
		return err
	}
	removeTemp = false
	return nil
}
