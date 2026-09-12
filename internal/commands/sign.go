/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package commands

import (
	"fmt"
	"io"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"

	"github.com/osspkg/pkg-build/pkg/paths"
)

func signDetachedRelease(releaseFile, privateKeyFile, passwd string) (signature []byte, resultErr error) {
	gpgPath, err := osexec.LookPath("gpg")
	if err != nil {
		return nil, fmt.Errorf("find gpg: %w", err)
	}

	tempRoot, err := paths.OpenPathRoot(os.TempDir())
	if err != nil {
		return nil, fmt.Errorf("open gpg temp root: %w", err)
	}
	home, err := paths.NewOwnedBuildDir(tempRoot, "pkg-build-gpg", "home")
	if err != nil {
		_ = tempRoot.Close()
		return nil, fmt.Errorf("create gpg home: %w", err)
	}
	defer func() {
		if err := home.Remove(); err != nil && resultErr == nil {
			resultErr = fmt.Errorf("remove gpg home: %w", err)
		}
		if err := tempRoot.Close(); err != nil && resultErr == nil {
			resultErr = fmt.Errorf("close gpg temp root: %w", err)
		}
	}()

	homePath := home.Path()
	commonArgs := []string{"--no-options", "--batch", "--homedir", homePath}
	importArgs := append(commonArgs, "--pinentry-mode", "loopback", "--passphrase-fd", "0", "--import", privateKeyFile)
	if err := runGPG(gpgPath, importArgs, strings.NewReader(passwd+"\n")); err != nil {
		return nil, fmt.Errorf("import private key: %w", err)
	}

	signatureFile := filepath.Join(homePath, "Release.gpg")
	signArgs := append(commonArgs,
		"--yes",
		"--pinentry-mode", "loopback",
		"--passphrase-fd", "0",
		"--detach-sign",
		"--output", signatureFile,
		releaseFile,
	)
	if err := runGPG(gpgPath, signArgs, strings.NewReader(passwd+"\n")); err != nil {
		return nil, fmt.Errorf("create detached signature: %w", err)
	}

	signatureRoot, err := os.OpenRoot(homePath)
	if err != nil {
		return nil, fmt.Errorf("open detached signature root: %w", err)
	}
	signature, readErr := signatureRoot.ReadFile("Release.gpg")
	closeErr := signatureRoot.Close()
	if readErr != nil {
		return nil, fmt.Errorf("read detached signature: %w", readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close detached signature root: %w", closeErr)
	}
	if len(signature) == 0 {
		return nil, fmt.Errorf("detached signature is empty")
	}
	return signature, nil
}

func runGPG(gpgPath string, args []string, stdin io.Reader) error {
	cmd := osexec.Command(gpgPath, args...)
	cmd.Stdin = stdin
	cmd.Env = gpgEnvironment(args)
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

func gpgEnvironment(args []string) []string {
	base := os.Environ()
	env := make([]string, 0, len(base)+1)
	for _, value := range base {
		if strings.HasPrefix(value, "GPG_AGENT_INFO=") || strings.HasPrefix(value, "GNUPGHOME=") {
			continue
		}
		env = append(env, value)
	}
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "--homedir" {
			env = append(env, "GNUPGHOME="+args[i+1])
			break
		}
	}
	return env
}
