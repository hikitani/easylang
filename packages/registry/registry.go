package registry

import (
	"errors"

	"github.com/hikitani/easylang/packages"
	"github.com/hikitani/easylang/packages/builtin"
	"github.com/hikitani/easylang/packages/iter"
)

type Registry struct {
	packages map[string]packages.Iface
}

func (reg *Registry) Get(name string) (packages.Iface, bool) {
	pkg, ok := reg.packages[name]
	return pkg, ok
}

func (reg *Registry) Register(pkg packages.Iface) error {
	if pkg.Name() == builtin.Package.Name() {
		if pkg != builtin.Package {
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

func New() *Registry {
	return &Registry{
		packages: map[string]packages.Iface{
			builtin.Package.Name(): builtin.Package,
			iter.Package.Name():    iter.Package,
		},
	}
}
