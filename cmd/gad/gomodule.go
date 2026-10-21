package main

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/cmd/internal/gaddoc"
	"github.com/gad-lang/gad/encoder"
	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/stdlib/helper"
	"github.com/gad-lang/gad/token"
	"github.com/gad-lang/gad/web/gadbridge"
	cc "github.com/moisespsena-go/command-context"
)

// `gad gomodule` (issue #17): a module written in gad made a builtin module in
// Go — a package that embeds the module compiled (`.gadb`), documents it with
// `gad:doc` comments (its API stub, `gaddoc api`'s, written beside it) and has
// a hook for a Go init of its own (`Init`, set by a hand-written file).

// gomoduleOpts are the options of `gad gomodule`.
type gomoduleOpts struct {
	pkg, module, out, api string
	imports               []string
}

func gomoduleCommand() *cc.Command {
	var (
		o       gomoduleOpts
		imports string
	)
	return &cc.Command{
		Name:  "gomodule",
		Usage: "[flags] MODULE.gad",
		Description: "Make a gad module a builtin module in Go (issue #17).\n\n" +
			"Writes into --out (a Go package): MODULE.gadb (the module compiled, linked\n" +
			"into the programs that import it), MODULE.go (the package: ModuleName, the\n" +
			"embedded bytecode, Module() — the importable —, AddTo, Imports, the Init\n" +
			"hook, and the module's docs as gad:doc comments), MODULE_test.go (links it\n" +
			"against the default builtins) and the API stub (--api, `gaddoc api`'s).\n\n" +
			"A Go init of the module — a hand-written file of the package setting Init —\n" +
			"runs after the gad part; the modules it imports are declared: --import, or\n" +
			"`//gad:import NAME[,NAME]` lines in the package's Go files.",
		New: func(ctx *cc.CommandContext) error {
			f := ctx.Flags()
			f.StringVar(&o.pkg, "pkg", "", "Go package name (default: the module name)")
			f.StringVar(&o.module, "module", "", "module name (default: the file name)")
			f.StringVar(&o.out, "out", ".", "output directory (the Go package)")
			f.StringVar(&o.api, "api", "", "API stub file (default: OUT/MODULE.gad)")
			f.StringVar(&imports, "import", "", "modules the Go init imports, comma separated")
			return nil
		},
		Run: func(ctx *cc.CommandContext) error {
			if len(ctx.Args) != 1 {
				return fmt.Errorf("usage: gad gomodule [flags] MODULE.gad")
			}
			for _, n := range strings.Split(imports, ",") {
				if n = strings.TrimSpace(n); n != "" {
					o.imports = append(o.imports, n)
				}
			}
			return runGomodule(ctx.Args[0], o)
		},
	}
}

// reGadImport is a `//gad:import NAME[,NAME]` directive of a Go file.
var reGadImport = regexp.MustCompile(`(?m)^//gad:import\s+(\S+)\s*$`)

func runGomodule(srcPath string, o gomoduleOpts) error {
	src, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	base := filepath.Base(srcPath)
	ext := filepath.Ext(base)
	kind := gad.SourceKindGad
	switch ext {
	case ".gadt":
		kind = gad.SourceKindGadt
	case ".gadx":
		kind = gad.SourceKindGadx
	}
	if o.module == "" {
		o.module = strings.TrimSuffix(base, ext)
	}
	if !reModuleName.MatchString(o.module) {
		return fmt.Errorf("invalid module name %q: use a lowercase snake_case identifier (--module)", o.module)
	}
	if o.pkg == "" {
		o.pkg = o.module
	}
	if o.api == "" {
		o.api = filepath.Join(o.out, o.module+".gad")
	}
	if err = os.MkdirAll(o.out, 0o755); err != nil {
		return err
	}

	// the module compiled: its imports by name only (the builtin modules); a
	// file cannot be resolved from the host
	data, err := encoder.CompileModuleWith(o.module, src, kind, helper.NewModuleMap(), nil)
	if err != nil {
		return fmt.Errorf("compile %s: %w", srcPath, err)
	}

	imports, err := gomoduleImports(o)
	if err != nil {
		return err
	}

	var doc []string
	if kind != gad.SourceKindGadx {
		if doc, err = gomoduleDoc(o.module, src, kind); err != nil {
			return fmt.Errorf("docs of %s: %w", srcPath, err)
		}
	} else {
		doc = []string{"gad:doc", "# " + o.module + " module"}
	}

	rel, err := filepath.Rel(o.out, srcPath)
	if err != nil {
		rel = srcPath
	}
	gen := "go run " + gadCmdPath() + " gomodule --pkg " + o.pkg + " --module " + o.module + " --out ."
	if apiRel, err := filepath.Rel(o.out, o.api); err == nil && apiRel != o.module+".gad" {
		gen += " --api " + filepath.ToSlash(apiRel)
	}
	if len(o.imports) > 0 {
		gen += " --import " + strings.Join(o.imports, ",")
	}
	gen += " " + filepath.ToSlash(rel)

	goSrc, err := format.Source([]byte(gomoduleGo(o, gen, imports, doc)))
	if err != nil {
		return fmt.Errorf("the Go source: %w", err)
	}
	testSrc, err := format.Source([]byte(gomoduleTest(o)))
	if err != nil {
		return fmt.Errorf("the Go test: %w", err)
	}
	files := map[string][]byte{
		o.module + ".gadb":    data,
		o.module + ".go":      goSrc,
		o.module + "_test.go": testSrc,
	}
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		p := filepath.Join(o.out, n)
		if err = os.WriteFile(p, files[n], 0o644); err != nil {
			return err
		}
		fmt.Println("wrote", p)
	}

	// the API stub: what `gaddoc api` makes of the package's gad:doc
	api, warnings, err := gaddoc.API(o.out, o.module)
	if err != nil {
		return fmt.Errorf("the API stub: %w", err)
	}
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "gad gomodule: %s: %s\n", o.module, w)
	}
	if err = os.WriteFile(o.api, []byte(api), 0o644); err != nil {
		return err
	}
	fmt.Println("wrote", o.api)
	return nil
}

