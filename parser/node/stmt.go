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
	ctx.WriteExprs(", ", s.LHS...)
	ctx.WriteString(" " + s.Token.String() + " ")
	ctx.WriteExprs(", ", s.RHS...)
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
	// DeferClaimed marks a function body whose `defer` statements have already
	// been claimed by an enclosing defer wrapper, so it must not be wrapped
	// again (its defers register to the outer $__defers list).
	DeferClaimed bool
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
	var (
		b    strings.Builder
		data = s.Stmts.String()
	)

	b.WriteString(s.LBrace.Value)
	rb := s.RBrace.Value

	if len(data) > 0 {
		b.WriteString(" ")
		b.WriteString(data)
		b.WriteString(" ")
	} else if len(rb) > 0 && rb[0] >= 'a' && rb[0] <= 'z' {
		b.WriteString(" ")
	}
	b.WriteString(rb)
	return b.String()
}

func (s *BlockStmt) IsEmtpy() bool {
	l := len(s.Stmts)
	if l > 0 {
		if _, ok := s.Stmts[0].(*CodeEndStmt); ok {
			_ = s.Stmts[1].(*CodeBeginStmt)
			l -= 2
		}
	}
	return l == 0
}

func (s *BlockStmt) WriteCode(ctx *CodeWriteContext) {
	var (
		lb = s.LBrace.Value
		rb = s.RBrace.Value
	)

	if ctx.Transpile != nil {
		lb = "{"
		rb = "}"
	}

	// A template control-flow body (e.g. `{% for … begin %}TEXT{% end %}`)
	// carries its own `{%`/`%}`/text segments and significant whitespace, so it
	// must be rendered inline: write the `begin` opener, the segments verbatim,
	// then the `end` closer. A missing opener is made explicit as `begin`.
	if ctx.Transpile == nil && isMixedBlock(s.Stmts) {
		if lb == "" {
			lb = "begin"
		}
		ctx.WriteString(lb)
		ctx.WriteStmts(s.Stmts...)
		ctx.WriteString(" " + rb)
		return
	}

	if len(s.Stmts) == 0 {
		var sep string
		if len(rb) > 0 && rb != "}" && rb != ")" {
			sep += " "
		}
		ctx.WriteString(lb + sep + rb)
		return
	}

	ctx.WriteString(lb)

	if !ctx.HasPrefix() && len(lb) > 0 && lb != "{" {
		ctx.WriteString(" ")
	}

	ctx.Depth++
	if ctx.HasPrefix() {
		ctx.WriteSemi()
	}
	ctx.WriteStmts(s.Stmts...)
	ctx.Depth--

	if ctx.HasPrefix() {
		ctx.WriteSemi()
	} else if !ctx.HasPrefix() && len(rb) > 0 && rb != "}" && rb != ")" {
		ctx.WriteString(" ")
	}

	ctx.WriteString(rb)
}

// isMixedBlock reports whether stmts contain mixed/template segments (the
// `{%`/`%}` markers, literal text or `{%= … %}` values), i.e. a block whose body
// is template content rather than plain code.
func isMixedBlock(stmts []Stmt) bool {
	for _, s := range stmts {
		switch s.(type) {
		case *CodeBeginStmt, *CodeEndStmt, *MixedTextStmt, *MixedValueStmt:
			return true
		}
	}
	return false
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
	ctx.WriteString(s.String())
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
	if b, _ := s.Expr.(*BinaryExpr); b != nil {
		b.WriteCodeWithParen(ctx, false)
	} else {
		s.Expr.WriteCode(ctx)
	}
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
	var str = "for " + s.Key.Name
	if s.Value != nil {
		str += ", " + s.Value.String()
	}
	str += " in " + s.Iterable.String() +
		" " + s.Body.String()
	if s.Else != nil {
		if str[len(str)-1] != ' ' {
			str += " "
		}
		str += "else"
		els := s.Else.String()
		if els[0] != ' ' {
			str += " "
		}
		str += els
	}
	return str
}

