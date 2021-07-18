package archive

import (
	"io"
	"os"
	"strconv"
	"time"
)

var (
	_ io.WriterTo = (*Header)(nil)
)

const headerSize = 60

var (
	signeture  = []byte("!<arch>\n")
	whitespace = []byte(" ")[0]
	x60        = []byte("\x60")
	x0A        = []byte("\x0a")
)

/**

**/
type Header struct {
	FileName  string
	Timestamp time.Time
	Mode      int64
	Size      int64
}

func (v *Header) WriteTo(w io.Writer) (int64, error) {
	data := make([]byte, headerSize)
	for i := 0; i < headerSize; i++ {
		data[i] = whitespace
	}

	copy(data[0:16], []byte(v.FileName))
	copy(data[16:28], []byte(strconv.FormatInt(v.Timestamp.Unix(), 10)))
	copy(data[28:34], []byte("0"))
	copy(data[34:40], []byte("0"))
	copy(data[40:48], []byte("100"+strconv.FormatInt(v.Mode, 8)))
	copy(data[48:58], []byte(strconv.FormatInt(v.Size, 10)))
	copy(data[58:60], append(x60, x0A...))

	i, err := w.Write(data)
	return int64(i), err
}

/**

**/
type Deb struct {
	file *os.File
}

func NewDeb(filename string) (*Deb, error) {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	if _, err = file.Write(signeture); err != nil {
		return nil, err
	}
	ar := &Deb{file: file}
	if err = ar.WriteData("debian-binary", []byte("2.0\n")); err != nil {
		return nil, err
	}
	return ar, nil
}

func (v *Deb) WriteFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close() //nolint: errcheck
	stat, serr := file.Stat()
	if serr != nil {
		return serr
	}
	h := &Header{
		FileName:  stat.Name(),
		Timestamp: stat.ModTime(),
		Mode:      int64(0644),
		Size:      stat.Size(),
	}
	if _, err := h.WriteTo(v.file); err != nil {
		return err
	}
	if cnt, err := io.Copy(v.file, file); err != nil {
		return err
	} else {
		if err := v.correct(cnt); err != nil {
			return err
		}
	}
	return nil
}

func (v *Deb) WriteData(filename string, b []byte) error {
	h := &Header{
		FileName:  filename,
		Timestamp: time.Now(),
		Mode:      int64(0644),
		Size:      int64(len(b)),
	}
	if _, err := h.WriteTo(v.file); err != nil {
		return err
	}
	if cnt, err := v.file.Write(b); err != nil {
		return err
	} else {
		if err := v.correct(int64(cnt)); err != nil {
			return err
		}
	}
	return nil
}

func (v *Deb) correct(size int64) error {
	if size%2 == 0 {
		return nil
	}
	if _, err := v.file.Write(x0A); err != nil {
		return err
	}
	return nil
}

func (v *Deb) Close() error {
	return v.file.Close()
}
