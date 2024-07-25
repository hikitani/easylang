package easylang

import (
	"errors"
	"math/big"

	"github.com/hikitani/easylang/builtin"
	"github.com/hikitani/easylang/variant"
)

type PackageRegister struct {
	packages map[string]Package
}

func (reg *PackageRegister) Register(pkg Package) error {
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

func NewPackageRegister() *PackageRegister {
	return &PackageRegister{
		packages: map[string]Package{
			builtinPkg.Name(): builtinPkg,
		},
	}
}

type PackageConstructor struct {
	name    string
	objects map[string]variant.Iface
}

func (p *PackageConstructor) AddVariant(name string, obj variant.Iface) *PackageConstructor {
	p.objects[name] = obj
	return p
}

func (p *PackageConstructor) AddNone(name string) *PackageConstructor {
	return p.AddVariant(name, variant.NewNone())
}

func (p *PackageConstructor) AddBool(name string, v bool) *PackageConstructor {
	return p.AddVariant(name, variant.NewBool(v))
}

func (p *PackageConstructor) AddInt(name string, v int) *PackageConstructor {
	return p.AddVariant(name, variant.Int(v))
}

func (p *PackageConstructor) AddUInt(name string, v int) *PackageConstructor {
	return p.AddVariant(name, variant.Int(v))
}

func (p *PackageConstructor) AddFloat(name string, v float64) *PackageConstructor {
	return p.AddVariant(name, variant.Float(v))
}

func (p *PackageConstructor) AddInf(name string) *PackageConstructor {
	return p.AddVariant(name, variant.Inf())
}

func (p *PackageConstructor) AddNegInf(name string) *PackageConstructor {
	return p.AddVariant(name, variant.NegInf())
}

func (p *PackageConstructor) AddBigFloat(name string, v *big.Float) *PackageConstructor {
	return p.AddVariant(name, variant.NewNum(v))
}

func (p *PackageConstructor) AddBigInt(name string, v *big.Int) *PackageConstructor {
	return p.AddVariant(name, variant.NewNum(new(big.Float).SetInt(v)))
}

func (p *PackageConstructor) AddBigRat(name string, v *big.Rat) *PackageConstructor {
	return p.AddVariant(name, variant.NewNum(new(big.Float).SetRat(v)))
}

func (p *PackageConstructor) AddString(name string, v string) *PackageConstructor {
	return p.AddVariant(name, variant.NewString(v))
}

func (p *PackageConstructor) AddBytes(name string, v []byte) *PackageConstructor {
	return p.AddVariant(name, variant.Bytes(v))
}

func (p *PackageConstructor) AddArray(name string, v []variant.Iface) *PackageConstructor {
	return p.AddVariant(name, variant.NewArray(v))
}

func (p *PackageConstructor) AddMap(name string, v map[string]variant.Iface) *PackageConstructor {
	keys := make([]variant.Iface, 0, len(v))
	vals := make([]variant.Iface, 0, len(v))

	for k, v := range v {
		keys = append(keys, variant.NewString(k))
		vals = append(vals, v)
	}

	return p.AddVariant(name, variant.MustNewObject(keys, vals))
}

func (p *PackageConstructor) AddFunc(name string, fn func(args variant.Args) (variant.Iface, error)) *PackageConstructor {
	return p.AddVariant(name, variant.NewFunc(nil, fn))
}

func (p *PackageConstructor) AddObjects(m map[string]variant.Iface) *PackageConstructor {
	for k, v := range m {
		p.AddVariant(k, v)
	}

	return p
}

func (p *PackageConstructor) Name() string {
	return p.name
}

func (p *PackageConstructor) Objects() map[string]variant.Iface {
	return p.objects
}

func (p *PackageConstructor) Build() Package {
	return p
}

func AddPackage(name string) *PackageConstructor {
	return &PackageConstructor{
		name:    name,
		objects: map[string]variant.Iface{},
	}
}

type Package interface {
	Name() string
	Objects() map[string]variant.Iface
}

var builtinPkg = AddPackage("builtin").
	AddObjects(builtin.Objects()).
	Build()
