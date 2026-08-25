package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// TestInterfaceArrayCheck covers github.com/gad-lang/gad issue #6: an
// `interface[] P` matches an array whose elements each satisfy the members;
// `interface[][][] P` matches an array nested that deep. Checked with `::`.
func TestInterfaceArrayCheck(t *testing.T) {
	// A flat array of satisfying dicts.
	testExpectRun(t, `
		interface[] points { x int, y int }
		r := [{x: 1, y: 2}, {x: 3, y: 4}] :: points
		return len(r)`, nil, Int(2))

	// The value is returned unchanged (a pure check).
	testExpectRun(t, `
		interface[] points { x int, y int }
		return ([{x: 1, y: 2}] :: points)[0].x`, nil, Int(1))

	// Deeply nested array.
	testExpectRun(t, `
		interface[][][] nested { x int, y int }
		r := [[[{x: 1, y: 2}, {x: 3, y: 4}]]] :: nested
		return r[0][0][1].y`, nil, Int(4))

	// An empty array satisfies vacuously.
	testExpectRun(t, `
		interface[] p { x int }
		return len([] :: p)`, nil, Int(0))
}

// TestInterfaceArrayReject covers the failing cases: a non-array, an element that
// does not satisfy the members, and a wrong nesting depth.
func TestInterfaceArrayReject(t *testing.T) {
	// A non-array value does not satisfy an array interface.
	expectErrHas(t, `
		interface[] p { x int, y int }
		d := {x: 1, y: 2}
		return d :: p`, nil, "not assignable")

	// An element missing a required field is rejected.
	expectErrHas(t, `
		interface[] p { x int, y int }
		return [{x: 1}] :: p`, nil, "not assignable")

	// Too shallow for the declared depth.
	expectErrHas(t, `
		interface[][] p { x int }
		return [{x: 1}] :: p`, nil, "not assignable")
}

// TestInterfaceArrayParam covers using an array interface as a parameter type.
func TestInterfaceArrayParam(t *testing.T) {
	testExpectRun(t, `
		interface[] points { x int, y int }
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
