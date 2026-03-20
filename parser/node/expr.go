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
	ctx.WriteSingleByte('(')
	e.LHS.WriteCode(ctx)
	ctx.WriteString(" " + e.Token.String() + " ")
	e.RHS.WriteCode(ctx)
	ctx.WriteSingleByte(')')
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
		s = (&ParenExpr{Expr: f}).String()
	} else {
		s = e.Func.String()
	}
	var buf = bytes.NewBufferString(s)
	e.CallArgs.StringW(buf)
	return buf.String()
}

func (e *CallExpr) WriteCode(ctx *CodeWriteContext) {
	if f, _ := e.Func.(*FuncExpr); f != nil {
		(&ParenExpr{Expr: f}).WriteCode(ctx)
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
	return e.Ident.Pos()
}

// End returns the position of first character immediately after the node.
func (e *TypedIdentExpr) End() source.Pos {
	return e.Type[len(e.Type)-1].End()
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
	return e.Args.Values[0].(*StringLit).Value()
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

// EmbedExpr represents an embed expression
type EmbedExpr struct {
	Path     string
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
	// import("moduleName")
	return source.Pos(int(e.TokenPos) + 10 + len(e.Path))
}

func (e *EmbedExpr) String() string {
	return `embed("` + e.Path + `")`
}

func (e *EmbedExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(e.String())
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
	LParen source.Pos
	RParen source.Pos
}

func (e *ParenExpr) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *ParenExpr) Pos() source.Pos {
	return e.LParen
}

// End returns the position of first character immediately after the node.
func (e *ParenExpr) End() source.Pos {
	return e.RParen + 1
}

func (e *ParenExpr) String() string {
	switch t := e.Expr.(type) {
	case *ParenExpr:
		return t.Expr.String()
	case *BinaryExpr:
		return t.String()
	default:
		return "(" + e.Expr.String() + ")"
	}
}

func (e *ParenExpr) WriteCode(ctx *CodeWriteContext) {
	switch e.Expr.(type) {
	case *ParenExpr, *BinaryExpr:
		e.Expr.WriteCode(ctx)
	default:
		ctx.WriteSingleByte('(')
		e.Expr.WriteCode(ctx)
		ctx.WriteSingleByte(')')
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
	LParen             source.Pos
	RParen             source.Pos
	PositionalElements Exprs
	NamedElements      Exprs
}

func (e *MultiParenExpr) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *MultiParenExpr) Pos() source.Pos {
	return e.LParen
}

// End returns the position of first character immediately after the node.
func (e *MultiParenExpr) End() source.Pos {
	return e.RParen + 1
}

func (e *MultiParenExpr) String() string {
	var s strings.Builder
	s.WriteString("(,")

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
	s.WriteByte(')')
	return s.String()
}

func (e *MultiParenExpr) ToMultiParenExpr() *MultiParenExpr {
	return e
}

func (e *MultiParenExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString("(,")
	nl := ctx.Flags.Has(CodeWriteContextFlagFormatParemValuesInNewLine)

	ctx.WriteItems(nl,
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

	ctx.WriteItems(nl,
		len(e.PositionalElements),
		func(i int) {
			e.PositionalElements[i].WriteCode(ctx)
		},
		func(newLine bool) {
			if newLine {
				ctx.WriteSecondLine()
			}
		})

	ctx.WriteString(")")
}

func (e *MultiParenExpr) ToCallArgs(strict bool) (args *CallArgs, err *NodeError) {
	args = new(CallArgs)
	args.LParen = e.LParen
	args.RParen = e.RParen

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
			case *StringLit:
				na.Lit = t
			case *IdentExpr:
				na.Ident = t
			case *ParenExpr:
				na.Exp = t
			default:
				na.Exp = &ParenExpr{Expr: t}
			}
			args.NamedArgs.Names = append(args.NamedArgs.Names, na)
			args.NamedArgs.Values = append(args.NamedArgs.Values, t.Value)
		case *KeyValuePairLit:
			switch t2 := t.Key.(type) {
			case *IdentExpr:
				args.NamedArgs.Names = append(args.NamedArgs.Names, &NamedArgExpr{Ident: t2})
			case *StringLit:
				args.NamedArgs.Names = append(args.NamedArgs.Names, &NamedArgExpr{Lit: t2})
			case *TypedIdentExpr:
				if strict {
					err = NewExpectedError(t2, &StringLit{}, &IdentExpr{})
					return
				}
				args.NamedArgs.Names = append(args.NamedArgs.Names, &NamedArgExpr{Ident: t2.Ident})
			default:
				if strict {
					err = NewExpectedError(t2, &StringLit{}, &IdentExpr{})
				} else {
					err = NewExpectedError(t2, &StringLit{}, &IdentExpr{}, &TypedIdentExpr{})
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
	params.LParen = e.LParen
	params.RParen = e.RParen

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
	if s, _ := e.Sel.(*StringLit); s != nil {
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
	if s, _ := e.Sel.(*StringLit); s != nil {
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
	if s, _ := e.Sel.(*StringLit); s != nil {
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
	if s, _ := e.Sel.(*StringLit); s != nil {
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

func (a *CallExprPositionalArgs) WriteCode(ctx *CodeWriteContext) {
	a.WriteCodeWithNamedSep(ctx, false)
}

func (a *CallExprPositionalArgs) WriteCodeWithNamedSep(ctx *CodeWriteContext, namedSep bool) {
	a.WriteCodeWithNamedSepFlag(CodeWriteContextFlagFormatCallParamsInNewLine, ctx, namedSep)
}

func (a *CallExprPositionalArgs) WriteCodeWithNamedSepFlag(flag CodeWriteContextFlag, ctx *CodeWriteContext, namedSep bool) {
	values := a.Values

	if a.Var != nil {
		values = append(values, a.Var)
	}

	ctx.WriteItems(
		ctx.Flags.Has(flag),
		len(values),
		func(i int) {
			values[i].WriteCode(ctx)
		},
		func(newLine bool) {
			if namedSep {
				ctx.WriteString("; ")
			} else if newLine {
				ctx.WriteSecondLine()
			}
		})
}

type NamedArgExpr struct {
	Lit   *StringLit
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

func (a *CallExprNamedArgs) WriteCode(ctx *CodeWriteContext) {
	a.WriteCodeWithFlag(CodeWriteContextFlagFormatCallParamsInNewLine, ctx)
}

func (a *CallExprNamedArgs) WriteCodeWithFlag(flag CodeWriteContextFlag, ctx *CodeWriteContext) {
	ctx.WriteItems(
		ctx.Flags.Has(flag),
		len(a.Names),
		func(i int) {
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
		},
		func(nl bool) {
			if nl {
				ctx.WriteSecondLine()
			}
		})
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
	ctx.WriteItemsSep(len(ctx.Prefix) > 0, len(e.Methods), "; ", "\n", func(i int) {
		e.Methods[i].WriteCode(ctx)
	}, func(newLine bool) {
		if newLine {
			ctx.WriteSecondLine()
		}
	})
	if len(e.Methods) > 0 {
		ctx.WritePrefix()
	}
	ctx.WriteString("}")
}

type FuncMethod struct {
	Params    FuncParams
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
	ctx.WriteString(" ")
	if m.BodyExpr != nil {
		ctx.WriteString("=> ")
		m.BodyExpr.WriteCode(ctx)
	} else {
		ctx.Depth++
		m.Body.WriteCodeInSelfDepth(ctx, false)
		ctx.Depth--
	}
}

func (m *FuncMethod) Func() *FuncExpr {
	return &FuncExpr{
		Type: &FuncType{
			FuncPos: m.Params.Pos(),
			Params:  m.Params,
		},
		Body:      m.Body,
		BodyExpr:  m.BodyExpr,
		LambdaPos: m.LambdaPos,
	}
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
		e.Body.WriteCodeInSelfDepth(ctx, true)
	}
}

// ClosureExpr represents a function closure literal.
type ClosureExpr struct {
	ast.NodeData
	Params FuncParams
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
	return e.Params.String() + e.sep() + " " + e.Body.String()
}

func (e *ClosureExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(e.Params.String(), e.sep(), " ")
	if block, ok := e.Body.(*BlockExpr); ok {
		block.WriteCodeInSelfDepth(ctx, true)
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
	inLineLine := ctx.Flags.Has(CodeWriteContextFlagFormatDictItemInNewLine)
	ctx.WriteItems(
		inLineLine,
		len(e.Elements),
		func(i int) {
			e.Elements[i].WriteCode(ctx)
		},
		func(newLine bool) {
			if newLine {
				ctx.WriteSecondLine()
			}
		})
	ctx.WritePrefix()
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
	inLineLine := ctx.Flags.Has(CodeWriteContextFlagFormatArrayItemInNewLine)
	ctx.WriteItems(
		inLineLine,
		len(e.Elements),
		func(i int) {
			e.Elements[i].WriteCode(ctx)
		},
		func(newLine bool) {
			if newLine {
				ctx.WriteSecondLine()
			}
		})
	ctx.WritePrefix()
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
	ctx.WriteString("(=")
	if len(e.Stmts) == 1 {
		if e, _ := e.Stmts[0].(*ExprStmt); e != nil {
			ctx.WriteSingleByte(' ')
			e.Expr.WriteCode(ctx)
			goto done
		}
	}

	if len(ctx.Prefix) > 0 {
		ctx.WriteSecondLine()
		ctx.Depth++
		ctx.WriteStmts(e.Stmts...)
		ctx.Depth--
		if len(ctx.Prefix) > 0 {
			ctx.WritePrefixedLine()
		}
	} else {
		ctx.WriteSingleByte(' ')
		ctx.Depth++
		ctx.WriteStmts(e.Stmts...)
		ctx.Depth--
	}
done:
	ctx.WriteSingleByte(')')
}
