package easylang

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"

	"github.com/hikitani/easylang/variant"
)

var (
	ErrStmtFinished = errors.New("stmt finished")
	ErrLoopContinue = errors.New("loop continue")
	ErrLoopBreak    = errors.New("loop break")
)

type ExprCodeGenerator[T Node] interface {
	CodeGen(node *T) ExprEvaler
}

type ExprEvaler interface {
	Eval() (variant.Iface, error)
}

type exprCodeFunc struct {
	fn func() (variant.Iface, error)
}

func (c *exprCodeFunc) Eval() (variant.Iface, error) {
	return c.fn()
}

func evaler(fn func() (variant.Iface, error)) ExprEvaler {
	return &exprCodeFunc{fn: fn}
}

type StmtInvoker interface {
	Invoke() error
}

type stmtInvokerFunc struct {
	fn func() error
}

func (s *stmtInvokerFunc) Invoke() error {
	return s.fn()
}

func invoker(fn func() error) StmtInvoker {
	return &stmtInvokerFunc{fn: fn}
}

type BasicLitCodeGen struct{}

func (ec *BasicLitCodeGen) CodeGen(node *BasicLit) (ExprEvaler, error) {
	if v := node.Number; v != nil {
		num := &big.Float{}
		_, _, err := num.Parse(*v, 0)
		if err != nil {
			return nil, fmt.Errorf("bad parser: failed to parse number, %w", err)
		}

		return evaler(func() (variant.Iface, error) {
			return variant.NewNum(num), nil
		}), nil
	}

	if v := node.String; v != nil {
		s := strings.Trim(*v, `"`)

		runes := make([]rune, 0, len(s))
		var atEsc bool
		jump := 0
		for i, ch := range s {
			if jump > 0 {
				jump--
				continue
			}

			if ch == '\\' {
				if lenAfter(s, i) < 1 {
					return nil, errors.New("bad string literal: backslash not escaped")
				}
				atEsc = true
				continue
			}

			if !atEsc {
				runes = append(runes, ch)
				continue
			}

			switch ch {
			case 'u':
				if lenAfter(s, i) < 4 {
					return nil, errors.New("bad string literal: invalid \\u char, expected 4 bytes (\\u0000)")
				}
				jump = 4

				sub := s[i+1 : (i+1)+jump]
				v, err := strconv.ParseUint(sub, 16, 32)
				if err != nil {
					return nil, fmt.Errorf("bad string literal: illegal char in escape sequence: %w", err)
				}

				runes = append(runes, rune(v))
			case 'U':
				if lenAfter(s, i) < 8 {
					return nil, errors.New("bad string literal: invalid \\U char, expected 8 bytes (\\U00000000)")
				}
				jump = 8

				sub := s[i+1 : (i+1)+jump]
				v, err := strconv.ParseUint(sub, 16, 32)
				if err != nil {
					return nil, fmt.Errorf("bad string literal: illegal char in escape sequence: %w", err)
				}

				runes = append(runes, rune(v))
			case 'a':
				runes = append(runes, '\a')
			case 'b':
				runes = append(runes, '\b')
			case 'f':
				runes = append(runes, '\f')
			case 'n':
				runes = append(runes, '\n')
			case 'r':
				runes = append(runes, '\r')
			case 't':
				runes = append(runes, '\t')
			case 'v':
				runes = append(runes, '\v')
			case '\\':
				runes = append(runes, '\\')
			case '\'':
				runes = append(runes, '\'')
			case '"':
				runes = append(runes, '"')
			}

			atEsc = false
		}

		return evaler(func() (variant.Iface, error) {
			return variant.NewString(string(runes)), nil
		}), nil
	}

	return nil, errors.New("unknown basic literal (expected string or number)")
}

type CompositeLitCodeGen struct {
	exprGen *ExprCodeGen
}

