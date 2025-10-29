/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package commands

import (
	"os"

	"go.osspkg.com/console"
	"go.osspkg.com/encrypt/pgp"
	"go.osspkg.com/ioutils/fs"
)

const (
	publicFilename  = "/public.pgp"
	privateFilename = "/private.pgp"
)

func CreatePGPCert() console.CommandGetter {
	return console.NewCommand(func(setter console.CommandSetter) {
		setter.Setup("new", "Generate PGP cert")
		setter.Flag(func(f console.FlagsSetter) {
			f.String("name", "User Name")
			f.String("email", "User Email")
			f.StringVar("comment", "", "Information about cert")
			f.StringVar("path", "", "Information about cert")
		})
		setter.ExecFunc(func(_ []string, name, email, comment, path string) {
			if len(path) == 0 {
				path = fs.CurrentDir()
			}
			crt, err := pgp.NewCertSHA512(pgp.Config{Name: name, Email: email, Comment: comment})
			console.FatalIfErr(err, "generate cert")
			console.FatalIfErr(os.WriteFile(path+privateFilename, crt.Private, 0600), "save private key")
			console.FatalIfErr(os.WriteFile(path+publicFilename, crt.Public, 0644), "save public cert")
		})
	})
}
