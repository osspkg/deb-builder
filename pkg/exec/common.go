/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package exec

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"go.osspkg.com/console"
	"go.osspkg.com/ioutils/fs"

	"github.com/osspkg/deb-builder/pkg/packages"
)

type Replacer interface {
	Replace(s string) string
}

func Build(build, ver string, archs []string, cb func(arch string, repl Replacer)) {
	for _, arch := range archs {

		replacer := strings.NewReplacer(
			`%arch%`, arch,
			`%version%`, packages.SplitVersion(ver),
			`%ver%`, packages.SplitVersion(ver),
		)

		if len(build) > 0 {
			out, err := execCommand(replacer.Replace(build), true)
			console.Warnf(out)
			console.FatalIfErr(err, "Failed to build resources for %s", arch)
		}

		cb(arch, replacer)
	}
}

//nolint:unparam
func execCommand(cmd string, line bool, envs ...string) (string, error) {
	key := "-ec"
	if line {
		key = "-xec"
	}
	c := exec.Command("/bin/sh", key, fmt.Sprintln(cmd, " <&-"))
	if len(envs) > 0 {
		c.Env = append(os.Environ(), envs...)
	}
	c.Dir = fs.CurrentDir()
	b, err := c.CombinedOutput()
	return string(b), err
}

func GitVersion() (string, error) {
	out, err := execCommand("git status --porcelain", false)
	if err != nil {
		return "", err
	}
	out = strings.TrimSpace(out)
	if len(strings.TrimSpace(out)) > 0 {
		return "", fmt.Errorf("has uncommitted changes")
	}

	out, err = execCommand("git for-each-ref --count=1 --sort=\"-committerdate\" "+
		"--format=\"%(refname:lstrip=-1)-build%(committerdate:format:%Y%m%d%H%M%S)-%(objectname:short=12)\" "+
		"refs/tags --merged HEAD~0\n", false)
	if err != nil {
		return "", err
	}
	out = strings.TrimSpace(out)
	if len(out) > 0 {
		info := strings.SplitN(out, ".", 2)
		major, err0 := strconv.ParseInt(strings.TrimPrefix(info[0], "v"), 10, 64)
		if err0 != nil {
			return "", err0
		}
		var epoch int64 = 1
		if major > 0 {
			epoch = major
		}

		return fmt.Sprintf("%d:%d.%s", epoch, major, info[1]), nil
	}

	out, err = execCommand("git -c log.showsignature=false log -1 --format=%H:%ct", false)
	if err != nil {
		return "", err
	}
	info := strings.Split(strings.TrimSpace(out), ":")
	hash := info[0][0:12]
	secs, err := strconv.ParseInt(info[1], 10, 64)
	if err != nil {
		return "", err
	}
	ts := time.Unix(secs, 0).Format("20060102150405")
	return fmt.Sprintf("1:0.0.1-build%s-%s", ts, hash), nil
}
