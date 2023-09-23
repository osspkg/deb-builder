/*
 *  Copyright (c) 2021-2023 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package pgp_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/osspkg/deb-builder/pkg/pgp"
	"github.com/stretchr/testify/require"
)

func TestPGP(t *testing.T) {
	enc := pgp.NewPGP()

	err := enc.Generate("/tmp", "Demo", "", "demo@email.xxx")
	require.NoError(t, err)

	keyFile, err := os.Open("/tmp/private.pgp")
	require.NoError(t, err)

	err = enc.LoadPrivateKey(keyFile, "")
	require.NoError(t, err)

	in := bytes.NewBufferString("Hello world")
	out := &bytes.Buffer{}

	err = enc.Sign(in, out)
	require.NoError(t, err)

	sign := `-----BEGIN PGP SIGNED MESSAGE-----
Hash: SHA512

Hello world
-----BEGIN PGP SIGNATURE-----`

	require.Contains(t, out.String(), sign)

	err = os.WriteFile("/tmp/message.dsc", out.Bytes(), 0644)
	require.NoError(t, err)
}
