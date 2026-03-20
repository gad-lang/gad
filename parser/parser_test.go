package parser_test

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/gad-lang/gad/parser/ast"
	. "github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/parser/test"
	"github.com/shopspring/decimal"
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
	defaultExpectParse0 := func(input string, fn expectedFn, opt ...test.ExpectOpt) {
		test.ExpectParse(t, input, fn, append(opt, func(ctx *test.ParseContext) {
			ctx.ParserOptions.Mode |= ParseMixed
			ctx.ScannerOptions.Mode |= ScanMixed
			ctx.ScannerOptions.MixedDelimiter.Start = []rune("‹‹‹")
			ctx.ScannerOptions.MixedDelimiter.End = []rune("›››")
		})...)
	}

	defaultExpectParse0(`‹‹‹
//!!0
x := func() { throw error("bad code")  }
›››<p>‹‹‹=
//!!1
x()
›››</p>`, func(p pfn) []Stmt {
		return stmts(
			codeBegin(lit("‹‹‹", p(1, 1)), false),
			assignStmt(
				exprs(EIdent("x", p(3, 1))),
				exprs(funcLit(funcType(p(3, 6), nil, p(3, 10), p(3, 11)),
					blockStmt(p(3, 13), p(3, 40), throwStmt(p(3, 15), callExpr(EIdent("error", p(3, 21)), p(3, 26), p(3, 37), callExprArgs(nil, stringLit("bad code", p(3, 27)))))))),
				token.Define, p(3, 3),
			),
			codeEnd(lit("›››", p(4, 1)), false),
			mixedTextStmt(p(4, 10), "<p>"),
			mixedValue(lit("‹‹‹", p(4, 13)), lit("›››", p(7, 1)), callExpr(EIdent("x", p(6, 1)), p(6, 2), p(6, 3))),
			mixedTextStmt(p(7, 10), "</p>"),
		)
	})

	defaultExpectParse := func(input string, fn expectedFn) {
		test.ExpectParse(t, input, fn, func(ctx *test.ParseContext) {
			ctx.ScannerOptions.MixedDelimiter = DefaultMixedDelimiter
		})
	}

	defaultExpectParse(`# gad: mixed
	{% 
	1
%} 
`, func(p pfn) []Stmt {
		return stmts(
			config(p(1, 1), KVp(EIdent("mixed", p(1, 8)))),
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
			config(p(1, 1), KVp(EIdent("mixed", p(1, 8)))),
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
			config(p(1, 1), KVp(EIdent("mixed", p(1, 8)))),
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
			config(p(1, 1), KVp(EIdent("mixed", p(1, 8)))),
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
			config(p(1, 1), KVp(EIdent("mixed", p(1, 8)))),
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
			config(p(1, 1), KVp(EIdent("mixed", p(1, 8)))),
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
	test.ExpectParseMixed(t, "‹- 1 -› a", func(p pfn) []Stmt {
		return stmts(
			codeBegin(lit("‹", p(1, 1)), true),
			exprStmt(intLit(1, p(1, 6))),
			codeEnd(lit("›", p(1, 9)), true),
			mixedTextStmt(p(1, 12), " a", RemoveLeftSpaces),
		)
	})
	test.ExpectParseStringMixed(t, "‹- var myfn -› a", "‹-; var myfn; -› a")
	test.ExpectParseStringMixed(t, "a ‹- 1 ›", "a ; ‹-; 1; ›")
	test.ExpectParseStringMixed(t, "‹ 1 ›", "‹; 1; ›")
	test.ExpectParseStringMixed(t, "‹ 1; 2; var a ›", "‹; 1; 2; var a; ›")
	test.ExpectParseStringMixed(t, "x ‹ 1; 2; var a › y", "x ; ‹; 1; 2; var a; › y")
	test.ExpectParseStringMixed(t, "‹var a›", `‹; var a; ›`)
	test.ExpectParseStringMixed(t, "‹=1›", "‹=1›")
	test.ExpectParseStringMixed(t, "a  ‹-= 1 -›\n\tb", "a  ; ‹-=1-›; \n\tb")
	test.ExpectParseStringMixed(t, "‹(› 2 ‹- ) ›", "‹; (› 2 ‹-); ›")
	test.ExpectParseStringMixed(t, "‹( -› 2 ‹- ) ›", "‹; (-› 2 ‹-); ›")
	test.ExpectParseStringMixed(t, "‹a = (› 2 ‹- ) ›", "‹; a = (› 2 ‹-); ›")
	test.ExpectParseStringMixed(t, "‹1›‹2›‹3›", `‹; 1; 2; 3; ›`)
	test.ExpectParseStringMixed(t, "‹1›‹›‹3›", `‹; 1; 3; ›`)
	test.ExpectParseStringMixed(t, "‹1›‹=2›‹3›", `‹; 1; ›‹=2›‹; 3; ›`)
	test.ExpectParseStringMixed(t, "abc", "abc")
	test.ExpectParseStringMixed(t, "a‹1›", "a; ‹; 1; ›")
	test.ExpectParseStringMixed(t, "a‹  1  ›b", "a; ‹; 1; ›b")
	test.ExpectParseStringMixed(t, "a‹1?2:3   ›b‹=   2 + 4›", "a; ‹; (1 ? 2 : 3); ›b‹=(2 + 4)›")
	test.ExpectParseStringMixed(t, "a‹1?2:3;fn();x++   ›b‹=   2 + 4›", "a; ‹; (1 ? 2 : 3); fn(); x++; ›b‹=(2 + 4)›")
	test.ExpectParseStringMixed(t, "a\n‹- 1›\tb\n‹-= 2 -›\n\nc", "a\n; ‹-; 1; ›\tb\n‹-=2-›\n\nc")
	test.ExpectParseStringMixed(t, `a‹=1›c‹x := 5›‹=x›`, "a; ‹=1›; c; ‹; x := 5; ›‹=x›")

	test.ExpectParseStringMixed(t, "‹if 1›2‹end›", "‹; if 1  ›2‹ end; ›")
	test.ExpectParseStringMixed(t, "‹if 1 then›2‹end›", "‹; if 1 then ›2‹ end; ›")
	test.ExpectParseStringMixed(t, "‹if 1 then›2‹else if 3 then›4‹end›", "‹; if 1 then ›2‹ else if 3 then ›4‹ end; ›")
	test.ExpectParseStringMixed(t, "‹if 1 then›2‹else›3‹end›", "‹; if 1 then ›2‹ else ›3‹ end; ›")
	test.ExpectParseStringMixed(t, "‹if 1 then›2‹if 2 then›3‹end›‹end›", "‹; if 1 then ›2‹; if 2 then ›3‹ end end; ›")
	test.ExpectParseStringMixed(t, "‹ if 1 then › 2 ‹ end ›", "‹; if 1 then › 2 ‹ end; ›")

	test.ExpectParseStringMixed(t, "‹for a in b›2‹end›", "‹; for _, a in b  ›2‹ end; ›")
	test.ExpectParseStringMixed(t, "‹for i:=0;i<2;i++›v‹end›", "‹; for i := 0 ; (i < 2)  ; i++ ›v‹ end; ›")
	test.ExpectParseStringMixed(t, "‹for e in list›1‹end›", "‹; for _, e in list  ›1‹ end; ›")
	test.ExpectParseStringMixed(t, "‹for e in list›‹=1›‹end›", "‹; for _, e in list  ›‹=1›‹ end; ›")
	test.ExpectParseStringMixed(t, "‹for e in list {›1‹}else{›2‹}›", "‹; for _, e in list { ›1‹ } else { ›2‹ }; ›")
	test.ExpectParseStringMixed(t, "‹for e in list {›1   ‹-}else{›2‹}›", "‹; for _, e in list { ›1   ‹- } else { ›2‹ }; ›")

	test.ExpectParseStringMixed(t, "‹try›1‹finally›2‹end›", "‹; try  ›1‹  finally  ›2‹ end; ›")
	test.ExpectParseStringMixed(t, "‹try›1‹catch e›2‹finally›3‹end›", "‹; try  ›1‹  catch e  ›2‹  finally  ›3‹ end; ›")
	test.ExpectParseStringMixed(t, "abc ‹=\n// my single comment\n\n/* long\n comment\n\n*/\n1›def", "abc ; ‹=1›; def")

	// example for auto generated mixed script mapping multiples sources
	test.ExpectParseStringMixed(t, `
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
	test.ExpectParseError(t, "var x;\n\nvar y;\nparam a,b\nvar z\nz2\nz3\nz4",
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

	test.ExpectParseError(t, `param a,b`,
		[2]string{"%v", "Parse Error: expected statement, found ','\n\tat test:1:8"},
		[2]string{"%+v", "Parse Error: expected statement, found ','\n\tat test:1:8\n\n       🠆 1| param a,b\n                   ^"},
	)

	test.ExpectParseString(t, `a := throw "my error"`, `a := throw "my error"`)

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
	test.ExpectParseString(t, "(a.b.|x().|y().z.|c(1).d.e.|f().g.h.i)", "((((a.b .| x()) .| y().z) .| c(1).d.e) .| f().g.h.i)")
	test.ExpectParseString(t, "a.b.|x().|y().z.|c(1).d.e.|f().g.h.i", "((((a.b .| x()) .| y().z) .| c(1).d.e) .| f().g.h.i)")
	test.ExpectParseString(t, "a.b.|x().|y().z.|c(1).d.e", "(((a.b .| x()) .| y().z) .| c(1).d.e)")
	test.ExpectParseString(t, "a.b.|x().|y()", "((a.b .| x()) .| y())")
	test.ExpectParseString(t, "a.b.|x().|y().z", "((a.b .| x()) .| y().z)")
	test.ExpectParseString(t, "a.b.|x().|y().z.|a(1).c", "(((a.b .| x()) .| y().z) .| a(1).c)")
}

func TestParseDecl(t *testing.T) {
	test.ExpectParse(t, `param a`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Param, p(1, 1), 0, 0,
					paramSpec(false, typedIdent(EIdent("a", p(1, 7)))),
				),
			),
		)
	})
	test.ExpectParse(t, `param *a;`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Param, p(1, 1), 0, 0,
					paramSpec(true, typedIdent(EIdent("a", p(1, 8)))),
				),
			),
		)
	})
	test.ExpectParse(t, `param (a, *b)`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Param, p(1, 1), p(1, 7), p(1, 13),
					paramSpec(false, typedIdent(EIdent("a", p(1, 8)))),
					paramSpec(true, typedIdent(EIdent("b", p(1, 12)))),
				),
			),
		)
	})
	test.ExpectParse(t, `param (a,
*b)`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Param, p(1, 1), p(1, 7), p(2, 3),
					paramSpec(false, typedIdent(EIdent("a", p(1, 8)))),
					paramSpec(true, typedIdent(EIdent("b", p(2, 2)))),
				),
			),
		)
	})

	test.ExpectParse(t, `param (a, *b; c=1, d=2, **e)`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Param, p(1, 1), p(1, 7), p(1, 28),
					paramSpec(false, typedIdent(EIdent("a", p(1, 8)))),
					paramSpec(true, typedIdent(EIdent("b", p(1, 12)))),
					nparamSpec(typedIdent(EIdent("c", p(1, 15))), intLit(1, p(1, 17))),
					nparamSpec(typedIdent(EIdent("d", p(1, 20))), intLit(2, p(1, 22))),
					nparamSpecVar(typedIdent(EIdent("e", p(1, 27)))),
				),
			),
		)
	})
	test.ExpectParse(t, `param (;c=1, d=2, **e)`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Param, p(1, 1), p(1, 7), p(1, 22),
					nparamSpec(typedIdent(EIdent("c", p(1, 9))), intLit(1, p(1, 11))),
					nparamSpec(typedIdent(EIdent("d", p(1, 14))), intLit(2, p(1, 16))),
					nparamSpecVar(typedIdent(EIdent("e", p(1, 21)))),
				),
			),
		)
	})
	test.ExpectParse(t, `param (;c=1, d=2, **e)`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Param, p(1, 1), p(1, 7), p(1, 22),
					nparamSpec(typedIdent(EIdent("c", p(1, 9))), intLit(1, p(1, 11))),
					nparamSpec(typedIdent(EIdent("d", p(1, 14))), intLit(2, p(1, 16))),
					nparamSpecVar(typedIdent(EIdent("e", p(1, 21)))),
				),
			),
		)
	})

	test.ExpectParse(t, `param (a, *b; c=1, d=2, x, **e)`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Param, p(1, 1), p(1, 7), p(1, 31),
					paramSpec(false, typedIdent(EIdent("a", p(1, 8)))),
					paramSpec(true, typedIdent(EIdent("b", p(1, 12)))),
					nparamSpec(typedIdent(EIdent("c", p(1, 15))), intLit(1, p(1, 17))),
					nparamSpec(typedIdent(EIdent("d", p(1, 20))), intLit(2, p(1, 22))),
					nparamSpec(typedIdent(EIdent("x", p(1, 25))), nil),
					nparamSpecVar(typedIdent(EIdent("e", p(1, 30)))),
				),
			),
		)
	})

	test.ExpectParseString(t, "param x", "param x")
	test.ExpectParseString(t, "param (\nx,\n)", "param (x)")
	test.ExpectParseString(t, "param (\nx,\ny)", "param (x, y)")
	test.ExpectParseString(t, "param (\nx,\ny,\n)", "param (x, y)")
	test.ExpectParseString(t, "param (x,y)", "param (x, y)")
	test.ExpectParseString(t, "param (x,\ny)", "param (x, y)")
	test.ExpectParseString(t, "param (x,\ny)", "param (x, y)")
	test.ExpectParseString(t, "param *x", "param *x")
	test.ExpectParseString(t, "param **x", "param (; **x)")
	test.ExpectParseString(t, "param b=2", "param (; b=2)")
	test.ExpectParseString(t, "param (x,*y)", "param (x, *y)")
	test.ExpectParseString(t, "param (x,\n*y)", "param (x, *y)")
	test.ExpectParseString(t, "param (;c=1, d=2, **e)", "param (; c=1, d=2, **e)")
	test.ExpectParseString(t, "param (a, *b;c=1, d=2, **e)", "param (a, *b; c=1, d=2, **e)")
	test.ExpectParseString(t, "param (a,\n*b\n; c=2,\nx=5)", "param (a, *b; c=2, x=5)")

	test.ExpectParseString(t, "param x int", "param x int")
	test.ExpectParseString(t, "param x a.b.c|int", "param x a.b.c|int")
	test.ExpectParseString(t, "param x a[1].b.(c).c|int", "param x a[1].b.(c).c|int")
	test.ExpectParseString(t, "param x int|bool", "param x int|bool")
	test.ExpectParseString(t, "param (\nx int,\n)", "param (x int)")
	test.ExpectParseString(t, "param (\nx int,\ny)", "param (x int, y)")
	test.ExpectParseString(t, "param (\nx int,\ny, z string|bool)", "param (x int, y, z string|bool)")
	test.ExpectParseString(t, "param *x int", "param *x int")
	test.ExpectParseString(t, "param *x int|bool", "param *x int|bool")
	test.ExpectParseString(t, "param **x int", "param (; **x int)")
	test.ExpectParseString(t, "param **x int|bool", "param (; **x int|bool)")
	test.ExpectParseString(t, "param b int=2", "param (; b int=2)")
	test.ExpectParseString(t, "param b bool|int=2", "param (; b bool|int=2)")
	test.ExpectParseString(t, "param (a, *b; x bool|int=2, **y)",
		"param (a, *b; x bool|int=2, **y)")

	test.ExpectParse(t, `global a`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Global, p(1, 1), 0, 0,
					paramSpec(false, typedIdent(EIdent("a", p(1, 8)))),
				),
			),
		)
	})
	test.ExpectParse(t, `
global a
global b`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Global, p(2, 1), 0, 0,
					paramSpec(false, typedIdent(EIdent("a", p(2, 8)))),
				),
			),
			declStmt(
				genDecl(token.Global, p(3, 1), 0, 0,
					paramSpec(false, typedIdent(EIdent("b", p(3, 8)))),
				),
			),
		)
	})
	test.ExpectParse(t, `global (a, b)`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Global, p(1, 1), p(1, 8), p(1, 13),
					paramSpec(false, typedIdent(EIdent("a", p(1, 9)))),
					paramSpec(false, typedIdent(EIdent("b", p(1, 12)))),
				),
			),
		)
	})
	test.ExpectParse(t, `global (a, 
b)`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Global, p(1, 1), p(1, 8), p(2, 2),
					paramSpec(false, typedIdent(EIdent("a", p(1, 9)))),
					paramSpec(false, typedIdent(EIdent("b", p(2, 1)))),
				),
			),
		)
	})
	test.ExpectParse(t, `global (a 
b)`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Global, p(1, 1), p(1, 8), p(2, 2),
					paramSpec(false, typedIdent(EIdent("a", p(1, 9)))),
					paramSpec(false, typedIdent(EIdent("b", p(2, 1)))),
				),
			),
		)
	})
	test.ExpectParseString(t, "global x", "global x")
	test.ExpectParseString(t, "global (\nx\n)", "global (x)")
	test.ExpectParseString(t, "global (x,y)", "global (x, y)")
	test.ExpectParseString(t, "global (x\ny)", "global (x, y)")
	test.ExpectParseString(t, "global (\nx\ny)", "global (x, y)")
	test.ExpectParseString(t, "global (x,\ny)", "global (x, y)")
	test.ExpectParseString(t, "global (x\ny)", "global (x, y)")

	test.ExpectParse(t, `var a`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Var, p(1, 1), 0, 0,
					valueSpec(
						[]*IdentExpr{EIdent("a", p(1, 5))},
						[]Expr{nil}),
				),
			),
		)
	})
	test.ExpectParse(t, `var a=1`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Var, p(1, 1), 0, 0,
					valueSpec(
						[]*IdentExpr{EIdent("a", p(1, 5))},
						[]Expr{intLit(1, p(1, 7))}),
				),
			),
		)
	})
	test.ExpectParse(t, `var a;var b`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Var, p(1, 1), 0, 0,
					valueSpec(
						[]*IdentExpr{EIdent("a", p(1, 5))},
						[]Expr{nil}),
				),
			),
			declStmt(
				genDecl(token.Var, p(1, 7), 0, 0,
					valueSpec(
						[]*IdentExpr{EIdent("b", p(1, 11))},
						[]Expr{nil}),
				),
			),
		)
	})
	test.ExpectParse(t, `var a="x";var b`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Var, p(1, 1), 0, 0,
					valueSpec(
						[]*IdentExpr{EIdent("a", p(1, 5))},
						[]Expr{stringLit("x", p(1, 7))}),
				),
			),
			declStmt(
				genDecl(token.Var, p(1, 11), 0, 0,
					valueSpec(
						[]*IdentExpr{EIdent("b", p(1, 15))},
						[]Expr{nil}),
				),
			),
		)
	})
	test.ExpectParse(t, `
var a
var b
`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Var, p(2, 1), 0, 0,
					valueSpec(
						[]*IdentExpr{EIdent("a", p(2, 5))},
						[]Expr{nil}),
				),
			),
			declStmt(
				genDecl(token.Var, p(3, 1), 0, 0,
					valueSpec(
						[]*IdentExpr{EIdent("b", p(3, 5))},
						[]Expr{nil}),
				),
			),
		)
	})
	test.ExpectParse(t, `
var a
var b=2
`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Var, p(2, 1), 0, 0,
					valueSpec(
						[]*IdentExpr{EIdent("a", p(2, 5))},
						[]Expr{nil}),
				),
			),
			declStmt(
				genDecl(token.Var, p(3, 1), 0, 0,
					valueSpec(
						[]*IdentExpr{EIdent("b", p(3, 5))},
						[]Expr{intLit(2, p(3, 7))}),
				),
			),
		)
	})
	test.ExpectParse(t, `var (a, b=2)`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Var, p(1, 1), p(1, 5), p(1, 12),
					valueSpec(
						[]*IdentExpr{EIdent("a", p(1, 6))},
						[]Expr{nil}),
					valueSpec(
						[]*IdentExpr{EIdent("b", p(1, 9))},
						[]Expr{intLit(2, p(1, 11))}),
				),
			),
		)
	})
	test.ExpectParse(t, `var (a=1, b=2)`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Var, p(1, 1), p(1, 5), p(1, 14),
					valueSpec(
						[]*IdentExpr{EIdent("a", p(1, 6))},
						[]Expr{intLit(1, p(1, 8))}),
					valueSpec(
						[]*IdentExpr{EIdent("b", p(1, 11))},
						[]Expr{intLit(2, p(1, 13))}),
				),
			),
		)
	})
	test.ExpectParse(t, `var (a=1,
b=2)`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Var, p(1, 1), p(1, 5), p(2, 4),
					valueSpec(
						[]*IdentExpr{EIdent("a", p(1, 6))},
						[]Expr{intLit(1, p(1, 8))}),
					valueSpec(
						[]*IdentExpr{EIdent("b", p(2, 1))},
						[]Expr{intLit(2, p(2, 3))}),
				),
			),
		)
	})
	test.ExpectParse(t, `var (a=1
b=2)`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Var, p(1, 1), p(1, 5), p(2, 4),
					valueSpec(
						[]*IdentExpr{EIdent("a", p(1, 6))},
						[]Expr{intLit(1, p(1, 8))}),
					valueSpec(
						[]*IdentExpr{EIdent("b", p(2, 1))},
						[]Expr{intLit(2, p(2, 3))}),
				),
			),
		)
	})
	test.ExpectParseString(t, "var x", "var x")
	test.ExpectParseString(t, "var (\nx\n)", "var (x)")
	test.ExpectParseString(t, "var (x,y)", "var (x, y)")
	test.ExpectParseString(t, "var (x\ny)", "var (x, y)")
	test.ExpectParseString(t, "var (\nx\ny)", "var (x, y)")
	test.ExpectParseString(t, "var (x,\ny)", "var (x, y)")
	test.ExpectParseString(t, "var (x=1,\ny)", "var (x = 1, y)")
	test.ExpectParseString(t, "var (x,\ny = 2)", "var (x, y = 2)")
	test.ExpectParseString(t, "var (x\ny)", "var (x, y)")
	test.ExpectParseString(t, `var (_, _a, $_a, a, A, $b, $, a1, $1, $b1, $$, ŝ, $ŝ)`,
		`var (_, _a, $_a, a, A, $b, $, a1, $1, $b1, $$, ŝ, $ŝ)`)

	test.ExpectParse(t, `const a = 1`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Const, p(1, 1), 0, 0,
					valueSpec(
						[]*IdentExpr{EIdent("a", p(1, 7))},
						[]Expr{intLit(1, p(1, 11))}),
				),
			),
		)
	})
	test.ExpectParse(t, `const a = 1; const b = 2`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Const, p(1, 1), 0, 0,
					valueSpec(
						[]*IdentExpr{EIdent("a", p(1, 7))},
						[]Expr{intLit(1, p(1, 11))}),
				),
			),
			declStmt(
				genDecl(token.Const, p(1, 14), 0, 0,
					valueSpec(
						[]*IdentExpr{EIdent("b", p(1, 20))},
						[]Expr{intLit(2, p(1, 24))}),
				),
			),
		)
	})
	test.ExpectParse(t, `const (a = 1, b = 2)`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Const, p(1, 1), p(1, 7), p(1, 20),
					valueSpec(
						[]*IdentExpr{EIdent("a", p(1, 8))},
						[]Expr{intLit(1, p(1, 12))}),
					valueSpec(
						[]*IdentExpr{EIdent("b", p(1, 15))},
						[]Expr{intLit(2, p(1, 19))}),
				),
			),
		)
	})
	test.ExpectParse(t, `
const (
    a = 1
    b = 2
)`, func(p pfn) []Stmt {
		return stmts(
			declStmt(
				genDecl(token.Const, p(2, 1), p(2, 7), p(5, 1),
					valueSpec(
						[]*IdentExpr{EIdent("a", p(3, 5))},
						[]Expr{intLit(1, p(3, 9))}),
					valueSpec(
						[]*IdentExpr{EIdent("b", p(4, 5))},
						[]Expr{intLit(2, p(4, 9))}),
				),
			),
		)
	})
	test.ExpectParseString(t, "const x=1", "const x = 1")
	test.ExpectParseString(t, "const (\nx=1\n)", "const (x = 1)")
	test.ExpectParseString(t, "const (x=1,y=2)", "const (x = 1, y = 2)")
	test.ExpectParseString(t, "const (x=1\ny=2)", "const (x = 1, y = 2)")
	test.ExpectParseString(t, "const (\nx=1\ny=2)", "const (x = 1, y = 2)")
	test.ExpectParseString(t, "const (x=1,\ny=2)", "const (x = 1, y = 2)")

	test.ExpectParseError(t, `param a,b`)
	test.ExpectParseError(t, `param (a... ,b)`)
	test.ExpectParseError(t, `param (... ,b)`)
	test.ExpectParseError(t, `param (...)`)
	test.ExpectParseError(t, `param ...`)
	test.ExpectParseError(t, `param (a, b...)`)
	test.ExpectParseError(t, `param a,b)`)
	test.ExpectParseError(t, `param (a,b`)
	test.ExpectParseError(t, `param (a...,b...`)
	test.ExpectParseError(t, `param (...a,...b`)
	test.ExpectParseError(t, `param a,`)
	test.ExpectParseError(t, `param ,a`)
	test.ExpectParseError(t, `global a...`)
	test.ExpectParseError(t, `global a,b`)
	test.ExpectParseError(t, `global a,b)`)
	test.ExpectParseError(t, `global (a,b`)
	test.ExpectParseError(t, `global a,`)
	test.ExpectParseError(t, `global ,a`)
	test.ExpectParseError(t, `var a,b`)
	test.ExpectParseError(t, `var ...a`)
	test.ExpectParseError(t, `var a...`)
	test.ExpectParseError(t, `var a,b)`)
	test.ExpectParseError(t, `var (a,b`)
	test.ExpectParseError(t, `var a,`)
	test.ExpectParseError(t, `var ,a`)
	test.ExpectParseError(t, `const a=1,b=2`)

	// After iota support, this should be valid.
	//	expectParseError(t, `const (a=1,b)`)

	test.ExpectParseError(t, `const a`)
	test.ExpectParseError(t, `const (a)`)
	test.ExpectParseError(t, `const (a,b)`)
	test.ExpectParseError(t, `const (a=1`)
	test.ExpectParseError(t, `const (a`)
	test.ExpectParseError(t, `const a=1,`)
	test.ExpectParseError(t, `const ,a=2`)
}

func TestCommaSepReturn(t *testing.T) {
	test.ExpectParse(t, "return 1, 23", func(p pfn) []Stmt {
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
	test.ExpectParse(t, "return 1, 23, 2.2, 12.34d", func(p pfn) []Stmt {
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
	test.ExpectParse(t, "return a, b", func(p pfn) []Stmt {
		return stmts(
			returnStmt(
				p(1, 1),
				arrayLit(
					p(1, 8),
					p(1, 12),
					EIdent("a", p(1, 8)),
					EIdent("b", p(1, 11)),
				),
			),
		)
	})
	test.ExpectParse(t, "func() { return a, b }", func(p pfn) []Stmt {
		return stmts(
			funcStmt(
				EFunc(
					funcType(p(1, 1), nil, p(1, 5), p(1, 6)),
					blockStmt(
						p(1, 8),
						p(1, 22),
						returnStmt(
							p(1, 10),
							arrayLit(
								p(1, 17),
								p(1, 21),
								EIdent("a", p(1, 17)),
								EIdent("b", p(1, 20)),
							),
						),
					),
				),
			),
		)
	})
	test.ExpectParseError(t, `return a,`)
	test.ExpectParseError(t, `return a,b,`)
	test.ExpectParseError(t, `return a,`)
	test.ExpectParseError(t, `func() { return a, }`)
	test.ExpectParseError(t, `func() { return a,b, }`)
}

func TestParseArray(t *testing.T) {
	test.ExpectParse(t, "[1, 2, 3]", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				arrayLit(p(1, 1), p(1, 9),
					intLit(1, p(1, 2)),
					intLit(2, p(1, 5)),
					intLit(3, p(1, 8)))))
	})

	test.ExpectParse(t, `
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
	test.ExpectParse(t, `
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

	test.ExpectParse(t, `[1, "foo", 12.34]`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				arrayLit(p(1, 1), p(1, 17),
					intLit(1, p(1, 2)),
					stringLit("foo", p(1, 5)),
					floatLit(12.34, p(1, 12)))))
	})

	test.ExpectParse(t, "a = [1, 2, 3]", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(EIdent("a", p(1, 1))),
				exprs(arrayLit(p(1, 5), p(1, 13),
					intLit(1, p(1, 6)),
					intLit(2, p(1, 9)),
					intLit(3, p(1, 12)))),
				token.Assign,
				p(1, 3)))
	})

	test.ExpectParse(t, "a = [1 + 2, b * 4, [4, c]]", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(EIdent("a", p(1, 1))),
				exprs(arrayLit(p(1, 5), p(1, 26),
					binaryExpr(
						intLit(1, p(1, 6)),
						intLit(2, p(1, 10)),
						token.Add,
						p(1, 8)),
					binaryExpr(
						EIdent("b", p(1, 13)),
						intLit(4, p(1, 17)),
						token.Mul,
						p(1, 15)),
					arrayLit(p(1, 20), p(1, 25),
						intLit(4, p(1, 21)),
						EIdent("c", p(1, 24))))),
				token.Assign,
				p(1, 3)))
	})

	test.ExpectParseString(t, "a = [1\n2\n3]", "a = [1, 2, 3]")
	test.ExpectParseString(t, "a = [1,2\n3]", "a = [1, 2, 3]")
	test.ExpectParseString(t, "a = [\n\n1\n2\n3\n]", "a = [1, 2, 3]")

	test.ExpectParseError(t, "[,]")
	test.ExpectParseError(t, "[1\n,]")
	test.ExpectParseError(t, "[1,\n2\n,]")
	test.ExpectParseError(t, `[1, 2, 3
	,]`)
	test.ExpectParseError(t, `[1, 2, 3, ,]`)
}

func TestParseAssignment(t *testing.T) {
	test.ExpectParse(t, "a = 5", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(EIdent("a", p(1, 1))),
				exprs(intLit(5, p(1, 5))),
				token.Assign,
				p(1, 3)))
	})

	test.ExpectParse(t, "a := 5", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(EIdent("a", p(1, 1))),
				exprs(intLit(5, p(1, 6))),
				token.Define,
				p(1, 3)))
	})

	test.ExpectParse(t, "a, b = 5, 10", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					EIdent("a", p(1, 1)),
					EIdent("b", p(1, 4))),
				exprs(
					intLit(5, p(1, 8)),
					intLit(10, p(1, 11))),
				token.Assign,
				p(1, 6)))
	})

	test.ExpectParse(t, "a, b := 5, 10", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					EIdent("a", p(1, 1)),
					EIdent("b", p(1, 4))),
				exprs(
					intLit(5, p(1, 9)),
					intLit(10, p(1, 12))),
				token.Define,
				p(1, 6)))
	})

	test.ExpectParse(t, "a, b = a + 2, b - 8", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					EIdent("a", p(1, 1)),
					EIdent("b", p(1, 4))),
				exprs(
					binaryExpr(
						EIdent("a", p(1, 8)),
						intLit(2, p(1, 12)),
						token.Add,
						p(1, 10)),
					binaryExpr(
						EIdent("b", p(1, 15)),
						intLit(8, p(1, 19)),
						token.Sub,
						p(1, 17))),
				token.Assign,
				p(1, 6)))
	})

	test.ExpectParse(t, "a = [1, 2, 3]", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(EIdent("a", p(1, 1))),
				exprs(arrayLit(p(1, 5), p(1, 13),
					intLit(1, p(1, 6)),
					intLit(2, p(1, 9)),
					intLit(3, p(1, 12)))),
				token.Assign,
				p(1, 3)))
	})

	test.ExpectParse(t, "a = [1 + 2, b * 4, [4, c]]", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(EIdent("a", p(1, 1))),
				exprs(arrayLit(p(1, 5), p(1, 26),
					binaryExpr(
						intLit(1, p(1, 6)),
						intLit(2, p(1, 10)),
						token.Add,
						p(1, 8)),
					binaryExpr(
						EIdent("b", p(1, 13)),
						intLit(4, p(1, 17)),
						token.Mul,
						p(1, 15)),
					arrayLit(p(1, 20), p(1, 25),
						intLit(4, p(1, 21)),
						EIdent("c", p(1, 24))))),
				token.Assign,
				p(1, 3)))
	})

	test.ExpectParse(t, "a += 5", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(EIdent("a", p(1, 1))),
				exprs(intLit(5, p(1, 6))),
				token.AddAssign,
				p(1, 3)))
	})

	test.ExpectParse(t, "a *= 5 + 10", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(EIdent("a", p(1, 1))),
				exprs(
					binaryExpr(
						intLit(5, p(1, 6)),
						intLit(10, p(1, 10)),
						token.Add,
						p(1, 8))),
				token.MulAssign,
				p(1, 3)))
	})

	test.ExpectParse(t, "a ||= 5", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(EIdent("a", p(1, 1))),
				exprs(intLit(5, p(1, 7))),
				token.LOrAssign,
				p(1, 3)))
	})

	test.ExpectParse(t, "a ||= 5 + 10", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(EIdent("a", p(1, 1))),
				exprs(
					binaryExpr(
						intLit(5, p(1, 7)),
						intLit(10, p(1, 11)),
						token.Add,
						p(1, 9))),
				token.LOrAssign,
				p(1, 3)))
	})

	test.ExpectParse(t, "a ??= 5", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(EIdent("a", p(1, 1))),
				exprs(intLit(5, p(1, 7))),
				token.NullichAssign,
				p(1, 3)))
	})

	test.ExpectParse(t, "a ??= 5 + 10", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(EIdent("a", p(1, 1))),
				exprs(
					binaryExpr(
						intLit(5, p(1, 7)),
						intLit(10, p(1, 11)),
						token.Add,
						p(1, 9))),
				token.NullichAssign,
				p(1, 3)))
	})

	test.ExpectParse(t, "a ++= 5", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(EIdent("a", p(1, 1))),
				exprs(intLit(5, p(1, 7))),
				token.IncAssign,
				p(1, 3)))
	})

	test.ExpectParse(t, "a --= 5", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(EIdent("a", p(1, 1))),
				exprs(intLit(5, p(1, 7))),
				token.DecAssign,
				p(1, 3)))
	})

	test.ExpectParse(t, "a **= 5 + 10", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(EIdent("a", p(1, 1))),
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
	test.ExpectParse(t, "false == nil", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				unaryExpr(
					boolLit(false, p(1, 1)),
					token.Null,
					p(1, 7))))
	})

	test.ExpectParse(t, "false != nil", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				unaryExpr(
					boolLit(false, p(1, 1)),
					token.NotNull,
					p(1, 7))))
	})

	test.ExpectParseString(t, "false == nil", "(false == nil)")
	test.ExpectParseString(t, "false != nil", "(false != nil)")
	test.ExpectParseString(t, "nil == nil", "(nil == nil)")
	test.ExpectParseString(t, "nil != nil", "(nil != nil)")

	test.ExpectParse(t, "a == nil ? b : c", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				condExpr(
					unaryExpr(
						EIdent("a", p(1, 1)),
						token.Null,
						p(1, 3)),
					EIdent("b", p(1, 12)),
					EIdent("c", p(1, 16)),
					p(1, 10),
					p(1, 14))))
	})

	test.ExpectParse(t, "a != nil ? b : c", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				condExpr(
					unaryExpr(
						EIdent("a", p(1, 1)),
						token.NotNull,
						p(1, 3)),
					EIdent("b", p(1, 12)),
					EIdent("c", p(1, 16)),
					p(1, 10),
					p(1, 14))))
	})

	test.ExpectParseString(t, "a == nil ? b : c", "((a == nil) ? b : c)")
	test.ExpectParseString(t, "a != nil ? b : c", "((a != nil) ? b : c)")
}

func TestParseBoolean(t *testing.T) {
	test.ExpectParse(t, "true", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				boolLit(true, p(1, 1))))
	})

	test.ExpectParse(t, "false", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				boolLit(false, p(1, 1))))
	})

	test.ExpectParse(t, "true != false", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				binaryExpr(
					boolLit(true, p(1, 1)),
					boolLit(false, p(1, 9)),
					token.NotEqual,
					p(1, 6))))
	})

	test.ExpectParse(t, "!false", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				unaryExpr(
					boolLit(false, p(1, 2)),
					token.Not,
					p(1, 1))))
	})
}

func TestParseFlag(t *testing.T) {
	test.ExpectParse(t, "yes", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				flagLit(true, p(1, 1))))
	})

	test.ExpectParse(t, "no", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				flagLit(false, p(1, 1))))
	})

	test.ExpectParse(t, "yes != no", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				binaryExpr(
					flagLit(true, p(1, 1)),
					flagLit(false, p(1, 8)),
					token.NotEqual,
					p(1, 5))))
	})

	test.ExpectParse(t, "!no", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				unaryExpr(
					flagLit(false, p(1, 2)),
					token.Not,
					p(1, 1))))
	})
}

func TestParseCallKeywords(t *testing.T) {
	test.ExpectParse(t, token.Callee.String(), func(p pfn) []Stmt {
		return stmts(exprStmt(caleeKw(p(1, 1))))
	})
	test.ExpectParse(t, token.Args.String(), func(p pfn) []Stmt {
		return stmts(exprStmt(argsKw(p(1, 1))))
	})
	test.ExpectParse(t, token.NamedArgs.String(), func(p pfn) []Stmt {
		return stmts(exprStmt(nargsKw(p(1, 1))))
	})
	test.ExpectParseString(t, token.Callee.String(), token.Callee.String())
	test.ExpectParseString(t, token.Args.String(), token.Args.String())
	test.ExpectParseString(t, token.NamedArgs.String(), token.NamedArgs.String())
}

func TestParseCall(t *testing.T) {
	test.New(t, "add(1, 2; x(){y++}, y()=>1, **d)").
		String("add(1, 2; x() { y++ }, y() => 1, **d)").
		Code("add(1, 2; x() {y++}, y() => 1, **d)").
		IndentedCode("add(1, 2; x() {\n\ty++\n}, y() => 1, **d)").
		FormattedCode(`add(
	1,
	2; 
	x() {
		y++
	},
	y() => 1,
	**d
)`)
	test.ExpectParse(t, "add(,)", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				callExpr(
					EIdent("add", p(1, 1)),
					p(1, 4), p(1, 6),
					callExprArgs(nil))))
	})
	test.ExpectParse(t, "add(\n\t,)", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				callExpr(
					EIdent("add", p(1, 1)),
					p(1, 4), p(2, 3),
					callExprArgs(nil))))
	})
	test.ExpectParse(t, "add(1, 2, 3)", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				callExpr(
					EIdent("add", p(1, 1)),
					p(1, 4), p(1, 12),
					callExprArgs(nil,
						intLit(1, p(1, 5)),
						intLit(2, p(1, 8)),
						intLit(3, p(1, 11))))))
	})
	test.ExpectParse(t, "add(1, 2, *v)", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				callExpr(
					EIdent("add", p(1, 1)),
					p(1, 4), p(1, 13),
					callExprArgs(
						argVar(p(1, 11), EIdent("v", p(1, 12))),
						intLit(1, p(1, 5)),
						intLit(2, p(1, 8))))))
	})
	test.ExpectParse(t, "a = add(1, 2, 3)", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					EIdent("a", p(1, 1))),
				exprs(
					callExpr(
						EIdent("add", p(1, 5)),
						p(1, 8), p(1, 16),
						callExprArgs(nil,
							intLit(1, p(1, 9)),
							intLit(2, p(1, 12)),
							intLit(3, p(1, 15))))),
				token.Assign,
				p(1, 3)))
	})
	test.ExpectParse(t, "a, b = add(1, 2, 3)", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					EIdent("a", p(1, 1)),
					EIdent("b", p(1, 4))),
				exprs(
					callExpr(
						EIdent("add", p(1, 8)),
						p(1, 11), p(1, 19),
						callExprArgs(nil, intLit(1, p(1, 12)),
							intLit(2, p(1, 15)),
							intLit(3, p(1, 18))))),
				token.Assign,
				p(1, 6)))
	})
	test.ExpectParse(t, "add(a + 1, 2 * 1, (b + c))", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				callExpr(
					EIdent("add", p(1, 1)),
					p(1, 4), p(1, 26),
					callExprArgs(nil,
						binaryExpr(
							EIdent("a", p(1, 5)),
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
								EIdent("b", p(1, 20)),
								EIdent("c", p(1, 24)),
								token.Add,
								p(1, 22)),
							p(1, 19), p(1, 25))))))
	})

	test.ExpectParseString(t, "a + add(b * c) + d", "((a + add((b * c))) + d)")
	test.ExpectParseString(t, "add(a, b, 1, 2 * 3, 4 + 5, add(6, 7 * 8))",
		"add(a, b, 1, (2 * 3), (4 + 5), add(6, (7 * 8)))")
	test.ExpectParseString(t, "f1(a) + f2(b) * f3(c)", "(f1(a) + (f2(b) * f3(c)))")
	test.ExpectParseString(t, "(f1(a) + f2(b)) * f3(c)",
		"((f1(a) + f2(b)) * f3(c))")
	test.ExpectParseString(t, "f(1,)", "f(1)")
	test.ExpectParseString(t, "f(1,\n)", "f(1)")
	test.ExpectParseString(t, "f(\n1,\n)", "f(1)")
	test.ExpectParseString(t, "f(1,2,)", "f(1, 2)")
	test.ExpectParseString(t, "f(1,2,\n)", "f(1, 2)")
	test.ExpectParseString(t, "f(1,\n2,)", "f(1, 2)")
	test.ExpectParseString(t, "f(1,\n2,\n)", "f(1, 2)")
	test.ExpectParseString(t, "f(1,\n2,)", "f(1, 2)")
	test.ExpectParseString(t, "f(\n1,\n2,)", "f(1, 2)")
	test.ExpectParseString(t, "f(\n1,\n2)", "f(1, 2)")

	test.ExpectParse(t, "func(a, b) { a + b }(1, 2)", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				callExpr(
					funcLit(
						funcType(p(1, 1), nil, p(1, 5), p(1, 10),
							funcArgs(nil,
								EIdent("a", p(1, 6)),
								EIdent("b", p(1, 9))),
						),
						blockStmt(
							p(1, 12), p(1, 20),
							exprStmt(
								binaryExpr(
									EIdent("a", p(1, 14)),
									EIdent("b", p(1, 18)),
									token.Add,
									p(1, 16))))),
					p(1, 21),
					p(1, 26),
					callExprArgs(nil,
						intLit(1, p(1, 22)),
						intLit(2, p(1, 25))))))
	})

	test.ExpectParse(t, `a.b()`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				callExpr(
					selectorExpr(
						EIdent("a", p(1, 1)),
						stringLit("b", p(1, 3))),
					p(1, 4), p(1, 5), NoPos)))
	})

	test.ExpectParse(t, `a.b.c()`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				callExpr(
					selectorExpr(
						selectorExpr(
							EIdent("a", p(1, 1)),
							stringLit("b", p(1, 3))),
						stringLit("c", p(1, 5))),
					p(1, 6), p(1, 7), NoPos)))
	})

	test.ExpectParse(t, `a["b"].c()`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				callExpr(
					selectorExpr(
						indexExpr(
							EIdent("a", p(1, 1)),
							stringLit("b", p(1, 3)),
							p(1, 2), p(1, 6)),
						stringLit("c", p(1, 8))),
					p(1, 9), p(1, 10), NoPos)))
	})

	test.ExpectParseError(t, `add(a*, 1)`)
	test.ExpectParseError(t, `add(a*, b*)`)
	test.ExpectParseError(t, `add(1, a*, b*)`)
	test.ExpectParseError(t, `add(*)`)
	test.ExpectParseError(t, `add(1, *)`)
	test.ExpectParseError(t, `add(1, *, )`)
	test.ExpectParseError(t, `add(a*)`)

	test.New(t, `
f(; x{() => nil; (y) => nil})
f(1; x{() => nil; (y) => nil})
f(1, 2; x{() => nil; (y) => nil})
f(1, *s; x{() => nil; (y) => nil})
f(; z, x{() => nil; (y) => nil})
f(1; z, x{() => nil; (y) => nil}, x)
f(; z=1, x{() => nil; (y) => nil})
f(1; z=1, x{() => nil; (y) => nil}, x=2)
`).
		String("f(; x=func {() => nil; (y) => nil; }); " +
			"f(1; x=func {() => nil; (y) => nil; }); " +
			"f(1, 2; x=func {() => nil; (y) => nil; }); " +
			"f(1, *s; x=func {() => nil; (y) => nil; }); " +
			"f(; z=yes, x=func {() => nil; (y) => nil; }); " +
			"f(1; z=yes, x=func {() => nil; (y) => nil; }, x=yes); " +
			"f(; z=1, x=func {() => nil; (y) => nil; }); " +
			"f(1; z=1, x=func {() => nil; (y) => nil; }, x=2)").
		Code("f(; x {() => nil; (y) => nil}); " +
			"f(1; x {() => nil; (y) => nil}); " +
			"f(1, 2; x {() => nil; (y) => nil}); " +
			"f(1, *s; x {() => nil; (y) => nil}); " +
			"f(; z, x {() => nil; (y) => nil}); " +
			"f(1; z, x {() => nil; (y) => nil}, x); " +
			"f(; z=1, x {() => nil; (y) => nil}); " +
			"f(1; z=1, x {() => nil; (y) => nil}, x=2)").
		FormattedCode(`f(; 
	x {
		() => nil

		(y) => nil
	}
)
f(
	1; 
	x {
		() => nil

		(y) => nil
	}
)
f(
	1,
	2; 
	x {
		() => nil

		(y) => nil
	}
)
f(
	1,
	*s; 
	x {
		() => nil

		(y) => nil
	}
)
f(; 
	z,
	x {
		() => nil

		(y) => nil
	}
)
f(
	1; 
	z,
	x {
		() => nil

		(y) => nil
	},
	x
)
f(; 
	z=1,
	x {
		() => nil

		(y) => nil
	}
)
f(
	1; 
	z=1,
	x {
		() => nil

		(y) => nil
	},
	x=2
)`)
}

func TestParseCallWithNamedArgs(t *testing.T) {
	test.ExpectParse(t, "add(;x=2)", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				callExpr(
					EIdent("add", p(1, 1)),
					p(1, 4), p(1, 9),
					callExprNamedArgs(
						[]*NamedArgExpr{{Ident: EIdent("x", p(1, 6))}},
						[]Expr{intLit(2, p(1, 8))},
					))))
	})
	test.ExpectParse(t, "add(;x=2,y=3)", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				callExpr(
					EIdent("add", p(1, 1)),
					p(1, 4), p(1, 13),
					callExprNamedArgs(
						[]*NamedArgExpr{{Ident: EIdent("x", p(1, 6))}, {Ident: EIdent("y", p(1, 10))}},
						[]Expr{intLit(2, p(1, 8)), intLit(3, p(1, 12))},
					))))
	})
	test.ExpectParse(t, "add(;x=2,**{})", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				callExpr(
					EIdent("add", p(1, 1)),
					p(1, 4), p(1, 14),
					callExprNamedArgs(
						[]*NamedArgExpr{{Ident: EIdent("x", p(1, 6))}, {Var: true, Exp: dictLit(12, 13)}},
						[]Expr{intLit(2, p(1, 8)), nil},
					))))
	})
	test.ExpectParse(t, "add(;\"x\"=2,y=3)", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				callExpr(
					EIdent("add", p(1, 1)),
					p(1, 4), p(1, 15),
					callExprNamedArgs(
						[]*NamedArgExpr{{Lit: stringLit("x", p(1, 6))}, {Ident: EIdent("y", p(1, 12))}},
						[]Expr{intLit(2, p(1, 10)), intLit(3, p(1, 14))},
					))))
	})

	test.ExpectParseString(t, "add(;x() => 3)", "add(; x() => 3)")
	test.ExpectParseString(t, "add(1, 2; x () { y++ })", "add(1, 2; x() { y++ })")
	test.ExpectParseString(t, `attrs(;"name")`, `attrs(; "name"=yes)`)
	test.ExpectParseString(t, "fn(a;b)", "fn(a; b=yes)")
	test.ExpectParseString(t, "fn(;**{y:5})", "fn(; **{y: 5})")
	test.ExpectParseString(t, "fn(1,*[2,3];x=4,**{y:5})", "fn(1, *[2, 3]; x=4, **{y: 5})")
	test.ExpectParseString(t, "fn(1; a=b)()", "fn(1; a=b)()")
}

func TestParseParenMultiValues(t *testing.T) {
	var mp *MultiParenExpr
	test.ExpectParseStringT(t, `(,)`, `(,)`, mp)
	test.ExpectParseStringT(t, `(,1)`, `(, 1)`, mp)
	test.ExpectParseStringT(t, `([a=1];b=2)`, `(, [a=1]; b=2)`, mp)
	test.ExpectParseStringT(t, `(,;a=1)`, `(,; a=1)`, mp)
	test.ExpectParseStringT(t, `(*a)`, `(, *a)`, mp)
	test.ExpectParseStringT(t, `(,;**a)`, `(,; **a)`, mp)
	test.ExpectParseStringT(t, `(1;ok)`, `(, 1; ok)`, mp)
	test.ExpectParseStringT(t, `(a, *b;c=1, **d)`, `(, a, *b; c=1, **d)`, mp)
	test.ExpectParseStringT(t, `(a;c=2, z=x(1))`, `(, a; c=2, z=x(1))`, mp)
	test.ExpectParseStringT(t, `(,x int)`, `(, x int)`, mp)
	test.ExpectParseStringT(t, `(,x int|bool)`, `(, x int|bool)`, mp)
	test.ExpectParseStringT(t, `(1,2,x int)`, `(, 1, 2, x int)`, mp)
	test.ExpectParseStringT(t, `(1,2,x int|bool)`, `(, 1, 2, x int|bool)`, mp)
	test.ExpectParseStringT(t, `(1,2,x int|bool)`, `(, 1, 2, x int|bool)`, mp)
	test.ExpectParseStringT(t, `(1,2,x int|bool;y)`, `(, 1, 2, x int|bool; y)`, mp)
	test.ExpectParseStringT(t, `(1,2,x int|bool;y,z str, a int|bool)`, `(, 1, 2, x int|bool; y, z str, a int|bool)`, mp)
	test.ExpectParseStringT(t, `(1,2,x int|bool;y,z str, a int|bool, b=2, c int|bool=true)`, `(, 1, 2, x int|bool; y, z str, a int|bool, b=2, c int|bool=true)`, mp)
	test.ExpectParseStringT(t, `(a;
c=2, 
  z=x(1))`, `(, a; c=2, z=x(1))`, mp)
}

func TestParseKeyValue(t *testing.T) {
	test.ExpectParseString(t, `[(1+1)=1]`, `[(1 + 1)=1]`)
	test.ExpectParse(t, `[a=1]`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				kv(EIdent("a", p(1, 2)), intLit(1, p(1, 4)))))
	})
	test.ExpectParseString(t, `[a=1]`, `[a=1]`)
}

func TestParseKeyValueArray(t *testing.T) {
	test.New(t, `(;a=1,b=2,
"c"
=
3,4=5,
true=false, 
my_closure() => 1,
my_func() {
	i++
},
myflag
d=#dVal
e=#(e val)
x A
x1 A1=10
y B|C=11
y1 B|C  =  12
z=
	5 + 
		7
)`).
		String(`(;a=1, b=2, "c"=3, 4=5, true=false, my_closure() => 1, my_func() { i++ }, myflag, d=#(dVal), e=#(e val), x A, x1 A1=10, y B|C=11, y1 B|C=12, z=(5 + 7))`).
		FormattedCode(`(;
	a=1,
	b=2,
	"c"=3,
	4=5,
	true=false,
	my_closure() => 1,
	my_func() {
		i++
	},
	myflag,
	d=#dVal,
	e=#(e val),
	x A,
	x1 A1=10,
	y B|C=11,
	y1 B|C=12,
	z=(5 + 7)
)`)

	test.New(t, `(;fn {
	() => 1
	(x) => x
})`).
		String(`(;fn=func {() => 1; (x) => x; })`).
		Code("(;fn {() => 1; (x) => x})").
		IndentedCode("(;\n\tfn {\n\t\t() => 1\n\n\t\t(x) => x\n\t}\n)", CodeWithFlags(CodeWriteContextFlagFormatKeyValueArrayItemInNewLine))

	test.ExpectParseString(t, `(;
x int
z
)`, `(;x int, z)`)
	test.ExpectParseString(t, `(;)`, `(;)`)
	test.ExpectParseString(t, `(;)`, `(;)`)
	test.ExpectParseString(t, `(
;
)`, `(;)`)
	test.ExpectParseString(t, `(;a=1)`, `(;a=1)`)
	test.ExpectParseString(t, `(;flag)`, `(;flag)`)
	test.ExpectParseString(t, `(;
flag
)`, `(;flag)`)

	test.New(t, `(;a=1,b=2,
"c"
=
3
4=5
true=false
my_closure() => 1
my_func() {
	i++
}
myflag)`).
		String(`(;a=1, b=2, "c"=3, 4=5, true=false, my_closure() => 1, my_func() { i++ }, myflag)`).
		IndentedCode(`(;
	a=1,
	b=2,
	"c"=3,
	4=5,
	true=false,
	my_closure() => 1,
	my_func() {
		i++
	},
	myflag
)`, CodeWithFlags(CodeWriteContextFlagFormatKeyValueArrayItemInNewLine))

	test.ExpectParseString(t, `(;a=1,b=2,"c"=3,4=5,true=false
myflag
d=#dVal,e=#(e val)
)`, `(;a=1, b=2, "c"=3, 4=5, true=false, myflag, d=#(dVal), e=#(e val))`)

	kva := &KeyValueArrayLit{}
	test.ExpectParseStringT(t, `(;**a)`, `(;**a)`, kva)
	test.ExpectParseStringT(t, `(;x=1, **a)`, `(;x=1, **a)`, kva)
	test.ExpectParseStringT(t, `(;a=1)`, `(;a=1)`, kva)
	test.ExpectParseStringT(t, `(;**a)`, `(;**a)`, kva)

	test.ExpectParseString(t, `(;a int)`, `(;a int)`)
	test.ExpectParseString(t, `(;a int|bool)`, `(;a int|bool)`)
	test.ExpectParseString(t, `(;a int=1)`, `(;a int=1)`)
	test.ExpectParseString(t, `(;a int|bool=1)`, `(;a int|bool=1)`)
	test.ExpectParseString(t, `(;a int|bool|str=1)`, `(;a int|bool|str=1)`)
}

func TestTemplateString(t *testing.T) {
	test.ExpectParseString(t, `#"A"`, `#"A"`)
	test.ExpectParseString(t, "#`A`", "#`A`")
	test.ExpectParseString(t, "#```A```", "#```A```")
}

func TestParseChar(t *testing.T) {
	test.ExpectParseExpr(t, `'A'`, charLit('A', 1))
	test.ExpectParseExpr(t, `'九'`, charLit('九', 1))

	test.ExpectParseError(t, `''`)
	test.ExpectParseError(t, `'AB'`)
	test.ExpectParseError(t, `'A九'`)

	test.ExpectParseExpr(t, `'A'`, charAsStringLit("A", 1), test.OptParseCharAsString)
	test.ExpectParseExpr(t, `'九'`, charAsStringLit("九", 1), test.OptParseCharAsString)
	test.ExpectParseExpr(t, `'A九'`, charAsStringLit("A九", 1), test.OptParseCharAsString)
	test.ExpectParseExpr(t, "'a\\'b'", charAsStringLit(`a'b`, 1), test.OptParseCharAsString)
}

func TestParseCondExpr(t *testing.T) {
	test.ExpectParse(t, "a ? b : c", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				condExpr(
					EIdent("a", p(1, 1)),
					EIdent("b", p(1, 5)),
					EIdent("c", p(1, 9)),
					p(1, 3),
					p(1, 7))))
	})
	test.ExpectParse(t, `a ?
b :
c`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				condExpr(
					EIdent("a", p(1, 1)),
					EIdent("b", p(1, 5)),
					EIdent("c", p(1, 9)),
					p(1, 3),
					p(1, 7))))
	})

	test.ExpectParseString(t, `a ? b : c`, "(a ? b : c)")
	test.ExpectParseString(t, `a + b ? c - d : e * f`,
		"((a + b) ? (c - d) : (e * f))")
	test.ExpectParseString(t, `a == b ? c + (d / e) : f ? g : h + i`,
		"((a == b) ? (c + (d / e)) : (f ? g : (h + i)))")
	test.ExpectParseString(t, `(a + b) ? (c - d) : (e * f)`,
		"((a + b) ? (c - d) : (e * f))")
	test.ExpectParseString(t, `a + (b ? c : d) - e`, "((a + ((b ? c : d))) - e)")
	test.ExpectParseString(t, `a ? b ? c : d : e`, "(a ? (b ? c : d) : e)")
	test.ExpectParseString(t, `a := b ? c : d`, "a := (b ? c : d)")
	test.ExpectParseString(t, `x := a ? b ? c : d : e`,
		"x := (a ? (b ? c : d) : e)")

	// ? : should be at the end of each line if it's multi-line
	test.ExpectParseError(t, `a 
? b 
: c`)
	test.ExpectParseError(t, `a ? (b : e)`)
	test.ExpectParseError(t, `(a ? b) : e`)
	test.ExpectParseError(t, `(b : e, c:d)`)
	test.ExpectParseError(t, `(b : e)`)
	test.ExpectParseError(t, `b : e`)
}

func TestParseReturn(t *testing.T) {
	test.ExpectParse(t, "return", func(p pfn) []Stmt {
		return stmts(returnStmt(p(1, 1), nil))
	})

	test.ExpectParse(t, "1 || return", func(p pfn) []Stmt {
		return stmts(exprStmt(binaryExpr(intLit(1, p(1, 1)), returnExpr(p(1, 6), nil), token.LOr, p(1, 3))))
	})

	test.ExpectParseString(t, `var x; x || return`,
		"var x; (x || return)")

	test.ExpectParseString(t, `return 1`,
		"return 1")

	test.ExpectParse(t, "return = myvar", func(p pfn) []Stmt {
		return stmts(returnAssignStmt(p(1, 1), EIdent("myvar", p(1, 10))))
	})

	test.ExpectParseError(t, "return = 1")
}

func TestParseForIn(t *testing.T) {
	test.ExpectParse(t, "for x in y {}", func(p pfn) []Stmt {
		return stmts(
			forInStmt(
				EIdent("_", p(1, 5)),
				EIdent("x", p(1, 5)),
				EIdent("y", p(1, 10)),
				blockStmt(p(1, 12), p(1, 13)),
				p(1, 1)))
	})

	test.ExpectParse(t, "for _ in y {}", func(p pfn) []Stmt {
		return stmts(
			forInStmt(
				EIdent("_", p(1, 5)),
				EIdent("_", p(1, 5)),
				EIdent("y", p(1, 10)),
				blockStmt(p(1, 12), p(1, 13)),
				p(1, 1)))
	})

	test.ExpectParse(t, "for x in [1, 2, 3] {}", func(p pfn) []Stmt {
		return stmts(
			forInStmt(
				EIdent("_", p(1, 5)),
				EIdent("x", p(1, 5)),
				arrayLit(
					p(1, 10), p(1, 18),
					intLit(1, p(1, 11)),
					intLit(2, p(1, 14)),
					intLit(3, p(1, 17))),
				blockStmt(p(1, 20), p(1, 21)),
				p(1, 1)))
	})

	test.ExpectParse(t, "for x, y in z {}", func(p pfn) []Stmt {
		return stmts(
			forInStmt(
				EIdent("x", p(1, 5)),
				EIdent("y", p(1, 8)),
				EIdent("z", p(1, 13)),
				blockStmt(p(1, 15), p(1, 16)),
				p(1, 1)))
	})

	test.ExpectParse(t, "for x, y in {k1: 1, k2: 2} {}", func(p pfn) []Stmt {
		return stmts(
			forInStmt(
				EIdent("x", p(1, 5)),
				EIdent("y", p(1, 8)),
				dictLit(
					p(1, 13), p(1, 26),
					dicElementLitE(
						EIdent("k1", p(1, 14)), p(1, 16), intLit(1, p(1, 18))),
					dicElementLitE(
						EIdent("k2", p(1, 21)), p(1, 23), intLit(2, p(1, 25)))),
				blockStmt(p(1, 28), p(1, 29)),
				p(1, 1)))
	})

	test.ExpectParse(t, "for x in y {} else {}", func(p pfn) []Stmt {
		return stmts(
			forInStmt(
				EIdent("_", p(1, 5)),
				EIdent("x", p(1, 5)),
				EIdent("y", p(1, 10)),
				blockStmt(p(1, 12), p(1, 13)),
				p(1, 1),
				blockStmt(p(1, 20), p(1, 21))))
	})

	test.ExpectParse(t, "for x in y do x else 1 end", func(p pfn) []Stmt {
		return stmts(
			forInStmt(
				EIdent("_", p(1, 5)),
				EIdent("x", p(1, 5)),
				EIdent("y", p(1, 10)),
				blockLitStmt(
					lit("do", p(1, 12)), ast.Literal{},
					exprStmt(
						EIdent("x", p(1, 15)),
					),
				),
				p(1, 1),
				blockLitStmt(lit("", p(1, 22)), lit("end", p(1, 24)),
					exprStmt(
						intLit(1, p(1, 22)),
					),
				)))
	})

	test.ExpectParseString(t, "for x in y do x else 1 end", "for _, x in y do x else 1 end")

	test.ExpectParse(t, "for x in y {} else 1 end", func(p pfn) []Stmt {
		return stmts(
			forInStmt(
				EIdent("_", p(1, 5)),
				EIdent("x", p(1, 5)),
				EIdent("y", p(1, 10)),
				blockStmt(p(1, 12), p(1, 13)),
				p(1, 1),
				blockLitStmt(lit("", p(1, 20)), lit("end", p(1, 22)),
					exprStmt(
						intLit(1, p(1, 20)),
					),
				)))
	})

	test.New(t, "for x in y do end").
		String("for _, x in y do end").
		Code("for x in y do end")

	test.New(t, "for x in y do 1 end").
		String("for _, x in y do 1 end").
		Code("for x in y do 1; end")

	test.New(t, "for x in y do 1; end").
		String("for _, x in y do 1 end").
		Code("for x in y do 1; end")

	test.New(t, "for x in y do else end").
		String("for _, x in y do else end").
		Code("for x in y do else end")

	test.New(t, "for x in y do 1 else end").
		String("for _, x in y do 1 else end").
		Code("for x in y do 1; else end")

	test.ExpectParseString(t, "for x in y do else end", "for _, x in y do else end")

	test.New(t, "for x in y do 1 else 2 end").
		String("for _, x in y do 1 else 2 end").
		Code("for x in y do 1; else 2; end")

	test.ExpectParseError(t, `for 1 in a {}`)
	test.ExpectParseError(t, `for "" in a {}`)
	test.ExpectParseError(t, `for k,2 in a {}`)
	test.ExpectParseError(t, `for 1,v in a {}`)
}

func TestParseFor(t *testing.T) {
	test.ExpectParse(t, "for {}", func(p pfn) []Stmt {
		return stmts(
			forStmt(nil, nil, nil, blockStmt(p(1, 5), p(1, 6)), p(1, 1)))
	})

	test.ExpectParse(t, "for a == 5 {}", func(p pfn) []Stmt {
		return stmts(
			forStmt(
				nil,
				binaryExpr(
					EIdent("a", p(1, 5)),
					intLit(5, p(1, 10)),
					token.Equal,
					p(1, 7)),
				nil,
				blockStmt(p(1, 12), p(1, 13)),
				p(1, 1)))
	})

	test.ExpectParse(t, "for a := 0; a == 5;  {}", func(p pfn) []Stmt {
		return stmts(
			forStmt(
				assignStmt(
					exprs(EIdent("a", p(1, 5))),
					exprs(intLit(0, p(1, 10))),
					token.Define, p(1, 7)),
				binaryExpr(
					EIdent("a", p(1, 13)),
					intLit(5, p(1, 18)),
					token.Equal,
					p(1, 15)),
				nil,
				blockStmt(p(1, 22), p(1, 23)),
				p(1, 1)))
	})

	test.ExpectParse(t, "for a := 0; a < 5; a++ {}", func(p pfn) []Stmt {
		return stmts(
			forStmt(
				assignStmt(
					exprs(EIdent("a", p(1, 5))),
					exprs(intLit(0, p(1, 10))),
					token.Define, p(1, 7)),
				binaryExpr(
					EIdent("a", p(1, 13)),
					intLit(5, p(1, 17)),
					token.Less,
					p(1, 15)),
				incDecStmt(
					EIdent("a", p(1, 20)),
					token.Inc, p(1, 21)),
				blockStmt(p(1, 24), p(1, 25)),
				p(1, 1)))
	})

	test.ExpectParse(t, "for ; a < 5; a++ {}", func(p pfn) []Stmt {
		return stmts(
			forStmt(
				nil,
				binaryExpr(
					EIdent("a", p(1, 7)),
					intLit(5, p(1, 11)),
					token.Less,
					p(1, 9)),
				incDecStmt(
					EIdent("a", p(1, 14)),
					token.Inc, p(1, 15)),
				blockStmt(p(1, 18), p(1, 19)),
				p(1, 1)))
	})

	test.ExpectParse(t, "for a := 0; ; a++ {}", func(p pfn) []Stmt {
		return stmts(
			forStmt(
				assignStmt(
					exprs(EIdent("a", p(1, 5))),
					exprs(intLit(0, p(1, 10))),
					token.Define, p(1, 7)),
				nil,
				incDecStmt(
					EIdent("a", p(1, 15)),
					token.Inc, p(1, 16)),
				blockStmt(p(1, 19), p(1, 20)),
				p(1, 1)))
	})

	test.ExpectParse(t, "for a == 5 && b != 4 {}", func(p pfn) []Stmt {
		return stmts(
			forStmt(
				nil,
				binaryExpr(
					binaryExpr(
						EIdent("a", p(1, 5)),
						intLit(5, p(1, 10)),
						token.Equal,
						p(1, 7)),
					binaryExpr(
						EIdent("b", p(1, 15)),
						intLit(4, p(1, 20)),
						token.NotEqual,
						p(1, 17)),
					token.LAnd,
					p(1, 12)),
				nil,
				blockStmt(p(1, 22), p(1, 23)),
				p(1, 1)))
	})

	test.ExpectParse(t, `for { break }`, func(p pfn) []Stmt {
		return stmts(
			forStmt(nil, nil, nil,
				blockStmt(p(1, 5), p(1, 13),
					breakStmt(p(1, 7)),
				),
				p(1, 1)),
		)
	})

	test.ExpectParse(t, `for { continue }`, func(p pfn) []Stmt {
		return stmts(
			forStmt(nil, nil, nil,
				blockStmt(p(1, 5), p(1, 16),
					continueStmt(p(1, 7)),
				),
				p(1, 1)),
		)
	})

	test.ExpectParseString(t, `for do continue end`, "for do continue end")

	// labels are parsed by parser but not supported by compiler yet
	// expectParseError(t, `for { break x }`)
}

func TestParseClosure(t *testing.T) {
	test.New(t, `(a,*b;c=2,**d) => 3`).
		String(`(a, *b; c=2, **d) => 3`).
		Type(&ClosureExpr{})
	test.ExpectParseStringT(t, `(a,*b;c=2) => 3`, `(a, *b; c=2) => 3`, &ClosureExpr{})
	test.ExpectParse(t, "a = (b, c, d) => d", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					EIdent("a", p(1, 1))),
				exprs(
					EClosure(
						funcParams(p(1, 1), p(1, 5), p(1, 13),
							funcArgs(nil,
								EIdent("b", p(1, 6)),
								EIdent("c", p(1, 9)),
								EIdent("d", p(1, 12))),
						),
						p(1, 15),
						token.Lambda,
						EIdent("d", p(1, 18)))),
				token.Assign,
				p(1, 3)))
	})

	test.ExpectParse(t, "a = (b, c, d) => {d}", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					EIdent("a", p(1, 1))),
				exprs(
					EClosure(
						funcParams(p(1, 5), p(1, 5), p(1, 13),
							funcArgs(nil,
								EIdent("b", p(1, 6)),
								EIdent("c", p(1, 9)),
								EIdent("d", p(1, 12))),
						),
						p(1, 15),
						token.Lambda,
						blockExpr(p(1, 18), p(1, 20),
							exprStmt(EIdent("d", p(1, 19)))))),
				token.Assign,
				p(1, 3)))
	})

	test.New(t, "() => nil").
		String("() => nil").
		Code("() => nil").
		IndentedCode("() => nil").
		Stmts(func(p test.Pfn) []Stmt {
			return stmts(exprStmt(EClosure(
				funcParams(p(1, 1), p(1, 2)),
				p(1, 4), token.Lambda,
				LNil(p(1, 7)),
			)))
		})
}

