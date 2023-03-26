package exec

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/dewep-online/deb-builder/pkg/config"
	"github.com/dewep-online/deb-builder/pkg/packages"
	"github.com/deweppro/go-sdk/console"
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
			out, err := Run(replacer.Replace(conf.Control.Build), nil)
			console.Warnf(out)
			console.FatalIfErr(err, "Failed to build resources for %s", arch)
		}

		cb(arch, replacer)
	}
}

func Run(cmd string, env []string) (string, error) {
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
