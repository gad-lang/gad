package gad_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/gad-lang/gad/parser"
	"github.com/stretchr/testify/require"

	. "github.com/gad-lang/gad"
)

func TestVMBinaryOperator(t *testing.T) {
	// pow
	TestExpectRun(t, `return 2 ** 3`, nil, DecimalFromInt(8))
	TestExpectRun(t, `x := 2; x **= 3; return x`, nil, DecimalFromInt(8))
	TestExpectRun(t, `return binaryOp(TBinOpPow, 2, 3)`, nil, DecimalFromInt(8))

	TestExpectRun(t, `return TBinOpAdd`, nil, TBinOpAdd)
	TestExpectRun(t, `return binaryOp(TBinOpAdd, 1, 1)`, nil, Int(2))

	// custom OP
	TestExpectRun(t, `return binaryOp(TBinOpMul, 2, 10)`, nil, Int(20))
	TestExpectRun(t, `
func binaryOp(_ TBinOpMul, p str, val int) {
	ret := p
	for i := 0; i < val-1; i++ {
		ret += "-" + p
	}
	return ret
}
return "a" * 3`, nil, Str("a-a-a"))
	TestExpectRun(t, `
// get original binary operator handler without methods
bo := rawCaller(binaryOp) 

func binaryOp(_ TBinOpAdd, p str, val str) {
	ret := p
	for i := 0; i < int(val)-1; i++ {
		// cant't uses ret += ... to prevents caller overflows on this method 
		ret = bo(TBinOpAdd, ret, bo(TBinOpAdd, "-", p))
	}
	return ret
}
return "a" + "3"`, nil, Str("a-a-a"))

	TestExpectRun(t, `return [4 ** 3, -4 ** 3, 4 ** -3, -4 ** -3] .| map((v,i) => str(v)) .| collect`, nil, Array{
		Str("64"),
		Str("-64"),
		Str("0.015625"),
		Str("-0.015625"),
	})
}

func TestVMDict(t *testing.T) {
	var d struct{}
	TestExpectRun(t, `return ({a:1} + {b:2})`, nil, Dict{"a": Int(1), "b": Int(2)})
	TestExpectRun(t, `d := {a:1}; d += {b:2}; return d`, nil, Dict{"a": Int(1), "b": Int(2)})
	TestExpectRun(t, `return {a:1,b:2} - ["a"]`, nil, Dict{"b": Int(2)})
	TestExpectRun(t, `return {a:1,b:2} - {a:1}`, nil, Dict{"b": Int(2)})
	TestExpectRun(t, `return {a:1,b:2} - (;a)`, nil, Dict{"b": Int(2)})
	TestExpectRun(t, `param d; return dict((userData(d) + {a:1}).|items()), dict(userData(d))`,
		NewTestOpts().Args(MustNewReflectValue(&d)),
		Array{Dict{"a": Int(1)}, Dict{"a": Int(1)}})
}

func TestVMArray(t *testing.T) {
	TestExpectRun(t, `return [1, 2 * 2, 3 + 3]`, nil, Array{Int(1), Int(4), Int(6)})
	TestExpectRun(t, `return [1, 2] + [3] + {c:4} + (;d=5)`, nil, Array{Int(1), Int(2), Int(3), Int(4), Int(5)})
	// array copy-by-reference
	TestExpectRun(t, `a1 := [1, 2, 3]; a2 := a1; a1[0] = 5; return a2`,
		nil, Array{Int(5), Int(2), Int(3)})
	TestExpectRun(t, `var out; func () { a1 := [1, 2, 3]; a2 := a1; a1[0] = 5; out = a2 }(); return out`,
		nil, Array{Int(5), Int(2), Int(3)})

	// array index set
	expectErrIs(t, `a1 := [1, 2, 3]; a1[3] = 5`, nil, ErrIndexOutOfBounds)

	// index operator
	arr := Array{Int(1), Int(2), Int(3), Int(4), Int(5), Int(6)}
	arrStr := `[1, 2, 3, 4, 5, 6]`
	arrLen := 6
	for idx := 0; idx < arrLen; idx++ {
		TestExpectRun(t, fmt.Sprintf("return %s[%d]", arrStr, idx),
			nil, arr[idx])
		TestExpectRun(t, fmt.Sprintf("return %s[0 + %d]", arrStr, idx),
			nil, arr[idx])
		TestExpectRun(t, fmt.Sprintf("return %s[1 + %d - 1]", arrStr, idx),
			nil, arr[idx])
		TestExpectRun(t, fmt.Sprintf("idx := %d; return %s[idx]", idx, arrStr),
			nil, arr[idx])
		TestExpectRun(t, fmt.Sprintf("return %s.(%d)", arrStr, idx),
			nil, arr[idx])
		TestExpectRun(t, fmt.Sprintf("return %s.(0 + %d)", arrStr, idx),
			nil, arr[idx])
		TestExpectRun(t, fmt.Sprintf("return %s.(1 + %d - 1)", arrStr, idx),
			nil, arr[idx])
		TestExpectRun(t, fmt.Sprintf("idx := %d; return %s.(idx)", idx, arrStr),
			nil, arr[idx])
	}
	expectErrIs(t, fmt.Sprintf("%s[%d]", arrStr, -10), nil, ErrIndexOutOfBounds)
	expectErrIs(t, fmt.Sprintf("%s[%d]", arrStr, arrLen), nil, ErrIndexOutOfBounds)

	// slice operator
	for low := 0; low < arrLen; low++ {
		TestExpectRun(t, fmt.Sprintf("return %s[%d:%d]", arrStr, low, low),
			nil, Array{})
		for high := low; high <= arrLen; high++ {
			TestExpectRun(t, fmt.Sprintf("return %s[%d:%d]", arrStr, low, high),
				nil, arr[low:high])
			TestExpectRun(t, fmt.Sprintf("return %s[0 + %d : 0 + %d]",
				arrStr, low, high), nil, arr[low:high])
			TestExpectRun(t, fmt.Sprintf("return %s[1 + %d - 1 : 1 + %d - 1]",
				arrStr, low, high), nil, arr[low:high])
			TestExpectRun(t, fmt.Sprintf("return %s[:%d]", arrStr, high),
				nil, arr[:high])
			TestExpectRun(t, fmt.Sprintf("return %s[%d:]", arrStr, low),
				nil, arr[low:])
		}
	}

	TestExpectRun(t, fmt.Sprintf("return %s[:]", arrStr), nil, arr)
	TestExpectRun(t, fmt.Sprintf("return %s[%d:%d]", arrStr, 2, 2), nil, Array{})
	TestExpectRun(t, `return "ab"[1]`, nil, Int('b'))
	TestExpectRun(t, `return "ab"[-1]`, nil, Int('b'))
	TestExpectRun(t, `return "ab"[-2]`, nil, Int('a'))
	expectErrIs(t, fmt.Sprintf("return %s[%d:\"\"]", arrStr, -1), nil, ErrType)
	expectErrIs(t, fmt.Sprintf("return %s[:%d]", arrStr, arrLen+1), nil, ErrIndexOutOfBounds)
	expectErrIs(t, fmt.Sprintf("%s[%d:%d]", arrStr, 2, 1), nil, ErrInvalidIndex)
	expectErrIs(t, fmt.Sprintf("%s[%d:]", arrStr, arrLen+1), nil, ErrInvalidIndex)
	expectErrIs(t, "return 1[0:]", nil, ErrType)
	expectErrIs(t, "return 1[0]", nil, ErrNotIndexable)
}

func TestVMDecl(t *testing.T) {
	TestExpectRun(t, `param a; return a`, nil, Nil)
	TestExpectRun(t, `param (a); return a`, nil, Nil)
	TestExpectRun(t, `param *a; return a`, nil, Array{})
	TestExpectRun(t, `param (a, *b); return b`, nil, Array{})
	TestExpectRun(t, `param (a, b); return [a, b]`,
		nil, Array{Nil, Nil})
	TestExpectRun(t, `param a; return a`,
		NewTestOpts().Args(Int(1)), Int(1))
	TestExpectRun(t, `param (a, b); return a + b`,
		NewTestOpts().Args(Int(1), Int(2)), Int(3))
	TestExpectRun(t, `param (a, *b); return b`,
		NewTestOpts().Args(Int(1)), Array{})
	TestExpectRun(t, `param (a, *b); return b+a`,
		NewTestOpts().Args(Int(1), Int(2)), Array{Int(2), Int(1)})
	TestExpectRun(t, `param *a; return a`,
		NewTestOpts().Args(Int(1), Int(2)), Array{Int(1), Int(2)})

	TestExpectRun(t, `param (a, b=2); return [a, b]`, NewTestOpts().Args(Int(1)),
		Array{Int(1), Int(2)})
	TestExpectRun(t, `param (a=-1,**namedArgs); return [a, namedArgs.dict]`, NewTestOpts().
		NamedArgs(Dict{"b": Int(2)}),
		Array{Int(-1), Dict{"b": Int(2)}})
	TestExpectRun(t, `param (;a=-1,**namedArgs); return [a, namedArgs.dict]`, NewTestOpts().
		NamedArgs(Dict{"a": Int(1), "b": Int(2)}),
		Array{Int(1), Dict{"b": Int(2)}})
	TestExpectRun(t, `param (**namedArgs); return namedArgs.dict`, NewTestOpts().
		NamedArgs(Dict{"a": Int(100)}),
		Dict{"a": Int(100)})
	TestExpectRun(t, `param (a, b=100,**namedArgs); return [a, b, namedArgs.dict]`, NewTestOpts().Args(Int(1)).
		NamedArgs(Dict{"b": Int(2), "c": Int(3)}),
		Array{Int(1), Int(2), Dict{"c": Int(3)}})
	TestExpectRun(t, `param (a, b=100,**namedArgs); return [a, b, namedArgs.dict]`, NewTestOpts().Args(Int(1)).
		NamedArgs(Dict{"c": Int(2), "d": Int(3)}),
		Array{Int(1), Int(100), Dict{"c": Int(2), "d": Int(3)}})

	expectErrHas(t, `func(){ param x; }`, NewTestOpts().CompilerError(),
		`Compile Error: param not allowed in this scope`)

	TestExpectRun(t, `global a; return a`, nil, Nil)
	TestExpectRun(t, `global (a); return a`, nil, Nil)
	TestExpectRun(t, `global (a, b); return [a, b]`,
		nil, Array{Nil, Nil})
	TestExpectRun(t, `global a; return a`,
		NewTestOpts().Globals(Dict{"a": Str("ok")}), Str("ok"))
	TestExpectRun(t, `global (a, b); return a+b`,
		NewTestOpts().Globals(Dict{"a": Int(1), "b": Int(2)}), Int(3))
	expectErrHas(t, `func() { global a }`, NewTestOpts().CompilerError(),
		`Compile Error: global not allowed in this scope`)

	TestExpectRun(t, `var a; return a`, nil, Nil)
	TestExpectRun(t, `var (a); return a`, nil, Nil)
	TestExpectRun(t, `var (a = 1); return a`, nil, Int(1))
	TestExpectRun(t, `var (a, b = 1); return a`, nil, Nil)
	TestExpectRun(t, `var (a, b = 1); return b`, nil, Int(1))
	TestExpectRun(t, `var (a,
		b = 1); return a`, nil, Nil)
	TestExpectRun(t, `var (a,
		b = 1); return b`, nil, Int(1))
	TestExpectRun(t, `var (a = 1, b = "x"); return b`, nil, Str("x"))
	TestExpectRun(t, `var (a = 1, b = "x"); return a`, nil, Int(1))
	TestExpectRun(t, `var (a = 1, b); return a`, nil, Int(1))
	TestExpectRun(t, `var (a = 1, b); return b`, nil, Nil)
	TestExpectRun(t, `var b = 1; return b`, nil, Int(1))
	TestExpectRun(t, `var (a, b, c); return [a, b, c]`,
		nil, Array{Nil, Nil, Nil})
	TestExpectRun(t, `return func(a) { var (b = 2,c); return [a, b, c] }(1)`,
		nil, Array{Int(1), Int(2), Nil})

	expectErrHas(t, `param x; global x`, NewTestOpts().CompilerError(),
		`Compile Error: "x" redeclared in this block`)
	expectErrHas(t, `param x; var x`, NewTestOpts().CompilerError(),
		`Compile Error: "x" redeclared in this block`)
	expectErrHas(t, `var x; param x`, NewTestOpts().CompilerError(),
		`Compile Error: "x" redeclared in this block`)
	expectErrHas(t, `var x; global x`, NewTestOpts().CompilerError(),
		`Compile Error: "x" redeclared in this block`)
	expectErrHas(t, `a := 1; if a { param x }`, NewTestOpts().CompilerError(),
		`Compile Error: param not allowed in this scope`)
	expectErrHas(t, `a := 1; if a { global x }`, NewTestOpts().CompilerError(),
		`Compile Error: global not allowed in this scope`)
	expectErrHas(t, `func() { param x }`, NewTestOpts().CompilerError(),
		`Compile Error: param not allowed in this scope`)
	expectErrHas(t, `func() { global x }`, NewTestOpts().CompilerError(),
		`Compile Error: global not allowed in this scope`)

	TestExpectRun(t, `param x; return func(x) { return x }(1)`, nil, Int(1))
	TestExpectRun(t, `
	param x
	return func(x) { 
		for i := 0; i < 1; i++ {
			return x
		}
	}(1)`, nil, Int(1))
	TestExpectRun(t, `
	param x
	func() {
		if x || !x {
			x = 2
		}
	}()
	return x`, NewTestOpts().Args(Int(0)), Int(2))
	TestExpectRun(t, `
	param x
	func() {
		if x || !x {
			func() {
				x = 2
			}()
		}
	}()
	return x`, NewTestOpts().Args(Int(0)), Int(2))
	TestExpectRun(t, `
	param x
	return func(x) { 
		for i := 0; i < 1; i++ {
			return x
		}
	}(1)`, nil, Int(1))
	TestExpectRun(t, `
	global x
	func() {
		if x || !x {
			x = 2
		}
	}()
	return x`, nil, Int(2))
	TestExpectRun(t, `
	global x
	func() {
		if x || !x {
			func() {
				x = 2
			}()
		}
	}()
	return x`, nil, Int(2))
}

func TestVMAssignment(t *testing.T) {
	expectErrHas(t, `a.b := 1`, NewTestOpts().CompilerError(),
		`Compile Error: operator ':=' not allowed with selector`)

	TestExpectRun(t, `a := 1; a = 2; return a`, nil, Int(2))
	TestExpectRun(t, `a := 1; a = a + 4; return a`, nil, Int(5))
	TestExpectRun(t, `a := 1; f1 := func() { a = 2; return a }; return f1()`,
		nil, Int(2))
	TestExpectRun(t, `a := 1; f1 := func() { a := 3; a = 2; return a }; return f1()`,
		nil, Int(2))

	TestExpectRun(t, `a := 1; return a`, nil, Int(1))
	TestExpectRun(t, `a := 1; func() { a = 2 }(); return a`, nil, Int(2))
	TestExpectRun(t, `a := 1; func() { a := 2 }(); return a`, nil, Int(1)) // "a := 2" shadows variable 'a' in upper scope
	TestExpectRun(t, `a := 1; return func() { b := 2; return b }()`, nil, Int(2))
	TestExpectRun(t, `
	return func() { 
		a := 2
		func() {
			a = 3 // a is free (non-local) variable
		}()
		return a
	}()
	`, nil, Int(3))

	TestExpectRun(t, `
	var out
	func() {
		a := 5
		out = func() {  	
			a := 4						
			return a
		}()
	}()
	return out`, nil, Int(4))

	expectErrHas(t, `a := 1; a := 2`, NewTestOpts().CompilerError(),
		`Compile Error: "a" redeclared in this block`) // redeclared in the same scope
	expectErrHas(t, `func() { a := 1; a := 2 }()`, NewTestOpts().CompilerError(),
		`Compile Error: "a" redeclared in this block`) // redeclared in the same scope

	TestExpectRun(t, `a := 1; a += 2; return a`, nil, Int(3))
	TestExpectRun(t, `a := 1; a += 4 - 2; return a`, nil, Int(3))
	TestExpectRun(t, `a := 3; a -= 1; return a`, nil, Int(2))
	TestExpectRun(t, `a := 3; a -= 5 - 4; return a`, nil, Int(2))
	TestExpectRun(t, `a := 2; a *= 4; return a`, nil, Int(8))
	TestExpectRun(t, `a := 2; a *= 1 + 3; return a`, nil, Int(8))
	TestExpectRun(t, `a := 10; a /= 2; return a`, nil, Int(5))
	TestExpectRun(t, `a := 10; a /= 5 - 3; return a`, nil, Int(5))

	// compound assignment operator does not define new variable
	expectErrHas(t, `a += 4`, NewTestOpts().CompilerError(), `Compile Error: unresolved reference "a"`)
	expectErrHas(t, `a -= 4`, NewTestOpts().CompilerError(), `Compile Error: unresolved reference "a"`)
	expectErrHas(t, `a *= 4`, NewTestOpts().CompilerError(), `Compile Error: unresolved reference "a"`)
	expectErrHas(t, `a /= 4`, NewTestOpts().CompilerError(), `Compile Error: unresolved reference "a"`)

	TestExpectRun(t, `
	f1 := func() {
		f2 := func() {
			a := 1
			a += 2
			return a
		};
		return f2();
	};
	return f1();`, nil, Int(3))
	TestExpectRun(t, `f1 := func() { f2 := func() { a := 1; a += 4 - 2; return a }; return f2(); }; return f1()`,
		nil, Int(3))
	TestExpectRun(t, `f1 := func() { f2 := func() { a := 3; a -= 1; return a }; return f2(); }; return f1()`,
		nil, Int(2))
	TestExpectRun(t, `f1 := func() { f2 := func() { a := 3; a -= 5 - 4; return a }; return f2(); }; return f1()`,
		nil, Int(2))
	TestExpectRun(t, `f1 := func() { f2 := func() { a := 2; a *= 4; return a }; return f2(); }; return f1()`,
		nil, Int(8))
	TestExpectRun(t, `f1 := func() { f2 := func() { a := 2; a *= 1 + 3; return a }; return f2(); }; return f1()`,
		nil, Int(8))
	TestExpectRun(t, `f1 := func() { f2 := func() { a := 10; a /= 2; return a }; return f2(); }; return f1()`,
		nil, Int(5))
	TestExpectRun(t, `f1 := func() { f2 := func() { a := 10; a /= 5 - 3; return a }; return f2(); }; return f1()`,
		nil, Int(5))
	TestExpectRun(t, `a := 1; f1 := func() { f2 := func() { a += 2; return a }; return f2(); }; return f1()`,
		nil, Int(3))
	TestExpectRun(t, `
	f1 := func(a) {
		return func(b) {
			c := a
			c += b * 2
			return c
		}
	}
	return f1(3)(4)
	`, nil, Int(11))

	TestExpectRun(t, `
	return func() {
		a := 1
		func() {
			a = 2
			func() {
				a = 3
				func() {
					a := 4 // declared new
				}()
			}()
		}()
		return a
	}()
	`, nil, Int(3))

	// write on free variables
	TestExpectRun(t, `
	f1 := func() {
		a := 5
		return func() {
			a += 3
			return a
		}()
	}
	return f1()
	`, nil, Int(8))

	TestExpectRun(t, `
	return func() {
		f1 := func() {
			a := 5
			add1 := func() { a += 1 }
			add2 := func() { a += 2 }
			a += 3
			return func() { a += 4; add1(); add2(); a += 5; return a }
		}
		return f1()
	}()()
	`, nil, Int(20))

	TestExpectRun(t, `
	it := func(seq, fn) {
		fn(seq[0])
		fn(seq[1])
		fn(seq[2])
	}

	foo := func(a) {
		b := 0
		it([1, 2, 3], func(x) {
			b = x + a
		})
		return b
	}
	return foo(2)
	`, nil, Int(5))

	TestExpectRun(t, `
	it := func(seq, fn) {
		fn(seq[0])
		fn(seq[1])
		fn(seq[2])
	}

	foo := func(a) {
		b := 0
		it([1, 2, 3], func(x) {
			b += x + a
		})
		return b
	}
	return foo(2)
	`, nil, Int(12))

	TestExpectRun(t, `
	return func() {
		a := 1
		func() {
			a = 2
		}()
		return a
	}()
	`, nil, Int(2))

	TestExpectRun(t, `
	f := func() {
		a := 1
		return {
			b: func() { a += 3 },
			c: func() { a += 2 },
			d: func() { return a },
		}
	}
	m := f()
	m.b()
	m.c()
	return m.d()
	`, nil, Int(6))

	TestExpectRun(t, `
	each := func(s, x) { for i:=0; i<len(s); i++ { x(s[i]) } }

	return func() {
		a := 100
		each([1, 2, 3], func(x) {
			a += x
		})
		a += 10
		return func(b) {
			return a + b
		}
	}()(20)
	`, nil, Int(136))

	// assigning different type value
	TestExpectRun(t, `a := 1; a = "foo"; return a`, nil, Str("foo"))
	TestExpectRun(t, `return func() { a := 1; a = "foo"; return a }()`, nil, Str("foo"))
	TestExpectRun(t, `
	return func() {
		a := 5
		return func() {
			a = "foo"
			return a
		}()
	}()`, nil, Str("foo")) // free

	// variables declared in if/for blocks
	TestExpectRun(t, `for a:=0; a<5; a++ {}; a := "foo"; return a`, nil, Str("foo"))
	TestExpectRun(t, `var out; func() { for a:=0; a<5; a++ {}; a := "foo"; out = a }(); return out`,
		nil, Str("foo"))
	TestExpectRun(t, `a:=0; if a:=1; a>0 { return a }; return 0`, nil, Int(1))
	TestExpectRun(t, `a:=1; if a:=0; a>0 { return a }; return a`, nil, Int(1))

	// selectors
	TestExpectRun(t, `a:=[1,2,3]; a[1] = 5; return a[1]`, nil, Int(5))
	TestExpectRun(t, `a:=[1,2,3]; a[1] += 5; return a[1]`, nil, Int(7))
	TestExpectRun(t, `a:={b:1,c:2}; a.b = 5; return a.b`, nil, Int(5))
	TestExpectRun(t, `a:={b:1,c:2}; a.b += 5; return a.b`, nil, Int(6))
	TestExpectRun(t, `a:={b:1,c:2}; a.b += a.c; return a.b`, nil, Int(3))
	TestExpectRun(t, `a:={b:1,c:2}; a.b += a.c; return a.c`, nil, Int(2))
	TestExpectRun(t, `
	a := {
		b: [1, 2, 3],
		c: {
			d: 8,
			e: "foo",
			f: [9, 8],
		},
	}
	a.c.f[1] += 2
	return a["c"]["f"][1]
	`, nil, Int(10))

	TestExpectRun(t, `
	a := {
		b: [1, 2, 3],
		c: {
			d: 8,
			e: "foo",
			f: [9, 8],
		},
	}
	a.c.h = "bar"
	return a.c.h
	`, nil, Str("bar"))

	expectErrIs(t, `
	a := {
		b: [1, 2, 3],
		c: {
			d: 8,
			e: "foo",
			f: [9, 8],
		},
	}
	a.x.e = "bar"`, nil, ErrNotIndexAssignable)

	// order of evaluation
	// left to right but in assignment RHS first then LHS
	TestExpectRun(t, `
	a := 1
	f := func() {
		a*=10
		return a
	}
	g := func() {
		a++
		return a
	}
	h := func() {
		a+=2
		return a
	}
	d := {}
	d[f()] = [g(), h()]
	return d
	`, nil, Dict{"40": Array{Int(2), Int(4)}})

	TestExpectRun(t, `a := nil; a ||= 1; return a`, nil, Int(1))
	TestExpectRun(t, `a := 0; a ||= 1; return a`, nil, Int(1))
	TestExpectRun(t, `a := ""; a ||= 1; return a`, nil, Int(1))
	TestExpectRun(t, `a := 1; a ||= 2; return a`, nil, Int(1))
	TestExpectRun(t, `c := false; a := 1; a ||= func(){c=true;return 2}(); return [c,a]`, nil, Array{False, Int(1)})
	TestExpectRun(t, `c := false; a := 0; a ||= func(){c=true;return 2}(); return [c,a]`, nil, Array{True, Int(2)})

	TestExpectRun(t, `a := 1; a ??= 2; return a`, nil, Int(1))
	TestExpectRun(t, `a := 0; a ??= 2; return a`, nil, Int(0))
	TestExpectRun(t, `a := nil; a ??= 2; return a`, nil, Int(2))
	TestExpectRun(t, `c := false; a := 1; a ??= func(){c=true;return 2}(); return [c,a]`, nil, Array{False, Int(1)})
	TestExpectRun(t, `c := false; a := nil; a ??= func(){c=true;return 2}(); return [c,a]`, nil, Array{True, Int(2)})
}

