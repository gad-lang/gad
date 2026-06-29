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
			SCodeBegin(Lit("‹‹‹", p(1, 1)), false),
			SAssign(
				exprs(EIdent("x", p(3, 1))),
				exprs(EFunc(funcType(p(3, 6), nil, p(3, 10), p(3, 11)),
					SBlock(p(3, 13), p(3, 40), SThrow(p(3, 15), ECall(EIdent("error", p(3, 21)), p(3, 26), p(3, 37), NewCallExprArgs(nil, stringLit("bad code", p(3, 27)))))))),
				token.Define, p(3, 3),
			),
			SCodeEnd(Lit("›››", p(4, 1)), false),
			SMixedText(p(4, 10), "<p>"),
			SMixedValue(Lit("‹‹‹", p(4, 13)), Lit("›››", p(7, 1)), ECall(EIdent("x", p(6, 1)), p(6, 2), p(6, 3))),
			SMixedText(p(7, 10), "</p>"),
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
			SConfig(p(1, 1), KVp(EIdent("mixed", p(1, 8)))),
			SMixedText(p(1, 14), "\t"),
			SCodeBegin(Lit("{%", p(2, 2)), false),
			SExpr(Int(1, p(3, 2))),
			SCodeEnd(Lit("%}", p(4, 1)), false),
			SMixedText(p(4, 3), " \n"),
		)
	})

	defaultExpectParse(`# gad: mixed
	{%   1   %} 
`, func(p pfn) []Stmt {
		return stmts(
			SConfig(p(1, 1), KVp(EIdent("mixed", p(1, 8)))),
			SMixedText(p(1, 14), "\t"),
			SCodeBegin(Lit("{%", p(2, 2)), false),
			SExpr(Int(1, p(2, 7))),
			SCodeEnd(Lit("%}", p(2, 11)), false),
			SMixedText(p(1, 26), " \n"),
		)
	})

	defaultExpectParse(`# gad: mixed
	{%   1   -%} 
`, func(p pfn) []Stmt {
		return stmts(
			SConfig(p(1, 1), KVp(EIdent("mixed", p(1, 8)))),
			SMixedText(p(1, 14), "\t"),
			SCodeBegin(Lit("{%", p(2, 2)), false),
			SExpr(Int(1, p(2, 7))),
			SCodeEnd(Lit("%}", p(2, 12)), true),
			SMixedText(p(1, 27), " \n", RemoveLeftSpaces),
		)
	})
	defaultExpectParse(`# gad: mixed
	{%   1   -%}
a
{%- =   2   -%}`, func(p pfn) []Stmt {
		return stmts(
			SConfig(p(1, 1), KVp(EIdent("mixed", p(1, 8)))),
			SMixedText(p(1, 14), "\t"),
			SCodeBegin(Lit("{%", p(2, 2)), false),
			SExpr(Int(1, p(2, 7))),
			SCodeEnd(Lit("%}", p(2, 12)), true),
			SMixedText(p(1, 27), "\na\n", RemoveLeftSpaces|RemoveRightSpaces),
			SMixedValue(Lit("{%", p(4, 1)), Lit("%}", p(4, 14)), Int(2, p(4, 9))),
		)
	})

	defaultExpectParse(`# gad: mixed
	{%   1   -%}
a
{%- =   2   -%}

{%   3   %}
`, func(p pfn) []Stmt {
		return stmts(
			SConfig(p(1, 1), KVp(EIdent("mixed", p(1, 8)))),
			SMixedText(p(1, 14), "\t"),
			SCodeBegin(Lit("{%", p(2, 2)), false),
			SExpr(Int(1, p(2, 7))),
			SCodeEnd(Lit("%}", p(2, 12)), true),
			SMixedText(p(1, 27), "\na\n", RemoveLeftSpaces|RemoveRightSpaces),
			SMixedValue(Lit("{%", p(4, 1)), Lit("%}", p(4, 14)), Int(2, p(4, 9))),
			SMixedText(p(4, 16), "\n\n", RemoveLeftSpaces),
			SCodeBegin(Lit("{%", p(6, 1)), false),
			SExpr(Int(3, p(6, 6))),
			SCodeEnd(Lit("%}", p(6, 10)), false),
			SMixedText(p(6, 12), "\n"),
		)
	})

	defaultExpectParse(`# gad: mixed
	{%   1   -%}
a
{%- =   2   -%}

{%   3   -%}
`, func(p pfn) []Stmt {
		return stmts(
			SConfig(p(1, 1), KVp(EIdent("mixed", p(1, 8)))),
			SMixedText(p(1, 14), "\t"),
			SCodeBegin(Lit("{%", p(2, 2)), false),
			SExpr(Int(1, p(2, 7))),
			SCodeEnd(Lit("%}", p(2, 12)), true),
			SMixedText(p(1, 27), "\na\n", RemoveLeftSpaces|RemoveRightSpaces),
			SMixedValue(Lit("{%", p(4, 1)), Lit("%}", p(4, 14)), Int(2, p(4, 9))),
			SMixedText(p(4, 16), "\n\n", RemoveLeftSpaces),
			SCodeBegin(Lit("{%", p(6, 1)), false),
			SExpr(Int(3, p(6, 6))),
			SCodeEnd(Lit("%}", p(6, 11)), true),
			SMixedText(p(6, 13), "\n", RemoveLeftSpaces),
		)
	})
	test.ExpectParseMixed(t, "‹- 1 -› a", func(p pfn) []Stmt {
		return stmts(
			SCodeBegin(Lit("‹", p(1, 1)), true),
			SExpr(Int(1, p(1, 6))),
			SCodeEnd(Lit("›", p(1, 9)), true),
			SMixedText(p(1, 12), " a", RemoveLeftSpaces),
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
	test.ExpectParseStringMixed(t, "‹if 1 begin›2‹end›", "‹; if 1 begin ›2‹ end; ›")
	test.ExpectParseStringMixed(t, "‹if 1 begin›2‹else if 3 begin›4‹end›", "‹; if 1 begin ›2‹ else if 3 begin ›4‹ end; ›")
	test.ExpectParseStringMixed(t, "‹if 1 begin›2‹else›3‹end›", "‹; if 1 begin ›2‹ else ›3‹ end; ›")
	test.ExpectParseStringMixed(t, "‹if 1 begin›2‹if 2 begin›3‹end›‹end›", "‹; if 1 begin ›2‹; if 2 begin ›3‹ end end; ›")
	test.ExpectParseStringMixed(t, "‹ if 1 begin › 2 ‹ end ›", "‹; if 1 begin › 2 ‹ end; ›")

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
if 1 begin
//
›
d
‹
//src:4
end
//
›
`, "\na\n; ‹; x := 2; ›\nb\n‹=(x ** 10)›\nc\n‹; if 1 begin ›\nd\n‹ end; ›\n")
}

func TestFormatDecl(t *testing.T) {
	// A single declaration is written without parentheses, for every keyword.
	test.New(t, "var (x)").Code("var x").FormattedCode("var x")
	test.New(t, "var x").Code("var x").FormattedCode("var x")
	test.New(t, "const x = 1").Code("const x = 1").FormattedCode("const x = 1")
	test.New(t, "global g").Code("global g").FormattedCode("global g")

	// A group keeps parentheses; inline stays compact, formatted splits one
	// spec per line (no trailing comma on the last).
	test.New(t, "var (x, y)").Code("var (x, y)").FormattedCode("var (\n\tx\n\ty\n)")
	test.New(t, "const (a = 1, b = 2)").Code("const (a = 1, b = 2)").
		FormattedCode("const (\n\ta = 1\n\tb = 2\n)")

	// A lone named param keeps its parens and the `;` (the single-decl rule
	// must not turn `param (; x)` into `param x`).
	test.New(t, "param x").Code("param x").FormattedCode("param x")
	test.New(t, "param (;x)").Code("param (; x)").FormattedCode("param (\n\t; x\n)")
	test.New(t, "param (a; x)").Code("param (a; x)").
		FormattedCode("param (\n\ta\n\t; x\n)")
	test.New(t, "param (a, b; c)").Code("param (a, b; c)")
	test.New(t, "param (;x, **y)").Code("param (; x, **y)")
}

func TestFormatCalcParams(t *testing.T) {
	// Short param lists stay inline under the column budget.
	test.New(t, "func(a int, b int) { return }").
		FormattedCalcCode("func(a int, b int) {\n\treturn\n}", 40)

	// When the list overflows, params split one per line (no commas); a typed
	// param keeps its ident and type together.
	test.New(t, "func(a int|bool|string, b int) { return }").
		FormattedCalcCode("func(\n\ta int | bool | string\n\tb int\n) {\n\treturn\n}", 28)

	// When a single param's type union is still too wide, its types wrap
	// greedily (packed per line) with the `|` trailing each wrapped line and the
	// continuation indented one extra level; the params themselves also pack
	// greedily, so the next param joins the union's last line when it fits.
	test.New(t, "f := func(verylongname int|boolean|string|number, other int) { return }").
		FormattedCalcCode("f := func(\n\tverylongname int | boolean |\n\t\tstring | number, other int\n) {\n\treturn\n}", 28)
}

func TestFormatCalcGreedy(t *testing.T) {
	// Array items wrap greedily: packed per line, no comma at a break, content
	// indented one level (no extra indent for continuation lines).
	test.New(t, "x := [1, 2, 3, 4, 5, 6, 7, 8]").
		FormattedCalcCode("x := [\n\t1, 2, 3, 4, 5\n\t6, 7, 8\n]", 14)

	// Key-value array items wrap greedily the same way.
	test.New(t, "x := (;a=1, b=2, c=3, d=4, e=5, f=6)").
		FormattedCalcCode("x := (;\n\ta=1, b=2, c=3\n\td=4, e=5, f=6\n)", 14)

	// A value-less declaration group wraps greedily (no extra indent).
	test.New(t, "var (a, b, c, d, e, f, g, h)").
		FormattedCalcCode("var (\n\ta, b, c, d, e\n\tf, g, h\n)", 14)

	// A short construct stays inline.
	test.New(t, "x := [1, 2, 3]").FormattedCalcCode("x := [1, 2, 3]", 80)

	// Function header params wrap greedily (no extra indent).
	test.New(t, "func(aa, bb, cc, dd, ee, ff, gg) { return }").
		FormattedCalcCode("func(\n\taa, bb, cc, dd\n\tee, ff, gg\n) {\n\treturn\n}", 16)

	// Call args wrap greedily.
	test.New(t, "f(aa, bb, cc, dd, ee, ff, gg)").
		FormattedCalcCode("f(\n\taa, bb, cc, dd\n\tee, ff, gg\n)", 16)

	// Named params/args: the `;` introduces the named section inline and the
	// items pack greedily.
	test.New(t, "f(aa, bb; xx=1, yy=2, zz=3, ww=4)").
		FormattedCalcCode("f(\n\taa, bb; xx=1\n\tyy=2, zz=3, ww=4\n)", 18)
}

func TestFormatReturnUnion(t *testing.T) {
	// Return-type unions are spaced around `|` just like parameter unions.
	test.New(t, "func() <x int|bool> { return 1 }").
		IndentedCode("func() <x int | bool> {\n\treturn 1\n}")
	test.New(t, "func() <x int|bool, y str> { return 1 }").
		IndentedCode("func() <x int | bool, y str> {\n\treturn 1\n}")
}

func TestFormatMixedMode(t *testing.T) {
	mixed := func(src string) *test.Parser {
		return test.New(t, src).WithMixed().
			WithScannerOptions(func(o *ScannerOptions) {
				o.Mode |= ScanMixed
				o.MixedDelimiter.Start = []rune("{%")
				o.MixedDelimiter.End = []rune("%}")
			})
	}

	// Tags stay inline; text is preserved verbatim; `{%= … %}` is padded; the
	// for-body block gains an explicit `begin` opener and a normalized `end`.
	src := "# gad: mixed\n{% for i, x in items begin %}[{%= x %}]{%- end -%}\n"
	mixed(src).FormattedCode(src)

	// Surrounding spaces around the terminator normalize to `{% end %}`.
	mixed("# gad: mixed\n{% if x begin %}y{%   end   %}\n").
		FormattedCode("# gad: mixed\n{% if x begin %}y{% end %}\n")

	// A for-loop without an explicit opener gains `begin`.
	mixed("# gad: mixed\n{% for x in items %}a{% end %}\n").
		FormattedCode("# gad: mixed\n{% for x in items begin %}a{% end %}\n")

	// Single (`-`, keep a newline) and double (`--`, strip all) trim markers
	// round-trip through the formatter.
	mixed("# gad: mixed\nA\n{%-- = 1 --%}\nB{%- = 2 -%}C").
		FormattedCode("# gad: mixed\nA\n{%-- = 1 --%}\nB{%- = 2 -%}C")
}

func TestTranspileMixed(t *testing.T) {
	parseMixed := func(src string) []Stmt {
		fs := source.NewFileSet()
		f := fs.AddFileData("t", -1, []byte(src))
		file, err := NewParserWithOptions(f,
			&ParserOptions{Mode: ParseMixed}, &ScannerOptions{Mode: ScanMixed}).ParseFile()
		require.NoError(t, err)
		return file.Stmts
	}

	src := "# gad: mixed\n{% name := \"Gad\" -%}\nHi, {%= name %}!\n" +
		"{% for x in [1, 2] begin -%}\n- {%= x %}\n{%- end %}"
	to := &TranspileOptions{RawStrFuncStart: "rawstr(", RawStrFuncEnd: ")", WriteFunc: "write"}
	out := Code(Stmts(parseMixed(src)),
		CodeWithFlags(CodeWriteContextFlagFormat), CodeWithPrefix("\t"), CodeTranspile(to))

	// Transpiled write(...) statements must be separated, not glued together.
	for _, bad := range []string{"))name", ")write(", "}for ", "}write("} {
		if strings.Contains(out, bad) {
			t.Fatalf("transpiled statements glued (%q):\n%s", bad, out)
		}
	}
	// The transpiled output must itself be valid Gad source.
	fs := source.NewFileSet()
	if _, err := NewParserWithOptions(fs.AddFileData("o", -1, []byte(out)), nil, nil).ParseFile(); err != nil {
		t.Fatalf("transpiled output does not parse: %v\n%s", err, out)
	}
}

func TestMixedTrimMarkers(t *testing.T) {
	// value parses src as a template and returns the run-time Value() of the
	// MixedText statement at index i.
	value := func(src string, i int) string {
		fs := source.NewFileSet()
		f := fs.AddFileData("t", -1, []byte(src))
		file, err := NewParserWithOptions(f,
			&ParserOptions{Mode: ParseMixed}, &ScannerOptions{Mode: ScanMixed}).ParseFile()
		require.NoError(t, err)
		return file.Stmts[i].(*MixedTextStmt).Value()
	}

	// `-` keeps a single boundary newline; `--` strips all boundary whitespace.
	// Stmts: [MixedText "A\n", value, MixedText "\nB"].
	if got := value("A\n{%- = 1 -%}\nB", 0); got != "A\n" {
		t.Errorf("single left text = %q, want %q", got, "A\n")
	}
	if got := value("A\n{%- = 1 -%}\nB", 2); got != "\nB" {
		t.Errorf("single right text = %q, want %q", got, "\nB")
	}
	if got := value("A\n{%-- = 1 --%}\nB", 0); got != "A" {
		t.Errorf("double left text = %q, want %q", got, "A")
	}
	if got := value("A\n{%-- = 1 --%}\nB", 2); got != "B" {
		t.Errorf("double right text = %q, want %q", got, "B")
	}
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
			SDecl(
				NewGenDecl(token.Param, p(1, 1), 0, 0,
					NewParamSpec(false, ETypedIdent(EIdent("a", p(1, 7)))),
				),
			),
		)
	})
	test.ExpectParse(t, `param *a;`, func(p pfn) []Stmt {
		return stmts(
			SDecl(
				NewGenDecl(token.Param, p(1, 1), 0, 0,
					NewParamSpec(true, ETypedIdent(EIdent("a", p(1, 8)))),
				),
			),
		)
	})
	test.ExpectParse(t, `param (a, *b)`, func(p pfn) []Stmt {
		return stmts(
			SDecl(
				NewGenDecl(token.Param, p(1, 1), p(1, 7), p(1, 13),
					NewParamSpec(false, ETypedIdent(EIdent("a", p(1, 8)))),
					NewParamSpec(true, ETypedIdent(EIdent("b", p(1, 12)))),
				),
			),
		)
	})
	test.ExpectParse(t, `param (a,
*b)`, func(p pfn) []Stmt {
		return stmts(
			SDecl(
				NewGenDecl(token.Param, p(1, 1), p(1, 7), p(2, 3),
					NewParamSpec(false, ETypedIdent(EIdent("a", p(1, 8)))),
					NewParamSpec(true, ETypedIdent(EIdent("b", p(2, 2)))),
				),
			),
		)
	})

	test.ExpectParse(t, `param (a, *b; c=1, d=2, **e)`, func(p pfn) []Stmt {
		return stmts(
			SDecl(
				NewGenDecl(token.Param, p(1, 1), p(1, 7), p(1, 28),
					NewParamSpec(false, ETypedIdent(EIdent("a", p(1, 8)))),
					NewParamSpec(true, ETypedIdent(EIdent("b", p(1, 12)))),
					NewNamedParamSpec(ETypedIdent(EIdent("c", p(1, 15))), Int(1, p(1, 17))),
					NewNamedParamSpec(ETypedIdent(EIdent("d", p(1, 20))), Int(2, p(1, 22))),
					NewNamedParamSpecVar(ETypedIdent(EIdent("e", p(1, 27)))),
				),
			),
		)
	})
	test.ExpectParse(t, `param (;c=1, d=2, **e)`, func(p pfn) []Stmt {
		return stmts(
			SDecl(
				NewGenDecl(token.Param, p(1, 1), p(1, 7), p(1, 22),
					NewNamedParamSpec(ETypedIdent(EIdent("c", p(1, 9))), Int(1, p(1, 11))),
					NewNamedParamSpec(ETypedIdent(EIdent("d", p(1, 14))), Int(2, p(1, 16))),
					NewNamedParamSpecVar(ETypedIdent(EIdent("e", p(1, 21)))),
				),
			),
		)
	})
	test.ExpectParse(t, `param (;c=1, d=2, **e)`, func(p pfn) []Stmt {
		return stmts(
			SDecl(
				NewGenDecl(token.Param, p(1, 1), p(1, 7), p(1, 22),
					NewNamedParamSpec(ETypedIdent(EIdent("c", p(1, 9))), Int(1, p(1, 11))),
					NewNamedParamSpec(ETypedIdent(EIdent("d", p(1, 14))), Int(2, p(1, 16))),
					NewNamedParamSpecVar(ETypedIdent(EIdent("e", p(1, 21)))),
				),
			),
		)
	})

	test.ExpectParse(t, `param (a, *b; c=1, d=2, x, **e)`, func(p pfn) []Stmt {
		return stmts(
			SDecl(
				NewGenDecl(token.Param, p(1, 1), p(1, 7), p(1, 31),
					NewParamSpec(false, ETypedIdent(EIdent("a", p(1, 8)))),
					NewParamSpec(true, ETypedIdent(EIdent("b", p(1, 12)))),
					NewNamedParamSpec(ETypedIdent(EIdent("c", p(1, 15))), Int(1, p(1, 17))),
					NewNamedParamSpec(ETypedIdent(EIdent("d", p(1, 20))), Int(2, p(1, 22))),
					NewNamedParamSpec(ETypedIdent(EIdent("x", p(1, 25))), nil),
					NewNamedParamSpecVar(ETypedIdent(EIdent("e", p(1, 30)))),
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
			SDecl(
				NewGenDecl(token.Global, p(1, 1), 0, 0,
					NewParamSpec(false, ETypedIdent(EIdent("a", p(1, 8)))),
				),
			),
		)
	})
	test.ExpectParse(t, `
global a
global b`, func(p pfn) []Stmt {
		return stmts(
			SDecl(
				NewGenDecl(token.Global, p(2, 1), 0, 0,
					NewParamSpec(false, ETypedIdent(EIdent("a", p(2, 8)))),
				),
			),
			SDecl(
				NewGenDecl(token.Global, p(3, 1), 0, 0,
					NewParamSpec(false, ETypedIdent(EIdent("b", p(3, 8)))),
				),
			),
		)
	})
	test.ExpectParse(t, `global (a, b)`, func(p pfn) []Stmt {
		return stmts(
			SDecl(
				NewGenDecl(token.Global, p(1, 1), p(1, 8), p(1, 13),
					NewParamSpec(false, ETypedIdent(EIdent("a", p(1, 9)))),
					NewParamSpec(false, ETypedIdent(EIdent("b", p(1, 12)))),
				),
			),
		)
	})
	test.ExpectParse(t, `global (a, 
b)`, func(p pfn) []Stmt {
		return stmts(
			SDecl(
				NewGenDecl(token.Global, p(1, 1), p(1, 8), p(2, 2),
					NewParamSpec(false, ETypedIdent(EIdent("a", p(1, 9)))),
					NewParamSpec(false, ETypedIdent(EIdent("b", p(2, 1)))),
				),
			),
		)
	})
	test.ExpectParse(t, `global (a 
b)`, func(p pfn) []Stmt {
		return stmts(
			SDecl(
				NewGenDecl(token.Global, p(1, 1), p(1, 8), p(2, 2),
					NewParamSpec(false, ETypedIdent(EIdent("a", p(1, 9)))),
					NewParamSpec(false, ETypedIdent(EIdent("b", p(2, 1)))),
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
			SDecl(
				NewGenDecl(token.Var, p(1, 1), 0, 0,
					NewValueSpec(
						[]*IdentExpr{EIdent("a", p(1, 5))},
						[]Expr{nil}),
				),
			),
		)
	})
	test.ExpectParse(t, `var a=1`, func(p pfn) []Stmt {
		return stmts(
			SDecl(
				NewGenDecl(token.Var, p(1, 1), 0, 0,
					NewValueSpec(
						[]*IdentExpr{EIdent("a", p(1, 5))},
						[]Expr{Int(1, p(1, 7))}),
				),
			),
		)
	})
	test.ExpectParse(t, `var a;var b`, func(p pfn) []Stmt {
		return stmts(
			SDecl(
				NewGenDecl(token.Var, p(1, 1), 0, 0,
					NewValueSpec(
						[]*IdentExpr{EIdent("a", p(1, 5))},
						[]Expr{nil}),
				),
			),
			SDecl(
				NewGenDecl(token.Var, p(1, 7), 0, 0,
					NewValueSpec(
						[]*IdentExpr{EIdent("b", p(1, 11))},
						[]Expr{nil}),
				),
			),
		)
	})
	test.ExpectParse(t, `var a="x";var b`, func(p pfn) []Stmt {
		return stmts(
			SDecl(
				NewGenDecl(token.Var, p(1, 1), 0, 0,
					NewValueSpec(
						[]*IdentExpr{EIdent("a", p(1, 5))},
						[]Expr{stringLit("x", p(1, 7))}),
				),
			),
			SDecl(
				NewGenDecl(token.Var, p(1, 11), 0, 0,
					NewValueSpec(
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
			SDecl(
				NewGenDecl(token.Var, p(2, 1), 0, 0,
					NewValueSpec(
						[]*IdentExpr{EIdent("a", p(2, 5))},
						[]Expr{nil}),
				),
			),
			SDecl(
				NewGenDecl(token.Var, p(3, 1), 0, 0,
					NewValueSpec(
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
			SDecl(
				NewGenDecl(token.Var, p(2, 1), 0, 0,
					NewValueSpec(
						[]*IdentExpr{EIdent("a", p(2, 5))},
						[]Expr{nil}),
				),
			),
			SDecl(
				NewGenDecl(token.Var, p(3, 1), 0, 0,
					NewValueSpec(
						[]*IdentExpr{EIdent("b", p(3, 5))},
						[]Expr{Int(2, p(3, 7))}),
				),
			),
		)
	})
	test.ExpectParse(t, `var (a, b=2)`, func(p pfn) []Stmt {
		return stmts(
			SDecl(
				NewGenDecl(token.Var, p(1, 1), p(1, 5), p(1, 12),
					NewValueSpec(
						[]*IdentExpr{EIdent("a", p(1, 6))},
						[]Expr{nil}),
					NewValueSpec(
						[]*IdentExpr{EIdent("b", p(1, 9))},
						[]Expr{Int(2, p(1, 11))}),
				),
			),
		)
	})
	test.ExpectParse(t, `var (a=1, b=2)`, func(p pfn) []Stmt {
		return stmts(
			SDecl(
				NewGenDecl(token.Var, p(1, 1), p(1, 5), p(1, 14),
					NewValueSpec(
						[]*IdentExpr{EIdent("a", p(1, 6))},
						[]Expr{Int(1, p(1, 8))}),
					NewValueSpec(
						[]*IdentExpr{EIdent("b", p(1, 11))},
						[]Expr{Int(2, p(1, 13))}),
				),
			),
		)
	})
	test.ExpectParse(t, `var (a=1,
b=2)`, func(p pfn) []Stmt {
		return stmts(
			SDecl(
				NewGenDecl(token.Var, p(1, 1), p(1, 5), p(2, 4),
					NewValueSpec(
						[]*IdentExpr{EIdent("a", p(1, 6))},
						[]Expr{Int(1, p(1, 8))}),
					NewValueSpec(
						[]*IdentExpr{EIdent("b", p(2, 1))},
						[]Expr{Int(2, p(2, 3))}),
				),
			),
		)
	})
	test.ExpectParse(t, `var (a=1
b=2)`, func(p pfn) []Stmt {
		return stmts(
			SDecl(
				NewGenDecl(token.Var, p(1, 1), p(1, 5), p(2, 4),
					NewValueSpec(
						[]*IdentExpr{EIdent("a", p(1, 6))},
						[]Expr{Int(1, p(1, 8))}),
					NewValueSpec(
						[]*IdentExpr{EIdent("b", p(2, 1))},
						[]Expr{Int(2, p(2, 3))}),
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
			SDecl(
				NewGenDecl(token.Const, p(1, 1), 0, 0,
					NewValueSpec(
						[]*IdentExpr{EIdent("a", p(1, 7))},
						[]Expr{Int(1, p(1, 11))}),
				),
			),
		)
	})
	test.ExpectParse(t, `const a = 1; const b = 2`, func(p pfn) []Stmt {
		return stmts(
			SDecl(
				NewGenDecl(token.Const, p(1, 1), 0, 0,
					NewValueSpec(
						[]*IdentExpr{EIdent("a", p(1, 7))},
						[]Expr{Int(1, p(1, 11))}),
				),
			),
			SDecl(
				NewGenDecl(token.Const, p(1, 14), 0, 0,
					NewValueSpec(
						[]*IdentExpr{EIdent("b", p(1, 20))},
						[]Expr{Int(2, p(1, 24))}),
				),
			),
		)
	})
	test.ExpectParse(t, `const (a = 1, b = 2)`, func(p pfn) []Stmt {
		return stmts(
			SDecl(
				NewGenDecl(token.Const, p(1, 1), p(1, 7), p(1, 20),
					NewValueSpec(
						[]*IdentExpr{EIdent("a", p(1, 8))},
						[]Expr{Int(1, p(1, 12))}),
					NewValueSpec(
						[]*IdentExpr{EIdent("b", p(1, 15))},
						[]Expr{Int(2, p(1, 19))}),
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
			SDecl(
				NewGenDecl(token.Const, p(2, 1), p(2, 7), p(5, 1),
					NewValueSpec(
						[]*IdentExpr{EIdent("a", p(3, 5))},
						[]Expr{Int(1, p(3, 9))}),
					NewValueSpec(
						[]*IdentExpr{EIdent("b", p(4, 5))},
						[]Expr{Int(2, p(4, 9))}),
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

	test.New(t, `const a = func() { const a1 = func() { const a2 = func() {}}}`).
		FormattedCode(`const a = func() {
	const a1 = func() {
		const a2 = func() {}
	}
}`)

	test.New(t, `const a = func() { const a1 = func() { const a2 = func() {x};y};z};s`).
		FormattedCode(`const a = func() {
	const a1 = func() {
		const a2 = func() {
			x
		}

		y
	}

	z
}

s`)

	test.New(t, `const a = func() { const a1 = func() { const (b=1,c=2,d=3)}}`).
		FormattedCode(`const a = func() {
	const a1 = func() {
		const (
			b = 1
			c = 2
			d = 3
		)
	}
}`)
	test.New(t, `const a = func() { const a1 = func() { const (b = iota,c,d)}}`).
		FormattedCode(`const a = func() {
	const a1 = func() {
		const (
			b = iota
			c
			d
		)
	}
}`)

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
			SReturn(
				p(1, 1),
				Array(
					p(1, 8),
					p(1, 13),
					Int(1, p(1, 8)),
					Int(23, p(1, 11)),
				),
			),
		)
	})
	test.ExpectParse(t, "return 1, 23, 2.2, 12.34d", func(p pfn) []Stmt {
		return stmts(
			SReturn(
				p(1, 1),
				Array(
					p(1, 8),
					p(1, 26),
					Int(1, p(1, 8)),
					Int(23, p(1, 11)),
					Float(2.2, p(1, 15)),
					Decimal("12.34", p(1, 20)),
				),
			),
		)
	})
	test.ExpectParse(t, "return a, b", func(p pfn) []Stmt {
		return stmts(
			SReturn(
				p(1, 1),
				Array(
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
			SFunc(
				EFunc(
					funcType(p(1, 1), nil, p(1, 5), p(1, 6)),
					SBlock(
						p(1, 8),
						p(1, 22),
						SReturn(
							p(1, 10),
							Array(
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
			SExpr(
				Array(p(1, 1), p(1, 9),
					Int(1, p(1, 2)),
					Int(2, p(1, 5)),
					Int(3, p(1, 8)))))
	})

	test.ExpectParse(t, `
[
	1, 
	2, 
	3,
]`, func(p pfn) []Stmt {
		return stmts(
			SExpr(
				Array(p(2, 1), p(6, 1),
					Int(1, p(3, 2)),
					Int(2, p(4, 2)),
					Int(3, p(5, 2)))))
	})
	test.ExpectParse(t, `
[
	1, 
	2, 
	3,

]`, func(p pfn) []Stmt {
		return stmts(
			SExpr(
				Array(p(2, 1), p(7, 1),
					Int(1, p(3, 2)),
					Int(2, p(4, 2)),
					Int(3, p(5, 2)))))
	})

	test.ExpectParse(t, `[1, "foo", 12.34]`, func(p pfn) []Stmt {
		return stmts(
			SExpr(
				Array(p(1, 1), p(1, 17),
					Int(1, p(1, 2)),
					stringLit("foo", p(1, 5)),
					Float(12.34, p(1, 12)))))
	})

	test.ExpectParse(t, "a = [1, 2, 3]", func(p pfn) []Stmt {
		return stmts(
			SAssign(
				exprs(EIdent("a", p(1, 1))),
				exprs(Array(p(1, 5), p(1, 13),
					Int(1, p(1, 6)),
					Int(2, p(1, 9)),
					Int(3, p(1, 12)))),
				token.Assign,
				p(1, 3)))
	})

	test.ExpectParse(t, "a = [1 + 2, b * 4, [4, c]]", func(p pfn) []Stmt {
		return stmts(
			SAssign(
				exprs(EIdent("a", p(1, 1))),
				exprs(Array(p(1, 5), p(1, 26),
					EBinary(
						Int(1, p(1, 6)),
						Int(2, p(1, 10)),
						token.Add,
						p(1, 8)),
					EBinary(
						EIdent("b", p(1, 13)),
						Int(4, p(1, 17)),
						token.Mul,
						p(1, 15)),
					Array(p(1, 20), p(1, 25),
						Int(4, p(1, 21)),
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
			SAssign(
				exprs(EIdent("a", p(1, 1))),
				exprs(Int(5, p(1, 5))),
				token.Assign,
				p(1, 3)))
	})

	test.ExpectParse(t, "a := 5", func(p pfn) []Stmt {
		return stmts(
			SAssign(
				exprs(EIdent("a", p(1, 1))),
				exprs(Int(5, p(1, 6))),
				token.Define,
				p(1, 3)))
	})

	test.ExpectParse(t, "a, b = 5, 10", func(p pfn) []Stmt {
		return stmts(
			SAssign(
				exprs(
					EIdent("a", p(1, 1)),
					EIdent("b", p(1, 4))),
				exprs(
					Int(5, p(1, 8)),
					Int(10, p(1, 11))),
				token.Assign,
				p(1, 6)))
	})

	test.ExpectParse(t, "a, b := 5, 10", func(p pfn) []Stmt {
		return stmts(
			SAssign(
				exprs(
					EIdent("a", p(1, 1)),
					EIdent("b", p(1, 4))),
				exprs(
					Int(5, p(1, 9)),
					Int(10, p(1, 12))),
				token.Define,
				p(1, 6)))
	})

	test.ExpectParse(t, "a, b = a + 2, b - 8", func(p pfn) []Stmt {
		return stmts(
			SAssign(
				exprs(
					EIdent("a", p(1, 1)),
					EIdent("b", p(1, 4))),
				exprs(
					EBinary(
						EIdent("a", p(1, 8)),
						Int(2, p(1, 12)),
						token.Add,
						p(1, 10)),
					EBinary(
						EIdent("b", p(1, 15)),
						Int(8, p(1, 19)),
						token.Sub,
						p(1, 17))),
				token.Assign,
				p(1, 6)))
	})

	test.ExpectParse(t, "a = [1, 2, 3]", func(p pfn) []Stmt {
		return stmts(
			SAssign(
				exprs(EIdent("a", p(1, 1))),
				exprs(Array(p(1, 5), p(1, 13),
					Int(1, p(1, 6)),
					Int(2, p(1, 9)),
					Int(3, p(1, 12)))),
				token.Assign,
				p(1, 3)))
	})

	test.ExpectParse(t, "a = [1 + 2, b * 4, [4, c]]", func(p pfn) []Stmt {
		return stmts(
			SAssign(
				exprs(EIdent("a", p(1, 1))),
				exprs(Array(p(1, 5), p(1, 26),
					EBinary(
						Int(1, p(1, 6)),
						Int(2, p(1, 10)),
						token.Add,
						p(1, 8)),
					EBinary(
						EIdent("b", p(1, 13)),
						Int(4, p(1, 17)),
						token.Mul,
						p(1, 15)),
					Array(p(1, 20), p(1, 25),
						Int(4, p(1, 21)),
						EIdent("c", p(1, 24))))),
				token.Assign,
				p(1, 3)))
	})

	test.ExpectParse(t, "a += 5", func(p pfn) []Stmt {
		return stmts(
			SAssign(
				exprs(EIdent("a", p(1, 1))),
				exprs(Int(5, p(1, 6))),
				token.AddAssign,
				p(1, 3)))
	})

	test.ExpectParse(t, "a *= 5 + 10", func(p pfn) []Stmt {
		return stmts(
			SAssign(
				exprs(EIdent("a", p(1, 1))),
				exprs(
					EBinary(
						Int(5, p(1, 6)),
						Int(10, p(1, 10)),
						token.Add,
						p(1, 8))),
				token.MulAssign,
				p(1, 3)))
	})

	test.ExpectParse(t, "a ||= 5", func(p pfn) []Stmt {
		return stmts(
			SAssign(
				exprs(EIdent("a", p(1, 1))),
				exprs(Int(5, p(1, 7))),
				token.LOrAssign,
				p(1, 3)))
	})

	test.ExpectParse(t, "a ||= 5 + 10", func(p pfn) []Stmt {
		return stmts(
			SAssign(
				exprs(EIdent("a", p(1, 1))),
				exprs(
					EBinary(
						Int(5, p(1, 7)),
						Int(10, p(1, 11)),
						token.Add,
						p(1, 9))),
				token.LOrAssign,
				p(1, 3)))
	})

	test.ExpectParse(t, "a ??= 5", func(p pfn) []Stmt {
		return stmts(
			SAssign(
				exprs(EIdent("a", p(1, 1))),
				exprs(Int(5, p(1, 7))),
				token.NullichAssign,
				p(1, 3)))
	})

	test.ExpectParse(t, "a ??= 5 + 10", func(p pfn) []Stmt {
		return stmts(
			SAssign(
				exprs(EIdent("a", p(1, 1))),
				exprs(
					EBinary(
						Int(5, p(1, 7)),
						Int(10, p(1, 11)),
						token.Add,
						p(1, 9))),
				token.NullichAssign,
				p(1, 3)))
	})

	test.ExpectParse(t, "a ++= 5", func(p pfn) []Stmt {
		return stmts(
			SAssign(
				exprs(EIdent("a", p(1, 1))),
				exprs(Int(5, p(1, 7))),
				token.IncAssign,
				p(1, 3)))
	})

	test.ExpectParse(t, "a --= 5", func(p pfn) []Stmt {
		return stmts(
			SAssign(
				exprs(EIdent("a", p(1, 1))),
				exprs(Int(5, p(1, 7))),
				token.DecAssign,
				p(1, 3)))
	})

	test.ExpectParse(t, "a **= 5 + 10", func(p pfn) []Stmt {
		return stmts(
			SAssign(
				exprs(EIdent("a", p(1, 1))),
				exprs(
					EBinary(
						Int(5, p(1, 7)),
						Int(10, p(1, 11)),
						token.Add,
						p(1, 9))),
				token.PowAssign,
				p(1, 3)))
	})
}

func TestParseUnaryNulls(t *testing.T) {
	test.ExpectParse(t, "false == nil", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				EUnary(
					Bool(false, p(1, 1)),
					token.Null,
					p(1, 7))))
	})

	test.ExpectParse(t, "false != nil", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				EUnary(
					Bool(false, p(1, 1)),
					token.NotNull,
					p(1, 7))))
	})

	test.ExpectParseString(t, "false == nil", "(false == nil)")
	test.ExpectParseString(t, "false != nil", "(false != nil)")
	test.ExpectParseString(t, "nil == nil", "(nil == nil)")
	test.ExpectParseString(t, "nil != nil", "(nil != nil)")

	test.ExpectParse(t, "a == nil ? b : c", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				ECond(
					EUnary(
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
			SExpr(
				ECond(
					EUnary(
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
			SExpr(
				Bool(true, p(1, 1))))
	})

	test.ExpectParse(t, "false", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				Bool(false, p(1, 1))))
	})

	test.ExpectParse(t, "true != false", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				EBinary(
					Bool(true, p(1, 1)),
					Bool(false, p(1, 9)),
					token.NotEqual,
					p(1, 6))))
	})

	test.ExpectParse(t, "!false", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				EUnary(
					Bool(false, p(1, 2)),
					token.Not,
					p(1, 1))))
	})
}

func TestParseFlag(t *testing.T) {
	test.ExpectParse(t, "yes", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				Flag(true, p(1, 1))))
	})

	test.ExpectParse(t, "no", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				Flag(false, p(1, 1))))
	})

	test.ExpectParse(t, "yes != no", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				EBinary(
					Flag(true, p(1, 1)),
					Flag(false, p(1, 8)),
					token.NotEqual,
					p(1, 5))))
	})

	test.ExpectParse(t, "!no", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				EUnary(
					Flag(false, p(1, 2)),
					token.Not,
					p(1, 1))))
	})
}

func TestParseCallKeywords(t *testing.T) {
	test.ExpectParse(t, token.Callee.String(), func(p pfn) []Stmt {
		return stmts(SExpr(CaleeKW(p(1, 1))))
	})
	test.ExpectParse(t, token.Args.String(), func(p pfn) []Stmt {
		return stmts(SExpr(ArgsKW(p(1, 1))))
	})
	test.ExpectParse(t, token.NamedArgs.String(), func(p pfn) []Stmt {
		return stmts(SExpr(NamedArgsKW(p(1, 1))))
	})
	test.ExpectParseString(t, token.Callee.String(), token.Callee.String())
	test.ExpectParseString(t, token.Args.String(), token.Args.String())
	test.ExpectParseString(t, token.NamedArgs.String(), token.NamedArgs.String())
}

func TestParseCallNewlineArgs(t *testing.T) {
	// call args and func params may be newline-separated (comma optional); a
	// comma may still be followed by a newline, and named args follow `;`.
	test.ExpectParseString(t, "f(1\n2\n3)", "f(1, 2, 3)")
	test.ExpectParseString(t, "f(1,\n2\n3)", "f(1, 2, 3)")
	test.ExpectParseString(t, "f(\n1\n2\n; x=3\n)", "f(1, 2; x=3)")
	test.ExpectParseString(t, "func(\na\nb\n){}", "func(a, b) {}")
	test.ExpectParseString(t, "func(a\nb int\nc){}", "func(a, b int, c) {}")
}

func TestParseCall(t *testing.T) {
	test.New(t, "add(1, 2; x(){y++}, y()=>1, **d)").
		String("add(1, 2; x() { y++ }, y() => 1, **d)").
		Code("add(1, 2; x() {y++}, y() => 1, **d)").
		FormattedCode(`add(
	1
	2
	; x() {
		y++
	},
	y() => 1,
	**d
)`)
	test.ExpectParse(t, "add(,)", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				ECall(
					EIdent("add", p(1, 1)),
					p(1, 4), p(1, 6),
					NewCallExprArgs(nil))))
	})
	test.ExpectParse(t, "add(\n\t,)", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				ECall(
					EIdent("add", p(1, 1)),
					p(1, 4), p(2, 3),
					NewCallExprArgs(nil))))
	})
	test.ExpectParse(t, "add(1, 2, 3)", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				ECall(
					EIdent("add", p(1, 1)),
					p(1, 4), p(1, 12),
					NewCallExprArgs(nil,
						Int(1, p(1, 5)),
						Int(2, p(1, 8)),
						Int(3, p(1, 11))))))
	})
	test.ExpectParse(t, "add(1, 2, *v)", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				ECall(
					EIdent("add", p(1, 1)),
					p(1, 4), p(1, 13),
					NewCallExprArgs(
						ArgVar(p(1, 11), EIdent("v", p(1, 12))),
						Int(1, p(1, 5)),
						Int(2, p(1, 8))))))
	})
	test.ExpectParse(t, "a = add(1, 2, 3)", func(p pfn) []Stmt {
		return stmts(
			SAssign(
				exprs(
					EIdent("a", p(1, 1))),
				exprs(
					ECall(
						EIdent("add", p(1, 5)),
						p(1, 8), p(1, 16),
						NewCallExprArgs(nil,
							Int(1, p(1, 9)),
							Int(2, p(1, 12)),
							Int(3, p(1, 15))))),
				token.Assign,
				p(1, 3)))
	})
	test.ExpectParse(t, "a, b = add(1, 2, 3)", func(p pfn) []Stmt {
		return stmts(
			SAssign(
				exprs(
					EIdent("a", p(1, 1)),
					EIdent("b", p(1, 4))),
				exprs(
					ECall(
						EIdent("add", p(1, 8)),
						p(1, 11), p(1, 19),
						NewCallExprArgs(nil, Int(1, p(1, 12)),
							Int(2, p(1, 15)),
							Int(3, p(1, 18))))),
				token.Assign,
				p(1, 6)))
	})
	test.ExpectParse(t, "add(a + 1, 2 * 1, (b + c))", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				ECall(
					EIdent("add", p(1, 1)),
					p(1, 4), p(1, 26),
					NewCallExprArgs(nil,
						EBinary(
							EIdent("a", p(1, 5)),
							Int(1, p(1, 9)),
							token.Add,
							p(1, 7)),
						EBinary(
							Int(2, p(1, 12)),
							Int(1, p(1, 16)),
							token.Mul,
							p(1, 14)),
						EParen(
							EBinary(
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
			SExpr(
				ECall(
					EFunc(
						funcType(p(1, 1), nil, p(1, 5), p(1, 10),
							funcArgs(nil,
								EIdent("a", p(1, 6)),
								EIdent("b", p(1, 9))),
						),
						SBlock(
							p(1, 12), p(1, 20),
							SExpr(
								EBinary(
									EIdent("a", p(1, 14)),
									EIdent("b", p(1, 18)),
									token.Add,
									p(1, 16))))),
					p(1, 21),
					p(1, 26),
					NewCallExprArgs(nil,
						Int(1, p(1, 22)),
						Int(2, p(1, 25))))))
	})

	test.ExpectParse(t, `a.b()`, func(p pfn) []Stmt {
		return stmts(
			SExpr(
				ECall(
					ESelector(
						EIdent("a", p(1, 1)),
						stringLit("b", p(1, 3))),
					p(1, 4), p(1, 5), NoPos)))
	})

	test.ExpectParse(t, `a.b.c()`, func(p pfn) []Stmt {
		return stmts(
			SExpr(
				ECall(
					ESelector(
						ESelector(
							EIdent("a", p(1, 1)),
							stringLit("b", p(1, 3))),
						stringLit("c", p(1, 5))),
					p(1, 6), p(1, 7), NoPos)))
	})

	test.ExpectParse(t, `a["b"].c()`, func(p pfn) []Stmt {
		return stmts(
			SExpr(
				ECall(
					ESelector(
						EIndex(
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
		FormattedCode(`f(; x {
	() => nil

	(y) => nil
})
f(
	1
	; x {
		() => nil

		(y) => nil
	}
)
f(
	1
	2
	; x {
		() => nil

		(y) => nil
	}
)
f(
	1
	*s
	; x {
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
	1
	; z,
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
	1
	; z=1,
	x {
		() => nil

		(y) => nil
	},
	x=2
)`)

	test.ExpectParse(t, "add(;x=2)", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				ECall(
					EIdent("add", p(1, 1)),
					p(1, 4), p(1, 9),
					callExprNamedArgs(
						[]*NamedArgExpr{{Ident: EIdent("x", p(1, 6))}},
						[]Expr{Int(2, p(1, 8))},
					))))
	})
	test.ExpectParse(t, "add(;x=2,y=3)", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				ECall(
					EIdent("add", p(1, 1)),
					p(1, 4), p(1, 13),
					callExprNamedArgs(
						[]*NamedArgExpr{{Ident: EIdent("x", p(1, 6))}, {Ident: EIdent("y", p(1, 10))}},
						[]Expr{Int(2, p(1, 8)), Int(3, p(1, 12))},
					))))
	})
	test.ExpectParse(t, "add(;x=2,**{})", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				ECall(
					EIdent("add", p(1, 1)),
					p(1, 4), p(1, 14),
					callExprNamedArgs(
						[]*NamedArgExpr{{Ident: EIdent("x", p(1, 6))}, {Var: true, Exp: EDict(12, 13)}},
						[]Expr{Int(2, p(1, 8)), nil},
					))))
	})
	test.ExpectParse(t, "add(;\"x\"=2,y=3)", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				ECall(
					EIdent("add", p(1, 1)),
					p(1, 4), p(1, 15),
					callExprNamedArgs(
						[]*NamedArgExpr{{Lit: stringLit("x", p(1, 6))}, {Ident: EIdent("y", p(1, 12))}},
						[]Expr{Int(2, p(1, 10)), Int(3, p(1, 14))},
					))))
	})

	test.ExpectParseString(t, "add(;x() => 3)", "add(; x() => 3)")
	test.ExpectParseString(t, "add(1, 2; x () { y++ })", "add(1, 2; x() { y++ })")
	test.ExpectParseString(t, `attrs(;"name")`, `attrs(; "name"=yes)`)
	test.ExpectParseString(t, "fn(a;b)", "fn(a; b=yes)")
	test.ExpectParseString(t, "fn(;**{y:5})", "fn(; **{y: 5})")
	test.ExpectParseString(t, "fn(1,*[2,3];x=4,**{y:5})", "fn(1, *[2, 3]; x=4, **{y: 5})")
	test.ExpectParseString(t, "fn(1; a=b)()", "fn(1; a=b)()")

	test.New(t, "add(a;b=1)").
		FormattedCode(`add(
	a
	; b=1
)`)

	test.New(t, "add(;b=1)").
		FormattedCode(`add(; b=1)`)

	test.New(t, "add(a, b)").
		FormattedCode(`add(
	a
	b
)`)

	test.New(t, "add(a)").
		FormattedCode(`add(a)`)

	test.New(t, "add()").
		FormattedCode(`add()`)
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
			SExpr(
				KV(EIdent("a", p(1, 2)), Int(1, p(1, 4)))))
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
	a=1
	b=2
	"c"=3
	4=5
	true=false
	my_closure() => 1
	my_func() {
		i++
	}
	myflag
	d=#dVal
	e=#(e val)
	x A
	x1 A1=10
	y B | C=11
	y1 B | C=12
	z=(5 + 7)
)`)

	test.New(t, `(;fn {
	() => 1
	(x) => x
})`).
		String(`(;fn=func {() => 1; (x) => x; })`).
		Code("(; fn {() => 1; (x) => x})").
		FormattedCode(`(; fn {
	() => 1

	(x) => x
})`)

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
	a=1
	b=2
	"c"=3
	4=5
	true=false
	my_closure() => 1
	my_func() {
		i++
	}
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

	// with keyValue expr and keyValue with dynamic key expr
	test.ExpectParseStringT(t, `(;[a=1],[("a"+"b")=4])`, `(;[a=1], [("a" + "b")=4])`, kva)
}

func TestTemplateStrLit(t *testing.T) {
	test.ExpectParseString(t, `#"A"`, `#"A"`)
	test.ExpectParseString(t, "#`A`", "#`A`")
	test.ExpectParseString(t, "#```A```", "#```A```")
}

func TestTemplateString(t *testing.T) {
	// build parses `src` (an `x := #<template>` assignment), extracts the
	// TemplateLit, parses its template body and compiles it via Build. It
	// returns the original file's position function and the positional
	// arguments of the generated `str(...)` call, unwrapping the ToRaw
	// wrapper produced for raw (backtick) templates.
	// ofPos is original pos build
	// tfPos is template pos build
	build := func(t *testing.T, src string) (ofPos test.Pfn, tfPos test.Pfn, _ []Expr) {
		t.Helper()
		of := test.New(t, src).File().File()
		tmpl := of.Stmts[0].(*AssignStmt).RHS[0].(*TemplateLit)
		tf, err := ParseTemplateString(tmpl.StringValue(), tmpl.StringValuePos())
		require.NoError(t, err)
		expr, err := tmpl.Build(tf.Stmts)
		require.NoError(t, err)

		call, ok := expr.(*CallExpr)
		if !ok {
			raw, isRaw := expr.(*ToRaw)
			require.True(t, isRaw, "expected *CallExpr or *ToRaw, got %T", expr)
			call, ok = raw.Expr.(*CallExpr)
			require.True(t, ok, "expected *CallExpr inside ToRaw, got %T", raw.Expr)
		}

		fn, ok := call.Func.(*IdentExpr)
		require.True(t, ok, "expected *IdentExpr func, got %T", call.Func)
		require.Equal(t, "str", fn.Name)
		require.Equal(t, tmpl.Pos(), fn.NamePos)

		return of.InputFile.Pos, tf.InputFile.Pos, call.Args.Values
	}

	// expectPos asserts that the original-source position ofp and the
	// template-file position tfp resolve to the same absolute Pos (the template
	// base-offset mapping) and that the node sits at that position.
	expectPos := func(t *testing.T, ofp, tfp, got Pos) {
		t.Helper()
		require.Equal(t, ofp, tfp, "ofPos must equal tfPos")
		require.Equal(t, ofp, got)
	}

	// expectText asserts that arg is a text segment with the given (unescaped)
	// value, located at ofp in the source and tfp in the template file.
	expectText := func(t *testing.T, arg Expr, value string, ofp, tfp Pos) {
		t.Helper()
		s, ok := arg.(*StrLit)
		require.True(t, ok, "expected *StrLit, got %T", arg)
		require.Equal(t, value, s.Value())
		expectPos(t, ofp, tfp, s.Pos())
	}

	// expectRawText asserts a text segment of a raw (backtick) template, kept
	// verbatim as a RawStrLit (no unquoting), located at ofp/tfp.
	expectRawText := func(t *testing.T, arg Expr, literal string, ofp, tfp Pos) {
		t.Helper()
		s, ok := arg.(*RawStrLit)
		require.True(t, ok, "expected *RawStrLit, got %T", arg)
		require.Equal(t, literal, s.Value())
		expectPos(t, ofp, tfp, s.Pos())
	}

	// expectIdent asserts that arg is an identifier with the given name,
	// located at ofp in the source and tfp in the template file.
	expectIdent := func(t *testing.T, arg Expr, name string, ofp, tfp Pos) {
		t.Helper()
		id, ok := arg.(*IdentExpr)
		require.True(t, ok, "expected *IdentExpr, got %T", arg)
		require.Equal(t, name, id.Name)
		expectPos(t, ofp, tfp, id.Pos())
	}

	t.Run("hello {user}!", func(t *testing.T) {
		var ofPos, tfPos test.Pfn
		of := test.New(t, `x := #"hello {user}!"`).File().Expect(func(p test.Pfn) []Stmt {
			ofPos = p
			return stmts(
				SAssign(
					exprs(EIdent("x", p(1, 1))),
					exprs(&TemplateLit{
						TokenPos: p(1, 6),
						Value:    Str("hello {user}!", p(1, 7)),
					}), token.Define, p(1, 3)))
		}).File()

		tmpl := of.Stmts[0].(*AssignStmt).RHS[0].(*TemplateLit)
		tf, err := ParseTemplateString(tmpl.StringValue(), tmpl.StringValuePos())
		require.NoError(t, err)
		test.NewFile(t, tf).Expect(func(pos test.Pfn) []Stmt {
			tfPos = pos
			return stmts(
				SMixedText(pos(1, 1), "hello "),
				SCodeBegin(Lit("{", pos(1, 7)), false),
				SExpr(EIdent("user", pos(1, 8))),
				SCodeEnd(Lit("}", pos(1, 12)), false),
				SMixedText(pos(1, 13), "!"),
			)
		})

		userStartIndex := of.InputFile.DataIndex(ofPos(1, 15))
		require.Equal(t, string(of.InputFile.Data.Bytes()[userStartIndex:userStartIndex+4]), "user")

		tUserStartIndex := of.InputFile.DataIndex(ofPos(1, 8))
		require.Equal(t, string(tf.InputFile.Data.Bytes()[tUserStartIndex:tUserStartIndex+4]), "user")

		require.Equal(t, ofPos(1, 15), tfPos(1, 8))

		// Build turns the parsed template into a `str(...)` call whose
		// arguments carry the source positions of the original template, so
		// each argument maps back to its location in `of` (p) while still being
		// reachable through the template file (tp).
		p, tp, args := build(t, `x := #"hello {user}!"`)
		require.Len(t, args, 3)
		expectText(t, args[0], "hello ", p(1, 8), tp(1, 1)) // leading text
		expectIdent(t, args[1], "user", p(1, 15), tp(1, 8)) // interpolated `user`
		expectText(t, args[2], "!", p(1, 20), tp(1, 13))    // trailing text
	})

	// A lone interpolation with no surrounding text yields a single argument.
	t.Run("{name}", func(t *testing.T) {
		p, tp, args := build(t, `x := #"{name}"`)
		require.Len(t, args, 1)
		expectIdent(t, args[0], "name", p(1, 9), tp(1, 2))
	})

	// Adjacent interpolations produce one argument each, with no empty text
	// segments between them.
	t.Run("{a}{b}", func(t *testing.T) {
		p, tp, args := build(t, `x := #"{a}{b}"`)
		require.Len(t, args, 2)
		expectIdent(t, args[0], "a", p(1, 9), tp(1, 2))
		expectIdent(t, args[1], "b", p(1, 12), tp(1, 5))
	})

	// Text and variables interleaved: each piece maps back to its column in
	// the original source.
	t.Run("a={x}, b={y}!", func(t *testing.T) {
		p, tp, args := build(t, `x := #"a={x}, b={y}!"`)
		require.Len(t, args, 5)
		expectText(t, args[0], "a=", p(1, 8), tp(1, 1))
		expectIdent(t, args[1], "x", p(1, 11), tp(1, 4))
		expectText(t, args[2], ", b=", p(1, 13), tp(1, 6))
		expectIdent(t, args[3], "y", p(1, 18), tp(1, 11))
		expectText(t, args[4], "!", p(1, 20), tp(1, 13))
	})

	// An interpolated binary expression keeps the positions of its operands
	// and operator mapped to the original source.
	t.Run("sum={a + b}.", func(t *testing.T) {
		p, tp, args := build(t, `x := #"sum={a + b}."`)
		require.Len(t, args, 3)
		expectText(t, args[0], "sum=", p(1, 8), tp(1, 1))

		bin, ok := args[1].(*BinaryExpr)
		require.True(t, ok, "expected *BinaryExpr, got %T", args[1])
		require.Equal(t, token.Add, bin.Token)
		expectPos(t, p(1, 15), tp(1, 8), bin.TokenPos)
		expectIdent(t, bin.LHS, "a", p(1, 13), tp(1, 6))
		expectIdent(t, bin.RHS, "b", p(1, 17), tp(1, 10))

		expectText(t, args[2], ".", p(1, 19), tp(1, 12))
	})

	// An interpolated selector expression (`user.name`).
	t.Run("{user.name}", func(t *testing.T) {
		p, tp, args := build(t, `x := #"{user.name}"`)
		require.Len(t, args, 1)

		sel, ok := args[0].(*SelectorExpr)
		require.True(t, ok, "expected *SelectorExpr, got %T", args[0])
		expectPos(t, p(1, 9), tp(1, 2), sel.Pos())
		expectIdent(t, sel.X, "user", p(1, 9), tp(1, 2))
		// The selector key is carried as a string literal `name`.
		expectText(t, sel.Sel, "name", p(1, 14), tp(1, 7))
	})

	// An interpolated function call (`upper(name)`).
	t.Run("{upper(name)}", func(t *testing.T) {
		p, tp, args := build(t, `x := #"{upper(name)}"`)
		require.Len(t, args, 1)

		call, ok := args[0].(*CallExpr)
		require.True(t, ok, "expected *CallExpr, got %T", args[0])
		expectIdent(t, call.Func, "upper", p(1, 9), tp(1, 2))
		require.Len(t, call.Args.Values, 1)
		expectIdent(t, call.Args.Values[0], "name", p(1, 15), tp(1, 8))
	})

	// A multiline raw (backtick) template spanning two source lines. The
	// generated call is wrapped in ToRaw, text segments stay verbatim
	// (newlines preserved), and interpolations on the second line map to their
	// real line/column in both files.
	t.Run("multiline raw", func(t *testing.T) {
		p, tp, args := build(t, "x := #`Hello {name},\nwelcome to {place}!`")
		require.Len(t, args, 5)
		expectRawText(t, args[0], "Hello ", p(1, 8), tp(1, 1))
		expectIdent(t, args[1], "name", p(1, 15), tp(1, 8))
		expectRawText(t, args[2], ",\nwelcome to ", p(1, 20), tp(1, 13))
		expectIdent(t, args[3], "place", p(2, 13), tp(2, 13))
		expectRawText(t, args[4], "!", p(2, 19), tp(2, 19))
	})

	// A heredoc template (triple backticks). Value strips the surrounding
	// backticks, the opening line and the closing line, so the template content
	// sits on its own lines: the source is two lines further down than the
	// template file, but interpolation positions still map between them.
	t.Run("multiline raw with multiples quotes", func(t *testing.T) {
		p, tp, args := build(t, "x := #```\n\nHello {name},\nwelcome to {place}!\n\n```")
		require.Len(t, args, 5)
		expectRawText(t, args[0], "\nHello ", p(2, 1), tp(1, 1))
		expectIdent(t, args[1], "name", p(3, 8), tp(2, 8))
		expectRawText(t, args[2], ",\nwelcome to ", p(3, 13), tp(2, 13))
		expectIdent(t, args[3], "place", p(4, 13), tp(3, 13))
		expectRawText(t, args[4], "!\n", p(4, 19), tp(3, 19))
	})

	// An indented heredoc: Value strips the common leading indentation from the
	// rendered text (the text segments come out trimmed: "Hello ", "!", ...),
	// yet interpolation positions still map to their real, *indented* columns in
	// the original source -- `name` at column 16 and `place` at column 21, not
	// the trimmed columns 8 and 13. Interior positions are preserved because the
	// body is parsed untrimmed and only the text values are stripped.
	t.Run("indented heredoc preserves interior positions", func(t *testing.T) {
		p, tp, args := build(t, "x := #```\n        Hello {name},\n        welcome to {place}!\n        ```")
		require.Len(t, args, 5)
		expectRawText(t, args[0], "Hello ", p(2, 1), tp(1, 1))
		expectIdent(t, args[1], "name", p(2, 16), tp(1, 16))
		expectRawText(t, args[2], ",\nwelcome to ", p(2, 21), tp(1, 21))
		expectIdent(t, args[3], "place", p(3, 21), tp(2, 21))
		expectRawText(t, args[4], "!", p(3, 27), tp(2, 27))
	})
}

func TestParseChar(t *testing.T) {
	test.ExpectParseExpr(t, `'A'`, Char('A', 1))
	test.ExpectParseExpr(t, `'九'`, Char('九', 1))

	test.ExpectParseError(t, `''`)
	test.ExpectParseError(t, `'AB'`)
	test.ExpectParseError(t, `'A九'`)

	test.ExpectParseExpr(t, `'A'`, charAsStrLit("A", 1), test.OptParseCharAsString)
	test.ExpectParseExpr(t, `'九'`, charAsStrLit("九", 1), test.OptParseCharAsString)
	test.ExpectParseExpr(t, `'A九'`, charAsStrLit("A九", 1), test.OptParseCharAsString)
	test.ExpectParseExpr(t, "'a\\'b'", charAsStrLit(`a'b`, 1), test.OptParseCharAsString)
}

func TestParseSameOperator(t *testing.T) {
	// `===` (Same) and `!==` (NotSame) are comparison-level binary operators.
	test.ExpectParseString(t, `a === b`, `(a === b)`)
	test.ExpectParseString(t, `a !== b`, `(a !== b)`)
	test.ExpectParseString(t, `a === b === c`, `((a === b) === c)`)
	test.ExpectParseString(t, `a == b === c`, `((a == b) === c)`)
	test.ExpectParseString(t, `!a !== b`, `((!a) !== b)`)
}

func TestParseCondExpr(t *testing.T) {
	test.ExpectParse(t, "a ? b : c", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				ECond(
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
			SExpr(
				ECond(
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
		return stmts(SReturn(p(1, 1), nil))
	})

	test.ExpectParse(t, "1 || return", func(p pfn) []Stmt {
		return stmts(SExpr(EBinary(Int(1, p(1, 1)), EReturnExpr(p(1, 6), nil), token.LOr, p(1, 3))))
	})

	test.ExpectParseString(t, `var x; x || return`,
		"var x; (x || return)")

	test.ExpectParseString(t, `return 1`,
		"return 1")

	test.ExpectParse(t, "return = myvar", func(p pfn) []Stmt {
		return stmts(SReturnAssign(p(1, 1), EIdent("myvar", p(1, 10))))
	})

	test.ExpectParseError(t, "return = 1")
}

func TestParseForIn(t *testing.T) {
	test.New(t, "for a, b in values {}").
		String("for a, b in values {}").
		FormattedCode("for a, b in values {}")

	test.New(t, "for a, b in values {a+b}").
		String("for a, b in values { (a + b) }").
		FormattedCode("for a, b in values {\n\ta + b\n}")

	test.New(t, "for a, b in values {a+b;c;x++;}").
		String("for a, b in values { (a + b); c; x++ }").
		FormattedCode(`for a, b in values {
	a + b
	c
	x++
}`)

	test.New(t, "func z() { for a, b in values {a+b;c;x++;} }").
		String("func z() { for a, b in values { (a + b); c; x++ } }").
		FormattedCode(`func z() {
	for a, b in values {
		a + b
		c
		x++
	}
}`)

	test.ExpectParse(t, "for x in y {}", func(p pfn) []Stmt {
		return stmts(
			SForIn(
				EIdent("_", p(1, 5)),
				EIdent("x", p(1, 5)),
				EIdent("y", p(1, 10)),
				SBlock(p(1, 12), p(1, 13)),
				p(1, 1)))
	})

	test.ExpectParse(t, "for _ in y {}", func(p pfn) []Stmt {
		return stmts(
			SForIn(
				EIdent("_", p(1, 5)),
				EIdent("_", p(1, 5)),
				EIdent("y", p(1, 10)),
				SBlock(p(1, 12), p(1, 13)),
				p(1, 1)))
	})

	test.ExpectParse(t, "for x in [1, 2, 3] {}", func(p pfn) []Stmt {
		return stmts(
			SForIn(
				EIdent("_", p(1, 5)),
				EIdent("x", p(1, 5)),
				Array(
					p(1, 10), p(1, 18),
					Int(1, p(1, 11)),
					Int(2, p(1, 14)),
					Int(3, p(1, 17))),
				SBlock(p(1, 20), p(1, 21)),
				p(1, 1)))
	})

	test.ExpectParse(t, "for x, y in z {}", func(p pfn) []Stmt {
		return stmts(
			SForIn(
				EIdent("x", p(1, 5)),
				EIdent("y", p(1, 8)),
				EIdent("z", p(1, 13)),
				SBlock(p(1, 15), p(1, 16)),
				p(1, 1)))
	})

	test.ExpectParse(t, "for x, y in {k1: 1, k2: 2} {}", func(p pfn) []Stmt {
		return stmts(
			SForIn(
				EIdent("x", p(1, 5)),
				EIdent("y", p(1, 8)),
				EDict(
					p(1, 13), p(1, 26),
					EDictElement(
						EIdent("k1", p(1, 14)), p(1, 16), Int(1, p(1, 18))),
					EDictElement(
						EIdent("k2", p(1, 21)), p(1, 23), Int(2, p(1, 25)))),
				SBlock(p(1, 28), p(1, 29)),
				p(1, 1)))
	})

	test.ExpectParse(t, "for x in y {} else {}", func(p pfn) []Stmt {
		return stmts(
			SForIn(
				EIdent("_", p(1, 5)),
				EIdent("x", p(1, 5)),
				EIdent("y", p(1, 10)),
				SBlock(p(1, 12), p(1, 13)),
				p(1, 1),
				SBlock(p(1, 20), p(1, 21))))
	})

	test.ExpectParse(t, "for x in y begin x else 1 end", func(p pfn) []Stmt {
		return stmts(
			SForIn(
				EIdent("_", p(1, 5)),
				EIdent("x", p(1, 5)),
				EIdent("y", p(1, 10)),
				SBlockLit(
					Lit("begin", p(1, 12)), ast.Literal{},
					SExpr(
						EIdent("x", p(1, 18)),
					),
				),
				p(1, 1),
				SBlockLit(Lit("", p(1, 25)), Lit("end", p(1, 27)),
					SExpr(
						Int(1, p(1, 25)),
					),
				)))
	})

	test.ExpectParseString(t, "for x in y begin x else 1 end", "for _, x in y begin x else 1 end")

	test.ExpectParse(t, "for x in y {} else 1 end", func(p pfn) []Stmt {
		return stmts(
			SForIn(
				EIdent("_", p(1, 5)),
				EIdent("x", p(1, 5)),
				EIdent("y", p(1, 10)),
				SBlock(p(1, 12), p(1, 13)),
				p(1, 1),
				SBlockLit(Lit("", p(1, 20)), Lit("end", p(1, 22)),
					SExpr(
						Int(1, p(1, 20)),
					),
				)))
	})

	test.New(t, "for x in y begin end").
		String("for _, x in y begin end").
		Code("for x in y begin end")

	test.New(t, "for x in y begin 1 end").
		String("for _, x in y begin 1 end").
		Code("for x in y begin 1 end")

	test.New(t, "for x in y begin 1; end").
		String("for _, x in y begin 1 end").
		Code("for x in y begin 1 end")

	test.New(t, "for x in y begin else end").
		String("for _, x in y begin else end").
		Code("for x in y begin else end")

	test.New(t, "for x in y begin 1 else end").
		String("for _, x in y begin 1 else end").
		Code("for x in y begin 1 else end")

	test.ExpectParseString(t, "for x in y begin else end", "for _, x in y begin else end")

	test.New(t, "for x in y begin 1 else 2 end").
		String("for _, x in y begin 1 else 2 end").
		Code("for x in y begin 1 else 2 end")

	test.ExpectParseError(t, `for 1 in a {}`)
	test.ExpectParseError(t, `for "" in a {}`)
	test.ExpectParseError(t, `for k,2 in a {}`)
	test.ExpectParseError(t, `for 1,v in a {}`)
}

func TestParseFor(t *testing.T) {
	test.ExpectParse(t, "for {}", func(p pfn) []Stmt {
		return stmts(
			SFor(nil, nil, nil, SBlock(p(1, 5), p(1, 6)), p(1, 1)))
	})

	test.ExpectParse(t, "for a == 5 {}", func(p pfn) []Stmt {
		return stmts(
			SFor(
				nil,
				EBinary(
					EIdent("a", p(1, 5)),
					Int(5, p(1, 10)),
					token.Equal,
					p(1, 7)),
				nil,
				SBlock(p(1, 12), p(1, 13)),
				p(1, 1)))
	})

	test.ExpectParse(t, "for a := 0; a == 5;  {}", func(p pfn) []Stmt {
		return stmts(
			SFor(
				SAssign(
					exprs(EIdent("a", p(1, 5))),
					exprs(Int(0, p(1, 10))),
					token.Define, p(1, 7)),
				EBinary(
					EIdent("a", p(1, 13)),
					Int(5, p(1, 18)),
					token.Equal,
					p(1, 15)),
				nil,
				SBlock(p(1, 22), p(1, 23)),
				p(1, 1)))
	})

	test.ExpectParse(t, "for a := 0; a < 5; a++ {}", func(p pfn) []Stmt {
		return stmts(
			SFor(
				SAssign(
					exprs(EIdent("a", p(1, 5))),
					exprs(Int(0, p(1, 10))),
					token.Define, p(1, 7)),
				EBinary(
					EIdent("a", p(1, 13)),
					Int(5, p(1, 17)),
					token.Less,
					p(1, 15)),
				SIncDec(
					EIdent("a", p(1, 20)),
					token.Inc, p(1, 21)),
				SBlock(p(1, 24), p(1, 25)),
				p(1, 1)))
	})

	test.ExpectParse(t, "for ; a < 5; a++ {}", func(p pfn) []Stmt {
		return stmts(
			SFor(
				nil,
				EBinary(
					EIdent("a", p(1, 7)),
					Int(5, p(1, 11)),
					token.Less,
					p(1, 9)),
				SIncDec(
					EIdent("a", p(1, 14)),
					token.Inc, p(1, 15)),
				SBlock(p(1, 18), p(1, 19)),
				p(1, 1)))
	})

	test.ExpectParse(t, "for a := 0; ; a++ {}", func(p pfn) []Stmt {
		return stmts(
			SFor(
				SAssign(
					exprs(EIdent("a", p(1, 5))),
					exprs(Int(0, p(1, 10))),
					token.Define, p(1, 7)),
				nil,
				SIncDec(
					EIdent("a", p(1, 15)),
					token.Inc, p(1, 16)),
				SBlock(p(1, 19), p(1, 20)),
				p(1, 1)))
	})

	test.ExpectParse(t, "for a == 5 && b != 4 {}", func(p pfn) []Stmt {
		return stmts(
			SFor(
				nil,
				EBinary(
					EBinary(
						EIdent("a", p(1, 5)),
						Int(5, p(1, 10)),
						token.Equal,
						p(1, 7)),
					EBinary(
						EIdent("b", p(1, 15)),
						Int(4, p(1, 20)),
						token.NotEqual,
						p(1, 17)),
					token.LAnd,
					p(1, 12)),
				nil,
				SBlock(p(1, 22), p(1, 23)),
				p(1, 1)))
	})

	test.ExpectParse(t, `for { break }`, func(p pfn) []Stmt {
		return stmts(
			SFor(nil, nil, nil,
				SBlock(p(1, 5), p(1, 13),
					SBreak(p(1, 7)),
				),
				p(1, 1)),
		)
	})

	test.ExpectParse(t, `for { continue }`, func(p pfn) []Stmt {
		return stmts(
			SFor(nil, nil, nil,
				SBlock(p(1, 5), p(1, 16),
					SContinue(p(1, 7)),
				),
				p(1, 1)),
		)
	})

	test.ExpectParseString(t, `for begin continue end`, "for begin continue end")

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
			SAssign(
				exprs(
					EIdent("a", p(1, 1))),
				exprs(
					EClosure(
						NewFuncParams(p(1, 1), p(1, 5), p(1, 13),
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
			SAssign(
				exprs(
					EIdent("a", p(1, 1))),
				exprs(
					EClosure(
						NewFuncParams(p(1, 5), p(1, 5), p(1, 13),
							funcArgs(nil,
								EIdent("b", p(1, 6)),
								EIdent("c", p(1, 9)),
								EIdent("d", p(1, 12))),
						),
						p(1, 15),
						token.Lambda,
						EBlock(p(1, 18), p(1, 20),
							SExpr(EIdent("d", p(1, 19)))))),
				token.Assign,
				p(1, 3)))
	})

	test.New(t, "() => nil").
		String("() => nil").
		Code("() => nil").
		IndentedCode("() => nil").
		Stmts(func(p test.Pfn) []Stmt {
			return stmts(SExpr(EClosure(
				NewFuncParams(p(1, 1), p(1, 2)),
				p(1, 4), token.Lambda,
				LNil(p(1, 7)),
			)))
		})
}

func TestParseFunction(t *testing.T) {
	test.ExpectParseString(t, "func(){}", "func() {}")
	test.ExpectParseString(t, "func(a int){}", "func(a int) {}")
	test.ExpectParseString(t, "func(a int|bool|int){}", "func(a int|bool) {}")
	// a typed param keeps its ident and type on the same line; the type union
	// may still continue after a `|` on the next line.
	test.ExpectParseString(t, "func(a int|\n\tbool){}", "func(a int|bool) {}")
	test.ExpectParse(t, "func fn (b) { return d }", func(p pfn) []Stmt {
		return stmts(
			SFunc(
				EFunc(
					funcType(p(1, 5), EIdent("fn", p(1, 6)), p(1, 9), p(1, 11),
						EIdent("fn", p(1, 6)),
						funcArgs(nil,
							EIdent("b", p(1, 10))),
					),
					SBlock(p(1, 13), p(1, 24),
						SReturn(p(1, 15), EIdent("d", p(1, 22)))))))
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
			SFunc(
				EFunc(
					funcType(p(1, 1), nil, p(1, 5), p(1, 8),
						NamedArgs(
							nil,
							[]*TypedIdentExpr{
								ETypedIdent(EIdent("x", p(1, 7))),
							},
							[]Expr{nil}),
					),
					SBlock(p(1, 9), p(1, 10)))),
		)
	})
	test.ExpectParseString(t, "func(;x){}", "func(; x) {}")
	test.ExpectParse(t, "a = func(b, c, d; e=1, f=2, **g) { return d }", func(p pfn) []Stmt {
		return stmts(
			SAssign(
				exprs(
					EIdent("a", p(1, 1))),
				exprs(
					EFunc(
						funcType(p(1, 5), nil, p(1, 9), p(1, 32),
							funcArgs(nil,
								EIdent("b", p(1, 10)),
								EIdent("c", p(1, 13)),
								EIdent("d", p(1, 16))),
							NamedArgs(
								EIdent("g", p(1, 31)),
								[]*TypedIdentExpr{
									ETypedIdent(EIdent("e", p(1, 19))),
									ETypedIdent(EIdent("f", p(1, 24))),
								},
								[]Expr{
									Int(1, p(1, 21)),
									Int(2, p(1, 26)),
								}),
						),
						SBlock(p(1, 34), p(1, 45),
							SReturn(p(1, 36), EIdent("d", p(1, 43)))))),
				token.Assign,
				p(1, 3)))
	})
	test.ExpectParse(t, "a = func(*args) { return args }", func(p pfn) []Stmt {
		return stmts(
			SAssign(
				exprs(
					EIdent("a", p(1, 1))),
				exprs(
					EFunc(
						funcType(p(1, 5), nil, p(1, 9), p(1, 15),
							funcArgs(EIdent("args", p(1, 11)))),
						SBlock(p(1, 17), p(1, 31),
							SReturn(p(1, 19),
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
			SFunc(
				EFunc(
					funcType(p(1, 5), nil, p(1, 5), p(1, 16),
						funcArgs(nil,
							EIdent("n", p(1, 6)),
							EIdent("a", p(1, 8)),
							EIdent("b", p(1, 10))),
						NamedArgs(EIdent("na", p(1, 14)), nil, nil),
					),
					SBlock(p(1, 18), p(1, 19)))),
		)
	})

	test.ExpectParse(t, "func(n,a,b;x=1,**na) {}", func(p pfn) []Stmt {
		return stmts(
			SFunc(
				EFunc(
					funcType(p(1, 5), nil, p(1, 5), p(1, 20),
						funcArgs(nil,
							EIdent("n", p(1, 6)),
							EIdent("a", p(1, 8)),
							EIdent("b", p(1, 10))),
						NamedArgs(
							EIdent("na", p(1, 18)),
							[]*TypedIdentExpr{ETypedIdent(EIdent("x", p(1, 12)))},
							[]Expr{Int(1, p(1, 14))}),
					),
					SBlock(p(1, 22), p(1, 23)))),
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
			SFunc(
				EFuncBodyE(
					funcType(p(1, 5), EIdent("fn", p(1, 6)), p(1, 8), p(1, 10),
						funcArgs(nil,
							EIdent("n", p(1, 9)),
						),
					),
					p(1, 12),
					Int(1, p(1, 15)),
				),
			),
		)
	})

	test.ExpectParse(t, "func fn(n) => 1", func(p pfn) []Stmt {
		return stmts(
			SFunc(
				EFuncBodyE(
					funcType(p(1, 5), EIdent("fn", p(1, 6)), p(1, 8), p(1, 10),
						funcArgs(nil,
							EIdent("n", p(1, 9)),
						),
					),
					p(1, 12),
					Int(1, p(1, 15)),
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
			return stmts(SFunc(EFuncBodyE(
				funcType(p(1, 1), EIdent("x", p(1, 1)), p(1, 2), p(1, 3)),
				p(1, 5), LNil(p(1, 8)),
			)))
		})

	test.New(t, "(x(){nil})").
		String("(x() { nil })").
		Code("(x() {nil})").
		IndentedCode("(x() {\n\tnil\n})").
		Stmts(func(p test.Pfn) []Stmt {
			return stmts(SExpr(EParen(EFunc(
				funcType(p(1, 2), EIdent("x", p(1, 2)), p(1, 3), p(1, 4)),
				SBlock(p(1, 5), p(1, 9), SExpr(LNil(p(1, 6)))),
			), p(1, 1), p(1, 10))))
		})

	test.New(t, "x(){nil}").
		String("x() { nil }").
		Code("x() {nil}").
		IndentedCode("x() {\n\tnil\n}").
		Stmts(func(p test.Pfn) []Stmt {
			return stmts(SFunc(EFunc(
				funcType(p(1, 1), EIdent("x", p(1, 1)), p(1, 2), p(1, 3)),
				SBlock(p(1, 4), p(1, 8), SExpr(&NilLit{TokenPos: p(1, 5)})),
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
			return stmts(SFunc(EFunc(
				funcType(p(1, 1), EIdent("x", p(1, 1)), p(1, 2), p(1, 33),
					funcArgs(
						EIdent("args", p(1, 11)),
						ETypedIdent(EIdent("a", p(1, 3)), EType(EIdent("int", p(1, 5)))),
					),
					NamedArgs(EIdent("kwargs", p(1, 27)), []*TypedIdentExpr{
						ETypedIdent(EIdent("b", p(1, 17))),
						ETypedIdent(EIdent("c", p(1, 20))),
					}, Exprs{nil, Int(1, p(1, 22))}),
				),
				SBlock(p(1, 34), p(1, 38), SExpr(LNil(p(1, 35)))),
			)))
		})
	test.ExpectParse(t, "[func () {}]", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				Array(p(1, 1), p(1, 12), EFunc(funcType(p(1, 2), nil, p(1, 7), p(1, 8)), SBlock(p(1, 10), p(1, 11))))))
	})
	test.ExpectParse(t, "return [func () {}]", func(p pfn) []Stmt {
		return stmts(SReturn(p(1, 1), Array(p(1, 8), p(1, 19), EFunc(funcType(p(1, 9), nil, p(1, 14), p(1, 15)), SBlock(p(1, 17), p(1, 18))))))
	})
	test.New(t, "func () {}()").String("(func() {})()").Code("(func() {})()")
	test.New(t, "func x() {return 1}()").
		String("(func x() { return 1 })()").
		IndentedCode("(func x() {\n\treturn 1\n})()")

	test.ExpectParse(t, "func fn (b) { return d }", func(p pfn) []Stmt {
		return stmts(
			SFunc(
				EFunc(
					funcType(p(1, 4), EIdent("fn", p(1, 6)), p(1, 9), p(1, 11),
						funcArgs(nil,
							EIdent("b", p(1, 10))),
					),
					SBlock(p(1, 13), p(1, 24),
						SReturn(p(1, 15), EIdent("d", p(1, 22)))))))
	})

	test.ExpectParse(t, "func klass.fn (b) { return d }", func(p pfn) []Stmt {
		return stmts(
			SFunc(
				EFunc(
					funcType(p(1, 4), ESelector(EIdent("klass", p(1, 6)), Str("fn", p(1, 12))), p(1, 15), p(1, 17),
						funcArgs(nil,
							EIdent("b", p(1, 16))),
					),
					SBlock(p(1, 19), p(1, 30),
						SReturn(p(1, 21), EIdent("d", p(1, 28)))))))
	})

	test.ExpectParse(t, `func klass["fn"] (b) { return d }`, func(p pfn) []Stmt {
		return stmts(
			SFunc(
				EFunc(
					funcType(p(1, 4),
						EIndex(
							EIdent("klass", p(1, 6)),
							Str("fn", p(1, 12)),
							p(1, 11),
							p(1, 16),
						), p(1, 18), p(1, 20),
						funcArgs(nil,
							EIdent("b", p(1, 19))),
					),
					SBlock(p(1, 22), p(1, 33),
						SReturn(p(1, 24), EIdent("d", p(1, 31)))))))
	})

	test.New(t, `func klass.fn["x"][y()].z (b) { return d }`).
		String(`func klass.fn["x"][y()].z(b) { return d }`).
		Code(`func klass.fn["x"][y()].z(b) {return d}`).
		IndentedCode(`func klass.fn["x"][y()].z(b) {
	return d
}`).
		Stmts(func(p pfn) []Stmt {
			return stmts(
				SFunc(
					EFunc(
						funcType(p(1, 4),
							ESelector(
								EIndex(
									EIndex(
										ESelector(
											EIdent("klass", p(1, 6)),
											Str("fn", p(1, 12)),
										),
										Str("x", p(1, 15)),
										p(1, 14),
										p(1, 18),
									),
									ECall(EIdent("y", p(1, 20)), p(1, 21), p(1, 22)),
									p(1, 19),
									p(1, 23),
								),
								Str("z", p(1, 25)),
							),
							p(1, 27), p(1, 29),
							funcArgs(nil,
								EIdent("b", p(1, 28))),
						),
						SBlock(p(1, 31), p(1, 42),
							SReturn(p(1, 33), EIdent("d", p(1, 40)))))))
		})
}

func TestParseFunctionReturnType(t *testing.T) {
	// round-trip rendering: single, multiple, named, union and lambda forms.
	test.ExpectParseString(t, "func(a) <int> { return a }", "func(a) <int> { return a }")
	test.ExpectParseString(t, "func(a, b) <int, str> { return a }", "func(a, b) <int, str> { return a }")
	test.ExpectParseString(t, "func() <int> {}", "func() <int> {}")
	test.ExpectParseString(t, "func(a) <x int> => a", "func(a) <x int> => a")
	test.ExpectParseString(t, "func(a) <x int|bool> => a", "func(a) <x int|bool> => a")
	// insignificant whitespace / newlines around the return list.
	test.ExpectParseString(t, "func(a)<int>{return a}", "func(a) <int> { return a }")
	test.ExpectParseString(t, "func(a) <\n\tint,\n\tstr> => a", "func(a) <int, str> => a")
	// methods carry their own return types.
	test.ExpectParseString(t,
		"func f { (a) <int> { return a }\n(a, b) <int, str> { return b } }",
		"func f {(a) <int> { return a }; (a, b) <int, str> { return b }; }")

	// single return type, full AST.
	test.ExpectParse(t, "func(a) <int> { return a }", func(p pfn) []Stmt {
		return stmts(
			SFunc(
				EFunc(
					funcType(p(1, 1), nil, p(1, 5), p(1, 7),
						funcArgs(nil, EIdent("a", p(1, 6))),
						FuncReturn(ETypedIdent(EIdent("int", p(1, 10)))),
					),
					SBlock(p(1, 15), p(1, 26),
						SReturn(p(1, 17), EIdent("a", p(1, 24)))))))
	})

	// multiple, named return types, full AST.
	test.ExpectParse(t, "func(a) <x int, y str> => a", func(p pfn) []Stmt {
		return stmts(
			SFunc(
				EFuncBodyE(
					funcType(p(1, 1), nil, p(1, 5), p(1, 7),
						funcArgs(nil, EIdent("a", p(1, 6))),
						FuncReturn(
							ETypedIdent(EIdent("x", p(1, 10)), EType(EIdent("int", p(1, 12)))),
							ETypedIdent(EIdent("y", p(1, 17)), EType(EIdent("str", p(1, 19)))),
						),
					),
					p(1, 24),
					EIdent("a", p(1, 27)))))
	})
}

func TestParseProperty(t *testing.T) {
	// single-accessor form: prop name(params) {body} (no surrounding braces).
	test.ExpectParseString(t, "prop x(n int) { v = n }", "prop x(n int) { v = n }")
	test.ExpectParseString(t, "prop x() { return v }", "prop x() { return v }")
	test.ExpectParseString(t, "prop x() => v", "prop x() => v")

	// multi-accessor form: prop name { (params) {body} ... } shares the
	// func-with-methods body syntax.
	test.ExpectParseString(t,
		"prop x { () { return v }\n(n int) { v = n } }",
		"prop x {() { return v }; (n int) { v = n }; }")

	// accessors carry their own return types.
	test.ExpectParseString(t,
		"prop x { () <int> { return v }\n(n int) { v = n } }",
		"prop x {() <int> { return v }; (n int) { v = n }; }")

	// lambda bodies and a typed setter.
	test.ExpectParseString(t,
		"prop x { () => v\n(n int) => n }",
		"prop x {() => v; (n int) => n; }")

	// selector and index names.
	test.ExpectParseString(t, "prop obj.x { () => v }", "prop obj.x {() => v; }")
	test.ExpectParseString(t, `prop obj["x"] { () => v }`, `prop obj["x"] {() => v; }`)

	// as an expression value (anonymous and named).
	test.ExpectParseString(t, "const p = prop y { () => v }", "const p = prop y {() => v; }")
	test.ExpectParseString(t, "const p = prop { () => v }", "const p = prop {() => v; }")

	// full AST for the single-accessor getter form.
	test.ExpectParse(t, "prop x() { return v }", func(p pfn) []Stmt {
		return stmts(&PropStmt{PropExpr: PropExpr{
			PropToken: TokenLit{Token: token.Prop, Literal: "prop", Pos: p(1, 1)},
			NameExpr:  EIdent("x", p(1, 6)),
			RBrace:    p(1, 22),
			Methods: []*FuncMethod{
				{
					Params: FuncParams{Args: funcArgs(nil), LParen: p(1, 7), RParen: p(1, 8)},
					Body:   SBlock(p(1, 10), p(1, 21), SReturn(p(1, 12), EIdent("v", p(1, 19)))),
				},
			},
		}})
	})
}

func TestParseShorthandFuncReturnType(t *testing.T) {
	// name(params) <ret> {body} shorthand.
	test.ExpectParseString(t, "foo(a) <int> { return a }", "foo(a) <int> { return a }")
	test.ExpectParseString(t, "foo(a, b) <int, str> { return a }", "foo(a, b) <int, str> { return a }")
	test.ExpectParseString(t, "foo(a) <x int|bool> => a", "foo(a) <x int|bool> => a")
	test.ExpectParseString(t, "foo(a)<int>{return a}", "foo(a) <int> { return a }")

	// the return-type list must not be confused with a comparison operator.
	test.ExpectParseString(t, "foo(1) < 2", "(foo(1) < 2)")
	test.ExpectParseString(t, "foo(a) < b > c", "((foo(a) < b) > c)")

	// dict-element funcs.
	test.ExpectParseString(t, "d := {g(a, b) <int, str> { return a }}", "d := {g(a, b) <int, str> { return a }}")
	test.ExpectParseString(t, "[a (x) <int> { return x }]", "[a(x) <int> { return x }]")
	test.ExpectParseString(t, "return (;f(a) <int> { return a })", "return (;f(a) <int> { return a })")

	// full AST for the named shorthand (FuncExpr wrapped in a FuncStmt).
	test.ExpectParse(t, "foo(a) <int> { return a }", func(p pfn) []Stmt {
		return stmts(
			SFunc(
				EFunc(
					funcType(NoPos, EIdent("foo", p(1, 1)), p(1, 4), p(1, 6),
						funcArgs(nil, EIdent("a", p(1, 5))),
						FuncReturn(ETypedIdent(EIdent("int", p(1, 9)))),
					),
					SBlock(p(1, 14), p(1, 25),
						SReturn(p(1, 16), EIdent("a", p(1, 23)))))))
	})
}

func TestParseClosureFuncReturnType(t *testing.T) {
	// (params) <ret> => body closure forms.
	test.ExpectParseString(t, "x := (a) <int> => a", "x := (a) <int> => a")
	test.ExpectParseString(t, "x := (a, b) <int, str> => [a, b]", "x := (a, b) <int, str> => [a, b]")
	test.ExpectParseString(t, "x := (a) <x int|bool> => a", "x := (a) <x int|bool> => a")
	test.ExpectParseString(t, "x := (a) <int> => {a}", "x := (a) <int> => { a }")

	// the return-type list must not be confused with a comparison operator.
	test.ExpectParseString(t, "x := (a) < b", "x := ((a) < b)")

	// dict-element closure (':' body).
	test.ExpectParseString(t, "d := {f(a) <int> : a}", "d := {f(a) <int> : a}")

	// full AST for a closure with a return type.
	test.ExpectParse(t, "a = (b) <int> => b", func(p pfn) []Stmt {
		cl := EClosure(
			NewFuncParams(p(1, 5), p(1, 7), funcArgs(nil, EIdent("b", p(1, 6)))),
			p(1, 15), token.Lambda, EIdent("b", p(1, 18)))
		cl.Return = FuncReturn(ETypedIdent(EIdent("int", p(1, 10))))
		return stmts(
			SAssign(
				exprs(EIdent("a", p(1, 1))),
				exprs(cl),
				token.Assign,
				p(1, 3)))
	})
}

func TestParseMethod(t *testing.T) {
	test.ExpectParse(t, "met fn (b) { return d }", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				EMethod(EFunc(
					funcType(p(1, 4), EIdent("fn", p(1, 5)), p(1, 8), p(1, 10),
						funcArgs(nil,
							EIdent("b", p(1, 9))),
					),
					SBlock(p(1, 12), p(1, 23),
						SReturn(p(1, 14), EIdent("d", p(1, 21))))))))
	})

	test.ExpectParse(t, "met klass.fn (b) { return d }", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				EMethod(EFunc(
					funcType(p(1, 4), ESelector(EIdent("klass", p(1, 5)), Str("fn", p(1, 11))), p(1, 14), p(1, 16),
						funcArgs(nil,
							EIdent("b", p(1, 15))),
					),
					SBlock(p(1, 18), p(1, 29),
						SReturn(p(1, 20), EIdent("d", p(1, 27))))))))
	})

	test.ExpectParse(t, `met klass["fn"] (b) { return d }`, func(p pfn) []Stmt {
		return stmts(
			SExpr(
				EMethod(
					EFunc(
						funcType(p(1, 4),
							EIndex(
								EIdent("klass", p(1, 5)),
								Str("fn", p(1, 11)),
								p(1, 10),
								p(1, 15),
							), p(1, 17), p(1, 19),
							funcArgs(nil,
								EIdent("b", p(1, 18))),
						),
						SBlock(p(1, 21), p(1, 32),
							SReturn(p(1, 23), EIdent("d", p(1, 30))))))))
	})

	test.New(t, `met klass.fn["x"][y()].z (b) { return d }`).
		String(`met klass.fn["x"][y()].z(b) { return d }`).
		Code(`met klass.fn["x"][y()].z(b) {return d}`).
		IndentedCode(`met klass.fn["x"][y()].z(b) {
	return d
}`).
		Stmts(func(p pfn) []Stmt {
			return stmts(
				SExpr(
					EMethod(EFunc(
						funcType(p(1, 4),
							ESelector(
								EIndex(
									EIndex(
										ESelector(
											EIdent("klass", p(1, 5)),
											Str("fn", p(1, 11)),
										),
										Str("x", p(1, 14)),
										p(1, 13),
										p(1, 17),
									),
									ECall(EIdent("y", p(1, 19)), p(1, 20), p(1, 21)),
									p(1, 18),
									p(1, 22),
								),
								Str("z", p(1, 24)),
							),
							p(1, 26), p(1, 28),
							funcArgs(nil,
								EIdent("b", p(1, 27))),
						),
						SBlock(p(1, 30), p(1, 41),
							SReturn(p(1, 32), EIdent("d", p(1, 39))))))))
		})
}

func TestParsePtr(t *testing.T) {
	test.New(t, "&x").String("&(x)").Code("&x")
}

func TestParseSpecialKeywords(t *testing.T) {
	test.New(t, "@main;x.@main").Stmts(func(p pfn) []Stmt {
		return stmts(
			SExpr(&IsMainLit{TokenPos: p(1, 1)}),
			SExpr(ESelector(EIdent("x", p(1, 7)), Str("@main", p(1, 9)))),
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
			SAssign(
				exprs(
					EIdent("a", p(1, 1))),
				exprs(
					EFunc(
						funcType(p(1, 5), nil, p(1, 9), p(1, 18),
							funcArgs(EIdent("z", p(1, 17)),
								EIdent("x", p(1, 10)),
								EIdent("y", p(1, 13)))),
						SBlock(p(1, 20), p(1, 31),
							SReturn(p(1, 22),
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
			SIf(
				nil,
				EUnary(EIdent("a", p(1, 4)),
					token.Null,
					p(1, 6)),
				SBlock(
					p(1, 13), p(1, 14)),
				nil,
				p(1, 1)))
	})

	test.ExpectParse(t, "if a != nil {}", func(p pfn) []Stmt {
		return stmts(
			SIf(
				nil,
				EUnary(EIdent("a", p(1, 4)),
					token.NotNull,
					p(1, 6)),
				SBlock(
					p(1, 13), p(1, 14)),
				nil,
				p(1, 1)))
	})

	test.ExpectParse(t, "if a == 5 {}", func(p pfn) []Stmt {
		return stmts(
			SIf(
				nil,
				EBinary(
					EIdent("a", p(1, 4)),
					Int(5, p(1, 9)),
					token.Equal,
					p(1, 6)),
				SBlock(
					p(1, 11), p(1, 12)),
				nil,
				p(1, 1)))
	})

	test.ExpectParse(t, "if a == 5 && b != 3 {}", func(p pfn) []Stmt {
		return stmts(
			SIf(
				nil,
				EBinary(
					EBinary(
						EIdent("a", p(1, 4)),
						Int(5, p(1, 9)),
						token.Equal,
						p(1, 6)),
					EBinary(
						EIdent("b", p(1, 14)),
						Int(3, p(1, 19)),
						token.NotEqual,
						p(1, 16)),
					token.LAnd,
					p(1, 11)),
				SBlock(
					p(1, 21), p(1, 22)),
				nil,
				p(1, 1)))
	})

	test.ExpectParse(t, "if a == 5 { a = 3; a = 1 }", func(p pfn) []Stmt {
		return stmts(
			SIf(
				nil,
				EBinary(
					EIdent("a", p(1, 4)),
					Int(5, p(1, 9)),
					token.Equal,
					p(1, 6)),
				SBlock(
					p(1, 11), p(1, 26),
					SAssign(
						exprs(EIdent("a", p(1, 13))),
						exprs(Int(3, p(1, 17))),
						token.Assign,
						p(1, 15)),
					SAssign(
						exprs(EIdent("a", p(1, 20))),
						exprs(Int(1, p(1, 24))),
						token.Assign,
						p(1, 22))),
				nil,
				p(1, 1)))
	})

	test.ExpectParse(t, "if a == 5 { a = 3; a = 1 } else { a = 2; a = 4 }",
		func(p pfn) []Stmt {
			return stmts(
				SIf(
					nil,
					EBinary(
						EIdent("a", p(1, 4)),
						Int(5, p(1, 9)),
						token.Equal,
						p(1, 6)),
					SBlock(
						p(1, 11), p(1, 26),
						SAssign(
							exprs(EIdent("a", p(1, 13))),
							exprs(Int(3, p(1, 17))),
							token.Assign,
							p(1, 15)),
						SAssign(
							exprs(EIdent("a", p(1, 20))),
							exprs(Int(1, p(1, 24))),
							token.Assign,
							p(1, 22))),
					SBlock(
						p(1, 33), p(1, 48),
						SAssign(
							exprs(EIdent("a", p(1, 35))),
							exprs(Int(2, p(1, 39))),
							token.Assign,
							p(1, 37)),
						SAssign(
							exprs(EIdent("a", p(1, 42))),
							exprs(Int(4, p(1, 46))),
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
			SIf(
				nil,
				EBinary(
					EIdent("a", p(2, 4)),
					Int(5, p(2, 9)),
					token.Equal,
					p(2, 6)),
				SBlock(
					p(2, 11), p(5, 1),
					SAssign(
						exprs(EIdent("b", p(3, 2))),
						exprs(Int(3, p(3, 6))),
						token.Assign,
						p(3, 4)),
					SAssign(
						exprs(EIdent("c", p(4, 2))),
						exprs(Int(1, p(4, 6))),
						token.Assign,
						p(4, 4))),
				SIf(
					nil,
					EBinary(
						EIdent("d", p(5, 11)),
						Int(3, p(5, 16)),
						token.Equal,
						p(5, 13)),
					SBlock(
						p(5, 18), p(8, 1),
						SAssign(
							exprs(EIdent("e", p(6, 2))),
							exprs(Int(8, p(6, 6))),
							token.Assign,
							p(6, 4)),
						SAssign(
							exprs(EIdent("f", p(7, 2))),
							exprs(Int(3, p(7, 6))),
							token.Assign,
							p(7, 4))),
					SBlock(
						p(8, 8), p(11, 1),
						SAssign(
							exprs(EIdent("g", p(9, 2))),
							exprs(Int(2, p(9, 6))),
							token.Assign,
							p(9, 4)),
						SAssign(
							exprs(EIdent("h", p(10, 2))),
							exprs(Int(4, p(10, 6))),
							token.Assign,
							p(10, 4))),
					p(5, 8)),
				p(2, 1)))
	})

	test.ExpectParse(t, "if a := 3; a < b {}", func(p pfn) []Stmt {
		return stmts(
			SIf(
				SAssign(
					exprs(EIdent("a", p(1, 4))),
					exprs(Int(3, p(1, 9))),
					token.Define, p(1, 6)),
				EBinary(
					EIdent("a", p(1, 12)),
					EIdent("b", p(1, 16)),
					token.Less, p(1, 14)),
				SBlock(
					p(1, 18), p(1, 19)),
				nil,
				p(1, 1)))
	})

	test.ExpectParse(t, "if a++; a < b {}", func(p pfn) []Stmt {
		return stmts(
			SIf(
				SIncDec(EIdent("a", p(1, 4)), token.Inc, p(1, 5)),
				EBinary(
					EIdent("a", p(1, 9)),
					EIdent("b", p(1, 13)),
					token.Less, p(1, 11)),
				SBlock(
					p(1, 15), p(1, 16)),
				nil,
				p(1, 1)))
	})

	test.ExpectParseString(t, "if a begin end", "if a begin end")
	test.ExpectParseString(t, "if a begin b end", "if a begin b end")
	test.ExpectParseString(t, "if true; a begin b end", "if true; a begin b end")
	test.ExpectParseString(t, "if a begin b else c end", "if a begin b else c end")
	test.ExpectParseString(t, "if a begin b; else c end", "if a begin b else c end")
	test.ExpectParseString(t, "if a begin b else if 1 begin 2 else c end", "if a begin b else if 1 begin 2 else c end")
	test.ExpectParseString(t, "if a begin b; else if 1 begin 2; else c end", "if a begin b else if 1 begin 2 else c end")

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
			SAssign(
				exprs(EIdent("a", p(1, 1))),
				exprs(EImport(p(1, 6), "mod1", p(1, 12), p(1, 19), p(1, 13))),
				token.Define, p(1, 3)))
	})

	test.ExpectParse(t, `a := import("mod1", 1; x=2)`, func(p pfn) []Stmt {
		return stmts(
			SAssign(
				exprs(EIdent("a", p(1, 1))),
				exprs(WithCallArgs(EImport(p(1, 6), "mod1", p(1, 12), p(1, 27), p(1, 13)), func(args *CallArgs) {
					args.Args.AppendValues(Int(1, p(1, 21)))
					args.NamedArgs.Append(ENamedArg().Ident(EIdent("x", p(1, 24))).Build(), Int(2, p(1, 26)))
				})),
				token.Define, p(1, 3)))
	})

	test.ExpectParse(t, `import("mod1").var1`, func(p pfn) []Stmt {
		return stmts(
			SExpr(
				ESelector(
					EImport(p(1, 1), "mod1", p(1, 7), p(1, 14), p(1, 8)),
					stringLit("var1", p(1, 16)))))
	})

	test.ExpectParse(t, `import("mod1").func1()`, func(p pfn) []Stmt {
		return stmts(
			SExpr(
				ECall(
					ESelector(
						EImport(p(1, 1), "mod1", p(1, 7), p(1, 14), p(1, 8)),
						stringLit("func1", p(1, 16))),
					p(1, 21), p(1, 22), NoPos)))
	})

	test.ExpectParse(t, `for x, y in import("mod1").v {}`, func(p pfn) []Stmt {
		return stmts(
			SForIn(
				EIdent("x", p(1, 5)),
				EIdent("y", p(1, 8)),
				ESelector(EImport(p(1, 13), "mod1", p(1, 19), p(1, 26), p(1, 20)),
					stringLit("v", p(1, 28))),
				SBlock(p(1, 30), p(1, 31)),
				p(1, 1)))
	})

	test.ExpectParseError(t, `import(1)`)
	test.ExpectParseError(t, `import('a')`)
}

func TestParseEmbed(t *testing.T) {
	test.ExpectParse(t, `a := embed("file";sources=[])`, func(p pfn) []Stmt {
		return stmts(
			SAssign(
				exprs(EIdent("a", p(1, 1))),
				exprs(EEmbed(NewCallArgs(p(1, 11), p(1, 29)).Arg(Str("file", p(1, 12))).NamedValue(EIdent("sources", p(1, 19)), Array(p(1, 27), p(1, 28))), p(1, 6))),
				token.Define, p(1, 3)))
	})

	test.ExpectParse(t, `a := embed("file")`, func(p pfn) []Stmt {
		return stmts(
			SAssign(
				exprs(EIdent("a", p(1, 1))),
				exprs(EEmbed(NewCallArgs(p(1, 11), p(1, 18)).Arg(Str("file", p(1, 12))), p(1, 6))),
				token.Define, p(1, 3)))
	})

	test.ExpectParse(t, `embed("file").var1`, func(p pfn) []Stmt {
		return stmts(
			SExpr(
				ESelector(
					EEmbed(NewCallArgs(p(1, 6), p(1, 13)).Arg(Str("file", p(1, 7))), p(1, 1)),
					stringLit("var1", p(1, 15)))))
	})

	test.ExpectParse(t, `for x, y in embed("file") {}`, func(p pfn) []Stmt {
		return stmts(
			SForIn(
				EIdent("x", p(1, 5)),
				EIdent("y", p(1, 8)),
				EEmbed(NewCallArgs(p(1, 18), p(1, 25)).Arg(Str("file", p(1, 19))), p(1, 13)),
				SBlock(p(1, 27), p(1, 28)),
				p(1, 1)))
	})

	test.ExpectParse(t, `embed("dir";sources=["a","b"])`, func(p pfn) []Stmt {
		return stmts(
			SExpr(
				EEmbed(NewCallArgs(p(1, 6), p(1, 30)).Arg(Str("dir", p(1, 7))).NamedValue(EIdent("sources", p(1, 13)), Array(p(1, 21), p(1, 29), Str("a", p(1, 22)), Str("b", p(1, 26)))), p(1, 1))))
	})

	test.ExpectParse(t, `embed("file";includes=["*.go"])`, func(p pfn) []Stmt {
		return stmts(
			SExpr(
				EEmbed(NewCallArgs(p(1, 6), p(1, 31)).Arg(Str("file", p(1, 7))).NamedValue(EIdent("includes", p(1, 14)), Array(p(1, 23), p(1, 30), Str("*.go", p(1, 24)))), p(1, 1))))
	})

	test.ExpectParse(t, `embed("dir";tree)`, func(p pfn) []Stmt {
		return stmts(
			SExpr(
				EEmbed(NewCallArgs(p(1, 6), p(1, 17)).Arg(Str("dir", p(1, 7))).NamedFlag(EIdent("tree", p(1, 13))), p(1, 1))))
	})

	test.ExpectParse(t, `embed("dir";includes_re=["[.go]"])`, func(p pfn) []Stmt {
		return stmts(
			SExpr(
				EEmbed(NewCallArgs(p(1, 6), p(1, 34)).Arg(Str("dir", p(1, 7))).NamedValue(EIdent("includes_re", p(1, 13)), Array(p(1, 25), p(1, 33), Str("[.go]", p(1, 26)))), p(1, 1))))
	})

	test.ExpectParse(t, `embed("dir";excludes_re=["[.go]"])`, func(p pfn) []Stmt {
		return stmts(
			SExpr(
				EEmbed(NewCallArgs(p(1, 6), p(1, 34)).Arg(Str("dir", p(1, 7))).NamedValue(EIdent("excludes_re", p(1, 13)), Array(p(1, 25), p(1, 33), Str("[.go]", p(1, 26)))), p(1, 1))))
	})

	test.ExpectParse(t, `embed("dir";config_file="cfg.yaml")`, func(p pfn) []Stmt {
		return stmts(
			SExpr(
				EEmbed(NewCallArgs(p(1, 6), p(1, 35)).Arg(Str("dir", p(1, 7))).NamedValue(EIdent("config_file", p(1, 13)), Str("cfg.yaml", p(1, 25))), p(1, 1))))
	})
	test.ExpectParseError(t, `embed(1)`)
	test.ExpectParseError(t, `embed('a')`)
	test.ExpectParseError(t, `embed()`)
	test.ExpectParseError(t, `embed("a","b")`)
	test.ExpectParseError(t, `embed("a";x=1)`)
	test.ExpectParseError(t, `embed("a";tree=1)`)
	test.ExpectParseError(t, `embed("a";sources=1)`)
	test.ExpectParseError(t, `embed("a";includes_re=1)`)
	test.ExpectParseError(t, `embed("a";excludes_re=1)`)
	test.ExpectParseError(t, `embed("a";config_file=1)`)
	test.ExpectParseError(t, `embed("a";config_file)`)
}

func TestParseIndex(t *testing.T) {
	test.ExpectParse(t, "[1, 2, 3][1]", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				EIndex(
					Array(p(1, 1), p(1, 9),
						Int(1, p(1, 2)),
						Int(2, p(1, 5)),
						Int(3, p(1, 8))),
					Int(1, p(1, 11)),
					p(1, 10), p(1, 12))))
	})

	test.ExpectParse(t, "[1, 2, 3][5 - a]", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				EIndex(
					Array(p(1, 1), p(1, 9),
						Int(1, p(1, 2)),
						Int(2, p(1, 5)),
						Int(3, p(1, 8))),
					EBinary(
						Int(5, p(1, 11)),
						EIdent("a", p(1, 15)),
						token.Sub,
						p(1, 13)),
					p(1, 10), p(1, 16))))
	})

	test.ExpectParse(t, "[1, 2, 3][5 : a]", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				ESlice(
					Array(p(1, 1), p(1, 9),
						Int(1, p(1, 2)),
						Int(2, p(1, 5)),
						Int(3, p(1, 8))),
					Int(5, p(1, 11)),
					EIdent("a", p(1, 15)),
					p(1, 10), p(1, 16))))
	})

	test.ExpectParse(t, "[1, 2, 3][a + 3 : b - 8]", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				ESlice(
					Array(p(1, 1), p(1, 9),
						Int(1, p(1, 2)),
						Int(2, p(1, 5)),
						Int(3, p(1, 8))),
					EBinary(
						EIdent("a", p(1, 11)),
						Int(3, p(1, 15)),
						token.Add,
						p(1, 13)),
					EBinary(
						EIdent("b", p(1, 19)),
						Int(8, p(1, 23)),
						token.Sub,
						p(1, 21)),
					p(1, 10), p(1, 24))))
	})

	test.ExpectParse(t, `({a: 1, b: 2})["b"]`, func(p pfn) []Stmt {
		return stmts(
			SExpr(
				EIndex(
					EParen(
						EDict(p(1, 2), p(1, 13),
							EDictElement(
								EIdent("a", p(1, 3)), p(1, 4), Int(1, p(1, 6))),
							EDictElement(
								EIdent("b", p(1, 9)), p(1, 10), Int(2, p(1, 12)))),
						p(1, 1), p(1, 14),
					),
					stringLit("b", p(1, 16)),
					p(1, 15), p(1, 19))))
	})

	test.ExpectParse(t, `({a: 1, b: 2})[a + b]`, func(p pfn) []Stmt {
		return stmts(
			SExpr(
				EIndex(
					EParen(
						EDict(p(1, 2), p(1, 13),
							EDictElement(
								EIdent("a", p(1, 3)), p(1, 4), Int(1, p(1, 6))),
							EDictElement(
								EIdent("b", p(1, 9)), p(1, 10), Int(2, p(1, 12)))),
						p(1, 1), p(1, 14),
					),
					EBinary(
						EIdent("a", p(1, 16)),
						EIdent("b", p(1, 20)),
						token.Add,
						p(1, 18)),
					p(1, 15), p(1, 21))))
	})
}

