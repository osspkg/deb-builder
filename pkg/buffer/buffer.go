package buffer

import (
	"bytes"

	"github.com/deweppro/go-app/console"
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
