package builtin

import (
	"errors"

	"github.com/hikitani/easylang/variant"
)

var ErrStopIteration = errors.New("StopIteration")

func NextIterator(v variant.Iface) (*variant.Func, error) {
	switch v := v.(type) {
	case *variant.Array:
		i := int64(0)
		return variant.NewFunc(
			[]string{}, func(args variant.Args) (variant.Iface, error) {
				if len(args) != 0 {
					return nil, errors.New("next() takes no arguments")
				}

				if i >= int64(v.Len()) {
					return nil, ErrStopIteration
				}

				elem, err := v.Get(i)
				if err != nil {
					return nil, err
				}
				i++

				return elem, nil
			},
		), nil
	case *variant.Object:
		keys, vals := v.Items()
		i := 0
		return variant.NewFunc(
			[]string{}, func(args variant.Args) (variant.Iface, error) {
				if len(args) != 0 {
					return nil, errors.New("next() takes no arguments")
				}

				if i >= len(keys) {
					return nil, ErrStopIteration
				}

				k, v := keys[i], vals[i]
				i++
				return variant.MustNewObject(
					[]variant.Iface{
						variant.NewString("key"),
						variant.NewString("value"),
					},
					[]variant.Iface{k, v},
				), nil
			},
		), nil
	}

	return nil, errors.New("argument must be an array or object")
}

func iterList(nextFn *variant.Func) *variant.Func {
	return variant.NewFunc([]string{}, func(args variant.Args) (variant.Iface, error) {
		if len(args) != 0 {
			return nil, errors.New("list() takes no arguments")
		}

		var elems []variant.Iface
		for {
			elem, err := nextFn.Call(nil)
			if errors.Is(err, ErrStopIteration) {
				break
			}

			if err != nil {
				return nil, err
			}

			elems = append(elems, elem)
		}

		return variant.NewArray(elems), nil
	})
}

func iterMax(nextFn *variant.Func) variant.Iface {
	return variant.NewFunc([]string{"max"}, func(args variant.Args) (variant.Iface, error) {
		if len(args) != 1 {
			return nil, errors.New("max() takes exactly one argument")
		}

		if args[0].Type() != variant.TypeNum {
			return nil, errors.New("max() takes a number")
		}

		max, err := variant.MustCast[*variant.Num](args[0]).AsInt64()
		if err != nil {
			return nil, err
		}

		i := int64(0)
		return iterObject(variant.NewFunc([]string{}, func(args variant.Args) (variant.Iface, error) {
			elem, err := nextFn.Call(nil)
			if errors.Is(err, ErrStopIteration) {
				return nil, ErrStopIteration
			}

			if err != nil {
				return nil, err
			}

			if i >= max {
				return nil, ErrStopIteration
			}

			i++
			return elem, nil
		})), nil
	})
}

func iterWhere(nextFn *variant.Func) variant.Iface {
	return variant.NewFunc([]string{"predicate"}, func(args variant.Args) (variant.Iface, error) {
		if len(args) != 1 {
			return nil, errors.New("where() takes exactly one argument")
		}

		if args[0].Type() != variant.TypeFunc {
			return nil, errors.New("where() takes a function")
		}

		predicate := variant.MustCast[*variant.Func](args[0])
		if len(predicate.Idents()) != 1 {
			return nil, errors.New("predicate must take exactly one argument")
		}

		return iterObject(variant.NewFunc([]string{}, func(args variant.Args) (variant.Iface, error) {
			for {
				elem, err := nextFn.Call(nil)
				if errors.Is(err, ErrStopIteration) {
					return nil, ErrStopIteration
				}

				if err != nil {
					return nil, err
				}

				res, err := predicate.Call(variant.Args{elem})
				if err != nil {
					return nil, err
				}

				if res.Type() != variant.TypeBool {
					return nil, errors.New("predicate must return a bool")
				}

				if ok := variant.MustCast[*variant.Bool](res).Bool(); ok {
					return elem, nil
				}
			}
		})), nil
	})
}

