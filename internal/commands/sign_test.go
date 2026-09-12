/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package commands

import (
	"bytes"
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/osspkg/pkg-build/pkg/pgp"
)

func TestSignDetachedReleaseWithGPG(t *testing.T) {
	gpgPath, err := osexec.LookPath("gpg")
	if err != nil {
		t.Fatalf("gpg is required for detached release signature verification: %v", err)
	}

	cert, err := pgp.NewCertSHA512(pgp.Config{Name: "pkg-build test", Email: "pkg-build@example.com"})
	if err != nil {
		t.Fatal(err)
	}

	keyDir := t.TempDir()
	privateKeyFile := filepath.Join(keyDir, "private.pgp")
	publicKeyFile := filepath.Join(keyDir, "public.pgp")
	if err := os.WriteFile(privateKeyFile, cert.Private, 0600); err != nil {
		t.Fatal(err)
	}
	publicKeyStore := pgp.New()
	if err := publicKeyStore.SetKey(cert.Private, ""); err != nil {
		t.Fatal(err)
	}
	publicKey, err := publicKeyStore.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(publicKeyFile, publicKey, 0644); err != nil {
		t.Fatal(err)
	}
	releaseData := []byte("Origin: pkg-build\nSHA256:\n test 4 Packages\n")
	releaseFile := filepath.Join(t.TempDir(), "Release")
	if err := os.WriteFile(releaseFile, releaseData, 0644); err != nil {
		t.Fatal(err)
	}
	verifyHome := filepath.Join(t.TempDir(), "gnupg")
	if err := os.Mkdir(verifyHome, 0700); err != nil {
		t.Fatal(err)
	}

	signer := pgp.New()
	if err := signer.SetKeyFromFile(privateKeyFile, ""); err != nil {
		t.Fatal(err)
	}
	var cleartext bytes.Buffer
	if err := signer.Sign(bytes.NewReader(releaseData), &cleartext); err != nil {
		t.Fatal(err)
	}
	cleartextFile := filepath.Join(filepath.Dir(releaseFile), "InRelease")
	if err := os.WriteFile(cleartextFile, cleartext.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	if err := runGPG(gpgPath, verifyHome, "--no-default-keyring", "--keyring", publicKeyFile, "--verify", cleartextFile); err != nil {
		t.Fatalf("verify cleartext signature: %v", err)
	}

	signature, err := signDetachedRelease(releaseData, signer)
	if err != nil {
		t.Fatal(err)
	}
	signatureFile := filepath.Join(filepath.Dir(releaseFile), "Release.gpg")
	if err := os.WriteFile(signatureFile, signature, 0644); err != nil {
		t.Fatal(err)
	}

	if err := runGPG(gpgPath, verifyHome, "--no-default-keyring", "--keyring", publicKeyFile, "--verify", signatureFile, releaseFile); err != nil {
		t.Fatalf("verify detached signature: %v", err)
	}

	if err := os.WriteFile(releaseFile, append(releaseData, []byte("tampered\n")...), 0644); err != nil {
		t.Fatal(err)
	}
	if err := runGPG(gpgPath, verifyHome, "--no-default-keyring", "--keyring", publicKeyFile, "--verify", signatureFile, releaseFile); err == nil {
		t.Fatal("tampered Release unexpectedly passed signature verification")
	}
}

func runGPG(gpgPath, home string, args ...string) error {
	commandArgs := []string{"--no-options", "--batch", "--homedir", home}
	commandArgs = append(commandArgs, args...)
	cmd := osexec.Command(gpgPath, commandArgs...)
	cmd.Env = gpgTestEnvironment(home)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message != "" {
			return fmt.Errorf("%w: %s", err, message)
		}
		return err
	}
	return nil
}

func gpgTestEnvironment(home string) []string {
	base := os.Environ()
	env := make([]string, 0, len(base)+1)
	for _, value := range base {
		if strings.HasPrefix(value, "GPG_AGENT_INFO=") || strings.HasPrefix(value, "GNUPGHOME=") {
			continue
		}
		env = append(env, value)
	}
	return append(env, "GNUPGHOME="+home)
}
