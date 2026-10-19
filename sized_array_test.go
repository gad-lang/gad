package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// Array types with sizes by level: `[2]T`, `[..3]T`, `[2..]T`, `[2..3]T`.
func TestSizedArrayTypes(t *testing.T) {
	// casts
	expectErrHas(t, `return [1, 2, 3] :: [2]int`, newOpts(), "level 1 has 3 items, wants exactly 2")
	testExpectRun(t, `return [1, 2, 3] :: [2..3]int`, nil, Array{Int(1), Int(2), Int(3)})
	testExpectRun(t, `return [[1, 2], [3, 4]] :: [][2]int`, nil, Array{Array{Int(1), Int(2)}, Array{Int(3), Int(4)}})
	expectErrHas(t, `return [[1, 2], [3]] :: [][2]int`, newOpts(), "level 2 has 1 items")
	expectErrHas(t, `return [1] :: [2..]int`, newOpts(), "level 1 has 1 items")
	expectErrHas(t, `return [1, 2, 3, 4] :: [..3]int`, newOpts(), "level 1 has 4 items")
	testExpectRun(t, `return [] :: [..3]int`, nil, Array{})
	// arithmetic of literals
	testExpectRun(t, `return len([1, 2, 3] :: [1+2]int)`, nil, Int(3))

	// params
	expectErrHas(t, `return ((v [2]int) => nil)([1, 2, 3])`, newOpts(), "[2]int")
	testExpectRun(t, `return ((v [2]int) => v[1])([1, 2])`, nil, Int(2))

	// a size by a variable, read when the value is checked
	testExpectRun(t, `n := 2; return len([1, 2] :: [n]int)`, nil, Int(2))
	expectErrHas(t, `n := 2; return [1, 2, 3] :: [n]int`, newOpts(), "wants exactly 2")
	testExpectRun(t, `f := (n) => ((v [n]int) => len(v)); return f(3)([1, 2, 3])`, nil, Int(3))
	expectErrHas(t, `n := "x"; return [1] :: [n]int`, newOpts(), "is not an int")
	expectErrHas(t, `n := -1; return [1] :: [n]int`, newOpts(), "is negative")

	// reflection
	testExpectRun(t, `const t = type [2..3][2]str; return [t.@size[0] == [nil, 2, 3], t.@size[1].eq, t.@size[1].min, t.@depth]`,
		nil, Array{True, Int(2), Nil, Int(2)})
	testExpectRun(t, `return [(type []int).@size[0] == [nil, nil, nil], (type []int).@elem[0] == int]`, nil, Array{True, True})
	testExpectRun(t, `return str(type [2..3][2]str)`, nil, Str("[2..3][2]str"))

	// a closure: the types of its params read its free variables
	testExpectRun(t, `f := (T) => { return (v []T) => len(v) }; return f(int)([1, 2])`, nil, Int(2))
	expectErrHas(t, `f := (T) => { return (v []T) => len(v) }; return f(int)(["a"])`, newOpts(), "expected []T")

	// compile errors
	expectCompileError(t, `return [1] :: [3..2]int`, "the minimum 3 is greater than the maximum 2")
	expectCompileError(t, `return [1] :: [-1]int`, "is negative")
	expectCompileError(t, `return [1] :: [len(x)]int`, "a size is an int")
}

