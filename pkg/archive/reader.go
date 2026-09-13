/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package archive

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
)

// MaxEntrySize is the maximum decompressed size returned by Read.
const MaxEntrySize int64 = 10 << 20

var (
	// ErrEntryTooLarge reports that an archive entry exceeds MaxEntrySize.
	ErrEntryTooLarge = errors.New("archive entry exceeds maximum size")
	// ErrInvalidEntry reports an unsafe or non-regular archive entry.
	ErrInvalidEntry = errors.New("invalid archive entry")
	// ErrReaderClosed reports an operation on a closed archive reader.
	ErrReaderClosed = errors.New("archive reader is closed")
)

type TGZReader struct {
	fd     *os.File
	gz     *gzip.Reader
	tar    *tar.Reader
	closed bool
}

func NewReader(filename string) (*TGZReader, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	gw, err := gzip.NewReader(file)
	if err != nil {
		if closeErr := file.Close(); closeErr != nil {
			return nil, errors.Join(err, fmt.Errorf("close archive: %w", closeErr))
		}
		return nil, err
	}
	tw := tar.NewReader(gw)
	return &TGZReader{fd: file, gz: gw, tar: tw}, nil
}

func (v *TGZReader) Close() error {
	if v.closed {
		return nil
	}
	v.closed = true
	gzipErr := v.gz.Close()
	fileErr := v.fd.Close()
	return errors.Join(gzipErr, fileErr)
}

func (v *TGZReader) Reset() error {
	if v.closed {
		return ErrReaderClosed
	}
	if _, err := v.fd.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if err := v.gz.Reset(v.fd); err != nil {
		return err
	}
	v.tar = tar.NewReader(v.gz)
	return nil
}

func (v *TGZReader) Read(filename string) ([]byte, error) {
	if v.closed {
		return nil, ErrReaderClosed
	}
	if err := validateEntryName(filename); err != nil {
		return nil, err
	}
	for {
		hdr, err := v.tar.Next()
		if err != nil {
			return nil, err
		}
		if err := validateEntryName(hdr.Name); err != nil {
			return nil, err
		}
		if hdr.Name == filename {
			if !hdr.FileInfo().Mode().IsRegular() {
				return nil, fmt.Errorf("%w: %q is not a regular file", ErrInvalidEntry, filename)
			}
			data, err := io.ReadAll(io.LimitReader(v.tar, MaxEntrySize+1))
			if err != nil {
				return nil, err
			}
			if int64(len(data)) > MaxEntrySize {
				return nil, fmt.Errorf("%w: %q is larger than %d bytes", ErrEntryTooLarge, filename, MaxEntrySize)
			}
			return data, nil
		}
	}
}

func validateEntryName(name string) error {
	clean := path.Clean(name)
	if name == "" || strings.IndexByte(name, 0) >= 0 || path.IsAbs(name) || clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("%w: %q", ErrInvalidEntry, name)
	}
	return nil
}
