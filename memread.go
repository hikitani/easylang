package easylang

import (
	"errors"
	"io"
	"unsafe"
)

var (
	_ io.Reader = &ReaderWithType{}
	_ io.Reader = MemReaderBool{}
	_ io.Reader = MemReaderFunc{}
)

type ReaderWithType struct {
	readed bool
	Type   VarType
	Parent io.Reader
}

func (r *ReaderWithType) Read(p []byte) (n int, err error) {
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

type MemReaderBool struct {
	v bool
}

// Read implements io.Reader.
func (m MemReaderBool) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return
	}

	boolread(&p[0], &m.v)
	return n, nil
}

func boolread(dst *byte, src *bool) {
	*dst = *(*byte)(unsafe.Pointer(src))
}

type MemReaderFunc struct{}

func (m MemReaderFunc) Read(p []byte) (n int, err error) {
	return 0, errors.New("function has no memory")
}
