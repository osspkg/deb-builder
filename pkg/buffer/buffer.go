/*
 *  Copyright (c) 2021-2023 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package buffer

import (
	"bytes"

	"github.com/osspkg/go-sdk/console"
)

type Buffer struct {
	a string
	b *bytes.Buffer
}

func New(arch string) *Buffer {
	return &Buffer{
		a: arch,
		b: &bytes.Buffer{},
	}
}

func (v *Buffer) Write(b []byte) {
	_, err := v.b.Write(b)
	console.FatalIfErr(err, "write %s package", v.a)
}

func (v *Buffer) Bytes() []byte {
	return v.b.Bytes()
}
