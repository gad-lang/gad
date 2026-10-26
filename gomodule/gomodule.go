// Package gomodule makes a module written in gad a builtin module of a Go
// package (issue #17; the `gad gomodule` command, gad's wasm `gadGomodule`):
// the module compiled (`.gadb`, linked into the programs that import it),
// the package's Go source — the module's docs as `gad:doc` comments, the
// hook of a Go init —, its test and the module's API stub (`gaddoc api`'s).
package gomodule

import (
	"fmt"
	"go/format"
	"regexp"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/encoder"
	"github.com/gad-lang/gad/internal/gaddoc"
	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	gadflate "github.com/gad-lang/gad/stdlib/compress/flate"
	gadbase64 "github.com/gad-lang/gad/stdlib/encoding/base64"
	gadfmt "github.com/gad-lang/gad/stdlib/fmt"
	gadjson "github.com/gad-lang/gad/stdlib/json"
	gadpath "github.com/gad-lang/gad/stdlib/path"
	gadreflect "github.com/gad-lang/gad/stdlib/reflect"
	gadstrings "github.com/gad-lang/gad/stdlib/strings"
	gadtest "github.com/gad-lang/gad/stdlib/test"
	gadtime "github.com/gad-lang/gad/stdlib/time"
	"github.com/gad-lang/gad/token"
	"github.com/gad-lang/gad/web/gadbridge"
)

// Options are what Make is told.
type Options struct {
	// Module is the module's import name (a lowercase snake_case identifier).
	Module string
	// Pkg is the Go package's name (default: Module).
	Pkg string
	// SourceType is the source's dialect, as gad's formatter takes it: "gad"
	// (or ""), "gadTemplate" (or "template", "gadt"), "gadx".
	SourceType string
	// Imports are the modules the Go init imports, besides the
	// `//gad:import NAME[,NAME]` lines of Files.
	Imports []string
	// Files are the other Go sources of the package (its hand-written ones:
	// the Go init), by file name: their `//gad:import` lines, and their
	// gad:doc in the API stub.
	Files map[string][]byte
	// Generate, when set, is the command of the package's `//go:generate`
	// line (CmdPath + " gomodule …").
	Generate string
	// ModuleMap resolves the modules the source imports (default:
	// BuiltinModules); a file cannot be resolved by the host that imports the
	// module.
	ModuleMap *gad.ModuleMap
}

// Result is what Make makes.
type Result struct {
	// Bytecode is the module compiled: MODULE.gadb.
	Bytecode []byte
	// Go is the package's source (MODULE.go), Test its test (MODULE_test.go).
	Go, Test []byte
	// API is the module's API stub (MODULE.gad): `gaddoc api`'s of the package.
	API string
	// Imports are the modules the Go init imports.
	Imports []string
	// Warnings are the API emitter's.
	Warnings []string

	module string
}

// Files are the files of the package Make makes, by name: MODULE.gadb,
// MODULE.go, MODULE_test.go and the API stub MODULE.gad.
func (r *Result) Files() map[string][]byte {
	return map[string][]byte{
		r.module + ".gadb":    r.Bytecode,
		r.module + ".go":      r.Go,
		r.module + "_test.go": r.Test,
		r.module + ".gad":     []byte(r.API),
	}
}

// reModuleName is a module name: a lowercase snake_case identifier.
var reModuleName = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// reGadImport is a `//gad:import NAME[,NAME]` line of a Go file.
var reGadImport = regexp.MustCompile(`(?m)^//gad:import\s+(\S+)\s*$`)

