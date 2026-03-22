package testhelper

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/test_helper/teststrings"
	"github.com/gad-lang/gad/zeroer"
	"github.com/igo9go/go-deepdump/spew"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func VMExpectErrHas(t *testing.T, script string, opts *VMTestOpts, expectMsg string) {
	t.Helper()
	if expectMsg == "" {
		panic("expected message must not be empty")
	}
	VMExpectErrorGen(t, script, opts, func(t *testing.T, retErr error) {
		t.Helper()
		if !strings.Contains(retErr.Error(), expectMsg) {
			require.Failf(t, "expectErrHas Failed",
				"expected error: %v, got: %v", expectMsg, retErr)
		}
	})
}

func VMExpectErrIs(t *testing.T, script string, opts *VMTestOpts, expectErr error) {
	t.Helper()
	VMExpectErrorGen(t, script, opts, func(t *testing.T, retErr error) {
		t.Helper()
		if !errors.Is(retErr, expectErr) {
			if re, ok := retErr.(*gad.RuntimeError); ok {
				if !errors.Is(re.Err, expectErr) {
					if gerr, _ := expectErr.(*gad.Error); gerr != nil {
						if gerr.Error() == re.Err.Error() {
							return
						}
					}
				}
			}
			require.Failf(t, "expectErrorIs Failed",
				"expected error: %v, got: %v", expectErr, retErr)
		}
	})
}

func VMExpectErrAs(t *testing.T, script string, opts *VMTestOpts, asErr any, eqErr any) {
	t.Helper()
	VMExpectErrorGen(t, script, opts, func(t *testing.T, retErr error) {
		t.Helper()
		if !errors.As(retErr, asErr) {
			require.Failf(t, "expectErrorAs Type Failed",
				"expected error type: %T, got: %T(%v)", asErr, retErr, retErr)
		}
		if eqErr != nil && !reflect.DeepEqual(eqErr, asErr) {
			require.Failf(t, "expectErrorAs Equality Failed",
				"errors not equal: %[1]T(%[1]v), got: %[2]T(%[2]v)", eqErr, retErr)
		}
	})
}

func VMExpectErrorGen(
	t *testing.T,
	script string,
	opts *VMTestOpts,
	callback func(*testing.T, error),
) {
	t.Helper()
	if opts == nil {
		opts = NewVMTestOpts()
	}
	type testCase struct {
		name   string
		opts   gad.CompilerOptions
		tracer bytes.Buffer
	}
	testCases := []testCase{
		{
			name: "default",
			opts: gad.CompilerOptions{
				ModuleMap:      opts.GetModuleMap(),
				OptimizeConst:  true,
				TraceParser:    true,
				TraceOptimizer: true,
				TraceCompiler:  true,
			},
		},
		{
			name: "unoptimized",
			opts: gad.CompilerOptions{
				ModuleMap:      opts.GetModuleMap(),
				TraceParser:    true,
				TraceOptimizer: true,
				TraceCompiler:  true,
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

			builtins := gad.NewBuiltins()
			builtins.AppendMap(opts.builtins)
			st := gad.NewSymbolTable(builtins.NameSet)

			co := gad.CompileOptions{
				CompilerOptions: tC.opts,
			}

			if opts.exprToTextFunc != "" {
				tC.opts.MixedExprToTextFunc = &node.IdentExpr{Name: opts.exprToTextFunc}
			}
			if opts.mixed {
				co.ParserOptions.Mode |= parser.ParseMixed
				if opts.mixedDelimiter != nil {
					co.ScannerOptions.MixedDelimiter = *opts.mixedDelimiter
				}
			}

			if opts.compileOptions != nil {
				opts.compileOptions(&co)
			}

			_, compiled, err := gad.Compile(st, []byte(script), co)
			if opts.IsCompilerErr {
				require.Error(t, err)
				callback(t, err)
				return
			}
			require.NoError(t, err)
			_, err = gad.NewVM(builtins.Build(), compiled).SetRecover(opts.IsNoPanic()).RunOpts(&gad.RunOpts{
				Globals:   opts.GetGlobals(),
				Args:      gad.Args{opts.GetArgs()},
				NamedArgs: opts.GetNameArgs(),
			})
			require.Error(t, err)
			callback(t, err)
		})
	}
}

type VMTestOpts struct {
	globals        gad.IndexGetSetter
	args           gad.Array
	namedArgs      *gad.NamedArgs
	moduleMap      *gad.ModuleMap
	Skip2pass      bool
	IsCompilerErr  bool
	noPanic        bool
	stdout         gad.Writer
	builtins       map[string]gad.Object
	exprToTextFunc string
	mixed          bool
	mixedDelimiter *parser.MixedDelimiter
	buffered       bool
	objectToWriter gad.ObjectToWriter
	init           func(opts *VMTestOpts, expect gad.Object) (*VMTestOpts, gad.Object)
	compileOptions func(opts *gad.CompileOptions)
	context        context.Context
}

