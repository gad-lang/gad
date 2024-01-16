package parser_test

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"

	. "github.com/gad-lang/gad/parser/ast"
	. "github.com/gad-lang/gad/parser/node"
	. "github.com/gad-lang/gad/parser/source"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/gad-lang/gad/token"

	. "github.com/gad-lang/gad/parser"
)

var update = flag.Bool("update", false, "update golden files")

func TestParserTrace(t *testing.T) {
	parse := func(input string, tracer io.Writer) {
		testFileSet := NewFileSet()
		testFile := testFileSet.AddFile("test", -1, len(input))
		p := NewParser(testFile, []byte(input), tracer)
		_, err := p.ParseFile()
		require.NoError(t, err)
	}
	sample := `
param (a, *args)
global (x, y)
var b
var (v1 = 1, v2)
const (
	c1 = 1
	c2 = 2
)
if w := ""; w {
	return
}
for i := 0; i < 10; i++ {
	if i == 5 {
		break
	}
	if i == 6 {
		try {
			x()
		} catch err {
			println(err)
			throw err
		} finally {
			return v1
		}
	}
}
counter := 0
for k,v in {a: 1, b: 2} {
	counter++
	println(k, v)
}
f := func() {
	return 0, error("err")
}
v1, v2 := f()
v3 := [v1*counter, v2/counter, 3u,
	4.7, 'y']
v3[1]
v3[1:]
v3[:2]
v3[:]
_ := import("strings")
time := import("time")
time.Now() + 10 * time.Second
c := counter ? v3 : nil
c ||= 1
d := c ?? 2 || 1
x := d?.a.b.("c")?.e ?? 5
`
	goldenFile := "testdata/trace.golden"
	f, err := os.Open(goldenFile)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, f.Close())
	}()
	if *update {
		require.NoError(t, f.Close())
		f, err = os.OpenFile(goldenFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		require.NoError(t, err)
		parse(sample, f)
		require.NoError(t, f.Close())
		f, err = os.Open(goldenFile)
		require.NoError(t, err)
	}
	golden, err := io.ReadAll(f)
	require.NoError(t, err)
	var out bytes.Buffer
	parse(sample, &out)
	require.Equal(t,
		strings.ReplaceAll(string(golden), "\r\n", "\n"),
		strings.ReplaceAll(out.String(), "\r\n", "\n"),
	)
}

func TestParserMixed(t *testing.T) {
	expectParseStringMode(t, ParseMixed, "# gad: writer=myfn\n#{- var myfn -} a", "# gad: writer=myfn; var myfn; #{= `a` }")
	expectParseStringMode(t, ParseMixed, "# gad: writer=myfn\na#{var myfn}b", "# gad: writer=myfn; #{= `a` }; var myfn; #{= `b` }")

	expectParseStringMode(t, ParseMixed, "#{var a}", `var a`)
	expectParseStringMode(t, ParseMixed, "#{for e in list do}1#{end}", "for _, e in list {#{= `1` }}")
	expectParseStringMode(t, ParseMixed, "#{for e in list do}1#{else}2#{end}", "for _, e in list {#{= `1` }} else {#{= `2` }}")
	expectParseStringMode(t, ParseMixed, "a  #{-= 1 -}\n\tb", "#{= `a` }; #{= 1 }; #{= `b` }")
	expectParseStringMode(t, ParseMixed, "#{ a := begin -} 2 #{- end }", "a := (`2`)")
	expectParseStringMode(t, ParseMixed, "#{ if 1 then } 2 #{ end }", "if 1 {#{= ` 2 ` }}")

	expectParseMode(t, ParseMixed, "a  #{-= 1 -}\n\tb", func(p pfn) []Stmt {
		return stmts(
			rawStringStmt(p(1, 1), "a"),
			toText(lit("#{=", p(1, 4)), lit("}", p(1, 12)), intLit(1, p(1, 9))),
			rawStringStmt(p(1, 13), "b"),
		)
	})
	expectParseMode(t, ParseMixed, `a  #{- 1}b#{1 + 2}c`, func(p pfn) []Stmt {
		return stmts(
			rawStringStmt(p(1, 1), "a"),
			exprStmt(intLit(1, p(1, 8))),
			rawStringStmt(p(1, 10), "b"),
			exprStmt(binaryExpr(intLit(1, p(1, 13)), intLit(2, p(1, 17)), token.Add, p(1, 15))),
			rawStringStmt(p(1, 19), "c"),
		)
	})
	expectParseMode(t, ParseMixed, "a  #{-= 1 }\n\tb", func(p pfn) []Stmt {
		return stmts(
			rawStringStmt(p(1, 1), "a"),
			toText(lit("#{=", p(1, 4)), lit("}", p(1, 11)), intLit(1, p(1, 9))),
			rawStringStmt(p(1, 12), "\n\tb"),
		)
	})
	expectParseMode(t, ParseMixed, "a  #{-= 1 -}\n\tb", func(p pfn) []Stmt {
		return stmts(
			rawStringStmt(p(1, 1), "a"),
			toText(lit("#{=", p(1, 4)), lit("}", p(1, 12)), intLit(1, p(1, 9))),
			rawStringStmt(p(1, 13), "b"),
		)
	})
	expectParseMode(t, ParseMixed, `a#{=1}b`, func(p pfn) []Stmt {
		return stmts(
			rawStringStmt(p(1, 1), "a"),
			toText(lit("#{=", p(1, 2)), lit("}", p(1, 6)), intLit(1, p(1, 5))),
			rawStringStmt(p(1, 7), "b"),
		)
	})

	expectParseMode(t, ParseMixed, `a#{=  1   }b`, func(p pfn) []Stmt {
		return stmts(
			rawStringStmt(p(1, 1), "a"),
			toText(lit("#{=", p(1, 2)), lit("}", p(1, 11)), intLit(1, p(1, 7))),
			rawStringStmt(p(1, 12), "b"),
		)
	})

	expectParseMode(t, ParseMixed, `a#{1}b#{1 + 2}c`, func(p pfn) []Stmt {
		return stmts(
			rawStringStmt(p(1, 1), "a"),
			exprStmt(intLit(1, p(1, 4))),
			rawStringStmt(p(1, 6), "b"),
			exprStmt(binaryExpr(intLit(1, p(1, 9)), intLit(2, p(1, 13)), token.Add, p(1, 11))),
			rawStringStmt(p(1, 15), "c"),
		)
	})

	expectParseMode(t, ParseMixed, `a#{1}b#{true}c`, func(p pfn) []Stmt {
		return stmts(
			rawStringStmt(p(1, 1), "a"),
			exprStmt(intLit(1, p(1, 4))),
			rawStringStmt(p(1, 6), "b"),
			exprStmt(boolLit(true, p(1, 9))),
			rawStringStmt(p(1, 14), "c"),
		)
	})

	expectParseMode(t, ParseMixed, `a#{1}b`, func(p pfn) []Stmt {
		return stmts(
			rawStringStmt(p(1, 1), "a"),
			exprStmt(intLit(1, p(1, 4))),
			rawStringStmt(p(1, 6), "b"),
		)
	})

	expectParseMode(t, ParseMixed, `abc`, func(p pfn) []Stmt {
		return stmts(
			rawStringStmt(p(1, 1), "abc"),
		)
	})

	expectParseStringMode(t, ParseMixed, "#{1}", `1`)
	expectParseStringMode(t, ParseMixed, "#{1}#{2}#{3}", `1; 2; 3`)
	expectParseStringMode(t, ParseMixed, "#{1}#{}#{3}", `1; 3`)
	expectParseStringMode(t, ParseMixed, "#{1}#{=2}#{3}", `1; #{= 2 }; 3`)
	expectParseStringMode(t, ParseMixed, "abc", "#{= `abc` }")
	expectParseStringMode(t, ParseMixed, "a#{1}", "#{= `a` }; 1")
	expectParseStringMode(t, ParseMixed, "a#{1}b", "#{= `a` }; 1; #{= `b` }")
	expectParseStringMode(t, ParseMixed, "a#{1}b#{= 2 + 4}", "#{= `a` }; 1; #{= `b` }; #{= (2 + 4) }")
	expectParseStringMode(t, ParseMixed, "a  #{- 1}", "#{= `a` }; 1")
	expectParseStringMode(t, ParseMixed, "a\n#{- 1}\tb\n#{-= 2 -}\n\nc", "#{= `a` }; 1; #{= `\\tb` }; #{= 2 }; #{= `c` }")
	expectParseStringMode(t, ParseMixed, `a#{=1}c#{x := 5}#{=x}`, "#{= `a` }; #{= 1 }; #{= `c` }; x := 5; #{= x }")

	expectParseStringMode(t, ParseMixed, "#{if true then}1#{else if a then}2#{else then}3#{fn()}#{end}", "if true {#{= `1` }} else if a {#{= `2` }} else {#{= `3` }; fn()}")
	expectParseStringMode(t, ParseMixed, "#{if true then}1#{else if a then}2#{else}3#{end}", "if true {#{= `1` }} else if a {#{= `2` }} else {#{= `3` }}")
	expectParseStringMode(t, ParseMixed, "#{if true then}1#{else if a then}2#{end}", "if true {#{= `1` }} else if a {#{= `2` }}")
	expectParseStringMode(t, ParseMixed, "#{if true then}1#{else}2#{end}", "if true {#{= `1` }} else {#{= `2` }}")
}

func TestParserError(t *testing.T) {
	err := &Error{Pos: SourceFilePos{
		Offset: 10, Line: 1, Column: 10,
	}, Msg: "test"}
	require.Equal(t, "Parse Error: test\n\tat 1:10", err.Error())
}

func TestParserErrorList(t *testing.T) {
	var list ErrorList
	list.Add(SourceFilePos{Offset: 20, Line: 2, Column: 10}, "error 2")
	list.Add(SourceFilePos{Offset: 30, Line: 3, Column: 10}, "error 3")
	list.Add(SourceFilePos{Offset: 10, Line: 1, Column: 10}, "error 1")
	list.Sort()
	require.Equal(t, "Parse Error: error 1\n\tat 1:10 (and 2 more errors)",
		list.Error())
}

func TestParsePipe(t *testing.T) {
	expectParseString(t, "(a.b.|x().|y().z.|c(1).d.e.|f().g.h.i)", "(((((a.b .| x()) .| y().z) .| c(1).d.e) .| f().g.h.i))")
	expectParseString(t, "a.b.|x().|y().z.|c(1).d.e.|f().g.h.i", "((((a.b .| x()) .| y().z) .| c(1).d.e) .| f().g.h.i)")
	expectParseString(t, "a.b.|x().|y().z.|c(1).d.e", "(((a.b .| x()) .| y().z) .| c(1).d.e)")
	expectParseString(t, "a.b.|x().|y()", "((a.b .| x()) .| y())")
	expectParseString(t, "a.b.|x().|y().z", "((a.b .| x()) .| y().z)")
	expectParseString(t, "a.b.|x().|y().z.|a(1).c", "(((a.b .| x()) .| y().z) .| a(1).c)")
}

