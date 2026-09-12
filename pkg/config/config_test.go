/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package config

import "testing"

func TestValidatePathFields(t *testing.T) {
	valid := Config{
		Package:      "demo-package",
		Version:      "1:0.0.1",
		Architecture: []string{"386", "amd64", "arm64"},
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
