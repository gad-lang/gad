package encoder_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"
	gotime "time"

	"github.com/gad-lang/gad"
	"github.com/stretchr/testify/require"

	"github.com/gad-lang/gad/stdlib/fmt"
	"github.com/gad-lang/gad/stdlib/json"
	"github.com/gad-lang/gad/stdlib/strings"
	"github.com/gad-lang/gad/stdlib/time"
	"github.com/gad-lang/gad/tests"

	. "github.com/gad-lang/gad/encoder"
)

var testObjects = []gad.Object{
	gad.Nil,
	gad.Int(-1), gad.Int(0), gad.Int(1),
	gad.Uint(0), ^gad.Uint(0),
	gad.Char('x'),
	gad.Bool(true), gad.Bool(false),
	gad.Float(0), gad.Float(1.2),
	gad.DecimalZero, gad.DecimalFromFloat(gad.Float(1.2)),
	gad.Str(""), gad.Str("abc"),
	gad.Bytes{}, gad.Bytes("foo"),
	gad.ErrIndexOutOfBounds,
	gad.Dict{"key": gad.Str("xxx")},
	&gad.SyncDict{Value: gad.Dict{"k": gad.Str("")}},
	gad.Array{gad.Nil, gad.True, gad.False},
}

func TestBytecode_file(t *testing.T) {
	temp := t.TempDir()

	bc := &gad.Bytecode{Constants: testObjects,
		Main: compFunc(
			[]byte("test instructions"),
			withLocals(4), withParams("*a"),
			withSourceMap(map[int]int{0: 1, 1: 2}),
		),
	}
	f, err := os.CreateTemp(temp, "mod.gadc")
	require.NoError(t, err)
	defer f.Close()

	var ms ModulesSpec
	ms, err = EncodeBytecodeTo(NewWriteContext(context.Background(), NewWriter(f)), bc)
	require.NoError(t, err)

	_, err = f.Seek(0, io.SeekStart)
	require.NoError(t, err)

	got, err := DecodeBytecodeFrom(NewReadContext(NewReader(f), ReadContextWithModules(ms)))
	require.NoError(t, err)
	testBytecodesEqual(t, bc, got)
}

func TestBytecode_full(t *testing.T) {
	src := `
fmt := import("fmt")
strings := import("strings")
time := import("time")
json := import("json")
srcmod := import("srcmod")

v := int(json.Unmarshal(json.Marshal(1)))
v = int(strings.join([v], ""))
v = srcmod.Incr(v)
v = srcmod.Decr(v)
v = int(fmt.sprintf("%d", v))
return int(v*time.Second/time.Second) // 1 (duration/duration is a float ratio)
`

	opts := gad.DefaultCompilerOptions
	opts.ModuleMap = gad.NewModuleMap().
		AddBuiltinModuleInit("fmt", fmt.ModuleInit).
		AddBuiltinModuleInit("strings", strings.ModuleInit).
		AddBuiltinModuleInit("time", time.ModuleInit).
		AddBuiltinModuleInit("json", json.ModuleInit).
		AddSourceModule("srcmod", []byte(`
export{
	Incr: func(x) { return x + 1 },
	Decr: func(x) { return x - 1 },
}
		`))

	bc, err := Compile([]byte(src), opts)
	require.NoError(t, err)

	wantRet, err := NewVM(bc).Run(nil)
	require.NoError(t, err)
	require.Equal(t, gad.Int(1), wantRet)

	temp := t.TempDir()
	f, err := os.CreateTemp(temp, "program.gadc")
	require.NoError(t, err)
	defer f.Close()

	var (
		buf bytes.Buffer
		ms  = GoModulesFromModulesMap(opts.ModuleMap)
	)

	logmicros(t, "encode time: %d microsecs", func() {
		_, err = EncodeBytecodeTo(NewWriteContext(context.Background(), NewWriter(&buf)), bc)
	})
	require.NoError(t, err)

	t.Logf("written size: %v bytes", buf.Len())

	_, err = buf.WriteTo(f)
	require.NoError(t, err)

	_, err = f.Seek(0, io.SeekStart)
	require.NoError(t, err)

	var gotBc *gad.Bytecode
	logmicros(t, "decode time: %d microsecs", func() {
		gotBc, err = DecodeBytecodeFrom(NewReadContext(NewReader(f), ReadContextWithGoModules(ms)))
	})

	require.NoError(t, err)
	require.NotNil(t, gotBc)

	var gotRet gad.Object
	logmicros(t, "run time: %d microsecs", func() {
		gotRet, err = NewVM(gotBc).Run(nil)
	})
	require.NoError(t, err)

	require.Equal(t, wantRet, gotRet)
}

