package main

import (
	"fmt"
	"strings"

	gadxnode "github.com/gad-lang/gad/gadx/node"
	gadxparser "github.com/gad-lang/gad/gadx/parser"
	gnode "github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
)

// gadxDoc renders Markdown documentation for a .gadx template. It documents the
// template's public surface — its top-level components (`@comp`) and functions
// (`@func`) — using the `/** … */` doc comment attached to each. Every symbol
// heading carries a `data-source-pos="LINE,COLUMN"` anchor pointing at the
// declaration, so a viewer can navigate from the docs back to the source.
//
// A leading file-level block comment (before any statement) becomes the module
// prose, mirroring gad's `/***` ROOT_BLOCK.
func gadxDoc(path string, src []byte) (string, error) {
	fs := source.NewFileSet()
	f := fs.AddFileData(path, -1, src)
	file, err := gadxparser.NewParser(f).ParseFile()
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString("# " + moduleName(path) + "\n")

	if prose := gadxLeadProse(file); prose != "" {
		b.WriteString("\n" + prose + "\n")
	}

	// Document the top-level API: exports, components, functions, parameters,
	// constants, variables and enums.
	var exports, comps, funcs, params, consts, vars, enums []gadxDocEntry
	for _, stmt := range file.Stmts {
		switch d := stmt.(type) {
		case *gadxnode.ExportStmt:
			exports = append(exports, gadxEntry(f, d.Name, gadxExportValue(d.Value), d.Doc, d.Pos()))
		case *gadxnode.CompDecl:
			comps = append(comps, gadxEntry(f, "+"+d.Name, gadxParams(d.ParamsRaw), d.Doc, d.Pos()))
		case *gadxnode.FuncDecl:
			funcs = append(funcs, gadxEntry(f, d.Name, gadxParams(d.ParamsRaw), d.Doc, d.Pos()))
		case *gadxnode.ParamStmt:
			params = append(params, gadxEntry(f, "@param", gadxDeclSig(d.Decl, "param"), d.Doc, d.Pos()))
		case *gadxnode.ConstStmt:
			consts = append(consts, gadxEntry(f, "@const", gadxVarSig(d.Decls), d.Doc, d.Pos()))
		case *gadxnode.VarStmt:
			vars = append(vars, gadxEntry(f, "@var", gadxVarSig(d.Decls), d.Doc, d.Pos()))
		case *gadxnode.EnumStmt:
			enums = append(enums, gadxEntry(f, d.Name, "", d.Doc, d.Pos()))
		}
	}

	gadxSection(&b, "Exports", exports)
	gadxSection(&b, "Components", comps)
	gadxSection(&b, "Functions", funcs)
	gadxSection(&b, "Parameters", params)
	gadxSection(&b, "Constants", consts)
	gadxSection(&b, "Variables", vars)
	gadxSection(&b, "Enums", enums)
	return b.String(), nil
}

// gadxExportValue renders the ` = value` suffix for a documented @export, or ""
// when the export has no value expression (`@export name`).
func gadxExportValue(v gnode.Expr) string {
	if v == nil {
		return ""
	}
	return " = " + v.String()
}

// gadxDeclSig renders a gad GenDecl's signature (e.g. `param (a, b; c=1)`),
// stripping the leading keyword so it reads as a parenthesized parameter list.
func gadxDeclSig(decl *gnode.GenDecl, keyword string) string {
	if decl == nil {
		return ""
	}
	s := strings.TrimSpace(decl.String())
	s = strings.TrimSpace(strings.TrimPrefix(s, keyword))
	return s
}

// gadxVarSig renders a `name = init` list for @var/@const declarations.
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

// gadxDocEntry is one documented gadx symbol.
type gadxDocEntry struct {
	name   string // display name (comp names are prefixed with `+`)
	params string // parenthesized parameter signature, or ""
	doc    string // attached /** … */ doc text
	line   int
	column int
}

func gadxEntry(f *source.File, name, params, doc string, pos source.Pos) gadxDocEntry {
	fp := source.MustFilePosition(f, pos)
	return gadxDocEntry{name: name, params: params, doc: doc, line: fp.Line, column: fp.Column}
}

// gadxParams normalizes a raw parameter string into a parenthesized signature,
// or "" when there are no parameters.
func gadxParams(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "(")
	raw = strings.TrimSuffix(raw, ")")
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	return "(" + raw + ")"
}

// gadxSection writes a `## Title` section with one `### symbol` entry per doc
// entry. Each heading embeds a data-source-pos anchor for source navigation.
func gadxSection(b *strings.Builder, title string, entries []gadxDocEntry) {
	if len(entries) == 0 {
		return
	}
	fmt.Fprintf(b, "\n## %s\n", title)
	for _, e := range entries {
		fmt.Fprintf(b, "\n### <span data-source-pos=\"%d,%d\">%s</span>%s\n",
			e.line, e.column, e.name, e.params)
		if doc := strings.TrimSpace(e.doc); doc != "" {
			b.WriteString("\n" + doc + "\n")
		}
	}
}

// gadxLeadProse returns the text of a file-level block/line comment that leads
// the file (before any renderable statement), to use as module prose. Returns ""
// when the first statement is not a comment.
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
