/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package commands

import (
	"go.osspkg.com/console"

	"github.com/osspkg/deb-builder/pkg/config"
)

func CreateConfig() console.CommandGetter {
	return console.NewCommand(func(setter console.CommandSetter) {
		setter.Setup("config", "Create config")
		setter.ExecFunc(func(_ []string) {
			console.FatalIfErr(config.Create(), "can`t create `.deb.yaml`")
		})
	})
}
