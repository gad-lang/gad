package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// The compiler makes a generic's instances (compiler_generics.go): each
// `Name[args]` of the block a declaration of its own, the parameters replaced.
func TestGenericsCompile(t *testing.T) {
	// too many arguments
	expectCompileError(t, `
	class Box[T] { item T }
	x := Box[int, str]`, "Box has 1 type parameters, given 2 type arguments")
	// a parameter twice
	expectCompileError(t, `class P[T, T] { a T }`, `type parameter "T" declared twice`)
	// a generic twice
	expectCompileError(t, `
	class P[T] { a T }
	interface P[T] { a T }`, `"P" redeclared in this block`)
	// a generic whose instances need ever more of them
	expectCompileError(t, `
	class L[T] { next? L[L[T]] }`, "more than 1000 instances")

	// an instance is declared once, however many times it is used
	testExpectRun(t, `
	class Box[T] { item T }
	a := Box[int]
	b := Box[int]
	return a == b, a.@name, Box[str] == a`, nil, Array{True, Str("Box[int]"), False})
	// a union argument is written as the name says
	testExpectRun(t, `
	class Box[T] { item T }
	return Box[int|str].@name`, nil, Str("Box[int|str]"))
	// used before the generic is declared, and in another's declaration
	testExpectRun(t, `
	b := Box[int](; item=1)
	class Holder { b Box[int]; p? Pair[str, int] }
	class Box[T] { item T }
	class Pair[K, V] { key K; value V }
	return Holder(; b=b).b.item`, nil, Int(1))
}

// A generic at run time: its parameters, and the types each instance checks.
func TestGenericsVM(t *testing.T) {
	// the aliases: the name alone is the declaration with them
	testExpectRun(t, `
	class P [X int, Y float] { x X; y Y }
	p := P(; x=1, y=2.5)
	return p.x, p.y, P.@name`, nil, Array{Int(1), Float(2.5), Str("P")})
	expectErrHas(t, `
	class P [X int, Y float] { x X; y Y }
	P(; x="1", y=2.5)`, newOpts(), `field "x" expects int, got str`)

	// the arguments; one left out is its alias; a parameter without one, any
	testExpectRun(t, `
	class Pair[K str, V int] { key K; value V }
	a := Pair[float, bool](; key=1.5, value=true)
	b := Pair[float](; key=2.5, value=3)
	return typeName(a), typeName(b), a.value, b.value`, nil,
		Array{Str("Pair[float, bool]"), Str("Pair[float, int]"), True, Int(3)})
	expectErrHas(t, `
	class Box[T] { item T }
	Box[int](; item="x")`, newOpts(), `field "item" expects int, got str`)
	testExpectRun(t, `
	class Box[T] { item T }
	return Box(; item="x").item, Box[any].@name`, nil, Array{Str("x"), Str("Box[any]")})

	// a class refers to itself — the define callback's first parameter —, and
	// a generic to its instance
	testExpectRun(t, `
	class Node { v int; next? Node }
	return Node(; v=1, next=Node(; v=2)).next.v`, nil, Int(2))
	testExpectRun(t, `
	class List[T] { v T; next? List[T] }
	l := List[str](; v="a", next=List[str](; v="b"))
	return l.next.v, typeName(l.next)`, nil, Array{Str("b"), Str("List[str]")})
	expectErrHas(t, `
	class List[T] { v T; next? List[T] }
	List[str](; v="a", next={v: 1})`, newOpts(), `field "v" expects str, got int`)

	// an interface: generic, and naming itself
	testExpectRun(t, `
	interface Named[T str] { name T }
	return {name: "a"} :: Named or "no", {name: 1} :: Named or "no", {name: 1} :: Named[int] or "no"`,
		nil, Array{Dict{"name": Str("a")}, Str("no"), Dict{"name": Int(1)}})
	testExpectRun(t, `
	interface N { v int; next? N }
	return {v: 1, next: {v: 2}} :: N or "no", {v: 1, next: {v: "x"}} :: N or "no"`,
		nil, Array{Dict{"v": Int(1), "next": Dict{"v": Int(2)}}, Str("no")})

	// a mixin: generic, naming itself, and typing by its interface
	testExpectRun(t, `
	mixin M[T int] { v T }
	class C { use M[str] }
	class D { use M }
	return C(; v="a").v, D(; v=1).v`, nil, Array{Str("a"), Int(1)})
	testExpectRun(t, `
	mixin M { v int = 1; next? M }
	class C { use M }
	return C(; next=C(; v=2)).next.v`, nil, Int(2))
	expectErrHas(t, `
	mixin M { v int = 1 }
	class D { v str = "x" }
	class H { m? M }
	H(; m=D())`, newOpts(), `field "m" expects M, got D`)

	// in a function body
	testExpectRun(t, `
	func f() {
		class Box[T int] { item T; next? Box[T] }
		return Box[str](; item="a", next=Box[str](; item="b")).next.item
	}
	return f()`, nil, Str("b"))
}

// `@tparams`: each type parameter and the type it is — an array of them, a
// union —; nothing for one not generic.
func TestGenericsTParams(t *testing.T) {
	testExpectRun(t, `
	class P [X int, Y float] { x X; y Y }
	return dict(P.@tparams).X == int, dict(P.@tparams).Y == float, dict(Pair[str].@tparams).V == int
	class Pair[K, V int] { key K; value V }`, nil, Array{True, True, True})
	testExpectRun(t, `
	class Box[T] { item T }
	t := Box.@tparams
	return t[0].k, dict(t).T == any, dict(Box[int|str].@tparams).T == [int, str]`, nil, Array{Str("T"), True, True})
	testExpectRun(t, `
	interface I[T str] { v T }
	mixin M[T int] { v T }
	return dict(I.@tparams).T == str, dict(I[bool].@tparams).T == bool, dict(M[str].@tparams).T == str`, nil, Array{True, True, True})
	testExpectRun(t, `
	class C {}
	interface I { v int }
	return len(C.@tparams), len(I.@tparams)`, nil, Array{Int(0), Int(0)})
}
