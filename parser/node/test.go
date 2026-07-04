package node

import (
	"strconv"

	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/source"
)

// TestKind selects whether a TestStmt declares a test or a benchmark.
type TestKind uint8

const (
	// TestKindTest is a `test NAME { … }` statement.
	TestKindTest TestKind = iota
	// TestKindBench is a `bench NAME { … }` statement.
	TestKindBench
)

// String returns the keyword spelling (`test` or `bench`).
func (k TestKind) String() string {
	if k == TestKindBench {
		return "bench"
	}
	return "test"
}

// TestStmt is a `test NAME { … }` or `bench NAME { … }` statement. NAME is an
// identifier or a string literal; the body runs with an injected `t` test
// context (see the `test` module). `test`/`bench` are contextual: they are only
// this statement when followed by a NAME and `{`, so they remain ordinary
// identifiers everywhere else.
type TestStmt struct {
	Kind    TestKind
	KwPos   source.Pos // position of the `test`/`bench` keyword
	Name    string     // the test name (identifier spelling or string value)
	NamePos source.Pos
	Quoted  bool // NAME was written as a string literal
	Body    *BlockStmt
	Doc     *ast.CommentGroup // doc comment preceding the statement; or nil
}

func (s *TestStmt) StmtNode() {}

func (s *TestStmt) Pos() source.Pos {
	if s.KwPos.IsValid() {
		return s.KwPos
	}
	return s.Body.Pos()
}

func (s *TestStmt) End() source.Pos { return s.Body.End() }

func (s *TestStmt) String() string { return Code(s) }

func (s *TestStmt) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteLeadDoc(s.Doc)
	ctx.WriteString(s.Kind.String())
	ctx.WriteString(" ")
	if s.Quoted || !isIdent(s.Name) {
		ctx.WriteString(strconv.Quote(s.Name))
	} else {
		ctx.WriteString(s.Name)
	}
	ctx.WriteString(" ")
	s.Body.WriteCode(ctx)
}

// isIdent reports whether name is a bare identifier (so it can be written
// without quotes): a letter or `_` followed by letters, digits or `_`.
func isIdent(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		switch {
		case r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
		case i > 0 && r >= '0' && r <= '9':
		default:
			return false
		}
	}
	return true
}
