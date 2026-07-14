package gad_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gad-lang/gad/importers"
	"github.com/gad-lang/gad/parser"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/gad-lang/gad/token"

	. "github.com/gad-lang/gad"
)

func makeInst(op Opcode, args ...int) []byte {
	b, err := MakeInstruction(make([]byte, 8), op, args...)
	if err != nil {
		panic(err)
	}
	return b
}

type bytecodeOption func(*Bytecode)

type ModuleSpecBuilder struct {
	bc *Bytecode
}

func (b *ModuleSpecBuilder) Name(name string) {
	b.bc.Modules = append(b.bc.Modules, &ModuleSpec{
		ModuleInfo: ModuleInfo{Name: name, URL: name},
		InitGoFunc: func(module *Module) CallerObject { return nil },
	})
}

func (b *ModuleSpecBuilder) Compiled(name string, f *CompiledFunction) {
	spec := &ModuleSpec{
		ModuleInfo:       ModuleInfo{Name: name, URL: name},
		InitCompiledFunc: f,
	}
	f.SetModule(spec)
	b.bc.Modules = append(b.bc.Modules, spec)
}

func (b *ModuleSpecBuilder) NameFile(name, file string) {
	b.bc.Modules = append(b.bc.Modules, &ModuleSpec{
		ModuleInfo: ModuleInfo{Name: name, URL: file},
		InitGoFunc: func(module *Module) CallerObject { return nil },
	})
}

func withModules(f func(b *ModuleSpecBuilder)) bytecodeOption {
	return func(bc *Bytecode) {
		f(&ModuleSpecBuilder{bc: bc})
	}
}

var mainModule = &ModuleSpec{ModuleInfo: ModuleInfo{Name: MainName, URL: MainName}}

func bytecode(
	consts []Object,
	cf *CompiledFunction,
	opts ...bytecodeOption,
) *Bytecode {
	if cf.GetModule() == nil {
		cf.SetModule(mainModule)
	}
	cf.FuncName = "#main"
	bc := &Bytecode{
		Constants: consts,
		Main:      cf,
	}

	for _, obj := range consts {
		if cf, _ := obj.(*CompiledFunction); cf != nil && cf.GetModule() == nil {
			cf.SetModule(mainModule)
		}
	}

	for _, f := range opts {
		f(bc)
	}
	return bc
}

type funcOpt func(*CompiledFunction)

func funcName(name string) funcOpt {
	return func(cf *CompiledFunction) {
		cf.FuncName = name
	}
}

func funcParams(names ...string) funcOpt {
	return func(cf *CompiledFunction) {
		cf.WithParams(names...)
	}
}

func funcNamedParams(params ...*NamedParam) funcOpt {
	return func(cf *CompiledFunction) {
		cf.SetNamedParams(params...)
	}
}

func funcLocals(numLocals int) funcOpt {
	return func(cf *CompiledFunction) {
		cf.NumLocals = numLocals
	}
}

func compFunc(insts []byte, opts ...funcOpt) *CompiledFunction {
	cf := &CompiledFunction{
		Instructions: insts,
	}
	for _, f := range opts {
		f(cf)
	}
	return cf
}

func concatInsts(insts ...[]byte) []byte {
	var out []byte
	for i := range insts {
		out = append(out, insts[i]...)
	}
	return out
}

func TestCompiler_BuiltinModuleSelector(t *testing.T) {
	// `module.NAME` for a builtin module namespace compiles to a single
	// OpGetBuiltin (the qualified builtin), with no namespace dict load + index.
	expectCompile(t, `base64.StdEncoding`, bytecode(
		Array{},
		compFunc(concatInsts(
			makeInst(OpGetBuiltin, int(BuiltinsMap["base64.StdEncoding"])),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	// a shadowing local `fmt` disables the optimization: the member is read by
	// indexing the local value, not via OpGetBuiltin.
	expectCompile(t, `fmt := {Print: 1}; return fmt.Print`, bytecode(
		Array{Str("Print"), Int(1)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpConstant, 1),
			makeInst(OpDict, 2),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetLocal, 0),
			makeInst(OpConstant, 0),
			makeInst(OpGetIndex, 1),
			makeInst(OpReturn, 1),
		), funcLocals(1)),
	))
}

func TestCompiler_CompileBlock(t *testing.T) {
	expectCompile(t, `1`, bytecode(
		Array{Int(1)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `{ 1 }`, bytecode(
		Array{Int(1)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `var x; { var x }`, bytecode(
		Array{},
		compFunc(concatInsts(
			makeInst(OpNil),
			makeInst(OpDefineLocal, 0),
			makeInst(OpNil),
			makeInst(OpDefineLocal, 1),
			makeInst(OpReturn, 0),
		), funcLocals(2)),
	))

	expectCompileError(t, `var x; { var z; var z }`, `Compile Error: "z" redeclared in this block`)
}

func TestCompiler_CompilePipe(t *testing.T) {
	expectCompile(t, `"a".|filter`, bytecode(
		Array{Str("a")},
		compFunc(concatInsts(
			makeInst(OpGetBuiltin, int(BuiltinFilter)),
			makeInst(OpConstant, 0),
			makeInst(OpCall, 1, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))
	expectCompile(t, `var a; a.|filter`, bytecode(
		Array{},
		compFunc(concatInsts(
			makeInst(OpNil),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetBuiltin, int(BuiltinFilter)),
			makeInst(OpGetLocal, 0),
			makeInst(OpCall, 1, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		),
			funcLocals(1)),
	))
	expectCompile(t, `global (a, x); a.|filter(x)`, bytecode(
		Array{Str("a"), Str("x")},
		compFunc(concatInsts(
			makeInst(OpGetBuiltin, int(BuiltinFilter)),
			makeInst(OpGetGlobal, 0),
			makeInst(OpGetGlobal, 1),
			makeInst(OpCall, 2, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		),
			funcLocals(0)),
	))
	expectCompile(t, `global (a, x); a.|map(x)`, bytecode(
		Array{Str("a"), Str("x")},
		compFunc(concatInsts(
			makeInst(OpGetBuiltin, int(BuiltinMap)),
			makeInst(OpGetGlobal, 0),
			makeInst(OpGetGlobal, 1),
			makeInst(OpCall, 2, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		),
			funcLocals(0)),
	))
	expectCompile(t, `global (a, x); a.|reduce(x)`, bytecode(
		Array{Str("a"), Str("x")},
		compFunc(concatInsts(
			makeInst(OpGetBuiltin, int(BuiltinReduce)),
			makeInst(OpGetGlobal, 0),
			makeInst(OpGetGlobal, 1),
			makeInst(OpCall, 2, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		),
			funcLocals(0)),
	))
	expectCompile(t, `var x; [].|x()`, bytecode(
		Array{},
		compFunc(concatInsts(
			makeInst(OpNil),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetLocal, 0),
			makeInst(OpArray, 0),
			makeInst(OpCall, 1, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		),
			funcLocals(1)),
	))
}

func TestCompiler_CompileIfNull(t *testing.T) {
	expectCompile(t, `var a; return ((a == nil)) ? 10 : 20`, bytecode(
		Array{Int(10), Int(20)},
		compFunc(concatInsts(
			makeInst(OpNil),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetLocal, 0),
			makeInst(OpJumpNotNil, 14),
			makeInst(OpConstant, 0),
			makeInst(OpJump, 17),
			makeInst(OpConstant, 1),
			makeInst(OpReturn, 1),
		),
			funcLocals(1)),
	))

	expectCompile(t, `var a; return a == nil ? 10 : 20`, bytecode(
		Array{Int(10), Int(20)},
		compFunc(concatInsts(
			makeInst(OpNil),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetLocal, 0),
			makeInst(OpJumpNotNil, 14),
			makeInst(OpConstant, 0),
			makeInst(OpJump, 17),
			makeInst(OpConstant, 1),
			makeInst(OpReturn, 1),
		),
			funcLocals(1)),
	))

	expectCompile(t, `var a; return a != nil ? 10 : 20`, bytecode(
		Array{Int(10), Int(20)},
		compFunc(concatInsts(
			makeInst(OpNil),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetLocal, 0),
			makeInst(OpJumpNil, 14),
			makeInst(OpConstant, 0),
			makeInst(OpJump, 17),
			makeInst(OpConstant, 1),
			makeInst(OpReturn, 1),
		),
			funcLocals(1)),
	))

	expectCompile(t, `var a; if (((a == nil))) { 10 } else { 20 }; 3333;`, bytecode(
		Array{Int(10), Int(20), Int(3333)},
		compFunc(concatInsts(
			makeInst(OpNil),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetLocal, 0),
			makeInst(OpJumpNotNil, 15),
			makeInst(OpConstant, 0),
			makeInst(OpPop),
			makeInst(OpJump, 19),
			makeInst(OpConstant, 1),
			makeInst(OpPop),
			makeInst(OpConstant, 2),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		),
			funcLocals(1)),
	))

	expectCompile(t, `var a; if a == nil { 10 } else { 20 }; 3333;`, bytecode(
		Array{Int(10), Int(20), Int(3333)},
		compFunc(concatInsts(
			makeInst(OpNil),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetLocal, 0),
			makeInst(OpJumpNotNil, 15),
			makeInst(OpConstant, 0),
			makeInst(OpPop),
			makeInst(OpJump, 19),
			makeInst(OpConstant, 1),
			makeInst(OpPop),
			makeInst(OpConstant, 2),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		),
			funcLocals(1)),
	))

	expectCompile(t, `var a; if a != nil { 10 } else { 20 }; 3333;`, bytecode(
		Array{Int(10), Int(20), Int(3333)},
		compFunc(concatInsts(
			makeInst(OpNil),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetLocal, 0),
			makeInst(OpJumpNil, 15),
			makeInst(OpConstant, 0),
			makeInst(OpPop),
			makeInst(OpJump, 19),
			makeInst(OpConstant, 1),
			makeInst(OpPop),
			makeInst(OpConstant, 2),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		),
			funcLocals(1)),
	))
}

func TestCompiler_Mixed(t *testing.T) {
	expectCompileMixed(t, "{% 1 -%} a", bytecode(
		Array{Int(1), RawStr("a")},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpPop),
			makeInst(OpGetBuiltin, int(BuiltinWrite)),
			makeInst(OpConstant, 1),
			makeInst(OpCall, 1, 0),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompileMixed(t, "a", bytecode(
		Array{RawStr("a")},
		compFunc(concatInsts(
			makeInst(OpGetBuiltin, int(BuiltinWrite)),
			makeInst(OpConstant, 0),
			makeInst(OpCall, 1, 0),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompileMixed(t, "{%- var myfn -%} a", bytecode(
		Array{RawStr("a")},
		compFunc(concatInsts(
			makeInst(OpNil),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetBuiltin, int(BuiltinWrite)),
			makeInst(OpConstant, 0),
			makeInst(OpCall, 1, 0),
			makeInst(OpReturn, 0),
		), funcLocals(1)),
	))

	expectCompileMixed(t, `a{%=1%}c`, bytecode(
		Array{RawStr("a"), Int(1), RawStr("c")},
		compFunc(concatInsts(
			makeInst(OpGetBuiltin, int(BuiltinWrite)),
			makeInst(OpConstant, 0),
			makeInst(OpConstant, 1),
			makeInst(OpConstant, 2),
			makeInst(OpCall, 3, 0),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompileMixed(t, `a{%=1%}c{%x := 5%}{%=x%}`, bytecode(
		Array{RawStr("a"), Int(1), RawStr("c"), Int(5)},
		compFunc(concatInsts(
			makeInst(OpGetBuiltin, int(BuiltinWrite)),
			makeInst(OpConstant, 0),
			makeInst(OpConstant, 1),
			makeInst(OpConstant, 2),
			makeInst(OpCall, 3, 0),
			makeInst(OpConstant, 3),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetBuiltin, int(BuiltinWrite)),
			makeInst(OpGetLocal, 0),
			makeInst(OpCall, 1, 0),
			makeInst(OpReturn, 0),
		), funcLocals(1)),
	))

	expectCompile(t, "# gad: mixed\n{% var myfn -%} a", bytecode(
		Array{RawStr("a")},
		compFunc(concatInsts(
			makeInst(OpNil),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetBuiltin, int(BuiltinWrite)),
			makeInst(OpConstant, 0),
			makeInst(OpCall, 1, 0),
			makeInst(OpReturn, 0),
		), funcLocals(1)),
	))

	expectCompile(t, "# gad: mixed\n{%- a := (%} a {%)%}", bytecode(
		Array{RawStr(" a ")},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpDefineLocal, 0),
			makeInst(OpReturn, 0),
		), funcLocals(1)),
	))

	expectCompile(t, "# gad: mixed\n{%- a := ( -%} a {%- )%}", bytecode(
		Array{RawStr("a")},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpDefineLocal, 0),
			makeInst(OpReturn, 0),
		), funcLocals(1)),
	))

	expectCompile(t, "# gad: mixed\n{%- a := ( -%} a {%- ); return a%}", bytecode(
		Array{RawStr("a")},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetLocal, 0),
			makeInst(OpReturn, 1),
		), funcLocals(1)),
	))

	expectCompile(t, "# gad: mixed\n{%- a := ( -%} a {%- ) %}{%return a%}", bytecode(
		Array{RawStr("a")},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetLocal, 0),
			makeInst(OpReturn, 1),
		), funcLocals(1)),
	))

	expectCompile(t, "# gad: mixed\n{%- a := ( -%} a {%- )%} b {%return a%}", bytecode(
		Array{RawStr("a"), RawStr(" b ")},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetBuiltin, int(BuiltinWrite)),
			makeInst(OpConstant, 1),
			makeInst(OpCall, 1, 0),
			makeInst(OpGetLocal, 0),
			makeInst(OpReturn, 1),
		), funcLocals(1)),
	))
}

func TestCompiler_CompileFuncWithNamedParams(t *testing.T) {
	expectCompile(t, `func f(;x int=1) {}`, bytecode(
		Array{
			Int(1),
			compFunc(concatInsts(
				makeInst(OpGetLocal, 0),
				makeInst(OpJumpNotNil, 10),
				makeInst(OpConstant, 0),
				makeInst(OpSetLocal, 0),
				makeInst(OpReturn, 0),
			),
				funcLocals(1),
				funcName("f"),
				funcNamedParams(
					&NamedParam{
						Name:  "x",
						Value: "1",
						TypesSymbols: []*SymbolInfo{
							{Name: "int"},
						},
					},
				)),
		},
		compFunc(concatInsts(
			makeInst(OpGetBuiltin, int(BuiltinFunc)),
			makeInst(OpConstant, 1),
			makeInst(OpCall, 1, 0),
			makeInst(OpDefineLocal, 0),
			makeInst(OpReturn, 0),
		), funcLocals(1)),
	))

	expectCompile(t, `return func(x;a=2) {return x+a}(1)`, bytecode(
		Array{
			Int(2),
			compFunc(concatInsts(
				makeInst(OpGetLocal, 1),
				makeInst(OpJumpNotNil, 10),
				makeInst(OpConstant, 0),
				makeInst(OpSetLocal, 1),
				makeInst(OpGetLocal, 0),
				makeInst(OpGetLocal, 1),
				makeInst(OpBinary, int(token.Add)),
				makeInst(OpReturn, 1),
			),
				funcLocals(2),
				funcParams("x"),
				funcNamedParams(
					&NamedParam{
						Name:  "a",
						Value: "2",
						TypesSymbols: []*SymbolInfo{
							{Name: "any"},
						},
					},
				), funcName("#1")),

			Int(1),
		},
		compFunc(concatInsts(
			makeInst(OpConstant, 1),
			makeInst(OpConstant, 2),
			makeInst(OpCall, 1, 0),
			makeInst(OpReturn, 1),
		)),
	))

}

func TestCompiler_AddMethods(t *testing.T) {
	expectCompile(t, `
var a;
met a {
	() {}
}
`,
		bytecode(
			Array{
				compFunc(concatInsts(
					makeInst(OpReturn, 0),
				), funcName("#1")),
			},
			compFunc(concatInsts(
				makeInst(OpNil),
				makeInst(OpDefineLocal, 0),
				makeInst(OpGetLocal, 0),
				makeInst(OpConstant, 0),
				makeInst(OpAddMethod, 0, 1),
				makeInst(OpPop),
				makeInst(OpReturn, 0),
			), funcLocals(1)),
		),
	)
	expectCompile(t, `
var a;
met a.x.y {
	() {}
}
`,
		bytecode(
			Array{
				Str("x"),
				Str("y"),
				compFunc(concatInsts(
					makeInst(OpReturn, 0),
				), funcName("#1")),
			},
			compFunc(concatInsts(
				makeInst(OpNil),
				makeInst(OpDefineLocal, 0),
				makeInst(OpGetLocal, 0),
				makeInst(OpConstant, 0),
				makeInst(OpGetIndex, 1),
				makeInst(OpConstant, 1),
				makeInst(OpConstant, 2),
				makeInst(OpAddMethod, 1, 1),
				makeInst(OpPop),
				makeInst(OpReturn, 0),
			), funcLocals(1)),
		),
	)
}