func TestParseFunction(t *testing.T) {
	test.ExpectParseString(t, "func(){}", "func() {}")
	test.ExpectParseString(t, "func(a int){}", "func(a int) {}")
	test.ExpectParseString(t, "func(a int|bool|int){}", "func(a int|bool) {}")
	test.ExpectParseString(t, "func(a \n int|\n\tbool){}", "func(a int|bool) {}")
	test.ExpectParse(t, "func fn (b) { return d }", func(p pfn) []Stmt {
		return stmts(
			funcStmt(
				funcLit(
					funcType(p(1, 5), EIdent("fn", p(1, 6)), p(1, 9), p(1, 11),
						EIdent("fn", p(1, 6)),
						funcArgs(nil,
							EIdent("b", p(1, 10))),
					),
					blockStmt(p(1, 13), p(1, 24),
						returnStmt(p(1, 15), EIdent("d", p(1, 22)))))))
	})
	test.ExpectParseString(t, "func(v int){}", "func(v int) {}")
	test.ExpectParseString(t, "func(v a.b.int){}", "func(v a.b.int) {}")
	test.ExpectParseString(t, "func(v a.(b).int){}", "func(v a.(b).int) {}")
	test.ExpectParseString(t, "func(v a.(b[1])[2].int){}", "func(v a.(b[1])[2].int) {}")
	test.ExpectParseString(t, "func(v a.(b[1])[2].int|x){}", "func(v a.(b[1])[2].int|x) {}")
	test.ExpectParseString(t, "func(v a.(b[1])[2].int|x.y.z){}", "func(v a.(b[1])[2].int|x.y.z) {}")
	test.ExpectParseString(t, "func(v a.(b[1])[2].int|x.y[2][4].z){}", "func(v a.(b[1])[2].int|x.y[2][4].z) {}")

	test.ExpectParse(t, "func(;x){}", func(p pfn) []Stmt {
		return stmts(
			funcStmt(
				EFunc(
					funcType(p(1, 1), nil, p(1, 5), p(1, 8),
						funcNamedArgs(
							nil,
							[]*TypedIdentExpr{
								typedIdent(EIdent("x", p(1, 7))),
							},
							[]Expr{nil}),
					),
					blockStmt(p(1, 9), p(1, 10)))),
		)
	})
	test.ExpectParseString(t, "func(;x){}", "func(; x) {}")
	test.ExpectParse(t, "a = func(b, c, d; e=1, f=2, **g) { return d }", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					EIdent("a", p(1, 1))),
				exprs(
					funcLit(
						funcType(p(1, 5), nil, p(1, 9), p(1, 32),
							funcArgs(nil,
								EIdent("b", p(1, 10)),
								EIdent("c", p(1, 13)),
								EIdent("d", p(1, 16))),
							funcNamedArgs(
								EIdent("g", p(1, 31)),
								[]*TypedIdentExpr{
									typedIdent(EIdent("e", p(1, 19))),
									typedIdent(EIdent("f", p(1, 24))),
								},
								[]Expr{
									intLit(1, p(1, 21)),
									intLit(2, p(1, 26)),
								}),
						),
						blockStmt(p(1, 34), p(1, 45),
							returnStmt(p(1, 36), EIdent("d", p(1, 43)))))),
				token.Assign,
				p(1, 3)))
	})
	test.ExpectParse(t, "a = func(*args) { return args }", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					EIdent("a", p(1, 1))),
				exprs(
					funcLit(
						funcType(p(1, 5), nil, p(1, 9), p(1, 15),
							funcArgs(EIdent("args", p(1, 11)))),
						blockStmt(p(1, 17), p(1, 31),
							returnStmt(p(1, 19),
								EIdent("args", p(1, 26)),
							),
						),
					),
				),
				token.Assign,
				p(1, 3)))
	})

	test.ExpectParse(t, "func(n,a,b;**na) {}", func(p pfn) []Stmt {
		return stmts(
			funcStmt(
				EFunc(
					funcType(p(1, 5), nil, p(1, 5), p(1, 16),
						funcArgs(nil,
							EIdent("n", p(1, 6)),
							EIdent("a", p(1, 8)),
							EIdent("b", p(1, 10))),
						funcNamedArgs(EIdent("na", p(1, 14)), nil, nil),
					),
					blockStmt(p(1, 18), p(1, 19)))),
		)
	})

	test.ExpectParse(t, "func(n,a,b;x=1,**na) {}", func(p pfn) []Stmt {
		return stmts(
			funcStmt(
				EFunc(
					funcType(p(1, 5), nil, p(1, 5), p(1, 20),
						funcArgs(nil,
							EIdent("n", p(1, 6)),
							EIdent("a", p(1, 8)),
							EIdent("b", p(1, 10))),
						funcNamedArgs(
							EIdent("na", p(1, 18)),
							[]*TypedIdentExpr{typedIdent(EIdent("x", p(1, 12)))},
							[]Expr{intLit(1, p(1, 14))}),
					),
					blockStmt(p(1, 22), p(1, 23)))),
		)
	})

	test.ExpectParseString(t, "func(){}", "func() {}")
	test.ExpectParseString(t, "func(,){}", "func() {}")
	test.ExpectParseString(t, "func(\n\t,){}", "func() {}")
	test.ExpectParseString(t, "func(\n){}", "func() {}")
	test.ExpectParseString(t, "func(a,){}", "func(a) {}")
	test.ExpectParseString(t, "func(,a){}", "func(a) {}")
	test.ExpectParseString(t, "func(\n\t,a){}", "func(a) {}")
	test.ExpectParseString(t, "func(\na,\n){}", "func(a) {}")
	test.ExpectParseString(t, "func(a,\n){}", "func(a) {}")
	test.ExpectParseString(t, "func(\na,\n){}", "func(a) {}")
	test.ExpectParseString(t, "func(a,b,\n){}", "func(a, b) {}")
	test.ExpectParseString(t, "func(a,\nb,\n){}", "func(a, b) {}")
	test.ExpectParseString(t, "func(a,\nb){}", "func(a, b) {}")
	test.ExpectParseString(t, "func(a,\nb,){}", "func(a, b) {}")
	test.ExpectParseString(t, "func(a,*b){}", "func(a, *b) {}")
	test.ExpectParseString(t, "func(a,*b,){}", "func(a, *b) {}")
	test.ExpectParseString(t, "func(a,*b,\n){}", "func(a, *b) {}")
	test.ExpectParseString(t, "func(a,b,*c,\n){}", "func(a, b, *c) {}")
	test.ExpectParseString(t, "func(a,b,\n*c,\n){}", "func(a, b, *c) {}")
	test.ExpectParseString(t, "func(\na,\nb,\n*c,\n){}", "func(a, b, *c) {}")

	test.ExpectParseString(t, "func(a;kw=2,){}", "func(a; kw=2) {}")
	test.ExpectParseString(t, "func(a,*b;c=1,**d\n){}", "func(a, *b; c=1, **d) {}")
	test.ExpectParseString(t, "func(\na,\n*b\n\n;\nc=\n\t1,\n\n**d\n \t\n){}", "func(a, *b; c=1, **d) {}")
	test.ExpectParseString(t, "func(a;kw=2,){}", "func(a; kw=2) {}")
	test.ExpectParseString(t, "func(a,*b;c=1,**d\n){}", "func(a, *b; c=1, **d) {}")
	test.ExpectParseString(t, "func(\na,\n*b\n\n;\nc=\n\t1,\n\n**d\n \t\n){}", "func(a, *b; c=1, **d) {}")
	test.ExpectParseString(t, "func(a\n,){}", "func(a) {}")
	test.ExpectParseString(t, "func(a\n\n,){}", "func(a) {}")
	test.ExpectParseString(t, "func(\n*a\n\n,){}", "func(*a) {}")
	test.ExpectParseString(t, "func(a\n,*b){}", "func(a, *b) {}")
	test.ExpectParseString(t, "func(a\n,*b){}", "func(a, *b) {}")
	test.ExpectParseString(t, "func(\na\n,*b){}", "func(a, *b) {}")
	test.ExpectParseString(t, "func(*a;\n**b){}", "func(*a; **b) {}")
	test.ExpectParseString(t, `func(;x int=1, y str="abc", **kw) {}`, `func(; x int=1, y str="abc", **kw) {}`)

	test.ExpectParseError(t, "func(*a,b){}")
	test.ExpectParseError(t, "func(a,*b;c=1,**d,**e){}")

	test.ExpectParse(t, "func fn(n) => 1", func(p pfn) []Stmt {
		return stmts(
			funcStmt(
				EFuncBodyE(
					funcType(p(1, 5), EIdent("fn", p(1, 6)), p(1, 8), p(1, 10),
						funcArgs(nil,
							EIdent("n", p(1, 9)),
						),
					),
					p(1, 12),
					intLit(1, p(1, 15)),
				),
			),
		)
	})

	test.ExpectParse(t, "func fn(n) => 1", func(p pfn) []Stmt {
		return stmts(
			funcStmt(
				funcClosure(
					funcType(p(1, 5), EIdent("fn", p(1, 6)), p(1, 8), p(1, 10),
						funcArgs(nil,
							EIdent("n", p(1, 9)),
						),
					),
					p(1, 12),
					intLit(1, p(1, 15)),
				),
			),
		)
	})
	test.ExpectParseString(t, "func fn(n) =>  1", `func fn(n) => 1`)
	test.ExpectParseString(t, "func(n) =>  1", `func(n) => 1`)

	test.New(t, "x() => nil").
		String("x() => nil").
		Code("x() => nil").
		IndentedCode("x() => nil").
		Stmts(func(p test.Pfn) []Stmt {
			return stmts(funcStmt(EFuncBodyE(
				funcType(p(1, 1), EIdent("x", p(1, 1)), p(1, 2), p(1, 3)),
				p(1, 5), LNil(p(1, 8)),
			)))
		})

	test.New(t, "(x(){nil})").
		String("(x() { nil })").
		Code("(x() {nil})").
		IndentedCode("(x() {\n\tnil\n})").
		Stmts(func(p test.Pfn) []Stmt {
			return stmts(exprStmt(parenExpr(EFunc(
				funcType(p(1, 2), EIdent("x", p(1, 2)), p(1, 3), p(1, 4)),
				blockStmt(p(1, 5), p(1, 9), exprStmt(LNil(p(1, 6)))),
			), p(1, 1), p(1, 10))))
		})

	test.New(t, "x(){nil}").
		String("x() { nil }").
		Code("x() {nil}").
		IndentedCode("x() {\n\tnil\n}").
		Stmts(func(p test.Pfn) []Stmt {
			return stmts(funcStmt(EFunc(
				funcType(p(1, 1), EIdent("x", p(1, 1)), p(1, 2), p(1, 3)),
				blockStmt(p(1, 4), p(1, 8), exprStmt(&NilLit{TokenPos: p(1, 5)})),
			)))
		})

	test.New(t, "x(){nil}()").
		String("(x() { nil })()").
		Code("(x() {nil})()").
		IndentedCode("(x() {\n\tnil\n})()")

	test.New(t, "x(a int, *args; b, c=1, **kwargs){nil}").
		String("x(a int, *args; b, c=1, **kwargs) { nil }").
		Code("x(a int, *args; b, c=1, **kwargs) {nil}").
		IndentedCode("x(a int, *args; b, c=1, **kwargs) {\n\tnil\n}").
		Stmts(func(p test.Pfn) []Stmt {
			return stmts(funcStmt(EFunc(
				funcType(p(1, 1), EIdent("x", p(1, 1)), p(1, 2), p(1, 33),
					funcArgs(
						EIdent("args", p(1, 11)),
						ETypedIdent(EIdent("a", p(1, 3)), EType(EIdent("int", p(1, 5)))),
					),
					funcNamedArgs(EIdent("kwargs", p(1, 27)), []*TypedIdentExpr{
						ETypedIdent(EIdent("b", p(1, 17))),
						ETypedIdent(EIdent("c", p(1, 20))),
					}, Exprs{nil, Int(1, p(1, 22))}),
				),
				blockStmt(p(1, 34), p(1, 38), exprStmt(LNil(p(1, 35)))),
			)))
		})
	test.ExpectParse(t, "[func () {}]", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				arrayLit(p(1, 1), p(1, 12), EFunc(funcType(p(1, 2), nil, p(1, 7), p(1, 8)), blockStmt(p(1, 10), p(1, 11))))))
	})
	test.ExpectParse(t, "return [func () {}]", func(p pfn) []Stmt {
		return stmts(returnStmt(p(1, 1), arrayLit(p(1, 8), p(1, 19), EFunc(funcType(p(1, 9), nil, p(1, 14), p(1, 15)), blockStmt(p(1, 17), p(1, 18))))))
	})
	test.New(t, "func () {}()").String("(func() {})()").Code("(func() {})()")
	test.New(t, "func x() {return 1}()").
		String("(func x() { return 1 })()").
		IndentedCode("(func x() {\n\treturn 1\n})()")

	test.ExpectParse(t, "func fn (b) { return d }", func(p pfn) []Stmt {
		return stmts(
			funcStmt(
				EFunc(
					funcType(p(1, 4), EIdent("fn", p(1, 6)), p(1, 9), p(1, 11),
						funcArgs(nil,
							EIdent("b", p(1, 10))),
					),
					blockStmt(p(1, 13), p(1, 24),
						returnStmt(p(1, 15), EIdent("d", p(1, 22)))))))
	})

	test.ExpectParse(t, "func class.fn (b) { return d }", func(p pfn) []Stmt {
		return stmts(
			funcStmt(
				EFunc(
					funcType(p(1, 4), ESelector(EIdent("class", p(1, 6)), String("fn", p(1, 12))), p(1, 15), p(1, 17),
						funcArgs(nil,
							EIdent("b", p(1, 16))),
					),
					blockStmt(p(1, 19), p(1, 30),
						returnStmt(p(1, 21), EIdent("d", p(1, 28)))))))
	})

	test.ExpectParse(t, `func class["fn"] (b) { return d }`, func(p pfn) []Stmt {
		return stmts(
			funcStmt(
				EFunc(
					funcType(p(1, 4),
						EIndex(
							EIdent("class", p(1, 6)),
							String("fn", p(1, 12)),
							p(1, 11),
							p(1, 16),
						), p(1, 18), p(1, 20),
						funcArgs(nil,
							EIdent("b", p(1, 19))),
					),
					blockStmt(p(1, 22), p(1, 33),
						returnStmt(p(1, 24), EIdent("d", p(1, 31)))))))
	})

	test.New(t, `func class.fn["x"][y()].z (b) { return d }`).
		String(`func class.fn["x"][y()].z(b) { return d }`).
		Code(`func class.fn["x"][y()].z(b) {return d}`).
		IndentedCode(`func class.fn["x"][y()].z(b) {
	return d
}`).
		Stmts(func(p pfn) []Stmt {
			return stmts(
				funcStmt(
					EFunc(
						funcType(p(1, 4),
							ESelector(
								EIndex(
									EIndex(
										ESelector(
											EIdent("class", p(1, 6)),
											String("fn", p(1, 12)),
										),
										String("x", p(1, 15)),
										p(1, 14),
										p(1, 18),
									),
									ECall(EIdent("y", p(1, 20)), p(1, 21), p(1, 22)),
									p(1, 19),
									p(1, 23),
								),
								String("z", p(1, 25)),
							),
							p(1, 27), p(1, 29),
							funcArgs(nil,
								EIdent("b", p(1, 28))),
						),
						blockStmt(p(1, 31), p(1, 42),
							returnStmt(p(1, 33), EIdent("d", p(1, 40)))))))
		})
}

