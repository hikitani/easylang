package easylang

import (
	"errors"

	"github.com/hikitani/easylang/builtin"
	"github.com/hikitani/easylang/variant"
)

type PackageRegister struct {
	packages map[string]Package
}

func (reg *PackageRegister) Register(pkg Package) error {
	if pkg.Name() == "builtin" {
		if _, ok := pkg.(*builtinPkg); !ok {
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
			"builtin": &builtinPkg{},
		},
	}
}

type Package interface {
	Name() string
	Objects() map[string]variant.Iface
}

var _ Package = &builtinPkg{}

type builtinPkg struct{}

// Name implements Package.
func (b *builtinPkg) Name() string {
	return "builtin"
}

// Objects implements Package.
func (b *builtinPkg) Objects() map[string]variant.Iface {
	return builtin.Objects()
}
