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
	"io"
	"strconv"
	"strings"

	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/parser/utils"
	"github.com/gad-lang/gad/repr"
	"github.com/gad-lang/gad/token"
	"github.com/shopspring/decimal"
)

// Expr represents an expression node in the AST.
type Expr interface {
	ast.Node
	ExprNode()
}

// ArrayLit represents an array literal.
type ArrayLit struct {
	Elements []Expr
	LBrack   source.Pos
	RBrack   source.Pos
}

func (e *ArrayLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *ArrayLit) Pos() source.Pos {
	return e.LBrack
}

// End returns the position of first character immediately after the node.
func (e *ArrayLit) End() source.Pos {
	return e.RBrack + 1
}

func (e *ArrayLit) String() string {
	var elements []string
	for _, m := range e.Elements {
		elements = append(elements, m.String())
	}
	return "[" + strings.Join(elements, ", ") + "]"
}

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

// BoolLit represents a boolean literal.
type BoolLit struct {
	Value    bool
	ValuePos source.Pos
	Literal  string
}

func (e *BoolLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *BoolLit) Pos() source.Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *BoolLit) End() source.Pos {
	return source.Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *BoolLit) String() string {
	return e.Literal
}

type CallArgs struct {
	LParen    source.Pos
	Args      CallExprArgs
	NamedArgs CallExprNamedArgs
	RParen    source.Pos
}

// Pos returns the position of first character belonging to the node.
func (c *CallArgs) Pos() source.Pos {
	return c.LParen
}

// End returns the position of first character immediately after the node.
func (c *CallArgs) End() source.Pos {
	return c.RParen + 1
}

func (c *CallArgs) String() string {
	var buf strings.Builder
	c.StringW(&buf)
	return buf.String()
}

func (c *CallArgs) StringW(w io.Writer) {
	c.StringArg(w, "(", ")")
}

func (c *CallArgs) StringArg(w io.Writer, lbrace, rbrace string) {
	io.WriteString(w, lbrace)
	if c.Args.Valid() {
		io.WriteString(w, c.Args.String())
	}
	if c.NamedArgs.Valid() {
		if c.Args.Valid() {
			io.WriteString(w, ", ")
		}
		io.WriteString(w, c.NamedArgs.String())
	}
	io.WriteString(w, rbrace)
}

// CallExpr represents a function call expression.
type CallExpr struct {
	Func Expr
	CallArgs
}

func (e *CallExpr) ExprNode() {}

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

// CharLit represents a character literal.
type CharLit struct {
	Value    rune
	ValuePos source.Pos
	Literal  string
}

func (e *CharLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *CharLit) Pos() source.Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *CharLit) End() source.Pos {
	return source.Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *CharLit) String() string {
	return e.Literal
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

// FloatLit represents a floating point literal.
type FloatLit struct {
	Value    float64
	ValuePos source.Pos
	Literal  string
}

func (e *FloatLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *FloatLit) Pos() source.Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *FloatLit) End() source.Pos {
	return source.Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *FloatLit) String() string {
	return e.Literal
}

// DecimalLit represents a floating point literal.
type DecimalLit struct {
	Value    decimal.Decimal
	ValuePos source.Pos
	Literal  string
}

func (e *DecimalLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *DecimalLit) Pos() source.Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *DecimalLit) End() source.Pos {
	return source.Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *DecimalLit) String() string {
	return e.Literal
}

// FuncLit represents a function literal.
type FuncLit struct {
	ast.NodeData
	Type *FuncType
	Body *BlockStmt
}

func (e *FuncLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *FuncLit) Pos() source.Pos {
	return e.Type.Pos()
}

// End returns the position of first character immediately after the node.
func (e *FuncLit) End() source.Pos {
	return e.Body.End()
}

func (e *FuncLit) String() string {
	return "func" + e.Type.String() + " " + e.Body.String()
}

// ClosureLit represents a function closure literal.
type ClosureLit struct {
	ast.NodeData
	Type *FuncType
	Body Expr
}

func (e *ClosureLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *ClosureLit) Pos() source.Pos {
	return e.Type.Pos()
}