func TestParseWith(t *testing.T) {
	// Statement forms.
	test.ExpectParseString(t, "with r { x() }", "with r { x() }")
	test.ExpectParseString(t, "with r as f { x() }", "with r as f { x() }")
	test.ExpectParseString(t, "with x = mk() { y() }", "with x = mk() { y() }")
	test.ExpectParseString(t, "with x := mk() { y() }", "with x := mk() { y() }")
	// The resource may be a call (its body `{` is not consumed as a func def).
	test.ExpectParseString(t, "with open(p) { use() }", "with open(p) { use() }")
	// Expression form (yields a value).
	test.ExpectParseString(t, "v := with r: r.read()", "v := (with r: r.read())")
	test.ExpectParseString(t, "v := with mk() as f: f.read()", "v := (with mk() as f: f.read())")
}

func TestParseAin(t *testing.T) {
	// `a ain b` parses as a binary operator.
	test.ExpectParse(t, "a ain b", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				EBinary(
					EIdent("a", p(1, 1)),
					EIdent("b", p(1, 7)),
					token.Ain,
					p(1, 3))))
	})

	// `ain` has comparison precedence and is left-associative, like `in` / `==`,
	// so `a ain b == c` groups as `(a ain b) == c`.
	test.ExpectParse(t, "a ain b == c", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				EBinary(
					EBinary(
						EIdent("a", p(1, 1)),
						EIdent("b", p(1, 7)),
						token.Ain,
						p(1, 3)),
					EIdent("c", p(1, 12)),
					token.Equal,
					p(1, 9))))
	})
}