func TestParseDecl(t *testing.T) {
	expectParse(t, `param a`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Param, p(1, 1), 0, 0,
					paramSpec(false, typedIdent(ident("a", p(1, 7)))),
				),
			),
		)
	})
	expectParse(t, `param *a;`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Param, p(1, 1), 0, 0,
					paramSpec(true, typedIdent(ident("a", p(1, 8)))),
				),
			),
		)
	})
	expectParse(t, `param (a, *b)`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Param, p(1, 1), p(1, 7), p(1, 13),
					paramSpec(false, typedIdent(ident("a", p(1, 8)))),
					paramSpec(true, typedIdent(ident("b", p(1, 12)))),
				),
			),
		)
	})
	expectParse(t, `param (a,
*b)`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Param, p(1, 1), p(1, 7), p(2, 3),
					paramSpec(false, typedIdent(ident("a", p(1, 8)))),
					paramSpec(true, typedIdent(ident("b", p(2, 2)))),
				),
			),
		)
	})

	expectParse(t, `param (a, *b; c=1, d=2, **e)`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Param, p(1, 1), p(1, 7), p(1, 28),
					paramSpec(false, typedIdent(ident("a", p(1, 8)))),
					paramSpec(true, typedIdent(ident("b", p(1, 12)))),
					nparamSpec(typedIdent(ident("c", p(1, 15))), intLit(1, p(1, 17))),
					nparamSpec(typedIdent(ident("d", p(1, 20))), intLit(2, p(1, 22))),
					nparamSpec(typedIdent(ident("e", p(1, 27))), nil),
				),
			),
		)
	})
	expectParse(t, `param (c=1, d=2, **e)`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Param, p(1, 1), p(1, 7), p(1, 21),
					nparamSpec(typedIdent(ident("c", p(1, 8))), intLit(1, p(1, 10))),
					nparamSpec(typedIdent(ident("d", p(1, 13))), intLit(2, p(1, 15))),
					nparamSpec(typedIdent(ident("e", p(1, 20))), nil),
				),
			),
		)
	})
	expectParse(t, `param (;c=1, d=2, **e)`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Param, p(1, 1), p(1, 7), p(1, 22),
					nparamSpec(typedIdent(ident("c", p(1, 9))), intLit(1, p(1, 11))),
					nparamSpec(typedIdent(ident("d", p(1, 14))), intLit(2, p(1, 16))),
					nparamSpec(typedIdent(ident("e", p(1, 21))), nil),
				),
			),
		)
	})

	expectParseString(t, "param x", "param x")
	expectParseString(t, "param (\nx,\n)", "param (x)")
	expectParseString(t, "param (\nx,\ny)", "param (x, y)")
	expectParseString(t, "param (\nx,\ny,\n)", "param (x, y)")
	expectParseString(t, "param (x,y)", "param (x, y)")
	expectParseString(t, "param (x,\ny)", "param (x, y)")
	expectParseString(t, "param (x,\ny)", "param (x, y)")
	expectParseString(t, "param *x", "param *x")
	expectParseString(t, "param **x", "param **x")
	expectParseString(t, "param b=2", "param b=2")
	expectParseString(t, "param (x,*y)", "param (x, *y)")
	expectParseString(t, "param (x,\n*y)", "param (x, *y)")
	expectParseString(t, "param (c=1, d=2, **e)", "param (c=1, d=2, **e)")
	expectParseString(t, "param (;c=1, d=2, **e)", "param (c=1, d=2, **e)")
	expectParseString(t, "param (a, *b;c=1, d=2, **e)", "param (a, *b, c=1, d=2, **e)")
	expectParseString(t, "param (a,\n*b\n; c=2,\nx=5)", "param (a, *b, c=2, x=5)")

	expectParseString(t, "param x:int", "param x:int")
	expectParseString(t, "param x:[int,bool]", "param x:[int, bool]")
	expectParseString(t, "param (\nx:int,\n)", "param (x:int)")
	expectParseString(t, "param (\nx:int,\ny)", "param (x:int, y)")
	expectParseString(t, "param (\nx:int,\ny, z:[string,bool])", "param (x:int, y, z:[string, bool])")
	expectParseString(t, "param *x:int", "param *x:int")
	expectParseString(t, "param *x:[int,bool]", "param *x:[int, bool]")
	expectParseString(t, "param **x:int", "param **x:int")
	expectParseString(t, "param **x:[int,bool]", "param **x:[int, bool]")
	expectParseString(t, "param b:int=2", "param b:int=2")
	expectParseString(t, "param b:[bool,int]=2", "param b:[bool, int]=2")
	expectParseString(t, "param (a, *b:string, x:[bool,int]=2, **y:[int])",
		"param (a, *b:string, x:[bool, int]=2, **y:int)")

	expectParse(t, `global a`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Global, p(1, 1), 0, 0,
					paramSpec(false, typedIdent(ident("a", p(1, 8)))),
				),
			),
		)
	})
	expectParse(t, `
global a
global b`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Global, p(2, 1), 0, 0,
					paramSpec(false, typedIdent(ident("a", p(2, 8)))),
				),
			),
			declStmt(
				genDecl(token.Global, p(3, 1), 0, 0,
					paramSpec(false, typedIdent(ident("b", p(3, 8)))),
				),
			),
		)
	})
	expectParse(t, `global (a, b)`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Global, p(1, 1), p(1, 8), p(1, 13),
					paramSpec(false, typedIdent(ident("a", p(1, 9)))),
					paramSpec(false, typedIdent(ident("b", p(1, 12)))),
				),
			),
		)
	})
	expectParse(t, `global (a, 
b)`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Global, p(1, 1), p(1, 8), p(2, 2),
					paramSpec(false, typedIdent(ident("a", p(1, 9)))),
					paramSpec(false, typedIdent(ident("b", p(2, 1)))),
				),
			),
		)
	})
	expectParse(t, `global (a 
b)`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Global, p(1, 1), p(1, 8), p(2, 2),
					paramSpec(false, typedIdent(ident("a", p(1, 9)))),
					paramSpec(false, typedIdent(ident("b", p(2, 1)))),
				),
			),
		)
	})
	expectParseString(t, "global x", "global x")
	expectParseString(t, "global (\nx\n)", "global (x)")
	expectParseString(t, "global (x,y)", "global (x, y)")
	expectParseString(t, "global (x\ny)", "global (x, y)")
	expectParseString(t, "global (\nx\ny)", "global (x, y)")
	expectParseString(t, "global (x,\ny)", "global (x, y)")
	expectParseString(t, "global (x\ny)", "global (x, y)")

	expectParse(t, `var a`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Var, p(1, 1), 0, 0,
					valueSpec(
						[]*Ident{ident("a", p(1, 5))},
						[]Expr{nil}),
				),
			),
		)
	})
	expectParse(t, `var a=1`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Var, p(1, 1), 0, 0,
					valueSpec(
						[]*Ident{ident("a", p(1, 5))},
						[]Expr{intLit(1, p(1, 7))}),
				),
			),
		)
	})
	expectParse(t, `var a;var b`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Var, p(1, 1), 0, 0,
					valueSpec(
						[]*Ident{ident("a", p(1, 5))},
						[]Expr{nil}),
				),
			),
			declStmt(
				genDecl(token.Var, p(1, 7), 0, 0,
					valueSpec(
						[]*Ident{ident("b", p(1, 11))},
						[]Expr{nil}),
				),
			),
		)
	})
	expectParse(t, `var a="x";var b`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Var, p(1, 1), 0, 0,
					valueSpec(
						[]*Ident{ident("a", p(1, 5))},
						[]Expr{stringLit("x", p(1, 7))}),
				),
			),
			declStmt(
				genDecl(token.Var, p(1, 11), 0, 0,
					valueSpec(
						[]*Ident{ident("b", p(1, 15))},
						[]Expr{nil}),
				),
			),
		)
	})
	expectParse(t, `
var a
var b
`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Var, p(2, 1), 0, 0,
					valueSpec(
						[]*Ident{ident("a", p(2, 5))},
						[]Expr{nil}),
				),
			),
			declStmt(
				genDecl(token.Var, p(3, 1), 0, 0,
					valueSpec(
						[]*Ident{ident("b", p(3, 5))},
						[]Expr{nil}),
				),
			),
		)
	})
	expectParse(t, `
var a
var b=2
`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Var, p(2, 1), 0, 0,
					valueSpec(
						[]*Ident{ident("a", p(2, 5))},
						[]Expr{nil}),
				),
			),
			declStmt(
				genDecl(token.Var, p(3, 1), 0, 0,
					valueSpec(
						[]*Ident{ident("b", p(3, 5))},
						[]Expr{intLit(2, p(3, 7))}),
				),
			),
		)
	})
	expectParse(t, `var (a, b=2)`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Var, p(1, 1), p(1, 5), p(1, 12),
					valueSpec(
						[]*Ident{ident("a", p(1, 6))},
						[]Expr{nil}),
					valueSpec(
						[]*Ident{ident("b", p(1, 9))},
						[]Expr{intLit(2, p(1, 11))}),
				),
			),
		)
	})
	expectParse(t, `var (a=1, b=2)`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Var, p(1, 1), p(1, 5), p(1, 14),
					valueSpec(
						[]*Ident{ident("a", p(1, 6))},
						[]Expr{intLit(1, p(1, 8))}),
					valueSpec(
						[]*Ident{ident("b", p(1, 11))},
						[]Expr{intLit(2, p(1, 13))}),
				),
			),
		)
	})
	expectParse(t, `var (a=1,
b=2)`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Var, p(1, 1), p(1, 5), p(2, 4),
					valueSpec(
						[]*Ident{ident("a", p(1, 6))},
						[]Expr{intLit(1, p(1, 8))}),
					valueSpec(
						[]*Ident{ident("b", p(2, 1))},
						[]Expr{intLit(2, p(2, 3))}),
				),
			),
		)
	})
	expectParse(t, `var (a=1
b=2)`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Var, p(1, 1), p(1, 5), p(2, 4),
					valueSpec(
						[]*Ident{ident("a", p(1, 6))},
						[]Expr{intLit(1, p(1, 8))}),
					valueSpec(
						[]*Ident{ident("b", p(2, 1))},
						[]Expr{intLit(2, p(2, 3))}),
				),
			),
		)
	})
	expectParseString(t, "var x", "var x")
	expectParseString(t, "var (\nx\n)", "var (x)")
	expectParseString(t, "var (x,y)", "var (x, y)")
	expectParseString(t, "var (x\ny)", "var (x, y)")
	expectParseString(t, "var (\nx\ny)", "var (x, y)")
	expectParseString(t, "var (x,\ny)", "var (x, y)")
	expectParseString(t, "var (x=1,\ny)", "var (x = 1, y)")
	expectParseString(t, "var (x,\ny = 2)", "var (x, y = 2)")
	expectParseString(t, "var (x\ny)", "var (x, y)")
	expectParseString(t, `var (_, _a, $_a, a, A, $b, $, a1, $1, $b1, $$, ŝ, $ŝ)`,
		`var (_, _a, $_a, a, A, $b, $, a1, $1, $b1, $$, ŝ, $ŝ)`)

	expectParse(t, `const a = 1`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Const, p(1, 1), 0, 0,
					valueSpec(
						[]*Ident{ident("a", p(1, 7))},
						[]Expr{intLit(1, p(1, 11))}),
				),
			),
		)
	})
	expectParse(t, `const a = 1; const b = 2`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Const, p(1, 1), 0, 0,
					valueSpec(
						[]*Ident{ident("a", p(1, 7))},
						[]Expr{intLit(1, p(1, 11))}),
				),
			),
			declStmt(
				genDecl(token.Const, p(1, 14), 0, 0,
					valueSpec(
						[]*Ident{ident("b", p(1, 20))},
						[]Expr{intLit(2, p(1, 24))}),
				),
			),
		)
	})
	expectParse(t, `const (a = 1, b = 2)`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Const, p(1, 1), p(1, 7), p(1, 20),
					valueSpec(
						[]*Ident{ident("a", p(1, 8))},
						[]Expr{intLit(1, p(1, 12))}),
					valueSpec(
						[]*Ident{ident("b", p(1, 15))},
						[]Expr{intLit(2, p(1, 19))}),
				),
			),
		)
	})
	expectParse(t, `
const (
    a = 1
    b = 2
)`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Const, p(2, 1), p(2, 7), p(5, 1),
					valueSpec(
						[]*Ident{ident("a", p(3, 5))},
						[]Expr{intLit(1, p(3, 9))}),
					valueSpec(
						[]*Ident{ident("b", p(4, 5))},
						[]Expr{intLit(2, p(4, 9))}),
				),
			),
		)
	})
	expectParseString(t, "const x=1", "const x = 1")
	expectParseString(t, "const (\nx=1\n)", "const (x = 1)")
	expectParseString(t, "const (x=1,y=2)", "const (x = 1, y = 2)")
	expectParseString(t, "const (x=1\ny=2)", "const (x = 1, y = 2)")
	expectParseString(t, "const (\nx=1\ny=2)", "const (x = 1, y = 2)")
	expectParseString(t, "const (x=1,\ny=2)", "const (x = 1, y = 2)")

	expectParseError(t, `param a,b`)
	expectParseError(t, `param (a... ,b)`)
	expectParseError(t, `param (... ,b)`)
	expectParseError(t, `param (...)`)
	expectParseError(t, `param ...`)
	expectParseError(t, `param (a, b...)`)
	expectParseError(t, `param a,b)`)
	expectParseError(t, `param (a,b`)
	expectParseError(t, `param (a...,b...`)
	expectParseError(t, `param (...a,...b`)
	expectParseError(t, `param a,`)
	expectParseError(t, `param ,a`)
	expectParseError(t, `global a...`)
	expectParseError(t, `global a,b`)
	expectParseError(t, `global a,b)`)
	expectParseError(t, `global (a,b`)
	expectParseError(t, `global a,`)
	expectParseError(t, `global ,a`)
	expectParseError(t, `var a,b`)
	expectParseError(t, `var ...a`)
	expectParseError(t, `var a...`)
	expectParseError(t, `var a,b)`)
	expectParseError(t, `var (a,b`)
	expectParseError(t, `var a,`)
	expectParseError(t, `var ,a`)
	expectParseError(t, `const a=1,b=2`)

	// After iota support, this should be valid.
	//	expectParseError(t, `const (a=1,b)`)

	expectParseError(t, `const a`)
	expectParseError(t, `const (a)`)
	expectParseError(t, `const (a,b)`)
	expectParseError(t, `const (a=1`)
	expectParseError(t, `const (a`)
	expectParseError(t, `const a=1,`)
	expectParseError(t, `const ,a=2`)
}

func TestCommaSepReturn(t *testing.T) {
	expectParse(t, "return 1, 23", func(p pfn) []Stmt {
		return stmts(
			returnStmt(
				p(1, 1),
				arrayLit(
					p(1, 8),
					p(1, 13),
					intLit(1, p(1, 8)),
					intLit(23, p(1, 11)),
				),
			),
		)
	})
	expectParse(t, "return 1, 23, 2.2, 12.34d", func(p pfn) []Stmt {
		return stmts(
			returnStmt(
				p(1, 1),
				arrayLit(
					p(1, 8),
					p(1, 26),
					intLit(1, p(1, 8)),
					intLit(23, p(1, 11)),
					floatLit(2.2, p(1, 15)),
					decimalLit("12.34", p(1, 20)),
				),
			),
		)
	})
	expectParse(t, "return a, b", func(p pfn) []Stmt {
		return stmts(
			returnStmt(
				p(1, 1),
				arrayLit(
					p(1, 8),
					p(1, 12),
					ident("a", p(1, 8)),
					ident("b", p(1, 11)),
				),
			),
		)
	})
	expectParse(t, "func() { return a, b }", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				funcLit(
					funcType(p(1, 1), p(1, 5), p(1, 6)),
					blockStmt(
						p(1, 8),
						p(1, 22),
						returnStmt(
							p(1, 10),
							arrayLit(
								p(1, 17),
								p(1, 21),
								ident("a", p(1, 17)),
								ident("b", p(1, 20)),
							),
						),
					),
				),
			),
		)
	})
	expectParseError(t, `return a,`)
	expectParseError(t, `return a,b,`)
	expectParseError(t, `return a,`)
	expectParseError(t, `func() { return a, }`)
	expectParseError(t, `func() { return a,b, }`)
}

func TestParseArray(t *testing.T) {
	expectParse(t, "[1, 2, 3]", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				arrayLit(p(1, 1), p(1, 9),
					intLit(1, p(1, 2)),
					intLit(2, p(1, 5)),
					intLit(3, p(1, 8)))))
	})

	expectParse(t, `
[
	1, 
	2, 
	3,
]`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				arrayLit(p(2, 1), p(6, 1),
					intLit(1, p(3, 2)),
					intLit(2, p(4, 2)),
					intLit(3, p(5, 2)))))
	})
	expectParse(t, `
[
	1, 
	2, 
	3,

]`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				arrayLit(p(2, 1), p(7, 1),
					intLit(1, p(3, 2)),
					intLit(2, p(4, 2)),
					intLit(3, p(5, 2)))))
	})

	expectParse(t, `[1, "foo", 12.34]`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				arrayLit(p(1, 1), p(1, 17),
					intLit(1, p(1, 2)),
					stringLit("foo", p(1, 5)),
					floatLit(12.34, p(1, 12)))))
	})

	expectParse(t, "a = [1, 2, 3]", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(arrayLit(p(1, 5), p(1, 13),
					intLit(1, p(1, 6)),
					intLit(2, p(1, 9)),
					intLit(3, p(1, 12)))),
				token.Assign,
				p(1, 3)))
	})

	expectParse(t, "a = [1 + 2, b * 4, [4, c]]", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(arrayLit(p(1, 5), p(1, 26),
					binaryExpr(
						intLit(1, p(1, 6)),
						intLit(2, p(1, 10)),
						token.Add,
						p(1, 8)),
					binaryExpr(
						ident("b", p(1, 13)),
						intLit(4, p(1, 17)),
						token.Mul,
						p(1, 15)),
					arrayLit(p(1, 20), p(1, 25),
						intLit(4, p(1, 21)),
						ident("c", p(1, 24))))),
				token.Assign,
				p(1, 3)))
	})

	expectParseError(t, "[,]")
	expectParseError(t, "[1\n,]")
	expectParseError(t, "[1,\n2\n,]")
	expectParseError(t, `[1, 2, 3
	,]`)
	expectParseError(t, `
[
	1, 
	2, 
	3
]`)
	expectParseError(t, `
[
	1, 
	2, 
	3

]`)
	expectParseError(t, `[1, 2, 3, ,]`)
}

func TestParseAssignment(t *testing.T) {
	expectParse(t, "a = 5", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(intLit(5, p(1, 5))),
				token.Assign,
				p(1, 3)))
	})

	expectParse(t, "a := 5", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(intLit(5, p(1, 6))),
				token.Define,
				p(1, 3)))
	})

	expectParse(t, "a, b = 5, 10", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					ident("a", p(1, 1)),
					ident("b", p(1, 4))),
				exprs(
					intLit(5, p(1, 8)),
					intLit(10, p(1, 11))),
				token.Assign,
				p(1, 6)))
	})

	expectParse(t, "a, b := 5, 10", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					ident("a", p(1, 1)),
					ident("b", p(1, 4))),
				exprs(
					intLit(5, p(1, 9)),
					intLit(10, p(1, 12))),
				token.Define,
				p(1, 6)))
	})

	expectParse(t, "a, b = a + 2, b - 8", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					ident("a", p(1, 1)),
					ident("b", p(1, 4))),
				exprs(
					binaryExpr(
						ident("a", p(1, 8)),
						intLit(2, p(1, 12)),
						token.Add,
						p(1, 10)),
					binaryExpr(
						ident("b", p(1, 15)),
						intLit(8, p(1, 19)),
						token.Sub,
						p(1, 17))),
				token.Assign,
				p(1, 6)))
	})

	expectParse(t, "a = [1, 2, 3]", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(arrayLit(p(1, 5), p(1, 13),
					intLit(1, p(1, 6)),
					intLit(2, p(1, 9)),
					intLit(3, p(1, 12)))),
				token.Assign,
				p(1, 3)))
	})

	expectParse(t, "a = [1 + 2, b * 4, [4, c]]", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(arrayLit(p(1, 5), p(1, 26),
					binaryExpr(
						intLit(1, p(1, 6)),
						intLit(2, p(1, 10)),
						token.Add,
						p(1, 8)),
					binaryExpr(
						ident("b", p(1, 13)),
						intLit(4, p(1, 17)),
						token.Mul,
						p(1, 15)),
					arrayLit(p(1, 20), p(1, 25),
						intLit(4, p(1, 21)),
						ident("c", p(1, 24))))),
				token.Assign,
				p(1, 3)))
	})

	expectParse(t, "a += 5", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(intLit(5, p(1, 6))),
				token.AddAssign,
				p(1, 3)))
	})

	expectParse(t, "a *= 5 + 10", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(
					binaryExpr(
						intLit(5, p(1, 6)),
						intLit(10, p(1, 10)),
						token.Add,
						p(1, 8))),
				token.MulAssign,
				p(1, 3)))
	})

	expectParse(t, "a ||= 5", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(intLit(5, p(1, 7))),
				token.LOrAssign,
				p(1, 3)))
	})

	expectParse(t, "a ||= 5 + 10", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(
					binaryExpr(
						intLit(5, p(1, 7)),
						intLit(10, p(1, 11)),
						token.Add,
						p(1, 9))),
				token.LOrAssign,
				p(1, 3)))
	})

	expectParse(t, "a ??= 5", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(intLit(5, p(1, 7))),
				token.NullichAssign,
				p(1, 3)))
	})

	expectParse(t, "a ??= 5 + 10", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(
					binaryExpr(
						intLit(5, p(1, 7)),
						intLit(10, p(1, 11)),
						token.Add,
						p(1, 9))),
				token.NullichAssign,
				p(1, 3)))
	})
}

