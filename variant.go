package easylang

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
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

func (v *VariantNum) IsZero() bool {
	n, acc := v.v.Int64()
	return n == 0 && acc == big.Exact
}

func (v *VariantNum) IsInf() bool {
	return v.v.IsInf()
}

func (v *VariantNum) Sign() int {
	return v.v.Sign()
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

func (v *VariantNum) AsInt64() (int64, error) {
	if !v.v.IsInt() {
		return 0, errors.New("number is not integer")
	}

	num, acc := v.v.Int64()
	if acc == big.Above && num == math.MinInt64 {
		return 0, errors.New("number less than -2^63 (min int64)")
	}

	if acc == big.Below && num == math.MaxInt64 {
		return 0, errors.New("number greater than 2^63 - 1 (max int64)")
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

func (v *VariantString) String() string {
	return v.v
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

func (v *VariantArray) Get(idx int64) (Variant, error) {
	norm := idx
	if idx < 0 {
		norm = int64(len(v.v)) + idx
	}

	if norm >= int64(len(v.v)) {
		return nil, fmt.Errorf("index %d out of range", idx)
	}

	return v.v[norm], nil
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
	v func(args []Variant) (Variant, error)
}

func (v *VariantFunc) Call(args []Variant) (Variant, error) {
	return v.v(args)
}

func (v *VariantFunc) MemReader() io.Reader {
	return MemReaderFunc{}
}

func (v *VariantFunc) Type() VarType {
	return TypeFunc
}

func VariantsIsDeepEqual(lval, rval Variant) bool {
	if lval == nil {
		return rval == nil
	} else if rval == nil {
		return false
	}

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
		larr, rarr := MustVariantCast[*VariantArray](lval), MustVariantCast[*VariantArray](rval)
		if len(larr.v) != len(rarr.v) {
			return false
		}

		for i := 0; i < len(larr.v); i++ {
			lv, rv := larr.v[i], rarr.v[i]
			if !VariantsIsDeepEqual(lv, rv) {
				return false
			}
		}

		return true
	case TypeObject:
		lobj, robj := MustVariantCast[*VariantObject](lval), MustVariantCast[*VariantObject](rval)
		if lobj.v == nil && robj.v == nil {
			return true
		}

		var llen, rlen int
		if lobj.v != nil {
			llen = len(lobj.v)
		}

		if robj.v != nil {
			rlen = len(robj.v)
		}

		if llen != rlen {
			return false
		}

		for k, lv := range lobj.v {
			rv, ok := robj.v[k]
			if !ok {
				return false
			}

			if !VariantsIsDeepEqual(lv, rv) {
				return false
			}
		}

		return true
	case TypeFunc:
		return false
	}
	panic("is equal: unknown type " + lval.Type().String())
}

func NewVarNone() *VariantNone {
	return &VariantNone{}
}

func NewVarBool(v bool) *VariantBool {
	return &VariantBool{v: v}
}

func NewVarNum(v *big.Float) *VariantNum {
	return &VariantNum{v: v}
}

func NewVarString(v string) *VariantString {
	return &VariantString{v: v}
}

func NewVarArray(v []Variant) *VariantArray {
	return &VariantArray{v: v}
}

func NewVarObject(v map[string]Variant) *VariantObject {
	return &VariantObject{v: v}
}

func NewVarFunc(v func(args []Variant) (Variant, error)) *VariantFunc {
	return &VariantFunc{v: v}
}

func NewVarInt[T ~int](v T) *VariantNum {
	f := new(big.Float).SetInt64(int64(v))
	return &VariantNum{v: f}
}

func NewVarFloat[T float32 | float64](v T) *VariantNum {
	f := new(big.Float).
		SetPrec(64).
		SetMode(big.ToNearestEven).
		SetFloat64(float64(v))
	return &VariantNum{v: f}
}

func NewVarInf() *VariantNum {
	f := new(big.Float).SetInf(false)
	return &VariantNum{v: f}
}

func NewVarNegInf() *VariantNum {
	f := new(big.Float).SetInf(true)
	return &VariantNum{v: f}
}
