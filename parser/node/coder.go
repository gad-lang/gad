package node

import (
	"fmt"
	"io"

	"github.com/gad-lang/gad/parser/ast"
)

type CodeWriter interface {
	io.Writer
	io.StringWriter
	io.ByteWriter
}

type CodeWriterContext struct {
	Stack []ast.Node
	CodeWriter
	ExprToTextFunc string
}

func (c *CodeWriterContext) Top() ast.Node {
	return c.Stack[len(c.Stack)-1]
}

func (c *CodeWriterContext) Push(n ast.Node) {
	c.Stack = append(c.Stack, n)
}

func (c *CodeWriterContext) Pop() {
	c.Stack = c.Stack[:len(c.Stack)-1]
}

func (c *CodeWriterContext) With(n ast.Node, cb func() error) (err error) {
	c.Push(n)
	err = cb()
	c.Pop()
	return
}

type Coder interface {
	WriteCode(ctx *CodeWriterContext) error
}

func WriteCode(ctx *CodeWriterContext, node ...ast.Node) (err error) {
	for _, node := range node {
		if err = ctx.With(node, func() error {
			switch n := node.(type) {
			case Coder:
				return n.WriteCode(ctx)
			default:
				_, err := fmt.Fprint(ctx, node)
				return err
			}
		}); err != nil {
			return
		}
	}
	return
}

func WriteCodeExprs(ctx *CodeWriterContext, sep string, expr ...Expr) (err error) {
	last := len(expr) - 1
	for i, e := range expr {
		if err = WriteCode(ctx, e); err != nil {
			return
		}
		if i != last {
			if _, err = ctx.WriteString(sep); err != nil {
				return
			}
		}
	}
	return
}

func WriteCodeStmts(ctx *CodeWriterContext, stmt ...Stmt) (err error) {
	last := len(stmt) - 1
	for i, e := range stmt {
		if err = WriteCode(ctx, e); err != nil {
			return
		}
		if i != last {
			if _, err = ctx.WriteString(";\n"); err != nil {
				return
			}
		}
	}
	return
}

func WriteCodeValidStmts(ctx *CodeWriterContext, stmt ...Stmt) (err error) {
	var smts []Stmt
	for _, s := range stmt {
		if s != nil {
			smts = append(smts, s)
		}
	}
	return WriteCodeStmts(ctx, smts...)
}
