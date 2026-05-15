package encoder_test

import (
	"bytes"
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
	var w bytes.Buffer
	err = EncodeObject(&w, v)
	if err != nil {
		return
	}
	return w.Bytes(), nil
}

func decode[t any](b []byte, opt ...ContextOption) (v t, err error) {
	return DecodeT[t](bytes.NewBuffer(b), NewContext(opt...))
}