func iterSelect(nextFn *variant.Func) variant.Iface {
	return variant.NewFunc([]string{"selector"}, func(args variant.Args) (variant.Iface, error) {
		if len(args) != 1 {
			return nil, errors.New("select() takes exactly one argument")
		}

		if args[0].Type() != variant.TypeFunc {
			return nil, errors.New("select() takes a selector function")
		}

		selector := variant.MustCast[*variant.Func](args[0])
		if len(selector.Idents()) != 1 {
			return nil, errors.New("selector must take exactly one argument")
		}

		return iterObject(variant.NewFunc([]string{}, func(args variant.Args) (variant.Iface, error) {
			elem, err := nextFn.Call(nil)
			if errors.Is(err, ErrStopIteration) {
				return nil, ErrStopIteration
			}

			if err != nil {
				return nil, err
			}

			return selector.Call(variant.Args{elem})
		})), nil
	})
}

func iterObject(nextV *variant.Func) *variant.Object {
	return variant.MustNewObject(
		[]variant.Iface{
			variant.NewString("list"),
			variant.NewString("max"),
			variant.NewString("where"),
			variant.NewString("select"),
		},
		[]variant.Iface{
			iterList(nextV),
			iterMax(nextV),
			iterWhere(nextV),
			iterSelect(nextV),
		},
	)
}

func Range(args variant.Args) (variant.Iface, error) {
	var (
		iterator *variant.Func
		err      error
	)
	switch len(args) {
	case 1:
		if args[0].Type() != variant.TypeNum {
			return nil, errors.New("range() first argument must be number")
		}
		iterator, err = rangeIterator(
			variant.Int(0),
			variant.MustCast[*variant.Num](args[0]),
			variant.Int(1),
		)
	case 2:
		if args[0].Type() != variant.TypeNum {
			return nil, errors.New("range() first argument must be number")
		}
		if args[1].Type() != variant.TypeNum {
			return nil, errors.New("range() second argument must be number")
		}

		iterator, err = rangeIterator(
			variant.MustCast[*variant.Num](args[0]),
			variant.MustCast[*variant.Num](args[1]),
			variant.Int(1),
		)
	case 3:
		if args[0].Type() != variant.TypeNum {
			return nil, errors.New("range() first argument must be number")
		}
		if args[1].Type() != variant.TypeNum {
			return nil, errors.New("range() second argument must be number")
		}
		if args[2].Type() != variant.TypeNum {
			return nil, errors.New("range() third argument must be number")
		}
		iterator, err = rangeIterator(
			variant.MustCast[*variant.Num](args[0]),
			variant.MustCast[*variant.Num](args[1]),
			variant.MustCast[*variant.Num](args[2]),
		)
	default:
		return nil, errors.New("expected range(start), range(start, stop) or range(start, stop, step)")
	}
	if err != nil {
		return nil, err
	}

	return iterObject(iterator), nil
}

func rangeIterator(start, stop, step *variant.Num) (*variant.Func, error) {
	if step.IsZero() {
		return nil, errors.New("step cannot be zero")
	}

	var condition func(*variant.Num) bool
	if step.LessThan(variant.Int(0)) {
		if start.LessThan(stop) {
			return variant.NewFunc([]string{}, func(args variant.Args) (variant.Iface, error) {
				return nil, ErrStopIteration
			}), nil
		}

		condition = start.LessOrEqualTo
	} else {
		if start.GreaterThan(stop) {
			return variant.NewFunc([]string{}, func(args variant.Args) (variant.Iface, error) {
				return nil, ErrStopIteration
			}), nil
		}

		condition = start.GreaterOrEqualTo
	}

	return variant.NewFunc([]string{}, func(args variant.Args) (variant.Iface, error) {
		if condition(stop) {
			return nil, ErrStopIteration
		}

		v := start.Copy()
		start.Add(step)
		return v, nil
	}), nil
}

func Iter(args variant.Args) (variant.Iface, error) {
	if len(args) != 1 {
		return nil, errors.New("iter() takes exactly one argument")
	}

	switch args[0].Type() {
	case variant.TypeArray, variant.TypeObject:
	default:
		return nil, errors.New("first argument must be an array or object")
	}

	nextV, err := NextIterator(args[0])
	if err != nil {
		panic("unreachable")
	}

	return iterObject(nextV), nil
}
