package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// `:>` transforms into a class, an enum value, an interface, a dict or a
// key-value array only; `::>`
// converts to any type; `:::` is gone (github.com/gad-lang/gad issue #16).
func TestCastOperators(t *testing.T) {
	// :> — enum member by its value (int, negative ones too, or uint) or name
	testExpectRun(t, `enum mode { r; w }; return [1 :> mode == mode.r, (2 :> mode).name, ("w" :> mode).name]`,
		nil, Array{True, Str("w"), Str("w")})
	testExpectRun(t, `enum sign { a = -1; b }; return [(-1 :> sign).name, (-2 :> sign).name]`,
		nil, Array{Str("a"), Str("b")})
	testExpectRun(t, `enum u { x = 5; y }; return [(6 :> u).name, (uint(6) :> u).name]`,
		nil, Array{Str("y"), Str("y")})
	expectErrHas(t, `enum mode { r; w }; return 3 :> mode`, newOpts(), "‹3› is not a member of ‹mode›")
	// :: never converts
	expectErrHas(t, `enum mode { r; w }; return 1 :: mode`, newOpts(), "is not assignable to ‹mode›")

	// :> — a field typed by an enum takes its member; other fields are checked
	testExpectRun(t, `enum mode { r; w }; class F { m mode }; return ({m: 2} :> F).m == mode.w`, nil, True)
	expectErrHas(t, `class P { x int }; return {x: "1"} :> P`, newOpts(), `field "x" expects int`)

	// :> refuses any other target, pointing to ::>
	for _, src := range []string{`return 1 :> str`, `return [1, 2] :> []str`, `type strs []str; return [1, 2] :> strs`} {
		expectErrHas(t, src, newOpts(), "use `::>` to convert")
	}

	// ::> converts (what ::: did)
	testExpectRun(t, `type strs []str; return str([1, 2] ::> strs)`, nil, Str(`strs["1", "2"]`))
	testExpectRun(t, `return [1 ::> str, "2" ::> int, 0 ::> bool, 5 ::> ((v) => v * 2)]`,
		nil, Array{Str("1"), Int(2), False, Int(10)})
	testExpectRun(t, `class P { x int }; return ({x: 1} ::> P).x`, nil, Int(1))

	// ::: is a compile error that says what replaced it
	expectCompileError(t, `return 1 ::: str`, "`:::` was replaced: `:>` transforms into a class, an enum value, an interface, a dict or a key-value array; `::>` converts to any type")

	// :> — a value already of the type is itself, not a copy
	testExpectRun(t, `a := [1, 2]; b := a :> array; b[0] = 9; return [a, a :> []int, "x" :> str]`,
		nil, Array{Array{Int(9), Int(2)}, Array{Int(9), Int(2)}, Str("x")})
	testExpectRun(t, `type strs []str; v := strs("a"); return (v :> strs) == v`, nil, True)

	// :> — into a dict or a key-value array: the pairs of the source
	testExpectRun(t, `return [str({a: no} :> keyValueArray), (;a=1, b=2) :> dict, str((;a=1, a=no) :> keyValueArray)]`,
		nil, Array{Str("(;a=no)"), Dict{"a": Int(1), "b": Int(2)}, Str("(;a=no)")})
	testExpectRun(t, `class P { x int }; return P(;x=1) :> dict`, nil, Dict{"x": Int(1)})
}
