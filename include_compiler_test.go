package gad

import (
	"testing"

	"github.com/gad-lang/gad/parser/source"
	"github.com/stretchr/testify/require"
)

// compileMainInclude compiles mainSrc with mods registered as plain Gad source
// modules and returns the bytecode. It runs in the internal package so tests can
// inspect instructions, constants and the source map directly.
func compileMainInclude(t *testing.T, mainSrc string, mods map[string]string) *Bytecode {
	t.Helper()
	mm := NewModuleMap()
	for name, src := range mods {
		mm.AddSourceModule(name, []byte(src))
	}
	opts := CompileOptions{}
	opts.ModuleMap = mm
	opts.ModuleFile = "main.gad"
	cr, err := Compile(NewSymbolTable(NewBuiltins().NameSet), []byte(mainSrc), opts)
	require.NoError(t, err)
	return cr.BC()
}

// TestCompileIncludeWraps verifies `include` lowers to the inlined statements
// wrapped in OpPushSource/OpPopSource, and that the pushed source name is the
// included file's URL constant.
func TestCompileIncludeWraps(t *testing.T) {
	bc := compileMainInclude(t, `include ("name.gad"); return x`,
		map[string]string{"name.gad": "x := 41 + 1"})

	var (
		pushes, pops int
		pushedNames  []string
	)
	IterateInstructions(bc.Main.Instructions, func(_ int, op Opcode, operands []int, _ int) bool {
		switch op {
		case OpPushSource:
			pushes++
			pushedNames = append(pushedNames, bc.Constants[operands[0]].ToString())
		case OpPopSource:
			pops++
		}
		return true
	})

	require.Equal(t, 1, pushes, "one OpPushSource per include")
	require.Equal(t, 1, pops, "one OpPopSource per include")
	require.Equal(t, []string{"name.gad"}, pushedNames)
}

// TestCompileIncludeSourcePositions verifies the inlined statements keep source
// positions pointing at the INCLUDED file (not the including module): every
// source-map entry that lands in the included file resolves to its name, and the
// included statement's line is preserved.
func TestCompileIncludeSourcePositions(t *testing.T) {
	// The include is a bare list, so the included statement `y := x + 1` sits on
	// line 3 of add.gad.
	bc := compileMainInclude(t, `x := 1; include ("add.gad"); return y`,
		map[string]string{"add.gad": "// a comment\n\ny := x + 1"})

	fset := bc.FileSet

	var (
		sawIncluded bool
		includedLns []int
	)
	for _, pos := range bc.Main.SourceMap {
		fp := fset.Position(source.Pos(pos))
		if fp.File != nil && fp.File.Name == "add.gad" {
			sawIncluded = true
			includedLns = append(includedLns, fp.Line)
		}
	}

	require.True(t, sawIncluded, "some instruction must map to the included file add.gad")
	require.Contains(t, includedLns, 3, "the included statement is on line 3 of add.gad")
}

// TestCompileIncludeErrorPosition verifies a compile error inside an included
// file is reported against the included file and its own line/column, not the
// including module.
func TestCompileIncludeErrorPosition(t *testing.T) {
	mm := NewModuleMap()
	// unresolved reference on line 3, column 8 of bad.gad
	mm.AddSourceModule("bad.gad", []byte("a := 1\nb := 2\nreturn undefinedThing"))
	opts := CompileOptions{}
	opts.ModuleMap = mm
	opts.ModuleFile = "main.gad"

	_, err := Compile(NewSymbolTable(NewBuiltins().NameSet), []byte(`include ("bad.gad")`), opts)
	require.Error(t, err)

	ce, ok := err.(*CompilerError)
	require.True(t, ok, "want *CompilerError, got %T", err)
	fp := ce.FileSet.Position(ce.Node.Pos())
	require.Equal(t, "bad.gad", fp.File.Name)
	require.Equal(t, 3, fp.Line)
}
