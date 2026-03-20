package test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Pfn func(int, int) source.Pos        // position conversion function
type ExpectedFn func(pos Pfn) []node.Stmt // callback function to return expected results

type PostFileCallback func(t *testing.T, f *source.File, pos func(line, column int) source.Pos)

type File struct {
	t      *testing.T
	actual *parser.File
}

func NewFile(t *testing.T, actual *parser.File) *File {
	return &File{t: t, actual: actual}
}

func (f *File) Expect(fn ExpectedFn, post ...PostFileCallback) *File {
	buildPos := func(line, column int) source.Pos {
		return source.Pos(int(source.MustFileLineStartPos(f.actual.InputFile, line)) + (column - 1))
	}

	expectedStmts := fn(buildPos)

	f.Equal(len(expectedStmts), len(f.actual.Stmts), "len(file.Stmts)")

	for i := 0; i < len(expectedStmts); i++ {
		f.EqualStmt(expectedStmts[i], f.actual.Stmts[i])
	}

	for _, pt := range post {
		pt(f.t, f.actual.InputFile, buildPos)
	}
	return f
}

func (f *File) Equal(expected, actual any, msgAndArgs ...any) {
	switch t := expected.(type) {
	case source.Pos:
		p, err := f.actual.InputFile.Position(t)
		assert.NoError(f.t, err, msgAndArgs...)
		expected = fmt.Sprintf("Pos(%d, %d)", p.Line, p.Column)
		if pos, ok := actual.(source.Pos); ok {
			p, err = f.actual.InputFile.Position(pos)
			assert.NoError(f.t, err, msgAndArgs...)
			actual = fmt.Sprintf("Pos(%d, %d)", p.Line, p.Column)
		}
	case ast.Literal:
		if actual, ok := actual.(ast.Literal); ok {
			f.Equal(t.Pos, actual.Pos, msgAndArgs...)
			require.Equal(f.t, t.Value, actual.Value, msgAndArgs...)
			return
		}
	case node.Token:
		if actual, ok := actual.(node.Token); ok {
			f.Equal(t.Pos, actual.Pos, msgAndArgs...)
			require.Equal(f.t, t.Token, actual.Token, msgAndArgs...)
			return
		}
	case node.TokenLit:
		if actual, ok := actual.(node.TokenLit); ok {
			f.Equal(t.Pos, actual.Pos, msgAndArgs...)
			require.Equal(f.t, t.Token, actual.Token, msgAndArgs...)
			require.Equal(f.t, t.Literal, actual.Literal, msgAndArgs...)
			return
		}
	case *node.FuncParams:
		if actual, ok := actual.(*node.FuncParams); ok {
			f.EqualFuncParams(t, actual)
			return
		}
	case *node.FuncType:
		if actual, ok := actual.(*node.FuncType); ok {
			f.Equal(&t.Params, &actual.Params, msgAndArgs...)
			return
		}
	}

	require.Equal(f.t, expected, actual, msgAndArgs...)
}

