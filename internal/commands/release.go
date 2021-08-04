package commands

import (
	"os"

	"deb-builder/pkg/pgp"

	"github.com/deweppro/go-app/console"
)

func GenerateRelease() console.CommandGetter {
	return console.NewCommand(func(setter console.CommandSetter) {
		setter.Setup("release", "Generate deb repository release")
		setter.Example("release --path=/data/release --public-key=./public.pgp --private-key=./private.pgp --passwd=1234 ")
		setter.Flag(func(f console.FlagsSetter) {
			f.String("path", "Path to deb repository")
			f.String("public-key", "PGP public key")
			f.String("private-key", "PGP private key")
			f.StringVar("passwd", "", "password for private key")
		})
		setter.ExecFunc(func(_ []string, path, pub, priv, passwd string) {
			pgpStore := pgp.NewPGP()

			pubKeyFile, err := os.Open(pub)
			console.FatalIfErr(err, "open PGP public key")
			defer func() {
				console.FatalIfErr(pubKeyFile.Close(), "close PGP public key")
			}()

			privKeyFile, err := os.Open(priv)
			console.FatalIfErr(err, "open PGP private key")
			defer func() {
				console.FatalIfErr(privKeyFile.Close(), "close PGP private key")
			}()

			console.FatalIfErr(pgpStore.LoadPrivateKey(privKeyFile, passwd), "read PGP private key")

		})
	})
}