func (c *CompositeLitCodeGen) CodeGen(node *CompositeLit) (ExprEvaler, error) {
	if node.ArrayLit != nil {
		lit := node.ArrayLit
		elems := lit.Elems
		if elems == nil {
			elems = &List[Expr]{}
		}

		if len(elems.X) == 0 {
			return evaler(func() (variant.Iface, error) {
				return variant.NewArray(nil), nil
			}), nil
		}

		evals := make([]ExprEvaler, 0, len(elems.X))
		for i, elExpr := range elems.X {
			if elExpr == nil {
				return nil, fmt.Errorf("bad array literal: invalid expression on %d position", i+1)
			}

			el, err := c.exprGen.CodeGen(elExpr)
			if err != nil {
				return nil, fmt.Errorf("bad array literal on %d position: %w", i+1, err)
			}

			evals = append(evals, el)
		}

		return evaler(func() (variant.Iface, error) {
			arr := variant.NewArray(make([]variant.Iface, 0, len(evals)))
			for i, eval := range evals {
				v, err := eval.Eval()
				if err != nil {
					return nil, fmt.Errorf("cannot evaluate expression of element %d of array: %w", i+1, err)
				}
				arr.Append(v)
			}

			return arr, nil
		}), nil
	}

	if node.ObjectLit != nil {
		items := node.ObjectLit.Items
		if items == nil {
			items = &List[KeyValueExpr]{}
		}

		if len(items.X) == 0 {
			return evaler(func() (variant.Iface, error) {
				return variant.MustNewObject(nil, nil), nil
			}), nil
		}

		kvEvals := make([][2]ExprEvaler, 0, len(items.X))
		for i, kv := range items.X {
			if kv == nil {
				return nil, fmt.Errorf("bad object literal: invalid item expression on %d position", i+1)
			}

			keyEval, err := c.exprGen.CodeGen(&kv.Key)
			if err != nil {
				return nil, fmt.Errorf("bad object literal: invalid key expression on position %d: %w", i+1, err)
			}

			valEval, err := c.exprGen.CodeGen(&kv.Value)
			if err != nil {
				return nil, fmt.Errorf("bad object literal: invalid value expression on position %d: %w", i+1, err)
			}

			kvEvals = append(kvEvals, [2]ExprEvaler{keyEval, valEval})
		}

		return evaler(func() (variant.Iface, error) {
			keys, vals := make([]variant.Iface, 0, len(kvEvals)), make([]variant.Iface, 0, len(kvEvals))
			for i, kv := range kvEvals {
				keyEval, valEval := kv[0], kv[1]
				key, err := keyEval.Eval()
				if err != nil {
					return nil, fmt.Errorf("cannot evaluate expression of key on position %d: %w", i+1, err)
				}

				val, err := valEval.Eval()
				if err != nil {
					return nil, fmt.Errorf("cannot evaluate expression of value on position %d: %w", i+1, err)
				}

				keys = append(keys, key)
				vals = append(vals, val)
			}

			return variant.MustNewObject(keys, vals), nil
		}), nil
	}

	return nil, errors.New("unknown composite literal (expected array or object)")
}

type OperandCodeGen struct {
	exprGen *ExprCodeGen
}

func (c *OperandCodeGen) CodeGen(node *Operand) (eval ExprEvaler, err error) {
	switch {
	case node.Func != nil:
		vars := c.exprGen.vars.WithScope()
		vars.ParentBlockScope = vars.LastScope()
		eval, err = (&FuncExprCodeGen{
			exprGen: &ExprCodeGen{vars: vars},
			blkGen:  &BlockStmtCodeGen{vars: vars},
		}).CodeGen(node.Func)
	case node.Block != nil:
		vars := c.exprGen.vars.WithScope()
		vars.ParentBlockScope = vars.LastScope()
		eval, err = (&BlockExprCodeGen{
			blkGen: &BlockStmtCodeGen{vars: vars},
		}).CodeGen(node.Block)
	case node.Literal != nil:
		lit := node.Literal
		switch {
		case lit.Basic != nil:
			eval, err = (&BasicLitCodeGen{}).CodeGen(lit.Basic)
		case lit.Composite != nil:
			eval, err = (&CompositeLitCodeGen{exprGen: c.exprGen}).CodeGen(lit.Composite)
		default:
			return nil, errors.New("bad literal: invalid expression (expected basic or composit literal)")
		}
	case node.ParenExpr != nil:
		eval, err = c.exprGen.CodeGen(node.ParenExpr)
	case node.Name != nil:
		name := node.Name.Name

		if IsConstValue(name) {
			switch name {
			case ConstValueNone:
				return evaler(func() (variant.Iface, error) {
					return variant.NewNone(), nil
				}), nil
			case ConstValueTrue:
				return evaler(func() (variant.Iface, error) {
					return variant.NewBool(true), nil
				}), nil
			case ConstValueFalse:
				return evaler(func() (variant.Iface, error) {
					return variant.NewBool(false), nil
				}), nil
			case ConstValueInf:
				return evaler(func() (variant.Iface, error) {
					return variant.NewNum(new(big.Float).SetInf(false)), nil
				}), nil
			}

			return nil, fmt.Errorf("unknown const value %s", name)
		}

		if IsKeyword(name) {
			return nil, fmt.Errorf("bad variable: name %s is keyword", name)
		}

		scope, reg := c.exprGen.vars.Register(name)

		eval = evaler(func() (variant.Iface, error) {
			v, ok := scope.GetVar(reg)
			if !ok {
				return nil, fmt.Errorf("variable %s not defined", name)
			}

			return v, nil
		})
	default:
		return nil, errors.New("unknown operand (expected literal, block, func, ident or parent expression)")
	}

	if err != nil {
		return nil, fmt.Errorf("bad operand: %w", err)
	}

	if eval == nil {
		panic("operand code gen: impossible nil eval")
	}

	if node.PX != nil {
		eval, err = (&PrimaryExprCodeGen{
			exprGen:  c.exprGen,
			prevEval: eval,
		}).CodeGen(node.PX)
		if err != nil {
			return nil, fmt.Errorf("bad operand: %w", err)
		}
	}

	return eval, nil
}

type PrimaryExprCodeGen struct {
	exprGen  *ExprCodeGen
	prevEval ExprEvaler
}