func TestParseUnaryNulls(t *testing.T) {
	expectParse(t, "false == nil", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				unaryExpr(
					boolLit(false, p(1, 1)),
					token.Null,
					p(1, 7))))
	})

	expectParse(t, "false != nil", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				unaryExpr(
					boolLit(false, p(1, 1)),
					token.NotNull,
					p(1, 7))))
	})

	expectParseString(t, "false == nil", "(false == nil)")
	expectParseString(t, "false != nil", "(false != nil)")
	expectParseString(t, "nil == nil", "(nil == nil)")
	expectParseString(t, "nil != nil", "(nil != nil)")

	expectParse(t, "a == nil ? b : c", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				condExpr(
					unaryExpr(
						ident("a", p(1, 1)),
						token.Null,
						p(1, 3)),
					ident("b", p(1, 12)),
					ident("c", p(1, 16)),
					p(1, 10),
					p(1, 14))))
	})

	expectParse(t, "a != nil ? b : c", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				condExpr(
					unaryExpr(
						ident("a", p(1, 1)),
						token.NotNull,
						p(1, 3)),
					ident("b", p(1, 12)),
					ident("c", p(1, 16)),
					p(1, 10),
					p(1, 14))))
	})

	expectParseString(t, "a == nil ? b : c", "((a == nil) ? b : c)")
	expectParseString(t, "a != nil ? b : c", "((a != nil) ? b : c)")
}

func TestParseBoolean(t *testing.T) {
	expectParse(t, "true", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				boolLit(true, p(1, 1))))
	})

	expectParse(t, "false", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				boolLit(false, p(1, 1))))
	})

	expectParse(t, "true != false", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				binaryExpr(
					boolLit(true, p(1, 1)),
					boolLit(false, p(1, 9)),
					token.NotEqual,
					p(1, 6))))
	})

	expectParse(t, "!false", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				unaryExpr(
					boolLit(false, p(1, 2)),
					token.Not,
					p(1, 1))))
	})
}

func TestParseFlag(t *testing.T) {
	expectParse(t, "yes", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				flagLit(true, p(1, 1))))
	})

	expectParse(t, "no", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				flagLit(false, p(1, 1))))
	})

	expectParse(t, "yes != no", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				binaryExpr(
					flagLit(true, p(1, 1)),
					flagLit(false, p(1, 8)),
					token.NotEqual,
					p(1, 5))))
	})

	expectParse(t, "!no", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				unaryExpr(
					flagLit(false, p(1, 2)),
					token.Not,
					p(1, 1))))
	})
}

func TestParseCallKeywords(t *testing.T) {
	expectParse(t, token.Callee.String(), func(p pfn) []Stmt {
		return stmts(exprStmt(caleeKw(p(1, 1))))
	})
	expectParse(t, token.Args.String(), func(p pfn) []Stmt {
		return stmts(exprStmt(argsKw(p(1, 1))))
	})
	expectParse(t, token.NamedArgs.String(), func(p pfn) []Stmt {
		return stmts(exprStmt(nargsKw(p(1, 1))))
	})
	expectParseString(t, token.Callee.String(), token.Callee.String())
	expectParseString(t, token.Args.String(), token.Args.String())
	expectParseString(t, token.NamedArgs.String(), token.NamedArgs.String())
}

func TestParseCall(t *testing.T) {
	expectParse(t, "add(1, 2, 3)", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				callExpr(
					ident("add", p(1, 1)),
					p(1, 4), p(1, 12),
					callExprArgs(nil,
						intLit(1, p(1, 5)),
						intLit(2, p(1, 8)),
						intLit(3, p(1, 11))))))
	})

	expectParse(t, "add(1, 2, *v)", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				callExpr(
					ident("add", p(1, 1)),
					p(1, 4), p(1, 13),
					callExprArgs(
						argVar(p(1, 11), ident("v", p(1, 12))),
						intLit(1, p(1, 5)),
						intLit(2, p(1, 8))))))
	})

	expectParse(t, "a = add(1, 2, 3)", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					ident("a", p(1, 1))),
				exprs(
					callExpr(
						ident("add", p(1, 5)),
						p(1, 8), p(1, 16),
						callExprArgs(nil,
							intLit(1, p(1, 9)),
							intLit(2, p(1, 12)),
							intLit(3, p(1, 15))))),
				token.Assign,
				p(1, 3)))
	})

	expectParse(t, "a, b = add(1, 2, 3)", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					ident("a", p(1, 1)),
					ident("b", p(1, 4))),
				exprs(
					callExpr(
						ident("add", p(1, 8)),
						p(1, 11), p(1, 19),
						callExprArgs(nil, intLit(1, p(1, 12)),
							intLit(2, p(1, 15)),
							intLit(3, p(1, 18))))),
				token.Assign,
				p(1, 6)))
	})

	expectParse(t, "add(a + 1, 2 * 1, (b + c))", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				callExpr(
					ident("add", p(1, 1)),
					p(1, 4), p(1, 26),
					callExprArgs(nil,
						binaryExpr(
							ident("a", p(1, 5)),
							intLit(1, p(1, 9)),
							token.Add,
							p(1, 7)),
						binaryExpr(
							intLit(2, p(1, 12)),
							intLit(1, p(1, 16)),
							token.Mul,
							p(1, 14)),
						parenExpr(
							binaryExpr(
								ident("b", p(1, 20)),
								ident("c", p(1, 24)),
								token.Add,
								p(1, 22)),
							p(1, 19), p(1, 25))))))
	})

	expectParseString(t, "a + add(b * c) + d", "((a + add((b * c))) + d)")
	expectParseString(t, "add(a, b, 1, 2 * 3, 4 + 5, add(6, 7 * 8))",
		"add(a, b, 1, (2 * 3), (4 + 5), add(6, (7 * 8)))")
	expectParseString(t, "f1(a) + f2(b) * f3(c)", "(f1(a) + (f2(b) * f3(c)))")
	expectParseString(t, "(f1(a) + f2(b)) * f3(c)",
		"(((f1(a) + f2(b))) * f3(c))")
	expectParseString(t, "f(1,)", "f(1)")
	expectParseString(t, "f(1,\n)", "f(1)")
	expectParseString(t, "f(\n1,\n)", "f(1)")
	expectParseString(t, "f(1,2,)", "f(1, 2)")
	expectParseString(t, "f(1,2,\n)", "f(1, 2)")
	expectParseString(t, "f(1,\n2,)", "f(1, 2)")
	expectParseString(t, "f(1,\n2,\n)", "f(1, 2)")
	expectParseString(t, "f(1,\n2,)", "f(1, 2)")
	expectParseString(t, "f(\n1,\n2,)", "f(1, 2)")
	expectParseString(t, "f(\n1,\n2)", "f(1, 2)")

	expectParse(t, "func(a, b) { a + b }(1, 2)", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				callExpr(
					funcLit(
						funcType(p(1, 1), p(1, 5), p(1, 10),
							funcArgs(nil,
								ident("a", p(1, 6)),
								ident("b", p(1, 9))),
						),
						blockStmt(
							p(1, 12), p(1, 20),
							exprStmt(
								binaryExpr(
									ident("a", p(1, 14)),
									ident("b", p(1, 18)),
									token.Add,
									p(1, 16))))),
					p(1, 21),
					p(1, 26),
					callExprArgs(nil,
						intLit(1, p(1, 22)),
						intLit(2, p(1, 25))))))
	})

	expectParse(t, `a.b()`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				callExpr(
					selectorExpr(
						ident("a", p(1, 1)),
						stringLit("b", p(1, 3))),
					p(1, 4), p(1, 5), NoPos)))
	})

	expectParse(t, `a.b.c()`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				callExpr(
					selectorExpr(
						selectorExpr(
							ident("a", p(1, 1)),
							stringLit("b", p(1, 3))),
						stringLit("c", p(1, 5))),
					p(1, 6), p(1, 7), NoPos)))
	})

	expectParse(t, `a["b"].c()`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				callExpr(
					selectorExpr(
						indexExpr(
							ident("a", p(1, 1)),
							stringLit("b", p(1, 3)),
							p(1, 2), p(1, 6)),
						stringLit("c", p(1, 8))),
					p(1, 9), p(1, 10), NoPos)))
	})

	expectParseError(t, `add(...a, 1)`)
	expectParseError(t, `add(a..., 1)`)
	expectParseError(t, `add(a..., b...)`)
	expectParseError(t, `add(1, a..., b...)`)
	expectParseError(t, `add(...)`)
	expectParseError(t, `add(1, ...)`)
	expectParseError(t, `add(1, ..., )`)
	expectParseError(t, `add(a...)`)
	expectParseError(t, `add(,)`)
	expectParseError(t, "add(\n,)")
}

func TestParseCallWithNamedArgs(t *testing.T) {
	expectParse(t, "add(x=2)", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				callExpr(
					ident("add", p(1, 1)),
					p(1, 4), p(1, 8),
					callExprNamedArgs(nil,
						[]NamedArgExpr{{Ident: ident("x", p(1, 5))}},
						[]Expr{intLit(2, p(1, 7))},
					))))
	})
	expectParse(t, "add(x=2,y=3)", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				callExpr(
					ident("add", p(1, 1)),
					p(1, 4), p(1, 12),
					callExprNamedArgs(nil,
						[]NamedArgExpr{{Ident: ident("x", p(1, 5))}, {Ident: ident("y", p(1, 9))}},
						[]Expr{intLit(2, p(1, 7)), intLit(3, p(1, 11))},
					))))
	})
	expectParse(t, "add(x=2,**{})", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				callExpr(
					ident("add", p(1, 1)),
					p(1, 4), p(1, 13),
					callExprNamedArgs(nargVar(Pos(9), dictLit(11, 12)),
						[]NamedArgExpr{{Ident: ident("x", p(1, 5))}},
						[]Expr{intLit(2, p(1, 7))},
					))))
	})
	expectParse(t, "add(\"x\"=2,y=3)", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				callExpr(
					ident("add", p(1, 1)),
					p(1, 4), p(1, 14),
					callExprNamedArgs(nil,
						[]NamedArgExpr{{Lit: stringLit("x", p(1, 5))}, {Ident: ident("y", p(1, 11))}},
						[]Expr{intLit(2, p(1, 9)), intLit(3, p(1, 13))},
					))))
	})

	expectParseString(t, `attrs(;"name")`, `attrs(name=on)`)
	expectParseString(t, "fn(a;b)", "fn(a, b=on)")
	expectParseString(t, "fn(**{y:5})", "fn(**{y: 5})")
	expectParseString(t, "fn(1,*[2,3],x=4,**{y:5})", "fn(1, *[2, 3], x=4, **{y: 5})")
	expectParseString(t, "fn(1, a=b)()", "fn(1, a=b)()")
}

func TestParseParenMultiValues(t *testing.T) {
	expectParseString(t, `(a,*b,c=2,**d) => 3`, `(a, *b, c=2, **d) => 3`)
	expectParseString(t, `(*a)`, `(*a)`)
	expectParseString(t, `(a, *b, c=1, **d)`, `(a, *b, c=1, **d)`)
	expectParseString(t, `(a,*b,c=2) => 3`, `(a, *b, c=2) => 3`)
	expectParseString(t, `(a,c=2, x(1))`, `(a, c=2, x(1))`)
	expectParseString(t, `(a,
c=2, 
  x(1))`, `(a, c=2, x(1))`)
}

func TestParseKeyValueArray(t *testing.T) {
	expectParseString(t, `(;)`, `(;)`)
	expectParseString(t, `(
;
)`, `(;)`)
	expectParseString(t, `(;a=1)`, `(;a=1)`)
	expectParseString(t, `(;flag)`, `(;flag)`)
	expectParseString(t, `(;
flag
)`, `(;flag)`)
	expectParseString(t, `(;a=1,b=2,"c"=3,4=5,true=false, myflag)`, `(;a=1, b=2, "c"=3, 4=5, true=false, myflag)`)
	expectParseString(t, `(;a=1,b=2,
"c"
=
3,4=5,
true=false, 
myflag)`, `(;a=1, b=2, "c"=3, 4=5, true=false, myflag)`)
}

func TestParseChar(t *testing.T) {
	expectParse(t, `'A'`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				charLit('A', 1)))
	})
	expectParse(t, `'九'`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				charLit('九', 1)))
	})

	expectParseError(t, `''`)
	expectParseError(t, `'AB'`)
	expectParseError(t, `'A九'`)
}

func TestParseCondExpr(t *testing.T) {
	expectParse(t, "a ? b : c", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				condExpr(
					ident("a", p(1, 1)),
					ident("b", p(1, 5)),
					ident("c", p(1, 9)),
					p(1, 3),
					p(1, 7))))
	})
	expectParse(t, `a ?
b :
c`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				condExpr(
					ident("a", p(1, 1)),
					ident("b", p(1, 5)),
					ident("c", p(1, 9)),
					p(1, 3),
					p(1, 7))))
	})

	expectParseString(t, `a ? b : c`, "(a ? b : c)")
	expectParseString(t, `a + b ? c - d : e * f`,
		"((a + b) ? (c - d) : (e * f))")
	expectParseString(t, `a == b ? c + (d / e) : f ? g : h + i`,
		"((a == b) ? (c + ((d / e))) : (f ? g : (h + i)))")
	expectParseString(t, `(a + b) ? (c - d) : (e * f)`,
		"(((a + b)) ? ((c - d)) : ((e * f)))")
	expectParseString(t, `a + (b ? c : d) - e`, "((a + ((b ? c : d))) - e)")
	expectParseString(t, `a ? b ? c : d : e`, "(a ? (b ? c : d) : e)")
	expectParseString(t, `a := b ? c : d`, "a := (b ? c : d)")
	expectParseString(t, `x := a ? b ? c : d : e`,
		"x := (a ? (b ? c : d) : e)")

	// ? : should be at the end of each line if it's multi-line
	expectParseError(t, `a 
? b 
: c`)
	expectParseError(t, `a ? (b : e)`)
	expectParseError(t, `(a ? b) : e`)
	expectParseError(t, `(b : e, c:d)`)
	expectParseError(t, `(b : e)`)
	expectParseError(t, `b : e`)
}

