package easylang

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFoo(t *testing.T) {
	vm := New()
	stmt, err := vm.Compile("", strings.NewReader(`
		using iter

		res = iter.
			range(10, 100, 3).
			where(|v| => v % 2 == 0).
			select(|v| => v * 2).
			max(10).
			list()

		res = iter.from(res).max(5).list()

		obj = {
			"arr": res,
			"cnt": iter.range(100).count()
		}
		println(obj.cnt)

		print((import "bar.ela").s)
	`))
	require.NoError(t, err)

	require.NoError(t, stmt.Invoke())
}
