package easylang

import (
	"math/big"
	"testing"

	"github.com/alecthomas/participle/v2"
	"github.com/hikitani/easylang/variant"
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

func TestExprCode(t *testing.T) {
	parser, err := participle.Build[Expr](
		participle.Lexer(lexdef),
		participle.Elide("Comment", "Whitespace"),
	)
	require.NoError(t, err)

	tests := []struct {
		Name           string
		Input          string
		Expected       variant.Iface
		IsFunc         bool
		IsCompileError bool
		IsRuntimeError bool
		Vars           *Vars
	}{
		{
			Name:     "String",
			Input:    `"Hello\n\t\U0001f3b1WORLD"`,
			Expected: variant.NewString("Hello\n\t🎱WORLD"),
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
			Expected: variant.Int(7),
		},
		{
			Name:     "Number_Int_Neg",
			Input:    `-7`,
			Expected: variant.Int(-7),
		},
		{
			Name:     "Number_Int_Underscore",
			Input:    `10_000`,
			Expected: variant.Int(10_000),
		},
		{
			Name:     "Number_Int_Binary",
			Input:    `0b101010`,
			Expected: variant.Int(0b101010),
		},
		{
			Name:     "Number_Int_Octal",
			Input:    `0o01234567`,
			Expected: variant.Int(0o01234567),
		},
		{
			Name:     "Number_Int_Hex",
			Input:    `0xffaabb`,
			Expected: variant.Int(0xffaabb),
		},
		{
			Name:     "Number_Float_Inf",
			Input:    `inf`,
			Expected: variant.Inf(),
		},
		{
			Name:     "Number_Float_Neg_Inf",
			Input:    `-inf`,
			Expected: variant.NegInf(),
		},
		{
			Name:     "Number_Float",
			Input:    `1_000.0203_405`,
			Expected: variant.NewNum(mustFloat("1000.0203405")),
		},
		{
			Name:   "IsFunc",
			Input:  `|| => {}`,
			IsFunc: true,
		},
		{
			Name:     "Array_Empty",
			Input:    `[]`,
			Expected: variant.NewArray(nil),
		},
		{
			Name:  "Array_Filled",
			Input: `[1, 2, "hello", [1,]]`,
			Expected: variant.NewArray([]variant.Iface{
				variant.Int(1), variant.Int(2),
				variant.NewString("hello"), variant.NewArray([]variant.Iface{variant.Int(1)}),
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
			Expected: variant.MustNewObject(nil, nil),
		},
		{
			Name: "Object_Filled",
			Input: `{
				"hello": "world",
				111: [],
				[1, 2, 3]: {1: 2},
			}`,
			Expected: variant.MustNewObject(
				[]variant.Iface{
					variant.NewString("hello"),
					variant.Int(111),
					variant.NewArray([]variant.Iface{variant.Int(1), variant.Int(2), variant.Int(3)})},
				[]variant.Iface{
					variant.NewString("world"),
					variant.NewArray(nil),
					variant.MustNewObject([]variant.Iface{variant.Int(1)}, []variant.Iface{variant.Int(2)}),
				},
			),
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
			Expected: variant.NewNone(),
		},
		{
			Name:     "ConstBool_True",
			Input:    `true`,
			Expected: variant.True(),
		},
		{
			Name:     "ConstBool_False",
			Input:    `false`,
			Expected: variant.False(),
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
					m: map[Register]variant.Iface{
						1: variant.NewString("hello world"),
					},
				},
			},
			Expected: variant.NewString("hello world"),
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
			Expected: variant.Int(3),
		},
		{
			Name:     "BlockExpr_NoReturn",
			Input:    `block {}`,
			Expected: variant.NewNone(),
		},
		{
			Name:     "Func_Simple",
			Input:    `(|| => 1 + 3)()`,
			Expected: variant.Int(4),
		},
		{
			Name:     "Func_Simple_WithArgs",
			Input:    `(|a, b| => a + b)(1, 3)`,
			Expected: variant.Int(4),
		},
		{
			Name: "Func_WithBlock",
			Input: `(|| => {
				a = 1
				b = 2
				return a + b
			})()`,
			Expected: variant.Int(3),
		},
		{
			Name:     "Func_WithBlock_NoReturn",
			Input:    `(|| => {})()`,
			Expected: variant.NewNone(),
		},
		{
			Name: "Func_WithBlock_WithArgs",
			Input: `(|a, b| => {
				if a > b {
					return a
				}

				return b
			})(1, 3)`,
			Expected: variant.Int(3),
		},
		{
			Name:     "Unary_Neg",
			Input:    `-1`,
			Expected: variant.Int(-1),
		},
		{
			Name:     "Unary_Not",
			Input:    `not true`,
			Expected: variant.False(),
		},
		{
			Name:     "Primary_ArrayIndex",
			Input:    `[1, 2, 3][0]`,
			Expected: variant.Int(1),
		},
		{
			Name:     "Primary_ArrayIndexExpr",
			Input:    `[1, 2, 3][1 + 1]`,
			Expected: variant.Int(3),
		},
		{
			Name:     "Primary_ArrayIndex_Negative",
			Input:    `[1, 2, 3][-1]`,
			Expected: variant.Int(3),
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
			Expected: variant.NewString("hello"),
		},
		{
			Name:     "Primary_ObjectMultiIndex",
			Input:    `{1: {"foo": "hello"}}[1, "foo"]`,
			Expected: variant.NewString("hello"),
		},
		{
			Name:     "Primary_ObjectMultiIndexV2",
			Input:    `{1: {"foo": "hello"}}[1]["foo"]`,
			Expected: variant.NewString("hello"),
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
			Expected: variant.NewString("hello"),
		},
		{
			Name:     "Primary_Selector_AsString",
			Input:    `{"0foo": {"bar": "hello"}}."0foo".bar`,
			Expected: variant.NewString("hello"),
		},
		{
			Name:           "Primary_Selector_NotFound",
			Input:          `{"foo": {"bar": "hello"}}.bar`,
			IsRuntimeError: true,
		},
		{
			Name:     "Binary_CmpOp_Less",
			Input:    `1 < 2`,
			Expected: variant.True(),
		},
		{
			Name:     "Binary_CmpOp_Less_False",
			Input:    `2 < 1`,
			Expected: variant.False(),
		},
		{
			Name:           "Binary_CmpOp_LessInvalid_DiffType",
			Input:          `"1" < 1`,
			IsRuntimeError: true,
		},
		{
			Name:     "Binary_CmpOp_LessOrEq",
			Input:    `1 <= 2`,
			Expected: variant.True(),
		},
		{
			Name:     "Binary_CmpOp_LessOrEq_False",
			Input:    `2 <= 1`,
			Expected: variant.False(),
		},
		{
			Name:     "Binary_CmpOp_LessOrEq_Exact",
			Input:    `2 <= 2`,
			Expected: variant.True(),
		},
		{
			Name:           "Binary_CmpOp_LessOrEqInvalid_DiffType",
			Input:          `"1" <= 1`,
			IsRuntimeError: true,
		},
		{
			Name:     "Binary_CmpOp_Greater",
			Input:    `2 > 1`,
			Expected: variant.True(),
		},
		{
			Name:     "Binary_CmpOp_Greater_False",
			Input:    `1 > 2`,
			Expected: variant.False(),
		},
		{
			Name:           "Binary_CmpOp_GreaterInvalid_DiffType",
			Input:          `"1" > 1`,
			IsRuntimeError: true,
		},
		{
			Name:     "Binary_CmpOp_GreaterOrEq",
			Input:    `2 >= 1`,
			Expected: variant.True(),
		},
		{
			Name:     "Binary_CmpOp_GreaterOrEq_False",
			Input:    `1 >= 2`,
			Expected: variant.False(),
		},
		{
			Name:     "Binary_CmpOp_GreaterOrEq_Exact",
			Input:    `2 >= 2`,
			Expected: variant.True(),
		},
		{
			Name:           "Binary_CmpOp_GreaterOrEqInvalid_DiffType",
			Input:          `"1" >= 1`,
			IsRuntimeError: true,
		},
		{
			Name:     "Binary_CmpOp_EqNum",
			Input:    `2 == 2`,
			Expected: variant.True(),
		},
		{
			Name:     "Binary_CmpOp_EqNum_False",
			Input:    `1 == 2`,
			Expected: variant.False(),
		},
		{
			Name:     "Binary_CmpOp_EqString",
			Input:    `"hello" == "hello"`,
			Expected: variant.True(),
		},
		{
			Name:     "Binary_CmpOp_EqString_False",
			Input:    `"hello" == "world"`,
			Expected: variant.False(),
		},
		{
			Name:     "Binary_CmpOp_EqNone",
			Input:    `none == none`,
			Expected: variant.True(),
		},
		{
			Name:     "Binary_CmpOp_EqBool",
			Input:    `true == true`,
			Expected: variant.True(),
		},
		{
			Name:     "Binary_CmpOp_EqBool_False",
			Input:    `true == false`,
			Expected: variant.False(),
		},
		{
			Name:     "Binary_CmpOp_EqArray",
			Input:    `[1, "2", true] == [1, "2", true]`,
			Expected: variant.True(),
		},
		{
			Name:     "Binary_CmpOp_EqArray_False",
			Input:    `[1, "2", true] == [1, 0, true]`,
			Expected: variant.False(),
		},
		{
			Name:     "Binary_CmpOp_EqObject",
			Input:    `{1: "hello", "foo": {true: false}} == {1: "hello", "foo": {true: false}}`,
			Expected: variant.True(),
		},
		{
			Name:     "Binary_CmpOp_EqObject_False",
			Input:    `{1: "hello", "foo": {true: false}} == {}`,
			Expected: variant.False(),
		},

		{
			Name:     "Binary_CmpOp_NotEqNum",
			Input:    `2 != 2`,
			Expected: variant.False(),
		},
		{
			Name:     "Binary_CmpOp_NotEqNum_True",
			Input:    `1 != 2`,
			Expected: variant.True(),
		},
		{
			Name:     "Binary_CmpOp_NotEqString",
			Input:    `"hello" != "hello"`,
			Expected: variant.False(),
		},
		{
			Name:     "Binary_CmpOp_NotEqString_True",
			Input:    `"hello" != "world"`,
			Expected: variant.True(),
		},
		{
			Name:     "Binary_CmpOp_NotEqNone",
			Input:    `none != none`,
			Expected: variant.False(),
		},
		{
			Name:     "Binary_CmpOp_NotEqBool",
			Input:    `true != true`,
			Expected: variant.False(),
		},
		{
			Name:     "Binary_CmpOp_NotEqBool_True",
			Input:    `true != false`,
			Expected: variant.True(),
		},
		{
			Name:     "Binary_CmpOp_NotEqArray",
			Input:    `[1, "2", true] != [1, "2", true]`,
			Expected: variant.False(),
		},
		{
			Name:     "Binary_CmpOp_NotEqArray_True",
			Input:    `[1, "2", true] != [1, 0, true]`,
			Expected: variant.True(),
		},
		{
			Name:     "Binary_CmpOp_NotEqObject",
			Input:    `{1: "hello", "foo": {true: false}} != {1: "hello", "foo": {true: false}}`,
			Expected: variant.False(),
		},
		{
			Name:     "Binary_CmpOp_NotEqObject_True",
			Input:    `{1: "hello", "foo": {true: false}} != {}`,
			Expected: variant.True(),
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
			Expected: variant.NewString("helloworld"),
		},
		{
			Name:     "Binary_Concat_Array",
			Input:    `["hello"] + ["world"]`,
			Expected: variant.NewArray([]variant.Iface{variant.NewString("hello"), variant.NewString("world")}),
		},

		{
			Name:     "Binary_ArithOp_Add",
			Input:    `2 + 2`,
			Expected: variant.Int(4),
		},
		{
			Name:     "Binary_ArithOp_Add_Inf",
			Input:    `inf + inf`,
			Expected: variant.Inf(),
		},
		{
			Name:     "Binary_ArithOp_Add_InfAndNum",
			Input:    `inf + 111`,
			Expected: variant.Inf(),
		},
		{
			Name:           "Binary_ArithOp_Add_Invalid",
			Input:          `inf + -inf`,
			IsRuntimeError: true,
		},
		{
			Name:     "Binary_ArithOp_Sub",
			Input:    `2 - 2`,
			Expected: variant.Int(0),
		},
		{
			Name:     "Binary_ArithOp_Sub_Inf",
			Input:    `inf - -inf`,
			Expected: variant.Inf(),
		},
		{
			Name:     "Binary_ArithOp_Sub_InfAndNum",
			Input:    `inf - 111`,
			Expected: variant.Inf(),
		},
		{
			Name:           "Binary_ArithOp_Sub_Invalid",
			Input:          `inf - inf`,
			IsRuntimeError: true,
		},
		{
			Name:     "Binary_ArithOp_Quo",
			Input:    `2 / 2`,
			Expected: variant.Int(1),
		},
		{
			Name:     "Binary_ArithOp_Quo_Inf",
			Input:    `2 / 0`,
			Expected: variant.Inf(),
		},
		{
			Name:     "Binary_ArithOp_Quo_NegInfInf",
			Input:    `-2 / 0`,
			Expected: variant.NegInf(),
		},
		{
			Name:     "Binary_ArithOp_Quo_Zero",
			Input:    `1 / inf`,
			Expected: variant.Int(0),
		},
		{
			Name:     "Binary_ArithOp_Quo_InfIntoNum",
			Input:    `inf / 999`,
			Expected: variant.Inf(),
		},
		{
			Name:     "Binary_ArithOp_Quo_NegInfIntoNum",
			Input:    `-inf / 999`,
			Expected: variant.NegInf(),
		},
		{
			Name:           "Binary_ArithOp_Quo_Invalid_ZeroIntoZero",
			Input:          `0 / 0`,
			IsRuntimeError: true,
		},
		{
			Name:           "Binary_ArithOp_Quo_Invalid_InfIntoInf",
			Input:          `inf / inf`,
			IsRuntimeError: true,
		},
		{
			Name:     "Binary_ArithOp_Mul",
			Input:    `2 * 3`,
			Expected: variant.Int(6),
		},
		{
			Name:     "Binary_ArithOp_Mul_Inf",
			Input:    `2 * inf`,
			Expected: variant.Inf(),
		},
		{
			Name:     "Binary_ArithOp_Mul_NegInf",
			Input:    `2 * -inf`,
			Expected: variant.NegInf(),
		},
		{
			Name:           "Binary_ArithOp_Mul_Invalid_ZeroAndInf",
			Input:          `inf * 0`,
			IsRuntimeError: true,
		},
		{
			Name:     "Binary_ArithOp_Mod_Int",
			Input:    `4 % 3`,
			Expected: variant.Int(1),
		},
		{
			Name:     "Binary_ArithOp_Mod_Int_NegX",
			Input:    `-4 % 3`,
			Expected: variant.Int(2),
		},
		{
			Name:     "Binary_ArithOp_Mod_Int_NegY",
			Input:    `4 % -3`,
			Expected: variant.Int(1),
		},
		{
			Name:     "Binary_ArithOp_Mod_Int_NegXY",
			Input:    `-4 % -3`,
			Expected: variant.Int(2),
		},
		{
			Name:     "Binary_ArithOp_Mod_Int_Inf",
			Input:    `inf % 4`,
			Expected: variant.Inf(),
		},
		{
			Name:     "Binary_ArithOp_Mod_Int_NegInf",
			Input:    `-inf % 4`,
			Expected: variant.NegInf(),
		},
		{
			Name:           "Binary_ArithOp_Mod_Int_InvalidInf",
			Input:          `4 % inf`,
			IsRuntimeError: true,
		},
		{
			Name:           "Binary_ArithOp_Mod_Int_InvalidZero",
			Input:          `4 % 0`,
			IsRuntimeError: true,
		},
		{
			Name: "Binary_ArithOp_Mod_Float",
			Input: `block {
				mod = 0.4 % 0.3
				expected_res = 0.1
				diff = mod - expected_res
				if diff < 0 {
					diff = -diff
				}

				return diff < 0.000_000_000_000_000_01
			}`,
			Expected: variant.True(),
		},
		{
			Name: "Binary_ArithOp_Mod_Float_NegX",
			Input: `block {
				mod = -0.4 % 0.3
				expected_res = 0.2
				diff = mod - expected_res
				if diff < 0 {
					diff = -diff
				}

				return diff < 0.000_000_000_000_000_01
			}`,
			Expected: variant.True(),
		},
		{
			Name: "Binary_ArithOp_Mod_Float_NegY",
			Input: `block {
				mod = 0.4 % -0.3
				expected_res = 0.1
				diff = mod - expected_res
				if diff < 0 {
					diff = -diff
				}

				return diff < 0.000_000_000_000_000_01
			}`,
			Expected: variant.True(),
		},
		{
			Name: "Binary_ArithOp_Mod_Float_NegXY",
			Input: `block {
				mod = -0.4 % -0.3
				expected_res = 0.2
				diff = mod - expected_res
				if diff < 0 {
					diff = -diff
				}

				return diff < 0.000_000_000_000_000_01
			}`,
			Expected: variant.True(),
		},
		{
			Name:     "Binary_ArithOp_Mod_Int_Inf",
			Input:    `inf % 4`,
			Expected: variant.Inf(),
		},
		{
			Name:     "Binary_ArithOp_Mod_Int_NegInf",
			Input:    `-inf % 4`,
			Expected: variant.NegInf(),
		},
		{
			Name:           "Binary_ArithOp_Mod_Float_InvalidZero",
			Input:          `4.123 % 0`,
			IsRuntimeError: true,
		},
		{
			Name:           "Binary_ArithOp_Mod_Float_InvalidInf",
			Input:          `4.123 % inf`,
			IsRuntimeError: true,
		},

		{
			Name:     "Binary_PredicateOp_And_True",
			Input:    `true and true`,
			Expected: variant.True(),
		},
		{
			Name:     "Binary_PredicateOp_And_False",
			Input:    `false and true`,
			Expected: variant.False(),
		},
		{
			Name:     "Binary_PredicateOp_Or_True",
			Input:    `(true or true) and (true or false) and (false or true)`,
			Expected: variant.True(),
		},
		{
			Name:     "Binary_PredicateOp_Or_False",
			Input:    `false and false`,
			Expected: variant.False(),
		},

		{
			Name: "Binary_Priority",

			/*
				Order:
				1. 2 * 2 = 4
				2. 4 % 3 = 1
				3. 1 * 2 = 2
				4. 2 / 2 = 1
				5. 4 - 1 = 3
				6. 3 + 1 = 4
				7. 4 == 4 = true
				8. true and true = true
				9. false or true = true
			*/
			Input:    `false or 2 * 2 - 4 % 3 * 2 / 2 + 1 == 4 and true`,
			Expected: variant.True(),
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
			_, ok := v.(*variant.Func)
			assert.True(t, ok, testCase.Name)
		} else {
			assert.True(t, variant.DeepEqual(testCase.Expected, v), testCase.Name)
		}
	}
}

func TestStmtCode(t *testing.T) {
	parser, err := participle.Build[ProgramFile](
		participle.Lexer(lexdef),
		participle.Elide("Comment", "Whitespace"),
	)
	require.NoError(t, err)

	tests := []struct {
		Name           string
		Input          string
		IsCompileError bool
		IsRuntimeError bool
		ExpectedVar    func(name string, is *assert.Assertions, vars *Vars)
	}{
		{
			Name:  "Stmt_Assign",
			Input: `foo = "hello"`,
			ExpectedVar: func(name string, is *assert.Assertions, vars *Vars) {
				r, ok := vars.Global.LookupRegister("foo")
				if !ok {
					is.Fail("register foo not found", name)
					return
				}

				v, ok := vars.Global.GetVar(r)
				if !ok {
					is.Fail("var foo not found", name)
					return
				}

				s, ok := v.(*variant.String)
				if !ok {
					is.Fail("var foo is not string", name)
					return
				}

				is.Equal(s.String(), "hello")
			},
		},
		{
			Name: "Stmt_Assign_Augmented",
			Input: `
				foo = "hello"
				foo += " world"
			`,
			ExpectedVar: func(name string, is *assert.Assertions, vars *Vars) {
				r, ok := vars.Global.LookupRegister("foo")
				if !ok {
					is.Fail("register foo not found", name)
					return
				}

				v, ok := vars.Global.GetVar(r)
				if !ok {
					is.Fail("var foo not found", name)
					return
				}

				s, ok := v.(*variant.String)
				if !ok {
					is.Fail("var foo is not string", name)
					return
				}

				is.Equal(s.String(), "hello world")
			},
		},
		{
			Name: "Stmt_Assign_Augmented_NameNotDefined",
			Input: `
				foo += " world"
			`,
			IsCompileError: true,
		},
		{
			Name: "Stmt_Assign_Augmented_BadType",
			Input: `
				foo = 1
				foo += " world"
			`,
			IsRuntimeError: true,
		},
		{
			Name: "Stmt_If_Simple",
			Input: `
			a = 1
			if a > 0 {
				b = a + 1
			}`,
			ExpectedVar: func(name string, is *assert.Assertions, vars *Vars) {
				r, ok := vars.debugChilds[0].LastScope().LookupRegister("b")
				if !ok {
					is.Fail("register b not found", name)
					return
				}

				v, ok := vars.debugChilds[0].GetVar(r)
				if !ok {
					is.Fail("var b not found", name)
					return
				}

				b, ok := v.(*variant.Num)
				if !ok {
					is.Fail("var b is not num", name)
					return
				}

				is.True(variant.DeepEqual(b, variant.Int(2)))
			},
		},
		{
			Name: "Stmt_If_Else_True",
			Input: `
			a = 1
			if a > 0 {
				b = a + 1
			} else {
				b = a - 1
			}`,
			ExpectedVar: func(name string, is *assert.Assertions, vars *Vars) {
				r, ok := vars.debugChilds[0].LastScope().LookupRegister("b")
				if !ok {
					is.Fail("register b not found", name)
					return
				}

				v, ok := vars.debugChilds[0].GetVar(r)
				if !ok {
					is.Fail("var b not found", name)
					return
				}

				b, ok := v.(*variant.Num)
				if !ok {
					is.Fail("var b is not num", name)
					return
				}

				is.True(variant.DeepEqual(b, variant.Int(2)))
			},
		},
		{
			Name: "Stmt_If_Else_False",
			Input: `
			a = 1
			if a < 0 {
				b = a + 1
			} else {
				b = a - 1
			}`,
			ExpectedVar: func(name string, is *assert.Assertions, vars *Vars) {
				r, ok := vars.debugChilds[1].LastScope().LookupRegister("b")
				if !ok {
					is.Fail("register b not found", name)
					return
				}

				v, ok := vars.debugChilds[1].GetVar(r)
				if !ok {
					is.Fail("var b not found", name)
					return
				}

				b, ok := v.(*variant.Num)
				if !ok {
					is.Fail("var b is not num", name)
					return
				}

				is.True(variant.DeepEqual(b, variant.Int(0)))
			},
		},
		{
			Name: "Stmt_If_ElseIf",
			Input: `
			a = 1
			if a < 0 {
				b = a + 1
			} else if a >= 1 {
				b = a - 1
			}`,
			ExpectedVar: func(name string, is *assert.Assertions, vars *Vars) {
				r, ok := vars.debugChilds[1].LastScope().LookupRegister("b")
				if !ok {
					is.Fail("register b not found", name)
					return
				}

				v, ok := vars.debugChilds[1].GetVar(r)
				if !ok {
					is.Fail("var b not found", name)
					return
				}

				b, ok := v.(*variant.Num)
				if !ok {
					is.Fail("var b is not num", name)
					return
				}

				is.True(variant.DeepEqual(b, variant.Int(0)))
			},
		},
		{
			Name: "Stmt_Return_Block",
			Input: `
			a = block {
				return "hello"
			}`,
			ExpectedVar: func(name string, is *assert.Assertions, vars *Vars) {
				r, ok := vars.Global.LookupRegister("a")
				if !ok {
					is.Fail("register a not found", name)
					return
				}

				v, ok := vars.Global.GetVar(r)
				if !ok {
					is.Fail("var a not found", name)
					return
				}

				s, ok := v.(*variant.String)
				if !ok {
					is.Fail("var a is not string", name)
					return
				}

				is.Equal(s.String(), "hello")
			},
		},
		{
			Name: "Stmt_Return_Func",
			Input: `
			a = || => {
				return "hello"
			}()`,
			ExpectedVar: func(name string, is *assert.Assertions, vars *Vars) {
				r, ok := vars.Global.LookupRegister("a")
				if !ok {
					is.Fail("register a not found", name)
					return
				}

				v, ok := vars.Global.GetVar(r)
				if !ok {
					is.Fail("var a not found", name)
					return
				}

				s, ok := v.(*variant.String)
				if !ok {
					is.Fail("var a is not string", name)
					return
				}

				is.Equal(s.String(), "hello")
			},
		},
		{
			Name:           "Stmt_Return_Invalid_Global",
			Input:          `return 1`,
			IsCompileError: true,
		},
		{
			Name: "Stmt_While",
			Input: `
			i = 0
			while i < 10 {
				i = i + 1
			}`,
			ExpectedVar: func(name string, is *assert.Assertions, vars *Vars) {
				r, ok := vars.Global.LookupRegister("i")
				if !ok {
					is.Fail("register i not found", name)
					return
				}

				v, ok := vars.Global.GetVar(r)
				if !ok {
					is.Fail("var i not found", name)
					return
				}

				i, ok := v.(*variant.Num)
				if !ok {
					is.Fail("var i is not num", name)
					return
				}

				is.True(variant.DeepEqual(i, variant.Int(10)))
			},
		},
		{
			Name: "Stmt_While_Break",
			Input: `
			i = 0
			while true {
				if i == 10 {
					break
				}
				i = i + 1
			}`,
			ExpectedVar: func(name string, is *assert.Assertions, vars *Vars) {
				r, ok := vars.Global.LookupRegister("i")
				if !ok {
					is.Fail("register i not found", name)
					return
				}

				v, ok := vars.Global.GetVar(r)
				if !ok {
					is.Fail("var i not found", name)
					return
				}

				i, ok := v.(*variant.Num)
				if !ok {
					is.Fail("var i is not num", name)
					return
				}

				is.True(variant.DeepEqual(i, variant.Int(10)))
			},
		},
		{
			Name: "Stmt_While_Continue",
			Input: `
			i = 0
			s = 0
			while i < 10 {
				i = i + 1

				if i % 2 == 0 {
					continue
				}

				s = s + 1
			}`,
			ExpectedVar: func(name string, is *assert.Assertions, vars *Vars) {
				r, ok := vars.Global.LookupRegister("s")
				if !ok {
					is.Fail("register s not found", name)
					return
				}

				v, ok := vars.Global.GetVar(r)
				if !ok {
					is.Fail("var s not found", name)
					return
				}

				s, ok := v.(*variant.Num)
				if !ok {
					is.Fail("var s is not num", name)
					return
				}

				is.True(variant.DeepEqual(s, variant.Int(5)))
			},
		},
		{
			Name: "Stmt_WhileNested_Break",
			Input: `
			i = 0
			j = 0
			while i < 10 {
				while true {
					j = j + 2
					break
				}
				i = i + 1
			}`,
			ExpectedVar: func(name string, is *assert.Assertions, vars *Vars) {
				r, ok := vars.Global.LookupRegister("j")
				if !ok {
					is.Fail("register j not found", name)
					return
				}

				v, ok := vars.Global.GetVar(r)
				if !ok {
					is.Fail("var j not found", name)
					return
				}

				j, ok := v.(*variant.Num)
				if !ok {
					is.Fail("var j is not num", name)
					return
				}

				is.True(variant.DeepEqual(j, variant.Int(20)))
			},
		},
		{
			Name: "Stmt_For_Array_ByVal",
			Input: `
			s = 0
			for v in [1, 2, 3] {
				s = s + v
			}
			`,
			ExpectedVar: func(name string, is *assert.Assertions, vars *Vars) {
				r, ok := vars.Global.LookupRegister("s")
				if !ok {
					is.Fail("register s not found", name)
					return
				}

				v, ok := vars.Global.GetVar(r)
				if !ok {
					is.Fail("var s not found", name)
					return
				}

				s, ok := v.(*variant.Num)
				if !ok {
					is.Fail("var s is not num", name)
					return
				}

				is.True(variant.DeepEqual(s, variant.Int(6)), name)
			},
		},
		{
			Name: "Stmt_For_Array_ByValWithIdx",
			Input: `
			s = 0
			for i, v in [1, 2, 3] {
				s = s + v
			}
			`,
			ExpectedVar: func(name string, is *assert.Assertions, vars *Vars) {
				r, ok := vars.Global.LookupRegister("s")
				if !ok {
					is.Fail("register s not found", name)
					return
				}

				v, ok := vars.Global.GetVar(r)
				if !ok {
					is.Fail("var s not found", name)
					return
				}

				s, ok := v.(*variant.Num)
				if !ok {
					is.Fail("var s is not num", name)
					return
				}

				is.True(variant.DeepEqual(s, variant.Int(6)), name)
			},
		},
		{
			Name: "Stmt_For_Array_ByIdx",
			Input: `
			s = 0
			arr = [1, 2, 3]
			for i, _ in arr {
				s = s + arr[i]
			}
			`,
			ExpectedVar: func(name string, is *assert.Assertions, vars *Vars) {
				r, ok := vars.Global.LookupRegister("s")
				if !ok {
					is.Fail("register s not found", name)
					return
				}

				v, ok := vars.Global.GetVar(r)
				if !ok {
					is.Fail("var s not found", name)
					return
				}

				s, ok := v.(*variant.Num)
				if !ok {
					is.Fail("var s is not num", name)
					return
				}

				is.True(variant.DeepEqual(s, variant.Int(6)), name)
			},
		},
		{
			Name: "Stmt_For_Object_ByKey",
			Input: `
			s = 0
			obj = {
				"1": 1,
				"2": 2,
				"3": 3,
			}
			for k in obj {
				s = s + obj[k]
			}
			`,
			ExpectedVar: func(name string, is *assert.Assertions, vars *Vars) {
				r, ok := vars.Global.LookupRegister("s")
				if !ok {
					is.Fail("register s not found", name)
					return
				}

				v, ok := vars.Global.GetVar(r)
				if !ok {
					is.Fail("var s not found", name)
					return
				}

				s, ok := v.(*variant.Num)
				if !ok {
					is.Fail("var s is not num", name)
					return
				}

				is.True(variant.DeepEqual(s, variant.Int(6)), name)
			},
		},
		{
			Name: "Stmt_For_Object_ByVal",
			Input: `
			s = 0
			obj = {
				"1": 1,
				"2": 2,
				"3": 3,
			}
			for _, v in obj {
				s = s + v
			}
			`,
			ExpectedVar: func(name string, is *assert.Assertions, vars *Vars) {
				r, ok := vars.Global.LookupRegister("s")
				if !ok {
					is.Fail("register s not found", name)
					return
				}

				v, ok := vars.Global.GetVar(r)
				if !ok {
					is.Fail("var s not found", name)
					return
				}

				s, ok := v.(*variant.Num)
				if !ok {
					is.Fail("var s is not num", name)
					return
				}

				is.True(variant.DeepEqual(s, variant.Int(6)), name)
			},
		},
		{
			Name: "Stmt_For_Continue",
			Input: `
			s = 0
			for v in [1, 2, 3] {
				if v % 2 == 0 {
					continue
				}
				s = s + v
			}
			`,
			ExpectedVar: func(name string, is *assert.Assertions, vars *Vars) {
				r, ok := vars.Global.LookupRegister("s")
				if !ok {
					is.Fail("register s not found", name)
					return
				}

				v, ok := vars.Global.GetVar(r)
				if !ok {
					is.Fail("var s not found", name)
					return
				}

				s, ok := v.(*variant.Num)
				if !ok {
					is.Fail("var s is not num", name)
					return
				}

				is.True(variant.DeepEqual(s, variant.Int(4)), name)
			},
		},
		{
			Name: "Stmt_For_Break",
			Input: `
			s = 0
			for v in [1, 2, 3] {
				if v % 2 == 0 {
					break
				}
				s = s + v
			}
			`,
			ExpectedVar: func(name string, is *assert.Assertions, vars *Vars) {
				r, ok := vars.Global.LookupRegister("s")
				if !ok {
					is.Fail("register s not found", name)
					return
				}

				v, ok := vars.Global.GetVar(r)
				if !ok {
					is.Fail("var s not found", name)
					return
				}

				s, ok := v.(*variant.Num)
				if !ok {
					is.Fail("var s is not num", name)
					return
				}

				is.True(variant.DeepEqual(s, variant.Int(1)), name)
			},
		},
		{
			Name: "Stmt_ForNested_Break",
			Input: `
			s = 0
			for v in [1, 2, 3] {
				for v in [2, 1] {
					if v % 2 != 0 {
						break
					}

					s = s + v
				}
				s = s + v
			}
			`,
			ExpectedVar: func(name string, is *assert.Assertions, vars *Vars) {
				r, ok := vars.Global.LookupRegister("s")
				if !ok {
					is.Fail("register s not found", name)
					return
				}

				v, ok := vars.Global.GetVar(r)
				if !ok {
					is.Fail("var s not found", name)
					return
				}

				s, ok := v.(*variant.Num)
				if !ok {
					is.Fail("var s is not num", name)
					return
				}

				is.True(variant.DeepEqual(s, variant.Int(12)), name)
			},
		},
		{
			Name: "Stmt_ForNested_Continue",
			Input: `
			s = 0
			for v in [1, 2, 3] {
				for v in [1, 2] {
					if v % 2 != 0 {
						continue
					}

					s = s + v
				}
				s = s + v
			}
			`,
			ExpectedVar: func(name string, is *assert.Assertions, vars *Vars) {
				r, ok := vars.Global.LookupRegister("s")
				if !ok {
					is.Fail("register s not found", name)
					return
				}

				v, ok := vars.Global.GetVar(r)
				if !ok {
					is.Fail("var s not found", name)
					return
				}

				s, ok := v.(*variant.Num)
				if !ok {
					is.Fail("var s is not num", name)
					return
				}

				is.True(variant.DeepEqual(s, variant.Int(12)), name)
			},
		},
	}

	is := assert.New(t)
	for _, testCase := range tests {
		stmt, err := parser.ParseString("", testCase.Input)
		if err != nil {
			is.Fail(err.Error(), testCase.Name)
			continue
		}

		vars := NewDebugVars()
		invoker, err := (&Program{vars: vars}).CodeGen(stmt)
		if testCase.IsCompileError {
			assert.Error(t, err, testCase.Name)
			continue
		}

		if err != nil {
			is.Fail(err.Error(), testCase.Name)
			continue
		}

		if testCase.IsRuntimeError {
			assert.Error(t, invoker.Invoke(), testCase.Name)
			continue
		}

		if err := invoker.Invoke(); err != nil {
			is.Fail(err.Error(), testCase.Name)
			continue
		}

		testCase.ExpectedVar(testCase.Name, is, vars)
	}
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