func TestParseForIn(t *testing.T) {
	expectParse(t, "for x in y {}", func(p pfn) []Stmt {
		return stmts(
			forInStmt(
				ident("_", p(1, 5)),
				ident("x", p(1, 5)),
				ident("y", p(1, 10)),
				blockStmt(p(1, 12), p(1, 13)),
				p(1, 1)))
	})

	expectParse(t, "for _ in y {}", func(p pfn) []Stmt {
		return stmts(
			forInStmt(
				ident("_", p(1, 5)),
				ident("_", p(1, 5)),
				ident("y", p(1, 10)),
				blockStmt(p(1, 12), p(1, 13)),
				p(1, 1)))
	})

	expectParse(t, "for x in [1, 2, 3] {}", func(p pfn) []Stmt {
		return stmts(
			forInStmt(
				ident("_", p(1, 5)),
				ident("x", p(1, 5)),
				arrayLit(
					p(1, 10), p(1, 18),
					intLit(1, p(1, 11)),
					intLit(2, p(1, 14)),
					intLit(3, p(1, 17))),
				blockStmt(p(1, 20), p(1, 21)),
				p(1, 1)))
	})

	expectParse(t, "for x, y in z {}", func(p pfn) []Stmt {
		return stmts(
			forInStmt(
				ident("x", p(1, 5)),
				ident("y", p(1, 8)),
				ident("z", p(1, 13)),
				blockStmt(p(1, 15), p(1, 16)),
				p(1, 1)))
	})

	expectParse(t, "for x, y in {k1: 1, k2: 2} {}", func(p pfn) []Stmt {
		return stmts(
			forInStmt(
				ident("x", p(1, 5)),
				ident("y", p(1, 8)),
				dictLit(
					p(1, 13), p(1, 26),
					mapElementLit(
						"k1", p(1, 14), p(1, 16), intLit(1, p(1, 18))),
					mapElementLit(
						"k2", p(1, 21), p(1, 23), intLit(2, p(1, 25)))),
				blockStmt(p(1, 28), p(1, 29)),
				p(1, 1)))
	})

	expectParse(t, "for x in y {} else {}", func(p pfn) []Stmt {
		return stmts(
			forInStmt(
				ident("_", p(1, 5)),
				ident("x", p(1, 5)),
				ident("y", p(1, 10)),
				blockStmt(p(1, 12), p(1, 13)),
				p(1, 1),
				blockStmt(p(1, 20), p(1, 21))))
	})

	expectParse(t, "for x in y do x; else 1 end", func(p pfn) []Stmt {
		return stmts(
			forInStmt(
				ident("_", p(1, 5)),
				ident("x", p(1, 5)),
				ident("y", p(1, 10)),
				blockStmt(
					p(1, 12), p(1, 18),
					exprStmt(
						ident("x", p(1, 15)),
					),
				),
				p(1, 1),
				blockStmt(p(1, 18), p(1, 23),
					exprStmt(
						intLit(1, p(1, 23)),
					),
				)))
	})

	expectParse(t, "for x in y {} else 1 end", func(p pfn) []Stmt {
		return stmts(
			forInStmt(
				ident("_", p(1, 5)),
				ident("x", p(1, 5)),
				ident("y", p(1, 10)),
				blockStmt(p(1, 12), p(1, 13)),
				p(1, 1),
				blockStmt(p(1, 15), p(1, 20),
					exprStmt(
						intLit(1, p(1, 20)),
					),
				)))
	})

	expectParseString(t, "for x in y do end", "for _, x in y {}")
	expectParseString(t, "for x in y do 1 end", "for _, x in y {1}")
	expectParseString(t, "for x in y do else end", "for _, x in y {} else {}")
	expectParseString(t, "for x in y do 1 else end", "for _, x in y {1} else {}")
	expectParseString(t, "for x in y do else end", "for _, x in y {} else {}")
	expectParseString(t, "for x in y do 1 else 2 end", "for _, x in y {1} else {2}")

	expectParseError(t, `for 1 in a {}`)
	expectParseError(t, `for "" in a {}`)
	expectParseError(t, `for k,2 in a {}`)
	expectParseError(t, `for 1,v in a {}`)
}

func TestParseFor(t *testing.T) {
	expectParse(t, "for {}", func(p pfn) []Stmt {
		return stmts(
			forStmt(nil, nil, nil, blockStmt(p(1, 5), p(1, 6)), p(1, 1)))
	})

	expectParse(t, "for a == 5 {}", func(p pfn) []Stmt {
		return stmts(
			forStmt(
				nil,
				binaryExpr(
					ident("a", p(1, 5)),
					intLit(5, p(1, 10)),
					token.Equal,
					p(1, 7)),
				nil,
				blockStmt(p(1, 12), p(1, 13)),
				p(1, 1)))
	})

	expectParse(t, "for a := 0; a == 5;  {}", func(p pfn) []Stmt {
		return stmts(
			forStmt(
				assignStmt(
					exprs(ident("a", p(1, 5))),
					exprs(intLit(0, p(1, 10))),
					token.Define, p(1, 7)),
				binaryExpr(
					ident("a", p(1, 13)),
					intLit(5, p(1, 18)),
					token.Equal,
					p(1, 15)),
				nil,
				blockStmt(p(1, 22), p(1, 23)),
				p(1, 1)))
	})

	expectParse(t, "for a := 0; a < 5; a++ {}", func(p pfn) []Stmt {
		return stmts(
			forStmt(
				assignStmt(
					exprs(ident("a", p(1, 5))),
					exprs(intLit(0, p(1, 10))),
					token.Define, p(1, 7)),
				binaryExpr(
					ident("a", p(1, 13)),
					intLit(5, p(1, 17)),
					token.Less,
					p(1, 15)),
				incDecStmt(
					ident("a", p(1, 20)),
					token.Inc, p(1, 21)),
				blockStmt(p(1, 24), p(1, 25)),
				p(1, 1)))
	})

	expectParse(t, "for ; a < 5; a++ {}", func(p pfn) []Stmt {
		return stmts(
			forStmt(
				nil,
				binaryExpr(
					ident("a", p(1, 7)),
					intLit(5, p(1, 11)),
					token.Less,
					p(1, 9)),
				incDecStmt(
					ident("a", p(1, 14)),
					token.Inc, p(1, 15)),
				blockStmt(p(1, 18), p(1, 19)),
				p(1, 1)))
	})

	expectParse(t, "for a := 0; ; a++ {}", func(p pfn) []Stmt {
		return stmts(
			forStmt(
				assignStmt(
					exprs(ident("a", p(1, 5))),
					exprs(intLit(0, p(1, 10))),
					token.Define, p(1, 7)),
				nil,
				incDecStmt(
					ident("a", p(1, 15)),
					token.Inc, p(1, 16)),
				blockStmt(p(1, 19), p(1, 20)),
				p(1, 1)))
	})

	expectParse(t, "for a == 5 && b != 4 {}", func(p pfn) []Stmt {
		return stmts(
			forStmt(
				nil,
				binaryExpr(
					binaryExpr(
						ident("a", p(1, 5)),
						intLit(5, p(1, 10)),
						token.Equal,
						p(1, 7)),
					binaryExpr(
						ident("b", p(1, 15)),
						intLit(4, p(1, 20)),
						token.NotEqual,
						p(1, 17)),
					token.LAnd,
					p(1, 12)),
				nil,
				blockStmt(p(1, 22), p(1, 23)),
				p(1, 1)))
	})

	expectParse(t, `for { break }`, func(p pfn) []Stmt {
		return stmts(
			forStmt(nil, nil, nil,
				blockStmt(p(1, 5), p(1, 13),
					breakStmt(p(1, 7)),
				),
				p(1, 1)),
		)
	})

	expectParse(t, `for { continue }`, func(p pfn) []Stmt {
		return stmts(
			forStmt(nil, nil, nil,
				blockStmt(p(1, 5), p(1, 16),
					continueStmt(p(1, 7)),
				),
				p(1, 1)),
		)
	})

	expectParseString(t, `for do continue end`, "for {continue}")

	// labels are parsed by parser but not supported by compiler yet
	// expectParseError(t, `for { break x }`)
}

func TestParseClosure(t *testing.T) {
	expectParse(t, "a = (b, c, d) => d", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					ident("a", p(1, 1))),
				exprs(
					closure(
						funcType(p(1, 1), p(1, 5), p(1, 13),
							funcArgs(nil,
								ident("b", p(1, 6)),
								ident("c", p(1, 9)),
								ident("d", p(1, 12))),
						),
						ident("d", p(1, 18)))),
				token.Assign,
				p(1, 3)))
	})

	expectParse(t, "a = (b, c, d) => {d}", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					ident("a", p(1, 1))),
				exprs(
					closure(
						funcType(p(1, 5), p(1, 5), p(1, 13),
							funcArgs(nil,
								ident("b", p(1, 6)),
								ident("c", p(1, 9)),
								ident("d", p(1, 12))),
						),
						blockExpr(p(1, 18), p(1, 20),
							exprStmt(ident("d", p(1, 19)))))),
				token.Assign,
				p(1, 3)))
	})
}

func TestParseFunction(t *testing.T) {
	expectParseString(t, "func(a:int){}", "func(a:int) {}")
	expectParseString(t, "func(a:[int,bool,int]){}", "func(a:[int, bool]) {}")
	expectParseString(t, "func(a:[\n int,\n\tbool]){}", "func(a:[int, bool]) {}")
	expectParseString(t, "func(){}", "func() {}")
	expectParse(t, "func fn (b) { return d }", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				funcLit(
					funcType(p(1, 5), p(1, 9), p(1, 11),
						ident("fn", p(1, 6)),
						funcArgs(nil,
							ident("b", p(1, 10))),
					),
					blockStmt(p(1, 13), p(1, 24),
						returnStmt(p(1, 15), ident("d", p(1, 22)))))))
	})

	expectParse(t, "a = func(b, c, d, e=1, f=2, **g) { return d }", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					ident("a", p(1, 1))),
				exprs(
					funcLit(
						funcType(p(1, 5), p(1, 9), p(1, 32),
							funcArgs(nil,
								ident("b", p(1, 10)),
								ident("c", p(1, 13)),
								ident("d", p(1, 16))),
							funcNamedArgs(
								typedIdent(ident("g", p(1, 31))),
								[]*TypedIdent{
									typedIdent(ident("e", p(1, 19))),
									typedIdent(ident("f", p(1, 24))),
								},
								[]Expr{
									intLit(1, p(1, 21)),
									intLit(2, p(1, 26)),
								}),
						),
						blockStmt(p(1, 34), p(1, 45),
							returnStmt(p(1, 36), ident("d", p(1, 43)))))),
				token.Assign,
				p(1, 3)))
	})
	expectParse(t, "a = func(*args) { return args }", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					ident("a", p(1, 1))),
				exprs(
					funcLit(
						funcType(p(1, 5), p(1, 9), p(1, 15),
							funcArgs(typedIdent(ident("args", p(1, 13))))),
						blockStmt(p(1, 17), p(1, 31),
							returnStmt(p(1, 19),
								ident("args", p(1, 26)),
							),
						),
					),
				),
				token.Assign,
				p(1, 3)))
	})

	expectParse(t, "func(n,a,b,**na) {}", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				funcLit(
					funcType(p(1, 5), p(1, 5), p(1, 16),
						funcArgs(nil,
							ident("n", p(1, 6)),
							ident("a", p(1, 8)),
							ident("b", p(1, 10))),
						funcNamedArgs(typedIdent(ident("na", p(1, 14))), nil, nil),
					),
					blockStmt(p(1, 18), p(1, 19)))),
		)
	})

	expectParse(t, "func(n,a,b,x=1,**na) {}", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				funcLit(
					funcType(p(1, 5), p(1, 5), p(1, 20),
						funcArgs(nil,
							ident("n", p(1, 6)),
							ident("a", p(1, 8)),
							ident("b", p(1, 10))),
						funcNamedArgs(
							typedIdent(ident("na", p(1, 18))),
							[]*TypedIdent{typedIdent(ident("x", p(1, 12)))},
							[]Expr{intLit(1, p(1, 14))}),
					),
					blockStmt(p(1, 22), p(1, 23)))),
		)
	})

	expectParseString(t, "func(){}", "func() {}")
	expectParseString(t, "func(\n){}", "func() {}")
	expectParseString(t, "func(a,){}", "func(a) {}")
	expectParseString(t, "func(\na,\n){}", "func(a) {}")
	expectParseString(t, "func(a,\n){}", "func(a) {}")
	expectParseString(t, "func(\na,\n){}", "func(a) {}")
	expectParseString(t, "func(a,b,\n){}", "func(a, b) {}")
	expectParseString(t, "func(a,\nb,\n){}", "func(a, b) {}")
	expectParseString(t, "func(a,\nb){}", "func(a, b) {}")
	expectParseString(t, "func(a,\nb,){}", "func(a, b) {}")
	expectParseString(t, "func(a,*b){}", "func(a, *b) {}")
	expectParseString(t, "func(a,*b,){}", "func(a, *b) {}")
	expectParseString(t, "func(a,*b,\n){}", "func(a, *b) {}")
	expectParseString(t, "func(a,b,*c,\n){}", "func(a, b, *c) {}")
	expectParseString(t, "func(a,b,\n*c,\n){}", "func(a, b, *c) {}")
	expectParseString(t, "func(\na,\nb,\n*c,\n){}", "func(a, b, *c) {}")

	expectParseString(t, "func(a,kw=2,){}", "func(a, kw=2) {}")
	expectParseString(t, "func(a,*b,c=1,**d\n){}", "func(a, *b, c=1, **d) {}")
	expectParseString(t, "func(\na,\n*b\n\n,\nc=\n\t1,\n\n**d\n \t\n){}", "func(a, *b, c=1, **d) {}")
	expectParseString(t, "func(a,kw=2,){}", "func(a, kw=2) {}")
	expectParseString(t, "func(a,*b,c=1,**d\n){}", "func(a, *b, c=1, **d) {}")
	expectParseString(t, "func(\na,\n*b\n\n,\nc=\n\t1,\n\n**d\n \t\n){}", "func(a, *b, c=1, **d) {}")
	expectParseString(t, "func(a\n,){}", "func(a) {}")
	expectParseString(t, "func(a\n\n,){}", "func(a) {}")
	expectParseString(t, "func(\n*a\n\n,){}", "func(*a) {}")
	expectParseString(t, "func(a\n,*b){}", "func(a, *b) {}")
	expectParseString(t, "func(a\n,*b){}", "func(a, *b) {}")
	expectParseString(t, "func(\na\n,*b){}", "func(a, *b) {}")
	expectParseString(t, "func(*a,\n**b){}", "func(*a, **b) {}")

	expectParseError(t, "func(,){}")
	expectParseError(t, "func(,a){}")
	expectParseError(t, "func(...a,b){}")
	expectParseError(t, "func(a,...b;c=1,...d,...e){}")
}

func TestParseVariadicFunctionWithArgs(t *testing.T) {
	expectParse(t, "a = func(x, y, *z) { return z }", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					ident("a", p(1, 1))),
				exprs(
					funcLit(
						funcType(p(1, 5), p(1, 9), p(1, 18),
							funcArgs(typedIdent(ident("z", p(1, 17))),
								ident("x", p(1, 10)),
								ident("y", p(1, 13)))),
						blockStmt(p(1, 20), p(1, 31),
							returnStmt(p(1, 22),
								ident("z", p(1, 29)),
							),
						),
					),
				),
				token.Assign,
				p(1, 3)))
	})

	expectParseError(t, "a = func(x, y, *z, invalid) { return z }")
	expectParseError(t, "a = func(*args, invalid) { return args }")
	expectParseError(t, "a = func(args*, invalid) { return args }")
}

