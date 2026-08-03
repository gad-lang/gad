package gadbridge

import (
	"fmt"
	"strings"

	giomnode "github.com/gad-lang/gad/giom/node"
	giomparser "github.com/gad-lang/gad/giom/parser"
	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/ast"
	gnode "github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
)

// Doc renders Markdown documentation for the input source. `.giom` templates are
// documented via the giom front-end (components, functions and the other
// top-level declarations, each with a data-source-pos anchor); everything else
// is documented as Gad (the module heading, a leading `/***` block and the
// exported symbols). It is filesystem-free, so the WASM bridge can produce docs
// for the editor's current buffer.
func Doc(path, src string) (string, error) {
	if strings.HasSuffix(path, ".giom") {
		return giomDocMarkdown(path, []byte(src))
	}
	return gadDocMarkdown(path, []byte(src))
}

// docModuleName is the file's base name without extension.
func docModuleName(path string) string {
	base := path
	if i := strings.LastIndexAny(base, "/\\"); i >= 0 {
		base = base[i+1:]
	}
	if i := strings.LastIndex(base, "."); i > 0 {
		base = base[:i]
	}
	return base
}

// --- Gad ---

func gadDocMarkdown(path string, src []byte) (string, error) {
	fs := source.NewFileSet()
	f := fs.AddFileData(path, -1, src)
	file, err := parser.NewParserWithOptions(
		f, &parser.ParserOptions{Mode: parser.ParseComments}, nil).ParseFile()
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString("# " + docModuleName(path) + "\n")

	// Leading /*** … ***/ root block as module prose.
	for _, g := range file.Comments {
		if len(g.List) > 0 && strings.HasPrefix(g.List[0].Text, "/***") {
			if c := cleanDoc(g.List[0].Text); c != "" {
				b.WriteString("\n" + c + "\n")
			}
			break
		}
	}

	// Exported symbols.
	var exports []string
	for _, stmt := range file.Stmts {
		es, ok := stmt.(*gnode.ExportStmt)
		if !ok {
			continue
		}
		name := gadExportName(es)
		if name == "" {
			continue
		}
		entry := "### " + name
		if es.ValueExpr != nil {
			if _, isFunc := es.ValueExpr.(*gnode.FuncExpr); !isFunc {
				entry += " = " + es.ValueExpr.String()
			}
		}
		if doc := gadDocText(es.Doc); doc != "" {
			entry += "\n\n" + doc
		}
		exports = append(exports, entry)
	}
	if len(exports) > 0 {
		b.WriteString("\n## Exports\n")
		for _, e := range exports {
			b.WriteString("\n" + e + "\n")
		}
	}
	return b.String(), nil
}