func testBytecodesEqual(t *testing.T, want, got *gad.Bytecode) {
	t.Helper()

	require.Equal(t, want.FileSet, got.FileSet)
	require.Equal(t, want.Main, got.Main)
	require.Equalf(t, want.Constants, got.Constants,
		"expected:%s\nactual:%s", tests.Sdump(want.Constants), tests.Sdump(want.Constants))
	testBytecodeConstants(t, NewVM(got).Init(), want.Constants, got.Constants)
	require.Equal(t, want.NumModules, got.NumModules)
}

func logmicros(t *testing.T, format string, f func()) {
	t0 := gotime.Now()
	f()
	t.Logf(format, gotime.Since(t0).Microseconds())
}

var builtins = gad.NewBuiltins().Build()

func NewSymbolTable() *gad.SymbolTable {
	return gad.NewSymbolTable(builtins.Builtins().NameSet)
}

func Compile(script []byte, opts gad.CompilerOptions) (bc *gad.Bytecode, err error) {
	cr1, err := gad.Compile(NewSymbolTable(), []byte(script), gad.CompileOptions{CompilerOptions: opts})
	bc = cr1.BC()
	return
}

func NewVM(bc *gad.Bytecode) *gad.VM {
	return gad.NewVM(builtins, bc).Init()
}

func TestEnumBytecodeRoundtrip(t *testing.T) {
	src := "enum Perm { Read, Write, Exec = 10, All = Read | Write }\n" +
		"return [Perm.Read.value, Perm.Exec.value, Perm.All.value, Perm.Exec.name]"

	bc, err := Compile([]byte(src), gad.CompilerOptions{})
	require.NoError(t, err)
	wantRet, err := NewVM(bc).Run(nil)
	require.NoError(t, err)

	var buf bytes.Buffer
	ms, err := EncodeBytecodeTo(NewWriteContext(context.Background(), NewWriter(&buf)), bc)
	require.NoError(t, err)

	gotBc, err := DecodeBytecodeFrom(NewReadContext(NewReader(bytes.NewReader(buf.Bytes())), ReadContextWithModules(ms)))
	require.NoError(t, err)

	gotRet, err := NewVM(gotBc).Run(nil)
	require.NoError(t, err)
	require.Equal(t, wantRet, gotRet)
}

func TestMetaTypeBytecodeRoundtrip(t *testing.T) {
	// A `type<X>` parameter compiles to a MetaType constant carrying X's symbol;
	// it must survive an encode/decode round-trip and still dispatch on the type
	// value after decode.
	src := "class Rect {}\nclass Point {}\n" +
		"func d { (t type<Rect>) => 1; (t type<Point>) => 2 }\n" +
		"return [d(Rect), d(Point)]"

	bc, err := Compile([]byte(src), gad.CompilerOptions{})
	require.NoError(t, err)
	wantRet, err := NewVM(bc).Run(nil)
	require.NoError(t, err)
	require.Equal(t, gad.Array{gad.Int(1), gad.Int(2)}, wantRet)

	var buf bytes.Buffer
	ms, err := EncodeBytecodeTo(NewWriteContext(context.Background(), NewWriter(&buf)), bc)
	require.NoError(t, err)

	gotBc, err := DecodeBytecodeFrom(NewReadContext(NewReader(bytes.NewReader(buf.Bytes())), ReadContextWithModules(ms)))
	require.NoError(t, err)

	gotRet, err := NewVM(gotBc).Run(nil)
	require.NoError(t, err)
	require.Equal(t, wantRet, gotRet)
}

