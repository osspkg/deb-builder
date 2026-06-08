/*
 *  Copyright (c) 2021-2026 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package exec

import (
	"bytes"
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
	return string(bytes.TrimSpace(b)), err
}

func GitVersion() (string, error) {
	out, err := execCommand(`git status --porcelain 2>/dev/null`, false)
	if err != nil {
		return "", fmt.Errorf("check uncommitted changes: %w", err)
	}
	if len(out) > 0 {
		return "", fmt.Errorf("has uncommitted changes")
	}

	lastTag, err := execCommand(`git describe --tags --abbrev=0 2>/dev/null`, false)
	if err != nil {
		return "", fmt.Errorf("get last git tag: %w", err)
	}
	if len(lastTag) == 0 {
		lastTag = "v0.0.0"
	}

	var epoch int64 = 1

	info := strings.SplitN(lastTag, ".", 2)
	major, err := strconv.ParseInt(strings.TrimPrefix(info[0], "v"), 10, 64)
	if err != nil {
		return "", fmt.Errorf("parse major version: %w", err)
	}
	if major > 1 {
		epoch = major
	}

	branch, err := execCommand(`git branch --show-current`, false)
	if err != nil {
		return "", fmt.Errorf("get branch: %w", err)
	}

	revTag := lastTag + ".."
	if lastTag == "v0.0.0" {
		revTag = ""
	}
	out, err = execCommand(fmt.Sprintf(`git rev-list --count %sHEAD`, revTag), false)
	if err != nil {
		return "", fmt.Errorf("get rev list: %w", err)
	}
	commitCount, err := strconv.ParseInt(out, 10, 64)
	if err != nil {
		return "", fmt.Errorf("parse commit count: %w", err)
	}

	lastHash, err := execCommand(`git rev-parse --short HEAD`, false)
	if err != nil {
		return "", fmt.Errorf("get last commit: %w", err)
	}

	out, err = execCommand(`git -c log.showsignature=false log -1 --format=%ct`, false)
	if err != nil {
		return "", fmt.Errorf("get last commit date: %w", err)
	}
	ts, err := strconv.ParseInt(out, 10, 64)
	if err != nil {
		return "", fmt.Errorf("parse timestamp: %w", err)
	}
	lastHashDate := time.Unix(ts, 0).Format("20060102")

	var buf strings.Builder
	fmt.Fprintf(&buf, "%d:%s", epoch, strings.TrimPrefix(lastTag, "v"))
	if commitCount > 0 {
		switch branch {
		case "master", "main":
			fmt.Fprintf(&buf, "-%d", commitCount)
		default:
			fmt.Fprintf(&buf, "~dev.%d", commitCount)
		}
		fmt.Fprintf(&buf, "-git.%s-%s", lastHashDate, lastHash)
	}

	return buf.String(), nil
}
