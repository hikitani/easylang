package easylang

import (
	"fmt"
	"io"
	"os"

	"github.com/alecthomas/participle/v2"
	"github.com/hikitani/easylang/lexer"
	"github.com/hikitani/easylang/packages/registry"
)

var parser = participle.MustBuild[ProgramFile](
	participle.Lexer(lexer.Definition()),
	participle.Elide(lexer.IgnoreTokens()...),
)

type Machine struct {
	vars     *Vars
	parser   *participle.Parser[ProgramFile]
	register *registry.Registry
}

func (m *Machine) Compile(filename string, f io.Reader) (StmtInvoker, error) {
	ast, err := m.parser.Parse(filename, f)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	invoker, err := (&Program{
		vars:     m.vars,
		register: m.register,
		imports: importsInfo{
			From:          os.DirFS("./"),
			ImportedPaths: map[string]struct{}{},
		},
	}).CodeGen(ast)
	if err != nil {
		return nil, fmt.Errorf("code gen: %w", err)
	}

	return invoker, nil
}

func New() *Machine {
	return &Machine{
		vars:     NewVars(),
		parser:   parser,
		register: registry.New(),
	}
}