func TestVMBitwise(t *testing.T) {
	TestExpectRun(t, `return 1 & 1`, nil, Int(1))
	TestExpectRun(t, `return 1 & 0`, nil, Int(0))
	TestExpectRun(t, `return 0 & 1`, nil, Int(0))
	TestExpectRun(t, `return 0 & 0`, nil, Int(0))
	TestExpectRun(t, `return 1 | 1`, nil, Int(1))
	TestExpectRun(t, `return 1 | 0`, nil, Int(1))
	TestExpectRun(t, `return 0 | 1`, nil, Int(1))
	TestExpectRun(t, `return 0 | 0`, nil, Int(0))
	TestExpectRun(t, `return 1 ^ 1`, nil, Int(0))
	TestExpectRun(t, `return 1 ^ 0`, nil, Int(1))
	TestExpectRun(t, `return 0 ^ 1`, nil, Int(1))
	TestExpectRun(t, `return 0 ^ 0`, nil, Int(0))
	TestExpectRun(t, `return 1 &^ 1`, nil, Int(0))
	TestExpectRun(t, `return 1 &^ 0`, nil, Int(1))
	TestExpectRun(t, `return 0 &^ 1`, nil, Int(0))
	TestExpectRun(t, `return 0 &^ 0`, nil, Int(0))
	TestExpectRun(t, `return 1 << 2`, nil, Int(4))
	TestExpectRun(t, `return 16 >> 2`, nil, Int(4))

	TestExpectRun(t, `return 1u & 1u`, nil, Uint(1))
	TestExpectRun(t, `return 1u & 0u`, nil, Uint(0))
	TestExpectRun(t, `return 0u & 1u`, nil, Uint(0))
	TestExpectRun(t, `return 0u & 0u`, nil, Uint(0))
	TestExpectRun(t, `return 1u | 1u`, nil, Uint(1))
	TestExpectRun(t, `return 1u | 0u`, nil, Uint(1))
	TestExpectRun(t, `return 0u | 1u`, nil, Uint(1))
	TestExpectRun(t, `return 0u | 0u`, nil, Uint(0))
	TestExpectRun(t, `return 1u ^ 1u`, nil, Uint(0))
	TestExpectRun(t, `return 1u ^ 0u`, nil, Uint(1))
	TestExpectRun(t, `return 0u ^ 1u`, nil, Uint(1))
	TestExpectRun(t, `return 0u ^ 0u`, nil, Uint(0))
	TestExpectRun(t, `return 1u &^ 1u`, nil, Uint(0))
	TestExpectRun(t, `return 1u &^ 0u`, nil, Uint(1))
	TestExpectRun(t, `return 0u &^ 1u`, nil, Uint(0))
	TestExpectRun(t, `return 0u &^ 0u`, nil, Uint(0))
	TestExpectRun(t, `return 1u << 2u`, nil, Uint(4))
	TestExpectRun(t, `return 16u >> 2u`, nil, Uint(4))

	TestExpectRun(t, `out := 1; out &= 1; return out`, nil, Int(1))
	TestExpectRun(t, `out := 1; out |= 0; return out`, nil, Int(1))
	TestExpectRun(t, `out := 1; out ^= 0; return out`, nil, Int(1))
	TestExpectRun(t, `out := 1; out &^= 0; return out`, nil, Int(1))
	TestExpectRun(t, `out := 1; out <<= 2; return out`, nil, Int(4))
	TestExpectRun(t, `out := 16; out >>= 2; return out`, nil, Int(4))

	TestExpectRun(t, `out := 1u; out &= 1u; return out`, nil, Uint(1))
	TestExpectRun(t, `out := 1u; out |= 0u; return out`, nil, Uint(1))
	TestExpectRun(t, `out := 1u; out ^= 0u; return out`, nil, Uint(1))
	TestExpectRun(t, `out := 1u; out &^= 0u; return out`, nil, Uint(1))
	TestExpectRun(t, `out := 1u; out <<= 2u; return out`, nil, Uint(4))
	TestExpectRun(t, `out := 16u; out >>= 2u; return out`, nil, Uint(4))

	TestExpectRun(t, `out := ^0; return out`, nil, Int(^0))
	TestExpectRun(t, `out := ^1; return out`, nil, Int(^1))
	TestExpectRun(t, `out := ^55; return out`, nil, Int(^55))
	TestExpectRun(t, `out := ^-55; return out`, nil, Int(^-55))

	TestExpectRun(t, `out := ^0u; return out`, nil, Uint(^uint64(0)))
	TestExpectRun(t, `out := ^1u; return out`, nil, Uint(^uint64(1)))
	TestExpectRun(t, `out := ^55u; return out`, nil, Uint(^uint64(55)))
}

func TestVMBoolean(t *testing.T) {
	TestExpectRun(t, `return true`, nil, True)
	TestExpectRun(t, `return false`, nil, False)
	TestExpectRun(t, `return 1 < 2`, nil, True)
	TestExpectRun(t, `return 1 > 2`, nil, False)
	TestExpectRun(t, `return 1 < 1`, nil, False)
	TestExpectRun(t, `return 1 > 2`, nil, False)
	TestExpectRun(t, `return 1 == 1`, nil, True)
	TestExpectRun(t, `return 1 != 1`, nil, False)
	TestExpectRun(t, `return 1 == 2`, nil, False)
	TestExpectRun(t, `return 1 != 2`, nil, True)
	TestExpectRun(t, `return 1 <= 2`, nil, True)
	TestExpectRun(t, `return 1 >= 2`, nil, False)
	TestExpectRun(t, `return 1 <= 1`, nil, True)
	TestExpectRun(t, `return 1 >= 2`, nil, False)

	TestExpectRun(t, `return true == true`, nil, True)
	TestExpectRun(t, `return false == false`, nil, True)
	TestExpectRun(t, `return true == false`, nil, False)
	TestExpectRun(t, `return true != false`, nil, True)
	TestExpectRun(t, `return false != true`, nil, True)
	TestExpectRun(t, `return (1 < 2) == true`, nil, True)
	TestExpectRun(t, `return (1 < 2) == false`, nil, False)
	TestExpectRun(t, `return (1 > 2) == true`, nil, False)
	TestExpectRun(t, `return (1 > 2) == false`, nil, True)
	TestExpectRun(t, `return !true`, nil, False)
	TestExpectRun(t, `return !false`, nil, True)

	TestExpectRun(t, `return 5 + true`, nil, Int(6))
	TestExpectRun(t, `return 5 + false`, nil, Int(5))
	TestExpectRun(t, `return 5 * true`, nil, Int(5))
	TestExpectRun(t, `return 5 * false`, nil, Int(0))
	TestExpectRun(t, `return -true`, nil, Int(-1))
	TestExpectRun(t, `return true + false`, nil, Int(1))
	TestExpectRun(t, `return true*false`, nil, Int(0))
	TestExpectRun(t, `return func() { return true + false }()`, nil, Int(1))
	TestExpectRun(t, `if (true + false) { return 10 }`, nil, Int(10))
	TestExpectRun(t, `return 10 + (true + false)`, nil, Int(11))
	TestExpectRun(t, `return (true + false) + 20`, nil, Int(21))
	TestExpectRun(t, `return !(true + false)`, nil, False)
	TestExpectRun(t, `return !(true - false)`, nil, False)
	expectErrIs(t, `return true/false`, nil, ErrZeroDivision)
	expectErrIs(t, `return 1/false`, nil, ErrZeroDivision)
}

func TestVMNil(t *testing.T) {
	TestExpectRun(t, `return nil ? 1 : 2`, nil, Int(2))
	TestExpectRun(t, `return nil == nil`, nil, True)
	TestExpectRun(t, `return nil == (nil ? 1 : nil)`,
		nil, True)
	TestExpectRun(t, `return copy(nil)`, nil, Nil)
	TestExpectRun(t, `return len(nil)`, nil, Int(0))

	testCases := []string{
		"true", "false", "0", "1", "1u", `""`, `"a"`, `bytes(0)`, "[]", "{}",
		"[1]", "{a:1}", `'a'`, "1.1", "0.0",
	}
	for _, tC := range testCases {
		t.Run(tC, func(t *testing.T) {
			TestExpectRun(t, fmt.Sprintf(`return nil == %s`, tC), nil, False)
			TestExpectRun(t, fmt.Sprintf(`return nil != %s`, tC), nil, True)
			TestExpectRun(t, fmt.Sprintf(`return nil < %s`, tC), nil, True)
			TestExpectRun(t, fmt.Sprintf(`return nil <= %s`, tC), nil, True)
			TestExpectRun(t, fmt.Sprintf(`return nil > %s`, tC), nil, False)
			TestExpectRun(t, fmt.Sprintf(`return nil >= %s`, tC), nil, False)

			TestExpectRun(t, fmt.Sprintf(`return %s == nil`, tC), nil, False)
			TestExpectRun(t, fmt.Sprintf(`return %s != nil`, tC), nil, True)
			TestExpectRun(t, fmt.Sprintf(`return %s > nil`, tC), nil, True)
			TestExpectRun(t, fmt.Sprintf(`return %s >= nil`, tC), nil, True)
			TestExpectRun(t, fmt.Sprintf(`return %s < nil`, tC), nil, False)
			TestExpectRun(t, fmt.Sprintf(`return %s <= nil`, tC), nil, False)
		})
	}
}

func TestVMKeyValue(t *testing.T) {
	TestExpectRun(t, `return [a=no]`, nil, &KeyValue{Str("a"), No})
	TestExpectRun(t, `return [a=yes]`, nil, &KeyValue{Str("a"), Yes})
	TestExpectRun(t, `return [a=1]`, nil, &KeyValue{Str("a"), Int(1)})
}

func TestVMKeyValueArray(t *testing.T) {
	TestExpectRun(t, `return (;flag)`, nil, KeyValueArray{&KeyValue{Str("flag"), Yes}})
	TestExpectRun(t, `return (;flag=yes)`, nil, KeyValueArray{&KeyValue{Str("flag"), Yes}})
	TestExpectRun(t, `return (;flag=no)`, nil, KeyValueArray{})
	TestExpectRun(t, `return str((;flag))`, nil, Str("(;flag)"))
	TestExpectRun(t, `return (;disabled).flag("disabled")`, nil, True)
	TestExpectRun(t, `return (;x=1).flag("x")`, nil, True)
	TestExpectRun(t, `return (;x=nil).flag("x")`, nil, False)
	TestExpectRun(t, `return (;x=1,x=2,y=3).values("x")`, nil, Array{Int(1), Int(2)})
	TestExpectRun(t, `return (;x=1,x=2,y=3).values()`, nil, Array{Int(1), Int(2), Int(3)})
	TestExpectRun(t, `return (;x=1,x=2,y=3,z=4).values("x", "y")`, nil, Array{Int(1), Int(2), Int(3)})
	TestExpectRun(t, `return str((;a=1,a=2,b=3).delete())`, nil, Str("(;a=1, a=2, b=3)"))
	TestExpectRun(t, `return str((;a=1,a=2,b=3).delete("b"))`, nil, Str("(;a=1, a=2)"))
	TestExpectRun(t, `return str((;a=1,a=2,b=3,c=4).delete("a","b"))`, nil, Str("(;c=4)"))
	TestExpectRun(t, `return (;a=1)[0]`, nil, &KeyValue{Str("a"), Int(1)})
	TestExpectRun(t, `return (;a=1)[0].k`, nil, Str("a"))
	TestExpectRun(t, `return (;a=1)[0].v`, nil, Int(1))
	TestExpectRun(t, `return (;a=1)[0].array`, nil, Array{Str("a"), Int(1)})
	TestExpectRun(t, `x := (;a); x[0].v = 2; return dict(x)`, nil, Dict{"a": Int(2)})
	TestExpectRun(t, `return (;**(;x=1,"x"=2))`, nil, KeyValueArray{
		&KeyValue{Str("x"), Int(1)},
		&KeyValue{Str("x"), Int(2)},
	})
	TestExpectRun(t, `return (;y=3,**(;x=1,"x"=2))`, nil, KeyValueArray{
		&KeyValue{Str("y"), Int(3)},
		&KeyValue{Str("x"), Int(1)},
		&KeyValue{Str("x"), Int(2)},
	})
}

func TestVMRegexp(t *testing.T) {
	re := `"a([bc])"`
	TestExpectRun(t, `return repr(regexp(`+re+`))`, nil, Str(ReprQuote("regexp:a([bc])")))
	TestExpectRun(t, `re := regexp(`+re+`); return [
	re ~ "ab", 
	re ~ `+"`ab`"+`, 
	re ~ "a",
	repr(re ~~ "ab"),
	repr(re ~~~ "ab"),
	repr(re ~~~ "ab\nac"),
	re.match("ab"),
	repr(re.find("ab")),
	repr(re.findAll("ab")),
	repr(re.findAll("ab\nac")),
	repr(re ~~ bytes("ab")),
	repr(re ~~~ bytes("ab")),
	repr(re ~~~ bytes("ab\nac")),
]`, nil, Array{
		True,
		True,
		False,
		Str(ReprQuote(`regexpStrsResult:["ab", "b"]`)),
		Str(ReprQuote(`regexpStrsSliceResult:[["ab", "b"]]`)),
		Str(ReprQuote(`regexpStrsSliceResult:[["ab", "b"], ["ac", "c"]]`)),
		True,
		Str(ReprQuote(`regexpStrsResult:["ab", "b"]`)),
		Str(ReprQuote(`regexpStrsSliceResult:[["ab", "b"]]`)),
		Str(ReprQuote(`regexpStrsSliceResult:[["ab", "b"], ["ac", "c"]]`)),
		Str(ReprQuote(`regexpBytesResult:[[97 98], [98]]`)),
		Str(ReprQuote(`regexpBytesSliceResult:[[[97 98], [98]]]`)),
		Str(ReprQuote(`regexpBytesSliceResult:[[[97 98], [98]], [[97 99], [99]]]`)),
	})
}

func TestVMIterator(t *testing.T) {
	rg := `Range := struct("Range", fields={
				Start:0,
				End:2,
			})`
	rgc := rg + `
		func iterator(r Range) => [r.Start, [(r.Start)=str('a' + r.Start)]]
		func iterator(r Range, state) => state >= r.End ? nil : [state+1, [(state+1)=str('a' + state+1)]]
	`

	TestExpectRun(t, rgc+`
		ret := []
		for k, v in Range() {
			ret = append(ret, [k, v])
		}
		return str(ret)
	`, nil, Str(`[[0, "a"], [1, "b"], [2, "c"]]`))

	TestExpectRun(t, rg+`
		func iterator(r Range) => [r.Start, [(r.Start)=str('a' + r.Start)]]
		func iterator(r Range, state) => state >= r.End ? nil : [state+1, [(state+1)=str('a' + state+1)]]

		ret := []
		for k, v in Range() {
			ret = append(ret, [k, v])
		}

		return str(ret)
	`, nil, Str(`[[0, "a"], [1, "b"], [2, "c"]]`))

	TestExpectRun(t, rg+`
			func iterator(r Range) => [r.Start, str('a' + r.Start)]
			func iterator(r Range, state) => state >= r.End ? nil : [state+1, str('a' + state+1)]

			return str(collect(values(Range())))
		`, nil, Str(`["a", "b", "c"]`))
	TestExpectRun(t, rg+`
			func iterator(r Range) => [r.Start, str('a' + r.Start)]
			func iterator(r Range, state) => state >= r.End ? nil : [state+1, str('a' + state+1)]

			return str([
				iterator(Range()),
				iterator(Range(), 0),
				iterator(Range(), 1),
				iterator(Range(), 2),
				iterator(Range(), 3),
		])
		`, nil, Str(`[[0, "a"], [1, "b"], [2, "c"], nil, nil]`))

	TestExpectRun(t, rg+`
			ret := [nil, nil]
			ret[0] = isIterable(Range())

			func iterator(r Range) => [r.Start, 'a' + r.Start, r.End > r.Start]
			func iterator(r Range, state) => [state+1, 'a' + state+1, r.End > state]

			ret[1] = isIterable(Range())

			return ret
		`, nil, Array{False, True})
	TestExpectRun(t, `return isIterable({})`, nil, True)
	TestExpectRun(t, `return isIterable([])`, nil, True)
	TestExpectRun(t, `return isIterable((;))`, nil, True)
	TestExpectRun(t, `return isIterable("a")`, nil, True)
	TestExpectRun(t, `return isIterable(bytes("a"))`, nil, True)
	TestExpectRun(t, `return isIterable(1)`, nil, False)
	TestExpectRun(t, `return isIterable(false)`, nil, False)
	TestExpectRun(t, `return isIterable(nil)`, nil, False)
	TestExpectRun(t, `return isIterable(1.2)`, nil, False)
	TestExpectRun(t, `return isIterable(1.2d)`, nil, False)

	TestExpectRun(t, `return isIterator(values({}))`, nil, True)
	TestExpectRun(t, `return isIterator(values([]))`, nil, True)
	TestExpectRun(t, `return isIterator(values((;)))`, nil, True)
	TestExpectRun(t, `return isIterator(values("a"))`, nil, True)
	TestExpectRun(t, `return isIterator(values(bytes("a")))`, nil, True)
	TestExpectRun(t, `return isIterator(1)`, nil, False)
	TestExpectRun(t, `return isIterator({})`, nil, False)
	TestExpectRun(t, `return isIterator([])`, nil, False)
	TestExpectRun(t, `return isIterator((;))`, nil, False)
	TestExpectRun(t, `return isIterator("a")`, nil, False)
	TestExpectRun(t, `return isIterator(bytes("a"))`, nil, False)
	TestExpectRun(t, `return isIterator(1)`, nil, False)
	TestExpectRun(t, `return isIterator(false)`, nil, False)
	TestExpectRun(t, `return isIterator(nil)`, nil, False)
	TestExpectRun(t, `return isIterator(1.2)`, nil, False)
	TestExpectRun(t, `return isIterator(1.2d)`, nil, False)

	TestExpectRun(t, `return repr(values({a:1, b:2}))`, nil,
		Str(`‹ValuesIterator:‹DictIterator:{a: 1, b: 2}››`))
	TestExpectRun(t, `return repr(values({a:1, b:2};sorted))`, nil,
		Str(`‹ValuesIterator:‹DictIterator:{a: 1, b: 2}››`))
	TestExpectRun(t, `return str(collect(values({a:1, b:2};sorted)))`, nil, Str("[1, 2]"))
	TestExpectRun(t, `return repr(keys({a:1, b:2};sorted))`, nil,
		Str(`‹KeysIterator:‹DictIterator:{a: 1, b: 2}››`))
	TestExpectRun(t, `return str(collect(keys({a:1, b:2};sorted)))`, nil,
		Str(`["a", "b"]`))
	TestExpectRun(t, `return repr(items({a:1, b:2};sorted))`, nil,
		Str(`‹ItemsIterator:‹DictIterator:{a: 1, b: 2}››`))
	TestExpectRun(t, `return str(collect(items({a:1, b:2};sorted)))`, nil, Str("[a=1, b=2]"))
	TestExpectRun(t, `return str(collect(items({a:1, b:2, c:3, d:4, e:5, f:6, g:7};step=3,sorted)))`, nil,
		Str("[a=1, d=4, g=7]"))

	TestExpectRun(t, `return repr(values([1,2]))`, nil, Str("‹ValuesIterator:‹ArrayIterator:[1, 2]››"))
	TestExpectRun(t, `return str(collect(values([1,2])))`, nil, Str("[1, 2]"))
	TestExpectRun(t, `return repr(keys([1,2]))`, nil, Str("‹KeysIterator:‹ArrayIterator:[1, 2]››"))
	TestExpectRun(t, `return str(collect(keys([2,5])))`, nil, Str("[0, 1]"))
	TestExpectRun(t, `return repr(items([2,5]))`, nil, Str(`‹ItemsIterator:‹ArrayIterator:[2, 5]››`))
	TestExpectRun(t, `return str(collect(items([2,5])))`, nil, Str("[0=2, 1=5]"))
	TestExpectRun(t, `return str(collect(values([1,2,3];reversed)))`, nil, Str("[3, 2, 1]"))
	TestExpectRun(t, `return str(collect(values([1,2,3];reversed)))`, nil, Str("[3, 2, 1]"))
	TestExpectRun(t, `return str(collect(values([1,2,3,4,5,6,7];step=2)))`, nil, Str("[1, 3, 5, 7]"))
	TestExpectRun(t, `return str(collect(values([1,2,3,4,5,6,7];step=2,reversed)))`, nil, Str("[7, 5, 3, 1]"))
	TestExpectRun(t, `return str(collect(values([1,2,3,4,5,6,7];step=3)))`, nil, Str("[1, 4, 7]"))
	TestExpectRun(t, `return str(collect(values([1,2,3,4,5,6,7];step=3,reversed)))`, nil, Str("[7, 4, 1]"))

	TestExpectRun(t, `return repr(values((;a=1,b=2)))`, nil,
		Str(`‹ValuesIterator:‹KeyValueArrayIterator:(;a=1, b=2)››`))
	TestExpectRun(t, `return str(collect(values((;a=1,b=2))))`, nil, Str("[1, 2]"))
	TestExpectRun(t, `return str(collect(keys((;a=1,b=2))))`, nil, Str(`["a", "b"]`))
	TestExpectRun(t, `return str(collect(items((;a=1,b=2))))`, nil, Str(`[a=1, b=2]`))

	TestExpectRun(t, `return repr(map([1,2], (k, v) => v))`, nil,
		Str(`‹MapIterator:‹‹ArrayIterator:[1, 2]› → ‹compiledFunction #2(k, v)›››`))

	TestExpectRun(t, `return str(collect(map(values([1,2]), (v, k) => v+10)))`, nil, Str("[11, 12]"))
	TestExpectRun(t, `return str(collect(values(filter([1,2,3,4,5], (v, k, _) => v%2))))`, nil, Str("[1, 3, 5]"))
	TestExpectRun(t, `return [1,2] .| map((v, k) => v+10) .| repr`, nil,
		Str(`‹MapIterator:‹‹ArrayIterator:[1, 2]› → ‹compiledFunction #3(v, k)›››`))
	TestExpectRun(t, `return [1,2] .| map((v, k) => v+10) .| values .| map((v, k) => v+10) .| repr`, nil,
		Str(`‹MapIterator:‹‹ValuesIterator:‹MapIterator:‹‹ArrayIterator:[1, 2]› → ‹compiledFunction #3(v, k)›››› → `+
			`‹compiledFunction #4(v, k)›››`))
	TestExpectRun(t, `return reduce([1,2,3], ((cur, v, k) => cur + v), 10)`, nil, Int(16))
	TestExpectRun(t, `return reduce([1,2], (cur, v, k) => cur + v)`, nil, Int(4))
	TestExpectRun(t, `return str(reduce([1,2,3], ((cur, v, k) => {cur.tot += v; cur[str(k+'a')] ??= v; cur}), {tot:100}))`,
		nil, Str("{a: 1, b: 2, c: 3, tot: 106}"))

	TestExpectRun(t, `a := []; it := iterator({a:"A",b:"B"};reversed); it.next; for k, v in it {a += [(k)=v]}; return str(a)`,
		nil, Str(`[a="A"]`))
	TestExpectRun(t, `a := []; it := iterator({a:"A",b:"B"};sorted); it.next; for k, v in it {a += [(k)=v]}; return str(a)`,
		nil, Str(`[b="B"]`))
	TestExpectRun(t, `a := []; it := iterator({a:"A",b:"B"};sorted); it.next; for {v := it.next; if v {a += v;} else {break;} }; return str(a)`,
		nil, Str(`["B"]`))
	TestExpectRun(t, `a := []; it := items(iterator({a:"A",b:"B"};sorted)); it.next; for {v := it.next; if v {a += v;} else {break;} }; return str(a)`,
		nil, Str(`[b="B"]`))
	TestExpectRun(t, `a := []; it := iterator({a:"A",b:"B"};sorted); for {v := it.next; if v {a += v;} else {break;} }; return str(a)`,
		nil, Str(`["A", "B"]`))
	TestExpectRun(t, `a := []; for k, v in iterator({a:"A",b:"B"};reversed) {a += [(k)=v]}; return str(a)`,
		nil, Str(`[b="B", a="A"]`))
	TestExpectRun(t, `a := []; for k, v in iterator({a:"A",b:"B"};sorted) {a += [(k)=v]}; return str(a)`,
		nil, Str(`[a="A", b="B"]`))
	TestExpectRun(t, `a := []; for k, v in (;a="A",b="B") {a += [(k)=v]}; return str(a)`,
		nil, Str(`[a="A", b="B"]`))
	TestExpectRun(t, `return str(collect(items(enumerate(iterator({a:"A",b:"B"};sorted)))))`,
		nil, Str(`[0=[a="A"], 1=[b="B"]]`))
	TestExpectRun(t, `return str(collect(items(enumerate({a:"A",b:"B"};sorted))))`,
		nil, Str(`[0=[a="A"], 1=[b="B"]]`))
	TestExpectRun(t, `return str(collect(zip([1,2,3],[4,5,6])))`,
		nil, Str(`[1, 2, 3, 4, 5, 6]`))
	TestExpectRun(t, `return str(collect(items(enumerate(zip([1,2,3],[4,5,6])))))`,
		nil, Str(`[0=[0=1], 1=[1=2], 2=[2=3], 3=[0=4], 4=[1=5], 5=[2=6]]`))
	TestExpectRun(t, `return str(collect(enumerate(items(zip([1,2,3],[4,5,6]));values)))`,
		nil, Str(`[0=1, 1=2, 2=3, 3=4, 4=5, 5=6]`))
	TestExpectRun(t, `return str(collect(enumerate(zip([1,2,3],[4,5,6]);keys)))`,
		nil, Str(`[0, 1, 2, 0, 1, 2]`))
}

