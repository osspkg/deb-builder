/*
 *  Copyright (c) 2021-2023 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

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
