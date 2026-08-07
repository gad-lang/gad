package gadx

import (
	"path/filepath"

	gad "github.com/gad-lang/gad"
	gadxnode "github.com/gad-lang/gad/gadx/node"
	"github.com/gad-lang/gad/parser/ast"
)

// This file provides thin test-only shims mirroring the former isolated gadx
// compiler API (Compile / NewCompiler / CompileFallback), now backed by gad's
// native Gadx front-end (selected by the .gadx ModuleFile extension). The real
// compilation logic lives in the gad module; these exist only so the existing
// gadx test suite keeps exercising Gadx compilation through a familiar entry
// point.

// CompileFallback is a retained sentinel: assigning it to
// CompilerOptions.FallbackFunc is a no-op because Compile forces gad's native
// Gadx front-end via the .gadx ModuleFile.
var CompileFallback func(*gad.Compiler, ast.Node) error

// Compile compiles Gadx source to Gad bytecode through gad's native Gadx
// front-end, preserving any ModuleMap / EmbededdMap on opts.
func Compile(st *gad.SymbolTable, src []byte, opts gad.CompileOptions) (*gadxnode.File, *gad.Bytecode, error) {
	if filepath.Ext(opts.ModuleFile) != ".gadx" {
		opts.ModuleFile = "module.gadx"
	}
	opts.FallbackFunc = nil
	res, err := gad.Compile(st, src, opts)
	return nil, res.BC(), err
}

// compilerShim mirrors the former gadx.Compiler: it binds a symbol table and
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
func (c *compilerShim) Compile(input []byte) (*gadxnode.File, *gad.Bytecode, error) {
	return Compile(c.st, input, c.opts)
}
