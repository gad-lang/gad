// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

// Copyright (c) 2019 Daniel Kang.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE.tengo file.

// Copyright 2009 The ToInterface Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.golang file.

package ast

import (
	"strings"

	"github.com/gad-lang/gad/parser/source"
)

// Node represents a node in the AST.
type Node interface {
	// Pos returns the position of first character belonging to the node.
	Pos() source.Pos
	// End returns the position of first character immediately after the node.
	End() source.Pos
	// String returns a string representation of the node.
	String() string
}

type DataNoder interface {
	Node
	SetData(key, value any)
	GetData(key any) (value any, ok bool)
}

type NodeData struct {
	data map[any]any
}

func (nd *NodeData) SetData(key, value any) {
	if nd.data == nil {
		nd.data = map[any]any{}
	}
	nd.data[key] = value
}

func (nd *NodeData) GetData(key any) (value any, ok bool) {
	value, ok = nd.data[key]
	return
}

// ----------------------------------------------------------------------------
// Comments

// A Comment node represents a single //-style or /*-style comment.
type Comment struct {
	Slash source.Pos // position of "/" starting the comment
	Text  string     // comment text (excluding '\n' for //-style comments)
}

// Pos returns the position of the comment's slash.
func (c *Comment) Pos() source.Pos { return c.Slash }

// End returns the position of first character immediately after the comment.
func (c *Comment) End() source.Pos {
	return source.Pos(int(c.Slash) + len(c.Text))
}

// A CommentGroup represents a sequence of comments
// with no other tokens and no empty lines between.
type CommentGroup struct {
	List []*Comment // len(List) > 0
}

// Pos returns the position of the first comment.
func (g *CommentGroup) Pos() source.Pos {
	return g.List[0].Pos()
}

// End returns the position of last comment's end position.
func (g *CommentGroup) End() source.Pos {
	return g.List[len(g.List)-1].End()
}

// Text returns the text of the comment.
// Comment markers (//, /*, and */), the first space of a line comment, and
// leading and trailing empty lines are removed.
// Multiple empty lines are reduced to one, and trailing space on lines is trimmed.
// Unless the result is empty, it is newline-terminated.
func (g *CommentGroup) Text() string {
	if g == nil {
		return ""
	}
	comments := make([]string, len(g.List))
	for i, c := range g.List {
		comments[i] = c.Text
	}

	lines := make([]string, 0, 10) // most comments are less than 10 lines
	for _, c := range comments {
		// Remove comment markers.
		// The parser has given us exactly the comment text.
		switch c[1] {
		case '/':
			// -style comment (no newline at the end)
			c = c[2:]
			if len(c) == 0 {
				// empty line
				break
			}
			if c[0] == ' ' {
				// strip first space - required for Example tests
				c = c[1:]
			}
		case '*':
			/*-style comment */
			c = c[2 : len(c)-2]
		}

		// Split on newlines.
		cl := strings.Split(c, "\n")

		// Walk lines, stripping trailing white space and adding to list.
		for _, l := range cl {
			lines = append(lines, stripTrailingWhitespace(l))
		}
	}

	// Remove leading blank lines; convert runs of
	// interior blank lines to a single blank line.
	n := 0
	for _, line := range lines {
		if line != "" || n > 0 && lines[n-1] != "" {
			lines[n] = line
			n++
		}
	}
	lines = lines[0:n]

	// Add final "" entry to get trailing newline from Join.
	if n > 0 && lines[n-1] != "" {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

func isWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

func stripTrailingWhitespace(s string) string {
	i := len(s)
	for i > 0 && isWhitespace(s[i-1]) {
		i--
	}
	return s[0:i]
}

type Literal struct {
	Value string
	Pos   source.Pos
}

func (l Literal) End() source.Pos {
	return l.Pos + source.Pos(len(l.Value))
}
