package easylang

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptr[T any](v T) *T {
	return &v
}

func valOrZero[T any](p *T) (v T) {
	if p != nil {
		return *p
	}
	return
}

type testCases[T any] []struct {
	Code      string
	Expected  T
	IsInvalid bool
}

func TestBasicLit(t *testing.T) {
	parser, err := participle.Build[BasicLit](
		participle.Elide("Comment", "Whitespace"),
		participle.Lexer(lexdef),
	)
	require.NoError(t, err)

	testCases := testCases[BasicLit]{
		{
			Code: "0",
			Expected: BasicLit{
				Number: ptr("0"),
			},
		},
		{
			Code: "123324543",
			Expected: BasicLit{
				Number: ptr("123324543"),
			},
		},
		{
			Code: "0b0010101010",
			Expected: BasicLit{
				Number: ptr("0b0010101010"),
			},
		},
		{
			Code: "0B0010101010",
			Expected: BasicLit{
				Number: ptr("0B0010101010"),
			},
		},
		{
			Code: "0o777",
			Expected: BasicLit{
				Number: ptr("0o777"),
			},
		},
		{
			Code: "0O777",
			Expected: BasicLit{
				Number: ptr("0O777"),
			},
		},
		{
			Code: "0xfff",
			Expected: BasicLit{
				Number: ptr("0xfff"),
			},
		},
		{
			Code: "0xFFF",
			Expected: BasicLit{
				Number: ptr("0xFFF"),
			},
		},
		{
			Code: "0Xfff",
			Expected: BasicLit{
				Number: ptr("0Xfff"),
			},
		},
		{
			Code: "0XFFF",
			Expected: BasicLit{
				Number: ptr("0XFFF"),
			},
		},
		{
			Code: `0123`,
			Expected: BasicLit{
				Number: ptr(`0123`),
			},
		},
		{
			Code: `""`,
			Expected: BasicLit{
				String: ptr(`""`),
			},
		},
		{
			Code: `"hello"`,
			Expected: BasicLit{
				String: ptr(`"hello"`),
			},
		},
		{
			Code: "\"hello\nworld\"",
			Expected: BasicLit{
				String: ptr("\"hello\nworld\""),
			},
		},
		{
			Code:      `hello`,
			IsInvalid: true,
		},
		{
			Code:      `0b222`,
			IsInvalid: true,
		},
	}

	is := assert.New(t)
	for i, testCase := range testCases {
		actual, err := parser.ParseString("", testCase.Code)
		if testCase.IsInvalid {
			is.Error(err, i)
			continue
		}

		isEq(t, testCase.Expected, actual)
		is.NoError(err, i)
	}
}

