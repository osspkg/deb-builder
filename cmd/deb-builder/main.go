package main

import (
	"github.com/dewep-online/deb-builder/internal/commands"

	"github.com/deweppro/go-sdk/console"
)

func main() {
	root := console.New("deb-builder", "help deb-builder")
	root.AddCommand(commands.CreateConfig())
	root.AddCommand(commands.Build())
	root.AddCommand(commands.GenerateRelease())

	pgpCmd := console.NewCommand(func(setter console.CommandSetter) {
		setter.Setup("pgp", "work with PGP")
		setter.AddCommand(commands.CreatePGPCert())
		setter.ExecFunc(func(_ []string) {

		})
	})

	root.AddCommand(pgpCmd)
	root.Exec()
}
