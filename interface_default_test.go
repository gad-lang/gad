package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// A field of an interface may have a default (`name? T = expr`): what `:>`
// gives it when it is not given — absent, or `no`. A given nil is a value.
func TestInterfaceFieldDefaults(t *testing.T) {
	const opts = `interface Opts { layout? str = "table"; n int = 1 + 2; tag? str; note? str = nil }; `
	testExpectRun(t, opts+`return {} :> Opts`, nil, Dict{"layout": Str("table"), "n": Int(3), "note": Nil})
	testExpectRun(t, opts+`return {layout: "cards", n: 9} :> Opts`, nil, Dict{"layout": Str("cards"), "n": Int(9), "note": Nil})
	// `no` is not given: the default
	testExpectRun(t, opts+`return {layout: no, n: no} :> Opts`, nil, Dict{"layout": Str("table"), "n": Int(3), "note": Nil})
	testExpectRun(t, opts+`return (;layout=no, tag="x") :> Opts`, nil,
		Dict{"layout": Str("table"), "n": Int(3), "note": Nil, "tag": Str("x")})
	// a given nil is a value: kept (a nullable field), refused (one that is not)
	testExpectRun(t, opts+`return ({layout: nil} :> Opts).layout`, nil, Nil)
	expectErrHas(t, opts+`return {n: nil} :> Opts`, newOpts(), `field "n" is required`)
	// a field reset with no default is not given
	testExpectRun(t, opts+`return "tag" in ({tag: no} :> Opts)`, nil, False)
	// `::` only checks: no default given
	expectErrHas(t, opts+`return {} :: Opts`, newOpts(), "is not assignable")

	// the default goes through the field's type, as a given value: an enum
	// takes the member of a name or a value
	testExpectRun(t, `enum mode { r; w }; interface M { m? mode = #w }; return [({} :> M).m == mode.w, ({m: 1} :> M).m == mode.r]`,
		nil, Array{True, True})

	// computed where the interface is declared: a function, a closure, a
	// computed value (called on each cast)
	testExpectRun(t, `k := 5; interface C { c? int = (= k * 2); f? <(x int) <r int>> = (x) => x + k }
		a := ({} :> C).c; k = 7; return [a, ({} :> C).c, ({} :> C).f(1)]`,
		nil, Array{Int(10), Int(14), Int(8)})

	// reflection
	testExpectRun(t, opts+`return [Opts.fields[0].default, Opts.fields[0].hasDefault, Opts.fields[2].hasDefault, Opts.fields[3].default, Opts.fields[3].hasDefault]`,
		nil, Array{Str("table"), True, False, Nil, True})
	testExpectRun(t, opts+`return str(Opts.fields[1])`, nil, Str("n int = 3"))

	// a runtime spec takes a default too
	testExpectRun(t, `interface S { **(;fields=(;x=(;types=[int], default=4))) }; return ({} :> S).x`, nil, Int(4))

	// an interface written as a type is a constant: so are its defaults
	expectCompileError(t, `k := 1; f := (v interface { x int = k }) => v`, "the default of an interface written as a type is a constant")
	testExpectRun(t, `f := (v interface { x? int = 2 }) => v :> interface { x? int = 2 }; return f({}).x`, nil, Int(2))
}

// `yes` — a key given without a value — against a field's type: a flag takes
// it; a bool takes it as true on `:>` (`::` refuses it); any other type wants
// a value, unless it takes a flag too.
func TestInterfaceFieldYes(t *testing.T) {
	const f = `interface F { required? flag; readonly? bool; layout? str; mode? flag|str }; `
	testExpectRun(t, f+`return [({required: yes} :> F).required, ({readonly: yes} :> F).readonly, ((;readonly) :> F).readonly, ({mode: yes} :> F).mode]`,
		nil, Array{Yes, True, True, Yes})
	testExpectRun(t, f+`return ({required: yes} :: F).required`, nil, Yes)
	expectErrHas(t, f+`return {readonly: yes} :: F`, newOpts(), "is not assignable to ‹F›")
	expectErrHas(t, f+`return {layout: yes} :> F`, newOpts(), `field "layout": given without a value (yes); want a value of ‹str›`)
	expectErrHas(t, f+`return {required: true} :> F`, newOpts(), `‹bool› is not assignable to ‹flag›`)
	// `no` is not given (a reset), whatever the type
	testExpectRun(t, f+`return len({required: no, readonly: no, layout: no} :> F)`, nil, Int(0))
}
