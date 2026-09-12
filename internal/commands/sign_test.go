/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package commands

import (
	"crypto"
	"os"
	osexec "os/exec"
	"path/filepath"
	"testing"

	"go.osspkg.com/encrypt/pgp"
)

func TestSignDetachedReleaseWithGPG(t *testing.T) {
	gpgPath, err := osexec.LookPath("gpg")
	if err != nil {
		t.Fatalf("gpg is required for detached release signature verification: %v", err)
	}

	cert, err := pgp.NewCert(pgp.Config{Name: "pkg-build test", Email: "pkg-build@example.com"}, crypto.SHA256, 2048)
	if err != nil {
		t.Fatal(err)
	}

	keyDir := t.TempDir()
	privateKeyFile := filepath.Join(keyDir, "private.pgp")
	publicKeyFile := filepath.Join(keyDir, "public.pgp")
	if err := os.WriteFile(privateKeyFile, cert.Private, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(publicKeyFile, cert.Public, 0644); err != nil {
		t.Fatal(err)
	}

	releaseFile := filepath.Join(t.TempDir(), "Release")
	releaseData := []byte("Origin: pkg-build\nSHA256:\n test 4 Packages\n")
	if err := os.WriteFile(releaseFile, releaseData, 0644); err != nil {
		t.Fatal(err)
	}

	signature, err := signDetachedRelease(releaseFile, privateKeyFile, "")
	if err != nil {
		t.Fatal(err)
	}
	signatureFile := filepath.Join(filepath.Dir(releaseFile), "Release.gpg")
	if err := os.WriteFile(signatureFile, signature, 0644); err != nil {
		t.Fatal(err)
	}

	verifyHome := filepath.Join(t.TempDir(), "gnupg")
	if err := os.Mkdir(verifyHome, 0700); err != nil {
		t.Fatal(err)
	}
	if err := runGPG(gpgPath, []string{"--no-options", "--batch", "--homedir", verifyHome, "--import", publicKeyFile}, nil); err != nil {
		t.Fatalf("import public key: %v", err)
	}
	if err := runGPG(gpgPath, []string{"--no-options", "--batch", "--homedir", verifyHome, "--verify", signatureFile, releaseFile}, nil); err != nil {
		t.Fatalf("verify detached signature: %v", err)
	}

	if err := os.WriteFile(releaseFile, append(releaseData, []byte("tampered\n")...), 0644); err != nil {
		t.Fatal(err)
	}
	if err := runGPG(gpgPath, []string{"--no-options", "--batch", "--homedir", verifyHome, "--verify", signatureFile, releaseFile}, nil); err == nil {
		t.Fatal("tampered Release unexpectedly passed signature verification")
	}
}