func (f *File) EqualStmt(expected, actual node.Stmt) {
	t := f.t
	if expected == nil || reflect.ValueOf(expected).IsNil() {
		require.Nil(t, actual, "expected nil, but got not nil")
		return
	}
	require.NotNil(t, actual, "expected not nil, but got nil")
	require.IsType(t, expected, actual)

	switch expected := expected.(type) {
	case *node.ExprStmt:
		f.EqualExpr(expected.Expr, actual.(*node.ExprStmt).Expr)
	case *node.EmptyStmt:
		f.Equal(expected.Implicit, actual.(*node.EmptyStmt).Implicit)
		f.Equal(expected.Semicolon, actual.(*node.EmptyStmt).Semicolon)
	case *node.BlockStmt:
		f.Equal(expected.LBrace, actual.(*node.BlockStmt).LBrace)
		f.Equal(expected.RBrace, actual.(*node.BlockStmt).RBrace)
		f.EqualStmts(expected.Stmts, actual.(*node.BlockStmt).Stmts)
	case *node.AssignStmt:
		f.EqualExprs(expected.LHS, actual.(*node.AssignStmt).LHS)
		f.EqualExprs(expected.RHS, actual.(*node.AssignStmt).RHS)
		f.Equal(int(expected.Token), int(actual.(*node.AssignStmt).Token))
		f.Equal(int(expected.TokenPos), int(actual.(*node.AssignStmt).TokenPos))
	case *node.DeclStmt:
		expectedDecl := expected.Decl.(*node.GenDecl)
		actualDecl := actual.(*node.DeclStmt).Decl.(*node.GenDecl)
		f.Equal(expectedDecl.Tok, actualDecl.Tok)
		f.Equal(expectedDecl.TokPos, actualDecl.TokPos)
		f.Equal(expectedDecl.Lparen, actualDecl.Lparen)
		f.Equal(expectedDecl.Rparen, actualDecl.Rparen)
		f.Equal(len(expectedDecl.Specs), len(actualDecl.Specs))
		for i, expSpec := range expectedDecl.Specs {
			actSpec := actualDecl.Specs[i]
			switch expectedSpec := expSpec.(type) {
			case *node.ParamSpec:
				actualSpec, ok := actSpec.(*node.ParamSpec)
				if !ok {
					require.Failf(t, "type error", "expected *ParamSpec, got %T", actSpec)
					return
				}
				f.Equal(expectedSpec.Ident, actualSpec.Ident)
				f.Equal(expectedSpec.Var, actualSpec.Var)
			case *node.NamedParamSpec:
				actualSpec, ok := actSpec.(*node.NamedParamSpec)
				if !ok {
					require.Failf(t, "type error", "expected *NamedParamSpec, got %T", actSpec)
					return
				}
				f.Equal(expectedSpec.Ident, actualSpec.Ident)
				if expectedSpec.Value != nil || actualSpec.Value != nil {
					f.EqualExpr(expectedSpec.Value, actualSpec.Value)
				}
			case *node.ValueSpec:
				actualSpec, ok := actSpec.(*node.ValueSpec)
				if !ok {
					require.Failf(t, "type error", "expected *ValueSpec, got %T", actSpec)
					return
				}
				f.Equal(expectedSpec.Idents, actualSpec.Idents)
				f.Equal(len(expectedSpec.Values), len(actualSpec.Values))
				if len(expectedSpec.Values) == len(actualSpec.Values) {
					for i, expr := range expectedSpec.Values {
						f.EqualExpr(expr, actualSpec.Values[i])
					}
				}
			default:
				require.Failf(t, "unknown type", "unknown Spec '%T'", expSpec)
			}
		}
	case *node.IfStmt:
		f.EqualStmt(expected.Init, actual.(*node.IfStmt).Init)
		f.EqualExpr(expected.Cond, actual.(*node.IfStmt).Cond)
		f.EqualStmt(expected.Body, actual.(*node.IfStmt).Body)
		f.EqualStmt(expected.Else, actual.(*node.IfStmt).Else)
		f.Equal(expected.IfPos, actual.(*node.IfStmt).IfPos)
	case *node.TryStmt:
		f.Equal(expected.TryPos, actual.(*node.TryStmt).TryPos)
		f.EqualStmt(expected.Body, actual.(*node.TryStmt).Body)
		f.EqualStmt(expected.Catch, actual.(*node.TryStmt).Catch)
		f.EqualStmt(expected.Finally, actual.(*node.TryStmt).Finally)
	case *node.CatchStmt:
		f.Equal(expected.CatchPos, actual.(*node.CatchStmt).CatchPos)
		f.Equal(expected.Ident, actual.(*node.CatchStmt).Ident)
		f.EqualStmt(expected.Body, actual.(*node.CatchStmt).Body)
	case *node.FinallyStmt:
		f.Equal(expected.FinallyPos, actual.(*node.FinallyStmt).FinallyPos)
		f.EqualStmt(expected.Body, actual.(*node.FinallyStmt).Body)
	case *node.ThrowStmt:
		f.Equal(expected.ThrowPos, actual.(*node.ThrowStmt).ThrowPos)
		f.EqualExpr(expected.Expr, actual.(*node.ThrowStmt).Expr)
	case *node.IncDecStmt:
		f.EqualExpr(expected.Expr, actual.(*node.IncDecStmt).Expr)
		f.Equal(expected.Token, actual.(*node.IncDecStmt).Token)
		f.Equal(expected.TokenPos, actual.(*node.IncDecStmt).TokenPos)
	case *node.ForStmt:
		f.EqualStmt(expected.Init, actual.(*node.ForStmt).Init)
		f.EqualExpr(expected.Cond, actual.(*node.ForStmt).Cond)
		f.EqualStmt(expected.Post, actual.(*node.ForStmt).Post)
		f.EqualStmt(expected.Body, actual.(*node.ForStmt).Body)
		f.Equal(expected.ForPos, actual.(*node.ForStmt).ForPos)
	case *node.ForInStmt:
		f.EqualExpr(expected.Key, actual.(*node.ForInStmt).Key)
		f.EqualExpr(expected.Value, actual.(*node.ForInStmt).Value)
		f.EqualExpr(expected.Iterable, actual.(*node.ForInStmt).Iterable)
		f.EqualStmt(expected.Body, actual.(*node.ForInStmt).Body)
		f.Equal(expected.ForPos, actual.(*node.ForInStmt).ForPos)
		f.EqualStmt(expected.Else, actual.(*node.ForInStmt).Else)
	case *node.ReturnStmt:
		f.EqualExpr(expected.Result, actual.(*node.ReturnStmt).Result)
		f.Equal(expected.ReturnPos, actual.(*node.ReturnStmt).ReturnPos)
	case *node.BranchStmt:
		f.EqualExpr(expected.Label, actual.(*node.BranchStmt).Label)
		f.Equal(expected.Token, actual.(*node.BranchStmt).Token)
		f.Equal(expected.TokenPos, actual.(*node.BranchStmt).TokenPos)
	case *node.MixedTextStmt:
		f.Equal(expected.Lit.Value, actual.(*node.MixedTextStmt).Lit.Value)
		f.Equal(expected.Lit.Pos, actual.(*node.MixedTextStmt).Lit.Pos)
		f.Equal(expected.Flags.String(), actual.(*node.MixedTextStmt).Flags.String(), "Flags")
	case *node.MixedValueStmt:
		f.Equal(expected.StartLit.Value, actual.(*node.MixedValueStmt).StartLit.Value)
		f.Equal(expected.StartLit.Pos, actual.(*node.MixedValueStmt).StartLit.Pos)
		f.Equal(expected.EndLit.Value, actual.(*node.MixedValueStmt).EndLit.Value)
		f.Equal(expected.EndLit.Pos, actual.(*node.MixedValueStmt).EndLit.Pos)
		f.EqualExpr(expected.Expr, actual.(*node.MixedValueStmt).Expr)
	case *node.ConfigStmt:
		f.Equal(expected.ConfigPos, actual.(*node.ConfigStmt).ConfigPos)
		f.Equal(expected.Options, actual.(*node.ConfigStmt).Options)
		f.Equal(len(expected.Elements), len(actual.(*node.ConfigStmt).Elements))
		for i, e := range expected.Elements {
			f.EqualExpr(e, actual.(*node.ConfigStmt).Elements[i])
		}
	case *node.CodeBeginStmt:
		f.Equal(expected.RemoveSpace, actual.(*node.CodeBeginStmt).RemoveSpace)
		f.Equal(expected.Lit.Pos, actual.(*node.CodeBeginStmt).Lit.Pos)
		f.Equal(expected.Lit.Value, actual.(*node.CodeBeginStmt).Lit.Value)
	case *node.CodeEndStmt:
		f.Equal(expected.RemoveSpace, actual.(*node.CodeEndStmt).RemoveSpace)
		f.Equal(expected.Lit.Pos, actual.(*node.CodeEndStmt).Lit.Pos)
		f.Equal(expected.Lit.Value, actual.(*node.CodeEndStmt).Lit.Value)
	case *node.FuncWithMethodsStmt:
		f.EqualExpr(&expected.FuncWithMethodsExpr, &actual.(*node.FuncWithMethodsStmt).FuncWithMethodsExpr)
	case *node.FuncStmt:
		f.EqualExpr(expected.Func, actual.(*node.FuncStmt).Func)
	default:
		panic(fmt.Errorf("unknown type: %T", expected))
	}
}