// TestIncludeBytecodeRoundtrip verifies bytecode built from an `include`
// (inlined statements wrapped in OpPushSource/OpPopSource, whose
// source name is a Str constant) survives an encode/decode round-trip: `@file`
// still reports the included source after decode.
func TestIncludeBytecodeRoundtrip(t *testing.T) {
	mm := gad.NewModuleMap()
	mm.AddSourceModule("name.gad", []byte(`x := 40; got := @file`))

	src := `include ("name.gad"); return [x + 2, got]`

	bc, err := Compile([]byte(src), gad.CompilerOptions{ModuleMap: mm})
	require.NoError(t, err)
	wantRet, err := NewVM(bc).Run(nil)
	require.NoError(t, err)
	require.Equal(t, gad.Array{gad.Int(42), gad.Str("name.gad")}, wantRet)

	var buf bytes.Buffer
	ms, err := EncodeBytecodeTo(NewWriteContext(context.Background(), NewWriter(&buf)), bc)
	require.NoError(t, err)

	gotBc, err := DecodeBytecodeFrom(NewReadContext(NewReader(bytes.NewReader(buf.Bytes())), ReadContextWithModules(ms)))
	require.NoError(t, err)

	gotRet, err := NewVM(gotBc).Run(nil)
	require.NoError(t, err)
	require.Equal(t, wantRet, gotRet)
}

// TestFuncMetadataBytecodeRoundtrip verifies a function's `[k=v, …]` metadata
// (CompiledFunction.Meta) survives an encode/decode round-trip: `fn.@meta` still
// reports it after decode.
func TestFuncMetadataBytecodeRoundtrip(t *testing.T) {
	src := "[route=\"/save\", auth]\nfunc save(x) { return x }\nreturn str(save.@meta)"

	bc, err := Compile([]byte(src), gad.CompilerOptions{})
	require.NoError(t, err)
	wantRet, err := NewVM(bc).Run(nil)
	require.NoError(t, err)
	require.Equal(t, gad.Str(`(;route="/save", auth)`), wantRet)

	var buf bytes.Buffer
	ms, err := EncodeBytecodeTo(NewWriteContext(context.Background(), NewWriter(&buf)), bc)
	require.NoError(t, err)

	gotBc, err := DecodeBytecodeFrom(NewReadContext(NewReader(bytes.NewReader(buf.Bytes())), ReadContextWithModules(ms)))
	require.NoError(t, err)

	gotRet, err := NewVM(gotBc).Run(nil)
	require.NoError(t, err)
	require.Equal(t, wantRet, gotRet)
}

// TestInterfaceSliceBytecodeRoundtrip verifies an array interface survives an
// encode/decode round-trip: its depth, leaf element types, `**rest`, metadata and
// field nullability/metadata (all encoded with the interface constant).
func TestInterfaceSliceBytecodeRoundtrip(t *testing.T) {
	src := `
		sat := func(v, T) { try { v :: T; return true } catch { return false } }
		[kind="nums"]
		interface numerics []<int|float>
		interface pts [] { [db=(;pk)]
			x int
			y? int }
		return [sat([1, 2.5], numerics), sat([1, "x"], numerics), sat([{x: 1}], pts),
			sat([{x: "a"}], pts), numerics.@depth, str(numerics.@meta), str(pts.x.@meta)]`

	bc, err := Compile([]byte(src), gad.CompilerOptions{})
	require.NoError(t, err)
	wantRet, err := NewVM(bc).Run(nil)
	require.NoError(t, err)
	require.Equal(t, gad.Array{gad.True, gad.False, gad.True, gad.False, gad.Int(1),
		gad.Str(`(;kind="nums")`), gad.Str(`(;db=(;pk))`)}, wantRet)

	var buf bytes.Buffer
	ms, err := EncodeBytecodeTo(NewWriteContext(context.Background(), NewWriter(&buf)), bc)
	require.NoError(t, err)
	gotBc, err := DecodeBytecodeFrom(NewReadContext(NewReader(bytes.NewReader(buf.Bytes())), ReadContextWithModules(ms)))
	require.NoError(t, err)
	gotRet, err := NewVM(gotBc).Run(nil)
	require.NoError(t, err)
	require.Equal(t, wantRet, gotRet)
}

