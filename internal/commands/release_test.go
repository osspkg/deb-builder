/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package commands

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/osspkg/pkg-build/pkg/packages"
)

func TestSortPackagesUsesTotalOrder(t *testing.T) {
	pkgs := []*packages.PackegesModel{
		{Package: "alpha", Version: "1.0", Architecture: "amd64", Filename: "alpha-amd64.deb"},
		{Package: "zeta", Version: "1.0", Architecture: "amd64", Filename: "zeta-amd64.deb"},
		{Package: "alpha", Version: "2.0", Architecture: "amd64", Filename: "alpha-new.deb"},
		{Package: "alpha", Version: "2.0", Architecture: "i386", Filename: "alpha-old.deb"},
	}

	sortPackages(pkgs)

	require.Equal(t, []string{
		"zeta",
		"alpha/2.0/i386",
		"alpha/2.0/amd64",
		"alpha/1.0/amd64",
	}, []string{
		pkgs[0].Package,
		pkgs[1].Package + "/" + pkgs[1].Version + "/" + pkgs[1].Architecture,
		pkgs[2].Package + "/" + pkgs[2].Version + "/" + pkgs[2].Architecture,
		pkgs[3].Package + "/" + pkgs[3].Version + "/" + pkgs[3].Architecture,
	})
}
