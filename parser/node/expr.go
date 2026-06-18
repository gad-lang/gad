// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

// Copyright (c) 2019 Daniel Kang.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE.tengo file.

// Copyright 2009 The ToInterface Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.golang file.

package node

import (
	"bytes"
	"strings"

	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/repr"
	"github.com/gad-lang/gad/token"
)

// BadExpr represents a bad expression.
type BadExpr struct {
	From source.Pos
	To   source.Pos
}

func (e *BadExpr) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *BadExpr) Pos() source.Pos {
	return e.From
}

// End returns the position of first character immediately after the node.
func (e *BadExpr) End() source.Pos {
	return e.To
}

func (e *BadExpr) String() string {
	return repr.Quote("bad expression")
}

func (e *BadExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(e.String())
}

// BinaryExpr represents a binary operator expression.
type BinaryExpr struct {
	LHS      Expr
	RHS      Expr
	Token    token.Token
	TokenPos source.Pos
}

func (e *BinaryExpr) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *BinaryExpr) Pos() source.Pos {
	return e.LHS.Pos()
}

// End returns the position of first character immediately after the node.
func (e *BinaryExpr) End() source.Pos {
	return e.RHS.End()
}

func (e *BinaryExpr) String() string {
	return "(" + e.LHS.String() + " " + e.Token.String() +
		" " + e.RHS.String() + ")"
}

func (e *BinaryExpr) WriteCode(ctx *CodeWriteContext) {
	e.WriteCodeWithParen(ctx, true)
}

func (e *BinaryExpr) WriteCodeWithParen(ctx *CodeWriteContext, paren bool) {
	if paren {
		ctx.WriteSingleByte('(')
	}
	e.LHS.WriteCode(ctx)
	ctx.WriteString(" " + e.Token.String() + " ")
	e.RHS.WriteCode(ctx)
	if paren {
		ctx.WriteSingleByte(')')
	}
}

type BoolExpr interface {
	Expr
	Bool() bool
}

// CallExpr represents a function call expression.
type CallExpr struct {
	Func Expr
	CallArgs
}

func (e *CallExpr) GetCallArgs() *CallArgs {
	return &e.CallArgs
}

func (e *CallExpr) ExprNode() {}

// CallPos returns the position of the fist valid call pos
func (e *CallExpr) CallPos() source.Pos {
	if e.CallArgs.LParen.IsValid() {
		return e.CallArgs.LParen
	}
	return e.Func.Pos()
}

// Pos returns the position of first character belonging to the node.
func (e *CallExpr) Pos() source.Pos {
	return e.Func.Pos()
}

// End returns the position of first character immediately after the node.
func (e *CallExpr) End() source.Pos {
	return e.RParen + 1
}

func (e *CallExpr) String() string {
	var s string
	if f, _ := e.Func.(*FuncExpr); f != nil {
		s = EParen(f, 0, 0).String()
	} else {
		s = e.Func.String()
	}
	var buf = bytes.NewBufferString(s)
	e.CallArgs.StringW(buf)
	return buf.String()
}

func (e *CallExpr) WriteCode(ctx *CodeWriteContext) {
	if f, _ := e.Func.(*FuncExpr); f != nil {
		(EParen(f, 0, 0)).WriteCode(ctx)
	} else {
		e.Func.WriteCode(ctx)
	}
	e.CallArgs.WriteCode(ctx)
}

func WithCallArgs[T interface{ GetCallArgs() *CallArgs }](e T, do func(args *CallArgs)) T {
	do(e.GetCallArgs())
	return e
}

// CondExpr represents a ternary conditional expression.
type CondExpr struct {
	Cond        Expr
	True        Expr
	False       Expr
	QuestionPos source.Pos
	ColonPos    source.Pos
}

func (e *CondExpr) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *CondExpr) Pos() source.Pos {
	return e.Cond.Pos()
}

// End returns the position of first character immediately after the node.
func (e *CondExpr) End() source.Pos {
	return e.False.End()
}

func (e *CondExpr) String() string {
	return "(" + e.Cond.String() + " ? " + e.True.String() +
		" : " + e.False.String() + ")"
}

func (e *CondExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteSingleByte('(')
	e.Cond.WriteCode(ctx)
	ctx.WriteString(" ? ")
	e.True.WriteCode(ctx)
	ctx.WriteString(" : ")
	e.False.WriteCode(ctx)
	ctx.WriteSingleByte(')')
}

// IdentExpr represents an identifier.
type IdentExpr struct {
	Name    string
	NamePos source.Pos
	Empty   bool
}

func (e *IdentExpr) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *IdentExpr) Pos() source.Pos {
	return e.NamePos
}

// End returns the position of first character immediately after the node.
func (e *IdentExpr) End() source.Pos {
	return source.Pos(int(e.NamePos) + len(e.Name))
}

func (e *IdentExpr) String() string {
	if e != nil {
		return e.Name
	}
	return nullRep
}

func (e *IdentExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(e.Name)
}

type TypeExpr struct {
	Expr
}

func (e *TypeExpr) ExprNode() {}

func (e *TypeExpr) Pos() source.Pos {
	return e.Expr.Pos()
}

func (e *TypeExpr) End() source.Pos {
	return e.Expr.End()
}

func (e *TypeExpr) Ident() *IdentExpr {
	var walk func(e Expr) *IdentExpr
	walk = func(e Expr) *IdentExpr {
		switch e := e.(type) {
		case *IdentExpr:
			return e
		case *IndexExpr:
			return walk(e.X)
		case *SelectorExpr:
			return walk(e.X)
		}
		return nil
	}
	return walk(e.Expr)
}

type TypedIdentExpr struct {
	Ident *IdentExpr
	Type  []*TypeExpr
}

func (e *TypedIdentExpr) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *TypedIdentExpr) Pos() source.Pos {
	if e.Ident != nil {
		return e.Ident.Pos()
	}
	if len(e.Type) > 0 {
		return e.Type[0].Pos()
	}
	return source.NoPos
}

// End returns the position of first character immediately after the node.
func (e *TypedIdentExpr) End() source.Pos {
	if len(e.Type) > 0 {
		return e.Type[len(e.Type)-1].End()
	}
	if e.Ident != nil {
		return e.Ident.End()
	}
	return source.NoPos
}

func (e *TypedIdentExpr) String() string {
	if e != nil {
		if l := len(e.Type); l == 0 {
			return e.Ident.String()
		} else {
			var s = make([]string, l)
			for i, t := range e.Type {
				s[i] = t.String()
			}
			return e.Ident.String() + " " + strings.Join(s, "|")
		}
	}
	return nullRep
}

func (e *TypedIdentExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(e.String())
}

// ImportExpr represents an import expression
type ImportExpr struct {
	CallExpr
}

func (e *ImportExpr) ModuleName() string {
	return e.Args.Values[0].(*StrLit).Value()
}

func (e *ImportExpr) ExprNode() {}

func (e *ImportExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(e.String())
}

func (e *ImportExpr) Build() (moduleName string, args CallArgs) {
	moduleName = e.ModuleName()
	call := e.CallExpr
	call.Args.Values = call.Args.Values[1:]
	args = call.CallArgs
	return
}

// EmbedExpr represents an embed expression that embeds external files or directories
// into the compiled program at compile time. It resolves to an EmbeddedNodeFS at runtime.
//
// Syntax: embed(path [; namedParams...])
//
// The path must be a single string or symbol literal specifying the file or directory to embed.
//
// Named parameters (all optional):
//   - sources: array of string/symbol literals specifying source paths
//   - includes: array of string/symbol literals for file inclusion patterns
//   - excludes: array of string/symbol literals for file exclusion patterns
//   - includes_re: array of string/symbol literals for regex file inclusion patterns
//   - excludes_re: array of string/symbol literals for regex file exclusion patterns
//   - config_file: string/symbol literal pointing to a YAML config file with the above named params
//   - tree: flag (no value) to embed directory tree recursively
//
// Example:
//
//	embed("file.txt")
//	embed("dir"; tree)
//	embed("dir"; includes=["*.go"], excludes=["*_test.go"])
//	embed("dir"; sources=["a", "b"], tree)
//	embed("dir"; config_file="embed.yaml")
type EmbedExpr struct {
	Args     CallArgs
	Token    token.Token
	TokenPos source.Pos
}

