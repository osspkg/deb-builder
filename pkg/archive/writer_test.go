/*
 *  Copyright (c) 2021-2023 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package archive_test

import (
	"os"
	"testing"

	"github.com/osspkg/deb-builder/pkg/archive"
	"github.com/stretchr/testify/require"
)

func TestTarGZ(t *testing.T) {
	trgz, err := archive.NewWriter("/tmp/test.tar.gz")
	require.NoError(t, err)

	err = os.WriteFile("/tmp/test.txt", []byte("aaaaa"), 0755)
	require.NoError(t, err)

	f, h, err := trgz.WriteData("hello.txt", []byte("bbbbb"))
	require.NoError(t, err)
	require.Equal(t, "6262626262d41d8cd98f00b204e9800998ecf8427e", h)
	require.Equal(t, "hello.txt", f)

	f, h, err = trgz.WriteFile("/tmp/test.txt", "var/log/test.log")
	require.NoError(t, err)
	require.Equal(t, "594f803b380a41396ed63dca39503542", h)
	require.Equal(t, "var/log/test.log", f)

	err = trgz.Close()
	require.NoError(t, err)
}
