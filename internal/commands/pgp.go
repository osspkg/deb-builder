package commands

import (
	"os"

	"github.com/dewep-online/deb-builder/pkg/pgp"
	"github.com/deweppro/go-sdk/console"
)

func CreatePGPCert() console.CommandGetter {
	return console.NewCommand(func(setter console.CommandSetter) {
		setter.Setup("new", "Generate PGP cert")
		setter.Example("new --name='User Name' --email=user.name@example.com --comment='information about cert' --path=/data/cert ")
		setter.Flag(func(f console.FlagsSetter) {
			f.String("name", "User Name")
			f.String("email", "User Email")
			f.StringVar("comment", "", "Information about cert")
			f.StringVar("path", "", "Information about cert")
		})
		setter.ExecFunc(func(_ []string, name, email, comment, path string) {
			if len(path) == 0 {
				var err error
				path, err = os.Getwd()
				console.FatalIfErr(err, "getting current folder")
			}
			console.FatalIfErr(pgp.NewPGP().Generate(path, name, comment, email), "generate cert")

		})
	})
}
