package easylang

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFoo(t *testing.T) {
	vm := New()
	stmt, err := vm.Compile(strings.NewReader(`
		res = range(10, 100, 3)
			.where(|v| => v % 2 == 0)
			.select(|v| => v * 2)
			.max(10)
			.list()

		obj = {
			"arr": res,
		}
		println(obj)
	`))
	require.NoError(t, err)

	stmt.Invoke()
}
