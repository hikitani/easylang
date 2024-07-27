package variant

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"strings"

	"github.com/ALTree/bigfloat"
)

type Type uint8

var typNames = [TypeEnd]string{
	"null", "bool", "number", "string", "array", "object", "func",
}

func (typ Type) String() string {
	return typNames[typ]
}

const (
	TypeNone Type = iota
	TypeBool
	TypeNum
	TypeString
	TypeArray
	TypeObject
	TypeFunc

	TypeEnd
)

var (
	_ Iface = &None{}
	_ Iface = &Bool{}
	_ Iface = &Num{}
	_ Iface = &String{}
	_ Iface = &Array{}
	_ Iface = &Object{}
	_ Iface = &Func{}
)

type Iface interface {
	Type() Type
	MemReader() io.Reader
	String() string
}

func MustCast[T Iface](v Iface) T {
	r, ok := v.(T)
	if !ok {
		panic("fatal on cast: expected " + v.Type().String() + " variant")
	}

	return r
}

type None struct{}

func (v *None) MemReader() io.Reader {
	return &readerWithType{Type: TypeNone}
}

func (v *None) Type() Type {
	return TypeNone
}

func (v *None) String() string {
	return "none"
}

type Bool struct {
	v bool
}

func (v *Bool) Bool() bool {
	return v.v
}

func (v *Bool) MemReader() io.Reader {
	return &readerWithType{
		Type:   TypeBool,
		Parent: memReaderBool{v: v.v},
	}
}

func (v *Bool) Type() Type {
	return TypeBool
}

func (v *Bool) String() string {
	if v.v {
		return "true"
	}

	return "false"
}

type Num struct {
	v *big.Float
}

func (v *Num) Value() *big.Float {
	return v.v
}

func (v *Num) Copy() *Num {
	return NewNum(new(big.Float).Set(v.v))
}

func (v *Num) Pow(exp *Num) *Num {
	return NewNum(bigfloat.Pow(v.v, exp.v))
}

func (v *Num) Add(other *Num) {
	v.v.Add(v.v, other.v)
}

func (v *Num) Neg() *Num {
	return NewNum(new(big.Float).Neg(v.v))
}

func (v *Num) IsZero() bool {
	n, acc := v.v.Int64()
	return n == 0 && acc == big.Exact
}

func (v *Num) IsInf() bool {
	return v.v.IsInf()
}

func (v *Num) Sign() int {
	return v.v.Sign()
}

func (v *Num) LessThan(than *Num) bool {
	return v.v.Cmp(than.v) == -1
}

func (v *Num) LessOrEqualTo(to *Num) bool {
	return v.v.Cmp(to.v) <= 0
}

func (v *Num) GreaterThan(than *Num) bool {
	return v.v.Cmp(than.v) == 1
}

func (v *Num) GreaterOrEqualTo(to *Num) bool {
	return v.v.Cmp(to.v) >= 0
}

func (v *Num) EqualTo(to *Num) bool {
	return v.v.Cmp(to.v) == 0
}

func (v *Num) Abs() *Num {
	return NewNum(new(big.Float).Abs(v.v))
}

