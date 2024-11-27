package testhelper

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	"github.com/stretchr/testify/require"

	"github.com/gad-lang/gad/parser"
)

func ParseTrace(t *testing.T, input, expected string) {
	parse := func(input string, tracer io.Writer) {
		testFileSet := source.NewFileSet()
		testFile := testFileSet.AddFile("test", -1, len(input))
		p := parser.NewParser(testFile, []byte(input), tracer)
		_, err := p.ParseFile()
		require.NoError(t, err)
	}
	var out bytes.Buffer
	parse(input, &out)
	require.Equal(t,
		strings.ReplaceAll(string(expected), "\r\n", "\n"),
		strings.ReplaceAll(out.String(), "\r\n", "\n"),
	)
}

type Pfn func(int, int) source.Pos       // position conversion function
type ExpectedFn func(pos Pfn) node.Stmts // callback function to return expected results

type parseTracer struct {
	out []string
}

func (o *parseTracer) Write(p []byte) (n int, err error) {
	o.out = append(o.out, string(p))
	return len(p), nil
}

type Option func(po *parser.ParserOptions, so *parser.ScannerOptions)

func Parse(t *testing.T, input string, do func(f *source.File, actual *parser.File, err error), opt ...Option) {
	testFileSet := source.NewFileSet()
	testFile := testFileSet.AddFile("test", -1, len(input))

	var (
		ok      bool
		options = func() (po *parser.ParserOptions, so *parser.ScannerOptions) {
			po = &parser.ParserOptions{}
			so = &parser.ScannerOptions{}
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
			p := parser.NewParserWithOptions(testFile, []byte(input), po, so)
			actual, _ := p.ParseFile()
			if actual != nil {
				t.Logf("Parsed:\n%s", actual.String())
			}
			t.Logf("Trace:\n%s", strings.Join(tr.out, ""))
		}
	}()

	po, so := options()

	p := parser.NewParserWithOptions(testFile, []byte(input), po, so)
	actual, err := p.ParseFile()
	do(testFile, actual, err)
	ok = true
}

func ExpectParse(t *testing.T, input string, fn ExpectedFn, opt ...Option) {
	Parse(t, input, func(f *source.File, actual *parser.File, err error) {
		require.NoError(t, err)

		expected := fn(func(line, column int) source.Pos {
			return source.Pos(int(f.LineStart(line)) + (column - 1))
		})
		require.Equal(t, len(expected), len(actual.Stmts), "count of file statements")

		for i := 0; i < len(expected); i++ {
			EqualStmt(t, expected[i], actual.Stmts[i])
		}
	}, opt...)
}

func ExpectParseString(t *testing.T, input, expected string, opt ...Option) {
	Parse(t, input, func(f *source.File, actual *parser.File, err error) {
		require.NoError(t, err)
		require.Equal(t, expected, actual.String())
	}, opt...)
}

func ExpectParseError(t *testing.T, input string, opt ...Option) {
	Parse(t, input, func(f *source.File, actual *parser.File, err error) {
		require.Error(t, err)
	}, opt...)
}

