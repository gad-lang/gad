// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

// Copyright (c) 2019 Daniel Kang.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE.tengo file.

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.golang file.

package parser

import (
	"bytes"
	"strings"

	"github.com/gad-lang/gad/token"
	"github.com/shopspring/decimal"
)

// Expr represents an expression node in the AST.
type Expr interface {
	Node
	exprNode()
}

// ArrayLit represents an array literal.
type ArrayLit struct {
	Elements []Expr
	LBrack   Pos
	RBrack   Pos
}

func (e *ArrayLit) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *ArrayLit) Pos() Pos {
	return e.LBrack
}

// End returns the position of first character immediately after the node.
func (e *ArrayLit) End() Pos {
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
	From Pos
	To   Pos
}

func (e *BadExpr) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *BadExpr) Pos() Pos {
	return e.From
}

// End returns the position of first character immediately after the node.
func (e *BadExpr) End() Pos {
	return e.To
}

func (e *BadExpr) String() string {
	return "<bad expression>"
}

// BinaryExpr represents a binary operator expression.
type BinaryExpr struct {
	LHS      Expr
	RHS      Expr
	Token    token.Token
	TokenPos Pos
}

func (e *BinaryExpr) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *BinaryExpr) Pos() Pos {
	return e.LHS.Pos()
}

// End returns the position of first character immediately after the node.
func (e *BinaryExpr) End() Pos {
	return e.RHS.End()
}

func (e *BinaryExpr) String() string {
	return "(" + e.LHS.String() + " " + e.Token.String() +
		" " + e.RHS.String() + ")"
}

// BoolLit represents a boolean literal.
type BoolLit struct {
	Value    bool
	ValuePos Pos
	Literal  string
}

func (e *BoolLit) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *BoolLit) Pos() Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *BoolLit) End() Pos {
	return Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *BoolLit) String() string {
	return e.Literal
}

// CallExpr represents a function call expression.
type CallExpr struct {
	Func      Expr
	LParen    Pos
	Args      CallExprArgs
	NamedArgs CallExprNamedArgs
	RParen    Pos
}

func (e *CallExpr) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *CallExpr) Pos() Pos {
	return e.Func.Pos()
}

// End returns the position of first character immediately after the node.
func (e *CallExpr) End() Pos {
	return e.RParen + 1
}

func (e *CallExpr) String() string {
	var buf = bytes.NewBufferString(e.Func.String())
	buf.WriteString("(")
	if e.Args.Valid() {
		buf.WriteString(e.Args.String())
	}
	if e.NamedArgs.Valid() {
		buf.WriteString("; ")
		buf.WriteString(e.NamedArgs.String())
	}
	buf.WriteString(")")
	return buf.String()
}

// CharLit represents a character literal.
type CharLit struct {
	Value    rune
	ValuePos Pos
	Literal  string
}

func (e *CharLit) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *CharLit) Pos() Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *CharLit) End() Pos {
	return Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *CharLit) String() string {
	return e.Literal
}

// CondExpr represents a ternary conditional expression.
type CondExpr struct {
	Cond        Expr
	True        Expr
	False       Expr
	QuestionPos Pos
	ColonPos    Pos
}

func (e *CondExpr) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *CondExpr) Pos() Pos {
	return e.Cond.Pos()
}

// End returns the position of first character immediately after the node.
func (e *CondExpr) End() Pos {
	return e.False.End()
}

func (e *CondExpr) String() string {
	return "(" + e.Cond.String() + " ? " + e.True.String() +
		" : " + e.False.String() + ")"
}

// FloatLit represents a floating point literal.
type FloatLit struct {
	Value    float64
	ValuePos Pos
	Literal  string
}

func (e *FloatLit) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *FloatLit) Pos() Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *FloatLit) End() Pos {
	return Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *FloatLit) String() string {
	return e.Literal
}

// DecimalLit represents a floating point literal.
type DecimalLit struct {
	Value    decimal.Decimal
	ValuePos Pos
	Literal  string
}

func (e *DecimalLit) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *DecimalLit) Pos() Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *DecimalLit) End() Pos {
	return Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *DecimalLit) String() string {
	return e.Literal
}

// FuncLit represents a function literal.
type FuncLit struct {
	Type *FuncType
	Body *BlockStmt
}

