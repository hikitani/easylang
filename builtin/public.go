package builtin

import "github.com/hikitani/easylang/variant"

func Objects() map[string]variant.Iface {
	return map[string]variant.Iface{
		"print":     variant.NewFunc(nil, Print),
		"println":   variant.NewFunc(nil, Println),
		"all":       variant.NewFunc(nil, All),
		"any":       variant.NewFunc(nil, Any),
		"sum":       variant.NewFunc(nil, Sum),
		"len":       variant.NewFunc([]string{"v"}, Len),
		"min":       variant.NewFunc([]string{"v"}, Min),
		"max":       variant.NewFunc([]string{"v"}, Max),
		"abs":       variant.NewFunc([]string{"v"}, Abs),
		"iterable":  variant.NewFunc([]string{"v"}, Iterable),
		"bool":      variant.NewFunc([]string{"v"}, Bool),
		"is_none":   variant.NewFunc([]string{"v"}, IsNone),
		"is_bool":   variant.NewFunc([]string{"v"}, IsBool),
		"is_number": variant.NewFunc([]string{"v"}, IsNumber),
		"is_string": variant.NewFunc([]string{"v"}, IsString),
		"is_array":  variant.NewFunc([]string{"v"}, IsArray),
		"is_object": variant.NewFunc([]string{"v"}, IsObject),
		"is_func":   variant.NewFunc([]string{"v"}, IsFunc),
		"str":       variant.NewFunc([]string{"v"}, Str),
		"pow":       variant.NewFunc([]string{"base", "exp"}, Pow),
		"iter":      variant.NewFunc([]string{"iterable"}, Iter),
		"range":     variant.NewFunc(nil, Range),
	}
}
