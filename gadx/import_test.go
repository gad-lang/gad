package gadx

import (
	"testing"

	"github.com/gad-lang/gad"
)

func testModuleMap() *gad.ModuleMap {
	mm := gad.NewModuleMap()
	mm.AddSourceModule("components.gadx", []byte{})
	mm.AddSourceModule("other.gadx", []byte{})
	mm.AddSourceModule("comps.gadx", []byte{})
	return mm
}

func compileSrc(t *testing.T, src string) {
	t.Helper()
	builtins := AppendBuiltins(gad.NewBuiltins())
	st := gad.NewSymbolTable(builtins.NameSet)
	opts := gad.CompileOptions{CompilerOptions: gad.CompilerOptions{
		FallbackFunc: CompileFallback,
		ModuleMap:    testModuleMap(),
	}}
	_, _, err := Compile(st, []byte(src), opts)
	if err != nil {
		t.Fatalf("compile: %v\nsrc: %s", err, src)
	}
}

func TestCompileImportBare(t *testing.T) {
	compileSrc(t, `@import "components.gadx"`)
}

func TestCompileImportNamed(t *testing.T) {
	compileSrc(t, `@import "components.gadx" as comps
@main
    p Hello`)
}

func TestCompileDestructureImport(t *testing.T) {
	compileSrc(t, `@import { page_wrapper } from "components.gadx"`)
}

func TestCompileDestructureImportWithMain(t *testing.T) {
	compileSrc(t, `@import { page_wrapper } from "components.gadx"
@main
    +page_wrapper("Test")
        p Hello`)
}

func TestCompileBothImportForms(t *testing.T) {
	compileSrc(t, `@import "components.gadx"
@import { page_wrapper } from "other.gadx"
@main
    p Hello`)
}

func TestCompileDestructureMultipleNames(t *testing.T) {
	compileSrc(t, `@import { page_wrapper, hero } from "components.gadx"
@main
    +page_wrapper("Test")
        p Hello`)
}

func TestCompileDestructureRename(t *testing.T) {
	compileSrc(t, `@import { page_wrapper: pw } from "components.gadx"
@main
    +pw("Test")
        p Hello`)
}

func TestCompileDestructureDefault(t *testing.T) {
	compileSrc(t, `@import { page_wrapper = nil } from "components.gadx"
@main
    p Hello`)
}

func TestCompileDestructureRest(t *testing.T) {
	compileSrc(t, `@import { page_wrapper, **rest } from "components.gadx"
@main
    p Hello`)
}

func TestCompileDestructureMixed(t *testing.T) {
	compileSrc(t, `@import { a, b: bb, c = 5, **rest } from "comps.gadx"
@main
    p Hello`)
}

func TestCompileImportThenDestructure(t *testing.T) {
	src := `@import "components.gadx" as comps
@import { page_wrapper } from "components.gadx"
@main
    +comps.page_wrapper("Old")
    +page_wrapper("New")
        p Hello`
	compileSrc(t, src)
}

func TestCompileDestructureOnlyImport(t *testing.T) {
	// Template using only destructure imports (no named imports)
	src := `@import { hero, post_card } from "components.gadx"
@main
    +hero("Title")
        p Body`
	compileSrc(t, src)
}

func TestCompileGlobal(t *testing.T) {
	compileSrc(t, `@global Model User
@main
    h1 "Hello"`)
}

func TestCompileGlobalMultiple(t *testing.T) {
	compileSrc(t, `@global App Config DB
@main
    h1 "Hello"`)
}

func TestCompileVar(t *testing.T) {
	compileSrc(t, `@var (a, b, c)
@main
    h1 "Hello"`)
}

func TestCompileVarWithInit(t *testing.T) {
	compileSrc(t, `@var (a, b = {}, x)
@main
    h1 "Hello"`)
}

func TestCompileVarSingle(t *testing.T) {
	compileSrc(t, `@var (count = 0)
@main
    h1 {count}`)
}

func TestCompileConst(t *testing.T) {
	compileSrc(t, `@const (a = 1, b = 2, c = 3)
@main
    h1 "Hello"`)
}

func TestCompileConstWithInit(t *testing.T) {
	compileSrc(t, `@const (a = 1, b = {}, x = 2)
@main
    h1 "Hello"`)
}

func TestCompileConstSingle(t *testing.T) {
	compileSrc(t, `@const (name = "test")
@main
    h1 {name}`)
}
