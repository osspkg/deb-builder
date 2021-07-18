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

	"deb-builder/pkg/utils"
)

type TarGZ struct {
	file *os.File
	gz   *gzip.Writer
	tar  *tar.Writer
	size int64
	dirs map[string]struct{}
}

func NewTarGZ(filename string) (*TarGZ, error) {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	gw := gzip.NewWriter(file)
	tw := tar.NewWriter(gw)
	return &TarGZ{file: file, gz: gw, tar: tw, dirs: make(map[string]struct{})}, nil
}

func (v *TarGZ) Size() int64 {
	return v.size
}

func (v *TarGZ) Close() error {
	if err := v.tar.Close(); err != nil {
		return err
	}
	if err := v.gz.Close(); err != nil {
		return err
	}
	if err := v.file.Close(); err != nil {
		return err
	}
	return nil
}

func (v *TarGZ) WriteData(filename string, b []byte) (string, string, error) {
	dst := utils.TarFilesPath(filename)
	if err := v.mkdirAll(dst); err != nil {
		return utils.CleanPath(dst), "", err
	}
	hdr := &tar.Header{
		Name:     dst,
		ModTime:  time.Now(),
		Mode:     int64(0644),
		Size:     int64(len(b)),
		Typeflag: tar.TypeReg,
	}
	if err := v.tar.WriteHeader(hdr); err != nil {
		return utils.CleanPath(dst), "", err
	}
	if size, err := v.tar.Write(b); err != nil {
		return utils.CleanPath(dst), "", err
	} else {
		v.size += int64(size)
	}
	return utils.CleanPath(dst), hex.EncodeToString(md5.New().Sum(b)), nil
}

func (v *TarGZ) WriteFile(src, dst string) (string, string, error) {
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
		ModTime:  stat.ModTime(),
		Mode:     int64(stat.Mode()),
		Size:     stat.Size(),
		Typeflag: tar.TypeReg,
	}
	if err := v.tar.WriteHeader(hdr); err != nil {
		return utils.CleanPath(dst), "", err
	}
	if size, err := io.Copy(v.tar, file); err != nil {
		return utils.CleanPath(dst), "", err
	} else {
		v.size += size
		file.Seek(0, 0) //nolint: errcheck
	}
	hx := md5.New()
	if _, err := io.Copy(hx, file); err != nil {
		return utils.CleanPath(dst), "", err
	}
	return utils.CleanPath(dst), hex.EncodeToString(hx.Sum(nil)), nil
}

func (v *TarGZ) mkdirAll(filename string) error {
	path, list := "", strings.Split(filename, "/")
	for i := 0; i < len(list)-1; i++ {
		path = path + list[i] + "/"
		if _, ok := v.dirs[path]; ok {
			continue
		}
		hdr := &tar.Header{
			Name:     path,
			ModTime:  time.Now(),
			Mode:     int64(0755),
			Typeflag: tar.TypeDir,
		}
		if err := v.tar.WriteHeader(hdr); err != nil {
			return err
		}
		v.dirs[path] = struct{}{}
	}
	return nil
}