func (e *FuncLit) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *FuncLit) Pos() Pos {
	return e.Type.Pos()
}

// End returns the position of first character immediately after the node.
func (e *FuncLit) End() Pos {
	return e.Body.End()
}

func (e *FuncLit) String() string {
	return "func" + e.Type.Params.String() + " " + e.Body.String()
}

// ArgsList represents a list of identifiers.
type ArgsList struct {
	Var    *Ident
	Values []*Ident
}

// Pos returns the position of first character belonging to the node.
func (n *ArgsList) Pos() Pos {
	if len(n.Values) > 0 {
		return n.Values[0].Pos()
	} else if n.Var != nil {
		return n.Var.Pos()
	}
	return NoPos
}

// End returns the position of first character immediately after the node.
func (n *ArgsList) End() Pos {
	if n.Var != nil {
		return n.Var.End()
	} else if l := len(n.Values); l > 0 {
		return n.Values[l-1].End()
	}
	return NoPos
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
		list = append(list, "..."+n.Var.String())
	}
	return strings.Join(list, ", ")
}

// NamedArgsList represents a list of identifier with value pairs.
type NamedArgsList struct {
	Var    *Ident
	Names  []*Ident
	Values []Expr
}

func (n *NamedArgsList) Add(name *Ident, value Expr) *NamedArgsList {
	n.Names = append(n.Names, name)
	n.Values = append(n.Values, value)
	return n
}

// Pos returns the position of first character belonging to the node.
func (n *NamedArgsList) Pos() Pos {
	if len(n.Names) > 0 {
		return n.Names[0].Pos()
	} else if n.Var != nil {
		return n.Var.Pos()
	}
	return NoPos
}

// End returns the position of first character immediately after the node.
func (n *NamedArgsList) End() Pos {
	if n.Var != nil {
		return n.Var.End()
	}
	if l := len(n.Names); l > 0 {
		if n.Var != nil {
			return n.Var.End()
		}
		return n.Values[l-1].End()
	}
	return NoPos
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
		list = append(list, "..."+n.Var.String())
	}
	return strings.Join(list, ", ")
}

// FuncParams represents a function paramsw.
type FuncParams struct {
	LParen    Pos
	Args      ArgsList
	NamedArgs NamedArgsList
	RParen    Pos
}

// Pos returns the position of first character belonging to the node.
func (n *FuncParams) Pos() (pos Pos) {
	if n.LParen.IsValid() {
		return n.LParen
	}
	if pos = n.Args.Pos(); pos != NoPos {
		return pos
	}
	if pos = n.NamedArgs.Pos(); pos != NoPos {
		return pos
	}
	return NoPos
}

// End returns the position of first character immediately after the node.
func (n *FuncParams) End() (pos Pos) {
	if n.RParen.IsValid() {
		return n.RParen + 1
	}
	if pos = n.NamedArgs.End(); pos != NoPos {
		return pos
	}
	if pos = n.Args.End(); pos != NoPos {
		return pos
	}
	return NoPos
}

func (n *FuncParams) String() string {
	buf := bytes.NewBufferString("(")
	buf.WriteString(n.Args.String())
	if buf.Len() > 1 && n.NamedArgs.Pos() != NoPos {
		buf.WriteString("; ")
	}
	buf.WriteString(n.NamedArgs.String())
	buf.WriteString(")")
	return buf.String()
}

// FuncType represents a function type definition.
type FuncType struct {
	FuncPos Pos
	Params  FuncParams
}

func (e *FuncType) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *FuncType) Pos() Pos {
	return e.FuncPos
}

// End returns the position of first character immediately after the node.
func (e *FuncType) End() Pos {
	return e.Params.End()
}

func (e *FuncType) String() string {
	return "func" + e.Params.String()
}

// Ident represents an identifier.
type Ident struct {
	Name    string
	NamePos Pos
}

func (e *Ident) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *Ident) Pos() Pos {
	return e.NamePos
}

// End returns the position of first character immediately after the node.
func (e *Ident) End() Pos {
	return Pos(int(e.NamePos) + len(e.Name))
}

func (e *Ident) String() string {
	if e != nil {
		return e.Name
	}
	return nullRep
}