func TestCompiler_CompileFuncWithMethods(t *testing.T) {
	expectCompile(t, `
func addToX {
        () {}

        (i int) {}

        (v float) {}
}

return addToX
`,
		bytecode(
			Array{
				Str("addToX"),
				compFunc(concatInsts(
					makeInst(OpReturn, 0),
				), funcName("#1")),
				compFunc(concatInsts(
					makeInst(OpReturn, 0),
				), funcName("#2"), funcParams("i int"), funcLocals(1)),
				compFunc(concatInsts(
					makeInst(OpReturn, 0),
				), funcName("#3"), funcParams("v float"), funcLocals(1)),
			},
			compFunc(concatInsts(
				makeInst(OpGetBuiltin, int(BuiltinFunc)),
				makeInst(OpConstant, 0),
				makeInst(OpConstant, 1),
				makeInst(OpConstant, 2),
				makeInst(OpConstant, 3),
				makeInst(OpCall, 4, 0),
				makeInst(OpDefineLocal, 0),
				makeInst(OpGetLocal, 0),
				makeInst(OpReturn, 1),
			), funcLocals(1)),
		),
	)
}

func TestCompiler_CompileImportCompilableModule(t *testing.T) {
	moduleMap := NewModuleMap()
	moduleMap.AddSourceModule("mod", []byte(`export a = 1`))
	moduleMap.AddBuiltinCompilableModule("cmod", func(ctx *BuiltinCompileModuleContext) (bc *Bytecode, err error) {
		var (
			script  = `mod := import("mod", 1); export a = mod.a; export b = 2`
			srcFile = ctx.SetFileData([]byte(script))
			opts    = ctx.Compiler.Options()
			p       = parser.NewParserWithOptions(srcFile, &opts.ParserOptions, &opts.ScannerOptions)
			pf      *parser.File
		)

		if pf, err = p.ParseFile(); err != nil {
			return
		}

		if err = ctx.Compile(pf.Stmts); err != nil {
			return
		}

		bc = ctx.Compiler.Bytecode()

		return
	})
	expectCompileWithOpts(t, `c := import("cmod")`,
		CompileOptions{CompilerOptions: CompilerOptions{
			ModuleMap: moduleMap,
		}},
		bytecode(
			Array{
				Int(1),
				Str("a"),
				Int(2),
				Str("b"),
			},
			compFunc(concatInsts(
				makeInst(OpLoadModule, 1),
				makeInst(OpJumpFalsy, 9),
				makeInst(OpInitModule, 0, 0),
				makeInst(OpDefineLocal, 0),
				makeInst(OpReturn, 0),
			), funcLocals(1)),
			withModules(func(b *ModuleSpecBuilder) {
				b.Compiled("cmod", compFunc(concatInsts(
					makeInst(OpLoadModule, 2),
					makeInst(OpJumpFalsy, 12),
					makeInst(OpConstant, 0),
					makeInst(OpInitModule, 1, 0),
					makeInst(OpDefineLocal, 0),
					makeInst(OpGetLocal, 0),
					makeInst(OpConstant, 1),
					makeInst(OpGetIndex, 1),
					makeInst(OpModule),
					makeInst(OpConstant, 1),
					makeInst(OpSetIndex),
					makeInst(OpConstant, 2),
					makeInst(OpModule),
					makeInst(OpConstant, 3),
					makeInst(OpSetIndex),
					makeInst(OpReturn, 0),
				), funcLocals(1)))

				b.Compiled("mod", compFunc(concatInsts(
					makeInst(OpConstant, 0),
					makeInst(OpModule),
					makeInst(OpConstant, 1),
					makeInst(OpSetIndex),
					makeInst(OpReturn, 0),
				), funcName("#main")))
			}),
		),
	)
}

func TestCompiler_CompileImport(t *testing.T) {
	moduleMap := NewModuleMap()
	moduleMap.AddSourceModule("mod", []byte(``))
	expectCompileWithOpts(t, `import("mod", 1)`,
		CompileOptions{CompilerOptions: CompilerOptions{
			ModuleMap: moduleMap,
		}},
		bytecode(
			Array{
				Int(1),
			},
			compFunc(concatInsts(
				makeInst(OpLoadModule, 1),
				makeInst(OpJumpFalsy, 12),
				makeInst(OpConstant, 0),
				makeInst(OpInitModule, 1, 0),
				makeInst(OpPop),
				makeInst(OpReturn, 0),
			)),
			withModules(func(b *ModuleSpecBuilder) {
				b.Compiled("mod", compFunc(concatInsts(
					makeInst(OpReturn, 0),
				)))
			}),
		),
	)

	expectCompileWithOpts(t, `import("mod", 1; x=2)`,
		CompileOptions{CompilerOptions: CompilerOptions{
			ModuleMap: moduleMap,
		}},
		bytecode(
			Array{
				Int(1),
				Str("x"),
				Int(2),
			},
			compFunc(concatInsts(
				makeInst(OpLoadModule, 1),
				makeInst(OpJumpFalsy, 23),
				makeInst(OpConstant, 0),
				makeInst(OpConstant, 1),
				makeInst(OpConstant, 2),
				makeInst(OpKeyValue, 1),
				makeInst(OpKeyValueArray, 1),
				makeInst(OpInitModule, 1, 2),
				makeInst(OpPop),
				makeInst(OpReturn, 0),
			)),
			withModules(func(b *ModuleSpecBuilder) {
				b.Compiled("mod", compFunc(concatInsts(
					makeInst(OpReturn, 0),
				)))
			}),
		),
	)

	expectCompileWithOpts(t, `import("mod")`,
		CompileOptions{CompilerOptions: CompilerOptions{
			ModuleMap: moduleMap,
		}},
		bytecode(
			Array{},
			compFunc(concatInsts(
				makeInst(OpLoadModule, 1),    // 0000 constant, module indexes
				makeInst(OpJumpFalsy, 9),     // 0005 if loaded no call is required
				makeInst(OpInitModule, 0, 0), // 0008 obtain return value from module
				makeInst(OpPop),              // 0014
				makeInst(OpReturn, 0),        // 0015
			)),
			withModules(func(b *ModuleSpecBuilder) {
				b.Compiled("mod", compFunc(concatInsts(
					makeInst(OpReturn, 0),
				)))
			}),
		),
	)
}

func TestCompiler_Export(t *testing.T) {
	// all local variables are initialized as nil
	expectCompile(t, `
var @exports
const a = 1
export a
export b = 2
export c(){return 3}
export func d(){return 3}
export e() => 4
export {f:5, g:6}
export [2**3] = 7
`, bytecode(
		Array{
			Int(1),
			Str("a"),
			Int(2),
			Str("b"),
			Int(3),
			compFunc(concatInsts(
				makeInst(OpConstant, 4),
				makeInst(OpReturn, 1),
			), funcName("c")),
			Str("c"),
			compFunc(concatInsts(
				makeInst(OpConstant, 4),
				makeInst(OpReturn, 1),
			), funcName("d")),
			Str("d"),
			Int(4),
			compFunc(concatInsts(
				makeInst(OpConstant, 9),
				makeInst(OpReturn, 1),
			), funcName("e")),
			Str("e"),
			Str("f"),
			Int(5),
			Str("g"),
			Int(6),
			Int(7),
		},
		compFunc(concatInsts(
			makeInst(OpNil),
			makeInst(OpDefineLocal, 0),
			makeInst(OpConstant, 0),
			makeInst(OpDefineLocal, 1),
			makeInst(OpGetLocal, 1),
			makeInst(OpModule),
			makeInst(OpConstant, 1),
			makeInst(OpSetIndex),
			makeInst(OpConstant, 2),
			makeInst(OpModule),
			makeInst(OpConstant, 3),
			makeInst(OpSetIndex),
			makeInst(OpConstant, 5),
			makeInst(OpModule),
			makeInst(OpConstant, 6),
			makeInst(OpSetIndex),
			makeInst(OpConstant, 7),
			makeInst(OpModule),
			makeInst(OpConstant, 8),
			makeInst(OpSetIndex),
			makeInst(OpConstant, 10),
			makeInst(OpModule),
			makeInst(OpConstant, 11),
			makeInst(OpSetIndex),
			makeInst(OpConstant, 12),
			makeInst(OpConstant, 13),
			makeInst(OpConstant, 14),
			makeInst(OpConstant, 15),
			makeInst(OpDict, 4),
			makeInst(OpExtendModule),
			makeInst(OpPop),
			makeInst(OpConstant, 16),
			makeInst(OpModule),
			makeInst(OpConstant, 2),
			makeInst(OpConstant, 4),
			makeInst(OpBinary, int(token.Pow)),
			makeInst(OpSetIndex),
			makeInst(OpReturn, 0),
		),
			funcLocals(2),
		),
	))
}

func TestCompiler_CompileToRaw(t *testing.T) {
	expectCompile(t, `raw "abc"`, bytecode(
		Array{RawStr("abc")},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))
	expectCompile(t, "raw `abc`", bytecode(
		Array{RawStr("abc")},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))
	expectCompile(t, "raw 1", bytecode(
		Array{Int(1)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpToRawStr),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))
}

func TestCompiler_CompileEmbed(t *testing.T) {
	// 1 instruction are generated for every source embed import.
	// If embed's returned value is already stored, ignore storing.
	embedMap := NewEmbedMap()
	embedMap.AddFile("file.js", []byte(`abcd`))
	expectCompileWithOpts(t, `embed("file.js")`,
		CompileOptions{CompilerOptions: CompilerOptions{
			EmbededdMap: embedMap,
		}},
		bytecode(
			Array{
				&Embedded{Name: "file.js", ReaderFactory: EmbeddedBytesReaderFactory(`abcd`)},
			},
			compFunc(concatInsts(
				makeInst(OpConstant, 0),
				makeInst(OpPop),
				makeInst(OpReturn, 0),
			)),
		),
	)

	expectCompileWithOpts(t, `embed("file.js"); embed("file.js")`,
		CompileOptions{CompilerOptions: CompilerOptions{
			EmbededdMap: embedMap,
		}},
		bytecode(
			Array{
				&Embedded{Name: "file.js", ReaderFactory: EmbeddedBytesReaderFactory(`abcd`)},
			},
			compFunc(concatInsts(
				makeInst(OpConstant, 0),
				makeInst(OpPop),
				makeInst(OpConstant, 0),
				makeInst(OpPop),
				makeInst(OpReturn, 0),
			)),
		),
	)

	// embed with different names creates separate constants
	multiMap := NewEmbedMap()
	multiMap.AddFile("a.js", []byte(`aaa`))
	multiMap.AddFile("b.js", []byte(`bbb`))
	expectCompileWithOpts(t, `embed("a.js"); embed("b.js")`,
		CompileOptions{CompilerOptions: CompilerOptions{
			EmbededdMap: multiMap,
		}},
		bytecode(
			Array{
				&Embedded{Name: "a.js", ReaderFactory: EmbeddedBytesReaderFactory(`aaa`)},
				&Embedded{Name: "b.js", ReaderFactory: EmbeddedBytesReaderFactory(`bbb`)},
			},
			compFunc(concatInsts(
				makeInst(OpConstant, 0),
				makeInst(OpPop),
				makeInst(OpConstant, 1),
				makeInst(OpPop),
				makeInst(OpReturn, 0),
			)),
		),
	)

	// embed error: empty path
	expectCompileError(t, `embed("")`, "Compile Error: empty path")

	// embed error: path not found in embed map
	expectCompileErrorWithOpts(t, `embed("nonexistent.js")`,
		CompileOptions{CompilerOptions: CompilerOptions{
			EmbededdMap: embedMap,
		}},
		"Compile Error: path 'nonexistent.js' not found")

	// embed with EmbeddedFile struct (with modTime)
	timeMap := NewEmbedMap()
	timeMap.Add("timedata", EmbeddedFile(Embedded{
		Name:          "timedata",
		ModTime:       time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC),
		ReaderFactory: EmbeddedBytesReaderFactory(`time data`),
	}))
	expectCompileWithOpts(t, `embed("timedata")`,
		CompileOptions{CompilerOptions: CompilerOptions{
			EmbededdMap: timeMap,
		}},
		bytecode(
			Array{
				&Embedded{
					Name:          "timedata",
					ModTime:       time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC),
					ReaderFactory: EmbeddedBytesReaderFactory(`time data`),
				},
			},
			compFunc(concatInsts(
				makeInst(OpConstant, 0),
				makeInst(OpPop),
				makeInst(OpReturn, 0),
			)),
		),
	)

	// embed in expression (assign to variable)
	expectCompileWithOpts(t, `x := embed("file.js"); return x`,
		CompileOptions{CompilerOptions: CompilerOptions{
			EmbededdMap: embedMap,
		}},
		bytecode(
			Array{
				&Embedded{Name: "file.js", ReaderFactory: EmbeddedBytesReaderFactory(`abcd`)},
			},
			compFunc(concatInsts(
				makeInst(OpConstant, 0),
				makeInst(OpDefineLocal, 0),
				makeInst(OpGetLocal, 0),
				makeInst(OpReturn, 1),
			),
				funcLocals(1),
			),
		),
	)

	// embed with EmbeddedFileImporter using temp file
	t.Run("file importer", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.txt")
		require.NoError(t, os.WriteFile(tmpFile, []byte(`hello`), 0644))

		impMap := NewEmbedMap()
		impMap.SetExtImporter(&importers.EmbeddedFileImporter{
			WorkDirs: []string{tmpDir},
		})
		_, bc, err := Compile(NewSymbolTable(NewBuiltins().NameSet), []byte(`embed("test.txt")`),
			CompileOptions{CompilerOptions: CompilerOptions{EmbededdMap: impMap}})
		require.NoError(t, err)
		require.Len(t, bc.Constants, 1)
		emb, ok := bc.Constants[0].(*Embedded)
		require.True(t, ok, "constant must be *Embedded")
		require.Equal(t, "test.txt", emb.Name)
		data, err := emb.Read()
		require.NoError(t, err)
		require.Equal(t, "hello", string(data))
	})

	// embed with sources param via file importer
	t.Run("sources param", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "mydir"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "mydir", "f.txt"), []byte(`content`), 0644))

		impMap := NewEmbedMap()
		impMap.SetExtImporter(&importers.EmbeddedFileImporter{
			WorkDirs: []string{tmpDir},
		})
		_, bc, err := Compile(NewSymbolTable(NewBuiltins().NameSet),
			[]byte(`embed("f.txt"; sources=["mydir"])`),
			CompileOptions{CompilerOptions: CompilerOptions{EmbededdMap: impMap}})
		require.NoError(t, err)
		require.Len(t, bc.Constants, 1)
		emb, ok := bc.Constants[0].(*Embedded)
		require.True(t, ok)
		require.Equal(t, "f.txt", emb.Name)
	})

	// embed with config_file YAML
	t.Run("config_file", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "dat"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "dat", "a.go"), []byte(`pkg a`), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "dat", "b.txt"), []byte(`content`), 0644))

		cfg := map[string]interface{}{"includes": []string{"*.go"}}
		cfgData, err := yaml.Marshal(cfg)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "embed.yaml"), cfgData, 0644))

		impMap := NewEmbedMap()
		impMap.SetExtImporter(&importers.EmbeddedFileImporter{
			WorkDirs: []string{tmpDir},
		})
		_, bc, err := Compile(NewSymbolTable(NewBuiltins().NameSet),
			[]byte(`embed("dat"; config_file="embed.yaml")`),
			CompileOptions{CompilerOptions: CompilerOptions{EmbededdMap: impMap}})
		require.NoError(t, err)
		require.Len(t, bc.Constants, 1)
		emb, ok := bc.Constants[0].(*Embedded)
		require.True(t, ok)
		require.Equal(t, "dat", emb.Name)
		// config had includes=["*.go"], so only .go file should be present
		require.NotNil(t, emb.GetNode("a.go"))
		require.Nil(t, emb.GetNode("b.txt"))
	})
}

