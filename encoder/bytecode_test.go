package encoder_test

import (
	"bytes"
	"io"
	"io/ioutil"
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

var baz gad.Object = gad.Str("baz")
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
	&gad.RuntimeError{Err: gad.ErrInvalidIndex},
	gad.Dict{"key": &gad.Function{Name: "f"}},
	&gad.SyncMap{Value: gad.Dict{"k": gad.Str("")}},
	gad.Array{gad.Nil, gad.True, gad.False},
	&time.Time{Value: gotime.Time{}},
	&json.EncoderOptions{Value: gad.Int(1)},
	&json.RawMessage{Value: gad.Bytes("bar")},
	&gad.ObjectPtr{Value: &baz},
}

func TestBytecode_Encode(t *testing.T) {
	testBytecodeSerialization(t, &gad.Bytecode{Main: compFunc(nil)}, nil)

	testBytecodeSerialization(t,
		&gad.Bytecode{Constants: testObjects,
			Main: compFunc(
				[]byte("test instructions"),
				withLocals(1), withParams("a"), withVarParams(),
			),
		},
		nil,
	)
}

func TestBytecode_file(t *testing.T) {
	temp := t.TempDir()

	bc := &gad.Bytecode{Constants: testObjects,
		Main: compFunc(
			[]byte("test instructions"),
			withLocals(4), withParams(), withVarParams(),
			withSourceMap(map[int]int{0: 1, 1: 2}),
		),
	}
	f, err := ioutil.TempFile(temp, "mod.gadc")
	require.NoError(t, err)
	defer f.Close()

	err = EncodeBytecodeTo(bc, f)
	require.NoError(t, err)

	_, err = f.Seek(0, io.SeekStart)
	require.NoError(t, err)

	got, err := DecodeBytecodeFrom(f, nil)
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
return v*time.Second/time.Second // 1
`

	opts := gad.DefaultCompilerOptions
	opts.ModuleMap = gad.NewModuleMap().
		AddBuiltinModule("fmt", fmt.Module).
		AddBuiltinModule("strings", strings.Module).
		AddBuiltinModule("time", time.Module).
		AddBuiltinModule("json", json.Module).
		AddSourceModule("srcmod", []byte(`
return {
	Incr: func(x) { return x + 1 },
	Decr: func(x) { return x - 1 },
}
		`))

	mmCopy := opts.ModuleMap.Copy()

	bc, err := gad.Compile([]byte(src), gad.CompileOptions{CompilerOptions: opts})
	require.NoError(t, err)

	wantRet, err := gad.NewVM(bc).Run(nil)
	require.NoError(t, err)
	require.Equal(t, gad.Int(1), wantRet)

	temp := t.TempDir()
	f, err := os.CreateTemp(temp, "program.gadc")
	require.NoError(t, err)
	defer f.Close()

	var buf bytes.Buffer

	logmicros(t, "encode time: %d microsecs", func() {
		err = EncodeBytecodeTo(bc, &buf)
	})
	require.NoError(t, err)

	t.Logf("written size: %v bytes", buf.Len())

	_, err = buf.WriteTo(f)
	require.NoError(t, err)

	_, err = f.Seek(0, io.SeekStart)
	require.NoError(t, err)

	var gotBc *gad.Bytecode
	logmicros(t, "decode time: %d microsecs", func() {
		gotBc, err = DecodeBytecodeFrom(f, mmCopy)
	})
	require.NoError(t, err)
	require.NotNil(t, gotBc)

	var gotRet gad.Object
	logmicros(t, "run time: %d microsecs", func() {
		gotRet, err = gad.NewVM(gotBc).Run(nil)
	})
	require.NoError(t, err)

	require.Equal(t, wantRet, gotRet)
}

func testBytecodeSerialization(t *testing.T, b *gad.Bytecode, modules *gad.ModuleMap) {
	t.Helper()

	var buf bytes.Buffer
	err := (*Bytecode)(b).Encode(&buf)
	require.NoError(t, err)

	r := &gad.Bytecode{}
	err = (*Bytecode)(r).Decode(bytes.NewReader(buf.Bytes()), modules)
	require.NoError(t, err)

	testBytecodesEqual(t, b, r)
}

func testBytecodesEqual(t *testing.T, want, got *gad.Bytecode) {
	t.Helper()

	require.Equal(t, want.FileSet, got.FileSet)
	require.Equal(t, want.Main, got.Main)
	require.Equalf(t, want.Constants, got.Constants,
		"expected:%s\nactual:%s", tests.Sdump(want.Constants), tests.Sdump(want.Constants))
	testBytecodeConstants(t, want.Constants, got.Constants)
	require.Equal(t, want.NumModules, got.NumModules)
}

func logmicros(t *testing.T, format string, f func()) {
	t0 := gotime.Now()
	f()
	t.Logf(format, gotime.Since(t0).Microseconds())
}