func (f *File) EqualExpr(expected, actual node.Expr) {
	t := f.t
	if expected == nil || reflect.ValueOf(expected).IsNil() {
		require.Nil(t, actual, "expected nil, but got not nil")
		return
	}
	require.NotNil(t, actual, "expected not nil, but got nil")
	require.IsType(t, expected, actual)

	switch expected := expected.(type) {
	case *node.IdentExpr:
		f.Equal(expected.Name, actual.(*node.IdentExpr).Name)
		f.Equal(expected.NamePos, actual.(*node.IdentExpr).NamePos)
	case *node.TypedIdentExpr:
		f.EqualExpr(expected.Ident, actual.(*node.TypedIdentExpr).Ident)
		var (
			etypes = make([]node.Expr, len(expected.Type))
			atypes = make([]node.Expr, len(actual.(*node.TypedIdentExpr).Type))
		)
		for i, expr := range expected.Type {
			etypes[i] = expr
		}
		for i, expr := range actual.(*node.TypedIdentExpr).Type {
			atypes[i] = expr
		}
		f.EqualExprs(etypes, atypes)
	case *node.TypeExpr:
		f.EqualExpr(expected.Expr, actual.(*node.TypeExpr).Expr)
	case *node.IntLit:
		f.Equal(expected.Value, actual.(*node.IntLit).Value)
		f.Equal(expected.ValuePos, actual.(*node.IntLit).ValuePos)
	case *node.FloatLit:
		f.Equal(expected.Value,
			actual.(*node.FloatLit).Value)
		f.Equal(expected.ValuePos, actual.(*node.FloatLit).ValuePos)
	case *node.DecimalLit:
		require.True(t, expected.Value.Equal(actual.(*node.DecimalLit).Value))
		f.Equal(expected.ValuePos, actual.(*node.DecimalLit).ValuePos)
	case *node.BoolLit:
		f.Equal(expected.Value, actual.(*node.BoolLit).Value)
		f.Equal(int(expected.ValuePos), int(actual.(*node.BoolLit).ValuePos))
	case *node.FlagLit:
		f.Equal(expected.Value, actual.(*node.FlagLit).Value)
		f.Equal(expected.ValuePos, actual.(*node.FlagLit).ValuePos)
	case *node.CharLit:
		f.Equal(expected.Value, actual.(*node.CharLit).Value)
		f.Equal(expected.ValuePos, actual.(*node.CharLit).ValuePos)
	case *node.StringLit:
		f.Equal(expected.Literal, actual.(*node.StringLit).Literal)
		f.Equal(expected.ValuePos, actual.(*node.StringLit).ValuePos)
	case *node.RawStringLit:
		f.Equal(expected.Value(), actual.(*node.RawStringLit).Value())
		f.Equal(expected.LiteralPos, actual.(*node.RawStringLit).LiteralPos)
	case *node.RawHeredocLit:
		f.Equal(expected.Literal, actual.(*node.RawHeredocLit).Literal)
		f.Equal(expected.LiteralPos, actual.(*node.RawHeredocLit).LiteralPos)
	case *node.ArrayExpr:
		f.Equal(expected.LBrack, actual.(*node.ArrayExpr).LBrack)
		f.Equal(expected.RBrack, actual.(*node.ArrayExpr).RBrack)
		f.EqualExprs(expected.Elements, actual.(*node.ArrayExpr).Elements)
	case *node.DictExpr:
		f.Equal(expected.LBrace, actual.(*node.DictExpr).LBrace)
		f.Equal(expected.RBrace, actual.(*node.DictExpr).RBrace)
		f.EqualDictElements(expected.Elements, actual.(*node.DictExpr).Elements)
	case *node.FuncDefLit:
		f.EqualExpr(expected.Expr, actual.(*node.FuncDefLit).Expr)
	case *node.NilLit:
		f.Equal(expected.TokenPos, actual.(*node.NilLit).TokenPos)
	case *node.ReturnExpr:
		f.Equal(expected.ReturnPos, actual.(*node.ReturnExpr).ReturnPos)
		f.EqualExpr(expected.Result, actual.(*node.ReturnExpr).Result)
	case *node.NullishSelectorExpr:
		f.EqualExpr(expected.Expr, actual.(*node.NullishSelectorExpr).Expr)
		f.EqualExpr(expected.Sel, actual.(*node.NullishSelectorExpr).Sel)
	case *node.BinaryExpr:
		f.EqualExpr(expected.LHS, actual.(*node.BinaryExpr).LHS)
		f.EqualExpr(expected.RHS, actual.(*node.BinaryExpr).RHS)
		f.Equal(expected.Token, actual.(*node.BinaryExpr).Token)
		f.Equal(expected.TokenPos, actual.(*node.BinaryExpr).TokenPos)
	case *node.UnaryExpr:
		f.EqualExpr(expected.Expr, actual.(*node.UnaryExpr).Expr)
		f.Equal(expected.Token, actual.(*node.UnaryExpr).Token)
		f.Equal(expected.TokenPos, actual.(*node.UnaryExpr).TokenPos)
	case *node.FuncExpr:
		f.EqualFuncType(expected.Type, actual.(*node.FuncExpr).Type)
		f.EqualStmt(expected.Body, actual.(*node.FuncExpr).Body)
		f.Equal(expected.LambdaPos, actual.(*node.FuncExpr).LambdaPos)
		f.EqualExpr(expected.BodyExpr, actual.(*node.FuncExpr).BodyExpr)
	case *node.CallExpr:
		actual := actual.(*node.CallExpr)
		f.EqualExpr(expected.Func, actual.Func)
		f.Equal(expected.LParen, actual.LParen)
		f.Equal(expected.RParen, actual.RParen)
		f.EqualExprs(expected.Args.Values, actual.Args.Values)

		if expected.Args.Var == nil && actual.Args.Var != nil {
			require.Nil(t, expected.Args.Var)
		}

		if expected.Args.Var != nil && actual.Args.Var == nil {
			require.NotNil(t, expected.Args.Var)
		}

		if expected.Args.Var != nil && actual.Args.Var != nil {
			f.Equal(expected.Args.Var.TokenPos,
				actual.Args.Var.TokenPos)
			f.EqualExpr(expected.Args.Var.Value,
				actual.Args.Var.Value)
		}

		f.EqualNamedArgsNames(expected.NamedArgs.Names, actual.NamedArgs.Names)
		f.EqualExprs(expected.NamedArgs.Values, actual.NamedArgs.Values)
	case *node.ParenExpr:
		f.EqualExpr(expected.Expr, actual.(*node.ParenExpr).Expr)
		f.Equal(expected.LParen, actual.(*node.ParenExpr).LParen)
		f.Equal(expected.RParen, actual.(*node.ParenExpr).RParen)
	case *node.IndexExpr:
		f.EqualExpr(expected.X, actual.(*node.IndexExpr).X)
		f.EqualExpr(expected.Index, actual.(*node.IndexExpr).Index)
		f.Equal(expected.LBrack, actual.(*node.IndexExpr).LBrack)
		f.Equal(expected.RBrack, actual.(*node.IndexExpr).RBrack)
	case *node.SliceExpr:
		f.EqualExpr(expected.Expr, actual.(*node.SliceExpr).Expr)
		f.EqualExpr(expected.Low, actual.(*node.SliceExpr).Low)
		f.EqualExpr(expected.High, actual.(*node.SliceExpr).High)
		f.Equal(expected.LBrack, actual.(*node.SliceExpr).LBrack)
		f.Equal(expected.RBrack, actual.(*node.SliceExpr).RBrack)
	case *node.SelectorExpr:
		f.EqualExpr(expected.X, actual.(*node.SelectorExpr).X)
		f.EqualExpr(expected.Sel, actual.(*node.SelectorExpr).Sel)
	case *node.ImportExpr:
		f.EqualExpr(&expected.CallExpr, &actual.(*node.ImportExpr).CallExpr)
	case *node.EmbedExpr:
		f.Equal(expected.Path, actual.(*node.EmbedExpr).Path)
		f.Equal(int(expected.TokenPos), int(actual.(*node.EmbedExpr).TokenPos))
		f.Equal(expected.Token, actual.(*node.EmbedExpr).Token)
	case *node.CondExpr:
		f.EqualExpr(expected.Cond, actual.(*node.CondExpr).Cond)
		f.EqualExpr(expected.True, actual.(*node.CondExpr).True)
		f.EqualExpr(expected.False, actual.(*node.CondExpr).False)
		f.Equal(expected.QuestionPos, actual.(*node.CondExpr).QuestionPos)
		f.Equal(expected.ColonPos, actual.(*node.CondExpr).ColonPos)
	case *node.CalleeKeywordExpr:
		f.Equal(expected.Literal, actual.(*node.CalleeKeywordExpr).Literal)
		f.Equal(expected.TokenPos, actual.(*node.CalleeKeywordExpr).TokenPos)
	case *node.ArgsKeywordExpr:
		f.Equal(expected.Literal, actual.(*node.ArgsKeywordExpr).Literal)
		f.Equal(expected.TokenPos, actual.(*node.ArgsKeywordExpr).TokenPos)
	case *node.NamedArgsKeywordExpr:
		f.Equal(expected.Literal, actual.(*node.NamedArgsKeywordExpr).Literal)
		f.Equal(expected.TokenPos, actual.(*node.NamedArgsKeywordExpr).TokenPos)
	case *node.ClosureExpr:
		f.Equal(expected.Lambda, actual.(*node.ClosureExpr).Lambda)
		f.EqualTypedIdents(expected.Params.Args.Values, actual.(*node.ClosureExpr).Params.Args.Values)
		f.EqualNamedArgs(&expected.Params.NamedArgs, &actual.(*node.ClosureExpr).Params.NamedArgs)
		f.EqualExpr(expected.Body, actual.(*node.ClosureExpr).Body)
	case *node.BlockExpr:
		f.EqualStmt(expected.BlockStmt, actual.(*node.BlockExpr).BlockStmt)
	case *node.KeyValueLit:
		f.EqualExpr(expected.Key, actual.(*node.KeyValueLit).Key)
		f.EqualExpr(expected.Value, actual.(*node.KeyValueLit).Value)
	case *node.KeyValuePairLit:
		f.EqualExpr(expected.Key, actual.(*node.KeyValuePairLit).Key)
		f.EqualExpr(expected.Value, actual.(*node.KeyValuePairLit).Value)
	case *node.FuncWithMethodsExpr:
		actual := actual.(*node.FuncWithMethodsExpr)
		f.Equal(expected.FuncToken, actual.FuncToken)
		f.EqualExpr(expected.NameExpr, actual.NameExpr)
		f.Equal(expected.LBrace, actual.LBrace)
		f.Equal(expected.RBrace, actual.RBrace)
		f.Equal(len(expected.Methods), len(actual.Methods), "methods count")

		for i, em := range expected.Methods {
			msg := fmt.Sprintf("method[%d]", i)
			am := actual.Methods[i]
			f.Equal(&em.Params, &am.Params, msg+".Params")
			f.Equal(&em.LambdaPos, &am.LambdaPos, msg+".LambdaPos")
			f.EqualStmt(em.Body, am.Body)
			f.EqualExpr(em.BodyExpr, am.BodyExpr)
		}
	case *node.MethodExpr:
		f.EqualExpr(expected.Expr, actual.(*node.MethodExpr).Expr)
	case *node.ComputedExpr:
		actual := actual.(*node.ComputedExpr)
		f.Equal(expected.StartPos, actual.StartPos)
		f.Equal(expected.EndPos, actual.EndPos)
		f.EqualStmts(expected.Stmts, actual.Stmts)
	case *node.IsMainLit:
		f.Equal(expected.TokenPos, actual.(*node.IsMainLit).TokenPos)
	case *node.SymbolLit:
		f.Equal(&expected.Lit, &actual.(*node.SymbolLit).Lit)
	default:
		panic(fmt.Errorf("unknown type: %T", expected))
	}
}

