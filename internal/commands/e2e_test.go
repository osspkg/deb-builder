/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package commands

import (
	"context"
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/osspkg/pkg-build/pkg/pgp"
)

func TestBuildAndReleaseEndToEnd(t *testing.T) {
	dpkgDeb := requireCommand(t, "dpkg-deb")
	gpgPath := requireCommand(t, "gpg")
	aptGet := requireCommand(t, "apt-get")

	_, testFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(testFile), "../.."))
	binary := filepath.Join(t.TempDir(), "pkg-build")
	runE2ECommand(t, repoRoot, "go", "build", "-o", binary, "./cmd/pkg-build")

	projectDir := t.TempDir()
	repositoryDir := filepath.Join(t.TempDir(), "repository")
	tmpDir := filepath.Join(t.TempDir(), "build")
	releaseTmpDir := filepath.Join(t.TempDir(), "release")
	configData := []byte(`ver: "2"
packages:
  - package: demo
    source: demo
    version: 1.2.3
    architecture:
      - amd64
    maintainer: pkg-build test <pkg-build@example.com>
    homepage: https://example.com/demo
    description:
      - Demo package
      - This package is built by the end to end test.
    section: utils
    priority: optional
    data:
      usr/share/doc/demo/README: "+hello from e2e\n"
`)
	configFile := filepath.Join(projectDir, ".deb.yaml")
	require.NoError(t, os.WriteFile(configFile, configData, 0600))

	runE2ECommand(t, projectDir, binary,
		"build",
		"--config="+filepath.Base(configFile),
		"--base-dir="+filepath.Join(repositoryDir, "pool", "contrib"),
		"--tmp-dir="+tmpDir,
		"--no-revision",
	)

	debFile := filepath.Join(repositoryDir, "pool", "contrib", "d", "demo", "demo_1.2.3_amd64.deb")
	if _, err := os.Stat(debFile); err != nil {
		t.Fatalf("build did not create %s: %v", debFile, err)
	}
	infoOutput := runE2ECommand(t, projectDir, dpkgDeb, "--info", debFile)
	require.Contains(t, string(infoOutput), "Package: demo")

	extractDir := filepath.Join(t.TempDir(), "extracted")
	runE2ECommand(t, projectDir, dpkgDeb, "--extract", debFile, extractDir)
	data, err := os.ReadFile(filepath.Join(extractDir, "usr/share/doc/demo/README"))
	require.NoError(t, err)
	require.Equal(t, "hello from e2e\n", string(data))

	cert, err := pgp.NewCertSHA512(pgp.Config{Name: "pkg-build e2e", Email: "pkg-build@example.com"})
	require.NoError(t, err)
	keyDir := t.TempDir()
	privateKeyFile := filepath.Join(keyDir, "private.pgp")
	publicKeyFile := filepath.Join(keyDir, "public.pgp")
	require.NoError(t, os.WriteFile(privateKeyFile, cert.Private, 0600))
	publicKeyStore := pgp.New()
	require.NoError(t, publicKeyStore.SetKey(cert.Private, ""))
	publicKey, err := publicKeyStore.PublicKey()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(publicKeyFile, publicKey, 0644))

	runE2ECommand(t, projectDir, binary,
		"release",
		"--release-dir="+repositoryDir,
		"--temp="+releaseTmpDir,
		"--private-key="+privateKeyFile,
		"--dist=bookworm",
		"--comp=contrib",
		"--origin=pkg-build e2e",
		"--label=pkg-build e2e",
	)

	distDir := filepath.Join(repositoryDir, "dists", "bookworm")
	for _, filename := range []string{
		filepath.Join(distDir, "Release"),
		filepath.Join(distDir, "InRelease"),
		filepath.Join(distDir, "Release.gpg"),
		filepath.Join(repositoryDir, "key.gpg"),
		filepath.Join(distDir, "contrib", "binary-amd64", "Packages"),
	} {
		if _, err := os.Stat(filename); err != nil {
			t.Fatalf("release did not create %s: %v", filename, err)
		}
	}

	packagesData, err := os.ReadFile(filepath.Join(distDir, "contrib", "binary-amd64", "Packages"))
	require.NoError(t, err)
	require.Contains(t, string(packagesData), "Package: demo\n")
	require.Contains(t, string(packagesData), "Filename: pool/contrib/d/demo/demo_1.2.3_amd64.deb\n")
	releaseData, err := os.ReadFile(filepath.Join(distDir, "Release"))
	require.NoError(t, err)
	require.Contains(t, string(releaseData), "Suite: bookworm\n")
	require.Contains(t, string(releaseData), "Codename: bookworm\n")
	require.Contains(t, string(releaseData), "Components: contrib\n")

	verifyHome := filepath.Join(t.TempDir(), "gnupg")
	require.NoError(t, os.Mkdir(verifyHome, 0700))
	require.NoError(t, runGPG(gpgPath, verifyHome,
		"--no-default-keyring", "--keyring", publicKeyFile,
		"--verify", filepath.Join(distDir, "InRelease")))
	require.NoError(t, runGPG(gpgPath, verifyHome,
		"--no-default-keyring", "--keyring", publicKeyFile,
		"--verify", filepath.Join(distDir, "Release.gpg"), filepath.Join(distDir, "Release")))

	aptDir := t.TempDir()
	sourceList := filepath.Join(aptDir, "sources.list")
	require.NoError(t, os.WriteFile(sourceList,
		[]byte(fmt.Sprintf("deb [trusted=yes] file:%s bookworm contrib\n", repositoryDir)), 0600))
	listsDir := filepath.Join(aptDir, "lists")
	cacheDir := filepath.Join(aptDir, "cache")
	require.NoError(t, os.MkdirAll(listsDir, 0700))
	require.NoError(t, os.MkdirAll(cacheDir, 0700))
	runE2ECommand(t, projectDir, aptGet,
		"-o", "Dir::Etc::sourcelist="+sourceList,
		"-o", "Dir::Etc::sourceparts=-",
		"-o", "Dir::State::lists="+listsDir+"/",
		"-o", "Dir::Cache="+cacheDir,
		"-o", "Dir::Etc::trustedparts=-",
		"-o", "Acquire::Languages=none",
		"-o", "APT::Get::List-Cleanup=false",
		"update",
	)
}

func requireCommand(t *testing.T, name string) string {
	t.Helper()
	path, err := osexec.LookPath(name)
	if err != nil {
		t.Skipf("%s is unavailable: %v", name, err)
	}
	return path
}

func runE2ECommand(t *testing.T, dir, name string, args ...string) []byte {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := osexec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command %s %s failed: %v\n%s", name, strings.Join(args, " "), err, output)
	}
	return output
}
