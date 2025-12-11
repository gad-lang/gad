package node

import (
	"bytes"

	"github.com/gad-lang/gad/parser/ast"
)

type Node interface {
	ast.Node
	Coder
}

// Expr represents an expression node in the AST.
type Expr interface {
	Node
	ExprNode()
}

type Exprs []Expr

func IsExpr(n Node) (ok bool) {
	_, ok = n.(Expr)
	return
}

// Stmt represents a statement in the AST.
type Stmt interface {
	Node
	StmtNode()
}

type Stmts []Stmt

func (s Stmts) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteStmts(s...)
}

func (s *Stmts) Append(n ...Stmt) {
	*s = append(*s, n...)
}

func (s *Stmts) Prepend(n ...Stmt) {
	*s = append(n, *s...)
}

func (s Stmts) Each(f func(i int, sep bool, s Stmt)) {
	sep := true

	for i, stmt := range s {
		f(i, sep && i > 0, stmt)

		switch stmt.(type) {
		case *CodeBeginStmt:
			sep = true
		case *CodeEndStmt, *ConfigStmt:
			sep = false
		}
	}
}

func (s Stmts) Filter(f func(i int, s Stmt) bool) (l Stmts) {
	for i, stmt := range s {
		if f(i, stmt) {
			l = append(l, stmt)
		}
	}
	return
}

func (s Stmts) Map(f func(i int, s Stmt) Stmt) (l Stmts) {
	for i, stmt := range s {
		if stmt = f(i, stmt); stmt != nil {
			l = append(l, stmt)
		}
	}
	return
}

func (s Stmts) String() string {
	var w bytes.Buffer
	s.Each(func(i int, sep bool, stmt Stmt) {
		if sep {
			w.WriteString("; ")
		}
		w.WriteString(stmt.String())
	})
	return w.String()
}

// IsStatement returns true if given value is implements interface{ StmtNode() }.
func IsStatement(v Node) (ok bool) {
	_, ok = v.(Stmt)
	return ok
}

type ToMultiParenConverter interface {
	Expr
	ToMultiParenExpr() *MultiParenExpr
}
