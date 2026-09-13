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

// ReleaseOptions contains the inputs needed to generate a signed repository.
type ReleaseOptions struct {
	ReleaseDir string
	TempDir    string
	PrivateKey string
	Password   string
	Origin     string
	Label      string
	Dist       string
	Component  string
}

// GenerateRelease is the CLI adapter for RunRelease.
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
			console.FatalIfErr(RunRelease(ReleaseOptions{
				ReleaseDir: path,
				TempDir:    tmp,
				PrivateKey: privKeyFile,
				Password:   passwd,
				Origin:     origin,
				Label:      label,
				Dist:       dist,
				Component:  comp,
			}), "release")
		})
	})
}

// RunRelease generates repository indexes and signatures without terminating
// the process on an error. The CLI wrapper above is responsible for reporting
// the returned error and choosing its exit behavior.
func RunRelease(options ReleaseOptions) (retErr error) {
	if err := validateReleasePathComponent("distribution", options.Dist); err != nil {
		return err
	}
	if err := validateReleasePathComponent("component", options.Component); err != nil {
		return err
	}

	pgpStore := pgp.New()
	if err := pgpStore.SetKeyFromFile(options.PrivateKey, options.Password); err != nil {
		return fmt.Errorf("read PGP private key: %w", err)
	}

	pkgs, err := discoverReleasePackages(options)
	if err != nil {
		return fmt.Errorf("list packages: %w", err)
	}

	sortPackages(pkgs)
	archs, err := releaseArchitectures(pkgs)
	if err != nil {
		return fmt.Errorf("detect package architectures: %w", err)
	}
	inRelease, err := writeReleaseIndexes(options, pkgs, archs)
	if err != nil {
		return err
	}
	if err := writeArchitectureReleaseFiles(options, archs); err != nil {
		return err
	}
	return writeSignedRelease(options, archs, inRelease, pgpStore)
}