func EqualStmt(t *testing.T, expected, actual node.Stmt) {
	if expected == nil || reflect.ValueOf(expected).IsNil() {
		require.Nil(t, actual, "expected nil, but got not nil")
		return
	}
	require.NotNil(t, actual, "expected not nil, but got nil")
	require.IsType(t, expected, actual)

	switch expected := expected.(type) {
	case *node.ExprStmt:
		EqualExpr(t, expected.Expr, actual.(*node.ExprStmt).Expr)
	case *node.EmptyStmt:
		require.Equal(t, expected.Implicit,
			actual.(*node.EmptyStmt).Implicit)
		require.Equal(t, expected.Semicolon,
			actual.(*node.EmptyStmt).Semicolon)
	case *node.BlockStmt:
		require.Equal(t, expected.LBrace,
			actual.(*node.BlockStmt).LBrace)
		require.Equal(t, expected.RBrace,
			actual.(*node.BlockStmt).RBrace)
		EqualStmts(t, expected.Stmts,
			actual.(*node.BlockStmt).Stmts)
	case *node.AssignStmt:
		EqualExprs(t, expected.LHS,
			actual.(*node.AssignStmt).LHS)
		EqualExprs(t, expected.RHS,
			actual.(*node.AssignStmt).RHS)
		require.Equal(t, int(expected.Token),
			int(actual.(*node.AssignStmt).Token))
		require.Equal(t, int(expected.TokenPos),
			int(actual.(*node.AssignStmt).TokenPos))
	case *node.DeclStmt:
		expectedDecl := expected.Decl.(*node.GenDecl)
		actualDecl := actual.(*node.DeclStmt).Decl.(*node.GenDecl)
		require.Equal(t, expectedDecl.Tok, actualDecl.Tok)
		require.Equal(t, expectedDecl.TokPos, actualDecl.TokPos)
		require.Equal(t, expectedDecl.Lparen, actualDecl.Lparen)
		require.Equal(t, expectedDecl.Rparen, actualDecl.Rparen)
		require.Equal(t, len(expectedDecl.Specs), len(actualDecl.Specs))
		for i, expSpec := range expectedDecl.Specs {
			actSpec := actualDecl.Specs[i]
			switch expectedSpec := expSpec.(type) {
			case *node.ParamSpec:
				actualSpec, ok := actSpec.(*node.ParamSpec)
				if !ok {
					require.Failf(t, "type error", "expected *ParamSpec, got %T", actSpec)
					return
				}
				require.Equal(t, expectedSpec.Ident, actualSpec.Ident)
				require.Equal(t, expectedSpec.Variadic, actualSpec.Variadic)
			case *node.NamedParamSpec:
				actualSpec, ok := actSpec.(*node.NamedParamSpec)
				if !ok {
					require.Failf(t, "type error", "expected *NamedParamSpec, got %T", actSpec)
					return
				}
				require.Equal(t, expectedSpec.Ident, actualSpec.Ident)
				if expectedSpec.Value != nil || actualSpec.Value != nil {
					EqualExpr(t, expectedSpec.Value, actualSpec.Value)
				}
			case *node.ValueSpec:
				actualSpec, ok := actSpec.(*node.ValueSpec)
				if !ok {
					require.Failf(t, "type error", "expected *ValueSpec, got %T", actSpec)
					return
				}
				require.Equal(t, expectedSpec.Idents, actualSpec.Idents)
				require.Equal(t, len(expectedSpec.Values), len(actualSpec.Values))
				if len(expectedSpec.Values) == len(actualSpec.Values) {
					for i, expr := range expectedSpec.Values {
						EqualExpr(t, expr, actualSpec.Values[i])
					}
				}
			default:
				require.Failf(t, "unknown type", "unknown Spec '%T'", expSpec)
			}
		}
	case *node.IfStmt:
		EqualStmt(t, expected.Init, actual.(*node.IfStmt).Init)
		EqualExpr(t, expected.Cond, actual.(*node.IfStmt).Cond)
		EqualStmt(t, expected.Body, actual.(*node.IfStmt).Body)
		EqualStmt(t, expected.Else, actual.(*node.IfStmt).Else)
		require.Equal(t, expected.IfPos, actual.(*node.IfStmt).IfPos)
	case *node.TryStmt:
		require.Equal(t, expected.TryPos, actual.(*node.TryStmt).TryPos)
		EqualStmt(t, expected.Body, actual.(*node.TryStmt).Body)
		EqualStmt(t, expected.Catch, actual.(*node.TryStmt).Catch)
		EqualStmt(t, expected.Finally, actual.(*node.TryStmt).Finally)
	case *node.CatchStmt:
		require.Equal(t, expected.CatchPos, actual.(*node.CatchStmt).CatchPos)
		require.Equal(t, expected.Ident, actual.(*node.CatchStmt).Ident)
		EqualStmt(t, expected.Body, actual.(*node.CatchStmt).Body)
	case *node.FinallyStmt:
		require.Equal(t, expected.FinallyPos, actual.(*node.FinallyStmt).FinallyPos)
		EqualStmt(t, expected.Body, actual.(*node.FinallyStmt).Body)
	case *node.ThrowStmt:
		require.Equal(t, expected.ThrowPos, actual.(*node.ThrowStmt).ThrowPos)
		EqualExpr(t, expected.Expr, actual.(*node.ThrowStmt).Expr)
	case *node.IncDecStmt:
		EqualExpr(t, expected.Expr,
			actual.(*node.IncDecStmt).Expr)
		require.Equal(t, expected.Token,
			actual.(*node.IncDecStmt).Token)
		require.Equal(t, expected.TokenPos,
			actual.(*node.IncDecStmt).TokenPos)
	case *node.ForStmt:
		EqualStmt(t, expected.Init, actual.(*node.ForStmt).Init)
		EqualExpr(t, expected.Cond, actual.(*node.ForStmt).Cond)
		EqualStmt(t, expected.Post, actual.(*node.ForStmt).Post)
		EqualStmt(t, expected.Body, actual.(*node.ForStmt).Body)
		require.Equal(t, expected.ForPos, actual.(*node.ForStmt).ForPos)
	case *node.ForInStmt:
		EqualExpr(t, expected.Key,
			actual.(*node.ForInStmt).Key)
		EqualExpr(t, expected.Value,
			actual.(*node.ForInStmt).Value)
		EqualExpr(t, expected.Iterable,
			actual.(*node.ForInStmt).Iterable)
		EqualStmt(t, expected.Body,
			actual.(*node.ForInStmt).Body)
		require.Equal(t, expected.ForPos,
			actual.(*node.ForInStmt).ForPos)
		EqualStmt(t, expected.Else,
			actual.(*node.ForInStmt).Else)
	case *node.ReturnStmt:
		EqualExpr(t, expected.Result,
			actual.(*node.ReturnStmt).Result)
		require.Equal(t, expected.ReturnPos,
			actual.(*node.ReturnStmt).ReturnPos)
	case *node.BranchStmt:
		EqualExpr(t, expected.Label,
			actual.(*node.BranchStmt).Label)
		require.Equal(t, expected.Token,
			actual.(*node.BranchStmt).Token)
		require.Equal(t, expected.TokenPos,
			actual.(*node.BranchStmt).TokenPos)
	case *node.MixedTextStmt:
		require.Equal(t, expected.Lit.Value,
			actual.(*node.MixedTextStmt).Lit.Value)
		require.Equal(t, expected.Lit.Pos,
			actual.(*node.MixedTextStmt).Lit.Pos)
		require.Equal(t, expected.Flags.String(),
			actual.(*node.MixedTextStmt).Flags.String(), "Flags")
	case *node.MixedValueStmt:
		require.Equal(t, expected.StartLit.Value,
			actual.(*node.MixedValueStmt).StartLit.Value)
		require.Equal(t, expected.StartLit.Pos,
			actual.(*node.MixedValueStmt).StartLit.Pos)
		require.Equal(t, expected.EndLit.Value,
			actual.(*node.MixedValueStmt).EndLit.Value)
		require.Equal(t, expected.EndLit.Pos,
			actual.(*node.MixedValueStmt).EndLit.Pos)
		EqualExpr(t, expected.Expr,
			actual.(*node.MixedValueStmt).Expr)
	case *node.ConfigStmt:
		require.Equal(t, expected.ConfigPos,
			actual.(*node.ConfigStmt).ConfigPos)
		require.Equal(t, expected.Options,
			actual.(*node.ConfigStmt).Options)
		require.Equal(t, len(expected.Elements),
			len(actual.(*node.ConfigStmt).Elements))
		for i, e := range expected.Elements {
			EqualExpr(t, e, actual.(*node.ConfigStmt).Elements[i])
		}
	case *node.CodeBeginStmt:
		require.Equal(t, expected.RemoveSpace,
			actual.(*node.CodeBeginStmt).RemoveSpace)
		require.Equal(t, expected.Lit.Pos,
			actual.(*node.CodeBeginStmt).Lit.Pos)
		require.Equal(t, expected.Lit.Value,
			actual.(*node.CodeBeginStmt).Lit.Value)
	case *node.CodeEndStmt:
		require.Equal(t, expected.RemoveSpace,
			actual.(*node.CodeEndStmt).RemoveSpace)
		require.Equal(t, expected.Lit.Pos,
			actual.(*node.CodeEndStmt).Lit.Pos)
		require.Equal(t, expected.Lit.Value,
			actual.(*node.CodeEndStmt).Lit.Value)
	default:
		panic(fmt.Errorf("unknown type: %T", expected))
	}
}

