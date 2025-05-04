/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"go.osspkg.com/ioutils/fs"
	"gopkg.in/yaml.v3"

	"github.com/osspkg/deb-builder/pkg/exec"
	"github.com/osspkg/deb-builder/pkg/utils"
)

const ConfigFileName = ".deb.yaml"

type (
	Config struct {
		Package      string            `yaml:"package"`
		Source       string            `yaml:"source"`
		Version      string            `yaml:"version"`
		Architecture []string          `yaml:"architecture"`
		Maintainer   string            `yaml:"maintainer"`
		Homepage     string            `yaml:"homepage"`
		Description  []string          `yaml:"description"`
		Section      string            `yaml:"section" default:"Universe"`
		Priority     string            `yaml:"priority"`
		Control      Control           `yaml:"control"`
		Data         map[string]string `yaml:"data"`
	}
	Control struct {
		Depends     []string `yaml:"depends"`
		Build       string   `yaml:"build"`
		Conffiles   []string `yaml:"conffiles"`
		PreInstall  string   `yaml:"preinst"`
		PostInstall string   `yaml:"postinst"`
		PreRemove   string   `yaml:"prerm"`
		PostRemove  string   `yaml:"postrm"`
	}
)

var versionRegexp = regexp.MustCompile(`\d+:\d+\.\d+\.\d+`)

func Detect(name string) (*Config, error) {
	dir := fs.CurrentDir()
	conf := &Config{}
	b, err := os.ReadFile(dir + "/" + name)
	if err != nil {
		return nil, err
	}
	if err = yaml.Unmarshal(b, conf); err != nil {
		return nil, err
	}
	if conf.Version == "git" {
		conf.Version, err = exec.GitVersion()
		if err != nil {
			return nil, fmt.Errorf("fail build git version: %w", err)
		}
	} else if !versionRegexp.MatchString(conf.Version) {
		return nil, fmt.Errorf("invalid version format, want format 0:0.0.0")
	}
	return conf, nil
}

func Create() error {
	dir := fs.CurrentDir()
	conf := &Config{
		Package:      filepath.Base(dir),
		Source:       filepath.Base(dir),
		Version:      "1:0.0.1 # or use `git` for build version by git commit",
		Architecture: []string{"386", "amd64", "arm", "arm64"},
		Maintainer:   utils.GetEnv("DEB_MAINTAINER", "User Name <user.name@example.com>"),
		Homepage:     "http://example.com/",
		Section:      `utils`,
		Priority:     `optional`,
		Description:  []string{"This is a demo utility", "It performs some actions. Don't forget to update this text."},
		Control: Control{
			Depends:     []string{"systemd | supervisor", "ca-certificates"},
			Conffiles:   []string{"/etc/" + filepath.Base(dir) + "/config.yaml"},
			Build:       "scripts/build.sh --arch=%arch% --ver=%ver%",
			PreInstall:  "scripts/preinst.sh",
			PostInstall: "scripts/postinst.sh",
			PreRemove:   "scripts/prerm.sh",
			PostRemove:  "scripts/postrm.sh",
		},
		Data: map[string]string{
			"bin/" + filepath.Base(dir):                  "build/bin/" + filepath.Base(dir) + "_%arch%",
			"etc/" + filepath.Base(dir) + "/config.yaml": "configs/config.yaml",
			"var/log/" + filepath.Base(dir) + ".log":     "+Write contents of file here after '+'",
		},
	}
	b, err := yaml.Marshal(conf)
	if err != nil {
		return err
	}
	if err = os.WriteFile(dir+"/"+ConfigFileName, b, 0755); err != nil {
		return err
	}
	return nil
}
