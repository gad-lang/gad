package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// TestFilterSkipsOnlyRejected guards IteratorStateCheck: after a filter callback
// rejected an element, iterators whose Next does not reset the state mode (such
// as a Range's) skipped every following element too, so filtering a range
// yielded nothing past the first rejection.
func TestFilterSkipsOnlyRejected(t *testing.T) {
	even := `(v, k, c) => v % 2 == 0`
	for _, c := range []struct {
		src  string
		want Object
	}{
		{`return collect(filter(1 .. 6, ` + even + `))`, Array{Int(2), Int(4), Int(6)}},
		{`return collect(filter(values(1 .. 4), ` + even + `))`, Array{Int(2), Int(4)}},
		{`return collect(filter([1, 2, 3, 4], ` + even + `))`, Array{Int(2), Int(4)}},
		{`return collect(map(filter(1 .. 6, ` + even + `), (v) => v * v; nokey))`,
			Array{Int(4), Int(16), Int(36)}},
	} {
		testExpectRun(t, c.src, nil, c.want)
	}
}

// TestIsChecksEveryValue guards the `is` builtin: every value must be of the
// type (it used to check only the first).
func TestIsChecksEveryValue(t *testing.T) {
	testExpectRun(t, `return [is(int, 1, 2), is(int, 1, "x"), is(int, "x", 1), is([int, str], 1, "x")]`,
		nil, Array{True, False, False, True})
}