// TestTypedArrayBytecodeRoundtrip verifies a typed array type declaration (with
// metadata and a member body) runs the same after an encode/decode round-trip.
func TestTypedArrayBytecodeRoundtrip(t *testing.T) {
	src := `
		[unit="m"]
		type nums []int {
			label = "x"
			methods { sum() { s := 0; for v in this { s += v }; return s } }
		}
		n := nums(1, 2, 3)
		return [str(n), n.sum(), str(nums.@meta)]`

	bc, err := Compile([]byte(src), gad.CompilerOptions{})
	require.NoError(t, err)
	wantRet, err := NewVM(bc).Run(nil)
	require.NoError(t, err)
	require.Equal(t, gad.Array{gad.Str(`nums[1, 2, 3]{label: "x"}`), gad.Int(6), gad.Str(`(;unit="m")`)}, wantRet)

	var buf bytes.Buffer
	ms, err := EncodeBytecodeTo(NewWriteContext(context.Background(), NewWriter(&buf)), bc)
	require.NoError(t, err)
	gotBc, err := DecodeBytecodeFrom(NewReadContext(NewReader(bytes.NewReader(buf.Bytes())), ReadContextWithModules(ms)))
	require.NoError(t, err)
	gotRet, err := NewVM(gotBc).Run(nil)
	require.NoError(t, err)
	require.Equal(t, wantRet, gotRet)
}

// TestInterfaceExtendsBytecodeRoundtrip verifies interface parents bound at run
// time (OpInterfaceExtends) — a single parent, a literal list and a variable
// holding one — behave the same after an encode/decode round-trip.
func TestInterfaceExtendsBytecodeRoundtrip(t *testing.T) {
	src := `
		sat := func(v, T) { try { v :: T; return true } catch { return false } }
		interface A { a int }
		interface B { b int }
		ps := [A, B]
		interface I1 { *A }
		interface I2 { *[A, B] }
		interface I3 { *ps; c int }
		return [sat({a: 1}, I1), sat({}, I1), sat({a: 1, b: 2}, I2), sat({a: 1}, I2),
			sat({a: 1, b: 2, c: 3}, I3), sat({a: 1, b: 2}, I3)]`

	bc, err := Compile([]byte(src), gad.CompilerOptions{})
	require.NoError(t, err)
	wantRet, err := NewVM(bc).Run(nil)
	require.NoError(t, err)
	require.Equal(t, gad.Array{gad.True, gad.False, gad.True, gad.False, gad.True, gad.False}, wantRet)

	var buf bytes.Buffer
	ms, err := EncodeBytecodeTo(NewWriteContext(context.Background(), NewWriter(&buf)), bc)
	require.NoError(t, err)
	gotBc, err := DecodeBytecodeFrom(NewReadContext(NewReader(bytes.NewReader(buf.Bytes())), ReadContextWithModules(ms)))
	require.NoError(t, err)
	gotRet, err := NewVM(gotBc).Run(nil)
	require.NoError(t, err)
	require.Equal(t, wantRet, gotRet)
}

// TestStructuralTypesBytecodeRoundtrip verifies the structural type constants a
// type position compiles to — an array type (`[]int`) and a pointer type
// (`*int`, `*<int|str>`) — survive an encode/decode round-trip, and that `&` and
// `.v` (the pointer opcodes) run the same afterwards.
func TestStructuralTypesBytecodeRoundtrip(t *testing.T) {
	src := `
		sat := func(f) { try { f(); return true } catch { return false } }
		count := func(xs []int) => len(xs)
		bump := func(p *int) { p.v += 1 }
		kind := func(v *<int|str>) => typeName(v.v)
		n := 1
		bump(&n)
		s := "a"
		return [count([1, 2]), n, kind(&s), sat(() => count(["x"])), sat(() => bump(&s))]`

	bc, err := Compile([]byte(src), gad.CompilerOptions{})
	require.NoError(t, err)
	wantRet, err := NewVM(bc).Run(nil)
	require.NoError(t, err)
	require.Equal(t, gad.Array{gad.Int(2), gad.Int(2), gad.Str("str"), gad.False, gad.False}, wantRet)

	var buf bytes.Buffer
	ms, err := EncodeBytecodeTo(NewWriteContext(context.Background(), NewWriter(&buf)), bc)
	require.NoError(t, err)
	gotBc, err := DecodeBytecodeFrom(NewReadContext(NewReader(bytes.NewReader(buf.Bytes())), ReadContextWithModules(ms)))
	require.NoError(t, err)
	gotRet, err := NewVM(gotBc).Run(nil)
	require.NoError(t, err)
	require.Equal(t, wantRet, gotRet)
}
