package shapes

import (
	"strings"
	"testing"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/stdlib/helper"
)

func run(t *testing.T, mm *gad.ModuleMap, src string) (gad.Object, error) {
	t.Helper()
	res, err := gad.Compile(gad.NewSymbolTable(gad.NewBuiltins().NameSet), []byte(src),
		gad.CompileOptions{CompilerOptions: gad.CompilerOptions{ModuleMap: mm}})
	if err != nil {
		return nil, err
	}
	return gad.NewVM(gad.NewBuiltins().Build(), res.Bytecode).Run()
}

// A program imports the module by name: the gad part's exports and the Go
// init's class — extending the gad class Shape, a gad method inherited and a
// Go one of its own.
func TestShapes(t *testing.T) {
	ret, err := run(t, AddTo(helper.NewModuleMap()), `
s := import("shapes")
g := s.GoShape(; name="box")
return [s.area(2, 3), s.surface(; side=3), s.PI > 3, s.Shape(; name="c").describe(),
	g.describe(), g.label(), typeName(g), (s.Unit.m).name, (g :> s.Shape).describe(), typeof(g) == s.GoShape]`)
	if err != nil {
		t.Fatal(err)
	}
	want := `[6, 9, true, "shape c", "shape box", "BOX", "GoShape", "m", "shape box", true]`
	if got := ret.ToString(); got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

// The Go init imports only declared modules.
func TestShapesUndeclaredImport(t *testing.T) {
	saved := Imports
	defer func() { Imports = saved }()
	Imports = nil
	_, err := run(t, AddTo(helper.NewModuleMap()), `import("shapes")`)
	if err == nil || !strings.Contains(err.Error(), `"strings" is not an import of it`) {
		t.Fatalf("want an undeclared import error, got %v", err)
	}
}