// End returns the position of first character immediately after the node.
func (e *ClosureLit) End() source.Pos {
	return e.Body.End()
}

func (e *ClosureLit) String() string {
	return e.Type.Params.String() + " => " + e.Body.String()
}

// ArgsList represents a list of identifiers.
type ArgsList struct {
	Var    *TypedIdent
	Values []*TypedIdent
}

// Pos returns the position of first character belonging to the node.
func (n *ArgsList) Pos() source.Pos {
	if len(n.Values) > 0 {
		return n.Values[0].Pos()
	} else if n.Var != nil {
		return n.Var.Pos()
	}
	return source.NoPos
}

// End returns the position of first character immediately after the node.
func (n *ArgsList) End() source.Pos {
	if n.Var != nil {
		return n.Var.End()
	} else if l := len(n.Values); l > 0 {
		return n.Values[l-1].End()
	}
	return source.NoPos
}

// NumFields returns the number of fields.
func (n *ArgsList) NumFields() int {
	if n == nil {
		return 0
	}
	return len(n.Values)
}

func (n *ArgsList) String() string {
	var list []string
	for _, e := range n.Values {
		list = append(list, e.String())
	}
	if n.Var != nil {
		list = append(list, "*"+n.Var.String())
	}
	return strings.Join(list, ", ")
}

// NamedArgsList represents a list of identifier with value pairs.
type NamedArgsList struct {
	Var    *TypedIdent
	Names  []*TypedIdent
	Values []Expr
}

func (n *NamedArgsList) Add(name *TypedIdent, value Expr) *NamedArgsList {
	n.Names = append(n.Names, name)
	n.Values = append(n.Values, value)
	return n
}

// Pos returns the position of first character belonging to the node.
func (n *NamedArgsList) Pos() source.Pos {
	if len(n.Names) > 0 {
		return n.Names[0].Pos()
	} else if n.Var != nil {
		return n.Var.Pos()
	}
	return source.NoPos
}

// End returns the position of first character immediately after the node.
func (n *NamedArgsList) End() source.Pos {
	if n.Var != nil {
		return n.Var.End()
	}
	if l := len(n.Names); l > 0 {
		if n.Var != nil {
			return n.Var.End()
		}
		return n.Values[l-1].End()
	}
	return source.NoPos
}

// NumFields returns the number of fields.
func (n *NamedArgsList) NumFields() int {
	if n == nil {
		return 0
	}
	return len(n.Names)
}

func (n *NamedArgsList) String() string {
	var list []string
	for i, e := range n.Names {
		list = append(list, e.String()+"="+n.Values[i].String())
	}
	if n.Var != nil {
		list = append(list, "**"+n.Var.String())
	}
	return strings.Join(list, ", ")
}

// FuncParams represents a function paramsw.
type FuncParams struct {
	LParen    source.Pos
	Args      ArgsList
	NamedArgs NamedArgsList
	RParen    source.Pos
}

// Pos returns the position of first character belonging to the node.
func (n *FuncParams) Pos() (pos source.Pos) {
	if n.LParen.IsValid() {
		return n.LParen
	}
	if pos = n.Args.Pos(); pos != source.NoPos {
		return pos
	}
	if pos = n.NamedArgs.Pos(); pos != source.NoPos {
		return pos
	}
	return source.NoPos
}

// End returns the position of first character immediately after the node.
func (n *FuncParams) End() (pos source.Pos) {
	if n.RParen.IsValid() {
		return n.RParen + 1
	}
	if pos = n.NamedArgs.End(); pos != source.NoPos {
		return pos
	}
	if pos = n.Args.End(); pos != source.NoPos {
		return pos
	}
	return source.NoPos
}

func (n *FuncParams) String() string {
	buf := bytes.NewBufferString("(")
	buf.WriteString(n.Args.String())
	if buf.Len() > 1 && n.NamedArgs.Pos() != source.NoPos {
		buf.WriteString(", ")
	}
	buf.WriteString(n.NamedArgs.String())
	buf.WriteString(")")
	return buf.String()
}

