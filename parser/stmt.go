// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

// Copyright (c) 2019 Daniel Kang.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE.tengo file.

// Copyright 2009 The ToInterface Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.golang file.

package parser

import (
	"strconv"
	"strings"

	"github.com/gad-lang/gad/token"
)

// Stmt represents a statement in the AST.
type Stmt interface {
	Node
	stmtNode()
}

// IsStatement returns true if given value is implements interface{ stmtNode() }.
func IsStatement(v any) bool {
	_, ok := v.(interface {
		stmtNode()
	})
	return ok
}

// AssignStmt represents an assignment statement.
type AssignStmt struct {
	LHS      []Expr
	RHS      []Expr
	Token    token.Token
	TokenPos Pos
}

func (s *AssignStmt) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *AssignStmt) Pos() Pos {
	return s.LHS[0].Pos()
}

// End returns the position of first character immediately after the node.
func (s *AssignStmt) End() Pos {
	return s.RHS[len(s.RHS)-1].End()
}

func (s *AssignStmt) String() string {
	var lhs, rhs []string
	for _, e := range s.LHS {
		lhs = append(lhs, e.String())
	}
	for _, e := range s.RHS {
		rhs = append(rhs, e.String())
	}
	return strings.Join(lhs, ", ") + " " + s.Token.String() +
		" " + strings.Join(rhs, ", ")
}

// BadStmt represents a bad statement.
type BadStmt struct {
	From Pos
	To   Pos
}

func (s *BadStmt) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *BadStmt) Pos() Pos {
	return s.From
}

// End returns the position of first character immediately after the node.
func (s *BadStmt) End() Pos {
	return s.To
}

func (s *BadStmt) String() string {
	return "<bad statement>"
}

// BlockStmt represents a block statement.
type BlockStmt struct {
	Stmts  []Stmt
	LBrace Pos
	RBrace Pos
}

func (s *BlockStmt) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *BlockStmt) Pos() Pos {
	return s.LBrace
}

// End returns the position of first character immediately after the node.
func (s *BlockStmt) End() Pos {
	return s.RBrace + 1
}

func (s *BlockStmt) String() string {
	var list []string
	for _, e := range s.Stmts {
		list = append(list, e.String())
	}
	return "{" + strings.Join(list, "; ") + "}"
}

// BranchStmt represents a branch statement.
type BranchStmt struct {
	Token    token.Token
	TokenPos Pos
	Label    *Ident
}

func (s *BranchStmt) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *BranchStmt) Pos() Pos {
	return s.TokenPos
}

// End returns the position of first character immediately after the node.
func (s *BranchStmt) End() Pos {
	if s.Label != nil {
		return s.Label.End()
	}

	return Pos(int(s.TokenPos) + len(s.Token.String()))
}

func (s *BranchStmt) String() string {
	var label string
	if s.Label != nil {
		label = " " + s.Label.Name
	}
	return s.Token.String() + label
}

// EmptyStmt represents an empty statement.
type EmptyStmt struct {
	Semicolon Pos
	Implicit  bool
}

func (s *EmptyStmt) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *EmptyStmt) Pos() Pos {
	return s.Semicolon
}

// End returns the position of first character immediately after the node.
func (s *EmptyStmt) End() Pos {
	if s.Implicit {
		return s.Semicolon
	}
	return s.Semicolon + 1
}

func (s *EmptyStmt) String() string {
	return ";"
}

// ExprStmt represents an expression statement.
type ExprStmt struct {
	Expr Expr
}

func (s *ExprStmt) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *ExprStmt) Pos() Pos {
	return s.Expr.Pos()
}

// End returns the position of first character immediately after the node.
func (s *ExprStmt) End() Pos {
	return s.Expr.End()
}

func (s *ExprStmt) String() string {
	return s.Expr.String()
}

// ForInStmt represents a for-in statement.
type ForInStmt struct {
	ForPos   Pos
	Key      *Ident
	Value    *Ident
	Iterable Expr
	Body     *BlockStmt
}

func (s *ForInStmt) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *ForInStmt) Pos() Pos {
	return s.ForPos
}

// End returns the position of first character immediately after the node.
func (s *ForInStmt) End() Pos {
	return s.Body.End()
}

func (s *ForInStmt) String() string {
	if s.Value != nil {
		return "for " + s.Key.String() + ", " + s.Value.String() +
			" in " + s.Iterable.String() + " " + s.Body.String()
	}
	return "for " + s.Key.String() + " in " + s.Iterable.String() +
		" " + s.Body.String()
}

// ForStmt represents a for statement.
type ForStmt struct {
	ForPos Pos
	Init   Stmt
	Cond   Expr
	Post   Stmt
	Body   *BlockStmt
}

func (s *ForStmt) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *ForStmt) Pos() Pos {
	return s.ForPos
}

// End returns the position of first character immediately after the node.
func (s *ForStmt) End() Pos {
	return s.Body.End()
}

func (s *ForStmt) String() string {
	var init, cond, post string
	if s.Init != nil {
		init = s.Init.String()
	}
	if s.Cond != nil {
		cond = s.Cond.String() + " "
	}
	if s.Post != nil {
		post = s.Post.String()
	}

	if init != "" || post != "" {
		return "for " + init + " ; " + cond + " ; " + post + s.Body.String()
	}
	return "for " + cond + s.Body.String()
}

// IfStmt represents an if statement.
type IfStmt struct {
	IfPos Pos
	Init  Stmt
	Cond  Expr
	Body  *BlockStmt
	Else  Stmt // else branch; or nil
}

func (s *IfStmt) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *IfStmt) Pos() Pos {
	return s.IfPos
}

// End returns the position of first character immediately after the node.
func (s *IfStmt) End() Pos {
	if s.Else != nil {
		return s.Else.End()
	}
	return s.Body.End()
}

