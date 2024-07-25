package builtin

import (
	"fmt"
	"os"

	"github.com/hikitani/easylang/variant"
)

func void() (variant.Iface, error) {
	return variant.NewNone(), nil
}

func Print(args variant.Args) (variant.Iface, error) {
	args.Print(os.Stdout)
	return void()
}

func Println(args variant.Args) (variant.Iface, error) {
	args.Print(os.Stdout)
	fmt.Println()
	return void()
}
