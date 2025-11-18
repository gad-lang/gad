package parser_test

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/gad-lang/gad/parser/ast"
	. "github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gad-lang/gad/token"

	. "github.com/gad-lang/gad/parser"
)

type (
	Pos           = source.Pos
	SourceFilePos = source.FilePos
)

const NoPos = source.NoPos

var NewFileSet = source.NewFileSet
var update = flag.Bool("update", false, "update golden files")

func TestParserTrace(t *testing.T) {
	parse := func(input string, tracer io.Writer) {
		testFileSet := NewFileSet()
		testFile := testFileSet.AddFileData("test", -1, []byte(input))
		p := NewParser(testFile, tracer)
		_, err := p.ParseFile()
		require.NoError(t, err)
	}
	sampleB, err := os.ReadFile("testdata/sample.gad")
	require.NoError(t, err)
	sample := string(sampleB)

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
	defaultExpectParse := func(input string, fn expectedFn) {
		expectParse(t, input, fn, func(po *ParserOptions, so *ScannerOptions) {
			so.MixedDelimiter = DefaultMixedDelimiter
		})
	}

	defaultExpectParse(`# gad: mixed
	{% 
	1
%} 
`, func(p pfn) []Stmt {
		return stmts(
			config(p(1, 1), KVp(ident("mixed", p(1, 8)))),
			mixedTextStmt(p(1, 14), "\t"),
			codeBegin(lit("{%", p(2, 2)), false),
			exprStmt(intLit(1, p(3, 2))),
			codeEnd(lit("%}", p(4, 1)), false),
			mixedTextStmt(p(4, 3), " \n"),
		)
	})

	defaultExpectParse(`# gad: mixed
	{%   1   %} 
`, func(p pfn) []Stmt {
		return stmts(
			config(p(1, 1), KVp(ident("mixed", p(1, 8)))),
			mixedTextStmt(p(1, 14), "\t"),
			codeBegin(lit("{%", p(2, 2)), false),
			exprStmt(intLit(1, p(2, 7))),
			codeEnd(lit("%}", p(2, 11)), false),
			mixedTextStmt(p(1, 26), " \n"),
		)
	})

	defaultExpectParse(`# gad: mixed
	{%   1   -%} 
`, func(p pfn) []Stmt {
		return stmts(
			config(p(1, 1), KVp(ident("mixed", p(1, 8)))),
			mixedTextStmt(p(1, 14), "\t"),
			codeBegin(lit("{%", p(2, 2)), false),
			exprStmt(intLit(1, p(2, 7))),
			codeEnd(lit("%}", p(2, 12)), true),
			mixedTextStmt(p(1, 27), " \n", RemoveLeftSpaces),
		)
	})
	defaultExpectParse(`# gad: mixed
	{%   1   -%}
a
{%- =   2   -%}`, func(p pfn) []Stmt {
		return stmts(
			config(p(1, 1), KVp(ident("mixed", p(1, 8)))),
			mixedTextStmt(p(1, 14), "\t"),
			codeBegin(lit("{%", p(2, 2)), false),
			exprStmt(intLit(1, p(2, 7))),
			codeEnd(lit("%}", p(2, 12)), true),
			mixedTextStmt(p(1, 27), "\na\n", RemoveLeftSpaces|RemoveRightSpaces),
			mixedValue(lit("{%", p(4, 1)), lit("%}", p(4, 14)), intLit(2, p(4, 9))),
		)
	})

	defaultExpectParse(`# gad: mixed
	{%   1   -%}
a
{%- =   2   -%}

{%   3   %}
`, func(p pfn) []Stmt {
		return stmts(
			config(p(1, 1), KVp(ident("mixed", p(1, 8)))),
			mixedTextStmt(p(1, 14), "\t"),
			codeBegin(lit("{%", p(2, 2)), false),
			exprStmt(intLit(1, p(2, 7))),
			codeEnd(lit("%}", p(2, 12)), true),
			mixedTextStmt(p(1, 27), "\na\n", RemoveLeftSpaces|RemoveRightSpaces),
			mixedValue(lit("{%", p(4, 1)), lit("%}", p(4, 14)), intLit(2, p(4, 9))),
			mixedTextStmt(p(4, 16), "\n\n", RemoveLeftSpaces),
			codeBegin(lit("{%", p(6, 1)), false),
			exprStmt(intLit(3, p(6, 6))),
			codeEnd(lit("%}", p(6, 10)), false),
			mixedTextStmt(p(6, 12), "\n"),
		)
	})

	defaultExpectParse(`# gad: mixed
	{%   1   -%}
a
{%- =   2   -%}

{%   3   -%}
`, func(p pfn) []Stmt {
		return stmts(
			config(p(1, 1), KVp(ident("mixed", p(1, 8)))),
			mixedTextStmt(p(1, 14), "\t"),
			codeBegin(lit("{%", p(2, 2)), false),
			exprStmt(intLit(1, p(2, 7))),
			codeEnd(lit("%}", p(2, 12)), true),
			mixedTextStmt(p(1, 27), "\na\n", RemoveLeftSpaces|RemoveRightSpaces),
			mixedValue(lit("{%", p(4, 1)), lit("%}", p(4, 14)), intLit(2, p(4, 9))),
			mixedTextStmt(p(4, 16), "\n\n", RemoveLeftSpaces),
			codeBegin(lit("{%", p(6, 1)), false),
			exprStmt(intLit(3, p(6, 6))),
			codeEnd(lit("%}", p(6, 11)), true),
			mixedTextStmt(p(6, 13), "\n", RemoveLeftSpaces),
		)
	})
	expectParseMode(t, ParseMixed, "‹- 1 -› a", func(p pfn) []Stmt {
		return stmts(
			codeBegin(lit("‹", p(1, 1)), true),
			exprStmt(intLit(1, p(1, 6))),
			codeEnd(lit("›", p(1, 9)), true),
			mixedTextStmt(p(1, 12), " a", RemoveLeftSpaces),
		)
	})
	expectParseStringMode(t, ParseMixed, "‹- var myfn -› a", "‹-; var myfn; -› a")
	expectParseStringMode(t, ParseMixed, "a ‹- 1 ›", "a ; ‹-; 1; ›")
	expectParseStringMode(t, ParseMixed, "‹ 1 ›", "‹; 1; ›")
	expectParseStringMode(t, ParseMixed, "‹ 1; 2; var a ›", "‹; 1; 2; var a; ›")
	expectParseStringMode(t, ParseMixed, "x ‹ 1; 2; var a › y", "x ; ‹; 1; 2; var a; › y")
	expectParseStringMode(t, ParseMixed, "‹var a›", `‹; var a; ›`)
	expectParseStringMode(t, ParseMixed, "‹=1›", "‹=1›")
	expectParseStringMode(t, ParseMixed, "a  ‹-= 1 -›\n\tb", "a  ; ‹-=1-›; \n\tb")
	expectParseStringMode(t, ParseMixed, "‹(› 2 ‹- ) ›", "‹; (› 2 ‹-); ›")
	expectParseStringMode(t, ParseMixed, "‹( -› 2 ‹- ) ›", "‹; (-› 2 ‹-); ›")
	expectParseStringMode(t, ParseMixed, "‹a = (› 2 ‹- ) ›", "‹; a = (› 2 ‹-); ›")
	expectParseStringMode(t, ParseMixed, "‹1›‹2›‹3›", `‹; 1; 2; 3; ›`)
	expectParseStringMode(t, ParseMixed, "‹1›‹›‹3›", `‹; 1; 3; ›`)
	expectParseStringMode(t, ParseMixed, "‹1›‹=2›‹3›", `‹; 1; ›‹=2›‹; 3; ›`)
	expectParseStringMode(t, ParseMixed, "abc", "abc")
	expectParseStringMode(t, ParseMixed, "a‹1›", "a; ‹; 1; ›")
	expectParseStringMode(t, ParseMixed, "a‹  1  ›b", "a; ‹; 1; ›b")
	expectParseStringMode(t, ParseMixed, "a‹1?2:3   ›b‹=   2 + 4›", "a; ‹; (1 ? 2 : 3); ›b‹=(2 + 4)›")
	expectParseStringMode(t, ParseMixed, "a‹1?2:3;fn();x++   ›b‹=   2 + 4›", "a; ‹; (1 ? 2 : 3); fn(); x++; ›b‹=(2 + 4)›")
	expectParseStringMode(t, ParseMixed, "a\n‹- 1›\tb\n‹-= 2 -›\n\nc", "a\n; ‹-; 1; ›\tb\n‹-=2-›\n\nc")
	expectParseStringMode(t, ParseMixed, `a‹=1›c‹x := 5›‹=x›`, "a; ‹=1›; c; ‹; x := 5; ›‹=x›")

	expectParseStringMode(t, ParseMixed, "‹if 1›2‹end›", "‹; if 1  ›2‹ end; ›")
	expectParseStringMode(t, ParseMixed, "‹if 1 then›2‹end›", "‹; if 1 then ›2‹ end; ›")
	expectParseStringMode(t, ParseMixed, "‹if 1 then›2‹else if 3 then›4‹end›", "‹; if 1 then ›2‹ else if 3 then ›4‹ end; ›")
	expectParseStringMode(t, ParseMixed, "‹if 1 then›2‹else›3‹end›", "‹; if 1 then ›2‹ else ›3‹ end; ›")
	expectParseStringMode(t, ParseMixed, "‹if 1 then›2‹if 2 then›3‹end›‹end›", "‹; if 1 then ›2‹; if 2 then ›3‹ end end; ›")
	expectParseStringMode(t, ParseMixed, "‹ if 1 then › 2 ‹ end ›", "‹; if 1 then › 2 ‹ end; ›")

	expectParseStringMode(t, ParseMixed, "‹for a in b›2‹end›", "‹; for _, a in b  ›2‹ end; ›")
	expectParseStringMode(t, ParseMixed, "‹for i:=0;i<2;i++›v‹end›", "‹; for i := 0 ; (i < 2)  ; i++ ›v‹ end; ›")
	expectParseStringMode(t, ParseMixed, "‹for e in list›1‹end›", "‹; for _, e in list  ›1‹ end; ›")
	expectParseStringMode(t, ParseMixed, "‹for e in list›‹=1›‹end›", "‹; for _, e in list  ›‹=1›‹ end; ›")
	expectParseStringMode(t, ParseMixed, "‹for e in list {›1‹}else{›2‹}›", "‹; for _, e in list { ›1‹ } else { ›2‹ }; ›")
	expectParseStringMode(t, ParseMixed, "‹for e in list {›1   ‹-}else{›2‹}›", "‹; for _, e in list { ›1   ‹- } else { ›2‹ }; ›")

	expectParseStringMode(t, ParseMixed, "‹try›1‹finally›2‹end›", "‹; try  ›1‹  finally  ›2‹ end; ›")
	expectParseStringMode(t, ParseMixed, "‹try›1‹catch e›2‹finally›3‹end›", "‹; try  ›1‹  catch e  ›2‹  finally  ›3‹ end; ›")
	expectParseStringMode(t, ParseMixed, "abc ‹=\n// my single comment\n\n/* long\n comment\n\n*/\n1›def", "abc ; ‹=1›; def")

	// example for auto generated mixed script mapping multiples sources
	expectParseStringMode(t, ParseMixed, `
a
‹
//src:1
x := 2
//
›
b
‹=
//src:2
x ** 10
//
›
c
‹
//src:3
if 1 then
//
›
d
‹
//src:4
end
//
›
`, "\na\n; ‹; x := 2; ›\nb\n‹=(x ** 10)›\nc\n‹; if 1 then ›\nd\n‹ end; ›\n")
}

