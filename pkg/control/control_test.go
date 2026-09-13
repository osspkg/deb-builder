/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package control

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/osspkg/pkg-build/pkg/config"
)

func TestSaveProducesValidDebianControl(t *testing.T) {
	if _, err := exec.LookPath("dpkg-deb"); err != nil {
		t.Skipf("dpkg-deb is unavailable: %v", err)
	}

	packageDir := t.TempDir()
	controlDir := filepath.Join(packageDir, "DEBIAN")
	require.NoError(t, os.Mkdir(controlDir, 0755))

	ctrl := NewControl(config.Config{
		Package:      "demo-package",
		Source:       "demo-package",
		Version:      "1.2.3",
		Architecture: []string{"amd64"},
		Maintainer:   "Package Maintainer <maintainer@example.com>",
		Section:      "utils",
		Priority:     "optional",
		Description: []string{
			"A package with a deliberately long summary that must be wrapped according to Debian control file rules",
			"The second paragraph must remain a paragraph and every continuation line must be indented.",
		},
	})
	ctrl.Arch("amd64")
	ctrl.DataSize(1)
	controlFile, err := ctrl.Save(controlDir, "")
	require.NoError(t, err)

	controlData, err := os.ReadFile(controlFile)
	require.NoError(t, err)
	descriptionIndex := strings.Index(string(controlData), "Description: ")
	require.NotEqual(t, -1, descriptionIndex)
	for _, line := range strings.Split(string(controlData)[descriptionIndex:], "\n")[1:] {
		if line != "" {
			require.Truef(t, strings.HasPrefix(line, " "), "description continuation is not indented: %q", line)
		}
	}

	output := filepath.Join(t.TempDir(), "demo-package.deb")
	cmd := exec.Command("dpkg-deb", "--build", "--root-owner-group", packageDir, output)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("dpkg-deb rejected control file: %v\n%s", err, output)
	}
}