func (c *PrimaryExprCodeGen) CodeGen(node *PrimaryExpr) (eval ExprEvaler, _ error) {
	var nextNode *PrimaryExpr
	switch {
	case node.IndexExpr != nil:
		nextNode = node.IndexExpr.PX
		args := node.IndexExpr.Index
		if args == nil {
			args = &List[Expr]{}
		}

		if len(args.X) == 0 {
			panic("syntax error: indexator must have at least once index")
		}

		idxEvals := make([]ExprEvaler, 0, len(args.X))
		for i, expr := range args.X {
			idxEval, err := c.exprGen.CodeGen(expr)
			if err != nil {
				return nil, fmt.Errorf("bad primary expression: index at %d position is invalid: %w", i+1, err)
			}

			idxEvals = append(idxEvals, idxEval)
		}

		eval = evaler(func() (variant.Iface, error) {
			prev, err := c.prevEval.Eval()
			if err != nil {
				return nil, err
			}

			switch prev.Type() {
			case variant.TypeArray:
				if len(idxEvals) != 1 {
					return nil, fmt.Errorf("array indexator must have 1 argument")
				}
				arr := variant.MustCast[*variant.Array](prev)

				idxEval := idxEvals[0]
				idx, err := idxEval.Eval()
				if err != nil {
					return nil, fmt.Errorf("cannot evaluate index: %w", err)
				}

				if idx.Type() != variant.TypeNum {
					return nil, fmt.Errorf("index must be number, got %s", idx.Type())
				}

				num, err := variant.MustCast[*variant.Num](idx).AsInt64()
				if err != nil {
					return nil, fmt.Errorf("cannot to represent number as unsigned integer: %w", err)
				}

				val, err := arr.Get(num)
				if err != nil {
					return nil, fmt.Errorf("cannot get array element: %w", err)
				}

				return val, nil
			case variant.TypeObject:
				obj := variant.MustCast[*variant.Object](prev)
				var res variant.Iface
				for i, idxEval := range idxEvals {
					idx, err := idxEval.Eval()
					if err != nil {
						return nil, fmt.Errorf("cannot evaluate index: %w", err)
					}

					v, err := obj.Get(idx)
					if err != nil {
						return nil, fmt.Errorf("cannot get value by index %d: %w", i, err)
					}

					if i != len(idxEvals)-1 {
						if v.Type() != variant.TypeObject {
							return nil, fmt.Errorf("value at index %d unsupports indexator (expected object, got %s)", i, v.Type())
						}

						obj = variant.MustCast[*variant.Object](v)
					} else {
						res = v
					}
				}

				return res, nil
			}

			return nil, fmt.Errorf("unsupported indexator for %s", prev.Type())
		})
	case node.CallExpr != nil:
		nextNode = node.CallExpr.PX
		args := node.CallExpr.Args
		if args == nil {
			args = &List[Expr]{}
		}

		argEvals := make([]ExprEvaler, 0, len(args.X))
		for i, expr := range args.X {
			argEval, err := c.exprGen.CodeGen(expr)
			if err != nil {
				return nil, fmt.Errorf("bad primary expression: argument at %d position is invalid: %w", i+1, err)
			}

			argEvals = append(argEvals, argEval)
		}

		eval = evaler(func() (variant.Iface, error) {
			prev, err := c.prevEval.Eval()
			if err != nil {
				return nil, err
			}

			if prev.Type() != variant.TypeFunc {
				return nil, fmt.Errorf("unsupported caller expression for %s (expected func)", prev.Type())
			}

			fn := variant.MustCast[*variant.Func](prev)
			args := make([]variant.Iface, 0, len(argEvals))
			for i, argEval := range argEvals {
				arg, err := argEval.Eval()
				if err != nil {
					return nil, fmt.Errorf("cannot evaluate argument at %d position: %w", i+1, err)
				}

				args = append(args, arg)
			}

			return fn.Call(args)
		})
	case node.SelectorExpr != nil:
		nextNode = node.SelectorExpr.PX
		sels := node.SelectorExpr.Sel
		if len(sels) == 0 {
			panic("expected selector, got nothing")
		}

		selVars := make([]*variant.String, 0, len(sels))
		for i, sel := range sels {
			var val *variant.String
			switch {
			case sel.Ident != nil:
				if sel.Ident.Name == "" {
					panic(fmt.Sprintf("bad primary expression: selector at %d position must be named", i+1))
				}

				val = variant.NewString(sel.Ident.Name)
			case sel.String != nil:
				strEval, err := (&BasicLitCodeGen{}).CodeGen(&BasicLit{String: sel.String})
				if err != nil {
					return nil, fmt.Errorf("bad primary expression: selector at %d position is invalid: %w", i+1, err)
				}

				res, err := strEval.Eval()
				if err != nil {
					panic(fmt.Sprintf("cannot evaluate selector at %d position: %s", i+1, err))
				}

				val = variant.MustCast[*variant.String](res)
			}

			selVars = append(selVars, val)
		}

		eval = evaler(func() (variant.Iface, error) {
			prev, err := c.prevEval.Eval()
			if err != nil {
				return nil, err
			}

			if prev.Type() != variant.TypeObject {
				return nil, fmt.Errorf("unsupported selector for %s (expected object)", prev.Type())
			}

			obj := variant.MustCast[*variant.Object](prev)
			var res variant.Iface
			for i, sel := range selVars {
				v, err := obj.Get(sel)
				if err != nil {
					return nil, fmt.Errorf("cannot get value by %s: %w", selVars[i], err)
				}

				if i != len(selVars)-1 {
					if v.Type() != variant.TypeObject {
						return nil, fmt.Errorf("unsupported selector %s for %s (expected object)", selVars[i+1], v.Type())
					}

					obj = variant.MustCast[*variant.Object](v)
				} else {
					res = v
				}
			}

			return res, nil
		})
	default:
		return nil, fmt.Errorf("unknown primary expression: expected selector, indexator or caller")
	}

	if nextNode != nil {
		var err error
		eval, err = (&PrimaryExprCodeGen{
			exprGen:  c.exprGen,
			prevEval: eval,
		}).CodeGen(nextNode)
		if err != nil {
			return nil, fmt.Errorf("bad primary expression: %w", err)
		}
	}

	return eval, nil
}

