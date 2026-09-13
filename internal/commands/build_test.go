/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package commands

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunBuildReturnsConfigError(t *testing.T) {
	err := RunBuild(BuildOptions{
		Config:  "missing.deb.yaml",
		BaseDir: t.TempDir(),
		TempDir: t.TempDir(),
	})
	require.Error(t, err)
}
