package builtin

import (
	"errors"

	"github.com/hikitani/easylang/variant"
)

func StrBytes(args variant.Args) (variant.Iface, error) {
	if len(args) != 1 {
		return nil, errors.New("str_bytes() takes exactly one argument")
	}

	if args[0].Type() != variant.TypeString {
		return nil, errors.New("str_bytes() takes string as argument")
	}

	return variant.MustCast[*variant.String](args[0]).AsBytes(), nil
}

func Str(args variant.Args) (variant.Iface, error) {
	if len(args) != 1 {
		return nil, errors.New("str() takes exactly one argument")
	}

	return variant.NewString(args[0].String()), nil
}