func TestParseLogical(t *testing.T) {
	test.ExpectParse(t, "2 ** 3", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				EBinary(
					Int(2, p(1, 1)),
					Int(3, p(1, 6)),
					token.Pow,
					p(1, 3))))
	})

	test.ExpectParse(t, "a && 5 || true", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				EBinary(
					EBinary(
						EIdent("a", p(1, 1)),
						Int(5, p(1, 6)),
						token.LAnd,
						p(1, 3)),
					Bool(true, p(1, 11)),
					token.LOr,
					p(1, 8))))
	})

	test.ExpectParse(t, "a || 5 && true", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				EBinary(
					EIdent("a", p(1, 1)),
					EBinary(
						Int(5, p(1, 6)),
						Bool(true, p(1, 11)),
						token.LAnd,
						p(1, 8)),
					token.LOr,
					p(1, 3))))
	})

	test.ExpectParse(t, "a && (5 || true)", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				EBinary(
					EIdent("a", p(1, 1)),
					EParen(
						EBinary(
							Int(5, p(1, 7)),
							Bool(true, p(1, 12)),
							token.LOr,
							p(1, 9)),
						p(1, 6), p(1, 16)),
					token.LAnd,
					p(1, 3))))
	})
}

func TestParseOrExpr(t *testing.T) {
	// structural check incl. positions
	test.ExpectParse(t, "a or b", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				&OrExpr{
					Expr:     EIdent("a", p(1, 1)),
					Fallback: EIdent("b", p(1, 6)),
					OrPos:    p(1, 3),
				}))
	})

	// round-trip / precedence checks
	test.ExpectParseString(t, "x() or 2", "x() or 2")
	// `or` has the lowest precedence: `1 + x or 2` => `(1 + x) or 2`
	test.ExpectParseString(t, "1 + x or 2", "(1 + x) or 2")
	// parentheses scope the `or` to the inner expression
	test.ExpectParseString(t, "1 + (x() or 2)", "(1 + (x() or 2))")
	// left associative
	test.ExpectParseString(t, "a or b or c", "a or b or c")
	// fallback may reference $err
	test.ExpectParseString(t, "x() or error($err)", "x() or error($err)")
	// `or` remains usable as a normal identifier outside infix position
	test.ExpectParseString(t, "or := 1", "or := 1")
}

