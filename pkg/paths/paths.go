/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package paths

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const ownedBuildDirAttempts = 32

// OwnedBuildDir identifies a directory created by one build invocation.
// Keeping the root together with the generated name prevents cleanup from
// falling back to an untrusted absolute path.
type OwnedBuildDir struct {
	root *os.Root
	name string
}

// OpenPathRoot creates and opens a filesystem root for build operations.
func OpenPathRoot(path string) (*os.Root, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("path root must not be empty")
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve path root: %w", err)
	}
	if filepath.Dir(abs) == abs {
		return nil, fmt.Errorf("refuse filesystem root %q", abs)
	}
	if err = os.MkdirAll(abs, 0755); err != nil {
		return nil, fmt.Errorf("create path root %q: %w", abs, err)
	}

	// Resolve a symlinked root before opening it. This keeps the explicit
	// allowlist visible and prevents a configured link to "/" from becoming
	// an unexpectedly broad deletion root.
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, fmt.Errorf("resolve path root %q: %w", abs, err)
	}
	if filepath.Dir(resolved) == resolved {
		return nil, fmt.Errorf("refuse filesystem root %q", resolved)
	}

	root, err := os.OpenRoot(resolved)
	if err != nil {
		return nil, fmt.Errorf("open path root %q: %w", resolved, err)
	}
	return root, nil
}

// NewOwnedBuildDir creates a unique temporary directory below root.
func NewOwnedBuildDir(root *os.Root, packageName, version string) (OwnedBuildDir, error) {
	if root == nil {
		return OwnedBuildDir{}, errors.New("build root must not be nil")
	}

	prefix := packageName + "_" + version + "-"
	for range ownedBuildDirAttempts {
		var suffix [8]byte
		if _, err := rand.Read(suffix[:]); err != nil {
			return OwnedBuildDir{}, fmt.Errorf("generate build directory name: %w", err)
		}
		name := prefix + hex.EncodeToString(suffix[:])
		if err := root.Mkdir(name, 0700); err == nil {
			return OwnedBuildDir{root: root, name: name}, nil
		} else if !errors.Is(err, fs.ErrExist) {
			return OwnedBuildDir{}, fmt.Errorf("create build directory %q: %w", name, err)
		}
	}

	return OwnedBuildDir{}, fmt.Errorf("could not allocate a unique build directory after %d attempts", ownedBuildDirAttempts)
}

// Path returns the filesystem path of the owned directory.
func (dir OwnedBuildDir) Path() string {
	return filepath.Join(dir.root.Name(), dir.name)
}

// Remove removes the owned directory after verifying that it is a directory.
func (dir OwnedBuildDir) Remove() error {
	if dir.root == nil || dir.name == "" {
		return errors.New("invalid owned build directory")
	}

	info, err := dir.root.Lstat(dir.name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect build directory %q: %w", dir.name, err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refuse to remove non-directory build path %q", dir.name)
	}
	if err = dir.root.RemoveAll(dir.name); err != nil {
		return fmt.Errorf("remove build directory %q: %w", dir.name, err)
	}
	return nil
}

// ValidateRootRelativePath rejects paths that escape a filesystem root.
func ValidateRootRelativePath(path string) error {
	if path == "" || strings.IndexByte(path, 0) >= 0 {
		return errors.New("path must be non-empty and must not contain NUL")
	}

	clean := filepath.Clean(path)
	if clean == "." || clean == ".." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path %q escapes root", path)
	}
	return nil
}

// DirectChildPath returns the direct-child name of targetPath under rootPath.
func DirectChildPath(rootPath, targetPath string) (string, error) {
	rootAbs, err := filepath.Abs(rootPath)
	if err != nil {
		return "", fmt.Errorf("resolve output root: %w", err)
	}
	targetAbs, err := filepath.Abs(targetPath)
	if err != nil {
		return "", fmt.Errorf("resolve output path: %w", err)
	}
	relative, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return "", fmt.Errorf("compare output path with root: %w", err)
	}
	if filepath.Dir(relative) != "." {
		return "", fmt.Errorf("output path %q is not a direct child of %q", targetPath, rootPath)
	}
	if err = ValidateRootRelativePath(relative); err != nil {
		return "", err
	}
	return relative, nil
}

// RemovePackageFile removes a regular package output below root.
func RemovePackageFile(root *os.Root, relativePath string) error {
	if root == nil {
		return errors.New("storage root must not be nil")
	}
	if err := ValidateRootRelativePath(relativePath); err != nil {
		return err
	}

	info, err := root.Lstat(relativePath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect package output %q: %w", relativePath, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("refuse to remove non-regular package output %q", relativePath)
	}
	if err = root.Remove(relativePath); err != nil {
		return fmt.Errorf("remove package output %q: %w", relativePath, err)
	}
	return nil
}

// EnsurePackageOutputAbsent verifies that a package output path is unused.
func EnsurePackageOutputAbsent(root *os.Root, relativePath string) error {
	if root == nil {
		return errors.New("storage root must not be nil")
	}
	if err := ValidateRootRelativePath(relativePath); err != nil {
		return err
	}

	info, err := root.Lstat(relativePath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect package output %q: %w", relativePath, err)
	}
	return fmt.Errorf("package output %q already exists as %s", relativePath, info.Mode())
}