func TestVMBuiltinFunction(t *testing.T) {
	TestExpectRun(t, `return append(nil)`,
		nil, Array{})
	TestExpectRun(t, `return append(nil, 1)`,
		nil, Array{Int(1)})
	TestExpectRun(t, `return append([], 1)`,
		nil, Array{Int(1)})
	TestExpectRun(t, `return append([], 1, 2)`,
		nil, Array{Int(1), Int(2)})
	TestExpectRun(t, `return append([0], 1, 2)`,
		nil, Array{Int(0), Int(1), Int(2)})
	TestExpectRun(t, `return append(bytes())`,
		nil, Bytes{})
	TestExpectRun(t, `return append(bytes(), 1, 2)`,
		nil, Bytes{1, 2})
	expectErrIs(t, `append()`, nil, ErrWrongNumArguments)
	expectErrIs(t, `append({})`, nil, ErrType)
	TestExpectRun(t, `return (;)`, nil, KeyValueArray{})
	TestExpectRun(t, `return append((;))`, nil, KeyValueArray{})
	TestExpectRun(t, `return append((;),(;a=1))`, nil, KeyValueArray{&KeyValue{Str("a"), Int(1)}})
	TestExpectRun(t, `return append((;a=1),(;b=2),{c:3},[d=4])`, nil, KeyValueArray{
		&KeyValue{Str("a"), Int(1)}, &KeyValue{Str("b"), Int(2)}, &KeyValue{Str("c"), Int(3)},
		&KeyValue{Str("d"), Int(4)}})

	TestExpectRun(t, `out := {}; delete(out, "a"); return out`,
		nil, Dict{})
	TestExpectRun(t, `out := {a: 1}; delete(out, "a"); return out`,
		nil, Dict{})
	TestExpectRun(t, `out := {a: 1}; delete(out, "b"); return out`,
		nil, Dict{"a": Int(1)})
	expectErrIs(t, `delete({})`, nil, ErrWrongNumArguments)
	expectErrIs(t, `delete({}, "", "")`, nil, ErrWrongNumArguments)
	expectErrIs(t, `delete([], "")`, nil, ErrType)
	TestExpectRun(t, `delete({}, 1)`, nil, Nil)

	g := &SyncDict{Value: Dict{"out": &SyncDict{Value: Dict{"a": Int(1)}}}}
	TestExpectRun(t, `global out; delete(out, "a"); return out`,
		NewTestOpts().Globals(g).Skip2Pass(), &SyncDict{Value: Dict{}})

	TestExpectRun(t, `return copy(nil)`, nil, Nil)
	TestExpectRun(t, `return copy(1)`, nil, Int(1))
	TestExpectRun(t, `return copy(1u)`, nil, Uint(1))
	TestExpectRun(t, `return copy('a')`, nil, Char('a'))
	TestExpectRun(t, `return copy(1.0)`, nil, Float(1.0))
	// expectRun(t, `return copy(1d)`, nil, DecimalFromUint(1))
	TestExpectRun(t, `return copy(1.0d)`, nil, MustDecimalFromString("1.0"))
	TestExpectRun(t, `return copy("x")`, nil, Str("x"))
	TestExpectRun(t, `return copy(true)`, nil, True)
	TestExpectRun(t, `return copy(false)`, nil, False)
	TestExpectRun(t, `a := {x: 1}; b := copy(a); a.x = 2; return b`,
		nil, Dict{"x": Int(1)})
	TestExpectRun(t, `a := {x: 1}; b := copy(a); b.x = 2; return a`,
		nil, Dict{"x": Int(1)})
	TestExpectRun(t, `a := {x: 1}; b := copy(a); return a == b`,
		nil, True)
	TestExpectRun(t, `a := [1]; b := copy(a); a[0] = 2; return b`,
		nil, Array{Int(1)})
	TestExpectRun(t, `a := [1]; b := copy(a); b[0] = 2; return a`,
		nil, Array{Int(1)})
	TestExpectRun(t, `a := [1]; b := copy(a); return a == b`,
		nil, True)
	TestExpectRun(t, `a := bytes(1); b := copy(a); a[0] = 2; return b`,
		nil, Bytes{1})
	TestExpectRun(t, `a := bytes(1); b := copy(a); b[0] = 2; return a`,
		nil, Bytes{1})
	TestExpectRun(t, `a := bytes(1); b := copy(a); return a == b`,
		nil, True)
	TestExpectRun(t, `a := [1,{c:2}]; b := copy(a);
			b[0] = 2
			b[1].c = 3
			return a == b, a[0], b[0], a[1] == b[1], a[1].c == b[1].c, b[1].c`,
		nil, Array{False, Int(1), Int(2), True, True, Int(3)})
	expectErrIs(t, `copy()`, nil, ErrWrongNumArguments)
	expectErrIs(t, `copy(1, 2, 3)`, nil, ErrWrongNumArguments)

	TestExpectRun(t, `return dcopy(nil)`, nil, Nil)
	TestExpectRun(t, `return dcopy(1)`, nil, Int(1))
	TestExpectRun(t, `return dcopy(1u)`, nil, Uint(1))
	TestExpectRun(t, `return dcopy('a')`, nil, Char('a'))
	TestExpectRun(t, `return dcopy(1.0)`, nil, Float(1.0))
	TestExpectRun(t, `return dcopy(1.0d)`, nil, MustDecimalFromString("1.0"))
	TestExpectRun(t, `return dcopy("x")`, nil, Str("x"))
	TestExpectRun(t, `return dcopy(true)`, nil, True)
	TestExpectRun(t, `return dcopy(false)`, nil, False)
	TestExpectRun(t, `a := {x: 1}; b := dcopy(a); a.x = 2; return b`,
		nil, Dict{"x": Int(1)})
	TestExpectRun(t, `a := {x: 1}; b := dcopy(a); b.x = 2; return a`,
		nil, Dict{"x": Int(1)})
	TestExpectRun(t, `a := {x: 1}; b := dcopy(a); return a == b`,
		nil, True)
	TestExpectRun(t, `a := [1]; b := dcopy(a); a[0] = 2; return b`,
		nil, Array{Int(1)})
	TestExpectRun(t, `a := [1]; b := dcopy(a); b[0] = 2; return a`,
		nil, Array{Int(1)})
	TestExpectRun(t, `a := [1]; b := dcopy(a); return a == b`,
		nil, True)
	TestExpectRun(t, `a := bytes(1); b := dcopy(a); a[0] = 2; return b`,
		nil, Bytes{1})
	TestExpectRun(t, `a := bytes(1); b := dcopy(a); b[0] = 2; return a`,
		nil, Bytes{1})
	TestExpectRun(t, `a := bytes(1); b := dcopy(a); return a == b`,
		nil, True)
	TestExpectRun(t, `a := [1,{c:2}]; b := dcopy(a);
			b[0] = 2
			a[1].c = 3
			b[1].c = 4
			return a == b, a[0], b[0], a[1] == b[1], a[1].c == b[1].c, a[1].c, b[1].c`,
		nil, Array{False, Int(1), Int(2), False, False, Int(3), Int(4)})
	expectErrIs(t, `dcopy()`, nil, ErrWrongNumArguments)
	expectErrIs(t, `dcopy(1, 2)`, nil, ErrWrongNumArguments)

	TestExpectRun(t, `return repeat("abc", 3)`, nil, Str("abcabcabc"))
	TestExpectRun(t, `return repeat("abc", 2)`, nil, Str("abcabc"))
	TestExpectRun(t, `return repeat("abc", 1)`, nil, Str("abc"))
	TestExpectRun(t, `return repeat("abc", 0)`, nil, Str(""))
	TestExpectRun(t, `return repeat(bytes(1, 2, 3), 3)`,
		nil, Bytes{1, 2, 3, 1, 2, 3, 1, 2, 3})
	TestExpectRun(t, `return repeat(bytes(1, 2, 3), 2)`,
		nil, Bytes{1, 2, 3, 1, 2, 3})
	TestExpectRun(t, `return repeat(bytes(1, 2, 3), 1)`,
		nil, Bytes{1, 2, 3})
	TestExpectRun(t, `return repeat(bytes(1, 2, 3), 0)`,
		nil, Bytes{})
	TestExpectRun(t, `return repeat([1, 2], 2)`,
		nil, Array{Int(1), Int(2), Int(1), Int(2)})
	TestExpectRun(t, `return repeat([1, 2], 1)`,
		nil, Array{Int(1), Int(2)})
	TestExpectRun(t, `return repeat([1, 2], 0)`,
		nil, Array{})
	TestExpectRun(t, `return repeat([true], 1)`, nil, Array{True})
	TestExpectRun(t, `return repeat([true], 2)`, nil, Array{True, True})
	TestExpectRun(t, `return repeat("", 3)`, nil, Str(""))
	TestExpectRun(t, `return repeat(bytes(), 3)`, nil, Bytes{})
	TestExpectRun(t, `return repeat([], 2)`, nil, Array{})
	expectErrIs(t, `return repeat("abc", "")`, nil, ErrType)
	expectErrIs(t, `return repeat("abc", -1)`, nil, ErrType)
	expectErrIs(t, `return repeat(bytes(1), -1)`, nil, ErrType)
	expectErrIs(t, `return repeat([1], -1)`, nil, ErrType)
	expectErrIs(t, `return repeat(bytes(1), [])`, nil, ErrType)
	expectErrIs(t, `return repeat([1], {})`, nil, ErrType)
	expectErrIs(t, `return repeat(nil, 1)`, nil, ErrType)
	expectErrIs(t, `return repeat(true, 1)`, nil, ErrType)
	expectErrIs(t, `return repeat(false, 1)`, nil, ErrType)
	expectErrIs(t, `return repeat(1, 1)`, nil, ErrType)
	expectErrIs(t, `return repeat(1u, 1)`, nil, ErrType)
	expectErrIs(t, `return repeat(1.1, 1)`, nil, ErrType)
	expectErrIs(t, `return repeat('a', 1)`, nil, ErrType)
	expectErrIs(t, `return repeat({}, 1)`, nil, ErrType)

	TestExpectRun(t, `return contains("xyz", "y")`, nil, True)
	TestExpectRun(t, `return contains("xyz", "a")`, nil, False)
	TestExpectRun(t, `return contains({a: 1}, "a")`, nil, True)
	TestExpectRun(t, `return contains({a: 1}, "b")`, nil, False)
	TestExpectRun(t, `return contains([1, 2, 3], 2)`, nil, True)
	TestExpectRun(t, `return contains([1, 2, 3], 4)`, nil, False)
	TestExpectRun(t, `return contains(bytes(1, 2, 3), 3)`, nil, True)
	TestExpectRun(t, `return contains(bytes(1, 2, 3), 4)`, nil, False)
	TestExpectRun(t, `return contains(bytes("abc"), "b")`, nil, True)
	TestExpectRun(t, `return contains(bytes("abc"), "d")`, nil, False)
	TestExpectRun(t, `return contains(bytes(1, 2, 3, 4), bytes(2, 3))`, nil, True)
	TestExpectRun(t, `return contains(bytes(1, 2, 3, 4), bytes(1, 3))`, nil, False)
	TestExpectRun(t, `return contains(nil, "")`, nil, False)
	TestExpectRun(t, `return contains(nil, 1)`, nil, False)
	g = &SyncDict{Value: Dict{"a": Int(1)}}
	TestExpectRun(t, `return contains(globals(), "a")`,
		NewTestOpts().Globals(g).Skip2Pass(), True)
	expectErrIs(t, `contains()`, nil, ErrWrongNumArguments)
	expectErrIs(t, `contains("", "", "")`, nil, ErrWrongNumArguments)
	expectErrIs(t, `contains(1, 2)`, nil, ErrType)

	TestExpectRun(t, `return len(nil)`, nil, Int(0))
	TestExpectRun(t, `return len(1)`, nil, Int(0))
	TestExpectRun(t, `return len(1u)`, nil, Int(0))
	TestExpectRun(t, `return len(true)`, nil, Int(0))
	TestExpectRun(t, `return len(1.1)`, nil, Int(0))
	TestExpectRun(t, `return len("")`, nil, Int(0))
	TestExpectRun(t, `return len([])`, nil, Int(0))
	TestExpectRun(t, `return len({})`, nil, Int(0))
	TestExpectRun(t, `return len(bytes())`, nil, Int(0))
	TestExpectRun(t, `return len("xyzw")`, nil, Int(4))
	TestExpectRun(t, `return len("çığöşü")`, nil, Int(12))
	TestExpectRun(t, `return len(chars("çığöşü"))`, nil, Int(6))
	TestExpectRun(t, `return len(["a"])`, nil, Int(1))
	TestExpectRun(t, `return len({a: 2})`, nil, Int(1))
	TestExpectRun(t, `return len(bytes(0, 1, 2))`, nil, Int(3))
	g = &SyncDict{Value: Dict{"a": Int(5)}}
	TestExpectRun(t, `return len(globals())`,
		NewTestOpts().Globals(g).Skip2Pass(), Int(1))
	expectErrIs(t, `len()`, nil, ErrWrongNumArguments)
	expectErrIs(t, `len([], [])`, nil, ErrWrongNumArguments)

	TestExpectRun(t, `return cap(nil)`, nil, Int(0))
	TestExpectRun(t, `return cap(1)`, nil, Int(0))
	TestExpectRun(t, `return cap(1u)`, nil, Int(0))
	TestExpectRun(t, `return cap(true)`, nil, Int(0))
	TestExpectRun(t, `return cap(1.1)`, nil, Int(0))
	TestExpectRun(t, `return cap("")`, nil, Int(0))
	TestExpectRun(t, `return cap([])`, nil, Int(0))
	TestExpectRun(t, `return cap({})`, nil, Int(0))
	TestExpectRun(t, `return cap(bytes())`, nil, Int(0))
	TestExpectRun(t, `return cap(bytes("a"))>=1`, nil, True)
	TestExpectRun(t, `return cap(bytes("abc"))>=3`, nil, True)
	TestExpectRun(t, `return cap(bytes("abc")[:3])>=3`, nil, True)
	TestExpectRun(t, `return cap([1])>0`, nil, True)
	TestExpectRun(t, `return cap([1,2,3])>=3`, nil, True)
	TestExpectRun(t, `return cap([1,2,3][:3])>=3`, nil, True)

	TestExpectRun(t, `return sort(nil)`,
		nil, Nil)
	TestExpectRun(t, `return sort("acb")`,
		nil, Str("abc"))
	TestExpectRun(t, `return sort(bytes("acb"))`,
		nil, Bytes(Str("abc")))
	TestExpectRun(t, `return sort([3, 2, 1])`,
		nil, Array{Int(1), Int(2), Int(3)})
	TestExpectRun(t, `return sort([3u, 2.0, 1])`,
		nil, Array{Int(1), Float(2), Uint(3)})
	TestExpectRun(t, `a := [3, 2, 1]; sort(a); return a`,
		nil, Array{Int(1), Int(2), Int(3)})
	expectErrIs(t, `sort()`, nil, ErrWrongNumArguments)
	expectErrIs(t, `sort([], [])`, nil, ErrWrongNumArguments)
	expectErrIs(t, `sort({})`, nil, ErrType)

	TestExpectRun(t, `return sortReverse(nil)`,
		nil, Nil)
	TestExpectRun(t, `return sortReverse("acb")`,
		nil, Str("cba"))
	TestExpectRun(t, `return sortReverse(bytes("acb"))`,
		nil, Bytes(Str("cba")))
	TestExpectRun(t, `return sortReverse([1, 2, 3])`,
		nil, Array{Int(3), Int(2), Int(1)})
	TestExpectRun(t, `a := [1, 2, 3]; sortReverse(a); return a`,
		nil, Array{Int(3), Int(2), Int(1)})
	expectErrIs(t, `sortReverse()`, nil, ErrWrongNumArguments)
	expectErrIs(t, `sortReverse([], [])`, nil, ErrWrongNumArguments)
	expectErrIs(t, `sortReverse({})`, nil, ErrType)

	TestExpectRun(t, `return error("x")`, nil,
		&Error{Name: "error", Message: "x"})
	TestExpectRun(t, `return error(1)`, nil,
		&Error{Name: "error", Message: "1"})
	TestExpectRun(t, `return error(nil)`, nil,
		&Error{Name: "error", Message: "nil"})
	expectErrIs(t, `error()`, nil, ErrWrongNumArguments)
	expectErrIs(t, `error(1,2,3)`, nil, ErrWrongNumArguments)

	TestExpectRun(t, `return typeName(true)`, nil, Str("bool"))
	TestExpectRun(t, `return typeName(nil)`, nil, Str("nil"))
	TestExpectRun(t, `return typeName(1)`, nil, Str("int"))
	TestExpectRun(t, `return typeName(1u)`, nil, Str("uint"))
	TestExpectRun(t, `return typeName(1.1)`, nil, Str("float"))
	TestExpectRun(t, `return typeName('a')`, nil, Str("char"))
	TestExpectRun(t, `return typeName("")`, nil, Str("str"))
	TestExpectRun(t, `return typeName([])`, nil, Str("array"))
	TestExpectRun(t, `return typeName({})`, nil, Str("dict"))
	TestExpectRun(t, `return typeName(error(""))`, nil, Str("error"))
	TestExpectRun(t, `return typeName(bytes())`, nil, Str("bytes"))
	TestExpectRun(t, `return typeName(func(){})`, nil, Str("compiledFunction"))
	TestExpectRun(t, `return typeName(append)`, nil, Str("builtinFunction"))
	TestExpectRun(t, `return typeName((;))`, nil, Str("keyValueArray"))
	TestExpectRun(t, `return typeName((;a,b=2))`, nil, Str("keyValueArray"))
	TestExpectRun(t, `return typeName(func(**na){return na}(;a,b=2))`, nil, Str("namedArgs"))
	TestExpectRun(t, `return typeName(buffer())`, nil, Str("buffer"))

	TestExpectRun(t, `b := buffer(); return write(b, "abc")`, nil, Int(3))
	TestExpectRun(t, `b := buffer(); write(b, "abc"); return str(b)`, nil, Str("abc"))
	TestExpectRun(t, `b := buffer(); write(b, "abc"); return str(read(b))`, nil, Str("abc"))
	TestExpectRun(t, `b := buffer(); write(b, "abc"); return [str(read(b,limit=2)), str(b)]`, nil,
		Array{Str("ab"), Str("c")})
	TestExpectRun(t, `b := buffer(); write(b, "abc"); return [str(read(b,limit=1)),str(read(b,limit=1)), str(b)]`,
		nil, Array{Str("a"), Str("b"), Str("c")})
	TestExpectRun(t, `b := buffer(); c := bytes(length=2); write(b, "abc"); return [str(read(b, c)),str(read(b, c)), str(char(c[0]))]`,
		nil, Array{Str("ab"), Str("c"), Str("c")})

	TestExpectRun(t, `w := buffer(); r := buffer(); write(r, "abc"); return [copy(w, r), str(w)]`,
		nil, Array{Int(3), Str("abc")})

	expectErrIs(t, `typeName()`, nil, ErrWrongNumArguments)
	expectErrIs(t, `typeName("", "")`, nil, ErrWrongNumArguments)

	TestExpectRun(t, `return str(keyValue("a",1))`,
		nil, Str("a=1"))
	TestExpectRun(t, `return str(keyValue("a b",1))`,
		nil, Str(`"a b"=1`))
	TestExpectRun(t, `return str(keyValueArray(nil,keyValue("a",1),{b:2},["c",3],
keyValueArray(keyValue("d",4),[e=5])))`,
		nil, Str(`(;a=1, b=2, 0="c", 1=3, d=4, e=5)`))

	TestExpectRun(t, `return sort(collect(keys({a:1,b:2})))`,
		nil, Array{Str("a"), Str("b")})
	TestExpectRun(t, `return str(collect(keys([5,6])))`,
		nil, Str("[0, 1]"))
	TestExpectRun(t, `return str(collect(keys((;a=1,b=2))))`,
		nil, Str(`["a", "b"]`))

	TestExpectRun(t, `return sort(collect(items({a:1,b:2})))`,
		nil, Array{&KeyValue{Str("a"), Int(1)}, &KeyValue{Str("b"), Int(2)}})
	TestExpectRun(t, `return str(collect(items([3, 2, 1])))`, nil, Str("[0=3, 1=2, 2=1]"))
	TestExpectRun(t, `return str(collect(items(keyValueArray(keyValue("a",1),keyValue("b",2)))))`,
		nil, Str(`[a=1, b=2]`))

	TestExpectRun(t, `return sort(collect(values({a:1,b:2})))`,
		nil, Array{Int(1), Int(2)})
	TestExpectRun(t, `return str(collect(values(keyValueArray(keyValue("a",1),keyValue("b",2)))))`,
		nil, Str(`[1, 2]`))

	TestExpectRun(t, `return str(buffer())`, nil, Str(""))
	TestExpectRun(t, `return str(buffer("abc"))`, nil, Str("abc"))
	TestExpectRun(t, `b := buffer("a"); write(b, "b", 1); write(b, true); return str(b)`,
		nil, Str("ab1true"))
	TestExpectRun(t, `b := buffer("a"); write(b, "b", 1); b.reset(); write(b, true); return str(b)`,
		nil, Str("true"))
	TestExpectRun(t, `return str(bytes(buffer("a")))`, nil, Str("a"))
	TestExpectRun(t, `return str(1, 2)`, nil, Str("12"))
	TestExpectRun(t, `return str(1, 2)`, nil, Str("12"))
	TestExpectRun(t, `return collect(values(map([1,2], (v, _) => v+1)))`, nil, Array{Int(2), Int(3)})
	TestExpectRun(t, `return collect(values(map([1,2], (v, k) => v+k)))`, nil, Array{Int(1), Int(3)})
	TestExpectRun(t, `return reduce([1,2], (cur, v, k) => cur + v)`, nil, Int(4))
	TestExpectRun(t, `return reduce([1,2], (cur, v, k) => cur + v, 10)`, nil, Int(13))
	TestExpectRun(t, `cur := 10; each([1,2], func(k, v) { cur += v });return cur`, nil, Int(13))

	var (
		c  srwcloser
		co = MustNewReflectValue(&c)
	)
	TestExpectRun(t, `param c; c.Reset(); return [str(typeof(close(c))), c.Closed]`, NewTestOpts().Args(co),
		Array{Str("gad_test.srwcloser"), True})
	TestExpectRun(t, `param c; c.Reset(); write(c, "a"); return [str(read(c;close)), c.Closed]`, NewTestOpts().Args(co),
		Array{Str("a"), True})
	TestExpectRun(t, `param c; c.Reset(); return [write(c,"bc";close), read(c), c.Closed]`, NewTestOpts().Args(co),
		Array{Int(2), Bytes("bc"), True})
	TestExpectRun(t, `param c; c.Reset(); write(c, "a"); return [c.Closed, read(c),  str(typeof(close(c))), c.Closed]`,
		NewTestOpts().Args(co),
		Array{False, Bytes("a"), Str("gad_test.srwcloser"), True})
	TestExpectRun(t, `return close(1)`, nil, Int(1))
	TestExpectRun(t, `
Point := struct("Point",fields={closed:false})
o := Point()
func close(p Point) {
	p.closed = true
	return p
}
return str(close(o))`, nil, Str("Point{closed: true}"))

	convs := []struct {
		f      string
		inputs map[string]Object
	}{
		{
			"int",
			map[string]Object{
				"1":       Int(1),
				"1u":      Int(1),
				"1d":      Int(1),
				"1.0":     Int(1),
				`'\x01'`:  Int(1),
				`'a'`:     Int(97),
				"true":    Int(1),
				"false":   Int(0),
				`"1"`:     Int(1),
				`"+123"`:  Int(123),
				`"-123"`:  Int(-123),
				`"0x10"`:  Int(16),
				`"0b101"`: Int(5),
			},
		},
		{
			"uint",
			map[string]Object{
				"1":       Uint(1),
				"1u":      Uint(1),
				"1d":      Uint(1),
				"1.0":     Uint(1),
				`'\x01'`:  Uint(1),
				`'a'`:     Uint(97),
				"true":    Uint(1),
				"false":   Uint(0),
				`"1"`:     Uint(1),
				"-1":      ^Uint(0),
				`"0x10"`:  Uint(16),
				`"0b101"`: Uint(5),
			},
		},
		{
			"char",
			map[string]Object{
				"1":      Char(1),
				"1u":     Char(1),
				"1d":     Char(1),
				"1.1":    Char(1),
				`'\x01'`: Char(1),
				"true":   Char(1),
				"false":  Char(0),
				`"1"`:    Char('1'),
				`""`:     Nil,
			},
		},
		{
			"float",
			map[string]Object{
				"1":      Float(1.0),
				"1u":     Float(1.0),
				"1.0":    Float(1.0),
				"1.3d":   Float(1.3),
				`'\x01'`: Float(1.0),
				"true":   Float(1.0),
				"false":  Float(0.0),
				`"1"`:    Float(1.0),
				`"1.1"`:  Float(1.1),
			},
		},
		{
			"decimal",
			map[string]Object{
				"1":      DecimalFromFloat(1.0),
				"1u":     DecimalFromFloat(1.0),
				"1.0":    DecimalFromFloat(1.0),
				`'\x01'`: DecimalFromFloat(1.0),
				"true":   DecimalFromFloat(1.0),
				"false":  DecimalFromFloat(0.0),
				`"1"`:    DecimalFromFloat(1.0),
				`"1.1"`:  DecimalFromFloat(1.1),
				"bytes(255, 255, 255, 250, 2, 7, 91, 205, 21)": MustDecimalFromString("123.456789"),
			},
		},
		{
			"str",
			map[string]Object{
				"1":                     Str("1"),
				"1u":                    Str("1"),
				"1.0":                   Str("1"),
				"123.4567890123456789d": Str("123.4567890123456789"),
				`'\x01'`:                Str("\x01"),
				"true":                  Str("true"),
				"false":                 Str("false"),
				`"1"`:                   Str("1"),
				`"1.1"`:                 Str("1.1"),
				`nil`:                   Str("nil"),
				`[]`:                    Str("[]"),
				`[1]`:                   Str("[1]"),
				`[1, 2]`:                Str("[1, 2]"),
				`{}`:                    Str("{}"),
				`{a: 1}`:                Str(`{a: 1}`),
				`{"a b": 1}`:            Str(`{"a b": 1}`),
				`error("an error")`:     Str(`error: an error`),
			},
		},
		{
			"bytes",
			map[string]Object{
				"1":           Bytes{1},
				"1u":          Bytes{1},
				`'\x01'`:      Bytes{1},
				"1, 2u":       Bytes{1, 2},
				"1, '\x02'":   Bytes{1, 2},
				"1u, 2":       Bytes{1, 2},
				`'\x01', 2u`:  Bytes{1, 2},
				`'\x01', 2`:   Bytes{1, 2},
				`bytes(1, 2)`: Bytes{1, 2},
				`"abc"`:       Bytes{'a', 'b', 'c'},
				"`abc`":       Bytes{'a', 'b', 'c'},
				"123.456789d": Bytes{255, 255, 255, 250, 2, 7, 91, 205, 21},
			},
		},
		{
			"chars",
			map[string]Object{
				`""`:             Array{},
				`"abc"`:          Array{Char('a'), Char('b'), Char('c')},
				`bytes("abc")`:   Array{Char('a'), Char('b'), Char('c')},
				`"a\xc5"`:        Nil, // incorrect UTF-8
				`bytes("a\xc5")`: Nil, // incorrect UTF-8
			},
		},
	}
	for i, conv := range convs {
		for k, v := range conv.inputs {
			t.Run(fmt.Sprintf("%s#%d#%v", conv.f, i, k), func(t *testing.T) {
				TestExpectRun(t, fmt.Sprintf(`return %s(%s)`, conv.f, k), nil, v)
			})
		}
	}

	expectErrIs(t, `int(1, 2)`, nil, ErrWrongNumArguments)
	expectErrIs(t, `uint(1, 2)`, nil, ErrWrongNumArguments)
	expectErrIs(t, `char(1, 2)`, nil, ErrWrongNumArguments)
	expectErrIs(t, `float(1, 2)`, nil, ErrWrongNumArguments)
	expectErrIs(t, `chars(1, 2)`, nil, ErrWrongNumArguments)

	expectErrIs(t, `int([])`, nil, ErrType)
	expectErrIs(t, `uint([])`, nil, ErrType)
	expectErrIs(t, `char([])`, nil, ErrType)
	expectErrIs(t, `float([])`, nil, ErrType)
	expectErrIs(t, `chars([])`, nil, ErrType)
	expectErrIs(t, `bytes(1, 2, "")`, nil, ErrType)

	type trueValues []string
	type falseValues []string

	isfuncs := []struct {
		f           string
		trueValues  trueValues
		falseValues falseValues
	}{
		{
			`is`,
			trueValues{
				`bool, false`,
				`error, error("test")`,
				`int, 1`,
				`uint, 1u`,
				`float, 1.2`,
				`decimal, 1.2d`,
				`str, ""`,
				`char, '1'`,
				`bytes, bytes()`,
				`buffer, buffer()`,
				`keyValue, keyValue("a",1)`,
				`[bool, int], false`,
			},
			falseValues{
				`bool, 1`,
				`error, 1`,
				`int, 1u`,
				`uint, 1`,
				`float, 1`,
				`decimal, 1.2`,
				`str, 1`,
				`char, 1`,
				`bytes, 1`,
				`buffer, 1`,
				`keyValue, 1`,
				`[bool, int], 1.2`,
			},
		},
		{
			`isError`,
			trueValues{
				`error("test")`,
			},
			falseValues{
				"1", "1u", `""`, "1.1", "'\x01'", `bytes()`, "nil",
				"true", "false", "[]", "{}",
			},
		},
		{
			`isInt`,
			trueValues{
				"0", "1", "-1",
			},
			falseValues{
				"1u", `""`, "1.1", "'\x01'", `bytes()`, "nil",
				`error("x")`,
				"true", "false", "[]", "{}",
			},
		},
		{
			`isUint`,
			trueValues{
				"0u", "1u", "-1u",
			},
			falseValues{
				"1", "-1", `""`, "1.1", "'\x01'", `bytes()`, "nil",
				`error("x")`, "true", "false", "[]", "{}",
			},
		},
		{
			`isFloat`,
			trueValues{
				"0.0", "1.0", "-1.0",
			},
			falseValues{
				"1", "-1", `""`, "1u", "'\x01'", `bytes()`, "nil",
				`error("x")`, "true", "false", "[]", "{}",
			},
		},
		{
			`isChar`,
			trueValues{
				"'\x01'", `'a'`, `'b'`,
			},
			falseValues{
				"1", "-1", `""`, "1u", "1.1", `bytes()`, "nil",
				`error("x")`, "true", "false", "[]", "{}",
			},
		},
		{
			`isBool`,
			trueValues{
				"true", "false",
			},
			falseValues{
				"1", "-1", `""`, "1u", "1.1", "'\x01'", `bytes()`, "nil",
				`error("x")`, "[]", "{}",
			},
		},
		{
			`isStr`,
			trueValues{
				`""`, `"abc"`,
			},
			falseValues{
				"1", "-1", "1u", "1.1", "'\x01'", `bytes()`, "nil",
				`error("x")`, "true", "false", "[]", "{}",
			},
		},
		{
			`isRawStr`,
			trueValues{
				"``", "`abc`",
			},
			falseValues{
				"1", "-1", "1u", "1.1", "'\x01'", `bytes()`, "nil",
				`error("x")`, "true", "false", "[]", "{}", `""`, `"a"`,
			},
		},
		{
			`isBytes`,
			trueValues{
				`bytes()`, `bytes(1, 2)`,
			},
			falseValues{
				"1", "-1", "1u", "1.1", "'\x01'", `""`, "nil",
				`error("x")`, "true", "false", "[]", "{}",
			},
		},
		{
			`isDict`,
			trueValues{
				`{}`, `{a: 1}`,
			},
			falseValues{
				"1", "-1", "1u", "1.1", "'\x01'", `""`, `bytes()`, "nil",
				`error("x")`, "true", "false", "[]",
			},
		},
		{
			`isSyncDict`,
			trueValues{},
			falseValues{
				"1", "-1", "1u", "1.1", "'\x01'", `""`, `bytes()`, "nil",
				`error("x")`, "true", "false", "[]", "{}",
			},
		},
		{
			`isArray`,
			trueValues{
				`[]`, `[0]`,
			},
			falseValues{
				"1", "-1", "1u", "1.1", "'\x01'", `""`, `bytes()`, "nil",
				`error("x")`, "true", "false", "{}",
			},
		},
		{
			`isNil`,
			trueValues{
				`nil`,
			},
			falseValues{
				"1", "-1", "1u", "1.1", "'\x01'", `""`, `bytes()`, `error("x")`,
				"true", "false", "{}", "[]",
			},
		},
		{
			`isFunction`,
			trueValues{
				`len`, `append`, `func(){}`,
			},
			falseValues{
				"1", "-1", "1u", "1.1", "'\x01'", `""`, `bytes()`, "nil",
				`error("x")`, "true", "false", "{}", "[]",
			},
		},
		{
			`isCallable`,
			trueValues{
				`len`, `append`,
			},
			falseValues{
				"1", "-1", "1u", "1.1", "'\x01'", `""`, `bytes()`, "nil",
				`error("x")`, "true", "false", "{}", "[]",
			},
		},
		{
			`isIterable`,
			trueValues{
				`[]`, `{}`, `"abc"`, `""`, `bytes()`,
			},
			falseValues{
				"1", "-1", "1u", "1.1", "'\x01'", "nil", `error("x")`,
				"true", "false",
			},
		},
		{
			`bool`,
			trueValues{
				"1", "1u", "-1", "1.1", "'\x01'", "true", `"abc"`, `bytes(1)`, "1.1d",
			},
			falseValues{
				"0", "0u", "nil", `error("x")`, "false", `[]`, `{}`, `""`, `bytes()`, "0d",
			},
		},
	}
	for i, isfunc := range isfuncs {
		for _, v := range isfunc.trueValues {
			t.Run(fmt.Sprintf("%s#%d %v true", isfunc.f, i, v), func(t *testing.T) {
				TestExpectRun(t, fmt.Sprintf(`return %s(%s)`, isfunc.f, v), nil, True)
			})
		}
		for _, v := range isfunc.falseValues {
			t.Run(fmt.Sprintf("%s#%d %v false", isfunc.f, i, v), func(t *testing.T) {
				TestExpectRun(t, fmt.Sprintf(`return %s(%s)`, isfunc.f, v), nil, False)
			})
		}

		if isfunc.f != "is" {
			if isfunc.f != "isError" {
				t.Run(fmt.Sprintf("%s#%d 2args", isfunc.f, i), func(t *testing.T) {
					expectErrIs(t, fmt.Sprintf(`%s(nil, nil)`, isfunc.f),
						nil, ErrWrongNumArguments)
				})
			} else {
				t.Run(fmt.Sprintf("%s#%d 3args", isfunc.f, i), func(t *testing.T) {
					expectErrIs(t, fmt.Sprintf(`%s(nil, nil, nil)`, isfunc.f),
						nil, ErrWrongNumArguments)
				})
			}
		}
	}

	TestExpectRun(t, `global sm; return isSyncDict(sm)`,
		NewTestOpts().Globals(Dict{"sm": &SyncDict{Value: Dict{}}}), True)

	TestExpectRun(t, `return isError(WrongNumArgumentsError.New(""), WrongNumArgumentsError)`,
		nil, True)
	TestExpectRun(t, `
	f := func(){ 
		throw NotImplementedError.New("test") 
	}
	try {
		f()
	} catch err {
		return isError(err, NotImplementedError)
	}`, nil, True)

	var stdOut bytes.Buffer
	stdOut.Reset()
	TestExpectRun(t, `printf("test")`, NewTestOpts().Out(&stdOut).Skip2Pass(), Nil)
	require.Equal(t, "test", stdOut.String())

	stdOut.Reset()
	TestExpectRun(t, `printf("test %d", 1)`, NewTestOpts().Out(&stdOut).Skip2Pass(), Nil)
	require.Equal(t, "test 1", stdOut.String())

	stdOut.Reset()
	TestExpectRun(t, `printf("test %d %d", 1, 2u)`, NewTestOpts().Out(&stdOut).Skip2Pass(), Nil)
	require.Equal(t, "test 1 2", stdOut.String())

	stdOut.Reset()
	TestExpectRun(t, `println()`, NewTestOpts().Out(&stdOut).Skip2Pass(), Nil)
	require.Equal(t, "\n", stdOut.String())

	stdOut.Reset()
	TestExpectRun(t, `println("test")`, NewTestOpts().Out(&stdOut).Skip2Pass(), Nil)
	require.Equal(t, "test\n", stdOut.String())

	stdOut.Reset()
	TestExpectRun(t, `println("test", 1)`, NewTestOpts().Out(&stdOut).Skip2Pass(), Nil)
	require.Equal(t, "test 1\n", stdOut.String())

	stdOut.Reset()
	TestExpectRun(t, `println("test", 1, 2u)`, NewTestOpts().Out(&stdOut).Skip2Pass(), Nil)
	require.Equal(t, "test 1 2\n", stdOut.String())

	TestExpectRun(t, `return sprintf("test")`,
		NewTestOpts().Out(&stdOut).Skip2Pass(), Str("test"))
	TestExpectRun(t, `return sprintf("test %d", 1)`,
		NewTestOpts().Out(&stdOut).Skip2Pass(), Str("test 1"))
	TestExpectRun(t, `return sprintf("test %d %t", 1, true)`,
		NewTestOpts().Out(&stdOut).Skip2Pass(), Str("test 1 true"))
	TestExpectRun(t, `f := func(*args;**kwargs){ return [args, kwargs.dict] };
		return wrap(f, 1, a=3)(2, b=4)`,
		nil, Array{Array{Int(1), Int(2)}, Dict{"a": Int(3), "b": Int(4)}})

	expectErrIs(t, `printf()`, nil, ErrWrongNumArguments)
	expectErrIs(t, `sprintf()`, nil, ErrWrongNumArguments)
}

