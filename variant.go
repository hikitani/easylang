package easylang

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strings"
)

type VarType uint8

var typNames = [TypeEnd]string{
	"null", "bool", "number", "string", "array", "object", "func",
}

func (typ VarType) String() string {
	return typNames[typ]
}

const (
	TypeNone VarType = iota
	TypeBool
	TypeNum
	TypeString
	TypeArray
	TypeObject
	TypeFunc

	TypeEnd
)

var (
	_ Variant = &VariantNone{}
	_ Variant = &VariantBool{}
	_ Variant = &VariantNum{}
	_ Variant = &VariantString{}
	_ Variant = &VariantArray{}
	_ Variant = &VariantObject{}
	_ Variant = &VariantFunc{}
)

type VarID uint64

type Variant interface {
	Type() VarType
	MemReader() io.Reader
}

func MustVariantCast[T Variant](v Variant) T {
	r, ok := v.(T)
	if !ok {
		panic("fatal on cast: expected " + v.Type().String() + " variant")
	}

	return r
}

type Ref struct {
	cnt uint64
	val Variant
}

func (r *Ref) IncRef() {
	// r
}

func NewRef(v Variant) *Ref {
	return &Ref{
		cnt: 0,
		val: v,
	}
}

type VariantNone struct{}

func (v *VariantNone) MemReader() io.Reader {
	return &ReaderWithType{Type: TypeNone}
}

func (v *VariantNone) Type() VarType {
	return TypeNone
}

type VariantBool struct {
	v bool
}

func (v *VariantBool) MemReader() io.Reader {
	return &ReaderWithType{
		Type:   TypeBool,
		Parent: MemReaderBool{v: v.v},
	}
}

func (v *VariantBool) Type() VarType {
	return TypeBool
}

type VariantNum struct {
	v *big.Float
}

func (v *VariantNum) LessThan(than *VariantNum) bool {
	return v.v.Cmp(than.v) == -1
}

func (v *VariantNum) LessOrEqualTo(to *VariantNum) bool {
	return v.v.Cmp(to.v) <= 0
}

func (v *VariantNum) GreaterThan(than *VariantNum) bool {
	return v.v.Cmp(than.v) == 1
}

func (v *VariantNum) GreaterOrEqualTo(to *VariantNum) bool {
	return v.v.Cmp(to.v) >= 0
}

func (v *VariantNum) EqualTo(to *VariantNum) bool {
	return v.v.Cmp(to.v) == 0
}

func (v *VariantNum) AsUInt64() (uint64, error) {
	if !v.v.IsInt() {
		return 0, errors.New("number is not integer")
	}

	num, acc := v.v.Uint64()
	if acc == big.Above {
		return 0, errors.New("number is negative")
	}

	if acc == big.Below {
		return 0, errors.New("number greater than 2^64")
	}

	return num, nil
}

func (v *VariantNum) MemReader() io.Reader {
	prec := v.v.Prec()
	cap := 10 + prec
	repr := v.v.Append(make([]byte, 0, cap), 'g', int(prec))
	return &ReaderWithType{
		Type:   TypeNum,
		Parent: bytes.NewBuffer(repr),
	}
}

func (v *VariantNum) Type() VarType {
	return TypeNum
}

type VariantString struct {
	v string
}

func (v *VariantString) MemReader() io.Reader {
	return &ReaderWithType{
		Type:   TypeString,
		Parent: strings.NewReader(v.v),
	}
}

func (v *VariantString) Type() VarType {
	return TypeString
}

type VariantArray struct {
	v []Variant
}

func (v *VariantArray) Len() int {
	return len(v.v)
}

func (v *VariantArray) Get(idx uint64) (Variant, error) {
	if idx < 0 || idx >= uint64(len(v.v)) {
		return nil, fmt.Errorf("index out of range")
	}

	return v.v[idx], nil
}

func (v VariantArray) MemReader() io.Reader {
	r := ReaderWithType{
		Type: TypeArray,
	}

	if len(v.v) == 0 {
		return &r
	}

	rr := make([]io.Reader, 0, len(v.v))
	for _, v := range v.v {
		rr = append(rr, v.MemReader())
	}

	r.Parent = io.MultiReader(rr...)
	return &r
}

func (v *VariantArray) Type() VarType {
	return TypeArray
}

type VariantObject struct {
	v map[string]Variant
}

func (v *VariantObject) Len() int {
	return len(v.v)
}

func (v *VariantObject) MemReader() io.Reader {
	r := ReaderWithType{
		Type: TypeObject,
	}

	if len(v.v) == 0 {
		return &r
	}

	rr := make([]io.Reader, 0, len(v.v)*2)
	for k, v := range v.v {
		rr = append(rr, strings.NewReader(k))
		rr = append(rr, v.MemReader())
	}

	r.Parent = io.MultiReader(rr...)
	return &r
}

func (v *VariantObject) Get(key Variant) (val Variant, err error) {
	kb, err := io.ReadAll(key.MemReader())
	if err != nil {
		return nil, fmt.Errorf("%s is not hashable", key.Type())
	}

	var ok bool
	val, ok = v.v[string(kb)]
	if !ok {
		return nil, errors.New("key not found")
	}

	return val, nil
}

func (v *VariantObject) Type() VarType {
	return TypeObject
}

type VariantFunc struct {
	expectedArgs int
	v            func(args []Variant) (Variant, error)
}

func (v *VariantFunc) ExpectedArgs() int {
	return v.expectedArgs
}

func (v *VariantFunc) Call(args []Variant) (Variant, error) {
	if v.expectedArgs != len(args) {
		return nil, fmt.Errorf("expected arguments %d, got %d", v.expectedArgs, len(args))
	}

	return v.v(args)
}

func (v *VariantFunc) MemReader() io.Reader {
	return MemReaderFunc{}
}

func (v *VariantFunc) Type() VarType {
	return TypeFunc
}

func VariantsIsEqual(lval, rval Variant) bool {
	if lval.Type() != rval.Type() {
		return false
	}

	switch lval.Type() {
	case TypeNone:
		return true
	case TypeBool:
		lb, rb := MustVariantCast[*VariantBool](lval), MustVariantCast[*VariantBool](rval)
		return lb.v == rb.v
	case TypeNum:
		lnum, rnum := MustVariantCast[*VariantNum](lval), MustVariantCast[*VariantNum](rval)
		return lnum.v.Cmp(rnum.v) == 0
	case TypeString:
		ls, rs := MustVariantCast[*VariantString](lval), MustVariantCast[*VariantString](rval)
		return ls.v == rs.v
	case TypeArray:
		return false
	case TypeObject:
		lobj, robj := MustVariantCast[*VariantObject](lval), MustVariantCast[*VariantObject](rval)
		if len(lobj.v) != len(robj.v) {
			return false
		}

		for k, lv := range lobj.v {
			rv, ok := robj.v[k]
			if !ok {
				return false
			}

			if !VariantsIsEqual(lv, rv) {
				return false
			}
		}

		return true
	case TypeFunc:
		return false
	}
	panic("is equal: unknown type " + lval.Type().String())
}
