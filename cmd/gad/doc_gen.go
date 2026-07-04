// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package main

import (
	"path/filepath"
	"strconv"
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

// docEntryKind classifies an entry into the Constants, Variables or Types
// section.
type docEntryKind int

const (
	docConst docEntryKind = iota
	docVar
	docType
)

// docBuckets groups documented entries into the Constants, Variables and Types
// sections.
type docBuckets struct {
	consts, vars, types []docEntry
}

// bucketize splits entries by kind into the Constants, Variables and Types
// sections.
func bucketize(entries []docEntry) (b docBuckets) {
	for _, e := range entries {
		switch e.kind {
		case docVar:
			b.vars = append(b.vars, e)
		case docType:
			b.types = append(b.types, e)
		default:
			b.consts = append(b.consts, e)
		}
	}
	return b
}

// empty reports whether all sections are empty.
func (b docBuckets) empty() bool {
	return len(b.consts) == 0 && len(b.vars) == 0 && len(b.types) == 0
}

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

// generateDoc renders the godoc-style Markdown for a Gad source file. It is a
// thin wrapper over DocGenerator.FromContent kept for the existing call sites and
// tests; FromContent holds the rendering pipeline.
func generateDoc(path string, src []byte, mustExported bool) (string, error) {
	return (&DocGenerator{MustExported: mustExported}).FromContent(path, src)
}

// internalEntries gathers the documented non-exported top-level declarations of
// file. A declaration is documented when a doc comment (`///`, `/**` or `/***`)
// ends on the line immediately above it; const/var blocks use their per-spec doc
// comments directly. `met`/`prop`-via-`met` extensions are not documented (their
// names are ambiguous), so doc comments on them are skipped.
func internalEntries(file *parser.File, f *source.File) (entries []docEntry) {
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
		entries = append(entries, e)
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
	return entries
}

// testEntries gathers the file's `test NAME { … }` and `bench NAME { … }`
// statements into documentation entries (name + doc comment), split into tests
// and benchmarks in source order. Nested `test`s (subtests) are included with a
// parent/child qualified name. Unlike API declarations these are listed even
// without a doc comment, since a test is itself documentation of behaviour.
func testEntries(file *parser.File) (tests, benches []docEntry) {
	var walk func(stmts []node.Stmt, prefix string)
	walk = func(stmts []node.Stmt, prefix string) {
		for _, stmt := range stmts {
			ts, ok := stmt.(*node.TestStmt)
			if !ok {
				continue
			}
			name := ts.Name
			if prefix != "" {
				name = prefix + "/" + name
			}
			e := docEntry{
				name:    name,
				keyword: ts.Kind.String(),
				code:    []string{ts.Kind.String() + " " + testDisplayName(ts) + " { … }"},
				doc:     docContent(ts.Doc),
			}
			if ts.Kind == node.TestKindBench {
				benches = append(benches, e)
			} else {
				tests = append(tests, e)
			}
			if ts.Body != nil {
				walk(ts.Body.Stmts, name) // nested subtests, depth-first
			}
		}
	}
	walk(file.Stmts, "")
	return tests, benches
}

// testDisplayName renders a test/bench name as written in source: quoted when it
// was a string literal (or is not a bare identifier), bare otherwise.
func testDisplayName(ts *node.TestStmt) string {
	if ts.Quoted {
		return strconv.Quote(ts.Name)
	}
	return ts.Name
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
	case *node.MethodInterfaceStmt:
		name := identName(s.NameExpr)
		if name == "" {
			return docEntry{}, false
		}
		return docEntry{name: name, kind: docType, keyword: "meti", code: []string{"meti " + name}, doc: doc}, true
	case *node.FuncStmt:
		name := funcName(s.Func)
		if name == "" {
			return docEntry{}, false
		}
		return funcEntry(name, s.Func, doc), true
	case *node.TestStmt:
		// `test`/`bench` statements are not API declarations; they are collected
		// separately (testEntries) into their own Tests/Benchs sections, so they
		// must not appear in the Constants/Variables/Types buckets here.
		return docEntry{}, false
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
	case *node.MethodInterfaceExpr:
		return docEntry{name: name, kind: docType, keyword: "meti",
			code: []string{name + " = " + firstLine(v.String())}, doc: doc}
	default:
		// A `:=` binding to a plain value is a (mutable) variable.
		return docEntry{name: name, kind: docVar, keyword: "var",
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
	kind := docConst
	if kw != "const" { // `var` declarations are variables
		kind = docVar
	}
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
			out = append(out, docEntry{name: name, kind: kind, keyword: kw, code: []string{code}, doc: dc})
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

// writeTOC writes a flat table of contents (Constants/Variables/Types) for the
// must-exported layout.
func writeTOC(b *strings.Builder, bk docBuckets, tests, benches []docEntry) {
	if bk.empty() && len(tests) == 0 && len(benches) == 0 {
		return
	}
	b.WriteString("\n## Table of Contents\n\n")
	writeTOCSection(b, "Constants", bk.consts)
	writeTOCSection(b, "Variables", bk.vars)
	writeTOCSection(b, "Types", bk.types)
	writeTOCSection(b, "Tests", tests)
	writeTOCSection(b, "Benchs", benches)
}

// writeTOCSection writes a top-level TOC bullet linking the section and its
// entries, or nothing when empty.
func writeTOCSection(b *strings.Builder, title string, entries []docEntry) {
	if len(entries) == 0 {
		return
	}
	b.WriteString("- [" + title + "](#" + anchor(title) + ")\n")
	for _, e := range entries {
		b.WriteString("  - [" + e.name + "](#" + anchor(e.name) + ")\n")
	}
}

// writeRootGroup writes a root section ("Exported"/"Internal") with Constants,
// Variables and Types subsections, or nothing when all are empty.
func writeRootGroup(b *strings.Builder, group string, bk docBuckets) {
	if bk.empty() {
		return
	}
	b.WriteString("\n## " + group + "\n")
	writeSection(b, 3, "Constants", bk.consts)
	writeSection(b, 3, "Variables", bk.vars)
	writeTypesSection(b, 3, bk.types)
}

// writeGroupedTOC writes a table of contents for the two-root-section layout,
// listing each entry name nested under its Exported/Internal group.
func writeGroupedTOC(b *strings.Builder, exp, internal docBuckets, tests, benches []docEntry) {
	if exp.empty() && internal.empty() && len(tests) == 0 && len(benches) == 0 {
		return
	}
	b.WriteString("\n## Table of Contents\n\n")
	writeTOCGroup(b, "Exported", exp)
	writeTOCGroup(b, "Internal", internal)
	writeTOCSection(b, "Tests", tests)
	writeTOCSection(b, "Benchs", benches)
}

// writeTOCGroup writes one Exported/Internal group of the grouped TOC.
func writeTOCGroup(b *strings.Builder, group string, bk docBuckets) {
	if bk.empty() {
		return
	}
	b.WriteString("- [" + group + "](#" + anchor(group) + ")\n")
	for _, entries := range [][]docEntry{bk.consts, bk.vars, bk.types} {
		for _, e := range entries {
			b.WriteString("  - [" + e.name + "](#" + anchor(e.name) + ")\n")
		}
	}
}

// writeSection writes a flat section (e.g. Constants) at the given heading
// level, rendering each entry one level deeper.
func writeSection(b *strings.Builder, level int, title string, entries []docEntry) {
	if len(entries) == 0 {
		return
	}
	b.WriteString("\n" + strings.Repeat("#", level) + " " + title + "\n")
	for _, e := range entries {
		writeEntry(b, level+1, e)
	}
}

// typeGroups orders the kind subsections of the Types section and maps each to
// the entry keyword(s) it collects.
var typeGroups = []struct {
	title    string
	keywords []string
}{
	{"Functions", []string{"func"}},
	{"Classes", []string{"class"}},
	{"Enums", []string{"enum"}},
	{"Methods", []string{"met"}},
	{"Properties", []string{"prop"}},
	{"Interfaces", []string{"meti"}},
}

// writeTypesSection writes the Types section at the given heading level, grouping
// its entries into kind subsections (Functions, Classes, Enums, Methods,
// Properties, Interfaces) one level deeper, with entries another level deeper.
// Entries whose keyword matches no known group are collected under "Other" so
// nothing is dropped.
func writeTypesSection(b *strings.Builder, level int, entries []docEntry) {
	if len(entries) == 0 {
		return
	}
	sh := strings.Repeat("#", level+1) // subsection heading
	b.WriteString("\n" + strings.Repeat("#", level) + " Types\n")

	claimed := make([]bool, len(entries))
	writeGroup := func(title string, match func(e docEntry) bool) {
		var group []docEntry
		for i, e := range entries {
			if !claimed[i] && match(e) {
				claimed[i] = true
				group = append(group, e)
			}
		}
		if len(group) == 0 {
			return
		}
		b.WriteString("\n" + sh + " " + title + "\n")
		for _, e := range group {
			writeEntry(b, level+2, e)
		}
	}

	for _, g := range typeGroups {
		kws := g.keywords
		writeGroup(g.title, func(e docEntry) bool {
			for _, kw := range kws {
				if e.keyword == kw {
					return true
				}
			}
			return false
		})
	}
	writeGroup("Other", func(docEntry) bool { return true })
}

// writeEntry renders one documented entry: its `keyword **name**` heading at the
// given level, then its signature/value code and doc (or its methods).
func writeEntry(b *strings.Builder, level int, e docEntry) {
	b.WriteString("\n" + strings.Repeat("#", level) + " " + e.keyword + " **" + e.name + "**\n")
	if len(e.methods) > 0 {
		// func-with-methods: the func-level doc introduces the methods.
		if e.doc != "" {
			b.WriteString("\n" + e.doc + "\n")
		}
		writeMethods(b, e.methods)
		return
	}
	writeCode(b, e.code)
	if e.doc != "" {
		b.WriteString("\n" + e.doc + "\n")
	}
}

// writeCode renders signature/value lines as a fenced ```gad code block (so they
// are highlighted as Gad), or nothing when there are no lines.
func writeCode(b *strings.Builder, lines []string) {
	if len(lines) == 0 {
		return
	}
	b.WriteString("\n```gad\n")
	for _, line := range lines {
		b.WriteString(line + "\n")
	}
	b.WriteString("```\n")
}

// writeMethods renders a func-with-methods body: the first method is the default
// (signature + doc); any remaining methods follow under "other methods".
func writeMethods(b *strings.Builder, methods []docMethod) {
	writeMethod := func(m docMethod) {
		writeCode(b, []string{m.sig})
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
