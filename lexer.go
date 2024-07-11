package easylang

import (
	"fmt"
	"strings"

	"github.com/alecthomas/participle/v2/lexer"
)

func digitsRe(prefix string, charClass string) string {
	return fmt.Sprintf(`%[1]s[%[2]s]+(?:_?[%[2]s]+)*\.?[%[2]s]*(?:_?[%[2]s]+)*`, prefix, charClass)
}

var (
	binaryDigitsRe = digitsRe("0(?:b|B)", "01")
	octalDigitsRe  = digitsRe("0(?:o|O)", "0-7")
	digits10Re     = digitsRe("", "0-9")
	hexDigitsRe    = digitsRe("0(?:x|X)", "0-9a-fA-F")
)

var lexdef = lexer.MustSimple([]lexer.SimpleRule{
	{Name: "Whitespace", Pattern: `[ \t]+`},
	{Name: "Comment", Pattern: "#[^\n]*\n?"},
	{Name: "FuncSign", Pattern: "=>"},
	{Name: "OpBinaryPrior1", Pattern: `==|!=|<=|>=`},
	{Name: "OpBinaryPrior2", Pattern: `and|or|<|>|\+|-|\*|/|%`},
	{Name: "OpUnary", Pattern: `-|not`},
	{Name: "Number", Pattern: strings.Join([]string{"inf", binaryDigitsRe, octalDigitsRe, hexDigitsRe, digits10Re}, "|")},
	{Name: "String", Pattern: `"(?:\\.|[^"])*"`},
	{Name: "Ident", Pattern: `[a-zA-Z_](?:[a-zA-Z_]|[0-9])*`},
	{Name: "EOL", Pattern: `[\n\r]+`},
	{Name: "Period", Pattern: "."},
	{Name: "Semicolon", Pattern: ","},
	{Name: "LParen", Pattern: `\(`},
	{Name: "RParen", Pattern: `\)`},
	{Name: "Brack", Pattern: `[\[\]]`},
	{Name: "Brace", Pattern: `[\{\}]`},
})

type ConstValue string

const (
	ConstValueNone  = "none"
	ConstValueTrue  = "true"
	ConstValueFalse = "false"
	ConstValueInf   = "inf"
)

var operatorPriorities = map[string]int{
	"*": 5, "/": 5, "%": 5,
	"+": 4, "-": 4,
	"==": 3, "!=": 3, "<": 3, "<=": 3, ">": 3, ">=": 3,
	"and": 2, "or": 1,
}

func IsConstValue(s string) bool {
	switch s {
	case ConstValueNone, ConstValueTrue, ConstValueFalse, ConstValueInf:
		return true
	}

	return false
}

func IsArithOp(op string) bool {
	switch op {
	case "+", "-", "*", "/", "%":
		return true
	}

	return false
}

func IsCmpOp(op string) bool {
	switch op {
	case "==", "!=", "<", "<=", ">", ">=":
		return true
	}

	return false
}

func IsPredicateOp(op string) bool {
	switch op {
	case "and", "or":
		return true
	}

	return false
}

func IsLiteralConstant(s string) bool {
	switch s {
	case "true", "false", "none":
		return true
	}

	return false
}

func IsKeyword(s string) bool {
	switch s {
	case "if", "else", "for", "in", "while",
		"return", "break", "continue", "block":
		return true
	}

	return false
}