func TestVMBuiltinObj(t *testing.T) {
	TestExpectRun(t, `return myval`,
		NewTestOpts().Builtins(map[string]Object{
			"myval": Int(2),
		}), Int(2))

	type S struct{}
	obj := MustToObject(&S{}).(*ReflectStruct)
	TestExpectRun(t, `return myval`,
		NewTestOpts().Builtins(map[string]Object{
			"myval": obj,
		}), obj)

	obj.Type().(*ReflectType).CallObject = func(o *ReflectStruct, c Call) (Object, error) {
		return Array{Bool(o == obj), Str("call *S result")}, nil
	}

	TestExpectRun(t, `return [myval, myval()]`,
		NewTestOpts().Builtins(map[string]Object{
			"myval": obj,
		}), Array{
			NewCallerObjectWithMethods(obj),
			Array{True, Str("call *S result")},
		})

	TestExpectRun(t, `
func myval(i int) {
	return "method with int value = "+i
}
return [repr(myval), myval(), myval(2)]`,
		NewTestOpts().Builtins(map[string]Object{
			"myval": obj,
		}), Array{
			Str("‹reflect:github.com/gad-lang/gad_test.S:{} with 1 methods:\n\t1. ‹compiledFunction #1(i int)››"),
			Array{True, Str("call *S result")},
			Str("method with int value = 2"),
		})
}

func TestObjectType(t *testing.T) {
	TestExpectRun(t, `
Point := struct(
	"Point", 
	fields={x:0, y:0},
)
func Point() => 2 
return str(Point())`,
		nil, Str(`2`))

	TestExpectRun(t, `
Point := struct(
	"Point", 
	fields={x:0, y:0},
)
func Point() => Point.new(x=2) 
return str(Point())`,
		nil, Str(`Point{x: 2}`))

	TestExpectRun(t, `
Point := struct(
	"Point", 
	fields={x:0, y:0},
)
func Point(x, y) => Point(x=x, y=y) // or Point.new(x=x, y=y)
func str(p Point) => "P" + p.x + p.y 
return str(Point(1,2))`,
		nil, Str(`P12`))

	TestExpectRun(t, `
Point := struct("Point", 
	fields={_x:0, _y:0}, 
	set={
		x: func(this, v) {this._x = v},
		y: func(this, v) {this._y = v},
	},
	get={
		x: (this) => this._x,
		y: (this) => this._y,
	},
	methods={
		addX: func(this, v) {this.x += v; return this.x},
	},
)
func Point(x, y) => Point(_x=x, _y=y)
p := Point(1, 2)
return str(p), p.x, p.y, p.addX(3)`,
		nil, Array{Str(`Point{_x: 1, _y: 2}`), Int(1), Int(2), Int(4)})

	TestExpectRun(t, `Point := struct("Point", fields={x:0, y:0}); return str(Point())`,
		nil, Str(`Point{}`))
	TestExpectRun(t, `
Point := struct("Point", fields={x:0, y:0}); 
func Point(x, y) => Point(x=x, y=y)
return str(Point(1, 2))`,
		nil, Str(`Point{x: 1, y: 2}`))
	TestExpectRun(t, `return struct("Point").name`,
		nil, Str("Point"))
	TestExpectRun(t, `
Point := struct(
	"Point", 
	fields={x:0, y:0},
)
func Point(x, y) => Point(x=x, y=y)
func str(p Point) => "P" + p.x + p.y 
return str(Point(1,2))`,
		nil, Str(`P12`))

	TestExpectRun(t, `
P1 := struct("P1",fields={x:0, y:0})
P2 := struct("P2",fields={x:0, y:0, z:0})
p1 := P1(x=10,y=11)
p2 := P2(x=1,y=2,z=3)
return [str(p1), str(p2), str(cast(P1,p2)), str(cast(P2,cast(P1,p2)))]
`,
		nil, Array{
			Str("P1{x: 10, y: 11}"),
			Str("P2{x: 1, y: 2, z: 3}"),
			Str("P1{x: 1, y: 2, z: 3}"),
			Str("P2{x: 1, y: 2, z: 3}"),
		})

	TestExpectRun(t, `
Point := struct(
	"Point", 
	fields={x:0, y:0}, 
)

func Point(x, y) => Point(x=x, y=y)
func str(p Point) => "P" + p.x + p.y 
func write(p Point) => write(typeName(p),"(", p.x,",",p.y,")")

return write(Point(10,20))`,
		NewTestOpts().Buffered(), Array{Int(12), Str(`Point(10,20)`)})

	TestExpectRun(t, `
Point := struct(
	"Point", 
	fields={x:0, y:0},
)
func Point(x, y) => Point(x=x, y=y)
func str(p Point) => "P" + p.x + p.y 
func write(p Point) => write(typeName(p),"(", p.x,",",p.y,")")

b := buffer()
write(b, Point(10,20))
return str(b)`,
		NewTestOpts(), Str(`P1020`))

	TestExpectRun(t, `Point := struct(
	"Point", 
	fields={x:0, y:0}, 
)

func Point(x, y) => Point(x=x, y=y)

func binaryOp(_ TBinOpMul, p Point, val int) {
	p.x *= val
	p.y *= val
	return p
}

return (Point(2,3)*3) .| dict
`, nil, Dict{"x": Int(6), "y": Int(9)})

	TestExpectRun(t, `
Point := struct(
	"Point", 
	fields={x:0, y:0},
)

func Point(x, y) => Point(x=x, y=y)
func int(p Point) => rawCaller(int)(p.x * p.y)
return [int(Point(2, 8)), str(int)]
`,
		nil, Array{Int(16), Str(ReprQuote("builtinType int") + " with 1 methods:\n\t" +
			"1. " + ReprQuote("compiledFunction #7(p Point)"))})
}

func TestCallerMethod(t *testing.T) {
	TestExpectRun(t, `
func f0() {
	return "abc"
}
addCallMethod(f0, (i int|uint) => i)
return f0(), f0(2), f0(uint(3))`,
		NewTestOpts(), Array{Str("abc"), Int(2), Uint(3)})

	TestExpectRun(t, `
func f0() {
	return "abc"
}
func f0(i int|uint) => i
return f0(), f0(2), f0(uint(3))`,
		NewTestOpts(), Array{Str("abc"), Int(2), Uint(3)})

	TestExpectRun(t, `
func f() => nil
func f(b bool) => nil
func f1(i int) => nil
func f1(i int, b bool) => nil
addCallMethod(f, f1)
return [str(f), str(f1)]`,
		NewTestOpts(), Array{Str(ReprQuote("compiledFunction f()") + " with 3 methods:\n\t" +
			"1. " + ReprQuote("compiledFunction #1(b bool)") + "\n\t" +
			"2. " + ReprQuote("compiledFunction f1(i int)") + "\n\t" +
			"3. " + ReprQuote("compiledFunction #3(i int, b bool)")),
			Str(ReprQuote("compiledFunction f1(i int)") + " with 1 methods:\n\t" +
				"1. " + ReprQuote("compiledFunction #3(i int, b bool)"))})

	TestExpectRun(t, `
func f0(i int) => i*2
func f0() => "no args"
func f0(s str) => s+"b"
return str(f0), f0(), f0(2), f0("a")`,
		NewTestOpts(),
		Array{
			Str(ReprQuote("compiledFunction f0(i int)") + " with 2 methods:\n\t" +
				"1. " + ReprQuote("compiledFunction #3()") + "\n\t" +
				"2. " + ReprQuote("compiledFunction #5(s str)")),
			Str("no args"),
			Int(4),
			Str("ab"),
		})

	TestExpectRun(t, `
func f0() {}
func f1() {}
func f2() {}
func f3(v bool) => v
func f4 (s str) => s

const ( 
	f5 = (b bytes) => nil
	f7 = (b bool,i int) => nil
)

func f6(s str,i int) {}

addCallMethod(f0, (i int) => i)
addCallMethod(f1, (i int) => i)
addCallMethod(f1, (i uint) => i)
addCallMethod(f2, (i int) => i)
addCallMethod(f3, (i int) => i)
addCallMethod(f4, (i int) => i)

return [
	[f0(0), f1(1), f2(2), f3(3), f4(4)],
	[str(f0),str(f1),str(f2),str(f3),str(f4),str(f5),str(f6),str(f7)],
]`,
		NewTestOpts(), Array{
			Array{Int(0), Int(1), Int(2), Int(3), Int(4)},
			Array{
				Str(ReprQuote("compiledFunction f0()") + " with 1 methods:\n\t" +
					"1. " + ReprQuote("compiledFunction #8(i int)")),
				Str(ReprQuote("compiledFunction f1()") + " with 2 methods:\n\t" +
					"1. " + ReprQuote("compiledFunction #9(i int)") + "\n\t" +
					"2. " + ReprQuote("compiledFunction #10(i uint)")),
				Str(ReprQuote("compiledFunction f2()") + " with 1 methods:\n\t" +
					"1. " + ReprQuote("compiledFunction #11(i int)")),
				Str(ReprQuote("compiledFunction f3(v bool)") + " with 1 methods:\n\t" +
					"1. " + ReprQuote("compiledFunction #12(i int)")),
				Str(ReprQuote("compiledFunction f4(s str)") + " with 1 methods:\n\t" +
					"1. " + ReprQuote("compiledFunction #13(i int)")),
				Str(ReprQuote("compiledFunction f5(b bytes)")),
				Str(ReprQuote("compiledFunction f6(s str, i int)")),
				Str(ReprQuote("compiledFunction f7(b bool, i int)")),
			},
		})
}

func TestBytes(t *testing.T) {
	TestExpectRun(t, `return bytes("Hello World!")`, nil, Bytes("Hello World!"))
	TestExpectRun(t, `return bytes("Hello") + bytes(" ") + bytes("World!")`,
		nil, Bytes("Hello World!"))
	TestExpectRun(t, `return bytes("Hello") + bytes(" ") + "World!"`,
		nil, Bytes("Hello World!"))
	TestExpectRun(t, `return "Hello " + bytes("World!")`,
		nil, Str("Hello World!"))

	// slice
	TestExpectRun(t, `return bytes("")[:]`, nil, Bytes{})
	TestExpectRun(t, `return bytes("abcde")[:]`, nil, Bytes(Str("abcde")))
	TestExpectRun(t, `return bytes("abcde")[0:]`, nil, Bytes(Str("abcde")))
	TestExpectRun(t, `return bytes("abcde")[:0]`, nil, Bytes{})
	TestExpectRun(t, `return bytes("abcde")[:1]`, nil, Bytes(Str("a")))
	TestExpectRun(t, `return bytes("abcde")[:2]`, nil, Bytes(Str("ab")))
	TestExpectRun(t, `return bytes("abcde")[0:2]`, nil, Bytes(Str("ab")))
	TestExpectRun(t, `return bytes("abcde")[1:]`, nil, Bytes(Str("bcde")))
	TestExpectRun(t, `return bytes("abcde")[1:5]`, nil, Bytes(Str("bcde")))
	TestExpectRun(t, `
	b1 := bytes("abcde")
	b2 := b1[:2]
	return b2[:len(b1)]`, nil, Bytes(Str("abcde")))
	TestExpectRun(t, `
	b1 := bytes("abcde")
	b2 := b1[:2]
	return cap(b1) == cap(b2)`, nil, True)

	// bytes[] -> int
	TestExpectRun(t, `return bytes("abcde")[0]`, nil, Int('a'))
	TestExpectRun(t, `return bytes("abcde")[1]`, nil, Int('b'))
	TestExpectRun(t, `return bytes("abcde")[4]`, nil, Int('e'))
	TestExpectRun(t, `return bytes("abcde")[-1]`, nil, Int('e'))
	TestExpectRun(t, `return bytes("abcde")[-2]`, nil, Int('d'))
	expectErrIs(t, `return bytes("abcde")[-10]`, nil, ErrIndexOutOfBounds)
	expectErrIs(t, `return bytes("abcde")[100]`, nil, ErrIndexOutOfBounds)
	expectErrIs(t, `b1 := bytes("abcde");	b2 := b1[:cap(b1)+1]`, nil, ErrIndexOutOfBounds)
}

func TestVMChar(t *testing.T) {
	TestExpectRun(t, `return 'a'`, nil, Char('a'))
	TestExpectRun(t, `return '九'`, nil, Char(20061))
	TestExpectRun(t, `return 'Æ'`, nil, Char(198))
	TestExpectRun(t, `return '0' + '9'`, nil, Char(105))
	TestExpectRun(t, `return '0' + 9`, nil, Char('9'))
	TestExpectRun(t, `return 1 + '9'`, nil, Char(1)+Char('9'))
	TestExpectRun(t, `return '9' - 4`, nil, Char('5'))
	TestExpectRun(t, `return '0' == '0'`, nil, True)
	TestExpectRun(t, `return '0' != '0'`, nil, False)
	TestExpectRun(t, `return '2' < '4'`, nil, True)
	TestExpectRun(t, `return '2' > '4'`, nil, False)
	TestExpectRun(t, `return '2' <= '4'`, nil, True)
	TestExpectRun(t, `return '2' >= '4'`, nil, False)
	TestExpectRun(t, `return '4' < '4'`, nil, False)
	TestExpectRun(t, `return '4' > '4'`, nil, False)
	TestExpectRun(t, `return '4' <= '4'`, nil, True)
	TestExpectRun(t, `return '4' >= '4'`, nil, True)
	TestExpectRun(t, `return '九' + "Hello"`, nil, Str("九Hello"))
	TestExpectRun(t, `return "Hello" + '九'`, nil, Str("Hello九"))
}

func TestVMCondExpr(t *testing.T) {
	TestExpectRun(t, `return true ? 5`, nil, Int(5))
	TestExpectRun(t, `true ? 5 : 10`, nil, Nil)
	TestExpectRun(t, `false ? 5 : 10; var a; return a`, nil, Nil)
	TestExpectRun(t, `return true ? 5 : 10`, nil, Int(5))
	TestExpectRun(t, `return false ? 5 : 10`, nil, Int(10))
	TestExpectRun(t, `return (1 == 1) ? 2 + 3 : 12 - 2`, nil, Int(5))
	TestExpectRun(t, `return (1 != 1) ? 2 + 3 : 12 - 2`, nil, Int(10))
	TestExpectRun(t, `return (1 == 1) ? true ? 10 - 8 : 1 + 3 : 12 - 2`, nil, Int(2))
	TestExpectRun(t, `return (1 == 1) ? false ? 10 - 8 : 1 + 3 : 12 - 2`, nil, Int(4))

	TestExpectRun(t, `
	out := 0
	f1 := func() { out += 10 }
	f2 := func() { out = -out }
	true ? f1() : f2()
	return out
	`, nil, Int(10))
	TestExpectRun(t, `
	out := 5
	f1 := func() { out += 10 }
	f2 := func() { out = -out }
	false ? f1() : f2()
	return out
	`, nil, Int(-5))
	TestExpectRun(t, `
	f1 := func(a) { return a + 2 }
	f2 := func(a) { return a - 2 }
	f3 := func(a) { return a + 10 }
	f4 := func(a) { return -a }

	f := func(c) {
		return c == 0 ? f1(c) : f2(c) ? f3(c) : f4(c)
	}

	return [f(0), f(1), f(2)]
	`, nil, Array{Int(2), Int(11), Int(-2)})

	TestExpectRun(t, `f := func(a) { return -a }; return f(true ? 5 : 3)`, nil, Int(-5))
	TestExpectRun(t, `return [false?5:10, true?1:2]`, nil, Array{Int(10), Int(1)})

	TestExpectRun(t, `
	return 1 > 2 ?
		1 + 2 + 3 :
		10 - 5`, nil, Int(5))
}

func TestVMThrowExpression(t *testing.T) {
	expectErrIs(t, `return throw "my-error"`, nil, &Error{Message: "my-error"})
	TestExpectRun(t, `return false ? throw "my-error" : 1`, nil, Int(1))
	expectErrIs(t, `return true ? throw "my-error" : 1`, nil, &Error{Message: "my-error"})
}

func TestVMEquality(t *testing.T) {
	testEquality(t, `1`, `1`, true)
	testEquality(t, `1`, `2`, false)

	testEquality(t, `1.0`, `1.0`, true)
	testEquality(t, `1.0`, `1.1`, false)

	testEquality(t, `true`, `true`, true)
	testEquality(t, `true`, `false`, false)

	testEquality(t, `"foo"`, `"foo"`, true)
	testEquality(t, `"foo"`, `"bar"`, false)

	testEquality(t, `'f'`, `'f'`, true)
	testEquality(t, `'f'`, `'b'`, false)

	testEquality(t, `[]`, `[]`, true)
	testEquality(t, `[1]`, `[1]`, true)
	testEquality(t, `[1]`, `[1, 2]`, false)
	testEquality(t, `["foo", "bar"]`, `["foo", "bar"]`, true)
	testEquality(t, `["foo", "bar"]`, `["bar", "foo"]`, false)

	testEquality(t, `{}`, `{}`, true)
	testEquality(t, `{a: 1, b: 2}`, `{b: 2, a: 1}`, true)
	testEquality(t, `{a: 1, b: 2}`, `{b: 2}`, false)
	testEquality(t, `{a: 1, b: {}}`, `{b: {}, a: 1}`, true)

	testEquality(t, `1`, `"foo"`, false)
	testEquality(t, `1`, `true`, true)
	testEquality(t, `[1]`, `["1"]`, false)
	testEquality(t, `[1, [2]]`, `[1, ["2"]]`, false)
	testEquality(t, `{a: 1}`, `{a: "1"}`, false)
	testEquality(t, `{a: 1, b: {c: 2}}`, `{a: 1, b: {c: "2"}}`, false)
}