func TestParseRegexLit(t *testing.T) {
	test.ExpectParseString(t, `r := /ab+/`, `r := /ab+/`)
	test.ExpectParseString(t, `r := /a+/p`, `r := /a+/p`)
	// escapes and char classes
	test.ExpectParseString(t, `r := /[0-9]+\/[0-9]+/`, `r := /[0-9]+\/[0-9]+/`)
	// division is unaffected (after a value, `/` is the operator)
	test.ExpectParseString(t, `x := 10 / 2`, `x := (10 / 2)`)
	test.ExpectParseString(t, `x := a / b / c`, `x := ((a / b) / c)`)
	// regex as a call argument / in operand position
	test.ExpectParseString(t, `f(/ab/)`, `f(/ab/)`)
}

func TestParseBytesLit(t *testing.T) {
	// hex and raw forms round-trip with the prefix preserved
	test.ExpectParseString(t, `x := h"ffccf1c2"`, `x := h"ffccf1c2"`)
	test.ExpectParseString(t, `x := b"Hello"`, `x := b"Hello"`)
	// raw string body
	test.ExpectParseString(t, "x := b`raw`", "x := b`raw`")
	// in operand position (call argument)
	test.ExpectParseString(t, `f(b"x")`, `f(b"x")`)
	// the prefix must be glued to the delimiter: a space breaks the literal
	test.ExpectParseError(t, `b "x"`)
}