// BuiltinModules are the builtin modules of gad that need no system (of
// helper.NewModuleMap, but http, os and filepath): what a module made a Go
// module may import, by default — the wasm of gad links no system module.
func BuiltinModules() *gad.ModuleMap {
	return gad.NewModuleMap().
		AddBuiltinModuleInit(gadtime.ModuleName, gadtime.ModuleInit).
		AddBuiltinModuleInit(gadstrings.ModuleName, gadstrings.ModuleInit).
		AddBuiltinModuleInit(gadfmt.ModuleName, gadfmt.ModuleInit).
		AddBuiltinModuleInit(gadjson.ModuleName, gadjson.ModuleInit).
		AddBuiltinModule(gadpath.ModuleName, gadpath.Module).
		AddBuiltinModuleInit(gadbase64.ModuleName, gadbase64.ModuleInit).
		AddBuiltinModuleInit(gadflate.ModuleName, gadflate.ModuleInit).
		AddBuiltinModuleInit(gadreflect.ModuleName, gadreflect.ModuleInit).
		AddBuiltinModuleInit(gadtest.ModuleName, gadtest.ModuleInit)
}

// SourceKind is the dialect of a source type (Options.SourceType).
func SourceKind(sourceType string) gad.SourceKind {
	switch sourceType {
	case "gadx":
		return gad.SourceKindGadx
	case "gadTemplate", "template", "gadt":
		return gad.SourceKindGadt
	}
	return gad.SourceKindGad
}

// Make makes src, a module written in gad, a builtin module of a Go package.
func Make(src []byte, o Options) (*Result, error) {
	if !reModuleName.MatchString(o.Module) {
		return nil, fmt.Errorf("invalid module name %q: use a lowercase snake_case identifier", o.Module)
	}
	if o.Pkg == "" {
		o.Pkg = o.Module
	}
	mm := o.ModuleMap
	if mm == nil {
		mm = BuiltinModules()
	}
	kind := SourceKind(o.SourceType)
	data, err := encoder.CompileModuleWith(o.Module, src, kind, mm, nil)
	if err != nil {
		return nil, fmt.Errorf("compile: %w", err)
	}
	imports := Imports(o.Imports, o.Files)

	var doc []string
	if kind != gad.SourceKindGadx {
		if doc, err = moduleDoc(o.Module, src, kind); err != nil {
			return nil, fmt.Errorf("docs: %w", err)
		}
	} else {
		doc = []string{"gad:doc", "# " + o.Module + " module"}
	}

	r := &Result{Bytecode: data, Imports: imports, module: o.Module}
	if r.Go, err = format.Source([]byte(goSource(&o, imports, doc))); err != nil {
		return nil, fmt.Errorf("the Go source: %w", err)
	}
	if r.Test, err = format.Source([]byte(testSource(&o))); err != nil {
		return nil, fmt.Errorf("the Go test: %w", err)
	}

	// the API stub: `gaddoc api`'s of the package
	files := map[string][]byte{o.Module + ".go": r.Go}
	for n, b := range o.Files {
		if n != o.Module+".go" && n != o.Module+"_test.go" {
			files[n] = b
		}
	}
	if r.API, r.Warnings, err = gaddoc.APIFiles(files, o.Module); err != nil {
		return nil, fmt.Errorf("the API stub: %w", err)
	}
	return r, nil
}

