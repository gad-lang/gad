package ide

import (
	"strings"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/giom"
)

// isGiom reports whether path names a Giom template (.giom). Giom files are
// compiled with gad's native Giom front-end (gad.CompileOptions.GiomOptions)
// rather than the plain Gad compiler.
func isGiom(path string) bool { return strings.HasSuffix(path, ".giom") }

// newBuiltins returns a builtins set suitable for compiling and running path.
// The giom namespace (giom.attr, giom.attrs, giom.escape, giom.write) is always
// registered so the tag-rendering code emitted for a Giom entrypoint — or for
// any `.giom` file imported by a plain Gad script — resolves both at compile
// time (symbol table) and at run time (VM). The same set seeds the symbol table
// and builds the VM. The path argument is retained for call-site clarity.
func newBuiltins(path string) *gad.Builtins {
	return giom.AppendBuiltins(gad.NewBuiltins())
}

// compileFor compiles src for path: Giom source (.giom) through gad's native
// Giom front-end, everything else (plain Gad and .gadt templates) through the
// plain Gad path. The caller is responsible for template (.gadt) mode on opts.
func compileFor(st *gad.SymbolTable, src []byte, path string, opts gad.CompileOptions) (*gad.Bytecode, error) {
	if isGiom(path) {
		opts.GiomOptions = &gad.GiomOptions{}
	}
	_, bc, err := gad.Compile(st, src, opts)
	return bc, err
}
