/*
 *  Copyright (c) 2021-2023 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package commands

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/osspkg/deb-builder/pkg/archive"
	"github.com/osspkg/deb-builder/pkg/config"
	"github.com/osspkg/deb-builder/pkg/control"
	"github.com/osspkg/deb-builder/pkg/exec"
	"github.com/osspkg/deb-builder/pkg/packages"
	"github.com/osspkg/deb-builder/pkg/utils"
	"github.com/osspkg/go-archives/ar"
	"github.com/osspkg/go-sdk/console"
)

func Build() console.CommandGetter {
	return console.NewCommand(func(setter console.CommandSetter) {
		setter.Setup("build", "build deb package")
		setter.Example("build")
		setter.Flag(func(fs console.FlagsSetter) {
			fs.StringVar("config", config.ConfigFileName, "Config file")
			fs.StringVar("base-dir", utils.GetEnv("DEB_STORAGE_BASE_DIR", "/tmp/deb-storage"), "Deb package base storage")
			fs.StringVar("tmp-dir", utils.GetEnv("DEB_BUILD_DIR", "/tmp/deb-build"), "Deb package build dir")
		})
		setter.ExecFunc(func(_ []string, debConf, baseDir, tmpDir string) {
			conf, err := config.Detect(debConf)
			console.FatalIfErr(err, "deb config not found")

			buildDir := fmt.Sprintf("%s/%s_%s", tmpDir, conf.Package, conf.Version)
			console.FatalIfErr(os.RemoveAll(buildDir), "clearing build directory")
			console.FatalIfErr(os.MkdirAll(buildDir, 0755), "creating build directory")

			storeDir := fmt.Sprintf("%s/%s/%s", baseDir, conf.Package[0:1], conf.Package)
			console.FatalIfErr(os.MkdirAll(storeDir, 0755), "creating storage directory")

			exec.Build(conf, func(arch string, replacer exec.Replacer) {

				// check file version

				debFile, revision, carch := packages.BuildName(storeDir, conf.Package, conf.Version, arch)

				// package

				cpkg := control.NewControlPkg(conf)

				// md5sums + data.tar.gz

				md5sum := control.NewMd5Sums()
				dataFile := buildDir + "/data.tar.gz"
				tg, err := archive.NewWriter(dataFile)
				console.FatalIfErr(err, "create data.tar.gz")
				for dst, src := range conf.Data {
					src = replacer.Replace(src)
					var (
						f, h string
						err1 error
					)

					switch src[0] {
					case '+':
						f, h, err1 = tg.WriteData(dst, []byte(src)[1:])
						console.FatalIfErr(err1, "write %s to data.tar.gz", src)
						md5sum.Add(f, h)
						console.Infof("Add: %s", dst)
					case '~':
						fullpath, err0 := filepath.Abs(src[1:])
						console.FatalIfErr(err0, "get full path for %s", src[1:])

						err2 := filepath.Walk(fullpath, func(path string, info fs.FileInfo, e error) error {
							if e != nil {
								return e
							}
							if info.IsDir() {
								return nil
							}
							walkedFile := strings.ReplaceAll(path, fullpath, dst)
							ff, hh, ee := tg.WriteFile(path, walkedFile)
							console.FatalIfErr(ee, "write %s to data.tar.gz", src)
							md5sum.Add(ff, hh)
							console.Infof("Add: %s", walkedFile)
							return nil
						})
						console.FatalIfErr(err2, "write %s to data.tar.gz", src)
					default:
						f, h, err1 = tg.WriteFile(src, dst)
						console.FatalIfErr(err1, "write %s to data.tar.gz", src)
						md5sum.Add(f, h)
						console.Infof("Add: %s", dst)
					}
				}
				console.FatalIfErr(tg.Close(), "close data.tar.gz")
				md5file, err := md5sum.Save(buildDir)
				console.FatalIfErr(err, "create md5sums")
				cpkg.AddFile(md5file)

				// control

				ctrl := control.NewControl(conf)
				ctrl.DataSize(tg.Size())
				ctrl.Arch(carch)
				ctrlFile, err := ctrl.Save(buildDir, revision)
				console.FatalIfErr(err, "create control")
				cpkg.AddFile(ctrlFile)

				// other control.tar.gz files

				other := control.NewOther(conf)
				console.FatalIfErr(other.WriteTo(buildDir), "prepare other control files")
				cpkg.AddFile(other.List()...)

				// control.tar.gz

				controlFile := buildDir + "/control.tar.gz"
				tg, err = archive.NewWriter(controlFile)
				console.FatalIfErr(err, "create control.tar.gz")
				for _, file := range cpkg.List() {
					if _, _, err1 := tg.WriteFile(file, filepath.Base(file)); err1 != nil {
						console.FatalIfErr(err1, "write %s to control.tar.gz", file)
					}
				}
				console.FatalIfErr(tg.Close(), "close file control.tar.gz")

				// build deb

				deb, err := ar.Open(debFile, 0644)
				console.FatalIfErr(err, "create %s", debFile)
				console.FatalIfErr(deb.Write("debian-binary", []byte("2.0\n"), 0644), "write debian-binary to %s", debFile)
				console.FatalIfErr(deb.Import(controlFile, 0644), "write %s to %s", controlFile, debFile)
				console.FatalIfErr(deb.Import(dataFile, 0644), "write %s to %s", dataFile, debFile)
				console.FatalIfErr(deb.Close(), "close file %s", debFile)

				console.Infof("Result: %s", debFile)
			})

		})
	})
}
