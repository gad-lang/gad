package encoder_test

import (
	"strings"
	"testing"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/encoder"
	gadstrings "github.com/gad-lang/gad/stdlib/strings"
)

// A module of gad with a Go init (issue #17): the init extends a class of the
// gad part with a Go method, using a declared import; an import not declared
// fails.
func TestGoModuleInit(t *testing.T) {
	data, err := encoder.CompileModule("shapes", []byte(`
export class Shape {
    name str = "?"
    methods {
        describe() => "shape " + this.name
    }
}
export area = func(w, h) => w * h
`))
	if err != nil {
		t.Fatal(err)
	}
	init := func(ctx *gad.ModuleInitContext) (gad.Dict, error) {
		base, err := ctx.Get("Shape")
		if err != nil {
			return nil, err
		}
		strs, err := ctx.Import("strings")
		if err != nil {
			return nil, err
		}
		cls := gad.NewClass("GoShape", ctx.Spec())
		cls.Extends(base.(*gad.Class), "")
		if err := cls.AddGoMethod("label", func(c gad.Call) (gad.Object, error) {
			this := c.Args.Get(0)
			name, err := this.(gad.IndexGetter).IndexGet(c.VM, gad.Str("name"))
			if err != nil {
				return nil, err
			}
			return strs.Call(c.VM, "toUpper", name)
		}); err != nil {
			return nil, err
		}
		return gad.Dict{"GoShape": cls}, nil
	}
	run := func(mod *encoder.GoModule, src string) (gad.Object, error) {
		mm := gad.NewModuleMap()
		mm.AddBuiltinModuleInit("strings", gadstrings.ModuleInit)
		mm.Add("shapes", mod)
		res, err := gad.Compile(gad.NewSymbolTable(gad.NewBuiltins().NameSet), []byte(src), gad.CompileOptions{CompilerOptions: gad.CompilerOptions{ModuleMap: mm}})
		if err != nil {
			return nil, err
		}
		return gad.NewVM(gad.NewBuiltins().Build(), res.Bytecode).Run()
	}
	ret, err := run(&encoder.GoModule{Data: data, Imports: []string{"strings"}, Init: init}, `
s := import("shapes")
g := s.GoShape(; name="box")
return [g.describe(), g.label(), s.area(2, 3), typeName(g)]`)
	if err != nil {
		t.Fatal(err)
	}
	if got := ret.ToString(); got != `["shape box", "BOX", 6, "GoShape"]` {
		t.Fatalf("got %s", got)
	}
	_, err = run(&encoder.GoModule{Data: data, Init: init}, `import("shapes")`)
	if err == nil || !strings.Contains(err.Error(), `"strings" is not an import of it`) {
		t.Fatalf("want an undeclared import error, got %v", err)
	}
}

// An export of both the gad part and the Go init is an error, unless the init
// replaces it (ctx.Replace).
func TestGoModuleInitNames(t *testing.T) {
	data, err := encoder.CompileModule("m", []byte(`export x = 1`))
	if err != nil {
		t.Fatal(err)
	}
	run := func(init gad.ModuleGoInit) (gad.Object, error) {
		mm := gad.NewModuleMap().Add("m", &encoder.GoModule{Data: data, Init: init})
		res, err := gad.Compile(gad.NewSymbolTable(gad.NewBuiltins().NameSet), []byte(`return import("m").x`),
			gad.CompileOptions{CompilerOptions: gad.CompilerOptions{ModuleMap: mm}})
		if err != nil {
			return nil, err
		}
		return gad.NewVM(gad.NewBuiltins().Build(), res.Bytecode).Run()
	}
	_, err = run(func(ctx *gad.ModuleInitContext) (gad.Dict, error) { return gad.Dict{"x": gad.Int(2)}, nil })
	if err == nil || !strings.Contains(err.Error(), "x exported by its gad part too") {
		t.Fatalf("want a duplicate export error, got %v", err)
	}
	ret, err := run(func(ctx *gad.ModuleInitContext) (gad.Dict, error) {
		ctx.Replace("x", gad.Int(3))
		return nil, nil
	})
	if err != nil || ret != gad.Int(3) {
		t.Fatalf("got %v, %v; want 3", ret, err)
	}
}
