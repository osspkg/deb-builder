/*
 *  Copyright (c) 2021-2023 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package utils

import (
	"io"
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
	v := os.Getenv(key)
	if len(v) == 0 {
		return def
	}
	return v
}

func FileExist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func FileStat(filename string, callFunc func(fi os.FileInfo)) {
	if info, err := os.Stat(filename); err == nil {
		callFunc(info)
	}
}

func CopyFile(dst, src string) error {
	source, err := os.OpenFile(src, os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	defer source.Close() //nolint: errcheck

	dist, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer dist.Close() //nolint: errcheck

	_, err = io.Copy(dist, source)
	return err
}
