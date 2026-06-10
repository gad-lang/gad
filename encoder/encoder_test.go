package encoder_test

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
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

func createFiles(t *testing.T, baseDir string, files map[string]string) {
	for file, data := range files {
		path := filepath.Join(baseDir, file)
		err := os.MkdirAll(filepath.Dir(path), 0755)
		require.NoError(t, err)
		err = os.WriteFile(path, []byte(data), 0644)
		require.NoError(t, err)
	}
}
