package easylang

type Register uint32

const (
	RegisterReturn Register = iota
)

type varmapper struct {
	m map[string]Register
	i Register
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
	m map[Register]Variant
}

func NewVarScope() *VarScope {
	return &VarScope{
		r: varmapper{m: map[string]Register{}, i: 1}, // i = 0 reserved for return value
		m: map[Register]Variant{},
	}
}

func (scope *VarScope) setter(name Register) (func(v Variant), bool) {
	if _, ok := scope.GetVar(name); ok {
		return func(v Variant) { scope.DefineVar(name, v) }, true
	}

	return nil, false
}

func (scope *VarScope) SetReturn(v Variant) {
	scope.DefineVar(RegisterReturn, v)
}

func (scope *VarScope) GetReturn() Variant {
	v, ok := scope.GetVar(RegisterReturn)
	if !ok {
		return &VariantNone{}
	}

	return v
}

func (scope *VarScope) GetVar(r Register) (Variant, bool) {
	v, ok := scope.m[r]
	return v, ok
}

func (scope *VarScope) VarByName(name string) Variant {
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

func (scope *VarScope) DefineVar(r Register, value Variant) {
	scope.m[r] = value
}

type Vars struct {
	Global           *VarScope
	Locals           []*VarScope
	ParentBlockScope *VarScope

	debug       bool
	debugChilds []*Vars
}

func (vars *Vars) setter(name Register) (func(v Variant), bool) {
	for i := len(vars.Locals) - 1; i >= 0; i-- {
		local := vars.Locals[i]

		if setter, ok := local.setter(name); ok {
			return setter, ok
		}
	}

	return vars.Global.setter(name)
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

func (vars *Vars) SetReturn(v Variant) {
	if vars.ParentBlockScope != nil {
		vars.ParentBlockScope.SetReturn(v)
		return
	}

	vars.LastScope().SetReturn(v)
}

func (vars *Vars) GetVar(name Register) (Variant, bool) {
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

func (vars *Vars) GetScope(level int) *VarScope {
	return vars.Locals[level]
}

func (vars *Vars) DefineGlobalVariable(r Register, value Variant) {
	vars.Global.DefineVar(r, value)
}

func (vars *Vars) DefineVariable(r Register, value Variant) {
	if len(vars.Locals) == 0 {
		vars.Global.DefineVar(r, value)
		return
	}

	vars.LastScope().DefineVar(r, value)
}

func (vars *Vars) Register(name string) (*VarScope, Register) {
	if len(vars.Locals) == 0 {
		return vars.Global, vars.Global.r.Register(name)
	}

	for _, scope := range vars.Locals {
		r, ok := scope.LookupRegister(name)
		if ok {
			return scope, r
		}
	}

	return vars.LastScope(), vars.LastScope().r.Register(name)
}

func (vars *Vars) SetOrDefineVariable(name Register, value Variant) {
	if setter, ok := vars.setter(name); ok {
		setter(value)
		return
	}

	vars.DefineVariable(name, value)
}

func NewVars() *Vars {
	return &Vars{
		Global: NewVarScope(),
	}
}

func NewDebugVars() *Vars {
	vars := NewVars()
	vars.debug = true
	return vars
}
