package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// TestClassInitFieldsPerInstance verifies that a non-literal field default is
// evaluated per instance (not once at class definition and shared), because the
// compiler moves it into the class's single `initFields` initialiser.
func TestClassInitFieldsPerInstance(t *testing.T) {
	// A counter default increments once per construction.
	testExpectRun(t, `
		seq := 0
		next := func() { seq++; return seq }
		class C { id = next(); name = "?" }
		return [C().id, C().id, C().id]`, nil, Array{Int(1), Int(2), Int(3)})

	// An expression default referencing an outer variable is recomputed per
	// instance (here the value is stable, but it is computed each time).
	testExpectRun(t, `
		g := 10
		class C { y = g + 5 }
		return [C().y, C().y]`, nil, Array{Int(15), Int(15)})
}

// TestClassInitFieldsFreshComposite verifies that a composite default (an array
// or dict literal) yields a fresh, independent value per instance instead of one
// shared object.
func TestClassInitFieldsFreshComposite(t *testing.T) {
	testExpectRun(t, `
		class T { items = [1, 2] }
		a := T()
		b := T()
		a.items[0] = 99
		return [a.items[0], b.items[0]]`, nil, Array{Int(99), Int(1)})
}

// TestClassInitFieldsLiteralStaysStatic verifies that a scalar-literal default is
// NOT moved into initFields — it stays a plain inline value.
func TestClassInitFieldsLiteralStaysStatic(t *testing.T) {
	testExpectRun(t, `
		class C { x = 2; s = "hi"; ok = true; n }
		c := C()
		return [c.x, c.s, c.ok, c.n == nil]`, nil, Array{Int(2), Str("hi"), True, True})
}

// TestClassInitFieldsOverridden verifies that an explicitly passed field value
// wins over its initFields-computed default.
func TestClassInitFieldsOverridden(t *testing.T) {
	testExpectRun(t, `
		g := 10
		class C { y = g + 5 }
		return C(; y = 99).y`, nil, Int(99))
}

// TestClassComputedExprStillPerField verifies the `(= expr)` ComputedExpr form is
// unchanged — it stays inline (evaluated per field, per instance), alongside the
// initFields mechanism for plain non-literal defaults.
func TestClassComputedExprStillPerField(t *testing.T) {
	// Postfix `n++` yields the value before the increment (0, 1, 2).
	testExpectRun(t, `
		n := 0
		class C { id = (= n++) }
		return [C().id, C().id, C().id]`, nil, Array{Int(0), Int(1), Int(2)})

	// Prefix `++n` yields the value after the increment (1, 2, 3).
	testExpectRun(t, `
		n := 0
		class C { id = (= ++n) }
		return [C().id, C().id, C().id]`, nil, Array{Int(1), Int(2), Int(3)})

	// A class mixing all three default kinds: literal (x), ComputedExpr (seq) and
	// a plain non-literal (y) — each behaves correctly and independently.
	testExpectRun(t, `
		g := 100
		n := 0
		class C { x = 1; y = g + 1; seq = (= n++) }
		c1 := C(); c2 := C()
		return [c1.x, c1.y, c1.seq, c2.seq]`, nil, Array{Int(1), Int(101), Int(0), Int(1)})
}
