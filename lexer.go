package easylang

import "github.com/alecthomas/participle/v2/lexer"

var lexdef = lexer.MustSimple([]lexer.SimpleRule{
	{"Whitespace", `[ \t]+`},
	{"Comment", "#[^\n]*\n?"},
	{"FuncSign", "=>"},
	{"OpBinary", `and|or|==|!=|<|<=|>|>=|\+|-|\*|/|%`},
	{"OpUnary", `-|not`},
	{"Number", `0b[01]*|0|\d+`},
	{"String", `"(?:\\.|[^"])*"`},
	{"Ident", `[a-zA-Z_](?:[a-zA-Z_]|[0-9])*`},
	{"EOL", `[\n\r]+`},
	{"Period", "."},
	{"Semicolon", ","},
	{"LParen", `\(`},
	{"RParen", `\)`},
	{"Brack", `[\[\]]`},
	{"Brace", `[\{\}]`},
})

type ConstValue string

const (
	ConstValueNone  = "none"
	ConstValueTrue  = "true"
	ConstValueFalse = "false"
)

var operatorPriorities = map[string]int{
	"*": 5, "/": 5, "%": 5,
	"+": 4, "-": 4,
	"==": 3, "!=": 3, "<": 3, "<=": 3, ">": 3, ">=": 3,
	"and": 2, "or": 1,
}

func IsConstValue(s string) bool {
	switch s {
	case ConstValueNone, ConstValueTrue, ConstValueFalse:
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
