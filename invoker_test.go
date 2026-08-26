package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// TestGadInvoker covers gad.invoker: it pre-resolves a function's overload for the
// parameter types held in the args array and returns a repeatable callable bound
// to that same array, so mutating the array feeds new values.
func TestGadInvoker(t *testing.T) {
	// Resolves str's int overload; the shared array feeds each value in turn.
	testExpectRun(t, `
		args := [int]
		inv := gad.invoker(str, args)
		s := ""
		for i := 0; i < 4; i++ { args[0] = i; s += inv() }
		return s`, nil, Str("0123"))

	// The invoker's result matches calling the function directly.
	testExpectRun(t, `
		args := [int]
		inv := gad.invoker(str, args)
		ok := true
		for i := 0; i < 10; i++ { args[0] = i; if inv() != str(i) { ok = false } }
		return ok`, nil, True)

	// A plain function (no overloads) is bound directly and reuses its args array.
	testExpectRun(t, `
		add := func(a, b) => a + b
		args := [0, 0]
		inv := gad.invoker(add, args)
		args[0] = 3; args[1] = 4
		return inv()`, nil, Int(7))

	// Two-argument overload resolution: [str, str] picks the str method, [int, int]
	// picks the int one.
	testExpectRun(t, `
		f(a int, b int) => a + b
		met f(a str, b str) => a + "-" + b
		sargs := [str, str]
		iargs := [int, int]
		sinv := gad.invoker(f, sargs)
		iinv := gad.invoker(f, iargs)
		sargs[0] = "x"; sargs[1] = "y"
		iargs[0] = 3; iargs[1] = 4
		return [sinv(), iinv()]`, nil, Array{Str("x-y"), Int(7)})

	// The returned object is an invoker.
	testExpectRun(t, `return typeName(gad.invoker(str, [int]))`, nil, Str("invoker"))
}

// TestGadInvokerNamedArgs covers the **nargs capture: named args passed at
// construction are forwarded to every invocation.
func TestGadInvokerNamedArgs(t *testing.T) {
	testExpectRun(t, `
		f := func(v; sep="-") => str(v) + sep
		args := [int]
		inv := gad.invoker(f, args; sep="|")
		args[0] = 5
		return inv()`, nil, Str("5|"))
}

// TestGadInvokerErrors covers the argument checks.
func TestGadInvokerErrors(t *testing.T) {
	expectErrHas(t, `return gad.invoker(42, [int])`, nil, "1st argument")
}