func (v *Num) AsUInt64() (uint64, error) {
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

func (v *Num) AsInt64() (int64, error) {
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

func (v *Num) MemReader() io.Reader {
	prec := v.v.Prec()
	cap := 10 + prec
	repr := v.v.Append(make([]byte, 0, cap), 'g', int(prec))
	return &readerWithType{
		Type:   TypeNum,
		Parent: bytes.NewBuffer(repr),
	}
}

func (v *Num) Type() Type {
	return TypeNum
}

func (v *Num) String() string {
	return v.v.String()
}

type String struct {
	v string
}

func (v *String) String() string {
	return v.v
}

func (v *String) MemReader() io.Reader {
	return &readerWithType{
		Type:   TypeString,
		Parent: strings.NewReader(v.v),
	}
}

func (v *String) Type() Type {
	return TypeString
}

func (v *String) AsBytes() *Array {
	return Bytes([]byte(v.String()))
}

type Array struct {
	bmode bool
	v     []Iface
	bs    []byte
}

func (v *Array) Len() int {
	if v.bmode {
		return len(v.bs)
	}
	return len(v.v)
}

func (v *Array) Slice() ([]Iface, bool) {
	return v.v, !v.bmode
}

func (v *Array) Concat(other *Array) *Array {
	if v.bmode && other.bmode {
		bs := make([]byte, 0, len(v.bs)+len(other.bs))
		return Bytes(append(append(bs, v.bs...), other.bs...))
	}

	larr := v.v
	if v.bmode {
		larr = make([]Iface, 0, len(v.bs))
		for _, b := range v.bs {
			larr = append(larr, UInt(b))
		}
	}

	rarr := other.v
	if other.bmode {
		rarr = make([]Iface, 0, len(other.bs))
		for _, b := range other.bs {
			rarr = append(rarr, UInt(b))
		}
	}

	return NewArray(append(larr, rarr...))
}

func (v *Array) Bytes() ([]byte, bool) {
	return v.bs, v.bmode
}

func (v *Array) GetByte(idx int64) (byte, error) {
	if !v.bmode {
		return 0, errors.New("use Get() instead for generic array")
	}

	norm := idx
	if idx < 0 {
		norm = int64(len(v.bs)) + idx
	}

	if norm >= int64(len(v.bs)) {
		return 0, fmt.Errorf("index %d out of range", idx)
	}

	return v.bs[idx], nil
}

func (v *Array) Get(idx int64) (Iface, error) {
	if v.bmode {
		b, err := v.GetByte(idx)
		if err != nil {
			return nil, err
		}

		return UInt(b), nil
	}

	norm := idx
	if idx < 0 {
		norm = int64(len(v.v)) + idx
	}

	if norm >= int64(len(v.v)) {
		return nil, fmt.Errorf("index %d out of range", idx)
	}

	return v.v[norm], nil
}

func (v *Array) Append(el ...Iface) {
	v.v = append(v.v, el...)
}

func (v Array) MemReader() io.Reader {
	r := readerWithType{
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

func (v *Array) Type() Type {
	return TypeArray
}

func (v *Array) String() string {
	var sb strings.Builder
	sb.WriteByte('[')

	for i, el := range v.v {
		sb.WriteString(el.String())
		if i != len(v.v)-1 {
			sb.WriteString(", ")
		}
	}

	sb.WriteByte(']')
	return sb.String()
}

type Object struct {
	v    map[string]Iface
	keys map[string]Iface
}

func (v *Object) Items() (keys []Iface, vals []Iface) {
	keys = make([]Iface, 0, len(v.keys))
	vals = make([]Iface, 0, len(v.v))
	for s, k := range v.keys {
		keys = append(keys, k)
		vals = append(vals, v.v[s])
	}
	return keys, vals
}

func (v *Object) Get(key Iface) (val Iface, err error) {
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

func (obj *Object) Set(k, v Iface) error {
	kb, err := io.ReadAll(k.MemReader())
	if err != nil {
		return fmt.Errorf("%s is not hashable", k.Type())
	}

	obj.v[string(kb)] = v
	obj.keys[string(kb)] = k
	return nil
}

func (v *Object) IterFunc(it func(k, v Iface) (cont, brk bool)) {
	for k, val := range v.v {
		cont, brk := it(v.keys[k], val)
		if cont {
			continue
		}

		if brk {
			break
		}
	}
}

func (v *Object) Len() int {
	return len(v.v)
}

func (v *Object) MemReader() io.Reader {
	r := readerWithType{
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

func (v *Object) Type() Type {
	return TypeObject
}

func (v *Object) String() string {
	var sb strings.Builder
	sb.WriteByte('{')

	i := 0
	for k := range v.keys {
		key, val := v.keys[k], v.v[k]

		sb.WriteString(key.String() + ": " + val.String())
		if i != len(v.keys)-1 {
			sb.WriteString(", ")
		}

		i++
	}

	sb.WriteByte('}')
	return sb.String()
}

type Args []Iface

func (args *Args) Print(w io.Writer) {
	for _, arg := range *args {
		fmt.Fprint(w, arg.String())
	}
}

type Func struct {
	idents []string
	v      func(args Args) (Iface, error)
}

func (v *Func) Idents() []string {
	return v.idents
}

func (v *Func) Call(args Args) (Iface, error) {
	return v.v(args)
}

func (v *Func) MemReader() io.Reader {
	return memReaderFunc{}
}

func (v *Func) Type() Type {
	return TypeFunc
}

func (v *Func) String() string {
	return "function"
}

func DeepEqual(x, y Iface) bool {
	if x == nil {
		return y == nil
	} else if y == nil {
		return false
	}

	if x.Type() != y.Type() {
		return false
	}

	switch x.Type() {
	case TypeNone:
		return true
	case TypeBool:
		lb, rb := MustCast[*Bool](x), MustCast[*Bool](y)
		return lb.v == rb.v
	case TypeNum:
		lnum, rnum := MustCast[*Num](x), MustCast[*Num](y)
		return lnum.v.Cmp(rnum.v) == 0
	case TypeString:
		ls, rs := MustCast[*String](x), MustCast[*String](y)
		return ls.v == rs.v
	case TypeArray:
		larr, rarr := MustCast[*Array](x), MustCast[*Array](y)
		if len(larr.v) != len(rarr.v) {
			return false
		}

		for i := 0; i < len(larr.v); i++ {
			lv, rv := larr.v[i], rarr.v[i]
			if !DeepEqual(lv, rv) {
				return false
			}
		}

		return true
	case TypeObject:
		lobj, robj := MustCast[*Object](x), MustCast[*Object](y)
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

			if !DeepEqual(lv, rv) {
				return false
			}
		}

		return true
	case TypeFunc:
		return false
	}
	panic("is equal: unknown type " + x.Type().String())
}

func NewNone() *None {
	return &None{}
}

func NewBool(v bool) *Bool {
	return &Bool{v: v}
}

func NewNum(v *big.Float) *Num {
	return &Num{v: v}
}

func NewString(v string) *String {
	return &String{v: v}
}

func NewArray(v []Iface) *Array {
	return &Array{v: v}
}

func NewObject(keys []Iface, values []Iface) (*Object, error) {
	if len(keys) != len(values) {
		return nil, errors.New("the number of keys does not match the number of values")
	}
	m := make(map[string]Iface, len(keys))
	ks := make(map[string]Iface, len(keys))
	for i := 0; i < len(keys); i++ {
		k, v := keys[i], values[i]
		kb, err := io.ReadAll(k.MemReader())
		if err != nil {
			return nil, fmt.Errorf("read key mem: %w", err)
		}

		m[string(kb)] = v
		ks[string(kb)] = k
	}

	return &Object{v: m, keys: ks}, nil
}

func MustNewObject(keys []Iface, values []Iface) *Object {
	obj, err := NewObject(keys, values)
	if err != nil {
		panic("object constructor: " + err.Error())
	}
	return obj
}

func FromMap[S ~string](m map[S]Iface) *Object {
	keys, vals := make([]Iface, 0, len(m)), make([]Iface, 0, len(m))
	for ks, v := range m {
		keys = append(keys, NewString(string(ks)))
		vals = append(vals, v)
	}

	return MustNewObject(keys, vals)
}

func NewFunc(argIdents []string, v func(args Args) (Iface, error)) *Func {
	return &Func{idents: argIdents, v: v}
}

func Int[T ~int](v T) *Num {
	f := new(big.Float).SetInt64(int64(v))
	return &Num{v: f}
}

func UInt[T ~uint | ~byte](v T) *Num {
	f := new(big.Float).SetUint64(uint64(v))
	return &Num{v: f}
}

func Float[T float32 | float64](v T) *Num {
	f := new(big.Float).
		SetPrec(64).
		SetMode(big.ToNearestEven).
		SetFloat64(float64(v))
	return &Num{v: f}
}

func Inf() *Num {
	f := new(big.Float).SetInf(false)
	return &Num{v: f}
}

func NegInf() *Num {
	f := new(big.Float).SetInf(true)
	return &Num{v: f}
}

func True() *Bool {
	return NewBool(true)
}

func False() *Bool {
	return NewBool(false)
}

func Bytes(bs []byte) *Array {
	return &Array{
		bmode: true,
		bs:    bs,
	}
}