func TestParseMethod(t *testing.T) {
	test.ExpectParse(t, "met fn (b) { return d }", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				EMethod(funcLit(
					funcType(p(1, 4), EIdent("fn", p(1, 5)), p(1, 8), p(1, 10),
						funcArgs(nil,
							EIdent("b", p(1, 9))),
					),
					blockStmt(p(1, 12), p(1, 23),
						returnStmt(p(1, 14), EIdent("d", p(1, 21))))))))
	})

	test.ExpectParse(t, "met class.fn (b) { return d }", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				EMethod(funcLit(
					funcType(p(1, 4), ESelector(EIdent("class", p(1, 5)), String("fn", p(1, 11))), p(1, 14), p(1, 16),
						funcArgs(nil,
							EIdent("b", p(1, 15))),
					),
					blockStmt(p(1, 18), p(1, 29),
						returnStmt(p(1, 20), EIdent("d", p(1, 27))))))))
	})

	test.ExpectParse(t, `met class["fn"] (b) { return d }`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				EMethod(
					EFunc(
						funcType(p(1, 4),
							EIndex(
								EIdent("class", p(1, 5)),
								String("fn", p(1, 11)),
								p(1, 10),
								p(1, 15),
							), p(1, 17), p(1, 19),
							funcArgs(nil,
								EIdent("b", p(1, 18))),
						),
						blockStmt(p(1, 21), p(1, 32),
							returnStmt(p(1, 23), EIdent("d", p(1, 30))))))))
	})

	test.New(t, `met class.fn["x"][y()].z (b) { return d }`).
		String(`met class.fn["x"][y()].z(b) { return d }`).
		Code(`met class.fn["x"][y()].z(b) {return d}`).
		IndentedCode(`met class.fn["x"][y()].z(b) {
	return d
}`).
		Stmts(func(p pfn) []Stmt {
			return stmts(
				exprStmt(
					EMethod(funcLit(
						funcType(p(1, 4),
							ESelector(
								EIndex(
									EIndex(
										ESelector(
											EIdent("class", p(1, 5)),
											String("fn", p(1, 11)),
										),
										String("x", p(1, 14)),
										p(1, 13),
										p(1, 17),
									),
									ECall(EIdent("y", p(1, 19)), p(1, 20), p(1, 21)),
									p(1, 18),
									p(1, 22),
								),
								String("z", p(1, 24)),
							),
							p(1, 26), p(1, 28),
							funcArgs(nil,
								EIdent("b", p(1, 27))),
						),
						blockStmt(p(1, 30), p(1, 41),
							returnStmt(p(1, 32), EIdent("d", p(1, 39))))))))
		})
}

