/*
 *  Copyright (c) 2021-2023 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package pgp

import (
	"bytes"
	"crypto"
	"fmt"
	"io"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/clearsign"
	"golang.org/x/crypto/openpgp/packet"
)

const (
	DefaultRSAKeyBits = 4096
)

type PGP struct {
	key  *openpgp.Entity
	conf *packet.Config
}

func NewPGP() *PGP {
	return &PGP{
		conf: &packet.Config{
			DefaultHash: crypto.SHA512,
		},
	}
}

func (v *PGP) LoadPrivateKey(r io.ReadSeeker, passwd string) error {
	block, err := armor.Decode(r)
	if err != nil {
		return fmt.Errorf("armor decode: %w", err)
	}
	if block.Type != openpgp.PrivateKeyType {
		return fmt.Errorf("invalid key type: %w", err)
	}
	if _, err = r.Seek(0, 0); err != nil {
		return fmt.Errorf("seek file: %w", err)
	}
	keys, err := openpgp.ReadArmoredKeyRing(r)
	if err != nil {
		return fmt.Errorf("read key: %w", err)
	}
	v.key = keys[0]
	if v.key.PrivateKey.Encrypted {
		if err = v.key.PrivateKey.Decrypt([]byte(passwd)); err != nil {
			return fmt.Errorf("invalid password: %w", err)
		}
	}
	return nil
}

func (v *PGP) Sign(in io.Reader, out io.Writer) error {
	w, err := clearsign.Encode(out, v.key.PrivateKey, v.conf)
	if err != nil {
		return err
	}

	if _, err = io.Copy(w, in); err != nil {
		return err
	}

	if err = w.Close(); err != nil {
		return err
	}
	// if err := openpgp.ArmoredDetachSignText(out, v.key, in, v.conf); err != nil {
	// 	return err
	// }
	return nil
}

func (v *PGP) GetPublic() ([]byte, error) {
	buf := &bytes.Buffer{}
	if err := v.key.Serialize(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (v *PGP) GetPublicBase64() ([]byte, error) {
	buf := &bytes.Buffer{}
	enc, err := armor.Encode(buf, openpgp.PublicKeyType, map[string]string{})
	if err != nil {
		return nil, err
	}
	if err = v.key.Serialize(enc); err != nil {
		return nil, err
	}
	if err = enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