func (e *EmbedExpr) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *EmbedExpr) Pos() source.Pos {
	return e.TokenPos
}

// End returns the position of first character immediately after the node.
func (e *EmbedExpr) End() source.Pos {
	return e.Args.End()
}

func (e *EmbedExpr) String() string {
	return `embed` + e.Args.String()
}

// Path returns the file or directory path specified as the first argument to embed.
func (e *EmbedExpr) Path() (s string) {
	switch t := e.Args.Args.Values[0].(type) {
	case *StrLit:
		s = t.Value()
	case *SymbolLit:
		s = t.Value()
	}
	return
}

// Sources returns the list of source paths from the "sources" named parameter.
func (e *EmbedExpr) Sources() []string {
	return e.getStrings("sources")
}

// Includes returns the list of inclusion patterns from the "includes" named parameter.
func (e *EmbedExpr) Includes() []string {
	return e.getStrings("includes")
}

// Excludes returns the list of exclusion patterns from the "excludes" named parameter.
func (e *EmbedExpr) Excludes() []string {
	return e.getStrings("excludes")
}

// IncludesRe returns the list of regex inclusion patterns from the "includes_re" named parameter.
func (e *EmbedExpr) IncludesRe() []string {
	return e.getStrings("includes_re")
}

// ExcludesRe returns the list of regex exclusion patterns from the "excludes_re" named parameter.
func (e *EmbedExpr) ExcludesRe() []string {
	return e.getStrings("excludes_re")
}

func (e *EmbedExpr) getNvalue(nameArg string) (v Expr, ok bool) {
	for i, expr := range e.Args.NamedArgs.Names {
		if expr.Ident != nil && expr.Ident.Name == nameArg {
			ok = true
			v = e.Args.NamedArgs.Values[i]
			break
		}
	}
	return
}

func (e *EmbedExpr) getStrings(nameArg string) (s []string) {
	if v, _ := e.getNvalue(nameArg); v != nil {
		switch t := v.(type) {
		case *ArrayExpr:
			for _, av := range t.Elements {
				switch a := av.(type) {
				case *StrLit:
					s = append(s, a.Value())
				case *SymbolLit:
					s = append(s, a.Value())
				}
			}
		}
	}
	return
}

// ConfigFile returns the config file path from the "config_file" named parameter.
func (e *EmbedExpr) ConfigFile() string {
	if v, ok := e.getNvalue("config_file"); ok && v != nil {
		switch t := v.(type) {
		case *StrLit:
			return t.Value()
		case *SymbolLit:
			return t.Value()
		}
	}
	return ""
}

// Tree returns true if the "tree" flag is set, indicating recursive directory embedding.
func (e *EmbedExpr) Tree() bool {
	if v, ok := e.getNvalue("tree"); ok {
		return v == nil
	}
	return false
}

func (e *EmbedExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString("embed")
	e.Args.WriteCode(ctx)
}

// IndexExpr represents an index expression.
type IndexExpr struct {
	X      Expr
	LBrack source.Pos
	Index  Expr
	RBrack source.Pos
}

func (e *IndexExpr) GetX() Expr {
	return e.X
}

func (e *IndexExpr) GetY() Expr {
	return e.Index
}

func (e *IndexExpr) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *IndexExpr) Pos() source.Pos {
	return e.X.Pos()
}

// End returns the position of first character immediately after the node.
func (e *IndexExpr) End() source.Pos {
	return e.RBrack + 1
}

func (e *IndexExpr) String() string {
	var index string
	if e.Index != nil {
		index = e.Index.String()
	}
	return e.X.String() + "[" + index + "]"
}

func (e *IndexExpr) WriteCode(ctx *CodeWriteContext) {
	e.X.WriteCode(ctx)
	ctx.WriteString("[")
	e.Index.WriteCode(ctx)
	ctx.WriteString("]")
}

// ParenExpr represents a parenthesis wrapped expression.
type ParenExpr struct {
	Expr   Expr
	LParen Token
	RParen Token
}

func (e *ParenExpr) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *ParenExpr) Pos() source.Pos {
	return e.LParen.Pos
}

// End returns the position of first character immediately after the node.
func (e *ParenExpr) End() source.Pos {
	return e.RParen.Pos + 1
}

func (e *ParenExpr) String() string {
	var s string
	switch t := e.Expr.(type) {
	case *ParenExpr:
		if e.LParen.Token == token.LParen {
			s = t.Expr.String()
		} else {
			s = t.String()
		}
	case *BinaryExpr:
		s = t.LHS.String() + " " + t.Token.String() + " " + t.RHS.String()
	default:
		s = t.String()
	}
	return e.LParen.Token.String() + s + e.RParen.Token.String()
}

func (e *ParenExpr) WriteCode(ctx *CodeWriteContext) {
	switch t := e.Expr.(type) {
	case *ParenExpr:
		if e.LParen.Token == token.LParen {
			e.Expr.WriteCode(ctx)
		} else {
			ctx.WriteString(e.LParen.Token.String())
			t.WriteCode(ctx)
			ctx.WriteString(e.RParen.Token.String())
		}
	case *BinaryExpr:
		if e.LParen.Token == token.LParen {
			t.WriteCode(ctx)
		} else {
			ctx.WriteString(e.LParen.Token.String())
			ctx.WriteString(t.LHS.String() + " " + t.Token.String() + " " + t.RHS.String())
			ctx.WriteString(e.RParen.Token.String())
		}
	default:
		ctx.WriteString(e.LParen.Token.String())
		e.Expr.WriteCode(ctx)
		ctx.WriteString(e.RParen.Token.String())
	}
}

func (e *ParenExpr) ToMultiParenExpr() *MultiParenExpr {
	return &MultiParenExpr{
		PositionalElements: Exprs{e.Expr},
		LParen:             e.LParen,
		RParen:             e.RParen,
	}
}

func (e *ParenExpr) Items() Exprs {
	return Exprs{e.Expr}
}

// MultiParenExpr represents a parenthesis wrapped expressions.
type MultiParenExpr struct {
	LParen             Token
	RParen             Token
	PositionalElements Exprs
	NamedElements      Exprs
}

func (e *MultiParenExpr) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *MultiParenExpr) Pos() source.Pos {
	return e.LParen.Pos
}

// End returns the position of first character immediately after the node.
func (e *MultiParenExpr) End() source.Pos {
	return e.RParen.Pos + 1
}

func (e *MultiParenExpr) String() string {
	var s strings.Builder
	s.WriteString(e.LParen.Token.String() + ",")

	if len(e.PositionalElements) > 0 {
		s.WriteString(" ")
	}

	for i, expr := range e.PositionalElements {
		if i > 0 {
			s.WriteString(", ")
		}
		s.WriteString(expr.String())
	}

	if len(e.NamedElements) > 0 {
		s.WriteString("; ")
		for i, expr := range e.NamedElements {
			if i > 0 {
				s.WriteString(", ")
			}
			s.WriteString(expr.String())
		}
	}
	s.WriteString(e.RParen.Token.String())
	return s.String()
}

func (e *MultiParenExpr) ToMultiParenExpr() *MultiParenExpr {
	return e
}

func (e *MultiParenExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(e.LParen.Token.String() + ",")
	var pl, nl = len(e.PositionalElements), len(e.NamedElements)
	if pl+nl > 0 {
		inNewLine := ctx.DecideNewLineFunc(
			CodeWriteContextFlagFormatParemValuesInNewLine, pl+nl, 1, func() {
				for i := 0; i < pl; i++ {
					if i > 0 {
						ctx.WriteString(", ")
					}
					e.PositionalElements[i].WriteCode(ctx)
				}
				if pl > 0 && nl > 0 {
					ctx.WriteString("; ")
				}
				for i := 0; i < nl; i++ {
					if i > 0 {
						ctx.WriteString(", ")
					}
					e.NamedElements[i].WriteCode(ctx)
				}
			})
		if pl > 0 {
			ctx.WriteItems(inNewLine,
				len(e.PositionalElements),
				func(i int) {
					e.PositionalElements[i].WriteCode(ctx)
				},
				func(newLine bool) {
					if len(e.PositionalElements) > 0 {
						ctx.WriteString("; ")
					} else if newLine {
						ctx.WriteSecondLine()
					}
				})
		}
		if nl > 0 {
			ctx.WriteItems(inNewLine,
				len(e.PositionalElements),
				func(i int) {
					e.PositionalElements[i].WriteCode(ctx)
				},
				func(newLine bool) {
					if newLine {
						ctx.WriteSecondLine()
					}
				})
		}
	}
	ctx.WriteString(e.RParen.Token.String())
}