func TestParserError(t *testing.T) {
	expectParseError(t, "var x;\n\nvar y;\nparam a,b\nvar z\nz2\nz3\nz4",
		[2]string{"%v", "Parse Error: expected statement, found ','\n\tat test:4:8"},
		[2]string{"%+v", "Parse Error: expected statement, found ','" +
			"\n\tat test:4:8" +
			"\n\n       🠆 4| param a,b" +
			"\n                   ^"},
		[2]string{"%+3.4v", "Parse Error: expected statement, found '," +
			"'\n\tat test:4:8" +
			"\n\n         1| var x;" +
			"\n         3| var y;" +
			"\n       🠆 4| param a,b" +
			"\n                   ^" +
			"\n         5| var z" +
			"\n         6| z2" +
			"\n         7| z3" +
			"\n         8| z4"},
	)

	expectParseError(t, `param a,b`,
		[2]string{"%v", "Parse Error: expected statement, found ','\n\tat test:1:8"},
		[2]string{"%+v", "Parse Error: expected statement, found ','\n\tat test:1:8\n\n       🠆 1| param a,b\n                   ^"},
	)

	expectParseString(t, `a := throw "my error"`, `a := throw "my error"`)

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

	expectParseString(t, "param x int", "param x int")
	expectParseString(t, "param x int|bool", "param x int|bool")
	expectParseString(t, "param (\nx int,\n)", "param (x int)")
	expectParseString(t, "param (\nx int,\ny)", "param (x int, y)")
	expectParseString(t, "param (\nx int,\ny, z string|bool)", "param (x int, y, z string|bool)")
	expectParseString(t, "param *x int", "param *x int")
	expectParseString(t, "param *x int|bool", "param *x int|bool")
	expectParseString(t, "param **x int", "param **x int")
	expectParseString(t, "param **x int|bool", "param **x int|bool")
	expectParseString(t, "param b int=2", "param b int=2")
	expectParseString(t, "param b bool|int=2", "param b bool|int=2")
	expectParseString(t, "param (a, *b string, x bool|int=2, **y int)",
		"param (a, *b string, x bool|int=2, **y int)")

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
						[]*IdentExpr{ident("a", p(1, 5))},
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
						[]*IdentExpr{ident("a", p(1, 5))},
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
						[]*IdentExpr{ident("a", p(1, 5))},
						[]Expr{nil}),
				),
			),
			declStmt(
				genDecl(token.Var, p(1, 7), 0, 0,
					valueSpec(
						[]*IdentExpr{ident("b", p(1, 11))},
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
						[]*IdentExpr{ident("a", p(1, 5))},
						[]Expr{stringLit("x", p(1, 7))}),
				),
			),
			declStmt(
				genDecl(token.Var, p(1, 11), 0, 0,
					valueSpec(
						[]*IdentExpr{ident("b", p(1, 15))},
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
						[]*IdentExpr{ident("a", p(2, 5))},
						[]Expr{nil}),
				),
			),
			declStmt(
				genDecl(token.Var, p(3, 1), 0, 0,
					valueSpec(
						[]*IdentExpr{ident("b", p(3, 5))},
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
						[]*IdentExpr{ident("a", p(2, 5))},
						[]Expr{nil}),
				),
			),
			declStmt(
				genDecl(token.Var, p(3, 1), 0, 0,
					valueSpec(
						[]*IdentExpr{ident("b", p(3, 5))},
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
						[]*IdentExpr{ident("a", p(1, 6))},
						[]Expr{nil}),
					valueSpec(
						[]*IdentExpr{ident("b", p(1, 9))},
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
						[]*IdentExpr{ident("a", p(1, 6))},
						[]Expr{intLit(1, p(1, 8))}),
					valueSpec(
						[]*IdentExpr{ident("b", p(1, 11))},
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
						[]*IdentExpr{ident("a", p(1, 6))},
						[]Expr{intLit(1, p(1, 8))}),
					valueSpec(
						[]*IdentExpr{ident("b", p(2, 1))},
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
						[]*IdentExpr{ident("a", p(1, 6))},
						[]Expr{intLit(1, p(1, 8))}),
					valueSpec(
						[]*IdentExpr{ident("b", p(2, 1))},
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
						[]*IdentExpr{ident("a", p(1, 7))},
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
						[]*IdentExpr{ident("a", p(1, 7))},
						[]Expr{intLit(1, p(1, 11))}),
				),
			),
			declStmt(
				genDecl(token.Const, p(1, 14), 0, 0,
					valueSpec(
						[]*IdentExpr{ident("b", p(1, 20))},
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
						[]*IdentExpr{ident("a", p(1, 8))},
						[]Expr{intLit(1, p(1, 12))}),
					valueSpec(
						[]*IdentExpr{ident("b", p(1, 15))},
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
						[]*IdentExpr{ident("a", p(3, 5))},
						[]Expr{intLit(1, p(3, 9))}),
					valueSpec(
						[]*IdentExpr{ident("b", p(4, 5))},
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

	expectParse(t, "a ++= 5", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(intLit(5, p(1, 7))),
				token.IncAssign,
				p(1, 3)))
	})

	expectParse(t, "a --= 5", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(intLit(5, p(1, 7))),
				token.DecAssign,
				p(1, 3)))
	})

	expectParse(t, "a **= 5 + 10", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(
					binaryExpr(
						intLit(5, p(1, 7)),
						intLit(10, p(1, 11)),
						token.Add,
						p(1, 9))),
				token.PowAssign,
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
	expectParse(t, "add(,)", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				callExpr(
					ident("add", p(1, 1)),
					p(1, 4), p(1, 6),
					callExprArgs(nil))))
	})
	expectParse(t, "add(\n\t,)", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				callExpr(
					ident("add", p(1, 1)),
					p(1, 4), p(2, 3),
					callExprArgs(nil))))
	})
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

	expectParseString(t, `attrs(;"name")`, `attrs(; "name"=yes)`)
	expectParseString(t, "fn(a;b)", "fn(a; b=yes)")
	expectParseString(t, "fn(**{y:5})", "fn(; **{y: 5})")
	expectParseString(t, "fn(1,*[2,3],x=4,**{y:5})", "fn(1, *[2, 3]; x=4, **{y: 5})")
	expectParseString(t, "fn(1, a=b)()", "fn(1; a=b)()")
}

