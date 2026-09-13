/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidatePathFields(t *testing.T) {
	valid := Config{
		Package:      "demo-package",
		Version:      "1:0.0.1",
		Architecture: []string{"386", "amd64", "arm64"},
		Maintainer:   "Package Maintainer <maintainer@example.com>",
		Description:  []string{"A package description"},
	}
	if err := Validate(valid); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}

	tests := []struct {
		name string
		cfg  Config
	}{
		{name: "empty package", cfg: Config{Version: valid.Version, Architecture: valid.Architecture}},
		{name: "package traversal", cfg: Config{Package: "../demo", Version: valid.Version, Architecture: valid.Architecture}},
		{name: "version traversal", cfg: Config{Package: valid.Package, Version: "1:0.0.1/../../outside", Architecture: valid.Architecture}},
		{name: "empty architecture", cfg: Config{Package: valid.Package, Version: valid.Version}},
		{name: "architecture traversal", cfg: Config{Package: valid.Package, Version: valid.Version, Architecture: []string{"amd64/../../outside"}}},
		{name: "control character", cfg: Config{Package: valid.Package, Version: "1:0.0.1\n", Architecture: valid.Architecture}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Validate(tt.cfg); err == nil {
				t.Fatal("expected invalid path field to be rejected")
			}
		})
	}
}

func TestDetectRejectsUnknownSchemaAndFields(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{
			name: "unknown schema",
			data: "ver: 3\npackage: demo\nversion: 1.2.3\narchitecture: [amd64]\nmaintainer: Test\ndescription: [Demo]\n",
		},
		{
			name: "unknown legacy field",
			data: "package: demo\nversion: 1.2.3\narchitecture: [amd64]\nmaintainer: Test\ndescription: [Demo]\nunknown: value\n",
		},
		{
			name: "unknown nested field",
			data: "package: demo\nversion: 1.2.3\narchitecture: [amd64]\nmaintainer: Test\ndescription: [Demo]\ncontrol:\n  unknown: value\n",
		},
		{
			name: "empty package list",
			data: "ver: 2\npackages: []\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withConfigFile(t, tt.data)
			if _, err := Detect(FileName); err == nil {
				t.Fatal("expected invalid configuration to be rejected")
			}
		})
	}
}

func TestDetectRejectsMissingRequiredFields(t *testing.T) {
	withConfigFile(t, "ver: 1\npackage: demo\nversion: 1.2.3\narchitecture: [amd64]\n")
	if _, err := Detect(FileName); err == nil {
		t.Fatal("expected missing required fields to be rejected")
	}
}

func TestDebianVersionFormat(t *testing.T) {
	valid := []string{
		"1.2.3",
		"1:2.0-1",
		"2.0~rc1",
		"1.2+dfsg-1",
		"1.2-ubuntu1",
	}
	for _, version := range valid {
		if !versionRegexp.MatchString(version) {
			t.Errorf("valid Debian version rejected: %q", version)
		}
	}

	invalid := []string{
		"x1.2.3",
		"1.2.3 trailing",
		"1.2-",
		"1.2!3",
		"1.2.3\n",
	}
	for _, version := range invalid {
		if versionRegexp.MatchString(version) {
			t.Errorf("invalid Debian version accepted: %q", version)
		}
	}
}

func withConfigFile(t *testing.T, data string) {
	t.Helper()
	dir := t.TempDir()
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(original) })
	if err = os.WriteFile(filepath.Join(dir, FileName), []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
}