func (e *MultiParenExpr) ToCallArgs(strict bool) (args *CallArgs, err *NodeError) {
	args = new(CallArgs)
	args.LParen = e.LParen.Pos
	args.RParen = e.RParen.Pos

	var n Expr

	for _, n = range e.PositionalElements {
		switch t := n.(type) {
		case *ArgVarLit:
			args.Args.Var = t
		default:
			args.Args.Values = append(args.Args.Values, t)
		}
	}

	for _, n = range e.NamedElements {
		switch t := n.(type) {
		case *KeyValueLit:
			na := &NamedArgExpr{}
			switch t := t.Key.(type) {
			case *StrLit:
				na.Lit = t
			case *IdentExpr:
				na.Ident = t
			case *ParenExpr:
				na.Exp = t
			default:
				na.Exp = EParen(t, 0, 0)
			}
			args.NamedArgs.Names = append(args.NamedArgs.Names, na)
			args.NamedArgs.Values = append(args.NamedArgs.Values, t.Value)
		case *KeyValuePairLit:
			switch t2 := t.Key.(type) {
			case *IdentExpr:
				args.NamedArgs.Names = append(args.NamedArgs.Names, &NamedArgExpr{Ident: t2})
			case *StrLit:
				args.NamedArgs.Names = append(args.NamedArgs.Names, &NamedArgExpr{Lit: t2})
			case *TypedIdentExpr:
				if strict {
					err = NewExpectedError(t2, &StrLit{}, &IdentExpr{})
					return
				}
				args.NamedArgs.Names = append(args.NamedArgs.Names, &NamedArgExpr{Ident: t2.Ident})
			default:
				if strict {
					err = NewExpectedError(t2, &StrLit{}, &IdentExpr{})
				} else {
					err = NewExpectedError(t2, &StrLit{}, &IdentExpr{}, &TypedIdentExpr{})
				}
				return
			}
			args.NamedArgs.Values = append(args.NamedArgs.Values, t.Value)
		case *NamedArgVarLit:
			args.NamedArgs.Names = append(args.NamedArgs.Names, &NamedArgExpr{Var: true, Exp: t.Value})
			args.NamedArgs.Values = append(args.NamedArgs.Values, nil)
		}
	}
	return
}

func (e *MultiParenExpr) ToFuncParams() (params FuncParams, err *NodeError) {
	params.LParen = e.LParen.Pos
	params.RParen = e.RParen.Pos

	var (
		i int
		n Expr
	)

exps:
	for i, n = range e.PositionalElements {
		switch t := n.(type) {
		case *IdentExpr:
			params.Args.Values = append(params.Args.Values, &TypedIdentExpr{Ident: t})
		case *TypedIdentExpr:
			params.Args.Values = append(params.Args.Values, t)
		case *ArgVarLit:
			switch t2 := t.Value.(type) {
			case *IdentExpr:
				params.Args.Var = &TypedIdentExpr{Ident: t2}
				break exps
			case *TypedIdentExpr:
				params.Args.Var = t2
				break exps
			default:
				err = NewExpectedError(t.Value, &IdentExpr{})
				return
			}
		default:
			err = NewExpectedError(t, &IdentExpr{}, &TypedIdentExpr{}, &ArgVarLit{})
			return
		}
	}

	if i < len(e.PositionalElements)-1 {
		err = NewUnExpectedError(e.PositionalElements[i])
		return
	}

nexps:
	for i, n = range e.NamedElements {
		switch t := n.(type) {
		case *KeyValuePairLit:
			switch t2 := t.Key.(type) {
			case *IdentExpr:
				params.NamedArgs.Names = append(params.NamedArgs.Names, &TypedIdentExpr{Ident: t2})
				params.NamedArgs.Values = append(params.NamedArgs.Values, t.Value)
			case *TypedIdentExpr:
				params.NamedArgs.Names = append(params.NamedArgs.Names, t2)
				params.NamedArgs.Values = append(params.NamedArgs.Values, t.Value)
			default:
				err = NewExpectedError(t2, &IdentExpr{}, &TypedIdentExpr{})
				return
			}
		case *NamedArgVarLit:
			switch t2 := t.Value.(type) {
			case *IdentExpr:
				params.NamedArgs.Var = t2
				break nexps
			case *TypedIdentExpr:
				params.NamedArgs.Var = t2.Ident
				break nexps
			default:
				err = NewExpectedError(t2, &IdentExpr{}, &TypedIdentExpr{})
				return
			}
		default:
			err = NewExpectedError(t, &KeyValuePairLit{}, &NamedArgVarLit{})
			return
		}
	}

	if i < len(e.NamedElements)-1 {
		err = NewUnExpectedError(e.NamedElements[i])
	}

	return
}

// SelectorExpr represents a selector expression.
type SelectorExpr struct {
	X   Expr
	Sel Expr
}

func (e *SelectorExpr) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *SelectorExpr) Pos() source.Pos {
	return e.X.Pos()
}

// End returns the position of first character immediately after the node.
func (e *SelectorExpr) End() source.Pos {
	return e.Sel.End()
}

func (e *SelectorExpr) String() string {
	r := e.X.String() + "."
	if s, _ := e.Sel.(*StrLit); s != nil {
		if s.CanIdent() {
			return r + s.Value()
		}
		return r + "(" + s.Literal + ")"
	}
	return r + e.Sel.String()
}

func (e *SelectorExpr) GetX() Expr {
	return e.X
}

func (e *SelectorExpr) GetY() Expr {
	return e.Sel
}

func (e *SelectorExpr) WriteCode(ctx *CodeWriteContext) {
	e.X.WriteCode(ctx)
	ctx.WriteSingleByte('.')
	if s, _ := e.Sel.(*StrLit); s != nil {
		if s.CanIdent() {
			ctx.WriteString(s.Value())
		} else {
			ctx.WriteString("(", s.Literal, ")")
		}
	} else {
		e.Sel.WriteCode(ctx)
	}
}

// NullishSelectorExpr represents a selector expression.
type NullishSelectorExpr struct {
	Expr Expr
	Sel  Expr
}

func (e *NullishSelectorExpr) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *NullishSelectorExpr) Pos() source.Pos {
	return e.Expr.Pos()
}

// End returns the position of first character immediately after the node.
func (e *NullishSelectorExpr) End() source.Pos {
	return e.Sel.End()
}

func (e *NullishSelectorExpr) String() string {
	r := e.Expr.String() + "?."
	if s, _ := e.Sel.(*StrLit); s != nil {
		if s.CanIdent() {
			return r + s.Value()
		}
		return r + "(" + s.Literal + ")"
	}
	return r + e.Sel.String()
}

func (e *NullishSelectorExpr) GetX() Expr {
	return e.Expr
}

func (e *NullishSelectorExpr) GetY() Expr {
	return e.Sel
}

func (e *NullishSelectorExpr) WriteCode(ctx *CodeWriteContext) {
	e.Expr.WriteCode(ctx)
	ctx.WriteString("?.")
	if s, _ := e.Sel.(*StrLit); s != nil {
		if s.CanIdent() {
			ctx.WriteString(s.Value())
		} else {
			ctx.WriteString("(", s.Literal, ")")
		}
	} else {
		e.Sel.WriteCode(ctx)
	}
}

// SliceExpr represents a slice expression.
type SliceExpr struct {
	Expr   Expr
	LBrack source.Pos
	Low    Expr
	High   Expr
	RBrack source.Pos
}

