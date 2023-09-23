/*
 *  Copyright (c) 2021-2023 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package main

import (
	"github.com/osspkg/deb-builder/internal/commands"
	"github.com/osspkg/go-sdk/console"
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
