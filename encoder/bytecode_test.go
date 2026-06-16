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
v = int(strings.Join([v], ""))
v = srcmod.Incr(v)
v = srcmod.Decr(v)
v = int(fmt.Sprintf("%d", v))
return v*time.second/time.second // 1
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
	_, bc, err = gad.Compile(NewSymbolTable(), []byte(script), gad.CompileOptions{CompilerOptions: opts})
	return
}

func NewVM(bc *gad.Bytecode) *gad.VM {
	return gad.NewVM(builtins, bc).Init()
}