func (e *SliceExpr) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *SliceExpr) Pos() source.Pos {
	return e.Expr.Pos()
}

// End returns the position of first character immediately after the node.
func (e *SliceExpr) End() source.Pos {
	return e.RBrack + 1
}

func (e *SliceExpr) String() string {
	var low, high string
	if e.Low != nil {
		low = e.Low.String()
	}
	if e.High != nil {
		high = e.High.String()
	}
	return e.Expr.String() + "[" + low + ":" + high + "]"
}

func (e *SliceExpr) WriteCode(ctx *CodeWriteContext) {
	e.Expr.WriteCode(ctx)
	ctx.WriteSingleByte('[')
	if e.Low != nil {
		e.Low.WriteCode(ctx)
	}
	ctx.WriteSingleByte(':')
	if e.High != nil {
		e.High.WriteCode(ctx)
	}
	ctx.WriteSingleByte(']')
}

// UnaryExpr represents an unary operator expression.
type UnaryExpr struct {
	Expr     Expr
	Token    token.Token
	TokenPos source.Pos
}

func (e *UnaryExpr) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *UnaryExpr) Pos() source.Pos {
	return e.Expr.Pos()
}

// End returns the position of first character immediately after the node.
func (e *UnaryExpr) End() source.Pos {
	return e.Expr.End()
}

func (e *UnaryExpr) String() string {
	if e.Token == token.Null {
		return "(" + e.Expr.String() + " == nil)"
	}
	if e.Token == token.NotNull {
		return "(" + e.Expr.String() + " != nil)"
	}
	return "(" + e.Token.String() + e.Expr.String() + ")"
}

func (e *UnaryExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteSingleByte('(')

	switch e.Token {
	case token.Null:
		e.Expr.WriteCode(ctx)
		ctx.WriteString(" == nil")
	case token.NotNull:
		e.Expr.WriteCode(ctx)
		ctx.WriteString(" != nil")
	default:
		ctx.WriteString(e.Token.String())
		e.Expr.WriteCode(ctx)
	}

	ctx.WriteSingleByte(')')
}

// CallExprPositionalArgs represents a call expression arguments.
type CallExprPositionalArgs struct {
	Values []Expr
	Var    *ArgVarLit
}

func (a *CallExprPositionalArgs) AppendValues(e ...Expr) {
	a.Values = append(a.Values, e...)
}

func (a *CallExprPositionalArgs) Valid() bool {
	return len(a.Values) > 0 || a.Var != nil
}

func (a *CallExprPositionalArgs) String() string {
	var s []string
	for _, v := range a.Values {
		s = append(s, v.String())
	}
	if a.Var != nil {
		s = append(s, a.Var.String())
	}
	return strings.Join(s, ", ")
}

func (a *CallExprPositionalArgs) WriteCodeWithNamed(ctx *CodeWriteContext, inNewLine, hasNamed bool) {
	values := a.Values

	if a.Var != nil {
		values = append(values, a.Var)
	}

	if l := len(values); l > 0 {
		if l == 1 && !hasNamed {
			values[0].WriteCode(ctx)
		} else {
			ctx.WriteItems(
				inNewLine,
				len(values),
				func(i int) {
					values[i].WriteCode(ctx)
				},
				func(newLine bool) {
					if newLine {
						ctx.WriteSecondLine()
					}
				})
		}
	}
}

type NamedArgExpr struct {
	Lit   *StrLit
	Ident *IdentExpr
	Exp   Expr
	Var   bool
}

func (e *NamedArgExpr) Name() string {
	if e.Lit != nil {
		return e.Lit.Value()
	}
	return e.Ident.Name
}

func (e *NamedArgExpr) String() string {
	var prefix string
	if e.Var {
		prefix = "**"
	}
	return prefix + e.Expr().String()
}

func (e *NamedArgExpr) Expr() Expr {
	if e.Lit != nil {
		return e.Lit
	}
	if e.Ident != nil {
		return e.Ident
	}
	return e.Exp
}

func (e *NamedArgExpr) WriteCode(ctx *CodeWriteContext) {
	if e.Var {
		ctx.WriteString("**")
	}
	e.Expr().WriteCode(ctx)
}

// CallExprNamedArgs represents a call expression keyword arguments.
type CallExprNamedArgs struct {
	Names  []*NamedArgExpr
	Values []Expr
}

func (a *CallExprNamedArgs) Var() *NamedArgExpr {
	if len(a.Names) > 0 {
		v := a.Names[len(a.Names)-1]
		if v.Var {
			return v
		}
	}
	return nil
}

func (a *CallExprNamedArgs) Append(name *NamedArgExpr, value Expr) *CallExprNamedArgs {
	a.Names = append(a.Names, name)
	a.Values = append(a.Values, value)
	return a
}

func (a *CallExprNamedArgs) AppendE(name Expr, value Expr) *CallExprNamedArgs {
	ne := new(NamedArgExpr)
	switch t := name.(type) {
	case *IdentExpr:
		ne.Ident = t
	case *StrLit:
		ne.Lit = t
	default:
		ne.Exp = t
	}
	a.Names = append(a.Names, ne)
	a.Values = append(a.Values, value)
	return a
}

func (a *CallExprNamedArgs) AppendFlagE(name Expr) *CallExprNamedArgs {
	return a.AppendE(name, nil)
}

func (a *CallExprNamedArgs) AppendS(name string, value Expr) *CallExprNamedArgs {
	a.Names = append(a.Names, &NamedArgExpr{Ident: &IdentExpr{Name: name}})
	a.Values = append(a.Values, value)
	return a
}

func (a *CallExprNamedArgs) AppendFlags(name ...string) *CallExprNamedArgs {
	for _, name := range name {
		a.Names = append(a.Names, &NamedArgExpr{Ident: &IdentExpr{Name: name}})
		a.Values = append(a.Values, nil)
	}
	return a
}

func (a *CallExprNamedArgs) Prepend(name *NamedArgExpr, value Expr) *CallExprNamedArgs {
	a.Names = append([]*NamedArgExpr{name}, a.Names...)
	a.Values = append([]Expr{value}, a.Values...)
	return a
}

func (a *CallExprNamedArgs) Get(name NamedArgExpr) (index int, value Expr) {
	names := name.String()
	index = -1
	for i, expr := range a.Names {
		if expr.String() == names {
			return i, a.Values[i]
		}
	}
	return
}

func (a *CallExprNamedArgs) Valid() bool {
	return len(a.Names) > 0
}

func (a *CallExprNamedArgs) NamesExpr() (r []Expr) {
	for _, v := range a.Names {
		r = append(r, v.Expr())
	}
	return r
}

func (a *CallExprNamedArgs) String() string {
	var s []string
	do := func(i int, name *NamedArgExpr) (es string) {
		if name.Var {
			return name.String()
		}

		if name.Exp != nil {
			es = "["
		}
		es += name.Expr().String()

		if v := a.Values[i]; v == nil {
			es += "=" + (&FlagLit{Value: true}).String()
		} else if f, _ := v.(*FuncDefLit); f != nil {
			es += f.String()
		} else {
			es += "=" + a.Values[i].String()
		}
		if name.Exp != nil {
			es += "]"
		}
		return
	}

	for i, name := range a.Names {
		s = append(s, do(i, name))
	}
	return strings.Join(s, ", ")
}

func (a *CallExprNamedArgs) WriteCode(ctx *CodeWriteContext, inNewLime, hasPositional bool) {
	a.WriteCodeWithFlag(inNewLime, hasPositional, ctx)
}

func (a *CallExprNamedArgs) writeItemCode(i int, ctx *CodeWriteContext) {
	name := a.Names[i]
	if name.Var {
		name.WriteCode(ctx)
		return
	}

	if name.Exp != nil {
		ctx.WriteSingleByte('[')
		defer ctx.WriteSingleByte(']')
	}

	if v := a.Values[i]; v == nil {
		if name.Lit != nil && name.Lit.CanIdent() {
			ctx.WriteString(name.Lit.Value())
		} else if name.Ident != nil {
			ctx.WriteString(name.Ident.String())
		} else {
			name.Expr().WriteCode(ctx)
		}
	} else {
		switch f := v.(type) {
		case *FuncDefLit:
			ctx.WriteString(name.Expr().String())
			f.WriteCode(ctx)
		case *FuncWithMethodsExpr:
			ctx.WriteString(name.Expr().String())
			ctx.WriteString(" ")
			f.WriteCode(ctx)
		default:
			ctx.WriteString(name.Expr().String() + "=")
			v.WriteCode(ctx)
		}
	}
}