func TestParsePtr(t *testing.T) {
	test.New(t, "&x").String("&(x)").Code("&x")
}

func TestParseSpecialKeywords(t *testing.T) {
	test.New(t, "@main;x.@main").Stmts(func(p pfn) []Stmt {
		return stmts(
			exprStmt(&IsMainLit{p(1, 1)}),
			exprStmt(ESelector(EIdent("x", p(1, 7)), String("@main", p(1, 9)))),
		)
	})
}

func TestParseFunctionWithMethods(t *testing.T) {
	test.New(t, "func{}").String("func {}").Code("func {}")
	test.New(t, "func x{}").String("func x {}").Code("func x {}")
	test.New(t, "func x{}").String("func x {}").Code("func x {}")
	test.New(t, "func fn {(n) =>1; (x) { x++ }; (y) { if y { return 0}; return 1 }}").
		String("func fn {(n) => 1; (x) { x++ }; (y) { if y { return 0 }; return 1 }; }").
		IndentedCode(`func fn {
	(n) => 1

	(x) {
		x++
	}

	(y) {
		if y {
			return 0
		}
		return 1
	}
}`)
	test.New(t, `func fn {
	(n) => 1;
	(x) {
		x++
	}


	(y) {
		if y {
			return 0
		}
		return 1
	};
}`).
		String("func fn {(n) => 1; (x) { x++ }; (y) { if y { return 0 }; return 1 }; }").
		IndentedCode(`func fn {
	(n) => 1

	(x) {
		x++
	}

	(y) {
		if y {
			return 0
		}
		return 1
	}
}`)
	test.ExpectParseString(t, "func fn {(n) =>1}", "func fn {(n) => 1; }")
	test.ExpectParseString(t, "func fn {(n) { i++; return 2}}", "func fn {(n) { i++; return 2 }; }")
}

