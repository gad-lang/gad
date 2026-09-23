package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// TestInterfaceNestedShortForm covers the `name: { … }` nested-interface field:
// it is checked RECURSIVELY when a value is cast to the interface (`::`), and the
// `name?: { … }` form makes the nested field nullable.
func TestInterfaceNestedShortForm(t *testing.T) {
	cast := func(src string) string {
		return `Box := interface { bounds: { w int, h int } }
			` + src
	}

	// A value whose nested `bounds` has both fields (right types) satisfies it.
	testExpectRun(t, cast(`
		try { ({ bounds: { w: 1, h: 2 } }) :: Box; return "ok" } catch { return "no" }`),
		nil, Str("ok"))

	// Missing a nested field is REJECTED (deep check).
	testExpectRun(t, cast(`
		try { ({ bounds: { w: 1 } }) :: Box; return "ok" } catch { return "no" }`),
		nil, Str("no"))

	// A wrong nested field type is REJECTED (recursive type check).
	testExpectRun(t, cast(`
		try { ({ bounds: { w: "x", h: 2 } }) :: Box; return "ok" } catch { return "no" }`),
		nil, Str("no"))

	// The nested field itself missing is rejected (non-nullable).
	testExpectRun(t, cast(`
		try { ({ other: 1 }) :: Box; return "ok" } catch { return "no" }`),
		nil, Str("no"))

	// `bounds?: { … }` (nullable): an absent/nil nested field is accepted, but a
	// PRESENT one is still checked deeply.
	testExpectRun(t, `
		Box := interface { bounds?: { w int, h int } }
		try { ({ other: 1 }) :: Box; return "absent-ok" } catch { return "no" }`,
		nil, Str("absent-ok"))
	testExpectRun(t, `
		Box := interface { bounds?: { w int, h int } }
		try { ({ bounds: { w: 1 } }) :: Box; return "ok" } catch { return "present-checked" }`,
		nil, Str("present-checked"))

	// Deeper nesting is checked all the way down.
	testExpectRun(t, `
		Box := interface { a: { b: { c int } } }
		try { ({ a: { b: { c: 1 } } }) :: Box; return "ok" } catch { return "no" }`,
		nil, Str("ok"))
	testExpectRun(t, `
		Box := interface { a: { b: { c int } } }
		try { ({ a: { b: { c: "x" } } }) :: Box; return "ok" } catch { return "no" }`,
		nil, Str("no"))
}

// TestInterfaceArrayShortForm covers the `name: []{ … }` array-interface field:
// the value must be an array whose elements each satisfy the nested interface,
// checked recursively; `[][]` nests deeper.
func TestInterfaceArrayShortForm(t *testing.T) {
	box := func(src string) string {
		return `Box := interface { items: []{ w int, h int } }
			` + src
	}

	// An array whose elements all satisfy the body is accepted.
	testExpectRun(t, box(`
		try { ({ items: [{ w: 1, h: 2 }, { w: 3, h: 4 }] }) :: Box; return "ok" } catch { return "no" }`),
		nil, Str("ok"))
	// One bad element is rejected.
	testExpectRun(t, box(`
		try { ({ items: [{ w: 1, h: 2 }, { w: 3 }] }) :: Box; return "ok" } catch { return "no" }`),
		nil, Str("no"))
	// A non-array value is rejected.
	testExpectRun(t, box(`
		try { ({ items: { w: 1, h: 2 } }) :: Box; return "ok" } catch { return "no" }`),
		nil, Str("no"))
	// An empty array vacuously satisfies it.
	testExpectRun(t, box(`
		try { ({ items: [] }) :: Box; return "ok" } catch { return "no" }`),
		nil, Str("ok"))

	// `[][]` — an array of arrays, checked to the leaf.
	testExpectRun(t, `
		Grid := interface { grid: [][]{ n int } }
		try { ({ grid: [[{ n: 1 }], [{ n: 2 }]] }) :: Grid; return "ok" } catch { return "no" }`,
		nil, Str("ok"))
	testExpectRun(t, `
		Grid := interface { grid: [][]{ n int } }
		try { ({ grid: [[{ n: 1 }], [{ x: 2 }]] }) :: Grid; return "ok" } catch { return "no" }`,
		nil, Str("no"))
}
