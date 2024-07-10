package easylang

import (
	"fmt"
	"io"
	"math/big"
	"testing"

	"github.com/alecthomas/participle/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustFloat(v string) *big.Float {
	r, _, err := big.ParseFloat(v, 0, 64, big.ToNearestEven)
	if err != nil {
		panic(err)
	}
	return r
}

func mustReprVar(v Variant) string {
	b, err := io.ReadAll(v.MemReader())
	if err != nil {
		panic(err)
	}

	return string(b)
}

func TestExprCode(t *testing.T) {
	parser, err := participle.Build[Expr](
		participle.Lexer(lexdef),
		participle.Elide("Comment", "Whitespace"),
	)
	require.NoError(t, err)

	tests := []struct {
		Name           string
		Input          string
		Expected       Variant
		IsFunc         bool
		IsCompileError bool
		IsRuntimeError bool
		Vars           *Vars
	}{
		{
			Name:     "String",
			Input:    `"Hello\n\t\U0001f3b1WORLD"`,
			Expected: NewVarString("Hello\n\t🎱WORLD"),
		},
		{
			Name:     "Number_Int",
			Input:    `007`,
			Expected: NewVarInt(7),
		},
		{
			Name:     "Number_Int_Neg",
			Input:    `-7`,
			Expected: NewVarInt(-7),
		},
		{
			Name:     "Number_Int_Underscore",
			Input:    `10_000`,
			Expected: NewVarInt(10_000),
		},
		{
			Name:     "Number_Int_Binary",
			Input:    `0b101010`,
			Expected: NewVarInt(0b101010),
		},
		{
			Name:     "Number_Int_Octal",
			Input:    `0o01234567`,
			Expected: NewVarInt(0o01234567),
		},
		{
			Name:     "Number_Int_Hex",
			Input:    `0xffaabb`,
			Expected: NewVarInt(0xffaabb),
		},
		{
			Name:     "Number_Float_Inf",
			Input:    `inf`,
			Expected: NewVarNum(new(big.Float).SetInf(false)),
		},
		{
			Name:     "Number_Float_Neg_Inf",
			Input:    `-inf`,
			Expected: NewVarNum(new(big.Float).SetInf(true)),
		},
		{
			Name:     "Number_Float",
			Input:    `1_000.0203_405`,
			Expected: NewVarNum(mustFloat("1000.0203405")),
		},
		{
			Name:     "Array_Empty",
			Input:    `[]`,
			Expected: NewVarArray(nil),
		},
		{
			Name:  "Array_Filled",
			Input: `[1, 2, "hello", [1,]]`,
			Expected: NewVarArray([]Variant{
				NewVarInt(1), NewVarInt(2),
				NewVarString("hello"), NewVarArray([]Variant{NewVarInt(1)}),
			}),
		},
		{
			Name:     "Object_Empty",
			Input:    `{}`,
			Expected: NewVarObject(nil),
		},
		{
			Name: "Object_Filled",
			Input: `{
				"hello": "world",
				111: [],
				[1, 2, 3]: {1: 2},
			}`,
			Expected: NewVarObject(map[string]Variant{
				mustReprVar(NewVarString("hello")): NewVarString("world"),
				mustReprVar(NewVarInt(111)):        NewVarArray(nil),
				mustReprVar(NewVarArray([]Variant{
					NewVarInt(1), NewVarInt(2), NewVarInt(3),
				})): NewVarObject(map[string]Variant{mustReprVar(NewVarInt(1)): NewVarInt(2)}),
			}),
		},
		{
			Name:     "ConstNone",
			Input:    `none`,
			Expected: NewVarNone(),
		},
		{
			Name:     "ConstBool_True",
			Input:    `true`,
			Expected: NewVarBool(true),
		},
		{
			Name:     "ConstBool_False",
			Input:    `false`,
			Expected: NewVarBool(false),
		},
		{
			Name:  "Var",
			Input: `foo`,
			Vars: &Vars{Global: &VarScope{
				r: varmapper{
					m: map[string]Register{
						"foo": 1,
					},
				},
				m: map[Register]Variant{
					1: NewVarString("hello world")},
			},
			},
			Expected: NewVarString("hello world"),
		},
	}

	for _, testCase := range tests {
		expr, err := parser.ParseString("", testCase.Input)
		if err != nil {
			assert.Fail(t, err.Error(), testCase.Name)
			continue
		}

		vars := testCase.Vars
		if vars == nil {
			vars = NewDebugVars()
		}

		eval, err := (&ExprCodeGen{vars: vars}).CodeGen(expr)
		if testCase.IsCompileError {
			assert.Error(t, err, testCase.Name)
			continue
		}

		if !assert.NoError(t, err, testCase.Name) {
			continue
		}

		v, err := eval.Eval()
		if testCase.IsRuntimeError {
			assert.Error(t, err, testCase.Name)
			continue
		}

		if testCase.IsFunc {
			_, ok := v.(*VariantFunc)
			assert.Equal(t, ok, testCase.Name)
		} else {
			assert.True(t, VariantsIsDeepEqual(testCase.Expected, v), testCase.Name)
		}
	}
}

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

	codegen.exprGen.vars.Global.r.Register("foo")
	fmt.Println(codegen.exprGen.vars.Global.VarByName("foo").(*VariantString).v)
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