func TestParseParenMultiValues(t *testing.T) {
	var mp *MultiParenExpr
	expectParseStringT(t, `(,)`, `(, )`, mp)
	expectParseStringT(t, `(,1)`, `(, 1)`, mp)
	expectParseStringT(t, `([a=1],b=2)`, `([a=1]; b=2)`, mp)
	expectParseStringT(t, `(a=1)`, `(, ; a=1)`, mp)
	expectParseStringT(t, `(*a)`, `(*a)`, mp)
	expectParseStringT(t, `(**a)`, `(, ; **a)`, mp)
	expectParseStringT(t, `(1;ok)`, `(1; ok)`, mp)
	expectParseStringT(t, `(a, *b, c=1, **d)`, `(a, *b; c=1, **d)`, mp)
	expectParseStringT(t, `(a,c=2, x(1))`, `(a; c=2, x(1))`, mp)
	expectParseStringT(t, `(a,
c=2, 
  x(1))`, `(a; c=2, x(1))`, mp)
}

func TestParseKeyValue(t *testing.T) {
	expectParse(t, `[a=1]`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				kv(ident("a", p(1, 2)), intLit(1, p(1, 4)))))
	})
	expectParseString(t, `[a=1]`, `[a=1]`)
}

func TestParseKeyValueArray(t *testing.T) {
	expectParseString(t, `(;)`, `(;)`)
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

	kva := &KeyValueArrayLit{}
	expectParseStringT(t, `(;**a)`, `(;**a)`, kva)
	expectParseStringT(t, `(;x=1, **a)`, `(;x=1, **a)`, kva)
	expectParseStringT(t, `(;a=1)`, `(;a=1)`, kva)
	expectParseStringT(t, `(;**a)`, `(;**a)`, kva)
}

func TestTemplateString(t *testing.T) {
	expectParseString(t, `#"A"`, `#"A"`)
	expectParseString(t, "#`A`", "#`A`")
	expectParseString(t, "#```A```", "#```A```")
}

func TestParseChar(t *testing.T) {
	expectParseExpr(t, `'A'`, charLit('A', 1))
	expectParseExpr(t, `'九'`, charLit('九', 1))

	expectParseError(t, `''`)
	expectParseError(t, `'AB'`)
	expectParseError(t, `'A九'`)

	expectParseExpr(t, `'A'`, charAsStringLit("A", 1), OptParseCharAsString)
	expectParseExpr(t, `'九'`, charAsStringLit("九", 1), OptParseCharAsString)
	expectParseExpr(t, `'A九'`, charAsStringLit("A九", 1), OptParseCharAsString)
	expectParseExpr(t, "'a\\'b'", charAsStringLit(`a'b`, 1), OptParseCharAsString)
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

func TestParseReturn(t *testing.T) {
	expectParse(t, "return", func(p pfn) []Stmt {
		return stmts(returnStmt(p(1, 1), nil))
	})
	expectParse(t, "1 || return", func(p pfn) []Stmt {
		return stmts(exprStmt(binaryExpr(intLit(1, p(1, 1)), returnExpr(p(1, 6), nil), token.LOr, p(1, 3))))
	})

	expectParseString(t, `var x; x || return`,
		"var x; (x || return)")
	expectParseString(t, `return 1`,
		"return 1")
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

	expectParse(t, "for x in y do x else 1 end", func(p pfn) []Stmt {
		return stmts(
			forInStmt(
				ident("_", p(1, 5)),
				ident("x", p(1, 5)),
				ident("y", p(1, 10)),
				blockLitStmt(
					lit("do", p(1, 12)), ast.Literal{},
					exprStmt(
						ident("x", p(1, 15)),
					),
				),
				p(1, 1),
				blockLitStmt(lit("", p(1, 22)), lit("end", p(1, 24)),
					exprStmt(
						intLit(1, p(1, 22)),
					),
				)))
	})

	expectParseString(t, "for x in y do x else 1 end", "for _, x in y do x else 1 end")

	expectParse(t, "for x in y {} else 1 end", func(p pfn) []Stmt {
		return stmts(
			forInStmt(
				ident("_", p(1, 5)),
				ident("x", p(1, 5)),
				ident("y", p(1, 10)),
				blockStmt(p(1, 12), p(1, 13)),
				p(1, 1),
				blockLitStmt(lit("", p(1, 20)), lit("end", p(1, 22)),
					exprStmt(
						intLit(1, p(1, 20)),
					),
				)))
	})

	expectParseString(t, "for x in y do end", "for _, x in y do end")
	expectParseString(t, "for x in y do 1 end", "for _, x in y do 1 end")

	expectParseString(t, "for x in y do else end", "for _, x in y do else end")
	expectParseString(t, "for x in y do 1 else end", "for _, x in y do 1 else end")
	expectParseString(t, "for x in y do else end", "for _, x in y do else end")
	expectParseString(t, "for x in y do 1 else 2 end", "for _, x in y do 1 else 2 end")

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

	expectParseString(t, `for do continue end`, "for do continue end")

	// labels are parsed by parser but not supported by compiler yet
	// expectParseError(t, `for { break x }`)
}

func TestParseClosure(t *testing.T) {
	expectParseStringT(t, `(a,*b,c=2,**d) => 3`, `(a, *b, c=2, **d) => 3`, &ClosureExpr{})
	expectParseStringT(t, `(a,*b,c=2) => 3`, `(a, *b, c=2) => 3`, &ClosureExpr{})
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
	expectParseString(t, "func(a int){}", "func(a int) {}")
	expectParseString(t, "func(a int|bool|int){}", "func(a int|bool) {}")
	expectParseString(t, "func(a \n int|\n\tbool){}", "func(a int|bool) {}")
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
								[]*TypedIdentExpr{
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
							[]*TypedIdentExpr{typedIdent(ident("x", p(1, 12)))},
							[]Expr{intLit(1, p(1, 14))}),
					),
					blockStmt(p(1, 22), p(1, 23)))),
		)
	})

	expectParseString(t, "func(){}", "func() {}")
	expectParseString(t, "func(,){}", "func() {}")
	expectParseString(t, "func(\n\t,){}", "func() {}")
	expectParseString(t, "func(\n){}", "func() {}")
	expectParseString(t, "func(a,){}", "func(a) {}")
	expectParseString(t, "func(,a){}", "func(a) {}")
	expectParseString(t, "func(\n\t,a){}", "func(a) {}")
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
	expectParseString(t, `func(;x int=1, y str="abc", **kw) {}`, `func(x int=1, y str="abc", **kw) {}`)

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

	expectParseString(t, "if a then end", "if a then end")
	expectParseString(t, "if a then b end", "if a then b end")
	expectParseString(t, "if true; a then b end", "if true; a then b end")
	expectParseString(t, "if a then b else c end", "if a then b else c end")
	expectParseString(t, "if a then b; else c end", "if a then b else c end")
	expectParseString(t, "if a then b else if 1 then 2 else c end", "if a then b else if 1 then 2 else c end")
	expectParseString(t, "if a then b; else if 1 then 2; else c end", "if a then b else if 1 then 2 else c end")

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

