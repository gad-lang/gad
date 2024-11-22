package test_helper

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	. "github.com/gad-lang/gad/parser/node"
	. "github.com/gad-lang/gad/parser/source"
	"github.com/stretchr/testify/require"

	. "github.com/gad-lang/gad/parser"
)

func ParseTrace(t *testing.T, input, expected string) {
	parse := func(input string, tracer io.Writer) {
		testFileSet := NewFileSet()
		testFile := testFileSet.AddFile("test", -1, len(input))
		p := NewParser(testFile, []byte(input), tracer)
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

type Pfn func(int, int) Pos         // position conversion function
type ExpectedFn func(pos Pfn) Stmts // callback function to return expected results

type parseTracer struct {
	out []string
}

func (o *parseTracer) Write(p []byte) (n int, err error) {
	o.out = append(o.out, string(p))
	return len(p), nil
}

type Option func(po *ParserOptions, so *ScannerOptions)

func Parse(t *testing.T, input string, do func(f *SourceFile, actual *File, err error), opt ...Option) {
	testFileSet := NewFileSet()
	testFile := testFileSet.AddFile("test", -1, len(input))

	var (
		ok      bool
		options = func() (po *ParserOptions, so *ScannerOptions) {
			po = &ParserOptions{}
			so = &ScannerOptions{}
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
			p := NewParserWithOptions(testFile, []byte(input), po, so)
			actual, _ := p.ParseFile()
			if actual != nil {
				t.Logf("Parsed:\n%s", actual.String())
			}
			t.Logf("Trace:\n%s", strings.Join(tr.out, ""))
		}
	}()

	po, so := options()

	p := NewParserWithOptions(testFile, []byte(input), po, so)
	actual, err := p.ParseFile()
	do(testFile, actual, err)
	ok = true
}

func ExpectParse(t *testing.T, input string, fn ExpectedFn, opt ...Option) {
	Parse(t, input, func(f *SourceFile, actual *File, err error) {
		require.NoError(t, err)

		expected := fn(func(line, column int) Pos {
			return Pos(int(f.LineStart(line)) + (column - 1))
		})
		require.Equal(t, len(expected), len(actual.Stmts), "count of file statements")

		for i := 0; i < len(expected); i++ {
			EqualStmt(t, expected[i], actual.Stmts[i])
		}
	}, opt...)
}

func ExpectParseString(t *testing.T, input, expected string, opt ...Option) {
	Parse(t, input, func(f *SourceFile, actual *File, err error) {
		require.NoError(t, err)
		require.Equal(t, expected, actual.String())
	}, opt...)
}

func ExpectParseError(t *testing.T, input string, opt ...Option) {
	Parse(t, input, func(f *SourceFile, actual *File, err error) {
		require.Error(t, err)
	}, opt...)
}

func EqualStmt(t *testing.T, expected, actual Stmt) {
	if expected == nil || reflect.ValueOf(expected).IsNil() {
		require.Nil(t, actual, "expected nil, but got not nil")
		return
	}
	require.NotNil(t, actual, "expected not nil, but got nil")
	require.IsType(t, expected, actual)

	switch expected := expected.(type) {
	case *ExprStmt:
		EqualExpr(t, expected.Expr, actual.(*ExprStmt).Expr)
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
		EqualStmts(t, expected.Stmts,
			actual.(*BlockStmt).Stmts)
	case *AssignStmt:
		EqualExprs(t, expected.LHS,
			actual.(*AssignStmt).LHS)
		EqualExprs(t, expected.RHS,
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
					EqualExpr(t, expectedSpec.Value, actualSpec.Value)
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
						EqualExpr(t, expr, actualSpec.Values[i])
					}
				}
			default:
				require.Failf(t, "unknown type", "unknown Spec '%T'", expSpec)
			}
		}
	case *IfStmt:
		EqualStmt(t, expected.Init, actual.(*IfStmt).Init)
		EqualExpr(t, expected.Cond, actual.(*IfStmt).Cond)
		EqualStmt(t, expected.Body, actual.(*IfStmt).Body)
		EqualStmt(t, expected.Else, actual.(*IfStmt).Else)
		require.Equal(t, expected.IfPos, actual.(*IfStmt).IfPos)
	case *TryStmt:
		require.Equal(t, expected.TryPos, actual.(*TryStmt).TryPos)
		EqualStmt(t, expected.Body, actual.(*TryStmt).Body)
		EqualStmt(t, expected.Catch, actual.(*TryStmt).Catch)
		EqualStmt(t, expected.Finally, actual.(*TryStmt).Finally)
	case *CatchStmt:
		require.Equal(t, expected.CatchPos, actual.(*CatchStmt).CatchPos)
		require.Equal(t, expected.Ident, actual.(*CatchStmt).Ident)
		EqualStmt(t, expected.Body, actual.(*CatchStmt).Body)
	case *FinallyStmt:
		require.Equal(t, expected.FinallyPos, actual.(*FinallyStmt).FinallyPos)
		EqualStmt(t, expected.Body, actual.(*FinallyStmt).Body)
	case *ThrowStmt:
		require.Equal(t, expected.ThrowPos, actual.(*ThrowStmt).ThrowPos)
		EqualExpr(t, expected.Expr, actual.(*ThrowStmt).Expr)
	case *IncDecStmt:
		EqualExpr(t, expected.Expr,
			actual.(*IncDecStmt).Expr)
		require.Equal(t, expected.Token,
			actual.(*IncDecStmt).Token)
		require.Equal(t, expected.TokenPos,
			actual.(*IncDecStmt).TokenPos)
	case *ForStmt:
		EqualStmt(t, expected.Init, actual.(*ForStmt).Init)
		EqualExpr(t, expected.Cond, actual.(*ForStmt).Cond)
		EqualStmt(t, expected.Post, actual.(*ForStmt).Post)
		EqualStmt(t, expected.Body, actual.(*ForStmt).Body)
		require.Equal(t, expected.ForPos, actual.(*ForStmt).ForPos)
	case *ForInStmt:
		EqualExpr(t, expected.Key,
			actual.(*ForInStmt).Key)
		EqualExpr(t, expected.Value,
			actual.(*ForInStmt).Value)
		EqualExpr(t, expected.Iterable,
			actual.(*ForInStmt).Iterable)
		EqualStmt(t, expected.Body,
			actual.(*ForInStmt).Body)
		require.Equal(t, expected.ForPos,
			actual.(*ForInStmt).ForPos)
		EqualStmt(t, expected.Else,
			actual.(*ForInStmt).Else)
	case *ReturnStmt:
		EqualExpr(t, expected.Result,
			actual.(*ReturnStmt).Result)
		require.Equal(t, expected.ReturnPos,
			actual.(*ReturnStmt).ReturnPos)
	case *BranchStmt:
		EqualExpr(t, expected.Label,
			actual.(*BranchStmt).Label)
		require.Equal(t, expected.Token,
			actual.(*BranchStmt).Token)
		require.Equal(t, expected.TokenPos,
			actual.(*BranchStmt).TokenPos)
	case *MixedTextStmt:
		require.Equal(t, expected.Lit.Value,
			actual.(*MixedTextStmt).Lit.Value)
		require.Equal(t, expected.Lit.Pos,
			actual.(*MixedTextStmt).Lit.Pos)
		require.Equal(t, expected.Flags.String(),
			actual.(*MixedTextStmt).Flags.String(), "Flags")
	case *MixedValueStmt:
		require.Equal(t, expected.StartLit.Value,
			actual.(*MixedValueStmt).StartLit.Value)
		require.Equal(t, expected.StartLit.Pos,
			actual.(*MixedValueStmt).StartLit.Pos)
		require.Equal(t, expected.EndLit.Value,
			actual.(*MixedValueStmt).EndLit.Value)
		require.Equal(t, expected.EndLit.Pos,
			actual.(*MixedValueStmt).EndLit.Pos)
		EqualExpr(t, expected.Expr,
			actual.(*MixedValueStmt).Expr)
	case *ConfigStmt:
		require.Equal(t, expected.ConfigPos,
			actual.(*ConfigStmt).ConfigPos)
		require.Equal(t, expected.Options,
			actual.(*ConfigStmt).Options)
		require.Equal(t, len(expected.Elements),
			len(actual.(*ConfigStmt).Elements))
		for i, e := range expected.Elements {
			EqualExpr(t, e, actual.(*ConfigStmt).Elements[i])
		}
	case *CodeBeginStmt:
		require.Equal(t, expected.RemoveSpace,
			actual.(*CodeBeginStmt).RemoveSpace)
		require.Equal(t, expected.Lit.Pos,
			actual.(*CodeBeginStmt).Lit.Pos)
		require.Equal(t, expected.Lit.Value,
			actual.(*CodeBeginStmt).Lit.Value)
	case *CodeEndStmt:
		require.Equal(t, expected.RemoveSpace,
			actual.(*CodeEndStmt).RemoveSpace)
		require.Equal(t, expected.Lit.Pos,
			actual.(*CodeEndStmt).Lit.Pos)
		require.Equal(t, expected.Lit.Value,
			actual.(*CodeEndStmt).Lit.Value)
	default:
		panic(fmt.Errorf("unknown type: %T", expected))
	}
}

