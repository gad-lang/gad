// Put relatively new features' tests in this test file.

package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

func TestVMDeferStmt(t *testing.T) {
	// runs after the body
	testExpectRun(t, `
	out := ""
	f := func() { defer { out += "d" }; out += "b" }
	f()
	return out`, nil, Str("bd"))

	// $ret can be modified by a defer
	testExpectRun(t, `f := func(x) { defer { $ret = $ret + 100 }; return x }; return f(5)`,
		nil, Int(105))

	// LIFO order, $ret threaded across defers
	testExpectRun(t, `f := func() { defer { $ret += "-A" }; defer { $ret += "-B" }; return "x" }; return f()`,
		nil, Str("x-B-A"))

	// `defer handler` calls the handler; `defer handler(x)` calls with args
	testExpectRun(t, `
	out := ""
	h := func(m) { out += m }
	f := func() { defer h("done"); defer h("step") }
	f()
	return out`, nil, Str("stepdone"))

	// defer_err recovers: sets $ret and clears $err
	testExpectRun(t, `
	f := func() { defer_err { $ret = "recovered:" + str($err); $err = nil }; throw "boom" }
	return f()`, nil, Str("recovered:error: boom"))

	// defer_ok runs only on success
	testExpectRun(t, `
	out := ""
	f := func(fail) {
		defer_ok { out += "ok " }
		defer_err { out += "err " }
		if fail { throw "x" }
	}
	f(false)
	try { f(true) } catch {}
	return out`, nil, Str("ok err "))

	// throw inside a defer becomes the function's error
	expectErrHas(t, `f := func() { defer { throw "from-defer" }; return 1 }; return f()`,
		newOpts(), "from-defer")

	// an error from the body still propagates when no defer suppresses it
	expectErrHas(t, `f := func() { defer { $ret = 1 }; throw "boom" }; return f()`,
		newOpts(), "boom")

	// nested functions have independent defers (inner runs at inner exit)
	testExpectRun(t, `
	out := ""
	g := func() {
		defer { out += "outer " }
		inner := func() { defer { out += "inner " }; out += "in " }
		inner()
		out += "out "
	}
	g()
	return out`, nil, Str("in inner out outer "))

	// conditional defer only registers (and runs) when reached
	testExpectRun(t, `
	f := func(c) { if c { defer { $ret += "y" } }; return "b" }
	return [f(true), f(false)]`, nil, Array{Str("by"), Str("b")})
}

func TestVMCodeStr(t *testing.T) {
	// inline form: body becomes a plain str
	testExpectRun(t, `return code a + b end`, nil, Str("a + b"))
	testExpectRun(t, `return typeName(code x end)`, nil, Str("str"))

	// block form: the body is captured verbatim (NOT evaluated as code), with the
	// opening line's indentation stripped from every line.
	testExpectRun(t, "x := code\n    a := 1\n    b := 2\nend\nreturn x",
		nil, Str("a := 1\nb := 2"))

	// a deeper-indented `end` belongs to the body; the fence is the `end` at the
	// opening indentation.
	testExpectRun(t,
		"x := code\n    begin\n        y := 1\n    end\nend\nreturn x",
		nil, Str("begin\n    y := 1\nend"))

	// nested in a function: dedent uses the opening line's indentation (4 here)
	testExpectRun(t,
		"f := func() {\n    return code\n        a\n    end\n}\nreturn f()",
		nil, Str("a"))

	// a `code` identifier with no matching `end` fence is unaffected
	testExpectRun(t, "code := 41\nreturn code + 1", nil, Int(42))
}

func TestVMBytesLit(t *testing.T) {
	// h"..." decodes a hexadecimal sequence to bytes
	testExpectRun(t, `return h"ffccf1c2"`, nil,
		Bytes{0xff, 0xcc, 0xf1, 0xc2})
	testExpectRun(t, `return typeName(h"ffccf1c2")`, nil, Str("bytes"))
	// uppercase hex digits work too
	testExpectRun(t, `return h"FFCC"`, nil, Bytes{0xff, 0xcc})
	// whitespace inside hex is ignored
	testExpectRun(t, `return h"ff cc f1 c2"`, nil,
		Bytes{0xff, 0xcc, 0xf1, 0xc2})
	// empty hex literal yields empty bytes
	testExpectRun(t, `return len(h"")`, nil, Int(0))

	// b"..." uses the UTF-8 bytes of the string content
	testExpectRun(t, `return b"Hello"`, nil, Bytes("Hello"))
	testExpectRun(t, `return typeName(b"Hello")`, nil, Str("bytes"))
	testExpectRun(t, `return str(b"Hello")`, nil, Str("Hello"))
	// escapes are processed in the regular string form
	testExpectRun(t, `return b"a\nb"`, nil, Bytes("a\nb"))
	// raw string form keeps escapes literal
	testExpectRun(t, "return b`a\\nb`", nil, Bytes(`a\nb`))
	// heredoc form
	testExpectRun(t, "return b\"\"\"\nab\ncd\n\"\"\"", nil, Bytes("ab\ncd"))
	// hex from a raw string
	testExpectRun(t, "return h`ffcc`", nil, Bytes{0xff, 0xcc})

	// usable in larger expressions
	testExpectRun(t, `return b"ab" + b"cd"`, nil, Bytes("abcd"))
	testExpectRun(t, `b := b"Hi"; return b[0]`, nil, Int('H'))

	// invalid hex content is a compile error
	expectErrHas(t, `return h"xy"`, newOpts().CompilerError(),
		`Compile Error: invalid bytes literal`)
	expectErrHas(t, `return h"abc"`, newOpts().CompilerError(),
		`Compile Error: invalid bytes literal`)
}

func TestVMRegexLit(t *testing.T) {
	// the literal evaluates to a regexp object
	testExpectRun(t, `return typeName(/ab+/)`, nil, Str("regexp"))
	testExpectRun(t, `r := /ab+/; return r.match("abbb")`, nil, True)
	testExpectRun(t, `r := /ab+/; return r.match("xyz")`, nil, False)
	// equivalent to the regexp() constructor
	testExpectRun(t, `return (/ab+/).match("abbb") == regexp("ab+").match("abbb")`,
		nil, True)
	// escapes and character classes
	testExpectRun(t, `r := /[0-9]+\/[0-9]+/; return r.match("12/34")`, nil, True)
	// POSIX flag compiles and matches
	testExpectRun(t, `return typeName(/a+/p)`, nil, Str("regexp"))
	testExpectRun(t, `r := /a+/p; return r.match("aaa")`, nil, True)
	// in operand positions
	testExpectRun(t, `f := func(re) { return re.match("ab") }; return f(/ab/)`, nil, True)
	// division still works (after a value `/` is the operator)
	testExpectRun(t, `return 10 / 2`, nil, Int(5))
	testExpectRun(t, `a := 12; b := 3; return a / b`, nil, Int(4))
	testExpectRun(t, `return [6/2, 8/4]`, nil, Array{Int(3), Int(2)})
}

