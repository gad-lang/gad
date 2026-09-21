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