func testEquality(t *testing.T, lhs, rhs string, expected bool) {
	t.Helper()
	// 1. equality is commutative
	// 2. equality and inequality must be always opposite
	TestExpectRun(t, fmt.Sprintf("return %s == %s", lhs, rhs), nil, Bool(expected))
	TestExpectRun(t, fmt.Sprintf("return %s == %s", rhs, lhs), nil, Bool(expected))
	TestExpectRun(t, fmt.Sprintf("return %s != %s", lhs, rhs), nil, Bool(!expected))
	TestExpectRun(t, fmt.Sprintf("return %s != %s", rhs, lhs), nil, Bool(!expected))
}

func TestVMBuiltinError(t *testing.T) {
	TestExpectRun(t, `return error(1)`, nil, &Error{Name: "error", Message: "1"})
	TestExpectRun(t, `return error(1).Literal`, nil, Str("error"))
	TestExpectRun(t, `return error(1).Message`, nil, Str("1"))
	TestExpectRun(t, `return error("some error")`, nil,
		&Error{Name: "error", Message: "some error"})
	TestExpectRun(t, `return error("some" + " error")`, nil,
		&Error{Name: "error", Message: "some error"})

	TestExpectRun(t, `return func() { return error(5) }()`, nil,
		&Error{Name: "error", Message: "5"})
	TestExpectRun(t, `return error(error("foo"))`, nil, &Error{Name: "error", Message: "error: foo"})

	TestExpectRun(t, `return error("some error").Literal`, nil, Str("error"))
	TestExpectRun(t, `return error("some error")["Literal"]`, nil, Str("error"))
	TestExpectRun(t, `return error("some error").Message`, nil, Str("some error"))
	TestExpectRun(t, `return error("some error")["Message"]`, nil, Str("some error"))

	TestExpectRun(t, `error("error").err`, nil, Nil)
	TestExpectRun(t, `error("error").value_`, nil, Nil)
	TestExpectRun(t, `error([1,2,3])[1]`, nil, Nil)
}

func TestVMFloat(t *testing.T) {
	TestExpectRun(t, `return 0.0`, nil, Float(0.0))
	TestExpectRun(t, `return -10.3`, nil, Float(-10.3))
	TestExpectRun(t, `return 3.2 + 2.0 * -4.0`, nil, Float(-4.8))
	TestExpectRun(t, `return 4 + 2.3`, nil, Float(6.3))
	TestExpectRun(t, `return 2.3 + 4`, nil, Float(6.3))
	TestExpectRun(t, `return +5.0`, nil, Float(5.0))
	TestExpectRun(t, `return -5.0 + +5.0`, nil, Float(0.0))
}

func TestVMForIn(t *testing.T) {
	// array
	TestExpectRun(t, `out := 0; for x in [1, 2, 3] { out += x }; return out`,
		nil, Int(6)) // value
	TestExpectRun(t, `out := 0; for i, x in [1, 2, 3] { out += i + x }; return out`,
		nil, Int(9)) // index, value
	TestExpectRun(t, `out := 0; func() { for i, x in [1, 2, 3] { out += i + x } }(); return out`,
		nil, Int(9)) // index, value
	TestExpectRun(t, `out := 0; for i, _ in [1, 2, 3] { out += i }; return out`,
		nil, Int(3)) // index, _
	TestExpectRun(t, `out := 0; func() { for i, _ in [1, 2, 3] { out += i  } }(); return out`,
		nil, Int(3)) // index, _

	// map
	TestExpectRun(t, `out := 0; for v in {a:2,b:3,c:4} { out += v }; return out`,
		nil, Int(9)) // value
	TestExpectRun(t, `out := ""; for k, v in {a:2,b:3,c:4} { out = k; if v==3 { break } }; return out`,
		nil, Str("b")) // key, value
	TestExpectRun(t, `out := ""; for k, _ in {a:2} { out += k }; return out`,
		nil, Str("a")) // key, _
	TestExpectRun(t, `out := 0; for _, v in {a:2,b:3,c:4} { out += v }; return out`,
		nil, Int(9)) // _, value
	TestExpectRun(t, `out := ""; func() { for k, v in {a:2,b:3,c:4} { out = k; if v==3 { break } } }(); return out`,
		nil, Str("b")) // key, value

	// syncMap
	g := Dict{"syncMap": &SyncDict{Value: Dict{"a": Int(2), "b": Int(3), "c": Int(4)}}}
	TestExpectRun(t, `out := 0; for v in globals().syncMap { out += v }; return out`,
		NewTestOpts().Globals(g).Skip2Pass(), Int(9)) // value
	TestExpectRun(t, `out := ""; for k, v in globals().syncMap { out = k; if v==3 { break } }; return out`,
		NewTestOpts().Globals(g).Skip2Pass(), Str("b")) // key, value
	TestExpectRun(t, `out := ""; for k, _ in globals().syncMap { out += k }; return out`,
		NewTestOpts().Globals(Dict{"syncMap": &SyncDict{Value: Dict{"a": Int(2)}}}).Skip2Pass(), Str("a")) // key, _
	TestExpectRun(t, `out := 0; for _, v in globals().syncMap { out += v }; return out`,
		NewTestOpts().Globals(g).Skip2Pass(), Int(9)) // _, value
	TestExpectRun(t, `out := ""; func() { for k, v in globals().syncMap { out = k; if v==3 { break } } }(); return out`,
		NewTestOpts().Globals(g).Skip2Pass(), Str("b")) // key, value

	// string
	TestExpectRun(t, `out := ""; for c in "abcde" { out += c }; return out`, nil, Str("abcde"))
	TestExpectRun(t, `out := ""; for i, c in "abcde" { if i == 2 { continue }; out += c }; return out`,
		nil, Str("abde"))

	// bytes
	TestExpectRun(t, `out := ""; for c in bytes("abcde") { out += char(c) }; return out`, nil, Str("abcde"))
	TestExpectRun(t, `out := ""; for i, c in bytes("abcde") { if i == 2 { continue }; out += char(c) }; return out`,
		nil, Str("abde"))

	expectErrIs(t, `a := 1; for k,v in a {}`, nil, ErrNotIterable)

	// nil iterator
	TestExpectRun(t, `for k, v in nil {return v}`, nil, Nil)
	TestExpectRun(t, `for k, v in nil {return v} else {return "is nil"}`, nil, Str("is nil"))

	// with else
	TestExpectRun(t, `var r = ""; for x in [] { r += str(x) } else { r += "@"}; r+="#"; return r`, nil, Str("@#"))
	TestExpectRun(t, `var r = ""; for x in [1] { r += str(x) } else { r += "@"}; r+="#"; return r`, nil, Str("1#"))
	TestExpectRun(t, `var r = ""; for x in [1,2] { r += str(x) } else { r += "@"}; r+="#"; return r`, nil, Str("12#"))
	TestExpectRun(t, `var r = (;); 
		for k, v in bytes("abc") { 
			r = append(r, keyValue(k, char(v))) 
		} else { 
			r = append(r, keyValue("else", true)) 
		}; 
		r = append(r, keyValue("done", yes))
		return str(r)`, nil, Str("(;0=a, 1=b, 2=c, done)"))
	TestExpectRun(t, `var r = (;); 
		for k, v in bytes("") { 
			r = append(r, keyValue(k, char(v))) 
		} else { 
			r = append(r, keyValue("else", yes)) 
		}; 
		r = append(r, keyValue("done", yes))
		return str(r)`, nil, Str("(;else, done)"))
}

func TestFor(t *testing.T) {
	TestExpectRun(t, `
	out := 0
	for {
		out++
		if out == 5 {
			break
		}
	}
	return out`, nil, Int(5))

	TestExpectRun(t, `
	out := 0
	a := 0
	for {
		a++
		if a == 3 { continue }
		if a == 5 { break }
		out += a
	}
	return out`, nil, Int(7)) // 1 + 2 + 4

	TestExpectRun(t, `
	out := 0
	a := 0
	for {
		a++
		if a == 3 { continue }
		out += a
		if a == 5 { break }
	}
	return out`, nil, Int(12)) // 1 + 2 + 4 + 5

	TestExpectRun(t, `
	out := 0
	for true {
		out++
		if out == 5 {
			break
		}
	}
	return out`, nil, Int(5))

	TestExpectRun(t, `
	a := 0
	for true {
		a++
		if a == 5 {
			break
		}
	}
	return a`, nil, Int(5))

	TestExpectRun(t, `
	out := 0
	a := 0
	for true {
		a++
		if a == 3 { continue }
		if a == 5 { break }
		out += a
	}
	return out`, nil, Int(7)) // 1 + 2 + 4

	TestExpectRun(t, `
	out := 0
	a := 0
	for true {
		a++
		if a == 3 { continue }
		out += a
		if a == 5 { break }
	}
	return out`, nil, Int(12)) // 1 + 2 + 4 + 5

	TestExpectRun(t, `
	out := 0
	func() {
		for true {
			out++
			if out == 5 {
				return
			}
		}
	}()
	return out`, nil, Int(5))

	TestExpectRun(t, `
	out := 0
	for a:=1; a<=10; a++ {
		out += a
	}
	return out`, nil, Int(55))

	TestExpectRun(t, `
	out := 0
	for a:=1; a<=3; a++ {
		for b:=3; b<=6; b++ {
			out += b
		}
	}
	return out`, nil, Int(54))

	TestExpectRun(t, `
	out := 0
	func() {
		for {
			out++
			if out == 5 {
				break
			}
		}
	}()
	return out`, nil, Int(5))

	TestExpectRun(t, `
	out := 0
	func() {
		for true {
			out++
			if out == 5 {
				break
			}
		}
	}()
	return out`, nil, Int(5))

	TestExpectRun(t, `
	return func() {
		a := 0
		for {
			a++
			if a == 5 {
				break
			}
		}
		return a
	}()`, nil, Int(5))

	TestExpectRun(t, `
	return func() {
		a := 0
		for true {
			a++
			if a== 5 {
				break
			}
		}
		return a
	}()`, nil, Int(5))

	TestExpectRun(t, `
	return func() {
		a := 0
		func() {
			for {
				a++
				if a == 5 {
					break
				}
			}
		}()
		return a
	}()`, nil, Int(5))

	TestExpectRun(t, `
	return func() {
		a := 0
		func() {
			for true {
				a++
				if a == 5 {
					break
				}
			}
		}()
		return a
	}()`, nil, Int(5))

	TestExpectRun(t, `
	return func() {
		sum := 0
		for a:=1; a<=10; a++ {
			sum += a
		}
		return sum
	}()`, nil, Int(55))

	TestExpectRun(t, `
	return func() {
		sum := 0
		for a:=1; a<=4; a++ {
			for b:=3; b<=5; b++ {
				sum += b
			}
		}
		return sum
	}()`, nil, Int(48)) // (3+4+5) * 4

	TestExpectRun(t, `
	a := 1
	for ; a<=10; a++ {
		if a == 5 {
			break
		}
	}
	return a`, nil, Int(5))

	TestExpectRun(t, `
	out := 0
	for a:=1; a<=10; a++ {
		if a == 3 {
			continue
		}
		out += a
		if a == 5 {
			break
		}
	}
	return out`, nil, Int(12)) // 1 + 2 + 4 + 5

	TestExpectRun(t, `
	out := 0
	for a:=1; a<=10; {
		if a == 3 {
			a++
			continue
		}
		out += a
		if a == 5 {
			break
		}
		a++
	}
	return out`, nil, Int(12)) // 1 + 2 + 4 + 5
}

func TestVMFunction(t *testing.T) {
	// function with no "return" statement returns nil value.
	TestExpectRun(t, `f1 := func() {}; return f1()`, nil, Nil)
	TestExpectRun(t, `f1 := func() {}; f2 := func() { return f1(); }; f1(); return f2()`,
		nil, Nil)
	TestExpectRun(t, `f := func(x) { x; }; return f(5);`, nil, Nil)

	TestExpectRun(t, `f := func(*x) { return x; }; return f(1, 2, 3);`,
		nil, Array{Int(1), Int(2), Int(3)})

	TestExpectRun(t, `f := func(a, b, *x) { return [a, b, x]; }; return f(8, 9, 1, 2, 3);`,
		nil, Array{Int(8), Int(9), Array{Int(1), Int(2), Int(3)}})

	TestExpectRun(t, `f := func(v) { x := 2; return func(a, *b){ return [a, b, v+x]}; }; return f(5)("a", "b");`,
		nil, Array{Str("a"), Array{Str("b")}, Int(7)})

	TestExpectRun(t, `f := func(*x) { return x; }; return f();`, nil, Array{})

	TestExpectRun(t, `f := func(a, b, *x) { return [a, b, x]; }; return f(8, 9);`,
		nil, Array{Int(8), Int(9), Array{}})

	TestExpectRun(t, `f := func(v) { x := 2; return func(a, *b){ return [a, b, v+x]}; }; return f(5)("a");`,
		nil, Array{Str("a"), Array{}, Int(7)})

	expectErrIs(t, `f := func(a, b, *x) { return [a, b, x]; }; f();`, nil, ErrWrongNumArguments)
	expectErrHas(t, `f := func(a, b, *x) { return [a, b, x]; }; f();`, nil, "want>=2 got=0")

	expectErrIs(t, `f := func(a, b, *x) { return [a, b, x]; }; f(1);`, nil, ErrWrongNumArguments)
	expectErrHas(t, `f := func(a, b, *x) { return [a, b, x]; }; f(1);`, nil, "want>=2 got=1")

	TestExpectRun(t, `f := func(x) { return x; }; return f(5);`, nil, Int(5))
	TestExpectRun(t, `f := func(x) { return x * 2; }; return f(5);`, nil, Int(10))
	TestExpectRun(t, `f := func(x, y) { return x + y; }; return f(5, 5);`, nil, Int(10))
	TestExpectRun(t, `f := func(x, y) { return x + y; }; return f(5 + 5, f(5, 5));`,
		nil, Int(20))
	TestExpectRun(t, `return func(x) { return x; }(5)`, nil, Int(5))
	TestExpectRun(t, `x := 10; f := func(x) { return x; }; f(5); return x;`, nil, Int(10))

	TestExpectRun(t, `
	f2 := func(a) {
		f1 := func(a) {
			return a * 2;
		};

		return f1(a) * 3;
	}
	return f2(10)`, nil, Int(60))

	TestExpectRun(t, `
	f1 := func(f) {
		a := [nil]
		a[0] = func() { return f(a) }
		return a[0]()
	}
	return f1(func(a) { return 2 })
	`, nil, Int(2))

	// closures
	TestExpectRun(t, `
	newAdder := func(x) {
		return func(y) { return x + y }
	}
	add2 := newAdder(2)
	return add2(5)`, nil, Int(7))
	TestExpectRun(t, `
	var out
	m := {a: 1}
	for k,v in m {
		func(){
			out = k
		}()
	}
	return out`, nil, Str("a"))

	TestExpectRun(t, `
	var out
	m := {a: 1}
	for k,v in m {
		func(){
			out = v
		}()
	}; return out`, nil, Int(1))
	// function as a argument
	TestExpectRun(t, `
	add := func(a, b) { return a + b };
	sub := func(a, b) { return a - b };
	applyFunc := func(a, b, f) { return f(a, b) };

	return applyFunc(applyFunc(2, 2, add), 3, sub);
	`, nil, Int(1))

	TestExpectRun(t, `f1 := func() { return 5 + 10; }; return f1();`,
		nil, Int(15))
	TestExpectRun(t, `f1 := func() { return 1 }; f2 := func() { return 2 }; return f1() + f2()`,
		nil, Int(3))
	TestExpectRun(t, `f1 := func() { return 1 }; f2 := func() { return f1() + 2 }; f3 := func() { return f2() + 3 }; return f3()`,
		nil, Int(6))
	TestExpectRun(t, `f1 := func() { return 99; 100 }; return f1();`,
		nil, Int(99))
	TestExpectRun(t, `f1 := func() { return 99; return 100 }; return f1();`,
		nil, Int(99))
	TestExpectRun(t, `f1 := func() { return 33; }; f2 := func() { return f1 }; return f2()();`,
		nil, Int(33))
	TestExpectRun(t, `var one; one = func() { one = 1; return one }; return one()`,
		nil, Int(1))
	TestExpectRun(t, `three := func() { one := 1; two := 2; return one + two }; return three()`,
		nil, Int(3))
	TestExpectRun(t, `three := func() { one := 1; two := 2; return one + two }; seven := func() { three := 3; four := 4; return three + four }; return three() + seven()`,
		nil, Int(10))
	TestExpectRun(t, `
	foo1 := func() {
		foo := 50
		return foo
	}
	foo2 := func() {
		foo := 100
		return foo
	}
	return foo1() + foo2()`, nil, Int(150))
	TestExpectRun(t, `
	g := 50;
	minusOne := func() {
		n := 1;
		return g - n;
	};
	minusTwo := func() {
		n := 2;
		return g - n;
	};
	return minusOne() + minusTwo()`, nil, Int(97))
	TestExpectRun(t, `
	f1 := func() {
		f2 := func() { return 1; }
		return f2
	};
	return f1()()`, nil, Int(1))

	TestExpectRun(t, `
	f1 := func(a) { return a; };
	return f1(4)`, nil, Int(4))
	TestExpectRun(t, `
	f1 := func(a, b) { return a + b; };
	return f1(1, 2)`, nil, Int(3))

	TestExpectRun(t, `
	sum := func(a, b) {
		c := a + b;
		return c;
	};
	return sum(1, 2);`, nil, Int(3))

	TestExpectRun(t, `
	sum := func(a, b) {
		c := a + b;
		return c;
	};
	return sum(1, 2) + sum(3, 4);`, nil, Int(10))

	TestExpectRun(t, `
	sum := func(a, b) {
		c := a + b
		return c
	};
	outer := func() {
		return sum(1, 2) + sum(3, 4)
	};
	return outer();`, nil, Int(10))

	TestExpectRun(t, `
	g := 10;

	sum := func(a, b) {
		c := a + b;
		return c + g;
	}

	outer := func() {
		return sum(1, 2) + sum(3, 4) + g;
	}

	return outer() + g
	`, nil, Int(50))

	expectErrIs(t, `func() { return 1; }(1)`, nil, ErrWrongNumArguments)
	expectErrIs(t, `func(a) { return a; }()`, nil, ErrWrongNumArguments)
	expectErrIs(t, `func(a, b) { return a + b; }(1)`, nil, ErrWrongNumArguments)

	TestExpectRun(t, `
	f1 := func(a) {
		return func() { return a; };
	};
	f2 := f1(99);
	return f2()
	`, nil, Int(99))

	TestExpectRun(t, `
	f1 := func(a, b) {
		return func(c) { return a + b + c };
	};
	f2 := f1(1, 2);
	return f2(8);
	`, nil, Int(11))
	TestExpectRun(t, `
	f1 := func(a, b) {
		c := a + b;
		return func(d) { return c + d };
	};
	f2 := f1(1, 2);
	return f2(8);
	`, nil, Int(11))
	TestExpectRun(t, `
	f1 := func(a, b) {
		c := a + b;
		return func(d) {
			e := d + c;
			return func(f) { return e + f };
		}
	};
	f2 := f1(1, 2);
	f3 := f2(3);
	return f3(8);
	`, nil, Int(14))
	TestExpectRun(t, `
	a := 1;
	f1 := func(b) {
		return func(c) {
			return func(d) { return a + b + c + d }
		};
	};
	f2 := f1(2);
	f3 := f2(3);
	return f3(8);
	`, nil, Int(14))
	TestExpectRun(t, `
	f1 := func(a, b) {
		one := func() { return a; };
		two := func() { return b; };
		return func() { return one() + two(); }
	};
	f2 := f1(9, 90);
	return f2();
	`, nil, Int(99))

	// function recursion
	TestExpectRun(t, `
	var fib
	fib = func(x) {
		if x == 0 {
			return 0
		} else if x == 1 {
			return 1
		} else {
			return fib(x-1) + fib(x-2)
		}
	}
	return fib(15)`, nil, Int(610))

	// function recursion
	TestExpectRun(t, `
	return func() {
		var sum
		sum = func(x) {
			return x == 0 ? 0 : x + sum(x-1)
		}
		return sum(5)
	}()`, nil, Int(15))

	// closure and block scopes
	TestExpectRun(t, `
	var out
	func() {
		a := 10
		func() {
			b := 5
			if true {
				out = a + b
			}
		}()
	}(); return out`, nil, Int(15))
	TestExpectRun(t, `
	var out
	func() {
		a := 10
		b := func() { return 5 }
		func() {
			if b() {
				out = a + b()
			}
		}()
	}(); return out`, nil, Int(15))
	TestExpectRun(t, `
	var out
	func() {
		a := 10
		func() {
			b := func() { return 5 }
			func() {
				if true {
					out = a + b()
				}
			}()
		}()
	}(); return out`, nil, Int(15))

	TestExpectRun(t, `return func() {}()`, nil, Nil)
	TestExpectRun(t, `return func(v) { if v { return true } }(1)`, nil, True)
	TestExpectRun(t, `return func(v) { if v { return true } }(0)`, nil, Nil)
	TestExpectRun(t, `return func(v) { if v { } else { return true } }(1)`, nil, Nil)
	TestExpectRun(t, `return func(v) { if v { return } }(1)`, nil, Nil)
	TestExpectRun(t, `return func(v) { if v { return } }(0)`, nil, Nil)
	TestExpectRun(t, `return func(v) { if v { } else { return } }(1)`, nil, Nil)
	TestExpectRun(t, `return func(v) { for ;;v++ { if v == 3 { return true } } }(1)`, nil, True)
	TestExpectRun(t, `return func(v) { for ;;v++ { if v == 3 { break } } }(1)`, nil, Nil)
	TestExpectRun(t, `
	f := func() { return 2 }
	return (func() {
		f := f()
		return f
	})()`, nil, Int(2))
}

func TestBlocksScope(t *testing.T) {
	TestExpectRun(t, `
	var f
	if true {
		a := 1
		f = func() {
			a = 2
		}
	}
	b := 3
	f()
	return b`, nil, Int(3))

	TestExpectRun(t, `
	var out
	func() {
		f := nil
		if true {
			a := 10
			f = func() {
				a = 20
			}
		}
		b := 5
		f()
		out = b
	}()
	return out`, nil, Int(5))

	TestExpectRun(t, `
	f := nil
	if true {
		a := 1
		b := 2
		f = func() {
			a = 3
			b = 4
		}
	}
	c := 5
	d := 6
	f()
	return c + d`, nil, Int(11))

	TestExpectRun(t, `
	fn := nil
	if true {
		a := 1
		b := 2
		if true {
			c := 3
			d := 4
			fn = func() {
				a = 5
				b = 6
				c = 7
				d = 8
			}
		}
	}
	e := 9
	f := 10
	fn()
	return e + f`, nil, Int(19))

	TestExpectRun(t, `
	out := 0
	func() {
		for x in [1, 2, 3] {
			out += x
		}
	}()
	return out`, nil, Int(6))

	TestExpectRun(t, `
	out := 0
	for x in [1, 2, 3] {
		out += x
	}
	return out`, nil, Int(6))

	TestExpectRun(t, `
	out := 1
	x := func(){
		out := out // get free variable's value with the same name
		return out
	}()
	out = 2
	return x`, nil, Int(1))

	TestExpectRun(t, `
	out := 1
	func(){
		out := out // get free variable's value with the same name
		return func(){
			out = 3 // this refers to out in upper block, not 'out' at top
		}
	}()()
	return out`, nil, Int(1))

	// symbol must be defined before compiling right hand side otherwise not resolved.
	expectErrHas(t, `
	f := func() {
		f()
	}`, NewTestOpts().CompilerError(), `Compile Error: unresolved reference "f"`)
}

