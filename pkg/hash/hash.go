/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package hash

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

type MultiHash struct {
	MD5, SHA1, SHA256 string
}

func CalcMultiHash(filename string) (MultiHash, error) {
	var mh MultiHash
	fd, err := os.Open(filename)
	if err != nil {
		return mh, err
	}
	defer fd.Close() //nolint:errcheck

	md5h := md5.New()
	sha1h := sha1.New()
	sha256h := sha256.New()

	multiWriter := io.MultiWriter(md5h, sha1h, sha256h)

	if _, err := io.Copy(multiWriter, fd); err != nil {
		return mh, err
	}

	mh.MD5 = hex.EncodeToString(md5h.Sum(nil))
	mh.SHA1 = hex.EncodeToString(sha1h.Sum(nil))
	mh.SHA256 = hex.EncodeToString(sha256h.Sum(nil))

	return mh, nil
}
