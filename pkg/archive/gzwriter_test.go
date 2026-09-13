/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package archive_test

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/osspkg/pkg-build/pkg/archive"
)

func TestGZWriteFileWritesCompleteStream(t *testing.T) {
	want := []byte("release index\n")
	filename := filepath.Join(t.TempDir(), "Packages.gz")

	require.NoError(t, archive.GZWriteFile(filename, want, 0600))

	file, err := os.Open(filename)
	require.NoError(t, err)
	defer file.Close()

	reader, err := gzip.NewReader(file)
	require.NoError(t, err)
	got, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	require.Equal(t, want, got)
}