func TestVMRegexpReplace(t *testing.T) {
	// replace method with a string template
	testExpectRun(t, `r := /o/; return r.replace("hello world", "0")`,
		nil, Str("hell0 w0rld"))
	// $1/$2 group expansion
	testExpectRun(t, `r := /(\d+)-(\d+)/; return r.replace("12-34", "$2/$1")`,
		nil, Str("34/12"))
	// callable replacement (invoked per match)
	testExpectRun(t, `r := /[a-z]+/; return r.replace("ab cd", func(m) { return "<" + m + ">" })`,
		nil, Str("<ab> <cd>"))
	// bytes subject -> bytes result
	testExpectRun(t, `r := /o/; return str(r.replace(bytes("foo"), "0"))`,
		nil, Str("f00"))

	// `|` operator yields a unary replacer function
	testExpectRun(t, `r := /o/; f := r | "0"; return f("hello world")`,
		nil, Str("hell0 w0rld"))
	testExpectRun(t, `r := /[a-z]+/; f := r | func(m) { return m + "!" }; return f("ab cd")`,
		nil, Str("ab! cd!"))
	// composes with the pipe operator
	testExpectRun(t, `r := /o/; return "hello world".|(r | "0")`,
		nil, Str("hell0 w0rld"))
}

func TestVMDeferbStmt(t *testing.T) {
	// runs at block exit, LIFO
	testExpectRun(t, `
	out := ""
	{ deferb { out += "d1 " }; deferb { out += "d2 " }; out += "b " }
	out += "after"
	return out`, nil, Str("b d2 d1 after"))

	// deferb_err recovers within the block; execution continues after it
	testExpectRun(t, `
	out := ""
	{
		deferb_err { out += "caught:" + str($err) + " "; $err = nil }
		out += "before "
		throw "boom"
		out += "unreached "
	}
	out += "after"
	return out`, nil, Str("before caught:error: boom after"))

	// deferb_ok runs only on success
	testExpectRun(t, `
	g := func(fail) {
		res := ""
		{
			deferb_ok { res += "ok " }
			deferb_err { res += "err " }
			if fail { throw "x" }
		}
		return res
	}
	return g(false)`, nil, Str("ok "))

	// unsuppressed block error still propagates
	expectErrHas(t, `
	g := func() { { deferb_err { } ; throw "x" } }
	return g()`, newOpts(), "x")

	// a throw inside a deferb handler is captured into $err (recoverable)
	testExpectRun(t, `
	out := ""
	f := func() {
		{
			deferb_err { out += "recovered:" + str($err) + " "; $err = nil }
			deferb { throw "from-handler" }
			out += "body "
		}
		out += "after"
		return out
	}
	return f()`, nil, Str("body recovered:error: from-handler after"))

	// $ret is inappropriate in a block: it is a shadowed nil local and does not
	// reach an enclosing function's return value
	testExpectRun(t, `
	f := func() {
		{ deferb { } }
		return "base"
	}
	return f()`, nil, Str("base"))

	// `return` inside the block runs the handlers AND preserves the value
	testExpectRun(t, `
	out := []
	h := func() {
		{
			deferb { out = append(out, "D") }
			out = append(out, "B")
			return "RET"
		}
	}
	r := h()
	return [r, out]`, nil, Array{Str("RET"), Array{Str("B"), Str("D")}})

	// each block has its own deferb scope (nested blocks are independent)
	testExpectRun(t, `
	out := ""
	{
		deferb { out += "outer " }
		{ deferb { out += "inner " }; out += "in " }
		out += "out "
	}
	return out`, nil, Str("in inner out outer "))
}

func TestVMComprehension(t *testing.T) {
	// array comprehension
	testExpectRun(t, `return [i * 2 for i in [1, 2, 3]]`,
		nil, Array{Int(2), Int(4), Int(6)})
	// with filter
	testExpectRun(t, `return [i for i in [1, 2, 3, 4, 5] if i % 2 == 0]`,
		nil, Array{Int(2), Int(4)})
	// nested generators
	testExpectRun(t, `return [i + j for i in [1, 2] for j in [10, 20]]`,
		nil, Array{Int(11), Int(21), Int(12), Int(22)})
	// empty source
	testExpectRun(t, `return [i for i in []]`, nil, Array{})
	// captures outer variable
	testExpectRun(t, `n := 10; return [i + n for i in [1, 2]]`, nil, Array{Int(11), Int(12)})
	// nested comprehension
	testExpectRun(t, `return [[j for j in [1, 2]] for i in [0, 0]]`,
		nil, Array{Array{Int(1), Int(2)}, Array{Int(1), Int(2)}})

	// dict comprehension: `[expr]` is a computed key (the value of i)
	testExpectRun(t, `return {[i]: i * i for i in [1, 2, 3]}`,
		nil, Dict{"1": Int(1), "2": Int(4), "3": Int(9)})
	// k, v iteration with filter and computed key
	testExpectRun(t, `return {[k]: v * 10 for k, v in {a: 1, b: 2} if v == 2}`,
		nil, Dict{"b": Int(20)})
	// a static key keeps its literal name (last write wins)
	testExpectRun(t, `return {n: i for i in [1, 2, 3]}`, nil, Dict{"n": Int(3)})
	// multiple keys (static + computed) and `_` reads the dict being built
	testExpectRun(t,
		`return {i: 100, x: 10 + i, z: (_.z ?? 20) + i, [i]: i * i for i in [1, 2, 3]}`,
		nil, Dict{
			"i": Int(100), "x": Int(13), "z": Int(26),
			"1": Int(1), "2": Int(4), "3": Int(9),
		})

	// loop variable does not leak into the surrounding scope
	expectErrHas(t, `x := [i for i in [1, 2]]; return i`,
		newOpts().CompilerError(), `Compile Error: unresolved reference "i"`)
}

