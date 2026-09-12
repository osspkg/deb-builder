/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package paths

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenPathRootRejectsFilesystemRoot(t *testing.T) {
	if _, err := OpenPathRoot(""); err == nil {
		t.Fatal("expected empty root to be rejected")
	}
	if _, err := OpenPathRoot(string(filepath.Separator)); err == nil {
		t.Fatal("expected filesystem root to be rejected")
	}
}

func TestRootOperationsStayWithinConfiguredRoot(t *testing.T) {
	rootPath := filepath.Join(t.TempDir(), "root")
	root, err := OpenPathRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	if err := root.MkdirAll("inside", 0700); err != nil {
		t.Fatal(err)
	}
	if err := root.MkdirAll("../outside", 0700); err == nil {
		t.Fatal("expected traversal outside root to be rejected")
	}
}

func TestOwnedBuildDirRemovesOnlyItsOwnDirectory(t *testing.T) {
	root, err := OpenPathRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	owned, err := NewOwnedBuildDir(root, "demo", "1:0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(owned.name, "demo_1:0.0.1-") {
		t.Fatalf("unexpected owned directory name %q", owned.name)
	}
	if err := root.WriteFile(filepath.Join(owned.name, "artifact"), []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := owned.Remove(); err != nil {
		t.Fatal(err)
	}
	if _, err := root.Lstat(owned.name); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("owned directory still exists: %v", err)
	}

	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	unsafe := OwnedBuildDir{root: root, name: "../outside"}
	if err := unsafe.Remove(); err == nil {
		t.Fatal("expected traversal cleanup to be rejected")
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("outside file was affected: %v", err)
	}
}

func TestRemovePackageFileRejectsTraversalAndSymlink(t *testing.T) {
	rootPath := t.TempDir()
	root, err := OpenPathRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := root.Symlink(outside, "link"); err != nil {
		t.Fatal(err)
	}
	if err := RemovePackageFile(root, "../outside"); err == nil {
		t.Fatal("expected package traversal to be rejected")
	}
	if err := RemovePackageFile(root, "link"); err == nil {
		t.Fatal("expected symlink package output to be rejected")
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("outside file was affected: %v", err)
	}

	if err := root.WriteFile("package.deb", []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := RemovePackageFile(root, "package.deb"); err != nil {
		t.Fatal(err)
	}
	if _, err := root.Lstat("package.deb"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("package output still exists: %v", err)
	}
}