func NewVMTestOpts() *VMTestOpts {
	return &VMTestOpts{}
}

func (t *VMTestOpts) GetGlobals() gad.IndexGetSetter {
	return t.globals
}

func (t *VMTestOpts) GetArgs() gad.Array {
	return t.args
}

func (t *VMTestOpts) GetNameArgs() *gad.NamedArgs {
	return t.namedArgs
}

func (t *VMTestOpts) GetModuleMap() *gad.ModuleMap {
	return t.moduleMap
}

func (t *VMTestOpts) Out(w io.Writer) *VMTestOpts {
	t.stdout = gad.NewWriter(w)
	return t
}

func (t *VMTestOpts) Globals(globals gad.IndexGetSetter) *VMTestOpts {
	t.globals = globals
	return t
}

func (t *VMTestOpts) Args(args ...gad.Object) *VMTestOpts {
	t.args = args
	return t
}

func (t *VMTestOpts) NamedArgs(args gad.Object) *VMTestOpts {
	switch at := args.(type) {
	case *gad.NamedArgs:
		t.namedArgs = at
	case gad.Dict:
		t.namedArgs = gad.NewNamedArgs(gad.MustConvertToKeyValueArray(nil, at))
	case gad.KeyValueArray:
		t.namedArgs = gad.NewNamedArgs(at)
	}
	return t
}

func (t *VMTestOpts) Init(f func(opts *VMTestOpts, expect gad.Object) (*VMTestOpts, gad.Object)) *VMTestOpts {
	t.init = f
	return t
}

func (t *VMTestOpts) Builtins(m map[string]gad.Object) *VMTestOpts {
	t.builtins = m
	return t
}

func (t *VMTestOpts) Skip2Pass() *VMTestOpts {
	t.Skip2pass = true
	return t
}

func (t *VMTestOpts) CompilerError() *VMTestOpts {
	t.IsCompilerErr = true
	return t
}

func (t *VMTestOpts) NoPanic() *VMTestOpts {
	t.noPanic = true
	return t
}

func (t *VMTestOpts) IsNoPanic() bool {
	return t.noPanic
}

func (t *VMTestOpts) Module(name string, module any) *VMTestOpts {
	if t.moduleMap == nil {
		t.moduleMap = gad.NewModuleMap()
	}
	switch v := module.(type) {
	case []byte:
		t.moduleMap.AddSourceModule(name, v)
	case string:
		t.moduleMap.AddSourceModule(name, []byte(v))
	case gad.Dict:
		t.moduleMap.AddBuiltinModule(name, v)
	case gad.ModuleInitFunc:
		t.moduleMap.AddBuiltinModuleInit(name, v)
	case gad.Importable:
		t.moduleMap.Add(name, v)
	default:
		panic(fmt.Errorf("invalid module type: %T", module))
	}
	return t
}

func (t *VMTestOpts) ExprToTextFunc(name string) *VMTestOpts {
	t.exprToTextFunc = name
	return t
}

func (t *VMTestOpts) WriteObject(o gad.ObjectToWriter) *VMTestOpts {
	t.objectToWriter = o
	return t
}

func (t *VMTestOpts) Mixed(d ...*parser.MixedDelimiter) *VMTestOpts {
	t.mixed = true
	if len(d) > 0 {
		t.mixedDelimiter = d[0]
	}
	return t
}

func (t *VMTestOpts) Buffered() *VMTestOpts {
	t.buffered = true
	return t
}

func (t *VMTestOpts) CompileOptions(f func(opts *gad.CompileOptions)) *VMTestOpts {
	t.compileOptions = f
	return t
}

func (t *VMTestOpts) Context(ctx context.Context) *VMTestOpts {
	t.context = ctx
	return t
}