func TestVMMatchExpr(t *testing.T) {
	// expression form: first matching arm wins, else is the default
	const f = `f := func(x) { return match (x) { 1: "one", 2: "two", else: "other" } }; `
	testExpectRun(t, f+`return f(1)`, nil, Str("one"))
	testExpectRun(t, f+`return f(2)`, nil, Str("two"))
	testExpectRun(t, f+`return f(9)`, nil, Str("other"))

	// single-line comma separators
	testExpectRun(t, `return match (2) { 1: "a", 2: "b", else: "z" }`, nil, Str("b"))
	// newline separators
	testExpectRun(t, "return match (3) {\n1: \"a\"\n3: \"c\"\nelse: \"z\"\n}", nil, Str("c"))

	// non-literal conditions evaluated against the subject
	testExpectRun(t, `a := 10; b := 20; return match (20) { a: "x", b: "y" }`,
		nil, Str("y"))
	// string subject
	testExpectRun(t, `return match ("hi") { "hi": 1, "bye": 2, else: 0 }`, nil, Int(1))

	// no matching arm and no else => throws
	expectErrHas(t, `return match (7) { 1: "a" }`, newOpts(),
		"match: no matching arm")

	// statement form: runs the matching block; returns from the enclosing func
	const g = `g := func(x) { match (x) { 1 { return "ONE" }, 2 { return "TWO" }, else { return "OTHER" } } }; `
	testExpectRun(t, g+`return g(1)`, nil, Str("ONE"))
	testExpectRun(t, g+`return g(2)`, nil, Str("TWO"))
	testExpectRun(t, g+`return g(5)`, nil, Str("OTHER"))

	// statement form, no else, no match => falls through (no effect)
	testExpectRun(t, `
	var out = 0
	k := func(x) { match (x) { 1 { out = 100 } } }
	k(5)
	return out`, nil, Int(0))
	testExpectRun(t, `
	var out = 0
	k := func(x) { match (x) { 1 { out = 100 } } }
	k(1)
	return out`, nil, Int(100))

	// match is usable inline as an expression value
	testExpectRun(t, `return 1 + match (2) { 2: 40, else: 0 }`, nil, Int(41))

	// `match` keyword does not break selector method names (e.g. regexp.match)
	testExpectRun(t, `re := regexp("ab"); return re.match("ab")`, nil, True)
}

func TestVMSpreadLiterals(t *testing.T) {
	// array merge
	testExpectRun(t, `a := [2, 3]; b := [5, 6]; return [1, *a, 4, *b]`,
		nil, Array{Int(1), Int(2), Int(3), Int(4), Int(5), Int(6)})
	testExpectRun(t, `a := [2, 3]; return [*a]`, nil, Array{Int(2), Int(3)})
	testExpectRun(t, `a := [2]; b := [3]; return [*a, *b]`, nil, Array{Int(2), Int(3)})
	testExpectRun(t, `return []`, nil, Array{})
	// spread yields a copy, source not mutated/aliased
	testExpectRun(t, `a := [1, 2]; b := [*a]; b[0] = 9; return [a, b]`,
		nil, Array{Array{Int(1), Int(2)}, Array{Int(9), Int(2)}})

	// dict merge (later keys win)
	testExpectRun(t, `b := {x:9}; d := {y:8, x:100}; return {a:1, *b, e:2, *d}`,
		nil, Dict{"a": Int(1), "e": Int(2), "x": Int(100), "y": Int(8)})
	testExpectRun(t, `b := {x:9}; return {*b}`, nil, Dict{"x": Int(9)})
	testExpectRun(t, `b := {x:9}; d := {y:8}; return {*b, *d}`,
		nil, Dict{"x": Int(9), "y": Int(8)})
	// spread yields a copy, source not mutated/aliased
	testExpectRun(t, `c := {x:1}; d := {*c}; d.x = 9; return [c, d]`,
		nil, Array{Dict{"x": Int(1)}, Dict{"x": Int(9)}})
}

func TestVMMixedParamsDestructure(t *testing.T) {
	const x = `x := (1, 2, *[3, 4]; c=5, **{d:6, e:7}); `

	// full destructure: positional with rest + named with rename/default/rest
	testExpectRun(t, x+`(a, b, **pos_rest; c, p:d, r=2, **named_rest) := x; return [a, b, pos_rest]`,
		nil, Array{Int(1), Int(2), Array{Int(3), Int(4)}})
	testExpectRun(t, x+`(a, b, **pos_rest; c, p:d, r=2, **named_rest) := x; return [c, p, r, named_rest]`,
		nil, Array{Int(5), Int(6), Int(2), Dict{"e": Int(7)}})

	// positional only
	testExpectRun(t, x+`(m, n) := x; return [m, n]`, nil, Array{Int(1), Int(2)})
	// positional rest collects the remainder
	testExpectRun(t, x+`(m, **rest) := x; return [m, rest]`,
		nil, Array{Int(1), Array{Int(2), Int(3), Int(4)}})

	// `=` assigns to pre-defined variables
	testExpectRun(t, x+`var (p, q, rest); (p, q, **rest; ) = x; return [p, q, rest]`,
		nil, Array{Int(1), Int(2), Array{Int(3), Int(4)}})

	// the MixedParams value round-trips through .positional/.named
	testExpectRun(t, x+`return x.positional`, nil, Array{Int(1), Int(2), Int(3), Int(4)})
	testExpectRun(t, x+`return dict(x.named)`, nil,
		Dict{"c": Int(5), "d": Int(6), "e": Int(7)})
}

func TestVMDictDestructure(t *testing.T) {
	const d = `d := {a:2, b:3, x:4, y:5}; `

	// `:=` defines new locals; plain name reads same-named key
	testExpectRun(t, d+`(;a) := d; return a`, nil, Int(2))
	// rename: variable `_b` from dict key `b`
	testExpectRun(t, d+`(;_b:b) := d; return _b`, nil, Int(3))
	// fallback default used only when the key is absent
	testExpectRun(t, d+`(;r=9) := d; return r`, nil, Int(9))
	testExpectRun(t, d+`(;a=9) := d; return a`, nil, Int(2))
	// **rest collects the keys not consumed
	testExpectRun(t, d+`(;a, _b:b, **other) := d; return other`,
		nil, Dict{"x": Int(4), "y": Int(5)})
	testExpectRun(t, d+`(;a, _b:b, **other) := d; return [a, _b]`,
		nil, Array{Int(2), Int(3)})
	// rest with all keys consumed -> empty dict; source dict is not mutated
	testExpectRun(t, d+`(;a:a, b2:b, x2:x, y2:y, **other) := d; return [other, d]`,
		nil, Array{Dict{}, Dict{"a": Int(2), "b": Int(3), "x": Int(4), "y": Int(5)}})

	// `=` assigns to predefined variables (all must already exist)
	testExpectRun(t, d+`var (p, q, rest); (;p:a, q:b, **rest) = d; return [p, q, rest]`,
		nil, Array{Int(2), Int(3), Dict{"x": Int(4), "y": Int(5)}})

	// errors
	expectErrHas(t, `d := {a:1}; (;a) = d; return a`,
		newOpts().CompilerError(), `Compile Error: unresolved reference "a"`)
}

