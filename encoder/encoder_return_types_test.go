package encoder_test

import (
	"fmt"
	"testing"

	"github.com/gad-lang/gad"
	"github.com/stretchr/testify/require"

	. "github.com/gad-lang/gad/encoder"
)

func withReturnTypes(rts ...*gad.ReturnVar) funcOpt {
	return func(cf *gad.CompiledFunction) {
		cf.ReturnVars = rts
	}
}

func sym(name string) *gad.SymbolInfo {
	return &gad.SymbolInfo{Name: name, Scope: gad.ScopeBuiltin}
}

func firstCompiledFunc(consts []gad.Object) *gad.CompiledFunction {
	for _, c := range consts {
		if cf, ok := c.(*gad.CompiledFunction); ok {
			return cf
		}
	}
	return nil
}

// TestEncDecCompiledFuncReturnTypes round-trips CompiledFunction values whose
// ReturnTypes field is populated directly, covering anonymous, multiple, named
// and union return shapes plus the no-return-type regression case.
func TestEncDecCompiledFuncReturnTypes(t *testing.T) {
	cases := []*gad.CompiledFunction{
		// anonymous single: "<int>"
		compFunc(nil, withReturnTypes(
			&gad.ReturnVar{TypesSymbols: gad.ParamType{sym("int")}},
		)),
		// anonymous multiple: "<int, str>"
		compFunc(nil,
			withParams("a", "b"),
			withReturnTypes(
				&gad.ReturnVar{TypesSymbols: gad.ParamType{sym("int")}},
				&gad.ReturnVar{TypesSymbols: gad.ParamType{sym("str")}},
			),
		),
		// named union: "<x int|bool>"
		compFunc(nil, withReturnTypes(
			&gad.ReturnVar{Name: "x", TypesSymbols: gad.ParamType{sym("int"), sym("bool")}},
		)),
		// mixed with instructions, params, locals and source map.
		compFunc(concatInsts(
			makeInst(gad.OpConstant, 0),
			makeInst(gad.OpReturn, 1),
		),
			withParams("a"),
			withLocals(1),
			withSourceMap(map[int]int{0: 1, 3: 1}),
			withReturnTypes(
				&gad.ReturnVar{Name: "out", TypesSymbols: gad.ParamType{sym("int")}},
			),
		),
		// regression: no return types — field must be absent and stay nil.
		compFunc(nil, withParams("a")),
	}

	for i, tC := range cases {
		msg := fmt.Sprintf("CompiledFunction #%d", i)
		data, err := encode(tC)
		require.NoError(t, err, msg)
		require.Greater(t, len(data), 0, msg)

		v, err := decode[*gad.CompiledFunction](data)
		require.NoError(t, err, msg)

		if len(v.Instructions) == 0 {
			v.Instructions = nil
		}

		require.Equal(t, tC, v, msg)
		require.Equal(t, tC.ReturnVars, v.ReturnVars, msg)
		require.Equal(t, tC.HeaderString(), v.HeaderString(), msg)
	}
}

type retSpec struct {
	name  string
	types string
}

// TestEncDecBytecodeReturnTypes compiles real source carrying return types,
// round-trips the whole bytecode through the encoder, and verifies the decoded
// function keeps its ReturnTypes and renders them identically.
func TestEncDecBytecodeReturnTypes(t *testing.T) {
	cases := []struct {
		script string
		suffix string
		want   []retSpec
	}{
		{`return func(a) <int> { return a }`, " <int>", []retSpec{{"", "int"}}},
		{`return func(a, b) <int, str> { return [a, b] }`, " <int, str>",
			[]retSpec{{"", "int"}, {"", "str"}}},
		{`return func(a int) <x int|bool> => a`, " <x int|bool>",
			[]retSpec{{"x", "int|bool"}}},
		{`return func(a) { return a }`, "", nil},
	}

	for i, c := range cases {
		msg := fmt.Sprintf("case #%d: %s", i, c.script)

		builtins := gad.NewBuiltins().Build()
		st := gad.NewSymbolTable(builtins.Builtins().NameSet)
		_, bc, err := gad.Compile(st, []byte(c.script), gad.CompileOptions{})
		require.NoError(t, err, msg)

		data, err := encode(bc)
		require.NoError(t, err, msg)

		v, err := decode[*gad.Bytecode](data, ReadContextWithGoModules(make(GoModules, 0)))
		require.NoError(t, err, msg)

		orig := firstCompiledFunc(bc.Constants)
		got := firstCompiledFunc(v.Constants)
		require.NotNil(t, orig, msg)
		require.NotNil(t, got, msg)

		// the encoded/decoded function must be identical, including ReturnTypes.
		require.Equal(t, orig.ReturnVars, got.ReturnVars, msg)
		require.Equal(t, orig.HeaderString(), got.HeaderString(), msg)
		require.True(t, len(got.HeaderString()) > 0, msg)
		require.Contains(t, got.HeaderString(), c.suffix, msg)

		require.Len(t, got.ReturnVars, len(c.want), msg)
		for j, w := range c.want {
			require.Equal(t, w.name, got.ReturnVars[j].Name, msg)
			require.Equal(t, w.types, got.ReturnVars[j].TypesSymbols.String(), msg)
		}
	}
}
