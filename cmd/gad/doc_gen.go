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
// heading, any ROOT_BLOCK (`/***`) prose, then the documented symbols. The file
// is parsed with comments so doc comments are attached to their nodes.
//
// When mustExported is true only the exported symbols are documented, in
// top-level Constants and Types sections. When it is false (the default) the
// documented internal (non-exported) declarations are included too, and the
// output is split into two root sections, "Exported" and "Internal", each with
// its own Constants and Types subsections.
func generateDoc(path string, src []byte, mustExported bool) (string, error) {
	fs := source.NewFileSet()
	f := fs.AddFileData(path, -1, src)
	file, err := parser.NewParserWithOptions(
		f, &parser.ParserOptions{Mode: parser.ParseComments}, nil).ParseFile()
	if err != nil {
		return "", err
	}

	var expConsts, expTypes []docEntry
	for _, stmt := range file.Stmts {
		es, _ := stmt.(*node.ExportStmt)
		if es == nil {
			continue
		}
		for _, e := range exportEntries(es) {
			if e.kind == docConst {
				expConsts = append(expConsts, e)
			} else {
				expTypes = append(expTypes, e)
			}
		}
	}

	var b strings.Builder
	b.WriteString("# " + moduleName(path) + "\n")

	for _, root := range rootBlocks(file.Comments) {
		b.WriteString("\n" + root + "\n")
	}

	if mustExported {
		writeTOC(&b, expConsts, expTypes)
		writeSection(&b, 2, "Constants", expConsts)
		writeSection(&b, 2, "Types", expTypes)
		return b.String(), nil
	}

	// Two-root-section mode: gather the documented internal declarations and
	// render Exported + Internal groups.
	intConsts, intTypes := internalEntries(file, f)
	writeGroupedTOC(&b, expConsts, expTypes, intConsts, intTypes)
	writeRootGroup(&b, "Exported", expConsts, expTypes)
	writeRootGroup(&b, "Internal", intConsts, intTypes)
	return b.String(), nil
}

// internalEntries gathers the documented non-exported top-level declarations of
// file. A declaration is documented when a doc comment (`///`, `/**` or `/***`)
// ends on the line immediately above it; const/var blocks use their per-spec doc
// comments directly. `met`/`prop`-via-`met` extensions are not documented (their
// names are ambiguous), so doc comments on them are skipped.
func internalEntries(file *parser.File, f *source.File) (consts, types []docEntry) {
	// Index doc comments by the line they END on, so a declaration on the next
	// line can find the doc immediately above it.
	docByEnd := map[int]*ast.CommentGroup{}
	for _, g := range file.Comments {
		if docContent(g) == "" {
			continue // ordinary // or /* */ comment, not a doc comment
		}
		docByEnd[source.MustFileLine(f, g.End())] = g
	}

	add := func(e docEntry) {
		if e.name == "" {
			return
		}
		if e.kind == docConst {
			consts = append(consts, e)
		} else {
			types = append(types, e)
		}
	}

	for _, stmt := range file.Stmts {
		if _, ok := stmt.(*node.ExportStmt); ok {
			continue // covered by the Exported section
		}
		// const/var blocks carry per-spec docs on the AST nodes themselves.
		if ds, ok := stmt.(*node.DeclStmt); ok {
			for _, e := range declStmtEntries(ds) {
				add(e)
			}
			continue
		}
		startLine := source.MustFileLine(f, stmt.Pos())
		doc := docContent(docByEnd[startLine-1])
		if doc == "" {
			continue // only documented internals are listed
		}
		if e, ok := internalStmtEntry(stmt, doc); ok {
			add(e)
		}
	}
	return consts, types
}

// internalStmtEntry builds a doc entry for a single documented internal
// statement (everything except const/var blocks, handled separately). The
// boolean is false for statement forms that do not introduce a named, documented
// declaration.
func internalStmtEntry(stmt node.Stmt, doc string) (docEntry, bool) {
	switch s := stmt.(type) {
	case *node.AssignStmt:
		if len(s.LHS) != 1 || len(s.RHS) != 1 {
			return docEntry{}, false
		}
		id, ok := s.LHS[0].(*node.IdentExpr)
		if !ok {
			return docEntry{}, false
		}
		return assignEntry(id.String(), s.RHS[0], doc), true
	case *node.FuncWithMethodsStmt:
		name := identName(s.NameExpr)
		if name == "" {
			return docEntry{}, false
		}
		return methodsEntry(name, &s.FuncWithMethodsExpr, doc), true
	case *node.ClassStmt:
		name := identName(s.NameExpr)
		if name == "" {
			return docEntry{}, false
		}
		return docEntry{name: name, kind: docType, keyword: "class", code: []string{"class " + name}, doc: doc}, true
	case *node.EnumStmt:
		name := identName(s.NameExpr)
		if name == "" {
			return docEntry{}, false
		}
		return docEntry{name: name, kind: docType, keyword: "enum", code: []string{"enum " + name}, doc: doc}, true
	case *node.PropStmt:
		name := identName(s.NameExpr)
		if name == "" {
			return docEntry{}, false
		}
		return docEntry{name: name, kind: docType, keyword: "prop", code: []string{"prop " + name}, doc: doc}, true
	case *node.FuncStmt:
		name := funcName(s.Func)
		if name == "" {
			return docEntry{}, false
		}
		return funcEntry(name, s.Func, doc), true
	case *node.ExprStmt:
		if fe, ok := s.Expr.(*node.FuncExpr); ok {
			name := funcName(fe)
			if name == "" {
				return docEntry{}, false
			}
			return funcEntry(name, fe, doc), true
		}
	}
	return docEntry{}, false
}

