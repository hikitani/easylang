package easylang

import (
	"fmt"
	"io"

	"github.com/alecthomas/participle/v2"
)

var parser = participle.MustBuild[ProgramFile](
	participle.Lexer(lexdef),
	participle.Elide("Comment", "Whitespace"),
)

type Machine struct {
	vars   *Vars
	parser *participle.Parser[ProgramFile]
}

func (m *Machine) Compile(f io.Reader) (StmtInvoker, error) {
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
		vars:   NewVars(),
		parser: parser,
	}
}
