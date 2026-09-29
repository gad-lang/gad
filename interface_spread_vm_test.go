package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// `**Expr` in an interface adds the members EXPR gives where the interface is
// declared: fields (a type, types, a spec), methods (headers), props (a type,
// a getter header, a spec) and funcs (a function, or a spec with headers); a
// dict by the order of its names; nil adds nothing; anything else is an error.
func TestVMInterfaceSpread(t *testing.T) {
	// fields: a type, a spec with nullable; enforced on a cast
	testExpectRun(t, `extra := {fields: {x: int, y: (; types=[int], nullable=true)}}
		interface P { name str; **extra }
		return [({name: "a", x: 1}) :: P != nil]`, nil, Array{True})
	expectErrHas(t, `interface P { **{fields: {x: int}} }; return ({x: "s"}) :: P`, nil, `not assignable`)

	// the order: declared fields, then a dict's by name, a key-value array's in its own
	testExpectRun(t, `interface P { z int; **{fields: {b: int, a: int}}; ** {fields: (; d=int, c=int)} }
		names := []; for f in P.fields { names += f.name }; return names`,
		nil, Array{Str("z"), Str("a"), Str("b"), Str("d"), Str("c")})

	// methods and props: a class that has them satisfies it
	testExpectRun(t, `interface S { **{methods: {area: <() <float>>}, props: {label: <() <str>>}} }
		class Q { methods { area() => 1.0 }; props { label = "q" } }
		return typeName(Q() :: S)`, nil, Str("Q"))
	expectErrHas(t, `interface S { **{methods: {area: <() <float>>}} }; return ({}) :: S`, nil, `not assignable`)

	// funcs: the context function checked against its header
	testExpectRun(t, `render := func(indent int, obj) => 1
		interface R { name str; **{funcs: {render: (; fn=render, headers=[<(indent int, @self)>])}} }
		return typeName(({name: "a"}) :: R)`, nil, Str("dict"))

	// a bare name, a parenthesized one; nil adds nothing
	testExpectRun(t, `e := {fields: {v: int}}; interface A { **e }; interface B { ** (e) }; interface C { a int; **nil }
		return [len(A.fields), len(B.fields), len(C.fields)]`, nil, Array{Int(1), Int(1), Int(1)})

	// errors say what is wrong
	expectErrHas(t, `interface P { **1 }; return P`, nil, `**Expr expects a dict of fields, methods, props and funcs, got int`)
	expectErrHas(t, `interface P { **{other: {}} }; return P`, nil, `**Expr has no "other"`)
	expectErrHas(t, `interface P { **{fields: {x: 1}} }; return P`, nil, `field x: a type or a list of types, not int`)
	expectErrHas(t, `interface P { **{methods: {m: 1}} }; return P`, nil, `method m: a func header or a list of them`)

	// the rest capture is `**<name>`
	testExpectRun(t, `d := {a: 1, b: 2}; return (d ::: interface { a int; **<rest> }).rest`, nil, Dict{"b": Int(2)})
}
