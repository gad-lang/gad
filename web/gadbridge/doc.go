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
	// Sections group the documented symbols by kind ("Public API", "Components",
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
	// Overloads holds the per-signature entries of a multi-signature function
	// (an `export func NAME { (sig) => … … }`); empty for a plain symbol.
	Overloads []DocOverload `json:"overloads,omitempty"`
	// Meta is the declaration's `[k=v, …]` metadata tag as source, or "".
	Meta string `json:"meta,omitempty"`
	// Members are the documented members of a type declaration (class fields,
	// properties, constructors and methods; interface requirements; enum
	// variants; a typed array type's body), each with its own metadata tag.
	Members []DocMember `json:"members,omitempty"`
	// Line/Column locate the declaration in the source (1-based; 0 when unknown),
	// for editor navigation (e.g. data-source-pos).
	Line   int `json:"line,omitempty"`
	Column int `json:"column,omitempty"`
}

// DocMember is one member of a documented type declaration.
type DocMember struct {
	// Group is the member kind: "Fields", "Properties", "Constructors",
	// "Methods", "Required" (an interface requirement) or "Variants".
	Group string `json:"group"`
	// Signature is the member as source, e.g. `x int = 0`, `sum() <int>`.
	Signature string `json:"signature"`
	// Meta is the member's `[k=v, …]` metadata tag as source, or "".
	Meta string `json:"meta,omitempty"`
	// Doc is the member's doc comment, or "".
	Doc string `json:"doc,omitempty"`
}

// DocOverload is one signature of a multi-signature function.
type DocOverload struct {
	// Signature is the parenthesized parameter list + return, e.g. `(r float) <float>`.
	Signature string `json:"signature"`
	// Doc is the overload's own doc comment, or "".
	Doc string `json:"doc,omitempty"`
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
// doc, meta: str, overloads: [ {signature, doc} ], members: [ {group,
// signature, meta, doc} ], line, column: int } ] } ] }.
func (d *DocData) GadDict() gad.Dict {
	secs := make(gad.Array, 0, len(d.Sections))
	for _, sec := range d.Sections {
		syms := make(gad.Array, 0, len(sec.Symbols))
		for _, s := range sec.Symbols {
			overloads := make(gad.Array, 0, len(s.Overloads))
			for _, o := range s.Overloads {
				overloads = append(overloads, gad.Dict{
					"signature": gad.Str(o.Signature),
					"doc":       gad.Str(o.Doc),
				})
			}
			members := make(gad.Array, 0, len(s.Members))
			for _, m := range s.Members {
				members = append(members, gad.Dict{
					"group":     gad.Str(m.Group),
					"signature": gad.Str(m.Signature),
					"meta":      gad.Str(m.Meta),
					"doc":       gad.Str(m.Doc),
				})
			}
			syms = append(syms, gad.Dict{
				"name":      gad.Str(s.Name),
				"signature": gad.Str(s.Signature),
				"doc":       gad.Str(s.Doc),
				"meta":      gad.Str(s.Meta),
				"overloads": overloads,
				"members":   members,
				"line":      gad.Int(s.Line),
				"column":    gad.Int(s.Column),
			})
		}
		secs = append(secs, gad.Dict{"title": gad.Str(sec.Title), "symbols": syms})
	}
	return gad.Dict{"prose": gad.Str(d.Prose), "sections": secs}
}

// RenderMarkdown renders a DocData as clean Markdown: the prose, then a
// `## Title` section per group with a `### name` entry per symbol. Source
// positions are kept in the structured DocSymbol (Line/Column) rather than
// embedded as raw HTML, so the rendered Markdown/HTML stays clean.
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
			// A `{data-src-pos="L,C"}` heading attribute carries the symbol's source
			// position so the rendered doc can navigate the editor on click (parsed
			// by goldmark's heading-attribute support and the frontend renderer). It
			// is stripped from the plain-Markdown source view.
			if s.Line > 0 {
				fmt.Fprintf(&b, "\n### %s%s {data-src-pos=\"%d,%d\"}\n", s.Name, s.Signature, s.Line, s.Column)
			} else {
				fmt.Fprintf(&b, "\n### %s%s\n", s.Name, s.Signature)
			}
			if s.Meta != "" {
				fmt.Fprintf(&b, "\n```gad\n%s\n```\n", s.Meta)
			}
			if s.Doc != "" {
				b.WriteString("\n" + s.Doc + "\n")
			}
			writeDocMembers(&b, s.Members)
			for _, o := range s.Overloads {
				fmt.Fprintf(&b, "\n```gad\n%s%s\n```\n", s.Name, o.Signature)
				if o.Doc != "" {
					b.WriteString("\n" + o.Doc + "\n")
				}
			}
		}
	}
	return b.String()
}