func TestParseCodeStr(t *testing.T) {
	// inline form round-trips verbatim
	test.ExpectParseString(t, `x := code a + b end`, `x := code a + b end`)
	// in operand position (call argument)
	test.ExpectParseString(t, `f(code a end)`, `f(code a end)`)
	// block form round-trips verbatim through Code() (literal kept as-is)
	test.New(t, "x := code\n    a := 1\nend").Code("x := code\n    a := 1\nend")
	// a `code` identifier with no fence stays an identifier
	test.ExpectParseString(t, `code = 1`, `code = 1`)
}

func TestParseDurationLit(t *testing.T) {
	// `dur …` round-trips with the keyword preserved
	test.ExpectParseString(t, `x := dur 1h30m`, `x := dur 1h30m`)
	test.ExpectParseString(t, `x := dur 500ms`, `x := dur 500ms`)
	// in operand position (call argument)
	test.ExpectParseString(t, `f(dur 2s)`, `f(dur 2s)`)
	// `dur` not followed by a number stays a plain identifier
	test.ExpectParseString(t, `dur := 2`, `dur := 2`)
}

func TestParseDeferStmt(t *testing.T) {
	test.ExpectParseString(t, `defer { x }`, `defer { x }`)
	test.ExpectParseString(t, `defer handler`, `defer handler`)
	test.ExpectParseString(t, `defer handler(x)`, `defer handler(x)`)
	test.ExpectParseString(t, `defer_ok { x }`, `defer_ok { x }`)
	test.ExpectParseString(t, `defer_err { x }`, `defer_err { x }`)
	test.ExpectParseString(t, `deferb { x }`, `deferb { x }`)
	test.ExpectParseString(t, `deferb_ok { x }`, `deferb_ok { x }`)
	test.ExpectParseString(t, `deferb_err handler(x)`, `deferb_err handler(x)`)

	// shortcut form: call passing $ret/$err
	test.ExpectParseString(t, `defer cleanup($ret, $err)`, `defer cleanup($ret, $err)`)
	test.ExpectParseString(t, `defer_ok log($ret)`, `defer_ok log($ret)`)
	test.ExpectParseString(t, `deferb_err report($err)`, `deferb_err report($err)`)

	// shortcut form: assignment / increment (braceless statement)
	test.ExpectParseString(t, `defer $ret += 1`, `defer $ret += 1`)
	test.ExpectParseString(t, `deferb out += "x"`, `deferb out += "x"`)
	test.ExpectParseString(t, `defer i++`, `defer i++`)
}

