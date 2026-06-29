// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package main

import (
	"path/filepath"
	"strings"

	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
)

// moduleName derives the documentation module name from a source path: its base
// name without the extension.
func moduleName(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// docEntryKind classifies an exported entry into the Constants or Types section.
type docEntryKind int

const (
	docConst docEntryKind = iota
	docType
)

// docMethod is one method of a func-with-methods/prop/meti entry.
type docMethod struct {
	sig string
	doc string
}

// docEntry is one exported, documented symbol.
type docEntry struct {
	name    string
	kind    docEntryKind
	keyword string      // "const", "func", "met", "prop", "meti" or ""
	code    []string    // signature/value lines shown in a code block
	doc     string      // rendered Markdown doc body
	methods []docMethod // for func-with-methods: rendered as default + others
}

// generateDoc renders the godoc-style Markdown for a Gad source file: the module
// heading, any ROOT_BLOCK (`/***`) prose, then Constants and Types sections for
// the documented exported symbols. The file is parsed with comments so doc
// comments are attached to their nodes.
func generateDoc(path string, src []byte) (string, error) {
	fs := source.NewFileSet()
	f := fs.AddFileData(path, -1, src)
	file, err := parser.NewParserWithOptions(
		f, &parser.ParserOptions{Mode: parser.ParseComments}, nil).ParseFile()
	if err != nil {
		return "", err
	}

	var consts, types []docEntry
	for _, stmt := range file.Stmts {
		es, _ := stmt.(*node.ExportStmt)
		if es == nil {
			continue
		}
		for _, e := range exportEntries(es) {
			if e.kind == docConst {
				consts = append(consts, e)
			} else {
				types = append(types, e)
			}
		}
	}

	var b strings.Builder
	b.WriteString("# " + moduleName(path) + "\n")

	for _, root := range rootBlocks(file.Comments) {
		b.WriteString("\n" + root + "\n")
	}

	writeTOC(&b, consts, types)
	writeSection(&b, "Constants", consts)
	writeSection(&b, "Types", types)
	return b.String(), nil
}

// exportEntries extracts the documented entries from a single export statement.
func exportEntries(es *node.ExportStmt) []docEntry {
	doc := docContent(es.Doc)

	switch v := es.ValueExpr.(type) {
	case *node.FuncExpr:
		return []docEntry{funcEntry(funcName(v), v, doc)}
	case *node.FuncWithMethodsExpr:
		return []docEntry{methodsEntry(identName(v.NameExpr), v, doc)}
	case *node.DictExpr:
		var out []docEntry
		for _, el := range v.Elements {
			if el.Key == nil {
				continue // spread
			}
			out = append(out, dictEntry(el, doc))
		}
		return out
	}

	// export IDENT [= value]
	name := identName(es.KeyExpr)
	if name == "" {
		return nil
	}
	e := docEntry{name: name, kind: docConst, keyword: "const", doc: doc}
	if es.ValueExpr != nil {
		e.code = []string{"const " + name + " = " + es.ValueExpr.String()}
	} else {
		e.code = []string{"const " + name}
	}
	return []docEntry{e}
}

// funcEntry builds a Types entry for a single-signature function.
func funcEntry(name string, fe *node.FuncExpr, doc string) docEntry {
	kw := "func"
	if fe.Type != nil && fe.Type.FuncPos != source.NoPos && len(fe.Type.Token.Literal) > 0 {
		kw = fe.Type.Token.Literal
	}
	sig := name
	if fe.Type != nil {
		sig = fe.Type.FuncHeader.String()
	}
	return docEntry{name: name, kind: docType, keyword: kw, code: []string{sig}, doc: doc}
}

// methodsEntry builds a Types entry for a func-with-methods value.
func methodsEntry(name string, e *node.FuncWithMethodsExpr, doc string) docEntry {
	methods := make([]docMethod, 0, len(e.Methods))
	for _, m := range e.Methods {
		methods = append(methods, docMethod{
			sig: m.Params.String() + node.FormatFuncReturn(m.Return),
			doc: docContent(m.Doc),
		})
	}
	return docEntry{name: name, kind: docType, keyword: "func", doc: doc, methods: methods}
}

// dictEntry builds an entry for one `export { key: value }` member.
func dictEntry(el *node.DictElementLit, parentDoc string) docEntry {
	name := strings.Trim(el.Key.String(), `"`)
	doc := docContent(el.Doc)
	if doc == "" {
		doc = parentDoc
	}
	switch v := el.Value.(type) {
	case *node.FuncExpr:
		e := funcEntry(name, v, doc)
		e.name = name
		// An anonymous dict-value function has no name in its signature; prefix
		// the member name so the code reads `name(params)`.
		if funcName(v) == "" && len(e.code) > 0 {
			e.code[0] = name + e.code[0]
		}
		return e
	}
	return docEntry{
		name: name, kind: docConst, keyword: "const",
		code: []string{"const " + name + " = " + valueString(el.Value)},
		doc:  doc,
	}
}

func valueString(e node.Expr) string {
	if e == nil {
		return ""
	}
	return e.String()
}

func funcName(fe *node.FuncExpr) string {
	if fe != nil && fe.Type != nil && fe.Type.NameExpr != nil {
		return fe.Type.NameExpr.String()
	}
	return ""
}

func identName(e node.Expr) string {
	if e == nil {
		return ""
	}
	return e.String()
}

// rootBlocks returns the Markdown content of each ROOT_BLOCK (`/***`) comment, in
// source order.
func rootBlocks(groups []*ast.CommentGroup) []string {
	var out []string
	for _, g := range groups {
		if len(g.List) > 0 && strings.HasPrefix(g.List[0].Text, "/***") {
			if c := blockContent(g.List[0].Text, "/***", "***/"); c != "" {
				out = append(out, c)
			}
		}
	}
	return out
}

// docContent extracts the Markdown body of a doc comment group (markers
// stripped). Mirrors the parser's doc-comment forms.
func docContent(g *ast.CommentGroup) string {
	if g == nil || len(g.List) == 0 {
		return ""
	}
	first := g.List[0].Text
	switch {
	case strings.HasPrefix(first, "/***"):
		return blockContent(first, "/***", "***/")
	case strings.HasPrefix(first, "/**"):
		return blockContent(first, "/**", "**/")
	case strings.HasPrefix(first, "///") && !strings.HasPrefix(first, "////"):
		lines := make([]string, len(g.List))
		for i, c := range g.List {
			lines[i] = strings.TrimPrefix(strings.TrimPrefix(c.Text, "///"), " ")
		}
		return strings.Join(lines, "\n")
	}
	return ""
}

// blockContent returns the inner text of a fenced block doc, dropping the
// opening and closing fence.
func blockContent(text, open, close string) string {
	body := strings.TrimPrefix(text, open)
	body = strings.TrimSuffix(body, close)
	return strings.Trim(body, "\n")
}

// writeTOC writes a table of contents for the non-empty sections.
func writeTOC(b *strings.Builder, consts, types []docEntry) {
	if len(consts) == 0 && len(types) == 0 {
		return
	}
	b.WriteString("\n## Table of Contents\n\n")
	if len(consts) > 0 {
		b.WriteString("- [Constants](#constants)\n")
		for _, e := range consts {
			b.WriteString("  - [" + e.name + "](#" + anchor(e.name) + ")\n")
		}
	}
	if len(types) > 0 {
		b.WriteString("- [Types](#types)\n")
		for _, e := range types {
			b.WriteString("  - [" + e.name + "](#" + anchor(e.name) + ")\n")
		}
	}
}

// writeSection writes a Constants/Types section with one subsection per entry.
func writeSection(b *strings.Builder, title string, entries []docEntry) {
	if len(entries) == 0 {
		return
	}
	b.WriteString("\n## " + title + "\n")
	for _, e := range entries {
		b.WriteString("\n### " + e.keyword + " **" + e.name + "**\n")
		if len(e.methods) > 0 {
			// func-with-methods: the func-level doc introduces the methods.
			if e.doc != "" {
				b.WriteString("\n" + e.doc + "\n")
			}
			writeMethods(b, e.methods)
			continue
		}
		b.WriteString("\n")
		for _, line := range e.code {
			b.WriteString("    " + line + "\n")
		}
		if e.doc != "" {
			b.WriteString("\n" + e.doc + "\n")
		}
	}
}

// writeMethods renders a func-with-methods body: the first method is the default
// (signature + doc); any remaining methods follow under "other methods".
func writeMethods(b *strings.Builder, methods []docMethod) {
	writeMethod := func(m docMethod) {
		b.WriteString("\n    " + m.sig + "\n")
		if m.doc != "" {
			b.WriteString("\n" + m.doc + "\n")
		}
	}
	writeMethod(methods[0])
	if len(methods) > 1 {
		b.WriteString("\n**other methods**\n")
		for _, m := range methods[1:] {
			writeMethod(m)
		}
	}
}

// anchor builds a GitHub-style heading anchor from a name.
func anchor(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "-"))
}