func TestParseEmbed(t *testing.T) {
	expectParse(t, `a := embed("file")`, func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(embedExpr("file", p(1, 6))),
				token.Define, p(1, 3)))
	})

	expectParse(t, `embed("file").var1`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				selectorExpr(
					embedExpr("file", p(1, 1)),
					stringLit("var1", p(1, 15)))))
	})

	expectParse(t, `for x, y in embed("file") {}`, func(p pfn) []Stmt {
		return stmts(
			forInStmt(
				ident("x", p(1, 5)),
				ident("y", p(1, 8)),
				embedExpr("file", p(1, 13)),
				blockStmt(p(1, 27), p(1, 28)),
				p(1, 1)))
	})

	expectParseError(t, `embed(1)`)
	expectParseError(t, `embed('a')`)
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

	expectParse(t, `({a: 1, b: 2})["b"]`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				indexExpr(
					parenExpr(
						dictLit(p(1, 2), p(1, 13),
							mapElementLit(
								"a", p(1, 3), p(1, 4), intLit(1, p(1, 6))),
							mapElementLit(
								"b", p(1, 9), p(1, 10), intLit(2, p(1, 12)))),
						p(1, 1), p(1, 14),
					),
					stringLit("b", p(1, 16)),
					p(1, 15), p(1, 19))))
	})

	expectParse(t, `({a: 1, b: 2})[a + b]`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				indexExpr(
					parenExpr(
						dictLit(p(1, 2), p(1, 13),
							mapElementLit(
								"a", p(1, 3), p(1, 4), intLit(1, p(1, 6))),
							mapElementLit(
								"b", p(1, 9), p(1, 10), intLit(2, p(1, 12)))),
						p(1, 1), p(1, 14),
					),
					binaryExpr(
						ident("a", p(1, 16)),
						ident("b", p(1, 20)),
						token.Add,
						p(1, 18)),
					p(1, 15), p(1, 21))))
	})
}

func TestParseLogical(t *testing.T) {
	expectParse(t, "2 ** 3", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				binaryExpr(
					intLit(2, p(1, 1)),
					intLit(3, p(1, 6)),
					token.Pow,
					p(1, 3))))
	})

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

func TestParseBlock(t *testing.T) {
	expectParse(t, "{}", func(p pfn) []Stmt {
		return stmts(blockStmt(p(1, 1), p(1, 2)))
	})

	expectParse(t, "x := 1; {x := 2}", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				Exprs{ident("x", p(1, 1))},
				Exprs{intLit(1, p(1, 6))},
				token.Define,
				p(1, 3),
			),
			blockStmt(
				p(1, 9),
				p(1, 16),
				assignStmt(
					Exprs{ident("x", p(1, 10))},
					Exprs{intLit(2, p(1, 15))},
					token.Define,
					p(1, 12),
				),
			),
		)
	})
}