type UnaryExprCodeGen struct {
	exprGen *ExprCodeGen
}

func (c *UnaryExprCodeGen) CodeGen(node *UnaryExpr) (ExprEvaler, error) {
	operandEval, err := (&OperandCodeGen{exprGen: c.exprGen}).CodeGen(&node.Operand)
	if err != nil {
		return nil, err
	}

	if node.UnaryOp == nil {
		return operandEval, nil
	}

	op := *node.UnaryOp
	switch op {
	case "-":
		return evaler(func() (variant.Iface, error) {
			v, err := operandEval.Eval()
			if err != nil {
				return nil, err
			}

			if v.Type() != variant.TypeNum {
				return nil, fmt.Errorf("%s doesn't support unary operator '-' (expected number)", v.Type())
			}

			num := variant.MustCast[*variant.Num](v)
			return num.Neg(), nil
		}), nil
	case "not":
		return evaler(func() (variant.Iface, error) {
			v, err := operandEval.Eval()
			if err != nil {
				return nil, err
			}

			if v.Type() != variant.TypeBool {
				return nil, fmt.Errorf("%s doesn't support unary operator 'not' (expected bool)", v.Type())
			}

			b := variant.MustCast[*variant.Bool](v)
			return variant.NewBool(!b.Bool()), nil
		}), nil
	}

	return nil, fmt.Errorf("unsupported unary operator %s", op)
}

type FuncExprCodeGen struct {
	exprGen *ExprCodeGen
	blkGen  *BlockStmtCodeGen
}

func (c *FuncExprCodeGen) CodeGen(node *FuncExpr) (ExprEvaler, error) {
	args := node.Args
	if args == nil {
		args = &List[Ident]{}
	}

	uniq := map[string]struct{}{}
	for _, v := range args.X {
		uniq[v.Name] = struct{}{}
	}

	if len(args.X) != len(uniq) {
		return nil, errors.New("bad function: argument names must be unique")
	}

	type ScopeAndReg struct {
		Scope *VarScope
		Reg   Register
	}
	regs := func(vars *Vars) []ScopeAndReg {
		var res []ScopeAndReg
		for _, arg := range args.X {
			scope, reg := vars.Register(arg.Name)
			res = append(res, ScopeAndReg{
				Scope: scope,
				Reg:   reg,
			})
		}
		return res
	}

	prefngen := func(regs []ScopeAndReg) func(vargs []variant.Iface) error {
		return func(vargs []variant.Iface) error {
			if len(vargs) != len(args.X) {
				return fmt.Errorf("expected arguments %d, got %d", len(args.X), len(vargs))
			}

			for i := 0; i < len(vargs); i++ {
				regs[i].Scope.DefineVar(regs[i].Reg, vargs[i])
			}

			return nil
		}
	}

	switch {
	case node.Expr != nil:
		vars := c.exprGen.vars
		prefn := prefngen(regs(vars))

		eval, err := c.exprGen.CodeGen(node.Expr)
		if err != nil {
			return nil, fmt.Errorf("bad function: invalid expression: %w", err)
		}

		return evaler(func() (variant.Iface, error) {
			return variant.NewFunc(func(vargs []variant.Iface) (variant.Iface, error) {
				if err := prefn(vargs); err != nil {
					return nil, err
				}

				return eval.Eval()
			}), nil
		}), nil
	case node.Block != nil:
		vars := c.blkGen.vars
		prefn := prefngen(regs(vars))

		invoker, err := c.blkGen.CodeGen(node.Block)
		if err != nil {
			return nil, fmt.Errorf("bad function: invalid block statement: %w", err)
		}

		return evaler(func() (variant.Iface, error) {
			return variant.NewFunc(func(vargs []variant.Iface) (variant.Iface, error) {
				if err := prefn(vargs); err != nil {
					return nil, err
				}

				err := invoker.Invoke()
				if err != nil && !errors.Is(err, ErrStmtFinished) {
					return nil, err
				}

				return vars.LastScope().GetReturn(), nil
			}), nil
		}), nil
	}

	return nil, fmt.Errorf("bad function expression")
}

type BlockExprCodeGen struct {
	blkGen *BlockStmtCodeGen
}

func (c *BlockExprCodeGen) CodeGen(node *BlockExpr) (ExprEvaler, error) {
	vars := c.blkGen.vars

	invoker, err := c.blkGen.CodeGen(&node.Block)
	if err != nil {
		return nil, fmt.Errorf("bad block expression: invalid block statement: %w", err)
	}

	return evaler(func() (variant.Iface, error) {
		err := invoker.Invoke()
		if err != nil && !errors.Is(err, ErrStmtFinished) {
			return nil, err
		}

		return vars.LastScope().GetReturn(), nil
	}), nil
}

