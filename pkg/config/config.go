/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"go.osspkg.com/ioutils/fs"
	"gopkg.in/yaml.v3"

	"github.com/osspkg/pkg-build/pkg/exec"
	"github.com/osspkg/pkg-build/pkg/utils"
)

const FileName = ".deb.yaml"

type (
	Version struct {
		Version string `yaml:"ver"`
	}

	Multi struct {
		Version  string   `yaml:"ver"`
		Packages []Config `yaml:"packages"`
	}
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

var (
	versionRegexp      = regexp.MustCompile(`\d+:\d+\.\d+\.\d+`)
	packageNameRegexp  = regexp.MustCompile(`^[a-z0-9][a-z0-9+.-]{0,99}$`)
	architectureRegexp = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
)

// Validate checks values that become filesystem path components during a
// build. Debian metadata validation remains separate, but these fields must
// be safe before they are used to derive temporary and storage paths.
func Validate(cfg Config) error {
	if !packageNameRegexp.MatchString(cfg.Package) {
		return fmt.Errorf("invalid package name %q", cfg.Package)
	}
	if err := validatePathComponent("version", cfg.Version); err != nil {
		return err
	}
	if len(cfg.Architecture) == 0 {
		return errors.New("architecture must not be empty")
	}
	for _, arch := range cfg.Architecture {
		if !architectureRegexp.MatchString(arch) {
			return fmt.Errorf("invalid architecture %q", arch)
		}
	}
	return nil
}

func validatePathComponent(field, value string) error {
	if value == "" || value == "." || value == ".." {
		return fmt.Errorf("%s must be a non-empty path component", field)
	}
	if strings.ContainsAny(value, `/\\`) {
		return fmt.Errorf("%s %q must not contain a path separator", field, value)
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return fmt.Errorf("%s must not contain control characters", field)
		}
	}
	return nil
}

func Detect(name string) ([]Config, error) {
	dir := fs.CurrentDir()

	b, err := os.ReadFile(dir + "/" + name)
	if err != nil {
		return nil, err
	}

	ver := Version{}
	if err = yaml.Unmarshal(b, &ver); err != nil {
		return nil, err
	}

	var out []Config
	switch ver.Version {
	case "", "0", "1":
		cfg := Config{}
		if err = yaml.Unmarshal(b, &cfg); err != nil {
			return nil, err
		}
		out = append(out, cfg)

	case "2":
		cfg := Multi{}
		if err = yaml.Unmarshal(b, &cfg); err != nil {
			return nil, err
		}
		out = append(out, cfg.Packages...)
	}

	for i := 0; i < len(out); i++ {
		if out[i].Version == "git" {
			out[i].Version, err = exec.GitVersion()
			if err != nil {
				return nil, fmt.Errorf("fail build git version: %w", err)
			}
		} else if !versionRegexp.MatchString(out[i].Version) {
			return nil, fmt.Errorf("invalid version format, want format 0:0.0.0")
		}
		if err = Validate(out[i]); err != nil {
			return nil, fmt.Errorf("invalid package config %d: %w", i, err)
		}
	}

	return out, nil
}

func Create() error {
	dir := fs.CurrentDir()
	conf := Config{
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

	cfg := Multi{
		Version:  "2",
		Packages: []Config{conf},
	}

	b, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	if err = os.WriteFile(dir+"/"+FileName, b, 0755); err != nil {
		return err
	}
	return nil
}
