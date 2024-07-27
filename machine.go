package easylang

import (
	"fmt"
	"io"

	"github.com/alecthomas/participle/v2"
	"github.com/hikitani/easylang/lexer"
	"github.com/hikitani/easylang/packages"
)

var parser = participle.MustBuild[ProgramFile](
	participle.Lexer(lexer.Definition()),
	participle.Elide(lexer.IgnoreTokens()...),
)

type Machine struct {
	vars     *Vars
	parser   *participle.Parser[ProgramFile]
	register *packages.Register
}

func (m *Machine) Compile(f io.Reader) (StmtInvoker, error) {
	builtinPkg, ok := m.register.Get("builtin")
	if !ok {
		panic("builtin package not found")
	}

	for name, obj := range builtinPkg.Objects() {
		r := m.vars.Global.Register(name)
		m.vars.Global.DefineVar(r, obj)
	}

	ast, err := m.parser.Parse("", f)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	invoker, err := (&Program{vars: m.vars}).CodeGen(ast)
	if err != nil {
		return nil, fmt.Errorf("code gen: %w", err)
	}

	return invoker, nil
}

func New() *Machine {
	return &Machine{
		vars:     NewVars(),
		parser:   parser,
		register: packages.NewRegister(),
	}
}