func (f *File) EqualFuncParams(expected, actual *node.FuncParams) {
	f.Equal(expected.LParen, actual.LParen)
	f.Equal(expected.RParen, actual.RParen)
	f.EqualTypedIdents(expected.Args.Values, actual.Args.Values)
	f.EqualExpr(expected.Args.Var, actual.Args.Var)
	f.EqualNamedArgs(&expected.NamedArgs, &actual.NamedArgs)
}

func (f *File) EqualFuncType(expected, actual *node.FuncType) {
	f.Equal(expected.NameExpr, actual.NameExpr)
	f.EqualFuncParams(&expected.Params, &actual.Params)
}

func (f *File) EqualNamedArgs(expected, actual *node.NamedArgsList) {
	if expected == nil && actual == nil {
		return
	}
	require.NotNil(f.t, expected, "expected is nil")
	require.NotNil(f.t, actual, "actual is nil")

	f.EqualExpr(expected.Var, actual.Var)
	f.EqualTypedIdents(expected.Names, actual.Names)
	f.EqualExprs(expected.Values, actual.Values)
}

func (f *File) EqualNamedArgsNames(expected, actual []*node.NamedArgExpr) {
	f.Equal(len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		f.EqualExpr(expected[i].Ident, actual[i].Ident)
		f.EqualExpr(expected[i].Lit, actual[i].Lit)
		f.EqualExpr(expected[i].Exp, actual[i].Exp)
		require.Equal(f.t, expected[i].Var, actual[i].Var)
	}
}

func (f *File) EqualTypedIdents(expected, actual []*node.TypedIdentExpr) {
	f.Equal(len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		f.EqualExpr(expected[i], actual[i])
	}
}

func (f *File) EqualExprs(expected, actual []node.Expr) {
	f.Equal(len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		f.EqualExpr(expected[i], actual[i])
	}
}

func (f *File) EqualStmts(expected, actual []node.Stmt) {
	f.Equal(len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		f.EqualStmt(expected[i], actual[i])
	}
}

func (f *File) EqualDictElements(
	expected, actual []*node.DictElementLit,
) {
	f.Equal(len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		f.EqualExpr(expected[i].Key, actual[i].Key)
		f.Equal(expected[i].ColonPos, actual[i].ColonPos)
		f.EqualExpr(expected[i].Value, actual[i].Value)
	}
}
