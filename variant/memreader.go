package variant

import (
	"errors"
	"io"
	"unsafe"
)

var (
	_ io.Reader = &readerWithType{}
	_ io.Reader = memReaderBool{}
	_ io.Reader = memReaderFunc{}
)

type readerWithType struct {
	readed bool
	Type   Type
	Parent io.Reader
}

func (r *readerWithType) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return
	}

	if !r.readed {
		p[0] = byte(r.Type)
		r.readed = true
		p = p[1:]
		n++
	}

	if r.Parent == nil {
		return n, io.EOF
	}

	nn, err := r.Parent.Read(p)
	return n + nn, err
}

type memReaderBool struct {
	v bool
}

// Read implements io.Reader.
func (m memReaderBool) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return
	}

	boolread(&p[0], &m.v)
	return n, io.EOF
}

func boolread(dst *byte, src *bool) {
	*dst = *(*byte)(unsafe.Pointer(src))
}

type memReaderFunc struct{}

func (m memReaderFunc) Read(p []byte) (n int, err error) {
	return 0, errors.New("function has no memory")
}