func TestParseIf(t *testing.T) {
	expectParse(t, "if a == nil {}", func(p pfn) []Stmt {
		return stmts(
			ifStmt(
				nil,
				unaryExpr(ident("a", p(1, 4)),
					token.Null,
					p(1, 6)),
				blockStmt(
					p(1, 13), p(1, 14)),
				nil,
				p(1, 1)))
	})

	expectParse(t, "if a != nil {}", func(p pfn) []Stmt {
		return stmts(
			ifStmt(
				nil,
				unaryExpr(ident("a", p(1, 4)),
					token.NotNull,
					p(1, 6)),
				blockStmt(
					p(1, 13), p(1, 14)),
				nil,
				p(1, 1)))
	})

	expectParse(t, "if a == 5 {}", func(p pfn) []Stmt {
		return stmts(
			ifStmt(
				nil,
				binaryExpr(
					ident("a", p(1, 4)),
					intLit(5, p(1, 9)),
					token.Equal,
					p(1, 6)),
				blockStmt(
					p(1, 11), p(1, 12)),
				nil,
				p(1, 1)))
	})

	expectParse(t, "if a == 5 && b != 3 {}", func(p pfn) []Stmt {
		return stmts(
			ifStmt(
				nil,
				binaryExpr(
					binaryExpr(
						ident("a", p(1, 4)),
						intLit(5, p(1, 9)),
						token.Equal,
						p(1, 6)),
					binaryExpr(
						ident("b", p(1, 14)),
						intLit(3, p(1, 19)),
						token.NotEqual,
						p(1, 16)),
					token.LAnd,
					p(1, 11)),
				blockStmt(
					p(1, 21), p(1, 22)),
				nil,
				p(1, 1)))
	})

	expectParse(t, "if a == 5 { a = 3; a = 1 }", func(p pfn) []Stmt {
		return stmts(
			ifStmt(
				nil,
				binaryExpr(
					ident("a", p(1, 4)),
					intLit(5, p(1, 9)),
					token.Equal,
					p(1, 6)),
				blockStmt(
					p(1, 11), p(1, 26),
					assignStmt(
						exprs(ident("a", p(1, 13))),
						exprs(intLit(3, p(1, 17))),
						token.Assign,
						p(1, 15)),
					assignStmt(
						exprs(ident("a", p(1, 20))),
						exprs(intLit(1, p(1, 24))),
						token.Assign,
						p(1, 22))),
				nil,
				p(1, 1)))
	})

	expectParse(t, "if a == 5 { a = 3; a = 1 } else { a = 2; a = 4 }",
		func(p pfn) []Stmt {
			return stmts(
				ifStmt(
					nil,
					binaryExpr(
						ident("a", p(1, 4)),
						intLit(5, p(1, 9)),
						token.Equal,
						p(1, 6)),
					blockStmt(
						p(1, 11), p(1, 26),
						assignStmt(
							exprs(ident("a", p(1, 13))),
							exprs(intLit(3, p(1, 17))),
							token.Assign,
							p(1, 15)),
						assignStmt(
							exprs(ident("a", p(1, 20))),
							exprs(intLit(1, p(1, 24))),
							token.Assign,
							p(1, 22))),
					blockStmt(
						p(1, 33), p(1, 48),
						assignStmt(
							exprs(ident("a", p(1, 35))),
							exprs(intLit(2, p(1, 39))),
							token.Assign,
							p(1, 37)),
						assignStmt(
							exprs(ident("a", p(1, 42))),
							exprs(intLit(4, p(1, 46))),
							token.Assign,
							p(1, 44))),
					p(1, 1)))
		})

	expectParse(t, `
if a == 5 { 
	b = 3 
	c = 1
} else if d == 3 { 
	e = 8
	f = 3
} else { 
	g = 2
	h = 4
}`, func(p pfn) []Stmt {
		return stmts(
			ifStmt(
				nil,
				binaryExpr(
					ident("a", p(2, 4)),
					intLit(5, p(2, 9)),
					token.Equal,
					p(2, 6)),
				blockStmt(
					p(2, 11), p(5, 1),
					assignStmt(
						exprs(ident("b", p(3, 2))),
						exprs(intLit(3, p(3, 6))),
						token.Assign,
						p(3, 4)),
					assignStmt(
						exprs(ident("c", p(4, 2))),
						exprs(intLit(1, p(4, 6))),
						token.Assign,
						p(4, 4))),
				ifStmt(
					nil,
					binaryExpr(
						ident("d", p(5, 11)),
						intLit(3, p(5, 16)),
						token.Equal,
						p(5, 13)),
					blockStmt(
						p(5, 18), p(8, 1),
						assignStmt(
							exprs(ident("e", p(6, 2))),
							exprs(intLit(8, p(6, 6))),
							token.Assign,
							p(6, 4)),
						assignStmt(
							exprs(ident("f", p(7, 2))),
							exprs(intLit(3, p(7, 6))),
							token.Assign,
							p(7, 4))),
					blockStmt(
						p(8, 8), p(11, 1),
						assignStmt(
							exprs(ident("g", p(9, 2))),
							exprs(intLit(2, p(9, 6))),
							token.Assign,
							p(9, 4)),
						assignStmt(
							exprs(ident("h", p(10, 2))),
							exprs(intLit(4, p(10, 6))),
							token.Assign,
							p(10, 4))),
					p(5, 8)),
				p(2, 1)))
	})

	expectParse(t, "if a := 3; a < b {}", func(p pfn) []Stmt {
		return stmts(
			ifStmt(
				assignStmt(
					exprs(ident("a", p(1, 4))),
					exprs(intLit(3, p(1, 9))),
					token.Define, p(1, 6)),
				binaryExpr(
					ident("a", p(1, 12)),
					ident("b", p(1, 16)),
					token.Less, p(1, 14)),
				blockStmt(
					p(1, 18), p(1, 19)),
				nil,
				p(1, 1)))
	})

	expectParse(t, "if a++; a < b {}", func(p pfn) []Stmt {
		return stmts(
			ifStmt(
				incDecStmt(ident("a", p(1, 4)), token.Inc, p(1, 5)),
				binaryExpr(
					ident("a", p(1, 9)),
					ident("b", p(1, 13)),
					token.Less, p(1, 11)),
				blockStmt(
					p(1, 15), p(1, 16)),
				nil,
				p(1, 1)))
	})

	expectParseString(t, "if a: b", "if a {b}")
	expectParseString(t, "if true; a: b", "if true; a {b}")
	expectParseString(t, "if a: b else: c", "if a {b} else {c}")
	expectParseString(t, "if a: b else if x: c else: d", "if a {b} else if x {c} else {d}")

	expectParseString(t, "if a then end", "if a {}")
	expectParseString(t, "if a then b end", "if a {b}")
	expectParseString(t, "if true; a then b end", "if true; a {b}")
	expectParseString(t, "if a then b; else c end", "if a {b} else {c}")
	expectParseString(t, "if a then b; else if 1 then 2; else c end", "if a {b} else if 1 {2} else {c}")

	expectParseError(t, `if {}`)
	expectParseError(t, `if a == b { } else a != b { }`)
	expectParseError(t, `if a == b { } else if { }`)
	expectParseError(t, `else { }`)
	expectParseError(t, `if ; {}`)
	expectParseError(t, `if a := 3; {}`)
	expectParseError(t, `if ; a < 3 {}`)
}

func TestParseImport(t *testing.T) {
	expectParse(t, `a := import("mod1")`, func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(importExpr("mod1", p(1, 6))),
				token.Define, p(1, 3)))
	})

	expectParse(t, `import("mod1").var1`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				selectorExpr(
					importExpr("mod1", p(1, 1)),
					stringLit("var1", p(1, 16)))))
	})

	expectParse(t, `import("mod1").func1()`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				callExpr(
					selectorExpr(
						importExpr("mod1", p(1, 1)),
						stringLit("func1", p(1, 16))),
					p(1, 21), p(1, 22), NoPos)))
	})

	expectParse(t, `for x, y in import("mod1") {}`, func(p pfn) []Stmt {
		return stmts(
			forInStmt(
				ident("x", p(1, 5)),
				ident("y", p(1, 8)),
				importExpr("mod1", p(1, 13)),
				blockStmt(p(1, 28), p(1, 29)),
				p(1, 1)))
	})

	expectParseError(t, `import(1)`)
	expectParseError(t, `import('a')`)
}

func TestParseIndex(t *testing.T) {
	expectParse(t, "[1, 2, 3][1]", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				indexExpr(
					arrayLit(p(1, 1), p(1, 9),
						intLit(1, p(1, 2)),
						intLit(2, p(1, 5)),
						intLit(3, p(1, 8))),
					intLit(1, p(1, 11)),
					p(1, 10), p(1, 12))))
	})

	expectParse(t, "[1, 2, 3][5 - a]", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				indexExpr(
					arrayLit(p(1, 1), p(1, 9),
						intLit(1, p(1, 2)),
						intLit(2, p(1, 5)),
						intLit(3, p(1, 8))),
					binaryExpr(
						intLit(5, p(1, 11)),
						ident("a", p(1, 15)),
						token.Sub,
						p(1, 13)),
					p(1, 10), p(1, 16))))
	})

	expectParse(t, "[1, 2, 3][5 : a]", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				sliceExpr(
					arrayLit(p(1, 1), p(1, 9),
						intLit(1, p(1, 2)),
						intLit(2, p(1, 5)),
						intLit(3, p(1, 8))),
					intLit(5, p(1, 11)),
					ident("a", p(1, 15)),
					p(1, 10), p(1, 16))))
	})

	expectParse(t, "[1, 2, 3][a + 3 : b - 8]", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				sliceExpr(
					arrayLit(p(1, 1), p(1, 9),
						intLit(1, p(1, 2)),
						intLit(2, p(1, 5)),
						intLit(3, p(1, 8))),
					binaryExpr(
						ident("a", p(1, 11)),
						intLit(3, p(1, 15)),
						token.Add,
						p(1, 13)),
					binaryExpr(
						ident("b", p(1, 19)),
						intLit(8, p(1, 23)),
						token.Sub,
						p(1, 21)),
					p(1, 10), p(1, 24))))
	})

	expectParse(t, `{a: 1, b: 2}["b"]`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				indexExpr(
					dictLit(p(1, 1), p(1, 12),
						mapElementLit(
							"a", p(1, 2), p(1, 3), intLit(1, p(1, 5))),
						mapElementLit(
							"b", p(1, 8), p(1, 9), intLit(2, p(1, 11)))),
					stringLit("b", p(1, 14)),
					p(1, 13), p(1, 17))))
	})

	expectParse(t, `{a: 1, b: 2}[a + b]`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				indexExpr(
					dictLit(p(1, 1), p(1, 12),
						mapElementLit(
							"a", p(1, 2), p(1, 3), intLit(1, p(1, 5))),
						mapElementLit(
							"b", p(1, 8), p(1, 9), intLit(2, p(1, 11)))),
					binaryExpr(
						ident("a", p(1, 14)),
						ident("b", p(1, 18)),
						token.Add,
						p(1, 16)),
					p(1, 13), p(1, 19))))
	})
}

func TestParseLogical(t *testing.T) {
	expectParse(t, "a && 5 || true", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				binaryExpr(
					binaryExpr(
						ident("a", p(1, 1)),
						intLit(5, p(1, 6)),
						token.LAnd,
						p(1, 3)),
					boolLit(true, p(1, 11)),
					token.LOr,
					p(1, 8))))
	})

	expectParse(t, "a || 5 && true", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				binaryExpr(
					ident("a", p(1, 1)),
					binaryExpr(
						intLit(5, p(1, 6)),
						boolLit(true, p(1, 11)),
						token.LAnd,
						p(1, 8)),
					token.LOr,
					p(1, 3))))
	})

	expectParse(t, "a && (5 || true)", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				binaryExpr(
					ident("a", p(1, 1)),
					parenExpr(
						binaryExpr(
							intLit(5, p(1, 7)),
							boolLit(true, p(1, 12)),
							token.LOr,
							p(1, 9)),
						p(1, 6), p(1, 16)),
					token.LAnd,
					p(1, 3))))
	})
}

func TestParseMap(t *testing.T) {
	expectParse(t, "{ key1: 1, key2: \"2\", key3: true }", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				dictLit(p(1, 1), p(1, 34),
					mapElementLit(
						"key1", p(1, 3), p(1, 7), intLit(1, p(1, 9))),
					mapElementLit(
						"key2", p(1, 12), p(1, 16), stringLit("2", p(1, 18))),
					mapElementLit(
						"key3", p(1, 23), p(1, 27), boolLit(true, p(1, 29))))))
	})

	expectParse(t, "{ \"key1\": 1 }", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				dictLit(p(1, 1), p(1, 13),
					mapElementLit(
						"key1", p(1, 3), p(1, 9), intLit(1, p(1, 11))))))
	})

	expectParse(t, "a = { key1: 1, key2: \"2\", key3: true }",
		func(p pfn) []Stmt {
			return stmts(assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(dictLit(p(1, 5), p(1, 38),
					mapElementLit(
						"key1", p(1, 7), p(1, 11), intLit(1, p(1, 13))),
					mapElementLit(
						"key2", p(1, 16), p(1, 20), stringLit("2", p(1, 22))),
					mapElementLit(
						"key3", p(1, 27), p(1, 31), boolLit(true, p(1, 33))))),
				token.Assign,
				p(1, 3)))
		})

	expectParse(t, "a = { key1: 1, key2: \"2\", key3: { k1: `bar`, k2: 4 } }",
		func(p pfn) []Stmt {
			return stmts(assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(dictLit(p(1, 5), p(1, 54),
					mapElementLit(
						"key1", p(1, 7), p(1, 11), intLit(1, p(1, 13))),
					mapElementLit(
						"key2", p(1, 16), p(1, 20), stringLit("2", p(1, 22))),
					mapElementLit(
						"key3", p(1, 27), p(1, 31),
						dictLit(p(1, 33), p(1, 52),
							mapElementLit(
								"k1", p(1, 35),
								p(1, 37), rawStringLit("bar", p(1, 39))),
							mapElementLit(
								"k2", p(1, 46),
								p(1, 48), intLit(4, p(1, 50))))))),
				token.Assign,
				p(1, 3)))
		})

	expectParse(t, `
{
	key1: 1,
	key2: "2",
	key3: true,
}`, func(p pfn) []Stmt {
		return stmts(exprStmt(
			dictLit(p(2, 1), p(6, 1),
				mapElementLit(
					"key1", p(3, 2), p(3, 6), intLit(1, p(3, 8))),
				mapElementLit(
					"key2", p(4, 2), p(4, 6), stringLit("2", p(4, 8))),
				mapElementLit(
					"key3", p(5, 2), p(5, 6), boolLit(true, p(5, 8))))))
	})

	expectParseError(t, "{,}")
	expectParseError(t, "{\n,}")
	expectParseError(t, "{key: 1\n,}")
	expectParseError(t, `
{
	key1: 1,
	key2: "2",
	key3: true
,}`)

	expectParseError(t, `{
key1: 1,
key2: 2
}`)
	expectParseError(t, `{1: 1}`)
}

func TestParsePrecedence(t *testing.T) {
	expectParseString(t, `a + b + c`, `((a + b) + c)`)
	expectParseString(t, `a + b * c`, `(a + (b * c))`)
	expectParseString(t, `x = 2 * 1 + 3 / 4`, `x = ((2 * 1) + (3 / 4))`)
}