// assignEntry builds an internal entry for a `name := value` (or `name = value`)
// declaration, classifying callable/type-producing right-hand sides as Types and
// everything else as Constants.
func assignEntry(name string, rhs node.Expr, doc string) docEntry {
	switch v := rhs.(type) {
	case *node.FuncExpr:
		e := funcEntry(name, v, doc)
		e.name = name
		// An anonymous func value has no name in its signature; prefix the bound
		// name so the code reads `name(params)`.
		if funcName(v) == "" && len(e.code) > 0 {
			e.code[0] = name + e.code[0]
		}
		return e
	case *node.FuncWithMethodsExpr:
		return methodsEntry(name, v, doc)
	case *node.ClosureExpr:
		return docEntry{name: name, kind: docType, keyword: "func",
			code: []string{name + " = " + firstLine(v.String())}, doc: doc}
	case *node.ClassExpr:
		return docEntry{name: name, kind: docType, keyword: "class",
			code: []string{name + " = " + firstLine(v.String())}, doc: doc}
	case *node.EnumExpr:
		return docEntry{name: name, kind: docType, keyword: "enum",
			code: []string{name + " = " + firstLine(v.String())}, doc: doc}
	default:
		return docEntry{name: name, kind: docConst, keyword: "var",
			code: []string{name + " = " + firstLine(rhs.String())}, doc: doc}
	}
}

// declStmtEntries builds Constant entries for the documented specs of a const/var
// block. The block's doc comment applies to the first spec when it has none of
// its own.
func declStmtEntries(ds *node.DeclStmt) []docEntry {
	gd, ok := ds.Decl.(*node.GenDecl)
	if !ok {
		return nil
	}
	kw := gd.Tok.String()
	var out []docEntry
	for i, sp := range gd.Specs {
		vs, ok := sp.(*node.ValueSpec)
		if !ok {
			continue
		}
		grp := vs.Doc
		if i == 0 && grp == nil {
			grp = gd.Doc
		}
		dc := docContent(grp)
		if dc == "" {
			continue
		}
		for j, id := range vs.Idents {
			name := id.String()
			code := kw + " " + name
			if j < len(vs.Values) && vs.Values[j] != nil {
				code += " = " + vs.Values[j].String()
			}
			out = append(out, docEntry{name: name, kind: docConst, keyword: kw, code: []string{code}, doc: dc})
		}
	}
	return out
}

// firstLine returns the first line of s, marking truncation with an ellipsis when
// s spans multiple lines, so multi-line literals stay on one doc code line.
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimRight(s[:i], " \t") + " …"
	}
	return s
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
// opening and closing fence. A single-line block (`/** text **/`) is trimmed of
// the spaces that padded the content from the fences so it renders flush; a
// multi-line block is left as-is so embedded Markdown indentation is preserved.
func blockContent(text, open, close string) string {
	body := strings.TrimPrefix(text, open)
	body = strings.TrimSuffix(body, close)
	body = strings.Trim(body, "\n")
	if !strings.Contains(body, "\n") {
		return strings.TrimSpace(body)
	}
	return body
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

// writeRootGroup writes a root section ("Exported"/"Internal") with Constants and
// Types subsections, or nothing when both are empty.
func writeRootGroup(b *strings.Builder, group string, consts, types []docEntry) {
	if len(consts) == 0 && len(types) == 0 {
		return
	}
	b.WriteString("\n## " + group + "\n")
	writeSection(b, 3, "Constants", consts)
	writeSection(b, 3, "Types", types)
}

// writeGroupedTOC writes a table of contents for the two-root-section layout,
// listing each entry name nested under its Exported/Internal group.
func writeGroupedTOC(b *strings.Builder, expConsts, expTypes, intConsts, intTypes []docEntry) {
	if len(expConsts)+len(expTypes)+len(intConsts)+len(intTypes) == 0 {
		return
	}
	b.WriteString("\n## Table of Contents\n\n")
	writeTOCGroup(b, "Exported", expConsts, expTypes)
	writeTOCGroup(b, "Internal", intConsts, intTypes)
}

// writeTOCGroup writes one Exported/Internal group of the grouped TOC.
func writeTOCGroup(b *strings.Builder, group string, consts, types []docEntry) {
	if len(consts) == 0 && len(types) == 0 {
		return
	}
	b.WriteString("- [" + group + "](#" + anchor(group) + ")\n")
	for _, e := range consts {
		b.WriteString("  - [" + e.name + "](#" + anchor(e.name) + ")\n")
	}
	for _, e := range types {
		b.WriteString("  - [" + e.name + "](#" + anchor(e.name) + ")\n")
	}
}

// writeSection writes a Constants/Types section at the given heading level (its
// entries one level deeper), with one subsection per entry.
func writeSection(b *strings.Builder, level int, title string, entries []docEntry) {
	if len(entries) == 0 {
		return
	}
	h := strings.Repeat("#", level)
	eh := h + "#"
	b.WriteString("\n" + h + " " + title + "\n")
	for _, e := range entries {
		b.WriteString("\n" + eh + " " + e.keyword + " **" + e.name + "**\n")
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