func TestParseDict(t *testing.T) {
	expectParse(t, "({})", func(p pfn) []Stmt {
		return stmts(exprStmt(parenExpr(dictLit(p(1, 2), p(1, 3)), p(1, 1), p(1, 4))))
	})

	expectParse(t, "({ \"key1\": 1 })", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				parenExpr(
					dictLit(p(1, 2), p(1, 14),
						mapElementLit(
							"key1", p(1, 4), p(1, 10), intLit(1, p(1, 12)))),
					p(1, 1), p(1, 15))))
	})

	expectParse(t, "({ key1: 1, key2: \"2\", key3: true })", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				parenExpr(
					dictLit(p(1, 2), p(1, 35),
						mapElementLit(
							"key1", p(1, 4), p(1, 8), intLit(1, p(1, 10))),
						mapElementLit(
							"key2", p(1, 13), p(1, 17), stringLit("2", p(1, 19))),
						mapElementLit(
							"key3", p(1, 24), p(1, 28), boolLit(true, p(1, 30)))),
					p(1, 1), p(1, 36))))
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
({
	key1: 1,
	key2: "2",
	key3: true,
})`, func(p pfn) []Stmt {
		return stmts(exprStmt(
			parenExpr(
				dictLit(p(2, 2), p(6, 1),
					mapElementLit(
						"key1", p(3, 2), p(3, 6), intLit(1, p(3, 8))),
					mapElementLit(
						"key2", p(4, 2), p(4, 6), stringLit("2", p(4, 8))),
					mapElementLit(
						"key3", p(5, 2), p(5, 6), boolLit(true, p(5, 8)))),
				p(2, 1), p(6, 2))))
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
	expectParseString(t, `2 * 1 + 3 / 4`, `((2 * 1) + (3 / 4))`)
	expectParseString(t, `a .| b`, `(a .| b)`)
	expectParseString(t, `a .| b .| c`, `((a .| b) .| c)`)
	expectParseString(t, `a .| b + c`, `((a .| b) + c)`)
	expectParseString(t, `a .| b * c`, `((a .| b) * c)`)
	expectParseString(t, `a ~ b`, `(a ~ b)`)
	expectParseString(t, `a ~ b ~ c`, `((a ~ b) ~ c)`)
	expectParseString(t, `a ~ b * c`, `((a ~ b) * c)`)
	expectParseString(t, `a ~ b ~ c .| d`, `(((a ~ b) ~ c) .| d)`)
	expectParseString(t, `a ~ b / c`, `((a ~ b) / c)`)
	expectParseString(t, `a ** b * c; d * e ** f`, `((a ** b) * c); (d * (e ** f))`)
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

	expectParse(t, "({k1:1}.k1)", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				parenExpr(
					selectorExpr(
						dictLit(
							p(1, 2), p(1, 7),
							mapElementLit(
								"k1", p(1, 3), p(1, 5), intLit(1, p(1, 6)))),
						stringLit("k1", p(1, 9))),
					p(1, 1), p(1, 11))))

	})

	expectParse(t, "({k1:1}).k1", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				selectorExpr(
					parenExpr(
						dictLit(
							p(1, 2), p(1, 7),
							mapElementLit(
								"k1", p(1, 3), p(1, 5), intLit(1, p(1, 6)))),
						p(1, 1), p(1, 8)),
					stringLit("k1", p(1, 10)))))

	})

	expectParse(t, "({k1:{v1:1}}.k1.v1)", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				parenExpr(
					selectorExpr(
						selectorExpr(
							dictLit(
								p(1, 2), p(1, 12),
								mapElementLit("k1", p(1, 3), p(1, 5),
									dictLit(p(1, 6), p(1, 11),
										mapElementLit(
											"v1", p(1, 7),
											p(1, 9), intLit(1, p(1, 10)))))),
							stringLit("k1", p(1, 14))),
						stringLit("v1", p(1, 17))),
					p(1, 1), p(1, 19))))
	})

	expectParse(t, "({k1:{v1:1}}).k1.v1", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				selectorExpr(
					selectorExpr(
						parenExpr(
							dictLit(
								p(1, 2), p(1, 12),
								mapElementLit("k1", p(1, 3), p(1, 5),
									dictLit(p(1, 6), p(1, 11),
										mapElementLit(
											"v1", p(1, 7),
											p(1, 9), intLit(1, p(1, 10)))))),
							p(1, 1), p(1, 13)),
						stringLit("k1", p(1, 15))),
					stringLit("v1", p(1, 18)))))
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
	expectParse(t, "\"foo\nbar\"", func(p pfn) []Stmt {
		return stmts(exprStmt(stringLit("foo\nbar", p(1, 1))))
	})
	expectParse(t, "\"foo\nbar\"", func(p pfn) []Stmt {
		return stmts(exprStmt(stringLit("foo\nbar", p(1, 1))))
	})
	expectParse(t, "\"foo\n"+"\n"+"bar\"", func(p pfn) []Stmt {
		return stmts(exprStmt(stringLit("foo\n\nbar", p(1, 1))))
	})
	expectParse(t, `"foo\n`+"\n"+`bar"`, func(p pfn) []Stmt {
		return stmts(exprStmt(stringLit("foo\\n\nbar", p(1, 1))))
	})
	expectParse(t, "`abc`", func(p pfn) []Stmt {
		return stmts(exprStmt(rawStringLit(`abc`, p(1, 1))))
	})
	expectParse(t, "```\nabc\n```", func(p pfn) []Stmt {
		return stmts(exprStmt(rawHeredocLit("```", `abc`, p(1, 1))))
	})
	expectParse(t, "a = \"foo\nbar\"", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(stringLit("foo\nbar", p(1, 5))),
				token.Assign,
				p(1, 3)))
	})
	expectParse(t, `a = "foo\nbar"`, func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(stringLit(`foo\nbar`, p(1, 5))),
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
	a`, func(p pfn) []Stmt {
		return stmts(
			config(p(1, 1), KVp(ident("mixed", p(1, 8)))),
			mixedTextStmt(p(2, 1), "\ta"),
		)
	})
	expectParse(t, `# gad: mixed, mixed_start = "[[[", mixed_end = "]]]"
y
[[[b]]]`, func(p pfn) []Stmt {
		return stmts(
			config(p(1, 1),
				KVp(ident("mixed", p(1, 8))),
				KVp(ident("mixed_start", p(1, 15)), stringLit("[[[", p(1, 29))),
				KVp(ident("mixed_end", p(1, 36)), stringLit("]]]", p(1, 48))),
			),
			mixedTextStmt(p(2, 1), "y\n"),
			codeBegin(lit("[[[", p(3, 1)), false),
			exprStmt(ident("b", p(3, 4))),
			codeEnd(lit("]]]", p(3, 5)), false),
		)
	})

	expectParseString(t, "# gad: mixed, mixed_start=\"[[[\", mixed_end=\"]]]\"\ny\n[[[b]]]",
		`# gad: mixed, mixed_start="[[[", mixed_end="]]]"`+"\ny\n[[[; b; ]]]")
	expectParseString(t, "# gad: mixed, mixed_start=\"[[[\", mixed_end=\"]]]\"\ny\n[[[b; true]]]",
		"# gad: mixed, mixed_start=\"[[[\", mixed_end=\"]]]\"\ny\n[[[; b; true; ]]]")
	expectParse(t, `# gad: mixed`, func(p pfn) []Stmt {
		return stmts(
			config(p(1, 1), KVp(ident("mixed", p(1, 8)))))
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

type opts func(po *ParserOptions, so *ScannerOptions)

var OptParseCharAsString opts = func(po *ParserOptions, so *ScannerOptions) {
	so.Mode |= ScanCharAsString
}

func expectParse(t *testing.T, input string, fn expectedFn, opt ...opts) {
	expectParseMode(t, 0, input, fn, opt...)
}

func expectParseStmt(t *testing.T, input string, stmt Stmt, opt ...opts) {
	expectParse(t, input, func(p pfn) []Stmt { return stmts(stmt) }, opt...)
}

func expectParseExpr(t *testing.T, input string, expr Expr, opt ...opts) {
	expectParseStmt(t, input, exprStmt(expr), opt...)
}

func expectParseMode(t *testing.T, mode Mode, input string, fn expectedFn, opt ...opts) {
	testFileSet := NewFileSet()
	testFile := testFileSet.AddFileData("test", -1, []byte(input))

	var (
		ok      bool
		options = func() (po *ParserOptions, so *ScannerOptions) {
			po = &ParserOptions{
				Mode: mode,
			}
			so = &ScannerOptions{
				MixedDelimiter: mixedDelimiter,
			}
			for _, o := range opt {
				o(po, so)
			}
			return
		}
	)
	defer func() {
		if !ok {
			// print Trace
			tr := &parseTracer{}
			po, so := options()
			po.Trace = tr
			p := NewParserWithOptions(testFile, po, so)
			actual, _ := p.ParseFile()
			if actual != nil {
				t.Logf("Parsed:\n%s", actual.String())
			}
			t.Logf("Trace:\n%s", strings.Join(tr.out, ""))
		}
	}()

	po, so := options()

	p := NewParserWithOptions(testFile, po, so)
	actual, err := p.ParseFile()
	require.NoError(t, err)

	expected := fn(func(line, column int) Pos {
		return Pos(int(source.MustFileLineStartPos(testFile, line)) + (column - 1))
	})

	ft := fileTest(t, testFile, actual.InputFile)
	ft.equal(len(expected), len(actual.Stmts), "len(file.Stmts)")

	for i := 0; i < len(expected); i++ {
		ft.equalStmt(expected[i], actual.Stmts[i])
	}

	ok = true
}

type fileTester struct {
	t        *testing.T
	expected *source.File
	actual   *source.File
}

func (f *fileTester) equal(expected, actual any, msgAndArgs ...any) {
	switch t := expected.(type) {
	case Pos:
		p := source.MustFilePosition(f.expected, t)
		expected = fmt.Sprintf("Pos(%d, %d)", p.Line, p.Column)
		if pos, ok := actual.(Pos); ok {
			p = source.MustFilePosition(f.actual, pos)
			actual = fmt.Sprintf("Pos(%d, %d)", p.Line, p.Column)
		}
	case ast.Literal:
		if actual, ok := actual.(ast.Literal); ok {
			f.equal(t.Pos, actual.Pos, msgAndArgs...)
			require.Equal(f.t, t.Value, actual.Value, msgAndArgs...)
			return
		}
	}

	require.Equal(f.t, expected, actual, msgAndArgs...)
}

func fileTest(t *testing.T, expected *source.File, actual *source.File) *fileTester {
	return &fileTester{t, expected, actual}
}

func expectParseError(t *testing.T, input string, e ...[2]string) {
	var (
		binput      = []byte(input)
		testFileSet = NewFileSet()
		testFile    = testFileSet.AddFileData("test", -1, binput)
	)

	var ok bool
	defer func() {
		if !ok {
			// print Trace
			tr := &parseTracer{}
			p := NewParser(testFile, tr)
			_, _ = p.ParseFile()
			t.Logf("Trace:\n%s", strings.Join(tr.out, ""))
		}
	}()

	p := NewParser(testFile, nil)
	_, err := p.ParseFile()
	require.Error(t, err)

	if len(e) > 0 {
		for i, ev := range e {
			s := fmt.Sprintf(ev[0], err)
			require.Equal(t, ev[1], s, "formatted error "+strconv.Itoa(i))
		}
	}

	ok = true
}

func expectParseString(t *testing.T, input, expected string) {
	expectParseStringMode(t, 0, input, expected)
}

func expectParseStringMode(t *testing.T, mode Mode, input, expected string) {
	expectParseStringModeT(t, mode, input, expected, nilVal)
}

var (
	nilVal  = (*any)(nil)
	nilType = reflect.TypeOf(nilVal)
)

func expectParseStringT[T any](t *testing.T, input, expected string, typ T) {
	expectParseStringModeT(t, 0, input, expected, typ)
}

func expectParseStringModeT[T any](t *testing.T, mode Mode, input, expected string, typ T) {
	t.Helper()

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
	if reflect.TypeOf(typ) != nilType {
		assert.Equal(t, reflect.TypeOf(typ).String(), reflect.TypeOf(actual.Stmts[0].(*ExprStmt).Expr).String())
	}
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

func paramSpec(variadic bool, ident *TypedIdentExpr) Spec {
	return &ParamSpec{
		Ident:    ident,
		Variadic: variadic,
	}
}

func nparamSpec(ident *TypedIdentExpr, value Expr) Spec {
	return &NamedParamSpec{
		Ident: ident,
		Value: value,
	}
}

func valueSpec(idents []*IdentExpr, values []Expr) Spec {
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
	return &ReturnStmt{Return: Return{Result: result, ReturnPos: pos}}
}

func returnExpr(pos Pos, result Expr) *ReturnExpr {
	return &ReturnExpr{Return: Return{Result: result, ReturnPos: pos}}
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
	key, value *IdentExpr,
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
	ident *IdentExpr,
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
		case *IdentExpr:
			f.Ident = t
		}
	}
	return f
}

func funcArgs(vari *TypedIdentExpr, names ...Expr) ArgsList {
	l := ArgsList{Var: vari}
	for _, name := range names {
		switch t := name.(type) {
		case *IdentExpr:
			l.Values = append(l.Values, typedIdent(t))
		case *TypedIdentExpr:
			l.Values = append(l.Values, t)
		}
	}
	return l
}

func funcNamedArgs(vari *TypedIdentExpr, names []*TypedIdentExpr, values []Expr) NamedArgsList {
	return NamedArgsList{Names: names, Var: vari, Values: values}
}

func blockStmt(lbrace, rbrace Pos, list ...Stmt) *BlockStmt {
	return &BlockStmt{Stmts: list, LBrace: lit("{", lbrace), RBrace: lit("}", rbrace)}
}

func blockLitStmt(lbrace, rbrace ast.Literal, list ...Stmt) *BlockStmt {
	return &BlockStmt{Stmts: list, LBrace: lbrace, RBrace: rbrace}
}

func blockExpr(lbrace, rbrace Pos, list ...Stmt) *BlockExpr {
	return &BlockExpr{BlockStmt: blockStmt(lbrace, rbrace, list...)}
}

func ident(name string, pos Pos) *IdentExpr {
	return &IdentExpr{Name: name, NamePos: pos}
}

func typedIdent(ident *IdentExpr, typ ...*IdentExpr) *TypedIdentExpr {
	return &TypedIdentExpr{Ident: ident, Type: typ}
}

func mixedTextStmt(pos Pos, vlit string, flags ...MixedTextStmtFlag) *MixedTextStmt {
	var f MixedTextStmtFlag
	for _, f = range flags {
	}
	return &MixedTextStmt{Lit: lit(vlit, pos), Flags: f}
}

func codeBegin(lit ast.Literal, removeSpace bool) *CodeBeginStmt {
	return &CodeBeginStmt{Lit: lit, RemoveSpace: removeSpace}
}

func codeEnd(lit ast.Literal, removeSpace bool) *CodeEndStmt {
	return &CodeEndStmt{Lit: lit, RemoveSpace: removeSpace}
}

func mixedValue(start, end ast.Literal, expr Expr) *MixedValueStmt {
	return &MixedValueStmt{Expr: expr, StartLit: start, EndLit: end}
}

func lit(value string, pos Pos) ast.Literal {
	return ast.Literal{Value: value, Pos: pos}
}

func kv(key Expr, value ...Expr) *KeyValueLit {
	kv := &KeyValueLit{Key: key}
	for _, expr := range value {
		kv.Value = expr
	}
	return kv
}

func config(start Pos, opts ...*KeyValuePairLit) *ConfigStmt {
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
	return &ImportExpr{ModuleName: moduleName, Token: token.Import, TokenPos: pos}
}

func embedExpr(path string, pos Pos) *EmbedExpr {
	return &EmbedExpr{Path: path, Token: token.Embed, TokenPos: pos}
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
	return &StringLit{Literal: `"` + value + `"`, ValuePos: pos}
}

func charAsStringLit(value string, pos Pos) *StringLit {
	return &StringLit{Literal: `'` + strings.ReplaceAll(value, "'", `\'`) + `'`, ValuePos: pos}
}

func rawStringLit(value string, pos Pos) *RawStringLit {
	return &RawStringLit{Literal: value, LiteralPos: pos, Quoted: value[0] == '`'}
}

func rawHeredocLit(q, value string, pos Pos) *RawHeredocLit {
	return &RawHeredocLit{Literal: q + "\n" + value + "\n" + q, LiteralPos: pos}
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

func arrayLit(lbracket, rbracket Pos, list ...Expr) *ArrayExpr {
	return &ArrayExpr{LBrack: lbracket, RBrack: rbracket, Elements: list}
}

func caleeKw(pos Pos) *CalleeKeywordExpr {
	return &CalleeKeywordExpr{TokenPos: pos, Literal: token.Callee.String()}
}

func argsKw(pos Pos) *ArgsKeywordExpr {
	return &ArgsKeywordExpr{TokenPos: pos, Literal: token.Args.String()}
}

func nargsKw(pos Pos) *NamedArgsKeywordExpr {
	return &NamedArgsKeywordExpr{TokenPos: pos, Literal: token.NamedArgs.String()}
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
) *DictExpr {
	return &DictExpr{LBrace: lbrace, RBrace: rbrace, Elements: list}
}

func funcLit(funcType *FuncType, body *BlockStmt) *FuncExpr {
	return &FuncExpr{Type: funcType, Body: body}
}

func closure(funcType *FuncType, body Expr) *ClosureExpr {
	return &ClosureExpr{Type: funcType, Body: body}
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

func (f *fileTester) equalStmt(expected, actual Stmt) {
	t := f.t
	if expected == nil || reflect.ValueOf(expected).IsNil() {
		require.Nil(t, actual, "expected nil, but got not nil")
		return
	}
	require.NotNil(t, actual, "expected not nil, but got nil")
	require.IsType(t, expected, actual)

	switch expected := expected.(type) {
	case *ExprStmt:
		f.equalExpr(expected.Expr, actual.(*ExprStmt).Expr)
	case *EmptyStmt:
		f.equal(expected.Implicit, actual.(*EmptyStmt).Implicit)
		f.equal(expected.Semicolon, actual.(*EmptyStmt).Semicolon)
	case *BlockStmt:
		f.equal(expected.LBrace, actual.(*BlockStmt).LBrace)
		f.equal(expected.RBrace, actual.(*BlockStmt).RBrace)
		f.equalStmts(expected.Stmts, actual.(*BlockStmt).Stmts)
	case *AssignStmt:
		f.equalExprs(expected.LHS, actual.(*AssignStmt).LHS)
		f.equalExprs(expected.RHS, actual.(*AssignStmt).RHS)
		f.equal(int(expected.Token), int(actual.(*AssignStmt).Token))
		f.equal(int(expected.TokenPos), int(actual.(*AssignStmt).TokenPos))
	case *DeclStmt:
		expectedDecl := expected.Decl.(*GenDecl)
		actualDecl := actual.(*DeclStmt).Decl.(*GenDecl)
		f.equal(expectedDecl.Tok, actualDecl.Tok)
		f.equal(expectedDecl.TokPos, actualDecl.TokPos)
		f.equal(expectedDecl.Lparen, actualDecl.Lparen)
		f.equal(expectedDecl.Rparen, actualDecl.Rparen)
		f.equal(len(expectedDecl.Specs), len(actualDecl.Specs))
		for i, expSpec := range expectedDecl.Specs {
			actSpec := actualDecl.Specs[i]
			switch expectedSpec := expSpec.(type) {
			case *ParamSpec:
				actualSpec, ok := actSpec.(*ParamSpec)
				if !ok {
					require.Failf(t, "type error", "expected *ParamSpec, got %T", actSpec)
					return
				}
				f.equal(expectedSpec.Ident, actualSpec.Ident)
				f.equal(expectedSpec.Variadic, actualSpec.Variadic)
			case *NamedParamSpec:
				actualSpec, ok := actSpec.(*NamedParamSpec)
				if !ok {
					require.Failf(t, "type error", "expected *NamedParamSpec, got %T", actSpec)
					return
				}
				f.equal(expectedSpec.Ident, actualSpec.Ident)
				if expectedSpec.Value != nil || actualSpec.Value != nil {
					f.equalExpr(expectedSpec.Value, actualSpec.Value)
				}
			case *ValueSpec:
				actualSpec, ok := actSpec.(*ValueSpec)
				if !ok {
					require.Failf(t, "type error", "expected *ValueSpec, got %T", actSpec)
					return
				}
				f.equal(expectedSpec.Idents, actualSpec.Idents)
				f.equal(len(expectedSpec.Values), len(actualSpec.Values))
				if len(expectedSpec.Values) == len(actualSpec.Values) {
					for i, expr := range expectedSpec.Values {
						f.equalExpr(expr, actualSpec.Values[i])
					}
				}
			default:
				require.Failf(t, "unknown type", "unknown Spec '%T'", expSpec)
			}
		}
	case *IfStmt:
		f.equalStmt(expected.Init, actual.(*IfStmt).Init)
		f.equalExpr(expected.Cond, actual.(*IfStmt).Cond)
		f.equalStmt(expected.Body, actual.(*IfStmt).Body)
		f.equalStmt(expected.Else, actual.(*IfStmt).Else)
		f.equal(expected.IfPos, actual.(*IfStmt).IfPos)
	case *TryStmt:
		f.equal(expected.TryPos, actual.(*TryStmt).TryPos)
		f.equalStmt(expected.Body, actual.(*TryStmt).Body)
		f.equalStmt(expected.Catch, actual.(*TryStmt).Catch)
		f.equalStmt(expected.Finally, actual.(*TryStmt).Finally)
	case *CatchStmt:
		f.equal(expected.CatchPos, actual.(*CatchStmt).CatchPos)
		f.equal(expected.Ident, actual.(*CatchStmt).Ident)
		f.equalStmt(expected.Body, actual.(*CatchStmt).Body)
	case *FinallyStmt:
		f.equal(expected.FinallyPos, actual.(*FinallyStmt).FinallyPos)
		f.equalStmt(expected.Body, actual.(*FinallyStmt).Body)
	case *ThrowStmt:
		f.equal(expected.ThrowPos, actual.(*ThrowStmt).ThrowPos)
		f.equalExpr(expected.Expr, actual.(*ThrowStmt).Expr)
	case *IncDecStmt:
		f.equalExpr(expected.Expr, actual.(*IncDecStmt).Expr)
		f.equal(expected.Token, actual.(*IncDecStmt).Token)
		f.equal(expected.TokenPos, actual.(*IncDecStmt).TokenPos)
	case *ForStmt:
		f.equalStmt(expected.Init, actual.(*ForStmt).Init)
		f.equalExpr(expected.Cond, actual.(*ForStmt).Cond)
		f.equalStmt(expected.Post, actual.(*ForStmt).Post)
		f.equalStmt(expected.Body, actual.(*ForStmt).Body)
		f.equal(expected.ForPos, actual.(*ForStmt).ForPos)
	case *ForInStmt:
		f.equalExpr(expected.Key, actual.(*ForInStmt).Key)
		f.equalExpr(expected.Value, actual.(*ForInStmt).Value)
		f.equalExpr(expected.Iterable, actual.(*ForInStmt).Iterable)
		f.equalStmt(expected.Body, actual.(*ForInStmt).Body)
		f.equal(expected.ForPos, actual.(*ForInStmt).ForPos)
		f.equalStmt(expected.Else, actual.(*ForInStmt).Else)
	case *ReturnStmt:
		f.equalExpr(expected.Result, actual.(*ReturnStmt).Result)
		f.equal(expected.ReturnPos, actual.(*ReturnStmt).ReturnPos)
	case *BranchStmt:
		f.equalExpr(expected.Label, actual.(*BranchStmt).Label)
		f.equal(expected.Token, actual.(*BranchStmt).Token)
		f.equal(expected.TokenPos, actual.(*BranchStmt).TokenPos)
	case *MixedTextStmt:
		f.equal(expected.Lit.Value, actual.(*MixedTextStmt).Lit.Value)
		f.equal(expected.Lit.Pos, actual.(*MixedTextStmt).Lit.Pos)
		f.equal(expected.Flags.String(), actual.(*MixedTextStmt).Flags.String(), "Flags")
	case *MixedValueStmt:
		f.equal(expected.StartLit.Value, actual.(*MixedValueStmt).StartLit.Value)
		f.equal(expected.StartLit.Pos, actual.(*MixedValueStmt).StartLit.Pos)
		f.equal(expected.EndLit.Value, actual.(*MixedValueStmt).EndLit.Value)
		f.equal(expected.EndLit.Pos, actual.(*MixedValueStmt).EndLit.Pos)
		f.equalExpr(expected.Expr, actual.(*MixedValueStmt).Expr)
	case *ConfigStmt:
		f.equal(expected.ConfigPos, actual.(*ConfigStmt).ConfigPos)
		f.equal(expected.Options, actual.(*ConfigStmt).Options)
		f.equal(len(expected.Elements), len(actual.(*ConfigStmt).Elements))
		for i, e := range expected.Elements {
			f.equalExpr(e, actual.(*ConfigStmt).Elements[i])
		}
	case *CodeBeginStmt:
		f.equal(expected.RemoveSpace, actual.(*CodeBeginStmt).RemoveSpace)
		f.equal(expected.Lit.Pos, actual.(*CodeBeginStmt).Lit.Pos)
		f.equal(expected.Lit.Value, actual.(*CodeBeginStmt).Lit.Value)
	case *CodeEndStmt:
		f.equal(expected.RemoveSpace, actual.(*CodeEndStmt).RemoveSpace)
		f.equal(expected.Lit.Pos, actual.(*CodeEndStmt).Lit.Pos)
		f.equal(expected.Lit.Value, actual.(*CodeEndStmt).Lit.Value)
	default:
		panic(fmt.Errorf("unknown type: %T", expected))
	}
}

func (f *fileTester) equalExpr(expected, actual Expr) {
	t := f.t
	if expected == nil || reflect.ValueOf(expected).IsNil() {
		require.Nil(t, actual, "expected nil, but got not nil")
		return
	}
	require.NotNil(t, actual, "expected not nil, but got nil")
	require.IsType(t, expected, actual)

	switch expected := expected.(type) {
	case *IdentExpr:
		f.equal(expected.Name, actual.(*IdentExpr).Name)
		f.equal(expected.NamePos, actual.(*IdentExpr).NamePos)
	case *TypedIdentExpr:
		f.equalExpr(expected.Ident, actual.(*TypedIdentExpr).Ident)
		f.equalIdents(expected.Type, actual.(*TypedIdentExpr).Type)
	case *IntLit:
		f.equal(expected.Value, actual.(*IntLit).Value)
		f.equal(expected.ValuePos, actual.(*IntLit).ValuePos)
	case *FloatLit:
		f.equal(expected.Value,
			actual.(*FloatLit).Value)
		f.equal(expected.ValuePos, actual.(*FloatLit).ValuePos)
	case *DecimalLit:
		require.True(t, expected.Value.Equal(actual.(*DecimalLit).Value))
		f.equal(expected.ValuePos, actual.(*DecimalLit).ValuePos)
	case *BoolLit:
		f.equal(expected.Value, actual.(*BoolLit).Value)
		f.equal(int(expected.ValuePos), int(actual.(*BoolLit).ValuePos))
	case *FlagLit:
		f.equal(expected.Value, actual.(*FlagLit).Value)
		f.equal(expected.ValuePos, actual.(*FlagLit).ValuePos)
	case *CharLit:
		f.equal(expected.Value, actual.(*CharLit).Value)
		f.equal(expected.ValuePos, actual.(*CharLit).ValuePos)
	case *StringLit:
		f.equal(expected.Literal, actual.(*StringLit).Literal)
		f.equal(expected.ValuePos, actual.(*StringLit).ValuePos)
	case *RawStringLit:
		f.equal(expected.UnquotedValue(), actual.(*RawStringLit).UnquotedValue())
		f.equal(expected.LiteralPos, actual.(*RawStringLit).LiteralPos)
	case *RawHeredocLit:
		f.equal(expected.Literal, actual.(*RawHeredocLit).Literal)
		f.equal(expected.LiteralPos, actual.(*RawHeredocLit).LiteralPos)
	case *ArrayExpr:
		f.equal(expected.LBrack, actual.(*ArrayExpr).LBrack)
		f.equal(expected.RBrack, actual.(*ArrayExpr).RBrack)
		f.equalExprs(expected.Elements, actual.(*ArrayExpr).Elements)
	case *DictExpr:
		f.equal(expected.LBrace, actual.(*DictExpr).LBrace)
		f.equal(expected.RBrace, actual.(*DictExpr).RBrace)
		f.equalMapElements(expected.Elements, actual.(*DictExpr).Elements)
	case *NilLit:
		f.equal(expected.TokenPos, actual.(*NilLit).TokenPos)
	case *ReturnExpr:
		f.equal(expected.ReturnPos, actual.(*ReturnExpr).ReturnPos)
		f.equalExpr(expected.Result, actual.(*ReturnExpr).Result)
	case *NullishSelectorExpr:
		f.equalExpr(expected.Expr, actual.(*NullishSelectorExpr).Expr)
		f.equalExpr(expected.Sel, actual.(*NullishSelectorExpr).Sel)
	case *BinaryExpr:
		f.equalExpr(expected.LHS, actual.(*BinaryExpr).LHS)
		f.equalExpr(expected.RHS, actual.(*BinaryExpr).RHS)
		f.equal(expected.Token, actual.(*BinaryExpr).Token)
		f.equal(expected.TokenPos, actual.(*BinaryExpr).TokenPos)
	case *UnaryExpr:
		f.equalExpr(expected.Expr, actual.(*UnaryExpr).Expr)
		f.equal(expected.Token, actual.(*UnaryExpr).Token)
		f.equal(expected.TokenPos, actual.(*UnaryExpr).TokenPos)
	case *FuncExpr:
		f.equalFuncType(expected.Type, actual.(*FuncExpr).Type)
		f.equalStmt(expected.Body, actual.(*FuncExpr).Body)
	case *CallExpr:
		actual := actual.(*CallExpr)
		f.equalExpr(expected.Func, actual.Func)
		f.equal(expected.LParen, actual.LParen)
		f.equal(expected.RParen, actual.RParen)
		f.equalExprs(expected.Args.Values, actual.Args.Values)

		if expected.Args.Var == nil && actual.Args.Var != nil {
			require.Nil(t, expected.Args.Var)
		}

		if expected.Args.Var != nil && actual.Args.Var == nil {
			require.NotNil(t, expected.Args.Var)
		}

		if expected.Args.Var != nil && actual.Args.Var != nil {
			f.equal(expected.Args.Var.TokenPos,
				actual.Args.Var.TokenPos)
			f.equalExpr(expected.Args.Var.Value,
				actual.Args.Var.Value)
		}

		if expected.NamedArgs.Var == nil && actual.NamedArgs.Var != nil {
			require.Nil(t, expected.NamedArgs.Var)
		}

		if expected.NamedArgs.Var != nil && actual.NamedArgs.Var == nil {
			require.NotNil(t, expected.NamedArgs.Var)
		}

		if expected.NamedArgs.Var != nil && actual.NamedArgs.Var != nil {
			f.equal(expected.NamedArgs.Var.TokenPos,
				actual.NamedArgs.Var.TokenPos)
			f.equalExpr(expected.NamedArgs.Var.Value,
				actual.NamedArgs.Var.Value)
		}

		f.equalNamedArgsNames(expected.NamedArgs.Names, actual.NamedArgs.Names)
		f.equalExprs(expected.NamedArgs.Values, actual.NamedArgs.Values)
	case *ParenExpr:
		f.equalExpr(expected.Expr, actual.(*ParenExpr).Expr)
		f.equal(expected.LParen, actual.(*ParenExpr).LParen)
		f.equal(expected.RParen, actual.(*ParenExpr).RParen)
	case *IndexExpr:
		f.equalExpr(expected.Expr, actual.(*IndexExpr).Expr)
		f.equalExpr(expected.Index, actual.(*IndexExpr).Index)
		f.equal(expected.LBrack, actual.(*IndexExpr).LBrack)
		f.equal(expected.RBrack, actual.(*IndexExpr).RBrack)
	case *SliceExpr:
		f.equalExpr(expected.Expr, actual.(*SliceExpr).Expr)
		f.equalExpr(expected.Low, actual.(*SliceExpr).Low)
		f.equalExpr(expected.High, actual.(*SliceExpr).High)
		f.equal(expected.LBrack, actual.(*SliceExpr).LBrack)
		f.equal(expected.RBrack, actual.(*SliceExpr).RBrack)
	case *SelectorExpr:
		f.equalExpr(expected.Expr, actual.(*SelectorExpr).Expr)
		f.equalExpr(expected.Sel, actual.(*SelectorExpr).Sel)
	case *ImportExpr:
		f.equal(expected.ModuleName, actual.(*ImportExpr).ModuleName)
		f.equal(int(expected.TokenPos), int(actual.(*ImportExpr).TokenPos))
		f.equal(expected.Token, actual.(*ImportExpr).Token)
	case *EmbedExpr:
		f.equal(expected.Path, actual.(*EmbedExpr).Path)
		f.equal(int(expected.TokenPos), int(actual.(*EmbedExpr).TokenPos))
		f.equal(expected.Token, actual.(*EmbedExpr).Token)
	case *CondExpr:
		f.equalExpr(expected.Cond, actual.(*CondExpr).Cond)
		f.equalExpr(expected.True, actual.(*CondExpr).True)
		f.equalExpr(expected.False, actual.(*CondExpr).False)
		f.equal(expected.QuestionPos, actual.(*CondExpr).QuestionPos)
		f.equal(expected.ColonPos, actual.(*CondExpr).ColonPos)
	case *CalleeKeywordExpr:
		f.equal(expected.Literal, actual.(*CalleeKeywordExpr).Literal)
		f.equal(expected.TokenPos, actual.(*CalleeKeywordExpr).TokenPos)
	case *ArgsKeywordExpr:
		f.equal(expected.Literal, actual.(*ArgsKeywordExpr).Literal)
		f.equal(expected.TokenPos, actual.(*ArgsKeywordExpr).TokenPos)
	case *NamedArgsKeywordExpr:
		f.equal(expected.Literal, actual.(*NamedArgsKeywordExpr).Literal)
		f.equal(expected.TokenPos, actual.(*NamedArgsKeywordExpr).TokenPos)
	case *ClosureExpr:
		f.equalFuncType(expected.Type, actual.(*ClosureExpr).Type)
		f.equalExpr(expected.Body, actual.(*ClosureExpr).Body)
	case *BlockExpr:
		f.equalStmt(expected.BlockStmt, actual.(*BlockExpr).BlockStmt)
	case *KeyValueLit:
		f.equalExpr(expected.Key, actual.(*KeyValueLit).Key)
		f.equalExpr(expected.Value, actual.(*KeyValueLit).Value)
	case *KeyValuePairLit:
		f.equalExpr(expected.Key, actual.(*KeyValuePairLit).Key)
		f.equalExpr(expected.Value, actual.(*KeyValuePairLit).Value)
	case *KeyValueSepLit:
		f.equal(expected.TokenPos, actual.(*KeyValueSepLit).TokenPos)
	default:
		panic(fmt.Errorf("unknown type: %T", expected))
	}
}

func (f *fileTester) equalFuncType(expected, actual *FuncType) {
	f.equal(expected.Params.LParen, actual.Params.LParen)
	f.equal(expected.Params.RParen, actual.Params.RParen)
	f.equalTypedIdents(expected.Params.Args.Values, actual.Params.Args.Values)
	f.equalNamedArgs(&expected.Params.NamedArgs, &actual.Params.NamedArgs)
}

func (f *fileTester) equalNamedArgs(expected, actual *NamedArgsList) {
	if expected == nil && actual == nil {
		return
	}
	require.NotNil(f.t, expected, "expected is nil")
	require.NotNil(f.t, actual, "actual is nil")

	f.equal(expected.Var, actual.Var)
	f.equalTypedIdents(expected.Names, actual.Names)
	f.equalExprs(expected.Values, actual.Values)
}

func (f *fileTester) equalNamedArgsNames(expected, actual []NamedArgExpr) {
	f.equal(len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		f.equalExpr(expected[i].Ident, actual[i].Ident)
		f.equalExpr(expected[i].Lit, actual[i].Lit)
	}
}

func (f *fileTester) equalIdents(expected, actual []*IdentExpr) {
	f.equal(len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		f.equalExpr(expected[i], actual[i])
	}
}

func (f *fileTester) equalTypedIdents(expected, actual []*TypedIdentExpr) {
	f.equal(len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		f.equalExpr(expected[i], actual[i])
	}
}

func (f *fileTester) equalExprs(expected, actual []Expr) {
	f.equal(len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		f.equalExpr(expected[i], actual[i])
	}
}

func (f *fileTester) equalStmts(expected, actual []Stmt) {
	f.equal(len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		f.equalStmt(expected[i], actual[i])
	}
}

func (f *fileTester) equalMapElements(
	expected, actual []*DictElementLit,
) {
	f.equal(len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		f.equal(expected[i].Key, actual[i].Key)
		f.equal(expected[i].KeyPos, actual[i].KeyPos)
		f.equal(expected[i].ColonPos, actual[i].ColonPos)
		f.equalExpr(expected[i].Value, actual[i].Value)
	}
}

func parseSource(
	filename string,
	src []byte,
	trace io.Writer,
	mode Mode,
) (res *File, err error) {
	fileSet := NewFileSet()
	file := fileSet.AddFileData(filename, -1, src)

	p := NewParserWithOptions(file, &ParserOptions{Trace: trace, Mode: mode}, &ScannerOptions{
		MixedDelimiter: mixedDelimiter,
	})
	return p.ParseFile()
}

var mixedDelimiter = MixedDelimiter{
	Start: []rune("‹"),
	End:   []rune("›"),
}
