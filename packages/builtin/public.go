package builtin

import (
	"github.com/hikitani/easylang/packages"
)

var Package = packages.
	New("builtin").
	AddFunc("print", Print).
	AddFunc("println", Println).
	AddFunc("all", All).
	AddFunc("any", Any).
	AddFunc("sum", Sum).
	AddFunc("len", Len).
	AddFunc("min", Min).
	AddFunc("max", Max).
	AddFunc("abs", Abs).
	AddFunc("iterable", Iterable).
	AddFunc("bool", Bool).
	AddFunc("is_none", IsNone).
	AddFunc("is_bool", IsBool).
	AddFunc("is_number", IsNumber).
	AddFunc("is_string", IsString).
	AddFunc("is_array", IsArray).
	AddFunc("is_object", IsObject).
	AddFunc("is_func", IsFunc).
	AddFunc("str", Str).
	AddFunc("pow", Pow).
	Build()
