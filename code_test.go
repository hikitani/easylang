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
			Name:           "String_InvalidBackslash",
			Input:          `"hello\"`,
			IsCompileError: true,
		},
		{
			Name:           "String_InvalidUnicode_Expected4bytes",
			Input:          `"hello\u00"`,
			IsCompileError: true,
		},
		{
			Name:           "String_InvalidUnicode_Expected8bytes",
			Input:          `"hello\Uffffff"`,
			IsCompileError: true,
		},
		{
			Name:           "String_InvalidUnicode4_NotHex",
			Input:          `"hello\uzzzz"`,
			IsCompileError: true,
		},
		{
			Name:           "String_InvalidUnicode8_NotHex",
			Input:          `"hello\Uffzzffhh11"`,
			IsCompileError: true,
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
			Name:   "IsFunc",
			Input:  `|| => {}`,
			IsFunc: true,
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
			Name:           "Array_InvalidElement",
			Input:          `["\"]`,
			IsCompileError: true,
		},
		{
			Name:           "Array_InvalidElementEval",
			Input:          `[1 + "hello"]`,
			IsRuntimeError: true,
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
			Name: "Object_InvalidKey",
			Input: `{
				"\": 1,
			}`,
			IsCompileError: true,
		},
		{
			Name: "Object_InvalidKeyEval",
			Input: `{
				1 + "2": 1,
			}`,
			IsRuntimeError: true,
		},
		{
			Name: "Object_InvalidValue",
			Input: `{
				"foo": "\",
			}`,
			IsCompileError: true,
		},
		{
			Name: "Object_InvalidValueEval",
			Input: `{
				"foo": 1 + "2",
			}`,
			IsRuntimeError: true,
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
			Vars: &Vars{
				Global: &VarScope{
					r: varmapper{
						m: map[string]Register{
							"foo": 1,
						},
					},
					m: map[Register]Variant{
						1: NewVarString("hello world"),
					},
				},
			},
			Expected: NewVarString("hello world"),
		},
		{
			Name:           "Var_invalid_NotDefined",
			Input:          `foo`,
			IsRuntimeError: true,
		},
		{
			Name:           "Var_Invalid_IsKeyword",
			Input:          `return`,
			IsCompileError: true,
		},
		{
			Name: "BlockExpr",
			Input: `block {
				a = 1
				b = 2
				return a + b
			}`,
			Expected: NewVarInt(3),
		},
		{
			Name:     "BlockExpr_NoReturn",
			Input:    `block {}`,
			Expected: NewVarNone(),
		},
		{
			Name:     "Func_Simple",
			Input:    `(|| => 1 + 3)()`,
			Expected: NewVarInt(4),
		},
		{
			Name:     "Func_Simple_WithArgs",
			Input:    `(|a, b| => a + b)(1, 3)`,
			Expected: NewVarInt(4),
		},
		{
			Name: "Func_WithBlock",
			Input: `(|| => {
				a = 1
				b = 2
				return a + b
			})()`,
			Expected: NewVarInt(3),
		},
		{
			Name:     "Func_WithBlock_NoReturn",
			Input:    `(|| => {})()`,
			Expected: NewVarNone(),
		},
		{
			Name: "Func_WithBlock_WithArgs",
			Input: `(|a, b| => {
				if a > b {
					return a
				}

				return b
			})(1, 3)`,
			Expected: NewVarInt(3),
		},
		{
			Name:     "Unary_Neg",
			Input:    `-1`,
			Expected: NewVarInt(-1),
		},
		{
			Name:     "Unary_Not",
			Input:    `not true`,
			Expected: NewVarBool(false),
		},
		{
			Name:     "Primary_ArrayIndex",
			Input:    `[1, 2, 3][0]`,
			Expected: NewVarInt(1),
		},
		{
			Name:     "Primary_ArrayIndexExpr",
			Input:    `[1, 2, 3][1 + 1]`,
			Expected: NewVarInt(3),
		},
		{
			Name:     "Primary_ArrayIndex_Negative",
			Input:    `[1, 2, 3][-1]`,
			Expected: NewVarInt(3),
		},
		{
			Name:           "Primary_ArrayIndex_Multi",
			Input:          `[1, 2, 3][1, 2]`,
			IsRuntimeError: true,
		},
		{
			Name:           "Primary_ArrayIndex_InvalidElem",
			Input:          `[1, 2, 3]["\"]`,
			IsCompileError: true,
		},
		{
			Name:           "Primary_ArrayIndex_InvalidElemExpr",
			Input:          `[1, 2, 3][1 + "2"]`,
			IsRuntimeError: true,
		},
		{
			Name:           "Primary_ArrayIndex_InvalidElemType",
			Input:          `[1, 2, 3]["2"]`,
			IsRuntimeError: true,
		},
		{
			Name:           "Primary_ArrayIndex_OutOfRange",
			Input:          `[1, 2, 3][3]`,
			IsRuntimeError: true,
		},
		{
			Name:           "Primary_ArrayIndex_OutOfMaxInt64",
			Input:          `[1, 2, 3][9_223_372_036_854_775_807 + 1]`,
			IsRuntimeError: true,
		},
		{
			Name:           "Primary_ArrayIndex_OutOfMinInt64",
			Input:          `[1, 2, 3][-9_223_372_036_854_775_808 - 1]`,
			IsRuntimeError: true,
		},
		{
			Name:     "Primary_ObjectIndex",
			Input:    `{1: "hello"}[1]`,
			Expected: NewVarString("hello"),
		},
		{
			Name:     "Primary_ObjectMultiIndex",
			Input:    `{1: {"foo": "hello"}}[1, "foo"]`,
			Expected: NewVarString("hello"),
		},
		{
			Name:     "Primary_ObjectMultiIndexV2",
			Input:    `{1: {"foo": "hello"}}[1]["foo"]`,
			Expected: NewVarString("hello"),
		},
		{
			Name:           "Primary_ObjectIndex_InvalidElem",
			Input:          `{1: {"foo": "hello"}}["\"]`,
			IsCompileError: true,
		},
		{
			Name:           "Primary_ObjectIndex_InvalidElemExpr",
			Input:          `{1: {"foo": "hello"}}[1 + "2"]`,
			IsRuntimeError: true,
		},
		{
			Name:           "Primary_ObjectIndex_KeyNotFound",
			Input:          `{1: {"foo": "hello"}}[2]`,
			IsRuntimeError: true,
		},
		{
			Name:           "Primary_ObjectIndex_NestedKeyNotFound",
			Input:          `{1: {"foo": "hello"}}[1, "bar"]`,
			IsRuntimeError: true,
		},
		{
			Name:           "Primary_Call_InvalidType",
			Input:          `1()`,
			IsRuntimeError: true,
		},
		{
			Name:           "Primary_Call_InvalidArgExpr",
			Input:          `(|a| => a)(1 + "2")`,
			IsRuntimeError: true,
		},
		{
			Name:           "Primary_Call_InvalidLenArgs",
			Input:          `(|a| => a)(1, 2, 3)`,
			IsRuntimeError: true,
		},
		{
			Name:     "Primary_Selector",
			Input:    `{"foo": {"bar": "hello"}}.foo.bar`,
			Expected: NewVarString("hello"),
		},
		{
			Name:     "Primary_Selector_AsString",
			Input:    `{"0foo": {"bar": "hello"}}."0foo".bar`,
			Expected: NewVarString("hello"),
		},
		{
			Name:           "Primary_Selector_NotFound",
			Input:          `{"foo": {"bar": "hello"}}.bar`,
			IsRuntimeError: true,
		},
		{
			Name:     "Binary_CmpOp_Less",
			Input:    `1 < 2`,
			Expected: NewVarBool(true),
		},
		{
			Name:     "Binary_CmpOp_Less_False",
			Input:    `2 < 1`,
			Expected: NewVarBool(false),
		},
		{
			Name:           "Binary_CmpOp_LessInvalid_DiffType",
			Input:          `"1" < 1`,
			IsRuntimeError: true,
		},
		{
			Name:     "Binary_CmpOp_LessOrEq",
			Input:    `1 <= 2`,
			Expected: NewVarBool(true),
		},
		{
			Name:     "Binary_CmpOp_LessOrEq_False",
			Input:    `2 <= 1`,
			Expected: NewVarBool(false),
		},
		{
			Name:     "Binary_CmpOp_LessOrEq_Exact",
			Input:    `2 <= 2`,
			Expected: NewVarBool(true),
		},
		{
			Name:           "Binary_CmpOp_LessOrEqInvalid_DiffType",
			Input:          `"1" <= 1`,
			IsRuntimeError: true,
		},
		{
			Name:     "Binary_CmpOp_Greater",
			Input:    `2 > 1`,
			Expected: NewVarBool(true),
		},
		{
			Name:     "Binary_CmpOp_Greater_False",
			Input:    `1 > 2`,
			Expected: NewVarBool(false),
		},
		{
			Name:           "Binary_CmpOp_GreaterInvalid_DiffType",
			Input:          `"1" > 1`,
			IsRuntimeError: true,
		},
		{
			Name:     "Binary_CmpOp_GreaterOrEq",
			Input:    `2 >= 1`,
			Expected: NewVarBool(true),
		},
		{
			Name:     "Binary_CmpOp_GreaterOrEq_False",
			Input:    `1 >= 2`,
			Expected: NewVarBool(false),
		},
		{
			Name:     "Binary_CmpOp_GreaterOrEq_Exact",
			Input:    `2 >= 2`,
			Expected: NewVarBool(true),
		},
		{
			Name:           "Binary_CmpOp_GreaterOrEqInvalid_DiffType",
			Input:          `"1" >= 1`,
			IsRuntimeError: true,
		},
		{
			Name:     "Binary_CmpOp_EqNum",
			Input:    `2 == 2`,
			Expected: NewVarBool(true),
		},
		{
			Name:     "Binary_CmpOp_EqNum_False",
			Input:    `1 == 2`,
			Expected: NewVarBool(false),
		},
		{
			Name:     "Binary_CmpOp_EqString",
			Input:    `"hello" == "hello"`,
			Expected: NewVarBool(true),
		},
		{
			Name:     "Binary_CmpOp_EqString_False",
			Input:    `"hello" == "world"`,
			Expected: NewVarBool(false),
		},
		{
			Name:     "Binary_CmpOp_EqNone",
			Input:    `none == none`,
			Expected: NewVarBool(true),
		},
		{
			Name:     "Binary_CmpOp_EqBool",
			Input:    `true == true`,
			Expected: NewVarBool(true),
		},
		{
			Name:     "Binary_CmpOp_EqBool_False",
			Input:    `true == false`,
			Expected: NewVarBool(false),
		},
		{
			Name:     "Binary_CmpOp_EqArray",
			Input:    `[1, "2", true] == [1, "2", true]`,
			Expected: NewVarBool(true),
		},
		{
			Name:     "Binary_CmpOp_EqArray_False",
			Input:    `[1, "2", true] == [1, 0, true]`,
			Expected: NewVarBool(false),
		},
		{
			Name:     "Binary_CmpOp_EqObject",
			Input:    `{1: "hello", "foo": {true: false}} == {1: "hello", "foo": {true: false}}`,
			Expected: NewVarBool(true),
		},
		{
			Name:     "Binary_CmpOp_EqObject_False",
			Input:    `{1: "hello", "foo": {true: false}} == {}`,
			Expected: NewVarBool(false),
		},

		{
			Name:     "Binary_CmpOp_NotEqNum",
			Input:    `2 != 2`,
			Expected: NewVarBool(false),
		},
		{
			Name:     "Binary_CmpOp_NotEqNum_True",
			Input:    `1 != 2`,
			Expected: NewVarBool(true),
		},
		{
			Name:     "Binary_CmpOp_NotEqString",
			Input:    `"hello" != "hello"`,
			Expected: NewVarBool(false),
		},
		{
			Name:     "Binary_CmpOp_NotEqString_True",
			Input:    `"hello" != "world"`,
			Expected: NewVarBool(true),
		},
		{
			Name:     "Binary_CmpOp_NotEqNone",
			Input:    `none != none`,
			Expected: NewVarBool(false),
		},
		{
			Name:     "Binary_CmpOp_NotEqBool",
			Input:    `true != true`,
			Expected: NewVarBool(false),
		},
		{
			Name:     "Binary_CmpOp_NotEqBool_True",
			Input:    `true != false`,
			Expected: NewVarBool(true),
		},
		{
			Name:     "Binary_CmpOp_NotEqArray",
			Input:    `[1, "2", true] != [1, "2", true]`,
			Expected: NewVarBool(false),
		},
		{
			Name:     "Binary_CmpOp_NotEqArray_True",
			Input:    `[1, "2", true] != [1, 0, true]`,
			Expected: NewVarBool(true),
		},
		{
			Name:     "Binary_CmpOp_NotEqObject",
			Input:    `{1: "hello", "foo": {true: false}} != {1: "hello", "foo": {true: false}}`,
			Expected: NewVarBool(false),
		},
		{
			Name:     "Binary_CmpOp_NotEqObject_True",
			Input:    `{1: "hello", "foo": {true: false}} != {}`,
			Expected: NewVarBool(true),
		},
		{
			Name:           "Binary_CmpOp_EqInvalid_DiffType",
			Input:          `"1" == 1`,
			IsRuntimeError: true,
		},
		{
			Name:           "Binary_CmpOp_NotEqInvalid_DiffType",
			Input:          `"1" != 1`,
			IsRuntimeError: true,
		},

		{
			Name:     "Binary_Concat_String",
			Input:    `"hello" + "world"`,
			Expected: NewVarString("helloworld"),
		},
		{
			Name:     "Binary_Concat_Array",
			Input:    `["hello"] + ["world"]`,
			Expected: NewVarArray([]Variant{NewVarString("hello"), NewVarString("world")}),
		},

		{
			Name:     "Binary_ArithOp_Add",
			Input:    `2 + 2`,
			Expected: NewVarInt(4),
		},
		{
			Name:     "Binary_ArithOp_Add_Inf",
			Input:    `inf + inf`,
			Expected: NewVarInf(),
		},
		{
			Name:     "Binary_ArithOp_Add_InfAndNum",
			Input:    `inf + 111`,
			Expected: NewVarInf(),
		},
		{
			Name:           "Binary_ArithOp_Add_Invalid",
			Input:          `inf + -inf`,
			IsRuntimeError: true,
		},
		{
			Name:     "Binary_ArithOp_Sub",
			Input:    `2 - 2`,
			Expected: NewVarInt(0),
		},
		{
			Name:     "Binary_ArithOp_Sub_Inf",
			Input:    `inf - -inf`,
			Expected: NewVarInf(),
		},
		{
			Name:     "Binary_ArithOp_Sub_InfAndNum",
			Input:    `inf - 111`,
			Expected: NewVarInf(),
		},
		{
			Name:           "Binary_ArithOp_Sub_Invalid",
			Input:          `inf - inf`,
			IsRuntimeError: true,
		},
		{
			Name:     "Binary_ArithOp_Quo",
			Input:    `2 / 2`,
			Expected: NewVarInt(1),
		},
		{
			Name:     "Binary_ArithOp_Mul",
			Input:    `2 * 3`,
			Expected: NewVarInt(6),
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
			assert.True(t, ok, testCase.Name)
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
