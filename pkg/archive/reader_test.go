/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package archive_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/osspkg/pkg-build/pkg/archive"
)

func TestReaderReadControlFile(t *testing.T) {
	want := []byte("Package: pkg-build\nVersion: 1.0.0\n")
	reader := openTestReader(t, "./control", want)

	got, err := reader.Read("./control")
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestReaderRejectsOversizedEntry(t *testing.T) {
	data := bytes.Repeat([]byte{'x'}, int(archive.MaxEntrySize)+1)
	reader := openTestReader(t, "./control", data)

	got, err := reader.Read("./control")
	require.ErrorIs(t, err, archive.ErrEntryTooLarge)
	require.Nil(t, got)
}

func TestReaderRejectsMalformedTar(t *testing.T) {
	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)
	_, err := gz.Write([]byte("not a tar stream"))
	require.NoError(t, err)
	require.NoError(t, gz.Close())

	filename := filepath.Join(t.TempDir(), "malformed.tar.gz")
	require.NoError(t, os.WriteFile(filename, compressed.Bytes(), 0600))
	reader, err := archive.NewReader(filename)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, reader.Close()) })

	_, err = reader.Read("./control")
	require.Error(t, err)
}

func TestReaderRejectsMissingControlFile(t *testing.T) {
	reader := openTestReader(t, "./other", []byte("content"))

	_, err := reader.Read("./control")
	require.Error(t, err)
}

func TestReaderRejectsTraversalEntryName(t *testing.T) {
	reader := openTestReader(t, "../outside", []byte("content"))

	_, err := reader.Read("../outside")
	require.ErrorIs(t, err, archive.ErrInvalidEntry)
	_, err = reader.Read("./control")
	require.ErrorIs(t, err, archive.ErrInvalidEntry)
}

func TestReaderRejectsSymlinkEntry(t *testing.T) {
	reader := openTestReaderWithHeader(t, &tar.Header{
		Name:     "./control",
		Typeflag: tar.TypeSymlink,
		Linkname: "../../outside",
	})

	_, err := reader.Read("./control")
	require.ErrorIs(t, err, archive.ErrInvalidEntry)
}

func TestReaderCanBeReadAgainAfterReset(t *testing.T) {
	want := []byte("Package: pkg-build\n")
	reader := openTestReader(t, "./control", want)

	got, err := reader.Read("./control")
	require.NoError(t, err)
	require.Equal(t, want, got)
	require.NoError(t, reader.Reset())

	got, err = reader.Read("./control")
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestReaderRejectsOperationsAfterClose(t *testing.T) {
	reader := openTestReader(t, "./control", []byte("content"))
	require.NoError(t, reader.Close())

	_, err := reader.Read("./control")
	require.Error(t, err)
	require.Error(t, reader.Reset())
}

func TestNewReaderClosesFileOnInvalidGzip(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "invalid.gz")
	require.NoError(t, os.WriteFile(filename, []byte("not a gzip stream"), 0600))

	before := openFileDescriptorCount(t)
	for range 32 {
		reader, err := archive.NewReader(filename)
		require.Error(t, err)
		require.Nil(t, reader)
	}
	after := openFileDescriptorCount(t)
	require.LessOrEqual(t, after, before+1)
}

func openFileDescriptorCount(t *testing.T) int {
	t.Helper()

	entries, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		t.Skipf("/proc/self/fd is unavailable: %v", err)
	}
	return len(entries)
}

func openTestReader(t *testing.T, name string, data []byte) *archive.TGZReader {
	t.Helper()

	return openTestReaderWithHeader(t, &tar.Header{
		Name: name,
		Mode: 0644,
		Size: int64(len(data)),
	}, data)
}

func openTestReaderWithHeader(t *testing.T, header *tar.Header, data ...[]byte) *archive.TGZReader {
	t.Helper()

	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)
	tarWriter := tar.NewWriter(gz)
	require.NoError(t, tarWriter.WriteHeader(header))
	if len(data) > 0 {
		_, err := tarWriter.Write(data[0])
		require.NoError(t, err)
	}
	require.NoError(t, tarWriter.Close())
	require.NoError(t, gz.Close())

	filename := filepath.Join(t.TempDir(), "control.tar.gz")
	require.NoError(t, os.WriteFile(filename, compressed.Bytes(), 0600))
	reader, err := archive.NewReader(filename)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, reader.Close()) })
	return reader
}