func (a *CallExprNamedArgs) WriteCodeWithFlag(inNewLine, hasPositional bool, ctx *CodeWriteContext) {
	if l := len(a.Names); l > 0 {
		if l == 1 && !hasPositional {
			ctx.WriteString("; ")
			a.writeItemCode(0, ctx)
		} else {
			var skip int
			if inNewLine {
				if !hasPositional {
					ctx.WriteString(";\n")
				}

				ctx.Depth++

				ctx.WriteString(ctx.CurrentPrefix())

				if hasPositional {
					ctx.WriteString("; ")
				}

				a.writeItemCode(0, ctx)
				ctx.Depth--

				if l > 1 {
					ctx.WriteString(",")
				} else {
					ctx.WriteString("\n")
				}
				skip++
			} else {
				ctx.WriteString("; ")
			}

			ctx.WriteItems(
				inNewLine,
				len(a.Names)-skip,
				func(i int) {
					a.writeItemCode(i+skip, ctx)
				},
				func(nl bool) {
					if nl {
						ctx.WriteSecondLine()
					}
				})
		}
	}
}

type CalleeKeywordExpr struct {
	TokenPos source.Pos
	Literal  string
}

func (c *CalleeKeywordExpr) Pos() source.Pos {
	return c.TokenPos
}

func (c *CalleeKeywordExpr) End() source.Pos {
	return c.TokenPos + source.Pos(len(token.Callee.String()))
}

func (c *CalleeKeywordExpr) String() string {
	return c.Literal
}

func (c *CalleeKeywordExpr) ExprNode() {
}

func (c *CalleeKeywordExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(c.Literal)
}

type ArgsKeywordExpr struct {
	TokenPos source.Pos
	Literal  string
}

func (c *ArgsKeywordExpr) Pos() source.Pos {
	return c.TokenPos
}

func (c *ArgsKeywordExpr) End() source.Pos {
	return c.TokenPos + source.Pos(len(c.Literal))
}

func (c *ArgsKeywordExpr) String() string {
	return c.Literal
}

func (c *ArgsKeywordExpr) ExprNode() {
}

func (c *ArgsKeywordExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(c.Literal)
}

type NamedArgsKeywordExpr struct {
	TokenPos source.Pos
	Literal  string
}

func (c *NamedArgsKeywordExpr) Pos() source.Pos {
	return c.TokenPos
}

func (c *NamedArgsKeywordExpr) End() source.Pos {
	return c.TokenPos + source.Pos(len(c.Literal))
}

func (c *NamedArgsKeywordExpr) String() string {
	return c.Literal
}

func (c *NamedArgsKeywordExpr) ExprNode() {
}

func (c *NamedArgsKeywordExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(c.Literal)
}

type BlockExpr struct {
	*BlockStmt
}

func (b BlockExpr) ExprNode() {}

// ThrowExpr represents an throw expression.
type ThrowExpr struct {
	ThrowPos source.Pos
	Expr     Expr
}

func (s *ThrowExpr) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (s *ThrowExpr) Pos() source.Pos {
	return s.ThrowPos
}

// End returns the position of first character immediately after the node.
func (s *ThrowExpr) End() source.Pos {
	return s.Expr.End()
}

func (s *ThrowExpr) String() string {
	var expr string
	if s.Expr != nil {
		expr = s.Expr.String()
	}
	return "throw " + expr
}

func (s *ThrowExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString("throw ")
	s.Expr.WriteCode(ctx)
}

// ReturnExpr represents an return expression.
type ReturnExpr struct {
	Return
}

func (s *ReturnExpr) ExprNode() {}

type MixedTextExpr struct {
	StartLit ast.Literal
	EndLit   ast.Literal
	Stmt     MixedTextStmt
}

func (e *MixedTextExpr) ExprNode() {}

func (e *MixedTextExpr) Pos() source.Pos {
	return e.StartLit.Pos
}

func (e *MixedTextExpr) End() source.Pos {
	return e.EndLit.End()
}

func (e *MixedTextExpr) String() string {
	var b strings.Builder
	if e.Stmt.Flags.Has(RemoveLeftSpaces) {
		b.WriteByte('-')
	}
	b.WriteString(e.StartLit.Value)
	b.WriteString(e.Stmt.String())
	b.WriteString(e.EndLit.Value)
	if e.Stmt.Flags.Has(RemoveRightSpaces) {
		b.WriteByte('-')
	}
	return b.String()
}

func (e *MixedTextExpr) WriteCode(ctx *CodeWriteContext) {
	if e.Stmt.Lit.Value == "" {
		ctx.WriteString(`""`)
	} else {
		ctx.WriteSingleByte('(')
		if e.Stmt.Flags.Has(RemoveLeftSpaces) {
			ctx.WriteSingleByte('-')
		}
		ctx.WriteString(e.StartLit.Value)
		e.Stmt.WriteCode(ctx)
		ctx.WriteString(e.EndLit.Value)
		if e.Stmt.Flags.Has(RemoveRightSpaces) {
			ctx.WriteSingleByte('-')
		}
		ctx.WriteSingleByte(')')
	}
}

type FuncWithMethodsStmt struct {
	FuncWithMethodsExpr
}

func (f FuncWithMethodsStmt) StmtNode() {
}

// FuncWithMethodsExpr represents the function with methods expression.
type FuncWithMethodsExpr struct {
	FuncToken TokenLit
	LBrace    source.Pos
	RBrace    source.Pos
	NameExpr  Expr
	Methods   []*FuncMethod
}

func (e *FuncWithMethodsExpr) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *FuncWithMethodsExpr) Pos() source.Pos {
	if e.FuncToken.Pos != source.NoPos {
		return e.FuncToken.Pos
	}
	if e.NameExpr != nil {
		return e.NameExpr.Pos()
	}
	return e.LBrace
}

// End returns the position of first character immediately after the node.
func (e *FuncWithMethodsExpr) End() source.Pos {
	return e.RBrace + 1
}

func (e *FuncWithMethodsExpr) NameIdent() *IdentExpr {
	if e.NameExpr == nil {
		return nil
	}
	return IdentOfSelector(e.NameExpr)
}

func (e *FuncWithMethodsExpr) Funcs() (f Exprs) {
	f = make(Exprs, len(e.Methods))
	for i, m := range e.Methods {
		f[i] = m.Func()
	}
	return
}

func (e *FuncWithMethodsExpr) String() string {
	var b strings.Builder
	if e.FuncToken.Valid() {
		b.WriteString(e.FuncToken.Token.String())
		b.WriteString(" ")
	}
	if e.NameExpr != nil {
		b.WriteString(e.NameExpr.String())
		b.WriteString(" ")
	}
	b.WriteString("{")
	if len(e.Methods) > 0 {
		for _, m := range e.Methods {
			b.WriteString(m.String())
			b.WriteString("; ")
		}
	}
	b.WriteString("}")
	return b.String()
}

func (e *FuncWithMethodsExpr) WriteCode(ctx *CodeWriteContext) {
	if e.FuncToken.Pos != source.NoPos {
		ctx.WriteString(e.FuncToken.Token.String())
		ctx.WriteString(" ")
	}
	if e.NameExpr != nil {
		ctx.WriteString(e.NameExpr.String())
		ctx.WriteString(" ")
	}

	ctx.WriteString("{")
	ctx.WriteItemsSep(ctx.HasPrefix(), len(e.Methods), "; ", "\n", func(i int) {
		e.Methods[i].WriteCode(ctx)
	}, func(newLine bool) {
		if newLine {
			ctx.WriteSecondLine()
		}
	})
	if len(e.Methods) > 0 && ctx.HasPrefix() {
		ctx.WritePrefix()
	}
	ctx.WriteString("}")
}

