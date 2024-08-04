package easylang

import "github.com/alecthomas/participle/v2/lexer"

type Node struct {
	Pos    lexer.Position
	EndPos lexer.Position
}

func (n Node) GetPos() lexer.Position {
	return n.Pos
}

func (n Node) GetEndPos() lexer.Position {
	return n.EndPos
}

type NodeBase interface {
	GetPos() lexer.Position
	GetEndPos() lexer.Position
}

type List[T NodeBase] struct {
	Node
	X []*T `@@ ( EOL* "," EOL* @@? )*`
}

type Ident struct {
	Node
	Name string `@Ident`
}

type Literal struct {
	Node
	Basic     *BasicLit     `  @@`
	Composite *CompositeLit `| @@`
}

type BasicLit struct {
	Node
	Number *string `  @Number`
	String *string `| @String`
}

type CompositeLit struct {
	Node
	ArrayLit  *ArrayLit  `  @@`
	ObjectLit *ObjectLit `| @@`
}

type ArrayLit struct {
	Node
	Elems *List[Expr] `"[" EOL* @@? EOL* "]"`
}

type ObjectLit struct {
	Node
	Items *List[KeyValueExpr] `"{" EOL* @@? EOL* "}"`
}

type KeyValueExpr struct {
	Node
	Key   Expr `@@ ":"`
	Value Expr `@@`
}

type Expr struct {
	Node
	UnaryExpr  UnaryExpr   `@@`
	BinaryExpr *BinaryExpr `@@?`
}

type BinaryExpr struct {
	Node
	Op   string      `@(OpBinaryPrior1 | OpBinaryPrior2 | OpBinaryArith) EOL*`
	X    UnaryExpr   `@@`
	Next *BinaryExpr `@@?`
}

type UnaryExpr struct {
	Node
	UnaryOp *string `@("-" | "not")?`
	Operand Operand `@@`
}

type PrimaryExpr struct {
	Node
	SelectorExpr *SelectorExpr `( @@`
	IndexExpr    *IndexExpr    `| @@`
	CallExpr     *CallExpr     `| @@ )`
}

type Operand struct {
	Node
	Block     *BlockExpr   `( @@`
	Func      *FuncExpr    `| @@`
	Import    *ImportExpr  `| @@`
	Literal   *Literal     `| @@`
	Name      *Ident       `| @@`
	ParenExpr *Expr        `| "(" EOL* @@ EOL* ")" )`
	PX        *PrimaryExpr `@@?`
}

type BlockExpr struct {
	Node
	Block BlockStmt `"block" @@`
}

type FuncExpr struct {
	Node
	Args  *List[Ident] `"|" EOL* @@? EOL* "|" FuncSign`
	Block *BlockStmt   `( @@`
	Expr  *Expr        `| @@ )`
}

type ImportExpr struct {
	Node
	Path string `"import" @String`
}

type SelectorExpr struct {
	Node
	Sel []SelectorExprPiece `"." EOL* @@ ("." EOL* @@)*`
	PX  *PrimaryExpr        `@@?`
}

type SelectorExprPiece struct {
	Node
	Ident  *Ident  `( @@`
	String *string `| @String )`
}

type IndexExpr struct {
	Node
	Index *List[Expr]  `"[" EOL* @@ EOL* "]"`
	PX    *PrimaryExpr `@@?`
}

type CallExpr struct {
	Node
	Args *List[Expr]  `"(" EOL* @@? EOL* ")"`
	PX   *PrimaryExpr `@@?`
}

type Stmt struct {
	Node
	If       *IfStmt       `( @@`
	For      *ForStmt      `| @@`
	While    *WhileStmt    `| @@`
	Return   *ReturnStmt   `| @@`
	Continue *ContinueStmt `| @@`
	Break    *BreakStmt    `| @@`
	Using    *UsingStmt    `| @@`
	Expr     *ExprStmt     `| @@ )`
}

type ExprStmt struct {
	Node
	IsPub       *string `@"pub"?`
	X           Expr    `@@`
	AugmentedOp *string `( @OpBinaryArith? `
	AssignX     *Expr   `  "=" @@ )?`
}

type BlockStmt struct {
	Node
	List *[]*Stmt `"{" EOL* ( @@ ( EOL+ @@? )* )? EOL* "}"`
}

type IfStmt struct {
	Node
	Cond      Expr       `"if" @@`
	Block     BlockStmt  `@@`
	ElseBlock *BlockStmt `( "else" ( @@`
	ElseIf    *IfStmt    `| @@ ) )?`
}

type ForStmt struct {
	Node
	IdentList *List[Ident] `"for" (@@ "in")?`
	OverX     Expr         `@@`
	Block     BlockStmt    `@@`
}

type WhileStmt struct {
	Node
	Cond  Expr      `"while" @@`
	Block BlockStmt `@@`
}

type ReturnStmt struct {
	Node
	ReturnExpr *Expr `"return" @@?`
}

type ContinueStmt struct {
	Node
	Key struct{} `"continue"`
}

type BreakStmt struct {
	Node
	Key struct{} `"break"`
}

type UsingStmt struct {
	Node
	Name  Ident  `"using" @@`
	Alias *Ident `("as" @@)?`
}

type ProgramFile struct {
	List *[]*Stmt `EOL* ( @@ ( EOL+ @@? )* )? EOL*`
}
