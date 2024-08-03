package builtin

import (
	"errors"

	"github.com/hikitani/easylang/variant"
)

func Len(args variant.Args) (variant.Iface, error) {
	if len(args) != 1 {
		return nil, errors.New("len() takes exactly 1 argument")
	}

	switch arg := args[0]; arg := arg.(type) {
	case *variant.String:
		return variant.Int(len(arg.String())), nil
	case *variant.Array:
		return variant.Int(arg.Len()), nil
	case *variant.Object:
		return variant.Int(arg.Len()), nil
	default:
		return nil, errors.New("len() argument must be string, array, or object")
	}
}

func Str(args variant.Args) (variant.Iface, error) {
	if len(args) != 1 {
		return nil, errors.New("str() takes exactly one argument")
	}

	return variant.NewString(args[0].String()), nil
}
