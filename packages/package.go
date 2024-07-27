package packages

import (
	"errors"
	"math/big"

	"github.com/hikitani/easylang/builtin"
	"github.com/hikitani/easylang/variant"
)

var builtinPkg = NewPackage("builtin").
	AddObjects(builtin.Objects()).
	Build()

type Register struct {
	packages map[string]Package
}

func (reg *Register) Get(name string) (Package, bool) {
	pkg, ok := reg.packages[name]
	return pkg, ok
}

func (reg *Register) Register(pkg Package) error {
	if pkg.Name() == "builtin" {
		if pkg != builtinPkg {
			return errors.New("package name 'builtin' is reserved")
		}

		return nil
	}

	if _, ok := reg.packages[pkg.Name()]; ok {
		return errors.New("package name '" + pkg.Name() + "' is already registered")
	}

	reg.packages[pkg.Name()] = pkg
	return nil
}

func NewRegister() *Register {
	return &Register{
		packages: map[string]Package{
			builtinPkg.Name(): builtinPkg,
		},
	}
}

type Constructor struct {
	name    string
	objects map[string]variant.Iface
}

func (p *Constructor) AddVariant(name string, obj variant.Iface) *Constructor {
	p.objects[name] = obj
	return p
}

func (p *Constructor) AddNone(name string) *Constructor {
	return p.AddVariant(name, variant.NewNone())
}

func (p *Constructor) AddBool(name string, v bool) *Constructor {
	return p.AddVariant(name, variant.NewBool(v))
}

func (p *Constructor) AddInt(name string, v int) *Constructor {
	return p.AddVariant(name, variant.Int(v))
}

func (p *Constructor) AddUInt(name string, v int) *Constructor {
	return p.AddVariant(name, variant.Int(v))
}

func (p *Constructor) AddFloat(name string, v float64) *Constructor {
	return p.AddVariant(name, variant.Float(v))
}

func (p *Constructor) AddInf(name string) *Constructor {
	return p.AddVariant(name, variant.Inf())
}

func (p *Constructor) AddNegInf(name string) *Constructor {
	return p.AddVariant(name, variant.NegInf())
}

func (p *Constructor) AddBigFloat(name string, v *big.Float) *Constructor {
	return p.AddVariant(name, variant.NewNum(v))
}

func (p *Constructor) AddBigInt(name string, v *big.Int) *Constructor {
	return p.AddVariant(name, variant.NewNum(new(big.Float).SetInt(v)))
}

func (p *Constructor) AddBigRat(name string, v *big.Rat) *Constructor {
	return p.AddVariant(name, variant.NewNum(new(big.Float).SetRat(v)))
}

func (p *Constructor) AddString(name string, v string) *Constructor {
	return p.AddVariant(name, variant.NewString(v))
}

func (p *Constructor) AddBytes(name string, v []byte) *Constructor {
	return p.AddVariant(name, variant.Bytes(v))
}

func (p *Constructor) AddArray(name string, v []variant.Iface) *Constructor {
	return p.AddVariant(name, variant.NewArray(v))
}

func (p *Constructor) AddMap(name string, v map[string]variant.Iface) *Constructor {
	keys := make([]variant.Iface, 0, len(v))
	vals := make([]variant.Iface, 0, len(v))

	for k, v := range v {
		keys = append(keys, variant.NewString(k))
		vals = append(vals, v)
	}

	return p.AddVariant(name, variant.MustNewObject(keys, vals))
}

func (p *Constructor) AddFunc(name string, fn func(args variant.Args) (variant.Iface, error)) *Constructor {
	return p.AddVariant(name, variant.NewFunc(nil, fn))
}

func (p *Constructor) AddObjects(m map[string]variant.Iface) *Constructor {
	for k, v := range m {
		p.AddVariant(k, v)
	}

	return p
}

func (p *Constructor) Name() string {
	return p.name
}

func (p *Constructor) Objects() map[string]variant.Iface {
	return p.objects
}

func (p *Constructor) Build() Package {
	return p
}

func NewPackage(name string) *Constructor {
	return &Constructor{
		name:    name,
		objects: map[string]variant.Iface{},
	}
}

type Package interface {
	Name() string
	Objects() map[string]variant.Iface
}
