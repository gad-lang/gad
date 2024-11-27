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
	"bufio"
	"io"
	"strings"

	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/stringw"
)

// File represents a file unit.
type File struct {
	InputFile *source.File
	Stmts     []node.Stmt
	Comments  []*ast.CommentGroup
}

// Pos returns the position of first character belonging to the node.
func (n *File) Pos() source.Pos {
	return source.Pos(n.InputFile.Base)
}

// End returns the position of first character immediately after the node.
func (n *File) End() source.Pos {
	return source.Pos(n.InputFile.Base + n.InputFile.Size)
}

func (n *File) StringTo(w stringw.StringWriter) {
	node.WriteCodeStmts(&node.CodeWriterContext{CodeWriter: w}, n.Stmts...)
}

func (n *File) String() string {
	var s strings.Builder
	n.StringTo(&s)
	return s.String()
}

func (n *File) TanspileTo(w io.Writer) (err error) {
	ctx := node.CodeWriterContext{
		CodeWriter: bufio.NewWriter(w),
	}
	return node.WriteCodeStmts(&ctx, n.Stmts...)
}

func (n *File) Tanspile() (s string, err error) {
	var b strings.Builder
	ctx := node.CodeWriterContext{
		CodeWriter: &b,
	}
	err = node.WriteCodeStmts(&ctx, n.Stmts...)
	s = b.String()
	return
}
