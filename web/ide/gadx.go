package ide

import (
	"strings"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/gadx"
)

// isGadx reports whether path names a Gadx template (.gadx). Gadx files are
// compiled with gad's native Gadx front-end (gad.CompileOptions.GadxOptions)
// rather than the plain Gad compiler.
func isGadx(path string) bool { return strings.HasSuffix(path, ".gadx") }

// newBuiltins returns a builtins set suitable for compiling and running path.
// The gadx namespace (gadx.attr, gadx.attrs, gadx.escape, gadx.write) is always
// registered so the tag-rendering code emitted for a Gadx entrypoint — or for
// any `.gadx` file imported by a plain Gad script — resolves both at compile
// time (symbol table) and at run time (VM). The same set seeds the symbol table
// and builds the VM. The path argument is retained for call-site clarity.
func newBuiltins(path string) *gad.Builtins {
	return gadx.AppendBuiltins(gad.NewBuiltins())
}

// compileFor compiles src for path: Gadx source (.gadx) through gad's native
// Gadx front-end, everything else (plain Gad and .gadt templates) through the
// plain Gad path. The caller is responsible for template (.gadt) mode on opts.
func compileFor(st *gad.SymbolTable, src []byte, path string, opts gad.CompileOptions) (*gad.CompileResult, error) {
	if isGadx(path) {
		opts.GadxOptions = &gad.GadxOptions{}
	}
	return gad.Compile(st, src, opts)
}

// warningsText renders compiler warnings as STDERR panel text (one per line,
// with source position + detail), or "" when there are none.
func warningsText(warnings []*gad.CompilerWarning) string {
	if len(warnings) == 0 {
		return ""
	}
	var b strings.Builder
	for _, w := range warnings {
		b.WriteString(w.Error())
		b.WriteByte('\n')
	}
	return b.String()
}