func TestParseVariadicFunctionWithArgs(t *testing.T) {
	test.ExpectParse(t, "a = func(x, y, *z) { return z }", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					EIdent("a", p(1, 1))),
				exprs(
					funcLit(
						funcType(p(1, 5), nil, p(1, 9), p(1, 18),
							funcArgs(EIdent("z", p(1, 17)),
								EIdent("x", p(1, 10)),
								EIdent("y", p(1, 13)))),
						blockStmt(p(1, 20), p(1, 31),
							returnStmt(p(1, 22),
								EIdent("z", p(1, 29)),
							),
						),
					),
				),
				token.Assign,
				p(1, 3)))
	})

	test.ExpectParseError(t, "a = func(x, y, *z, invalid) { return z }")
	test.ExpectParseError(t, "a = func(*args, invalid) { return args }")
	test.ExpectParseError(t, "a = func(args*, invalid) { return args }")
}

func TestParseIf(t *testing.T) {
	test.ExpectParse(t, "if a == nil {}", func(p pfn) []Stmt {
		return stmts(
			ifStmt(
				nil,
				unaryExpr(EIdent("a", p(1, 4)),
					token.Null,
					p(1, 6)),
				blockStmt(
					p(1, 13), p(1, 14)),
				nil,
				p(1, 1)))
	})

	test.ExpectParse(t, "if a != nil {}", func(p pfn) []Stmt {
		return stmts(
			ifStmt(
				nil,
				unaryExpr(EIdent("a", p(1, 4)),
					token.NotNull,
					p(1, 6)),
				blockStmt(
					p(1, 13), p(1, 14)),
				nil,
				p(1, 1)))
	})

	test.ExpectParse(t, "if a == 5 {}", func(p pfn) []Stmt {
		return stmts(
			ifStmt(
				nil,
				binaryExpr(
					EIdent("a", p(1, 4)),
					intLit(5, p(1, 9)),
					token.Equal,
					p(1, 6)),
				blockStmt(
					p(1, 11), p(1, 12)),
				nil,
				p(1, 1)))
	})

	test.ExpectParse(t, "if a == 5 && b != 3 {}", func(p pfn) []Stmt {
		return stmts(
			ifStmt(
				nil,
				binaryExpr(
					binaryExpr(
						EIdent("a", p(1, 4)),
						intLit(5, p(1, 9)),
						token.Equal,
						p(1, 6)),
					binaryExpr(
						EIdent("b", p(1, 14)),
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

	test.ExpectParse(t, "if a == 5 { a = 3; a = 1 }", func(p pfn) []Stmt {
		return stmts(
			ifStmt(
				nil,
				binaryExpr(
					EIdent("a", p(1, 4)),
					intLit(5, p(1, 9)),
					token.Equal,
					p(1, 6)),
				blockStmt(
					p(1, 11), p(1, 26),
					assignStmt(
						exprs(EIdent("a", p(1, 13))),
						exprs(intLit(3, p(1, 17))),
						token.Assign,
						p(1, 15)),
					assignStmt(
						exprs(EIdent("a", p(1, 20))),
						exprs(intLit(1, p(1, 24))),
						token.Assign,
						p(1, 22))),
				nil,
				p(1, 1)))
	})

	test.ExpectParse(t, "if a == 5 { a = 3; a = 1 } else { a = 2; a = 4 }",
		func(p pfn) []Stmt {
			return stmts(
				ifStmt(
					nil,
					binaryExpr(
						EIdent("a", p(1, 4)),
						intLit(5, p(1, 9)),
						token.Equal,
						p(1, 6)),
					blockStmt(
						p(1, 11), p(1, 26),
						assignStmt(
							exprs(EIdent("a", p(1, 13))),
							exprs(intLit(3, p(1, 17))),
							token.Assign,
							p(1, 15)),
						assignStmt(
							exprs(EIdent("a", p(1, 20))),
							exprs(intLit(1, p(1, 24))),
							token.Assign,
							p(1, 22))),
					blockStmt(
						p(1, 33), p(1, 48),
						assignStmt(
							exprs(EIdent("a", p(1, 35))),
							exprs(intLit(2, p(1, 39))),
							token.Assign,
							p(1, 37)),
						assignStmt(
							exprs(EIdent("a", p(1, 42))),
							exprs(intLit(4, p(1, 46))),
							token.Assign,
							p(1, 44))),
					p(1, 1)))
		})

	test.ExpectParse(t, `
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
					EIdent("a", p(2, 4)),
					intLit(5, p(2, 9)),
					token.Equal,
					p(2, 6)),
				blockStmt(
					p(2, 11), p(5, 1),
					assignStmt(
						exprs(EIdent("b", p(3, 2))),
						exprs(intLit(3, p(3, 6))),
						token.Assign,
						p(3, 4)),
					assignStmt(
						exprs(EIdent("c", p(4, 2))),
						exprs(intLit(1, p(4, 6))),
						token.Assign,
						p(4, 4))),
				ifStmt(
					nil,
					binaryExpr(
						EIdent("d", p(5, 11)),
						intLit(3, p(5, 16)),
						token.Equal,
						p(5, 13)),
					blockStmt(
						p(5, 18), p(8, 1),
						assignStmt(
							exprs(EIdent("e", p(6, 2))),
							exprs(intLit(8, p(6, 6))),
							token.Assign,
							p(6, 4)),
						assignStmt(
							exprs(EIdent("f", p(7, 2))),
							exprs(intLit(3, p(7, 6))),
							token.Assign,
							p(7, 4))),
					blockStmt(
						p(8, 8), p(11, 1),
						assignStmt(
							exprs(EIdent("g", p(9, 2))),
							exprs(intLit(2, p(9, 6))),
							token.Assign,
							p(9, 4)),
						assignStmt(
							exprs(EIdent("h", p(10, 2))),
							exprs(intLit(4, p(10, 6))),
							token.Assign,
							p(10, 4))),
					p(5, 8)),
				p(2, 1)))
	})

	test.ExpectParse(t, "if a := 3; a < b {}", func(p pfn) []Stmt {
		return stmts(
			ifStmt(
				assignStmt(
					exprs(EIdent("a", p(1, 4))),
					exprs(intLit(3, p(1, 9))),
					token.Define, p(1, 6)),
				binaryExpr(
					EIdent("a", p(1, 12)),
					EIdent("b", p(1, 16)),
					token.Less, p(1, 14)),
				blockStmt(
					p(1, 18), p(1, 19)),
				nil,
				p(1, 1)))
	})

	test.ExpectParse(t, "if a++; a < b {}", func(p pfn) []Stmt {
		return stmts(
			ifStmt(
				incDecStmt(EIdent("a", p(1, 4)), token.Inc, p(1, 5)),
				binaryExpr(
					EIdent("a", p(1, 9)),
					EIdent("b", p(1, 13)),
					token.Less, p(1, 11)),
				blockStmt(
					p(1, 15), p(1, 16)),
				nil,
				p(1, 1)))
	})

	test.ExpectParseString(t, "if a then end", "if a then end")
	test.ExpectParseString(t, "if a then b end", "if a then b end")
	test.ExpectParseString(t, "if true; a then b end", "if true; a then b end")
	test.ExpectParseString(t, "if a then b else c end", "if a then b else c end")
	test.ExpectParseString(t, "if a then b; else c end", "if a then b else c end")
	test.ExpectParseString(t, "if a then b else if 1 then 2 else c end", "if a then b else if 1 then 2 else c end")
	test.ExpectParseString(t, "if a then b; else if 1 then 2; else c end", "if a then b else if 1 then 2 else c end")

	test.ExpectParseError(t, `if {}`)
	test.ExpectParseError(t, `if a == b { } else a != b { }`)
	test.ExpectParseError(t, `if a == b { } else if { }`)
	test.ExpectParseError(t, `else { }`)
	test.ExpectParseError(t, `if ; {}`)
	test.ExpectParseError(t, `if a := 3; {}`)
	test.ExpectParseError(t, `if ; a < 3 {}`)
}

func TestParseImport(t *testing.T) {
	test.ExpectParse(t, `a := import("mod1")`, func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(EIdent("a", p(1, 1))),
				exprs(importExpr(p(1, 6), "mod1", p(1, 12), p(1, 19), p(1, 13))),
				token.Define, p(1, 3)))
	})

	test.ExpectParse(t, `a := import("mod1", 1; x=2)`, func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(EIdent("a", p(1, 1))),
				exprs(WithCallArgs(importExpr(p(1, 6), "mod1", p(1, 12), p(1, 27), p(1, 13)), func(args *CallArgs) {
					args.Args.AppendValues(intLit(1, p(1, 21)))
					args.NamedArgs.Append(ENamedArg().Ident(EIdent("x", p(1, 24))).Build(), intLit(2, p(1, 26)))
				})),
				token.Define, p(1, 3)))
	})

	test.ExpectParse(t, `import("mod1").var1`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				selectorExpr(
					importExpr(p(1, 1), "mod1", p(1, 7), p(1, 14), p(1, 8)),
					stringLit("var1", p(1, 16)))))
	})

	test.ExpectParse(t, `import("mod1").func1()`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				callExpr(
					selectorExpr(
						importExpr(p(1, 1), "mod1", p(1, 7), p(1, 14), p(1, 8)),
						stringLit("func1", p(1, 16))),
					p(1, 21), p(1, 22), NoPos)))
	})

	test.ExpectParse(t, `for x, y in import("mod1").v {}`, func(p pfn) []Stmt {
		return stmts(
			forInStmt(
				EIdent("x", p(1, 5)),
				EIdent("y", p(1, 8)),
				selectorExpr(importExpr(p(1, 13), "mod1", p(1, 19), p(1, 26), p(1, 20)),
					stringLit("v", p(1, 28))),
				blockStmt(p(1, 30), p(1, 31)),
				p(1, 1)))
	})

	test.ExpectParseError(t, `import(1)`)
	test.ExpectParseError(t, `import('a')`)
}

func TestParseEmbed(t *testing.T) {
	test.ExpectParse(t, `a := embed("file")`, func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(EIdent("a", p(1, 1))),
				exprs(embedExpr("file", p(1, 6))),
				token.Define, p(1, 3)))
	})

	test.ExpectParse(t, `embed("file").var1`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				selectorExpr(
					embedExpr("file", p(1, 1)),
					stringLit("var1", p(1, 15)))))
	})

	test.ExpectParse(t, `for x, y in embed("file") {}`, func(p pfn) []Stmt {
		return stmts(
			forInStmt(
				EIdent("x", p(1, 5)),
				EIdent("y", p(1, 8)),
				embedExpr("file", p(1, 13)),
				blockStmt(p(1, 27), p(1, 28)),
				p(1, 1)))
	})

	test.ExpectParseError(t, `embed(1)`)
	test.ExpectParseError(t, `embed('a')`)
}

func TestParseIndex(t *testing.T) {
	test.ExpectParse(t, "[1, 2, 3][1]", func(p pfn) []Stmt {
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

	test.ExpectParse(t, "[1, 2, 3][5 - a]", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				indexExpr(
					arrayLit(p(1, 1), p(1, 9),
						intLit(1, p(1, 2)),
						intLit(2, p(1, 5)),
						intLit(3, p(1, 8))),
					binaryExpr(
						intLit(5, p(1, 11)),
						EIdent("a", p(1, 15)),
						token.Sub,
						p(1, 13)),
					p(1, 10), p(1, 16))))
	})

	test.ExpectParse(t, "[1, 2, 3][5 : a]", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				sliceExpr(
					arrayLit(p(1, 1), p(1, 9),
						intLit(1, p(1, 2)),
						intLit(2, p(1, 5)),
						intLit(3, p(1, 8))),
					intLit(5, p(1, 11)),
					EIdent("a", p(1, 15)),
					p(1, 10), p(1, 16))))
	})

	test.ExpectParse(t, "[1, 2, 3][a + 3 : b - 8]", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				sliceExpr(
					arrayLit(p(1, 1), p(1, 9),
						intLit(1, p(1, 2)),
						intLit(2, p(1, 5)),
						intLit(3, p(1, 8))),
					binaryExpr(
						EIdent("a", p(1, 11)),
						intLit(3, p(1, 15)),
						token.Add,
						p(1, 13)),
					binaryExpr(
						EIdent("b", p(1, 19)),
						intLit(8, p(1, 23)),
						token.Sub,
						p(1, 21)),
					p(1, 10), p(1, 24))))
	})

	test.ExpectParse(t, `({a: 1, b: 2})["b"]`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				indexExpr(
					parenExpr(
						dictLit(p(1, 2), p(1, 13),
							dicElementLitE(
								EIdent("a", p(1, 3)), p(1, 4), intLit(1, p(1, 6))),
							dicElementLitE(
								EIdent("b", p(1, 9)), p(1, 10), intLit(2, p(1, 12)))),
						p(1, 1), p(1, 14),
					),
					stringLit("b", p(1, 16)),
					p(1, 15), p(1, 19))))
	})

	test.ExpectParse(t, `({a: 1, b: 2})[a + b]`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				indexExpr(
					parenExpr(
						dictLit(p(1, 2), p(1, 13),
							dicElementLitE(
								EIdent("a", p(1, 3)), p(1, 4), intLit(1, p(1, 6))),
							dicElementLitE(
								EIdent("b", p(1, 9)), p(1, 10), intLit(2, p(1, 12)))),
						p(1, 1), p(1, 14),
					),
					binaryExpr(
						EIdent("a", p(1, 16)),
						EIdent("b", p(1, 20)),
						token.Add,
						p(1, 18)),
					p(1, 15), p(1, 21))))
	})
}

func TestParseLogical(t *testing.T) {
	test.ExpectParse(t, "2 ** 3", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				binaryExpr(
					intLit(2, p(1, 1)),
					intLit(3, p(1, 6)),
					token.Pow,
					p(1, 3))))
	})

	test.ExpectParse(t, "a && 5 || true", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				binaryExpr(
					binaryExpr(
						EIdent("a", p(1, 1)),
						intLit(5, p(1, 6)),
						token.LAnd,
						p(1, 3)),
					boolLit(true, p(1, 11)),
					token.LOr,
					p(1, 8))))
	})

	test.ExpectParse(t, "a || 5 && true", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				binaryExpr(
					EIdent("a", p(1, 1)),
					binaryExpr(
						intLit(5, p(1, 6)),
						boolLit(true, p(1, 11)),
						token.LAnd,
						p(1, 8)),
					token.LOr,
					p(1, 3))))
	})

	test.ExpectParse(t, "a && (5 || true)", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				binaryExpr(
					EIdent("a", p(1, 1)),
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
	test.ExpectParse(t, "{}", func(p pfn) []Stmt {
		return stmts(blockStmt(p(1, 1), p(1, 2)))
	})

	test.ExpectParse(t, "x := 1; {x := 2}", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				Exprs{EIdent("x", p(1, 1))},
				Exprs{intLit(1, p(1, 6))},
				token.Define,
				p(1, 3),
			),
			blockStmt(
				p(1, 9),
				p(1, 16),
				assignStmt(
					Exprs{EIdent("x", p(1, 10))},
					Exprs{intLit(2, p(1, 15))},
					token.Define,
					p(1, 12),
				),
			),
		)
	})
}

func TestParseDict(t *testing.T) {
	test.New(t, "({})").
		String("({})").
		Code("({})").
		FormattedCode("({})").
		Stmts(func(p pfn) []Stmt {
			return stmts(exprStmt(parenExpr(dictLit(p(1, 2), p(1, 3)), p(1, 1), p(1, 4))))
		})

	test.ExpectParse(t, "({ \"key1\": 1 })", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				parenExpr(
					dictLit(p(1, 2), p(1, 14),
						dicElementLit(
							"key1", p(1, 4), p(1, 10), intLit(1, p(1, 12)))),
					p(1, 1), p(1, 15))))
	})

	test.ExpectParse(t, "({ key1: 1, key2: \"2\", key3: true })", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				parenExpr(
					dictLit(p(1, 2), p(1, 35),
						dicElementLitE(EIdent("key1", p(1, 4)), p(1, 8), intLit(1, p(1, 10))),
						dicElementLitE(EIdent("key2", p(1, 13)), p(1, 17), stringLit("2", p(1, 19))),
						dicElementLitE(EIdent("key3", p(1, 24)), p(1, 28), boolLit(true, p(1, 30)))),
					p(1, 1), p(1, 36))))
	})

	test.ExpectParse(t, "a = { key1: 1 }",
		func(p pfn) []Stmt {
			return stmts(assignStmt(
				exprs(EIdent("a", p(1, 1))),
				exprs(dictLit(p(1, 5), p(1, 15),
					dicElementLitE(EIdent("key1", p(1, 7)), p(1, 11), intLit(1, p(1, 13))))),
				token.Assign,
				p(1, 3)))
		})

	test.New(t, "a = { key1: 1, key2: \"2\", key3: { k1: `bar`, k2: 4 } }").
		String("a = {key1: 1, key2: \"2\", key3: {k1: `bar`, k2: 4}}").
		FormattedCode("a = {\n\tkey1: 1,\n\tkey2: \"2\",\n\tkey3: {\n\t\tk1: `bar`,\n\t\tk2: 4\n\t}\n}")

	test.New(t, `({ "key1": 1, #key2:2, #(key 3): #3, #(key 4): #(value	4), true: 5, false:6, yes:7, no:8`+
		"\n1:9, u2:10, 3d:11, 4.56:12, (x+1):2})").
		String(`({key1: 1, #(key2): 2, #(key 3): #(3), #(key 4): #(value	4), true: 5, false: 6, yes: 7, no: 8, 1: 9, u2: 10, 3d: 11, 4.56: 12, (x + 1): 2})`).
		FormattedCode(`({
	"key1": 1,
	#key2: 2,
	#(key 3): #3,
	#(key 4): #(value	4),
	"true": 5,
	"false": 6,
	"yes": 7,
	"no": 8,
	1: 9,
	u2: 10,
	3d: 11,
	4.56: 12,
	(x + 1): 2
})`)

	test.ExpectParseError(t, "{,}")
	test.ExpectParseError(t, "{\n,}")
	test.ExpectParseError(t, "{key: 1\n,}")
	test.ExpectParseError(t, `
{
	key1: 1,
	key2: "2",
	key3: true
,}`)

	test.ExpectParseError(t, `{
key1: 1,
key2: 2
}`)
	test.ExpectParseError(t, `{1: 1}`)

	test.ExpectParse(t, `({
	x(): 10,
})`, func(p pfn) []Stmt {
		return stmts(exprStmt(parenExpr(dictLit(p(1, 2), p(3, 1),
			dicElementLitE(EIdent("x", p(2, 2)), 0,
				EDictElementClosure(EClosure(funcParams(p(2, 3), p(2, 4)), p(2, 5), token.Colon, intLit(10, p(2, 7))))),
		), p(1, 1), p(3, 2))))
	})

	test.New(t, `({
	x(): 10,
	y() {
		return 11
	},
})`).
		String("({x() : 10, y() { return 11 }})").
		FormattedCode(`({
	x() : 10,
	y() {
		return 11
	}
})`).
		Stmts(func(p pfn) []Stmt {
			return stmts(exprStmt(parenExpr(dictLit(p(1, 2), p(6, 1),
				dicElementLitE(EIdent("x", p(2, 2)), 0, EDictElementClosure(EClosure(funcParams(p(2, 3), p(2, 4)), p(2, 5), token.Colon, intLit(10, p(2, 7))))),
				dicElementLitE(EIdent("y", p(3, 2)), 0, EDictElementFunc(funcLit(funcType(0, nil, p(3, 3), p(3, 4)), blockStmt(p(3, 6), p(5, 2), returnStmt(p(4, 3), intLit(11, p(4, 10))))))),
			), p(1, 1), p(6, 2))))
		})

	test.ExpectParse(t, "({x{y{b:1}}})", func(p pfn) []Stmt {
		return stmts(exprStmt(parenExpr(dictLit(
			p(1, 2), p(1, 12),
			dicElementLitE(EIdent("x", p(1, 3)), 0, dictLit(
				p(1, 4), p(1, 11),
				dicElementLitE(EIdent("y", p(1, 5)), 0, dictLit(
					p(1, 6), p(1, 10),
					dicElementLitE(EIdent("b", p(1, 7)), p(1, 8), intLit(1, p(1, 9))),
				)),
			)),
		), p(1, 1), p(1, 13))))
	})

	test.ExpectParseString(t, `({x():10})`, `({x() : 10})`)
	test.ExpectParseString(t, `({x():10,y() {return 11}})`, `({x() : 10, y() { return 11 }})`)
	test.ExpectParseString(t, `({x:() => 10})`, `({x() : 10})`)
	test.ExpectParseString(t, `({x: func() => 10})`, `({x() : 10})`)
	test.ExpectParseString(t, `({x: func() { return 10 }})`, `({x() { return 10 }})`)
}

func TestParsePrecedence(t *testing.T) {
	test.ExpectParseString(t, `a + b + c`, `((a + b) + c)`)
	test.ExpectParseString(t, `a + b * c`, `(a + (b * c))`)
	test.ExpectParseString(t, `2 * 1 + 3 / 4`, `((2 * 1) + (3 / 4))`)
	test.ExpectParseString(t, `a .| b`, `(a .| b)`)
	test.ExpectParseString(t, `a .| b .| c`, `((a .| b) .| c)`)
	test.ExpectParseString(t, `a .| b + c`, `((a .| b) + c)`)
	test.ExpectParseString(t, `a .| b * c`, `((a .| b) * c)`)
	test.ExpectParseString(t, `a ~ b`, `(a ~ b)`)
	test.ExpectParseString(t, `a ~ b ~ c`, `((a ~ b) ~ c)`)
	test.ExpectParseString(t, `a ~ b * c`, `((a ~ b) * c)`)
	test.ExpectParseString(t, `a ~ b ~ c .| d`, `(((a ~ b) ~ c) .| d)`)
	test.ExpectParseString(t, `a ~ b / c`, `((a ~ b) / c)`)
	test.ExpectParseString(t, `a ** b * c; d * e ** f`, `((a ** b) * c); (d * (e ** f))`)
}

func TestParseNullishSelector(t *testing.T) {
	test.ExpectParseString(t, `a?.(k)`, `a?.(k)`)
	test.ExpectParseString(t, `a?.(k+x)`, `a?.(k + x)`)
	test.ExpectParse(t, "a?.b.c?.d", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				nullishSelector(
					selectorExpr(
						nullishSelector(
							EIdent("a", p(1, 1)),
							stringLit("b", p(1, 4))),
						stringLit("c", p(1, 6))),
					stringLit("d", p(1, 9)))))
	})
	test.ExpectParse(t, "a?.b.c?.d.e", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				selectorExpr(
					nullishSelector(
						selectorExpr(
							nullishSelector(
								EIdent("a", p(1, 1)),
								stringLit("b", p(1, 4))),
							stringLit("c", p(1, 6))),
						stringLit("d", p(1, 9))),
					stringLit("e", p(1, 11)))))
	})
	test.ExpectParse(t, "a?.b.c?.d.e?.f.g", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				selectorExpr(
					nullishSelector(
						selectorExpr(
							nullishSelector(
								selectorExpr(
									nullishSelector(
										EIdent("a", p(1, 1)),
										stringLit("b", p(1, 4))),
									stringLit("c", p(1, 6))),
								stringLit("d", p(1, 9))),
							stringLit("e", p(1, 11))),
						stringLit("f", p(1, 14))),
					stringLit("g", p(1, 16)))))
	})
	test.ExpectParse(t, "a?.b?.c", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				nullishSelector(
					nullishSelector(
						EIdent("a", p(1, 1)),
						stringLit("b", p(1, 4))),
					stringLit("c", p(1, 7)))))
	})
	test.ExpectParse(t, "a?.b", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				nullishSelector(
					EIdent("a", p(1, 1)),
					stringLit("b", p(1, 4)))))
	})
	test.ExpectParseString(t, "a?.b", "a?.b")
	test.ExpectParseString(t, `a?.b["c"+x]?.d`, `a?.b[("c" + x)]?.d`)
	test.ExpectParseString(t, "a?.b.c", "a?.b.c")
	test.ExpectParseString(t, "a?.b.c?.d.e?.f.g", "a?.b.c?.d.e?.f.g")
	test.ExpectParseString(t, `a["b"+"c"]?.d`, `a[("b" + "c")]?.d`)
	test.ExpectParseString(t, `a.b["b"+"c"]?.d`, `a.b[("b" + "c")]?.d`)
	test.ExpectParseString(t, `a?.("b"+"c")?.d`, `a?.("b" + "c")?.d`)
	test.ExpectParseString(t, `d.("a").e`, `d.("a").e`)
	test.ExpectParseString(t, `d.("a"+"b").e`, `d.("a" + "b").e`)
	test.ExpectParseString(t, `d.("a").e ?? 1`, `(d.("a").e ?? 1)`)
	test.ExpectParseString(t, `d.("a"+"b").e ?? 1`, `(d.("a" + "b").e ?? 1)`)
	test.ExpectParseString(t, `a?.("" || "b")?.d.e?.(b ?? "f")`, `a?.("" || "b")?.d.e?.(b ?? "f")`)
	test.ExpectParseString(t, `a?.(k)?.c`, `a?.(k)?.c`)
}