type FuncMethod struct {
	Params    FuncParams
	Return    []*TypedIdentExpr
	Body      *BlockStmt
	LambdaPos source.Pos
	BodyExpr  Expr
}

func (m *FuncMethod) Pos() source.Pos {
	return m.Params.Pos()
}

func (m *FuncMethod) End() source.Pos {
	if m.BodyExpr != nil {
		return m.BodyExpr.Pos()
	}
	return m.Body.End()
}

func (m *FuncMethod) String() string {
	var b strings.Builder
	b.WriteString(m.Params.String())
	b.WriteString(FormatFuncReturn(m.Return))
	b.WriteString(" ")
	if m.BodyExpr != nil {
		b.WriteString("=> ")
		b.WriteString(m.BodyExpr.String())
	} else {
		b.WriteString(m.Body.String())
	}
	return b.String()
}

func (m *FuncMethod) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(m.Params.String())
	ctx.WriteString(FormatFuncReturn(m.Return))
	ctx.WriteString(" ")
	if m.BodyExpr != nil {
		ctx.WriteString("=> ")
		m.BodyExpr.WriteCode(ctx)
	} else {
		m.Body.WriteCode(ctx)
	}
}

func (m *FuncMethod) Func() *FuncExpr {
	return &FuncExpr{
		Type: &FuncType{
			FuncPos:    m.Params.Pos(),
			FuncHeader: FuncHeader{Params: m.Params, Return: m.Return},
		},
		Body:      m.Body,
		BodyExpr:  m.BodyExpr,
		LambdaPos: m.LambdaPos,
	}
}

// PropStmt represents a property declaration statement, e.g.
// `prop name { () { ... } (v int) { ... } }`.
type PropStmt struct {
	PropExpr
}

func (s PropStmt) StmtNode() {}

// PropExpr represents a property expression: a named value defined by one
// or more accessor methods, sharing the func-with-methods body syntax but
// introduced by the `prop` keyword. A method with no parameters is the getter
// and a method with one parameter is a setter.
type PropExpr struct {
	PropToken TokenLit
	LBrace    source.Pos
	RBrace    source.Pos
	NameExpr  Expr
	Methods   []*FuncMethod
}

func (e *PropExpr) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *PropExpr) Pos() source.Pos {
	if e.PropToken.Pos != source.NoPos {
		return e.PropToken.Pos
	}
	if e.NameExpr != nil {
		return e.NameExpr.Pos()
	}
	return e.LBrace
}

// End returns the position of first character immediately after the node.
func (e *PropExpr) End() source.Pos {
	return e.RBrace + 1
}

func (e *PropExpr) NameIdent() *IdentExpr {
	if e.NameExpr == nil {
		return nil
	}
	return IdentOfSelector(e.NameExpr)
}

func (e *PropExpr) Funcs() (f Exprs) {
	f = make(Exprs, len(e.Methods))
	for i, m := range e.Methods {
		f[i] = m.Func()
	}
	return
}

// single reports whether the property was written in the single-accessor form
// `prop name(params) {body}` (no surrounding braces).
func (e *PropExpr) single() bool {
	return e.LBrace == source.NoPos && len(e.Methods) == 1
}

func (e *PropExpr) String() string {
	var b strings.Builder
	if e.PropToken.Valid() {
		b.WriteString(e.PropToken.Token.String())
		b.WriteString(" ")
	}
	if e.NameExpr != nil {
		b.WriteString(e.NameExpr.String())
		if !e.single() {
			b.WriteString(" ")
		}
	}
	if e.single() {
		b.WriteString(e.Methods[0].String())
		return b.String()
	}
	b.WriteString("{")
	for _, m := range e.Methods {
		b.WriteString(m.String())
		b.WriteString("; ")
	}
	b.WriteString("}")
	return b.String()
}

func (e *PropExpr) WriteCode(ctx *CodeWriteContext) {
	if e.PropToken.Pos != source.NoPos {
		ctx.WriteString(e.PropToken.Token.String())
		ctx.WriteString(" ")
	}
	if e.NameExpr != nil {
		ctx.WriteString(e.NameExpr.String())
		if !e.single() {
			ctx.WriteString(" ")
		}
	}

	if e.single() {
		e.Methods[0].WriteCode(ctx)
		return
	}

	ctx.WriteString("{")
	ctx.WriteItemsSep(ctx.HasPrefix(), len(e.Methods), "; ", "\n", func(i int) {
		e.Methods[i].WriteCode(ctx)
	}, func(newLine bool) {
		if newLine {
			ctx.WriteSecondLine()
		}
	})
	if len(e.Methods) > 0 && ctx.HasPrefix() {
		ctx.WritePrefix()
	}
	ctx.WriteString("}")
}

// MethodInterfaceStmt is the statement form of a method interface, e.g.
// `meti Name { () }`, which binds the interface to a const.
type MethodInterfaceStmt struct {
	MethodInterfaceExpr
}

func (s MethodInterfaceStmt) StmtNode() {}

// MethodInterfaceExpr is a set of required function headers introduced by the
// `meti` keyword: `meti { () }`, `meti { (), (v) <int> }`, `meti Name { … }`.
// Each header is a FuncHeaderExpr written without the surrounding angle
// brackets.
type MethodInterfaceExpr struct {
	MetiToken TokenLit
	NameExpr  Expr
	LBrace    source.Pos
	RBrace    source.Pos
	Headers   []*FuncHeaderExpr
}

func (e *MethodInterfaceExpr) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *MethodInterfaceExpr) Pos() source.Pos {
	if e.MetiToken.Pos != source.NoPos {
		return e.MetiToken.Pos
	}
	if e.NameExpr != nil {
		return e.NameExpr.Pos()
	}
	return e.LBrace
}

// End returns the position of first character immediately after the node.
func (e *MethodInterfaceExpr) End() source.Pos {
	return e.RBrace + 1
}

func (e *MethodInterfaceExpr) NameIdent() *IdentExpr {
	if e.NameExpr == nil {
		return nil
	}
	return IdentOfSelector(e.NameExpr)
}

func (e *MethodInterfaceExpr) String() string {
	var b strings.Builder
	if e.MetiToken.Valid() {
		b.WriteString(e.MetiToken.Token.String())
		b.WriteString(" ")
	}
	if e.NameExpr != nil {
		b.WriteString(e.NameExpr.String())
		b.WriteString(" ")
	}
	b.WriteString("{")
	for _, h := range e.Headers {
		b.WriteString(h.FuncHeader.String())
		b.WriteString("; ")
	}
	b.WriteString("}")
	return b.String()
}

// armsInNewLine reports whether the headers should be one per line.
func (e *MethodInterfaceExpr) headersInNewLine(ctx *CodeWriteContext) bool {
	return ctx.HasPrefix() && ctx.Flags.Has(CodeWriteContextFlagFormatMethodInterfaceInNewLine)
}

func (e *MethodInterfaceExpr) WriteCode(ctx *CodeWriteContext) {
	if e.MetiToken.Pos != source.NoPos {
		ctx.WriteString(e.MetiToken.Token.String())
		ctx.WriteString(" ")
	}
	if e.NameExpr != nil {
		ctx.WriteString(e.NameExpr.String())
		ctx.WriteString(" ")
	}
	ctx.WriteString("{")
	if e.headersInNewLine(ctx) {
		ctx.Depth++
		for i := range e.Headers {
			if i > 0 {
				ctx.WriteString(",")
			}
			ctx.WriteSemi()
			ctx.WriteString(e.Headers[i].FuncHeader.String())
		}
		ctx.Depth--
		ctx.WriteSemi()
	} else {
		for i := range e.Headers {
			if i > 0 {
				ctx.WriteString(",")
			}
			ctx.WriteString(" ")
			ctx.WriteString(e.Headers[i].FuncHeader.String())
		}
		if len(e.Headers) > 0 {
			ctx.WriteString(" ")
		}
	}
	ctx.WriteString("}")
}

// FuncExpr represents a function literal.
type FuncExpr struct {
	Type      *FuncType
	Body      *BlockStmt
	LambdaPos source.Pos
	BodyExpr  Expr
}

