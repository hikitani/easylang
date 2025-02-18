package easylang

import (
	"fmt"

	"github.com/hikitani/easylang/packages/builtin"
	"github.com/hikitani/easylang/variant"
)

type Register uint32

const (
	RegisterReturn Register = iota
)

type varmapper struct {
	m    map[string]Register
	pubs map[string]struct{}
	i    Register
}

func (v *varmapper) RegisterPub(name string) Register {
	v.pubs[name] = struct{}{}
	return v.Register(name)
}

func (v *varmapper) Register(name string) Register {
	reg, ok := v.m[name]
	if ok {
		return reg
	}

	v.m[name] = v.i
	defer func() { v.i++ }()
	return v.i
}

type VarScope struct {
	r varmapper
	m map[Register]variant.Iface
}

func NewVarScope() *VarScope {
	return &VarScope{
		r: varmapper{
			i:    1, // i = 0 reserved for return value
			m:    map[string]Register{},
			pubs: map[string]struct{}{},
		},
		m: map[Register]variant.Iface{},
	}
}

func (scope *VarScope) SetReturn(v variant.Iface) {
	scope.DefineVar(RegisterReturn, v)
}

func (scope *VarScope) GetReturn() variant.Iface {
	v, ok := scope.GetVar(RegisterReturn)
	if !ok {
		return &variant.None{}
	}

	return v
}

func (scope *VarScope) Register(name string) Register {
	return scope.r.Register(name)
}

func (scope *VarScope) RegisterPub(name string) Register {
	return scope.r.RegisterPub(name)
}

func (scope *VarScope) GetVar(r Register) (variant.Iface, bool) {
	v, ok := scope.m[r]
	return v, ok
}

func (scope *VarScope) VarByName(name string) variant.Iface {
	r, ok := scope.r.m[name]
	if !ok {
		panic("var '" + name + "' not found")
	}

	return scope.m[r]
}

func (scope *VarScope) LookupRegister(name string) (Register, bool) {
	r, ok := scope.r.m[name]
	return r, ok
}

func (scope *VarScope) IsPublic(name string) bool {
	_, ok := scope.r.pubs[name]
	return ok
}

func (scope *VarScope) DefineVar(r Register, value variant.Iface) {
	scope.m[r] = value
}

type Vars struct {
	Global           *VarScope
	Locals           []*VarScope
	ParentBlockScope *VarScope

	debug       bool
	debugChilds []*Vars
}

func (vars *Vars) WithScope() *Vars {
	locals := make([]*VarScope, len(vars.Locals)+1)
	copy(locals, vars.Locals)
	locals[len(locals)-1] = NewVarScope()
	child := &Vars{
		Global:           vars.Global,
		Locals:           locals,
		ParentBlockScope: vars.ParentBlockScope,
	}

	if vars.debug {
		vars.debugChilds = append(vars.debugChilds, child)
	}

	return child
}

func (vars *Vars) Unscope() *Vars {
	if len(vars.Locals) == 0 {
		panic("local vars not created, impossible to unscope")
	}

	locals := make([]*VarScope, len(vars.Locals)-1)
	copy(locals, vars.Locals)
	return &Vars{
		Global: vars.Global,
		Locals: locals,
	}
}

func (vars *Vars) SetReturn(v variant.Iface) {
	if vars.ParentBlockScope != nil {
		vars.ParentBlockScope.SetReturn(v)
		return
	}

	vars.LastScope().SetReturn(v)
}

func (vars *Vars) GetVar(name Register) (variant.Iface, bool) {
	for i := len(vars.Locals) - 1; i >= 0; i-- {
		local := vars.Locals[i]

		v, ok := local.GetVar(name)
		if ok {
			return v, ok
		}
	}

	return vars.Global.GetVar(name)
}

func (vars *Vars) LastScope() *VarScope {
	return vars.Locals[len(vars.Locals)-1]
}

func (vars *Vars) Register(name string) (*VarScope, Register) {
	if len(vars.Locals) == 0 {
		return vars.Global, vars.Global.r.Register(name)
	}

	for i := len(vars.Locals) - 1; i >= 0; i-- {
		scope := vars.Locals[i]
		r, ok := scope.LookupRegister(name)
		if ok {
			return scope, r
		}
	}

	if r, ok := vars.Global.LookupRegister(name); ok {
		return vars.Global, r
	}

	return vars.LastScope(), vars.LastScope().Register(name)
}

func (vars *Vars) RegisterPub(name string) (*VarScope, Register, error) {
	_, ok := vars.Global.LookupRegister(name)
	if !ok {
		r := vars.Global.RegisterPub(name)
		return vars.Global, r, nil
	}

	return nil, 0, fmt.Errorf("var '%s' already defined as pub", name)
}

func (vars *Vars) Published() *variant.Object {
	var keys, vals []variant.Iface
	for pubname := range vars.Global.r.pubs {
		keys = append(keys, variant.NewString(pubname))
		vals = append(vals, vars.Global.VarByName(pubname))
	}

	return variant.MustNewObject(keys, vals)
}

func (vars *Vars) LookupRegister(name string) (*VarScope, Register, bool) {
	for i := len(vars.Locals) - 1; i >= 0; i-- {
		scope := vars.Locals[i]
		r, ok := scope.LookupRegister(name)
		if ok {
			return scope, r, ok
		}
	}

	r, ok := vars.Global.LookupRegister(name)
	return vars.Global, r, ok
}

func NewVars() *Vars {
	vars := &Vars{
		Global: NewVarScope(),
	}

	for name, obj := range builtin.Package.Objects() {
		r := vars.Global.Register(name)
		vars.Global.DefineVar(r, obj)
	}

	return vars
}

func NewDebugVars() *Vars {
	vars := NewVars()
	vars.debug = true
	return vars
}
