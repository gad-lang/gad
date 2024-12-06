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
	"strconv"
	"strings"

	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/repr"
	"github.com/gad-lang/gad/token"
	"github.com/gad-lang/gad/utils"
)

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

func (s *AssignStmt) WriteCode(ctx *CodeWriteContext) {
	ctx.WritePrefix()
	ctx.WriteExprs(", ", s.LHS...)
	ctx.WriteString(" " + s.Token.String() + " ")
	ctx.Depth++
	ctx.WriteExprs(", ", s.RHS...)
	ctx.Depth--
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

func (s *BadStmt) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(s.String())
}

// BlockStmt represents a block statement.
type BlockStmt struct {
	Stmts  Stmts
	LBrace ast.Literal
	RBrace ast.Literal
	Scoped bool
}

func (s *BlockStmt) StmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *BlockStmt) Pos() source.Pos {
	return s.LBrace.Pos
}

// End returns the position of first character immediately after the node.
func (s *BlockStmt) End() source.Pos {
	return s.RBrace.End()
}

func (s *BlockStmt) String() string {
	var b strings.Builder
	if s.Scoped {
		b.WriteRune('.')
	}
	b.WriteString(s.LBrace.Value)
	b.WriteString(s.Stmts.String())
	b.WriteString(s.RBrace.Value)
	return b.String()
}

func (s *BlockStmt) WriteCode(ctx *CodeWriteContext) {
	s.WriteCodeInSelfDepth(ctx, false)
}

func (s *BlockStmt) WriteCodeInSelfDepth(ctx *CodeWriteContext, selfDepth bool) {
	if s.Scoped {
		ctx.WritePrefix()
		ctx.WriteString(".{")
		ctx.WriteSecondLine()
		selfDepth = true
	} else {
		ctx.WriteByte('{')
		ctx.WriteSecondLine()
	}
	if selfDepth {
		ctx.Depth++
		ctx.WriteStmts(s.Stmts...)
		ctx.Depth--
	} else {
		ctx.WriteStmts(s.Stmts...)
	}
	ctx.WriteSemi()
	if selfDepth {
		ctx.WritePrefix()
	} else {
		ctx.WritePrevPrefix()
	}
	ctx.WriteByte('}')
}