func TestVMDestructuring(t *testing.T) {
	expectErrHas(t, `x, y = nil; return x`,
		newOpts().CompilerError(), `Compile Error: unresolved reference "x"`)
	expectErrHas(t, `var (x, y); x, y := nil; return x`,
		newOpts().CompilerError(), `Compile Error: no new variable on left side`)
	expectErrHas(t, `x, y = 1, 2`, newOpts().CompilerError(),
		`Compile Error: multiple expressions on the right side not supported`)

	testExpectRun(t, `x, y := nil; return x`, nil, Nil)
	testExpectRun(t, `x, y := nil; return y`, nil, Nil)
	testExpectRun(t, `x, y := 1; return x`, nil, Int(1))
	testExpectRun(t, `x, y := 1; return y`, nil, Nil)
	testExpectRun(t, `x, y := []; return x`, nil, Nil)
	testExpectRun(t, `x, y := []; return y`, nil, Nil)
	testExpectRun(t, `x, y := [1]; return x`, nil, Int(1))
	testExpectRun(t, `x, y := [1]; return y`, nil, Nil)
	testExpectRun(t, `x, y := [1, 2]; return x`, nil, Int(1))
	testExpectRun(t, `x, y := [1, 2]; return y`, nil, Int(2))
	testExpectRun(t, `x, y := [1, 2, 3]; return x`, nil, Int(1))
	testExpectRun(t, `x, y := [1, 2, 3]; return y`, nil, Int(2))
	testExpectRun(t, `var x; x, y := [1]; return x`, nil, Int(1))
	testExpectRun(t, `var x; x, y := [1]; return y`, nil, Nil)

	testExpectRun(t, `x, y, z := nil; return x`, nil, Nil)
	testExpectRun(t, `x, y, z := nil; return y`, nil, Nil)
	testExpectRun(t, `x, y, z := nil; return z`, nil, Nil)
	testExpectRun(t, `x, y, z := 1; return x`, nil, Int(1))
	testExpectRun(t, `x, y, z := 1; return y`, nil, Nil)
	testExpectRun(t, `x, y, z := 1; return z`, nil, Nil)
	testExpectRun(t, `x, y, z := []; return x`, nil, Nil)
	testExpectRun(t, `x, y, z := []; return y`, nil, Nil)
	testExpectRun(t, `x, y, z := []; return z`, nil, Nil)
	testExpectRun(t, `x, y, z := [1]; return x`, nil, Int(1))
	testExpectRun(t, `x, y, z := [1]; return y`, nil, Nil)
	testExpectRun(t, `x, y, z := [1]; return z`, nil, Nil)
	testExpectRun(t, `x, y, z := [1, 2]; return x`, nil, Int(1))
	testExpectRun(t, `x, y, z := [1, 2]; return y`, nil, Int(2))
	testExpectRun(t, `x, y, z := [1, 2]; return z`, nil, Nil)
	testExpectRun(t, `x, y, z := [1, 2, 3]; return x`, nil, Int(1))
	testExpectRun(t, `x, y, z := [1, 2, 3]; return y`, nil, Int(2))
	testExpectRun(t, `x, y, z := [1, 2, 3]; return z`, nil, Int(3))
	testExpectRun(t, `x, y, z := [1, 2, 3, 4]; return z`, nil, Int(3))

	// test index assignments
	testExpectRun(t, `
	var (x = {}, y, z)
	x.a, y, z = [1, 2, 3, 4]; return x`, nil, Dict{"a": Int(1)})
	testExpectRun(t, `
	var (x = {}, y, z)
	x.a, y, z = [1, 2, 3, 4]; return y`, nil, Int(2))
	testExpectRun(t, `
	var (x = {}, y, z)
	x.a, y, z = [1, 2, 3, 4]; return z`, nil, Int(3))

	testExpectRun(t, `
	var (x = {}, y, z)
	y, x.a, z = [1, 2, 3, 4]; return x`, nil, Dict{"a": Int(2)})
	testExpectRun(t, `
	var (x = {}, y, z)
	y, x.a, z = [1, 2, 3, 4]; return y`, nil, Int(1))
	testExpectRun(t, `
	var (x = {}, y, z)
	y, x.a, z = [1, 2, 3, 4]; return z`, nil, Int(3))

	testExpectRun(t, `
	var (x = [0], y, z)
	x[0], y, z = [1, 2, 3, 4]; return x`, nil, Array{Int(1)})
	testExpectRun(t, `
	var (x = [0], y, z)
	x[0], y, z = [1, 2, 3, 4]; return y`, nil, Int(2))
	testExpectRun(t, `
	var (x = [0], y, z)
	x[0], y, z = [1, 2, 3, 4]; return z`, nil, Int(3))

	testExpectRun(t, `
	var (x = [0], y, z)
	y, x[0], z = [1, 2, 3, 4]; return x`, nil, Array{Int(2)})
	testExpectRun(t, `
	var (x = [0], y, z)
	y, x[0], z = [1, 2, 3, 4]; return y`, nil, Int(1))
	testExpectRun(t, `
	var (x = [0], y, z)
	y, x[0], z = [1, 2, 3, 4]; return z`, nil, Int(3))

	// test function calls
	testExpectRun(t, `
	fn := func() { 
		return [1, error("abc")]
	}
	x, y := fn()
	return [x, str(y)]`, nil, Array{Int(1), Str("error: abc")})

	testExpectRun(t, `
	fn := func() { 
		return [1]
	}
	x, y := fn()
	return [x, y]`, nil, Array{Int(1), Nil})
	testExpectRun(t, `
	fn := func() { 
		return
	}
	x, y := fn()
	return [x, y]`, nil, Array{Nil, Nil})
	testExpectRun(t, `
	fn := func() { 
		return [1, 2, 3]
	}
	x, y := fn()
	t := {a: x}
	return [x, y, t]`, nil, Array{Int(1), Int(2), Dict{"a": Int(1)}})
	testExpectRun(t, `
	fn := func() { 
		return {}
	}
	x, y := fn()
	return [x, y]`, nil, Array{Dict{}, Nil})
	testExpectRun(t, `
	fn := func(v) { 
		return [1, v, 3]
	}
	var x = 10
	x, y := fn(x)
	t := {a: x}
	return [x, y, t]`, nil, Array{Int(1), Int(10), Dict{"a": Int(1)}})

	// test any expression
	testExpectRun(t, `x, y :=  {}; return [x, y]`, nil, Array{Dict{}, Nil})
	testExpectRun(t, `
	var x = 2
	if x > 0 {
		fn := func(v) { 
			return [3*v, 4*v]
		}
		var y
		x, y = fn(x)
		if y != 8 {
			throw sprintf("y value expected: %s, got: %s", 8, y)
		}
	}
	return x
	`, nil, Int(6))
	testExpectRun(t, `
	var x = 2
	if x > 0 {
		fn := func(v) { 
			return [3*v, 4*v]
		}
		// new x symbol is created within if block
		// x in upper block is not affected
		x, y := fn(x)
		if y != 8 {
			throw sprintf("y value expected: %s, got: %s", 8, y)
		}
	}
	return x
	`, nil, Int(2))

	testExpectRun(t, `
	var x = 2
	if x > 0 {
		fn := func(v) {
			try {
				ret := v/2
			} catch err {
				return [0, err]
			} finally {
				if err == nil {
					return ret
				}
			}
		}
		a, err := fn("str")
		if !isError(err) {
			throw err
		}
		if a != 0 {
			throw "a is not 0"
		}
		a, err = fn(6)
		if err != nil {
			throw sprintf("unexpected error: %s", err)
		}
		if a != 3 {
			throw "a is not 3"
		}
		x = a
	}
	// return map to check stack pointer is correct
	return {x: x}
	`, nil, Dict{"x": Int(3)})
	testExpectRun(t, `
	for x,y := [1, 2]; true; x++ {
		if x == 10 {
			return [x, y]
		}
	}
	`, nil, Array{Int(10), Int(2)})
	testExpectRun(t, `
	if x,y := [1, 2]; true {
		return [x, y]
	}
	`, nil, Array{Int(1), Int(2)})
	testExpectRun(t, `
	var x = 0
	for true {
		x, y := [x]
		x++
		break
	}
	return x`, nil, Int(0))
	testExpectRun(t, `
	x, y := func(n) {
		return repeat([n], n)
	}(3)
	return [x, y]`, nil, Array{Int(3), Int(3)})
	// closures
	testExpectRun(t, `
	var x = 10
	a, b := func(n) {
		x = n
	}(3)
	return [x, a, b]`, nil, Array{Int(3), Nil, Nil})
	testExpectRun(t, `
	var x = 10
	a, b := func(*args) {
		x, y := args
		return [x, y]
	}(1, 2)
	return [x, a, b]`, nil, Array{Int(10), Int(1), Int(2)})
	testExpectRun(t, `
	var x = 10
	a, b := func(*args) {
		var y
		x, y = args
		return [x, y]
	}(1, 2)
	return [x, a, b]`, nil, Array{Int(1), Int(1), Int(2)})

	// return implicit array if return statement's expressions are comma
	// separated which is a part of destructuring implementation to mimic multi
	// return values.
	parseErr := `Parse Error: expected operand, found 'EOF'`
	expectErrHas(t, `return 1,`,
		newOpts().CompilerError(), parseErr)
	expectErrHas(t, `return 1, 2,`,
		newOpts().CompilerError(), parseErr)
	expectErrHas(t, `var a; return a,`,
		newOpts().CompilerError(), parseErr)
	expectErrHas(t, `var (a, b); return a, b,`,
		newOpts().CompilerError(), parseErr)
	expectErrHas(t, `return 1,`,
		newOpts().CompilerError(), parseErr)
	expectErrHas(t, `return 1, 2,`,
		newOpts().CompilerError(), parseErr)
	expectErrHas(t, `var a; return a,`,
		newOpts().CompilerError(), parseErr)
	expectErrHas(t, `var (a, b); return a, b,`,
		newOpts().CompilerError(), parseErr)

	parseErr = `Parse Error: expected operand, found '}'`
	expectErrHas(t, `func(){ return 1, }`,
		newOpts().CompilerError(), parseErr)
	expectErrHas(t, `func(){ return 1, 2,}`,
		newOpts().CompilerError(), parseErr)

	expectErrHas(t, `func(){ var a; return a,}`,
		newOpts().CompilerError(), parseErr)
	expectErrHas(t, `func(){ var (a, b); return a, b,}`,
		newOpts().CompilerError(), parseErr)
	expectErrHas(t, `func(){ return 1,}`,
		newOpts().CompilerError(), parseErr)
	expectErrHas(t, `func(){ return 1, 2,}`,
		newOpts().CompilerError(), parseErr)
	expectErrHas(t, `func(){ var a; return a,}`,
		newOpts().CompilerError(), parseErr)
	expectErrHas(t, `func(){ var (a, b); return a, b,}`,
		newOpts().CompilerError(), parseErr)

	testExpectRun(t, `return 1, 2`, nil, Array{Int(1), Int(2)})
	testExpectRun(t, `a := 1; return a, a`, nil, Array{Int(1), Int(1)})
	testExpectRun(t, `a := 1; return a, 2`, nil, Array{Int(1), Int(2)})
	testExpectRun(t, `a := 1; return 2, a`, nil, Array{Int(2), Int(1)})
	testExpectRun(t, `a := 1; return 2, a, [3]`, nil,
		Array{Int(2), Int(1), Array{Int(3)}})
	testExpectRun(t, `a := 1; return [2, a], [3]`, nil,
		Array{Array{Int(2), Int(1)}, Array{Int(3)}})
	testExpectRun(t, `return {}, []`, nil, Array{Dict{}, Array{}})
	testExpectRun(t, `return func(){ return 1}(), []`, nil, Array{Int(1), Array{}})
	testExpectRun(t, `return func(){ return 1}(), [2]`, nil,
		Array{Int(1), Array{Int(2)}})
	testExpectRun(t, `
	f := func() {
		return 1, 2
	}
	a, b := f()
	return a, b`, nil, Array{Int(1), Int(2)})
	testExpectRun(t, `
	a, b := func() {
		return 1, error("x")
	}()
	return a, "" + b`, nil, Array{Int(1), Str("error: x")})
	testExpectRun(t, `
	a, b := func(a, b) {
		return a + 1, b + 1
	}(1, 2)
	return a, b, a*2, 3/b`, nil, Array{Int(2), Int(3), Int(4), Int(1)})
	testExpectRun(t, `
	return func(a, b) {
		return a + 1, b + 1
	}(1, 2), 4`, nil, Array{Array{Int(2), Int(3)}, Int(4)})

	testExpectRun(t, `
	param *args

	mapEach := func(seq, fn) {
	
		if !isArray(seq) {
			return error("want array, got " + typeName(seq))
		}
	
		var out = []
	
		if sz := len(seq); sz > 0 {
			out = repeat([0], sz)
		} else {
			return out
		}
	
		try {
			for i, v in seq {
				out[i] = fn(v)
			}
		} catch err {
			println(err)
		} finally {
			return out, err
		}
	}
	
	global multiplier
	
	v, err := mapEach(args, func(x) { return x*multiplier })
	if err != nil {
		return err
	}
	return v
	`, newOpts().
		Globals(Dict{"multiplier": Int(2)}).
		Args(Int(1), Int(2), Int(3), Int(4)),
		Array{Int(2), Int(4), Int(6), Int(8)})

	testExpectRun(t, `
	global goFunc
	// ...
	v, err := goFunc(2)
	if err != nil {
		return str(err)
	}
	`, newOpts().
		Globals(Dict{"goFunc": &Function{
			Value: func(Call) (Object, error) {
				// ...
				return Array{
					Nil,
					ErrIndexOutOfBounds.NewError("message"),
				}, nil
			},
		}}),
		Str("IndexOutOfBoundsError: message"))
}