// FuncType represents a function type definition.
type FuncType struct {
	Token        token.Token
	FuncPos      source.Pos
	Ident        *Ident
	Params       FuncParams
	AllowMethods bool
}

func (e *FuncType) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *FuncType) Pos() source.Pos {
	return e.FuncPos
}

// End returns the position of first character immediately after the node.
func (e *FuncType) End() source.Pos {
	return e.Params.End()
}

func (e *FuncType) String() string {
	var s string
	if e.Ident != nil {
		s += " "
		s += e.Ident.String()
	}
	return s + e.Params.String()
}

// Ident represents an identifier.
type Ident struct {
	Name    string
	NamePos source.Pos
}

func (e *Ident) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *Ident) Pos() source.Pos {
	return e.NamePos
}

// End returns the position of first character immediately after the node.
func (e *Ident) End() source.Pos {
	return source.Pos(int(e.NamePos) + len(e.Name))
}

func (e *Ident) String() string {
	if e != nil {
		return e.Name
	}
	return nullRep
}

type TypedIdent struct {
	Ident *Ident
	Type  []*Ident
}

func (e *TypedIdent) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *TypedIdent) Pos() source.Pos {
	if e.Ident != nil {
		return e.Ident.Pos()
	}
	return e.Ident.Pos()
}

// End returns the position of first character immediately after the node.
func (e *TypedIdent) End() source.Pos {
	if len(e.Type) == 0 {
		return e.Ident.End()
	}
	return e.Type[len(e.Type)-1].End()
}

func (e *TypedIdent) String() string {
	if e != nil {
		l := len(e.Type)
		switch l {
		case 0:
			return e.Ident.String()
		case 1:
			return e.Ident.String() + ":" + e.Type[0].String()
		default:
			var s = make([]string, len(e.Type))
			for i, ident := range e.Type {
				s[i] = ident.String()
			}
			return e.Ident.String() + ":[" + strings.Join(s, ", ") + "]"
		}
	}
	return nullRep
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
	return `import("` + e.ModuleName + `")"`
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

// IntLit represents an integer literal.
type IntLit struct {
	Value    int64
	ValuePos source.Pos
	Literal  string
}

func (e *IntLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *IntLit) Pos() source.Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *IntLit) End() source.Pos {
	return source.Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *IntLit) String() string {
	return e.Literal
}

// UintLit represents an unsigned integer literal.
type UintLit struct {
	Value    uint64
	ValuePos source.Pos
	Literal  string
}

func (e *UintLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *UintLit) Pos() source.Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *UintLit) End() source.Pos {
	return source.Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *UintLit) String() string {
	return e.Literal
}

// DictElementLit represents a map element.
type DictElementLit struct {
	Key      string
	KeyPos   source.Pos
	ColonPos source.Pos
	Value    Expr
}

func (e *DictElementLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *DictElementLit) Pos() source.Pos {
	return e.KeyPos
}

// End returns the position of first character immediately after the node.
func (e *DictElementLit) End() source.Pos {
	return e.Value.End()
}

func (e *DictElementLit) String() string {
	return e.Key + ": " + e.Value.String()
}

// DictLit represents a map literal.
type DictLit struct {
	LBrace   source.Pos
	Elements []*DictElementLit
	RBrace   source.Pos
}

func (e *DictLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *DictLit) Pos() source.Pos {
	return e.LBrace
}

// End returns the position of first character immediately after the node.
func (e *DictLit) End() source.Pos {
	return e.RBrace + 1
}

func (e *DictLit) String() string {
	var elements []string
	for _, m := range e.Elements {
		elements = append(elements, m.String())
	}
	return "{" + strings.Join(elements, ", ") + "}"
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
	return "(" + e.Expr.String() + ")"
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
	var s = make([]string, len(e.Exprs))
	for i, expr := range e.Exprs {
		s[i] = expr.String()
	}
	return "(" + strings.Join(s, ", ") + ")"
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
			return r + s.Value
		}
		return r + "(" + s.Literal + ")"
	}
	return r + e.Sel.String()
}

