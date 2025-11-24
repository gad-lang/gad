package node

import (
	"bytes"
	"strings"

	"github.com/gad-lang/gad/parser/source"
)

// ArgsList represents a list of identifiers.
type ArgsList struct {
	Var    *TypedIdentExpr
	Values []*TypedIdentExpr
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

func (n *ArgsList) PrependValue(v ...*TypedIdentExpr) {
	n.Values = append(v, n.Values...)
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
	Var    *TypedIdentExpr
	Names  []*TypedIdentExpr
	Values []Expr
}

func (n *NamedArgsList) Add(name *TypedIdentExpr, value Expr) *NamedArgsList {
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
	i := len(n.Names)
	if n.Var != nil {
		i++
	}
	return i
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
	if buf.Len() > 1 && n.NamedArgs.NumFields() > 0 {
		buf.WriteString(", ")
	}
	buf.WriteString(n.NamedArgs.String())
	buf.WriteString(")")
	return buf.String()
}

func (n *FuncParams) Caller() (c *CallArgs) {
	c = &CallArgs{}
	for _, value := range n.Args.Values {
		c.Args.Values = append(c.Args.Values, value.Ident)
	}
	if n.Args.Var != nil {
		c.Args.Var = &ArgVarLit{Value: n.Args.Var.Ident}
	}

	for i, name := range n.NamedArgs.Names {
		c.NamedArgs.Names = append(c.NamedArgs.Names, NamedArgExpr{Ident: name.Ident})
		c.NamedArgs.Values = append(c.NamedArgs.Values, n.NamedArgs.Values[i])
	}

	if n.NamedArgs.Var != nil {
		c.NamedArgs.Var = &NamedArgVarLit{Value: n.NamedArgs.Var.Ident}
	}
	return
}

func (n FuncParams) WithNamedValuesNil() (c *FuncParams) {
	c = &n

	c.NamedArgs.Values = append([]Expr{}, n.NamedArgs.Values...)
	for i := range c.NamedArgs.Values {
		c.NamedArgs.Values[i] = &NilLit{}
	}
	return
}

// FuncType represents a function type definition.
type FuncType struct {
	FuncPos      source.Pos
	Ident        *IdentExpr
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