func TestVMConst(t *testing.T) {
	expectErrHas(t, `const x = 1; x = 2`, newOpts().CompilerError(),
		`Compile Error: assignment to constant variable "x"`)
	expectErrHas(t, `const x = 1; x := 2`, newOpts().CompilerError(),
		`Compile Error: "x" redeclared in this block`)
	expectErrHas(t, `const (x = 1, x = 2)`, newOpts().CompilerError(),
		`Compile Error: "x" redeclared in this block`)
	expectErrHas(t, `const x`, newOpts().CompilerError(),
		`Parse Error: missing initializer in const declaration`)
	expectErrHas(t, `const (x, y = 2)`, newOpts().CompilerError(),
		`Parse Error: missing initializer in const declaration`)

	// After iota support, `const (x=1,y)` does not throw error, like ToInterface. It
	// uses last expression as initializer.
	testExpectRun(t, `const (x = 1, y)`, nil, Nil)

	expectErrHas(t, `const (x, y)`, newOpts().CompilerError(),
		`Parse Error: missing initializer in const declaration`)
	expectErrHas(t, `
	const x = 1
	func f() {
		x = 2
	}`, newOpts().CompilerError(),
		`Compile Error: assignment to constant variable "x"`)
	expectErrHas(t, `
	const x = 1
	if x > 0 {
		x = 2
	}`, newOpts().CompilerError(),
		`Compile Error: assignment to constant variable "x"`)
	expectErrHas(t, `
	const x = 1
	if x > 0 {
		return func() {
			x = 2
		}
	}`, newOpts().CompilerError(),
		`Compile Error: assignment to constant variable "x"`)
	expectErrHas(t, `
	const x = 1
	if x = 2; x > 0 {
		return
	}`, newOpts().CompilerError(),
		`Compile Error: assignment to constant variable "x"`)
	expectErrHas(t, `
	const x = 1
	for x = 1; x < 10; x++ {
		return
	}`, newOpts().CompilerError(),
		`Compile Error: assignment to constant variable "x"`)
	expectErrHas(t, `
	const x = 1
	func f() {
		var y
		x, y = [1, 2]
	}`, newOpts().CompilerError(),
		`Compile Error: assignment to constant variable "x"`)
	expectErrHas(t, `
	x := 1
	func f() {
		const y = 2
		x, y = [1, 2]
	}`, newOpts().CompilerError(),
		`Compile Error: assignment to constant variable "y"`)
	expectErrHas(t, `const x = 1;global x`, newOpts().CompilerError(),
		`Compile Error: "x" redeclared in this block`)
	expectErrHas(t, `const x = 1;param x`, newOpts().CompilerError(),
		`Compile Error: "x" redeclared in this block`)
	expectErrHas(t, `global x; const x = 1`, newOpts().CompilerError(),
		`Compile Error: "x" redeclared in this block`)
	expectErrHas(t, `param x; const x = 1`, newOpts().CompilerError(),
		`Compile Error: "x" redeclared in this block`)
	expectErrHas(t, `
	const x = 1
	if [2] { // not optimized
		x = 2
	}`, newOpts().CompilerError(),
		`Compile Error: assignment to constant variable "x"`)
	expectErrHas(t, `
	const x = 1
	if [2] { // not optimized
		func f() {
			x = 2
		}
	}`, newOpts().CompilerError(),
		`Compile Error: assignment to constant variable "x"`)
	expectErrHas(t, `
	const x = 1
	if x {
		func f() {
			func f2() {
				for {
					x = 2
				}
			}
		}
	}`, newOpts().CompilerError(),
		`Compile Error: assignment to constant variable "x"`)

	// FIXME: Compiler does not compile if or else blocks if condition is
	// a *BoolLit (which may be reduced by optimizer). So compiler does not
	// check whether a constant is reassigned in block to throw an error.
	// A few examples for this issue.
	testExpectRun(t, `
	const x = 1
	if true {
		
	} else {
		// block is not compiled
		x = 2
	}
	return x
	`, nil, Int(1))
	testExpectRun(t, `
	const x = 1
	if false {
		// block is not compiled
		x = 2
	}
	return x
	`, nil, Int(1))

	testExpectRun(t, `const x = 1; return x`, nil, Int(1))
	testExpectRun(t, `const x = "1"; return x`, nil, Str("1"))
	testExpectRun(t, `const x = []; return x`, nil, Array{})
	testExpectRun(t, `const x = []; return x`, nil, Array{})
	testExpectRun(t, `const x = nil; return x`, nil, Nil)
	testExpectRun(t, `const (x = 1, y = "2"); return x, y`, nil,
		Array{Int(1), Str("2")})
	testExpectRun(t, `
	const (
		x = 1
		y = "2"
	)
	return x, y`, nil, Array{Int(1), Str("2")})
	testExpectRun(t, `
	const (
		x = 1
		y = x + 1
	)
	return x, y`, nil, Array{Int(1), Int(2)})
	testExpectRun(t, `
	const x = 1
	return func() {
		const x = x + 1
		return x
	}()`, nil, Int(2))
	testExpectRun(t, `
	const x = 1
	return func() {
		x := x + 1
		return x
	}()`, nil, Int(2))
	testExpectRun(t, `
	const x = 1
	return func() {
		return func() {
			return x + 1
		}()
	}()`, nil, Int(2))
	testExpectRun(t, `
	const x = 1
	for x := 10; x < 100; x++{
		return x
	}`, nil, Int(10))
	testExpectRun(t, `
	const (i = 1, v = 2)
	for i,v in [10] {
		v = -1
		return i
	}`, nil, Int(0))
	testExpectRun(t, `
	const x = 1
	return func() {
		const y = 2
		const x = y
		return x
	}() + x
	`, nil, Int(3))
	testExpectRun(t, `
	const x = 1
	return func() {
		const y = 2
		var x = y
		return x
	}() + x
	`, nil, Int(3))
	testExpectRun(t, `
	const x = 1
	func() {
		x, y := [2, 3]
	}()
	return x
	`, nil, Int(1))
	testExpectRun(t, `
	const x = 1
	for i := 0; i < 1; i++ {
		x, y := [2, 3]
		break
	}
	return x
	`, nil, Int(1))
	testExpectRun(t, `
	const x = 1
	if [1] {
		x, y := [2, 3]
	}
	return x
	`, nil, Int(1))

	testExpectRun(t, `
	return func() {
		const x = 1
		func() {
			x, y := [2, 3]
		}()
		return x
	}()
	`, nil, Int(1))
	testExpectRun(t, `
	return func() {
		const x = 1
		for i := 0; i < 1; i++ {
			x, y := [2, 3]
			break
		}
		return x
	}()
	`, nil, Int(1))
	testExpectRun(t, `
	return func(){
		const x = 1
		if [1] {
			x, y := [2, 3]
		}
		return x
	}()
	`, nil, Int(1))
	testExpectRun(t, `
	return func(){
		const x = 1
		if [1] {
			var y
			x, y := [2, 3]
		}
		return x
	}()
	`, nil, Int(1))
}

