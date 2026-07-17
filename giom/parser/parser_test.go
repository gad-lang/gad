package parser

import (
	"strings"
	"testing"

	gnode "github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	giomnode "github.com/gad-lang/gad/giom/node"
)

func parseLine(t *testing.T, src string) *giomnode.File {
	t.Helper()
	fs := source.NewFileSet()
	f := fs.AddFileData("test.giom", -1, []byte(src))
	p := NewParser(f)
	file, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse error: %v\nsrc: %q", err, src)
	}
	return file
}

func expectStmtCount(t *testing.T, file *giomnode.File, n int) {
	t.Helper()
	if len(file.Stmts) != n {
		t.Fatalf("expected %d statements, got %d", n, len(file.Stmts))
	}
}

func expectCodeStmt(t *testing.T, file *giomnode.File, idx int, contains string) {
	t.Helper()
	if idx >= len(file.Stmts) {
		t.Fatalf("statement index %d out of range (len=%d)", idx, len(file.Stmts))
	}
	cs, ok := file.Stmts[idx].(*giomnode.CodeStmt)
	if !ok {
		t.Fatalf("expected *giomnode.CodeStmt at index %d, got %T", idx, file.Stmts[idx])
	}
	for _, stmt := range cs.Stmts {
		got := stmt.String()
		if !strings.Contains(got, contains) {
			t.Fatalf("expected CodeStmt to contain %q, got %q", contains, got)
		}
	}
}

func TestImportBare(t *testing.T) {
	file := parseLine(t, `@import "components.giom"`)
	expectStmtCount(t, file, 1)
	expectCodeStmt(t, file, 0, "import")
}

func TestImportNamed(t *testing.T) {
	file := parseLine(t, `@import "components.giom" as comps`)
	expectStmtCount(t, file, 1)
	expectCodeStmt(t, file, 0, "import")
	expectCodeStmt(t, file, 0, "comps")
}

func TestImportDestructureSingle(t *testing.T) {
	file := parseLine(t, `@import { page_wrapper } from "comps.giom"`)
	expectStmtCount(t, file, 1)
	code := codeStmtStr(t, file, 0)
	if !strings.Contains(code, "page_wrapper") {
		t.Fatalf("expected destructure entry 'page_wrapper', got: %s", code)
	}
}

func TestImportDestructureMultiple(t *testing.T) {
	file := parseLine(t, `@import { page_wrapper, hero, post_card } from "comps.giom"`)
	expectStmtCount(t, file, 1)
	code := codeStmtStr(t, file, 0)
	for _, name := range []string{"page_wrapper", "hero", "post_card"} {
		if !strings.Contains(code, name) {
			t.Fatalf("expected destructure entry %q, got: %s", name, code)
		}
	}
}

func TestImportDestructureRename(t *testing.T) {
	file := parseLine(t, `@import { page_wrapper: pw, hero: h } from "comps.giom"`)
	expectStmtCount(t, file, 1)
	code := codeStmtStr(t, file, 0)
	if !strings.Contains(code, "pw = giom_import_0.page_wrapper") || !strings.Contains(code, "h = giom_import_0.hero") {
		t.Fatalf("expected renamed explicit imports, got: %s", code)
	}
}

func TestImportDestructureDefault(t *testing.T) {
	file := parseLine(t, `@import { page_wrapper = default_wrapper } from "comps.giom"`)
	expectStmtCount(t, file, 1)
	code := codeStmtStr(t, file, 0)
	if !strings.Contains(code, "page_wrapper = (giom_import_0.page_wrapper ?? default_wrapper)") {
		t.Fatalf("expected explicit import with default, got: %s", code)
	}
}

func TestImportDestructureRest(t *testing.T) {
	file := parseLine(t, `@import { page_wrapper, **rest } from "comps.giom"`)
	expectStmtCount(t, file, 1)
	code := codeStmtStr(t, file, 0)
	if !strings.Contains(code, "rest = giom_import_0") {
		t.Fatalf("expected explicit rest import, got: %s", code)
	}
}

