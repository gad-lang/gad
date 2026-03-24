// Copyright 2009 The ToInterface Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.golang file.

// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package node

import (
	"fmt"
	"strings"

	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/repr"
	"github.com/gad-lang/gad/token"
)

// ----------------------------------------------------------------------------
// Declarations

type (
	// Spec node represents a single (non-parenthesized) variable declaration.
	// The Spec type stands for any of *ParamSpec or *ValueSpec.
	Spec interface {
		ast.Node
		Coder
		specNode()
	}
)

var _ Spec = (*ParamSpec)(nil)

// A ParamSpec node represents a parameter declaration
type ParamSpec struct {
	Ident *TypedIdentExpr
	Var   bool
}

// specNode() ensures that only spec nodes can be assigned to a Spec.
func (*ParamSpec) specNode() {}

// Pos returns the position of first character belonging to the spec.
func (s *ParamSpec) Pos() source.Pos { return s.Ident.Pos() }

// Pos returns the position of first character belonging to the spec.
func (s *ValueSpec) Pos() source.Pos { return s.Idents[0].Pos() }

// End returns the position of first character immediately after the spec.
func (s *ParamSpec) End() source.Pos {
	return s.Ident.End()
}

func (s *ParamSpec) String() string {
	str := s.Ident.String()
	if s.Var {
		str = "*" + str
	}
	return str
}

func (s *ParamSpec) WriteCode(ctx *CodeWriteContext) {
	if s.Var {
		ctx.WriteSingleByte('*')
	}
	s.Ident.WriteCode(ctx)
}

var _ Spec = (*NamedParamSpec)(nil)

// A NamedParamSpec node represents a named parameter declaration
type NamedParamSpec struct {
	Ident *TypedIdentExpr
	Value Expr
	Var   bool
}

// specNode() ensures that only spec nodes can be assigned to a Spec.
func (*NamedParamSpec) specNode() {}

// Pos returns the position of first character belonging to the spec.
func (s *NamedParamSpec) Pos() source.Pos { return s.Ident.Pos() }

// End returns the position of first character immediately after the spec.
func (s *NamedParamSpec) End() source.Pos {
	if s.Value == nil {
		return s.Ident.End()
	}
	return s.Value.End()
}

func (s *NamedParamSpec) String() string {
	str := s.Ident.String()
	if s.Var {
		return "**" + str
	}
	if s.Value == nil {
		return str
	}
	return str + "=" + s.Value.String()
}

func (s *NamedParamSpec) WriteCode(ctx *CodeWriteContext) {
	if s.Var {
		ctx.WriteString("**")
		s.Ident.WriteCode(ctx)
	} else if s.Value == nil {
		s.Ident.WriteCode(ctx)
	} else {
		s.Ident.WriteCode(ctx)
		ctx.WriteSingleByte('=')
		s.Value.WriteCode(ctx)
	}
}

var _ Spec = (*ValueSpec)(nil)

// A ValueSpec node represents a variable declaration
type ValueSpec struct {
	Idents []*IdentExpr // TODO: slice is reserved for tuple assignment
	Values []Expr       // initial values; or nil
	Data   any          // iota
}

// specNode() ensures that only spec nodes can be assigned to a Spec.
func (*ValueSpec) specNode() {}

// End returns the position of first character immediately after the spec.
func (s *ValueSpec) End() source.Pos {
	if n := len(s.Values); n > 0 && s.Values[n-1] != nil {
		return s.Values[n-1].End()
	}
	return s.Idents[len(s.Idents)-1].End()
}

func (s *ValueSpec) String() string {
	vals := make([]string, 0, len(s.Idents))
	for i := range s.Idents {
		if s.Values[i] != nil {
			vals = append(vals, fmt.Sprintf("%s = %v", s.Idents[i], s.Values[i]))
		} else {
			vals = append(vals, s.Idents[i].String())
		}
	}
	return strings.Join(vals, ", ")
}

func (s *ValueSpec) WriteCode(ctx *CodeWriteContext) {
	last := len(s.Idents) - 1
	for i := range s.Idents {
		s.Idents[i].WriteCode(ctx)
		if s.Values[i] != nil {
			ctx.WriteString(" = ")
			s.Values[i].WriteCode(ctx)
		}
		if i != last {
			ctx.WriteString(", ")
		}
	}
}

