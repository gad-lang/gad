package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// TestInterfaceFlatMergeSourceExtends verifies `@flat` resolves a source-level
// `interface { *A; *B }` (symbol parents) and merges same-name members by
// signature instead of rejecting them by name (issue #10).
func TestInterfaceFlatMergeSourceExtends(t *testing.T) {
	// A getter, its setters and a method's overloads all combine across A and B.
	testExpectRun(t, `
		interface A { get y int; set y; x { (); (v int) } }
		interface B { set y str; x(v bool) }
		f := interface { *A; *B }.@flat
		return [len(f.props), len(f.methods[0].headers)]`,
		// y: one merged prop; x: 3 overloads ((), (v int), (v bool)).
		nil, Array{Int(1), Int(3)})

	// A merged property renders as a single `prop y { … }` with its accessors.
	testExpectRun(t, `
		interface A { get y int; set y str; set y }
		return str(A.@flat.props[0])`,
		nil, Str("prop y { get int; set str|any }"))
}

// TestInterfacePropRender covers the property render forms: a getter + single
// setter of one type collapses to `prop x T` (`prop x` when untyped); other
// combinations use the `prop x { get …; set … }` braces form.
func TestInterfacePropRender(t *testing.T) {
	// getter + single setter, same type -> compact.
	testExpectRun(t, `interface A { get x int; set x int }; return str(A.@flat.props[0])`,
		nil, Str("prop x int"))
	// getter + single setter, both untyped -> compact, no type.
	testExpectRun(t, `interface A { get x; set x }; return str(A.@flat.props[0])`,
		nil, Str("prop x"))
	// getter + single setter, different types -> braces.
	testExpectRun(t, `interface A { get x int; set x bool }; return str(A.@flat.props[0])`,
		nil, Str("prop x { get int; set bool }"))
	// getter + two setter overloads -> braces with a set union.
	testExpectRun(t, `interface A { get x int; set x int; set x bool }; return str(A.@flat.props[0])`,
		nil, Str("prop x { get int; set int|bool }"))
}

// TestInterfaceFlatDedup verifies identical signatures across interfaces are
// deduplicated (no error): a diamond, a repeated setter and a repeated overload.
func TestInterfaceFlatDedup(t *testing.T) {
	// Diamond: A reached directly and through B — foo counted once.
	testExpectRun(t, `
		interface A { foo() }
		interface B { *A; bar() }
		f := interface { *A; *B }.@flat
		return len(f.methods)`,
		nil, Int(2))

	// Identical setter and identical method overload dedup.
	testExpectRun(t, `
		interface A { set y str; x(v bool) }
		interface B { set y str; x(v bool) }
		f := interface { *A; *B }.@flat
		return [len(f.props), len(f.methods[0].headers)]`,
		nil, Array{Int(1), Int(1)})
}

// TestInterfaceFlatConflicts verifies the genuine conflicts (issue #10): a name
// used as two kinds, and a method overload with the same params but a different
// return type.
func TestInterfaceFlatConflicts(t *testing.T) {
	// `get x int` where x is a method -> kind conflict.
	expectErrHas(t, `
		interface A { x { (); (v int) } }
		return interface { *A; get x int }.@flat`,
		nil, `declared both as a prop and a method`)

	// Same params, different return TYPE (`<_ int>` vs none) -> conflict.
	expectErrHas(t, `
		interface A { x { () <_ int>; (v bool) } }
		interface B { x() }
		return interface { *A; *B }.@flat`,
		nil, `same parameters but different return types`)
}
