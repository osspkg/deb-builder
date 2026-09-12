/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package pgp

import (
	"bytes"
	"crypto"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/gopenpgp/v3/constants"
	pgpcrypto "github.com/ProtonMail/gopenpgp/v3/crypto"
	"github.com/ProtonMail/gopenpgp/v3/profile"
)

type (
	Config struct {
		Name, Email, Comment string
	}

	Cert struct {
		Public  []byte
		Private []byte
	}

	Signer interface {
		SetKey(b []byte, passwd string) error
		SetKeyFromFile(filename string, passwd string) error
		PublicKey() ([]byte, error)
		PublicKeyBase64() ([]byte, error)
		Sign(in io.Reader, out io.Writer) error
		SignDetached(in io.Reader, out io.Writer) error
	}
)

type signer struct {
	handle  *pgpcrypto.PGPHandle
	key     *pgpcrypto.Key
	keyRing *pgpcrypto.KeyRing
}

// New returns a signer configured for RFC 4880-compatible OpenPGP keys.
func New() Signer {
	keyProfile := profile.RFC4880()
	keyProfile.Hash = crypto.SHA512
	keyProfile.SignHash = hashPointer(crypto.SHA512)
	return &signer{
		handle: pgpcrypto.PGPWithProfile(keyProfile),
	}
}

// NewCertSHA512 generates a 4096-bit RSA OpenPGP certificate.
func NewCertSHA512(c Config) (*Cert, error) {
	keyProfile := profile.RFC4880()
	keyProfile.Hash = crypto.SHA512
	keyProfile.SignHash = hashPointer(crypto.SHA512)
	pgpHandle := pgpcrypto.PGPWithProfile(keyProfile)

	key, err := pgpHandle.KeyGeneration().
		AddUserId(c.Name, c.Email).
		New().
		GenerateKeyWithSecurity(constants.HighSecurity)
	if err != nil {
		return nil, fmt.Errorf("generate OpenPGP key: %w", err)
	}
	defer key.ClearPrivateParams()

	privateKey, err := key.Armor()
	if err != nil {
		return nil, fmt.Errorf("serialize private OpenPGP key: %w", err)
	}
	publicKey, err := key.GetArmoredPublicKey()
	if err != nil {
		return nil, fmt.Errorf("serialize public OpenPGP key: %w", err)
	}
	return &Cert{
		Private: []byte(privateKey),
		Public:  []byte(publicKey),
	}, nil
}

func hashPointer(hash crypto.Hash) *crypto.Hash {
	return &hash
}

func (s *signer) SetKey(b []byte, passwd string) error {
	key, err := pgpcrypto.NewKey(b)
	if err != nil {
		return fmt.Errorf("read OpenPGP key: %w", err)
	}
	if !key.IsPrivate() {
		return errors.New("OpenPGP key does not contain private key material")
	}

	locked, err := key.IsLocked()
	if err != nil {
		return fmt.Errorf("inspect OpenPGP key: %w", err)
	}
	if locked {
		key, err = key.Unlock([]byte(passwd))
		if err != nil {
			return fmt.Errorf("unlock OpenPGP key: %w", err)
		}
	}

	keyRing, err := pgpcrypto.NewKeyRing(key)
	if err != nil {
		return fmt.Errorf("create OpenPGP key ring: %w", err)
	}
	s.key = key
	s.keyRing = keyRing
	return nil
}

func (s *signer) SetKeyFromFile(filename string, passwd string) error {
	// The path is explicitly supplied as the private-key configuration value.
	// It is not used to construct a command or to select an archive entry.
	//nolint:gosec // the caller controls the private-key file path by design
	keyFile, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("read key from file: %w", err)
	}
	defer keyFile.Close() //nolint:errcheck // read errors are returned below

	keyData, err := io.ReadAll(keyFile)
	if err != nil {
		return fmt.Errorf("read key from file: %w", err)
	}
	return s.SetKey(keyData, passwd)
}

func (s *signer) PublicKey() ([]byte, error) {
	if s.key == nil {
		return nil, errors.New("OpenPGP key is empty")
	}
	return s.key.GetPublicKey()
}

func (s *signer) PublicKeyBase64() ([]byte, error) {
	if s.key == nil {
		return nil, errors.New("OpenPGP key is empty")
	}
	publicKey, err := s.key.GetArmoredPublicKey()
	if err != nil {
		return nil, err
	}
	return []byte(publicKey), nil
}