func TestCompiler_CompileInterpolatedStringLit(t *testing.T) {
	t.Run("plain string", func(t *testing.T) {
		st := NewSymbolTable(NewBuiltins().NameSet)
		_, bc, err := Compile(st, []byte(`return #"hello"`), CompileOptions{})
		require.NoError(t, err)
		require.NotNil(t, bc)

		// Run it
		vm := NewVM(NewBuiltins().Build(), bc)
		ret, err := vm.Run()
		require.NoError(t, err)
		require.Equal(t, Str("hello"), ret)
	})

	t.Run("with interpolation", func(t *testing.T) {
		st := NewSymbolTable(NewBuiltins().NameSet)
		_, bc, err := Compile(st, []byte(`name := "world"; return #"hello {name}"`), CompileOptions{})
		require.NoError(t, err)
		require.NotNil(t, bc)

		vm := NewVM(NewBuiltins().Build(), bc)
		ret, err := vm.Run()
		require.NoError(t, err)
		require.Equal(t, Str("hello world"), ret)
	})

	t.Run("multiple interpolations", func(t *testing.T) {
		st := NewSymbolTable(NewBuiltins().NameSet)
		_, bc, err := Compile(st, []byte(`a := 1; b := 2; return #"{a} + {b} = {a+b}"`), CompileOptions{})
		require.NoError(t, err)
		require.NotNil(t, bc)

		vm := NewVM(NewBuiltins().Build(), bc)
		ret, err := vm.Run()
		require.NoError(t, err)
		require.Equal(t, Str("1 + 2 = 3"), ret)
	})

	runTmpl := func(t *testing.T, src string) Object {
		t.Helper()
		st := NewSymbolTable(NewBuiltins().NameSet)
		_, bc, err := Compile(st, []byte(src), CompileOptions{})
		require.NoError(t, err)
		require.NotNil(t, bc)
		vm := NewVM(NewBuiltins().Build(), bc)
		ret, err := vm.Run()
		require.NoError(t, err)
		return ret
	}

	t.Run("heredoc plain", func(t *testing.T) {
		require.Equal(t, Str("hello"), runTmpl(t, `return #"""hello"""`))
	})

	t.Run("heredoc with interpolation", func(t *testing.T) {
		require.Equal(t, Str("hello world"),
			runTmpl(t, `name := "world"; return #"""hello {name}"""`))
	})

	t.Run("heredoc interprets escapes", func(t *testing.T) {
		// unlike the raw ``` heredoc, \t and \n are interpreted
		require.Equal(t, Str("a\tb\nc"), runTmpl(t, `return #"""a\tb\nc"""`))
	})

	t.Run("heredoc escapes with interpolation", func(t *testing.T) {
		require.Equal(t, Str("1\t2"),
			runTmpl(t, `a := 1; b := 2; return #"""{a}\t{b}"""`))
	})

	t.Run("heredoc multiline strips indentation", func(t *testing.T) {
		src := "name := \"bob\"\nreturn #\"\"\"\n\t\tHello {name}\n\t\tBye\n\t\"\"\""
		require.Equal(t, Str("Hello bob\nBye"), runTmpl(t, src))
	})

	t.Run("heredoc multiline strips indentation then escapes", func(t *testing.T) {
		src := "return #\"\"\"\n\t\ta\\tb\n\t\tc\n\t\"\"\""
		require.Equal(t, Str("a\tb\nc"), runTmpl(t, src))
	})
}

