package encoder_test

import (
	"fmt"
	"testing"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/test_helper/compare"
	"github.com/stretchr/testify/require"

	. "github.com/gad-lang/gad/encoder"
)

func TestEncDecBytecode(t *testing.T) {
	testEncDecBytecode(t, `
	f := func(arg0, arg1, *varg; na0=100, **na) {
		return [arg0, arg1, varg, na0, na.dict]
	}
	return f(1,2,3;na0=4,na1=5)`, nil, gad.Array{gad.Int(1), gad.Int(2), gad.Array{gad.Int(3)}, gad.Int(4), gad.Dict{"na1": gad.Int(5)}})

	testEncDecBytecode(t, `
	param (arg0, arg1, *varg; na0=100, na1=200, **na)
	return [arg0, arg1, varg, na0, na1, na.dict]`, &testopts{
		args:      gad.Array{gad.Int(1), gad.Int(2), gad.Int(3)},
		namedArgs: gad.Dict{"na0": gad.Int(4), "na2": gad.Int(5)},
	}, gad.Array{gad.Int(1), gad.Int(2), gad.Array{gad.Int(3)}, gad.Int(4), gad.Int(200), gad.Dict{"na2": gad.Int(5)}})

	testEncDecBytecode(t, `
	param (arg0, arg1, *varg; na0=100, na1=200, **na)
	return [arg0, arg1, varg, na0, na1, na.dict]`, &testopts{
		args:      gad.Array{gad.Int(1), gad.Int(2), gad.Int(3)},
		namedArgs: gad.Dict{"na2": gad.Int(5)},
	}, gad.Array{gad.Int(1), gad.Int(2), gad.Array{gad.Int(3)}, gad.Int(100), gad.Int(200), gad.Dict{"na2": gad.Int(5)}})

	testEncDecBytecode(t, `
	f := func(arg0, arg1, *varg; na0=100, **na) {
		return [arg0, arg1, varg, na0, na.dict]
	}
	return f(1,2,3;na0=4,na1=5)`, nil, gad.Array{gad.Int(1), gad.Int(2), gad.Array{gad.Int(3)}, gad.Int(4), gad.Dict{"na1": gad.Int(5)}})

	testEncDecBytecode(t, `
	f := func() {
		return [nil, true, false, "", -1, 0, 1, 2u, 3.0, 123.456d, 'a', bytes(0, 1, 2)]
	}
	f()
	m := {a: 1, b: ["abc"], c: {x: bytes()}, builtins: [append, len]}`, nil, gad.Nil)

	testEncDecBytecode(t, `
	f := func(arg0, arg1, *varg; na0=3, **na) {
		return [arg0, arg1, varg, na0, na.dict, nil, true, false, "", -1, 0, 1, 2u, 3.0, 123.456d, 'a', bytes(0, 1, 2)]
	}
	f(1,2;na0=4,na1=5)
	m := {a: 1, b: ["abc"], c: {x: bytes()}, builtins: [append, len]}`, nil, gad.Nil)
}

func TestEncDecBytecode_modules(t *testing.T) {
	testEncDecBytecode(t, `
	mod1 := import("mod1")
	mod2 := import("mod2")
	return mod1.run() + mod2.run()
	`, newOpts().
		Module("mod1", gad.Dict{
			"run": &gad.Function{
				FuncName: "run",
				Value: func(gad.Call) (gad.Object, error) {
					return gad.Str("mod1"), nil
				},
			},
		}).
		Module("mod2", `export{run: func(){ return "mod2" }}`), gad.Str("mod1mod2"))
}

func testEncDecBytecode(t *testing.T, script string, opts *testopts, expected gad.Object) {
	t.Helper()
	if opts == nil {
		opts = newOpts()
	}

	if cfn, ok := expected.(*gad.CompiledFunction); ok {
		expected = cfn
	}

	var initialModuleMap *gad.ModuleMap
	if opts.moduleMap != nil {
		initialModuleMap = opts.moduleMap.Copy()
	}

	builtins := gad.NewBuiltins().Build()
	st := gad.NewSymbolTable(builtins.Builtins().NameSet)

	_, bc, err := gad.Compile(st, []byte(script),
		gad.CompileOptions{CompilerOptions: gad.CompilerOptions{
			ModuleMap: opts.moduleMap,
		}},
	)

	var goModules GoModules
	if initialModuleMap != nil {
		goModules = GoModulesFromModulesMap(initialModuleMap)
	} else {
		goModules = make(GoModules, 0)
	}

	require.NoError(t, err)
	vm := gad.NewVM(builtins, bc)
	items := gad.MustConvertToKeyValueArray(nil, opts.namedArgs)
	ret, err := vm.RunOpts(&gad.RunOpts{
		Globals:   opts.globals,
		Args:      gad.Args{opts.args},
		NamedArgs: gad.NewNamedArgs(items),
	})
	require.NoError(t, err)
	require.Equal(t, expected, ret)

	data, err := encode(bc)
	require.NoError(t, err, "Encode")
	require.Greater(t, len(data), 0, "Encoded data")

	var v *gad.Bytecode
	v, err = decode[*gad.Bytecode](data, ContextWithGoModules(goModules))
	require.NoError(t, err, "Decode")

	testDecodedBytecodeEqual(t, builtins, bc, v)
	items = gad.MustConvertToKeyValueArray(nil, opts.namedArgs)
	ret, err = gad.NewVM(builtins, v).RunOpts(&gad.RunOpts{
		Globals:   opts.globals,
		Args:      gad.Args{opts.args},
		NamedArgs: gad.NewNamedArgs(items),
	})
	require.NoError(t, err)
	require.Equal(t, expected, ret)
	// ensure moduleMap is not updated during compilation and decoding
	require.Equal(t, initialModuleMap, opts.moduleMap)
}

func testDecodedBytecodeEqual(t *testing.T, builtins *gad.StaticBuiltins, expected, got *gad.Bytecode) {
	t.Helper()
	msg := fmt.Sprintf("actual:%s\ndecoded:%s\n", expected, got)

	testBytecodeConstants(t, gad.NewVM(builtins, expected).Init(), expected.Constants, got.Constants)
	compare.Equal(t, expected.Main, got.Main, msg)
	require.Equal(t, expected.NumModules, got.NumModules, msg)
	if expected.FileSet == nil {
		require.Nil(t, got.FileSet, msg)
	} else {
		require.Equal(t, expected.FileSet.Base, got.FileSet.Base, msg)
		require.Equal(t, len(expected.FileSet.Files), len(got.FileSet.Files), msg)
		for i, f := range expected.FileSet.Files {
			f2 := got.FileSet.Files[i]
			require.Equal(t, f.Base, f2.Base, msg)
			require.Equal(t, f.Lines, f2.Lines, msg)
			require.Equal(t, f.Name, f2.Name, msg)
			require.Equal(t, f.Size, f2.Size, msg)
		}
		require.NotNil(t, expected.FileSet.LastFile, msg)
		require.NotNil(t, got.FileSet.LastFile, msg)
	}
}
