package easylang

import (
	"fmt"
	"io"

	"github.com/alecthomas/participle/v2"
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

func New() (*Machine, error) {
	parser, err := participle.Build[ProgramFile](
		participle.Lexer(lexdef),
		participle.Elide("Comment", "Whitespace"),
	)
	if err != nil {
		return nil, fmt.Errorf("build parser: %w", err)
	}

	return &Machine{
		vars:   NewVars(),
		parser: parser,
	}, nil
}
