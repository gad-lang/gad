package node

import (
	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/source"
)

// ExampleStmt is `example [NAME] { … }` (issue #22): code verified by its
// inline checks — `= EXPR`, `<<< text` inside it, and `<<<` right after it for
// all it printed. Named, it is a snippet: `@snippet NAME` in a doc comment
// renders its body and the checks after it. Its code is of the scope around it,
// as the code of a file is: a later example uses an earlier one's definitions
// (its context, not rendered with it).
type ExampleStmt struct {
	KwPos source.Pos
	Name  *IdentExpr // nil: unnamed
	Body  *BlockStmt
	Doc   *ast.CommentGroup
}

func (s *ExampleStmt) StmtNode() {}

func (s *ExampleStmt) Pos() source.Pos { return s.KwPos }

func (s *ExampleStmt) End() source.Pos { return s.Body.End() }

func (s *ExampleStmt) String() string { return Code(s) }

func (s *ExampleStmt) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteLeadDoc(s.Doc)
	ctx.WriteString("example ")
	if s.Name != nil {
		ctx.WriteString(s.Name.Name + " ")
	}
	s.Body.WriteCode(ctx)
}