func (s *signer) Sign(in io.Reader, out io.Writer) error {
	message, err := io.ReadAll(in)
	if err != nil {
		return fmt.Errorf("read message to sign: %w", err)
	}

	signHandle, err := s.newSignHandle(false)
	if err != nil {
		return err
	}
	defer signHandle.ClearPrivateParams()

	// The clearsign writer emits the final line ending itself. Avoid giving it
	// a second one, which would add an unsigned blank line before the armor.
	cleartextMessage := bytes.TrimSuffix(message, []byte{'\n'})
	signed, err := signHandle.SignCleartext(cleartextMessage)
	if err != nil {
		return fmt.Errorf("create cleartext signature: %w", err)
	}
	signed, err = addCleartextArmorChecksum(signed)
	if err != nil {
		return fmt.Errorf("complete cleartext signature armor: %w", err)
	}
	if _, err = io.Copy(out, bytes.NewReader(signed)); err != nil {
		return fmt.Errorf("write cleartext signature: %w", err)
	}
	return nil
}

func addCleartextArmorChecksum(signed []byte) ([]byte, error) {
	const (
		signatureStart = "-----BEGIN PGP SIGNATURE-----"
		signatureEnd   = "-----END PGP SIGNATURE-----"
	)

	start := bytes.Index(signed, []byte(signatureStart))
	if start < 0 {
		return nil, errors.New("signature armor header is missing")
	}
	endOffset := bytes.Index(signed[start:], []byte(signatureEnd))
	if endOffset < 0 {
		return nil, errors.New("signature armor footer is missing")
	}
	end := start + endOffset + len(signatureEnd)

	block, err := armor.Decode(bytes.NewReader(signed[start:end]))
	if err != nil {
		return nil, fmt.Errorf("decode signature armor: %w", err)
	}
	body, err := io.ReadAll(block.Body)
	if err != nil {
		return nil, fmt.Errorf("read signature armor: %w", err)
	}

	var armored bytes.Buffer
	encoder, err := armor.Encode(&armored, block.Type, block.Header)
	if err != nil {
		return nil, fmt.Errorf("initialize signature armor: %w", err)
	}
	if _, err = encoder.Write(body); err != nil {
		return nil, fmt.Errorf("write signature armor: %w", err)
	}
	if err = encoder.Close(); err != nil {
		return nil, fmt.Errorf("close signature armor: %w", err)
	}

	result := make([]byte, 0, len(signed)+8)
	result = append(result, signed[:start]...)
	result = append(result, armored.Bytes()...)
	result = append(result, signed[end:]...)
	return result, nil
}

func (s *signer) SignDetached(in io.Reader, out io.Writer) error {
	message, err := io.ReadAll(in)
	if err != nil {
		return fmt.Errorf("read message to sign: %w", err)
	}

	signHandle, err := s.newSignHandle(true)
	if err != nil {
		return err
	}
	defer signHandle.ClearPrivateParams()

	signature, err := signHandle.Sign(message, pgpcrypto.Bytes)
	if err != nil {
		return fmt.Errorf("create detached signature: %w", err)
	}
	if _, err = io.Copy(out, bytes.NewReader(signature)); err != nil {
		return fmt.Errorf("write detached signature: %w", err)
	}
	return nil
}

func (s *signer) newSignHandle(detached bool) (pgpcrypto.PGPSign, error) {
	if s.key == nil || s.keyRing == nil {
		return nil, errors.New("OpenPGP key is empty")
	}

	keyCopy, err := s.key.Copy()
	if err != nil {
		return nil, fmt.Errorf("copy OpenPGP signing key: %w", err)
	}
	keyRing, err := pgpcrypto.NewKeyRing(keyCopy)
	if err != nil {
		keyCopy.ClearPrivateParams()
		return nil, fmt.Errorf("create OpenPGP signing key ring: %w", err)
	}

	builder := s.handle.Sign().SigningKeys(keyRing)
	if detached {
		builder.Detached()
	}
	signHandle, err := builder.New()
	if err != nil {
		keyRing.ClearPrivateParams()
		return nil, fmt.Errorf("initialize OpenPGP signer: %w", err)
	}
	return signHandle, nil
}