func TestConstIota(t *testing.T) {
	testExpectRun(t, `const x = iota; return x`, nil, Int(0))
	testExpectRun(t, `const x = iota; const y = iota; return x, y`, nil, Array{Int(0), Int(0)})
	testExpectRun(t, `const(x = iota, y = iota); return x, y`, nil, Array{Int(0), Int(1)})
	testExpectRun(t, `const(x = iota, y); return x, y`, nil, Array{Int(0), Int(1)})

	testExpectRun(t, `const(x = 1+iota, y); return x, y`, nil, Array{Int(1), Int(2)})
	testExpectRun(t, `const(x = 1+iota, y=iota); return x, y`, nil, Array{Int(1), Int(1)})
	testExpectRun(t, `const(x = 1+iota, y, z); return x, y, z`, nil, Array{Int(1), Int(2), Int(3)})
	testExpectRun(t, `const(x = iota+1, y, z); return x, y, z`, nil, Array{Int(1), Int(2), Int(3)})

	testExpectRun(t, `const(_ = iota+1, y, z); return y, z`, nil, Array{Int(2), Int(3)})

	testExpectRun(t, `
	const (
		x = [iota]
	)
	return x`, nil, Array{Int(0)})

	testExpectRun(t, `
	const (
		x = []
	)
	return x`, nil, Array{})

	testExpectRun(t, `
	const (
		x = [iota, iota]
	)
	return x`, nil, Array{Int(0), Int(0)})

	testExpectRun(t, `
	const (
		x = [iota, iota]
		y
	)
	return x, y`, nil, Array{Array{Int(0), Int(0)}, Array{Int(1), Int(1)}})

	testExpectRun(t, `
	const (
		x = [iota, iota]
		y
		z
	)
	return x, y, z`, nil,
		Array{Array{Int(0), Int(0)}, Array{Int(1), Int(1)}, Array{Int(2), Int(2)}})

	testExpectRun(t, `
	const (
		x = [iota, iota]
		y
	)
	x[0] = 2
	return x, y`, nil, Array{Array{Int(2), Int(0)}, Array{Int(1), Int(1)}})

	testExpectRun(t, `
	const (
		x = {}
	)
	return x`, nil, Dict{})

	testExpectRun(t, `
	const (
		x = {iota: 1}
	)
	return x`, nil, Dict{"iota": Int(1)})

	testExpectRun(t, `
	const (
		x = {k: iota}
	)
	return x`, nil, Dict{"k": Int(0)})

	testExpectRun(t, `
	const (
		x = {k: iota}
		y
	)
	return x, y`, nil, Array{Dict{"k": Int(0)}, Dict{"k": Int(1)}})

	testExpectRun(t, `
	const (
		x = {k: iota}
		y
	)
	x["k"] = 2
	return x, y`, nil, Array{Dict{"k": Int(2)}, Dict{"k": Int(1)}})

	testExpectRun(t, `
	const (
		x = {k: iota}
		y
		z
	)
	return x, y, z`, nil,
		Array{Dict{"k": Int(0)}, Dict{"k": Int(1)}, Dict{"k": Int(2)}})

	testExpectRun(t, `
	const (
		_ = 1 << iota
		x
		y
	)
	return x, y`, nil, Array{Int(2), Int(4)})

	testExpectRun(t, `
	const (
		x = 1 << iota
		_
		y
	)
	return x, y`, nil, Array{Int(1), Int(4)})

	testExpectRun(t, `
	const (
		x = 1 << iota
		a
		y = a
		z
	)
	return x, y, z`, nil, Array{Int(1), Int(2), Int(2)})

	testExpectRun(t, `
	const (
		x = 1 << iota
		_
		_
		z
	)
	return x, z`, nil, Array{Int(1), Int(8)})

	testExpectRun(t, `
	iota := 1
	const (
		x = 1 << iota
	)
	return x, iota`, nil, Array{Int(2), Int(1)})

	testExpectRun(t, `
	iota := 1
	const (
		x = 1 << iota
		y
	)
	return x, y`, nil, Array{Int(2), Int(2)})

	expectErrHas(t, `const iota = 1`,
		newOpts().CompilerError(), "Compile Error: assignment to iota")

	expectErrHas(t, `const iota = iota + 1`,
		newOpts().CompilerError(), "Compile Error: assignment to iota")

	expectErrHas(t, `
	const (
		x = 1 << iota
		iota
		y
	)
	return x, iota, y`,
		newOpts().CompilerError(), "Compile Error: assignment to iota")

	expectErrHas(t, `const x = iota; return iota`,
		newOpts().CompilerError(), `Compile Error: unresolved reference "iota"`)

	testExpectRun(t, `
	const (
		x = iota
		y
	)
	iota := 3
	return x, y, iota`, nil, Array{Int(0), Int(1), Int(3)})

	testExpectRun(t, `
	const (
		x = iota
		y
	)
	iota := 3
	const (
		a = 10+iota
		b
	)
	return x, y, iota, a, b`, nil, Array{Int(0), Int(1), Int(3), Int(13), Int(13)})

	testExpectRun(t, `
	const (
		x = iota
		y
	)
	const (
		a = 10+iota
		b
	)
	return x, y, a, b`, nil, Array{Int(0), Int(1), Int(10), Int(11)})

	testExpectRun(t, `
	const (
		x = func() { return 1 }()
		y
		z
	)
	return x, y, z`, nil, Array{Int(1), Int(1), Int(1)})

	testExpectRun(t, `
	const (
		x = func(x) { return x }(iota)
		y
		z
	)
	return x, y, z`, nil, Array{Int(0), Int(1), Int(2)})

	testExpectRun(t, `
	a:=0
	const (
		x = func() { a++; return a }()
		y
		z
	)
	return x, y, z`, nil, Array{Int(1), Int(2), Int(3)})

	testExpectRun(t, `
	const (
		x = 1+iota
		y = func() { return 1+x }()
		z
	)
	return x, y, z`, nil, Array{Int(1), Int(2), Int(2)})

	testExpectRun(t, `
	const (
		x = func() { return 1 }
		y
		z
	)
	return x(), y(), z()`, nil, Array{Int(1), Int(1), Int(1)})

	testExpectRun(t, `
	const (
		x = func() { return 1 }
		y
		z
	)
	return str([x, y, z])`, nil,
		Str("[‹compiledFunction: (main).#1()›, ‹compiledFunction: (main).#2()›, ‹compiledFunction: (main).#3()›]"))

	testExpectRun(t, `
	var a
	const (
		x = func() { return a }
		y
		z
	)
	return x != y && y != z`, nil, True)

	testExpectRun(t, `
	return func() {
		const (
			x = 1 << iota
			_
			y
		)
		return x, y
	}()`, nil, Array{Int(1), Int(4)})

	testExpectRun(t, `
	iota := 2
	return func() {
		const (
			x = 1 << iota
			_
			y
		)
		return x, y
	}()`, nil, Array{Int(4), Int(4)})

	testExpectRun(t, `
	const (
		x = 1 + iota + func() { 
			const (
				_ = iota
				r
			)
			return r
		}()
		y
		_
	)
	return x,y`, nil, Array{Int(2), Int(3)})

	testExpectRun(t, `
	const (x = iota%2?"odd":"even", y, z)
	return x,y,z`, nil, Array{Str("even"), Str("odd"), Str("even")})
}