func TestVMIf(t *testing.T) {
	TestExpectRun(t, `var out; if (true) { out = 10 }; return out`,
		nil, Int(10))
	TestExpectRun(t, `var out; if (false) { out = 10 }; return out`,
		nil, Nil)
	TestExpectRun(t, `var out; if (false) { out = 10 } else { out = 20 }; return out`,
		nil, Int(20))
	TestExpectRun(t, `var out; if (1) { out = 10 }; return out`,
		nil, Int(10))
	TestExpectRun(t, `var out; if (0) { out = 10 } else { out = 20 }; return out`,
		nil, Int(20))
	TestExpectRun(t, `var out; if (1 < 2) { out = 10 }; return out`,
		nil, Int(10))
	TestExpectRun(t, `var out; if (1 > 2) { out = 10 }; return out`,
		nil, Nil)
	TestExpectRun(t, `var out; if (1 < 2) { out = 10 } else { out = 20 }; return out`,
		nil, Int(10))
	TestExpectRun(t, `var out; if (1 > 2) { out = 10 } else { out = 20 }; return out`,
		nil, Int(20))
	TestExpectRun(t, `var out; if (1 < 2) { out = 10 } else if (1 > 2) { out = 20 } else { out = 30 }; return out`,
		nil, Int(10))
	TestExpectRun(t, `var out; if (1 > 2) { out = 10 } else if (1 < 2) { out = 20 } else { out = 30 }; return out`,
		nil, Int(20))
	TestExpectRun(t, `var out; if (1 > 2) { out = 10 } else if (1 == 2) { out = 20 } else { out = 30 }; return out`,
		nil, Int(30))
	TestExpectRun(t, `var out; if (1 > 2) { out = 10 } else if (1 == 2) { out = 20 } else if (1 < 2) { out = 30 } else { out = 40 }; return out`,
		nil, Int(30))
	TestExpectRun(t, `var out; if (1 > 2) { out = 10 } else if (1 < 2) { out = 20; out = 21; out = 22 } else { out = 30 }; return out`,
		nil, Int(22))
	TestExpectRun(t, `var out; if (1 > 2) { out = 10 } else if (1 == 2) { out = 20 } else { out = 30; out = 31; out = 32}; return out`,
		nil, Int(32))
	TestExpectRun(t, `var out; if (1 > 2) { out = 10 } else if (1 < 2) { if (1 == 2) { out = 21 } else { out = 22 } } else { out = 30 }; return out`,
		nil, Int(22))
	TestExpectRun(t, `var out; if (1 > 2) { out = 10 } else if (1 < 2) { if (1 == 2) { out = 21 } else if (2 == 3) { out = 22 } else { out = 23 } } else { out = 30 }; return out`,
		nil, Int(23))
	TestExpectRun(t, `var out; if (1 > 2) { out = 10 } else if (1 == 2) { if (1 == 2) { out = 21 } else if (2 == 3) { out = 22 } else { out = 23 } } else { out = 30 }; return out`,
		nil, Int(30))
	TestExpectRun(t, `var out; if (1 > 2) { out = 10 } else if (1 == 2) { out = 20 } else { if (1 == 2) { out = 31 } else if (2 == 3) { out = 32 } else { out = 33 } }; return out`,
		nil, Int(33))

	TestExpectRun(t, `var out; if a:=0; a<1 { out = 10 }; return out`, nil, Int(10))
	TestExpectRun(t, `var out; a:=0; if a++; a==1 { out = 10 }; return out`, nil, Int(10))
	TestExpectRun(t, `
	var out
	func() {
		a := 1
		if a++; a > 1 {
			out = a
		}
	}()
	return out`, nil, Int(2))
	TestExpectRun(t, `
	var out
	func() {
		a := 1
		if a++; a == 1 {
			out = 10
		} else {
			out = 20
		}
	}()
	return out`, nil, Int(20))
	TestExpectRun(t, `
	var out
	func() {
		a := 1

		func() {
			if a++; a > 1 {
				a++
			}
		}()

		out = a
	}()
	return out`, nil, Int(3))

	// expression statement in init (should not leave objects on stack)
	TestExpectRun(t, `a := 1; if a; a { return a }`, nil, Int(1))
	TestExpectRun(t, `a := 1; if a + 4; a { return a }`, nil, Int(1))
}

func TestVMIncDec(t *testing.T) {
	TestExpectRun(t, `out := 0; out++; return out`, nil, Int(1))
	TestExpectRun(t, `out := 0; out--; return out`, nil, -Int(1))
	TestExpectRun(t, `a := 0; a++; out := a; return out`, nil, Int(1))
	TestExpectRun(t, `a := 0; a++; a--; out := a; return out`, nil, Int(0))

	// this seems strange but it works because 'a += b' is
	// translated into 'a = a + b' and string type takes other types for + operator.
	TestExpectRun(t, `a := "foo"; a++; return a`, nil, Str("foo1"))
	expectErrIs(t, `a := "foo"; a--`, nil, ErrType)
	expectErrHas(t, `a := "foo"; a--`, nil,
		`TypeError: unsupported operand types for '-': 'str' and 'int'`)

	expectErrHas(t, `a++`, NewTestOpts().CompilerError(),
		`Compile Error: unresolved reference "a"`) // not declared
	expectErrHas(t, `a--`, NewTestOpts().CompilerError(),
		`Compile Error: unresolved reference "a"`) // not declared
	expectErrHas(t, `4++`, NewTestOpts().CompilerError(),
		`Compile Error: unresolved reference ""`)
}

func TestVMInteger(t *testing.T) {
	TestExpectRun(t, `return 5`, nil, Int(5))
	TestExpectRun(t, `return 10`, nil, Int(10))
	TestExpectRun(t, `return -5`, nil, Int(-5))
	TestExpectRun(t, `return -10`, nil, Int(-10))
	TestExpectRun(t, `return 5 + 5 + 5 + 5 - 10`, nil, Int(10))
	TestExpectRun(t, `return 2 * 2 * 2 * 2 * 2`, nil, Int(32))
	TestExpectRun(t, `return -50 + 100 + -50`, nil, Int(0))
	TestExpectRun(t, `return 5 * 2 + 10`, nil, Int(20))
	TestExpectRun(t, `return 5 + 2 * 10`, nil, Int(25))
	TestExpectRun(t, `return 20 + 2 * -10`, nil, Int(0))
	TestExpectRun(t, `return 50 / 2 * 2 + 10`, nil, Int(60))
	TestExpectRun(t, `return 2 * (5 + 10)`, nil, Int(30))
	TestExpectRun(t, `return 3 * 3 * 3 + 10`, nil, Int(37))
	TestExpectRun(t, `return 3 * (3 * 3) + 10`, nil, Int(37))
	TestExpectRun(t, `return (5 + 10 * 2 + 15 /3) * 2 + -10`, nil, Int(50))
	TestExpectRun(t, `return 5 % 3`, nil, Int(2))
	TestExpectRun(t, `return 5 % 3 + 4`, nil, Int(6))
	TestExpectRun(t, `return +5`, nil, Int(5))
	TestExpectRun(t, `return +5 + -5`, nil, Int(0))

	TestExpectRun(t, `return 9 + '0'`, nil, Char('9'))
	TestExpectRun(t, `return '9' - 5`, nil, Char('4'))

	TestExpectRun(t, `return 5u`, nil, Uint(5))
	TestExpectRun(t, `return 10u`, nil, Uint(10))
	TestExpectRun(t, `return -5u`, nil, Uint(^uint64(0)-4))
	TestExpectRun(t, `return -10u`, nil, Uint(^uint64(0)-9))
	TestExpectRun(t, `return 5 + 5 + 5 + 5 - 10u`, nil, Uint(10))
	TestExpectRun(t, `return 2 * 2 * 2u * 2 * 2`, nil, Uint(32))
	TestExpectRun(t, `return -50 + 100u + -50`, nil, Uint(0))
	TestExpectRun(t, `return 5u * 2 + 10`, nil, Uint(20))
	TestExpectRun(t, `return 5 + 2u * 10`, nil, Uint(25))
	TestExpectRun(t, `return 20u + 2 * -10`, nil, Uint(0))
	TestExpectRun(t, `return 50 / 2u * 2 + 10`, nil, Uint(60))
	TestExpectRun(t, `return 2 * (5u + 10)`, nil, Uint(30))
	TestExpectRun(t, `return 3 * 3 * 3u + 10`, nil, Uint(37))
	TestExpectRun(t, `return 3u * (3 * 3) + 10`, nil, Uint(37))
	TestExpectRun(t, `return (5 + 10u * 2 + 15 /3) * 2 + -10`, nil, Uint(50))
	TestExpectRun(t, `return 5 % 3u`, nil, Uint(2))
	TestExpectRun(t, `return 5u % 3 + 4`, nil, Uint(6))
	TestExpectRun(t, `return 5 % 3 + 4u`, nil, Uint(6))
	TestExpectRun(t, `return +5u`, nil, Uint(5))
	TestExpectRun(t, `return +5u + -5`, nil, Uint(0))

	TestExpectRun(t, `return 9u + '0'`, nil, Char('9'))
	TestExpectRun(t, `return '9' - 5u`, nil, Char('4'))
}

func TestVMLogical(t *testing.T) {
	TestExpectRun(t, `true && true`, nil, Nil)
	TestExpectRun(t, `false || true`, nil, Nil)
	TestExpectRun(t, `return true && true`, nil, True)
	TestExpectRun(t, `return true && false`, nil, False)
	TestExpectRun(t, `return false && true`, nil, False)
	TestExpectRun(t, `return false && false`, nil, False)
	TestExpectRun(t, `return !true && true`, nil, False)
	TestExpectRun(t, `return !true && false`, nil, False)
	TestExpectRun(t, `return !false && true`, nil, True)
	TestExpectRun(t, `return !false && false`, nil, False)

	TestExpectRun(t, `return true || true`, nil, True)
	TestExpectRun(t, `return true || false`, nil, True)
	TestExpectRun(t, `return false || true`, nil, True)
	TestExpectRun(t, `return false || false`, nil, False)
	TestExpectRun(t, `return !true || true`, nil, True)
	TestExpectRun(t, `return !true || false`, nil, False)
	TestExpectRun(t, `return !false || true`, nil, True)
	TestExpectRun(t, `return !false || false`, nil, True)

	TestExpectRun(t, `return 1 && 2`, nil, Int(2))
	TestExpectRun(t, `return 1 || 2`, nil, Int(1))
	TestExpectRun(t, `return 1 && 0`, nil, Int(0))
	TestExpectRun(t, `return 1 || 0`, nil, Int(1))
	TestExpectRun(t, `return 1 && (0 || 2)`, nil, Int(2))
	TestExpectRun(t, `return 0 || (0 || 2)`, nil, Int(2))
	TestExpectRun(t, `return 0 || (0 && 2)`, nil, Int(0))
	TestExpectRun(t, `return 0 || (2 && 0)`, nil, Int(0))

	TestExpectRun(t, `var out; t:=func() {out = 3; return true}; f:=func() {out = 7; return false}; t() && f(); return out`,
		nil, Int(7))
	TestExpectRun(t, `var out; t:=func() {out = 3; return true}; f:=func() {out = 7; return false}; f() && t(); return out`,
		nil, Int(7))
	TestExpectRun(t, `var out; t:=func() {out = 3; return true}; f:=func() {out = 7; return false}; f() || t(); return out`,
		nil, Int(3))
	TestExpectRun(t, `var out; t:=func() {out = 3; return true}; f:=func() {out = 7; return false}; t() || f(); return out`,
		nil, Int(3))
	TestExpectRun(t, `var out; t:=func() {out = 3; return true}; f:=func() {out = 7; return false}; !t() && f(); return out`,
		nil, Int(3))
	TestExpectRun(t, `var out; t:=func() {out = 3; return true}; f:=func() {out = 7; return false}; !f() && t(); return out`,
		nil, Int(3))
	TestExpectRun(t, `var out; t:=func() {out = 3; return true}; f:=func() {out = 7; return false}; !f() || t(); return out`,
		nil, Int(7))
	TestExpectRun(t, `var out; t:=func() {out = 3; return true}; f:=func() {out = 7; return false}; !t() || f(); return out`,
		nil, Int(7))

	TestExpectRun(t, `false ?? true`, nil, Nil)
	TestExpectRun(t, `return true ?? true`, nil, True)
	TestExpectRun(t, `return nil ?? 1`, nil, Int(1))
	TestExpectRun(t, `return false ?? 1`, nil, False)
	TestExpectRun(t, `return nil ?? 1 ?? 2`, nil, Int(1))
	TestExpectRun(t, `return nil ?? nil ?? 2`, nil, Int(2))
	TestExpectRun(t, `var (called = false, f = func() {called = true;return 1}); return [f() ?? 2, called]`, nil, Array{Int(1), True})
	TestExpectRun(t, `var (c = "", f = func(v,r) {c += v;return r}); return [f("u",nil) ?? f("1",1) ?? f("2",2) , c]`, nil, Array{Int(1), Str("u1")})
	TestExpectRun(t, `var (c = "", f = func(v,r) {c += v;return r}); return [f("1",1) ?? f("2",2) , c]`, nil, Array{Int(1), Str("1")})
	TestExpectRun(t, `return nil ?? 0 || 2`, nil, Int(2))
	TestExpectRun(t, `return nil ?? 1 || 2`, nil, Int(1))
	TestExpectRun(t, `return 3 ?? 1 || 2`, nil, Int(3))
}

func TestVMMap(t *testing.T) {
	TestExpectRun(t, `
	return {
		one: 10 - 9,
		two: 1 + 1,
		three: 6 / 2,
	}`, nil, Dict{
		"one":   Int(1),
		"two":   Int(2),
		"three": Int(3),
	})

	TestExpectRun(t, `
	return {
		"one": 10 - 9,
		"two": 1 + 1,
		"three": 6 / 2,
	}`, nil, Dict{
		"one":   Int(1),
		"two":   Int(2),
		"three": Int(3),
	})

	TestExpectRun(t, `return {foo: 5}["foo"]`, nil, Int(5))
	TestExpectRun(t, `return {foo: 5}["bar"]`, nil, Nil)
	TestExpectRun(t, `key := "foo"; return {foo: 5}[key]`, nil, Int(5))
	TestExpectRun(t, `return {}["foo"]`, nil, Nil)

	TestExpectRun(t, `
	m := {
		foo: func(x) {
			return x * 2
		},
	}
	return m["foo"](2) + m["foo"](3)
	`, nil, Int(10))

	// map assignment is copy-by-reference
	TestExpectRun(t, `m1 := {k1: 1, k2: "foo"}; m2 := m1; m1.k1 = 5; return m2.k1`,
		nil, Int(5))
	TestExpectRun(t, `m1 := {k1: 1, k2: "foo"}; m2 := m1; m2.k1 = 3; return m1.k1`,
		nil, Int(3))
	TestExpectRun(t, `var out; func() { m1 := {k1: 1, k2: "foo"}; m2 := m1; m1.k1 = 5; out = m2.k1 }(); return out`,
		nil, Int(5))
	TestExpectRun(t, `var out; func() { m1 := {k1: 1, k2: "foo"}; m2 := m1; m2.k1 = 3; out = m1.k1 }(); return out`,
		nil, Int(3))
}

func TestVMSourceModules(t *testing.T) {
	TestExpectRun(t, `out := import("mod1"); return out`,
		NewTestOpts().Module("mod1", `return __name__, __file__, __is_module__`),
		Array{Str("mod1"), Str("source:mod1"), True})

	TestExpectRun(t, `return __name__, __file__, __is_module__`,
		nil,
		Array{Str(MainName), Str("file:" + MainName), False})

	// module return none
	TestExpectRun(t, `out := import("mod1"); return out`,
		NewTestOpts().Module("mod1", `fn := func() { return 5.0 }; a := 2`),
		Nil)

	// module return values
	TestExpectRun(t, `return import("mod1")`,
		NewTestOpts().Module("mod1", `return 5`), Int(5))
	TestExpectRun(t, `return import("mod1")`,
		NewTestOpts().Module("mod1", `return "foo"`), Str("foo"))

	// module return compound types
	TestExpectRun(t, `out := import("mod1"); return out`,
		NewTestOpts().Module("mod1", `return [1, 2, 3]`), Array{Int(1), Int(2), Int(3)})
	TestExpectRun(t, `out := import("mod1"); return out`,
		NewTestOpts().Module("mod1", `return {a: 1, b: 2}`), Dict{"a": Int(1), "b": Int(2)})

	// if returned values are not imumutable, they can be updated
	TestExpectRun(t, `m1 := import("mod1"); m1.a = 5; return m1`,
		NewTestOpts().Module("mod1", `return {a: 1, b: 2}`), Dict{"a": Int(5), "b": Int(2)})
	TestExpectRun(t, `m1 := import("mod1"); m1[1] = 5; return m1`,
		NewTestOpts().Module("mod1", `return [1, 2, 3]`), Array{Int(1), Int(5), Int(3)})
	// modules are evaluated once, calling in different scopes returns same object
	TestExpectRun(t, `
	m1 := import("mod1")
	m1.a = 5
	func(){
		m11 := import("mod1")
		m11.a = 6
	}()
	return m1`, NewTestOpts().Module("mod1", `return {a: 1, b: 2}`), Dict{"a": Int(6), "b": Int(2)})

	// module returning function
	TestExpectRun(t, `out := import("mod1")(); return out`,
		NewTestOpts().Module("mod1", `return func() { return 5.0 }`), Float(5.0))
	// returned function that reads module variable
	TestExpectRun(t, `out := import("mod1")(); return out`,
		NewTestOpts().Module("mod1", `a := 1.5; return func() { return a + 5.0 }`), Float(6.5))
	// returned function that reads local variable
	TestExpectRun(t, `out := import("mod1")(); return out`,
		NewTestOpts().Module("mod1", `return func() { a := 1.5; return a + 5.0 }`), Float(6.5))
	// returned function that reads free variables
	TestExpectRun(t, `out := import("mod1")(); return out`,
		NewTestOpts().Module("mod1", `return func() { a := 1.5; return func() { return a + 5.0 }() }`), Float(6.5))

	// recursive function in module
	TestExpectRun(t, `return import("mod1")`,
		NewTestOpts().Module("mod1", `
	var a
	a = func(x) {
		return x == 0 ? 0 : x + a(x-1)
	}
	return a(5)`), Int(15))

	TestExpectRun(t, `out := import("mod1"); return out`,
		NewTestOpts().Module("mod1", `
	return func() {
		var a
		a = func(x) {
			return x == 0 ? 0 : x + a(x-1)
		}
		return a(5)
	}()
	`), Int(15))

	// (main) -> mod1 -> mod2
	TestExpectRun(t, `return import("mod1")()`,
		NewTestOpts().Module("mod1", `return import("mod2")`).
			Module("mod2", `return func() { return 5.0 }`),
		Float(5.0))
	// (main) -> mod1 -> mod2
	//        -> mod2
	TestExpectRun(t, `import("mod1"); return import("mod2")()`,
		NewTestOpts().Module("mod1", `return import("mod2")`).
			Module("mod2", `return func() { return 5.0 }`),
		Float(5.0))
	// (main) -> mod1 -> mod2 -> mod3
	//        -> mod2 -> mod3
	TestExpectRun(t, `import("mod1"); return import("mod2")()`,
		NewTestOpts().Module("mod1", `return import("mod2")`).
			Module("mod2", `return import("mod3")`).
			Module("mod3", `return func() { return 5.0 }`),
		Float(5.0))

	// cyclic imports
	// (main) -> mod1 -> mod2 -> mod1
	expectErrHas(t, `import("mod1")`,
		NewTestOpts().Module("mod1", `import("mod2")`).
			Module("mod2", `import("mod1")`).CompilerError(),
		"Compile Error: cyclic module import: mod1\n\tat mod2:1:1")
	// (main) -> mod1 -> mod2 -> mod3 -> mod1
	expectErrHas(t, `import("mod1")`,
		NewTestOpts().Module("mod1", `import("mod2")`).
			Module("mod2", `import("mod3")`).
			Module("mod3", `import("mod1")`).CompilerError(),
		"Compile Error: cyclic module import: mod1\n\tat mod3:1:1")
	// (main) -> mod1 -> mod2 -> mod3 -> mod2
	expectErrHas(t, `import("mod1")`,
		NewTestOpts().Module("mod1", `import("mod2")`).
			Module("mod2", `import("mod3")`).
			Module("mod3", `import("mod2")`).CompilerError(),
		"Compile Error: cyclic module import: mod2\n\tat mod3:1:1")

	// unknown modules
	expectErrHas(t, `import("mod0")`,
		NewTestOpts().Module("mod1", `a := 5`).CompilerError(), "Compile Error: module 'mod0' not found")
	expectErrHas(t, `import("mod1")`,
		NewTestOpts().Module("mod1", `import("mod2")`).CompilerError(), "Compile Error: module 'mod2' not found")

	TestExpectRun(t, `m1 := import("mod1"); m1.a.b = 5; return m1.a.b`,
		NewTestOpts().Module("mod1", `return {a: {b: 3}}`), Int(5))

	// make sure module has same builtin functions
	TestExpectRun(t, `out := import("mod1"); return out`,
		NewTestOpts().Module("mod1", `return func() { return typeName(0) }()`), Str("int"))

	// module cannot access outer scope
	expectErrHas(t, `a := 5; import("mod1")`, NewTestOpts().Module("mod1", `return a`).CompilerError(),
		"Compile Error: unresolved reference \"a\"\n\tat mod1:1:8")

	// runtime error within modules
	expectErrIs(t, `
	a := 1;
	b := import("mod1");
	b(a)`,
		NewTestOpts().Module("mod1", `
	return func(a) {
	   a()
	}
	`), ErrNotCallable)

	// module with no return
	TestExpectRun(t, `out := import("mod0"); return out`,
		NewTestOpts().Module("mod0", ``), Nil)
	TestExpectRun(t, `out := import("mod0"); return out`,
		NewTestOpts().Module("mod0", `if 0 { return true }`), Nil)
	TestExpectRun(t, `out := import("mod0"); return out`,
		NewTestOpts().Module("mod0", `if 1 { } else { }`), Nil)
	TestExpectRun(t, `out := import("mod0"); return out`,
		NewTestOpts().Module("mod0", `for v:=0;;v++ { if v == 3 { break } }`), Nil)

	// importing same module multiple times returns same object
	TestExpectRun(t, `
	m1 := import("mod")
	m2 := import("mod")
	return m1 == m2
	`, NewTestOpts().Module("mod", `return { x: 1 }`), True)
	TestExpectRun(t, `
	m1 := import("mod")
	m2 := import("mod")
	m1.x = 2
	f := func() {
		return import("mod")
	}
	return [m1 == m2, m2 == import("mod"), m1 == f()]
	`, NewTestOpts().Module("mod", `return { x: 1 }`), Array{True, True, True})
	TestExpectRun(t, `
	mod2 := import("mod2")
	mod1 := import("mod1")
	return mod1.mod2 == mod2
	`, NewTestOpts().Module("mod1", `m2 := import("mod2"); m2.x = 2; return { x: 1, mod2: m2 }`).
		Module("mod2", "m := { x: 0 }; return m"), True)

}

func TestVMUnary(t *testing.T) {
	TestExpectRun(t, `!true`, nil, Nil)
	TestExpectRun(t, `true`, nil, Nil)
	TestExpectRun(t, `!false`, nil, Nil)
	TestExpectRun(t, `false`, nil, Nil)
	TestExpectRun(t, `return !false`, nil, True)
	TestExpectRun(t, `return !0`, nil, True)
	TestExpectRun(t, `return !5`, nil, False)
	TestExpectRun(t, `return !!true`, nil, True)
	TestExpectRun(t, `return !!false`, nil, False)
	TestExpectRun(t, `return !!5`, nil, True)

	TestExpectRun(t, `-1`, nil, Nil)
	TestExpectRun(t, `+1`, nil, Nil)
	TestExpectRun(t, `return -1`, nil, Int(-1))
	TestExpectRun(t, `return +1`, nil, Int(1))
	TestExpectRun(t, `return -0`, nil, Int(0))
	TestExpectRun(t, `return +0`, nil, Int(0))
	TestExpectRun(t, `return ^1`, nil, Int(^int64(1)))
	TestExpectRun(t, `return ^0`, nil, Int(^int64(0)))

	TestExpectRun(t, `-1u`, nil, Nil)
	TestExpectRun(t, `+1u`, nil, Nil)
	TestExpectRun(t, `return -1u`, nil, Uint(^uint64(0)))
	TestExpectRun(t, `return +1u`, nil, Uint(1))
	TestExpectRun(t, `return -0u`, nil, Uint(0))
	TestExpectRun(t, `return +0u`, nil, Uint(0))
	TestExpectRun(t, `return ^1u`, nil, Uint(^uint64(1)))
	TestExpectRun(t, `return ^0u`, nil, Uint(^uint64(0)))

	TestExpectRun(t, `-true`, nil, Nil)
	TestExpectRun(t, `+false`, nil, Nil)
	TestExpectRun(t, `return -true`, nil, Int(-1))
	TestExpectRun(t, `return +true`, nil, Int(1))
	TestExpectRun(t, `return -false`, nil, Int(0))
	TestExpectRun(t, `return +false`, nil, Int(0))
	TestExpectRun(t, `return ^true`, nil, Int(^int64(1)))
	TestExpectRun(t, `return ^false`, nil, Int(^int64(0)))

	TestExpectRun(t, `-'a'`, nil, Nil)
	TestExpectRun(t, `+'a'`, nil, Nil)
	TestExpectRun(t, `return -'a'`, nil, Int(-rune('a')))
	TestExpectRun(t, `return +'a'`, nil, Char('a'))
	TestExpectRun(t, `return ^'a'`, nil, Int(^rune('a')))

	TestExpectRun(t, `-1.0`, nil, Nil)
	TestExpectRun(t, `+1.0`, nil, Nil)
	TestExpectRun(t, `return -1.0`, nil, Float(-1.0))
	TestExpectRun(t, `return +1.0`, nil, Float(1.0))
	TestExpectRun(t, `return -0.0`, nil, Float(0.0))
	TestExpectRun(t, `return +0.0`, nil, Float(0.0))

	TestExpectRun(t, `return nil == nil`, nil, True)
	TestExpectRun(t, `return 1 == nil`, nil, False)
	TestExpectRun(t, `return nil != nil`, nil, False)
	TestExpectRun(t, `return 1 != nil`, nil, True)

	expectErrIs(t, `return ^1.0`, nil, ErrType)
	expectErrHas(t, `return ^1.0`, nil, `TypeError: invalid type for unary '^': 'float'`)
}

func TestVMScopes(t *testing.T) {
	// shadowed local variable
	TestExpectRun(t, `
	c := 5
	if a := 3; a {
		c := 6
	} else {
		c := 7
	}
	return c
	`, nil, Int(5))

	// shadowed function local variable
	TestExpectRun(t, `
	return func() {
		c := 5
		if a := 3; a {
			c := 6
		} else {
			c := 7
		}
		return c
	}()
	`, nil, Int(5))

	// 'b' is declared in 2 separate blocks
	TestExpectRun(t, `
	c := 5
	if a := 3; a {
		b := 8
		c = b
	} else {
		b := 9
		c = b
	}
	return c
	`, nil, Int(8))

	// shadowing inside for statement
	TestExpectRun(t, `
	a := 4
	b := 5
	for i:=0;i<3;i++ {
		b := 6
		for j:=0;j<2;j++ {
			b := 7
			a = i*j
		}
	}
	return a`, nil, Int(2))

	// shadowing variable declared in init statement
	TestExpectRun(t, `
	var out
	if a := 5; a {
		a := 6
		out = a
	}
	return out`, nil, Int(6))
	TestExpectRun(t, `
	var out
	a := 4
	if a := 5; a {
		a := 6
		out = a
	}
	return out`, nil, Int(6))
	TestExpectRun(t, `
	var out
	a := 4
	if a := 0; a {
		a := 6
		out = a
	} else {
		a := 7
		out = a
	}
	return out`, nil, Int(7))
	TestExpectRun(t, `
	var out
	a := 4
	if a := 0; a {
		out = a
	} else {
		out = a
	}
	return out`, nil, Int(0))
	// shadowing function level
	TestExpectRun(t, `
	a := 5
	func() {
		a := 6
		a = 7
	}()
	return a`, nil, Int(5))
	TestExpectRun(t, `
	a := 5
	func() {
		if a := 7; true {
			a = 8
		}
	}()
	return a`, nil, Int(5))
	TestExpectRun(t, `
	a := 5
	func() {
		if a := 7; true {
			a = 8
		}
	}()
	var (b, c, d)
	return [a, b, c, d]`, nil, Array{Int(5), Nil, Nil, Nil})
	TestExpectRun(t, `
	var f
	a := 5
	func() {
		if a := 7; true {
			f = func() {
				a = 8
			}
		}
	}()
	f()
	return a`, nil, Int(5))
	TestExpectRun(t, `
	if a := 7; false {
		a = 8
		return a
	} else {
		a = 9
		return a
	}`, nil, Int(9))
	TestExpectRun(t, `
	if a := 7; false {
		a = 8
		return a
	} else if false {
		a = 9
		return a
	} else {
		a = 10
		return a	
	}`, nil, Int(10))
	TestExpectRun(t, `var a;	if a == nil { return 10 } else { return 20 };`, nil, Int(10))
	TestExpectRun(t, `var a;	if a != nil { return 10 } else { return 20 };`, nil, Int(20))
}