// ImportExpr represents an import expression
type ImportExpr struct {
	ModuleName string
	Token      token.Token
	TokenPos   Pos
}

func (e *ImportExpr) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *ImportExpr) Pos() Pos {
	return e.TokenPos
}

// End returns the position of first character immediately after the node.
func (e *ImportExpr) End() Pos {
	// import("moduleName")
	return Pos(int(e.TokenPos) + 10 + len(e.ModuleName))
}

func (e *ImportExpr) String() string {
	return `import("` + e.ModuleName + `")"`
}

// IndexExpr represents an index expression.
type IndexExpr struct {
	Expr   Expr
	LBrack Pos
	Index  Expr
	RBrack Pos
}

func (e *IndexExpr) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *IndexExpr) Pos() Pos {
	return e.Expr.Pos()
}

// End returns the position of first character immediately after the node.
func (e *IndexExpr) End() Pos {
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
	ValuePos Pos
	Literal  string
}

func (e *IntLit) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *IntLit) Pos() Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *IntLit) End() Pos {
	return Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *IntLit) String() string {
	return e.Literal
}

// UintLit represents an unsigned integer literal.
type UintLit struct {
	Value    uint64
	ValuePos Pos
	Literal  string
}

func (e *UintLit) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *UintLit) Pos() Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *UintLit) End() Pos {
	return Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *UintLit) String() string {
	return e.Literal
}

// MapElementLit represents a map element.
type MapElementLit struct {
	Key      string
	KeyPos   Pos
	ColonPos Pos
	Value    Expr
}

func (e *MapElementLit) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *MapElementLit) Pos() Pos {
	return e.KeyPos
}

// End returns the position of first character immediately after the node.
func (e *MapElementLit) End() Pos {
	return e.Value.End()
}

func (e *MapElementLit) String() string {
	return e.Key + ": " + e.Value.String()
}

// MapLit represents a map literal.
type MapLit struct {
	LBrace   Pos
	Elements []*MapElementLit
	RBrace   Pos
}

func (e *MapLit) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *MapLit) Pos() Pos {
	return e.LBrace
}

// End returns the position of first character immediately after the node.
func (e *MapLit) End() Pos {
	return e.RBrace + 1
}

func (e *MapLit) String() string {
	var elements []string
	for _, m := range e.Elements {
		elements = append(elements, m.String())
	}
	return "{" + strings.Join(elements, ", ") + "}"
}

// ParenExpr represents a parenthesis wrapped expression.
type ParenExpr struct {
	Expr   Expr
	LParen Pos
	RParen Pos
}

func (e *ParenExpr) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *ParenExpr) Pos() Pos {
	return e.LParen
}

// End returns the position of first character immediately after the node.
func (e *ParenExpr) End() Pos {
	return e.RParen + 1
}

func (e *ParenExpr) String() string {
	return "(" + e.Expr.String() + ")"
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

func (e *SelectorExpr) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *SelectorExpr) Pos() Pos {
	return e.Expr.Pos()
}

// End returns the position of first character immediately after the node.
func (e *SelectorExpr) End() Pos {
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

func (e *NullishSelectorExpr) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *NullishSelectorExpr) Pos() Pos {
	return e.Expr.Pos()
}

// End returns the position of first character immediately after the node.
func (e *NullishSelectorExpr) End() Pos {
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
	LBrack Pos
	Low    Expr
	High   Expr
	RBrack Pos
}

func (e *SliceExpr) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *SliceExpr) Pos() Pos {
	return e.Expr.Pos()
}

// End returns the position of first character immediately after the node.
func (e *SliceExpr) End() Pos {
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
	ValuePos Pos
	Literal  string
}

func (e *StringLit) CanIdent() bool {
	for _, r := range e.Value {
		if !isLetter(r) {
			return false
		}
	}
	return true
}

func (e *StringLit) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *StringLit) Pos() Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *StringLit) End() Pos {
	return Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *StringLit) String() string {
	return e.Literal
}

// UnaryExpr represents an unary operator expression.
type UnaryExpr struct {
	Expr     Expr
	Token    token.Token
	TokenPos Pos
}

