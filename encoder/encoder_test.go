package encoder_test

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"testing"
	gotime "time"

	"github.com/gad-lang/gad"
	. "github.com/gad-lang/gad/encoder"
	"github.com/stretchr/testify/require"
)

type funcOpt func(*gad.CompiledFunction)

func withParams(names ...string) funcOpt {
	return func(cf *gad.CompiledFunction) {
		cf.WithParams(names...)
	}
}

func withLocals(numLocals int) funcOpt {
	return func(cf *gad.CompiledFunction) {
		cf.NumLocals = numLocals
	}
}

func withSourceMap(m map[int]int) funcOpt {
	return func(cf *gad.CompiledFunction) {
		cf.SourceMap = m
	}
}

func compFunc(insts []byte, opts ...funcOpt) *gad.CompiledFunction {
	cf := &gad.CompiledFunction{
		Instructions: insts,
	}
	for _, f := range opts {
		f(cf)
	}
	return cf
}

func makeInst(op gad.Opcode, args ...int) []byte {
	b, err := gad.MakeInstruction(make([]byte, 8), op, args...)
	if err != nil {
		panic(err)
	}
	return b
}

func concatInsts(insts ...[]byte) []byte {
	var out []byte
	for i := range insts {
		out = append(out, insts[i]...)
	}
	return out
}

type testopts struct {
	globals       gad.IndexGetSetter
	args          []gad.Object
	namedArgs     gad.Dict
	moduleMap     *gad.ModuleMap
	skip2pass     bool
	isCompilerErr bool
	noPanic       bool
}

func newOpts() *testopts {
	return &testopts{}
}

func (t *testopts) Globals(globals gad.IndexGetSetter) *testopts {
	t.globals = globals
	return t
}

func (t *testopts) Args(args ...gad.Object) *testopts {
	t.args = args
	return t
}

func (t *testopts) Skip2Pass() *testopts {
	t.skip2pass = true
	return t
}

func (t *testopts) CompilerError() *testopts {
	t.isCompilerErr = true
	return t
}

func (t *testopts) NoPanic() *testopts {
	t.noPanic = true
	return t
}

func (t *testopts) Module(name string, module any) *testopts {
	if t.moduleMap == nil {
		t.moduleMap = gad.NewModuleMap()
	}
	switch v := module.(type) {
	case []byte:
		t.moduleMap.AddSourceModule(name, v)
	case string:
		t.moduleMap.AddSourceModule(name, []byte(v))
	case map[string]gad.Object:
		t.moduleMap.AddBuiltinModule(name, v)
	case gad.Dict:
		t.moduleMap.AddBuiltinModule(name, v)
	case gad.Importable:
		t.moduleMap.Add(name, v)
	default:
		panic(fmt.Errorf("invalid module type: %T", module))
	}
	return t
}

func testBytecodeConstants(t *testing.T, vm *gad.VM, expected, decoded []gad.Object) {
	t.Helper()
	if len(decoded) != len(expected) {
		t.Fatalf("constants length not equal want %d, got %d", len(decoded), len(expected))
	}

	for i := range decoded {
		require.Equalf(t, expected[i], decoded[i],
			"constant index %d not equal want %v, got %v", i, expected[i], decoded[i])
		require.NotNil(t, decoded[i])
	}
}

const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand = rand.New(
	rand.NewSource(gotime.Now().UnixNano()))

func randStringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func randString(length int) string {
	return randStringWithCharset(length, charset)
}

func encode(v any) (b []byte, err error) {
	var (
		w bytes.Buffer
	)
	err = EncodeObject(NewWriteContext(context.Background(), NewWriter(&w)), v)
	if err != nil {
		return
	}
	return w.Bytes(), nil
}

func eencode(v any) (b, eb []byte, err error) {
	var (
		w  bytes.Buffer
		ew bytes.Buffer
	)
	err = EncodeObject(NewWriteContext(context.Background(), NewWriter(&w), WriteContextWithEmbededWriter(NewWriter(&ew))), v)
	if err != nil {
		return
	}
	return w.Bytes(), ew.Bytes(), nil
}

func decode[t any](b []byte, opt ...ReadContextOption) (v t, err error) {
	return DecodeT[t](NewReadContext(NewReader(bytes.NewReader(b)), opt...))
}

func edecode[t any](b, eb []byte, opt ...ReadContextOption) (v t, err error) {
	opt = append(opt, ReadContextWithEmbeddedReader(bytes.NewReader(eb)))
	return DecodeT[t](NewReadContext(NewReader(bytes.NewReader(b)), opt...))
}

func TestEnumEncoding(t *testing.T) {
	e := gad.NewEnum("Perm", nil)
	e.AddValue("Read", gad.Uint(1))
	e.AddValue("Write", gad.Uint(2))
	e.AddValue("Exec", gad.Int(10))

	b, err := encode(e)
	require.NoError(t, err)

	got, err := decode[*gad.Enum](b)
	require.NoError(t, err)
	require.Equal(t, "Perm", got.Name())

	arr := got.ToArray()
	require.Len(t, arr, 3)
	wantNames := []string{"Read", "Write", "Exec"}
	wantVals := []gad.Object{gad.Uint(1), gad.Uint(2), gad.Int(10)}
	for i, v := range arr {
		ev := v.(*gad.EnumValue)
		require.Equal(t, wantNames[i], ev.Name)
		require.Equal(t, i, ev.Index)
		require.True(t, wantVals[i].Equal(ev.Value), "value %d: want %v got %v", i, wantVals[i], ev.Value)
	}
}

