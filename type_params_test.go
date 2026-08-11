package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// TestTypeParametersFunc covers type parameters on a named function: each
// parameter/return type that names a type parameter is checked against the
// parameter's constraint, and a mismatch is rejected.
func TestTypeParametersFunc(t *testing.T) {
	const set = `func mySet[T indexable, K int|uint, V number](target T, k K, v V) <T> {
		target[k] = v
		return target
	}
	`

	// Happy path: array is indexable (T), 1 is an int (K), 99 is a number (V).
	testExpectRun(t, set+`return mySet([10, 20, 30], 1, 99)`,
		nil, Array{Int(10), Int(99), Int(30)})
	// A uint key and a float value also satisfy K and V.
	testExpectRun(t, set+`return mySet([0, 0], 1u, 3.5)`,
		nil, Array{Int(0), Float(3.5)})

	// A str key violates K (int|uint). K expands to concrete builtin types, which
	// a named function checks during method dispatch, so the call is rejected.
	// (An interface/union constraint such as T or V is permissive at named-function
	// dispatch — the same as writing the constraint type directly — so those are
	// exercised on anonymous functions in TestTypeParametersAnonAndClosure.)
	testExpectRun(t, set+`try { mySet([0], "k", 1); return "ok" } catch e { return "reject" }`,
		nil, Str("reject"))
}

// TestTypeParametersAnonAndClosure covers type parameters on anonymous functions
// (`func[…]` / `func […]`) and on the closure/lambda body form.
func TestTypeParametersAnonAndClosure(t *testing.T) {
	// Anonymous function, no space before the brackets.
	testExpectRun(t, `x := func[T number](v T) => v; return x(5)`, nil, Int(5))
	testExpectRun(t, `x := func[T number](v T) => v; return x(2.5)`, nil, Float(2.5))
	// With a space before the brackets and a block body + return type.
	testExpectRun(t, `y := func [T number](v T) <T> { return v * 2 }; return y(3)`, nil, Int(6))
	// Rejection through the constraint.
	testExpectRun(t, `x := func[T number](v T) => v
		try { x("s"); return "ok" } catch e { return "reject" }`, nil, Str("reject"))
}

// TestTypeParametersDictShorthand covers type parameters on a dict method
// shorthand, both the `:` closure form and the `{ … }` block form.
func TestTypeParametersDictShorthand(t *testing.T) {
	testExpectRun(t, `d := { twice[T number](v T) <T>: v * 2 }; return d.twice(21)`,
		nil, Int(42))
	testExpectRun(t, `d := { inc[T number](v T) <T> { return v + 1 } }; return d.inc(9)`,
		nil, Int(10))
	testExpectRun(t, `d := { twice[T number](v T) <T>: v * 2 }
		try { d.twice("x"); return "ok" } catch e { return "reject" }`, nil, Str("reject"))
}

// TestTypeParametersHeaderAndMeti covers type parameters in a func-header value
// (`<[…](…)>`) and in a method interface (`meti { […](…) }`), including that a
// generic function satisfies a generic meti.
func TestTypeParametersHeaderAndMeti(t *testing.T) {
	// A func-header value parses with type parameters and yields a header value.
	testExpectRun(t, `h := <[T number](v T) <T>>; return typeName(h)`,
		nil, Str("FunctionHeader"))
	// A generic function implements a generic meti.
	testExpectRun(t, `m := meti { [T number](v T) <T> }
		f := func[T number](v T) <T> => v
		return implements(f, m)`, nil, True)
}

// TestTypeParametersConstraintForms covers different constraint shapes: a single
// interface, a builtin union spelled inline, and a named builtin union.
func TestTypeParametersConstraintForms(t *testing.T) {
	// Single-type constraint reused for two parameters.
	testExpectRun(t, `f := func[T str](a T, b T) => a + b; return f("x", "y")`, nil, Str("xy"))
	// A type parameter nested inside an inline union in a parameter type.
	testExpectRun(t, `f := func[N number](v str|N) => 1
		return [f("s"), f(3), f(1.5)]`, nil, Array{Int(1), Int(1), Int(1)})
	// A type parameter used as the return type `<T>` compiles and runs (return
	// types are declarations, not runtime-enforced, mirroring a direct `<int>`).
	testExpectRun(t, `f := func[T int](v T) <T> => v; return f(7)`, nil, Int(7))
}
