/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package commands

import (
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/osspkg/pkg-build/pkg/hash"
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

func TestReleaseArchitecturesIncludesDiscoveredArchitectures(t *testing.T) {
	archs, err := releaseArchitectures([]*packages.PackegesModel{
		{Architecture: "riscv64"},
		{Architecture: "all"},
	})
	require.NoError(t, err)
	require.Contains(t, archs, "riscv64")
	require.Contains(t, archs, "amd64")
}

func TestReleaseArchitecturesRejectsUnsafeArchitecture(t *testing.T) {
	_, err := releaseArchitectures([]*packages.PackegesModel{{Architecture: "../../outside"}})
	require.Error(t, err)
}

func TestWriteAtomicFileReplacesWithRequestedMode(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "Release")
	require.NoError(t, os.WriteFile(filename, []byte("old"), 0755))

	require.NoError(t, writeAtomicFile(filename, []byte("new")))
	data, err := os.ReadFile(filename)
	require.NoError(t, err)
	require.Equal(t, []byte("new"), data)
	info, err := os.Stat(filename)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0644), info.Mode().Perm())
}

func TestInReleaseMetadataContainsDistributionAndComponent(t *testing.T) {
	model := packages.InReleaseModel{
		Suite:         "bookworm",
		Codename:      "bookworm",
		Components:    "contrib",
		Architectures: "amd64 riscv64",
	}
	data, err := model.Encode()
	require.NoError(t, err)
	require.Contains(t, string(data), "Suite: bookworm\n")
	require.Contains(t, string(data), "Codename: bookworm\n")
	require.Contains(t, string(data), "Components: contrib\n")
	require.Contains(t, string(data), "Architectures: amd64 riscv64\n")
	require.True(t, strings.HasSuffix(string(data), "\n"))
}

func TestReleaseMetadataAcceptedByAPT(t *testing.T) {
	aptGet, err := osexec.LookPath("apt-get")
	if err != nil {
		t.Skipf("apt-get is unavailable: %v", err)
	}

	repo := t.TempDir()
	binaryDir := filepath.Join(repo, "dists", "bookworm", "contrib", "binary-amd64")
	require.NoError(t, os.MkdirAll(binaryDir, 0755))
	packagesFile := filepath.Join(binaryDir, "Packages")
	packagesGZFile := filepath.Join(binaryDir, "Packages.gz")
	require.NoError(t, os.WriteFile(packagesFile, nil, 0644))
	compressed, err := gzipData(nil)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(packagesGZFile, compressed, 0644))

	packagesHash, err := hash.CalcMultiHash(packagesFile)
	require.NoError(t, err)
	packagesGZHash, err := hash.CalcMultiHash(packagesGZFile)
	require.NoError(t, err)
	release := packages.InReleaseModel{
		Origin:        "pkg-build test",
		Label:         "pkg-build test",
		Suite:         "bookworm",
		Codename:      "bookworm",
		Architectures: "amd64",
		Components:    "contrib",
		MD5Sum: fmt.Sprintf(
			"\n %s %d contrib/binary-amd64/Packages\n %s %d contrib/binary-amd64/Packages.gz",
			packagesHash.MD5, fileSize(t, packagesFile), packagesGZHash.MD5, fileSize(t, packagesGZFile),
		),
		SHA1: fmt.Sprintf(
			"\n %s %d contrib/binary-amd64/Packages\n %s %d contrib/binary-amd64/Packages.gz",
			packagesHash.SHA1, fileSize(t, packagesFile), packagesGZHash.SHA1, fileSize(t, packagesGZFile),
		),
		SHA256: fmt.Sprintf(
			"\n %s %d contrib/binary-amd64/Packages\n %s %d contrib/binary-amd64/Packages.gz",
			packagesHash.SHA256, fileSize(t, packagesFile), packagesGZHash.SHA256, fileSize(t, packagesGZFile),
		),
	}
	releaseData, err := release.Encode()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(repo, "dists", "bookworm", "Release"), releaseData, 0644))

	aptDir := t.TempDir()
	sourceList := filepath.Join(aptDir, "sources.list")
	require.NoError(t, os.WriteFile(sourceList, []byte(fmt.Sprintf("deb [trusted=yes] file:%s bookworm contrib\n", repo)), 0600))
	listsDir := filepath.Join(aptDir, "lists")
	cacheDir := filepath.Join(aptDir, "cache")
	require.NoError(t, os.MkdirAll(listsDir, 0700))
	require.NoError(t, os.MkdirAll(cacheDir, 0700))

	cmd := osexec.Command(aptGet,
		"-o", "Dir::Etc::sourcelist="+sourceList,
		"-o", "Dir::Etc::sourceparts=-",
		"-o", "Dir::State::lists="+listsDir+"/",
		"-o", "Dir::Cache="+cacheDir,
		"-o", "Dir::Etc::trustedparts=-",
		"-o", "Acquire::Languages=none",
		"-o", "APT::Get::List-Cleanup=false",
		"update",
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("apt-get rejected Release metadata: %v\n%s", err, output)
	}
}

func fileSize(t *testing.T, filename string) int64 {
	t.Helper()
	info, err := os.Stat(filename)
	require.NoError(t, err)
	return info.Size()
}
