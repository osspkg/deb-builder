package commands

import (
	"github.com/dewep-online/deb-builder/pkg/config"
	"github.com/deweppro/go-app/console"
)

func CreateConfig() console.CommandGetter {
	return console.NewCommand(func(setter console.CommandSetter) {
		setter.Setup("config", "create config")
		setter.Example("config")
		setter.ExecFunc(func(_ []string) {
			console.FatalIfErr(config.Create(), "can`t create `.deb.yaml`")
		})
	})
}