func TestParseNullishSelector(t *testing.T) {
	expectParseString(t, `a?.(k)`, `a?.(k)`)
	expectParseString(t, `a?.(k+x)`, `a?.((k + x))`)
	expectParse(t, "a?.b.c?.d", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				nullishSelector(
					selectorExpr(
						nullishSelector(
							ident("a", p(1, 1)),
							stringLit("b", p(1, 4))),
						stringLit("c", p(1, 6))),
					stringLit("d", p(1, 9)))))
	})
	expectParse(t, "a?.b.c?.d.e", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				selectorExpr(
					nullishSelector(
						selectorExpr(
							nullishSelector(
								ident("a", p(1, 1)),
								stringLit("b", p(1, 4))),
							stringLit("c", p(1, 6))),
						stringLit("d", p(1, 9))),
					stringLit("e", p(1, 11)))))
	})
	expectParse(t, "a?.b.c?.d.e?.f.g", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				selectorExpr(
					nullishSelector(
						selectorExpr(
							nullishSelector(
								selectorExpr(
									nullishSelector(
										ident("a", p(1, 1)),
										stringLit("b", p(1, 4))),
									stringLit("c", p(1, 6))),
								stringLit("d", p(1, 9))),
							stringLit("e", p(1, 11))),
						stringLit("f", p(1, 14))),
					stringLit("g", p(1, 16)))))
	})
	expectParse(t, "a?.b?.c", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				nullishSelector(
					nullishSelector(
						ident("a", p(1, 1)),
						stringLit("b", p(1, 4))),
					stringLit("c", p(1, 7)))))
	})
	expectParse(t, "a?.b", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				nullishSelector(
					ident("a", p(1, 1)),
					stringLit("b", p(1, 4)))))
	})
	expectParseString(t, "a?.b", "a?.b")
	expectParseString(t, `a?.b["c"+x]?.d`, `a?.b[("c" + x)]?.d`)
	expectParseString(t, "a?.b.c", "a?.b.c")
	expectParseString(t, "a?.b.c?.d.e?.f.g", "a?.b.c?.d.e?.f.g")
	expectParseString(t, `a["b"+"c"]?.d`, `a[("b" + "c")]?.d`)
	expectParseString(t, `a.b["b"+"c"]?.d`, `a.b[("b" + "c")]?.d`)
	expectParseString(t, `a?.("b"+"c")?.d`, `a?.(("b" + "c"))?.d`)
	expectParseString(t, `d.("a").e`, `d.("a").e`)
	expectParseString(t, `d.("a"+"b").e`, `d.(("a" + "b")).e`)
	expectParseString(t, `d.("a").e ?? 1`, `(d.("a").e ?? 1)`)
	expectParseString(t, `d.("a"+"b").e ?? 1`, `(d.(("a" + "b")).e ?? 1)`)
	expectParseString(t, `a?.("" || "b")?.d.e?.(b ?? "f")`, `a?.(("" || "b"))?.d.e?.((b ?? "f"))`)
	expectParseString(t, `a?.(k)?.c`, `a?.(k)?.c`)
}

func TestParseSelector(t *testing.T) {
	expectParse(t, "a.b", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				selectorExpr(
					ident("a", p(1, 1)),
					stringLit("b", p(1, 3)))))
	})

	expectParse(t, "a.b.c", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				selectorExpr(
					selectorExpr(
						ident("a", p(1, 1)),
						stringLit("b", p(1, 3))),
					stringLit("c", p(1, 5)))))
	})

	expectParse(t, "a.(b).c", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				selectorExpr(
					selectorExpr(
						ident("a", p(1, 1)),
						parenExpr(ident("b", p(1, 4)), p(1, 3), p(1, 5))),
					stringLit("c", p(1, 7)))))
	})

	expectParse(t, "{k1:1}.k1", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				selectorExpr(
					dictLit(
						p(1, 1), p(1, 6),
						mapElementLit(
							"k1", p(1, 2), p(1, 4), intLit(1, p(1, 5)))),
					stringLit("k1", p(1, 8)))))

	})
	expectParse(t, "{k1:{v1:1}}.k1.v1", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				selectorExpr(
					selectorExpr(
						dictLit(
							p(1, 1), p(1, 11),
							mapElementLit("k1", p(1, 2), p(1, 4),
								dictLit(p(1, 5), p(1, 10),
									mapElementLit(
										"v1", p(1, 6),
										p(1, 8), intLit(1, p(1, 9)))))),
						stringLit("k1", p(1, 13))),
					stringLit("v1", p(1, 16)))))
	})

	expectParse(t, "a.b = 4", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					selectorExpr(
						ident("a", p(1, 1)),
						stringLit("b", p(1, 3)))),
				exprs(intLit(4, p(1, 7))),
				token.Assign, p(1, 5)))
	})

	expectParse(t, "a.b.c = 4", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					selectorExpr(
						selectorExpr(
							ident("a", p(1, 1)),
							stringLit("b", p(1, 3))),
						stringLit("c", p(1, 5)))),
				exprs(intLit(4, p(1, 9))),
				token.Assign, p(1, 7)))
	})

	expectParse(t, "a.b.c = 4 + 5", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					selectorExpr(
						selectorExpr(
							ident("a", p(1, 1)),
							stringLit("b", p(1, 3))),
						stringLit("c", p(1, 5)))),
				exprs(
					binaryExpr(
						intLit(4, p(1, 9)),
						intLit(5, p(1, 13)),
						token.Add,
						p(1, 11))),
				token.Assign, p(1, 7)))
	})

	expectParse(t, "a[0].c = 4", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					selectorExpr(
						indexExpr(
							ident("a", p(1, 1)),
							intLit(0, p(1, 3)),
							p(1, 2), p(1, 4)),
						stringLit("c", p(1, 6)))),
				exprs(intLit(4, p(1, 10))),
				token.Assign, p(1, 8)))
	})

	expectParse(t, "a.b[0].c = 4", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					selectorExpr(
						indexExpr(
							selectorExpr(
								ident("a", p(1, 1)),
								stringLit("b", p(1, 3))),
							intLit(0, p(1, 5)),
							p(1, 4), p(1, 6)),
						stringLit("c", p(1, 8)))),
				exprs(intLit(4, p(1, 12))),
				token.Assign, p(1, 10)))
	})

	expectParse(t, "a.b[0][2].c = 4", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					selectorExpr(
						indexExpr(
							indexExpr(
								selectorExpr(
									ident("a", p(1, 1)),
									stringLit("b", p(1, 3))),
								intLit(0, p(1, 5)),
								p(1, 4), p(1, 6)),
							intLit(2, p(1, 8)),
							p(1, 7), p(1, 9)),
						stringLit("c", p(1, 11)))),
				exprs(intLit(4, p(1, 15))),
				token.Assign, p(1, 13)))
	})

	expectParse(t, `a.b["key1"][2].c = 4`, func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					selectorExpr(
						indexExpr(
							indexExpr(
								selectorExpr(
									ident("a", p(1, 1)),
									stringLit("b", p(1, 3))),
								stringLit("key1", p(1, 5)),
								p(1, 4), p(1, 11)),
							intLit(2, p(1, 13)),
							p(1, 12), p(1, 14)),
						stringLit("c", p(1, 16)))),
				exprs(intLit(4, p(1, 20))),
				token.Assign, p(1, 18)))
	})

	expectParse(t, "a[0].b[2].c = 4", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					selectorExpr(
						indexExpr(
							selectorExpr(
								indexExpr(
									ident("a", p(1, 1)),
									intLit(0, p(1, 3)),
									p(1, 2), p(1, 4)),
								stringLit("b", p(1, 6))),
							intLit(2, p(1, 8)),
							p(1, 7), p(1, 9)),
						stringLit("c", p(1, 11)))),
				exprs(intLit(4, p(1, 15))),
				token.Assign, p(1, 13)))
	})
}

func TestParseSemicolon(t *testing.T) {
	expectParse(t, "1", func(p pfn) []Stmt {
		return stmts(
			exprStmt(intLit(1, p(1, 1))))
	})

	expectParse(t, "1;", func(p pfn) []Stmt {
		return stmts(
			exprStmt(intLit(1, p(1, 1))))
	})

	expectParse(t, "1;;", func(p pfn) []Stmt {
		return stmts(
			exprStmt(intLit(1, p(1, 1))))
	})

	expectParse(t, `1
`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(intLit(1, p(1, 1))))
	})

	expectParse(t, `1
;`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(intLit(1, p(1, 1))))
	})

	expectParse(t, `1;
;`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(intLit(1, p(1, 1))))
	})
}

func TestParseString(t *testing.T) {
	expectParse(t, `a = "foo\nbar"`, func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(stringLit("foo\nbar", p(1, 5))),
				token.Assign,
				p(1, 3)))
	})

	expectParse(t, "a = `raw string`", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(rawStringLit("`raw string`", p(1, 5))),
				token.Assign,
				p(1, 3)))
	})
}

func TestParseConfig(t *testing.T) {
	expectParse(t, `# gad: mixed
# gad: mixed=false
a`, func(p pfn) []Stmt {
		return stmts(
			config(p(1, 1), kv(ident("mixed", p(1, 8)))),
			config(p(2, 1), kv(ident("mixed", p(2, 8)), boolLit(false, p(2, 14)))),
			exprStmt(ident("a", p(3, 1))),
		)
	})
	expectParse(t, `# gad: mixed
y
#{b}`, func(p pfn) []Stmt {
		return stmts(
			config(p(1, 1), kv(ident("mixed", p(1, 8)))),
			rawStringStmt(p(2, 1), "y\n"),
			exprStmt(ident("b", p(3, 3))),
		)
	})
	expectParse(t, `# gad: mixed
a
#{b}`, func(p pfn) []Stmt {
		return stmts(
			config(p(1, 1), kv(ident("mixed", p(1, 8)))),
			rawStringStmt(p(2, 1), "a\n"),
			exprStmt(ident("b", p(3, 3))),
		)
	})
	expectParse(t, `# gad: mixed`, func(p pfn) []Stmt {
		return stmts(
			config(p(1, 1), kv(ident("mixed", p(1, 8)))))
	})
}

func TestParseTryThrow(t *testing.T) {
	expectParse(t, `try {} catch e {} finally {}`, func(p pfn) []Stmt {
		return stmts(
			tryStmt(p(1, 1),
				blockStmt(p(1, 5), p(1, 6)),
				catchStmt(p(1, 8), ident("e", p(1, 14)),
					blockStmt(p(1, 16), p(1, 17))),
				finallyStmt(p(1, 19),
					blockStmt(p(1, 27), p(1, 28))),
			),
		)
	})
	expectParse(t, `try {} finally {}`, func(p pfn) []Stmt {
		return stmts(
			tryStmt(p(1, 1),
				blockStmt(p(1, 5), p(1, 6)),
				nil,
				finallyStmt(p(1, 8),
					blockStmt(p(1, 16), p(1, 17))),
			),
		)
	})
	expectParse(t, `try {
} finally {}`, func(p pfn) []Stmt {
		return stmts(
			tryStmt(p(1, 1),
				blockStmt(p(1, 5), p(2, 1)),
				nil,
				finallyStmt(p(2, 3),
					blockStmt(p(2, 11), p(2, 12))),
			),
		)
	})
	expectParse(t, `try {} catch {}`, func(p pfn) []Stmt {
		return stmts(
			tryStmt(p(1, 1),
				blockStmt(p(1, 5), p(1, 6)),
				catchStmt(p(1, 8), nil,
					blockStmt(p(1, 14), p(1, 15))),
				nil,
			),
		)
	})
	expectParse(t, `try {
} catch {}`, func(p pfn) []Stmt {
		return stmts(
			tryStmt(p(1, 1),
				blockStmt(p(1, 5), p(2, 1)),
				catchStmt(p(2, 3), nil,
					blockStmt(p(2, 9), p(2, 10))),
				nil,
			),
		)
	})
	expectParse(t, `throw "error"`, func(p pfn) []Stmt {
		return stmts(
			throwStmt(p(1, 1), stringLit("error", p(1, 7))),
		)
	})
	expectParse(t, `throw 1`, func(p pfn) []Stmt {
		return stmts(
			throwStmt(p(1, 1), intLit(1, p(1, 7))),
		)
	})

	expectParseError(t, `try catch {}`)
	expectParseError(t, `try finally {}`)
	expectParseError(t, `try {} catch;`)
	expectParseError(t, `try {} catch`)
	expectParseError(t, `try {} finally`)
	expectParseError(t, `try {} finally;`)
	expectParseError(t, `try {}
	catch {}`)
	expectParseError(t, `try {}
	finally {}`)
	expectParseError(t, `try {
	} catch {}
	finally {}`)
	expectParseError(t, `throw;`)
	expectParseError(t, `throw`)

	expectParseString(t, `try then catch then finally then end`, "try {} catch {} finally {}")
	expectParseString(t, `try then catch then end`, "try {} catch {}")
}

func TestParseRBraceEOF(t *testing.T) {
	expectParseError(t, `if true {}}`)
	expectParseError(t, `if true {}}else{}`)
	expectParseError(t, `a:=1; if true {}}else{}`)
	expectParseError(t, `if true {}} else{} return`)
	expectParseError(t, `if true {} else if true {}{`)
	expectParseError(t, `
if true {

}
} else{

}

return`)
}

func TestParseLinesSep(t *testing.T) {
	expectParseString(t, "\r\r1+\r\r2+\r\r\r3\r\r\n  \t", `((1 + 2) + 3)`)
	expectParseString(t, "1+\n2+\n3", `((1 + 2) + 3)`)
	expectParseString(t, "1+\r\n2+\n3", `((1 + 2) + 3)`)
	expectParseString(t, "1+\r2+\n3", `((1 + 2) + 3)`)
	expectParseString(t, "1+\r2+\r3", `((1 + 2) + 3)`)
	expectParseString(t, "\r\r1+\r2+\r3", `((1 + 2) + 3)`)
	expectParseString(t, "\r\r1+\r\r2+\r\r\r3", `((1 + 2) + 3)`)
}

type pfn func(int, int) Pos          // position conversion function
type expectedFn func(pos pfn) []Stmt // callback function to return expected results

type parseTracer struct {
	out []string
}

func (o *parseTracer) Write(p []byte) (n int, err error) {
	o.out = append(o.out, string(p))
	return len(p), nil
}

func expectParse(t *testing.T, input string, fn expectedFn) {
	expectParseMode(t, 0, input, fn)
}

func expectParseMode(t *testing.T, mode Mode, input string, fn expectedFn) {
	testFileSet := NewFileSet()
	testFile := testFileSet.AddFile("test", -1, len(input))

	var ok bool
	defer func() {
		if !ok {
			// print Trace
			tr := &parseTracer{}
			p := NewParser(testFile, []byte(input), tr)
			actual, _ := p.ParseFile()
			if actual != nil {
				t.Logf("Parsed:\n%s", actual.String())
			}
			t.Logf("Trace:\n%s", strings.Join(tr.out, ""))
		}
	}()

	p := NewParserWithOptions(testFile, []byte(input), &ParserOptions{
		Mode: mode,
	}, nil)
	actual, err := p.ParseFile()
	require.NoError(t, err)

	expected := fn(func(line, column int) Pos {
		return Pos(int(testFile.LineStart(line)) + (column - 1))
	})
	require.Equal(t, len(expected), len(actual.Stmts))

	for i := 0; i < len(expected); i++ {
		equalStmt(t, expected[i], actual.Stmts[i])
	}

	ok = true
}

func expectParseError(t *testing.T, input string) {
	testFileSet := NewFileSet()
	testFile := testFileSet.AddFile("test", -1, len(input))

	var ok bool
	defer func() {
		if !ok {
			// print Trace
			tr := &parseTracer{}
			p := NewParser(testFile, []byte(input), tr)
			_, _ = p.ParseFile()
			t.Logf("Trace:\n%s", strings.Join(tr.out, ""))
		}
	}()

	p := NewParser(testFile, []byte(input), nil)
	_, err := p.ParseFile()
	require.Error(t, err)
	ok = true
}

func expectParseString(t *testing.T, input, expected string) {
	expectParseStringMode(t, 0, input, expected)
}