func (e *SelectorExpr) SelectorExpr() Expr {
	return e.Expr
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
			return r + s.Value
		}
		return r + "(" + s.Literal + ")"
	}
	return r + e.Sel.String()
}

func (e *NullishSelectorExpr) SelectorExpr() Expr {
	return e.Expr
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

// StringLit represents a string literal.
type StringLit struct {
	Value    string
	ValuePos source.Pos
	Literal  string
}

func (e *StringLit) CanIdent() bool {
	var skip int
	if e.Value != "" && e.Value[0] == '!' {
		skip++
	}
	for _, r := range e.Value[skip:] {
		if !utils.IsLetter(r) {
			return false
		}
	}
	return true
}

func (e *StringLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *StringLit) Pos() source.Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *StringLit) End() source.Pos {
	return source.Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *StringLit) String() string {
	return e.Literal
}

type RawStringLit struct {
	Literal    string
	LiteralPos source.Pos
	Quoted     bool
}

func (e *RawStringLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *RawStringLit) Pos() source.Pos {
	return e.LiteralPos
}

// End returns the position of first character immediately after the node.
func (e *RawStringLit) End() source.Pos {
	return source.Pos(int(e.LiteralPos) + len(e.Literal))
}

func (e *RawStringLit) String() string {
	return e.QuotedValue()
}

func (e *RawStringLit) UnquotedValue() string {
	if e.Quoted {
		s, _ := strconv.Unquote(e.Literal)
		return s
	} else {
		return e.Literal
	}
}