func TestExpr(t *testing.T) {
	parser, err := participle.Build[Expr](
		participle.Lexer(lexdef),
		participle.Elide("Comment", "Whitespace"),
	)
	require.NoError(t, err)

	testCases := testCases[Expr]{
		{
			Code: `(
				goo(1)
			).buz()[
				"334",
			]`,
			Expected: Expr{UnaryExpr: UnaryExpr{Operand: Operand{
				ParenExpr: &Expr{
					UnaryExpr: UnaryExpr{
						Operand: Operand{
							Name: &Ident{Name: "goo"},
							PX: &PrimaryExpr{CallExpr: &CallExpr{Args: &List[Expr]{X: []*Expr{
								{
									UnaryExpr: UnaryExpr{Operand: Operand{
										Literal: &Literal{Basic: &BasicLit{
											Number: ptr("1"),
										}},
									}},
								},
							}}}},
						},
					},
				},

				PX: &PrimaryExpr{SelectorExpr: &SelectorExpr{
					Sel: []SelectorExprPiece{{
						Ident: &Ident{Name: "buz"},
					}},
					PX: &PrimaryExpr{CallExpr: &CallExpr{
						PX: &PrimaryExpr{IndexExpr: &IndexExpr{Index: &List[Expr]{X: []*Expr{
							{
								UnaryExpr: UnaryExpr{Operand: Operand{Literal: &Literal{
									Basic: &BasicLit{String: ptr(`"334"`)},
								}}},
							},
						}}}},
					}},
				}},
			}}},
		},
		{
			Code: `12 + (34 / 3)`,
			Expected: Expr{
				UnaryExpr: UnaryExpr{Operand: Operand{Literal: &Literal{Basic: &BasicLit{
					Number: ptr("12"),
				}}}},
				BinaryExpr: &BinaryExpr{
					Op: "+",
					X: UnaryExpr{Operand: Operand{ParenExpr: &Expr{
						UnaryExpr: UnaryExpr{Operand: Operand{Literal: &Literal{Basic: &BasicLit{
							Number: ptr("34"),
						}}}},
						BinaryExpr: &BinaryExpr{
							Op: "/",
							X: UnaryExpr{Operand: Operand{Literal: &Literal{Basic: &BasicLit{
								Number: ptr("3"),
							}}}},
						},
					}}},
				},
			},
		},
		{
			Code: `{
				12 + 22: [1, "2", 3],
				"12": 234,
			}`,
			Expected: Expr{UnaryExpr: UnaryExpr{Operand: Operand{Literal: &Literal{Composite: &CompositeLit{
				ObjectLit: &ObjectLit{Items: &List[KeyValueExpr]{X: []*KeyValueExpr{
					{
						Key: Expr{
							UnaryExpr: UnaryExpr{Operand: Operand{Literal: &Literal{Basic: &BasicLit{
								Number: ptr("12"),
							}}}},
							BinaryExpr: &BinaryExpr{
								Op: "+",
								X: UnaryExpr{Operand: Operand{Literal: &Literal{Basic: &BasicLit{
									Number: ptr("22"),
								}}}},
							},
						},
						Value: Expr{UnaryExpr: UnaryExpr{Operand: Operand{Literal: &Literal{
							Composite: &CompositeLit{ArrayLit: &ArrayLit{Elems: &List[Expr]{X: []*Expr{
								{UnaryExpr: UnaryExpr{Operand: Operand{Literal: &Literal{Basic: &BasicLit{
									Number: ptr("1"),
								}}}}},
								{UnaryExpr: UnaryExpr{Operand: Operand{Literal: &Literal{Basic: &BasicLit{
									String: ptr(`"2"`),
								}}}}},
								{UnaryExpr: UnaryExpr{Operand: Operand{Literal: &Literal{Basic: &BasicLit{
									Number: ptr("3"),
								}}}}},
							}}}},
						}}}},
					},
					{
						Key: Expr{UnaryExpr: UnaryExpr{Operand: Operand{Literal: &Literal{Basic: &BasicLit{
							String: ptr(`"12"`),
						}}}}},
						Value: Expr{UnaryExpr: UnaryExpr{Operand: Operand{Literal: &Literal{Basic: &BasicLit{
							Number: ptr("234"),
						}}}}},
					},
				}}},
			}}}}},
		},
		{
			Code: `|| => 1`,
			Expected: Expr{UnaryExpr: UnaryExpr{Operand: Operand{Func: &FuncExpr{
				Expr: &Expr{UnaryExpr: UnaryExpr{Operand: Operand{Literal: &Literal{Basic: &BasicLit{
					Number: ptr("1"),
				}}}}},
			}}}},
		},
		{
			Code: `|a, b| => a`,
			Expected: Expr{UnaryExpr: UnaryExpr{Operand: Operand{Func: &FuncExpr{
				Args: &List[Ident]{X: []*Ident{
					{
						Name: "a",
					},
					{
						Name: "b",
					},
				}},
				Expr: &Expr{UnaryExpr: UnaryExpr{Operand: Operand{
					Name: &Ident{Name: "a"},
				}}},
			}}}},
		},
		{
			Code: `|| => { return 1 }`,
			Expected: Expr{UnaryExpr: UnaryExpr{Operand: Operand{Func: &FuncExpr{
				Block: &BlockStmt{List: &[]*Stmt{
					{
						Return: &ReturnStmt{
							ReturnExpr: &Expr{UnaryExpr: UnaryExpr{Operand: Operand{Literal: &Literal{
								Basic: &BasicLit{Number: ptr("1")},
							}}}},
						},
					},
				}},
			}}}},
		},
		{
			Code:      `()`,
			IsInvalid: true,
		},
		{
			Code:      `foo[]`,
			IsInvalid: true,
		},
		{
			Code:      `.bar`,
			IsInvalid: true,
		},
		{
			Code: `foo
			.bar`,
			IsInvalid: true,
		},
		{
			Code:      `foo(,,)`,
			IsInvalid: true,
		},
		{
			Code:      `||`,
			IsInvalid: true,
		},
		{
			Code:      `=> {}`,
			IsInvalid: true,
		},
		{
			Code:      `|| =>`,
			IsInvalid: true,
		},
		{
			Code:      `|,,| => 1`,
			IsInvalid: true,
		},
		{
			Code:      ``,
			IsInvalid: true,
		},
	}

	is := assert.New(t)
	for i, testCase := range testCases {
		x, err := parser.ParseString("", testCase.Code)
		if testCase.IsInvalid {
			is.Error(err, i)
			continue
		}

		is.NoError(err, i)
		isEq(t, testCase.Expected, x)
	}
}

