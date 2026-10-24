package gad_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gad-lang/gad"
)

// evalReturn runs src and is what it returns.
func evalReturn(t *testing.T, src string) gad.Object {
	t.Helper()
	st := gad.NewSymbolTable(gad.NewBuiltins().NameSet)
	res, err := gad.Compile(st, []byte(src), gad.CompileOptions{})
	require.NoError(t, err, src)
	ret, err := gad.NewVM(gad.NewBuiltins().Build(), res.Bytecode).RunOpts(&gad.RunOpts{})
	require.NoError(t, err, src)
	return ret
}

// A value's literal — its print inside a collection (issue #25) — is gad code
// that reads back as an equal value of the same type.
func TestPrintLiteralRoundTrip(t *testing.T) {
	for _, c := range []struct{ src, literal string }{
		{`2026-01-31D`, `2026-01-31D`},
		{`2026-01-31DT`, `2026-01-31DT`},
		{`time.strToCdtime("2026-01-31 10:30:00.5")`, `2026-01-31T10:30:00.5`},
		{`2026-01-31T10:30DT`, `2026-01-31T10:30`},
		{`20260131T10:30:01.000000001`, `2026-01-31T10:30:01.000000001`},
		{`10:30T`, `10:30T`},
		{`10:30:15.25T`, `10:30:15.25T`},
		{`time.ctime(23, 59, 59; ns=999999999)`, `23:59:59.999999999T`},
		{`time.strToCtime("00:00")`, `00:00T`},
		{`dur 1h30m`, `dur 1h30m0s`},
		{`dur -2s`, `dur -2s`},
		{`2026-01-31T`, `2026-01-31T`},
		{`1781609136U`, `1781609136U`},
		{`1781609136.000001U`, `1781609136.000001U`},
		{`time.strToTime("2026-01-31T10:30:00.5-03:00")`, `time.in(1769866200.500U, time.strToLocation("-03:00"))`},
		{`time.in(1781609136U, time.strToLocation("America/Sao_Paulo"))`, `time.in(1781609136U, time.strToLocation("America/Sao_Paulo"))`},
	} {
		t.Run(c.src, func(t *testing.T) {
			v := evalReturn(t, "return "+c.src)
			lit := evalReturn(t, "return str(["+c.src+"])").ToString()
			require.Equal(t, "["+c.literal+"]", lit)
			back := evalReturn(t, "return "+c.literal)
			require.Equal(t, v.Type(), back.Type())
			require.True(t, v.Equal(back), "%s != %s", v, back)
			if !strings.Contains(c.src, "-03:00") {
				// a zone by its offset reads back named "-03:00", not "-03"
				require.Equal(t, v.ToString(), back.ToString())
			}
		})
	}
}

// The literals of the collections and the values of declared types: what a
// declaration (prelude) names reads back the same.
func TestPrintLiteralRoundTripDeclared(t *testing.T) {
	const prelude = `
type ints []int
type sm dict[str, int]
class P { x int; y str }
`
	for _, c := range []struct{ src, literal string }{
		{`1..3`, `1..3`},
		{`1u..9u/2u`, `1u..9u/2u`},
		{`2026-01-01D..2026-01-03D`, `2026-01-01D..2026-01-03D`},
		{`keyValue("a b", 2u)`, `["a b"=2u]`},
		{`(;a=1, b="x")`, `(;a=1, b="x")`},
		{`ints(1, 2)`, `ints(1, 2)`},
		{`sm({a: 1})`, `sm({a: 1})`},
		{`syncDict({a: 1u})`, `syncDict({a: 1u})`},
		{`P(; x=1, y="a")`, `P(;x=1, y="a")`},
	} {
		t.Run(c.src, func(t *testing.T) {
			lit := evalReturn(t, prelude+"return str(["+c.src+"])").ToString()
			require.Equal(t, "["+c.literal+"]", lit)
			// the literal makes an equal value: printed the same
			back := evalReturn(t, prelude+"return str(["+c.literal+"])").ToString()
			require.Equal(t, lit, back)
		})
	}
}

// The text, the literal and the repr of each kind of value (issue #25).
func TestPrintForms(t *testing.T) {
	for _, c := range []struct{ src, text, literal, repr string }{
		{`2026-01-31D`, `2026-01-31`, `2026-01-31D`, `‹cdate: 2026-01-31D›`},
		{`dur 90s`, `1m30s`, `dur 1m30s`, `‹duration: dur 1m30s›`},
		{`1u..3u`, `1u..3u`, `1u..3u`, `‹Range: ‹uint: 1u›..‹uint: 3u››`},
		{`keyValue("a", 1)`, `[a=1]`, `[a=1]`, `‹keyValue: [a=‹int: 1›]›`},
		{`(;a=1)`, `(;a=1)`, `(;a=1)`, `‹keyValueArray: (;a=‹int: 1›)›`},
		{`syncDict({a: 1})`, `{a: 1}`, `syncDict({a: 1})`, `‹syncDict: {a: ‹int: 1›}›`},
	} {
		t.Run(c.src, func(t *testing.T) {
			r := evalReturn(t, "x := "+c.src+"; return [str(x), str([x]), repr(x)]").(gad.Array)
			require.Equal(t, c.text, r[0].ToString())
			require.Equal(t, "["+c.literal+"]", r[1].ToString())
			if c.repr != "" {
				require.Equal(t, c.repr, r[2].ToString())
			}
		})
	}
	// a pointer: `&value`, the value as its literal
	r := evalReturn(t, `x := "a"; p := &x; return [str(p), repr(p)]`).(gad.Array)
	require.Equal(t, gad.Array{gad.Str(`&"a"`), gad.Str(`‹ptr: &‹str: "a"››`)}, r)
	// fmt's verbs take the gad text, never the Go value
	r = evalReturn(t, `return [sprintf("%v|%s|%q|%#v", 1..3, 1..3, 1..3, 1..3), sprintf("%-6v|", (;a=1))]`).(gad.Array)
	require.Equal(t, gad.Array{gad.Str(`1..3|1..3|"1..3"|‹Range: ‹int: 1›..‹int: 3››`), gad.Str(`(;a=1)|`)}, r)
}

// A class instance: the fields set (all with zeros), by the class's name (the
// full one with typesAsFullNames).
func TestPrintClassInstance(t *testing.T) {
	r := evalReturn(t, `
class P { x int; y int; name str }
p := P(; x=1)
return [str(p), str([p]), repr(p), str(p; zeros), str(p; typesAsFullNames)]`).(gad.Array)
	require.Equal(t, gad.Array{
		gad.Str(`P{x: 1}`), gad.Str(`[P(;x=1)]`), gad.Str(`‹P: {x: ‹int: 1›}›`),
		gad.Str(`P{x: 1, y: nil, name: nil}`), gad.Str(`(main).P{x: 1}`),
	}, r)
}

// A type in a collection prints its name — the code that names it.
func TestPrintTypeLiteral(t *testing.T) {
	r := evalReturn(t, `
class P { x int }
type ints []int
return [str([int, str, P, ints, time.cdate, int|str, any]), str([P]; typesAsFullNames)]`).(gad.Array)
	require.Equal(t, gad.Array{gad.Str(`[int, str, P, ints, time.cdate, int|str, any]`), gad.Str(`[(main).P]`)}, r)
}
