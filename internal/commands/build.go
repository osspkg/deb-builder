package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dewep-online/deb-builder/pkg/archive"
	"github.com/dewep-online/deb-builder/pkg/config"
	"github.com/dewep-online/deb-builder/pkg/control"
	"github.com/dewep-online/deb-builder/pkg/exec"
	"github.com/dewep-online/deb-builder/pkg/utils"
	"github.com/deweppro/go-app/console"
)

func Build() console.CommandGetter {
	return console.NewCommand(func(setter console.CommandSetter) {
		setter.Setup("build", "build deb package")
		setter.Example("build")
		setter.Flag(func(fs console.FlagsSetter) {
			fs.StringVar("base-dir", utils.GetEnv("DEB_STORAGE_BASE_DIR", "/tmp/deb-storage"), "deb package base storage")
			fs.StringVar("tmp-dir", utils.GetEnv("DEB_BUILD_DIR", "/tmp/deb-build"), "deb package build dir")
		})
		setter.ExecFunc(func(_ []string, baseDir, tmpDir string) {
			conf, err := config.Detect()
			console.FatalIfErr(err, "deb config not found")

			buildDir := fmt.Sprintf("%s/%s_%s", tmpDir, conf.Package, conf.Version)
			console.FatalIfErr(os.RemoveAll(buildDir), "clearing build directory")
			console.FatalIfErr(os.MkdirAll(buildDir, 0755), "creating build directory")

			storeDir := fmt.Sprintf("%s/%s/%s", baseDir, conf.Package[0:1], conf.Package)
			console.FatalIfErr(os.MkdirAll(storeDir, 0755), "creating storage directory")

			exec.Build(conf, func(arch string) {

				// package

				cpkg := control.NewControlPkg(conf)

				// md5sums + data.tar.gz

				md5sum := control.NewMd5Sums()
				dataFile := buildDir + "/data.tar.gz"
				tg, err := archive.NewTarGZ(dataFile)
				console.FatalIfErr(err, "create data.tar.gz")
				for src, dst := range conf.Data {
					if f, h, err1 := tg.WriteFile(src, dst); err1 != nil {
						console.FatalIfErr(err1, "write %s to data.tar.gz", src)
					} else {
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
				ctrl.Arch(arch)
				ctrlFile, err := ctrl.Save(buildDir)
				console.FatalIfErr(err, "create control")
				cpkg.AddFile(ctrlFile)

				// other control.tar.gz files

				other := control.NewOther(conf)
				console.FatalIfErr(other.WriteTo(buildDir), "prepare other control files")
				cpkg.AddFile(other.List()...)

				// control.tar.gz

				controlFile := buildDir + "/control.tar.gz"
				tg, err = archive.NewTarGZ(controlFile)
				console.FatalIfErr(err, "create control.tar.gz")
				for _, file := range cpkg.List() {
					if _, _, err1 := tg.WriteFile(file, filepath.Base(file)); err1 != nil {
						console.FatalIfErr(err1, "write %s to control.tar.gz", file)
					}
				}
				console.FatalIfErr(tg.Close(), "close file control.tar.gz")

				// build deb

				debFile := fmt.Sprintf("%s/%s_%s_%s.deb", storeDir, conf.Package, conf.Version, arch)
				console.FatalIfErr(os.RemoveAll(debFile), "remove old deb file")
				deb, err := archive.NewDeb(debFile)
				console.FatalIfErr(err, "create %s", debFile)
				console.FatalIfErr(deb.WriteFile(controlFile), "write %s to %s", controlFile, debFile)
				console.FatalIfErr(deb.WriteFile(dataFile), "write %s to %s", dataFile, debFile)
				console.FatalIfErr(deb.Close(), "close file %s", debFile)

			})

		})
	})
}