type ExprCodeGen struct {
	vars *Vars
}

func (c *ExprCodeGen) CodeGen(node *Expr) (ExprEvaler, error) {
	unaryEval, err := (&UnaryExprCodeGen{exprGen: c}).CodeGen(&node.UnaryExpr)
	if err != nil {
		return nil, err
	}

	if node.BinaryExpr == nil {
		return unaryEval, nil
	}

	type opinfo struct {
		op      string
		prior   int
		origPos int
	}
	var ops []opinfo
	evals := []ExprEvaler{unaryEval}
	binExpr := node.BinaryExpr

	for i := 0; binExpr != nil; i++ {
		ops = append(ops, opinfo{
			op:      binExpr.Op,
			prior:   operatorPriorities[binExpr.Op],
			origPos: i,
		})

		eval, err := (&UnaryExprCodeGen{exprGen: c}).CodeGen(&binExpr.X)
		if err != nil {
			return nil, fmt.Errorf("bad operand at %s position", binExpr.X.GetPos())
		}
		evals = append(evals, eval)
		binExpr = binExpr.Next
	}

	sort.Slice(ops, func(i, j int) bool {
		return ops[i].prior > ops[j].prior
	})

	getVal := func(eval ExprEvaler, stack *[]variant.Iface) (val variant.Iface, err error) {
		if eval == nil {
			// front := (*stack)[0]
			// *stack = (*stack)[1:]

			front := (*stack)[len(*stack)-1]
			*stack = (*stack)[:len(*stack)-1]
			return front, nil
		}

		val, err = eval.Eval()
		if err != nil {
			return nil, fmt.Errorf("cannot evaluate expression: %w", err)
		}
		return
	}

	stackCap := (len(ops) + 1) / 2
	return evaler(func() (variant.Iface, error) {
		evalMask := make([]bool, len(evals))
		stack := make([]variant.Iface, 0, stackCap)

		var leval, reval ExprEvaler
		for _, opinfo := range ops {
			i := opinfo.origPos
			if !evalMask[i] {
				leval = evals[i]
			} else {
				leval = nil
			}

			if !evalMask[i+1] {
				reval = evals[i+1]
			} else {
				reval = nil
			}

			evalMask[i], evalMask[i+1] = true, true

			rval, err := getVal(reval, &stack)
			if err != nil {
				return nil, err
			}

			lval, err := getVal(leval, &stack)
			if err != nil {
				return nil, err
			}

			res, err := evalBinary(opinfo.op, lval, rval)
			if err != nil {
				return nil, err
			}

			stack = append(stack, res)
		}

		return stack[0], nil
	}), nil
}

