package pgp

import (
	"bytes"
	"crypto"
	"io"

	"github.com/deweppro/go-errors"
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
		return errors.WrapMessage(err, "Armor decode")
	}
	if block.Type != openpgp.PrivateKeyType {
		return errors.WrapMessage(err, "invalid key type")
	}
	if _, err = r.Seek(0, 0); err != nil {
		return errors.WrapMessage(err, "seek file")
	}
	keys, err := openpgp.ReadArmoredKeyRing(r)
	if err != nil {
		return errors.WrapMessage(err, "read key")
	}
	v.key = keys[0]
	if v.key.PrivateKey.Encrypted {
		if err := v.key.PrivateKey.Decrypt([]byte(passwd)); err != nil {
			return errors.WrapMessage(err, "invalid password")
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
