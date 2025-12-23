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

	"github.com/gad-lang/gad/internal"
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
	var buf = bytes.NewBufferString(e.Func.String())
	e.CallArgs.StringW(buf)
	return buf.String()
}

func (e *CallExpr) WriteCode(ctx *CodeWriteContext) {
	e.Func.WriteCode(ctx)
	e.CallArgs.WriteCode(ctx)
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

type TypedIdentExpr struct {
	Ident *IdentExpr
	Type  []*IdentExpr
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
			for i, ident := range e.Type {
				s[i] = ident.String()
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
	ModuleName string
	Token      token.Token
	TokenPos   source.Pos
}

func (e *ImportExpr) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *ImportExpr) Pos() source.Pos {
	return e.TokenPos
}

// End returns the position of first character immediately after the node.
func (e *ImportExpr) End() source.Pos {
	// import("moduleName")
	return source.Pos(int(e.TokenPos) + 10 + len(e.ModuleName))
}

func (e *ImportExpr) String() string {
	return `import("` + e.ModuleName + `")`
}

func (e *ImportExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(e.String())
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
	Expr   Expr
	LBrack source.Pos
	Index  Expr
	RBrack source.Pos
}

func (e *IndexExpr) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *IndexExpr) Pos() source.Pos {
	return e.Expr.Pos()
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
	return e.Expr.String() + "[" + index + "]"
}

func (e *IndexExpr) WriteCode(ctx *CodeWriteContext) {
	e.Expr.WriteCode(ctx)
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
		Exprs:  Exprs{e.Expr},
		LParen: e.LParen,
		RParen: e.RParen,
	}
}

func (e *ParenExpr) Items() Exprs {
	return Exprs{e.Expr}
}

