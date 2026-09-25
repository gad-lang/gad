package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// TestClassExtendsArray covers a class spreading a list of parents: a literal
// `*[A, B]`, a variable `*ps`, and `[alias=Parent]` key-value items (alone or
// nested in the list) naming the parent's key in `inst.@parents`.
func TestClassExtendsArray(t *testing.T) {
	testExpectRun(t, `
		class A { a = 1 }
		class B { b = 2 }
		class C { *[A, B] }
		ps := [A, B]
		class D { *ps }
		c := C(); d := D()
		return [c.a, c.b, d.a, d.b, len(C.@parents), len(D.@parents)]`,
		nil, Array{Int(1), Int(2), Int(1), Int(2), Int(2), Int(2)})

	// Aliases: `*[alias=A]`, `*[A, [alias=B]]`, a list in a variable, and the
	// `*A: alias` form (compiled to `[alias=A]`) all key @parents by the alias;
	// a parent with no alias is keyed by its class name.
	testExpectRun(t, `
		class A { a = 1 }
		class B { b = 2 }
		class C1 { *[x=A] }
		class C2 { *[A, [y=B]] }
		ps := [A, [z=B]]
		class C3 { *ps }
		class C4 { *A: w }
		pk := func(o) => sort(collect(keys(o.@parents)))
		return [pk(C1()), pk(C2()), pk(C3()), pk(C4()), C2(; b=5).@parents.y.b]`,
		nil, Array{
			Array{Str("x")}, Array{Str("A"), Str("y")}, Array{Str("A"), Str("z")},
			Array{Str("w")}, Int(5),
		})

	// The former `[Parent, "alias"]` pair is not a parent spec (use `[alias=Parent]`).
	testExpectRun(t, `class A { a = 1 }
		try { class C { *[A, "x"] }; return "ok" } catch { return "no" }`, nil, Str("no"))
	// A non-class item is rejected, as is an aliased non-class.
	testExpectRun(t, `try { class C { *[1] }; return "ok" } catch { return "no" }`, nil, Str("no"))
	testExpectRun(t, `try { class C { *[x=1] }; return "ok" } catch { return "no" }`, nil, Str("no"))
}

// TestMixinExtendsArray covers a mixin spreading a list of parent mixins,
// literal and from a variable (nesting flattens).
func TestMixinExtendsArray(t *testing.T) {
	testExpectRun(t, `
		mixin A { a = 1 }
		mixin B { b = 2 }
		mixin Z { z = 3 }
		mixin M { *[A, B] }
		ps := [A, [B, Z]]
		mixin N { *ps }
		class C { use M }
		class D { use N }
		c := C(); d := D()
		return [c.a, c.b, len(M.@parents), d.a, d.b, d.z, len(N.@parents)]`,
		nil, Array{Int(1), Int(2), Int(2), Int(1), Int(2), Int(3), Int(3)})

	testExpectRun(t, `class K {}
		try { mixin M { *[K] }; return "ok" } catch { return "no" }`, nil, Str("no"))
}

// TestInterfaceExtendsArray covers an interface extending a list of parents: a
// literal `*[A, B]` (resolved at compile time) and a variable `*ps` holding an
// array of interfaces (resolved and flattened at run time) — both enforced by
// `::` and merged by `@flat`.
func TestInterfaceExtendsArray(t *testing.T) {
	src := func(ext string) string {
		return `
		interface A { a int }
		interface B { b int }
		ps := [A, [B]]
		interface I { ` + ext + ` }
		ok := func(v) { try { v :: I; return true } catch { return false } }
		return [ok({a: 1, b: 2}), ok({a: 1}), str(I.@flat)]`
	}
	for _, ext := range []string{"*[A, B]", "*ps"} {
		testExpectRun(t, src(ext), nil, Array{True, False, Str("interface (main).I {a int; b int}")})
	}

	// Every parent must be an interface: a bad item fails where the interface is
	// declared.
	testExpectRun(t, `try { interface I { *[1] }; return "ok" } catch { return "no" }`,
		nil, Str("no"))
	testExpectRun(t, `ps := [1]; try { interface I { *ps }; return "ok" } catch { return "no" }`,
		nil, Str("no"))

	// Parents resolve where the interface is declared, so using it from a closure
	// still enforces a local parent (it used to read the closure's frame and pass
	// every value).
	testExpectRun(t, `
		interface A { a int }
		interface I { *A }
		ok := func(v) { try { v :: I; return true } catch { return false } }
		return [ok({a: 1}), ok({})]`, nil, Array{True, False})
}