func TestVM_Invoke(t *testing.T) {
	applyPool := &Function{
		FuncName: "applyPool",
		Value: func(c Call) (Object, error) {
			inv := NewInvoker(c.VM, c.Args.Shift())
			inv.Acquire()
			defer inv.Release()
			return inv.Invoke(c.Args, &c.NamedArgs)
		},
	}
	applyNoPool := &Function{
		FuncName: "applyNoPool",
		Value: func(c Call) (Object, error) {
			args := make([]Object, 0, c.Args.Length()-1)
			for i := 1; i < c.Args.Length(); i++ {
				args = append(args, c.Args.Get(i))
			}
			inv := NewInvoker(c.VM, c.Args.Get(0))
			return inv.Invoke(Args{args}, &c.NamedArgs)
		},
	}
	for _, apply := range []*Function{applyPool, applyNoPool} {
		t.Run(apply.FuncName, func(t *testing.T) {
			t.Run("apply", func(t *testing.T) {
				scr := `
global apply
sum := func(*args) {
	println("called f", args)
	s := 0
	for v in args {
		println("v", v)
		s += v
	}
	return s
}
return apply(sum, 1, 2, 3)
`
				testExpectRun(t, scr,
					newOpts().Globals(Dict{"apply": apply}),
					Int(6),
				)
			})

			t.Run("apply indirect", func(t *testing.T) {
				scr := `
global apply
sum := func(*args) {
	println("sum args", args)
	s := 0
	for v in args {
		println("v", v)
		s += v
	}
	return s
}
f := func(fn, *args) {
	return fn(*args)
}
return apply(f, sum, 1, 2, 3)
`
				testExpectRun(t, scr,
					newOpts().Globals(Dict{"apply": apply}),
					Int(6),
				)
			})

			t.Run("apply indirect 2", func(t *testing.T) {
				scr := `
global apply
sum := func(*args) {
	println("sum args", args)
	s := 0
	for v in args {
		println("v", v)
		s += v
	}
	return s
}
f := func(fn, *args) {
	return apply(fn, *args)
}
return apply(f, sum, 1, 2, 3)
`
				testExpectRun(t, scr,
					newOpts().Globals(Dict{"apply": apply}),
					Int(6),
				)
			})

			t.Run("apply go func", func(t *testing.T) {
				sum := &Function{
					Value: func(c Call) (Object, error) {
						s := Int(0)
						for i := 0; i < c.Args.Length(); i++ {
							s += c.Args.Get(i).(Int)
						}
						return s, nil
					},
				}
				scr := `
global (apply, sum)
return apply(sum, 1, 2, 3)
`
				testExpectRun(t, scr,
					newOpts().Globals(Dict{"apply": apply, "sum": sum}),
					Int(6),
				)
			})

			t.Run("module state", func(t *testing.T) {
				scr := `
module := import("module")
module.counter = 1

global apply

inc := func(a) {
	module := import("module")
	module.counter += a
}
apply(inc, 3)
return module.counter
`
				t.Run("builtin", func(t *testing.T) {
					testExpectRun(t, scr,
						newOpts().
							Globals(Dict{"apply": apply}).
							Module("module", Dict{}),
						Int(4),
					)
				})
				t.Run("source", func(t *testing.T) {
					testExpectRun(t, scr,
						newOpts().
							Globals(Dict{"apply": apply}).
							Module("module", `return {}`),
						Int(4),
					)
				})
			})

			t.Run("closure", func(t *testing.T) {
				scr := `
global apply

counter := 1
f1 := func(a) {
	counter += a
}

f2 := func(a) {
	counter += a
}
apply(f1, 3)
apply(f2, 5)
return counter
`
				testExpectRun(t, scr,
					newOpts().Globals(Dict{"apply": apply}),
					Int(9),
				)
			})

			t.Run("global", func(t *testing.T) {
				scr := `
global apply
global counter

f1 := func(a) {
	counter += a
}

f2 := func(a) {
	counter += a
}
apply(f1, 3)
apply(f2, 5)
return counter
`
				expected := Int(9)
				globals := Dict{"apply": apply, "counter": Int(1)}
				testExpectRun(t, scr,
					newOpts().Globals(globals).Skip2Pass(),
					expected,
				)
				if expected != globals["counter"] {
					t.Fatalf("expected %s, got %v", expected, globals["counter"])
				}
			})
		})
	}
}

