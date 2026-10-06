package gadx

import (
	"strings"
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

// `@import card from "card.gadx"` draws card.gadx's main as card, the same
// as `@import { main: card } from "card.gadx"`; an error in it is reported at
// its line of the template.
func TestImportMainRender(t *testing.T) {
	mods := map[string]string{"card": "@export comp main(title)\n    p.card {= title }\n"}
	for _, src := range []string{
		"@import card from \"card.gadx\"\n@main\n    +card(\"Zz\")\n",
		"@import { main: card } from \"card.gadx\"\n@main\n    +card(\"Zz\")\n",
	} {
		if got, err := portRun(t, src, nil, mods); err != nil || got != `<p class="card">Zz</p>` {
			t.Errorf("%s: %q %v", src, got, err)
		}
	}
	builtins := AppendBuiltins(gad.NewBuiltins())
	_, _, err := Compile(gad.NewSymbolTable(builtins.NameSet), []byte("p x\n@import { a b } from \"comps.gadx\"\n"), gad.CompileOptions{
		CompilerOptions: gad.CompilerOptions{FallbackFunc: CompileFallback, ModuleMap: testModuleMap()},
	})
	if err == nil || !strings.Contains(err.Error(), ":2:") {
		t.Errorf("the error of an @import not at its line: %v", err)
	}
}
