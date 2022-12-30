package commands

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/dewep-online/deb-builder/pkg/archive"
	"github.com/dewep-online/deb-builder/pkg/config"
	"github.com/dewep-online/deb-builder/pkg/control"
	"github.com/dewep-online/deb-builder/pkg/exec"
	"github.com/dewep-online/deb-builder/pkg/packages"
	"github.com/dewep-online/deb-builder/pkg/utils"
	"github.com/deweppro/go-app/console"
	"github.com/deweppro/go-archives/ar"
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

			exec.Build(conf, func(arch string) {

				replacer := strings.NewReplacer(
					`%arch%`, arch,
					`%version%`, packages.SplitVersion(conf.Version),
				)

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
					case '~':
						err1 := filepath.Walk(src[1:], func(path string, info fs.FileInfo, e error) error {
							if e != nil {
								return e
							}
							if info.IsDir() {
								return nil
							}
							ff, hh, ee := tg.WriteFile(path, strings.ReplaceAll(path, src[1:], dst))
							console.FatalIfErr(ee, "write %s to data.tar.gz", src)
							md5sum.Add(ff, hh)
							return nil
						})
						console.FatalIfErr(err1, "write %s to data.tar.gz", src)
					default:
						f, h, err1 = tg.WriteFile(src, dst)
						console.FatalIfErr(err1, "write %s to data.tar.gz", src)
						md5sum.Add(f, h)
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