// Imports are the modules a Go init imports: names, and the `//gad:import`
// lines of files.
func Imports(names []string, files map[string][]byte) []string {
	set := map[string]bool{}
	for _, n := range names {
		set[n] = true
	}
	for _, b := range files {
		for _, m := range reGadImport.FindAllStringSubmatch(string(b), -1) {
			for _, name := range strings.Split(m[1], ",") {
				if name = strings.TrimSpace(name); name != "" {
					set[name] = true
				}
			}
		}
	}
	out := make([]string, 0, len(set))
	for n := range set {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// CmdPath is the package `go generate` runs gad from: of the version of
// this gad (`…/cmd/gad@VERSION`, run apart from the module of the package,
// which need not depend on gad's command), or as it is in gad's own module.
func CmdPath() string {
	const path = "github.com/gad-lang/gad/cmd/gad"
	if bi, ok := debug.ReadBuildInfo(); ok {
		if v := bi.Main.Version; bi.Main.Path == "github.com/gad-lang/gad" && v != "" && v != "(devel)" {
			return path + "@" + v
		}
	}
	return path
}

// goSource is the Go source of the package.
func goSource(o *Options, imports []string, doc []string) string {
	var b strings.Builder
	b.WriteString("// Code generated by gad gomodule; DO NOT EDIT.\n\n")
	if o.Generate != "" {
		b.WriteString("//go:generate " + o.Generate + "\n\n")
	}
	fmt.Fprintf(&b, "// Package %s is the gad module %q (made by `gad gomodule`): its gad part\n", o.Pkg, o.Module)
	b.WriteString("// compiled, embedded and linked into the programs that import it, and the\n")
	b.WriteString("// hook of a Go init of its own (Init).\n")
	fmt.Fprintf(&b, "package %s\n\n", o.Pkg)
	b.WriteString("import (\n\t_ \"embed\"\n\n\t\"github.com/gad-lang/gad\"\n\t\"github.com/gad-lang/gad/encoder\"\n)\n\n")
	for _, l := range doc {
		if l == "" {
			b.WriteString("//\n")
		} else {
			b.WriteString("// " + l + "\n")
		}
	}
	b.WriteString("\n")
	fmt.Fprintf(&b, "// ModuleName is the import name of the module.\nconst ModuleName = %q\n\n", o.Module)
	fmt.Fprintf(&b, "// Bytecode is the module compiled (`.gadb`).\n//\n//go:embed %s.gadb\nvar Bytecode []byte\n\n", o.Module)
	b.WriteString("// Imports are the modules the Go init imports (`--import`, `//gad:import`).\nvar Imports = []string{")
	for i, n := range imports {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%q", n)
	}
	b.WriteString("}\n\n")
	b.WriteString("// Init is the Go init of the module: set by a hand-written file of the\n" +
		"// package (in its init), it runs after the gad part, in the VM that imports\n" +
		"// the module, and what it returns is exported beside the gad part's exports.\n" +
		"// nil: none.\nvar Init gad.ModuleGoInit\n\n")
	b.WriteString("// Module is the importable module: `mm.Add(ModuleName, Module())`.\n" +
		"func Module() gad.Importable {\n\treturn &encoder.GoModule{Data: Bytecode, Imports: Imports, Init: Init}\n}\n\n")
	b.WriteString("// AddTo adds the module to mm.\nfunc AddTo(mm *gad.ModuleMap) *gad.ModuleMap {\n\treturn mm.Add(ModuleName, Module())\n}\n")
	return b.String()
}

// testSource is the Go test of the package: the module links against the
// default builtins and modules.
func testSource(o *Options) string {
	return strings.ReplaceAll(`// Code generated by gad gomodule; DO NOT EDIT.

package PKG

import (
	"testing"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/stdlib/helper"
)

// The module links in a program of the default builtins and modules, and its
// init runs.
func TestModuleImport(t *testing.T) {
	mm := AddTo(helper.NewModuleMap())
	res, err := gad.Compile(gad.NewSymbolTable(gad.NewBuiltins().NameSet),
		[]byte("return import(\""+ModuleName+"\")"),
		gad.CompileOptions{CompilerOptions: gad.CompilerOptions{ModuleMap: mm}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = gad.NewVM(gad.NewBuiltins().Build(), res.Bytecode).Run(); err != nil {
		t.Fatal(err)
	}
}
`, "PKG", o.Pkg)
}

// moduleDoc are the gad:doc comment lines of the module: its doc (the
// file's leading `/** … **/`), its functions (`## Functions`: signatures and
// docs) and its other exports (`## Declarations`: the declarations, their
// bodies `nil`).
func moduleDoc(module string, src []byte, kind gad.SourceKind) ([]string, error) {
	st := "gad"
	po := &parser.ParserOptions{Mode: parser.ParseComments}
	so := &parser.ScannerOptions{}
	if kind == gad.SourceKindGadt {
		st = "gadTemplate"
		po.Mode |= parser.ParseMixed
		so.Mode |= parser.ScanMixed | parser.ScanConfigDisabled
		so.MixedDelimiter = parser.DefaultMixedDelimiter
	}
	dd, err := gadbridge.ExtractDoc(string(src), st)
	if err != nil {
		return nil, err
	}
	fs := source.NewFileSet()
	file, err := parser.NewParserWithOptions(fs.AddFileData(module, -1, src), po, so).ParseFile()
	if err != nil {
		return nil, err
	}

	lines := []string{"gad:doc", "# " + module + " module"}
	if dd.Prose != "" {
		lines = append(lines, "")
		lines = append(lines, docLines(dd.Prose)...)
	}

	g := &gomoduleDocs{locals: topLevelDecls(file.Stmts), src: string(src), file: file.InputFile, comments: file.Comments}
	for _, s := range file.Stmts {
		if es, ok := s.(*node.ExportStmt); ok {
			g.export(es)
		}
	}
	// the declarations first: the functions' types name them
	if len(g.decls) > 0 {
		lines = append(lines, "", "## Declarations", "")
		lines = append(lines, g.decls...)
	}
	if len(g.funcs) > 0 {
		lines = append(lines, "", "## Functions", "")
		lines = append(lines, g.funcs...)
	}
	return lines, nil
}

// gomoduleDocs gathers the docs of a module's exports.
type gomoduleDocs struct {
	locals   map[string]node.Stmt
	src      string
	file     *source.File
	comments []*ast.CommentGroup
	funcs    []string
	decls    []string
}

// docLines are the lines of a doc, trimmed at their end, blank ones empty.
func docLines(doc string) (out []string) {
	for _, l := range strings.Split(strings.TrimSpace(doc), "\n") {
		out = append(out, strings.TrimRight(l, " \t"))
	}
	return
}

// fn documents the function name: a signature (`name(params) <ret>`, `<any>`
// when it declares none) and its doc.
func (g *gomoduleDocs) fn(name string, tps []*node.TypedIdentExpr, params node.FuncParams, ret []*node.TypedIdentExpr, doc string) {
	r := node.FormatFuncReturn(ret)
	if r == "" {
		r = " <any>"
	}
	g.funcs = append(g.funcs, name+node.FormatTypeParams(tps)+params.String()+r)
	if doc != "" {
		g.funcs = append(g.funcs, docLines(doc)...)
	}
	g.funcs = append(g.funcs, "")
}

// decl documents a declaration: its code, with its doc, in a gad fence.
func (g *gomoduleDocs) decl(code, doc string) {
	g.decls = append(g.decls, "```gad")
	if doc != "" {
		g.decls = append(g.decls, "/**")
		g.decls = append(g.decls, docLines(doc)...)
		g.decls = append(g.decls, "**/")
	}
	g.decls = append(g.decls, strings.Split(code, "\n")...)
	g.decls = append(g.decls, "```", "")
}

// export documents an export statement.
func (g *gomoduleDocs) export(es *node.ExportStmt) {
	doc := commentText(es.Doc)
	name := ""
	switch k := es.KeyExpr.(type) {
	case *node.IdentExpr:
		name = k.Name
	case *node.StrLit:
		name = k.Value()
	}
	if es.Prelude != nil {
		g.declared(name, es.Prelude, doc, es.Const)
		return
	}
	switch v := es.ValueExpr.(type) {
	case nil:
		// `export NAME`: NAME as it is declared
		if d := g.locals[name]; d != nil {
			g.declared(name, d, doc, es.Const || isConstDecl(d))
		} else {
			g.decl("export const "+name+" = nil", doc)
		}
	case *node.DictExpr:
		// `export {name: local, …}`: each local as it is declared
		for _, e := range v.Elements {
			if e.Spread != nil {
				continue
			}
			key := keyName(e.Key)
			if id, ok := e.Value.(*node.IdentExpr); ok {
				if d := g.locals[id.Name]; d != nil {
					g.declared(key, d, "", false)
					continue
				}
			}
			g.value(key, e.Value, "", false)
		}
	default:
		g.value(name, v, doc, es.Const)
	}
}

// declared documents name, exported by the declaration d.
func (g *gomoduleDocs) declared(name string, d node.Stmt, doc string, isConst bool) {
	switch t := d.(type) {
	case *node.FuncStmt:
		if t.Func != nil && t.Func.Type != nil {
			g.fn(name, t.Func.Type.TypeParams, t.Func.Type.Params, t.Func.Type.Return, doc)
		}
	case *node.FuncWithMethodsStmt:
		if doc == "" {
			doc = commentText(t.Doc)
		}
		for i, m := range t.Methods {
			md := commentText(m.Doc)
			if i == 0 && md == "" {
				md = doc
			}
			g.fn(name, m.TypeParams, m.Params, m.Return, md)
		}
	case *node.DeclStmt:
		if _, v := declValue(t); v != nil {
			g.value(name, v, doc, isConst)
		}
	case *node.AssignStmt:
		if len(t.RHS) == 1 {
			g.value(name, t.RHS[0], doc, isConst)
		}
	default:
		code := g.code(d)
		if !strings.HasPrefix(code, "export ") {
			code = "export " + code
		}
		g.decl(code, doc)
	}
}

// value documents name exported as v: a function by its signature, a
// literal as it is, anything else nil.
func (g *gomoduleDocs) value(name string, v node.Expr, doc string, isConst bool) {
	switch f := v.(type) {
	case *node.FuncExpr:
		if f.Type != nil {
			g.fn(name, f.Type.TypeParams, f.Type.Params, f.Type.Return, doc)
			return
		}
	case *node.ClosureExpr:
		g.fn(name, f.TypeParams, f.Params, f.Return, doc)
		return
	}
	lit := "nil"
	switch v.(type) {
	case *node.IntLit, *node.UintLit, *node.FloatLit, *node.DecimalLit, *node.StrLit, *node.CharLit,
		*node.BoolLit, *node.FlagLit, *node.NilLit, *node.RawStrLit:
		lit = node.Code(v)
	}
	kw := "export "
	if isConst {
		kw = "export const "
	}
	g.decl(kw+name+" = "+lit, doc)
}

// code is the source of the declaration d, its bodies (of its methods, of
// the functions it holds) `nil`: its API, as written — the docs of its
// members kept.
func (g *gomoduleDocs) code(d node.Stmt) string {
	type cut struct {
		from, to int
		with     string
	}
	off := func(p source.Pos) int {
		o, err := g.file.Offset(p)
		if err != nil {
			return -1
		}
		return o
	}
	var cuts []cut
	add := func(body ast.Node, with string) {
		if body == nil {
			return
		}
		from, to := off(body.Pos()), off(body.End())
		if from >= 0 && to >= from {
			cuts = append(cuts, cut{from, to, with})
		}
	}
	node.Walk(d, func(n ast.Node) bool {
		switch m := n.(type) {
		case *node.FuncMethod:
			if m.Body != nil {
				add(m.Body, "=> nil")
			} else {
				add(m.BodyExpr, "nil")
			}
			return false
		case *node.FuncExpr:
			if m.Body != nil {
				add(m.Body, "=> nil")
			} else {
				add(m.BodyExpr, "nil")
			}
			return false
		case *node.ClosureExpr:
			add(m.Body, "nil")
			return false
		}
		return true
	})
	src := g.src
	from, to := off(d.Pos()), off(d.End())
	if from < 0 || to > len(src) {
		return strings.TrimSpace(node.Code(d))
	}
	sort.Slice(cuts, func(i, j int) bool { return cuts[i].from > cuts[j].from })
	text := src[from:to]
	for _, c := range cuts {
		if c.from < from || c.to > to {
			continue
		}
		text = text[:c.from-from] + c.with + text[c.to-from:]
	}
	return dedent(strings.TrimSpace(string(text)))
}

// isConstDecl reports whether d declares a constant (`const NAME = …`).
func isConstDecl(d node.Stmt) bool {
	if ds, ok := d.(*node.DeclStmt); ok {
		if gd, ok := ds.Decl.(*node.GenDecl); ok {
			return gd.Tok == token.Const
		}
	}
	return false
}

// keyName is the name a dict key is.
func keyName(k node.Expr) string {
	switch t := k.(type) {
	case *node.IdentExpr:
		return t.Name
	case *node.StrLit:
		return t.Value()
	}
	return node.Code(k)
}

// declValue is the name and the value a declaration (`NAME = value`) holds.
func declValue(d *node.DeclStmt) (*node.IdentExpr, node.Expr) {
	gd, ok := d.Decl.(*node.GenDecl)
	if !ok || len(gd.Specs) != 1 {
		return nil, nil
	}
	vs, ok := gd.Specs[0].(*node.ValueSpec)
	if !ok || len(vs.Idents) != 1 || len(vs.Values) != 1 {
		return nil, nil
	}
	return vs.Idents[0], vs.Values[0]
}

// topLevelDecls are the module's top-level declarations, by name.
func topLevelDecls(stmts []node.Stmt) map[string]node.Stmt {
	m := map[string]node.Stmt{}
	for _, s := range stmts {
		switch t := s.(type) {
		case *node.FuncStmt:
			if t.Func != nil && t.Func.Type != nil {
				if id, ok := t.Func.Type.NameExpr.(*node.IdentExpr); ok {
					m[id.Name] = s
				}
			}
		case *node.AssignStmt:
			if len(t.LHS) == 1 && len(t.RHS) == 1 {
				if id, ok := t.LHS[0].(*node.IdentExpr); ok {
					m[id.Name] = s
				}
			}
		case *node.DeclStmt:
			if id, _ := declValue(t); id != nil {
				m[id.Name] = s
			}
		case *node.TypeDeclStmt:
			if id, ok := t.NameExpr.(*node.IdentExpr); ok {
				m[id.Name] = s
			}
		case *node.InterfaceStmt:
			if id, ok := t.NameExpr.(*node.IdentExpr); ok {
				m[id.Name] = s
			}
		case *node.EnumStmt:
			if id, ok := t.NameExpr.(*node.IdentExpr); ok {
				m[id.Name] = s
			}
		}
	}
	return m
}

// commentText is the text of a doc comment (`/** … **/`, `///`, `/?`), cleaned.
func commentText(g *ast.CommentGroup) string {
	if g == nil {
		return ""
	}
	var parts []string
	for _, c := range g.List {
		t := c.Text
		switch {
		case strings.HasPrefix(t, "/**"):
			t = strings.TrimPrefix(t, "/**")
			t = strings.TrimSuffix(t, "**/")
			t = strings.TrimSuffix(t, "*/")
		case strings.HasPrefix(t, "///"):
			t = strings.TrimPrefix(t, "///")
		case strings.HasPrefix(t, "//"):
			t = strings.TrimPrefix(t, "//")
		case strings.HasPrefix(t, "/*"):
			t = strings.TrimSuffix(strings.TrimPrefix(t, "/*"), "*/")
		}
		parts = append(parts, t)
	}
	return dedent(strings.Join(parts, "\n"))
}

// dedent removes the indentation the lines of s share.
func dedent(s string) string {
	lines := strings.Split(strings.Trim(s, "\n"), "\n")
	min := -1
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		n := len(l) - len(strings.TrimLeft(l, " \t"))
		if min < 0 || n < min {
			min = n
		}
	}
	for i, l := range lines {
		if len(l) >= min && min > 0 {
			lines[i] = l[min:]
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
