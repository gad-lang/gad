package encoder_test

import (
	"testing"

	"github.com/gad-lang/gad"
	"github.com/stretchr/testify/require"

	. "github.com/gad-lang/gad/encoder"
)

const linkedModuleSrc = `
k := 10
add := func(a int; b=1) => a + b + k
interface Opts { layout? str = "table"; n int = 1 + 2 }
enum Mode { r; w }
contract TSel { target type<str|enum>; meta { options []str } }
contract TEnum { *TSel; target type<enum>; meta { options []target = target.@members } }
draw := func { (f TEnum) => "enum"; (f TSel) => "select"; (f any) => "other" }
sized := (v [2]int) => len(v)
export {add: add, Opts: Opts, Mode: Mode, TSel: TSel, TEnum: TEnum, draw: draw, sized: sized, greet: "hi " + str(k)}
`

// A module compiled on its own, imported as bytecode: linked into the program
// (its constants after the program's), run by the program's VM.
func TestBytecodeModule(t *testing.T) {
	data, err := CompileModule("m", []byte(linkedModuleSrc))
	require.NoError(t, err)
	mm := gad.NewModuleMap().Add("m", BytecodeModule(data))
	src := `
	before := "a constant of the program before the module's"
	m := import("m")
	class F { [options=["a"]] a str; e m.Mode; n int }
	return [before, m.add(1), m.add(1; b=5), {} :> m.Opts, (m.Mode.w).name,
		m.draw(F.a), m.draw(F.e), m.draw(F.n), (F.e :> m.TEnum).meta.options == m.Mode.@members,
		m.sized([1, 2]), m.greet]`
	res, err := gad.Compile(gad.NewSymbolTable(gad.NewBuiltins().NameSet), []byte(src), gad.CompileOptions{CompilerOptions: gad.CompilerOptions{ModuleMap: mm}})
	require.NoError(t, err)
	ret, err := gad.NewVM(gad.NewBuiltins().Build(), res.Bytecode).Run()
	require.NoError(t, err)
	require.Equal(t, gad.Array{
		gad.Str("a constant of the program before the module's"), gad.Int(12), gad.Int(16),
		gad.Dict{"layout": gad.Str("table"), "n": gad.Int(3)}, gad.Str("w"),
		gad.Str("select"), gad.Str("enum"), gad.Str("other"), gad.True, gad.Int(2), gad.Str("hi 10"),
	}, ret)

	// imported twice (two programs): each links a copy of its own
	res2, err := gad.Compile(gad.NewSymbolTable(gad.NewBuiltins().NameSet), []byte(`x := 1; return import("m").add(x)`), gad.CompileOptions{CompilerOptions: gad.CompilerOptions{ModuleMap: mm}})
	require.NoError(t, err)
	ret, err = gad.NewVM(gad.NewBuiltins().Build(), res2.Bytecode).Run()
	require.NoError(t, err)
	require.Equal(t, gad.Int(12), ret)
}

// A module compiled on its own imports none.
func TestBytecodeModuleNoImports(t *testing.T) {
	data, err := CompileModule("m", []byte(`export {x: import("x")}`))
	if err != nil {
		return // refused already
	}
	mm := gad.NewModuleMap().Add("m", BytecodeModule(data))
	_, err = gad.Compile(gad.NewSymbolTable(gad.NewBuiltins().NameSet), []byte(`return import("m")`), gad.CompileOptions{CompilerOptions: gad.CompilerOptions{ModuleMap: mm}})
	require.Error(t, err)
}