func (e *UnaryExpr) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *UnaryExpr) Pos() Pos {
	return e.Expr.Pos()
}

// End returns the position of first character immediately after the node.
func (e *UnaryExpr) End() Pos {
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
	TokenPos Pos
}

func (e *NilLit) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *NilLit) Pos() Pos {
	return e.TokenPos
}

// End returns the position of first character immediately after the node.
func (e *NilLit) End() Pos {
	return e.TokenPos + 9 // len(nil) == 9
}

func (e *NilLit) String() string {
	return "nil"
}

type EllipsisValue struct {
	Pos   Pos
	Value Expr
}

// CallExprArgs represents a call expression arguments.
type CallExprArgs struct {
	Values   []Expr
	Ellipsis *EllipsisValue
}

func (a *CallExprArgs) Valid() bool {
	return len(a.Values) > 0 || a.Ellipsis != nil
}

func (a *CallExprArgs) String() string {
	var s []string
	for _, v := range a.Values {
		s = append(s, v.String())
	}
	if a.Ellipsis != nil {
		s = append(s, "..."+a.Ellipsis.Value.String())
	}
	return strings.Join(s, ", ")
}

type NamedArgExpr struct {
	String *StringLit
	Ident  *Ident
}

func (e *NamedArgExpr) Name() string {
	if e.String != nil {
		return e.String.Value
	}
	return e.Ident.Name
}

func (e *NamedArgExpr) NameString() *StringLit {
	if e.String != nil {
		return e.String
	}
	return &StringLit{Value: e.Ident.Name, ValuePos: e.Ident.NamePos}
}

func (e *NamedArgExpr) Expr() Expr {
	if e.String != nil {
		return e.String
	}
	return e.Ident
}

// CallExprNamedArgs represents a call expression keyword arguments.
type CallExprNamedArgs struct {
	Names    []NamedArgExpr
	Values   []Expr
	Ellipsis *EllipsisValue
}

func (a *CallExprNamedArgs) Valid() bool {
	return len(a.Names) > 0 || a.Ellipsis != nil
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
			s = append(s, name.Expr().String())
		} else {
			s = append(s, name.Expr().String()+"="+a.Values[i].String())
		}
	}
	if a.Ellipsis != nil {
		s = append(s, "..."+a.Ellipsis.Value.String())
	}
	return strings.Join(s, ", ")
}

// KeyValueLit represents a key value element.
type KeyValueLit struct {
	Key      Expr
	KeyPos   Pos
	ColonPos Pos
	Value    Expr
}

func (e *KeyValueLit) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *KeyValueLit) Pos() Pos {
	return e.KeyPos
}

// End returns the position of first character immediately after the node.
func (e *KeyValueLit) End() Pos {
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
	LBrace   Pos
	Elements []*KeyValueLit
	RBrace   Pos
}

func (e *KeyValueArrayLit) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *KeyValueArrayLit) Pos() Pos {
	return e.LBrace
}

// End returns the position of first character immediately after the node.
func (e *KeyValueArrayLit) End() Pos {
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
	TokenPos Pos
	Literal  string
}

func (c *CalleeKeyword) Pos() Pos {
	return c.TokenPos
}

func (c *CalleeKeyword) End() Pos {
	return c.TokenPos + Pos(len(token.Callee.String()))
}

func (c *CalleeKeyword) String() string {
	return c.Literal
}

func (c *CalleeKeyword) exprNode() {
}

type ArgsKeyword struct {
	TokenPos Pos
	Literal  string
}

func (c *ArgsKeyword) Pos() Pos {
	return c.TokenPos
}

func (c *ArgsKeyword) End() Pos {
	return c.TokenPos + Pos(len(c.Literal))
}

func (c *ArgsKeyword) String() string {
	return c.Literal
}

func (c *ArgsKeyword) exprNode() {
}

type NamedArgsKeyword struct {
	TokenPos Pos
	Literal  string
}

func (c *NamedArgsKeyword) Pos() Pos {
	return c.TokenPos
}

func (c *NamedArgsKeyword) End() Pos {
	return c.TokenPos + Pos(len(c.Literal))
}

func (c *NamedArgsKeyword) String() string {
	return c.Literal
}

func (c *NamedArgsKeyword) exprNode() {
}