func (e *FuncExpr) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *FuncExpr) Pos() source.Pos {
	if e.Type == nil {
		if e.BodyExpr != nil {
			return e.BodyExpr.Pos()
		}
		return e.Body.Pos()
	}
	return e.Type.Pos()
}

// End returns the position of first character immediately after the node.
func (e *FuncExpr) End() source.Pos {
	if e.BodyExpr != nil {
		return e.BodyExpr.End()
	}
	return e.Body.End()
}

func (e *FuncExpr) prefix() (s string) {
	if e.Type != nil {
		if e.Type.FuncPos != 0 {
			if len(e.Type.Token.Literal) > 0 {
				s = e.Type.Token.Literal
			} else {
				s = "func"
			}
		}
		if t := e.Type.String(); len(t) > 0 {
			if e.Type.NameExpr != nil && len(s) > 0 {
				s += " "
			}
			s += t + " "
		}
	} else {
		s = "func"
	}
	return
}

func (e *FuncExpr) String() string {
	s := e.prefix()
	if e.BodyExpr != nil {
		s += "=> " + e.BodyExpr.String()
	} else {
		s += e.Body.String()
	}
	return s
}

func (e *FuncExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(e.prefix())

	if e.BodyExpr != nil {
		ctx.WriteString("=> ")
		ctx.WriteString(e.BodyExpr.String())
	} else {
		e.Body.WriteCode(ctx)
	}
}

// ClosureExpr represents a function closure literal.
type ClosureExpr struct {
	ast.NodeData
	Params FuncParams
	Return []*TypedIdentExpr
	Lambda Token
	Body   Expr
}

func (e *ClosureExpr) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *ClosureExpr) Pos() source.Pos {
	return e.Params.Pos()
}

// End returns the position of first character immediately after the node.
func (e *ClosureExpr) End() source.Pos {
	return e.Body.End()
}

func (e *ClosureExpr) sep() string {
	if e.Lambda.Valid() {
		return " " + e.Lambda.Token.String()
	}
	return ""
}

func (e *ClosureExpr) String() string {
	return e.Params.String() + FormatFuncReturn(e.Return) + e.sep() + " " + e.Body.String()
}

func (e *ClosureExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(e.Params.String(), FormatFuncReturn(e.Return), e.sep(), " ")
	if block, ok := e.Body.(*BlockExpr); ok {
		block.WriteCode(ctx)
	} else {
		e.Body.WriteCode(ctx)
	}
}

type MethodExpr struct {
	Expr
}

// DictExpr represents a map literal.
type DictExpr struct {
	LBrace   source.Pos
	Elements []*DictElementLit
	RBrace   source.Pos
}

func (e *DictExpr) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *DictExpr) Pos() source.Pos {
	return e.LBrace
}

// End returns the position of first character immediately after the node.
func (e *DictExpr) End() source.Pos {
	return e.RBrace + 1
}

func (e *DictExpr) String() string {
	var elements []string
	for _, m := range e.Elements {
		elements = append(elements, m.String())
	}
	return "{" + strings.Join(elements, ", ") + "}"
}

func (e *DictExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteSingleByte('{')
	if l := len(e.Elements); l > 0 {
		if l == 1 {
			ctx.WriteSingleByte(' ')
			e.Elements[0].WriteCode(ctx)
			ctx.WriteSingleByte(' ')
		} else {
			inLineLine := ctx.DecideNewLine(
				CodeWriteContextFlagFormatDictItemInNewLine, len(e.Elements), ", ", 1,
				func(i int) { e.Elements[i].WriteCode(ctx) })
			ctx.WriteItemsSep(
				inLineLine,
				len(e.Elements),
				", ",
				"",
				func(i int) {
					e.Elements[i].WriteCode(ctx)
				},
				func(newLine bool) {
					if newLine {
						ctx.WriteSecondLine()
					}
				})
			if inLineLine {
				ctx.WritePrefix()
			}
		}
	}
	ctx.WriteSingleByte('}')
}

// ArrayExpr represents an array literal.
type ArrayExpr struct {
	Elements []Expr
	LBrack   source.Pos
	RBrack   source.Pos
}

func (e *ArrayExpr) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *ArrayExpr) Pos() source.Pos {
	return e.LBrack
}

// End returns the position of first character immediately after the node.
func (e *ArrayExpr) End() source.Pos {
	return e.RBrack + 1
}

func (e *ArrayExpr) String() string {
	var elements []string
	for _, m := range e.Elements {
		elements = append(elements, m.String())
	}
	return "[" + strings.Join(elements, ", ") + "]"
}

func (e *ArrayExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteSingleByte('[')
	if l := len(e.Elements); l > 0 {
		if l == 1 {
			e.Elements[0].WriteCode(ctx)
		} else {
			inLineLine := ctx.DecideNewLine(
				CodeWriteContextFlagFormatArrayItemInNewLine, len(e.Elements), ", ", 1,
				func(i int) { e.Elements[i].WriteCode(ctx) })
			ctx.WriteItemsSep(
				inLineLine,
				len(e.Elements),
				", ",
				"",
				func(i int) {
					e.Elements[i].WriteCode(ctx)
				},
				func(newLine bool) {
					if newLine {
						ctx.WriteSecondLine()
					}
				})
			if inLineLine {
				ctx.WritePrefix()
			}
		}
	}
	ctx.WriteSingleByte(']')
}

type Ptr struct {
	TokenPos source.Pos
	Expr
}

func (e *Ptr) String() string {
	return "&(" + e.Expr.String() + ")"
}

func (e *Ptr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteSingleByte('&')
	e.Expr.WriteCode(ctx)
}

type ComputedExpr struct {
	StartPos source.Pos
	EndPos   source.Pos
	Stmts    Stmts
}

func (e *ComputedExpr) ExprNode() {}

func (e *ComputedExpr) Pos() source.Pos {
	if e.StartPos > 0 {
		return e.StartPos
	}
	return e.Stmts[0].Pos()
}

func (e *ComputedExpr) End() source.Pos {
	if e.EndPos > 0 {
		return e.EndPos
	}
	return e.Stmts[len(e.Stmts)-1].End()
}

func (e *ComputedExpr) String() string {
	return "(= " + e.Stmts.String() + ")"
}

func (e *ComputedExpr) WriteCode(ctx *CodeWriteContext) {
	if len(e.Stmts) == 1 {
		ctx.WriteString("(= ")
		e.Stmts[0].WriteCode(ctx)
		ctx.WriteSingleByte(')')
	} else {
		(&BlockStmt{
			LBrace: Lit("(=", e.StartPos),
			RBrace: Lit(")", e.StartPos),
			Stmts:  e.Stmts,
		}).WriteCode(ctx)
	}
}

type ToRaw struct {
	TokenPos source.Pos
	Expr     Expr
}

func (r *ToRaw) Pos() source.Pos {
	if r.TokenPos > 0 {
		return r.TokenPos
	}
	return r.Expr.Pos()
}

func (r *ToRaw) End() source.Pos {
	return r.Expr.End()
}

func (r *ToRaw) String() string {
	return "raw " + r.Expr.String()
}

func (r *ToRaw) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString("raw ")
	r.Expr.WriteCode(ctx)
}

func (r *ToRaw) ExprNode() {
}

// ComprehensionClause is one clause of a comprehension: a `for k, v in iter`
// generator (For == true) or an `if cond` filter (For == false).
type ComprehensionClause struct {
	For      bool
	Key      *IdentExpr // for k, v in ...; nil when only a value var is given
	Value    *IdentExpr // for v in ...
	Iterable Expr
	Cond     Expr // if cond
}

func (c *ComprehensionClause) String() string {
	if !c.For {
		return "if " + c.Cond.String()
	}
	var b strings.Builder
	b.WriteString("for ")
	if c.Key != nil {
		b.WriteString(c.Key.String())
		b.WriteString(", ")
	}
	b.WriteString(c.Value.String())
	b.WriteString(" in ")
	b.WriteString(c.Iterable.String())
	return b.String()
}