func TestParseSelector(t *testing.T) {
	test.ExpectParse(t, "a.b", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				selectorExpr(
					EIdent("a", p(1, 1)),
					stringLit("b", p(1, 3)))))
	})

	test.ExpectParse(t, "a.b.c", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				selectorExpr(
					selectorExpr(
						EIdent("a", p(1, 1)),
						stringLit("b", p(1, 3))),
					stringLit("c", p(1, 5)))))
	})

	test.ExpectParse(t, "a.(b).c", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				selectorExpr(
					selectorExpr(
						EIdent("a", p(1, 1)),
						parenExpr(EIdent("b", p(1, 4)), p(1, 3), p(1, 5))),
					stringLit("c", p(1, 7)))))
	})

	test.ExpectParse(t, "({k1:1}.k1)", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				parenExpr(
					selectorExpr(
						dictLit(
							p(1, 2), p(1, 7),
							dicElementLitE(
								EIdent("k1", p(1, 3)), p(1, 5), intLit(1, p(1, 6)))),
						stringLit("k1", p(1, 9))),
					p(1, 1), p(1, 11))))

	})

	test.ExpectParse(t, "({k1:1}).k1", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				selectorExpr(
					parenExpr(
						dictLit(
							p(1, 2), p(1, 7),
							dicElementLitE(
								EIdent("k1", p(1, 3)), p(1, 5), intLit(1, p(1, 6)))),
						p(1, 1), p(1, 8)),
					stringLit("k1", p(1, 10)))))

	})

	test.ExpectParse(t, "({k1:{v1:1}}.k1.v1)", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				parenExpr(
					selectorExpr(
						selectorExpr(
							dictLit(
								p(1, 2), p(1, 12),
								dicElementLitE(EIdent("k1", p(1, 3)), p(1, 5),
									dictLit(p(1, 6), p(1, 11),
										dicElementLitE(
											EIdent("v1", p(1, 7)),
											p(1, 9), intLit(1, p(1, 10)))))),
							stringLit("k1", p(1, 14))),
						stringLit("v1", p(1, 17))),
					p(1, 1), p(1, 19))))
	})

	test.ExpectParse(t, "({k1:{v1:1}}).k1.v1", func(p pfn) []Stmt {
		return stmts(
			exprStmt(
				selectorExpr(
					selectorExpr(
						parenExpr(
							dictLit(
								p(1, 2), p(1, 12),
								dicElementLitE(EIdent("k1", p(1, 3)), p(1, 5),
									dictLit(p(1, 6), p(1, 11),
										dicElementLitE(
											EIdent("v1", p(1, 7)),
											p(1, 9), intLit(1, p(1, 10)))))),
							p(1, 1), p(1, 13)),
						stringLit("k1", p(1, 15))),
					stringLit("v1", p(1, 18)))))
	})

	test.ExpectParse(t, "a.b = 4", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					selectorExpr(
						EIdent("a", p(1, 1)),
						stringLit("b", p(1, 3)))),
				exprs(intLit(4, p(1, 7))),
				token.Assign, p(1, 5)))
	})

	test.ExpectParse(t, "a.b.c = 4", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					selectorExpr(
						selectorExpr(
							EIdent("a", p(1, 1)),
							stringLit("b", p(1, 3))),
						stringLit("c", p(1, 5)))),
				exprs(intLit(4, p(1, 9))),
				token.Assign, p(1, 7)))
	})

	test.ExpectParse(t, "a.b.c = 4 + 5", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					selectorExpr(
						selectorExpr(
							EIdent("a", p(1, 1)),
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

	test.ExpectParse(t, "a[0].c = 4", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					selectorExpr(
						indexExpr(
							EIdent("a", p(1, 1)),
							intLit(0, p(1, 3)),
							p(1, 2), p(1, 4)),
						stringLit("c", p(1, 6)))),
				exprs(intLit(4, p(1, 10))),
				token.Assign, p(1, 8)))
	})

	test.ExpectParse(t, "a.b[0].c = 4", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					selectorExpr(
						indexExpr(
							selectorExpr(
								EIdent("a", p(1, 1)),
								stringLit("b", p(1, 3))),
							intLit(0, p(1, 5)),
							p(1, 4), p(1, 6)),
						stringLit("c", p(1, 8)))),
				exprs(intLit(4, p(1, 12))),
				token.Assign, p(1, 10)))
	})

	test.ExpectParse(t, "a.b[0][2].c = 4", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					selectorExpr(
						indexExpr(
							indexExpr(
								selectorExpr(
									EIdent("a", p(1, 1)),
									stringLit("b", p(1, 3))),
								intLit(0, p(1, 5)),
								p(1, 4), p(1, 6)),
							intLit(2, p(1, 8)),
							p(1, 7), p(1, 9)),
						stringLit("c", p(1, 11)))),
				exprs(intLit(4, p(1, 15))),
				token.Assign, p(1, 13)))
	})

	test.ExpectParse(t, `a.b["key1"][2].c = 4`, func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					selectorExpr(
						indexExpr(
							indexExpr(
								selectorExpr(
									EIdent("a", p(1, 1)),
									stringLit("b", p(1, 3))),
								stringLit("key1", p(1, 5)),
								p(1, 4), p(1, 11)),
							intLit(2, p(1, 13)),
							p(1, 12), p(1, 14)),
						stringLit("c", p(1, 16)))),
				exprs(intLit(4, p(1, 20))),
				token.Assign, p(1, 18)))
	})

	test.ExpectParse(t, "a[0].b[2].c = 4", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(
					selectorExpr(
						indexExpr(
							selectorExpr(
								indexExpr(
									EIdent("a", p(1, 1)),
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
	test.ExpectParse(t, "1", func(p pfn) []Stmt {
		return stmts(
			exprStmt(intLit(1, p(1, 1))))
	})

	test.ExpectParse(t, "1;", func(p pfn) []Stmt {
		return stmts(
			exprStmt(intLit(1, p(1, 1))))
	})

	test.ExpectParse(t, "1;;", func(p pfn) []Stmt {
		return stmts(
			exprStmt(intLit(1, p(1, 1))))
	})

	test.ExpectParse(t, `1
`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(intLit(1, p(1, 1))))
	})

	test.ExpectParse(t, `1
;`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(intLit(1, p(1, 1))))
	})

	test.ExpectParse(t, `1;
;`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(intLit(1, p(1, 1))))
	})
}

