package easylang

import (
	"fmt"
	"testing"

	"github.com/alecthomas/participle/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStringCodegen(t *testing.T) {
	parser, err := participle.Build[BasicLit](
		participle.Lexer(lexdef),
		participle.Elide("Comment", "Whitespace"),
	)
	require.NoError(t, err)

	lit, err := parser.ParseString("", `"Hello\n\t\U0001f3b1WORLD"`)
	require.NoError(t, err)

	eval, err := (&BasicLitCodeGen{}).CodeGen(lit)
	require.NoError(t, err)

	v, err := eval.Eval()
	require.NoError(t, err)
	assert.Equal(t, v.(*VariantString).v, "Hello\n\t🎱WORLD")
}

func TestNumberCodegen(t *testing.T) {
	parser, err := participle.Build[BasicLit](
		participle.Lexer(lexdef),
		participle.Elide("Comment", "Whitespace"),
	)
	require.NoError(t, err)

	lit, err := parser.ParseString("", `123456`)
	require.NoError(t, err)

	eval, err := (&BasicLitCodeGen{}).CodeGen(lit)
	require.NoError(t, err)

	v, err := eval.Eval()
	require.NoError(t, err)

	num := v.(*VariantNum)
	require.True(t, num.v.IsInt())
	n, _ := num.v.Int(nil)
	assert.Equal(t, n.Int64(), int64(123456))
}

func TestExprCodegen(t *testing.T) {
	parser, err := participle.Build[Expr](
		participle.Lexer(lexdef),
		participle.Elide("Comment", "Whitespace"),
	)
	require.NoError(t, err)

	expr, err := parser.ParseString("", `1 == 1 or 1 != 1`)
	require.NoError(t, err)

	eval, err := (&ExprCodeGen{}).CodeGen(expr)
	require.NoError(t, err)

	v, err := eval.Eval()
	require.NoError(t, err)

	num := v.(*VariantBool)
	fmt.Println(num.v)
}

func TestExprStmtCodegen(t *testing.T) {
	parser, err := participle.Build[ExprStmt](
		participle.Lexer(lexdef),
		participle.Elide("Comment", "Whitespace"),
	)
	require.NoError(t, err)

	expr, err := parser.ParseString("", `foo = "hello" + " world"`)
	require.NoError(t, err)

	codegen := &ExprStmtCodeGen{
		exprGen: &ExprCodeGen{vars: NewVars()},
	}
	invoker, err := codegen.CodeGen(expr)

	require.NoError(t, err)
	require.NoError(t, invoker.Invoke())

	fmt.Println(codegen.exprGen.vars.Global.m["foo"].(*VariantString).v)
}

func TestProgram(t *testing.T) {
	parser, err := participle.Build[ProgramFile](
		participle.Lexer(lexdef),
		participle.Elide("Comment", "Whitespace"),
	)
	require.NoError(t, err)

	ast, err := parser.ParseString("", `
		s = ""
		sum = 0
		for i, el in ["hello", "world"] {
			sum = sum + i
			s = s + " " + el
		}

	`)
	require.NoError(t, err)

	vars := NewDebugVars()
	program, err := (&Program{
		vars: vars,
	}).CodeGen(ast)

	require.NoError(t, err)
	err = program.Invoke()
	require.NoError(t, err)
}

func BenchmarkProgram(b *testing.B) {
	parser, err := participle.Build[ProgramFile](
		participle.Lexer(lexdef),
		participle.Elide("Comment", "Whitespace"),
	)
	require.NoError(b, err)

	ast, err := parser.ParseString("", `
		s = ""
		sum = 0
		for i, el in ["hello", "world"] {
			sum = sum + i
			s = s + " " + el
		}
	`)
	require.NoError(b, err)

	vars := NewDebugVars()
	program, err := (&Program{
		vars: vars,
	}).CodeGen(ast)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Invoke()
	}
}