// globalNames extracts the declared identifier names of a `@global` statement
// from its lowered Gad declaration (or the legacy Names slice).
func globalNames(t *testing.T, gs *giomnode.GlobalStmt) []string {
	t.Helper()
	if gs.Decl == nil {
		return gs.Names
	}
	var names []string
	for _, sp := range gs.Decl.Specs {
		switch s := sp.(type) {
		case *gnode.ParamSpec:
			names = append(names, s.Ident.Ident.Name)
		case *gnode.NamedParamSpec:
			names = append(names, s.Ident.Ident.Name)
		}
	}
	return names
}

func TestGlobal(t *testing.T) {
	file := parseLine(t, `@global Model User`)
	if len(file.Stmts) != 1 {
		t.Fatalf("expected 1 stmt, got %d", len(file.Stmts))
	}
	gs, ok := file.Stmts[0].(*giomnode.GlobalStmt)
	if !ok {
		t.Fatalf("expected *giomnode.GlobalStmt, got %T", file.Stmts[0])
	}
	names := globalNames(t, gs)
	if len(names) != 2 || names[0] != "Model" || names[1] != "User" {
		t.Fatalf("expected [Model User], got %v", names)
	}
}

func TestGlobalSingle(t *testing.T) {
	file := parseLine(t, `@global App`)
	if len(file.Stmts) != 1 {
		t.Fatalf("expected 1 stmt, got %d", len(file.Stmts))
	}
	gs, ok := file.Stmts[0].(*giomnode.GlobalStmt)
	if !ok {
		t.Fatalf("expected *giomnode.GlobalStmt, got %T", file.Stmts[0])
	}
	names := globalNames(t, gs)
	if len(names) != 1 || names[0] != "App" {
		t.Fatalf("expected [App], got %v", names)
	}
}

func TestGlobalCommaAndDefaults(t *testing.T) {
	file := parseLine(t, `@global a, b, c = 1`)
	gs := file.Stmts[0].(*giomnode.GlobalStmt)
	names := globalNames(t, gs)
	if len(names) != 3 || names[0] != "a" || names[2] != "c" {
		t.Fatalf("expected [a b c], got %v", names)
	}
}

func TestVar(t *testing.T) {
	file := parseLine(t, `@var (a, b = {}, x)`)
	if len(file.Stmts) != 1 {
		t.Fatalf("expected 1 stmt, got %d", len(file.Stmts))
	}
	vs, ok := file.Stmts[0].(*giomnode.VarStmt)
	if !ok {
		t.Fatalf("expected *giomnode.VarStmt, got %T", file.Stmts[0])
	}
	if len(vs.Decls) != 3 {
		t.Fatalf("expected 3 decls, got %d", len(vs.Decls))
	}
	if vs.Decls[0].Name != "a" || vs.Decls[0].Init != nil {
		t.Fatalf("expected a with nil init, got name=%q init=%v", vs.Decls[0].Name, vs.Decls[0].Init)
	}
	if vs.Decls[1].Name != "b" || vs.Decls[1].Init == nil {
		t.Fatalf("expected b with init, got name=%q init=%v", vs.Decls[1].Name, vs.Decls[1].Init)
	}
	if vs.Decls[2].Name != "x" || vs.Decls[2].Init != nil {
		t.Fatalf("expected x with nil init, got name=%q init=%v", vs.Decls[2].Name, vs.Decls[2].Init)
	}
}

func TestVarNoInit(t *testing.T) {
	file := parseLine(t, `@var (a, b, c)`)
	if len(file.Stmts) != 1 {
		t.Fatalf("expected 1 stmt, got %d", len(file.Stmts))
	}
	vs, ok := file.Stmts[0].(*giomnode.VarStmt)
	if !ok {
		t.Fatalf("expected *giomnode.VarStmt, got %T", file.Stmts[0])
	}
	if len(vs.Decls) != 3 {
		t.Fatalf("expected 3 decls, got %d", len(vs.Decls))
	}
	for i, d := range vs.Decls {
		if d.Init != nil {
			t.Fatalf("decl %d expected nil init, got %v", i, d.Init)
		}
	}
}

