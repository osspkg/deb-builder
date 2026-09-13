/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package packages

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"go.osspkg.com/ioutils/fs"

	"github.com/osspkg/pkg-build/pkg/utils"
)

var pkgArchAlias = map[string]string{
	"386": "i386",
}

// BuildNameReservation owns an atomically allocated package output path.
// The path is kept on disk until Commit or Abort is called so another build
// process cannot select the same revision.
type BuildNameReservation struct {
	path      string
	name      string
	root      *os.Root
	file      *os.File
	committed bool
	aborted   bool
}

func (v *BuildNameReservation) Path() string {
	if v == nil {
		return ""
	}
	return v.path
}

func (v *BuildNameReservation) Commit() error {
	if v == nil || v.committed || v.aborted {
		return errors.New("invalid package name reservation")
	}

	var err error
	if v.file != nil {
		err = v.file.Close()
		v.file = nil
	}
	if v.root != nil {
		err = errors.Join(err, v.root.Close())
		v.root = nil
	}
	v.committed = true
	return err
}

func (v *BuildNameReservation) Abort() error {
	if v == nil || v.committed || v.aborted {
		return nil
	}

	var closeErr error
	if v.file != nil {
		closeErr = v.file.Close()
		v.file = nil
	}
	var removeErr, rootErr error
	if v.root != nil {
		removeErr = v.root.Remove(v.name)
		rootErr = v.root.Close()
		v.root = nil
	} else {
		removeErr = os.Remove(v.path)
	}
	v.aborted = true
	return errors.Join(closeErr, removeErr, rootErr)
}

func SplitVersion(v string) string {
	if strings.Contains(v, ":") {
		vv := strings.SplitN(v, ":", 2)
		if len(vv) == 2 {
			return vv[1]
		}
	}
	return v
}

func BuildName(dir, name, version, arch string, noRevision bool) (string, string, string) {
	if v, ok := pkgArchAlias[arch]; ok {
		arch = v
	}

	version = strings.ReplaceAll(version, ":", ".")
	subver := ""
	callFunc := func() string {
		return fmt.Sprintf("%s/%s_%s%s_%s.deb", dir, name, version, subver, arch)
	}
	path := callFunc()

	if noRevision {
		return path, subver, arch
	}

	revision := 1
	for {
		utils.FileStat(path, func(fi os.FileInfo) {
			subver = fmt.Sprintf("-%d", revision)
			path = callFunc()
			revision++
		})

		if !fs.FileExist(path) {
			return path, subver, arch
		}
	}
}

// ReserveBuildName atomically allocates the first unused package filename.
// The empty file created by the reservation can be passed directly to ar.Open.
func ReserveBuildName(dir, name, version, arch string, noRevision bool) (*BuildNameReservation, string, string, error) {
	if v, ok := pkgArchAlias[arch]; ok {
		arch = v
	}

	version = strings.ReplaceAll(version, ":", ".")
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, "", arch, fmt.Errorf("open package output root: %w", err)
	}
	for revision := 0; ; revision++ {
		subver := ""
		if revision > 0 {
			subver = fmt.Sprintf("-%d", revision)
		}
		filename := fmt.Sprintf("%s_%s%s_%s.deb", name, version, subver, arch)
		path := fmt.Sprintf("%s/%s", dir, filename)
		file, err := root.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0644)
		if err == nil {
			return &BuildNameReservation{path: path, name: filename, root: root, file: file}, subver, arch, nil
		}
		if !errors.Is(err, os.ErrExist) {
			_ = root.Close()
			return nil, "", arch, fmt.Errorf("reserve package output %q: %w", path, err)
		}
		if noRevision {
			_ = root.Close()
			return nil, "", arch, fmt.Errorf("package output %q already exists", path)
		}
	}
}
