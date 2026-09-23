package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// TestTryAcrossInlineCall pins the unwinding of an error thrown inside a same-VM
// sub-run (VM.Call → callCompiledInline: a ComputedValue `(= …)`, an iterator
// callback …) and caught by a `try` in an OUTER frame. The error must leave the
// nested loop at its boundary frame and be caught by the outer loop — before the
// fix the outer catch ran inside the nested loop, which repeated a `for … in`
// iteration and crashed with a nil dereference on the next return.
func TestTryAcrossInlineCall(t *testing.T) {
	// caught by the caller's try; the program then returns normally
	testExpectRun(t, `
		fails := func(f) { try { f(); return false } catch { return true } }
		return [fails((= 1/0)), fails((= 1 :: str)), fails((= 5))]`,
		nil, Array{True, True, False})

	// inside `for … in`: each element runs exactly once
	testExpectRun(t, `
		n := 0
		for f in [(= 1/0), (= 1/0), (= 3)] {
			try { f(); n += 100 } catch { n += 1 }
		}
		return n`, nil, Int(102))

	// `finally` runs on both paths
	testExpectRun(t, `
		r := []
		h := func(f) { try { f() } catch { r += "c" } finally { r += "f" } }
		h((= 1/0)); h((= 1))
		return r`, nil, Array{Str("c"), Str("f"), Str("f")})

	// a try inside the sub-run itself still handles its own error
	testExpectRun(t, `
		f := func() { try { throw "x" } catch { return "inside" } }
		g := (= f())
		return g()`, nil, Str("inside"))

	// nested sub-runs: the innermost error is caught by the nearest outer try
	testExpectRun(t, `
		k := func(f) { try { return f() } catch { return "caught" } }
		return k((= k((= 1/0))))`, nil, Str("caught"))
}