func TestCodeNewNodes(t *testing.T) {
	// expression-form nodes round-trip on a single line via Code()
	test.New(t, `x := a or b`).Code(`x := a or b`)
	test.New(t, `x := [i * 2 for i in a if i > 1]`).Code(`x := [(i * 2) for i in a if (i > 1)]`)
	test.New(t, `x := {[k]: v for k, v in m}`).Code(`x := {[k]: v for k, v in m}`)
	test.New(t, `x := match (a) { 1: "one", else: "other" }`).
		Code(`x := match (a) { 1: "one", else: "other" }`)
	test.New(t, `x := /ab+/`).Code(`x := /ab+/`)
	test.New(t, `x := h"ffcc"`).Code(`x := h"ffcc"`)
	test.New(t, `x := b"Hello"`).Code(`x := b"Hello"`)

	// statement-form match with block arms: with the match formatter flag, one
	// arm per line, bodies indented
	test.New(t, `match a { 1 { b = 1 }, else { b = 2 } }`).
		FormattedCode("match a {\n\t1 {\n\t\tb = 1\n\t}\n\telse {\n\t\tb = 2\n\t}\n}")
}

func TestParseComprehension(t *testing.T) {
	test.ExpectParseString(t, `x := [i for i in a]`, `x := [i for i in a]`)
	test.ExpectParseString(t, `x := [i * 2 for i in a if i > 1]`,
		`x := [(i * 2) for i in a if (i > 1)]`)
	test.ExpectParseString(t, `x := [i + j for i in a for j in b]`,
		`x := [(i + j) for i in a for j in b]`)
	test.ExpectParseString(t, `x := {k: v for k, v in m}`, `x := {k: v for k, v in m}`)
	test.ExpectParseString(t, `x := {i: i for i in a if i}`, `x := {i: i for i in a if i}`)
	// computed key + multiple keys + $ access
	test.ExpectParseString(t, `x := {[i]: i * i for i in a}`, `x := {[i]: (i * i) for i in a}`)
	test.ExpectParseString(t, `x := {a: 1, [k]: v for k, v in m}`,
		`x := {a: 1, [k]: v for k, v in m}`)
}