type nameCaller struct {
	Dict
	counts map[string]int
}

func (n *nameCaller) CallName(name string, c Call) (Object, error) {
	fn := n.Dict[name]
	args := make([]Object, 0, c.Args.Length())
	for i := 0; i < c.Args.Length(); i++ {
		args = append(args, c.Args.Get(i))
	}
	ret, err := NewInvoker(c.VM, fn).Invoke(Args{args}, &c.NamedArgs)
	n.counts[name]++
	return ret, err
}

var _ NameCallerObject = &nameCaller{}

func TestVMCallName(t *testing.T) {
	newobject := func() *nameCaller {
		var f = &Function{
			Value: func(c Call) (Object, error) {
				return c.Args.Get(0).(Int) + 1, nil
			},
		}

		return &nameCaller{Dict: Dict{"add1": f}, counts: map[string]int{}}
	}
	scr := `
global object

object.sub1 = func(a) {
	return a - 1
}

return [object.add1(10), object.sub1(10)]
`

	t.Run("basic", func(t *testing.T) {
		testExpectRun(t, scr,
			newOpts().Globals(Dict{"object": newobject()}),
			Array{Int(11), Int(9)},
		)
	})

	t.Run("counts single pass", func(t *testing.T) {
		object := newobject()
		testExpectRun(t, scr,
			newOpts().Globals(Dict{"object": object}).Skip2Pass(),
			Array{Int(11), Int(9)},
		)
		if object.counts["add1"] != 1 {
			t.Fatalf("expected 1, got %d", object.counts["add1"])
		}
		if object.counts["sub1"] != 1 {
			t.Fatalf("expected 1, got %d", object.counts["sub1"])
		}
	})

	t.Run("counts all pass", func(t *testing.T) {
		object := newobject()
		testExpectRun(t, scr,
			newOpts().Globals(Dict{"object": object}),
			Array{Int(11), Int(9)},
		)
		if object.counts["add1"] <= 0 {
			t.Fatalf("expected >0, got %d", object.counts["add1"])
		}
		if object.counts["sub1"] <= 0 {
			t.Fatalf("expected >0, got %d", object.counts["sub1"])
		}
	})
}