func evalBinary(op string, lval, rval variant.Iface) (variant.Iface, error) {
	if op == "+" && rval.Type() == variant.TypeString && lval.Type() == variant.TypeString {
		rs, ls := variant.MustCast[*variant.String](rval), variant.MustCast[*variant.String](lval)
		return variant.NewString(ls.String() + rs.String()), nil
	}

	if op == "+" && rval.Type() == variant.TypeArray && lval.Type() == variant.TypeArray {
		rs, ls := variant.MustCast[*variant.Array](rval), variant.MustCast[*variant.Array](lval)
		arr := variant.NewArray(ls.Slice())
		arr.Append(rs.Slice()...)
		return arr, nil
	}

	if IsCmpOp(op) {
		if rval.Type() != lval.Type() {
			return nil, fmt.Errorf("unsupported operand type for %s: %s and %s", op, lval.Type(), rval.Type())
		}

		b := false
		switch op {
		case "==":
			b = variant.DeepEqual(lval, rval)
		case "!=":
			b = !variant.DeepEqual(lval, rval)
		case "<", "<=", ">", ">=":
			if rval.Type() != variant.TypeNum {
				return nil, fmt.Errorf("unsupported operand type for %s: %s and %s", op, lval.Type(), rval.Type())
			}

			lnum, rnum := variant.MustCast[*variant.Num](lval), variant.MustCast[*variant.Num](rval)

			switch op {
			case "<":
				b = lnum.LessThan(rnum)
			case "<=":
				b = lnum.LessOrEqualTo(rnum)
			case ">":
				b = lnum.GreaterThan(rnum)
			case ">=":
				b = lnum.GreaterOrEqualTo(rnum)
			default:
				panic("unreachable")
			}
		default:
			return nil, fmt.Errorf("unknown operation '%s %s %s'", lval.Type(), op, rval.Type())
		}

		return variant.NewBool(b), nil
	}

	if IsArithOp(op) {
		if rval.Type() != variant.TypeNum || lval.Type() != variant.TypeNum {
			return nil, fmt.Errorf("unsupported operand type for %s: %s and %s", op, lval.Type(), rval.Type())
		}
		rnum, lnum := variant.MustCast[*variant.Num](rval), variant.MustCast[*variant.Num](lval)
		num := new(big.Float)
		switch op {
		case "+":
			if lnum.IsInf() && rnum.IsInf() && lnum.Sign() != rnum.Sign() {
				return nil, errors.New("op '+': addition of inf and inf with opposite signs")
			}
			num.Add(lnum.Value(), rnum.Value())
		case "-":
			if lnum.IsInf() && rnum.IsInf() && lnum.Sign() == rnum.Sign() {
				return nil, errors.New("op '-': subtraction of inf from inf with equal signs")
			}
			num.Sub(lnum.Value(), rnum.Value())
		case "/":
			if lnum.IsZero() && rnum.IsZero() {
				return nil, errors.New("op '/': division of zero into zero")
			}
			if lnum.IsInf() && rnum.IsInf() {
				return nil, errors.New("op '/': division of inf into inf")
			}
			num.Quo(lnum.Value(), rnum.Value())
		case "*":
			if (lnum.IsZero() && rnum.IsInf()) || (lnum.IsInf() && rnum.IsZero()) {
				return nil, errors.New("op '*': one operand is zero and the other operand an infinity")
			}
			num.Mul(lnum.Value(), rnum.Value())
		case "%":
			if rnum.Value().IsInf() {
				return nil, errors.New("op '%': modulus with inf")
			}

			if rnum.IsZero() {
				return nil, errors.New("op '%': modulus with zero")
			}

			if lnum.Value().IsInt() && rnum.Value().IsInt() {
				var x, y big.Int
				lnum.Value().Int(&x)
				rnum.Value().Int(&y)
				num.SetInt(x.Mod(&x, &y))
			} else if div := new(big.Float).Quo(lnum.Value(), rnum.Value()); div.IsInf() {
				num.Set(div)
			} else {
				// div = x / y
				// x % y = x - int(div) * y

				// 1. int(div)
				divInt, _ := div.Int(nil)
				// 2. int(div) * y
				mul := new(big.Float).Mul(div.SetInt(divInt), rnum.Value())
				// 3. x - int(div) * y
				num.Sub(lnum.Value(), mul)

				if lnum.Sign() < 0 {
					if rnum.Sign() > 0 {
						num.Add(rnum.Value(), num)
					} else {
						num.Add(mul.Neg(rnum.Value()), num)
					}
				}
			}
		default:
			return nil, fmt.Errorf("unknown operation 'number %s number'", op)
		}

		return variant.NewNum(num), nil
	}

	if IsPredicateOp(op) {
		if rval.Type() != variant.TypeBool || lval.Type() != variant.TypeBool {
			return nil, fmt.Errorf("unsupported operand type for %s: %s and %s", op, lval.Type(), rval.Type())
		}
		rb, lb := variant.MustCast[*variant.Bool](rval), variant.MustCast[*variant.Bool](lval)
		var b bool
		switch op {
		case "and":
			b = lb.Bool() && rb.Bool()
		case "or":
			b = lb.Bool() || rb.Bool()
		default:
			return nil, fmt.Errorf("unknown operation 'bool %s bool'", op)
		}
		return variant.NewBool(b), nil
	}

	return nil, fmt.Errorf("unknown operation '%s %s %s'", lval.Type(), op, rval.Type())
}

func lenAfter(s string, pos int) int {
	return max(0, len(s)-(pos+1))
}

type ContinueStmtCodeGen struct{}

func (c *ContinueStmtCodeGen) CodeGen(node *ContinueStmt) (StmtInvoker, error) {
	return invoker(func() error {
		return ErrLoopContinue
	}), nil
}

type BreakStmtCodeGen struct{}

func (c *BreakStmtCodeGen) CodeGen(node *BreakStmt) (StmtInvoker, error) {
	return invoker(func() error {
		return ErrLoopBreak
	}), nil
}

type ReturnStmtCodeGen struct {
	vars *Vars
}

func (c *ReturnStmtCodeGen) CodeGen(node *ReturnStmt) (StmtInvoker, error) {
	ret := func(v variant.Iface) error {
		c.vars.SetReturn(v)
		return ErrStmtFinished
	}
	if node.ReturnExpr == nil {
		return invoker(func() error {
			return ret(variant.NewNone())
		}), nil
	}

	eval, err := (&ExprCodeGen{vars: c.vars}).CodeGen(node.ReturnExpr)
	if err != nil {
		return nil, fmt.Errorf("bad return statement: %w", err)
	}

	return invoker(func() error {
		v, err := eval.Eval()
		if err != nil {
			return err
		}

		return ret(v)
	}), nil
}

type ExprStmtCodeGen struct {
	exprGen *ExprCodeGen
}

func (c *ExprStmtCodeGen) CodeGen(node *ExprStmt) (StmtInvoker, error) {
	if node.AssignX == nil {
		leval, err := c.exprGen.CodeGen(&node.X)
		if err != nil {
			return nil, fmt.Errorf("invalid lhs operand: %w", err)
		}

		return invoker(func() error {
			_, err := leval.Eval()
			if err != nil {
				return err
			}

			return nil
		}), nil
	}

	if node.X.BinaryExpr != nil {
		return nil, errors.New("lhs must be addressable")
	}

	unary := node.X.UnaryExpr
	if unary.UnaryOp != nil {
		return nil, fmt.Errorf("lhs must be addressable (unary operator %s disallowed)", *unary.UnaryOp)
	}

	if unary.Operand.Name == nil {
		return nil, fmt.Errorf("lhs must be addressable")
	}

	name := unary.Operand.Name.Name
	reval, err := c.exprGen.CodeGen(node.AssignX)
	if err != nil {
		return nil, fmt.Errorf("invalid rhs operand: %w", err)
	}

	if _, ok := c.exprGen.vars.LookupRegister(name); !ok {
		if node.AugmentedOp != nil {
			return nil, fmt.Errorf("name '%s' is not defined", name)
		}
	}

	scope, reg := c.exprGen.vars.Register(name)

	return invoker(func() error {
		v, err := reval.Eval()
		if err != nil {
			return err
		}

		if node.AugmentedOp != nil {
			lval, ok := scope.GetVar(reg)
			if !ok {
				panic("unreachable")
			}

			v, err = evalBinary(*node.AugmentedOp, lval, v)
			if err != nil {
				return err
			}
		}

		scope.DefineVar(reg, v)
		return nil
	}), nil
}

