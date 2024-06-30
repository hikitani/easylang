package easylang

type VarScope struct {
	m map[string]Variant
}

func NewVarScope() *VarScope {
	return &VarScope{
		m: map[string]Variant{},
	}
}

func (scope *VarScope) setter(name string) (func(v Variant), bool) {
	if _, ok := scope.GetVar(name); ok {
		return func(v Variant) { scope.DefineVar(name, v) }, true
	}

	return nil, false
}

func (scope *VarScope) SetReturn(v Variant) {
	scope.DefineVar("@r", v)
}

func (scope *VarScope) GetReturn() Variant {
	v, ok := scope.GetVar("@r")
	if !ok {
		return &VariantNone{}
	}

	return v
}

func (scope *VarScope) GetVar(name string) (Variant, bool) {
	v, ok := scope.m[name]
	return v, ok
}

func (scope *VarScope) DefineVar(name string, value Variant) {
	scope.m[name] = value
}

type Vars struct {
	Global *VarScope
	Locals []*VarScope

	debug       bool
	debugChilds []*Vars
}

func (vars *Vars) setter(name string) (func(v Variant), bool) {
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
		Global: vars.Global,
		Locals: locals,
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

func (vars *Vars) GetVar(name string) (Variant, bool) {
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

func (vars *Vars) DefineGlobalVariable(name string, value Variant) {
	vars.Global.DefineVar(name, value)
}

func (vars *Vars) DefineVariable(name string, value Variant) {
	if len(vars.Locals) == 0 {
		vars.Global.DefineVar(name, value)
		return
	}

	vars.LastScope().DefineVar(name, value)
	return
}

func (vars *Vars) SetOrDefineVariable(name string, value Variant) {
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
