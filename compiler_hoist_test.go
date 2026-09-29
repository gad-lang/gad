package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// A const is visible in its whole block (compiler_hoist.go).
func TestConstHoist(t *testing.T) {
	// a const used before its declaration, and one another
	testExpectRun(t, `
	x := a
	const a = b + 1
	const b = 10
	return x, a, b`, nil, Array{Int(11), Int(11), Int(10)})

	// functions: called before, and each other (the body sees the const)
	testExpectRun(t, `
	r := isEven(10)
	func isEven(n) { return n == 0 ? true : isOdd(n-1) }
	func isOdd(n) { return n == 0 ? false : isEven(n-1) }
	return r`, nil, True)
	testExpectRun(t, `
	func f(n) => g(n) * 2
	x := f(2)
	func g(n) => n + k
	const k = 1
	return x`, nil, Int(6))

	// classes referring to each other, in a cycle
	testExpectRun(t, `
	class A { b? B; x int = 1 }
	class B { a? A }
	a := A(;b=B(;a=A()))
	return a.b.a.x`, nil, Int(1))
	testExpectRun(t, `
	a := A(;b=B())
	class A { b? B }
	class B { n int = 2 }
	return a.b.n`, nil, Int(2))

	// extending a class declared after
	testExpectRun(t, `
	class C { *D }
	class D { y int = 7 }
	return C().y`, nil, Int(7))

	// a typed array of a class whose field is that array
	testExpectRun(t, `
	type items []Item
	class Item { name str; children? items }
	i := Item(;name="a", children=items(Item(;name="b")))
	return i.children[0].name`, nil, Str("b"))
	testExpectRun(t, `
	class Item { name str; children? items }
	type items []Item
	i := Item(;name="a", children=items(Item(;name="b")))
	return i.children[0].name`, nil, Str("b"))

	// enum, interface, marker type, type union, mixin
	testExpectRun(t, `
	x := [str(Color.green), I != nil, str(U), T.n]
	enum Color { red, green }
	interface I { n int }
	type U <A|B>
	class A {}
	class B {}
	type T { n = 3 }
	return x`, nil, Array{Str("2"), True, Str("A|B"), Int(3)})
	testExpectRun(t, `
	class P { use M }
	mixin M {
		n = 5
		methods { m() => this.n }
	}
	return P().m()`, nil, Int(5))

	// a mixin whose field is a class declared after, which uses it
	testExpectRun(t, `
	p := P(;other=P(;n=4))
	mixin M { n int = 1; other? P }
	class P { use M }
	return p.other.n + p.n`, nil, Int(5))

	// in a function body and in a block
	testExpectRun(t, `
	func outer() {
		r := inner()
		func inner() => K * 2
		const K = 21
		return r
	}
	return outer()`, nil, Int(42))
	testExpectRun(t, `
	x := 0
	if true {
		x = Z
		const Z = 3
	}
	return x`, nil, Int(3))

	// still read-only
	expectErrHas(t, `
	a = 2
	const a = 1`, newOpts().CompilerError(), `Compile Error: assignment to constant variable "a"`)
	// a cycle of values
	expectErrHas(t, `
	const x = y
	const y = x`, newOpts().CompilerError(), "Compile Error: initialization cycle: x refers to y refers to x")
	expectErrHas(t, `
	class A { use M }
	mixin M { use N }
	mixin N { use M }`, newOpts().CompilerError(), "Compile Error: initialization cycle: M refers to N refers to M")
	expectErrHas(t, `
	class A { *B }
	class B { *A }`, newOpts().CompilerError(), "Compile Error: initialization cycle: A refers to B refers to A")
	// the value of a const needed before a variable it uses
	expectErrHas(t, `
	x := a
	v := 1
	const a = v`, newOpts().CompilerError(), `Compile Error: unresolved reference "v"`)
	// declared twice
	expectErrHas(t, `
	const a = 1
	func a() {}`, newOpts().CompilerError(), `Compile Error: "a" redeclared in this block`)
	// iota keeps its order
	expectErrHas(t, `
	x := i0
	const (i0 = iota, i1)`, newOpts().CompilerError(), `Compile Error: unresolved reference "i0"`)
}