func (s *ForInStmt) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString("for ")

	if !s.Key.Empty {
		ctx.WriteString(s.Key.Name)
		if s.Value != nil {
			ctx.WriteString(", ")
		}
	}

	if s.Value != nil {
		s.Value.WriteCode(ctx)
	}

	ctx.WriteString(" in ")
	s.Iterable.WriteCode(ctx)
	ctx.WriteString(" ")

	s.Body.WriteCode(ctx)

	if s.Else != nil {
		var space string
		if !s.Else.IsEmtpy() {
			space = " "
		}

		ctx.WriteString(" else" + space)
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

	s.Body.WriteCode(ctx)
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
		elseStmt = s.Else.String()
		if elseStmt[0] != ' ' {
			elseStmt = " " + elseStmt
		}
		elseStmt = "else" + elseStmt
	}
	return "if " + initStmt + s.Cond.String() + " " +
		s.Body.String() + elseStmt
}

func (s *IfStmt) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString("if ")
	if s.Init != nil {
		ctx.WithoutPrefix().WriteStmts(s.Init)
		ctx.WriteString("; ")
	}
	s.Cond.WriteCode(ctx)
	ctx.WriteSingleByte(' ')
	s.Body.WriteCode(ctx)
	if s.Else != nil {
		ctx.WriteString(" else ")
		if block, ok := s.Else.(*BlockStmt); ok {
			block.WriteCode(ctx)
		} else {
			ctx.WriteStmts(s.Else)
		}
	}
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
	s.Expr.WriteCode(ctx)
	ctx.WriteString(s.Token.String())
}

// ReturnStmt represents a return statement.
type ReturnStmt struct {
	Return
}

func (s *ReturnStmt) StmtNode() {}

func (s *ReturnStmt) WriteCode(ctx *CodeWriteContext) {
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

// DeferVariant selects when a deferred handler runs.
type DeferVariant int

const (
	DeferAlways DeferVariant = iota // defer
	DeferOnOk                       // defer_ok: only when no error
	DeferOnErr                      // defer_err: only when an error occurred
)

func (v DeferVariant) String() string {
	switch v {
	case DeferOnOk:
		return "defer_ok"
	case DeferOnErr:
		return "defer_err"
	default:
		return "defer"
	}
}

// DeferStmt represents a `defer`, `defer_ok` or `defer_err` statement. Exactly
// one of Body (`defer { ... }`) or Call (`defer handler` / `defer handler(x)`)
// is set. The handler runs when the enclosing function returns; inside it the
// locals `$ret` (return value) and `$err` (caught error) are available and may
// be modified.
type DeferStmt struct {
	DeferPos source.Pos
	Variant  DeferVariant
	Body     *BlockStmt
	Call     Expr
	// Block reports a `deferb*` statement, which runs at the end of the
	// enclosing block instead of the enclosing function.
	Block bool
}

func (s *DeferStmt) StmtNode() {}

// Keyword returns the source keyword for this statement (defer / defer_ok /
// deferb_err / ...).
func (s *DeferStmt) Keyword() string {
	kw := s.Variant.String()
	if s.Block {
		kw = "deferb" + kw[len("defer"):]
	}
	return kw
}

// Pos returns the position of first character belonging to the node.
func (s *DeferStmt) Pos() source.Pos { return s.DeferPos }

// End returns the position of first character immediately after the node.
func (s *DeferStmt) End() source.Pos {
	if s.Body != nil {
		return s.Body.End()
	}
	return s.Call.End()
}

func (s *DeferStmt) String() string {
	if s.Body != nil {
		return s.Keyword() + " " + s.Body.String()
	}
	return s.Keyword() + " " + s.Call.String()
}

func (s *DeferStmt) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(s.Keyword() + " ")
	if s.Body != nil {
		s.Body.WriteCode(ctx)
	} else {
		s.Call.WriteCode(ctx)
	}
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
	// RemoveLeftSpaces / RemoveRightSpaces are the single-dash markers (`-%}`
	// trims the following text's leading blanks; `{%-` trims the preceding
	// text's trailing blanks) and preserve a single boundary newline.
	RemoveLeftSpaces MixedTextStmtFlag = 1 << iota
	RemoveRightSpaces
	// RemoveLeftAll / RemoveRightAll are the double-dash markers (`--%}` / `{%--`)
	// and strip ALL boundary whitespace, newlines included.
	RemoveLeftAll
	RemoveRightAll
)

