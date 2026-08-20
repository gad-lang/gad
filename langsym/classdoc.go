package langsym

import (
	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/token"
)

// ClassMemberDocs returns the lead doc comments of the members (fields,
// properties, methods) of the class named className, when that class is declared
// in src. It lets runtime member completion attach the source documentation of a
// same-file class instance, which the live value does not carry.
func ClassMemberDocs(src []byte, className string) map[string]string {
	if className == "" {
		return nil
	}
	fs := source.NewFileSet()
	sf := fs.AddFileData("t.gad", -1, src)
	po := &parser.ParserOptions{Mode: parser.ParseComments}
	// The source is often mid-edit (the caret line does not parse); a partial
	// AST still contains the class declaration, so use it even on error.
	f, _ := parser.NewParserWithOptions(sf, po, nil).ParseFile()
	if f == nil {
		return nil
	}

	target := findClass(f, className)
	if target == nil {
		return nil
	}
	return memberDocs(target)
}

// findClass locates the class expression declared under className: a `class Name`
// statement, or a `Name := class { … }` / `const Name = class { … }` binding.
func findClass(f *parser.File, className string) *node.ClassExpr {
	var target *node.ClassExpr
	node.Walk(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *node.ClassStmt:
			if identName(x.NameExpr) == className {
				target = &x.ClassExpr
			}
		case *node.ClassExpr:
			if identName(x.NameExpr) == className {
				target = x
			}
		case *node.AssignStmt:
			if x.Token == token.Define {
				bindClass(x.LHS, x.RHS, className, &target)
			}
		case *node.GenDecl:
			for _, spec := range x.Specs {
				if vs, ok := spec.(*node.ValueSpec); ok {
					idents := make([]node.Expr, len(vs.Idents))
					for i, id := range vs.Idents {
						idents[i] = id
					}
					bindClass(idents, vs.Values, className, &target)
				}
			}
		}
		return target == nil // stop as soon as the class is found
	})
	return target
}

// bindClass sets *target when lhs[i] names className and rhs[i] is a class.
func bindClass(lhs, rhs []node.Expr, className string, target **node.ClassExpr) {
	for i, l := range lhs {
		if identName(l) == className && i < len(rhs) {
			if ce, ok := rhs[i].(*node.ClassExpr); ok {
				*target = ce
			}
		}
	}
}

// memberDocs maps each documented member name of c to its doc text.
func memberDocs(c *node.ClassExpr) map[string]string {
	docs := map[string]string{}
	put := func(name string, g *ast.CommentGroup) {
		if name == "" || g == nil {
			return
		}
		if t := docText(g); t != "" {
			docs[name] = t
		}
	}
	for _, fld := range c.Fields {
		if fld.Name != nil {
			put(identName(fld.Name), fld.Doc)
		}
	}
	for _, p := range c.Props {
		put(identName(p.NameExpr), memberDoc(p))
	}
	for _, m := range c.Methods {
		put(identName(m.NameExpr), memberDoc(m))
	}
	return docs
}

// memberDoc is a class member's doc: its own, or the first overload's.
func memberDoc(m *node.ClassMemberExpr) *ast.CommentGroup {
	if m.Doc != nil {
		return m.Doc
	}
	if len(m.Methods) > 0 {
		return m.Methods[0].Doc
	}
	return nil
}

// identName returns the identifier name of an expression that is (or wraps) an
// identifier: *IdentExpr or *TypedIdentExpr.
func identName(e ast.Node) string {
	switch x := e.(type) {
	case *node.IdentExpr:
		return x.Name
	case *node.TypedIdentExpr:
		if x.Ident != nil {
			return x.Ident.Name
		}
	}
	return ""
}
