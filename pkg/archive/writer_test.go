/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package archive_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/osspkg/pkg-build/pkg/archive"
)

func TestTarGZ(t *testing.T) {
	trgz, err := archive.NewWriter("/tmp/test.tar.gz")
	require.NoError(t, err)

	err = os.WriteFile("/tmp/test.txt", []byte("aaaaa"), 0755)
	require.NoError(t, err)

	f, h, err := trgz.WriteData("hello.txt", []byte("bbbbb"))
	require.NoError(t, err)
	require.Equal(t, "a21075a36eeddd084e17611a238c7101", h)
	require.Equal(t, "hello.txt", f)

	f, h, err = trgz.WriteFile("/tmp/test.txt", "var/log/test.log")
	require.NoError(t, err)
	require.Equal(t, "594f803b380a41396ed63dca39503542", h)
	require.Equal(t, "var/log/test.log", f)

	err = trgz.Close()
	require.NoError(t, err)
}

func TestTarGZIsReproducible(t *testing.T) {
	var archives [][]byte
	for range 2 {
		filename := filepath.Join(t.TempDir(), "data.tar.gz")
		writer, err := archive.NewWriter(filename)
		require.NoError(t, err)
		_, _, err = writer.WriteData("var/lib/demo/data", []byte("stable content"))
		require.NoError(t, err)
		require.NoError(t, writer.Close())

		data, err := os.ReadFile(filename)
		require.NoError(t, err)
		archives = append(archives, data)
	}

	require.Equal(t, archives[0], archives[1])
}
