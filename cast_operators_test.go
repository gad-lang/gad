package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// `:>` transforms into a class, an enum value or an interface only; `::>`
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
	for _, src := range []string{`return 1 :> str`, `return [1, 2] :> array`, `type strs []str; return [1, 2] :> strs`} {
		expectErrHas(t, src, newOpts(), "use `::>` to convert")
	}

	// ::> converts (what ::: did)
	testExpectRun(t, `type strs []str; return str([1, 2] ::> strs)`, nil, Str(`strs["1", "2"]`))
	testExpectRun(t, `return [1 ::> str, "2" ::> int, 0 ::> bool, 5 ::> ((v) => v * 2)]`,
		nil, Array{Str("1"), Int(2), False, Int(10)})
	testExpectRun(t, `class P { x int }; return ({x: 1} ::> P).x`, nil, Int(1))

	// ::: is a compile error that says what replaced it
	expectCompileError(t, `return 1 ::: str`, "`:::` was replaced: `:>` transforms into a class, an enum value or an interface; `::>` converts to any type")
}
