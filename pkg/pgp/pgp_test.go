/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package pgp_test

import (
	"bytes"
	"testing"

	pgpcrypto "github.com/ProtonMail/gopenpgp/v3/crypto"

	"github.com/osspkg/pkg-build/pkg/pgp"
)

func TestSignerUnlocksPasswordProtectedKey(t *testing.T) {
	const password = "test-password"

	cert, err := pgp.NewCertSHA512(pgp.Config{Name: "pkg-build test", Email: "pkg-build@example.com"})
	if err != nil {
		t.Fatal(err)
	}

	key, err := pgpcrypto.NewKey(cert.Private)
	if err != nil {
		t.Fatal(err)
	}
	lockedKey, err := pgpcrypto.PGP().LockKey(key, []byte(password))
	if err != nil {
		t.Fatal(err)
	}
	lockedPrivate, err := lockedKey.Armor()
	if err != nil {
		t.Fatal(err)
	}

	signer := pgp.New()
	if err := signer.SetKey([]byte(lockedPrivate), password); err != nil {
		t.Fatal(err)
	}

	var signature bytes.Buffer
	if err := signer.SignDetached(bytes.NewBufferString("Release data\n"), &signature); err != nil {
		t.Fatal(err)
	}
	if signature.Len() == 0 {
		t.Fatal("detached signature is empty")
	}

	if err := pgp.New().SetKey([]byte(lockedPrivate), "wrong-password"); err == nil {
		t.Fatal("wrong password unexpectedly unlocked the key")
	}
}
