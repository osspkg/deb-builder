/*
 *  Copyright (c) 2021-2023 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package archive

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
)

type TGZReader struct {
	fd  *os.File
	gz  *gzip.Reader
	tar *tar.Reader
}

func NewReader(filename string) (*TGZReader, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	gw, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	tw := tar.NewReader(gw)
	return &TGZReader{fd: file, gz: gw, tar: tw}, nil
}

func (v *TGZReader) Close() error {
	if err := v.gz.Close(); err != nil {
		return err
	}
	if err := v.fd.Close(); err != nil {
		return err
	}
	return nil
}

func (v *TGZReader) Reset() error {
	return v.gz.Reset(v.tar)
}

func (v *TGZReader) Read(filename string) ([]byte, error) {
	for {
		hdr, err := v.tar.Next()
		if err != nil {
			return nil, err
		}
		if hdr.Name == filename {
			return io.ReadAll(v.tar)
		}
	}
}