func TestParseString(t *testing.T) {
	test.ExpectParse(t, "\"foo\nbar\"", func(p pfn) []Stmt {
		return stmts(exprStmt(stringLit("foo\nbar", p(1, 1))))
	})
	test.ExpectParse(t, "\"foo\nbar\"", func(p pfn) []Stmt {
		return stmts(exprStmt(stringLit("foo\nbar", p(1, 1))))
	})
	test.ExpectParse(t, "\"foo\n"+"\n"+"bar\"", func(p pfn) []Stmt {
		return stmts(exprStmt(stringLit("foo\n\nbar", p(1, 1))))
	})
	test.ExpectParse(t, `"foo\n`+"\n"+`bar"`, func(p pfn) []Stmt {
		return stmts(exprStmt(stringLit("foo\\n\nbar", p(1, 1))))
	})
	test.ExpectParse(t, "`abc`", func(p pfn) []Stmt {
		return stmts(exprStmt(rawStringLit(`abc`, p(1, 1))))
	})
	test.ExpectParse(t, "```\nabc\n```", func(p pfn) []Stmt {
		return stmts(exprStmt(rawHeredocLit("```", `abc`, p(1, 1))))
	})
	test.ExpectParse(t, "a = \"foo\nbar\"", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(EIdent("a", p(1, 1))),
				exprs(stringLit("foo\nbar", p(1, 5))),
				token.Assign,
				p(1, 3)))
	})
	test.ExpectParse(t, `a = "foo\nbar"`, func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(EIdent("a", p(1, 1))),
				exprs(stringLit(`foo\nbar`, p(1, 5))),
				token.Assign,
				p(1, 3)))
	})
	test.ExpectParse(t, "a = `raw string`", func(p pfn) []Stmt {
		return stmts(
			assignStmt(
				exprs(EIdent("a", p(1, 1))),
				exprs(rawStringLit("`raw string`", p(1, 5))),
				token.Assign,
				p(1, 3)))
	})
}

