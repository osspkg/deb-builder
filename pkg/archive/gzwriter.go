package archive

import (
	"compress/gzip"
	"io/fs"
	"os"
)

func GZWriteFile(filename string, data []byte, perm fs.FileMode) error {
	fd, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer fd.Close()
	gzw := gzip.NewWriter(fd)
	defer gzw.Close()

	_, err = gzw.Write(data)
	return err
}