func expectParseStringMode(t *testing.T, mode Mode, input, expected string) {
	var ok bool
	defer func() {
		if !ok {
			// print Trace
			tr := &parseTracer{}
			_, _ = parseSource("test", []byte(input), tr, mode)
			t.Logf("Trace:\n%s", strings.Join(tr.out, ""))
		}
	}()

	actual, err := parseSource("test", []byte(input), nil, mode)
	require.NoError(t, err)
	require.Equal(t, expected, actual.String())
	ok = true
}

func stmts(s ...Stmt) []Stmt {
	return s
}

func exprStmt(x Expr) *ExprStmt {
	return &ExprStmt{Expr: x}
}

func declStmt(decl Decl) *DeclStmt {
	return &DeclStmt{Decl: decl}
}

func genDecl(
	tok token.Token,
	tokPos, lparen, rparen Pos,
	specs ...Spec,
) Decl {
	return &GenDecl{
		Tok:    tok,
		TokPos: tokPos,
		Lparen: lparen,
		Specs:  specs,
		Rparen: rparen,
	}
}

func paramSpec(variadic bool, ident *TypedIdent) Spec {
	return &ParamSpec{
		Ident:    ident,
		Variadic: variadic,
	}
}

func nparamSpec(ident *TypedIdent, value Expr) Spec {
	return &NamedParamSpec{
		Ident: ident,
		Value: value,
	}
}

func valueSpec(idents []*Ident, values []Expr) Spec {
	return &ValueSpec{
		Idents: idents,
		Values: values,
	}
}

func assignStmt(
	lhs, rhs []Expr,
	token token.Token,
	pos Pos,
) *AssignStmt {
	return &AssignStmt{LHS: lhs, RHS: rhs, Token: token, TokenPos: pos}
}

func returnStmt(pos Pos, result Expr) *ReturnStmt {
	return &ReturnStmt{Result: result, ReturnPos: pos}
}

func forStmt(
	init Stmt,
	cond Expr,
	post Stmt,
	body *BlockStmt,
	pos Pos,
) *ForStmt {
	return &ForStmt{
		Cond: cond, Init: init, Post: post, Body: body, ForPos: pos,
	}
}

func forInStmt(
	key, value *Ident,
	seq Expr,
	body *BlockStmt,
	pos Pos,
	elseb ...*BlockStmt,
) *ForInStmt {
	f := &ForInStmt{
		Key: key, Value: value, Iterable: seq, Body: body, ForPos: pos,
	}
	for _, f.Else = range elseb {
	}
	return f
}

func breakStmt(pos Pos) *BranchStmt {
	return &BranchStmt{
		Token:    token.Break,
		TokenPos: pos,
	}
}

func continueStmt(pos Pos) *BranchStmt {
	return &BranchStmt{
		Token:    token.Continue,
		TokenPos: pos,
	}
}

func ifStmt(
	init Stmt,
	cond Expr,
	body *BlockStmt,
	elseStmt Stmt,
	pos Pos,
) *IfStmt {
	return &IfStmt{
		Init: init, Cond: cond, Body: body, Else: elseStmt, IfPos: pos,
	}
}

func tryStmt(
	tryPos Pos,
	body *BlockStmt,
	catch *CatchStmt,
	finally *FinallyStmt,
) *TryStmt {
	return &TryStmt{TryPos: tryPos, Body: body, Catch: catch, Finally: finally}
}

func catchStmt(
	catchPos Pos,
	ident *Ident,
	body *BlockStmt,
) *CatchStmt {
	return &CatchStmt{CatchPos: catchPos, Ident: ident, Body: body}
}

func finallyStmt(
	finallyPos Pos,
	body *BlockStmt,
) *FinallyStmt {
	return &FinallyStmt{FinallyPos: finallyPos, Body: body}
}

func throwStmt(
	throwPos Pos,
	expr Expr,
) *ThrowStmt {
	return &ThrowStmt{ThrowPos: throwPos, Expr: expr}
}

func incDecStmt(
	expr Expr,
	tok token.Token,
	pos Pos,
) *IncDecStmt {
	return &IncDecStmt{Expr: expr, Token: tok, TokenPos: pos}
}

func funcType(pos, lparen, rparen Pos, v ...any) *FuncType {
	f := &FuncType{Params: FuncParams{LParen: lparen, RParen: rparen}, FuncPos: pos}
	for _, v := range v {
		switch t := v.(type) {
		case ArgsList:
			f.Params.Args = t
		case NamedArgsList:
			f.Params.NamedArgs = t
		case *Ident:
			f.Ident = t
		}
	}
	return f
}

func funcArgs(vari *TypedIdent, names ...Expr) ArgsList {
	l := ArgsList{Var: vari}
	for _, name := range names {
		switch t := name.(type) {
		case *Ident:
			l.Values = append(l.Values, typedIdent(t))
		case *TypedIdent:
			l.Values = append(l.Values, t)
		}
	}
	return l
}

func funcNamedArgs(vari *TypedIdent, names []*TypedIdent, values []Expr) NamedArgsList {
	return NamedArgsList{Names: names, Var: vari, Values: values}
}

func blockStmt(lbrace, rbrace Pos, list ...Stmt) *BlockStmt {
	return &BlockStmt{Stmts: list, LBrace: lbrace, RBrace: rbrace}
}

func blockExpr(lbrace, rbrace Pos, list ...Stmt) *BlockExpr {
	return &BlockExpr{BlockStmt: &BlockStmt{Stmts: list, LBrace: lbrace, RBrace: rbrace}}
}

func ident(name string, pos Pos) *Ident {
	return &Ident{Name: name, NamePos: pos}
}

func typedIdent(ident *Ident, typ ...*Ident) *TypedIdent {
	return &TypedIdent{Ident: ident, Type: typ}
}

func rawStringStmt(pos Pos, lit string) *RawStringStmt {
	return &RawStringStmt{MixedExprRune: '#', Lits: []*RawStringLit{{LiteralPos: pos, Literal: lit}}}
}

func toText(start, end Literal, expr Expr) *ExprToTextStmt {
	return &ExprToTextStmt{Expr: expr, StartLit: start, EndLit: end}
}

func lit(value string, pos Pos) Literal {
	return Literal{Value: value, Pos: pos}
}

func kv(key Expr, value ...Expr) *KeyValueLit {
	kv := &KeyValueLit{Key: key}
	for _, expr := range value {
		kv.Value = expr
	}
	return kv
}

func config(start Pos, opts ...*KeyValueLit) *ConfigStmt {
	c := &ConfigStmt{ConfigPos: start, Elements: opts}
	c.ParseElements()
	return c
}

func nullishSelector(
	sel,
	expr Expr,
) *NullishSelectorExpr {
	return &NullishSelectorExpr{Expr: sel, Sel: expr}
}

func binaryExpr(
	x, y Expr,
	op token.Token,
	pos Pos,
) *BinaryExpr {
	return &BinaryExpr{LHS: x, RHS: y, Token: op, TokenPos: pos}
}

func condExpr(
	cond, trueExpr, falseExpr Expr,
	questionPos, colonPos Pos,
) *CondExpr {
	return &CondExpr{
		Cond: cond, True: trueExpr, False: falseExpr,
		QuestionPos: questionPos, ColonPos: colonPos,
	}
}

func unaryExpr(x Expr, op token.Token, pos Pos) *UnaryExpr {
	return &UnaryExpr{Expr: x, Token: op, TokenPos: pos}
}

func importExpr(moduleName string, pos Pos) *ImportExpr {
	return &ImportExpr{
		ModuleName: moduleName, Token: token.Import, TokenPos: pos,
	}
}

func exprs(list ...Expr) []Expr {
	return list
}

func intLit(value int64, pos Pos) *IntLit {
	return &IntLit{Value: value, ValuePos: pos}
}

func floatLit(value float64, pos Pos) *FloatLit {
	return &FloatLit{Value: value, ValuePos: pos}
}

func decimalLit(value string, pos Pos) *DecimalLit {
	v, _ := decimal.NewFromString(value)
	return &DecimalLit{Value: v, ValuePos: pos}
}

func stringLit(value string, pos Pos) *StringLit {
	return &StringLit{Value: value, ValuePos: pos}
}

func rawStringLit(value string, pos Pos) *RawStringLit {
	return &RawStringLit{Literal: value, LiteralPos: pos, Quoted: value[0] == '`'}
}

func charLit(value rune, pos Pos) *CharLit {
	return &CharLit{
		Value: value, ValuePos: pos, Literal: fmt.Sprintf("'%c'", value),
	}
}

func boolLit(value bool, pos Pos) *BoolLit {
	return &BoolLit{Value: value, ValuePos: pos}
}

func flagLit(value bool, pos Pos) *FlagLit {
	return &FlagLit{Value: value, ValuePos: pos}
}

func arrayLit(lbracket, rbracket Pos, list ...Expr) *ArrayLit {
	return &ArrayLit{LBrack: lbracket, RBrack: rbracket, Elements: list}
}

func caleeKw(pos Pos) *CalleeKeyword {
	return &CalleeKeyword{TokenPos: pos, Literal: token.Callee.String()}
}

func argsKw(pos Pos) *ArgsKeyword {
	return &ArgsKeyword{TokenPos: pos, Literal: token.Args.String()}
}

func nargsKw(pos Pos) *NamedArgsKeyword {
	return &NamedArgsKeyword{TokenPos: pos, Literal: token.NamedArgs.String()}
}

func mapElementLit(
	key string,
	keyPos Pos,
	colonPos Pos,
	value Expr,
) *DictElementLit {
	return &DictElementLit{
		Key: key, KeyPos: keyPos, ColonPos: colonPos, Value: value,
	}
}

func dictLit(
	lbrace, rbrace Pos,
	list ...*DictElementLit,
) *DictLit {
	return &DictLit{LBrace: lbrace, RBrace: rbrace, Elements: list}
}

func funcLit(funcType *FuncType, body *BlockStmt) *FuncLit {
	return &FuncLit{Type: funcType, Body: body}
}

func closure(funcType *FuncType, body Expr) *ClosureLit {
	return &ClosureLit{Type: funcType, Body: body}
}

func parenExpr(x Expr, lparen, rparen Pos) *ParenExpr {
	return &ParenExpr{Expr: x, LParen: lparen, RParen: rparen}
}

func callExpr(
	f Expr,
	lparen, rparen Pos,
	args ...any,
) (ce *CallExpr) {
	ce = &CallExpr{Func: f, CallArgs: CallArgs{LParen: lparen, RParen: rparen}}
	for _, v := range args {
		switch t := v.(type) {
		case CallExprArgs:
			ce.Args = t
		case CallExprNamedArgs:
			ce.NamedArgs = t
		}
	}
	return ce
}

func argVar(pos Pos, value Expr) *ArgVarLit {
	return &ArgVarLit{TokenPos: pos, Value: value}
}

func nargVar(pos Pos, value Expr) *NamedArgVarLit {
	return &NamedArgVarLit{TokenPos: pos, Value: value}
}

func callExprArgs(
	argVar *ArgVarLit,
	args ...Expr,
) (ce CallExprArgs) {
	return CallExprArgs{Var: argVar, Values: args}
}

func callExprNamedArgs(
	argVar *NamedArgVarLit,
	names []NamedArgExpr, values []Expr,
) (ce CallExprNamedArgs) {
	return CallExprNamedArgs{Var: argVar, Names: names, Values: values}
}

func indexExpr(
	x, index Expr,
	lbrack, rbrack Pos,
) *IndexExpr {
	return &IndexExpr{
		Expr: x, Index: index, LBrack: lbrack, RBrack: rbrack,
	}
}

func sliceExpr(
	x, low, high Expr,
	lbrack, rbrack Pos,
) *SliceExpr {
	return &SliceExpr{
		Expr: x, Low: low, High: high, LBrack: lbrack, RBrack: rbrack,
	}
}

func selectorExpr(x, sel Expr) *SelectorExpr {
	return &SelectorExpr{Expr: x, Sel: sel}
}

