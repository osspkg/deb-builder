package config

import (
	"deb-builder/pkg/utils"
	"os"
	"path/filepath"

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

func Detect() (*Config, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	conf := &Config{}
	b, err := os.ReadFile(dir + "/" + ConfigFileName)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(b, conf); err != nil {
		return nil, err
	}
	return conf, nil
}

func Create() error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	conf := &Config{
		Package:      filepath.Base(dir),
		Source:       filepath.Base(dir),
		Version:      "0.0.1",
		Architecture: []string{"i386", "amd64"},
		Maintainer:   utils.GetEnv("DEB_MAINTAINER", "user <user@email>"),
		Homepage:     "https://site.path/",
		Section:      `utils`,
		Priority:     `optional`,
		Description:  []string{"Title", "Info text 1", "Info text 2"},
		Control: Control{
			Depends:     []string{"systemd | supervisor", "ca-certificates"},
			Conffiles:   []string{"/etc/" + filepath.Base(dir) + "/config.yaml"},
			Build:       "scripts/build.sh %s",
			PreInstall:  "scripts/preinst.sh",
			PostInstall: "scripts/postinst.sh",
			PreRemove:   "scripts/prerm.sh",
			PostRemove:  "scripts/postrm.sh",
		},
		Data: map[string]string{
			"build/bin/" + filepath.Base(dir): "bin/" + filepath.Base(dir),
			"configs/config.yaml":             "etc/" + filepath.Base(dir) + "/config.yaml",
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