// MultiParenExpr represents a parenthesis wrapped expressions.
type MultiParenExpr struct {
	Exprs  []Expr
	LParen source.Pos
	RParen source.Pos
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
	s.WriteByte('(')
	var left, right = e.Split()

	if l := len(left) + len(right); l < 2 {
		if l == 0 || !internal.TSType(append(left, right...)[0], &ArgVarLit{}) {
			s.WriteString(", ")
		}
	}

	for i, expr := range left {
		if i > 0 {
			s.WriteString(", ")
		}
		s.WriteString(expr.String())
	}

	if len(right) > 0 {
		s.WriteString("; ")
		for i, expr := range right {
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

func (e *MultiParenExpr) Split() (left []Expr, right []Expr) {
	var (
		add = func(expr Expr) {
			left = append(left, expr)
		}
		righting   bool
		startRight = func() {
			righting = true
			add = func(expr Expr) {
				right = append(right, expr)
			}
		}
	)

expr:
	for _, expr := range e.Exprs {
		if !righting {
			switch t := expr.(type) {
			case *KeyValueSepLit:
				startRight()
				continue expr
			case *KeyValuePairLit, *NamedArgVarLit:
				startRight()
			case *ArgVarLit:
				add(t)
				startRight()
				continue expr
			}
		}

		add(expr)
	}
	return
}

func (e *MultiParenExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteSingleByte('(')

	left, right := e.Split()
	ctx.WriteExprs(", ", left...)

	if len(right) > 0 {
		ctx.WriteString("; ")
		ctx.WriteExprs(", ", right...)
	}

	ctx.WriteSingleByte(')')
}

func (e *MultiParenExpr) ToCallArgs(strict bool) (args *CallArgs, err *NodeError) {
	args = new(CallArgs)
	args.LParen = e.LParen
	args.RParen = e.RParen

	left, right := e.Split()

	var n Expr

	for _, n = range left {
		switch t := n.(type) {
		case *ArgVarLit:
			args.Args.Var = t
		default:
			args.Args.Values = append(args.Args.Values, t)
		}
	}

	for _, n = range right {
		switch t := n.(type) {
		case *KeyValueLit:
			na := NamedArgExpr{}
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
				args.NamedArgs.Names = append(args.NamedArgs.Names, NamedArgExpr{Ident: t2})
			case *StringLit:
				args.NamedArgs.Names = append(args.NamedArgs.Names, NamedArgExpr{Lit: t2})
			case *TypedIdentExpr:
				if strict {
					err = NewExpectedError(t2, &StringLit{}, &IdentExpr{})
					return
				}
				args.NamedArgs.Names = append(args.NamedArgs.Names, NamedArgExpr{Ident: t2.Ident})
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
			args.NamedArgs.Var = t
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
	for _, n = range e.Exprs {
		switch t := n.(type) {
		case *IdentExpr:
			params.Args.Values = append(params.Args.Values, &TypedIdentExpr{Ident: t})
		case *TypedIdentExpr:
			params.Args.Values = append(params.Args.Values, t)
		case *ArgVarLit:
			switch t2 := t.Value.(type) {
			case *IdentExpr:
				params.Args.Var = &TypedIdentExpr{
					Ident: t2,
				}
				i++
				break exps
			case *TypedIdentExpr:
				params.Args.Var = t2
				i++
				break exps
			default:
				err = NewExpectedError(t.Value, &IdentExpr{})
				return
			}
		case *KeyValuePairLit, *NamedArgVarLit:
			break exps
		case *KeyValueSepLit:
		default:
			err = NewExpectedError(t, &IdentExpr{}, &TypedIdentExpr{}, &NamedArgVarLit{}, &KeyValuePairLit{},
				&ArgVarLit{})
			return
		}
		i++
	}

	if i < len(e.Exprs) {
	nexps:
		for _, n = range e.Exprs[i:] {
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
					params.NamedArgs.Var = &TypedIdentExpr{Ident: t2}
					i++
					break nexps
				case *TypedIdentExpr:
					params.NamedArgs.Var = t2
					i++
					break nexps
				default:
					err = NewExpectedError(t2, &IdentExpr{}, &TypedIdentExpr{})
					return
				}
			case *KeyValueSepLit:
			default:
				err = NewExpectedError(t, &KeyValuePairLit{}, &NamedArgVarLit{})
				return
			}
			i++
		}

		if i < len(e.Exprs) {
			err = NewUnExpectedError(e.Exprs[i])
			return
		}
	}

	return
}

type ExprSelector interface {
	Expr
	SelectorExpr() Expr
}

// SelectorExpr represents a selector expression.
type SelectorExpr struct {
	Expr Expr
	Sel  Expr
}

func (e *SelectorExpr) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *SelectorExpr) Pos() source.Pos {
	return e.Expr.Pos()
}

// End returns the position of first character immediately after the node.
func (e *SelectorExpr) End() source.Pos {
	return e.Sel.End()
}

func (e *SelectorExpr) String() string {
	r := e.Expr.String() + "."
	if s, _ := e.Sel.(*StringLit); s != nil {
		if s.CanIdent() {
			return r + s.Value()
		}
		return r + "(" + s.Literal + ")"
	}
	return r + e.Sel.String()
}

func (e *SelectorExpr) SelectorExpr() Expr {
	return e.Expr
}

func (e *SelectorExpr) WriteCode(ctx *CodeWriteContext) {
	e.Expr.WriteCode(ctx)
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

func (e *NullishSelectorExpr) SelectorExpr() Expr {
	return e.Expr
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

// CallExprArgs represents a call expression arguments.
type CallExprArgs struct {
	Values []Expr
	Var    *ArgVarLit
}

func (a *CallExprArgs) Valid() bool {
	return len(a.Values) > 0 || a.Var != nil
}

func (a *CallExprArgs) String() string {
	var s []string
	for _, v := range a.Values {
		s = append(s, v.String())
	}
	if a.Var != nil {
		s = append(s, a.Var.String())
	}
	return strings.Join(s, ", ")
}

func (a *CallExprArgs) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteExprs(", ", a.Values...)
	if a.Var != nil {
		if len(a.Values) > 0 {
			ctx.WriteString(", ")
		}
		ctx.WriteSingleByte('*')
		a.Var.Value.WriteCode(ctx)
	}
}

type NamedArgExpr struct {
	Lit   *StringLit
	Ident *IdentExpr
	Exp   Expr
}

func (e *NamedArgExpr) Name() string {
	if e.Lit != nil {
		return e.Lit.Value()
	}
	return e.Ident.Name
}

func (e *NamedArgExpr) String() string {
	return e.Expr().String()
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
	e.Expr().WriteCode(ctx)
}

// CallExprNamedArgs represents a call expression keyword arguments.
type CallExprNamedArgs struct {
	Names  []NamedArgExpr
	Values []Expr
	Var    *NamedArgVarLit
}

func (a *CallExprNamedArgs) Append(name NamedArgExpr, value Expr) *CallExprNamedArgs {
	a.Names = append(a.Names, name)
	a.Values = append(a.Values, value)
	return a
}

func (a *CallExprNamedArgs) AppendS(name string, value Expr) *CallExprNamedArgs {
	a.Names = append(a.Names, NamedArgExpr{Ident: &IdentExpr{Name: name}})
	a.Values = append(a.Values, value)
	return a
}

func (a *CallExprNamedArgs) Prepend(name NamedArgExpr, value Expr) *CallExprNamedArgs {
	a.Names = append([]NamedArgExpr{name}, a.Names...)
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
	return len(a.Names) > 0 || a.Var != nil
}

func (a *CallExprNamedArgs) NamesExpr() (r []Expr) {
	for _, v := range a.Names {
		r = append(r, v.Expr())
	}
	return r
}

func (a *CallExprNamedArgs) String() string {
	var s []string
	do := func(i int, name NamedArgExpr) (es string) {
		if name.Exp != nil {
			es = "["
		}
		es += name.Expr().String() + "="
		if a.Values[i] == nil {
			es += (&FlagLit{Value: true}).String()
		} else {
			es += a.Values[i].String()
		}
		if name.Exp != nil {
			es += "]"
		}
		return
	}

	for i, name := range a.Names {
		s = append(s, do(i, name))
	}
	if a.Var != nil {
		s = append(s, a.Var.String())
	}
	return strings.Join(s, ", ")
}

func (a *CallExprNamedArgs) WriteCode(ctx *CodeWriteContext) {
	l := len(a.Names) - 1
	do := func(i int, name NamedArgExpr) {
		if name.Exp != nil {
			ctx.WriteSingleByte('[')
			defer ctx.WriteSingleByte(']')
		}

		if a.Values[i] == nil {
			if name.Lit != nil && name.Lit.CanIdent() {
				ctx.WriteString(name.Lit.Value())
			} else if name.Ident != nil {
				ctx.WriteString(name.Ident.String())
			} else {
				name.Expr().WriteCode(ctx)
			}
		} else {
			ctx.WriteString(name.Expr().String() + "=")
			a.Values[i].WriteCode(ctx)
		}
	}
	for i, name := range a.Names {
		do(i, name)
		if i != l || a.Var != nil {
			ctx.WriteString(", ")
		}
	}
	if a.Var != nil {
		ctx.WriteString(a.Var.String())
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

// FuncExpr represents a function literal.
type FuncExpr struct {
	ast.NodeData
	Type      *FuncType
	Body      *BlockStmt
	LambdaPos source.Pos
	BodyExpr  Expr
}

func (e *FuncExpr) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *FuncExpr) Pos() source.Pos {
	return e.Type.Pos()
}

// End returns the position of first character immediately after the node.
func (e *FuncExpr) End() source.Pos {
	if e.BodyExpr != nil {
		return e.BodyExpr.End()
	}
	return e.Body.End()
}

func (e *FuncExpr) String() string {
	var f string
	if e.Type.FuncPos != 0 {
		f = "func"
	}
	s := f + e.Type.String() + " "
	if e.BodyExpr != nil {
		s += "=> " + e.BodyExpr.String()
	} else {
		s += e.Body.String()
	}
	return s
}

func (e *FuncExpr) WriteCode(ctx *CodeWriteContext) {
	var f string
	if e.Type.FuncPos != 0 {
		f = "func"
	}

	ctx.WriteString(f + e.Type.String() + " ")

	if e.BodyExpr != nil {
		ctx.WriteString(e.BodyExpr.String())
	} else {
		e.Body.WriteCodeInSelfDepth(ctx, true)
	}
}

// ClosureExpr represents a function closure literal.
type ClosureExpr struct {
	ast.NodeData
	Params      FuncParams
	LambdaToken token.Token
	LambdaPos   source.Pos
	Body        Expr
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

func (e *ClosureExpr) String() string {
	return e.Params.String() + " " + e.LambdaToken.String() + " " + e.Body.String()
}

func (e *ClosureExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(e.Params.String(), " ", e.LambdaToken.String(), " ")
	if block, ok := e.Body.(*BlockExpr); ok {
		block.WriteCodeInSelfDepth(ctx, true)
	} else {
		e.Body.WriteCode(ctx)
	}
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
	for i, m := range e.Elements {
		if i > 0 {
			ctx.WriteString(", ")
		}
		m.WriteCode(ctx)
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
	ctx.WriteExprs(", ", e.Elements...)
	ctx.WriteSingleByte(']')
}