func EqualExpr(t *testing.T, expected, actual node.Expr) {
	if expected == nil || reflect.ValueOf(expected).IsNil() {
		require.Nil(t, actual, "expected nil, but got not nil")
		return
	}
	require.NotNil(t, actual, "expected not nil, but got nil")
	require.IsType(t, expected, actual)

	switch expected := expected.(type) {
	case *node.Ident:
		require.Equal(t, expected.Name,
			actual.(*node.Ident).Name)
		require.Equal(t, int(expected.NamePos),
			int(actual.(*node.Ident).NamePos))
	case *node.TypedIdent:
		EqualExpr(t, expected.Ident, actual.(*node.TypedIdent).Ident)
		EqualIdents(t, expected.Type, actual.(*node.TypedIdent).Type)
	case *node.IntLit:
		require.Equal(t, expected.Value,
			actual.(*node.IntLit).Value)
		require.Equal(t, int(expected.ValuePos),
			int(actual.(*node.IntLit).ValuePos))
	case *node.FloatLit:
		require.Equal(t, expected.Value,
			actual.(*node.FloatLit).Value)
		require.Equal(t, int(expected.ValuePos),
			int(actual.(*node.FloatLit).ValuePos))
	case *node.DecimalLit:
		require.True(t, expected.Value.Equal(actual.(*node.DecimalLit).Value))
		require.Equal(t, int(expected.ValuePos),
			int(actual.(*node.DecimalLit).ValuePos))
	case *node.BoolLit:
		require.Equal(t, expected.Value,
			actual.(*node.BoolLit).Value)
		require.Equal(t, int(expected.ValuePos),
			int(actual.(*node.BoolLit).ValuePos))
	case *node.FlagLit:
		require.Equal(t, expected.Value,
			actual.(*node.FlagLit).Value)
		require.Equal(t, int(expected.ValuePos),
			int(actual.(*node.FlagLit).ValuePos))
	case *node.CharLit:
		require.Equal(t, expected.Value,
			actual.(*node.CharLit).Value)
		require.Equal(t, int(expected.ValuePos),
			int(actual.(*node.CharLit).ValuePos))
	case *node.StringLit:
		require.Equal(t, expected.Value,
			actual.(*node.StringLit).Value)
		require.Equal(t, int(expected.ValuePos),
			int(actual.(*node.StringLit).ValuePos))
	case *node.RawStringLit:
		require.Equal(t, expected.UnquotedValue(),
			actual.(*node.RawStringLit).UnquotedValue())
		require.Equal(t, int(expected.LiteralPos),
			int(actual.(*node.RawStringLit).LiteralPos))
	case *node.ArrayLit:
		require.Equal(t, expected.LBrack,
			actual.(*node.ArrayLit).LBrack)
		require.Equal(t, expected.RBrack,
			actual.(*node.ArrayLit).RBrack)
		EqualExprs(t, expected.Elements,
			actual.(*node.ArrayLit).Elements)
	case *node.DictLit:
		require.Equal(t, expected.LBrace,
			actual.(*node.DictLit).LBrace)
		require.Equal(t, expected.RBrace,
			actual.(*node.DictLit).RBrace)
		EqualMapElements(t, expected.Elements,
			actual.(*node.DictLit).Elements)
	case *node.NilLit:
		require.Equal(t, expected.TokenPos,
			actual.(*node.NilLit).TokenPos)
	case *node.ReturnExpr:
		require.Equal(t, expected.ReturnPos,
			actual.(*node.ReturnExpr).ReturnPos)
		EqualExpr(t, expected.Result,
			actual.(*node.ReturnExpr).Result)
	case *node.NullishSelectorExpr:
		EqualExpr(t, expected.Expr,
			actual.(*node.NullishSelectorExpr).Expr)
		EqualExpr(t, expected.Sel,
			actual.(*node.NullishSelectorExpr).Sel)
	case *node.BinaryExpr:
		EqualExpr(t, expected.LHS,
			actual.(*node.BinaryExpr).LHS)
		EqualExpr(t, expected.RHS,
			actual.(*node.BinaryExpr).RHS)
		require.Equal(t, expected.Token,
			actual.(*node.BinaryExpr).Token)
		require.Equal(t, expected.TokenPos,
			actual.(*node.BinaryExpr).TokenPos)
	case *node.UnaryExpr:
		EqualExpr(t, expected.Expr,
			actual.(*node.UnaryExpr).Expr)
		require.Equal(t, expected.Token,
			actual.(*node.UnaryExpr).Token)
		require.Equal(t, expected.TokenPos,
			actual.(*node.UnaryExpr).TokenPos)
	case *node.FuncLit:
		EqualFuncType(t, expected.Type,
			actual.(*node.FuncLit).Type)
		EqualStmt(t, expected.Body,
			actual.(*node.FuncLit).Body)
	case *node.CallExpr:
		actual := actual.(*node.CallExpr)
		EqualExpr(t, expected.Func,
			actual.Func)
		require.Equal(t, expected.LParen,
			actual.LParen)
		require.Equal(t, expected.RParen,
			actual.RParen)
		EqualExprs(t, expected.Args.Values,
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
			EqualExpr(t, expected.Args.Var.Value,
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
			EqualExpr(t, expected.NamedArgs.Var.Value,
				actual.NamedArgs.Var.Value)
		}

		EqualNamedArgsNames(t, expected.NamedArgs.Names,
			actual.NamedArgs.Names)
		EqualExprs(t, expected.NamedArgs.Values,
			actual.NamedArgs.Values)
	case *node.ParenExpr:
		EqualExpr(t, expected.Expr,
			actual.(*node.ParenExpr).Expr)
		require.Equal(t, expected.LParen,
			actual.(*node.ParenExpr).LParen)
		require.Equal(t, expected.RParen,
			actual.(*node.ParenExpr).RParen)
	case *node.IndexExpr:
		EqualExpr(t, expected.Expr,
			actual.(*node.IndexExpr).Expr)
		EqualExpr(t, expected.Index,
			actual.(*node.IndexExpr).Index)
		require.Equal(t, expected.LBrack,
			actual.(*node.IndexExpr).LBrack)
		require.Equal(t, expected.RBrack,
			actual.(*node.IndexExpr).RBrack)
	case *node.SliceExpr:
		EqualExpr(t, expected.Expr,
			actual.(*node.SliceExpr).Expr)
		EqualExpr(t, expected.Low,
			actual.(*node.SliceExpr).Low)
		EqualExpr(t, expected.High,
			actual.(*node.SliceExpr).High)
		require.Equal(t, expected.LBrack,
			actual.(*node.SliceExpr).LBrack)
		require.Equal(t, expected.RBrack,
			actual.(*node.SliceExpr).RBrack)
	case *node.SelectorExpr:
		EqualExpr(t, expected.Expr,
			actual.(*node.SelectorExpr).Expr)
		EqualExpr(t, expected.Sel,
			actual.(*node.SelectorExpr).Sel)
	case *node.ImportExpr:
		require.Equal(t, expected.ModuleName,
			actual.(*node.ImportExpr).ModuleName)
		require.Equal(t, int(expected.TokenPos),
			int(actual.(*node.ImportExpr).TokenPos))
		require.Equal(t, expected.Token,
			actual.(*node.ImportExpr).Token)
	case *node.CondExpr:
		EqualExpr(t, expected.Cond,
			actual.(*node.CondExpr).Cond)
		EqualExpr(t, expected.True,
			actual.(*node.CondExpr).True)
		EqualExpr(t, expected.False,
			actual.(*node.CondExpr).False)
		require.Equal(t, expected.QuestionPos,
			actual.(*node.CondExpr).QuestionPos)
		require.Equal(t, expected.ColonPos,
			actual.(*node.CondExpr).ColonPos)
	case *node.CalleeKeyword:
		require.Equal(t, expected.Literal,
			actual.(*node.CalleeKeyword).Literal)
		require.Equal(t, expected.TokenPos,
			actual.(*node.CalleeKeyword).TokenPos)
	case *node.ArgsKeyword:
		require.Equal(t, expected.Literal,
			actual.(*node.ArgsKeyword).Literal)
		require.Equal(t, expected.TokenPos,
			actual.(*node.ArgsKeyword).TokenPos)
	case *node.NamedArgsKeyword:
		require.Equal(t, expected.Literal,
			actual.(*node.NamedArgsKeyword).Literal)
		require.Equal(t, expected.TokenPos,
			actual.(*node.NamedArgsKeyword).TokenPos)
	case *node.ClosureLit:
		EqualFuncType(t, expected.Type,
			actual.(*node.ClosureLit).Type)
		EqualExpr(t, expected.Body,
			actual.(*node.ClosureLit).Body)
	case *node.BlockExpr:
		EqualStmt(t, expected.BlockStmt,
			actual.(*node.BlockExpr).BlockStmt)
	case *node.KeyValueLit:
		EqualExpr(t, expected.Key,
			actual.(*node.KeyValueLit).Key)
		EqualExpr(t, expected.Value,
			actual.(*node.KeyValueLit).Value)
	default:
		panic(fmt.Errorf("unknown type: %T", expected))
	}
}

func EqualFuncType(t *testing.T, expected, actual *node.FuncType) {
	require.Equal(t, expected.Params.LParen, actual.Params.LParen)
	require.Equal(t, expected.Params.RParen, actual.Params.RParen)
	EqualTypedIdents(t, expected.Params.Args.Values, actual.Params.Args.Values)
	EqualNamedArgs(t, &expected.Params.NamedArgs, &actual.Params.NamedArgs)
}

func EqualNamedArgs(t *testing.T, expected, actual *node.NamedArgsList) {
	if expected == nil && actual == nil {
		return
	}
	require.NotNil(t, expected, "expected is nil")
	require.NotNil(t, actual, "actual is nil")

	require.Equal(t, expected.Var, actual.Var)
	EqualTypedIdents(t, expected.Names, actual.Names)
	EqualExprs(t, expected.Values, actual.Values)
}

func EqualNamedArgsNames(t *testing.T, expected, actual []node.NamedArgExpr) {
	require.Equal(t, len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		EqualExpr(t, expected[i].Ident, actual[i].Ident)
		EqualExpr(t, expected[i].Lit, actual[i].Lit)
	}
}

func EqualIdents(t *testing.T, expected, actual []*node.Ident) {
	require.Equal(t, len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		EqualExpr(t, expected[i], actual[i])
	}
}

func EqualTypedIdents(t *testing.T, expected, actual []*node.TypedIdent) {
	require.Equal(t, len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		EqualExpr(t, expected[i], actual[i])
	}
}

func EqualExprs(t *testing.T, expected, actual []node.Expr) {
	require.Equal(t, len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		EqualExpr(t, expected[i], actual[i])
	}
}

func EqualStmts(t *testing.T, expected, actual []node.Stmt) {
	require.Equal(t, len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		EqualStmt(t, expected[i], actual[i])
	}
}

func EqualMapElements(
	t *testing.T,
	expected, actual []*node.DictElementLit,
) {
	require.Equal(t, len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		require.Equal(t, expected[i].Key, actual[i].Key)
		require.Equal(t, expected[i].KeyPos, actual[i].KeyPos)
		require.Equal(t, expected[i].ColonPos, actual[i].ColonPos)
		EqualExpr(t, expected[i].Value, actual[i].Value)
	}
}
