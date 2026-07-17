package main

import (
	"strings"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/giom"
	"github.com/gad-lang/gad/importers"
)

// isGiomFile reports whether path names a Giom template (.giom), which is
// compiled with the Giom front-end (giom.Compile) instead of the Gad compiler.
func isGiomFile(path string) bool { return strings.HasSuffix(path, ".giom") }

// giomBuiltins returns a builtins set with the giom namespace (giom.attr,
// giom.attrs, giom.escape, giom.write) registered when path is a Giom template,
// so the generated tag-rendering code resolves at compile and run time. Non-Giom
// paths get the plain Gad builtins.
func giomBuiltins(path string) *gad.Builtins {
	b := gad.NewBuiltins()
	if isGiomFile(path) {
		giom.AppendBuiltins(b)
	}
	return b
}

// giomModuleImporter swaps mm's external importer for a giom-aware one so that
// .giom files imported from a .giom entrypoint compile as templates (plain .gad
// sources still pass through unchanged).
func giomModuleImporter(mm *gad.ModuleMap, workdir string, sourcePath *importers.PathList) {
	mm.SetExtImporter(&giom.FileImporter{
		WorkDir:      workdir,
		FileReader:   importers.ShebangReadFile,
		NameResolver: importers.OsDirsNameResolverPtr(sourcePath),
	})
}