func (s MixedTextStmtFlag) Has(f MixedTextStmtFlag) bool {
	return s&f != 0
}

func (s MixedTextStmtFlag) String() string {
	var v []string
	for _, f := range []struct {
		flag MixedTextStmtFlag
		name string
	}{
		{RemoveLeftSpaces, "RemoveLeftSpaces"},
		{RemoveRightSpaces, "RemoveRightSpaces"},
		{RemoveLeftAll, "RemoveLeftAll"},
		{RemoveRightAll, "RemoveRightAll"},
	} {
		if s.Has(f.flag) {
			v = append(v, f.name)
		}
	}
	return strings.Join(v, "|")
}

// isMixedSpace reports whether c is ASCII whitespace handled by the template
// trim markers.
func isMixedSpace(c byte) bool {
	switch c {
	case ' ', '\t', '\n', '\r', '\v', '\f':
		return true
	}
	return false
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

// trimmed applies the template trim markers to the literal, returning the
// number of leading bytes removed (for position tracking) and the result.
// Single-dash markers preserve a single boundary newline; double-dash markers
// strip all boundary whitespace.
func (s *MixedTextStmt) trimmed() (start int, v string) {
	v = s.Lit.Value
	if s.Flags.Has(RemoveLeftSpaces) || s.Flags.Has(RemoveLeftAll) {
		i, nl := 0, false
		for i < len(v) && isMixedSpace(v[i]) {
			if v[i] == '\n' {
				nl = true
			}
			i++
		}
		start, v = i, v[i:]
		if nl && !s.Flags.Has(RemoveLeftAll) {
			v = "\n" + v
		}
	}
	if s.Flags.Has(RemoveRightSpaces) || s.Flags.Has(RemoveRightAll) {
		j, nl := len(v), false
		for j > 0 && isMixedSpace(v[j-1]) {
			if v[j-1] == '\n' {
				nl = true
			}
			j--
		}
		v = v[:j]
		if nl && !s.Flags.Has(RemoveRightAll) {
			v += "\n"
		}
	}
	return
}

func (s *MixedTextStmt) Value() string {
	_, v := s.trimmed()
	return v
}

func (s *MixedTextStmt) ValidLit() ast.Literal {
	start, v := s.trimmed()
	return ast.Literal{
		Value: v,
		Pos:   s.Lit.Pos + source.Pos(start),
	}
}

func (s *MixedTextStmt) WriteCode(ctx *CodeWriteContext) {
	if ctx.Transpile != nil {
		value := s.Value()
		if len(value) == 0 {
			return
		}
		ctx.WriteString(ctx.Transpile.WriteFunc)
		ctx.WriteSingleByte('(')
		ctx.WriteString(ctx.Transpile.RawStrFuncStart)
		ctx.WriteString(strconv.Quote(value))
		ctx.WriteString(ctx.Transpile.RawStrFuncEnd)
		ctx.WriteSingleByte(')')
	} else {
		ctx.WriteString(s.Lit.Value)
	}
}

// MixedValueStmt is an inline value-emitting tag in mixed/template mode, e.g.
// `{%= expr %}`, whose Expr is evaluated and written into the surrounding text.
// StartLit/EndLit hold the opening/closing delimiters (`{%`/`%}`); Eq reports
// the `=` value marker. RemoveLeftSpace/RemoveRightSpace mirror the `-` trim
// markers (`{%- … -%}`) that strip surrounding whitespace from the adjacent
// text at run time.
type MixedValueStmt struct {
	Expr             Expr
	StartLit         ast.Literal
	EndLit           ast.Literal
	RemoveLeftSpace  bool
	RemoveRightSpace bool
	// RemoveLeftAll/RemoveRightAll are the double-dash markers (`{%--= … --%}`)
	// that strip ALL adjacent whitespace (newlines included).
	RemoveLeftAll  bool
	RemoveRightAll bool
	Eq             bool
}

// leftMark / rightMark return the trim marker (“, `-` or `--`) for each side.
func (s *MixedValueStmt) leftMark() string {
	if s.RemoveLeftAll {
		return "--"
	}
	if s.RemoveLeftSpace {
		return "-"
	}
	return ""
}

func (s *MixedValueStmt) rightMark() string {
	if s.RemoveRightAll {
		return "--"
	}
	if s.RemoveRightSpace {
		return "-"
	}
	return ""
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
	b.WriteString(s.leftMark())
	if s.Eq {
		b.WriteByte('=')
	}
	b.WriteString(s.Expr.String())
	b.WriteString(s.rightMark())
	b.WriteString(s.EndLit.Value)
	return b.String()
}

func (s *MixedValueStmt) WriteCode(ctx *CodeWriteContext) {
	if ctx.Transpile != nil {
		ctx.WriteString(ctx.Transpile.WriteFunc)
		ctx.WriteSingleByte('(')
		s.Expr.WriteCode(ctx)
		ctx.WriteSingleByte(')')
	} else {
		// Normalize to `{%= expr %}` (with the trim markers `{%- … -%}` /
		// `{%-- … --%}` and a single space padding the expression).
		ctx.WriteString(s.StartLit.Value)
		if m := s.leftMark(); m != "" {
			ctx.WriteString(m + " ")
		}
		if s.Eq {
			ctx.WriteString("= ")
		} else {
			ctx.WriteSingleByte(' ')
		}
		s.Expr.WriteCode(ctx)
		if m := s.rightMark(); m != "" {
			ctx.WriteString(" " + m)
		} else {
			ctx.WriteSingleByte(' ')
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
	Elements  []*KeyValuePairLit
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
	return "# gad: " + strings.Join(elements, ", ") + "\n"
}

func (c *ConfigStmt) SkipCode(ctx *CodeWriteContext) bool {
	return ctx.Transpile != nil
}

func (c *ConfigStmt) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString("# gad: ")
	last := len(c.Elements) - 1
	for i, el := range c.Elements {
		ctx.WriteString(el.String())
		if i != last {
			ctx.WriteString(", ")
		}
	}
	// The directive owns its line; the following mixed text starts fresh.
	ctx.WriteString("\n")
}

// strLitValue returns the string value of a string or raw-string literal.
func strLitValue(e Expr) (string, bool) {
	switch s := e.(type) {
	case *StrLit:
		return s.Value(), true
	case *RawStrLit:
		return s.Value(), true
	}
	return "", false
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
		case "delimiter":
			// delimiter = [START, END]
			if arr, ok := k.Value.(*ArrayExpr); ok && len(arr.Elements) == 2 {
				if s, ok := strLitValue(arr.Elements[0]); ok {
					c.Options.MixedStart = s
				}
				if s, ok := strLitValue(arr.Elements[1]); ok {
					c.Options.MixedEnd = s
				}
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

// CodeBeginStmt is the opening delimiter of a mixed/template code block, e.g.
// `{%` (or `{%-`). Lit holds the literal delimiter; RemoveSpace reports the `-`
// trim suffix (`{%-`), which strips trailing whitespace from the preceding text
// at run time. The statements that follow, up to the matching CodeEndStmt, are
// ordinary Gad code.
type CodeBeginStmt struct {
	Lit         ast.Literal
	RemoveSpace bool
	// RemoveAllSpace reports the double-dash trim suffix (`{%--`), which strips
	// ALL trailing whitespace (newlines included) from the preceding text.
	RemoveAllSpace bool
}

func (s CodeBeginStmt) Pos() source.Pos {
	return s.Lit.Pos
}

func (s CodeBeginStmt) End() source.Pos {
	return s.Lit.End()
}

func (s CodeBeginStmt) String() string {
	if s.RemoveAllSpace {
		return s.Lit.Value + "--"
	}
	if s.RemoveSpace {
		return s.Lit.Value + "-"
	}
	return s.Lit.Value
}

func (s CodeBeginStmt) StmtNode() {
}

func (CodeBeginStmt) SkipCode(ctx *CodeWriteContext) bool {
	return ctx.Transpile != nil
}

func (s *CodeBeginStmt) WriteCode(ctx *CodeWriteContext) {
	if ctx.Transpile == nil {
		ctx.WriteString(s.String())
		// Indentation of the enclosed code block is managed by WriteStmts.
	}
}

// CodeEndStmt is the closing delimiter of a mixed/template code block, e.g.
// `%}` (or `-%}`). Lit holds the literal delimiter; RemoveSpace reports the `-`
// trim prefix (`-%}`), which strips leading whitespace from the following text
// at run time.
type CodeEndStmt struct {
	Lit         ast.Literal
	RemoveSpace bool
	// RemoveAllSpace reports the double-dash trim prefix (`--%}`), which strips
	// ALL leading whitespace (newlines included) from the following text.
	RemoveAllSpace bool
}

func (s CodeEndStmt) Pos() source.Pos {
	return s.Lit.Pos
}

func (s CodeEndStmt) End() source.Pos {
	return s.Lit.End()
}

func (s CodeEndStmt) String() string {
	if s.RemoveAllSpace {
		return "--" + s.Lit.Value
	}
	if s.RemoveSpace {
		return "-" + s.Lit.Value
	}
	return s.Lit.Value
}

func (s CodeEndStmt) StmtNode() {
}
func (CodeEndStmt) SkipCode(ctx *CodeWriteContext) bool {
	return ctx.Transpile != nil
}

func (s *CodeEndStmt) WriteCode(ctx *CodeWriteContext) {
	if ctx.Transpile == nil {
		ctx.WriteString(s.String())
		// De-indentation back to the block level is managed by WriteStmts.
	}
}

type FuncStmt struct {
	Func *FuncExpr
}

func (f FuncStmt) StmtNode() {
}

// Pos returns the position of first character belonging to the node.
func (f *FuncStmt) Pos() source.Pos {
	return f.Func.Pos()
}

// End returns the position of first character immediately after the node.
func (f *FuncStmt) End() source.Pos {
	return f.Func.End()
}

func (f *FuncStmt) String() string {
	return f.Func.String()
}

func (f *FuncStmt) WriteCode(ctx *CodeWriteContext) {
	f.Func.WriteCode(ctx)
}

type ExportStmt struct {
	TokenPos  source.Pos
	KeyExpr   Expr
	ValueExpr Expr
	Doc       *ast.CommentGroup // doc comment preceding the export; or nil
}

func (s *ExportStmt) End() source.Pos {
	if s.ValueExpr == nil {
		return s.KeyExpr.End()
	}
	return s.ValueExpr.End()
}

func (s *ExportStmt) StmtNode() {
}

func (s *ExportStmt) Pos() source.Pos {
	if s.TokenPos == 0 {
		if s.KeyExpr != nil {
			return s.ValueExpr.Pos()
		}
		return s.KeyExpr.Pos()
	}
	return s.TokenPos
}

func (s *ExportStmt) String() string {
	str := "export "
	if s.KeyExpr != nil {
		str += s.KeyExpr.String()
		if s.ValueExpr != nil {
			str += " = "
		}
	}
	if s.ValueExpr != nil {
		str += s.ValueExpr.String()
	}
	return str
}

func (s *ExportStmt) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString("export ")
	if s.KeyExpr != nil {
		s.KeyExpr.WriteCode(ctx)

		if s.ValueExpr != nil {
			ctx.WriteString(" = ")
		}
	}
	if s.ValueExpr != nil {
		s.ValueExpr.WriteCode(ctx)
	}
}
