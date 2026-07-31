package main

import (
	"strings"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/giom"
)

// isGiomFile reports whether path names a Giom template (.giom), which is
// compiled with gad's native Giom front-end (gad.CompileOptions.GiomOptions)
// instead of the plain Gad compiler.
func isGiomFile(path string) bool { return strings.HasSuffix(path, ".giom") }

// giomBuiltins returns a builtins set with the giom namespace (giom.attr,
// giom.attrs, giom.escape, giom.write) always registered so the tag-rendering
// code emitted for a Giom entrypoint — or for any `.giom` file imported by a
// plain Gad script — resolves at both compile and run time. The path argument is
// retained for call-site clarity.
func giomBuiltins(path string) *gad.Builtins {
	return giom.AppendBuiltins(gad.NewBuiltins())
}
