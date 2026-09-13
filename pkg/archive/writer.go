/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package archive

import (
	"archive/tar"
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
	"strings"
	"time"

	"github.com/osspkg/pkg-build/pkg/utils"
)

type TGZWriter struct {
	fd   *os.File
	gz   *gzip.Writer
	tar  *tar.Writer
	size int64
	dirs map[string]struct{}
}

var archiveModTime = time.Unix(0, 0).UTC()

func NewWriter(filename string) (*TGZWriter, error) {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	gw := gzip.NewWriter(file)
	tw := tar.NewWriter(gw)
	return &TGZWriter{fd: file, gz: gw, tar: tw, dirs: make(map[string]struct{})}, nil
}

func (v *TGZWriter) Size() int64 {
	return v.size
}

func (v *TGZWriter) Close() error {
	if err := v.tar.Close(); err != nil {
		return err
	}
	if err := v.gz.Close(); err != nil {
		return err
	}
	if err := v.fd.Close(); err != nil {
		return err
	}
	return nil
}

func (v *TGZWriter) WriteData(filename string, b []byte) (string, string, error) {
	dst := utils.TarFilesPath(filename)
	if err := v.mkdirAll(dst); err != nil {
		return utils.CleanPath(dst), "", err
	}
	hdr := &tar.Header{
		Name:     dst,
		ModTime:  archiveModTime,
		Mode:     int64(0644),
		Size:     int64(len(b)),
		Typeflag: tar.TypeReg,
		Format:   tar.FormatGNU,
	}
	if err := v.tar.WriteHeader(hdr); err != nil {
		return utils.CleanPath(dst), "", err
	}
	if size, err := v.tar.Write(b); err != nil {
		return utils.CleanPath(dst), "", err
	} else {
		v.size += int64(size)
	}
	sum := md5.Sum(b)
	return utils.CleanPath(dst), hex.EncodeToString(sum[:]), nil
}

func (v *TGZWriter) WriteFile(src, dst string) (string, string, error) {
	dst = utils.TarFilesPath(dst)
	file, err := os.Open(src)
	if err != nil {
		return utils.CleanPath(dst), "", err
	}
	defer file.Close() //nolint: errcheck
	stat, err1 := file.Stat()
	if err1 != nil {
		return utils.CleanPath(dst), "", err1
	}
	if err := v.mkdirAll(dst); err != nil {
		return utils.CleanPath(dst), "", err
	}
	hdr := &tar.Header{
		Name:     dst,
		ModTime:  archiveModTime,
		Mode:     int64(stat.Mode()),
		Size:     stat.Size(),
		Typeflag: tar.TypeReg,
		Format:   tar.FormatGNU,
	}
	if err := v.tar.WriteHeader(hdr); err != nil {
		return utils.CleanPath(dst), "", err
	}
	if size, err := io.Copy(v.tar, file); err != nil {
		return utils.CleanPath(dst), "", err
	} else {
		v.size += size
		if _, err = file.Seek(0, 0); err != nil {
			return "", "", err
		}
	}
	hx := md5.New()
	if _, err := io.Copy(hx, file); err != nil {
		return utils.CleanPath(dst), "", err
	}
	return utils.CleanPath(dst), hex.EncodeToString(hx.Sum(nil)), nil
}

func (v *TGZWriter) mkdirAll(filename string) error {
	path, list := "", strings.Split(filename, "/")
	for i := 0; i < len(list)-1; i++ {
		path = path + list[i] + "/"
		if _, ok := v.dirs[path]; ok {
			continue
		}
		hdr := &tar.Header{
			Name:     path,
			ModTime:  archiveModTime,
			Mode:     int64(0755),
			Typeflag: tar.TypeDir,
			Format:   tar.FormatGNU,
		}
		if err := v.tar.WriteHeader(hdr); err != nil {
			return err
		}
		v.dirs[path] = struct{}{}
	}
	return nil
}
