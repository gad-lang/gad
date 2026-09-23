package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// TestInterfaceArrayCheck covers github.com/gad-lang/gad issue #6: an
// `interface P []` matches an array whose elements each satisfy the members;
// `interface P [][][]` matches an array nested that deep. Checked with `::`.
func TestInterfaceArrayCheck(t *testing.T) {
	// A flat array of satisfying dicts.
	testExpectRun(t, `
		interface points [] { x int, y int }
		r := [{x: 1, y: 2}, {x: 3, y: 4}] :: points
		return len(r)`, nil, Int(2))

	// The value is returned unchanged (a pure check).
	testExpectRun(t, `
		interface points [] { x int, y int }
		return ([{x: 1, y: 2}] :: points)[0].x`, nil, Int(1))

	// Deeply nested array.
	testExpectRun(t, `
		interface nested [][][] { x int, y int }
		r := [[[{x: 1, y: 2}, {x: 3, y: 4}]]] :: nested
		return r[0][0][1].y`, nil, Int(4))

	// An empty array satisfies vacuously.
	testExpectRun(t, `
		interface p [] { x int }
		return len([] :: p)`, nil, Int(0))
}

// TestInterfaceArrayReject covers the failing cases: a non-array, an element that
// does not satisfy the members, and a wrong nesting depth.
func TestInterfaceArrayReject(t *testing.T) {
	// A non-array value does not satisfy an array interface.
	expectErrHas(t, `
		interface p [] { x int, y int }
		d := {x: 1, y: 2}
		return d :: p`, nil, "not assignable")

	// An element missing a required field is rejected.
	expectErrHas(t, `
		interface p [] { x int, y int }
		return [{x: 1}] :: p`, nil, "not assignable")

	// Too shallow for the declared depth.
	expectErrHas(t, `
		interface p [][] { x int }
		return [{x: 1}] :: p`, nil, "not assignable")
}

// TestInterfaceArrayParam covers using an array interface as a parameter type.
func TestInterfaceArrayParam(t *testing.T) {
	testExpectRun(t, `
		interface points [] { x int, y int }
		f := func(ps points) => len(ps)
		return f([{x: 1, y: 2}, {x: 3, y: 4}])`, nil, Int(2))
}

// TestInterfaceArrayTransform covers `:::` on an array interface: each leaf is
// transformed (its class-typed fields are built), at any nesting depth.
func TestInterfaceArrayTransform(t *testing.T) {
	base := "class Point { x int; y int }\n"

	// Each element's `p` dict becomes a Point.
	testExpectRun(t, base+`
		wrapped := [{p: {x: 1, y: 2}}, {p: {x: 3, y: 4}}] ::: interface[] { p Point }
		return [typeName(wrapped[0].p), wrapped[1].p.y]`,
		nil, Array{Str("Point"), Int(4)})

	// Nested arrays coerce their leaves too.
	testExpectRun(t, base+`
		deep := [[[{p: {x: 9, y: 8}}]]] ::: interface[][][] { p Point }
		return [typeName(deep[0][0][0].p), deep[0][0][0].p.x]`,
		nil, Array{Str("Point"), Int(9)})
}

// TestInterfaceSliceElemTypes covers the array-of-types interface — the named
// analogue of an anonymous `[]<int|uint>` type: `interface P []<int|uint>` (or a
// bare `interface P []int`) matches an array whose leaves match the element types.
func TestInterfaceSliceElemTypes(t *testing.T) {
	sat := `sat := func(v, T) { try { v :: T; return true } catch { return false } }
	`
	testExpectRun(t, sat+`
		interface numerics []<int|uint|float>
		interface grid [][]int
		return [sat([1, 2u, 3.5], numerics), sat([1, "x"], numerics), sat(1, numerics),
			sat([[1], [2, 3]], grid), sat([1], grid)]`,
		nil, Array{True, False, False, True, False})

	// The long form `[] interface { … }` is the same as `[] { … }`.
	testExpectRun(t, sat+`
		interface users [] interface { name; id }
		return [sat([{name: "a", id: 1}], users), sat([{name: "a"}], users)]`,
		nil, Array{True, False})

	// Reflection: @depth, @elem, @meta, and the shape in the interface's string.
	testExpectRun(t, `
		[kind="nums"]
		interface numerics []<int|float>
		interface pts [] { x int }
		return [numerics.@depth, len(numerics.@elem), str(numerics.@meta), pts.@elem,
			str(numerics)[-13:]]`,
		nil, Array{Int(1), Int(2), Str(`(;kind="nums")`), Nil, Str("[]<int|float>")})

	// `:::` converts an array (checking each leaf) and passes a typed array through.
	testExpectRun(t, `
		interface nums []<int|float>
		return str([1, 2.5] ::: nums)`, nil, Str("[1, 2.5]"))
	testExpectRun(t, `
		interface nums []<int|float>
		try { [1, "x"] ::: nums; return "ok" } catch { return "no" }`, nil, Str("no"))
}
