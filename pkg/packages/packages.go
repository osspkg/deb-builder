/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package packages

type PackegesModel struct {
	Package      string `key:"Package"`
	Source       string `key:"Source"`
	Version      string `key:"Version"`
	Architecture string `key:"Architecture"`
	Maintainer   string `key:"Maintainer"`
	Filename     string `key:"Filename"`
	Size         int64  `key:"Size"`
	MD5sum       string `key:"MD5sum"`
	SHA1         string `key:"SHA1"`
	SHA256       string `key:"SHA256"`
	Raw          string `key:"_"`
}

func (v *PackegesModel) Decode(data []byte) error {
	return decode(data, v)
}

func (v *PackegesModel) Encode() ([]byte, error) {
	return encode(v)
}

//////////////////////////////////////////////////////////////////////////////////////////////////

type ReleaseModel struct {
	Component    string `key:"Component"`
	Origin       string `key:"Origin"`
	Label        string `key:"Label"`
	Architecture string `key:"Architecture"`
	Description  string `key:"Description"`
}

func (v *ReleaseModel) Decode(data []byte) error {
	return decode(data, v)
}

func (v *ReleaseModel) Encode() ([]byte, error) {
	return encode(v)
}

//////////////////////////////////////////////////////////////////////////////////////////////////

type InReleaseModel struct {
	Origin        string `key:"Origin"`
	Label         string `key:"Label"`
	Component     string `key:"Component"`
	Codename      string `key:"Codename"`
	Date          string `key:"Date"`
	Architectures string `key:"Architectures"`
	Description   string `key:"Description"`
	MD5Sum        string `key:"MD5Sum"`
	SHA1          string `key:"SHA1"`
	SHA256        string `key:"SHA256"`
}

func (v *InReleaseModel) Decode(data []byte) error {
	return decode(data, v)
}

func (v *InReleaseModel) Encode() ([]byte, error) {
	return encode(v)
}