// gadDocText returns the cleaned text of an export's doc comment group.
func gadDocText(g *ast.CommentGroup) string {
	if g == nil || len(g.List) == 0 {
		return ""
	}
	var parts []string
	for _, c := range g.List {
		if t := cleanDoc(c.Text); t != "" {
			parts = append(parts, t)
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

// cleanDoc strips comment markers from a raw doc comment, handling Gad's
// `/*** ***/` and `/** **/` block forms and `///` / `//` line forms.
func cleanDoc(raw string) string {
	s := strings.TrimSpace(raw)
	for _, p := range [][2]string{{"/***", "***/"}, {"/**", "**/"}, {"/*", "*/"}} {
		if strings.HasPrefix(s, p[0]) && strings.HasSuffix(s, p[1]) && len(s) >= len(p[0])+len(p[1]) {
			return strings.TrimSpace(s[len(p[0]) : len(s)-len(p[1])])
		}
	}
	if strings.HasPrefix(s, "///") {
		return strings.TrimSpace(s[3:])
	}
	if strings.HasPrefix(s, "//") {
		return strings.TrimSpace(s[2:])
	}
	return s
}

func gadExportName(es *gnode.ExportStmt) string {
	if id, ok := es.KeyExpr.(*gnode.IdentExpr); ok {
		return id.Name
	}
	if fe, ok := es.ValueExpr.(*gnode.FuncExpr); ok && fe.Type.NameExpr != nil {
		if id, ok := fe.Type.NameExpr.(*gnode.IdentExpr); ok {
			return id.Name
		}
	}
	if es.KeyExpr != nil {
		return es.KeyExpr.String()
	}
	return ""
}

// --- Giom ---

func giomDocMarkdown(path string, src []byte) (string, error) {
	fs := source.NewFileSet()
	f := fs.AddFileData(path, -1, src)
	file, err := giomparser.NewParser(f).ParseFile()
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString("# " + docModuleName(path) + "\n")

	if prose := giomLeadProse(file); prose != "" {
		b.WriteString("\n" + prose + "\n")
	}

	var exports, comps, funcs, params, consts, vars, enums []giomDocEntry
	for _, stmt := range file.Stmts {
		switch d := stmt.(type) {
		case *giomnode.ExportStmt:
			exports = append(exports, giomEntry(f, d.Name, giomExportValue(d.Value), d.Doc, d.Pos()))
		case *giomnode.CompDecl:
			comps = append(comps, giomEntry(f, "+"+d.Name, giomParams(d.ParamsRaw), d.Doc, d.Pos()))
		case *giomnode.FuncDecl:
			funcs = append(funcs, giomEntry(f, d.Name, giomParams(d.ParamsRaw), d.Doc, d.Pos()))
		case *giomnode.ParamStmt:
			params = append(params, giomEntry(f, "@param", giomDeclSig(d.Decl, "param"), d.Doc, d.Pos()))
		case *giomnode.ConstStmt:
			consts = append(consts, giomEntry(f, "@const", giomVarSig(d.Decls), d.Doc, d.Pos()))
		case *giomnode.VarStmt:
			vars = append(vars, giomEntry(f, "@var", giomVarSig(d.Decls), d.Doc, d.Pos()))
		case *giomnode.EnumStmt:
			enums = append(enums, giomEntry(f, d.Name, "", d.Doc, d.Pos()))
		}
	}

	giomSection(&b, "Exports", exports)
	giomSection(&b, "Components", comps)
	giomSection(&b, "Functions", funcs)
	giomSection(&b, "Parameters", params)
	giomSection(&b, "Constants", consts)
	giomSection(&b, "Variables", vars)
	giomSection(&b, "Enums", enums)
	return b.String(), nil
}

type giomDocEntry struct {
	name, params, doc string
	line, column      int
}

func giomEntry(f *source.File, name, params, doc string, pos source.Pos) giomDocEntry {
	fp := source.MustFilePosition(f, pos)
	return giomDocEntry{name: name, params: params, doc: doc, line: fp.Line, column: fp.Column}
}

func giomExportValue(v gnode.Expr) string {
	if v == nil {
		return ""
	}
	return " = " + v.String()
}

func giomDeclSig(decl *gnode.GenDecl, keyword string) string {
	if decl == nil {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(decl.String()), keyword))
}

func giomVarSig(decls []giomnode.VarDecl) string {
	var parts []string
	for _, d := range decls {
		if d.Init != nil {
			parts = append(parts, d.Name+" = "+d.Init.String())
		} else {
			parts = append(parts, d.Name)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

func giomParams(raw string) string {
	raw = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(raw), "("), ")"))
	if raw == "" {
		return ""
	}
	return "(" + raw + ")"
}

func giomSection(b *strings.Builder, title string, entries []giomDocEntry) {
	if len(entries) == 0 {
		return
	}
	fmt.Fprintf(b, "\n## %s\n", title)
	for _, e := range entries {
		fmt.Fprintf(b, "\n### <span data-source-pos=\"%d,%d\">%s</span>%s\n", e.line, e.column, e.name, e.params)
		if doc := strings.TrimSpace(e.doc); doc != "" {
			b.WriteString("\n" + doc + "\n")
		}
	}
}

func giomLeadProse(file *giomnode.File) string {
	for _, stmt := range file.Stmts {
		c, ok := stmt.(*giomnode.CommentStmt)
		if !ok {
			return ""
		}
		if text := strings.TrimSpace(c.Text); text != "" {
			return text
		}
	}
	return ""
}