func equalStmt(t *testing.T, expected, actual Stmt) {
	if expected == nil || reflect.ValueOf(expected).IsNil() {
		require.Nil(t, actual, "expected nil, but got not nil")
		return
	}
	require.NotNil(t, actual, "expected not nil, but got nil")
	require.IsType(t, expected, actual)

	switch expected := expected.(type) {
	case *ExprStmt:
		equalExpr(t, expected.Expr, actual.(*ExprStmt).Expr)
	case *EmptyStmt:
		require.Equal(t, expected.Implicit,
			actual.(*EmptyStmt).Implicit)
		require.Equal(t, expected.Semicolon,
			actual.(*EmptyStmt).Semicolon)
	case *BlockStmt:
		require.Equal(t, expected.LBrace,
			actual.(*BlockStmt).LBrace)
		require.Equal(t, expected.RBrace,
			actual.(*BlockStmt).RBrace)
		equalStmts(t, expected.Stmts,
			actual.(*BlockStmt).Stmts)
	case *AssignStmt:
		equalExprs(t, expected.LHS,
			actual.(*AssignStmt).LHS)
		equalExprs(t, expected.RHS,
			actual.(*AssignStmt).RHS)
		require.Equal(t, int(expected.Token),
			int(actual.(*AssignStmt).Token))
		require.Equal(t, int(expected.TokenPos),
			int(actual.(*AssignStmt).TokenPos))
	case *DeclStmt:
		expectedDecl := expected.Decl.(*GenDecl)
		actualDecl := actual.(*DeclStmt).Decl.(*GenDecl)
		require.Equal(t, expectedDecl.Tok, actualDecl.Tok)
		require.Equal(t, expectedDecl.TokPos, actualDecl.TokPos)
		require.Equal(t, expectedDecl.Lparen, actualDecl.Lparen)
		require.Equal(t, expectedDecl.Rparen, actualDecl.Rparen)
		require.Equal(t, len(expectedDecl.Specs), len(actualDecl.Specs))
		for i, expSpec := range expectedDecl.Specs {
			actSpec := actualDecl.Specs[i]
			switch expectedSpec := expSpec.(type) {
			case *ParamSpec:
				actualSpec, ok := actSpec.(*ParamSpec)
				if !ok {
					require.Failf(t, "type error", "expected *ParamSpec, got %T", actSpec)
					return
				}
				require.Equal(t, expectedSpec.Ident, actualSpec.Ident)
				require.Equal(t, expectedSpec.Variadic, actualSpec.Variadic)
			case *NamedParamSpec:
				actualSpec, ok := actSpec.(*NamedParamSpec)
				if !ok {
					require.Failf(t, "type error", "expected *NamedParamSpec, got %T", actSpec)
					return
				}
				require.Equal(t, expectedSpec.Ident, actualSpec.Ident)
				if expectedSpec.Value != nil || actualSpec.Value != nil {
					equalExpr(t, expectedSpec.Value, actualSpec.Value)
				}
			case *ValueSpec:
				actualSpec, ok := actSpec.(*ValueSpec)
				if !ok {
					require.Failf(t, "type error", "expected *ValueSpec, got %T", actSpec)
					return
				}
				require.Equal(t, expectedSpec.Idents, actualSpec.Idents)
				require.Equal(t, len(expectedSpec.Values), len(actualSpec.Values))
				if len(expectedSpec.Values) == len(actualSpec.Values) {
					for i, expr := range expectedSpec.Values {
						equalExpr(t, expr, actualSpec.Values[i])
					}
				}
			default:
				require.Failf(t, "unknown type", "unknown Spec '%T'", expSpec)
			}
		}
	case *IfStmt:
		equalStmt(t, expected.Init, actual.(*IfStmt).Init)
		equalExpr(t, expected.Cond, actual.(*IfStmt).Cond)
		equalStmt(t, expected.Body, actual.(*IfStmt).Body)
		equalStmt(t, expected.Else, actual.(*IfStmt).Else)
		require.Equal(t, expected.IfPos, actual.(*IfStmt).IfPos)
	case *TryStmt:
		require.Equal(t, expected.TryPos, actual.(*TryStmt).TryPos)
		equalStmt(t, expected.Body, actual.(*TryStmt).Body)
		equalStmt(t, expected.Catch, actual.(*TryStmt).Catch)
		equalStmt(t, expected.Finally, actual.(*TryStmt).Finally)
	case *CatchStmt:
		require.Equal(t, expected.CatchPos, actual.(*CatchStmt).CatchPos)
		require.Equal(t, expected.Ident, actual.(*CatchStmt).Ident)
		equalStmt(t, expected.Body, actual.(*CatchStmt).Body)
	case *FinallyStmt:
		require.Equal(t, expected.FinallyPos, actual.(*FinallyStmt).FinallyPos)
		equalStmt(t, expected.Body, actual.(*FinallyStmt).Body)
	case *ThrowStmt:
		require.Equal(t, expected.ThrowPos, actual.(*ThrowStmt).ThrowPos)
		equalExpr(t, expected.Expr, actual.(*ThrowStmt).Expr)
	case *IncDecStmt:
		equalExpr(t, expected.Expr,
			actual.(*IncDecStmt).Expr)
		require.Equal(t, expected.Token,
			actual.(*IncDecStmt).Token)
		require.Equal(t, expected.TokenPos,
			actual.(*IncDecStmt).TokenPos)
	case *ForStmt:
		equalStmt(t, expected.Init, actual.(*ForStmt).Init)
		equalExpr(t, expected.Cond, actual.(*ForStmt).Cond)
		equalStmt(t, expected.Post, actual.(*ForStmt).Post)
		equalStmt(t, expected.Body, actual.(*ForStmt).Body)
		require.Equal(t, expected.ForPos, actual.(*ForStmt).ForPos)
	case *ForInStmt:
		equalExpr(t, expected.Key,
			actual.(*ForInStmt).Key)
		equalExpr(t, expected.Value,
			actual.(*ForInStmt).Value)
		equalExpr(t, expected.Iterable,
			actual.(*ForInStmt).Iterable)
		equalStmt(t, expected.Body,
			actual.(*ForInStmt).Body)
		require.Equal(t, expected.ForPos,
			actual.(*ForInStmt).ForPos)
		equalStmt(t, expected.Else,
			actual.(*ForInStmt).Else)
	case *ReturnStmt:
		equalExpr(t, expected.Result,
			actual.(*ReturnStmt).Result)
		require.Equal(t, expected.ReturnPos,
			actual.(*ReturnStmt).ReturnPos)
	case *BranchStmt:
		equalExpr(t, expected.Label,
			actual.(*BranchStmt).Label)
		require.Equal(t, expected.Token,
			actual.(*BranchStmt).Token)
		require.Equal(t, expected.TokenPos,
			actual.(*BranchStmt).TokenPos)
	case *RawStringStmt:
		require.Equal(t, len(expected.Lits), len(actual.(*RawStringStmt).Lits))
		for i, lit := range expected.Lits {
			equalExpr(t, lit, actual.(*RawStringStmt).Lits[i])
		}
	case *ExprToTextStmt:
		require.Equal(t, expected.StartLit.Value,
			actual.(*ExprToTextStmt).StartLit.Value)
		require.Equal(t, expected.StartLit.Pos,
			actual.(*ExprToTextStmt).StartLit.Pos)
		require.Equal(t, expected.EndLit.Value,
			actual.(*ExprToTextStmt).EndLit.Value)
		require.Equal(t, expected.EndLit.Pos,
			actual.(*ExprToTextStmt).EndLit.Pos)
		equalExpr(t, expected.Expr,
			actual.(*ExprToTextStmt).Expr)
	case *ConfigStmt:
		require.Equal(t, expected.ConfigPos,
			actual.(*ConfigStmt).ConfigPos)
		require.Equal(t, expected.Options,
			actual.(*ConfigStmt).Options)
		require.Equal(t, len(expected.Elements),
			len(actual.(*ConfigStmt).Elements))
		for i, e := range expected.Elements {
			equalExpr(t, e, actual.(*ConfigStmt).Elements[i])
		}
	default:
		panic(fmt.Errorf("unknown type: %T", expected))
	}
}

func equalExpr(t *testing.T, expected, actual Expr) {
	if expected == nil || reflect.ValueOf(expected).IsNil() {
		require.Nil(t, actual, "expected nil, but got not nil")
		return
	}
	require.NotNil(t, actual, "expected not nil, but got nil")
	require.IsType(t, expected, actual)

	switch expected := expected.(type) {
	case *Ident:
		require.Equal(t, expected.Name,
			actual.(*Ident).Name)
		require.Equal(t, int(expected.NamePos),
			int(actual.(*Ident).NamePos))
	case *TypedIdent:
		equalExpr(t, expected.Ident, actual.(*TypedIdent).Ident)
		equalIdents(t, expected.Type, actual.(*TypedIdent).Type)
	case *IntLit:
		require.Equal(t, expected.Value,
			actual.(*IntLit).Value)
		require.Equal(t, int(expected.ValuePos),
			int(actual.(*IntLit).ValuePos))
	case *FloatLit:
		require.Equal(t, expected.Value,
			actual.(*FloatLit).Value)
		require.Equal(t, int(expected.ValuePos),
			int(actual.(*FloatLit).ValuePos))
	case *DecimalLit:
		require.True(t, expected.Value.Equal(actual.(*DecimalLit).Value))
		require.Equal(t, int(expected.ValuePos),
			int(actual.(*DecimalLit).ValuePos))
	case *BoolLit:
		require.Equal(t, expected.Value,
			actual.(*BoolLit).Value)
		require.Equal(t, int(expected.ValuePos),
			int(actual.(*BoolLit).ValuePos))
	case *FlagLit:
		require.Equal(t, expected.Value,
			actual.(*FlagLit).Value)
		require.Equal(t, int(expected.ValuePos),
			int(actual.(*FlagLit).ValuePos))
	case *CharLit:
		require.Equal(t, expected.Value,
			actual.(*CharLit).Value)
		require.Equal(t, int(expected.ValuePos),
			int(actual.(*CharLit).ValuePos))
	case *StringLit:
		require.Equal(t, expected.Value,
			actual.(*StringLit).Value)
		require.Equal(t, int(expected.ValuePos),
			int(actual.(*StringLit).ValuePos))
	case *RawStringLit:
		require.Equal(t, expected.UnquotedValue(),
			actual.(*RawStringLit).UnquotedValue())
		require.Equal(t, int(expected.LiteralPos),
			int(actual.(*RawStringLit).LiteralPos))
	case *ArrayLit:
		require.Equal(t, expected.LBrack,
			actual.(*ArrayLit).LBrack)
		require.Equal(t, expected.RBrack,
			actual.(*ArrayLit).RBrack)
		equalExprs(t, expected.Elements,
			actual.(*ArrayLit).Elements)
	case *DictLit:
		require.Equal(t, expected.LBrace,
			actual.(*DictLit).LBrace)
		require.Equal(t, expected.RBrace,
			actual.(*DictLit).RBrace)
		equalMapElements(t, expected.Elements,
			actual.(*DictLit).Elements)
	case *NilLit:
		require.Equal(t, expected.TokenPos,
			actual.(*NilLit).TokenPos)
	case *NullishSelectorExpr:
		equalExpr(t, expected.Expr,
			actual.(*NullishSelectorExpr).Expr)
		equalExpr(t, expected.Sel,
			actual.(*NullishSelectorExpr).Sel)
	case *BinaryExpr:
		equalExpr(t, expected.LHS,
			actual.(*BinaryExpr).LHS)
		equalExpr(t, expected.RHS,
			actual.(*BinaryExpr).RHS)
		require.Equal(t, expected.Token,
			actual.(*BinaryExpr).Token)
		require.Equal(t, expected.TokenPos,
			actual.(*BinaryExpr).TokenPos)
	case *UnaryExpr:
		equalExpr(t, expected.Expr,
			actual.(*UnaryExpr).Expr)
		require.Equal(t, expected.Token,
			actual.(*UnaryExpr).Token)
		require.Equal(t, expected.TokenPos,
			actual.(*UnaryExpr).TokenPos)
	case *FuncLit:
		equalFuncType(t, expected.Type,
			actual.(*FuncLit).Type)
		equalStmt(t, expected.Body,
			actual.(*FuncLit).Body)
	case *CallExpr:
		actual := actual.(*CallExpr)
		equalExpr(t, expected.Func,
			actual.Func)
		require.Equal(t, expected.LParen,
			actual.LParen)
		require.Equal(t, expected.RParen,
			actual.RParen)
		equalExprs(t, expected.Args.Values,
			actual.Args.Values)

		if expected.Args.Var == nil && actual.Args.Var != nil {
			require.Nil(t, expected.Args.Var)
		}

		if expected.Args.Var != nil && actual.Args.Var == nil {
			require.NotNil(t, expected.Args.Var)
		}

		if expected.Args.Var != nil && actual.Args.Var != nil {
			require.Equal(t, expected.Args.Var.TokenPos,
				actual.Args.Var.TokenPos)
			equalExpr(t, expected.Args.Var.Value,
				actual.Args.Var.Value)
		}

		if expected.NamedArgs.Var == nil && actual.NamedArgs.Var != nil {
			require.Nil(t, expected.NamedArgs.Var)
		}

		if expected.NamedArgs.Var != nil && actual.NamedArgs.Var == nil {
			require.NotNil(t, expected.NamedArgs.Var)
		}

		if expected.NamedArgs.Var != nil && actual.NamedArgs.Var != nil {
			require.Equal(t, expected.NamedArgs.Var.TokenPos,
				actual.NamedArgs.Var.TokenPos)
			equalExpr(t, expected.NamedArgs.Var.Value,
				actual.NamedArgs.Var.Value)
		}

		equalNamedArgsNames(t, expected.NamedArgs.Names,
			actual.NamedArgs.Names)
		equalExprs(t, expected.NamedArgs.Values,
			actual.NamedArgs.Values)
	case *ParenExpr:
		equalExpr(t, expected.Expr,
			actual.(*ParenExpr).Expr)
		require.Equal(t, expected.LParen,
			actual.(*ParenExpr).LParen)
		require.Equal(t, expected.RParen,
			actual.(*ParenExpr).RParen)
	case *IndexExpr:
		equalExpr(t, expected.Expr,
			actual.(*IndexExpr).Expr)
		equalExpr(t, expected.Index,
			actual.(*IndexExpr).Index)
		require.Equal(t, expected.LBrack,
			actual.(*IndexExpr).LBrack)
		require.Equal(t, expected.RBrack,
			actual.(*IndexExpr).RBrack)
	case *SliceExpr:
		equalExpr(t, expected.Expr,
			actual.(*SliceExpr).Expr)
		equalExpr(t, expected.Low,
			actual.(*SliceExpr).Low)
		equalExpr(t, expected.High,
			actual.(*SliceExpr).High)
		require.Equal(t, expected.LBrack,
			actual.(*SliceExpr).LBrack)
		require.Equal(t, expected.RBrack,
			actual.(*SliceExpr).RBrack)
	case *SelectorExpr:
		equalExpr(t, expected.Expr,
			actual.(*SelectorExpr).Expr)
		equalExpr(t, expected.Sel,
			actual.(*SelectorExpr).Sel)
	case *ImportExpr:
		require.Equal(t, expected.ModuleName,
			actual.(*ImportExpr).ModuleName)
		require.Equal(t, int(expected.TokenPos),
			int(actual.(*ImportExpr).TokenPos))
		require.Equal(t, expected.Token,
			actual.(*ImportExpr).Token)
	case *CondExpr:
		equalExpr(t, expected.Cond,
			actual.(*CondExpr).Cond)
		equalExpr(t, expected.True,
			actual.(*CondExpr).True)
		equalExpr(t, expected.False,
			actual.(*CondExpr).False)
		require.Equal(t, expected.QuestionPos,
			actual.(*CondExpr).QuestionPos)
		require.Equal(t, expected.ColonPos,
			actual.(*CondExpr).ColonPos)
	case *CalleeKeyword:
		require.Equal(t, expected.Literal,
			actual.(*CalleeKeyword).Literal)
		require.Equal(t, expected.TokenPos,
			actual.(*CalleeKeyword).TokenPos)
	case *ArgsKeyword:
		require.Equal(t, expected.Literal,
			actual.(*ArgsKeyword).Literal)
		require.Equal(t, expected.TokenPos,
			actual.(*ArgsKeyword).TokenPos)
	case *NamedArgsKeyword:
		require.Equal(t, expected.Literal,
			actual.(*NamedArgsKeyword).Literal)
		require.Equal(t, expected.TokenPos,
			actual.(*NamedArgsKeyword).TokenPos)
	case *ClosureLit:
		equalFuncType(t, expected.Type,
			actual.(*ClosureLit).Type)
		equalExpr(t, expected.Body,
			actual.(*ClosureLit).Body)
	case *BlockExpr:
		equalStmt(t, expected.BlockStmt,
			actual.(*BlockExpr).BlockStmt)
	case *KeyValueLit:
		equalExpr(t, expected.Key,
			actual.(*KeyValueLit).Key)
		equalExpr(t, expected.Value,
			actual.(*KeyValueLit).Value)
	default:
		panic(fmt.Errorf("unknown type: %T", expected))
	}
}

func equalFuncType(t *testing.T, expected, actual *FuncType) {
	require.Equal(t, expected.Params.LParen, actual.Params.LParen)
	require.Equal(t, expected.Params.RParen, actual.Params.RParen)
	equalTypedIdents(t, expected.Params.Args.Values, actual.Params.Args.Values)
	equalNamedArgs(t, &expected.Params.NamedArgs, &actual.Params.NamedArgs)
}

func equalNamedArgs(t *testing.T, expected, actual *NamedArgsList) {
	if expected == nil && actual == nil {
		return
	}
	require.NotNil(t, expected, "expected is nil")
	require.NotNil(t, actual, "actual is nil")

	require.Equal(t, expected.Var, actual.Var)
	equalTypedIdents(t, expected.Names, actual.Names)
	equalExprs(t, expected.Values, actual.Values)
}

func equalNamedArgsNames(t *testing.T, expected, actual []NamedArgExpr) {
	require.Equal(t, len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		equalExpr(t, expected[i].Ident, actual[i].Ident)
		equalExpr(t, expected[i].Lit, actual[i].Lit)
	}
}

func equalIdents(t *testing.T, expected, actual []*Ident) {
	require.Equal(t, len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		equalExpr(t, expected[i], actual[i])
	}
}

func equalTypedIdents(t *testing.T, expected, actual []*TypedIdent) {
	require.Equal(t, len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		equalExpr(t, expected[i], actual[i])
	}
}

func equalExprs(t *testing.T, expected, actual []Expr) {
	require.Equal(t, len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		equalExpr(t, expected[i], actual[i])
	}
}

func equalStmts(t *testing.T, expected, actual []Stmt) {
	require.Equal(t, len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		equalStmt(t, expected[i], actual[i])
	}
}

func equalMapElements(
	t *testing.T,
	expected, actual []*DictElementLit,
) {
	require.Equal(t, len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		require.Equal(t, expected[i].Key, actual[i].Key)
		require.Equal(t, expected[i].KeyPos, actual[i].KeyPos)
		require.Equal(t, expected[i].ColonPos, actual[i].ColonPos)
		equalExpr(t, expected[i].Value, actual[i].Value)
	}
}

func parseSource(
	filename string,
	src []byte,
	trace io.Writer,
	mode Mode,
) (res *File, err error) {
	fileSet := NewFileSet()
	file := fileSet.AddFile(filename, -1, len(src))

	p := NewParserWithOptions(file, src, &ParserOptions{Trace: trace, Mode: mode}, nil)
	return p.ParseFile()
}