func TestParseMatchExpr(t *testing.T) {
	// the subject no longer needs parentheses
	test.ExpectParseString(t,
		`x := match a { 1: "one", 2: "two", else: "other" }`,
		`x := match a { 1: "one", 2: "two", else: "other" }`)
	// `(a)` is preserved as a parenthesized expression
	test.ExpectParseString(t,
		`x := match (a) { 1: "one", else: "other" }`,
		`x := match (a) { 1: "one", else: "other" }`)
	// comma and newline separators between arms
	test.ExpectParseString(t,
		"x := match a {\n1: \"one\"\n2: \"two\"\nelse: \"other\"\n}",
		`x := match a { 1: "one", 2: "two", else: "other" }`)
	// non-literal conditions
	test.ExpectParseString(t,
		`x := match a { b: 1, c + 1: 2 }`,
		`x := match a { b: 1, (c + 1): 2 }`)
	// multiple conditions per arm (comma- and/or newline-separated)
	test.ExpectParseString(t,
		`x := match a { 1, 2, 3: "low", else: "hi" }`,
		`x := match a { 1, 2, 3: "low", else: "hi" }`)
	test.ExpectParseString(t,
		"x := match a {\n1, 2\n3: \"x\"\nelse: \"y\"\n}",
		`x := match a { 1, 2, 3: "x", else: "y" }`)
	// statement form, multi-condition arms
	test.ExpectParseString(t,
		`match a { 1, 2 { b = 1 } else { b = 2 } }`,
		`match a { 1, 2 { b = 1 }, else { b = 2 } }`)
	// an empty match is valid
	test.ExpectParseString(t, `x := match a {}`, `x := match a {}`)
}

func TestParseMatchExprError(t *testing.T) {
	// `else` may not be the only arm
	test.ExpectParseError(t, `x := match a { else: 2 }`)
}

func TestFormatMatchArms(t *testing.T) {
	// NEW_LINE_CALC: a match that fits the budget stays inline.
	test.New(t, `x := match n { 1: "a", else: "b" }`).
		FormattedCalcCode(`x := match n { 1: "a", else: "b" }`, 80)

	// NEW_LINE_CALC: when the inline arms overflow, each arm goes on its own
	// line with no comma between arms (the newline separates them).
	test.New(t, `x := match n { 1, 2: "one or two", 3: "three", else: "other" }`).
		FormattedCalcCode("x := match n {\n\t1, 2: \"one or two\"\n\t3: \"three\"\n\telse: \"other\"\n}", 40)

	// NEW_LINE_CALC: a single arm whose conditions overflow wraps them greedily
	// (packed per line, broken only on overflow, no comma at the break).
	test.New(t, `match i { 1, 2, 3, 4, 5, 6, 7, 8 {} }`).
		FormattedCalcCode("match i {\n\t1, 2, 3\n\t\t4, 5, 6\n\t\t7, 8 {}\n}", 10)

	// Primitive arms are sorted ascending (else stays last).
	test.New(t, `x := match n { 3: "c", 1: "a", 2: "b", else: "z" }`).
		FormattedCalcCode(`x := match n { 1: "a", 2: "b", 3: "c", else: "z" }`, 80)

	// Force-all: arms always split, one per line, no commas.
	test.New(t, `x := match n { 2: "b", 1: "a", else: "c" }`).
		FormattedCode("x := match n {\n\t1: \"a\"\n\t2: \"b\"\n\telse: \"c\"\n}")
}

func TestParseMethodInterface(t *testing.T) {
	test.ExpectParseString(t, `x := meti { () }`, `x := meti {(); }`)
	test.ExpectParseString(t, `x := meti { (), (v) <int> }`, `x := meti {(); (v) <int>; }`)
	test.ExpectParseString(t, `x := meti Z { (a int) <r bool> }`, `x := meti Z {(a int) <r bool>; }`)
	// statement form binds a const
	test.ExpectParseString(t, `meti S { () }`, `meti S {(); }`)
}

func TestParseFuncHeaderExpr(t *testing.T) {
	test.ExpectParseString(t, `x := <()>`, `x := <()>`)
	test.ExpectParseString(t, `x := <(v int)>`, `x := <(v int)>`)
	test.ExpectParseString(t, `x := <(a, b str)>`, `x := <(a, b str)>`)
	// nested return list closes with `>>` (the scanner's Shr is split)
	test.ExpectParseString(t, `x := <(v int) <r uint|int>>`, `x := <(v int) <r uint|int>>`)
	test.ExpectParseString(t, `x := <(a, b str) <int, str>>`, `x := <(a, b str) <int, str>>`)
}

func TestParsePrefixIncDec(t *testing.T) {
	// prefix ++/-- are unary operators (parenthesized like other unary exprs)
	test.ExpectParseString(t, `++x`, `(++x)`)
	test.ExpectParseString(t, `--x`, `(--x)`)
	test.ExpectParseString(t, `y := ++x`, `y := (++x)`)
	test.ExpectParseString(t, `return --x`, `return (--x)`)
	// postfix form is unchanged
	test.ExpectParseString(t, `x++`, `x++`)

	// binary `a ++ b` / `a -- b` (an operand follows); left-associative
	test.ExpectParseString(t, `a ++ b`, `(a ++ b)`)
	test.ExpectParseString(t, `a -- b`, `(a -- b)`)
	test.ExpectParseString(t, `a ++ b ++ c`, `((a ++ b) ++ c)`)
	// postfix is preserved when no operand follows (e.g. a for-loop post stmt)
	test.ExpectParseString(t, `for i := 0; i < 5; i++ {}`, `for i := 0 ; (i < 5)  ; i++{}`)
}

func TestParseSpreadLiterals(t *testing.T) {
	// array spread/merge
	test.ExpectParseString(t, `[1, 2, *a, 4, *b]`, `[1, 2, *a, 4, *b]`)
	test.ExpectParseString(t, `[*a]`, `[*a]`)
	test.ExpectParseString(t, `[*a, *b]`, `[*a, *b]`)
	// dict spread/merge (wrapped in `x := ...` to force expression context)
	test.ExpectParseString(t, `x := {a: 1, *b, c: 2, *d}`, `x := {a: 1, *b, c: 2, *d}`)
	test.ExpectParseString(t, `x := {*b}`, `x := {*b}`)
	test.ExpectParseString(t, `x := {*b, *d}`, `x := {*b, *d}`)
}

func TestParseMixedParamsDestructure(t *testing.T) {
	// `**rest` is now accepted in the positional section of a paren group
	test.ExpectParseString(t, `(a, b, **pr; c, p:d, r=2, **nr) := x`,
		`(, a, b, **pr; c, p:d, r=2, **nr) := x`)
	test.ExpectParseString(t, `(a, b) := x`, `(, a, b) := x`)
}

func TestParseDictDestructure(t *testing.T) {
	// the `(;...)` LHS parses to a KeyValueArrayLit; `:` is a rename mapping,
	// `=` is a fallback default and `**` is the optional rest target.
	test.ExpectParseString(t, `(;a, _b:b, r=2, **other) := d`,
		`(;a, _b:b, r=2, **other) := d`)
	test.ExpectParseString(t, `(;a, _b:b, r=2, **other) = d`,
		`(;a, _b:b, r=2, **other) = d`)
	test.ExpectParseString(t, `(;a) := d`, `(;a) := d`)
	test.ExpectParseString(t, `(;x:k) := d`, `(;x:k) := d`)
	test.ExpectParseString(t, `(;a, **rest) := d`, `(;a, **rest) := d`)
}

