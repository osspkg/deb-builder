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

func CopyFile(src, dst string) error {
	source, err := os.OpenFile(dst, os.O_RDONLY|os.O_CREATE|os.O_TRUNC, 0)
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
