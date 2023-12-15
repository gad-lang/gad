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
	"fmt"
	"strings"

	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/repr"
	"github.com/gad-lang/gad/token"
)

// Stmt represents a statement in the AST.
type Stmt interface {
	ast.Node
	StmtNode()
}

// IsStatement returns true if given value is implements interface{ StmtNode() }.
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
	TokenPos source.Pos
}

func (s *AssignStmt) StmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *AssignStmt) Pos() source.Pos {
	return s.LHS[0].Pos()
}

// End returns the position of first character immediately after the node.
func (s *AssignStmt) End() source.Pos {
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
	From source.Pos
	To   source.Pos
}

func (s *BadStmt) StmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *BadStmt) Pos() source.Pos {
	return s.From
}

// End returns the position of first character immediately after the node.
func (s *BadStmt) End() source.Pos {
	return s.To
}

func (s *BadStmt) String() string {
	return repr.Quote("bad statement")
}

// BlockStmt represents a block statement.
type BlockStmt struct {
	Stmts  []Stmt
	LBrace source.Pos
	RBrace source.Pos
}

func (s *BlockStmt) StmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *BlockStmt) Pos() source.Pos {
	return s.LBrace
}

// End returns the position of first character immediately after the node.
func (s *BlockStmt) End() source.Pos {
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
	TokenPos source.Pos
	Label    *Ident
}

func (s *BranchStmt) StmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *BranchStmt) Pos() source.Pos {
	return s.TokenPos
}

// End returns the position of first character immediately after the node.
func (s *BranchStmt) End() source.Pos {
	if s.Label != nil {
		return s.Label.End()
	}

	return source.Pos(int(s.TokenPos) + len(s.Token.String()))
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
	Semicolon source.Pos
	Implicit  bool
}

func (s *EmptyStmt) StmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *EmptyStmt) Pos() source.Pos {
	return s.Semicolon
}

// End returns the position of first character immediately after the node.
func (s *EmptyStmt) End() source.Pos {
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

func (s *ExprStmt) StmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *ExprStmt) Pos() source.Pos {
	return s.Expr.Pos()
}

// End returns the position of first character immediately after the node.
func (s *ExprStmt) End() source.Pos {
	return s.Expr.End()
}

func (s *ExprStmt) String() string {
	return s.Expr.String()
}

// ForInStmt represents a for-in statement.
type ForInStmt struct {
	ForPos   source.Pos
	Key      *Ident
	Value    *Ident
	Iterable Expr
	Body     *BlockStmt
	Else     *BlockStmt
}

func (s *ForInStmt) StmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *ForInStmt) Pos() source.Pos {
	return s.ForPos
}

// End returns the position of first character immediately after the node.
func (s *ForInStmt) End() source.Pos {
	return s.Body.End()
}

func (s *ForInStmt) String() string {
	var str = "for " + s.Key.String()
	if s.Value != nil {
		str += ", " + s.Value.String()
	}
	str += " in " + s.Iterable.String() +
		" " + s.Body.String()
	if s.Else != nil {
		str += " else " + s.Else.String()
	}
	return str
}

// ForStmt represents a for statement.
type ForStmt struct {
	ForPos source.Pos
	Init   Stmt
	Cond   Expr
	Post   Stmt
	Body   *BlockStmt
}

func (s *ForStmt) StmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *ForStmt) Pos() source.Pos {
	return s.ForPos
}

// End returns the position of first character immediately after the node.
func (s *ForStmt) End() source.Pos {
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

	var str = "for "

	if init != "" || post != "" {
		str += init + " ; " + cond + " ; " + post
	} else {
		str += cond
	}

	str += s.Body.String()
	return str
}

// IfStmt represents an if statement.
type IfStmt struct {
	IfPos source.Pos
	Init  Stmt
	Cond  Expr
	Body  *BlockStmt
	Else  Stmt // else branch; or nil
}

func (s *IfStmt) StmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *IfStmt) Pos() source.Pos {
	return s.IfPos
}

// End returns the position of first character immediately after the node.
func (s *IfStmt) End() source.Pos {
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
	TokenPos source.Pos
}

func (s *IncDecStmt) StmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *IncDecStmt) Pos() source.Pos {
	return s.Expr.Pos()
}

// End returns the position of first character immediately after the node.
func (s *IncDecStmt) End() source.Pos {
	return source.Pos(int(s.TokenPos) + 2)
}

func (s *IncDecStmt) String() string {
	return s.Expr.String() + s.Token.String()
}

// ReturnStmt represents a return statement.
type ReturnStmt struct {
	ReturnPos source.Pos
	Result    Expr
}

func (s *ReturnStmt) StmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *ReturnStmt) Pos() source.Pos {
	return s.ReturnPos
}

// End returns the position of first character immediately after the node.
func (s *ReturnStmt) End() source.Pos {
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
	TryPos  source.Pos
	Body    *BlockStmt
	Catch   *CatchStmt   // catch branch; or nil
	Finally *FinallyStmt // finally branch; or nil
}

func (s *TryStmt) StmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *TryStmt) Pos() source.Pos {
	return s.TryPos
}

// End returns the position of first character immediately after the node.
func (s *TryStmt) End() source.Pos {
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
	CatchPos source.Pos
	Ident    *Ident // can be nil if ident is missing
	Body     *BlockStmt
}

func (s *CatchStmt) StmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *CatchStmt) Pos() source.Pos {
	return s.CatchPos
}