func (s *IfStmt) String() string {
	var initStmt, elseStmt string
	if s.Init != nil {
		initStmt = s.Init.String() + "; "
	}
	if s.Else != nil {
		elseStmt = " else " + s.Else.String()
	}
	return "if " + initStmt + s.Cond.String() + " " +
		s.Body.String() + elseStmt
}

// IncDecStmt represents increment or decrement statement.
type IncDecStmt struct {
	Expr     Expr
	Token    token.Token
	TokenPos Pos
}

func (s *IncDecStmt) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *IncDecStmt) Pos() Pos {
	return s.Expr.Pos()
}

// End returns the position of first character immediately after the node.
func (s *IncDecStmt) End() Pos {
	return Pos(int(s.TokenPos) + 2)
}

func (s *IncDecStmt) String() string {
	return s.Expr.String() + s.Token.String()
}

// ReturnStmt represents a return statement.
type ReturnStmt struct {
	ReturnPos Pos
	Result    Expr
}

func (s *ReturnStmt) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *ReturnStmt) Pos() Pos {
	return s.ReturnPos
}

// End returns the position of first character immediately after the node.
func (s *ReturnStmt) End() Pos {
	if s.Result != nil {
		return s.Result.End()
	}
	return s.ReturnPos + 6
}

func (s *ReturnStmt) String() string {
	if s.Result != nil {
		return "return " + s.Result.String()
	}
	return "return"
}

// TryStmt represents an try statement.
type TryStmt struct {
	TryPos  Pos
	Body    *BlockStmt
	Catch   *CatchStmt   // catch branch; or nil
	Finally *FinallyStmt // finally branch; or nil
}

func (s *TryStmt) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *TryStmt) Pos() Pos {
	return s.TryPos
}

// End returns the position of first character immediately after the node.
func (s *TryStmt) End() Pos {
	if s.Finally != nil {
		return s.Finally.End()
	}
	if s.Catch != nil {
		return s.Catch.End()
	}
	return s.Body.End()
}

func (s *TryStmt) String() string {
	ret := "try " + s.Body.String()

	if s.Catch != nil {
		ret += " " + s.Catch.String()
	}
	if s.Finally != nil {
		ret += " " + s.Finally.String()
	}
	return ret
}

// CatchStmt represents an catch statement.
type CatchStmt struct {
	CatchPos Pos
	Ident    *Ident // can be nil if ident is missing
	Body     *BlockStmt
}

func (s *CatchStmt) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *CatchStmt) Pos() Pos {
	return s.CatchPos
}

// End returns the position of first character immediately after the node.
func (s *CatchStmt) End() Pos {
	return s.Body.End()
}

func (s *CatchStmt) String() string {
	var ident string
	if s.Ident != nil {
		ident = s.Ident.String() + " "
	}
	return "catch " + ident + s.Body.String()
}

// FinallyStmt represents an finally statement.
type FinallyStmt struct {
	FinallyPos Pos
	Body       *BlockStmt
}

func (s *FinallyStmt) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *FinallyStmt) Pos() Pos {
	return s.FinallyPos
}

// End returns the position of first character immediately after the node.
func (s *FinallyStmt) End() Pos {
	return s.Body.End()
}

func (s *FinallyStmt) String() string {
	return "finally " + s.Body.String()
}

// ThrowStmt represents an throw statement.
type ThrowStmt struct {
	ThrowPos Pos
	Expr     Expr
}

func (s *ThrowStmt) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *ThrowStmt) Pos() Pos {
	return s.ThrowPos
}

// End returns the position of first character immediately after the node.
func (s *ThrowStmt) End() Pos {
	return s.Expr.End()
}

func (s *ThrowStmt) String() string {
	var expr string
	if s.Expr != nil {
		expr = s.Expr.String()
	}
	return "throw " + expr
}

// TextStmt represents an TextStmt.
type TextStmt struct {
	Literal string
	TextPos Pos
}

func (e *TextStmt) stmtNode() {
}

func (e *TextStmt) exprNode() {
}

// Pos returns the position of first character belonging to the node.
func (e *TextStmt) Pos() Pos {
	return e.TextPos
}

// End returns the position of first character immediately after the node.
func (e *TextStmt) End() Pos {
	return Pos(int(e.TextPos) + len(e.Literal))
}

func (e *TextStmt) String() string {
	if e != nil {
		return "#{= " + strconv.Quote(e.Literal) + " }"
	}
	return nullRep
}

// ExprToTextStmt represents to text wrapped expression.
type ExprToTextStmt struct {
	Expr     Expr
	StartLit Literal
	EndLit   Literal
}

func (e *ExprToTextStmt) stmtNode() {}

func (e *ExprToTextStmt) exprNode() {
}

// Pos returns the position of first character belonging to the node.
func (e *ExprToTextStmt) Pos() Pos {
	return e.StartLit.Pos
}

// End returns the position of first character immediately after the node.
func (e *ExprToTextStmt) End() Pos {
	return e.EndLit.Pos
}

func (e *ExprToTextStmt) String() string {
	return e.StartLit.Value + " " + e.Expr.String() + " " + e.EndLit.Value
}

type ConfigOptions struct {
	Mixed     bool
	NoMixed   bool
	WriteFunc Expr
}

type ConfigStmt struct {
	ConfigPos Pos
	EndPos    Pos
	Literal   string
	Options   ConfigOptions
}

func (c *ConfigStmt) Pos() Pos {
	return c.ConfigPos
}

func (c *ConfigStmt) End() Pos {
	return c.EndPos
}

func (c *ConfigStmt) String() string {
	return "# gad: " + c.Literal
}

func (c *ConfigStmt) stmtNode() {
}