func VMTestExpectRun(t *testing.T, script string, opts *VMTestOpts, expect gad.Object) {
	t.Helper()

	if opts == nil {
		opts = NewVMTestOpts()
	} else {
		optsCopy := *opts
		opts = &optsCopy
	}

	type testCase struct {
		name   string
		opts   gad.CompileOptions
		tracer bytes.Buffer
	}

	if opts.init == nil {
		opts.init = func(opts *VMTestOpts, ex gad.Object) (*VMTestOpts, gad.Object) {
			return opts, expect
		}
	}

	testCases := []testCase{
		{
			name: "default",
			opts: gad.CompileOptions{
				CompilerOptions: gad.CompilerOptions{
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
			opts: gad.CompileOptions{
				CompilerOptions: gad.CompilerOptions{
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
			builtins := gad.NewBuiltins()
			builtins.AppendMap(opts.builtins)

			st := gad.NewSymbolTable(builtins.NameSet)

			if opts.exprToTextFunc != "" {
				tC.opts.MixedExprToTextFunc = &node.IdentExpr{Name: opts.exprToTextFunc}
			}
			if opts.mixed {
				tC.opts.ParserOptions.Mode |= parser.ParseMixed
				if opts.mixedDelimiter != nil {
					tC.opts.ScannerOptions.MixedDelimiter = *opts.mixedDelimiter
				}
			}

			if opts.compileOptions != nil {
				opts.compileOptions(&tC.opts)
			}

			pf, gotBc, err := gad.Compile(st, []byte(script), tC.opts)
			require.NoError(t, err)
			// create a copy of the bytecode before execution to test bytecode
			// change after execution
			expectBc := *gotBc
			expectBc.Main = gotBc.Main.Copy().(*gad.CompiledFunction)
			expectBc.Constants = gad.Array(gotBc.Constants).Copy().(gad.Array)
			vm := gad.NewVM(builtins.Build(), gotBc)
			var noTrace bool
			defer func() {
				if noTrace {
					return
				}
				if r := recover(); r != nil {
					fmt.Fprintf(os.Stderr, "------- Start Trace -------\n%s"+
						"\n------- End Trace -------\n", tC.tracer.String())
					gotBc.Fprint(vm.Builtins.Builtins(), os.Stderr)

					fmt.Fprintf(os.Stderr, "------- Parsed Code -------\n%s"+
						"\n------- Parsed Code -------\n", teststrings.Indent("\t\t", pf.BuildCode(
						func(ctx *node.CodeWriteContext) {
							ctx.Transpile = gad.TranspileOptions()
						}, node.CodeWithPrefix("\t"))))

					t.Fatalf("panic: %v", r)
				}
			}()

			vm.Builtins = builtins.Build()
			vm.Setup(gad.SetupOpts{
				Context: opts.context,
			})

			opts, expect := opts.init(opts, expect)

			ropts := &gad.RunOpts{
				Globals:        opts.globals,
				Args:           gad.Args{opts.args},
				ObjectToWriter: opts.objectToWriter,
			}
			if opts.namedArgs != nil {
				ropts.NamedArgs = opts.namedArgs.Copy().(*gad.NamedArgs)
			}
			var buf *bytes.Buffer
			if opts.buffered {
				buf = &bytes.Buffer{}
				ropts.StdOut = buf
			} else if opts.stdout != nil {
				ropts.StdOut = opts.stdout
			}
			got, err := vm.SetRecover(opts.noPanic).RunOpts(ropts)
			lines := strings.Split(script, "\n")
			for i, line := range lines {
				lines[i] = fmt.Sprintf("%03d| %s", i+1, line)
			}
			if !assert.NoErrorf(t, err, "Code:\n%s\n", strings.Join(lines, "\n")) {
				gotBc.Fprint(vm.Builtins.Builtins(), os.Stderr)
			}
			if opts.buffered {
				got = gad.Array{got, gad.Str(buf.String())}
			}
			if !reflect.DeepEqual(expect, got) {
				var bcBuf bytes.Buffer
				gotBc.Fprint(vm.Builtins.Builtins(), &bcBuf)

				r1, err := vm.CallBuiltin(gad.BuiltinRepr, gad.Dict{"indent": gad.Yes}.ToNamedArgs(), expect)
				require.NoError(t, err)

				r2, err := vm.CallBuiltin(gad.BuiltinRepr, gad.Dict{"indent": gad.Yes}.ToNamedArgs(), got)
				require.NoError(t, err)

				e, g := r1.ToString(), r2.ToString()

				stmts := make([]any, len(pf.Stmts))
				for i, stmt := range pf.Stmts {
					stmts[i] = stmt
				}
				var sbuf strings.Builder
				cs := &spew.ConfigState{
					SortKeys:                true,
					DisableMethods:          true,
					DisablePointerAddresses: true,
					DisablePointerMethods:   true,
					Indent:                  "\t",
					DisableCapacities:       true,
					FieldFilter: func(structType reflect.Type, field reflect.StructField, value reflect.Value) bool {
						return !zeroer.IsZero(value)
					},
				}
				cs.Fdump(&sbuf, pf.Stmts)

				teststrings.EqualStringf(t, e, g, "Result not equal:\nParsed:\n%s\n\n\tParsed Structure:\n%s\n\n\tBytecode:\n%s",
					teststrings.Indent("\t\t", pf.BuildCode(node.CodeWithPrefix("\t"))),
					teststrings.Indent("\t\t", sbuf.String()),
					teststrings.Indent("\t\t", bcBuf.String()))
				noTrace = true
			}
			gad.TestBytecodesEqual(t, &expectBc, gotBc, true)
		})
	}
}
