package builtin

import (
	"errors"

	"github.com/hikitani/easylang/variant"
)

func All(args variant.Args) (variant.Iface, error) {
	for _, arg := range args {
		v, _ := Bool(variant.Args{arg})
		if !variant.MustCast[*variant.Bool](v).Bool() {
			return variant.False(), nil
		}
	}

	return variant.True(), nil
}

func Any(args variant.Args) (variant.Iface, error) {
	for _, arg := range args {
		v, _ := Bool(variant.Args{arg})
		if variant.MustCast[*variant.Bool](v).Bool() {
			return variant.True(), nil
		}
	}

	return variant.False(), nil
}

func Iterable(args variant.Args) (variant.Iface, error) {
	if len(args) != 1 {
		return nil, errors.New("iterable() takes exactly one argument")
	}

	switch args[0].Type() {
	case variant.TypeArray, variant.TypeObject:
		return variant.True(), nil
	}

	return variant.False(), nil
}

func Bool(args variant.Args) (variant.Iface, error) {
	if len(args) != 1 {
		return nil, errors.New("bool() takes exactly one argument")
	}

	switch arg := args[0].(type) {
	case *variant.None:
		return variant.False(), nil
	case *variant.Bool:
		return arg, nil
	case *variant.Num:
		return variant.NewBool(!arg.IsZero()), nil
	case *variant.String:
		return variant.NewBool(arg.String() != ""), nil
	case *variant.Array:
		return variant.NewBool(arg.Len() != 0), nil
	case *variant.Object:
		return variant.NewBool(arg.Len() != 0), nil
	case *variant.Func:
		return variant.True(), nil
	}

	panic("unreachable")
}

func IsNumber(args variant.Args) (variant.Iface, error) {
	if len(args) != 1 {
		return nil, errors.New("is_number() takes exactly one argument")
	}

	switch args[0].Type() {
	case variant.TypeNum:
		return variant.True(), nil
	}

	return variant.False(), nil
}

func IsNone(args variant.Args) (variant.Iface, error) {
	if len(args) != 1 {
		return nil, errors.New("is_none() takes exactly one argument")
	}

	switch args[0].Type() {
	case variant.TypeNone:
		return variant.True(), nil
	}

	return variant.False(), nil
}

func IsBool(args variant.Args) (variant.Iface, error) {
	if len(args) != 1 {
		return nil, errors.New("is_bool() takes exactly one argument")
	}

	switch args[0].Type() {
	case variant.TypeBool:
		return variant.True(), nil
	}

	return variant.False(), nil
}

func IsString(args variant.Args) (variant.Iface, error) {
	if len(args) != 1 {
		return nil, errors.New("is_string() takes exactly one argument")
	}

	switch args[0].Type() {
	case variant.TypeString:
		return variant.True(), nil
	}

	return variant.False(), nil
}

func IsArray(args variant.Args) (variant.Iface, error) {
	if len(args) != 1 {
		return nil, errors.New("is_array() takes exactly one argument")
	}

	switch args[0].Type() {
	case variant.TypeArray:
		return variant.True(), nil
	}

	return variant.False(), nil
}

func IsObject(args variant.Args) (variant.Iface, error) {
	if len(args) != 1 {
		return nil, errors.New("is_object() takes exactly one argument")
	}

	switch args[0].Type() {
	case variant.TypeObject:
		return variant.True(), nil
	}

	return variant.False(), nil
}

func IsFunc(args variant.Args) (variant.Iface, error) {
	if len(args) != 1 {
		return nil, errors.New("is_func() takes exactly one argument")
	}

	switch args[0].Type() {
	case variant.TypeFunc:
		return variant.True(), nil
	}

	return variant.False(), nil
}
