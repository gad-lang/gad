package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// An interface has itself, as a class has: a field type naming it, and the
// local types of the function declaring it, still read once that function
// returned — the interface checked anywhere later.
// okFn is ok(v, T): whether v is assignable to T.
const okFn = `
ok := func(v, T) {
	try { v :: T } catch _ { return false }
	return true
}`

func TestInterfaceSelfReference(t *testing.T) {
	// itself, used after the function declaring it returned
	testExpectRun(t, okFn+`
		mk := func() {
			interface O { icon str|O; width? int }
			return O
		}
		O := mk()
		z := 99
		return [{icon: {icon: "x", width: 2}} :: O, ok({icon: {icon: 1}}, O)]`,
		nil, Array{Dict{"icon": Dict{"icon": Str("x"), "width": Int(2)}}, False})

	// the spec of an icon: the union named, a meta's type
	testExpectRun(t, okFn+`
		mk := func() {
			interface IconOptions { icon str|uuid|IconOptions; width? int }
			type Icon <str|uuid|IconOptions>
			return [IconOptions, Icon]
		}
		[IconOptions, Icon] := mk()
		v := {icon: {icon: uuid("0f8fad5b-d9cb-469f-a165-70867728950e")}, width: 3}
		return [ok(v, IconOptions), ok(v, Icon), ok("x", Icon), ok(1, Icon), str(Icon)]`,
		nil, Array{True, True, True, False, Str("str|uuid|IconOptions")})

	// a local type, its cell captured as a closure does
	testExpectRun(t, okFn+`
		mk := func() {
			class C { a str }
			interface O { c C; next? O }
			return [O, C]
		}
		[O, C] := mk()
		return [ok({c: C(; a="x"), next: {c: C(; a="y")}}, O), ok({c: 1}, O)]`,
		nil, Array{True, False})

	// a free variable, and a group of the interface naming the interface
	testExpectRun(t, okFn+`
		mk := func() {
			class C { a str }
			return func() {
				interface O { { c? C; inner? O } }
				return O
			}()
		}
		O := mk()
		return [ok({$1: {inner: {$1: {}}}}, O), ok({$1: {inner: 1}}, O)]`,
		nil, Array{True, False})

	// a type declared after the interface: read in the frame, while it runs
	testExpectRun(t, okFn+`
		f := func() {
			interface A { b B }
			interface B { x int }
			return [ok({b: {x: 1}}, A), ok({b: {x: "1"}}, A)]
		}
		return f()`,
		nil, Array{True, False})

	// a reflected field type is the interface itself
	testExpectRun(t, okFn+`
		mk := func() { interface O { o? O }; return O }
		O := mk()
		return O.fields[0].types[0] == O`,
		nil, True)
}
