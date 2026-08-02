package giom

import (
	gad "github.com/gad-lang/gad"
	giomnode "github.com/gad-lang/gad/giom/node"
	"github.com/gad-lang/gad/parser/ast"
)

// This file provides thin test-only shims mirroring the former isolated giom
// compiler API (Compile / NewCompiler / CompileFallback), now backed by gad's
// native Giom front-end (gad.CompileOptions.GiomOptions). The real compilation
// logic lives in the gad module; these exist only so the existing giom test
// suite keeps exercising Giom compilation through a familiar entry point.

// CompileFallback is a retained sentinel: assigning it to
// CompilerOptions.FallbackFunc is a no-op because Compile forces gad's native
// Giom fallback via GiomOptions.
var CompileFallback func(*gad.Compiler, ast.Node) error

// Compile compiles Giom source to Gad bytecode through gad's native Giom
// front-end, preserving any ModuleMap / EmbededdMap on opts.
func Compile(st *gad.SymbolTable, src []byte, opts gad.CompileOptions) (*giomnode.File, *gad.Bytecode, error) {
	opts.GiomOptions = &gad.GiomOptions{}
	opts.FallbackFunc = nil
	res, err := gad.Compile(st, src, opts)
	return nil, res.BC(), err
}

// compilerShim mirrors the former giom.Compiler: it binds a symbol table and
// options and compiles multiple inputs with them.
type compilerShim struct {
	st   *gad.SymbolTable
	opts gad.CompileOptions
}

// NewCompiler returns a compiler bound to st and opts (test shim).
func NewCompiler(st *gad.SymbolTable, opts gad.CompileOptions) *compilerShim {
	return &compilerShim{st: st, opts: opts}
}

// Compile compiles input with the bound symbol table and options.
func (c *compilerShim) Compile(input []byte) (*giomnode.File, *gad.Bytecode, error) {
	return Compile(c.st, input, c.opts)
}
