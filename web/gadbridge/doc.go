package gadbridge

import (
	"fmt"
	"strings"

	"github.com/gad-lang/gad"
	gadxnode "github.com/gad-lang/gad/gadx/node"
	gadxparser "github.com/gad-lang/gad/gadx/parser"
	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/ast"
	gnode "github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
)

// DocData is the structured documentation extracted from a source buffer — a
// JSON-serializable tree that a caller can render however it likes (the default
// Markdown renderer is RenderMarkdown; the CLI can render HTML via a gadx
// template). This is the shape the WASM bridge and the gad doc API return.
type DocData struct {
	// Prose is the module-level description (a leading `/***` block or a gadx
	// leading comment), or "".
	Prose string `json:"prose,omitempty"`
	// Sections group the documented symbols by kind ("Exports", "Components",
	// "Functions", "Parameters", "Constants", "Variables", "Enums").
	Sections []DocSection `json:"sections,omitempty"`
}

// DocSection is a named group of symbols.
type DocSection struct {
	Title   string      `json:"title"`
	Symbols []DocSymbol `json:"symbols"`
}

// DocSymbol is one documented declaration.
type DocSymbol struct {
	Name string `json:"name"`
	// Signature is the parenthesized parameter list / value suffix, or "".
	Signature string `json:"signature,omitempty"`
	// Doc is the attached doc comment text, or "".
	Doc string `json:"doc,omitempty"`
	// Line/Column locate the declaration in the source (1-based; 0 when unknown),
	// for editor navigation (e.g. data-source-pos).
	Line   int `json:"line,omitempty"`
	Column int `json:"column,omitempty"`
}

// ExtractDoc extracts the structured documentation from a source buffer.
// sourceType selects the dialect: "gadx", "gadTemplate" (or "template"), or
// "gad" (default).
func ExtractDoc(src, sourceType string) (*DocData, error) {
	if sourceType == "gadx" {
		return gadxDocData([]byte(src))
	}
	return gadDocData([]byte(src), sourceType)
}

// Doc extracts documentation and renders it as Markdown (the default renderer).
// For custom rendering, use ExtractDoc and render the structure yourself.
func Doc(src, sourceType string) (string, error) {
	d, err := ExtractDoc(src, sourceType)
	if err != nil {
		return "", err
	}
	return RenderMarkdown(d), nil
}

// GadDict converts the structured documentation into a Gad dict, the shape a
// `.gaddoc.gadx` / `.gaddoc-md.gadx` template consumes via `param (doc dict)`.
// Layout: { prose: str, sections: [ { title: str, symbols: [ { name, signature,
// doc: str, line, column: int } ] } ] }.
func (d *DocData) GadDict() gad.Dict {
	secs := make(gad.Array, 0, len(d.Sections))
	for _, sec := range d.Sections {
		syms := make(gad.Array, 0, len(sec.Symbols))
		for _, s := range sec.Symbols {
			syms = append(syms, gad.Dict{
				"name":      gad.Str(s.Name),
				"signature": gad.Str(s.Signature),
				"doc":       gad.Str(s.Doc),
				"line":      gad.Int(s.Line),
				"column":    gad.Int(s.Column),
			})
		}
		secs = append(secs, gad.Dict{"title": gad.Str(sec.Title), "symbols": syms})
	}
	return gad.Dict{"prose": gad.Str(d.Prose), "sections": secs}
}

// RenderMarkdown renders a DocData as Markdown (prose, then a `## Title` section
// per group with a `### name` entry — the gadx entries carry a data-source-pos
// anchor for navigation).
func RenderMarkdown(d *DocData) string {
	var b strings.Builder
	if d.Prose != "" {
		b.WriteString(d.Prose + "\n")
	}
	for _, sec := range d.Sections {
		if len(sec.Symbols) == 0 {
			continue
		}
		fmt.Fprintf(&b, "\n## %s\n", sec.Title)
		for _, s := range sec.Symbols {
			if s.Line > 0 {
				fmt.Fprintf(&b, "\n### <span data-source-pos=\"%d,%d\">%s</span>%s\n", s.Line, s.Column, s.Name, s.Signature)
			} else {
				fmt.Fprintf(&b, "\n### %s%s\n", s.Name, s.Signature)
			}
			if s.Doc != "" {
				b.WriteString("\n" + s.Doc + "\n")
			}
		}
	}
	return b.String()
}

// --- Gad ---

func gadDocData(src []byte, sourceType string) (*DocData, error) {
	fs := source.NewFileSet()
	f := fs.AddFileData("buffer", -1, src)
	po := &parser.ParserOptions{Mode: parser.ParseComments}
	so := &parser.ScannerOptions{}
	if sourceType == "gadTemplate" || sourceType == "template" {
		po.Mode |= parser.ParseMixed
		so.Mode |= parser.ScanMixed | parser.ScanConfigDisabled
		so.MixedDelimiter = parser.DefaultMixedDelimiter
	}
	file, err := parser.NewParserWithOptions(f, po, so).ParseFile()
	if err != nil {
		return nil, err
	}

	d := &DocData{}

	// Leading /*** … ***/ root block as module prose.
	for _, g := range file.Comments {
		if len(g.List) > 0 && strings.HasPrefix(g.List[0].Text, "/***") {
			d.Prose = cleanDoc(g.List[0].Text)
			break
		}
	}
	// In mixed/template mode the leading /*** … ***/ block is literal text (not a
	// comment), so recover the module prose directly from the source.
	if d.Prose == "" && (sourceType == "gadTemplate" || sourceType == "template") {
		d.Prose = leadingRootBlock(src)
	}

	var exports []DocSymbol
	for _, stmt := range file.Stmts {
		es, ok := stmt.(*gnode.ExportStmt)
		if !ok {
			continue
		}
		name := gadExportName(es)
		if name == "" {
			continue
		}
		sym := DocSymbol{Name: name, Doc: gadDocText(es.Doc)}
		if es.ValueExpr != nil {
			if _, isFunc := es.ValueExpr.(*gnode.FuncExpr); !isFunc {
				sym.Signature = " = " + es.ValueExpr.String()
			}
		}
		fp := source.MustFilePosition(f, es.Pos())
		sym.Line, sym.Column = fp.Line, fp.Column
		exports = append(exports, sym)
	}
	if len(exports) > 0 {
		d.Sections = append(d.Sections, DocSection{Title: "Exports", Symbols: exports})
	}
	return d, nil
}

