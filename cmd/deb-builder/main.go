/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package main

import (
	"go.osspkg.com/console"

	"github.com/osspkg/deb-builder/internal/commands"
)

func main() {
	root := console.New("deb-builder", "help deb-builder")

	root.AddCommand(commands.CreateConfig())
	root.AddCommand(commands.Build())
	root.AddCommand(commands.GenerateRelease())

	pgpCmd := console.NewCommand(func(setter console.CommandSetter) {
		setter.Setup("pgp", "Work with PGP")
		setter.AddCommand(commands.CreatePGPCert())
		setter.ExecFunc(func(_ []string) {

		})
	})

	root.AddCommand(pgpCmd)
	root.Exec()
}