type StmtCodeGen struct {
	isLoopScope   bool
	isGlobalScope bool
	vars          *Vars
}

func (c StmtCodeGen) CodeGen(node *Stmt) (invoker StmtInvoker, err error) {
	switch {
	case node.If != nil:
		invoker, err = (&IfStmtCodeGen{
			isLoopScope: c.isLoopScope,
			vars:        c.vars,
		}).CodeGen(node.If)
	case node.For != nil:
		invoker, err = (&ForStmtCodeGen{
			vars: c.vars,
		}).CodeGen(node.For)
	case node.While != nil:
		invoker, err = (&WhileStmtCodeGen{
			vars: c.vars,
		}).CodeGen(node.While)
	case node.Return != nil:
		if c.isGlobalScope {
			return nil, errors.New("return statement cannot be used in global scope")
		}

		invoker, err = (&ReturnStmtCodeGen{
			vars: c.vars,
		}).CodeGen(node.Return)
	case node.Continue != nil:
		if !c.isLoopScope {
			return nil, errors.New("continue statement cannot be used outside of a loop")
		}

		invoker, err = (&ContinueStmtCodeGen{}).CodeGen(node.Continue)
	case node.Break != nil:
		if !c.isLoopScope {
			return nil, errors.New("break statement cannot be used outside of a loop")
		}

		invoker, err = (&BreakStmtCodeGen{}).CodeGen(node.Break)
	case node.Expr != nil:
		invoker, err = (&ExprStmtCodeGen{
			exprGen: &ExprCodeGen{vars: c.vars},
		}).CodeGen(node.Expr)
	default:
		return nil, fmt.Errorf("statement not defined (expected if, for, while, assignment, return or expr statement)")
	}

	return
}

type BlockStmtCodeGen struct {
	vars        *Vars
	isLoopScope bool
}

func (c *BlockStmtCodeGen) CodeGen(node *BlockStmt) (StmtInvoker, error) {
	var list []*Stmt
	if node.List != nil {
		list = *node.List
	}

	invokers := make([]StmtInvoker, 0, len(list))
	for _, stmt := range list {
		if stmt == nil {
			return nil, errors.New("bad block statement")
		}

		invoker, err := (&StmtCodeGen{vars: c.vars, isLoopScope: c.isLoopScope}).CodeGen(stmt)
		if err != nil {
			return nil, fmt.Errorf("bad statement: %w", err)
		}

		invokers = append(invokers, invoker)
	}

	return invoker(func() error {
		for _, invoker := range invokers {
			if err := invoker.Invoke(); err != nil {
				return err
			}
		}

		return nil
	}), nil
}

type WhileStmtCodeGen struct {
	vars *Vars
}

func (c *WhileStmtCodeGen) CodeGen(node *WhileStmt) (StmtInvoker, error) {
	condEval, err := (&ExprCodeGen{vars: c.vars}).CodeGen(&node.Cond)
	if err != nil {
		return nil, fmt.Errorf("invalid while condition expression: %w", err)
	}

	vars := c.vars.WithScope()
	blkInvoker, err := (&BlockStmtCodeGen{
		vars:        vars,
		isLoopScope: true,
	}).CodeGen(&node.Block)
	if err != nil {
		return nil, fmt.Errorf("invalid while block statement: %w", err)
	}

	return invoker(func() error {
		for {
			cond, err := condEval.Eval()
			if err != nil {
				return err
			}

			if cond.Type() != variant.TypeBool {
				return errors.New("condition expression must be bool")
			}

			b := variant.MustCast[*variant.Bool](cond)
			if !b.Bool() {
				return nil
			}

			err = blkInvoker.Invoke()
			if errors.Is(err, ErrLoopBreak) {
				break
			}

			if errors.Is(err, ErrLoopContinue) {
				continue
			}

			if err != nil {
				return err
			}
		}
		return nil
	}), nil
}

type ForStmtCodeGen struct {
	vars *Vars
}

