/*
 *  Copyright (c) 2021-2023 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package exec

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/osspkg/deb-builder/pkg/config"
	"github.com/osspkg/deb-builder/pkg/packages"
	"github.com/osspkg/go-sdk/console"
)

type Replacer interface {
	Replace(s string) string
}

func Build(conf *config.Config, cb func(arch string, repl Replacer)) {
	for _, arch := range conf.Architecture {

		replacer := strings.NewReplacer(
			`%arch%`, arch,
			`%version%`, packages.SplitVersion(conf.Version),
		)

		if len(conf.Control.Build) > 0 {
			out, err := execCommand(replacer.Replace(conf.Control.Build), nil)
			console.Warnf(out)
			console.FatalIfErr(err, "Failed to build resources for %s", arch)
		}

		cb(arch, replacer)
	}
}

func execCommand(cmd string, env []string) (string, error) {
	c := exec.Command("/bin/sh", "-xec", fmt.Sprintln(cmd, " <&-"))
	if len(env) > 0 {
		c.Env = append(os.Environ(), env...)
	}
	if dir, err := os.Getwd(); err == nil {
		c.Dir = dir
	}
	b, err := c.CombinedOutput()
	return string(b), err
}
