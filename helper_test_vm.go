package gad

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"testing"

	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestOpts struct {
	globals        IndexGetSetter
	args           Array
	namedArgs      *NamedArgs
	moduleMap      *ModuleMap
	Skip2pass      bool
	IsCompilerErr  bool
	noPanic        bool
	stdout         Writer
	builtins       map[string]Object
	exprToTextFunc string
	mixed          bool
	buffered       bool
	objectToWriter ObjectToWriter
	init           func(opts *TestOpts, expect Object) (*TestOpts, Object)
}

func NewTestOpts() *TestOpts {
	return &TestOpts{}
}

func (t *TestOpts) GetGlobals() IndexGetSetter {
	return t.globals
}

func (t *TestOpts) GetArgs() Array {
	return t.args
}

func (t *TestOpts) GetNameArgs() *NamedArgs {
	return t.namedArgs
}

func (t *TestOpts) GetModuleMap() *ModuleMap {
	return t.moduleMap
}

func (t *TestOpts) Out(w io.Writer) *TestOpts {
	t.stdout = NewWriter(w)
	return t
}

func (t *TestOpts) Globals(globals IndexGetSetter) *TestOpts {
	t.globals = globals
	return t
}

func (t *TestOpts) Args(args ...Object) *TestOpts {
	t.args = args
	return t
}

func (t *TestOpts) NamedArgs(args Object) *TestOpts {
	switch at := args.(type) {
	case *NamedArgs:
		t.namedArgs = at
	case Dict:
		arr, _ := at.Items(nil)
		t.namedArgs = NewNamedArgs(arr)
	case KeyValueArray:
		t.namedArgs = NewNamedArgs(at)
	}
	return t
}

func (t *TestOpts) Init(f func(opts *TestOpts, expect Object) (*TestOpts, Object)) *TestOpts {
	t.init = f
	return t
}

func (t *TestOpts) Builtins(m map[string]Object) *TestOpts {
	t.builtins = m
	return t
}

func (t *TestOpts) Skip2Pass() *TestOpts {
	t.Skip2pass = true
	return t
}

func (t *TestOpts) CompilerError() *TestOpts {
	t.IsCompilerErr = true
	return t
}

func (t *TestOpts) NoPanic() *TestOpts {
	t.noPanic = true
	return t
}

func (t *TestOpts) IsNoPanic() bool {
	return t.noPanic
}

func (t *TestOpts) Module(name string, module any) *TestOpts {
	if t.moduleMap == nil {
		t.moduleMap = NewModuleMap()
	}
	switch v := module.(type) {
	case []byte:
		t.moduleMap.AddSourceModule(name, v)
	case string:
		t.moduleMap.AddSourceModule(name, []byte(v))
	case map[string]Object:
		t.moduleMap.AddBuiltinModule(name, v)
	case Dict:
		t.moduleMap.AddBuiltinModule(name, v)
	case Importable:
		t.moduleMap.Add(name, v)
	default:
		panic(fmt.Errorf("invalid module type: %T", module))
	}
	return t
}

func (t *TestOpts) ExprToTextFunc(name string) *TestOpts {
	t.exprToTextFunc = name
	return t
}

func (t *TestOpts) WriteObject(o ObjectToWriter) *TestOpts {
	t.objectToWriter = o
	return t
}

func (t *TestOpts) Mixed() *TestOpts {
	t.mixed = true
	return t
}

func (t *TestOpts) Buffered() *TestOpts {
	t.buffered = true
	return t
}

func TestExpectRun(t *testing.T, script string, opts *TestOpts, expect Object) {
	t.Helper()
	if opts == nil {
		opts = NewTestOpts()
	} else {
		optsCopy := *opts
		opts = &optsCopy
	}
	type testCase struct {
		name   string
		opts   CompileOptions
		tracer bytes.Buffer
	}

	if opts.init == nil {
		opts.init = func(opts *TestOpts, ex Object) (*TestOpts, Object) {
			return opts, expect
		}
	}

	testCases := []testCase{
		{
			name: "default",
			opts: CompileOptions{
				CompilerOptions: CompilerOptions{
					ModuleMap:      opts.moduleMap,
					OptimizeConst:  true,
					TraceParser:    true,
					TraceOptimizer: true,
					TraceCompiler:  true,
				},
			},
		},
		{
			name: "unoptimized",
			opts: CompileOptions{
				CompilerOptions: CompilerOptions{
					ModuleMap:      opts.moduleMap,
					TraceParser:    true,
					TraceOptimizer: true,
					TraceCompiler:  true,
				},
			},
		},
	}
	if opts.Skip2pass {
		testCases = testCases[:1]
	}
	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			t.Helper()
			tC.opts.Trace = &tC.tracer // nolint exportloopref
			builtins := NewBuiltins()
			builtins.AppendMap(opts.builtins)
			tC.opts.SymbolTable = NewSymbolTable(builtins)

			if opts.exprToTextFunc != "" {
				tC.opts.MixedExprToTextFunc = &node.IdentExpr{Name: opts.exprToTextFunc}
			}
			if opts.mixed {
				tC.opts.ParserOptions.Mode |= parser.ParseMixed
			}
			gotBc, err := Compile([]byte(script), tC.opts)
			require.NoError(t, err)
			// create a copy of the bytecode before execution to test bytecode
			// change after execution
			expectBc := *gotBc
			expectBc.Main = gotBc.Main.Copy().(*CompiledFunction)
			expectBc.Constants = Array(gotBc.Constants).Copy().(Array)
			vm := NewVM(gotBc)
			defer func() {
				if r := recover(); r != nil {
					fmt.Fprintf(os.Stderr, "------- Start Trace -------\n%s"+
						"\n------- End Trace -------\n", tC.tracer.String())
					gotBc.Fprint(os.Stderr)
					panic(r)
				}
			}()
			vm.Setup(SetupOpts{
				Builtins: tC.opts.SymbolTable.Builtins(),
			})

			opts, expect := opts.init(opts, expect)

			ropts := &RunOpts{
				Globals:        opts.globals,
				Args:           Args{opts.args},
				ObjectToWriter: opts.objectToWriter,
			}
			if opts.namedArgs != nil {
				ropts.NamedArgs = opts.namedArgs.Copy().(*NamedArgs)
			}
			var buf *bytes.Buffer
			if opts.buffered {
				buf = &bytes.Buffer{}
				ropts.StdOut = buf
			} else if opts.stdout != nil {
				ropts.StdOut = opts.stdout
			}
			got, err := vm.SetRecover(opts.noPanic).RunOpts(ropts)
			if !assert.NoErrorf(t, err, "Code:\n%s\n", script) {
				gotBc.Fprint(os.Stderr)
			}
			if opts.buffered {
				got = Array{got, Str(buf.String())}
			}
			if !reflect.DeepEqual(expect, got) {
				var buf bytes.Buffer
				gotBc.Fprint(&buf)
				t.Fatalf("Objects not equal:\nExpected:\n%s\nGot:\n%s\nScript:\n%s\n%s\n",
					tests.Sdump(expect), tests.Sdump(got), script, buf.String())
			}
			TestBytecodesEqual(t, &expectBc, gotBc, true)
		})
	}
}
