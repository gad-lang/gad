package main

import (
	"strings"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/gadx"
)

// isGadxFile reports whether path names a Gadx template (.gadx), which is
// compiled with gad's native Gadx front-end (gad.CompileOptions.GadxOptions)
// instead of the plain Gad compiler.
func isGadxFile(path string) bool { return strings.HasSuffix(path, ".gadx") }

// gadxBuiltins returns a builtins set with the gadx namespace (gadx.attr,
// gadx.attrs, gadx.escape, gadx.write) always registered so the tag-rendering
// code emitted for a Gadx entrypoint — or for any `.gadx` file imported by a
// plain Gad script — resolves at both compile and run time. The path argument is
// retained for call-site clarity.
func gadxBuiltins(path string) *gad.Builtins {
	return gadx.AppendBuiltins(gad.NewBuiltins())
}
