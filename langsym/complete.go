package langsym

import (
	"strings"

	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/source"
)

// Symbol is a completion candidate.
type Symbol struct {
	Label string `json:"label"`
	Kind  string `json:"kind"`          // variable | function | keyword | constant | field | module
	Doc   string `json:"doc,omitempty"` // documentation, if any
}

// Completions returns the identifiers in scope at the caret (a 0-based file
// offset): every declaration visible at that point, inner scopes shadowing
// outer ones. Keyword/builtin/member candidates are added by the caller.
func Completions(f *parser.File, sf *source.File, offset int) []Symbol {
	r := newResolver(f, sf)
	p := r.pos(offset)
	seen := map[string]bool{}
	var out []Symbol
	for s := r.scopeAt(p); s != nil; s = s.parent {
		for i := range s.decls {
			d := &s.decls[i]
			if d.Pos > p || seen[d.Name] {
				continue // declared later, or shadowed by a nearer declaration
			}
			seen[d.Name] = true
			doc := d.Doc
			if doc == nil {
				doc = r.leadDoc(d.Pos) // `:=` lead comment, associated by line
			}
			out = append(out, Symbol{Label: d.Name, Kind: "variable", Doc: docText(doc)})
		}
	}
	return out
}

// docText renders a doc comment group to plain text, stripping the comment
// markers (`///`, `//`, `/** … **/`, `/* … */`).
func docText(g *ast.CommentGroup) string {
	if g == nil {
		return ""
	}
	var lines []string
	for _, c := range g.List {
		for _, ln := range strings.Split(c.Text, "\n") {
			ln = strings.TrimSpace(ln)
			for _, p := range []string{"///", "//", "/***", "/**", "/*", "***", "**", "*"} {
				if strings.HasPrefix(ln, p) {
					ln = strings.TrimPrefix(ln, p)
					break
				}
			}
			for _, s := range []string{"***/", "**/", "*/"} {
				ln = strings.TrimSuffix(ln, s)
			}
			lines = append(lines, strings.TrimSpace(ln))
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
