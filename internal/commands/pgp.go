package commands

import (
	"deb-builder/pkg/pgp"
	"os"

	"github.com/deweppro/go-app/console"
)

func CreatePGPCert() console.CommandGetter {
	return console.NewCommand(func(setter console.CommandSetter) {
		setter.Setup("new", "Generate PGP cert")
		setter.Example("new --name='User Name' --email=user.name@example.com --comment='information about cert' ")
		setter.Flag(func(f console.FlagsSetter) {
			f.String("name", "User Name")
			f.String("email", "User Email")
			f.StringVar("comment", "", "Information about cert")
		})
		setter.ExecFunc(func(_ []string, name, email, comment string) {
			dir, err := os.Getwd()
			console.FatalIfErr(err, "getting current folder")
			console.FatalIfErr(pgp.NewPGP().Generate(dir, name, comment, email), "generate cert")

		})
	})
}