func (c *ForStmtCodeGen) CodeGen(node *ForStmt) (StmtInvoker, error) {
	varnames := node.IdentList
	if varnames == nil {
		varnames = &List[Ident]{}
	}

	if len(varnames.X) > 2 {
		return nil, errors.New("bad for statement: expected 0, 1 or 2 variables")
	}

	overEval, err := (&ExprCodeGen{vars: c.vars}).CodeGen(&node.OverX)
	if err != nil {
		return nil, fmt.Errorf("bad for statement: invalid collection expression")
	}

	blkVars := c.vars.WithScope()
	blkInvoker, err := (&BlockStmtCodeGen{vars: blkVars, isLoopScope: true}).CodeGen(&node.Block)
	if err != nil {
		return nil, fmt.Errorf("bad for statement: invalid block statement: %w", err)
	}

	iterArr := func(i int, el variant.Iface) {}
	iterObj := func(k variant.Iface, el variant.Iface) {}

	scope := blkVars.LastScope()
	switch len(varnames.X) {
	case 0:
	case 1:
		r1 := scope.Register(varnames.X[0].Name)
		iterArr = func(_ int, el variant.Iface) {
			scope.DefineVar(r1, el)
		}
		iterObj = func(k variant.Iface, _ variant.Iface) {
			scope.DefineVar(r1, k)
		}
	case 2:
		r1 := scope.Register(varnames.X[0].Name)
		r2 := scope.Register(varnames.X[1].Name)
		iterArr = func(i int, el variant.Iface) {
			scope.DefineVar(r1, variant.Int(i))
			scope.DefineVar(r2, el)
		}
		iterObj = func(k variant.Iface, el variant.Iface) {
			scope.DefineVar(r1, k)
			scope.DefineVar(r2, el)
		}
	default:
		panic("unreachable")
	}

	return invoker(func() error {
		v, err := overEval.Eval()
		if err != nil {
			return err
		}

		switch v.Type() {
		case variant.TypeArray:
			arr := variant.MustCast[*variant.Array](v)
			if arr.Len() == 0 {
				return nil
			}

			for i, el := range arr.Slice() {
				iterArr(i, el)
				err := blkInvoker.Invoke()
				if errors.Is(err, ErrLoopBreak) {
					break
				}

				if errors.Is(err, ErrLoopContinue) {
					continue
				}

				if err != nil {
					return err
				}
			}
		case variant.TypeObject:
			obj := variant.MustCast[*variant.Object](v)
			if obj.Len() == 0 {
				return nil
			}

			var err error
			obj.IterFunc(func(k, v variant.Iface) (cont bool, brk bool) {
				iterObj(k, v)
				err = blkInvoker.Invoke()
				if errors.Is(err, ErrLoopBreak) {
					brk = true
					return
				}

				if errors.Is(err, ErrLoopContinue) {
					cont = true
					return
				}

				return
			})
		default:
			return fmt.Errorf("%s not iterable (expected array or object)", v.Type())
		}

		return nil
	}), nil
}

type IfStmtCodeGen struct {
	isLoopScope bool
	vars        *Vars
}

func (c *IfStmtCodeGen) CodeGen(node *IfStmt) (StmtInvoker, error) {
	condEval, err := (&ExprCodeGen{vars: c.vars}).CodeGen(&node.Cond)
	if err != nil {
		return nil, fmt.Errorf("bad if statement: invalid condition expression: %w", err)
	}

	blkVars := c.vars.WithScope()
	blkInvoker, err := (&BlockStmtCodeGen{vars: blkVars, isLoopScope: c.isLoopScope}).CodeGen(&node.Block)
	if err != nil {
		return nil, fmt.Errorf("bad if statement: invalid block statement: %w", err)
	}

	var elseBlkInvoker, nextIfInvoker StmtInvoker
	switch {
	case node.ElseBlock != nil:
		elseBlkVars := c.vars.WithScope()
		elseBlkInvoker, err = (&BlockStmtCodeGen{vars: elseBlkVars, isLoopScope: c.isLoopScope}).CodeGen(node.ElseBlock)
		if err != nil {
			return nil, fmt.Errorf("bad if statement: invalid else block statement: %w", err)
		}
	case node.ElseIf != nil:
		nextIfInvoker, err = (&IfStmtCodeGen{vars: c.vars, isLoopScope: c.isLoopScope}).CodeGen(node.ElseIf)
		if err != nil {
			return nil, fmt.Errorf("bad if statement: invalid else if block statement: %w", err)
		}
	}

	return invoker(func() error {
		cond, err := condEval.Eval()
		if err != nil {
			return err
		}

		if cond.Type() != variant.TypeBool {
			return errors.New("condition expression must be bool")
		}

		b := variant.MustCast[*variant.Bool](cond)
		if b.Bool() {
			return blkInvoker.Invoke()
		}

		if elseBlkInvoker != nil {
			return elseBlkInvoker.Invoke()
		}

		if nextIfInvoker != nil {
			return nextIfInvoker.Invoke()
		}

		return nil
	}), nil
}

type Program struct {
	vars *Vars
}

func (c *Program) CodeGen(node *ProgramFile) (StmtInvoker, error) {
	stmts := node.List
	if stmts == nil {
		stmts = &[]*Stmt{}
	}

	stmtInvokers := make([]StmtInvoker, 0, len(*stmts))
	for _, stmt := range *stmts {
		stmtInvoker, err := (&StmtCodeGen{vars: c.vars, isGlobalScope: true}).CodeGen(stmt)
		if err != nil {
			return nil, err
		}

		stmtInvokers = append(stmtInvokers, stmtInvoker)
	}

	return invoker(func() error {
		for _, invoker := range stmtInvokers {
			if err := invoker.Invoke(); err != nil {
				return err
			}
		}

		return nil
	}), nil
}
