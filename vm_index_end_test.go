package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// `end` in the brackets of an index or a slice is the last index (Julia's):
// len(x) - 1.
func TestVMIndexEnd(t *testing.T) {
	testExpectRun(t, `return [1, 2][end]`, nil, Int(2))
	testExpectRun(t, `a := [1, 2, 3, 4]; k := 2; return [a[end], a[end-1], a[end-k]]`, nil,
		Array{Int(4), Int(3), Int(2)})
	testExpectRun(t, `a := [1, 2, 3, 4]; return [a[1:end], a[end-1:], a[:end]]`, nil,
		Array{Array{Int(2), Int(3)}, Array{Int(3), Int(4)}, Array{Int(1), Int(2), Int(3)}})
	// the innermost brackets own it
	testExpectRun(t, `a := [1, 2, 3, 4]; b := [0, 1, 2]; return [a[b[end]], a[b[end]-end+3]]`, nil,
		Array{Int(3), Int(3)})
	testExpectRun(t, `m := [[1, 2], [3, 4, 5]]; return [m[end][end], m[0][end], m[end][0]]`, nil,
		Array{Int(5), Int(2), Int(3)})
	// what is indexed is evaluated once
	testExpectRun(t, `n := 0; f := func() { n++; return [1, 2, 3] }; return [f()[end], n]`, nil,
		Array{Int(3), Int(1)})
	testExpectRun(t, `a := [1, 2, 3]; a[end] = 30; a[end-1] += 20; return a`, nil,
		Array{Int(1), Int(22), Int(30)})
	testExpectRun(t, `d := {x: [1, 2]}; d.x[end] = 9; return d.x`, nil, Array{Int(1), Int(9)})
	// in a function inside the brackets
	testExpectRun(t, `a := [1, 2, 3]; return a[(() => end - 1)()]`, nil, Int(2))
	// `begin … end` blocks are untouched
	testExpectRun(t, "a := [1, 2, 3]\nif true begin\n  return a[end]\nend", nil, Int(3))

	expectErrHas(t, `x := end`, newOpts().CompilerError(), `expected operand`)
}