func (c *ComprehensionClause) WriteCode(ctx *CodeWriteContext) {
	if !c.For {
		ctx.WriteString("if ")
		c.Cond.WriteCode(ctx)
		return
	}
	ctx.WriteString("for ")
	if c.Key != nil {
		c.Key.WriteCode(ctx)
		ctx.WriteString(", ")
	}
	c.Value.WriteCode(ctx)
	ctx.WriteString(" in ")
	c.Iterable.WriteCode(ctx)
}

func comprehensionClausesString(clauses []*ComprehensionClause) string {
	var b strings.Builder
	for _, cl := range clauses {
		b.WriteString(" ")
		b.WriteString(cl.String())
	}
	return b.String()
}

func writeComprehensionClauses(ctx *CodeWriteContext, clauses []*ComprehensionClause) {
	for _, cl := range clauses {
		ctx.WriteString(" ")
		cl.WriteCode(ctx)
	}
}

// ArrayComprehension represents `[elem for x in it if cond ...]`.
type ArrayComprehension struct {
	LBrack  source.Pos
	Element Expr
	Clauses []*ComprehensionClause
	RBrack  source.Pos
}

func (e *ArrayComprehension) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *ArrayComprehension) Pos() source.Pos { return e.LBrack }

// End returns the position of first character immediately after the node.
func (e *ArrayComprehension) End() source.Pos { return e.RBrack + 1 }

func (e *ArrayComprehension) String() string {
	return "[" + e.Element.String() + comprehensionClausesString(e.Clauses) + "]"
}
func (e *ArrayComprehension) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString("[")
	e.Element.WriteCode(ctx)
	writeComprehensionClauses(ctx, e.Clauses)
	ctx.WriteString("]")
}

// DictComprehension represents `{k1: v1, [ke]: ve, ... for x in it if cond}`.
// Each iteration assigns every element into the dict being built; element keys
// may be static (`name:`) or computed (`[expr]:`). Inside value expressions the
// special variable `_` refers to the dict being built.
type DictComprehension struct {
	LBrace   source.Pos
	Elements []*DictElementLit
	Clauses  []*ComprehensionClause
	RBrace   source.Pos
}

func (e *DictComprehension) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *DictComprehension) Pos() source.Pos { return e.LBrace }

// End returns the position of first character immediately after the node.
func (e *DictComprehension) End() source.Pos { return e.RBrace + 1 }

func (e *DictComprehension) String() string {
	var b strings.Builder
	b.WriteString("{")
	for i, el := range e.Elements {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(el.String())
	}
	b.WriteString(comprehensionClausesString(e.Clauses))
	b.WriteString("}")
	return b.String()
}
func (e *DictComprehension) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString("{")
	for i, el := range e.Elements {
		if i > 0 {
			ctx.WriteString(", ")
		}
		el.WriteCode(ctx)
	}
	writeComprehensionClauses(ctx, e.Clauses)
	ctx.WriteString("}")
}

// MatchArm is a single arm of a MatchExpr. A normal arm has one or more Conds,
// matched against the subject with OR semantics (`A, B: …`); the `else` arm has
// no Conds. Exactly one of Result (expression form `conds: result`) or Body
// (statement form `conds { body }`) is set.
type MatchArm struct {
	Conds  []Expr     // conditions (OR); empty for the else arm
	Result Expr       // `conds: result`
	Body   *BlockStmt // `conds { body }`
}

// IsElse reports whether this arm is the default `else` arm.
func (a *MatchArm) IsElse() bool { return len(a.Conds) == 0 }

func (a *MatchArm) writeConds(ctx *CodeWriteContext) {
	if a.IsElse() {
		ctx.WriteString("else")
		return
	}
	for i, c := range a.Conds {
		if i > 0 {
			ctx.WriteString(", ")
		}
		c.WriteCode(ctx)
	}
}

func (a *MatchArm) String() string {
	var b strings.Builder
	if a.IsElse() {
		b.WriteString("else")
	} else {
		for i, c := range a.Conds {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(c.String())
		}
	}
	if a.Body != nil {
		b.WriteString(" ")
		b.WriteString(a.Body.String())
	} else {
		b.WriteString(": ")
		b.WriteString(a.Result.String())
	}
	return b.String()
}

func (a *MatchArm) WriteCode(ctx *CodeWriteContext) {
	a.writeConds(ctx)
	if a.Body != nil {
		ctx.WriteString(" ")
		a.Body.WriteCode(ctx)
	} else {
		ctx.WriteString(": ")
		a.Result.WriteCode(ctx)
	}
}

// MatchExpr represents a PHP8-like match: `match subject { cond: result, ... }`
// (expression form, yields a value) or `match subject { cond { body }, ... }`
// (statement form, runs the matching block). Each arm holds one or more
// conditions compared against the subject with strict equality; the first arm
// with a matching condition wins. An optional `else` arm is the default. When
// nothing matches and there is no `else`, the match yields nil.
type MatchExpr struct {
	MatchPos source.Pos
	Expr     Expr // subject
	Arms     []*MatchArm
	LBrace   source.Pos
	RBrace   source.Pos
}

func (e *MatchExpr) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *MatchExpr) Pos() source.Pos { return e.MatchPos }

// End returns the position of first character immediately after the node.
func (e *MatchExpr) End() source.Pos { return e.RBrace + 1 }

// IsStmt reports whether the match uses statement-form (block) arms.
func (e *MatchExpr) IsStmt() bool {
	for _, a := range e.Arms {
		if a.Body != nil {
			return true
		}
	}
	return false
}

func (e *MatchExpr) String() string {
	var b strings.Builder
	b.WriteString("match ")
	b.WriteString(e.Expr.String())
	b.WriteString(" {")
	for i, a := range e.Arms {
		if i > 0 {
			b.WriteString(", ")
		} else {
			b.WriteString(" ")
		}
		b.WriteString(a.String())
	}
	if len(e.Arms) > 0 {
		b.WriteString(" ")
	}
	b.WriteString("}")
	return b.String()
}

// armsInNewLine reports whether arms should each go on their own line, based on
// the active formatter flag for the match form (expression vs statement).
func (e *MatchExpr) armsInNewLine(ctx *CodeWriteContext) bool {
	if !ctx.HasPrefix() {
		return false
	}
	if e.IsStmt() {
		return ctx.Flags.Has(CodeWriteContextFlagFormatMatchStmtArmsInNewLine)
	}
	return ctx.Flags.Has(CodeWriteContextFlagFormatMatchExprArmsInNewLine)
}

func (e *MatchExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString("match ")
	e.Expr.WriteCode(ctx)
	ctx.WriteString(" {")

	if e.armsInNewLine(ctx) {
		// one arm per line; the `else` arm is not preceded by a comma
		ctx.Depth++
		for i, a := range e.Arms {
			if i > 0 && !a.IsElse() {
				ctx.WriteString(",")
			}
			ctx.WriteSemi()
			a.WriteCode(ctx)
		}
		ctx.Depth--
		ctx.WriteSemi()
	} else {
		for i, a := range e.Arms {
			if i > 0 {
				ctx.WriteString(",")
			}
			ctx.WriteString(" ")
			a.WriteCode(ctx)
		}
		if len(e.Arms) > 0 {
			ctx.WriteString(" ")
		}
	}
	ctx.WriteString("}")
}

// OrExpr represents an error-fallback expression: `expr or fallback`.
// If evaluating Expr throws an error, Fallback is evaluated instead, with the
// caught error bound to the local `$err`. The whole expression yields the value
// of Expr on success, or the value of Fallback when Expr throws.
type OrExpr struct {
	Expr     Expr
	Fallback Expr
	OrPos    source.Pos
}

func (e *OrExpr) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *OrExpr) Pos() source.Pos {
	return e.Expr.Pos()
}

// End returns the position of first character immediately after the node.
func (e *OrExpr) End() source.Pos {
	return e.Fallback.End()
}

func (e *OrExpr) String() string {
	return e.Expr.String() + " or " + e.Fallback.String()
}

func (e *OrExpr) WriteCode(ctx *CodeWriteContext) {
	e.Expr.WriteCode(ctx)
	ctx.WriteString(" or ")
	e.Fallback.WriteCode(ctx)
}