func EqualExpr(t *testing.T, expected, actual Expr) {
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
		EqualExpr(t, expected.Ident, actual.(*TypedIdent).Ident)
		EqualIdents(t, expected.Type, actual.(*TypedIdent).Type)
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
		EqualExprs(t, expected.Elements,
			actual.(*ArrayLit).Elements)
	case *DictLit:
		require.Equal(t, expected.LBrace,
			actual.(*DictLit).LBrace)
		require.Equal(t, expected.RBrace,
			actual.(*DictLit).RBrace)
		EqualMapElements(t, expected.Elements,
			actual.(*DictLit).Elements)
	case *NilLit:
		require.Equal(t, expected.TokenPos,
			actual.(*NilLit).TokenPos)
	case *ReturnExpr:
		require.Equal(t, expected.ReturnPos,
			actual.(*ReturnExpr).ReturnPos)
		EqualExpr(t, expected.Result,
			actual.(*ReturnExpr).Result)
	case *NullishSelectorExpr:
		EqualExpr(t, expected.Expr,
			actual.(*NullishSelectorExpr).Expr)
		EqualExpr(t, expected.Sel,
			actual.(*NullishSelectorExpr).Sel)
	case *BinaryExpr:
		EqualExpr(t, expected.LHS,
			actual.(*BinaryExpr).LHS)
		EqualExpr(t, expected.RHS,
			actual.(*BinaryExpr).RHS)
		require.Equal(t, expected.Token,
			actual.(*BinaryExpr).Token)
		require.Equal(t, expected.TokenPos,
			actual.(*BinaryExpr).TokenPos)
	case *UnaryExpr:
		EqualExpr(t, expected.Expr,
			actual.(*UnaryExpr).Expr)
		require.Equal(t, expected.Token,
			actual.(*UnaryExpr).Token)
		require.Equal(t, expected.TokenPos,
			actual.(*UnaryExpr).TokenPos)
	case *FuncLit:
		EqualFuncType(t, expected.Type,
			actual.(*FuncLit).Type)
		EqualStmt(t, expected.Body,
			actual.(*FuncLit).Body)
	case *CallExpr:
		actual := actual.(*CallExpr)
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
	case *ParenExpr:
		EqualExpr(t, expected.Expr,
			actual.(*ParenExpr).Expr)
		require.Equal(t, expected.LParen,
			actual.(*ParenExpr).LParen)
		require.Equal(t, expected.RParen,
			actual.(*ParenExpr).RParen)
	case *IndexExpr:
		EqualExpr(t, expected.Expr,
			actual.(*IndexExpr).Expr)
		EqualExpr(t, expected.Index,
			actual.(*IndexExpr).Index)
		require.Equal(t, expected.LBrack,
			actual.(*IndexExpr).LBrack)
		require.Equal(t, expected.RBrack,
			actual.(*IndexExpr).RBrack)
	case *SliceExpr:
		EqualExpr(t, expected.Expr,
			actual.(*SliceExpr).Expr)
		EqualExpr(t, expected.Low,
			actual.(*SliceExpr).Low)
		EqualExpr(t, expected.High,
			actual.(*SliceExpr).High)
		require.Equal(t, expected.LBrack,
			actual.(*SliceExpr).LBrack)
		require.Equal(t, expected.RBrack,
			actual.(*SliceExpr).RBrack)
	case *SelectorExpr:
		EqualExpr(t, expected.Expr,
			actual.(*SelectorExpr).Expr)
		EqualExpr(t, expected.Sel,
			actual.(*SelectorExpr).Sel)
	case *ImportExpr:
		require.Equal(t, expected.ModuleName,
			actual.(*ImportExpr).ModuleName)
		require.Equal(t, int(expected.TokenPos),
			int(actual.(*ImportExpr).TokenPos))
		require.Equal(t, expected.Token,
			actual.(*ImportExpr).Token)
	case *CondExpr:
		EqualExpr(t, expected.Cond,
			actual.(*CondExpr).Cond)
		EqualExpr(t, expected.True,
			actual.(*CondExpr).True)
		EqualExpr(t, expected.False,
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
		EqualFuncType(t, expected.Type,
			actual.(*ClosureLit).Type)
		EqualExpr(t, expected.Body,
			actual.(*ClosureLit).Body)
	case *BlockExpr:
		EqualStmt(t, expected.BlockStmt,
			actual.(*BlockExpr).BlockStmt)
	case *KeyValueLit:
		EqualExpr(t, expected.Key,
			actual.(*KeyValueLit).Key)
		EqualExpr(t, expected.Value,
			actual.(*KeyValueLit).Value)
	default:
		panic(fmt.Errorf("unknown type: %T", expected))
	}
}