func TestVMNullishSelector(t *testing.T) {
	TestExpectRun(t, `a := {b: 1}; return a?.b`, nil, Int(1))
	TestExpectRun(t, `a := {b: {c:{d:1}}}; return a?.b.c.d`, nil, Int(1))
	TestExpectRun(t, `a := {b: {c:{d:1}}}; k := "c"; return a?.b.(k).d`, nil, Int(1))
	TestExpectRun(t, `a := {b: {c:{d:1}}}; k := "x"; return a?.b.(k)?.d`, nil, Nil)
	TestExpectRun(t, `a := {b: {c:{d:{}}}}; return a?.b.c.d.e`, nil, Nil)
	TestExpectRun(t, `a := {b: {c:{d:{}}}}; return a?.b.c.d.e?.f.g`, nil, Nil)
	TestExpectRun(t, `a := {b: {c: {d: {e: {f: {g: 1} } } } } }; return a?.b?.c.d.e.f.g`, nil, Int(1))
	TestExpectRun(t, `a := {b: {c: {d: {e: {f: {g: 1} } } } } }; return a?.(""+"b")?.c.d?.e.f.g`, nil, Int(1))
	TestExpectRun(t, `a := {b: {c: {d: {e: {f: {g: 1} } } } } }; return (a[""+"b"])?.c.d?.e.f.g`, nil, Int(1))
	TestExpectRun(t, `var (a = {b: {c: {d: {e: {f: {g: 1} } } } } }, b); 
		return a?.("b").c.d.e.f.g,
               a?.("b"+"").c.d.e.f.g,
               a?.("" || "b").c.d.e.f.g,
               a?.("" || "b").c.d.(nil ?? "e").f.g,
               a?.("b" || "x").c.d.("e" ?? "z").f.g`, nil,
		Array{Int(1), Int(1), Int(1), Int(1), Int(1)})
	TestExpectRun(t, `a := {}; return (a[""+"b"])?.c.d?.e.f.g`, nil, Nil)
	TestExpectRun(t, `a := nil; return a?.b`, nil, Nil)
	TestExpectRun(t, `a := nil; return a?.b.c.d`, nil, Nil)
	TestExpectRun(t, `a := {}; return a?.b?.c.d`, nil, Nil)
}

func TestVMSelector(t *testing.T) {
	TestExpectRun(t, `a := {k1: 5, k2: "foo"}; return a.k1`, nil, Int(5))
	TestExpectRun(t, `a := {k1: 5, k2: "foo"}; return a.k2`, nil, Str("foo"))
	TestExpectRun(t, `a := {k1: 5, k2: "foo"}; return a.k3`, nil, Nil)

	TestExpectRun(t, `
	a := {
		b: {
			c: 4,
			a: false,
		},
		c: "foo bar",
	}
	_ := a.b.c
	return a.b.c`, nil, Int(4))

	TestExpectRun(t, `a := {b: 1, c: "foo"}; a.b = 2; return a.b`, nil, Int(2))
	TestExpectRun(t, `a := {b: 1, c: "foo"}; a.c = 2; return a.c`, nil, Int(2))
	TestExpectRun(t, `a := {b: {c: 1}}; a.b.c = 2; return a.b.c`, nil, Int(2))
	TestExpectRun(t, `a := {b: 1}; a.c = 2; return a`, nil, Dict{"b": Int(1), "c": Int(2)})
	TestExpectRun(t, `a := {b: {c: 1}}; a.b.d = 2; return a`, nil,
		Dict{"b": Dict{"c": Int(1), "d": Int(2)}})

	TestExpectRun(t, `return func() { a := {b: 1, c: "foo"}; a.b = 2; return a.b }()`, nil, Int(2))
	TestExpectRun(t, `return func() { a := {b: 1, c: "foo"}; a.c = 2; return a.c }()`, nil, Int(2))
	TestExpectRun(t, `return func() { a := {b: {c: 1}}; a.b.c = 2; return a.b.c }()`, nil, Int(2))
	TestExpectRun(t, `return func() { a := {b: 1}; a.c = 2; return a }()`, nil,
		Dict{"b": Int(1), "c": Int(2)})
	TestExpectRun(t, `return func() { a := {b: {c: 1}}; a.b.d = 2; return a }()`, nil,
		Dict{"b": Dict{"c": Int(1), "d": Int(2)}})

	TestExpectRun(t, `return func() { a := {b: 1, c: "foo"}; func() { a.b = 2 }(); return a.b }()`, nil, Int(2))
	TestExpectRun(t, `return func() { a := {b: 1, c: "foo"}; func() { a.c = 2 }(); return a.c }()`, nil, Int(2))
	TestExpectRun(t, `return func() { a := {b: {c: 1}}; func() { a.b.c = 2 }(); return a.b.c }()`, nil, Int(2))
	TestExpectRun(t, `return func() { a := {b: 1}; func() { a.c = 2 }(); return a }()`, nil,
		Dict{"b": Int(1), "c": Int(2)})
	TestExpectRun(t, `return func() { a := {b: {c: 1}}; func() { a.b.d = 2 }(); return a }()`,
		nil, Dict{"b": Dict{"c": Int(1), "d": Int(2)}})

	TestExpectRun(t, `
	a := {
		b: [1, 2, 3],
		c: {
			d: 8,
			e: "foo",
			f: [9, 8],
		},
	}
	return [a.b[2], a.c.d, a.c.e, a.c.f[1]]
	`, nil, Array{Int(3), Int(8), Str("foo"), Int(8)})

	TestExpectRun(t, `
	var out
	func() {
		a := [1, 2, 3]
		b := 9
		a[1] = b
		b = 7     // make sure a[1] has a COPY of value of 'b'
		out = a[1]
	}()
	return out
	`, nil, Int(9))

	expectErrIs(t, `a := {b: {c: 4,a: false,},c: "foo bar",};_ := a.x.c;return a.x.c`, nil, ErrNotIndexable)
	expectErrIs(t, `a := {b: {c: 1}}; a.d.c = 2`, nil, ErrNotIndexAssignable)
	expectErrIs(t, `a := [1, 2, 3]; a.b = 2`, nil, ErrType)
	expectErrIs(t, `a := "foo"; a.b = 2`, nil, ErrNotIndexAssignable)
	expectErrIs(t, `func() { a := {b: {c: 1}}; a.d.c = 2 }()`, nil, ErrNotIndexAssignable)
	expectErrIs(t, `func() { a := [1, 2, 3]; a.b = 2 }()`, nil, ErrType)
	expectErrIs(t, `func() { a := "foo"; a.b = 2 }()`, nil, ErrNotIndexAssignable)
}

func TestVMStackOverflow(t *testing.T) {
	expectErrIs(t, `var f; f = func() { return f() + 1 }; f()`, nil, ErrStackOverflow)
}

func TestVMString(t *testing.T) {
	TestExpectRun(t, `return "Hello World!"`, nil, Str("Hello World!"))
	TestExpectRun(t, `return "Hello" + " " + "World!"`, nil, Str("Hello World!"))

	TestExpectRun(t, `return "Hello" == "Hello"`, nil, True)
	TestExpectRun(t, `return "Hello" == "World"`, nil, False)
	TestExpectRun(t, `return "Hello" != "Hello"`, nil, False)
	TestExpectRun(t, `return "Hello" != "World"`, nil, True)

	TestExpectRun(t, `return "Hello" > "World"`, nil, False)
	TestExpectRun(t, `return "World" < "Hello"`, nil, False)
	TestExpectRun(t, `return "Hello" < "World"`, nil, True)
	TestExpectRun(t, `return "World" > "Hello"`, nil, True)
	TestExpectRun(t, `return "Hello" >= "World"`, nil, False)
	TestExpectRun(t, `return "Hello" <= "World"`, nil, True)
	TestExpectRun(t, `return "Hello" >= "Hello"`, nil, True)
	TestExpectRun(t, `return "World" <= "World"`, nil, True)

	// index operator
	str := "abcdef"
	strStr := `"abcdef"`
	strLen := 6
	for idx := 0; idx < strLen; idx++ {
		TestExpectRun(t, fmt.Sprintf("return %s[%d]", strStr, idx), nil, Int(str[idx]))
		TestExpectRun(t, fmt.Sprintf("return %s[0 + %d]", strStr, idx), nil, Int(str[idx]))
		TestExpectRun(t, fmt.Sprintf("return %s[1 + %d - 1]", strStr, idx), nil, Int(str[idx]))
		TestExpectRun(t, fmt.Sprintf("idx := %d; return %s[idx]", idx, strStr), nil, Int(str[idx]))
	}

	TestExpectRun(t, fmt.Sprintf("return %s[%d]", strStr, -1), nil, Int(str[5]))
	expectErrIs(t, fmt.Sprintf("%s[%d]", strStr, strLen), nil, ErrIndexOutOfBounds)

	// slice operator
	for low := 0; low < strLen; low++ {
		TestExpectRun(t, fmt.Sprintf("return %s[%d:%d]", strStr, low, low), nil, Str(""))
		for high := low; high <= strLen; high++ {
			TestExpectRun(t, fmt.Sprintf("return %s[%d:%d]", strStr, low, high),
				nil, Str(str[low:high]))
			TestExpectRun(t,
				fmt.Sprintf("return %s[0 + %d : 0 + %d]", strStr, low, high),
				nil, Str(str[low:high]))
			TestExpectRun(t,
				fmt.Sprintf("return %s[1 + %d - 1 : 1 + %d - 1]",
					strStr, low, high),
				nil, Str(str[low:high]))
			TestExpectRun(t,
				fmt.Sprintf("return %s[:%d]", strStr, high),
				nil, Str(str[:high]))
			TestExpectRun(t,
				fmt.Sprintf("return %s[%d:]", strStr, low),
				nil, Str(str[low:]))
		}
	}

	TestExpectRun(t, fmt.Sprintf("return %s[:]", strStr), nil, Str(str[:]))
	TestExpectRun(t, fmt.Sprintf("return %s[:]", strStr), nil, Str(str))
	TestExpectRun(t, fmt.Sprintf("return %s[%d:]", strStr, 0), nil, Str(str))
	TestExpectRun(t, fmt.Sprintf("return %s[:%d]", strStr, strLen), nil, Str(str))
	TestExpectRun(t, fmt.Sprintf("return %s[%d:%d]", strStr, 2, 2), nil, Str(""))
	TestExpectRun(t, fmt.Sprintf("return %s[%d:]", strStr, -1), nil, Str("f"))
	TestExpectRun(t, fmt.Sprintf("return %s[:%d]", strStr, -3), nil, Str("abc"))
	TestExpectRun(t, fmt.Sprintf("return %s[%d:%d]", strStr, -5, -3), nil, Str("bc"))
	TestExpectRun(t, fmt.Sprintf("return %s[%d:%d]", strStr, 0, -3), nil, Str("abc"))

	expectErrIs(t, fmt.Sprintf("%s[%d:]", strStr, strLen+1), nil, ErrInvalidIndex)
	expectErrIs(t, fmt.Sprintf("%s[%d:%d]", strStr, 2, 1), nil, ErrInvalidIndex)

	// string concatenation with other types
	TestExpectRun(t, `return "foo" + 1`, nil, Str("foo1"))
	// Float.ToString() returns the smallest number of digits
	// necessary such that ParseFloat will return f exactly.
	expectErrIs(t, `return 1 + "foo"`, nil, ErrType)
	TestExpectRun(t, `return "foo" + 1.0`, nil, Str("foo1")) // <- note '1' instead of '1.0'
	expectErrIs(t, `return 1.0 + "foo"`, nil, ErrType)
	TestExpectRun(t, `return "foo" + 1.5`, nil, Str("foo1.5"))
	expectErrIs(t, `return 1.5 + "foo"`, nil, ErrType)
	TestExpectRun(t, `return "foo" + true`, nil, Str("footrue"))
	expectErrIs(t, `return true + "foo"`, nil, ErrType)
	TestExpectRun(t, `return "foo" + 'X'`, nil, Str("fooX"))
	TestExpectRun(t, `return 'X' + "foo"`, nil, Str("Xfoo"))
	TestExpectRun(t, `return "foo" + error(5)`, nil, Str("fooerror: 5"))
	TestExpectRun(t, `return "foo" + nil`, nil, Str("foonil"))
	expectErrIs(t, `return nil + "foo"`, nil, ErrType)

	// Decimal.ToString() returns the smallest number of digits
	// necessary such that ParseDecimal will return f exactly.
	expectErrIs(t, `return 1d + "foo"`, nil, ErrType)
	TestExpectRun(t, `return "foo" + 1.0d`, nil, Str("foo1")) // <- note '1' instead of '1.0'
	expectErrIs(t, `return 1.0d + "foo"`, nil, ErrType)
	TestExpectRun(t, `return "foo" + 1.5d`, nil, Str("foo1.5"))
	expectErrIs(t, `return 1.5d + "foo"`, nil, ErrType)

	// array adds rhs object to the array
	TestExpectRun(t, `return [1, 2, 3] + "foo"`,
		nil, Array{Int(1), Int(2), Int(3), Str("foo")})
	// also works with "+=" operator
	TestExpectRun(t, `out := "foo"; out += 1.5; return out`, nil, Str("foo1.5"))
	expectErrHas(t, `"foo" - "bar"`,
		nil, `TypeError: unsupported operand types for '-': 'str' and 'str'`)

	TestExpectRun(t, `return ['abc', 'd\'e', 'f"g']`,
		NewTestOpts().CompileOptions(func(opts *CompileOptions) {
			opts.ScannerOptions.Mode |= parser.ScanCharAsString
		}), Array{Str("abc"), Str("d'e"), Str(`f"g`)})
}

func TestVMMultiParen(t *testing.T) {
	r := &MixedParams{
		Positional: Array{Int(1), Int(2), Int(3)},
		Named: KeyValueArray{
			&KeyValue{K: Str("a"), V: Int(4)},
			&KeyValue{K: Str("b"), V: Int(5)},
		},
	}
	TestExpectRun(t, fmt.Sprintf("return (1,*[2,3],a=4,**{b:5})"), nil, r)
	TestExpectRun(t, fmt.Sprintf("return (1,2,*[3],a=4,b=5)"), nil, r)
	TestExpectRun(t, fmt.Sprintf(`return (1,2,*[3],a=4,"b"=5)`), nil, r)
}

func TestVMTailCall(t *testing.T) {
	TestExpectRun(t, `
	f1 := (a) => a; return f1(*[1])`, nil, Int(1))
	TestExpectRun(t, `return (() => 5 + 10)()`, nil, Int(15))
	TestExpectRun(t, `return (() => {5 + 10})()`, nil, Int(15))
	TestExpectRun(t, `return ((b) => {a:=5; a + b})(10)`, nil, Int(15))
	TestExpectRun(t, `return ((b) => {a:=5; return a + b})(10)`, nil, Int(15))
	TestExpectRun(t, `return (() => {if 1 {2}})()`, nil, Nil)
	TestExpectRun(t, `return (() => {if 1 {2}; 3})()`, nil, Int(3))

	TestExpectRun(t, `
	var (fac, v1 = 100, v2 = 200)
	fac = func(n, *args,**na) {
		if n == 1 {
			return [args, __args__.array, na.dict, __named_args__.dict]
		}
		v1++
		v2++
		return fac(n-1, v1, v2,x1=v1,x2=v2)
	}
	return fac(10, 2, 3)`, nil, Array{
		Array{Int(109), Int(209)},
		Array{Array{Int(1)}, Array{Int(109), Int(209)}},
		Dict{"x1": Int(109), "x2": Int(209)},
		Dict{"x1": Int(109), "x2": Int(209)},
	})

	TestExpectRun(t, `
	var (fac, v1 = 100, v2 = 200)
	fac = func(n;**na) {
		if n == 1 {
			return [na.dict]
		}
		v1++
		v2++
		return fac(n-1,x1=v1,x2=v2)
	}
	return fac(10)`, nil, Array{Dict{"x1": Int(109), "x2": Int(209)}})

	TestExpectRun(t, `
	var (fac, v1 = 100, v2 = 200)
	fac = func(n,a,b;**na) {
		if n == 1 {
			return [a,b,na.dict]
		}
		v1++
		v2++
		return fac(n-1,v1,v2;x1=v1,x2=v2)
	}
	return fac(4,0,0,x3=2)`, nil, Array{Int(103), Int(203), Dict{"x1": Int(103), "x2": Int(203)}})

	TestExpectRun(t, `
	var (fac, v1 = 100, v2 = 200)
	fac = func(n,a,b;**namedArgs) {
		if n == 1 {
			return [a,b]
		}
		v1++
		v2++
		return fac(n-1,v1,v2)
	}
	return fac(4,0,0)`, nil, Array{Int(103), Int(203)})

	TestExpectRun(t, `
	var fac
	fac = func(n) {
		if n == 2 {
			return __args__[0]
		}
		return fac(n+1)
	}
	return fac(0)`, nil, Int(2))

	TestExpectRun(t, `
	var fac
	fac = func(n, a) {
		if n == 1 {
			return a
		}
		return fac(n-1, n*a)
	}
	return fac(5, 1)`, nil, Int(120))

	TestExpectRun(t, `
	var fac
	fac = func(n, a) {
		if n == 1 {
			return a
		}
		x := {foo: fac} // indirection for test
		return x.foo(n-1, n*a)
	}
	return fac(5, 1)`, nil, Int(120))

	TestExpectRun(t, `
	var fib
	fib = func(x, s) {
		if x == 0 {
			return 0 + s
		} else if x == 1 {
			return 1 + s
		}
		return fib(x-1, fib(x-2, s))
	}
	return fib(15, 0)`, nil, Int(610))

	TestExpectRun(t, `
	var fib
	fib = func(n, a, b) {
		if n == 0 {
			return a
		} else if n == 1 {
			return b
		}
		return fib(n-1, b, a + b)
	}
	return fib(15, 0, 1)`, nil, Int(610))

	TestExpectRun(t, `
	var (foo, out = 0)
	foo = func(a) {
		if a == 0 {
			return
		}
		out += a
		foo(a-1)
	}
	foo(10)
	return out`, nil, Int(55))

	TestExpectRun(t, `
	var f1
	f1 = func() {
		var f2
		f2 = func(n, s) {
			if n == 0 { return s }
			return f2(n-1, n + s)
		}
		return f2(5, 0)
	}
	return f1()`, nil, Int(15))

	// tail-call replacing loop
	// without tail-call optimization, this code will cause stack overflow
	TestExpectRun(t, `
	var iter
	iter = func(n, max) {
		if n == max {
			return n
		}
		return iter(n+1, max)
	}
	return iter(0, 9999)`, nil, Int(9999))

	TestExpectRun(t, `
	var (iter, c = 0)
	iter = func(n, max) {
		if n == max {
			return
		}
		c++
		iter(n+1, max)
	}
	iter(0, 9999)
	return c`, nil, Int(9999))
}

func TestVMTailCallFreeVars(t *testing.T) {
	TestExpectRun(t, `
	var out
	func() {
		a := 10
		f2 := 0
		f2 = func(n, s) {
			if n == 0 {
				return s + a
			}
			return f2(n-1, n+s)
		}
		out = f2(5, 0)
	}()
	return out`, nil, Int(25))
}

func TestVMCall(t *testing.T) {
	TestExpectRun(t, `f := func() {}; return f()`, nil, Nil)
	TestExpectRun(t, `func f (a) { return a; }; return f(1)`, nil, Int(1))
	TestExpectRun(t, `f := func(a) { return a; }; return f(1)`, nil, Int(1))
	TestExpectRun(t, `f := func(a, b) { return [a, b]; }; return f(1, 2)`, nil, Array{Int(1), Int(2)})
	expectErrIs(t, `f := func() { return; }; return f(1)`, nil, ErrWrongNumArguments)
	expectErrHas(t, `f := func() { return; }; return f(1)`, nil, `want=0 got=1`)

	TestExpectRun(t, `f := func(*a) { return a; }; return f()`, nil, Array{})
	TestExpectRun(t, `f := func(*a) { return a; }; return f(1)`, nil, Array{Int(1)})
	TestExpectRun(t, `f := func(*a) { return a; }; return f(1, 2)`, nil, Array{Int(1), Int(2)})
	expectErrIs(t, `f := func(a, *b) { return a; }; return f()`, nil, ErrWrongNumArguments)
	expectErrHas(t, `f := func(a, *b) { return a; }; return f()`, nil, `want>=1 got=0`)
	expectErrHas(t, `f := func(a, b, *c) { return a; }; return f(1)`, nil, `want>=2 got=1`)

	TestExpectRun(t, `f := func(a, *b) { return a; }; return f(1, 2)`, nil, Int(1))
	TestExpectRun(t, `f := func(a, *b) { return b; }; return f(1)`, nil, Array{})
	TestExpectRun(t, `f := func(a, *b) { return b; }; return f(1, 2)`, nil, Array{Int(2)})
	TestExpectRun(t, `f := func(a, *b) { return b; }; return f(1, 2, 3)`, nil, Array{Int(2), Int(3)})

	TestExpectRun(t, `f := func(a, b, *c) { return a; }; return f(1, 2)`, nil, Int(1))
	TestExpectRun(t, `f := func(a, b, *c) { return b; }; return f(1, 2)`, nil, Int(2))
	TestExpectRun(t, `f := func(a, b, *c) { return c; }; return f(1, 2)`, nil, Array{})
	TestExpectRun(t, `f := func(a, b, *c) { return c; }; return f(1, 2, 3)`, nil, Array{Int(3)})
	TestExpectRun(t, `f := func(a, b, *c) { return c; }; return f(1, 2, 3, 4)`, nil, Array{Int(3), Int(4)})

	TestExpectRun(t, `f := func(a) { return a; }; return f(*[1])`, nil, Int(1))
	TestExpectRun(t, `f := func(a, b) { return [a, b]; }; return f(*[1, 2])`, nil, Array{Int(1), Int(2)})
	TestExpectRun(t, `f := func(a, b) { return [a, b]; }; return f(1, *[2])`, nil, Array{Int(1), Int(2)})
	TestExpectRun(t, `f := func() { return; }; return f(*[])`, nil, Nil)

	TestExpectRun(t, `f := func(a, *b) { return a; }; return f(1, *[2])`, nil, Int(1))
	TestExpectRun(t, `f := func(a, *b) { return b; }; return f(1, *[2])`, nil, Array{Int(2)})
	TestExpectRun(t, `f := func(a, *b) { return b; }; return f(1, *[2, 3])`, nil, Array{Int(2), Int(3)})
	TestExpectRun(t, `f := func(a, *b) { return a; }; return f(*[1, 2, 3])`, nil, Int(1))
	TestExpectRun(t, `f := func(a, *b) { return b; }; return f(*[1, 2, 3])`, nil, Array{Int(2), Int(3)})

	TestExpectRun(t, `f := func(*a) { return a; }; return f(1, 2, *[3, 4])`, nil, Array{Int(1), Int(2), Int(3), Int(4)})
	TestExpectRun(t, `f := func(a, *b) { return a; }; return f(1, 2, *[3, 4])`, nil, Int(1))
	TestExpectRun(t, `f := func(a, *b) { return b; }; return f(1, 2, *[3, 4])`, nil, Array{Int(2), Int(3), Int(4)})
	TestExpectRun(t, `f := func(a, *b) { return b; }; return f(1, 2, *[])`, nil, Array{Int(2)})
	// if args and params match, 'c' points to the given array not nil.
	TestExpectRun(t, `f := func(a, b, *c) { return c; }; return f(1, 2, *[])`, nil, Array{})

	expectErrIs(t, `f := func(a, *b) { return a; }; return f(*[])`, nil, ErrWrongNumArguments)
	expectErrHas(t, `f := func(a, *b) { return a; }; return f(*[])`, nil, `want>=1 got=0`)
	expectErrHas(t, `f := func(a, b, *c) { return a; }; return f(*[1])`, nil, `want>=2 got=1`)
	expectErrHas(t, `f := func(a, b, *c) { return a; }; return f(1, *[])`, nil, `want>=2 got=1`)
	expectErrHas(t, `f := func(a, b, c, *d) { return a; }; return f(1, *[])`, nil, `want>=3 got=1`)
	expectErrIs(t, `f := func(a, b, c, *d) { return a; }; return f(1, *[2])`, nil, ErrWrongNumArguments)
	expectErrHas(t, `f := func(a, b, c, *d) { return a; }; return f(1, *[2])`, nil, `want>=3 got=2`)

	expectErrIs(t, `f := func(a, b) { return a; }; return f(1, *[2, 3])`, nil, ErrWrongNumArguments)
	expectErrHas(t, `f := func(a, b) { return a; }; return f(1, 2, *[3])`, nil, `want=2 got=3`)
	expectErrHas(t, `f := func(a, b) { return a; }; return f(1, *[2, 3])`, nil, `want=2 got=3`)
	expectErrHas(t, `f := func(a, b) { return a; }; return f(*[1, 2, 3])`, nil, `want=2 got=3`)

	TestExpectRun(t, `f := func(a, *b) { var x; return [x, a]; }; return f(1, 2)`, nil, Array{Nil, Int(1)})
	TestExpectRun(t, `f := func(a, *b) { var x; return [x, b]; }; return f(1, 2)`, nil, Array{Nil, Array{Int(2)}})

	TestExpectRun(t, `global f; return f()`, NewTestOpts().Globals(
		Dict{"f": &Function{Value: func(c Call) (Object, error) {
			return Int(c.Args.Length()), nil
		}}}), Int(0))
	TestExpectRun(t, `global f; return f(1)`, NewTestOpts().Globals(
		Dict{"f": &Function{Value: func(c Call) (Object, error) {
			return Array{Int(c.Args.Length()), c.Args.Get(0)}, nil
		}}}), Array{Int(1), Int(1)})
	TestExpectRun(t, `global f; return f(1, 2)`, NewTestOpts().Globals(
		Dict{"f": &Function{Value: func(c Call) (Object, error) {
			return Array{Int(c.Args.Length()), c.Args.Get(0), c.Args.Get(1)}, nil
		}}}), Array{Int(2), Int(1), Int(2)})
	TestExpectRun(t, `global f; return f(*[])`, NewTestOpts().Globals(
		Dict{"f": &Function{Value: func(c Call) (Object, error) {
			return Array{Int(c.Args.Length()), c.Args.Values()}, nil
		}}}), Array{Int(0), Array{}})
	TestExpectRun(t, `global f; return f(*[1])`, NewTestOpts().Globals(
		Dict{"f": &Function{Value: func(c Call) (Object, error) {
			return Array{Int(c.Args.Length()), c.Args.Values()}, nil
		}}}), Array{Int(1), Array{Int(1)}})
	TestExpectRun(t, `global f; return f(1, *[])`, NewTestOpts().Globals(
		Dict{"f": &Function{Value: func(c Call) (Object, error) {
			return Array{Int(c.Args.Length()), c.Args.Values()}, nil
		}}}), Array{Int(1), Array{Int(1)}})
	TestExpectRun(t, `global f; return f(1, *[2])`, NewTestOpts().Globals(
		Dict{"f": &Function{Value: func(c Call) (Object, error) {
			return Array{Int(c.Args.Length()), c.Args.Values()}, nil
		}}}), Array{Int(2), Array{Int(1), Int(2)}})
	TestExpectRun(t, `global f; return f(1, 2, *[3])`, NewTestOpts().Globals(
		Dict{"f": &Function{Value: func(c Call) (Object, error) {
			return Array{Int(c.Args.Length()), c.Args.Values()}, nil
		}}}), Array{Int(3), Array{Int(1), Int(2), Int(3)}})
	TestExpectRun(t, `global f; return f(1, 2, 3)`, NewTestOpts().Globals(
		Dict{"f": &Function{Value: func(c Call) (Object, error) {
			return Array{Int(c.Args.Length()), c.Args.Values()}, nil
		}}}), Array{Int(3), Array{Int(1), Int(2), Int(3)}})

	expectErrIs(t, `global f; return f()`, NewTestOpts().Globals(
		Dict{"f": &Function{Value: func(c Call) (Object, error) {
			return nil, ErrWrongNumArguments
		}}}), ErrWrongNumArguments)
	expectErrIs(t, `global f; return f()`, NewTestOpts().Globals(Dict{"f": Nil}),
		ErrNotCallable)

	TestExpectRun(t, `a := { b: func(x) { return x + 2 } }; return a.b(5)`, nil, Int(7))
	TestExpectRun(t, `a := { b: { c: func(x) { return x + 2 } } }; return a.b.c(5)`,
		nil, Int(7))
	TestExpectRun(t, `a := { b: { c: func(x) { return x + 2 } } }; return a["b"].c(5)`,
		nil, Int(7))
	expectErrIs(t, `
	a := 1
	b := func(a, c) {
	c(a)
	}
	c := func(a) {
	a()
	}
	b(a, c)
	`, nil, ErrNotCallable)

	TestExpectRun(t, `return {a: str(*[0])}`, nil, Dict{"a": Str("0")})
	TestExpectRun(t, `return {a: str([0])}`, nil, Dict{"a": Str("[0]")})
	TestExpectRun(t, `return {a: bytes(*repeat([0], 4096))}`,
		nil, Dict{"a": make(Bytes, 4096)})

	TestExpectRun(t, `return BUILTIN_VAR`,
		NewTestOpts().Builtins(map[string]Object{
			"BUILTIN_VAR": Int(100),
		}), Int(100))
}

