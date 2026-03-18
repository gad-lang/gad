package test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	"github.com/stretchr/testify/require"
)

type ParseContext struct {
	ParserOptions  *parser.ParserOptions
	ScannerOptions *parser.ScannerOptions
	PostTests      []PostFileCallback
}

func (c *ParseContext) PostTest(f func(t *testing.T, f *source.File, pos func(line, column int) source.Pos)) {
	c.PostTests = append(c.PostTests, f)
}

type ExpectOpt func(ctx *ParseContext)

var OptParseCharAsString ExpectOpt = func(ctx *ParseContext) {
	ctx.ScannerOptions.Mode |= parser.ScanCharAsString
}

func PostTest(do func(t *testing.T, f *source.File, pos func(line, column int) source.Pos)) ExpectOpt {
	return func(ctx *ParseContext) {
		ctx.PostTest(do)
	}
}

func ExpectParse(t *testing.T, input string, fn ExpectedFn, opt ...ExpectOpt) {
	ExpectParseMode(t, 0, input, fn, opt...)
}

func ExpectParseStmt(t *testing.T, input string, stmt node.Stmt, opt ...ExpectOpt) {
	ExpectParse(t, input, func(p Pfn) []node.Stmt { return node.Stmts{stmt} }, opt...)
}

func ExpectParseExpr(t *testing.T, input string, expr node.Expr, opt ...ExpectOpt) {
	ExpectParseStmt(t, input, node.SExpr(expr), opt...)
}

func ExpectParseMode(t *testing.T, mode parser.Mode, input string, fn ExpectedFn, opt ...ExpectOpt) {
	p := New(t, input).WithMode(mode)

	var (
		options = func() (ctx *ParseContext) {
			ctx = &ParseContext{
				ParserOptions:  p.GetParserOptions(),
				ScannerOptions: p.GetScannerOptions(),
			}
			for _, o := range opt {
				o(ctx)
			}
			return
		}
	)

	ctx := options()

	p.File().Expect(fn, ctx.PostTests...)
}

func ExpectParseError(t *testing.T, input string, e ...[2]string) {
	var (
		binput      = []byte(input)
		testFileSet = source.NewFileSet()
		testFile    = testFileSet.AddFileData("test", -1, binput)
	)

	var ok bool
	defer func() {
		if !ok {
			// print Trace
			tr := &Tracer{}
			p := parser.NewParser(testFile, tr)
			_, _ = p.ParseFile()
			t.Logf("Trace:\n%s", strings.Join(tr.Out, ""))
		}
	}()

	p := parser.NewParser(testFile, nil)
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

func ExpectParseString(t *testing.T, input, expected string) {
	ExpectParseStringMode(t, 0, input, expected)
}

func ExpectParseStringMode(t *testing.T, mode parser.Mode, input, expected string) {
	ExpectParseStringModeT(t, mode, input, expected, nil)
}

func ExpectParseStringMixed(t *testing.T, input, expected string) {
	ExpectParseStringModeT(t, parser.ParseMixed, input, expected, nil)
}

func ExpectParseMixed(t *testing.T, input string, fn ExpectedFn) {
	ExpectParseMode(t, parser.ParseMixed, input, fn)
}

func ExpectParseStringT(t *testing.T, input, expected string, typ any) {
	New(t, input).String(expected).Type(typ)
}

func ExpectParseStringModeT(t *testing.T, mode parser.Mode, input, expected string, typ any) {
	New(t, input).WithMode(mode).String(expected).Type(typ)
}
