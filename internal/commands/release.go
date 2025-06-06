/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package commands

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"go.osspkg.com/archives/ar"
	"go.osspkg.com/console"

	"github.com/osspkg/deb-builder/pkg/archive"
	"github.com/osspkg/deb-builder/pkg/buffer"
	"github.com/osspkg/deb-builder/pkg/hash"
	"github.com/osspkg/deb-builder/pkg/packages"
	"github.com/osspkg/deb-builder/pkg/pgp"
	"github.com/osspkg/deb-builder/pkg/utils"
)

const (
	PathComponent    = "%s/pool/%s/"
	PathDistribution = "%s/dists/%s/"
	PathBinary       = "%s/dists/%s/%s/binary-%s/"
)

var archs = []string{"i386", "amd64", "arm", "arm64"}

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
		setter.ExecFunc(func(_ []string, path, tmp, priv, passwd, origin, label, dist, comp string) {
			/**
			LOAD PGP
			*/
			pgpStore := pgp.NewPGP()
			privKeyFile, err := os.Open(priv)
			console.FatalIfErr(err, "open PGP private key")
			defer func() {
				console.FatalIfErr(privKeyFile.Close(), "close PGP private key")
			}()
			console.FatalIfErr(pgpStore.LoadPrivateKey(privKeyFile, passwd), "read PGP private key")

			/**
			Validate dirs
			*/

			for _, arch := range archs {
				dir := fmt.Sprintf(PathBinary, path, dist, comp, arch)
				console.FatalIfErr(os.MkdirAll(dir, 0755), "validate dirs")
			}

			/**
			Packages
			*/

			pkgs := make([]*packages.PackegesModel, 0, 1000)
			pathcomp := fmt.Sprintf(PathComponent, path, comp)
			err = filepath.Walk(pathcomp, func(filename string, info fs.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() && filepath.Ext(info.Name()) != "deb" {
					return nil
				}
				shortName := strings.Replace(filename, path+"/", "", 1)
				console.Infof("deb: %s", shortName)

				arch, err := ar.Open(filename, info.Mode().Perm())
				if err != nil {
					return fmt.Errorf("open deb: %w", err)
				}
				defer arch.Close() //nolint:errcheck
				if err = arch.Export("control.tar.gz", tmp); err != nil {
					return fmt.Errorf("export control.tar.gz: %w", err)
				}

				tgz, err := archive.NewReader(tmp + "/control.tar.gz")
				if err != nil {
					return fmt.Errorf("open control.tar.gz: %w", err)
				}
				defer tgz.Close() //nolint:errcheck
				control, err := tgz.Read("./control")
				if err != nil {
					return fmt.Errorf("read control: %w", err)
				}

				pkgModel := &packages.PackegesModel{}
				if err = pkgModel.Decode(control); err != nil {
					return fmt.Errorf("decode control: %w", err)
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

			sort.Slice(pkgs, func(i, j int) bool {
				return pkgs[i].Package > pkgs[j].Package &&
					pkgs[i].Version > pkgs[j].Version
			})

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

				err = os.WriteFile(dir+"Packages", pkgBuffer[arch].Bytes(), 0755)
				console.FatalIfErr(err, "write amd64 Packages")
				err = archive.GZWriteFile(dir+"Packages.gz", pkgBuffer[arch].Bytes(), 0755)
				console.FatalIfErr(err, "write amd64 Packages.gz")
			}

			for _, arch := range archs {
				releasePkg := packages.ReleaseModel{
					Component:    "main",
					Origin:       origin,
					Label:        label,
					Architecture: arch,
					Description:  "Packages for Ubuntu and Debian",
				}
				releaseInfo, err2 := releasePkg.Encode()
				console.FatalIfErr(err2, "encode release info")

				dir := fmt.Sprintf(PathBinary, path, dist, comp, arch)
				err = os.WriteFile(dir+"Release", releaseInfo, 0755)
				console.FatalIfErr(err, "write %s Packages", arch)
			}

			/**
			InRelease
			*/

			inReleaseModel := &packages.InReleaseModel{
				Origin:        origin,
				Label:         label,
				Component:     "main",
				Codename:      "stable",
				Date:          time.Now().UTC().Format(time.RFC1123),
				Architectures: "i386 amd64 arm arm64",
				Description:   "Packages for Ubuntu and Debian",
				MD5Sum:        "",
				SHA1:          "",
				SHA256:        "",
			}

			for _, inr := range inRelease {
				inrHash, err1 := hash.CalcMultiHash(inr)
				console.FatalIfErr(err1, "calc multi hash: %s", inr)
				shortName := strings.Replace(inr, fmt.Sprintf(PathDistribution, path, dist), "", 1)
				stats, err3 := os.Stat(inr)
				console.FatalIfErr(err3, "file stat: %s", inr)

				inReleaseModel.MD5Sum += fmt.Sprintf("\n %s %d %s", inrHash.MD5, stats.Size(), shortName)
				inReleaseModel.SHA1 += fmt.Sprintf("\n %s %d %s", inrHash.SHA1, stats.Size(), shortName)
				inReleaseModel.SHA256 += fmt.Sprintf("\n %s %d %s", inrHash.SHA256, stats.Size(), shortName)
			}

			inReleaseInfo, err := inReleaseModel.Encode()
			console.FatalIfErr(err, "encode Release")
			err = os.WriteFile(fmt.Sprintf(PathDistribution, path, dist)+"Release", inReleaseInfo, 0755)
			console.FatalIfErr(err, "write Release")

			in := bytes.NewBuffer(inReleaseInfo)
			out := &bytes.Buffer{}
			console.FatalIfErr(pgpStore.Sign(in, out), "sign Release")
			err = os.WriteFile(fmt.Sprintf(PathDistribution, path, dist)+"InRelease", out.Bytes(), 0755)
			console.FatalIfErr(err, "write InRelease")

			/**
			Copy Release.gpg
			*/

			pubKeyB64, err := pgpStore.GetPublicBase64()
			console.FatalIfErr(err, "read public key")
			err = os.WriteFile(fmt.Sprintf(PathDistribution, path, dist)+"Release.gpg", pubKeyB64, 0755)
			console.FatalIfErr(err, "write Release.gpg")

			pubKey, err := pgpStore.GetPublic()
			console.FatalIfErr(err, "read public key")
			err = os.WriteFile(path+"/key.gpg", pubKey, 0755)
			console.FatalIfErr(err, "write key.gpg")

			info := `
=========================== amd64 ===========================

curl -fsSL https://[yourdomain]/key.gpg | sudo gpg --dearmor -o /etc/apt/keyrings/[yourdomain].gpg
sudo chmod a+r /etc/apt/keyrings/[yourdomain].gpg
sudo tee /etc/apt/sources.list.d/[yourdomain].list <<'EOF'
deb [arch=arm64 signed-by=/etc/apt/keyrings/[yourdomain].gpg] https://[yourdomain]/ stable main
EOF
sudo apt update

`

			console.Infof(info)

		})
	})
}
