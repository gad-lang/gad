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
// contains p, descending into `#"…{expr}…"` string interpolations.
func (r *resolver) identAt(f *parser.File, p source.Pos) (name string, pos source.Pos) {
	return identAtNode(f, p)
}

// identAtNode finds the innermost identifier at p under root. A `{ … }` island
// inside an InterpolatedStringLit is not part of the parsed AST (the literal
// holds the raw string), so when p falls inside one the island is parsed on
// demand — its expressions carry absolute source positions — and searched
// recursively (interpolations can nest).
func identAtNode(root ast.Node, p source.Pos) (name string, pos source.Pos) {
	node.Walk(root, func(n ast.Node) bool {
		switch e := n.(type) {
		case *node.IdentExpr:
			if !e.Empty && e.Pos() <= p && p < e.End() {
				name, pos = e.Name, e.Pos()
			}
		case *node.InterpolatedStringLit:
			if e.Pos() <= p && p < e.End() {
				if sub := parseInterpolation(e); sub != nil {
					if nm, ps := identAtNode(sub, p); nm != "" {
						name, pos = nm, ps
					}
				}
			}
		}
		return true
	})
	return
}

// parseInterpolation parses the interpolation islands of an InterpolatedStringLit
// into a File whose expressions carry absolute source positions (the same
// lowering the compiler uses), or nil when it does not parse. Mirrors the value
// dispatch in compiler_nodes.go.
func parseInterpolation(nd *node.InterpolatedStringLit) *parser.File {
	var (
		tmpl string
		raw  bool
	)
	switch t := nd.Value.(type) {
	case *node.StrLit:
		tmpl = t.InterpolationTemplate()
	case *node.RawStrLit:
		tmpl, raw = t.Value(), true
	case *node.RawHeredocLit:
		tmpl, raw = t.RawContent(), true
	case *node.HeredocLit:
		tmpl = t.RawContent()
	case *node.SymbolLit:
		tmpl = t.Value()
	default:
		return nil
	}
	f, err := parser.ParseInterpolatedStringMode(tmpl, nd.StringValuePos(), raw)
	if err != nil {
		return nil
	}
	return f
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
