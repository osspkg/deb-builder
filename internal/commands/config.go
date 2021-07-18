package commands

import (
	"deb-builder/pkg/config"

	"github.com/deweppro/go-app/console"
)

func CreateConfig() console.CommandGetter {
	return console.NewCommand(func(setter console.CommandSetter) {
		setter.Setup("new-conf", "create config")
		setter.Example("new-conf")
		setter.ExecFunc(func(_ []string) {
			console.FatalIfErr(config.Create(), "can`t create `.deb.yaml`")
		})
	})
}
