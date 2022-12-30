package exec

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/dewep-online/deb-builder/pkg/config"
	"github.com/deweppro/go-app/console"
)

func Build(conf *config.Config, cb func(arch string)) {
	for _, v := range conf.Architecture {

		if len(conf.Control.Build) > 0 {
			out, err := Run(conf.Control.Build+" "+v, nil)
			console.Warnf(out)
			console.FatalIfErr(err, "Failed to build resources for %s", v)
		}

		cb(v)
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