// Decl wraps methods for all declaration nodes.
type Decl interface {
	ast.Node
	declNode()
	Coder
}

var _ Decl = (*DeclStmt)(nil)

// A DeclStmt node represents a declaration in a statement list.
type DeclStmt struct {
	Decl // *GenDecl with VAR token
}

func (*DeclStmt) StmtNode() {}

var _ Decl = (*BadDecl)(nil)

// A BadDecl node is a placeholder for declarations containing
// syntax errors for which no correct declaration nodes can be
// created.
type BadDecl struct {
	From, To source.Pos // position range of bad declaration
}

// Pos returns the position of first character belonging to the node.
func (d *BadDecl) Pos() source.Pos { return d.From }

// End returns the position of first character immediately after the node.
func (d *BadDecl) End() source.Pos { return d.To }
func (*BadDecl) declNode()         {}
func (*BadDecl) String() string    { return repr.Quote("bad declaration") }

func (d *BadDecl) WriteCode(ctx *CodeWriteContext) {
	ctx.Printf("`bad decl from %v to %v`", d.From, d.To)
}

var _ Decl = (*GenDecl)(nil)

// A GenDecl node (generic declaration node) represents a variable declaration.
// A valid Lparen position (Lparen.Line > 0) indicates a parenthesized declaration.
//
// Relationship between Tok value and Specs element type:
//
//	token.Var     *ValueSpec
type GenDecl struct {
	TokPos source.Pos  // position of Tok
	Tok    token.Token // Var
	Lparen source.Pos  // position of '(', if any
	Specs  []Spec
	Rparen source.Pos // position of ')', if any
}

// Pos returns the position of first character belonging to the node.
func (d *GenDecl) Pos() source.Pos { return d.TokPos }

// End returns the position of first character immediately after the node.
func (d *GenDecl) End() source.Pos {
	if d.Rparen.IsValid() {
		return d.Rparen + 1
	}
	return d.Specs[0].End()
}

func (*GenDecl) declNode() {}

func (d *GenDecl) String() string {
	var (
		sb                strings.Builder
		paren             = d.Lparen > 0
		namedParamPrinted bool
	)

	if !paren && len(d.Specs) > 0 {
		if _, ok := d.Specs[0].(*NamedParamSpec); ok {
			paren = true
		}
	}

	sb.WriteString(d.Tok.String())

	if paren {
		sb.WriteString(" (")
	} else {
		sb.WriteString(" ")
	}

	last := len(d.Specs) - 1
	for i := range d.Specs {
		if _, ok := d.Specs[i].(*NamedParamSpec); ok {
			if !namedParamPrinted {
				sb.WriteString("; ")
				namedParamPrinted = true
			}
		}

		sb.WriteString(d.Specs[i].String())
		if i != last {
			if _, ok := d.Specs[i].(*ParamSpec); ok {
				if _, ok := d.Specs[i+1].(*NamedParamSpec); ok {
					continue
				}
			}
			sb.WriteString(", ")
		}
	}

	if paren {
		sb.WriteString(")")
	}
	return sb.String()
}

func (d *GenDecl) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(d.Tok.String())

	if len(d.Specs) > 1 {
		ctx.WriteString(" (")
		inLineLine := ctx.Flags.Has(CodeWriteContextFlagFormatDeclItemInNewLine)
		ctx.WriteItemsSep(
			inLineLine,
			len(d.Specs),
			", ",
			"",
			func(i int) {
				d.Specs[i].WriteCode(ctx)
			},
			func(newLine bool) {
				if newLine {
					ctx.WriteSecondLine()
				}
			})
		ctx.WritePrefix()
		ctx.WriteSingleByte(')')
	} else {
		ctx.WriteSingleByte(' ')
		d.Specs[0].WriteCode(ctx)
	}
}

func (d *GenDecl) Params() (positional []*ParamSpec, named []*NamedParamSpec) {
	if d.Tok != token.Param {
		return
	}

	for _, sp := range d.Specs {
		switch s := sp.(type) {
		case *NamedParamSpec:
			named = append(named, s)
		case *ParamSpec:
			positional = append(positional, s)
		}
	}

	return
}
