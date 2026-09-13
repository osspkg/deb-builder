/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package archive

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io/fs"
	"os"
)

func GZWriteFile(filename string, data []byte, perm fs.FileMode) (retErr error) {
	fd, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}

	gzw := gzip.NewWriter(fd)
	defer func() {
		if err := gzw.Close(); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("close gzip writer: %w", err))
		}
		if err := fd.Close(); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("close output file: %w", err))
		}
	}()

	_, retErr = gzw.Write(data)
	return retErr
}
