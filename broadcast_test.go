package gad_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/gad-lang/gad"
	"github.com/gad-lang/gad/token"
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
		// the call form is postfix; without options it is `a .op b`
		{`return [1, 2].+(1) .* 2`, I(4, 6)},
		{`return [1, 2] .+ (1) * 2`, I(3, 4)},
		// options are values: an expression, a spread key-value array
		{`n := yes; return [1, [2]].+(1; nested=n)`, Array{Int(2), I(3)}},
		{`n := 0; return [[1], [2]].+([[3], [4]]; nested=n)`, Array{I(1, 3), I(2, 4)}},
		{`o := (; nested); return [1, [2]].+(1; **o)`, Array{Int(2), I(3)}},
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
	expectErrHas(t, `return [1].+(1; deep)`, nil, `.+: unknown option "deep"`)
}

// optionsAdd handles `.+` with its options: it returns them.
type optionsAdd struct{ Array }

func (optionsAdd) BinOpAddOptions(_ *VM, right Object, options BinOpOptions) (Object, error) {
	return Array{right, options.Dict(), Bool(options.Nested())}, nil
}

// A left operand's BinOp{Op}Options handler takes `.op` and its options.
func TestBroadcastHandler(t *testing.T) {
	r, err := Broadcast(nil, token.Add, optionsAdd{}, Int(1), BinOpOptions{"x": Int(2)}.WithNested(true))
	require.NoError(t, err)
	require.Equal(t, Array{Int(1), Dict{"x": Int(2), "nested": Yes}, True}, r)

	var o BinOpOptions
	require.False(t, o.Nested())
	require.Equal(t, "", o.Unknown(BinOpOptionNested))
	o = BinOpOptionsOf(KeyValueArray{{K: Str("nested"), V: Str("yes")}, {K: Str("b"), V: Nil}, {K: Str("a"), V: Nil}})
	require.True(t, o.Nested())
	require.Equal(t, "a", o.Unknown(BinOpOptionNested))
	o.SetNested(false)
	require.False(t, o.Nested())
}