func TestFuncHeaderObjectEncoding(t *testing.T) {
	// A func-header compiled to a constant: types are stored as symbols and the
	// header carries its module for the module-qualified FullName.
	h := &gad.FuncHeaderObject{
		FuncName: "fh#1",
		Module:   gad.NewModuleSpecFromName("mymod"),
		Params: gad.Array{
			&gad.TypedIdent{Name: "a", TypesSymbols: gad.ParamType{{Name: "int", Index: 3, Scope: gad.ScopeBuiltin}}},
			&gad.TypedIdent{Name: "b", TypesSymbols: gad.ParamType{{Name: "any", Index: 0, Scope: gad.ScopeBuiltin}}},
		},
		Return: gad.Array{
			&gad.TypedIdent{Name: "r", TypesSymbols: gad.ParamType{{Name: "str", Index: 11, Scope: gad.ScopeBuiltin}}},
		},
	}

	b, err := encode(h)
	require.NoError(t, err)

	got, err := decode[*gad.FuncHeaderObject](b)
	require.NoError(t, err)

	require.Equal(t, "fh#1", got.FuncName)
	require.NotNil(t, got.Module)
	require.Equal(t, "mymod", got.Module.Name)
	require.Equal(t, "mymod.fh#1", got.FullName())

	require.Len(t, got.Params, 2)
	p0 := got.Params[0].(*gad.TypedIdent)
	require.Equal(t, "a", p0.Name)
	require.Len(t, p0.TypesSymbols, 1)
	require.Equal(t, "int", p0.TypesSymbols[0].Name)
	require.Equal(t, gad.ScopeBuiltin, p0.TypesSymbols[0].Scope)

	require.Len(t, got.Return, 1)
	require.Equal(t, "r", got.Return[0].(*gad.TypedIdent).Name)
	// the whole header round-trips equal
	require.True(t, h.Equal(got), "want %v got %v", h, got)
}

func TestMethodInterfaceEncoding(t *testing.T) {
	mi := &gad.MethodInterface{
		MIName: "meti#1",
		Headers: []*gad.FuncHeaderObject{
			{FuncName: "fh#1", Params: gad.Array{
				&gad.TypedIdent{Name: "_", TypesSymbols: gad.ParamType{{Name: "int", Index: 3, Scope: gad.ScopeBuiltin}}},
			}},
			{FuncName: "fh#2", Return: gad.Array{
				&gad.TypedIdent{Name: "r", TypesSymbols: gad.ParamType{{Name: "str", Index: 11, Scope: gad.ScopeBuiltin}}},
			}},
		},
	}
	b, err := encode(mi)
	require.NoError(t, err)
	got, err := decode[*gad.MethodInterface](b)
	require.NoError(t, err)
	require.Equal(t, "meti#1", got.MIName)
	require.Len(t, got.Headers, 2)
	require.Equal(t, "fh#1", got.Headers[0].FuncName)
	require.Equal(t, "int", got.Headers[0].Params[0].(*gad.TypedIdent).TypesSymbols[0].Name)
	require.True(t, mi.Equal(got))
}

func TestInterfaceEncoding(t *testing.T) {
	getter := &gad.FuncHeaderObject{FuncName: "area", Return: gad.Array{
		&gad.TypedIdent{Name: "_", TypesSymbols: gad.ParamType{{Name: "int", Index: 3, Scope: gad.ScopeBuiltin}}},
	}}
	i := &gad.Interface{
		IName:   "Shape",
		Module:  gad.NewModuleSpecFromName("mymod"),
		Extends: gad.ParamType{{Name: "Base", Index: 5, Scope: gad.ScopeGlobal}},
		Fields: []*gad.InterfaceField{
			{Name: "id", TypesSymbols: gad.ParamType{{Name: "int", Index: 3, Scope: gad.ScopeBuiltin}}},
		},
		Props: []*gad.InterfaceProp{
			{Name: "area", Getter: getter},
		},
		Methods: []*gad.InterfaceMethod{
			{Name: "draw", Headers: []*gad.FuncHeaderObject{{FuncName: "draw#1"}}},
		},
	}
	b, err := encode(i)
	require.NoError(t, err)
	got, err := decode[*gad.Interface](b)
	require.NoError(t, err)

	require.Equal(t, "mymod.Shape", got.FullName())
	require.Len(t, got.Extends, 1)
	require.Equal(t, "Base", got.Extends[0].Name)
	require.Len(t, got.Fields, 1)
	require.Equal(t, "id", got.Fields[0].Name)
	require.Same(t, got, got.Fields[0].Iface) // back-ref restored
	require.Len(t, got.Props, 1)
	require.Equal(t, "area", got.Props[0].Getter.FuncName)
	require.Same(t, got, got.Props[0].Iface)
	require.Len(t, got.Methods, 1)
	require.Equal(t, "draw", got.Methods[0].Name)
	require.Same(t, got, got.Methods[0].Iface)
	require.True(t, i.Equal(got))
}
