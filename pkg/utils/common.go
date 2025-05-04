/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package utils

import (
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