func (e *RawStringLit) QuotedValue() string {
	if e.Quoted {
		return e.Literal
	} else {
		return utils.Quote(e.Literal, '`')
	}
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

// NilLit represents an nil literal.
type NilLit struct {
	TokenPos source.Pos
}

func (e *NilLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *NilLit) Pos() source.Pos {
	return e.TokenPos
}

// End returns the position of first character immediately after the node.
func (e *NilLit) End() source.Pos {
	return e.TokenPos + 9 // len(nil) == 9
}

func (e *NilLit) String() string {
	return "nil"
}

type EllipsisValue struct {
	Pos   source.Pos
	Value Expr
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

type NamedArgExpr struct {
	Lit   *StringLit
	Ident *Ident
}

func (e *NamedArgExpr) Name() string {
	if e.Lit != nil {
		return e.Lit.Value
	}
	return e.Ident.Name
}

func (e *NamedArgExpr) NameString() *StringLit {
	if e.Lit != nil {
		return e.Lit
	}
	return &StringLit{Value: e.Ident.Name, ValuePos: e.Ident.NamePos}
}

func (e *NamedArgExpr) String() string {
	return e.Expr().String()
}

func (e *NamedArgExpr) Expr() Expr {
	if e.Lit != nil {
		return e.Lit
	}
	return e.Ident
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
	a.Names = append(a.Names, NamedArgExpr{Ident: &Ident{Name: name}})
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
	for i, name := range a.Names {
		if a.Values[i] == nil {
			s = append(s, name.Expr().String()+"=true")
		} else {
			s = append(s, name.Expr().String()+"="+a.Values[i].String())
		}
	}
	if a.Var != nil {
		s = append(s, a.Var.String())
	}
	return strings.Join(s, ", ")
}

// KeyValueLit represents a key value element.
type KeyValueLit struct {
	Key   Expr
	Value Expr
}

func (e *KeyValueLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *KeyValueLit) Pos() source.Pos {
	return e.Key.Pos()
}

// End returns the position of first character immediately after the node.
func (e *KeyValueLit) End() source.Pos {
	if e.Value == nil {
		return e.Key.End()
	}
	return e.Value.End()
}

func (e *KeyValueLit) String() string {
	if e.Value == nil {
		return e.Key.String()
	}
	return e.Key.String() + "=" + e.Value.String()
}

// KeyValueArrayLit represents a key value array literal.
type KeyValueArrayLit struct {
	LBrace   source.Pos
	Elements []*KeyValueLit
	RBrace   source.Pos
}

func (e *KeyValueArrayLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *KeyValueArrayLit) Pos() source.Pos {
	return e.LBrace
}

// End returns the position of first character immediately after the node.
func (e *KeyValueArrayLit) End() source.Pos {
	return e.RBrace + 1
}

func (e *KeyValueArrayLit) String() string {
	var elements []string
	for _, m := range e.Elements {
		elements = append(elements, m.String())
	}
	return "(;" + strings.Join(elements, ", ") + ")"
}

type CalleeKeyword struct {
	TokenPos source.Pos
	Literal  string
}

func (c *CalleeKeyword) Pos() source.Pos {
	return c.TokenPos
}

func (c *CalleeKeyword) End() source.Pos {
	return c.TokenPos + source.Pos(len(token.Callee.String()))
}

func (c *CalleeKeyword) String() string {
	return c.Literal
}

func (c *CalleeKeyword) ExprNode() {
}

type ArgsKeyword struct {
	TokenPos source.Pos
	Literal  string
}

func (c *ArgsKeyword) Pos() source.Pos {
	return c.TokenPos
}

func (c *ArgsKeyword) End() source.Pos {
	return c.TokenPos + source.Pos(len(c.Literal))
}

func (c *ArgsKeyword) String() string {
	return c.Literal
}

func (c *ArgsKeyword) ExprNode() {
}

type NamedArgsKeyword struct {
	TokenPos source.Pos
	Literal  string
}

func (c *NamedArgsKeyword) Pos() source.Pos {
	return c.TokenPos
}

func (c *NamedArgsKeyword) End() source.Pos {
	return c.TokenPos + source.Pos(len(c.Literal))
}

func (c *NamedArgsKeyword) String() string {
	return c.Literal
}

func (c *NamedArgsKeyword) ExprNode() {
}

type BlockExpr struct {
	*BlockStmt
}

func (b BlockExpr) ExprNode() {}

// StdInLit represents an STDIN literal.
type StdInLit struct {
	TokenPos source.Pos
}

func (e *StdInLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *StdInLit) Pos() source.Pos {
	return e.TokenPos
}

// End returns the position of first character immediately after the node.
func (e *StdInLit) End() source.Pos {
	return e.TokenPos + 5 // len(STDIN) == 5
}

func (e *StdInLit) String() string {
	return "STDIN"
}

// StdOutLit represents an STDOUT literal.
type StdOutLit struct {
	TokenPos source.Pos
}

func (e *StdOutLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *StdOutLit) Pos() source.Pos {
	return e.TokenPos
}

// End returns the position of first character immediately after the node.
func (e *StdOutLit) End() source.Pos {
	return e.TokenPos + 6 // len(STDOUT) == 6
}

func (e *StdOutLit) String() string {
	return "STDOUT"
}

// StdErrLit represents an STDERR literal.
type StdErrLit struct {
	TokenPos source.Pos
}

func (e *StdErrLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *StdErrLit) Pos() source.Pos {
	return e.TokenPos
}

// End StdErrLit the position of first character immediately after the node.
func (e *StdErrLit) End() source.Pos {
	return e.TokenPos + 6 // len(STDERR) == 6
}

func (e *StdErrLit) String() string {
	return "STDERR"
}

// ArgVarLit represents an variadic of argument.
type ArgVarLit struct {
	TokenPos source.Pos
	Value    Expr
}

func (e *ArgVarLit) ExprNode() {}

func (e *ArgVarLit) Pos() source.Pos {
	return e.TokenPos
}

func (e *ArgVarLit) End() source.Pos {
	return e.Value.End() + 6
}

func (e *ArgVarLit) String() string {
	return "*" + e.Value.String()
}

// NamedArgVarLit represents an variadic of named argument.
type NamedArgVarLit struct {
	TokenPos source.Pos
	Value    Expr
}

func (e *NamedArgVarLit) ExprNode() {}

func (e *NamedArgVarLit) Pos() source.Pos {
	return e.TokenPos
}

func (e *NamedArgVarLit) End() source.Pos {
	return e.Value.End() + 6
}

func (e *NamedArgVarLit) String() string {
	return "**" + e.Value.String()
}