func TestParseBlock(t *testing.T) {
	test.ExpectParse(t, "{}", func(p pfn) []Stmt {
		return stmts(SBlock(p(1, 1), p(1, 2)))
	})

	test.ExpectParse(t, "x := 1; {x := 2}", func(p pfn) []Stmt {
		return stmts(
			SAssign(
				Exprs{EIdent("x", p(1, 1))},
				Exprs{Int(1, p(1, 6))},
				token.Define,
				p(1, 3),
			),
			SBlock(
				p(1, 9),
				p(1, 16),
				SAssign(
					Exprs{EIdent("x", p(1, 10))},
					Exprs{Int(2, p(1, 15))},
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
			return stmts(SExpr(EParen(EDict(p(1, 2), p(1, 3)), p(1, 1), p(1, 4))))
		})

	test.ExpectParse(t, "({ \"key1\": 1 })", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				EParen(
					EDict(p(1, 2), p(1, 14),
						EDictElementStr(
							"key1", p(1, 4), p(1, 10), Int(1, p(1, 12)))),
					p(1, 1), p(1, 15))))
	})

	test.ExpectParse(t, "({ key1: 1, key2: \"2\", key3: true })", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				EParen(
					EDict(p(1, 2), p(1, 35),
						EDictElement(EIdent("key1", p(1, 4)), p(1, 8), Int(1, p(1, 10))),
						EDictElement(EIdent("key2", p(1, 13)), p(1, 17), stringLit("2", p(1, 19))),
						EDictElement(EIdent("key3", p(1, 24)), p(1, 28), Bool(true, p(1, 30)))),
					p(1, 1), p(1, 36))))
	})

	test.ExpectParse(t, "a = { key1: 1 }",
		func(p pfn) []Stmt {
			return stmts(SAssign(
				exprs(EIdent("a", p(1, 1))),
				exprs(EDict(p(1, 5), p(1, 15),
					EDictElement(EIdent("key1", p(1, 7)), p(1, 11), Int(1, p(1, 13))))),
				token.Assign,
				p(1, 3)))
		})

	test.New(t, "a = { key1: 1, key2: \"2\", key3: { k1: `bar`, k2: 4 } }").
		String("a = {key1: 1, key2: \"2\", key3: {k1: `bar`, k2: 4}}").
		FormattedCode("a = {\n\tkey1: 1\n\tkey2: \"2\"\n\tkey3: {\n\t\tk1: `bar`\n\t\tk2: 4\n\t}\n}")

	test.New(t, `({ "key1": 1, #key2:2, #(key 3): #3, #(key 4): #(value	4), true: 5, false:6, yes:7, no:8`+
		"\n1:9, u2:10, 3d:11, 4.56:12, [x+1]:2})").
		String(`({key1: 1, #(key2): 2, #(key 3): #(3), #(key 4): #(value	4), true: 5, false: 6, yes: 7, no: 8, 1: 9, u2: 10, 3d: 11, 4.56: 12, [x + 1]: 2})`).
		FormattedCode(`({
	"key1": 1
	#key2: 2
	#(key 3): #3
	#(key 4): #(value	4)
	true: 5
	false: 6
	yes: 7
	no: 8
	1: 9
	u2: 10
	3d: 11
	4.56: 12
	[x + 1]: 2
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
		return stmts(SExpr(EParen(EDict(p(1, 2), p(3, 1),
			EDictElement(EIdent("x", p(2, 2)), 0,
				EDictElementClosure(EClosure(NewFuncParams(p(2, 3), p(2, 4)), p(2, 5), token.Colon, Int(10, p(2, 7))))),
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
	x() : 10
	y() {
		return 11
	}
})`).
		Stmts(func(p pfn) []Stmt {
			return stmts(SExpr(EParen(EDict(p(1, 2), p(6, 1),
				EDictElement(EIdent("x", p(2, 2)), 0, EDictElementClosure(EClosure(NewFuncParams(p(2, 3), p(2, 4)), p(2, 5), token.Colon, Int(10, p(2, 7))))),
				EDictElement(EIdent("y", p(3, 2)), 0, EDictElementFunc(EFunc(funcType(0, nil, p(3, 3), p(3, 4)), SBlock(p(3, 6), p(5, 2), SReturn(p(4, 3), Int(11, p(4, 10))))))),
			), p(1, 1), p(6, 2))))
		})

	test.ExpectParse(t, "({x{y{b:1}}})", func(p pfn) []Stmt {
		return stmts(SExpr(EParen(EDict(
			p(1, 2), p(1, 12),
			EDictElement(EIdent("x", p(1, 3)), 0, EDict(
				p(1, 4), p(1, 11),
				EDictElement(EIdent("y", p(1, 5)), 0, EDict(
					p(1, 6), p(1, 10),
					EDictElement(EIdent("b", p(1, 7)), p(1, 8), Int(1, p(1, 9))),
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
	// the range operator `..` binds tighter than `/` so the `/ step` groups
	// outside the range.
	test.ExpectParseString(t, `1 .. 2`, `(1 .. 2)`)
	test.ExpectParseString(t, `1..2`, `(1 .. 2)`)
	test.ExpectParseString(t, `1 .. 10 / 2`, `((1 .. 10) / 2)`)
	test.ExpectParseString(t, `(1 .. 10) / 2`, `((1 .. 10) / 2)`)
	test.ExpectParseString(t, `a.b .. c`, `(a.b .. c)`)
	test.ExpectParseString(t, `1.5 .. 2.0`, `(1.5 .. 2.0)`)
	// user binary operators bind like the other multiplicative operators.
	test.ExpectParseString(t, `a <<< b`, `(a <<< b)`)
	test.ExpectParseString(t, `a >>> b`, `(a >>> b)`)
	test.ExpectParseString(t, `a %% b`, `(a %% b)`)
	test.ExpectParseString(t, `a <<< b + c`, `((a <<< b) + c)`)
	test.ExpectParseString(t, `a + b %% c`, `(a + (b %% c))`)
	test.ExpectParseString(t, `a << b <<< c`, `((a << b) <<< c)`)
	// the `in` membership operator (comparison precedence), and its
	// disambiguation from the for-in separator.
	test.ExpectParseString(t, `a in b`, `(a in b)`)
	test.ExpectParseString(t, `1 in [1, 2, 3]`, `(1 in [1, 2, 3])`)
	test.ExpectParseString(t, `a in b && c in d`, `((a in b) && (c in d))`)
	test.ExpectParseString(t, `a + b in c`, `((a + b) in c)`)
	test.ExpectParseString(t, `if x in y { }`, `if (x in y) {}`)
	test.ExpectParseString(t, `for x in y { }`, `for _, x in y {}`)
	test.ExpectParseString(t, `for (x in y) { }`, `for (x in y) {}`)
}

func TestParseNullishSelector(t *testing.T) {
	test.ExpectParseString(t, `a?.(k)`, `a?.(k)`)
	test.ExpectParseString(t, `a?.(k+x)`, `a?.(k + x)`)
	test.ExpectParse(t, "a?.b.c?.d", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				ENullish(
					ESelector(
						ENullish(
							EIdent("a", p(1, 1)),
							stringLit("b", p(1, 4))),
						stringLit("c", p(1, 6))),
					stringLit("d", p(1, 9)))))
	})
	test.ExpectParse(t, "a?.b.c?.d.e", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				ESelector(
					ENullish(
						ESelector(
							ENullish(
								EIdent("a", p(1, 1)),
								stringLit("b", p(1, 4))),
							stringLit("c", p(1, 6))),
						stringLit("d", p(1, 9))),
					stringLit("e", p(1, 11)))))
	})
	test.ExpectParse(t, "a?.b.c?.d.e?.f.g", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				ESelector(
					ENullish(
						ESelector(
							ENullish(
								ESelector(
									ENullish(
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
			SExpr(
				ENullish(
					ENullish(
						EIdent("a", p(1, 1)),
						stringLit("b", p(1, 4))),
					stringLit("c", p(1, 7)))))
	})
	test.ExpectParse(t, "a?.b", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				ENullish(
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
			SExpr(
				ESelector(
					EIdent("a", p(1, 1)),
					stringLit("b", p(1, 3)))))
	})

	test.ExpectParse(t, "a.b.c", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				ESelector(
					ESelector(
						EIdent("a", p(1, 1)),
						stringLit("b", p(1, 3))),
					stringLit("c", p(1, 5)))))
	})

	test.ExpectParse(t, "a.(b).c", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				ESelector(
					ESelector(
						EIdent("a", p(1, 1)),
						EParen(EIdent("b", p(1, 4)), p(1, 3), p(1, 5))),
					stringLit("c", p(1, 7)))))
	})

	test.ExpectParse(t, "({k1:1}.k1)", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				EParen(
					ESelector(
						EDict(
							p(1, 2), p(1, 7),
							EDictElement(
								EIdent("k1", p(1, 3)), p(1, 5), Int(1, p(1, 6)))),
						stringLit("k1", p(1, 9))),
					p(1, 1), p(1, 11))))

	})

	test.ExpectParse(t, "({k1:1}).k1", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				ESelector(
					EParen(
						EDict(
							p(1, 2), p(1, 7),
							EDictElement(
								EIdent("k1", p(1, 3)), p(1, 5), Int(1, p(1, 6)))),
						p(1, 1), p(1, 8)),
					stringLit("k1", p(1, 10)))))

	})

	test.ExpectParse(t, "({k1:{v1:1}}.k1.v1)", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				EParen(
					ESelector(
						ESelector(
							EDict(
								p(1, 2), p(1, 12),
								EDictElement(EIdent("k1", p(1, 3)), p(1, 5),
									EDict(p(1, 6), p(1, 11),
										EDictElement(
											EIdent("v1", p(1, 7)),
											p(1, 9), Int(1, p(1, 10)))))),
							stringLit("k1", p(1, 14))),
						stringLit("v1", p(1, 17))),
					p(1, 1), p(1, 19))))
	})

	test.ExpectParse(t, "({k1:{v1:1}}).k1.v1", func(p pfn) []Stmt {
		return stmts(
			SExpr(
				ESelector(
					ESelector(
						EParen(
							EDict(
								p(1, 2), p(1, 12),
								EDictElement(EIdent("k1", p(1, 3)), p(1, 5),
									EDict(p(1, 6), p(1, 11),
										EDictElement(
											EIdent("v1", p(1, 7)),
											p(1, 9), Int(1, p(1, 10)))))),
							p(1, 1), p(1, 13)),
						stringLit("k1", p(1, 15))),
					stringLit("v1", p(1, 18)))))
	})

	test.ExpectParse(t, "a.b = 4", func(p pfn) []Stmt {
		return stmts(
			SAssign(
				exprs(
					ESelector(
						EIdent("a", p(1, 1)),
						stringLit("b", p(1, 3)))),
				exprs(Int(4, p(1, 7))),
				token.Assign, p(1, 5)))
	})

	test.ExpectParse(t, "a.b.c = 4", func(p pfn) []Stmt {
		return stmts(
			SAssign(
				exprs(
					ESelector(
						ESelector(
							EIdent("a", p(1, 1)),
							stringLit("b", p(1, 3))),
						stringLit("c", p(1, 5)))),
				exprs(Int(4, p(1, 9))),
				token.Assign, p(1, 7)))
	})

	test.ExpectParse(t, "a.b.c = 4 + 5", func(p pfn) []Stmt {
		return stmts(
			SAssign(
				exprs(
					ESelector(
						ESelector(
							EIdent("a", p(1, 1)),
							stringLit("b", p(1, 3))),
						stringLit("c", p(1, 5)))),
				exprs(
					EBinary(
						Int(4, p(1, 9)),
						Int(5, p(1, 13)),
						token.Add,
						p(1, 11))),
				token.Assign, p(1, 7)))
	})

	test.ExpectParse(t, "a[0].c = 4", func(p pfn) []Stmt {
		return stmts(
			SAssign(
				exprs(
					ESelector(
						EIndex(
							EIdent("a", p(1, 1)),
							Int(0, p(1, 3)),
							p(1, 2), p(1, 4)),
						stringLit("c", p(1, 6)))),
				exprs(Int(4, p(1, 10))),
				token.Assign, p(1, 8)))
	})

	test.ExpectParse(t, "a.b[0].c = 4", func(p pfn) []Stmt {
		return stmts(
			SAssign(
				exprs(
					ESelector(
						EIndex(
							ESelector(
								EIdent("a", p(1, 1)),
								stringLit("b", p(1, 3))),
							Int(0, p(1, 5)),
							p(1, 4), p(1, 6)),
						stringLit("c", p(1, 8)))),
				exprs(Int(4, p(1, 12))),
				token.Assign, p(1, 10)))
	})

	test.ExpectParse(t, "a.b[0][2].c = 4", func(p pfn) []Stmt {
		return stmts(
			SAssign(
				exprs(
					ESelector(
						EIndex(
							EIndex(
								ESelector(
									EIdent("a", p(1, 1)),
									stringLit("b", p(1, 3))),
								Int(0, p(1, 5)),
								p(1, 4), p(1, 6)),
							Int(2, p(1, 8)),
							p(1, 7), p(1, 9)),
						stringLit("c", p(1, 11)))),
				exprs(Int(4, p(1, 15))),
				token.Assign, p(1, 13)))
	})

	test.ExpectParse(t, `a.b["key1"][2].c = 4`, func(p pfn) []Stmt {
		return stmts(
			SAssign(
				exprs(
					ESelector(
						EIndex(
							EIndex(
								ESelector(
									EIdent("a", p(1, 1)),
									stringLit("b", p(1, 3))),
								stringLit("key1", p(1, 5)),
								p(1, 4), p(1, 11)),
							Int(2, p(1, 13)),
							p(1, 12), p(1, 14)),
						stringLit("c", p(1, 16)))),
				exprs(Int(4, p(1, 20))),
				token.Assign, p(1, 18)))
	})

	test.ExpectParse(t, "a[0].b[2].c = 4", func(p pfn) []Stmt {
		return stmts(
			SAssign(
				exprs(
					ESelector(
						EIndex(
							ESelector(
								EIndex(
									EIdent("a", p(1, 1)),
									Int(0, p(1, 3)),
									p(1, 2), p(1, 4)),
								stringLit("b", p(1, 6))),
							Int(2, p(1, 8)),
							p(1, 7), p(1, 9)),
						stringLit("c", p(1, 11)))),
				exprs(Int(4, p(1, 15))),
				token.Assign, p(1, 13)))
	})
}

func TestParseSemicolon(t *testing.T) {
	test.ExpectParse(t, "1", func(p pfn) []Stmt {
		return stmts(
			SExpr(Int(1, p(1, 1))))
	})

	test.ExpectParse(t, "1;", func(p pfn) []Stmt {
		return stmts(
			SExpr(Int(1, p(1, 1))))
	})

	test.ExpectParse(t, "1;;", func(p pfn) []Stmt {
		return stmts(
			SExpr(Int(1, p(1, 1))))
	})

	test.ExpectParse(t, `1
`, func(p pfn) []Stmt {
		return stmts(
			SExpr(Int(1, p(1, 1))))
	})

	test.ExpectParse(t, `1
;`, func(p pfn) []Stmt {
		return stmts(
			SExpr(Int(1, p(1, 1))))
	})

	test.ExpectParse(t, `1;
;`, func(p pfn) []Stmt {
		return stmts(
			SExpr(Int(1, p(1, 1))))
	})
}

func TestParseString(t *testing.T) {
	test.ExpectParse(t, "\"foo\nbar\"", func(p pfn) []Stmt {
		return stmts(SExpr(stringLit("foo\nbar", p(1, 1))))
	})
	test.ExpectParse(t, "\"foo\nbar\"", func(p pfn) []Stmt {
		return stmts(SExpr(stringLit("foo\nbar", p(1, 1))))
	})
	test.ExpectParse(t, "\"foo\n"+"\n"+"bar\"", func(p pfn) []Stmt {
		return stmts(SExpr(stringLit("foo\n\nbar", p(1, 1))))
	})
	test.ExpectParse(t, `"foo\n`+"\n"+`bar"`, func(p pfn) []Stmt {
		return stmts(SExpr(stringLit("foo\\n\nbar", p(1, 1))))
	})
	test.ExpectParse(t, "`abc`", func(p pfn) []Stmt {
		return stmts(SExpr(RawStr(`abc`, p(1, 1))))
	})
	test.ExpectParse(t, "```\nabc\n```", func(p pfn) []Stmt {
		return stmts(SExpr(rawHeredocLit("```", `abc`, p(1, 1))))
	})
	test.ExpectParse(t, `"""`+"\nabc\n"+`"""`, func(p pfn) []Stmt {
		return stmts(SExpr(heredocLit(`"""`, `abc`, p(1, 1))))
	})
	test.ExpectParse(t, "a = \"foo\nbar\"", func(p pfn) []Stmt {
		return stmts(
			SAssign(
				exprs(EIdent("a", p(1, 1))),
				exprs(stringLit("foo\nbar", p(1, 5))),
				token.Assign,
				p(1, 3)))
	})
	test.ExpectParse(t, `a = "foo\nbar"`, func(p pfn) []Stmt {
		return stmts(
			SAssign(
				exprs(EIdent("a", p(1, 1))),
				exprs(stringLit(`foo\nbar`, p(1, 5))),
				token.Assign,
				p(1, 3)))
	})
	test.ExpectParse(t, "a = `raw string`", func(p pfn) []Stmt {
		return stmts(
			SAssign(
				exprs(EIdent("a", p(1, 1))),
				exprs(RawStr("`raw string`", p(1, 5))),
				token.Assign,
				p(1, 3)))
	})

	test.ExpectParse(t, "raw \"foo\nbar\"", func(p pfn) []Stmt {
		return stmts(SExpr(EToRaw(p(1, 1), stringLit("foo\nbar", p(1, 5)))))
	})
	test.ExpectParseString(t, `1 + raw "foo"`, `(1 + raw "foo")`)
	test.ExpectParseString(t, `raw x()`, `raw x()`)
}

func TestParseSymbol(t *testing.T) {
	test.ExpectParse(t, "#abc", func(p pfn) []Stmt {
		return stmts(SExpr(LSymbol(p(1, 1), "abc", false)))
	})
	test.ExpectParse(t, "#(a\n\\)\tbc)", func(p pfn) []Stmt {
		return stmts(SExpr(LSymbol(p(1, 1), "a\n)\tbc", true)))
	})
}

func TestParseConfig(t *testing.T) {
	test.ExpectParse(t, `# gad: mixed
	a`, func(p pfn) []Stmt {
		return stmts(
			SConfig(p(1, 1), KVp(EIdent("mixed", p(1, 8)))),
			SMixedText(p(2, 1), "\ta"),
		)
	})
	test.ExpectParse(t, `# gad: mixed, delimiter = ["[[[", "]]]"]
y
[[[b]]]`, func(p pfn) []Stmt {
		return stmts(
			SConfig(p(1, 1),
				KVp(EIdent("mixed", p(1, 8))),
				KVp(EIdent("delimiter", p(1, 15)),
					Array(p(1, 27), p(1, 40),
						stringLit("[[[", p(1, 28)),
						stringLit("]]]", p(1, 35)),
					)),
			),
			SMixedText(p(2, 1), "y\n"),
			SCodeBegin(Lit("[[[", p(3, 1)), false),
			SExpr(EIdent("b", p(3, 4))),
			SCodeEnd(Lit("]]]", p(3, 5)), false),
		)
	})

	test.ExpectParseString(t, "# gad: mixed, delimiter=[\"[[[\", \"]]]\"]\ny\n[[[b]]]",
		"# gad: mixed, delimiter=[\"[[[\", \"]]]\"]\ny\n[[[; b; ]]]")
	test.ExpectParseString(t, "# gad: mixed, delimiter=[\"[[[\", \"]]]\"]\ny\n[[[b; true]]]",
		"# gad: mixed, delimiter=[\"[[[\", \"]]]\"]\ny\n[[[; b; true; ]]]")
	test.ExpectParse(t, `# gad: mixed`, func(p pfn) []Stmt {
		return stmts(
			SConfig(p(1, 1), KVp(EIdent("mixed", p(1, 8)))))
	})
}

func TestParseTryThrow(t *testing.T) {
	test.ExpectParse(t, `try {} catch e {} finally {}`, func(p pfn) []Stmt {
		return stmts(
			STry(p(1, 1),
				SBlock(p(1, 5), p(1, 6)),
				SCatch(p(1, 8), EIdent("e", p(1, 14)),
					SBlock(p(1, 16), p(1, 17))),
				SFinally(p(1, 19),
					SBlock(p(1, 27), p(1, 28))),
			),
		)
	})
	test.ExpectParse(t, `try {} finally {}`, func(p pfn) []Stmt {
		return stmts(
			STry(p(1, 1),
				SBlock(p(1, 5), p(1, 6)),
				nil,
				SFinally(p(1, 8),
					SBlock(p(1, 16), p(1, 17))),
			),
		)
	})
	test.ExpectParse(t, `try {
} finally {}`, func(p pfn) []Stmt {
		return stmts(
			STry(p(1, 1),
				SBlock(p(1, 5), p(2, 1)),
				nil,
				SFinally(p(2, 3),
					SBlock(p(2, 11), p(2, 12))),
			),
		)
	})
	test.ExpectParse(t, `try {} catch {}`, func(p pfn) []Stmt {
		return stmts(
			STry(p(1, 1),
				SBlock(p(1, 5), p(1, 6)),
				SCatch(p(1, 8), nil,
					SBlock(p(1, 14), p(1, 15))),
				nil,
			),
		)
	})
	test.ExpectParse(t, `try {
} catch {}`, func(p pfn) []Stmt {
		return stmts(
			STry(p(1, 1),
				SBlock(p(1, 5), p(2, 1)),
				SCatch(p(2, 3), nil,
					SBlock(p(2, 9), p(2, 10))),
				nil,
			),
		)
	})
	test.ExpectParse(t, `throw "error"`, func(p pfn) []Stmt {
		return stmts(
			SThrow(p(1, 1), stringLit("error", p(1, 7))),
		)
	})
	test.ExpectParse(t, `throw 1`, func(p pfn) []Stmt {
		return stmts(
			SThrow(p(1, 1), Int(1, p(1, 7))),
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
			return stmts(SExpr(EComputed(p(1, 1), p(1, 4), SExpr(Int(1, p(1, 3))))))
		})

	test.New(t, "(= 1|2 )").
		String("(= (1 | 2))").
		Code("(= 1 | 2)")

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
		Code("func a() {a := (= (; x=(= y(); z)))}").
		IndentedCode(`func a() {
	a := (= (; x=(=
		y()
		z
	)))
}`)
}

func TestParseExportStmt(t *testing.T) {
	test.New(t, `export a; export a = 2; export func f(){return 1}; export f2(){return 1}; export x() => 3; export["abc"] = 2; export {x:1, y:2}; export (cfn()); export (import("abc"))`).
		String("export a; export a = 2; export func f() { return 1 }; export f2() { return 1 }; export x() => 3; export [\"abc\"] = 2; export {x: 1, y: 2}; export (cfn()); export (import(\"abc\"))").
		Code("export a; export a = 2; export func f() {return 1}; export f2() {return 1}; export x() => 3; export [\"abc\"] = 2; export {x: 1, y: 2}; export (cfn()); export (import(\"abc\"))").
		FormattedCode(`export a

export a = 2

export func f() {
	return 1
}

export f2() {
	return 1
}

export x() => 3

export ["abc"] = 2

export {
	x: 1
	y: 2
}

export (cfn())

export (import("abc"))`)
}

type pfn = test.Pfn               // position conversion function
type expectedFn = test.ExpectedFn // callback function to return expected results

func stmts(s ...Stmt) []Stmt {
	return s
}

func exprs(list ...Expr) []Expr {
	return list
}

// funcType keeps the positional name argument used throughout the tests; the
// rest delegates to the node constructor.
func funcType(pos source.Pos, ident Expr, lparen, rparen Pos, v ...any) *FuncType {
	f := NewFuncType(pos, lparen, rparen, v...)
	f.NameExpr = ident
	return f
}

// funcArgs accepts either an *IdentExpr or a *TypedIdentExpr as the variadic
// param, which the node Args constructor does not.
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
			l.Values = append(l.Values, ETypedIdent(t))
		case *TypedIdentExpr:
			l.Values = append(l.Values, t)
		}
	}
	return l
}

// stringLit builds a string literal without escaping, unlike node.Str.
func stringLit(value string, pos Pos) *StrLit {
	return &StrLit{Literal: `"` + value + `"`, ValuePos: pos}
}

func charAsStrLit(value string, pos Pos) *StrLit {
	return &StrLit{Literal: `'` + strings.ReplaceAll(value, "'", `\'`) + `'`, ValuePos: pos}
}

func rawHeredocLit(q, value string, pos Pos) *RawHeredocLit {
	return RawHeredoc(q+"\n"+value+"\n"+q, pos)
}

func heredocLit(q, value string, pos Pos) *HeredocLit {
	return Heredoc(q+"\n"+value+"\n"+q, pos)
}

func callExprNamedArgs(
	names []*NamedArgExpr, values []Expr,
) (ce CallExprNamedArgs) {
	return CallExprNamedArgs{Names: names, Values: values}
}