// gadCmdPath is the package `go generate` runs gad from: of the version of
// this gad (`…/cmd/gad@VERSION`, run apart from the module of the package,
// which need not depend on gad's command), or as it is in gad's own module.
func gadCmdPath() string {
	const path = "github.com/gad-lang/gad/cmd/gad"
	if bi, ok := debug.ReadBuildInfo(); ok {
		if v := bi.Main.Version; bi.Main.Path == "github.com/gad-lang/gad" && v != "" && v != "(devel)" {
			return path + "@" + v
		}
	}
	return path
}

// gomoduleImports are the modules the Go init imports: --import, and the
// `//gad:import` lines of the package's Go files (but the generated ones).
func gomoduleImports(o gomoduleOpts) ([]string, error) {
	set := map[string]bool{}
	for _, n := range o.imports {
		set[n] = true
	}
	entries, err := os.ReadDir(o.out)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".go") || n == o.module+".go" || n == o.module+"_test.go" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(o.out, n))
		if err != nil {
			return nil, err
		}
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
	return out, nil
}

// gomoduleGo is the Go source of the package.
func gomoduleGo(o gomoduleOpts, gen string, imports []string, doc []string) string {
	var b strings.Builder
	b.WriteString("// Code generated by gad gomodule; DO NOT EDIT.\n\n")
	b.WriteString("//go:generate " + gen + "\n\n")
	fmt.Fprintf(&b, "// Package %s is the gad module %q (made by `gad gomodule`): its gad part\n", o.pkg, o.module)
	b.WriteString("// compiled, embedded and linked into the programs that import it, and the\n")
	b.WriteString("// hook of a Go init of its own (Init).\n")
	fmt.Fprintf(&b, "package %s\n\n", o.pkg)
	b.WriteString("import (\n\t_ \"embed\"\n\n\t\"github.com/gad-lang/gad\"\n\t\"github.com/gad-lang/gad/encoder\"\n)\n\n")
	for _, l := range doc {
		if l == "" {
			b.WriteString("//\n")
		} else {
			b.WriteString("// " + l + "\n")
		}
	}
	b.WriteString("\n")
	fmt.Fprintf(&b, "// ModuleName is the import name of the module.\nconst ModuleName = %q\n\n", o.module)
	fmt.Fprintf(&b, "// Bytecode is the module compiled (`.gadb`).\n//\n//go:embed %s.gadb\nvar Bytecode []byte\n\n", o.module)
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

// gomoduleTest is the Go test of the package: the module links against the
// default builtins and modules.
func gomoduleTest(o gomoduleOpts) string {
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
`, "PKG", o.pkg)
}

// gomoduleDoc are the gad:doc comment lines of the module: its doc (the
// file's leading `/** … **/`), its functions (`## Functions`: signatures and
// docs) and its other exports (`## Declarations`: the declarations, their
// bodies `nil`).
func gomoduleDoc(module string, src []byte, kind gad.SourceKind) ([]string, error) {
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
