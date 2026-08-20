package langsym

import (
	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
)

// Definition resolves the identifier at the caret (a 0-based file offset) to its
// declaration and returns the declaration's 0-based file offset. ok is false when
// the caret is not on an identifier or no declaration is in scope.
func Definition(f *parser.File, sf *source.File, offset int) (declOffset int, ok bool) {
	r := newResolver(f, sf)
	p := r.pos(offset)
	name, ip := r.identAt(f, p)
	if name == "" {
		return 0, false
	}
	d := r.resolve(name, ip)
	if d == nil {
		return 0, false
	}
	off, err := sf.Offset(d.Pos)
	if err != nil {
		return 0, false
	}
	return off, true
}

// identAt returns the name and position of the innermost identifier whose span
// contains p.
func (r *resolver) identAt(f *parser.File, p source.Pos) (name string, pos source.Pos) {
	node.Walk(f, func(n ast.Node) bool {
		if id, ok := n.(*node.IdentExpr); ok && !id.Empty {
			if id.Pos() <= p && p < id.End() {
				name, pos = id.Name, id.Pos()
			}
		}
		return true
	})
	return
}

// resolve finds the declaration of name visible at p: the nearest enclosing
// scope's declaration of name with Pos <= p (respecting declaration order and
// shadowing); if none appears before p in any scope, the nearest declaration of
// name (e.g. a hoisted function).
func (r *resolver) resolve(name string, p source.Pos) *Decl {
	var fallback *Decl
	for s := r.scopeAt(p); s != nil; s = s.parent {
		var before *Decl
		for i := range s.decls {
			d := &s.decls[i]
			if d.Name != name {
				continue
			}
			if fallback == nil {
				fallback = d
			}
			if d.Pos <= p && (before == nil || d.Pos > before.Pos) {
				before = d
			}
		}
		if before != nil {
			return before
		}
	}
	return fallback
}