func TestVMCallCompiledFunction(t *testing.T) {
	TestExpectRun(t, `
	f := func(*argv, **nav) {
		return [copy(__args__.values), __named_args__.dict, str(__callee__)]
	}
	return f(1,2,3, *[8,9],na0=4,na1=5)`, nil,
		Array{
			Array{Int(1), Int(2), Int(3), Int(8), Int(9)},
			Dict{"na0": Int(4), "na1": Int(5)},
			Str(ReprQuote("compiledFunction #2(*argv, **nav)")),
		})

	script := `
	var v = 0
	return {
		"add": func(x) {
			v+=x
			return v
		},
		"sub": func(x) {
			v-=x
			return v
		},
	}
	`
	c, err := Compile([]byte(script), CompileOptions{})
	if err != nil {
		t.Fatal(err)
	}
	vm := NewVM(c)
	f, err := vm.Run(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	// locals := vm.GetLocals(nil)
	// t.Log(f)
	require.Contains(t, f.(Dict), "add")
	require.Contains(t, f.(Dict), "sub")
	add := f.(Dict)["add"].(*CompiledFunction)
	ret, err := vm.RunCompiledFunction(add, Int(10))
	if err != nil {
		t.Fatal(err)
	}
	// t.Log(ret)
	require.Equal(t, Int(10), ret.(Int))

	ret, err = vm.RunCompiledFunction(add, Int(10))
	if err != nil {
		t.Fatal(err)
	}
	// t.Log(ret)
	require.Equal(t, Int(20), ret.(Int))

	sub := f.(Dict)["sub"].(*CompiledFunction)
	ret, err = vm.RunCompiledFunction(sub, Int(1))
	if err != nil {
		t.Fatal(err)
	}
	// t.Log(ret)
	require.Equal(t, Int(19), ret.(Int))

	ret, err = vm.RunCompiledFunction(sub, Int(1))
	if err != nil {
		t.Fatal(err)
	}
	// t.Log(ret)
	require.Equal(t, Int(18), ret.(Int))
	// for i := range locals {
	// 	fmt.Printf("%#v\n", locals[i])
	// 	fmt.Printf("%#v\n", *locals[i].(*ObjectPtr).Value)
	// }

	TestExpectRun(t, `
	f := func(arg0, arg1, *varg, na0=100, **na) {
		return [arg0, arg1, copy(varg), na0, na.dict]
	}
	return f(1,2,3,na0=4,na1=5)`, nil,
		Array{Int(1), Int(2), Array{Int(3)}, Int(4), Dict{"na1": Int(5)}})
}

func TestVMPipe(t *testing.T) {
	TestExpectRun(t, `param arr; v := arr.|map((v, _) => v+1;update).|values.|collect; return [v, str(v)]`, NewTestOpts().Init(func(opts *TestOpts, expect Object) (*TestOpts, Object) {
		ex := Array{Int(1)}
		opts.Args(ex)
		return opts, Array{ex, Str("[2]")}
	}), Nil)

	TestExpectRun(t, `inc := (arr) => arr.|map(func(v, _) => (v+1); update) 
	return [1,2,3].|inc.|reduce((sum, v,_) => (sum+v), 0).|(v) => (v*(2)).|((v) => [v])`, nil,
		Array{Int(18)})

	TestExpectRun(t, `
	first := (arr) => arr[0]
	return [10].|first()`, nil,
		Int(10))

	TestExpectRun(t, `
	f := (v) => v*2
	return (10).|f`, nil,
		Int(20))

	TestExpectRun(t, `
	first := (arr) => arr[0]
	return [10].|first()`, nil,
		Int(10))

	TestExpectRun(t, `
	first := (arr, v) => arr[0] + v
	return [10].|first(2)`, nil,
		Int(12))

	TestExpectRun(t, `
	return [10].|{a:{b:(arr, v) => arr[0] + v}}.a.b(2)`, nil,
		Int(12))

	TestExpectRun(t, `
	return (10).|{a:{b:(v) => v*2}}.a.b`, nil,
		Int(20))
}

func TestVMCallWithNamedArgs(t *testing.T) {
	TestExpectRun(t, `return func(;a=2) { return a }(;"a"=3)`, nil, Int(3))
	TestExpectRun(t, `return func(;a=2) { return a }(;a=3)`, nil, Int(3))
	TestExpectRun(t, `return func(x;a=2) { return x+a }(1)`, nil, Int(3))
	TestExpectRun(t, `return func(x;a=2,b=3) { return x+a+b }(1)`, nil, Int(6))
	TestExpectRun(t, `return func(x;a=2) { return x+a }(1;a=3)`, nil, Int(4))
	TestExpectRun(t, `return func(x;a=2) { return x+a }(1;a=3,**{"a":4})`, nil, Int(4))
	TestExpectRun(t, `return func(x;a=2) { return x+a }(1;a=4,**{"a":90})`, nil, Int(5))
	TestExpectRun(t, `return func(x;a=2) { return x+a }(1;a=3,**{})`, nil, Int(4))
	TestExpectRun(t, `return func(*z,a="A", b="B", **c) { return [z,a,b,c.dict] }(5,*[6,7,8,9],**{"a":"na", "b":"nb", "c":"C", "d":"D"})`,
		nil, Array{Array{Int(5), Int(6), Int(7), Int(8), Int(9)}, Str("na"), Str("nb"), Dict{"c": Str("C"), "d": Str("D")}})
	TestExpectRun(t, `return func(*z;a=false, b="B", **c) { return [a,b,c.dict] }(5,*[6,7,8,9];a=true,**{"a":"na", "b":"nb", "c":"C", "d":"D"})`,
		nil, Array{True, Str("nb"), Dict{"c": Str("C"), "d": Str("D")}})
	TestExpectRun(t, `return func(*z;a=false, b="B", **c) { return [a,b,c.dict] }(5,*[6,7,8,9];a=true,**{"b":"nb", "c":"C", "d":"D"})`,
		nil, Array{True, Str("nb"), Dict{"c": Str("C"), "d": Str("D")}})
	TestExpectRun(t, `return func(x, y, *z;a="A", b="B", **c) { return [x,y,z,a,b,c.dict] }(5,*[6,7,8,9];**{"a":"na", "b":"nb", "c":"C", "d":"D"})`,
		nil, Array{Int(5), Int(6), Array{Int(7), Int(8), Int(9)}, Str("na"), Str("nb"), Dict{"c": Str("C"), "d": Str("D")}})
	TestExpectRun(t, `return func(x, y, *z;a="A", b="B", **c) { return [x,y,z,a,b,c.dict] }(5,*[6,7,8,9],**{})`,
		nil, Array{Int(5), Int(6), Array{Int(7), Int(8), Int(9)}, Str("A"), Str("B"), Dict{}})
	TestExpectRun(t, `truncate := func(text; limit=3) {if len(text) > limit { return text[:limit]+"..." }; return text}
return [ truncate("abcd"), truncate("abc"), truncate("ab"),	truncate("abcd";limit=2) ]
`, nil, Array{Str("abc..."), Str("abc"), Str("ab"), Str("ab...")})
	TestExpectRun(t, `
f1 := func(b=1,**c) { return c }
f2 := func(a=5,**c) {
	z := f1(;flag, **c)
	return str([c,z])
}
return f2(;a=1,b=2,c=3,d=4,e=5)
`, nil, Str("[(;b=2, c=3, d=4, e=5), (;flag, c=3, d=4, e=5)]"))

	TestExpectRun(t, `return func(a=2) { return a }(**(;a=3))`, nil, Int(3))
	TestExpectRun(t, `f := func(**kw){return kw};return str(f(;x=1,x=2))`, nil, Str("(;x=1, x=2)"))
	TestExpectRun(t, `f := func(**kw){return kw};return str(f(;x=2).dict)`, nil, Str(`{x: 2}`))
	TestExpectRun(t, `f := func(**kw){return kw};return str(f(;x=1,x=2).array)`, nil, Str(`(;x=1, x=2)`))
	TestExpectRun(t, `f := func(;x=1,**kw){return kw};return str(f(;x=1,x=2).ready)`, nil, Str(`(;x=1, x=2)`))
	TestExpectRun(t, `f := func(;x=1,**kw){return kw};return str(f(;x=1,x=2).readyNames)`, nil, Str(`["x"]`))
	TestExpectRun(t, `f := func(;x=1,**kw){return [1, kw]};_, ret := f(;x=1,x=2); return str(ret.ready)`, nil, Str(`(;x=1, x=2)`))
	TestExpectRun(t, `f := func(;x=1,**kw){return kw};return str(f(;x=1,x=2,y=3,**{x:4}).src)`, nil, Str(`[(;x=1, x=2, y=3), (;x=4)]`))
	TestExpectRun(t, `f := func(**kw){return kw};return str(f(;**(func() {return [[100=1],["x"=2]]})()))`, nil, Str(`(;100=1, x=2)`))
	TestExpectRun(t, `f := func(**kw){return kw};return str(f(;**(;100=1,x=2,flag,x=4,"a b"=7)))`, nil, Str(`(;100=1, x=2, flag, x=4, "a b"=7)`))
	TestExpectRun(t, `f := func(**kw){return kw};return str(f(;"x y"=2,"user.name"="the user",abc="de"))`, nil, Str(`(;"x y"=2, "user.name"="the user", abc="de")`))
	TestExpectRun(t, `f := func(**kw){return __named_args__};return str(f(;"x y"=2,"user.name"="the user",abc="de"))`, nil, Str(`(;"x y"=2, "user.name"="the user", abc="de")`))

	TestExpectRun(t, `return func(;a int=2) { return a }()`, nil, Int(2))
	TestExpectRun(t, `return func(;a int=2) { return a }(;a=3)`, nil, Int(3))
	TestExpectRun(t, `f := func(;a int|uint=2) { return str(typeof(a)) }; return f(;a=1), f(;a=1u)`, nil,
		Array{Str("‹builtinType int›"), Str("‹builtinType uint›")})
	expectErrHas(t, `func(;a int=2) { return a }(;a="3")`, nil, "invalid type for named argument 'a': expected int, found str")
	expectErrHas(t, `func(;a int|uint=2) { return a }(;a="3")`, nil, "invalid type for named argument 'a': expected int|uint, found str")
}

func TestVMClosure(t *testing.T) {
	TestExpectRun(t, `
	param arg0
	var (f, y=0)
	f = func(x) {
		if x<=0{
			return 0
		}
		y++
		return f(x-1)
	}
	f(arg0)
	return y`, NewTestOpts().Args(Int(100)), Int(100))

	TestExpectRun(t, `
	x:=func(){
		a:=10
		g:=func(){
			b:=20
			y:=func(){
				b=21
				a=11
			}()
			return b
		}
		t := g()
		return [a, t]
	}
	return x()`, nil, Array{Int(11), Int(21)})

	TestExpectRun(t, `
	var f
	for i:=0; i<3; i++ {
		f = func(){
			return i
		}
	}
	return f()
	`, nil, Int(3))

	TestExpectRun(t, `
	fns :=  []
	for i:=0; i<3; i++ {
		i := i
		fns = append(fns, func(){
			return i
		})
	}

	ret := []
	for f in fns {
		ret = append(ret, f())
	}
	return ret
	`, nil, Array{Int(0), Int(1), Int(2)})
}

func TestVMCallFunctionWithNamedArgs(t *testing.T) {
	scr := `
global fn
return [
	fn(),
	fn(1),
	fn(1,2),
	fn(1,2,*[3,4]),
	fn(*[3,4]; **{a:5}),
	fn(**{a:5}),
	fn(1,2,*[3,4]; **{a:5}),
	fn(1,2,*[3,4]; a=5, **{b:6}),
]
`
	TestExpectRun(t, scr,
		NewTestOpts().Globals(Dict{"fn": &Function{
			Name: "fn",
			Value: func(c Call) (Object, error) {
				args := c.Args.Values()
				nargs := c.NamedArgs.Dict()
				if args == nil {
					args = Array{}
				}
				if nargs == nil {
					nargs = Dict{}
				}
				return Array{args, nargs}, nil
			},
		}}),
		Array{
			Array{Array{}, Dict{}},
			Array{Array{Int(1)}, Dict{}},
			Array{Array{Int(1), Int(2)}, Dict{}},
			Array{Array{Int(1), Int(2), Int(3), Int(4)}, Dict{}},
			Array{Array{Int(3), Int(4)}, Dict{"a": Int(5)}},
			Array{Array{}, Dict{"a": Int(5)}},
			Array{Array{Int(1), Int(2), Int(3), Int(4)}, Dict{"a": Int(5)}},
			Array{Array{Int(1), Int(2), Int(3), Int(4)}, Dict{"a": Int(5), "b": Int(6)}},
		},
	)
}

func TestVMCallCallableObjectWithNamedArgs(t *testing.T) {
	scr := `
global fn
return [
	fn(),
	fn(1),
	fn(1,2),
	fn(1,2,*[3,4]),
	fn(*[3,4]; **{a:5}),
	fn(**{a:5}),
	fn(1,2,*[3,4]; **{a:5}),
	fn(1,2,*[3,4]; a=5, **{b:6}),
	fn(*[1,2], **{a:3}),
	fn(*[1,2], a=3, **{b:4}),
]
`
	TestExpectRun(t, scr,
		NewTestOpts().Globals(Dict{"fn": &callerObject{}}),
		Array{
			Array{Array{}, Dict{}},
			Array{Array{Int(1)}, Dict{}},
			Array{Array{Int(1), Int(2)}, Dict{}},
			Array{Array{Int(1), Int(2), Int(3), Int(4)}, Dict{}},
			Array{Array{Int(3), Int(4)}, Dict{"a": Int(5)}},
			Array{Array{}, Dict{"a": Int(5)}},
			Array{Array{Int(1), Int(2), Int(3), Int(4)}, Dict{"a": Int(5)}},
			Array{Array{Int(1), Int(2), Int(3), Int(4)}, Dict{"a": Int(5), "b": Int(6)}},
			Array{Array{Int(1), Int(2)}, Dict{"a": Int(3)}},
			Array{Array{Int(1), Int(2)}, Dict{"a": Int(3), "b": Int(4)}},
		},
	)
}

func TestVMMixedOutput(t *testing.T) {
	TestExpectRun(t, `# gad: mixed
{%obstart() -%}
a
{%- = 2 -%}
b
{%- return str(obend())%}
`,
		NewTestOpts(),
		Str("a2b"),
	)

	TestExpectRun(t, `# gad: mixed
{%obstart() -%}
a
{%- obstart() -%}
{%- = 2 -%}
b
{%- flush(); obend() -%}
{%- return str(obend())%}
`,
		NewTestOpts(),
		Str("a2b"),
	)
	exprToText := ExprToTextOverride(
		"expr2text",
		func(vm *VM, w Writer, old func(w Writer, expr Object) (n Int, err error), expr Object) (n Int, err error) {
			var b strings.Builder
			n, err = old(NewWriter(&b), expr)
			w.Write([]byte(strings.ReplaceAll(b.String(), `"`, `\"`)))
			return
		},
	)

	TestExpectRun(t, `
{%-
	global value
	obstart()
-%}
{key:"{%= value%}"}
{%- return str(obend())%}
`,
		NewTestOpts().
			Mixed().
			ExprToTextFunc("expr2text").
			Builtins(map[string]Object{
				"expr2text": exprToText,
			}).
			Globals(Dict{
				"value": Str(`a"b`),
			}),
		Str(`{key:"a\"b"}`),
	)

	TestExpectRun(t, `{%global value-%}{key:"{%= value%}"}`,
		NewTestOpts().
			Mixed().
			Buffered().
			ExprToTextFunc("expr2text").
			Builtins(map[string]Object{
				"expr2text": exprToText,
			}).
			Globals(Dict{
				"value": Str(`a"b`),
			}),
		Array{Nil, Str(`{key:"a\"b"}`)},
	)

	TestExpectRun(t, `{%var value%}{%= value%}`,
		NewTestOpts().
			Mixed().
			Buffered().
			WriteObject(ObjectToWriterFunc(func(_ *VM, w io.Writer, obj Object) (bool, int64, error) {
				var n int
				n, err := w.Write([]byte("value"))
				return true, int64(n), err
			})),
		Array{Nil, Str(`value`)},
	)

	TestExpectRun(t, `var value; return write(1, value, 2, {})`,
		NewTestOpts().
			Buffered().
			WriteObject(ObjectToWriters{
				ObjectToWriterFunc(func(_ *VM, w io.Writer, obj Object) (handled bool, n int64, err error) {
					if obj == Nil {
						n, err := w.Write([]byte("-"))
						return true, int64(n), err
					}
					return
				}),
				DefaultObjectToWrite,
			}),
		Array{Int(5), Str(`1-2{}`)},
	)
}

func TestVMReflectSlice(t *testing.T) {
	TestExpectRun(t, `param s;return func(z, *x) { return append([], *x) }(100, *s)`,
		NewTestOpts().Args(MustToObject([]int{4, 7})),
		Array{Int(4), Int(7)},
	)
	TestExpectRun(t, `param s;return func(*x) { return append([], *x) }(*s)`,
		NewTestOpts().Args(MustToObject([]int{4, 7})),
		Array{Int(4), Int(7)},
	)
	TestExpectRun(t, "param s;return append([], *s)",
		NewTestOpts().Args(MustToObject([]int{4, 7})),
		Array{Int(4), Int(7)},
	)
}

func TestVMReturn(t *testing.T) {
	TestExpectRun(t, `return`, nil, Nil)
	TestExpectRun(t, `return 1`, nil, Int(1))
	TestExpectRun(t, `return 1, 2`, nil, Array{Int(1), Int(2)})
	TestExpectRun(t, `nil || return`, nil, Nil)
	TestExpectRun(t, `0 || return`, nil, Nil)
	TestExpectRun(t, `0 || return; 1`, nil, Nil)
	TestExpectRun(t, `0 || return 2`, nil, Int(2))
	TestExpectRun(t, `0 || return 2 && 3`, nil, Int(3))
	TestExpectRun(t, `0 || (return 2) && 3`, nil, Int(2))
	TestExpectRun(t, `1 && return 2`, nil, Int(2))
	TestExpectRun(t, `1 && (return 2) && 3`, nil, Int(2))
	TestExpectRun(t, `1 && (return 2) || 3`, nil, Int(2))
	TestExpectRun(t, `1 && (return 2, 3) || 4`, nil, Array{Int(2), Int(3)})
}

type callerObject struct {
	Dict
}

func (n *callerObject) CanCall() bool {
	return true
}

func (*callerObject) Call(c Call) (Object, error) {
	nargs := c.NamedArgs.Dict()
	if nargs == nil {
		nargs = Dict{}
	}
	return Array{c.Args.Values(), nargs}, nil
}

var _ CallerObject = &callerObject{}

func expectErrHas(t *testing.T, script string, opts *TestOpts, expectMsg string) {
	t.Helper()
	if expectMsg == "" {
		panic("expected message must not be empty")
	}
	expectErrorGen(t, script, opts, func(t *testing.T, retErr error) {
		t.Helper()
		if !strings.Contains(retErr.Error(), expectMsg) {
			require.Failf(t, "expectErrHas Failed",
				"expected error: %v, got: %v", expectMsg, retErr)
		}
	})
}

func expectErrIs(t *testing.T, script string, opts *TestOpts, expectErr error) {
	t.Helper()
	expectErrorGen(t, script, opts, func(t *testing.T, retErr error) {
		t.Helper()
		if !errors.Is(retErr, expectErr) {
			if re, ok := retErr.(*RuntimeError); ok {
				if !errors.Is(re.Err, expectErr) {
					if gerr, _ := expectErr.(*Error); gerr != nil {
						if gerr.Error() == re.Err.Error() {
							return
						}
					}
				}
			}
			require.Failf(t, "expectErrorIs Failed",
				"expected error: %v, got: %v", expectErr, retErr)
		}
	})
}

func expectErrAs(t *testing.T, script string, opts *TestOpts, asErr any, eqErr any) {
	t.Helper()
	expectErrorGen(t, script, opts, func(t *testing.T, retErr error) {
		t.Helper()
		if !errors.As(retErr, asErr) {
			require.Failf(t, "expectErrorAs Type Failed",
				"expected error type: %T, got: %T(%v)", asErr, retErr, retErr)
		}
		if eqErr != nil && !reflect.DeepEqual(eqErr, asErr) {
			require.Failf(t, "expectErrorAs Equality Failed",
				"errors not equal: %[1]T(%[1]v), got: %[2]T(%[2]v)", eqErr, retErr)
		}
	})
}

func expectErrorGen(
	t *testing.T,
	script string,
	opts *TestOpts,
	callback func(*testing.T, error),
) {
	t.Helper()
	if opts == nil {
		opts = NewTestOpts()
	}
	type testCase struct {
		name   string
		opts   CompilerOptions
		tracer bytes.Buffer
	}
	testCases := []testCase{
		{
			name: "default",
			opts: CompilerOptions{
				ModuleMap:      opts.GetModuleMap(),
				OptimizeConst:  true,
				TraceParser:    true,
				TraceOptimizer: true,
				TraceCompiler:  true,
			},
		},
		{
			name: "unoptimized",
			opts: CompilerOptions{
				ModuleMap:      opts.GetModuleMap(),
				TraceParser:    true,
				TraceOptimizer: true,
				TraceCompiler:  true,
			},
		},
	}
	if opts.Skip2pass {
		testCases = testCases[:1]
	}
	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			t.Helper()
			tC.opts.Trace = &tC.tracer // nolint exportloopref
			compiled, err := Compile([]byte(script), CompileOptions{CompilerOptions: tC.opts})
			if opts.IsCompilerErr {
				require.Error(t, err)
				callback(t, err)
				return
			}
			require.NoError(t, err)
			_, err = NewVM(compiled).SetRecover(opts.IsNoPanic()).RunOpts(&RunOpts{
				Globals:   opts.GetGlobals(),
				Args:      Args{opts.GetArgs()},
				NamedArgs: opts.GetNameArgs(),
			})
			require.Error(t, err)
			callback(t, err)
		})
	}
}

type srwcloser struct {
	Closed bool
	bytes.Buffer
}

func (s *srwcloser) Reset() {
	s.Closed = false
	s.Buffer.Reset()
}

func (s *srwcloser) Close() error {
	s.Closed = true
	return nil
}
