/*
 *  Copyright (c) 2021-2026 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package utils

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	rootPrefix = "./"
	unixSlash  = "/"
)

func TarFilesPath(v string) string {
	return rootPrefix + CleanPath(v)
}

func FullPath(v string) string {
	return filepath.Clean(unixSlash + v)
}

func CleanPath(v string) string {
	return strings.TrimLeft(FullPath(v), unixSlash)
}

func GetEnv(key, def string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	return v
}

func FileStat(filename string, callFunc func(fi os.FileInfo)) {
	if info, err := os.Stat(filename); err == nil {
		callFunc(info)
	}
}

func MultiPrefix(value string, prefixes ...string) (string, bool) {
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return prefix, true
		}
	}
	return "", false
}

func MustValueAfterPrefix(value, prefix string) string {
	indx := strings.Index(value, prefix)
	if indx == -1 {
		panic(fmt.Sprintf("value does not have prefix `%s`: %s", prefix, value))
	}
	return value[indx+len(prefix):]
}

func DirWalk(path string, walkFunc func(path string) error, ignore ...string) error {
	return filepath.Walk(path, func(path string, info fs.FileInfo, e error) error {
		if e != nil {
			return e
		}
		if info.IsDir() {
			return nil
		}

		for _, ign := range ignore {
			if strings.Contains(path, ign) {
				return nil
			}
		}

		return walkFunc(path)
	})
}

func RootDirWalk(path string, walkFunc func(path string, r io.Reader) error, ignore ...string) error {
	r, err := os.OpenRoot(path)
	if err != nil {
		return fmt.Errorf("open root path %s: %w", path, err)
	}
	defer r.Close() //nolint:errcheck

	return fs.WalkDir(r.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.Type().IsRegular() {
			return nil
		}

		for _, ign := range ignore {
			if strings.Contains(path, ign) {
				return nil
			}
		}

		f, err := r.Open(path)
		if err != nil {
			return fmt.Errorf("open %s: %w", path, err)
		}
		defer f.Close() //nolint:errcheck

		return walkFunc(path, f)
	})
}
