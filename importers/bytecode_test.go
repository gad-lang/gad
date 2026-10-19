package importers_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/gadx"
	"github.com/gad-lang/gad/importers"
	"github.com/stretchr/testify/require"
)

// WriteBytecode: a module imported from disk is written beside it as .gadb,
// which a later import without the extension links (while not older than the
// source); a module that imports others is compiled from its source only.
func TestFileImporterWriteBytecode(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) string {
		p := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
		return p
	}
	m := write("m.gad", "k := 2\nexport double(x) => x * k\nexport enum Mode { r; w }\n")
	write("uses.gad", "export {x: import(\"./m\").double(1)}\n")
	main := []byte(`m := import("./m"); return [m.double(3), m.Mode.w.name]`)

	runWith := func(writeBC bool, src []byte) gad.Object {
		mm := gad.NewModuleMap().SetExtImporter(&importers.FileImporter{WorkDir: dir, WriteBytecode: writeBC})
		ret, err := run(&bytes.Buffer{}, src, gad.CompilerOptions{ModuleMap: mm})
		require.NoError(t, err)
		return ret
	}
	want := gad.Array{gad.Int(6), gad.Str("w")}

	// off: no .gadb written
	require.Equal(t, want, runWith(false, main))
	_, err := os.Stat(filepath.Join(dir, "m.gadb"))
	require.True(t, os.IsNotExist(err))

	// on: written, and the same result
	require.Equal(t, want, runWith(true, main))
	st, err := os.Stat(filepath.Join(dir, "m.gadb"))
	require.NoError(t, err)

	// found first without the extension: the source gone, the .gadb is linked
	require.NoError(t, os.Rename(m, m+".bak"))
	require.Equal(t, want, runWith(false, main))
	require.NoError(t, os.Rename(m+".bak", m))

	// a source newer than its .gadb: the source, not the old bytecode
	write("m.gad", "export double(x) => x * 10\nexport enum Mode { r; w }\n")
	require.NoError(t, os.Chtimes(m, time.Now(), st.ModTime().Add(time.Hour)))
	require.Equal(t, gad.Array{gad.Int(30), gad.Str("w")}, runWith(false, main))

	// a module that imports others: written too; the program imports what it
	// does, by their names
	require.Equal(t, gad.Int(10), runWith(true, []byte(`return import("./uses").x`)))
	_, err = os.Stat(filepath.Join(dir, "uses.gadb"))
	require.NoError(t, err)
	require.Equal(t, gad.Int(10), runWith(false, []byte(`return import("./uses").x`)))
}

// A .gadx module (a template of components) is written as .gadb too.
func TestFileImporterWriteBytecodeGadx(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "comps.gadx"), []byte("@export comp card(title)\n    div.card {title}\n"), 0o644))
	mm := gad.NewModuleMap().AddBuiltinModule("gadx", gadx.Module())
	builtins := gadx.AppendBuiltins(gad.NewBuiltins())
	mm.SetExtImporter(&importers.FileImporter{WorkDir: dir, WriteBytecode: true, SourceKind: gad.SourceKindGadx, ModuleMap: mm, Builtins: builtins.NameSet})
	runGadx := func() gad.Object {
		res, err := gad.Compile(gad.NewSymbolTable(builtins.NameSet), []byte(`c := import("./comps"); return typeName(c.card)`),
			gad.CompileOptions{CompilerOptions: gad.CompilerOptions{ModuleMap: mm}})
		require.NoError(t, err)
		ret, err := gad.NewVM(builtins.Build(), res.Bytecode).Run()
		require.NoError(t, err)
		return ret
	}
	first := runGadx()
	_, err := os.Stat(filepath.Join(dir, "comps.gadb"))
	require.NoError(t, err)
	// linked from the .gadb, the source gone
	require.NoError(t, os.Remove(filepath.Join(dir, "comps.gadx")))
	require.Equal(t, first, runGadx())

	// a program of other builtins refuses it
	res, err := gad.Compile(gad.NewSymbolTable(gad.NewBuiltins().NameSet), []byte(`return import("./comps")`),
		gad.CompileOptions{CompilerOptions: gad.CompilerOptions{ModuleMap: gad.NewModuleMap().SetExtImporter(&importers.FileImporter{WorkDir: dir})}})
	_ = res
	require.ErrorContains(t, err, "compiled with other builtins")
}
