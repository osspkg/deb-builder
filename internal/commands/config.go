/*
 *  Copyright (c) 2021-2023 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package commands

import (
	"github.com/osspkg/deb-builder/pkg/config"
	"github.com/osspkg/go-sdk/console"
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