func TestConst(t *testing.T) {
	file := parseLine(t, `@const (a = 1, b = {}, x = 2)`)
	if len(file.Stmts) != 1 {
		t.Fatalf("expected 1 stmt, got %d", len(file.Stmts))
	}
	cs, ok := file.Stmts[0].(*giomnode.ConstStmt)
	if !ok {
		t.Fatalf("expected *giomnode.ConstStmt, got %T", file.Stmts[0])
	}
	if len(cs.Decls) != 3 {
		t.Fatalf("expected 3 decls, got %d", len(cs.Decls))
	}
	if cs.Decls[0].Name != "a" || cs.Decls[0].Init == nil {
		t.Fatalf("expected a with init, got name=%q init=%v", cs.Decls[0].Name, cs.Decls[0].Init)
	}
	if cs.Decls[1].Name != "b" || cs.Decls[1].Init == nil {
		t.Fatalf("expected b with init, got name=%q init=%v", cs.Decls[1].Name, cs.Decls[1].Init)
	}
	if cs.Decls[2].Name != "x" || cs.Decls[2].Init == nil {
		t.Fatalf("expected x with init, got name=%q init=%v", cs.Decls[2].Name, cs.Decls[2].Init)
	}
}

func TestConstSingle(t *testing.T) {
	file := parseLine(t, `@const (count = 0)`)
	if len(file.Stmts) != 1 {
		t.Fatalf("expected 1 stmt, got %d", len(file.Stmts))
	}
	cs, ok := file.Stmts[0].(*giomnode.ConstStmt)
	if !ok {
		t.Fatalf("expected *giomnode.ConstStmt, got %T", file.Stmts[0])
	}
	if len(cs.Decls) != 1 || cs.Decls[0].Name != "count" || cs.Decls[0].Init == nil {
		t.Fatalf("expected [count=0], got %v", cs.Decls)
	}
}

func TestVarSingle(t *testing.T) {
	file := parseLine(t, `@var (count = 0)`)
	if len(file.Stmts) != 1 {
		t.Fatalf("expected 1 stmt, got %d", len(file.Stmts))
	}
	vs, ok := file.Stmts[0].(*giomnode.VarStmt)
	if !ok {
		t.Fatalf("expected *giomnode.VarStmt, got %T", file.Stmts[0])
	}
	if len(vs.Decls) != 1 || vs.Decls[0].Name != "count" || vs.Decls[0].Init == nil {
		t.Fatalf("expected [count=0], got %v", vs.Decls)
	}
}

func TestImportDestructureMixed(t *testing.T) {
	file := parseLine(t, `@import { a, b: bb, c = 5, **rest } from "comps.giom"`)
	expectStmtCount(t, file, 1)
	code := codeStmtStr(t, file, 0)
	for _, part := range []string{"a = giom_import_0.a", "bb = giom_import_0.b", "c = (giom_import_0.c ?? 5)", "rest = giom_import_0"} {
		if !strings.Contains(code, part) {
			t.Fatalf("expected part %q, got: %s", part, code)
		}
	}
}

func TestImportDestructureWithMain(t *testing.T) {
	src := `@import { page_wrapper } from "comps.giom"
@main
    +page_wrapper("Test")
        p Hello`
	file := parseLine(t, src)
	expectStmtCount(t, file, 2)
	code := codeStmtStr(t, file, 0)
	if !strings.Contains(code, "import") || !strings.Contains(code, "page_wrapper") {
		t.Fatalf("expected import and page_wrapper, got %q", code)
	}
}

func codeStmtStr(t *testing.T, file *giomnode.File, idx int) string {
	t.Helper()
	if idx >= len(file.Stmts) {
		t.Fatalf("statement index %d out of range (len=%d)", idx, len(file.Stmts))
	}
	cs, ok := file.Stmts[idx].(*giomnode.CodeStmt)
	if !ok {
		t.Fatalf("expected *giomnode.CodeStmt at index %d, got %T", idx, file.Stmts[idx])
	}
	var parts []string
	for _, stmt := range cs.Stmts {
		parts = append(parts, stmt.String())
	}
	return strings.Join(parts, "; ")
}
