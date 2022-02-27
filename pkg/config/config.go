package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/dewep-online/deb-builder/pkg/utils"
	"gopkg.in/yaml.v2"
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
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	conf := &Config{}
	b, err := os.ReadFile(dir + "/" + name)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(b, conf); err != nil {
		return nil, err
	}
	if !versionRegexp.MatchString(conf.Version) {
		return nil, fmt.Errorf("invalid version format, want format 0:0.0.0")
	}
	return conf, nil
}

func Create() error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	conf := &Config{
		Package:      filepath.Base(dir) + "-app",
		Source:       filepath.Base(dir),
		Version:      "1:0.0.1",
		Architecture: []string{"386", "amd64", "arm", "arm64"},
		Maintainer:   utils.GetEnv("DEB_MAINTAINER", "User Name <user.name@example.com>"),
		Homepage:     "http://example.com/",
		Section:      `utils`,
		Priority:     `optional`,
		Description:  []string{"This is a demo utility", "It performs some actions. Don't forget to update this text."},
		Control: Control{
			Depends:     []string{"systemd | supervisor", "ca-certificates"},
			Conffiles:   []string{"/etc/" + filepath.Base(dir) + "/config.yaml"},
			Build:       "scripts/build.sh",
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
	if err := os.WriteFile(dir+"/"+ConfigFileName, b, 0755); err != nil {
		return err
	}
	return nil
}