func discoverReleasePackages(options ReleaseOptions) ([]*packages.PackagesModel, error) {
	pkgs := make([]*packages.PackagesModel, 0, 1000)
	pathcomp := fmt.Sprintf(PathComponent, options.ReleaseDir, options.Component)
	err := filepath.Walk(pathcomp, func(filename string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !info.Mode().IsRegular() || filepath.Ext(info.Name()) != ".deb" {
			return nil
		}
		shortName, err := filepath.Rel(options.ReleaseDir, filename)
		if err != nil {
			return fmt.Errorf("relative package path: %w", err)
		}
		shortName = filepath.ToSlash(shortName)
		console.Infof("deb: %s", shortName)

		pkgModel, err := readPackageModel(filename, info.Mode().Perm(), options.TempDir)
		if err != nil {
			return err
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
	if err != nil {
		return nil, err
	}
	return pkgs, nil
}

func writeReleaseIndexes(options ReleaseOptions, pkgs []*packages.PackagesModel, archs []string) ([]string, error) {
	for _, arch := range archs {
		dir := fmt.Sprintf(PathBinary, options.ReleaseDir, options.Dist, options.Component, arch)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("validate dirs: %w", err)
		}
	}

	pkgBuffer := make(map[string]*buffer.Buffer, len(archs))
	for _, arch := range archs {
		pkgBuffer[arch] = buffer.New(arch)
	}
	for _, pkg := range pkgs {
		pkgInfo, err := pkg.Encode()
		if err != nil {
			return nil, fmt.Errorf("encode package: %w", err)
		}
		pkgInfo = append(pkgInfo, []byte("\n\n")...)
		if pkg.Architecture == "all" {
			for _, arch := range archs {
				if _, err := pkgBuffer[arch].WriteError(pkgInfo); err != nil {
					return nil, fmt.Errorf("write %s package: %w", arch, err)
				}
			}
			continue
		}
		if pb, ok := pkgBuffer[pkg.Architecture]; ok {
			if _, err := pb.WriteError(pkgInfo); err != nil {
				return nil, fmt.Errorf("write %s package: %w", pkg.Architecture, err)
			}
		}
	}

	inRelease := make([]string, 0, len(archs)*2)
	for _, arch := range archs {
		dir := fmt.Sprintf(PathBinary, options.ReleaseDir, options.Dist, options.Component, arch)
		packagesData := pkgBuffer[arch].Bytes()
		packagesFile := filepath.Join(dir, "Packages")
		packagesGZFile := filepath.Join(dir, "Packages.gz")
		if err := writeAtomicFile(packagesFile, packagesData); err != nil {
			return nil, fmt.Errorf("write %s Packages: %w", arch, err)
		}
		compressed, err := gzipData(packagesData)
		if err != nil {
			return nil, fmt.Errorf("compress %s Packages: %w", arch, err)
		}
		if err := writeAtomicFile(packagesGZFile, compressed); err != nil {
			return nil, fmt.Errorf("write %s Packages.gz: %w", arch, err)
		}
		inRelease = append(inRelease, packagesFile, packagesGZFile)
	}
	return inRelease, nil
}

func writeArchitectureReleaseFiles(options ReleaseOptions, archs []string) error {
	for _, arch := range archs {
		releasePkg := packages.ReleaseModel{
			Component:    options.Component,
			Origin:       options.Origin,
			Label:        options.Label,
			Architecture: arch,
			Description:  "Packages for Ubuntu and Debian",
		}
		releaseInfo, err := releasePkg.Encode()
		if err != nil {
			return fmt.Errorf("encode release info: %w", err)
		}
		dir := fmt.Sprintf(PathBinary, options.ReleaseDir, options.Dist, options.Component, arch)
		if err := writeAtomicFile(filepath.Join(dir, "Release"), releaseInfo); err != nil {
			return fmt.Errorf("write %s Release: %w", arch, err)
		}
	}
	return nil
}

func writeSignedRelease(options ReleaseOptions, archs []string, files []string, pgpStore pgp.Signer) error {
	inReleaseModel := &packages.InReleaseModel{
		Origin:        options.Origin,
		Label:         options.Label,
		Suite:         options.Dist,
		Component:     options.Component,
		Codename:      options.Dist,
		Date:          time.Now().UTC().Format(time.RFC1123),
		Architectures: strings.Join(archs, " "),
		Components:    options.Component,
		Description:   "Packages for Ubuntu and Debian",
	}
	if err := addReleaseHashes(options, inReleaseModel, files); err != nil {
		return err
	}
	inReleaseInfo, err := inReleaseModel.Encode()
	if err != nil {
		return fmt.Errorf("encode Release: %w", err)
	}
	distDir := fmt.Sprintf(PathDistribution, options.ReleaseDir, options.Dist)
	if err := writeAtomicFile(filepath.Join(distDir, "Release"), inReleaseInfo); err != nil {
		return fmt.Errorf("write Release: %w", err)
	}

	in := bytes.NewBuffer(inReleaseInfo)
	out := &bytes.Buffer{}
	if err := pgpStore.Sign(in, out); err != nil {
		return fmt.Errorf("sign Release: %w", err)
	}
	if err := writeAtomicFile(filepath.Join(distDir, "InRelease"), out.Bytes()); err != nil {
		return fmt.Errorf("write InRelease: %w", err)
	}
	releaseSignature, err := signDetachedRelease(inReleaseInfo, pgpStore)
	if err != nil {
		return fmt.Errorf("sign Release.gpg: %w", err)
	}
	if err := writeAtomicFile(filepath.Join(distDir, "Release.gpg"), releaseSignature); err != nil {
		return fmt.Errorf("write Release.gpg: %w", err)
	}
	pubKey, err := pgpStore.PublicKey()
	if err != nil {
		return fmt.Errorf("read public key: %w", err)
	}
	if err := writeAtomicFile(filepath.Join(options.ReleaseDir, "key.gpg"), pubKey); err != nil {
		return fmt.Errorf("write key.gpg: %w", err)
	}

	info := fmt.Sprintf(`
=========================== %s ===========================

curl -fsSL https://[yourdomain]/key.gpg | sudo gpg --dearmor -o /etc/apt/keyrings/[yourdomain].gpg
sudo chmod a+r /etc/apt/keyrings/[yourdomain].gpg
sudo tee /etc/apt/sources.list.d/[yourdomain].list <<'EOF'
deb [signed-by=/etc/apt/keyrings/[yourdomain].gpg] https://[yourdomain]/ %s %s
EOF
sudo apt update

`, strings.Join(archs, " "), options.Dist, options.Component)
	console.Infof(info)
	return nil
}

func addReleaseHashes(options ReleaseOptions, model *packages.InReleaseModel, files []string) error {
	distDir := fmt.Sprintf(PathDistribution, options.ReleaseDir, options.Dist)
	for _, filename := range files {
		inrHash, err := hash.CalcMultiHash(filename)
		if err != nil {
			return fmt.Errorf("calc multi hash: %s: %w", filename, err)
		}
		shortName, err := filepath.Rel(distDir, filename)
		if err != nil {
			return fmt.Errorf("relative release path: %w", err)
		}
		stats, err := os.Stat(filename)
		if err != nil {
			return fmt.Errorf("file stat: %s: %w", filename, err)
		}
		shortName = filepath.ToSlash(shortName)
		model.MD5Sum += fmt.Sprintf("\n %s %d %s", inrHash.MD5, stats.Size(), shortName)
		model.SHA1 += fmt.Sprintf("\n %s %d %s", inrHash.SHA1, stats.Size(), shortName)
		model.SHA256 += fmt.Sprintf("\n %s %d %s", inrHash.SHA256, stats.Size(), shortName)
	}
	return nil
}

func readPackageModel(filename string, permission os.FileMode, tempDir string) (ret *packages.PackagesModel, retErr error) {
	arch, err := ar.Open(filename, permission)
	if err != nil {
		return nil, fmt.Errorf("open deb: %w", err)
	}
	if err := arch.Export("control.tar.gz", tempDir); err != nil {
		return nil, errors.Join(fmt.Errorf("export control.tar.gz: %w", err), arch.Close())
	}
	if err := arch.Close(); err != nil {
		return nil, fmt.Errorf("close deb: %w", err)
	}

	tgz, err := archive.NewReader(filepath.Join(tempDir, "control.tar.gz"))
	if err != nil {
		return nil, fmt.Errorf("open control.tar.gz: %w", err)
	}
	controlData, err := tgz.Read(control.ControlFileName)
	closeErr := tgz.Close()
	if err != nil {
		return nil, errors.Join(fmt.Errorf("read control: %w", err), closeErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close control.tar.gz: %w", closeErr)
	}

	pkgModel := &packages.PackagesModel{}
	if err := pkgModel.Decode(controlData); err != nil {
		return nil, fmt.Errorf("decode control: %w", err)
	}
	if pkgModel.Package == "" || pkgModel.Version == "" || pkgModel.Architecture == "" {
		return nil, errors.New("control is missing package, version, or architecture")
	}
	return pkgModel, nil
}

func sortPackages(pkgs []*packages.PackagesModel) {
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

func releaseArchitectures(pkgs []*packages.PackagesModel) ([]string, error) {
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

	if err := file.Chmod(0644); err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		closed = true
		return err
	}
	closed = true
	if err := os.Rename(file.Name(), filename); err != nil {
		return err
	}
	removeTemp = false
	return nil
}