func TestParseSymbol(t *testing.T) {
	test.ExpectParse(t, "#abc", func(p pfn) []Stmt {
		return stmts(exprStmt(LSymbol(p(1, 1), "abc", false)))
	})
	test.ExpectParse(t, "#(a\n\\)\tbc)", func(p pfn) []Stmt {
		return stmts(exprStmt(LSymbol(p(1, 1), "a\n)\tbc", true)))
	})
}

func TestParseConfig(t *testing.T) {
	test.ExpectParse(t, `# gad: mixed
	a`, func(p pfn) []Stmt {
		return stmts(
			config(p(1, 1), KVp(EIdent("mixed", p(1, 8)))),
			mixedTextStmt(p(2, 1), "\ta"),
		)
	})
	test.ExpectParse(t, `# gad: mixed, mixed_start = "[[[", mixed_end = "]]]"
y
[[[b]]]`, func(p pfn) []Stmt {
		return stmts(
			config(p(1, 1),
				KVp(EIdent("mixed", p(1, 8))),
				KVp(EIdent("mixed_start", p(1, 15)), stringLit("[[[", p(1, 29))),
				KVp(EIdent("mixed_end", p(1, 36)), stringLit("]]]", p(1, 48))),
			),
			mixedTextStmt(p(2, 1), "y\n"),
			codeBegin(lit("[[[", p(3, 1)), false),
			exprStmt(EIdent("b", p(3, 4))),
			codeEnd(lit("]]]", p(3, 5)), false),
		)
	})

	test.ExpectParseString(t, "# gad: mixed, mixed_start=\"[[[\", mixed_end=\"]]]\"\ny\n[[[b]]]",
		`# gad: mixed, mixed_start="[[[", mixed_end="]]]"`+"\ny\n[[[; b; ]]]")
	test.ExpectParseString(t, "# gad: mixed, mixed_start=\"[[[\", mixed_end=\"]]]\"\ny\n[[[b; true]]]",
		"# gad: mixed, mixed_start=\"[[[\", mixed_end=\"]]]\"\ny\n[[[; b; true; ]]]")
	test.ExpectParse(t, `# gad: mixed`, func(p pfn) []Stmt {
		return stmts(
			config(p(1, 1), KVp(EIdent("mixed", p(1, 8)))))
	})
}

func TestParseTryThrow(t *testing.T) {
	test.ExpectParse(t, `try {} catch e {} finally {}`, func(p pfn) []Stmt {
		return stmts(
			tryStmt(p(1, 1),
				blockStmt(p(1, 5), p(1, 6)),
				catchStmt(p(1, 8), EIdent("e", p(1, 14)),
					blockStmt(p(1, 16), p(1, 17))),
				finallyStmt(p(1, 19),
					blockStmt(p(1, 27), p(1, 28))),
			),
		)
	})
	test.ExpectParse(t, `try {} finally {}`, func(p pfn) []Stmt {
		return stmts(
			tryStmt(p(1, 1),
				blockStmt(p(1, 5), p(1, 6)),
				nil,
				finallyStmt(p(1, 8),
					blockStmt(p(1, 16), p(1, 17))),
			),
		)
	})
	test.ExpectParse(t, `try {
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
	test.ExpectParse(t, `try {} catch {}`, func(p pfn) []Stmt {
		return stmts(
			tryStmt(p(1, 1),
				blockStmt(p(1, 5), p(1, 6)),
				catchStmt(p(1, 8), nil,
					blockStmt(p(1, 14), p(1, 15))),
				nil,
			),
		)
	})
	test.ExpectParse(t, `try {
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
	test.ExpectParse(t, `throw "error"`, func(p pfn) []Stmt {
		return stmts(
			throwStmt(p(1, 1), stringLit("error", p(1, 7))),
		)
	})
	test.ExpectParse(t, `throw 1`, func(p pfn) []Stmt {
		return stmts(
			throwStmt(p(1, 1), intLit(1, p(1, 7))),
		)
	})

	test.ExpectParseError(t, `try catch {}`)
	test.ExpectParseError(t, `try finally {}`)
	test.ExpectParseError(t, `try {} catch;`)
	test.ExpectParseError(t, `try {} catch`)
	test.ExpectParseError(t, `try {} finally`)
	test.ExpectParseError(t, `try {} finally;`)
	test.ExpectParseError(t, `try {}
	catch {}`)
	test.ExpectParseError(t, `try {}
	finally {}`)
	test.ExpectParseError(t, `try {
	} catch {}
	finally {}`)
	test.ExpectParseError(t, `throw;`)
	test.ExpectParseError(t, `throw`)
}

func TestParseRBraceEOF(t *testing.T) {
	test.ExpectParseError(t, `if true {}}`)
	test.ExpectParseError(t, `if true {}}else{}`)
	test.ExpectParseError(t, `a:=1; if true {}}else{}`)
	test.ExpectParseError(t, `if true {}} else{} return`)
	test.ExpectParseError(t, `if true {} else if true {}{`)
	test.ExpectParseError(t, `
if true {

}
} else{

}

return`)
}

func TestParseLinesSep(t *testing.T) {
	test.ExpectParseString(t, "\r\r1+\r\r2+\r\r\r3\r\r\n  \t", `((1 + 2) + 3)`)
	test.ExpectParseString(t, "1+\n2+\n3", `((1 + 2) + 3)`)
	test.ExpectParseString(t, "1+\r\n2+\n3", `((1 + 2) + 3)`)
	test.ExpectParseString(t, "1+\r2+\n3", `((1 + 2) + 3)`)
	test.ExpectParseString(t, "1+\r2+\r3", `((1 + 2) + 3)`)
	test.ExpectParseString(t, "\r\r1+\r2+\r3", `((1 + 2) + 3)`)
	test.ExpectParseString(t, "\r\r1+\r\r2+\r\r\r3", `((1 + 2) + 3)`)
}

func TestComputedExpr(t *testing.T) {
	test.New(t, "(=1)").
		String("(= 1)").
		Code("(= 1)").
		Stmts(func(p test.Pfn) []Stmt {
			return stmts(exprStmt(EComputed(p(1, 1), p(1, 4), exprStmt(Int(1, p(1, 3))))))
		})

	test.New(t, "(= 1|2 )").
		String("(= (1 | 2))").
		Code("(= (1 | 2))")

	test.New(t, "(= 1; x)").
		String("(= 1; x)").
		Code("(= 1; x)").
		IndentedCode(`(=
	1
	x
)`)

	test.New(t, "a:=(=1)").
		String("a := (= 1)").
		Code("a := (= 1)")

	test.New(t, "a:=(= {1;x} )").
		String("a := (= 1; x)").
		Code("a := (= 1; x)").
		IndentedCode(`a := (=
	1
	x
)`)

	test.New(t, "a:=(= 1; {func x() {i++}; z()}; x )").
		String("a := (= 1; { func x() { i++ }; z() }; x)").
		Code("a := (= 1; {func x() {i++}; z()}; x)").
		IndentedCode(`a := (=
	1
	{
		func x() {
			i++
		}
		z()
	}
	x
)`)

	test.New(t, "func a() {a:=(= (;x=(=y();z)))}").
		String("func a() { a := (= (;x=(= y(); z))) }").
		Code("func a() {a := (= (;x=(= y(); z)))}").
		IndentedCode(`func a() {
	a := (= (;x=(=
		y()
		z
	)))
}`)
}

type pfn = test.Pfn               // position conversion function
type expectedFn = test.ExpectedFn // callback function to return expected results

func stmts(s ...Stmt) []Stmt {
	return s
}

func exprStmt(x Expr) *ExprStmt {
	return &ExprStmt{Expr: x}
}

func funcStmt(x *FuncExpr) *FuncStmt {
	return &FuncStmt{Func: x}
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
		Ident: ident,
		Var:   variadic,
	}
}

func nparamSpec(ident *TypedIdentExpr, value Expr) Spec {
	return &NamedParamSpec{
		Ident: ident,
		Value: value,
	}
}

func nparamSpecVar(ident *TypedIdentExpr) Spec {
	return &NamedParamSpec{
		Ident: ident,
		Var:   true,
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

func returnAssignStmt(pos Pos, result Expr) *ReturnStmt {
	return &ReturnStmt{Return: Return{Result: result, ReturnPos: pos, Assign: true}}
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

func funcParams(lparen, rparen Pos, v ...any) *FuncParams {
	p := &FuncParams{LParen: lparen, RParen: rparen}
	for _, v := range v {
		switch t := v.(type) {
		case ArgsList:
			p.Args = t
		case NamedArgsList:
			p.NamedArgs = t
		}
	}
	return p
}

func funcType(pos source.Pos, ident Expr, lparen, rparen Pos, v ...any) *FuncType {
	f := &FuncType{
		NameExpr: ident,
		Params:   FuncParams{LParen: lparen, RParen: rparen},
		FuncPos:  pos,
	}
	for _, v := range v {
		switch t := v.(type) {
		case ArgsList:
			f.Params.Args = t
		case NamedArgsList:
			f.Params.NamedArgs = t
		}
	}
	return f
}

func funcArgs(vari any, names ...Expr) ArgsList {
	l := ArgsList{}
	if vari != nil {
		switch t := vari.(type) {
		case *IdentExpr:
			l.Var = &TypedIdentExpr{Ident: t}
		case *TypedIdentExpr:
			l.Var = t
		default:
			panic(fmt.Errorf("unknown variable type: %T", vari))
		}
	}
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

func funcNamedArgs(vari *IdentExpr, names []*TypedIdentExpr, values []Expr) NamedArgsList {
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

func typedIdent(ident *IdentExpr, typ ...*TypeExpr) *TypedIdentExpr {
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

var importExpr = EImport

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

func dicElementLit(
	key string,
	keyPos Pos,
	colonPos Pos,
	value Expr,
) *DictElementLit {
	return &DictElementLit{
		Key: String(key, keyPos), ColonPos: colonPos, Value: value,
	}
}

func dicElementLitE(
	key Expr,
	colonPos Pos,
	value Expr,
) *DictElementLit {
	return &DictElementLit{
		Key: key, ColonPos: colonPos, Value: value,
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

func funcClosure(funcType *FuncType, lambdaPos source.Pos, body Expr) *FuncExpr {
	return &FuncExpr{
		Type:      funcType,
		LambdaPos: lambdaPos,
		BodyExpr:  body,
	}
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
		case CallExprPositionalArgs:
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

func callExprArgs(
	argVar *ArgVarLit,
	args ...Expr,
) (ce CallExprPositionalArgs) {
	return CallExprPositionalArgs{Var: argVar, Values: args}
}

func callExprNamedArgs(
	names []*NamedArgExpr, values []Expr,
) (ce CallExprNamedArgs) {
	return CallExprNamedArgs{Names: names, Values: values}
}

func indexExpr(
	x, index Expr,
	lbrack, rbrack Pos,
) *IndexExpr {
	return &IndexExpr{
		X: x, Index: index, LBrack: lbrack, RBrack: rbrack,
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
	return &SelectorExpr{X: x, Sel: sel}
}
