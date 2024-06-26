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
		specNode()
	}

	// A ValueSpec node represents a variable declaration
	ValueSpec struct {
		Idents []*Ident // TODO: slice is reserved for tuple assignment
		Values []Expr   // initial values; or nil
		Data   any      // iota
	}

	// A ParamSpec node represents a parameter declaration
	ParamSpec struct {
		Ident    *TypedIdent
		Variadic bool
	}

	// A NamedParamSpec node represents a named parameter declaration
	NamedParamSpec struct {
		Ident *TypedIdent
		Value Expr
	}
)

// Pos returns the position of first character belonging to the spec.
func (s *ParamSpec) Pos() source.Pos { return s.Ident.Pos() }

// Pos returns the position of first character belonging to the spec.
func (s *NamedParamSpec) Pos() source.Pos { return s.Ident.Pos() }

// Pos returns the position of first character belonging to the spec.
func (s *ValueSpec) Pos() source.Pos { return s.Idents[0].Pos() }

// End returns the position of first character immediately after the spec.
func (s *ParamSpec) End() source.Pos {
	return s.Ident.End()
}

// End returns the position of first character immediately after the spec.
func (s *NamedParamSpec) End() source.Pos {
	if s.Value == nil {
		return s.Ident.End()
	}
	return s.Value.End()
}

// End returns the position of first character immediately after the spec.
func (s *ValueSpec) End() source.Pos {
	if n := len(s.Values); n > 0 && s.Values[n-1] != nil {
		return s.Values[n-1].End()
	}
	return s.Idents[len(s.Idents)-1].End()
}

func (s *ParamSpec) String() string {
	str := s.Ident.String()
	if s.Variadic {
		str = "*" + str
	}
	return str
}

func (s *ParamSpec) WriteCode(ctx *CodeWriterContext) (err error) {
	if s.Variadic {
		if err = ctx.WriteByte('*'); err != nil {
			return
		}
	}
	return WriteCode(ctx, s.Ident)
}

func (s *NamedParamSpec) String() string {
	str := s.Ident.String()
	if s.Value == nil {
		return "**" + str
	}
	return str + "=" + s.Value.String()
}

func (s *NamedParamSpec) WriteCode(ctx *CodeWriterContext) (err error) {
	if s.Value == nil {
		if _, err = ctx.WriteString("**"); err != nil {
			return
		}
		return WriteCode(ctx, s.Ident)
	}
	if err = WriteCode(ctx, s.Ident); err != nil {
		return
	}
	if err = ctx.WriteByte('='); err != nil {
		return
	}
	return WriteCode(ctx, s.Value)
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

func (s *ValueSpec) WriteCode(ctx *CodeWriterContext) (err error) {
	last := len(s.Idents) - 1
	for i := range s.Idents {
		if err = WriteCode(ctx, s.Idents[i]); err != nil {
			return
		}
		if s.Values[i] != nil {
			if _, err = ctx.WriteString(" ="); err != nil {
				return
			}
			if err = WriteCode(ctx, s.Values[i]); err != nil {
				return
			}
		}
		if i != last {
			if _, err = ctx.WriteString(", "); err != nil {
				return
			}
		}
	}
	return
}

// specNode() ensures that only spec nodes can be assigned to a Spec.
func (*ParamSpec) specNode() {}

// specNode() ensures that only spec nodes can be assigned to a Spec.
func (*NamedParamSpec) specNode() {}

// specNode() ensures that only spec nodes can be assigned to a Spec.
func (*ValueSpec) specNode() {}

// Decl wraps methods for all declaration nodes.
type Decl interface {
	ast.Node
	declNode()
}

// A DeclStmt node represents a declaration in a statement list.
type DeclStmt struct {
	Decl // *GenDecl with VAR token
}

func (*DeclStmt) StmtNode() {}

// A BadDecl node is a placeholder for declarations containing
// syntax errors for which no correct declaration nodes can be
// created.
type BadDecl struct {
	From, To source.Pos // position range of bad declaration
}

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
func (d *BadDecl) Pos() source.Pos { return d.From }

// Pos returns the position of first character belonging to the node.
func (d *GenDecl) Pos() source.Pos { return d.TokPos }

// End returns the position of first character immediately after the node.
func (d *BadDecl) End() source.Pos { return d.To }

// End returns the position of first character immediately after the node.
func (d *GenDecl) End() source.Pos {
	if d.Rparen.IsValid() {
		return d.Rparen + 1
	}
	return d.Specs[0].End()
}

func (*BadDecl) declNode() {}
func (*GenDecl) declNode() {}

func (*BadDecl) String() string { return repr.Quote("bad declaration") }
func (d *GenDecl) String() string {
	var sb strings.Builder
	sb.WriteString(d.Tok.String())
	if d.Lparen > 0 {
		sb.WriteString(" (")
	} else {
		sb.WriteString(" ")
	}
	last := len(d.Specs) - 1
	for i := range d.Specs {
		sb.WriteString(d.Specs[i].String())
		if i != last {
			if _, ok := d.Specs[i].(*ParamSpec); ok {
				if _, ok := d.Specs[i+1].(*NamedParamSpec); ok {
					sb.WriteString(", ")
					continue
				}
			}
			sb.WriteString(", ")
		}
	}
	if d.Rparen > 0 {
		sb.WriteString(")")
	}
	return sb.String()
}

func (d *GenDecl) WriteCode(ctx *CodeWriterContext) (err error) {
	if _, err = ctx.WriteString(d.Tok.String()); err != nil {
		return
	}
	if d.Lparen > 0 {
		if _, err = ctx.WriteString(" ("); err != nil {
			return
		}
	} else if err = ctx.WriteByte(' '); err != nil {
		return
	}
	last := len(d.Specs) - 1
	for i, spec := range d.Specs {
		if err = WriteCode(ctx, spec); err != nil {
			return
		}

		if i != last {
			if _, ok := d.Specs[i].(*ParamSpec); ok {
				if _, ok := d.Specs[i+1].(*NamedParamSpec); ok {
					if _, err = ctx.WriteString(", "); err != nil {
						return
					}
					continue
				}
			}
			if _, err = ctx.WriteString(", "); err != nil {
				return
			}
		}
	}
	if d.Rparen > 0 {
		return ctx.WriteByte(')')
	}
	return
}