// A typed array type with sizes: `type pair [2]str` — the first level is the
// number of items, kept on append and delete.
func TestSizedTypedArrayTypes(t *testing.T) {
	testExpectRun(t, `type pair [2]str; return str(pair("a", "b"))`, nil, Str(`pair["a", "b"]`))
	expectErrHas(t, `type pair [2]str; return pair("a")`, newOpts(), "pair: 1 items, wants exactly 2")
	expectErrHas(t, `type pair [2]str; return ["a", "b", "c"] ::> pair`, newOpts(), "pair: 3 items, wants exactly 2")
	testExpectRun(t, `type pair [2]str; return str([1, 2] ::> pair)`, nil, Str(`pair["1", "2"]`))
	// append and delete
	expectErrHas(t, `type pair [2]str; p := pair("a", "b"); p += "c"`, newOpts(), "pair: 3 items, wants exactly 2")
	expectErrHas(t, `type pair [2]str; p := pair("a", "b"); delete p[0]`, newOpts(), "pair: 1 items, wants exactly 2")
	testExpectRun(t, `type upTo3 [..3]int; p := upTo3(1); p += 2; p += 3; return len(p)`, nil, Int(3))
	expectErrHas(t, `type upTo3 [..3]int; p := upTo3(1, 2, 3); p += 4`, newOpts(), "wants at most 3")
	expectErrHas(t, `type upTo3 [..3]int; p := upTo3(1, 2, 3); return p + 4`, newOpts(), "wants at most 3")
	// a slice that leaves the size is an array
	testExpectRun(t, `type pair [2]str; return typeName(pair("a", "b")[:1])`, nil, Str("array"))
	// the levels of the items
	testExpectRun(t, `type options [][2]str; return len(options(["a", "A"], ["b", "B"]))`, nil, Int(2))
	expectErrHas(t, `type options [][2]str; return options(["a"])`, newOpts(), "level 2 has 1 items")
	// a size by a variable
	testExpectRun(t, `n := 2; type ns [n]int; return len(ns(1, 2))`, nil, Int(2))
	// reflection
	testExpectRun(t, `type options [..5][2]str; return [options.@size[0] == [nil, nil, 5], options.@size[1].eq, options.@depth]`,
		nil, Array{True, Int(2), Int(2)})
	testExpectRun(t, `type ints []int; return [ints.@size[0] == [nil, nil, nil], ints.@elem == int]`, nil, Array{True, True})
}

// `::>` into an array type: the sizes checked, the leaves transformed.
func TestArrayTypeTransform(t *testing.T) {
	testExpectRun(t, `return [1, 2] ::> []str`, nil, Array{Str("1"), Str("2")})
	testExpectRun(t, `return [[1, 2], [3, 4]] ::> [][2]str`, nil, Array{Array{Str("1"), Str("2")}, Array{Str("3"), Str("4")}})
	expectErrHas(t, `return [[1, 2], [3]] ::> [][2]str`, newOpts(), "level 2 has 1 items, wants exactly 2")
	expectErrHas(t, `return [1, 2, 3] ::> [2]str`, newOpts(), "level 1 has 3 items, wants exactly 2")
	testExpectRun(t, `return [1, "a"] ::> []<int|str>`, nil, Array{Int(1), Str("a")})
	testExpectRun(t, `type ints []int; return ints(1, 2) ::> [2]str`, nil, Array{Str("1"), Str("2")})
	expectErrHas(t, `return 1 ::> []str`, newOpts(), "level 1 is not an array")
}

// Overloads told apart by the sizes: exactly a size first, then the
// narrowest range, then no size, then the order of declaration.
func TestSizedArrayOverloads(t *testing.T) {
	testExpectRun(t, `f := func { (a []int) => 1; (a [2]int) => 2 }; return [f([1, 2]), f([1, 2, 3]), f([])]`,
		nil, Array{Int(2), Int(1), Int(1)})
	testExpectRun(t, `f := func { (a [1..9]int) => "wide"; (a [2..3]int) => "narrow"; (a [3]int) => "exact"; (a []int) => "any" }
		return [f([1, 2, 3]), f([1, 2]), f([1, 2, 3, 4]), f([]), f([1, 2, 3, 4, 5, 6, 7, 8, 9, 10])]`,
		nil, Array{Str("exact"), Str("narrow"), Str("wide"), Str("any"), Str("any")})
	// not an array: the other overloads
	testExpectRun(t, `f := func { (a [2]int) => "pair"; (a int) => "int" }; return [f([1, 2]), f(1)]`,
		nil, Array{Str("pair"), Str("int")})
	// none accepts: the error of the first
	expectErrHas(t, `f := func { (a [2]int) => 2; (a [3]int) => 3 }; return f([1])`, newOpts(), "expected [2]int")
	// the same types twice is still an error
	expectErrHas(t, `f := func { (a [2]int) => 1; (a [2]int) => 2 }; return f([1, 2])`, newOpts(), "ErrMethodDuplication")
	// with other params
	testExpectRun(t, `f := func { (a [2]int, b str) => 2; (a []int, b str) => 0 }; return [f([1, 2], "x"), f([1], "x")]`,
		nil, Array{Int(2), Int(0)})
}