func TestCompiler_Compile(t *testing.T) {
	// all local variables are initialized as nil
	expectCompile(t, `var a`, bytecode(
		Array{},
		compFunc(concatInsts(
			makeInst(OpNil),
			makeInst(OpDefineLocal, 0),
			makeInst(OpReturn, 0),
		),
			funcLocals(1),
		),
	))
	expectCompile(t, `var (a, b, c)`, bytecode(
		Array{},
		compFunc(concatInsts(
			makeInst(OpNil),
			makeInst(OpDefineLocal, 0),
			makeInst(OpNil),
			makeInst(OpDefineLocal, 1),
			makeInst(OpNil),
			makeInst(OpDefineLocal, 2),
			makeInst(OpReturn, 0),
		),
			funcLocals(3),
		),
	))
	expectCompile(t, `var a = nil`, bytecode(
		Array{},
		compFunc(concatInsts(
			makeInst(OpNil),
			makeInst(OpDefineLocal, 0),
			makeInst(OpReturn, 0),
		),
			funcLocals(1),
		),
	))
	expectCompile(t, `a := nil`, bytecode(
		Array{},
		compFunc(concatInsts(
			makeInst(OpNil),
			makeInst(OpDefineLocal, 0),
			makeInst(OpReturn, 0),
		),
			funcLocals(1),
		),
	))

	expectCompile(t, `param (;a=1, **na)`, bytecode(
		Array{Int(1)},
		compFunc(concatInsts(
			makeInst(OpGetLocal, 0),
			makeInst(OpJumpNotNil, 10),
			makeInst(OpConstant, 0),
			makeInst(OpSetLocal, 0),
			makeInst(OpReturn, 0),
		),
			funcLocals(2),
			funcNamedParams(NewNamedParam("a", "1"), NewVarNamedParam("na")),
		),
	))

	// multiple declaration requires parentheses
	expectCompileError(t, `param a, b`, `Parse Error: expected statement, found ','`)
	expectCompileError(t, `global a, b`, `Parse Error: expected ';', found ','`)
	expectCompileError(t, `var a, b`, `Parse Error: expected ';', found ','`)
	// param declaration can only be at the top scope
	expectCompileError(t, `func f() { param a }`, `Compile Error: param not allowed in this scope`)

	// force to set nil
	expectCompile(t, `a := (nil)`, bytecode(
		Array{},
		compFunc(concatInsts(
			makeInst(OpNil),
			makeInst(OpDefineLocal, 0),
			makeInst(OpReturn, 0),
		),
			funcLocals(1),
		),
	))
	expectCompile(t, `var (a, b=1, c=2)`, bytecode(
		Array{Int(1), Int(2)},
		compFunc(concatInsts(
			makeInst(OpNil),
			makeInst(OpDefineLocal, 0),
			makeInst(OpConstant, 0),
			makeInst(OpDefineLocal, 1),
			makeInst(OpConstant, 1),
			makeInst(OpDefineLocal, 2),
			makeInst(OpReturn, 0),
		),
			funcLocals(3),
		),
	))
	// parameters are initialized as nil
	expectCompile(t, `param a`, bytecode(
		Array{},
		compFunc(concatInsts(
			makeInst(OpReturn, 0),
		),
			funcParams("a"),
			funcLocals(1),
		),
	))
	expectCompile(t, `param (a, b, *c)`, bytecode(
		Array{},
		compFunc(concatInsts(
			makeInst(OpReturn, 0),
		),
			funcParams("a", "b", "*c"),
			funcLocals(3),
		),
	))
	expectCompile(t, `global a`, bytecode(
		Array{Str("a")},
		compFunc(concatInsts(
			makeInst(OpReturn, 0),
		)),
	))
	expectCompile(t, `global (a, b); var c`, bytecode(
		Array{Str("a"), Str("b")},
		compFunc(concatInsts(
			makeInst(OpNil),
			makeInst(OpDefineLocal, 0),
			makeInst(OpReturn, 0),
		),
			funcLocals(1),
		),
	))
	expectCompile(t, `param (arg1, *varg); global (a, b); var c = arg1; c = b`, bytecode(
		Array{Str("a"), Str("b")},
		compFunc(concatInsts(
			makeInst(OpGetLocal, 0),
			makeInst(OpDefineLocal, 2),
			makeInst(OpGetGlobal, 1),
			makeInst(OpSetLocal, 2),
			makeInst(OpReturn, 0),
		),
			funcParams("arg1", "*varg"),
			funcLocals(3),
		),
	))

	expectCompile(t, `1 + 2`, bytecode(
		Array{Int(1), Int(2)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpConstant, 1),
			makeInst(OpBinary, int(token.Add)),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	// `===` compiles to OpBinary(Same); it is not constant-folded.
	expectCompile(t, `1 === 2`, bytecode(
		Array{Int(1), Int(2)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpConstant, 1),
			makeInst(OpBinary, int(token.Same)),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	// `!==` desugars to `!(a === b)`: OpBinary(Same) then OpUnary(Not).
	expectCompile(t, `1 !== 2`, bytecode(
		Array{Int(1), Int(2)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpConstant, 1),
			makeInst(OpBinary, int(token.Same)),
			makeInst(OpUnary, int(token.Not)),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	// `ain` compiles to OpBinary(Ain) and is not constant-folded (it is dispatched
	// at runtime through gad.binOp, like `in` / `===`).
	expectCompile(t, `1 ain 2`, bytecode(
		Array{Int(1), Int(2)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpConstant, 1),
			makeInst(OpBinary, int(token.Ain)),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `1; 2`, bytecode(
		Array{Int(1), Int(2)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpPop),
			makeInst(OpConstant, 1),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `1 - 2`, bytecode(
		Array{Int(1), Int(2)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpConstant, 1),
			makeInst(OpBinary, int(token.Sub)),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `1 * 2`, bytecode(
		Array{Int(1), Int(2)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpConstant, 1),
			makeInst(OpBinary, int(token.Mul)),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `1 ** 2`, bytecode(
		Array{Int(1), Int(2)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpConstant, 1),
			makeInst(OpBinary, int(token.Pow)),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `2 / 1`, bytecode(
		Array{Int(2), Int(1)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpConstant, 1),
			makeInst(OpBinary, int(token.Quo)),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `true`, bytecode(
		Array{},
		compFunc(concatInsts(
			makeInst(OpTrue),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `false`, bytecode(
		Array{},
		compFunc(concatInsts(
			makeInst(OpFalse),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `yes`, bytecode(
		Array{},
		compFunc(concatInsts(
			makeInst(OpYes),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `no`, bytecode(
		Array{},
		compFunc(concatInsts(
			makeInst(OpNo),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `1 > 2`, bytecode(
		Array{Int(1), Int(2)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpConstant, 1),
			makeInst(OpBinary, int(token.Greater)),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `1 < 2`, bytecode(
		Array{Int(1), Int(2)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpConstant, 1),
			makeInst(OpBinary, int(token.Less)),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `1 >= 2`, bytecode(
		Array{Int(1), Int(2)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpConstant, 1),
			makeInst(OpBinary, int(token.GreaterEq)),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `1 <= 2`, bytecode(
		Array{Int(1), Int(2)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpConstant, 1),
			makeInst(OpBinary, int(token.LessEq)),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `1 == 2`, bytecode(
		Array{Int(1), Int(2)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpConstant, 1),
			makeInst(OpEqual),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `1 != 2`, bytecode(
		Array{Int(1), Int(2)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpConstant, 1),
			makeInst(OpNotEqual),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `true == false`, bytecode(
		Array{},
		compFunc(concatInsts(
			makeInst(OpTrue),
			makeInst(OpFalse),
			makeInst(OpEqual),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `true != false`, bytecode(
		Array{},
		compFunc(concatInsts(
			makeInst(OpTrue),
			makeInst(OpFalse),
			makeInst(OpNotEqual),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `yes == no`, bytecode(
		Array{},
		compFunc(concatInsts(
			makeInst(OpYes),
			makeInst(OpNo),
			makeInst(OpEqual),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `yes != no`, bytecode(
		Array{},
		compFunc(concatInsts(
			makeInst(OpYes),
			makeInst(OpNo),
			makeInst(OpNotEqual),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `return yes != no`, bytecode(
		Array{},
		compFunc(concatInsts(
			makeInst(OpYes),
			makeInst(OpNo),
			makeInst(OpNotEqual),
			makeInst(OpReturn, 1),
		)),
	))

	expectCompile(t, `-1`, bytecode(
		Array{Int(1)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpUnary, int(token.Sub)),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `!true`, bytecode(
		Array{},
		compFunc(concatInsts(
			makeInst(OpTrue),
			makeInst(OpUnary, int(token.Not)),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `!yes`, bytecode(
		Array{},
		compFunc(concatInsts(
			makeInst(OpYes),
			makeInst(OpUnary, int(token.Not)),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	// `if true` => skips else
	expectCompile(t, `if true { 10 }; 3333`, bytecode(
		Array{Int(10), Int(3333)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpPop),
			makeInst(OpConstant, 1),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	// `if (true)` => normal if
	expectCompile(t, `if (true) { 10 }; 3333`, bytecode(
		Array{Int(10), Int(3333)},
		compFunc(concatInsts(
			makeInst(OpTrue),         // 0000
			makeInst(OpJumpFalsy, 8), // 0001
			makeInst(OpConstant, 0),  // 0004
			makeInst(OpPop),          // 0007
			makeInst(OpConstant, 1),  // 0008
			makeInst(OpPop),          // 0011
			makeInst(OpReturn, 0),    // 0012
		)),
	))

	// `if true` => skips else
	expectCompile(t, `if true { 10 } else { 20 }; 3333;`, bytecode(
		Array{Int(10), Int(3333)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpPop),
			makeInst(OpConstant, 1),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	// `if true` => skips else
	expectCompile(t, `if true { 10 } else {}; 3333;`, bytecode(
		Array{Int(10), Int(3333)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpPop),
			makeInst(OpConstant, 1),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	// `if true` => no jumps
	expectCompile(t, `if true { 10 }; 3333;`, bytecode(
		Array{Int(10), Int(3333)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpPop),
			makeInst(OpConstant, 1),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	// `if false` => skip if block but OpJump is put
	// TODO: improve this, unnecessary jump
	expectCompile(t, `if false { 10 }; 3333;`, bytecode(
		Array{Int(3333)},
		compFunc(concatInsts(
			makeInst(OpJump, 3),     // 0000
			makeInst(OpConstant, 0), // 0003
			makeInst(OpPop),         // 0006
			makeInst(OpReturn, 0),   // 0007
		)),
	))

	// `if false` => goes to else block
	// TODO: improve this, unnecessary jump
	expectCompile(t, `if false { 10 } else { 20 }; 3333;`, bytecode(
		Array{Int(20), Int(3333)},
		compFunc(concatInsts(
			makeInst(OpJump, 6),     // 0000
			makeInst(OpJump, 10),    // 0003
			makeInst(OpConstant, 0), // 0006
			makeInst(OpPop),         // 0009
			makeInst(OpConstant, 1), // 0010
			makeInst(OpPop),         // 0013
			makeInst(OpReturn, 0),   // 0014
		)),
	))

	// `if (true)` => normal if
	expectCompile(t, `if (true) { 10 } else { 20 }; 3333;`, bytecode(
		Array{Int(10), Int(20), Int(3333)},
		compFunc(concatInsts(
			makeInst(OpTrue),          // 0000
			makeInst(OpJumpFalsy, 11), // 0001
			makeInst(OpConstant, 0),   // 0004
			makeInst(OpPop),           // 0007
			makeInst(OpJump, 15),      // 0008
			makeInst(OpConstant, 1),   // 0011
			makeInst(OpPop),           // 0014
			makeInst(OpConstant, 2),   // 0015
			makeInst(OpPop),           // 0018
			makeInst(OpReturn, 0),     // 0019
		)),
	))

	expectCompile(t, `"string"`, bytecode(
		Array{Str("string")},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `"str" + "ing"`, bytecode(
		Array{Str("str"), Str("ing")},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpConstant, 1),
			makeInst(OpBinary, int(token.Add)),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, "`raw string`", bytecode(
		Array{RawStr("raw string")},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, "```raw heredoc string```", bytecode(
		Array{RawStr("raw heredoc string")},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, "```\nraw heredoc string\n```", bytecode(
		Array{RawStr("raw heredoc string")},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, "```\nraw heredoc string\n           ```", bytecode(
		Array{RawStr("raw heredoc string")},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, "```\n  raw heredoc\n  string\nx\n```", bytecode(
		Array{RawStr("raw heredoc\nstring\nx")},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, "```\n  raw heredoc\n\n  string\nx\n```", bytecode(
		Array{RawStr("raw heredoc\n\nstring\nx")},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, "```\n\t\traw  \n\t\theredoc\n\t\t string\n\tx\n```", bytecode(
		Array{RawStr("raw  \nheredoc\n string\nx")},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `"""heredoc string"""`, bytecode(
		Array{Str("heredoc string")},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, "\"\"\"\nheredoc string\n\"\"\"", bytecode(
		Array{Str("heredoc string")},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	// indentation stripping and escape processing
	expectCompile(t, "\"\"\"\n\t\ta\\tb\n\t\tc\n\t\"\"\"", bytecode(
		Array{Str("a\tb\nc")},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `a := 1; b := 2; a += b`, bytecode(
		Array{Int(1), Int(2)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpDefineLocal, 0),
			makeInst(OpConstant, 1),
			makeInst(OpDefineLocal, 1),
			makeInst(OpGetLocal, 0),
			makeInst(OpGetLocal, 1),
			makeInst(OpSelfAssign, int(token.Add)),
			makeInst(OpSetLocal, 0),
			makeInst(OpReturn, 0),
		),
			funcLocals(2),
		)))

	expectCompile(t, `var (a = 1, b = 2); a += b`, bytecode(
		Array{Int(1), Int(2)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpDefineLocal, 0),
			makeInst(OpConstant, 1),
			makeInst(OpDefineLocal, 1),
			makeInst(OpGetLocal, 0),
			makeInst(OpGetLocal, 1),
			makeInst(OpSelfAssign, int(token.Add)),
			makeInst(OpSetLocal, 0),
			makeInst(OpReturn, 0),
		),
			funcLocals(2),
		)))

	expectCompile(t, `var (a, b = 1); a = b + 1`, bytecode(
		Array{Int(1)},
		compFunc(concatInsts(
			makeInst(OpNil),
			makeInst(OpDefineLocal, 0),
			makeInst(OpConstant, 0),
			makeInst(OpDefineLocal, 1),
			makeInst(OpGetLocal, 1),
			makeInst(OpConstant, 0),
			makeInst(OpBinary, int(token.Add)),
			makeInst(OpSetLocal, 0),
			makeInst(OpReturn, 0),
		),
			funcLocals(2),
		)))

	expectCompile(t, `var (a, b)`, bytecode(
		Array{},
		compFunc(concatInsts(
			makeInst(OpNil),
			makeInst(OpDefineLocal, 0),
			makeInst(OpNil),
			makeInst(OpDefineLocal, 1),
			makeInst(OpReturn, 0),
		),
			funcLocals(2),
		)))

	expectCompile(t, `a := 1; b := 2; a /= b`, bytecode(
		Array{Int(1), Int(2)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpDefineLocal, 0),
			makeInst(OpConstant, 1),
			makeInst(OpDefineLocal, 1),
			makeInst(OpGetLocal, 0),
			makeInst(OpGetLocal, 1),
			makeInst(OpSelfAssign, int(token.Quo)),
			makeInst(OpSetLocal, 0),
			makeInst(OpReturn, 0),
		),
			funcLocals(2),
		)))

	expectCompile(t, `[]`, bytecode(
		Array{},
		compFunc(concatInsts(
			makeInst(OpArray, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `[1, 2, 3]`, bytecode(
		Array{Int(1), Int(2), Int(3)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpConstant, 1),
			makeInst(OpConstant, 2),
			makeInst(OpArray, 3),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `[1 + 2, 3 - 4, 5 * 6, 7 ** 8]`, bytecode(
		Array{Int(1), Int(2), Int(3), Int(4), Int(5), Int(6), Int(7), Int(8)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpConstant, 1),
			makeInst(OpBinary, int(token.Add)),
			makeInst(OpConstant, 2),
			makeInst(OpConstant, 3),
			makeInst(OpBinary, int(token.Sub)),
			makeInst(OpConstant, 4),
			makeInst(OpConstant, 5),
			makeInst(OpBinary, int(token.Mul)),
			makeInst(OpConstant, 6),
			makeInst(OpConstant, 7),
			makeInst(OpBinary, int(token.Pow)),
			makeInst(OpArray, 4),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `({})`, bytecode(
		Array{},
		compFunc(concatInsts(
			makeInst(OpDict, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `({a: 2, b: 4, c: 6})`, bytecode(
		Array{Str("a"), Int(2), Str("b"), Int(4), Str("c"), Int(6)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpConstant, 1),
			makeInst(OpConstant, 2),
			makeInst(OpConstant, 3),
			makeInst(OpConstant, 4),
			makeInst(OpConstant, 5),
			makeInst(OpDict, 6),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `({a: 2 + 3, b: 5 * 6})`, bytecode(
		Array{Str("a"), Int(2), Int(3), Str("b"), Int(5), Int(6)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpConstant, 1),
			makeInst(OpConstant, 2),
			makeInst(OpBinary, int(token.Add)),
			makeInst(OpConstant, 3),
			makeInst(OpConstant, 4),
			makeInst(OpConstant, 5),
			makeInst(OpBinary, int(token.Mul)),
			makeInst(OpDict, 4),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `[1, 2, 3][1 + 1]`, bytecode(
		Array{Int(1), Int(2), Int(3)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpConstant, 1),
			makeInst(OpConstant, 2),
			makeInst(OpArray, 3),
			makeInst(OpConstant, 0),
			makeInst(OpConstant, 0),
			makeInst(OpBinary, int(token.Add)),
			makeInst(OpGetIndex, 1),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `({a: 2})[2 - 1]`, bytecode(
		Array{Str("a"), Int(2), Int(1)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpConstant, 1),
			makeInst(OpDict, 2),
			makeInst(OpConstant, 1),
			makeInst(OpConstant, 2),
			makeInst(OpBinary, int(token.Sub)),
			makeInst(OpGetIndex, 1),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `[1, 2, 3][:]`, bytecode(
		Array{Int(1), Int(2), Int(3)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpConstant, 1),
			makeInst(OpConstant, 2),
			makeInst(OpArray, 3),
			makeInst(OpNil),
			makeInst(OpNil),
			makeInst(OpSliceIndex),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `[1, 2, 3][0 : 2]`, bytecode(
		Array{Int(1), Int(2), Int(3), Int(0)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpConstant, 1),
			makeInst(OpConstant, 2),
			makeInst(OpArray, 3),
			makeInst(OpConstant, 3),
			makeInst(OpConstant, 1),
			makeInst(OpSliceIndex),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `[1, 2, 3][ : 2]`, bytecode(
		Array{Int(1), Int(2), Int(3)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpConstant, 1),
			makeInst(OpConstant, 2),
			makeInst(OpArray, 3),
			makeInst(OpNil),
			makeInst(OpConstant, 1),
			makeInst(OpSliceIndex),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `[1, 2, 3][0 : ]`, bytecode(
		Array{Int(1), Int(2), Int(3), Int(0)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpConstant, 1),
			makeInst(OpConstant, 2),
			makeInst(OpArray, 3),
			makeInst(OpConstant, 3),
			makeInst(OpNil),
			makeInst(OpSliceIndex),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `f1 := func(a) { return a }; f1(*[1, 2]);`, bytecode(
		Array{
			compFunc(concatInsts(
				makeInst(OpGetLocal, 0),
				makeInst(OpReturn, 1),
			),
				funcParams("a"),
				funcLocals(1),
			),
			Int(1),
			Int(2),
		},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetLocal, 0),
			makeInst(OpConstant, 1),
			makeInst(OpConstant, 2),
			makeInst(OpArray, 2),
			makeInst(OpCall, 1, 1),
			makeInst(OpPop),
			makeInst(OpReturn, 0)),
			funcLocals(1),
		),
	))

	expectCompile(t, `func f1 (a) { func f2(b) { return a + b } }`, bytecode(
		Array{
			compFunc(concatInsts(
				makeInst(OpGetFree, 0),
				makeInst(OpGetLocal, 0),
				makeInst(OpBinary, int(token.Add)),
				makeInst(OpReturn, 1),
			),
				funcParams("b"),
				funcLocals(1),
			),

			compFunc(concatInsts(
				makeInst(OpGetBuiltin, int(BuiltinFunc)),
				makeInst(OpGetLocalPtr, 0),
				makeInst(OpClosure, 0, 1),
				makeInst(OpCall, 1, 0),
				makeInst(OpDefineLocal, 1),
				makeInst(OpReturn, 0),
			),
				funcName("f1"),
				funcParams("a"),
				funcLocals(2),
			),
		},
		compFunc(concatInsts(
			makeInst(OpGetBuiltin, int(BuiltinFunc)),
			makeInst(OpConstant, 1),
			makeInst(OpCall, 1, 0),
			makeInst(OpDefineLocal, 0),
			makeInst(OpReturn, 0),
		), funcLocals(1)),
	))

	expectCompileError(t, `func (a) {  }`, `func stmt require ident`)
	expectCompileError(t, `func f(a) { func(b) { return a + b } }`, `func stmt require ident`)

	for _, s := range []string{
		`f1 := func(a) { return a }; f1(*[1, 2]);`,
		`f1 := (a) => a; f1(*[1, 2]);`} {
		expectCompile(t, s, bytecode(
			Array{
				compFunc(concatInsts(
					makeInst(OpGetLocal, 0),
					makeInst(OpReturn, 1),
				),
					funcParams("a"),
					funcLocals(1),
				),
				Int(1),
				Int(2),
			},
			compFunc(concatInsts(
				makeInst(OpConstant, 0),
				makeInst(OpDefineLocal, 0),
				makeInst(OpGetLocal, 0),
				makeInst(OpConstant, 1),
				makeInst(OpConstant, 2),
				makeInst(OpArray, 2),
				makeInst(OpCall, 1, 1),
				makeInst(OpPop),
				makeInst(OpReturn, 0)),
				funcLocals(1),
			),
		))
	}

	for _, s := range []string{`func f() { return 5 + 10 }`, `f() => 5 + 10`} {
		expectCompile(t, s, bytecode(
			Array{
				Int(5),
				Int(10),
				compFunc(concatInsts(
					makeInst(OpConstant, 0),
					makeInst(OpConstant, 1),
					makeInst(OpBinary, int(token.Add)),
					makeInst(OpReturn, 1),
				), funcName("f")),
			},
			compFunc(concatInsts(
				makeInst(OpGetBuiltin, int(BuiltinFunc)),
				makeInst(OpConstant, 2),
				makeInst(OpCall, 1, 0),
				makeInst(OpDefineLocal, 0),
				makeInst(OpReturn, 0),
			), funcLocals(1)),
		))
	}

	expectCompile(t, `func f() { 1; 2 }`, bytecode(
		Array{
			Int(1),
			Int(2),
			compFunc(concatInsts(
				makeInst(OpConstant, 0),
				makeInst(OpPop),
				makeInst(OpConstant, 1),
				makeInst(OpPop),
				makeInst(OpReturn, 0),
			), funcName("f")),
		},
		compFunc(concatInsts(
			makeInst(OpGetBuiltin, int(BuiltinFunc)),
			makeInst(OpConstant, 2),
			makeInst(OpCall, 1, 0),
			makeInst(OpDefineLocal, 0),
			makeInst(OpReturn, 0),
		), funcLocals(1)),
	))

	expectCompile(t, `func f() { 1; return 2 }`, bytecode(
		Array{
			Int(1),
			Int(2),
			compFunc(concatInsts(
				makeInst(OpConstant, 0),
				makeInst(OpPop),
				makeInst(OpConstant, 1),
				makeInst(OpReturn, 1),
			), funcName("f")),
		},
		compFunc(concatInsts(
			makeInst(OpGetBuiltin, int(BuiltinFunc)),
			makeInst(OpConstant, 2),
			makeInst(OpCall, 1, 0),
			makeInst(OpDefineLocal, 0),
			makeInst(OpReturn, 0),
		), funcLocals(1)),
	))

	expectCompile(t, `func f() { if(true) { return 1 } else { return 2 } }`, bytecode(
		Array{
			Int(1),
			Int(2),
			compFunc(concatInsts(
				makeInst(OpTrue),          // 0000
				makeInst(OpJumpFalsy, 12), // 0001
				makeInst(OpConstant, 0),   // 0004
				makeInst(OpReturn, 1),     // 0007
				makeInst(OpJump, 17),      // 0009
				makeInst(OpConstant, 1),   // 0012
				makeInst(OpReturn, 1),     // 0015
				makeInst(OpReturn, 0),     // 0017
			), funcName("f")),
		},
		compFunc(concatInsts(
			makeInst(OpGetBuiltin, int(BuiltinFunc)),
			makeInst(OpConstant, 2),
			makeInst(OpCall, 1, 0),
			makeInst(OpDefineLocal, 0),
			makeInst(OpReturn, 0),
		), funcLocals(1)),
	))

	expectCompile(t, `func f() { 1; if(true) { 2 } else { 3 }; 4 }`, bytecode(
		Array{
			Int(1),
			Int(2),
			Int(3),
			Int(4),
			compFunc(concatInsts(
				makeInst(OpConstant, 0),   // 0000
				makeInst(OpPop),           // 0003
				makeInst(OpTrue),          // 0004
				makeInst(OpJumpFalsy, 15), // 0005
				makeInst(OpConstant, 1),   // 0008
				makeInst(OpPop),           // 0011
				makeInst(OpJump, 19),      // 0012
				makeInst(OpConstant, 2),   // 0015
				makeInst(OpPop),           // 0018
				makeInst(OpConstant, 3),   // 0019
				makeInst(OpPop),           // 0022
				makeInst(OpReturn, 0),     // 0023
			), funcName("f")),
		},
		compFunc(concatInsts(
			makeInst(OpGetBuiltin, int(BuiltinFunc)),
			makeInst(OpConstant, 4),
			makeInst(OpCall, 1, 0),
			makeInst(OpDefineLocal, 0),
			makeInst(OpReturn, 0),
		), funcLocals(1)),
	))

	expectCompile(t, `func f() { }`, bytecode(
		Array{
			compFunc(concatInsts(
				makeInst(OpReturn, 0),
			), funcName("f")),
		},
		compFunc(concatInsts(
			makeInst(OpGetBuiltin, int(BuiltinFunc)),
			makeInst(OpConstant, 0),
			makeInst(OpCall, 1, 0),
			makeInst(OpDefineLocal, 0),
			makeInst(OpReturn, 0),
		), funcLocals(1)),
	))

	expectCompile(t, `f () => { a: 1 }`, bytecode(
		Array{
			Str("a"),
			Int(1),
			compFunc(concatInsts(
				makeInst(OpConstant, 0),
				makeInst(OpConstant, 1),
				makeInst(OpDict, 2),
				makeInst(OpReturn, 1),
			), funcName("f")),
		},
		compFunc(concatInsts(
			makeInst(OpGetBuiltin, int(BuiltinFunc)),
			makeInst(OpConstant, 2),
			makeInst(OpCall, 1, 0),
			makeInst(OpDefineLocal, 0),
			makeInst(OpReturn, 0),
		), funcLocals(1)),
	))

	expectCompile(t, `func() { 24 }()`, bytecode(
		Array{
			Int(24),
			compFunc(concatInsts(
				makeInst(OpConstant, 0),
				makeInst(OpPop),
				makeInst(OpReturn, 0),
			)),
		},
		compFunc(concatInsts(
			makeInst(OpConstant, 1),
			makeInst(OpCall, 0, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `() => { 24 }()`, bytecode(
		Array{
			Int(24),
			compFunc(concatInsts(
				makeInst(OpConstant, 0),
				makeInst(OpReturn, 1),
			)),
		},
		compFunc(concatInsts(
			makeInst(OpConstant, 1),
			makeInst(OpCall, 0, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `func() { return 24 }()`, bytecode(
		Array{
			Int(24),
			compFunc(concatInsts(
				makeInst(OpConstant, 0),
				makeInst(OpReturn, 1),
			)),
		},
		compFunc(concatInsts(
			makeInst(OpConstant, 1),
			makeInst(OpCall, 0, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `(() => 24)()`, bytecode(
		Array{
			Int(24),
			compFunc(concatInsts(
				makeInst(OpConstant, 0),
				makeInst(OpReturn, 1),
			)),
		},
		compFunc(concatInsts(
			makeInst(OpConstant, 1),
			makeInst(OpCall, 0, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `() => { return 24 }()`, bytecode(
		Array{
			Int(24),
			compFunc(concatInsts(
				makeInst(OpConstant, 0),
				makeInst(OpReturn, 1),
			)),
		},
		compFunc(concatInsts(
			makeInst(OpConstant, 1),
			makeInst(OpCall, 0, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `func f () { 24 }; f();`, bytecode(
		Array{
			Int(24),
			compFunc(concatInsts(
				makeInst(OpConstant, 0),
				makeInst(OpPop),
				makeInst(OpReturn, 0),
			)),
		},
		compFunc(concatInsts(
			makeInst(OpGetBuiltin, int(BuiltinFunc)),
			makeInst(OpConstant, 1),
			makeInst(OpCall, 1, 0),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetLocal, 0),
			makeInst(OpCall, 0, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		),
			funcLocals(1),
		),
	))

	expectCompile(t, `f := func() { 24 }; f();`, bytecode(
		Array{
			Int(24),
			compFunc(concatInsts(
				makeInst(OpConstant, 0),
				makeInst(OpPop),
				makeInst(OpReturn, 0),
			)),
		},
		compFunc(concatInsts(
			makeInst(OpConstant, 1),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetLocal, 0),
			makeInst(OpCall, 0, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		),
			funcLocals(1),
		),
	))

	expectCompile(t, `f := () => 24; f()`, bytecode(
		Array{
			Int(24),
			compFunc(concatInsts(
				makeInst(OpConstant, 0),
				makeInst(OpReturn, 1),
			)),
		},
		compFunc(concatInsts(
			makeInst(OpConstant, 1),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetLocal, 0),
			makeInst(OpCall, 0, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		),
			funcLocals(1),
		),
	))

	expectCompile(t, `f := func() { return 24 }; f();`, bytecode(
		Array{
			Int(24),
			compFunc(concatInsts(
				makeInst(OpConstant, 0),
				makeInst(OpReturn, 1),
			)),
		},
		compFunc(concatInsts(
			makeInst(OpConstant, 1),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetLocal, 0),
			makeInst(OpCall, 0, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		),
			funcLocals(1),
		),
	))

	expectCompile(t, `n := 55; func f() { n };`, bytecode(
		Array{
			Int(55),
			compFunc(concatInsts(
				makeInst(OpGetFree, 0),
				makeInst(OpPop),
				makeInst(OpReturn, 0),
			), funcName("f")),
		},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetBuiltin, int(BuiltinFunc)),
			makeInst(OpGetLocalPtr, 0),
			makeInst(OpClosure, 1, 1),
			makeInst(OpCall, 1, 0),
			makeInst(OpDefineLocal, 1),
			makeInst(OpReturn, 0),
		),
			funcLocals(2),
		),
	))

	expectCompile(t, `func f() { n := 55; return n }`, bytecode(
		Array{
			Int(55),
			compFunc(concatInsts(
				makeInst(OpConstant, 0),
				makeInst(OpDefineLocal, 0),
				makeInst(OpGetLocal, 0),
				makeInst(OpReturn, 1),
			),
				funcName("f"),
				funcLocals(1),
			),
		},
		compFunc(concatInsts(
			makeInst(OpGetBuiltin, int(BuiltinFunc)),
			makeInst(OpConstant, 1),
			makeInst(OpCall, 1, 0),
			makeInst(OpDefineLocal, 0),
			makeInst(OpReturn, 0),
		), funcLocals(1)),
	))

	expectCompile(t, `(func() { a := 55; b := 77; return a + b })`, bytecode(
		Array{
			Int(55),
			Int(77),
			compFunc(concatInsts(
				makeInst(OpConstant, 0),
				makeInst(OpDefineLocal, 0),
				makeInst(OpConstant, 1),
				makeInst(OpDefineLocal, 1),
				makeInst(OpGetLocal, 0),
				makeInst(OpGetLocal, 1),
				makeInst(OpBinary, int(token.Add)),
				makeInst(OpReturn, 1),
			),
				funcLocals(2),
			),
		},
		compFunc(concatInsts(
			makeInst(OpConstant, 2),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `f := func(a) { return a }; f(24);`, bytecode(
		Array{
			compFunc(concatInsts(
				makeInst(OpGetLocal, 0),
				makeInst(OpReturn, 1),
			),
				funcParams("a"),
				funcLocals(1),
			),
			Int(24),
		},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetLocal, 0),
			makeInst(OpConstant, 1),
			makeInst(OpCall, 1, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		),
			funcLocals(1),
		),
	))

	expectCompile(t, `f := func(*a) { return a }; f(1, 2, 3);`, bytecode(
		Array{
			compFunc(concatInsts(
				makeInst(OpGetLocal, 0),
				makeInst(OpReturn, 1),
			),
				funcParams("*a"),
				funcLocals(1),
			),
			Int(1),
			Int(2),
			Int(3),
		},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetLocal, 0),
			makeInst(OpConstant, 1),
			makeInst(OpConstant, 2),
			makeInst(OpConstant, 3),
			makeInst(OpCall, 3, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		),
			funcLocals(1),
		),
	))

	expectCompile(t, `f := func(a, b, c) { a; b; return c; }; f(24, 25, 26);`, bytecode(
		Array{
			compFunc(concatInsts(
				makeInst(OpGetLocal, 0),
				makeInst(OpPop),
				makeInst(OpGetLocal, 1),
				makeInst(OpPop),
				makeInst(OpGetLocal, 2),
				makeInst(OpReturn, 1),
			),
				funcParams("a", "b", "c"),
				funcLocals(3),
			),
			Int(24),
			Int(25),
			Int(26),
		},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetLocal, 0),
			makeInst(OpConstant, 1),
			makeInst(OpConstant, 2),
			makeInst(OpConstant, 3),
			makeInst(OpCall, 3, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		),
			funcLocals(1),
		),
	))

	expectCompile(t, `(func() { n := 55; n = 23; return n })`, bytecode(
		Array{
			Int(55),
			Int(23),
			compFunc(concatInsts(
				makeInst(OpConstant, 0),
				makeInst(OpDefineLocal, 0),
				makeInst(OpConstant, 1),
				makeInst(OpSetLocal, 0),
				makeInst(OpGetLocal, 0),
				makeInst(OpReturn, 1),
			),
				funcLocals(1),
			),
		},
		compFunc(concatInsts(
			makeInst(OpConstant, 2),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `len([]);`, bytecode(
		Array{},
		compFunc(concatInsts(
			makeInst(OpGetBuiltin, int(BuiltinLen)),
			makeInst(OpArray, 0),
			makeInst(OpCall, 1, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `(func() { return len([]) })`, bytecode(
		Array{
			compFunc(concatInsts(
				makeInst(OpGetBuiltin, int(BuiltinLen)),
				makeInst(OpArray, 0),
				makeInst(OpCall, 1, 0),
				makeInst(OpReturn, 1),
			)),
		},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `(func(a) { (func(b) { return a + b }) })`, bytecode(
		Array{
			compFunc(concatInsts(
				makeInst(OpGetFree, 0),
				makeInst(OpGetLocal, 0),
				makeInst(OpBinary, int(token.Add)),
				makeInst(OpReturn, 1),
			),
				funcParams("b"),
				funcLocals(1),
			),

			compFunc(concatInsts(
				makeInst(OpGetLocalPtr, 0),
				makeInst(OpClosure, 0, 1),
				makeInst(OpPop),
				makeInst(OpReturn, 0),
			),
				funcParams("a"),
				funcLocals(1),
			),
		},
		compFunc(concatInsts(
			makeInst(OpConstant, 1),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `(func(a) {
                return func(b) {
                        return func(c) {
                                return a + b + c
                        }
                }
        })`, bytecode(
		Array{
			compFunc(concatInsts(
				makeInst(OpGetFree, 0),
				makeInst(OpGetFree, 1),
				makeInst(OpBinary, int(token.Add)),
				makeInst(OpGetLocal, 0),
				makeInst(OpBinary, int(token.Add)),
				makeInst(OpReturn, 1),
			),
				funcName("#1"),
				funcParams("c"),
				funcLocals(1),
			),

			compFunc(concatInsts(
				makeInst(OpGetFreePtr, 0),
				makeInst(OpGetLocalPtr, 0),
				makeInst(OpClosure, 0, 2),
				makeInst(OpReturn, 1),
			),
				funcName("#2"),
				funcParams("b"),
				funcLocals(1),
			),

			compFunc(concatInsts(
				makeInst(OpGetLocalPtr, 0),
				makeInst(OpClosure, 1, 1),
				makeInst(OpReturn, 1),
			),
				funcName("#3"),
				funcParams("a"),
				funcLocals(1),
			),
		},
		compFunc(concatInsts(
			makeInst(OpConstant, 2),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `
        g := 55;
        (func() {
                a := 66;

                return func() {
                        b := 77;

                        return func() {
                                c := 88;

                                return g + a + b + c;
                        }
                }
        })`, bytecode(
		Array{
			Int(55),
			Int(66),
			Int(77),
			Int(88),
			compFunc(concatInsts(
				makeInst(OpConstant, 3),
				makeInst(OpDefineLocal, 0),
				makeInst(OpGetFree, 0),
				makeInst(OpGetFree, 1),
				makeInst(OpBinary, int(token.Add)),
				makeInst(OpGetFree, 2),
				makeInst(OpBinary, int(token.Add)),
				makeInst(OpGetLocal, 0),
				makeInst(OpBinary, int(token.Add)),
				makeInst(OpReturn, 1),
			),
				funcLocals(1),
			),

			compFunc(concatInsts(
				makeInst(OpConstant, 2),
				makeInst(OpDefineLocal, 0),
				makeInst(OpGetFreePtr, 0),
				makeInst(OpGetFreePtr, 1),
				makeInst(OpGetLocalPtr, 0),
				makeInst(OpClosure, 4, 3),
				makeInst(OpReturn, 1),
			),
				funcLocals(1),
			),

			compFunc(concatInsts(
				makeInst(OpConstant, 1),
				makeInst(OpDefineLocal, 0),
				makeInst(OpGetFreePtr, 0),
				makeInst(OpGetLocalPtr, 0),
				makeInst(OpClosure, 5, 2),
				makeInst(OpReturn, 1),
			),
				funcLocals(1),
			),
		},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetLocalPtr, 0),
			makeInst(OpClosure, 6, 1),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		),
			funcLocals(1),
		),
	))

	// Block variables not used as free variable is set to nil after loop.
	// If block variable is not used as free variable it is reused.
	expectCompile(t, `for i:=0; i<10; i++ {}; j := 1`, bytecode(
		Array{Int(0), Int(10), Int(1)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),                // 0000
			makeInst(OpDefineLocal, 0),             // 0003
			makeInst(OpGetLocal, 0),                // 0005
			makeInst(OpConstant, 1),                // 0007
			makeInst(OpBinary, int(token.Less)),    // 0010
			makeInst(OpJumpFalsy, 27),              // 0012
			makeInst(OpGetLocal, 0),                // 0015
			makeInst(OpConstant, 2),                // 0017
			makeInst(OpSelfAssign, int(token.Add)), // 0020
			makeInst(OpSetLocal, 0),                // 0022
			makeInst(OpJump, 5),                    // 0024
			makeInst(OpConstant, 2),                // 0027
			makeInst(OpDefineLocal, 0),             // 0030
			makeInst(OpReturn, 0),                  // 0032
		),
			funcLocals(1),
		),
	))

	expectCompile(t, `m := {}; for k, v in m { }`, bytecode(
		Array{},
		compFunc(concatInsts(
			makeInst(OpDict, 0),        // 0000
			makeInst(OpDefineLocal, 0), // 0003
			makeInst(OpGetLocal, 0),    // 0005
			makeInst(OpIterInit),       // 0007
			makeInst(OpDefineLocal, 1), // 0008 :it
			makeInst(OpGetLocal, 1),    // 0010 :it
			makeInst(OpIterNext),       // 0012
			makeInst(OpJumpFalsy, 29),  // 0013
			makeInst(OpGetLocal, 1),    // 0016
			makeInst(OpIterKey),        // 0018
			makeInst(OpDefineLocal, 2), // 0019 k
			makeInst(OpGetLocal, 1),    // 0021 :it
			makeInst(OpIterValue),      // 0023
			makeInst(OpDefineLocal, 3), // 0024 v
			makeInst(OpJump, 10),       // 0026
			makeInst(OpReturn, 0),      // 0029
		),
			funcLocals(4), // m, :it, k, v
		),
	))

	expectCompile(t, `a := 0; a == 0 && a != 1 || a < 1`, bytecode(
		Array{Int(0), Int(1)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),             // 0000
			makeInst(OpDefineLocal, 0),          // 0003
			makeInst(OpGetLocal, 0),             // 0005
			makeInst(OpConstant, 0),             // 0007
			makeInst(OpEqual),                   // 0010
			makeInst(OpAndJump, 20),             // 0011
			makeInst(OpGetLocal, 0),             // 0014
			makeInst(OpConstant, 1),             // 0016
			makeInst(OpNotEqual),                // 0019
			makeInst(OpOrJump, 30),              // 0020
			makeInst(OpGetLocal, 0),             // 0023
			makeInst(OpConstant, 1),             // 0025
			makeInst(OpBinary, int(token.Less)), // 0028
			makeInst(OpPop),                     // 0030
			makeInst(OpReturn, 0),               // 0031
		),
			funcLocals(1),
		),
	))

	expectCompile(t, `try { a:=0 } catch err { } finally { err; a; }; x:=1`, bytecode(
		Array{Int(0), Int(1)},
		compFunc(concatInsts(
			makeInst(OpSetupTry, 16, 19), // 0000 // catch and finally positions
			makeInst(OpConstant, 0),      // 0005
			makeInst(OpDefineLocal, 0),   // 0008 a
			makeInst(OpNil),              // 0010
			makeInst(OpDefineLocal, 1),   // 0011 err
			makeInst(OpJump, 19),         // 0013 // jump to finally if no error
			makeInst(OpSetupCatch),       // 0016
			makeInst(OpSetLocal, 1),      // 0017
			makeInst(OpSetupFinally),     // 0019
			makeInst(OpGetLocal, 1),      // 0020
			makeInst(OpPop),              // 0022
			makeInst(OpGetLocal, 0),      // 0023
			makeInst(OpPop),              // 0025
			makeInst(OpThrow, 0),         // 0026
			makeInst(OpConstant, 1),      // 0028
			makeInst(OpDefineLocal, 0),   // 0031 x
			makeInst(OpReturn, 0),        // 0033
		),
			funcLocals(2),
		),
	))

	expectCompile(t, `try { a:=0 } catch err { }`, bytecode(
		Array{Int(0)},
		compFunc(concatInsts(
			makeInst(OpSetupTry, 16, 19), // 0000
			makeInst(OpConstant, 0),      // 0005
			makeInst(OpDefineLocal, 0),   // 0008 a
			makeInst(OpNil),              // 0010
			makeInst(OpDefineLocal, 1),   // 0011 err
			makeInst(OpJump, 19),         // 0010
			makeInst(OpSetupCatch),       // 0013
			makeInst(OpSetLocal, 1),      // 0014
			makeInst(OpSetupFinally),     // 0016 always OpSetupFinally
			makeInst(OpThrow, 0),         // 0023
			makeInst(OpReturn, 0),        // 0025
		),
			funcLocals(2),
		),
	))

	expectCompile(t, `try { a:=0; throw "an error" } catch { }`, bytecode(
		Array{Int(0), Str("an error")},
		compFunc(concatInsts(
			makeInst(OpSetupTry, 18, 20), // 0000
			makeInst(OpConstant, 0),      // 0005
			makeInst(OpDefineLocal, 0),   // 0008 a
			makeInst(OpConstant, 1),      // 0010
			makeInst(OpThrow, 1),         // 0013
			makeInst(OpJump, 20),         // 0015
			makeInst(OpSetupCatch),       // 0018
			makeInst(OpPop),              // 0019
			makeInst(OpSetupFinally),     // 0020
			makeInst(OpThrow, 0),         // 0021
			makeInst(OpReturn, 0),        // 0023
		),
			funcLocals(1),
		),
	))
	expectCompileError(t, `try {};`, `Parse Error: expected 'finally', found ';'`)
	expectCompileError(t, `catch {}`, `Parse Error: expected statement, found 'catch'`)
	expectCompileError(t, `finally {}`, `Parse Error: expected statement, found 'finally'`)
	// catch and finally must in the same line with right brace.
	expectCompileError(t, `try {}
        catch {}`, `Parse Error: expected 'finally', found newline`)
	expectCompileError(t, `try {
        } catch {}
        finally {}`, `Parse Error: expected statement, found 'finally'`)

	expectCompile(t, `nil || 1`, bytecode(
		Array{Int(1)},
		compFunc(concatInsts(
			makeInst(OpNil),
			makeInst(OpOrJump, 7),   // 0020
			makeInst(OpConstant, 0), // 0025
			makeInst(OpPop),         // 0030
			makeInst(OpReturn, 0),   // 0031
		)),
	))

	expectCompile(t, `nil ?? 1`, bytecode(
		Array{Int(1)},
		compFunc(concatInsts(
			makeInst(OpNil),
			makeInst(OpJumpNotNil, 7), // 0020
			makeInst(OpConstant, 0),   // 0025
			makeInst(OpPop),           // 0030
			makeInst(OpReturn, 0),     // 0031
		)),
	))

	expectCompile(t, `a := 1; a ??= 2`, bytecode(
		Array{Int(1), Int(2)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetLocal, 0),
			makeInst(OpJumpNotNil, 15),
			makeInst(OpConstant, 1),
			makeInst(OpSetLocal, 0),
			makeInst(OpReturn, 0),
		),
			funcLocals(1),
		)))

	expectCompile(t, `a := 1; b := 2; a ??= b`, bytecode(
		Array{Int(1), Int(2)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpDefineLocal, 0),
			makeInst(OpConstant, 1),
			makeInst(OpDefineLocal, 1),
			makeInst(OpGetLocal, 0),
			makeInst(OpJumpNotNil, 19),
			makeInst(OpGetLocal, 1),
			makeInst(OpSetLocal, 0),
			makeInst(OpReturn, 0),
		),
			funcLocals(2),
		)))

	expectCompile(t, `a := 1; a ||= 2`, bytecode(
		Array{Int(1), Int(2)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetLocal, 0),
			makeInst(OpOrJump, 15),
			makeInst(OpConstant, 1),
			makeInst(OpSetLocal, 0),
			makeInst(OpReturn, 0),
		),
			funcLocals(1),
		)))

	expectCompile(t, `a := 1; b := 2; a ||= b`, bytecode(
		Array{Int(1), Int(2)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpDefineLocal, 0),
			makeInst(OpConstant, 1),
			makeInst(OpDefineLocal, 1),
			makeInst(OpGetLocal, 0),
			makeInst(OpOrJump, 19),
			makeInst(OpGetLocal, 1),
			makeInst(OpSetLocal, 0),
			makeInst(OpReturn, 0),
		),
			funcLocals(2),
		)))

	expectCompile(t, `var $a`, bytecode(
		Array{},
		compFunc(concatInsts(
			makeInst(OpNil),
			makeInst(OpDefineLocal, 0),
			makeInst(OpReturn, 0),
		),
			funcLocals(1),
		),
	))

	expectCompile(t, `$ := 1`, bytecode(
		Array{Int(1)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpDefineLocal, 0),
			makeInst(OpReturn, 0),
		),
			funcLocals(1),
		),
	))

	// 4 instructions are generated for every source module import.
	// If module's returned value is already stored, ignore storing.
	moduleMap := NewModuleMap()
	moduleMap.AddSourceModule("mod", []byte(``))
	expectCompileWithOpts(t, `import("mod")`,
		CompileOptions{CompilerOptions: CompilerOptions{
			ModuleMap: moduleMap,
		}},
		bytecode(
			Array{},
			compFunc(concatInsts(
				makeInst(OpLoadModule, 1),    // 0000 constant, module indexes
				makeInst(OpJumpFalsy, 9),     // 0005 if loaded no call is required
				makeInst(OpInitModule, 0, 0), // 0008 obtain return value from module
				makeInst(OpPop),              // 0014
				makeInst(OpReturn, 0),        // 0015
			)),
			withModules(func(b *ModuleSpecBuilder) {
				b.Compiled("mod", compFunc(concatInsts(
					makeInst(OpReturn, 0),
				)))
			}),
		),
	)

	// 3 instructions are generated for non-source module import.
	// If module's value is already stored, ignore storing.
	moduleMap = NewModuleMap()
	moduleMap.AddBuiltinModule("mod", Dict{})
	expectCompileWithOpts(t, `import("mod")`,
		CompileOptions{CompilerOptions: CompilerOptions{
			ModuleMap: moduleMap,
		}},
		bytecode(
			Array{},
			compFunc(concatInsts(
				makeInst(OpLoadModule, 1),    // 0000 constant, module indexes
				makeInst(OpJumpFalsy, 9),     // 0005 if loaded no call is required
				makeInst(OpInitModule, 0, 0), // 0005 if loaded no call is required
				makeInst(OpPop),              // 0011
				makeInst(OpReturn, 0),        // 0012
			)),
			withModules(func(b *ModuleSpecBuilder) {
				b.Name("mod")
			}),
		),
	)

	// unknown module name
	expectCompileError(t, `import("user1")`, "Compile Error: module 'user1' not found")
	expectCompileError(t, `import("")`, "Compile Error: empty module name")
	// too many errors

	expectCompileError(t, `
        (func() {
                fn := fn()
        })()    
        `, `Compile Error: unresolved reference "fn"`)

	expectCompile(t, `x, y := []`,
		bytecode(
			// 2: number of LHS idents
			// 0: array index to assign to x
			// 1: array index to assign to y
			Array{Int(2), Int(0), Int(1)},
			compFunc(concatInsts(
				makeInst(OpGetBuiltin,
					int(BuiltinMakeArray)), // load builtin to call
				makeInst(OpConstant, 0),    // load lhs length
				makeInst(OpArray, 0),       // rhs empty array
				makeInst(OpCall, 2, 0),     // call builtin :makeArray(2, [])
				makeInst(OpDefineLocal, 0), // set builtin call result to :array
				makeInst(OpGetLocal, 0),    // load :array
				makeInst(OpConstant, 1),    // load 0 (array index)
				makeInst(OpGetIndex, 1),    // :array[0]
				makeInst(OpDefineLocal, 1), // x = :array[0]
				makeInst(OpGetLocal, 0),    // load :array
				makeInst(OpConstant, 2),    // load 1 (array index)
				makeInst(OpGetIndex, 1),    // :array[1]
				makeInst(OpDefineLocal, 2), // y = :array[1]
				makeInst(OpNil),            // load nil
				makeInst(OpSetLocal, 0),    // cleanup -> :array = nil
				makeInst(OpReturn, 0),
			),
				// x,y and :array hidden variable
				funcLocals(3),
			),
		),
	)

	expectCompile(t, `(func() { return 1, 2 })`,
		bytecode(
			Array{
				Int(1),
				Int(2),
				compFunc(concatInsts(
					makeInst(OpConstant, 0),
					makeInst(OpConstant, 1),
					makeInst(OpArray, 2),
					makeInst(OpReturn, 1),
				),
				),
			},
			compFunc(concatInsts(
				makeInst(OpConstant, 2),
				makeInst(OpPop),
				makeInst(OpReturn, 0),
			),
			),
		),
	)

	expectCompile(t, `var a; return a["b"]`,
		bytecode(
			Array{Str("b")},
			compFunc(concatInsts(
				makeInst(OpNil),
				makeInst(OpDefineLocal, 0),
				makeInst(OpGetLocal, 0),
				makeInst(OpConstant, 0),
				makeInst(OpGetIndex, 1),
				makeInst(OpReturn, 1),
			),
				funcLocals(1),
			),
		),
	)

	expectCompile(t, `var a; return a["b"]["c"][2]`,
		bytecode(
			Array{Str("b"), Str("c"), Int(2)},
			compFunc(concatInsts(
				makeInst(OpNil),
				makeInst(OpDefineLocal, 0),
				makeInst(OpGetLocal, 0),
				makeInst(OpConstant, 0),
				makeInst(OpConstant, 1),
				makeInst(OpConstant, 2),
				makeInst(OpGetIndex, 3),
				makeInst(OpReturn, 1),
			),
				funcLocals(1),
			),
		),
	)

	expectCompile(t, `f := func(*a) { return a }; f(1, 2, 3);`, bytecode(
		Array{
			compFunc(concatInsts(
				makeInst(OpGetLocal, 0),
				makeInst(OpReturn, 1),
			),
				funcParams("*a"),
				funcLocals(1),
			),
			Int(1),
			Int(2),
			Int(3),
		},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetLocal, 0),
			makeInst(OpConstant, 1),
			makeInst(OpConstant, 2),
			makeInst(OpConstant, 3),
			makeInst(OpCall, 3, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		),
			funcLocals(1),
		),
	))
}

func TestCompilerDeferStmt(t *testing.T) {
	// defer outside a function body is rejected with a friendly message
	expectCompileError(t, `defer { x }`,
		`Compile Error: defer is only allowed inside a function body`)

	// deferb outside a block is rejected
	expectCompileError(t, `deferb { x }`,
		`Compile Error: deferb is only allowed inside a block`)

	// a defer-using function desugars into a wrapper that creates extra
	// compiled functions (the $__body thunk and one handler closure per defer)
	st := NewSymbolTable(NewBuiltins().NameSet)
	_, bc, err := Compile(st, []byte(`f := func() { defer { x := 1 } }`),
		CompileOptions{})
	require.NoError(t, err)

	var fnCount int
	for _, cnst := range bc.Constants {
		if _, ok := cnst.(*CompiledFunction); ok {
			fnCount++
		}
	}
	// f itself + $__body thunk + defer handler closure
	require.GreaterOrEqual(t, fnCount, 3,
		"expected defer desugar to generate the thunk and handler closures")

	// shortcut forms compile: a call (passing $ret/$err) and a braceless
	// assignment both desugar without error.
	for _, src := range []string{
		`cleanup := func(r, e) {}; f := func() { defer cleanup($ret, $err); return 1 }`,
		`f := func() { defer $ret += 1; return 1 }`,
		`f := func() { out := ""; { deferb out += "x" } }`,
		`f := func() { n := 0; { deferb n++ } }`,
	} {
		_, _, err := Compile(NewSymbolTable(NewBuiltins().NameSet), []byte(src), CompileOptions{})
		require.NoError(t, err, src)
	}
}

func TestCompilerArrayComprehension(t *testing.T) {
	// `[i for i in [9]]` desugars to: :compr = []; for i in [9] { :compr =
	// append(:compr, i) }; <push :compr>
	expectCompile(t, `return [i for i in [9]]`, bytecode(
		Array{Int(9)},
		compFunc(concatInsts(
			makeInst(OpArray, 0),                       // 0000 :compr = []
			makeInst(OpDefineLocal, 0),                 // 0003
			makeInst(OpConstant, 0),                    // 0005 9
			makeInst(OpArray, 1),                       // 0008 [9]
			makeInst(OpIterInit),                       // 0011
			makeInst(OpDefineLocal, 1),                 // 0012 iterator
			makeInst(OpGetLocal, 1),                    // 0014 loop head
			makeInst(OpIterNext),                       // 0016
			makeInst(OpJumpFalsy, 40),                  // 0017 -> end
			makeInst(OpGetLocal, 1),                    // 0020
			makeInst(OpIterValue),                      // 0022
			makeInst(OpDefineLocal, 2),                 // 0023 i
			makeInst(OpGetBuiltin, int(BuiltinAppend)), // 0025
			makeInst(OpGetLocal, 0),                    // 0028 :compr
			makeInst(OpGetLocal, 2),                    // 0030 i
			makeInst(OpCall, 2, 0),                     // 0032 append(:compr, i)
			makeInst(OpSetLocal, 0),                    // 0035 :compr = ...
			makeInst(OpJump, 14),                       // 0037 loop
			makeInst(OpGetLocal, 0),                    // 0040 push :compr
			makeInst(OpReturn, 1),                      // 0042
			makeInst(OpReturn, 0),                      // 0044 implicit trailing return
		),
			funcLocals(3),
		),
	))
}

func TestCompilerMatchExpr(t *testing.T) {
	// subject -> :match local; each arm condition compares with OpEqual; a match
	// jumps to the arm body, otherwise control falls to the next arm. The else
	// arm is the fallthrough default.
	expectCompile(t, `return match 1 { 1: "a", else: "b" }`, bytecode(
		Array{Int(1), Str("a"), Str("b")},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),    // 0000 subject 1
			makeInst(OpDefineLocal, 0), // 0003 :match
			makeInst(OpGetLocal, 0),    // 0005 :match
			makeInst(OpConstant, 0),    // 0007 cond 1
			makeInst(OpEqual),          // 0010
			makeInst(OpJumpFalsy, 17),  // 0011 -> next arm
			makeInst(OpJump, 20),       // 0014 -> body
			makeInst(OpJump, 26),       // 0017 -> else
			makeInst(OpConstant, 1),    // 0020 "a"
			makeInst(OpJump, 29),       // 0023 -> end
			makeInst(OpConstant, 2),    // 0026 else "b"
			makeInst(OpReturn, 1),      // 0029
		),
			funcLocals(1),
		),
	))

	// statement form leaves no value on the stack (no OpNil, no OpPop), and a
	// no-match with no else simply falls through.
	expectCompile(t, `match 1 { 1 {} }`, bytecode(
		Array{Int(1)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),    // 0000 subject 1
			makeInst(OpDefineLocal, 0), // 0003 :match
			makeInst(OpGetLocal, 0),    // 0005 :match
			makeInst(OpConstant, 0),    // 0007 cond 1
			makeInst(OpEqual),          // 0010
			makeInst(OpJumpFalsy, 17),  // 0011 -> next arm
			makeInst(OpJump, 20),       // 0014 -> body
			makeInst(OpJump, 23),       // 0017 -> after arm
			makeInst(OpJump, 23),       // 0020 body end -> end
			makeInst(OpReturn, 0),      // 0023
		),
			funcLocals(1),
		),
	))
}

func TestCompilerMatchExprForms(t *testing.T) {
	// multiple conditions per arm (OR): each cond jumps to the shared body.
	expectCompile(t, `return match 1 { 1, 2: "a", else: "b" }`, bytecode(
		Array{Int(1), Int(2), Str("a"), Str("b")},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),    // 0000 subject 1
			makeInst(OpDefineLocal, 0), // 0003 :match
			makeInst(OpGetLocal, 0),    // 0005 cond 1
			makeInst(OpConstant, 0),    // 0007
			makeInst(OpEqual),          // 0010
			makeInst(OpJumpFalsy, 17),  // 0011 -> next cond
			makeInst(OpJump, 32),       // 0014 -> body
			makeInst(OpGetLocal, 0),    // 0017 cond 2
			makeInst(OpConstant, 1),    // 0019
			makeInst(OpEqual),          // 0022
			makeInst(OpJumpFalsy, 29),  // 0023 -> next arm
			makeInst(OpJump, 32),       // 0026 -> body
			makeInst(OpJump, 38),       // 0029 -> else
			makeInst(OpConstant, 2),    // 0032 "a"
			makeInst(OpJump, 41),       // 0035 -> end
			makeInst(OpConstant, 3),    // 0038 else "b"
			makeInst(OpReturn, 1),      // 0041
		),
			funcLocals(1),
		),
	))

	// no else: a no-match falls through to OpNil (the expression yields nil).
	expectCompile(t, `return match 1 { 1: "a", 2: "b" }`, bytecode(
		Array{Int(1), Str("a"), Int(2), Str("b")},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),    // 0000 subject 1
			makeInst(OpDefineLocal, 0), // 0003 :match
			makeInst(OpGetLocal, 0),    // 0005 arm0 cond 1
			makeInst(OpConstant, 0),    // 0007
			makeInst(OpEqual),          // 0010
			makeInst(OpJumpFalsy, 17),  // 0011 -> next arm
			makeInst(OpJump, 20),       // 0014 -> body0
			makeInst(OpJump, 26),       // 0017 -> arm1
			makeInst(OpConstant, 1),    // 0020 "a"
			makeInst(OpJump, 48),       // 0023 -> end
			makeInst(OpGetLocal, 0),    // 0026 arm1 cond 2
			makeInst(OpConstant, 2),    // 0028
			makeInst(OpEqual),          // 0031
			makeInst(OpJumpFalsy, 38),  // 0032 -> after
			makeInst(OpJump, 41),       // 0035 -> body1
			makeInst(OpJump, 47),       // 0038 -> nil
			makeInst(OpConstant, 3),    // 0041 "b"
			makeInst(OpJump, 48),       // 0044 -> end
			makeInst(OpNil),            // 0047 no-match default
			makeInst(OpReturn, 1),      // 0048
		),
			funcLocals(1),
		),
	))

	// an empty match yields nil.
	expectCompile(t, `return match 1 {}`, bytecode(
		Array{Int(1)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),    // 0000 subject 1
			makeInst(OpDefineLocal, 0), // 0003 :match
			makeInst(OpNil),            // 0005
			makeInst(OpReturn, 1),      // 0006
		),
			funcLocals(1),
		),
	))
}

func TestCompilerPrefixIncDec(t *testing.T) {
	// ++a : load a, apply the unary Inc operator, store back, then yield a
	expectCompile(t, `a := 1; return ++a`, bytecode(
		Array{Int(1)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),           // 0000 a := 1
			makeInst(OpDefineLocal, 0),        // 0003
			makeInst(OpGetLocal, 0),           // 0005 load a
			makeInst(OpUnary, int(token.Inc)), // 0007 a + 1
			makeInst(OpSetLocal, 0),           // 0009 store a
			makeInst(OpGetLocal, 0),           // 0011 yield a
			makeInst(OpReturn, 1),             // 0013
		),
			funcLocals(1),
		),
	))
}

func TestCompilerMatchStmtForms(t *testing.T) {
	// statement form, multi-condition arm + else: bodies leave no value, and the
	// whole match is value-less (no OpNil, no OpPop).
	expectCompile(t, `match 1 { 1, 2 {} else {} }`, bytecode(
		Array{Int(1), Int(2)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),    // 0000 subject 1
			makeInst(OpDefineLocal, 0), // 0003 :match
			makeInst(OpGetLocal, 0),    // 0005 cond 1
			makeInst(OpConstant, 0),    // 0007
			makeInst(OpEqual),          // 0010
			makeInst(OpJumpFalsy, 17),  // 0011 -> next cond
			makeInst(OpJump, 32),       // 0014 -> body
			makeInst(OpGetLocal, 0),    // 0017 cond 2
			makeInst(OpConstant, 1),    // 0019
			makeInst(OpEqual),          // 0022
			makeInst(OpJumpFalsy, 29),  // 0023 -> next arm
			makeInst(OpJump, 32),       // 0026 -> body
			makeInst(OpJump, 35),       // 0029 -> else
			makeInst(OpJump, 35),       // 0032 body end -> end
			makeInst(OpReturn, 0),      // 0035
		),
			funcLocals(1),
		),
	))
}

func TestCompilerSpreadLiterals(t *testing.T) {
	// array merge: runs build sub-arrays joined with `+`
	expectCompile(t, `a := [9]; return [1, *a, 4]`, bytecode(
		Array{Int(9), Int(1), Int(4)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),            // 0000 9
			makeInst(OpArray, 1),               // 0003 [9]
			makeInst(OpDefineLocal, 0),         // 0006 a
			makeInst(OpConstant, 1),            // 0008 1
			makeInst(OpArray, 1),               // 0011 [1]
			makeInst(OpGetLocal, 0),            // 0014 a
			makeInst(OpBinary, int(token.Add)), // 0016 [1] + a
			makeInst(OpConstant, 2),            // 0018 4
			makeInst(OpArray, 1),               // 0021 [4]
			makeInst(OpBinary, int(token.Add)), // 0024 + [4]
			makeInst(OpReturn, 1),              // 0026
		),
			funcLocals(1),
		),
	))

	// dict merge: runs build sub-dicts joined with `+`
	expectCompile(t, `a := {p:9}; return {x:1, *a}`, bytecode(
		Array{Str("p"), Int(9), Str("x"), Int(1)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),            // 0000 "p"
			makeInst(OpConstant, 1),            // 0003 9
			makeInst(OpDict, 2),                // 0006 {p:9}
			makeInst(OpDefineLocal, 0),         // 0009 a
			makeInst(OpConstant, 2),            // 0011 "x"
			makeInst(OpConstant, 3),            // 0014 1
			makeInst(OpDict, 2),                // 0017 {x:1}
			makeInst(OpGetLocal, 0),            // 0020 a
			makeInst(OpBinary, int(token.Add)), // 0022 {x:1} + a
			makeInst(OpReturn, 1),              // 0024
		),
			funcLocals(1),
		),
	))
}

func TestCompilerRegexLit(t *testing.T) {
	// `/re/` and `/re/p` compile to the regexp() constructor (POSIX via named arg)
	st := NewSymbolTable(NewBuiltins().NameSet)
	_, _, err := Compile(st, []byte(`a := /ab+/; b := /a+/p`), CompileOptions{})
	require.NoError(t, err)

	// the pattern is compiled at compile time, so an invalid one errors then
	st = NewSymbolTable(NewBuiltins().NameSet)
	_, _, err = Compile(st, []byte(`a := /(/`), CompileOptions{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid regex")
}

func TestCompilerMixedParamsDestructure(t *testing.T) {
	// the MultiParenExpr LHS compiles (positional index/slice + dict destructure
	// of the named side); just assert it compiles cleanly.
	st := NewSymbolTable(NewBuiltins().NameSet)
	_, _, err := Compile(st,
		[]byte(`x := (1, 2; c=3); (a, b, **pr; c, p:d, r=2, **nr) := x`),
		CompileOptions{})
	require.NoError(t, err)
}

func TestCompilerDictDestructure(t *testing.T) {
	// (;a, b:_b, **o) := d  evaluates d once into :dict, copies it (because of
	// **o), reads each key (key "b" into the renamed _b), deletes consumed keys,
	// and binds the remainder to o.
	expectCompile(t, `d := {a:1}; (;a, b:_b, **o) := d`, bytecode(
		Array{Str("a"), Int(1), Str("b")},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),                    // 0000 "a"
			makeInst(OpConstant, 1),                    // 0003 1
			makeInst(OpDict, 2),                        // 0006 {a:1}
			makeInst(OpDefineLocal, 0),                 // 0009 d
			makeInst(OpGetBuiltin, int(BuiltinDict)),   // 0011 dict()
			makeInst(OpGetLocal, 0),                    // 0014 d
			makeInst(OpCall, 1, 0),                     // 0016 dict(d)
			makeInst(OpDefineLocal, 1),                 // 0019 :dict = dict(d)
			makeInst(OpGetBuiltin, int(BuiltinCopy)),   // 0021
			makeInst(OpGetLocal, 1),                    // 0024
			makeInst(OpCall, 1, 0),                     // 0026 copy(:dict)
			makeInst(OpSetLocal, 1),                    // 0029 :dict = copy
			makeInst(OpGetLocal, 1),                    // 0031
			makeInst(OpConstant, 0),                    // 0033 "a"
			makeInst(OpGetIndex, 1),                    // 0036 :dict["a"]
			makeInst(OpDefineLocal, 2),                 // 0038 a
			makeInst(OpGetBuiltin, int(BuiltinDelete)), // 0040
			makeInst(OpGetLocal, 1),                    // 0043
			makeInst(OpConstant, 0),                    // 0045 "a"
			makeInst(OpCall, 2, 0),                     // 0048 delete(:dict,"a")
			makeInst(OpPop),                            // 0051
			makeInst(OpGetLocal, 1),                    // 0052
			makeInst(OpConstant, 2),                    // 0054 "b"
			makeInst(OpGetIndex, 1),                    // 0057 :dict["b"]
			makeInst(OpDefineLocal, 3),                 // 0059 _b
			makeInst(OpGetBuiltin, int(BuiltinDelete)), // 0061
			makeInst(OpGetLocal, 1),                    // 0064
			makeInst(OpConstant, 2),                    // 0066 "b"
			makeInst(OpCall, 2, 0),                     // 0069 delete(:dict,"b")
			makeInst(OpPop),                            // 0072
			makeInst(OpGetLocal, 1),                    // 0073
			makeInst(OpDefineLocal, 4),                 // 0075 o = :dict
			makeInst(OpNil),                            // 0077
			makeInst(OpSetLocal, 1),                    // 0078 cleanup :dict
			makeInst(OpReturn, 0),                      // 0080
		),
			funcLocals(5),
		),
	))
}

func TestCompilerOrExpr(t *testing.T) {
	// `a or 2` desugars to a try/catch that stores the result in a temp local
	// (:or). On a thrown error the fallback is evaluated and bound `$err`; if the
	// fallback is itself an error it is re-thrown, else it becomes the value.
	expectCompile(t, `a := 1; return a or 2`, bytecode(
		Array{Int(1), Int(2)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),                     // 0000 a := 1
			makeInst(OpDefineLocal, 0),                  // 0003 a
			makeInst(OpNil),                             // 0005 :or = nil
			makeInst(OpDefineLocal, 1),                  // 0006 :or
			makeInst(OpSetupTry, 20, 43),                // 0008
			makeInst(OpGetLocal, 0),                     // 0013 eval a
			makeInst(OpSetLocal, 1),                     // 0015 :or = a
			makeInst(OpJump, 43),                        // 0017 -> finally
			makeInst(OpSetupCatch),                      // 0020
			makeInst(OpDefineLocal, 2),                  // 0021 $err
			makeInst(OpConstant, 1),                     // 0023 eval fallback 2
			makeInst(OpSetLocal, 1),                     // 0026 :or = 2
			makeInst(OpGetBuiltin, int(BuiltinIsError)), // 0028
			makeInst(OpGetLocal, 1),                     // 0031
			makeInst(OpCall, 1, 0),                      // 0033 isError(:or)
			makeInst(OpJumpFalsy, 43),                   // 0036 not error -> finally
			makeInst(OpGetLocal, 1),                     // 0039
			makeInst(OpThrow, 1),                        // 0041 re-throw error fallback
			makeInst(OpSetupFinally),                    // 0043
			makeInst(OpThrow, 0),                        // 0044 implicit re-throw (no-op)
			makeInst(OpGetLocal, 1),                     // 0046 push result
			makeInst(OpReturn, 1),                       // 0048
		),
			funcLocals(3),
		),
	))
}

func TestCompilerReturn(t *testing.T) {
	expectCompile(t, `return`, bytecode(
		Array{},
		compFunc(concatInsts(
			makeInst(OpReturn, 0),
		),
			funcLocals(0),
		),
	))
	expectCompile(t, `return 1`, bytecode(
		Array{Int(1)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpReturn, 1),
		),
			funcLocals(0),
		),
	))
	expectCompile(t, `nil || return`, bytecode(
		Array{},
		compFunc(concatInsts(
			makeInst(OpNil),
			makeInst(OpOrJump, 6),
			makeInst(OpReturn, 0),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		),
			funcLocals(0),
		),
	))
	expectCompile(t, `nil || return 1`, bytecode(
		Array{Int(1)},
		compFunc(concatInsts(
			makeInst(OpNil),
			makeInst(OpOrJump, 9),
			makeInst(OpConstant, 0),
			makeInst(OpReturn, 1),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		),
			funcLocals(0),
		),
	))
}

func TestCompilerFor(t *testing.T) {
	expectCompile(t, `var r = ""; for x in [] { r += str(x) } else { r += "@"}; r+="#"; return r`, bytecode(
		Array{Str(""), Str("@"), Str("#")},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpDefineLocal, 0),
			makeInst(OpArray, 0),
			makeInst(OpIterInit),
			makeInst(OpDefineLocal, 1),
			makeInst(OpGetLocal, 1),
			makeInst(OpIterNextElse, 24, 46),
			makeInst(OpGetLocal, 1),
			makeInst(OpIterNext),
			makeInst(OpJumpFalsy, 55),
			makeInst(OpGetLocal, 1),
			makeInst(OpIterValue),
			makeInst(OpDefineLocal, 2),
			makeInst(OpGetLocal, 0),
			makeInst(OpGetBuiltin, int(BuiltinStr)),
			makeInst(OpGetLocal, 2),
			makeInst(OpCall, 1, 0),
			makeInst(OpSelfAssign, int(token.Add)),
			makeInst(OpSetLocal, 0),
			makeInst(OpJump, 18),
			makeInst(OpGetLocal, 0),
			makeInst(OpConstant, 1),
			makeInst(OpSelfAssign, int(token.Add)),
			makeInst(OpSetLocal, 0),
			makeInst(OpGetLocal, 0),
			makeInst(OpConstant, 2),
			makeInst(OpSelfAssign, int(token.Add)),
			makeInst(OpSetLocal, 0),
			makeInst(OpGetLocal, 0),
			makeInst(OpReturn, 1),
			makeInst(OpReturn, 0),
		),
			funcLocals(3),
		)))
}

func TestCompilerNullishSelector(t *testing.T) {
	expectCompile(t, `var a; (a["I"+"DX"])?.d`, bytecode(
		Array{
			Str("I"),  // 1
			Str("DX"), // 2
			Str("d"),  // 3
		},
		compFunc(concatInsts(
			makeInst(OpNil),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetLocal, 0),
			makeInst(OpConstant, 0),
			makeInst(OpConstant, 1),
			makeInst(OpBinary, int(token.Add)),
			makeInst(OpGetIndex, 1),
			makeInst(OpJumpNil, 23),
			makeInst(OpConstant, 2),
			makeInst(OpGetIndex, 1),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		),
			funcLocals(1),
		)))

	expectCompile(t, `var a; a?.b["c"]?.d.e?.f.g`, bytecode(
		Array{Str("b"), Str("c"), Str("d"), Str("e"), Str("f"), Str("g")},
		compFunc(concatInsts(
			makeInst(OpNil),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetLocal, 0),
			makeInst(OpJumpNil, 44),
			makeInst(OpConstant, 0),
			makeInst(OpGetIndex, 1),
			makeInst(OpConstant, 1),
			makeInst(OpGetIndex, 1),
			makeInst(OpJumpNil, 44),
			makeInst(OpConstant, 2),
			makeInst(OpGetIndex, 1),
			makeInst(OpConstant, 3),
			makeInst(OpGetIndex, 1),
			makeInst(OpJumpNil, 44),
			makeInst(OpConstant, 4),
			makeInst(OpGetIndex, 1),
			makeInst(OpConstant, 5),
			makeInst(OpGetIndex, 1),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		),
			funcLocals(1),
		)))

	// a?.b.c.d.e.f.g
	// true
	expectCompile(t, `var a; a?.b.c?.d.e?.f.g`, bytecode(
		Array{Str("b"), Str("c"), Str("d"), Str("e"), Str("f"), Str("g")},
		compFunc(concatInsts(
			makeInst(OpNil),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetLocal, 0),
			makeInst(OpJumpNil, 44),
			makeInst(OpConstant, 0),
			makeInst(OpGetIndex, 1),
			makeInst(OpConstant, 1),
			makeInst(OpGetIndex, 1),
			makeInst(OpJumpNil, 44),
			makeInst(OpConstant, 2),
			makeInst(OpGetIndex, 1),
			makeInst(OpConstant, 3),
			makeInst(OpGetIndex, 1),
			makeInst(OpJumpNil, 44),
			makeInst(OpConstant, 4),
			makeInst(OpGetIndex, 1),
			makeInst(OpConstant, 5),
			makeInst(OpGetIndex, 1),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		),
			funcLocals(1),
		)))

	expectCompile(t, `var a; a?.b.c`, bytecode(
		Array{Str("b"), Str("c")},
		compFunc(concatInsts(
			makeInst(OpNil),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetLocal, 0),
			makeInst(OpJumpNil, 18),
			makeInst(OpConstant, 0),
			makeInst(OpGetIndex, 1),
			makeInst(OpConstant, 1),
			makeInst(OpGetIndex, 1),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		),
			funcLocals(1),
		)))

	expectCompile(t, `var a; a?.b`, bytecode(
		Array{Str("b")},
		compFunc(concatInsts(
			makeInst(OpNil),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetLocal, 0),
			makeInst(OpJumpNil, 13),
			makeInst(OpConstant, 0),
			makeInst(OpGetIndex, 1),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		),
			funcLocals(1),
		)))

	expectCompile(t, `var a; a?.b.c`, bytecode(
		Array{Str("b"), Str("c")},
		compFunc(concatInsts(
			makeInst(OpNil),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetLocal, 0),
			makeInst(OpJumpNil, 18),
			makeInst(OpConstant, 0),
			makeInst(OpGetIndex, 1),
			makeInst(OpConstant, 1),
			makeInst(OpGetIndex, 1),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		),
			funcLocals(1),
		)))

	expectCompile(t, `var a; a?.b?.c`, bytecode(
		Array{Str("b"), Str("c")},
		compFunc(concatInsts(
			makeInst(OpNil),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetLocal, 0),
			makeInst(OpJumpNil, 21),
			makeInst(OpConstant, 0),
			makeInst(OpGetIndex, 1),
			makeInst(OpJumpNil, 21),
			makeInst(OpConstant, 1),
			makeInst(OpGetIndex, 1),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		),
			funcLocals(1),
		)))

	expectCompile(t, `@fn`, bytecode(nil,
		compFunc(concatInsts(
			makeInst(OpCallee),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		))))

	expectCompile(t, `@args`, bytecode(nil,
		compFunc(concatInsts(
			makeInst(OpArgs),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		))))

	expectCompile(t, `@nargs`, bytecode(nil,
		compFunc(concatInsts(
			makeInst(OpNamedArgs),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		))))
}

func TestCompilerStdIO(t *testing.T) {
	expectCompile(t, `STDIN`, bytecode(nil,
		compFunc(concatInsts(
			makeInst(OpStdIn),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		))))

	expectCompile(t, `STDOUT`, bytecode(nil,
		compFunc(concatInsts(
			makeInst(OpStdOut),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		))))

	expectCompile(t, `STDERR`, bytecode(nil,
		compFunc(concatInsts(
			makeInst(OpStdErr),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		))))

	expectCompile(t, `STDIN = nil`, bytecode(
		Array{Int(0)},
		compFunc(concatInsts(
			makeInst(OpGetBuiltin, int(BuiltinStdIO)),
			makeInst(OpConstant, 0),
			makeInst(OpNil),
			makeInst(OpCall, 2, 0),
			makeInst(OpReturn, 0),
		))))

	expectCompile(t, `STDIN = nil`, bytecode(
		Array{Int(0)},
		compFunc(concatInsts(
			makeInst(OpGetBuiltin, int(BuiltinStdIO)),
			makeInst(OpConstant, 0),
			makeInst(OpNil),
			makeInst(OpCall, 2, 0),
			makeInst(OpReturn, 0),
		))))

	expectCompile(t, `STDOUT = nil`, bytecode(
		Array{Int(1)},
		compFunc(concatInsts(
			makeInst(OpGetBuiltin, int(BuiltinStdIO)),
			makeInst(OpConstant, 0),
			makeInst(OpNil),
			makeInst(OpCall, 2, 0),
			makeInst(OpReturn, 0),
		))))

	expectCompile(t, `STDERR = nil`, bytecode(
		Array{Int(2)},
		compFunc(concatInsts(
			makeInst(OpGetBuiltin, int(BuiltinStdIO)),
			makeInst(OpConstant, 0),
			makeInst(OpNil),
			makeInst(OpCall, 2, 0),
			makeInst(OpReturn, 0),
		))))
}

func TestCompilerScopes(t *testing.T) {
	expectCompile(t, `
        if a := 1; a {
                a = 2
                b := a
        } else {
                a = 3
                b := a
        }`, bytecode(
		Array{Int(1), Int(2), Int(3)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpDefineLocal, 0),
			makeInst(OpGetLocal, 0),
			makeInst(OpJumpFalsy, 22),
			makeInst(OpConstant, 1),
			makeInst(OpSetLocal, 0),
			makeInst(OpGetLocal, 0),
			makeInst(OpDefineLocal, 1),
			makeInst(OpJump, 31),
			makeInst(OpConstant, 2),
			makeInst(OpSetLocal, 0),
			makeInst(OpGetLocal, 0),
			makeInst(OpDefineLocal, 1),
			makeInst(OpReturn, 0),
		),
			funcLocals(2),
		)),
	)

	expectCompile(t, `
        (func() {
                if a := 1; a {
                        a = 2
                        b := a
                } else {
                        a = 3
                        b := a
                }
        })`, bytecode(
		Array{
			Int(1),
			Int(2),
			Int(3),
			compFunc(concatInsts(
				makeInst(OpConstant, 0),
				makeInst(OpDefineLocal, 0),
				makeInst(OpGetLocal, 0),
				makeInst(OpJumpFalsy, 22),
				makeInst(OpConstant, 1),
				makeInst(OpSetLocal, 0),
				makeInst(OpGetLocal, 0),
				makeInst(OpDefineLocal, 1),
				makeInst(OpJump, 31),
				makeInst(OpConstant, 2),
				makeInst(OpSetLocal, 0),
				makeInst(OpGetLocal, 0),
				makeInst(OpDefineLocal, 1),
				makeInst(OpReturn, 0),
			),
				funcLocals(2),
			),
		},
		compFunc(concatInsts(
			makeInst(OpConstant, 3),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))

	expectCompile(t, `
        () => {
                if a := 1; a {
                        a = 2
                        b := a
                } else {
                        a = 3
                        b := a
                }
        }`, bytecode(
		Array{
			Int(1),
			Int(2),
			Int(3),
			compFunc(concatInsts(
				makeInst(OpConstant, 0),
				makeInst(OpDefineLocal, 0),
				makeInst(OpGetLocal, 0),
				makeInst(OpJumpFalsy, 22),
				makeInst(OpConstant, 1),
				makeInst(OpSetLocal, 0),
				makeInst(OpGetLocal, 0),
				makeInst(OpDefineLocal, 1),
				makeInst(OpJump, 31),
				makeInst(OpConstant, 2),
				makeInst(OpSetLocal, 0),
				makeInst(OpGetLocal, 0),
				makeInst(OpDefineLocal, 1),
				makeInst(OpReturn, 0),
			),
				funcLocals(2),
			),
		},
		compFunc(concatInsts(
			makeInst(OpConstant, 3),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))
}

func TestCompilerFuncWithMethods(t *testing.T) {
	expectCompile(t, `func f0() {
        return 100
}
met f0(i int) {
        return i
}`,
		bytecode(
			Array{
				Int(100),
				compFunc(concatInsts(
					makeInst(OpConstant, 0),
					makeInst(OpReturn, 1),
				)),
				compFunc(concatInsts(
					makeInst(OpGetLocal, 0),
					makeInst(OpReturn, 1),
				), funcLocals(1), funcParams("i int")),
			},
			compFunc(concatInsts(
				makeInst(OpGetBuiltin, int(BuiltinFunc)),
				makeInst(OpConstant, 1),
				makeInst(OpCall, 1, 0),
				makeInst(OpDefineLocal, 0),
				makeInst(OpGetLocal, 0),
				makeInst(OpConstant, 2),
				makeInst(OpAddMethod, 0, 1),
				makeInst(OpPop),
				makeInst(OpReturn, 0),
			),
				funcLocals(1)),
		))
}

func TestCompilerProp(t *testing.T) {
	// statement form: `prop name { ... }` lowers to a const bound to a
	// Prop(name, methods...) constructor call.
	expectCompile(t, `
prop x {
        () {}

        (i int) {}

        (v float) {}
}

return x
`,
		bytecode(
			Array{
				Str("x"),
				compFunc(concatInsts(
					makeInst(OpReturn, 0),
				), funcName("#1")),
				compFunc(concatInsts(
					makeInst(OpReturn, 0),
				), funcName("#2"), funcParams("i int"), funcLocals(1)),
				compFunc(concatInsts(
					makeInst(OpReturn, 0),
				), funcName("#3"), funcParams("v float"), funcLocals(1)),
			},
			compFunc(concatInsts(
				makeInst(OpGetBuiltin, int(BuiltinProp)),
				makeInst(OpConstant, 0),
				makeInst(OpConstant, 1),
				makeInst(OpConstant, 2),
				makeInst(OpConstant, 3),
				makeInst(OpCall, 4, 0),
				makeInst(OpDefineLocal, 0),
				makeInst(OpGetLocal, 0),
				makeInst(OpReturn, 1),
			), funcLocals(1)),
		),
	)

	// expression form (single accessor): `prop name(params) {body}` lowers to
	// the same constructor call, used directly as a value.
	expectCompile(t, `return prop x() {}`,
		bytecode(
			Array{
				Str("x"),
				compFunc(concatInsts(
					makeInst(OpReturn, 0),
				), funcName("#1")),
			},
			compFunc(concatInsts(
				makeInst(OpGetBuiltin, int(BuiltinProp)),
				makeInst(OpConstant, 0),
				makeInst(OpConstant, 1),
				makeInst(OpCall, 2, 0),
				makeInst(OpReturn, 1),
			), funcLocals(0)),
		),
	)
}

func TestCompilerKeyValue(t *testing.T) {
	expectCompile(t, `[a=1]`,
		bytecode(
			Array{
				Str("a"),
				Int(1),
			},
			compFunc(concatInsts(
				makeInst(OpConstant, 0),
				makeInst(OpConstant, 1),
				makeInst(OpKeyValue, 1),
				makeInst(OpPop),
				makeInst(OpReturn, 0),
			),
				funcLocals(0)),
		))
	expectCompile(t, `[a=yes]`,
		bytecode(
			Array{
				Str("a"),
			},
			compFunc(concatInsts(
				makeInst(OpConstant, 0),
				makeInst(OpYes),
				makeInst(OpKeyValue, 1),
				makeInst(OpPop),
				makeInst(OpReturn, 0),
			),
				funcLocals(0)),
		))
	expectCompile(t, `[a=no]`,
		bytecode(
			Array{
				Str("a"),
			},
			compFunc(concatInsts(
				makeInst(OpConstant, 0),
				makeInst(OpKeyValue, 0),
				makeInst(OpPop),
				makeInst(OpReturn, 0),
			),
				funcLocals(0)),
		))
	expectCompile(t, `(;a=1,b=yes,c,d=no)`,
		bytecode(
			Array{
				Str("a"),
				Int(1),
				Str("b"),
				Str("c"),
			},
			compFunc(concatInsts(
				makeInst(OpConstant, 0),
				makeInst(OpConstant, 1),
				makeInst(OpKeyValue, 1),
				makeInst(OpConstant, 2),
				makeInst(OpYes),
				makeInst(OpKeyValue, 1),
				makeInst(OpConstant, 3),
				makeInst(OpYes),
				makeInst(OpKeyValue, 1),
				makeInst(OpKeyValueArray, 3),
				makeInst(OpPop),
				makeInst(OpReturn, 0),
			),
				funcLocals(0)),
		))
}

func TestCompilerMultiparen(t *testing.T) {
	expectCompile(t, `(1,*[2,3];a=4,**{})`, bytecode(
		Array{
			Int(1),
			Int(2),
			Int(3),
			Str("a"),
			Int(4),
		},
		compFunc(concatInsts(
			makeInst(OpGetBuiltin, int(BuiltinMixedParams)),
			makeInst(OpConstant, 0),
			makeInst(OpConstant, 1),
			makeInst(OpConstant, 2),
			makeInst(OpArray, 2),
			makeInst(OpConstant, 3),
			makeInst(OpConstant, 4),
			makeInst(OpKeyValue, 1),
			makeInst(OpDict, 0),
			makeInst(OpNamedParamsVar),
			makeInst(OpKeyValueArray, 2),
			makeInst(OpCall, 2, 3),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))
}

func TestCompiler_CompileReturnAssign(t *testing.T) {
	expectCompile(t, `x := 1; return = x; x = 2`, bytecode(
		Array{Int(1), Int(2)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpDefineLocal, 0),
			makeInst(OpSetReturn, 0),
			makeInst(OpConstant, 1),
			makeInst(OpSetLocal, 0),
			makeInst(OpReturn, 0),
		),
			funcLocals(1)),
	))

	expectCompile(t, `x := 1; return = x; y := 2; return = y`, bytecode(
		Array{Int(1), Int(2)},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpDefineLocal, 0),
			makeInst(OpSetReturn, 0),
			makeInst(OpConstant, 1),
			makeInst(OpDefineLocal, 1),
			makeInst(OpSetReturn, 1),
			makeInst(OpReturn, 0),
		),
			funcLocals(2)),
	))

	expectCompileError(t, `return = 1`, "Parse Error: expected *Ident, found *node.IntLit\n\tat (main):1:10")
}

func TestCompiler_CompileSymbol(t *testing.T) {
	expectCompile(t, `#a;#(A
bc \)
x	z
)`, bytecode(
		Array{Str("a"), Str("A\nbc )\nx\tz\n")},
		compFunc(concatInsts(
			makeInst(OpConstant, 0),
			makeInst(OpPop),
			makeInst(OpConstant, 1),
			makeInst(OpPop),
			makeInst(OpReturn, 0),
		)),
	))
}

func TestCompilerFuncReturnType(t *testing.T) {
	// Return-type annotations are compiled onto CompiledFunction.ReturnVars and
	// rendered by HeaderString, without affecting the generated instructions.
	compileFn := func(t *testing.T, script string) *CompiledFunction {
		t.Helper()
		_, bc, err := Compile(NewSymbolTable(NewBuiltins().NameSet), []byte(script), CompileOptions{})
		require.NoError(t, err)
		for _, c := range bc.Constants {
			if cf, ok := c.(*CompiledFunction); ok {
				return cf
			}
		}
		t.Fatalf("no compiled function constant in: %s", script)
		return nil
	}

	type ret struct {
		name  string
		types string
	}

	for _, c := range []struct {
		name   string
		script string
		want   []ret
		header string
	}{
		{
			"anonymous single",
			`return func(a) <int> { return a }`,
			[]ret{{"", "int"}},
			" <int>",
		},
		{
			"anonymous multiple",
			`return func(a, b) <int, str> { return [a, b] }`,
			[]ret{{"", "int"}, {"", "str"}},
			" <int, str>",
		},
		{
			"named union",
			`return func(a int) <x int|bool> => a`,
			[]ret{{"x", "int|bool"}},
			" <x int|bool>",
		},
		{
			"shorthand name",
			`x := foo(a) <int> { return a }; return x`,
			[]ret{{"", "int"}},
			" <int>",
		},
		{
			"shorthand dict element",
			`return {g(a, b) <int, str> { return a }}`,
			[]ret{{"", "int"}, {"", "str"}},
			" <int, str>",
		},
		{
			"closure lambda",
			`x := (a) <int> => a; return x`,
			[]ret{{"", "int"}},
			" <int>",
		},
		{
			"closure dict element",
			`return {f(a) <x int|bool> : a}`,
			[]ret{{"x", "int|bool"}},
			" <x int|bool>",
		},
		{
			"no return type",
			`return func(a) { return a }`,
			nil,
			"",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := compileFn(t, c.script)
			require.Len(t, f.ReturnVars, len(c.want))
			for i, w := range c.want {
				require.Equal(t, w.name, f.ReturnVars[i].Name)
				require.Equal(t, w.types, f.ReturnVars[i].TypesSymbols.String())
			}
			// The rendered suffix is appended to the function header.
			require.Equal(t, c.header, FormatReturnVars(f.ReturnVars))
			require.True(t, strings.HasSuffix(f.HeaderString(), c.header))
		})
	}

	// Unresolved return types are reported, mirroring parameter type resolution.
	_, _, err := Compile(NewSymbolTable(NewBuiltins().NameSet),
		[]byte(`return func(a) <NopeType> { return a }`), CompileOptions{})
	require.Error(t, err)
	require.Contains(t, err.Error(), `unresolved reference "NopeType"`)
}

func expectCompileError(t *testing.T, script string, errStr string) {
	t.Helper()
	expectCompileErrorWithOpts(t, script, CompileOptions{}, errStr)
}

func expectCompileErrorWithOpts(t *testing.T,
	script string, opts CompileOptions, errStr string) {

	t.Helper()
	_, _, err := Compile(NewSymbolTable(NewBuiltins().NameSet), []byte(script), opts)
	require.Error(t, err)
	require.Contains(t, err.Error(), errStr)
}

func expectCompile(t *testing.T, script string, expected *Bytecode) {
	t.Helper()
	expectCompileWithOpts(t, script, CompileOptions{}, expected)
}

func expectCompileMixed(t *testing.T, script string, expected *Bytecode) {
	t.Helper()
	expectCompileWithOpts(t, script, CompileOptions{
		ParserOptions: parser.ParserOptions{
			Mode: parser.ParseMixed,
		},
	}, expected)
}

type expectCompileOptions struct {
	builtins *Builtins
	st       *SymbolTable
	opts     *TestBytecodesEqualOptions
}

// SourceMap comparison is ignored if it is nil.
func expectCompileWithOpts(t *testing.T,
	script string, opts CompileOptions, expected *Bytecode, opt ...*expectCompileOptions) {

	var eopts *expectCompileOptions
	for _, eopts = range opt {
	}

	if eopts == nil {
		eopts = &expectCompileOptions{}
	}

	if eopts.builtins == nil {
		eopts.builtins = NewBuiltins()
	}

	if eopts.st == nil {
		eopts.st = NewSymbolTable(eopts.builtins.NameSet)
	}

	t.Helper()
	_, got, err := Compile(eopts.st, []byte(script), opts)
	require.NoError(t, err)
	TestBytecodesEqual(t, expected, got, expected.Main.SourceMap != nil, eopts.opts)
}