// End returns the position of first character immediately after the node.
func (s *CatchStmt) End() source.Pos {
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
	FinallyPos source.Pos
	Body       *BlockStmt
}

func (s *FinallyStmt) StmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *FinallyStmt) Pos() source.Pos {
	return s.FinallyPos
}

// End returns the position of first character immediately after the node.
func (s *FinallyStmt) End() source.Pos {
	return s.Body.End()
}

func (s *FinallyStmt) String() string {
	return "finally " + s.Body.String()
}

// ThrowStmt represents an throw statement.
type ThrowStmt struct {
	ThrowPos source.Pos
	Expr     Expr
}

func (s *ThrowStmt) StmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *ThrowStmt) Pos() source.Pos {
	return s.ThrowPos
}

// End returns the position of first character immediately after the node.
func (s *ThrowStmt) End() source.Pos {
	return s.Expr.End()
}

func (s *ThrowStmt) String() string {
	var expr string
	if s.Expr != nil {
		expr = s.Expr.String()
	}
	return "throw " + expr
}

// RawStringStmt represents an RawStringStmt.
type RawStringStmt struct {
	Lits []*RawStringLit
}

func (e *RawStringStmt) Pos() source.Pos {
	return e.Lits[0].Pos()
}

func (e *RawStringStmt) End() source.Pos {
	return e.Lits[len(e.Lits)-1].Pos()
}

func (e *RawStringStmt) StmtNode() {
}

func (e *RawStringStmt) ExprNode() {
}

func (e *RawStringStmt) TrimLinePrefix(prefix string) {
	for _, lit := range e.Lits {
		lines := strings.Split(lit.Literal, "\n")
		for i, line := range lines {
			lines[i] = strings.TrimLeft(line, prefix)
		}
		lit.Literal = strings.Join(lines, "\n")
	}
}

func (e *RawStringStmt) Quoted() string {
	var b strings.Builder
	b.WriteByte('`')
	for _, lit := range e.Lits {
		s := lit.QuotedValue()
		b.WriteString(s[1 : len(s)-1])
	}
	b.WriteByte('`')
	return b.String()
}

func (e *RawStringStmt) Unquoted() string {
	var b strings.Builder
	for _, lit := range e.Lits {
		b.WriteString(lit.UnquotedValue())
	}
	return b.String()
}

func (e *RawStringStmt) String() string {
	if e != nil {
		return "#{= " + e.Quoted() + " }"
	}
	return nullRep
}

func (e *RawStringStmt) Value() string {
	var b strings.Builder
	for _, lit := range e.Lits {
		b.WriteString(lit.UnquotedValue())
	}
	return b.String()
}

// ExprToTextStmt represents to text wrapped expression.
type ExprToTextStmt struct {
	Expr     Expr
	StartLit ast.Literal
	EndLit   ast.Literal
}

func NewExprToTextStmt(expr Expr) *ExprToTextStmt {
	return &ExprToTextStmt{
		Expr:     expr,
		StartLit: ast.Literal{Value: "#{="},
		EndLit:   ast.Literal{Value: "}"},
	}
}

func (e *ExprToTextStmt) StmtNode() {}

func (e *ExprToTextStmt) ExprNode() {
}

// Pos returns the position of first character belonging to the node.
func (e *ExprToTextStmt) Pos() source.Pos {
	return e.StartLit.Pos
}

// End returns the position of first character immediately after the node.
func (e *ExprToTextStmt) End() source.Pos {
	return e.EndLit.Pos
}

func (e *ExprToTextStmt) String() string {
	return e.StartLit.Value + " " + e.Expr.String() + " " + e.EndLit.Value
}

type ConfigOptions struct {
	Mixed          bool
	NoMixed        bool
	WriteFunc      Expr
	ExprToTextFunc Expr
}

type ConfigStmt struct {
	ConfigPos source.Pos
	Elements  []*KeyValueLit
	Options   ConfigOptions
}

func (c *ConfigStmt) Pos() source.Pos {
	return c.ConfigPos
}

func (c *ConfigStmt) End() source.Pos {
	if len(c.Elements) == 0 {
		return c.ConfigPos + 1
	}
	return c.Elements[len(c.Elements)-1].End()
}

func (c *ConfigStmt) String() string {
	var elements []string
	for _, m := range c.Elements {
		elements = append(elements, m.String())
	}
	return "# gad: " + strings.Join(elements, ", ")
}

func (c *ConfigStmt) ParseElements() {
	for _, k := range c.Elements {
		switch k.Key.String() {
		case "mixed":
			if k.Value == nil {
				c.Options.Mixed = true
			} else if b, ok := k.Value.(*BoolLit); ok {
				if b.Value {
					c.Options.Mixed = true
				} else {
					c.Options.NoMixed = true
				}
			}
		case "writer":
			if k.Value != nil {
				c.Options.WriteFunc = k.Value
			}
		case "expr_to_text":
			if k.Value != nil {
				c.Options.ExprToTextFunc = k.Value
			}
		}
	}
}

func (c *ConfigStmt) StmtNode() {
}

type StmtsExpr struct {
	Stmts []Stmt
}

func (s *StmtsExpr) Pos() source.Pos {
	return s.Stmts[0].Pos()
}

func (s *StmtsExpr) End() source.Pos {
	return s.Stmts[len(s.Stmts)-1].End()
}

func (s *StmtsExpr) String() string {
	var str = make([]string, len(s.Stmts))
	for i, stmt := range s.Stmts {
		str[i] = fmt.Sprint(stmt)
	}
	return strings.Join(str, "; ")
}

func (s *StmtsExpr) ExprNode() {
}
