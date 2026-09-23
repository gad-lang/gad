package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// TestSliceTypeInterfaceField covers `[]T` as an interface field's type: an
// array nested to the written depth whose leaves are of the element types.
func TestSliceTypeInterfaceField(t *testing.T) {
	cast := func(decl, value string) string {
		return `T := interface { xs ` + decl + ` }
			try { ({ xs: ` + value + ` }) :: T; return "ok" } catch { return "no" }`
	}

	// `[]int`: every element must be an int.
	testExpectRun(t, cast(`[]int`, `[1, 2, 3]`), nil, Str("ok"))
	testExpectRun(t, cast(`[]int`, `[1, "a"]`), nil, Str("no"))
	// not an array at all
	testExpectRun(t, cast(`[]int`, `1`), nil, Str("no"))
	// an empty array has no element to reject
	testExpectRun(t, cast(`[]int`, `[]`), nil, Str("ok"))

	// `[][]int`: an array OF arrays, so the depth is checked too.
	testExpectRun(t, cast(`[][]int`, `[[1], [2, 3]]`), nil, Str("ok"))
	testExpectRun(t, cast(`[][]int`, `[1, 2]`), nil, Str("no"))
	testExpectRun(t, cast(`[][][]int`, `[[[1]]]`), nil, Str("ok"))
	testExpectRun(t, cast(`[][][]int`, `[[1]]`), nil, Str("no"))

	// `[]<int|str>`: an element matches ANY of the enveloped types.
	testExpectRun(t, cast(`[]<int|str>`, `[1, "a"]`), nil, Str("ok"))
	testExpectRun(t, cast(`[]<int|str>`, `[1, true]`), nil, Str("no"))
	// `[]<int>` is the long form of `[]int`
	testExpectRun(t, cast(`[]<int>`, `[1]`), nil, Str("ok"))
	testExpectRun(t, cast(`[]<int>`, `["a"]`), nil, Str("no"))

	// `?` after the name makes the FIELD nullable, not its elements.
	testExpectRun(t, `T := interface { xs? []int }
		try { ({}) :: T; return "ok" } catch { return "no" }`, nil, Str("ok"))
	testExpectRun(t, `T := interface { xs? []int }
		try { ({ xs: ["a"] }) :: T; return "ok" } catch { return "no" }`, nil, Str("no"))
}

// TestFuncHeaderType covers a function header used where a type goes: the value
// must be a CALLABLE whose signature the header matches.
func TestFuncHeaderType(t *testing.T) {
	cast := func(value string) string {
		return `T := interface { f <(x int) <ret any>> }
			try { ({ f: ` + value + ` }) :: T; return "ok" } catch { return "no" }`
	}
	testExpectRun(t, cast(`func(x int) => x`), nil, Str("ok"))
	testExpectRun(t, cast(`1`), nil, Str("no"))
	// wrong arity
	testExpectRun(t, cast(`func(x int, y int) => x`), nil, Str("no"))

	// As a slice element it keeps its envelope — `[]<(x int) …>` — and among
	// several types it is enveloped like any other (`[]<<(x int)>|str>`).
	testExpectRun(t, `T := interface { fs []<(x int) <ret any>> }
		try { ({ fs: [func(x int) => x] }) :: T; return "ok" } catch { return "no" }`,
		nil, Str("ok"))
	testExpectRun(t, `T := interface { fs []<(x int) <ret any>> }
		try { ({ fs: [1] }) :: T; return "ok" } catch { return "no" }`,
		nil, Str("no"))
	testExpectRun(t, `T := interface { fs []<<(x int)>|str> }
		try { ({ fs: [func(x int) => x, "a"] }) :: T; return "ok" } catch { return "no" }`,
		nil, Str("ok"))
}

// TestSliceTypeOtherPositions covers the positions besides an interface field
// where a type is written: a function parameter, a class field and a `param`
// declaration. The expression parser reads `[` as an index and `<` as a
// comparison, so each of these needs the type to be recognized first.
func TestSliceTypeOtherPositions(t *testing.T) {
	// A function parameter: the argument is checked on the call.
	testExpectRun(t, `
		func f(xs []int) => xs[0]
		return f([7, 8])`, nil, Int(7))
	testExpectRun(t, `
		func f(xs []int) => xs[0]
		try { f(["a"]) ; return "ok" } catch { return "no" }`, nil, Str("no"))
	// A function-header parameter type.
	testExpectRun(t, `
		func g(cb <(x int)>) => cb(2)
		return g(func(x int) => x * 3)`, nil, Int(6))
	testExpectRun(t, `
		func g(cb <(x int)>) => cb(2)
		try { g(1); return "ok" } catch { return "no" }`, nil, Str("no"))

	// A class field: the value is checked when the instance is built.
	testExpectRun(t, `
		class C { xs []int = [] }
		return C(; xs = [1, 2]).xs[1]`, nil, Int(2))
	testExpectRun(t, `
		class C { xs []int = [] }
		try { C(; xs = 1); return "ok" } catch { return "no" }`, nil, Str("no"))

	// A `param` declaration, in both the single and the parenthesized form.
	testExpectRun(t, `param xs []int
		return xs[0]`, newOpts().Args(Array{Int(4)}), Int(4))
	testExpectRun(t, `param (xs []int, n int)
		return xs[0] + n`, newOpts().Args(Array{Int(4)}, Int(1)), Int(5))
}

// TestSliceTypeAmbiguity pins the expressions a structural type must NOT steal:
// `[` stays an index and `<` stays a comparison.
func TestSliceTypeAmbiguity(t *testing.T) {
	// an index written with a space before `[`
	testExpectRun(t, `xs := [10, 20]; return xs [1]`, nil, Int(20))
	// a comparison whose right side is parenthesized, and a chained one
	testExpectRun(t, `a := 5; b := 2; c := 1; return [a < (b), a < (b) > c]`,
		nil, Array{False, False})
	// the same shapes as call arguments, where a parameter list is parsed too
	testExpectRun(t, `
		func f(v) => v
		a := 5; b := 2; c := 1
		return [f(a < (b)), f(a < (b) > c)]`, nil, Array{False, False})
}

// TestNamedSliceType covers the named slice-type declarations:
//
//	type numerics []<int|uint|float>   // a named slice of types
//	type users []{ name; id }          // a named slice interface
func TestNamedSliceType(t *testing.T) {
	// A named slice of types: usable as a cast target and a parameter type.
	testExpectRun(t, `
		type numerics []<int|uint|float>
		try { [1, 2.5, 3] :: numerics; return "ok" } catch { return "no" }`,
		nil, Str("ok"))
	testExpectRun(t, `
		type numerics []<int|uint|float>
		try { [1, "x"] :: numerics; return "ok" } catch { return "no" }`,
		nil, Str("no"))
	testExpectRun(t, `
		type numerics []<int|uint|float>
		func f(xs numerics) => len(xs)
		return f([1, 2, 3])`, nil, Int(3))

	// A named slice interface: each element must satisfy the inline interface.
	testExpectRun(t, `
		type users []{ name; id }
		try { [{name: "a", id: 1}, {name: "b", id: 2}] :: users; return "ok" } catch { return "no" }`,
		nil, Str("ok"))
	testExpectRun(t, `
		type users []{ name; id }
		try { [{name: "a"}] :: users; return "ok" } catch { return "no" }`,
		nil, Str("no"))
	testExpectRun(t, `
		type users []{ name; id }
		func g(us users) => len(us)
		return g([{name: "a", id: 1}])`, nil, Int(1))
}