// BranchStmt represents a branch statement.
type BranchStmt struct {
	Token    token.Token
	TokenPos source.Pos
	Label    *IdentExpr
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

func (s *BranchStmt) WriteCode(ctx *CodeWriteContext) {
	ctx.WritePrefix()
	ctx.WriteString(s.String())
	ctx.WriteSemi()
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

func (s *EmptyStmt) WriteCode(*CodeWriteContext) {}

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

func (s *ExprStmt) WriteCode(ctx *CodeWriteContext) {
	ctx.WritePrefix()
	s.Expr.WriteCode(ctx)
}

// ForInStmt represents a for-in statement.
type ForInStmt struct {
	ForPos   source.Pos
	Key      *IdentExpr
	Value    *IdentExpr
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

func (s *ForInStmt) WriteCode(ctx *CodeWriteContext) {
	ctx.WritePrefix()
	ctx.WriteString("for " + s.Key.String())

	if s.Value != nil {
		ctx.WriteString(", " + s.Value.String())
	}

	ctx.WriteString(" in " + s.Iterable.String() + " ")

	s.Body.WriteCodeInSelfDepth(ctx, true)

	if s.Else != nil {
		ctx.WriteString(" else ")
		s.Else.WriteCode(ctx)
	}
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

func (s *ForStmt) WriteCode(ctx *CodeWriteContext) {
	ctx.WritePrefix()
	ctx.WriteString("for ")

	if s.Init != nil {
		ctx.WithoutPrefix().WriteStmts(s.Init)
		if s.Cond != nil || s.Post != nil {
			ctx.WriteString("; ")
		}
	}

	if s.Cond != nil {
		s.Cond.WriteCode(ctx)
		if s.Post != nil {
			ctx.WriteString("; ")
		}
	}

	if s.Post != nil {
		ctx.WriteStmts(s.Post)
	}

	s.Body.WriteCodeInSelfDepth(ctx, true)
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

func (s *IfStmt) WriteCode(ctx *CodeWriteContext) {
	ctx.WritePrefix()
	ctx.WriteString("if ")
	if s.Init != nil {
		ctx.WithoutPrefix().WriteStmts(s.Init)
		ctx.WriteString("; ")
	}
	s.Cond.WriteCode(ctx)
	ctx.WriteByte(' ')
	ctx.Depth++
	s.Body.WriteCode(ctx)
	ctx.Depth--
	if s.Else != nil {
		ctx.WriteString(" else ")
		if block, ok := s.Else.(*BlockStmt); ok {
			block.WriteCodeInSelfDepth(ctx, true)
		} else {
			ctx.WriteStmts(s.Else)
		}
	}
	return
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

func (s *IncDecStmt) WriteCode(ctx *CodeWriteContext) {
	ctx.WritePrefix()
	s.Expr.WriteCode(ctx)
	ctx.WriteString(s.Token.String())
}

// ReturnStmt represents a return statement.
type ReturnStmt struct {
	Return
}

func (s *ReturnStmt) StmtNode() {}

func (s *ReturnStmt) WriteCode(ctx *CodeWriteContext) {
	ctx.WritePrefix()
	s.Return.WriteCode(ctx)
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

func (s *TryStmt) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString("try")
	s.Body.WriteCode(ctx)

	if s.Catch != nil {
		ctx.WriteString(" ")
		s.Catch.WriteCode(ctx)
	}

	if s.Finally != nil {
		ctx.WriteString(" ")
		s.Finally.WriteCode(ctx)
	}
}

// CatchStmt represents an catch statement.
type CatchStmt struct {
	CatchPos source.Pos
	Ident    *IdentExpr // can be nil if ident is missing
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

func (s *CatchStmt) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString("catch " + s.Ident.String())
	s.Body.WriteCode(ctx)
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

func (s *FinallyStmt) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString("finally ")
	s.Body.WriteCode(ctx)
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

func (s *ThrowStmt) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString("throw ")
	if s.Expr != nil {
		s.Expr.WriteCode(ctx)
	}
}

type MixedTextStmtFlag uint

const (
	RemoveLeftSpaces MixedTextStmtFlag = 1 << iota
	RemoveRightSpaces
)

func (s MixedTextStmtFlag) Has(f MixedTextStmtFlag) bool {
	return s&f != 0
}

func (s MixedTextStmtFlag) String() string {
	var v []string
	if s.Has(RemoveLeftSpaces) {
		v = append(v, "RemoveLeftSpaces")
	}
	if s.Has(RemoveRightSpaces) {
		v = append(v, "RemoveRightSpaces")
	}
	return strings.Join(v, "|")
}

// MixedTextStmt represents an MixedTextStmt.
type MixedTextStmt struct {
	Lit    ast.Literal
	Flags  MixedTextStmtFlag
	LParen source.Pos
	RParen source.Pos
}

func (s *MixedTextStmt) Pos() source.Pos {
	return s.Lit.Pos
}

func (s *MixedTextStmt) End() source.Pos {
	return s.Lit.End()
}

func (s *MixedTextStmt) StmtNode() {
}

func (s *MixedTextStmt) ExprNode() {
}

func (s *MixedTextStmt) TrimLinePrefix(prefix string) {
	lit := s.Lit
	lines := strings.Split(lit.Value, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimLeft(line, prefix)
	}
	lit.Value = strings.Join(lines, "\n")
}

func (s *MixedTextStmt) String() string {
	return s.Lit.Value
}

func (s *MixedTextStmt) Value() string {
	_, v := utils.TrimStringSpace(s.Lit.Value, s.Flags.Has(RemoveLeftSpaces), s.Flags.Has(RemoveRightSpaces))
	return v
}

func (s *MixedTextStmt) ValidLit() ast.Literal {
	start, v := utils.TrimStringSpace(s.Lit.Value, s.Flags.Has(RemoveLeftSpaces), s.Flags.Has(RemoveRightSpaces))
	return ast.Literal{
		Value: v,
		Pos:   s.Lit.Pos + source.Pos(start),
	}
}

func (s *MixedTextStmt) WriteCode(ctx *CodeWriteContext) {
	if ctx.Transpile != nil {
		ctx.WritePrefix()
		ctx.WriteString(ctx.Transpile.WriteFunc)
		ctx.WriteByte('(')
		ctx.WriteString(ctx.Transpile.RawStrFuncStart)
		ctx.WriteString(strconv.Quote(s.Value()))
		ctx.WriteString(ctx.Transpile.RawStrFuncEnd)
		ctx.WriteByte(')')
		ctx.WriteSemi()
	} else {
		ctx.WriteString(s.Lit.Value)
	}
}

// MixedValueStmt represents to text wrapped expression.
type MixedValueStmt struct {
	Expr             Expr
	StartLit         ast.Literal
	EndLit           ast.Literal
	RemoveLeftSpace  bool
	RemoveRightSpace bool
	Eq               bool
}

func (s *MixedValueStmt) StmtNode() {}

func (s *MixedValueStmt) ExprNode() {
}

// Pos returns the position of first character belonging to the node.
func (s *MixedValueStmt) Pos() source.Pos {
	return s.StartLit.Pos
}

// End returns the position of first character immediately after the node.
func (s *MixedValueStmt) End() source.Pos {
	return s.EndLit.Pos
}

func (s *MixedValueStmt) String() string {
	var b strings.Builder
	b.WriteString(s.StartLit.Value)
	if s.RemoveLeftSpace {
		b.WriteByte('-')
	}
	if s.Eq {
		b.WriteByte('=')
	}
	b.WriteString(s.Expr.String())
	if s.RemoveRightSpace {
		b.WriteByte('-')
	}
	b.WriteString(s.EndLit.Value)
	return b.String()
}

func (s *MixedValueStmt) WriteCode(ctx *CodeWriteContext) {
	if ctx.Transpile != nil {
		ctx.WritePrefix()
		ctx.WriteString(ctx.Transpile.WriteFunc)
		ctx.WriteByte('(')
		s.Expr.WriteCode(ctx)
		ctx.WriteByte(')')
		ctx.WriteSemi()
	} else {
		ctx.WriteString(s.StartLit.Value)
		if s.RemoveLeftSpace {
			ctx.WriteByte('-')
		}
		if s.Eq {
			ctx.WriteByte('=')
		}
		s.Expr.WriteCode(ctx)
		if s.RemoveRightSpace {
			ctx.WriteByte('-')
		}
		ctx.WriteString(s.EndLit.Value)
	}
}

type ConfigOptions struct {
	Mixed      bool
	NoMixed    bool
	MixedStart string
	MixedEnd   string
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
		elements = append(elements, m.ElementString())
	}
	return "# gad: " + strings.Join(elements, ", ")
}

func (c *ConfigStmt) WriteCode(ctx *CodeWriteContext) {
	if ctx.Transpile == nil {
		ctx.WritePrefix()
		ctx.WriteString("# gad: ")

		last := len(c.Elements) - 1
		for i, el := range c.Elements {
			ctx.WriteString(el.ElementString())
			if i != last {
				ctx.WriteString(", ")
			}
		}
		ctx.WriteByte('\n')
	}
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
			} else if b, ok := k.Value.(*FlagLit); ok {
				if b.Value {
					c.Options.Mixed = true
				} else {
					c.Options.NoMixed = true
				}
			}
		case "mixed_start":
			if s, ok := k.Value.(*StringLit); ok {
				c.Options.MixedStart = s.Value()
			}
		case "mixed_end":
			if s, ok := k.Value.(*StringLit); ok {
				c.Options.MixedEnd = s.Value()
			}
		}
	}
}

func (c *ConfigStmt) StmtNode() {
}

type StmtsExpr struct {
	Stmts Stmts
}

func (s *StmtsExpr) Pos() source.Pos {
	return s.Stmts[0].Pos()
}

func (s *StmtsExpr) End() source.Pos {
	return s.Stmts[len(s.Stmts)-1].End()
}

func (s *StmtsExpr) String() string {
	var w bytes.Buffer
	NewCodeWriteContext(NewCodeWriter(&w)).WriteStmts(s.Stmts...)
	return w.String()
}

func (s *StmtsExpr) ExprNode() {
}

func (s *StmtsExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.Depth++
	ctx.WriteStmts(s.Stmts...)
	ctx.Depth--
}

type CodeBeginStmt struct {
	Lit         ast.Literal
	RemoveSpace bool
}

func (c CodeBeginStmt) Pos() source.Pos {
	return c.Lit.Pos
}

func (c CodeBeginStmt) End() source.Pos {
	return c.Lit.End()
}

func (c CodeBeginStmt) String() string {
	if c.RemoveSpace {
		return c.Lit.Value + "-"
	}
	return c.Lit.Value
}

func (c CodeBeginStmt) StmtNode() {
}

func (s *CodeBeginStmt) WriteCode(ctx *CodeWriteContext) {
	if ctx.Transpile == nil {
		ctx.WriteString(s.String())
		ctx.Depth++
	}
}

type CodeEndStmt struct {
	Lit         ast.Literal
	RemoveSpace bool
}

func (c CodeEndStmt) Pos() source.Pos {
	return c.Lit.Pos
}

func (c CodeEndStmt) End() source.Pos {
	return c.Lit.End()
}

func (c CodeEndStmt) String() string {
	if c.RemoveSpace {
		return "-" + c.Lit.Value
	}
	return c.Lit.Value
}

func (c CodeEndStmt) StmtNode() {
}

func (s *CodeEndStmt) WriteCode(ctx *CodeWriteContext) {
	if ctx.Transpile == nil {
		ctx.WriteString(s.String())
		ctx.Depth--
	}
}