func TestStmt(t *testing.T) {
	parser, err := participle.Build[ProgramFile](
		participle.Lexer(lexdef),
		participle.Elide("Comment", "Whitespace"),
	)
	require.NoError(t, err)

	testCases := testCases[ProgramFile]{
		{
			Code: `
			a = 1
			`,
			Expected: ProgramFile{
				List: &[]*Stmt{
					{
						Expr: &ExprStmt{
							X: Expr{UnaryExpr: UnaryExpr{Operand: Operand{
								Name: &Ident{Name: "a"},
							}}},
							AssignX: &Expr{
								UnaryExpr: UnaryExpr{Operand: Operand{Literal: &Literal{
									Basic: &BasicLit{
										Number: ptr("1"),
									},
								}}},
							},
						},
					},
				},
			},
		},
		{
			Code: `
			if a < b {
				a = b
			} else if b < a {
				b = a
			} else {
				c = a
			}
			`,
			Expected: ProgramFile{List: &[]*Stmt{
				{
					If: &IfStmt{
						Cond: Expr{
							UnaryExpr: UnaryExpr{Operand: Operand{Name: &Ident{Name: "a"}}},
							BinaryExpr: &BinaryExpr{
								Op: "<",
								X:  UnaryExpr{Operand: Operand{Name: &Ident{Name: "b"}}},
							},
						},
						Block: BlockStmt{List: &[]*Stmt{
							{
								Expr: &ExprStmt{
									X:       Expr{UnaryExpr: UnaryExpr{Operand: Operand{Name: &Ident{Name: "a"}}}},
									AssignX: &Expr{UnaryExpr: UnaryExpr{Operand: Operand{Name: &Ident{Name: "b"}}}},
								},
							},
						}},
						ElseIf: &IfStmt{
							Cond: Expr{
								UnaryExpr: UnaryExpr{Operand: Operand{Name: &Ident{Name: "b"}}},
								BinaryExpr: &BinaryExpr{
									Op: "<",
									X:  UnaryExpr{Operand: Operand{Name: &Ident{Name: "a"}}},
								},
							},
							Block: BlockStmt{List: &[]*Stmt{
								{
									Expr: &ExprStmt{
										X:       Expr{UnaryExpr: UnaryExpr{Operand: Operand{Name: &Ident{Name: "b"}}}},
										AssignX: &Expr{UnaryExpr: UnaryExpr{Operand: Operand{Name: &Ident{Name: "a"}}}},
									},
								},
							}},
							ElseBlock: &BlockStmt{List: &[]*Stmt{
								{
									Expr: &ExprStmt{
										X:       Expr{UnaryExpr: UnaryExpr{Operand: Operand{Name: &Ident{Name: "c"}}}},
										AssignX: &Expr{UnaryExpr: UnaryExpr{Operand: Operand{Name: &Ident{Name: "a"}}}},
									},
								},
							}},
						},
					},
				},
			}},
		},
		{
			Code: `
			for n in [1, 2] {
				i = i + n
			}
			`,
			Expected: ProgramFile{List: &[]*Stmt{
				{
					For: &ForStmt{
						IdentList: &List[Ident]{X: []*Ident{{Name: "n"}}},
						OverX: Expr{UnaryExpr: UnaryExpr{Operand: Operand{
							Literal: &Literal{Composite: &CompositeLit{ArrayLit: &ArrayLit{
								Elems: &List[Expr]{X: []*Expr{
									{
										UnaryExpr: UnaryExpr{Operand: Operand{Literal: &Literal{
											Basic: &BasicLit{Number: ptr("1")},
										}}},
									},
									{
										UnaryExpr: UnaryExpr{Operand: Operand{Literal: &Literal{
											Basic: &BasicLit{Number: ptr("2")},
										}}},
									},
								}},
							}}},
						}}},
						Block: BlockStmt{List: &[]*Stmt{
							{
								Expr: &ExprStmt{
									X: Expr{UnaryExpr: UnaryExpr{Operand: Operand{
										Name: &Ident{Name: "i"},
									}}},
									AssignX: &Expr{
										UnaryExpr: UnaryExpr{Operand: Operand{
											Name: &Ident{Name: "i"},
										}},
										BinaryExpr: &BinaryExpr{
											Op: "+",
											X: UnaryExpr{Operand: Operand{
												Name: &Ident{Name: "n"},
											}},
										},
									},
								},
							},
						}},
					},
				},
			}},
		},
		{
			Code: `
			while true {
				foo()
			}`,
			Expected: ProgramFile{List: &[]*Stmt{
				{
					While: &WhileStmt{
						Cond: Expr{UnaryExpr: UnaryExpr{Operand: Operand{Name: &Ident{Name: "true"}}}},
						Block: BlockStmt{List: &[]*Stmt{
							{
								Expr: &ExprStmt{X: Expr{UnaryExpr: UnaryExpr{Operand: Operand{
									Name: &Ident{Name: "foo"},
									PX:   &PrimaryExpr{CallExpr: &CallExpr{}},
								}}}},
							},
						}},
					},
				},
			}},
		},
		{
			Code: `
			a = block {
				if 1 > 2 {
					return 1
				}

				return 2
			}
			`,
			Expected: ProgramFile{List: &[]*Stmt{
				{
					Expr: &ExprStmt{
						X: Expr{UnaryExpr: UnaryExpr{Operand: Operand{Name: &Ident{Name: "a"}}}},
						AssignX: &Expr{UnaryExpr: UnaryExpr{Operand: Operand{
							Block: &BlockExpr{Block: BlockStmt{List: &[]*Stmt{
								{
									If: &IfStmt{
										Cond: Expr{
											UnaryExpr: UnaryExpr{Operand: Operand{Literal: &Literal{
												Basic: &BasicLit{Number: ptr("1")},
											}}},
											BinaryExpr: &BinaryExpr{
												Op: ">",
												X: UnaryExpr{Operand: Operand{Literal: &Literal{Basic: &BasicLit{
													Number: ptr("2"),
												}}}},
											},
										},
										Block: BlockStmt{List: &[]*Stmt{
											{
												Return: &ReturnStmt{ReturnExpr: &Expr{UnaryExpr: UnaryExpr{
													Operand: Operand{Literal: &Literal{Basic: &BasicLit{
														Number: ptr("1"),
													}}},
												}}},
											},
										}},
									},
								},
								{
									Return: &ReturnStmt{ReturnExpr: &Expr{UnaryExpr: UnaryExpr{
										Operand: Operand{Literal: &Literal{Basic: &BasicLit{
											Number: ptr("2"),
										}}},
									}}},
								},
							}}},
						}}},
					},
				},
			}},
		},
	}

	is := assert.New(t)
	for i, testCase := range testCases {
		x, err := parser.ParseString("", testCase.Code)
		if testCase.IsInvalid {
			is.Error(err, i)
			continue
		}

		is.NoError(err, i)
		isEq(t, testCase.Expected, x)
	}
}

func isEq(t *testing.T, x, y any) {
	var xb, yb bytes.Buffer
	require.NoError(t, json.NewEncoder(&xb).Encode(x))
	require.NoError(t, json.NewEncoder(&yb).Encode(y))
	var xm, ym map[string]any
	require.NoError(t, json.NewDecoder(&xb).Decode(&xm))
	require.NoError(t, json.NewDecoder(&yb).Decode(&ym))
	delNodeFields(xm)
	delNodeFields(ym)
	assert.Equal(t, xm, ym)
}

func delNodeFields(m map[string]any) {
	for k, v := range m {
		if k == "Pos" || k == "EndPos" {
			delete(m, k)
			continue
		}

		if v == nil {
			continue
		}

		if m, ok := v.(map[string]any); ok {
			delNodeFields(m)
		}

		if v, ok := v.([]any); ok {
			var newV []any
			for _, el := range v {
				elm, ok := el.(map[string]any)
				if !ok {
					newV = append(newV, el)
					continue
				}

				delNodeFields(elm)
				newV = append(newV, elm)
			}
			m[k] = newV
		}
	}
}

func newNode() Node {
	return Node{
		Pos: lexer.Position{},
	}
}