func EqualFuncType(t *testing.T, expected, actual *FuncType) {
	require.Equal(t, expected.Params.LParen, actual.Params.LParen)
	require.Equal(t, expected.Params.RParen, actual.Params.RParen)
	EqualTypedIdents(t, expected.Params.Args.Values, actual.Params.Args.Values)
	EqualNamedArgs(t, &expected.Params.NamedArgs, &actual.Params.NamedArgs)
}

func EqualNamedArgs(t *testing.T, expected, actual *NamedArgsList) {
	if expected == nil && actual == nil {
		return
	}
	require.NotNil(t, expected, "expected is nil")
	require.NotNil(t, actual, "actual is nil")

	require.Equal(t, expected.Var, actual.Var)
	EqualTypedIdents(t, expected.Names, actual.Names)
	EqualExprs(t, expected.Values, actual.Values)
}

func EqualNamedArgsNames(t *testing.T, expected, actual []NamedArgExpr) {
	require.Equal(t, len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		EqualExpr(t, expected[i].Ident, actual[i].Ident)
		EqualExpr(t, expected[i].Lit, actual[i].Lit)
	}
}

func EqualIdents(t *testing.T, expected, actual []*Ident) {
	require.Equal(t, len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		EqualExpr(t, expected[i], actual[i])
	}
}

func EqualTypedIdents(t *testing.T, expected, actual []*TypedIdent) {
	require.Equal(t, len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		EqualExpr(t, expected[i], actual[i])
	}
}

func EqualExprs(t *testing.T, expected, actual []Expr) {
	require.Equal(t, len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		EqualExpr(t, expected[i], actual[i])
	}
}

func EqualStmts(t *testing.T, expected, actual []Stmt) {
	require.Equal(t, len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		EqualStmt(t, expected[i], actual[i])
	}
}

func EqualMapElements(
	t *testing.T,
	expected, actual []*DictElementLit,
) {
	require.Equal(t, len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		require.Equal(t, expected[i].Key, actual[i].Key)
		require.Equal(t, expected[i].KeyPos, actual[i].KeyPos)
		require.Equal(t, expected[i].ColonPos, actual[i].ColonPos)
		EqualExpr(t, expected[i].Value, actual[i].Value)
	}
}
