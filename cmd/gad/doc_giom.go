package main

import (
	"fmt"
	"strings"

	giomnode "github.com/gad-lang/gad/giom/node"
	giomparser "github.com/gad-lang/gad/giom/parser"
	gnode "github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
)

// giomDoc renders Markdown documentation for a .giom template. It documents the
// template's public surface — its top-level components (`@comp`) and functions
// (`@func`) — using the `/** … */` doc comment attached to each. Every symbol
// heading carries a `data-source-pos="LINE,COLUMN"` anchor pointing at the
// declaration, so a viewer can navigate from the docs back to the source.
//
// A leading file-level block comment (before any statement) becomes the module
// prose, mirroring gad's `/***` ROOT_BLOCK.
func giomDoc(path string, src []byte) (string, error) {
	fs := source.NewFileSet()
	f := fs.AddFileData(path, -1, src)
	file, err := giomparser.NewParser(f).ParseFile()
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString("# " + moduleName(path) + "\n")

	if prose := giomLeadProse(file); prose != "" {
		b.WriteString("\n" + prose + "\n")
	}

	// Document the top-level API: components, functions, parameters, constants,
	// variables and enums.
	var comps, funcs, params, consts, vars, enums []giomDocEntry
	for _, stmt := range file.Stmts {
		switch d := stmt.(type) {
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

	giomSection(&b, "Components", comps)
	giomSection(&b, "Functions", funcs)
	giomSection(&b, "Parameters", params)
	giomSection(&b, "Constants", consts)
	giomSection(&b, "Variables", vars)
	giomSection(&b, "Enums", enums)
	return b.String(), nil
}

// giomDeclSig renders a gad GenDecl's signature (e.g. `param (a, b; c=1)`),
// stripping the leading keyword so it reads as a parenthesized parameter list.
func giomDeclSig(decl *gnode.GenDecl, keyword string) string {
	if decl == nil {
		return ""
	}
	s := strings.TrimSpace(decl.String())
	s = strings.TrimSpace(strings.TrimPrefix(s, keyword))
	return s
}

// giomVarSig renders a `name = init` list for @var/@const declarations.
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

// giomDocEntry is one documented giom symbol.
type giomDocEntry struct {
	name   string // display name (comp names are prefixed with `+`)
	params string // parenthesized parameter signature, or ""
	doc    string // attached /** … */ doc text
	line   int
	column int
}

func giomEntry(f *source.File, name, params, doc string, pos source.Pos) giomDocEntry {
	fp := source.MustFilePosition(f, pos)
	return giomDocEntry{name: name, params: params, doc: doc, line: fp.Line, column: fp.Column}
}

// giomParams normalizes a raw parameter string into a parenthesized signature,
// or "" when there are no parameters.
func giomParams(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "(")
	raw = strings.TrimSuffix(raw, ")")
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	return "(" + raw + ")"
}

// giomSection writes a `## Title` section with one `### symbol` entry per doc
// entry. Each heading embeds a data-source-pos anchor for source navigation.
func giomSection(b *strings.Builder, title string, entries []giomDocEntry) {
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

// giomLeadProse returns the text of a file-level block/line comment that leads
// the file (before any renderable statement), to use as module prose. Returns ""
// when the first statement is not a comment.
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
