package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// `a .op b` applies op item by item (Julia's broadcast); `a.op(b; nested)`
// descends into nested arrays.
func TestBroadcast(t *testing.T) {
	I := func(v ...int) Array {
		out := make(Array, len(v))
		for i, n := range v {
			out[i] = Int(n)
		}
		return out
	}
	B := func(v ...bool) Array {
		out := make(Array, len(v))
		for i, b := range v {
			out[i] = Bool(b)
		}
		return out
	}
	for _, c := range []struct {
		src  string
		want Object
	}{
		{`return [1, 2] .+ 2 == [3, 4]`, True},
		{`return [1, 2] .+ 2`, I(3, 4)},
		{`return [1, 2] .* [10, 100]`, I(10, 200)},
		{`return 10 .- [1, 2]`, I(9, 8)},
		{`return 1 .+ 2`, Int(3)},
		{`return [1, 2] .< 2`, B(true, false)},
		{`return [1, 2] .>= [1, 3]`, B(true, false)},
		{`return [1, 2] .== [1, 3]`, B(true, false)},
		{`return [1, 2] .!= 1`, B(false, true)},
		{`return [1, 2] .=== [1, 2u]`, B(true, false)},
		{`return [1, 2] .!== [1, 2u]`, B(false, true)},
		{`return [1, 2] .<< 1`, I(2, 4)},
		{`return [5, 6] .% 4`, I(1, 2)},
		{`return [6, 3] .& 2`, I(2, 2)},
		{`return [] .+ 1`, Array{}},
		{`return ["a", "b"] .+ "!"`, Array{Str("a!"), Str("b!")}},
		// one level: an item that is an array takes op as a whole
		{`return [[1, 2]] .< [[1, 3]]`, B(true)},
		{`return [[1], [2]] .+ [[3], [4]]`, Array{I(1, 3), I(2, 4)}},
		// nested
		{`return [1, [2, [3]]].*(2; nested)`, Array{Int(2), Array{Int(4), I(6)}}},
		{`return [1, [2]].+([10, [20]]; nested)`, Array{Int(11), I(22)}},
		{`return [1, [2]].+([10, 20]; nested=yes)`, Array{Int(11), I(22)}},
		{`return [[1], [2]].+([[3], [4]]; nested=no)`, Array{I(1, 3), I(2, 4)}},
		{`return [1, 2].+(1; nested)[1]`, Int(3)},
		// precedence: as its operator
		{`return [1, 2] .+ 1 * 2`, I(3, 4)},
		{`return [1, 2] .* 2 + [9]`, I(2, 4, 9)},
		// typed arrays broadcast; the result is an array
		{`type ints []int; return ints(1, 2) .+ ints(10, 20)`, I(11, 22)},
		// the user's overloads apply to every item
		{`met gad.binOpMul(p str, n int) => p + "x" + str(n); return ["a", "b"] .* 2`,
			Array{Str("ax2"), Str("bx2")}},
		// a `.` before a name is still a selector
		{`d := {a: [1]}; return d.a .+ 1`, I(2)},
	} {
		t.Run(c.src, func(t *testing.T) {
			testExpectRun(t, c.src, nil, c.want)
		})
	}
	expectErrHas(t, `b := [1]; return [1, 2] .+ b`, nil, ".+: arrays of different lengths (2 and 1)")
	expectErrIs(t, `b := ["a"]; return [1] .- b`, nil, ErrType)
}