// writeDocMembers renders a type's members grouped by kind: a `#### Group`
// heading, then per member a fenced block (its metadata tag above the signature)
// and its doc.
func writeDocMembers(b *strings.Builder, members []DocMember) {
	group := ""
	for _, m := range members {
		if m.Group != group {
			group = m.Group
			fmt.Fprintf(b, "\n#### %s\n", group)
		}
		code := m.Signature
		if m.Meta != "" {
			code = m.Meta + "\n" + code
		}
		fmt.Fprintf(b, "\n```gad\n%s\n```\n", code)
		if m.Doc != "" {
			b.WriteString("\n" + m.Doc + "\n")
		}
	}
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

	// Module prose is a leading `/** … **/` block comment (three-star `/*** … ***/`
	// is still accepted) that is DETACHED from the code — followed by a blank line,
	// or with no statement immediately below it (end of file). A block comment
	// glued directly to a statement documents that statement, not the module.
	firstStmtLine := -1
	if len(file.Stmts) > 0 {
		firstStmtLine = source.MustFilePosition(f, file.Stmts[0].Pos()).Line
	}
	for _, g := range file.Comments {
		if len(g.List) == 0 || !strings.HasPrefix(g.List[0].Text, "/**") {
			continue
		}
		endLine := source.MustFilePosition(f, g.End()-1).Line
		if firstStmtLine < 0 || firstStmtLine > endLine+1 { // blank line / no stmt after
			d.Prose = cleanDoc(g.List[0].Text)
		}
		break // only the leading block is considered for the module doc
	}
	// In mixed/template mode the leading doc block is literal text (not a comment),
	// so recover the module prose directly from the source.
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
		switch v := es.ValueExpr.(type) {
		case *gnode.FuncExpr:
			// A function export carries its typed signature — the parameter list
			// and return types, without the name — so `name + signature` reads
			// `f(a int, b int) <int>`.
			if v.Type != nil {
				sym.Signature = v.Type.Params.String() + gnode.FormatFuncReturn(v.Type.Return)
			}
		case *gnode.FuncWithMethodsExpr:
			// A multi-signature export (`export func NAME { (sig) => … … }`): each
			// method is one overload with its own signature and doc.
			if v.Doc != nil && sym.Doc == "" {
				sym.Doc = gadDocText(v.Doc)
			}
			for _, m := range v.Methods {
				sym.Overloads = append(sym.Overloads, DocOverload{
					Signature: m.Params.String() + gnode.FormatFuncReturn(m.Return),
					Doc:       gadDocText(m.Doc),
				})
			}
		case nil:
			// Declaration exports (`export class/func/type … Name …`, `export
			// NAME = value`) desugar to a prelude declaration + a name export, so
			// the value/signature lives in the prelude, not ValueExpr.
			fillFromPrelude(&sym, es)
		default:
			sym.Signature = " = " + es.ValueExpr.String()
		}
		fp := source.MustFilePosition(f, es.Pos())
		sym.Line, sym.Column = fp.Line, fp.Column
		exports = append(exports, sym)
	}
	if len(exports) > 0 {
		d.Sections = append(d.Sections, DocSection{Title: "Public API", Symbols: exports})
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

// fillFromPrelude populates a symbol's signature/overloads from a declaration
// export's prelude (the `export <decl>` desugaring), so the public-API entry
// reflects the exported func signature, marker/union type or constant value.
func fillFromPrelude(sym *DocSymbol, es *gnode.ExportStmt) {
	switch d := es.Prelude.(type) {
	case *gnode.FuncStmt:
		if d.Func != nil && d.Func.Type != nil {
			sym.Signature = d.Func.Type.Params.String() + gnode.FormatFuncReturn(d.Func.Type.Return)
			sym.Meta = gnode.MetaCode(d.Func.Meta)
		}
	case *gnode.FuncWithMethodsStmt:
		if d.Doc != nil && sym.Doc == "" {
			sym.Doc = gadDocText(d.Doc)
		}
		for _, m := range d.Methods {
			sym.Overloads = append(sym.Overloads, DocOverload{
				Signature: m.Params.String() + gnode.FormatFuncReturn(m.Return),
				Doc:       gadDocText(m.Doc),
			})
		}
	case *gnode.TypeDeclStmt:
		kw := "class"
		if d.Mixin {
			kw = "mixin"
		} else if d.Static {
			kw = "type"
		}
		sym.Signature = " " + kw
		sym.Meta = gnode.MetaCode(d.Meta)
		sym.Members = classDocMembers(&d.TypeLitExpr)
	case *gnode.InterfaceStmt:
		sym.Signature = " interface"
		// An array interface shows its shape: `interface []{…}` for a member body,
		// `interface []<int | str>` for element types.
		if d.ArrayDepth > 0 {
			if len(d.ElemTypes) > 0 {
				sig := gnode.InterfaceExpr{ArrayDepth: d.ArrayDepth, ElemTypes: d.ElemTypes}
				sym.Signature = " " + sig.String()
			} else {
				sym.Signature += " " + strings.Repeat("[]", d.ArrayDepth) + "{…}"
			}
		}
		sym.Meta = gnode.MetaCode(d.Meta)
		sym.Members = interfaceDocMembers(&d.InterfaceExpr)
	case *gnode.EnumStmt:
		sym.Signature = " enum"
		sym.Meta = gnode.MetaCode(d.Meta)
		for _, f := range d.Fields {
			if f.Name == nil || f.Name.Name == "_" {
				continue
			}
			v := *f
			v.Doc, v.Meta = nil, nil
			sym.Members = append(sym.Members, DocMember{Group: "Variants", Signature: v.String(),
				Meta: gnode.MetaCode(f.Meta), Doc: gadDocText(f.Doc)})
		}
	case *gnode.TypedArrayTypeStmt:
		sig := *d
		sig.Doc, sig.Meta, sig.Body, sig.NameExpr = nil, nil, nil, gnode.EIdent("", d.NameExpr.Pos())
		sym.Signature = " " + strings.TrimSpace(strings.TrimPrefix(sig.String(), "type "))
		sym.Meta = gnode.MetaCode(d.Meta)
		if d.Body != nil {
			sym.Members = classDocMembers(d.Body)
		}
	case *gnode.DeclStmt:
		if _, val := declPreludeValue(d); val != nil {
			if u, ok := val.(*gnode.TypeUnionExpr); ok {
				sym.Signature = " " + u.String()
			} else {
				sym.Signature = " = " + val.String()
			}
		}
	}
}

// classDocMembers lists a class-like body's members (fields, properties,
// constructors, methods) with their metadata tags and docs.
func classDocMembers(e *gnode.TypeLitExpr) (ms []DocMember) {
	for _, f := range e.Fields {
		sig := f.Name.String()
		if f.Value != nil {
			sig += " = " + f.Value.String()
		}
		ms = append(ms, DocMember{Group: "Fields", Signature: sig, Meta: gnode.MetaCode(f.Meta), Doc: gadDocText(f.Doc)})
	}
	add := func(group string, members []*gnode.ClassMemberExpr) {
		for _, m := range members {
			name := ""
			if id, _ := m.NameExpr.(*gnode.IdentExpr); id != nil {
				name = id.Name
			}
			for _, fm := range m.Methods {
				doc := gadDocText(fm.Doc)
				if doc == "" {
					doc = gadDocText(m.Doc)
				}
				ms = append(ms, DocMember{Group: group,
					Signature: name + fm.Params.String() + gnode.FormatFuncReturn(fm.Return),
					Meta:      gnode.MetaCode(m.Meta), Doc: doc})
			}
		}
	}
	add("Properties", e.Props)
	for _, fm := range e.New {
		ms = append(ms, DocMember{Group: "Constructors",
			Signature: "new" + fm.Params.String(), Doc: gadDocText(fm.Doc)})
	}
	add("Methods", e.Methods)
	return
}

// interfaceDocMembers lists an interface's requirements (fields, accessors,
// methods) with their metadata tags and docs.
func interfaceDocMembers(e *gnode.InterfaceExpr) (ms []DocMember) {
	for _, m := range e.Members {
		v := *m
		v.Doc, v.Meta = nil, nil
		ms = append(ms, DocMember{Group: "Required", Signature: v.String(), Meta: gnode.MetaCode(m.Meta), Doc: gadDocText(m.Doc)})
	}
	for _, m := range e.Methods {
		v := *m
		v.Doc, v.Meta = nil, nil
		ms = append(ms, DocMember{Group: "Required", Signature: v.String(), Meta: gnode.MetaCode(m.Meta), Doc: gadDocText(m.Doc)})
	}
	return
}

// declPreludeValue returns the name and value of a const/var prelude declaration.
func declPreludeValue(ds *gnode.DeclStmt) (*gnode.IdentExpr, gnode.Expr) {
	gd, _ := ds.Decl.(*gnode.GenDecl)
	if gd == nil || len(gd.Specs) == 0 {
		return nil, nil
	}
	vs, _ := gd.Specs[0].(*gnode.ValueSpec)
	if vs == nil || len(vs.Idents) == 0 || len(vs.Values) == 0 {
		return nil, nil
	}
	return vs.Idents[0], vs.Values[0]
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
	if fm, ok := es.ValueExpr.(*gnode.FuncWithMethodsExpr); ok {
		if id := fm.NameIdent(); id != nil {
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
			comps = append(comps, gadxSym(f, t.Name, gadxParams(t.ParamsRaw), t.Doc, t.Pos()))
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
	add("Public API", exports)
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
