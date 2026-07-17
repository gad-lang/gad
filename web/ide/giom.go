package ide

import (
	"strings"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/giom"
	"github.com/gad-lang/gad/importers"
)

// isGiom reports whether path names a Giom template (.giom). Giom files are
// compiled with the Giom front-end (giom.Compile) rather than the Gad compiler.
func isGiom(path string) bool { return strings.HasSuffix(path, ".giom") }

// newBuiltins returns a builtins set suitable for compiling and running path.
// For .giom files the giom namespace (giom.attr, giom.attrs, giom.escape,
// giom.write) is registered so the generated tag-rendering code resolves both at
// compile time (symbol table) and at run time (VM). The same set must seed the
// symbol table and build the VM.
func newBuiltins(path string) *gad.Builtins {
	b := gad.NewBuiltins()
	if isGiom(path) {
		giom.AppendBuiltins(b)
	}
	return b
}

// compileFor compiles src for path: Giom source (.giom) through giom.Compile,
// everything else (plain Gad and .gadt templates) through gad.Compile with opts
// left untouched. The caller is responsible for template (.gadt) mode on opts.
func compileFor(st *gad.SymbolTable, src []byte, path string, opts gad.CompileOptions) (*gad.Bytecode, error) {
	if isGiom(path) {
		_, bc, err := giom.Compile(st, src, opts)
		return bc, err
	}
	_, bc, err := gad.Compile(st, src, opts)
	return bc, err
}

// useGiomImporter swaps the module map's external importer for a giom-aware one
// so that .giom files imported from a .giom entrypoint are compiled as templates.
// giom.FileImporter also serves plain .gad sources unchanged, so it is a safe
// superset of the default importer for a Giom run.
func useGiomImporter(mm *gad.ModuleMap, workdir string) {
	if mm == nil {
		return
	}
	mm.SetExtImporter(&giom.FileImporter{
		WorkDir:    workdir,
		FileReader: importers.ShebangReadFile,
	})
}