// leadingRootBlock returns the cleaned prose of a leading `/*** … ***/` root
// block for a mixed/template file, or "". Because template text outside the code
// delimiters is emitted verbatim, the module doc of a `.gadt` lives inside the
// leading code island — `{% /*** … ***/ %}` (any `-`/`--` trim markers allowed) —
// so the block is skipped past the opening delimiter here. A bare leading
// `/*** … ***/` (before any code) is also accepted. An optional `#!…` shebang
// line is skipped first.
func leadingRootBlock(src []byte) string {
	s := string(src)
	if strings.HasPrefix(s, "#!") {
		if i := strings.IndexByte(s, '\n'); i >= 0 {
			s = s[i+1:]
		}
	}
	s = strings.TrimLeft(s, " \t\r\n")
	// Step past a leading `{%` (and its `-`/`--` trim markers) so the module doc
	// may live inside the first code island, where it is not emitted as text.
	if start := string(parser.DefaultMixedDelimiter.Start); strings.HasPrefix(s, start) {
		s = strings.TrimLeft(s[len(start):], "-")
		s = strings.TrimLeft(s, " \t\r\n")
	}
	// Accept a `/*** … ***/` root block, a `/** … **/` block, or a normal
	// `/* … */` comment (longest opener first); cleanDoc strips the markers.
	for _, m := range [][2]string{{"/***", "***/"}, {"/**", "**/"}, {"/*", "*/"}} {
		if strings.HasPrefix(s, m[0]) {
			if end := strings.Index(s, m[1]); end >= 0 {
				return cleanDoc(s[:end+len(m[1])])
			}
			return ""
		}
	}
	return ""
}

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

// cleanDoc strips comment markers, handling Gad's `/*** ***/` and `/** **/`
// block forms and `///` / `//` line forms.
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

// --- Gadx ---

func gadxDocData(src []byte) (*DocData, error) {
	fs := source.NewFileSet()
	f := fs.AddFileData("buffer", -1, src)
	file, err := gadxparser.NewParser(f).ParseFile()
	if err != nil {
		return nil, err
	}

	d := &DocData{Prose: gadxLeadProse(file)}

	add := func(title string, syms []DocSymbol) {
		if len(syms) > 0 {
			d.Sections = append(d.Sections, DocSection{Title: title, Symbols: syms})
		}
	}
	var exports, comps, funcs, params, consts, vars, enums []DocSymbol
	for _, stmt := range file.Stmts {
		switch t := stmt.(type) {
		case *gadxnode.ExportStmt:
			exports = append(exports, gadxSym(f, t.Name, gadxExportValue(t.Value), t.Doc, t.Pos()))
		case *gadxnode.CompDecl:
			comps = append(comps, gadxSym(f, "+"+t.Name, gadxParams(t.ParamsRaw), t.Doc, t.Pos()))
		case *gadxnode.FuncDecl:
			funcs = append(funcs, gadxSym(f, t.Name, gadxParams(t.ParamsRaw), t.Doc, t.Pos()))
		case *gadxnode.ParamStmt:
			params = append(params, gadxSym(f, "@param", gadxDeclSig(t.Decl, "param"), t.Doc, t.Pos()))
		case *gadxnode.ConstStmt:
			consts = append(consts, gadxSym(f, "@const", gadxVarSig(t.Decls), t.Doc, t.Pos()))
		case *gadxnode.VarStmt:
			vars = append(vars, gadxSym(f, "@var", gadxVarSig(t.Decls), t.Doc, t.Pos()))
		case *gadxnode.EnumStmt:
			enums = append(enums, gadxSym(f, t.Name, "", t.Doc, t.Pos()))
		}
	}
	add("Exports", exports)
	add("Components", comps)
	add("Functions", funcs)
	add("Parameters", params)
	add("Constants", consts)
	add("Variables", vars)
	add("Enums", enums)
	return d, nil
}

func gadxSym(f *source.File, name, sig, doc string, pos source.Pos) DocSymbol {
	fp := source.MustFilePosition(f, pos)
	return DocSymbol{Name: name, Signature: sig, Doc: strings.TrimSpace(doc), Line: fp.Line, Column: fp.Column}
}

func gadxExportValue(v gnode.Expr) string {
	if v == nil {
		return ""
	}
	return " = " + v.String()
}

func gadxDeclSig(decl *gnode.GenDecl, keyword string) string {
	if decl == nil {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(decl.String()), keyword))
}

func gadxVarSig(decls []gadxnode.VarDecl) string {
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

func gadxParams(raw string) string {
	raw = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(raw), "("), ")"))
	if raw == "" {
		return ""
	}
	return "(" + raw + ")"
}

func gadxLeadProse(file *gadxnode.File) string {
	for _, stmt := range file.Stmts {
		c, ok := stmt.(*gadxnode.CommentStmt)
		if !ok {
			return ""
		}
		if text := strings.TrimSpace(c.Text); text != "" {
			return text
		}
	}
	return ""
}
