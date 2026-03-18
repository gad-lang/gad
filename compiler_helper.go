package gad

import (
	"github.com/gad-lang/gad/parser/ast"
)

func (c *Compiler) requireSymbol(nd ast.Node, name string) (s *Symbol, err error) {
	var ok bool
	if s, ok = c.symbolTable.Resolve(name); !ok {
		err = c.errorf(nd, "unresolved reference %q", name)
	}
	return
}
