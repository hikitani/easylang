package builtin

import (
	"errors"
	"fmt"

	"github.com/hikitani/easylang/variant"
)

func Min(args variant.Args) (variant.Iface, error) {
	if len(args) == 0 {
		return variant.NewNone(), nil
	}

	min := args[0]
	typ := args[0].Type()
	switch typ {
	case variant.TypeNum, variant.TypeString:
	default:
		return nil, errors.New("min() arguments must be number or string")
	}

	for _, arg := range args[1:] {
		if arg.Type() != typ {
			return nil, fmt.Errorf("types mismatch: %s != %s", typ, arg.Type())
		}

		switch typ {
		case variant.TypeNum:
			a, b := variant.MustCast[*variant.Num](min), variant.MustCast[*variant.Num](arg)
			if b.LessThan(a) {
				min = arg
			}
		case variant.TypeString:
			a, b := variant.MustCast[*variant.String](min), variant.MustCast[*variant.String](arg)
			if b.String() < a.String() {
				min = arg
			}
		default:
			return nil, errors.New("min() arguments must be number or string")
		}
	}

	return min, nil
}

func Max(args variant.Args) (variant.Iface, error) {
	if len(args) == 0 {
		return variant.NewNone(), nil
	}

	max := args[0]
	typ := args[0].Type()
	switch typ {
	case variant.TypeNum, variant.TypeString:
	default:
		return nil, errors.New("max() arguments must be number or string")
	}

	for _, arg := range args[1:] {
		if arg.Type() != typ {
			return nil, fmt.Errorf("types mismatch: %s != %s", typ, arg.Type())
		}

		switch typ {
		case variant.TypeNum:
			a, b := variant.MustCast[*variant.Num](max), variant.MustCast[*variant.Num](arg)
			if b.GreaterThan(a) {
				max = arg
			}
		case variant.TypeString:
			a, b := variant.MustCast[*variant.String](max), variant.MustCast[*variant.String](arg)
			if b.String() > a.String() {
				max = arg
			}
		default:
			return nil, errors.New("max() arguments must be number or string")
		}
	}

	return max, nil
}

func Abs(args variant.Args) (variant.Iface, error) {
	if len(args) != 1 {
		return nil, errors.New("abs() takes exactly one argument")
	}

	if args[0].Type() != variant.TypeNum {
		return nil, errors.New("abs() argument must be number")
	}

	a := variant.MustCast[*variant.Num](args[0])
	return a.Abs(), nil
}

func Sum(args variant.Args) (variant.Iface, error) {
	s := variant.Int(0)
	for _, arg := range args {
		if arg.Type() != variant.TypeNum {
			return nil, errors.New("sum() arguments must be number")
		}

		a := variant.MustCast[*variant.Num](arg)
		s.Value().Add(s.Value(), a.Value())
	}

	return s, nil
}

func Pow(args variant.Args) (variant.Iface, error) {
	if len(args) != 2 {
		return nil, errors.New("pow() takes exactly two arguments")
	}

	if args[0].Type() != variant.TypeNum {
		return nil, errors.New("pow() first argument must be number")
	}

	if args[1].Type() != variant.TypeNum {
		return nil, errors.New("pow() second argument must be number")
	}

	a, b := variant.MustCast[*variant.Num](args[0]), variant.MustCast[*variant.Num](args[1])
	if a.Sign() < 0 {
		return nil, errors.New("pow() first argument must be positive")
	}

	return a.Pow(b), nil
}
