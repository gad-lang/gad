package parser

import (
	"testing"

	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	giomnode "github.com/gad-lang/gad/giom/node"
)

// parseFileWith parses src into a fresh FileSet/File and returns both so tests
// can resolve node positions to file:line:column.
func parseFileWith(t *testing.T, src string) (*source.FileSet, *source.File, *giomnode.File) {
	t.Helper()
	fs := source.NewFileSet()
	f := fs.AddFileData("test.giom", -1, []byte(src))
	p := NewParser(f)
	file, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse error: %v\nsrc: %q", err, src)
	}
	return fs, f, file
}

// TestScannerPopulatesLineTable verifies the scanner registers a line offset
// for every source line, so positions resolve to real lines instead of always
// line 1 (the pre-fix behavior left File.Lines == [0]).
func TestScannerPopulatesLineTable(t *testing.T) {
	src := "@main\n" + // line 1
		"    ~ a := 1\n" + // line 2
		"    ~ b := 2\n" + // line 3
		"    p done\n" // line 4
	_, f, _ := parseFileWith(t, src)

	if f.LineCount() < 4 {
		t.Fatalf("expected line table with >= 4 lines, got %d (%v)", f.LineCount(), f.Lines)
	}
}

// posLineCol resolves a node position to its 1-based source line and column.
func posLineCol(fs *source.FileSet, pos source.Pos) (line, col int) {
	p := fs.Position(pos)
	return p.Line, p.Column
}

// findFirstCode returns the first CodeStmt found by depth-first walk over the
// file's top-level statements and comp bodies.
func findFirstCode(file *giomnode.File) *giomnode.CodeStmt {
	var walk func(stmts node.Stmts) *giomnode.CodeStmt
	walk = func(stmts node.Stmts) *giomnode.CodeStmt {
		for _, s := range stmts {
			switch n := s.(type) {
			case *giomnode.CodeStmt:
				if len(n.Stmts) > 0 {
					return n
				}
			case *giomnode.CompDecl:
				if c := walk(n.Body); c != nil {
					return c
				}
			case *giomnode.TagStmt:
				if c := walk(n.Body); c != nil {
					return c
				}
			}
		}
		return nil
	}
	if c := walk(file.Stmts); c != nil {
		return c
	}
	for _, comp := range file.Comps {
		if c := walk(comp.Body); c != nil {
			return c
		}
	}
	return nil
}

// TestCodeStmtPositionMapsToSourceLine verifies that the inner GAD statement of
// a single-line code directive carries a position that resolves to the correct
// giom source line, rather than a fragment-local line 1.
func TestCodeStmtPositionMapsToSourceLine(t *testing.T) {
	src := "@main\n" + // line 1
		"    p intro\n" + // line 2
		"    ~ value := compute()\n" // line 3

	fs, _, file := parseFileWith(t, src)
	code := findFirstCode(file)
	if code == nil {
		t.Fatal("no code statement found")
	}
	// line 3, col 7: `value` in `    ~ value := compute()`
	if line, col := posLineCol(fs, code.Stmts[0].Pos()); line != 3 || col != 7 {
		t.Fatalf("code statement resolved to %d:%d, want 3:7", line, col)
	}
}

// TestMultiLineCodePositionMapsToSourceLine verifies each statement in a ~~
// block resolves to its own source line.
func TestMultiLineCodePositionMapsToSourceLine(t *testing.T) {
	src := "@main\n" + // line 1
		"    ~~\n" + // line 2
		"    first()\n" + // line 3
		"    second()\n" + // line 4
		"    ~~\n" // line 5

	fs, _, file := parseFileWith(t, src)
	code := findFirstCode(file)
	if code == nil || len(code.Stmts) < 2 {
		t.Fatalf("expected >= 2 statements in ~~ block, got %v", code)
	}
	if line, col := posLineCol(fs, code.Stmts[0].Pos()); line != 3 || col != 5 {
		t.Fatalf("first statement resolved to %d:%d, want 3:5", line, col)
	}
	if line, col := posLineCol(fs, code.Stmts[1].Pos()); line != 4 || col != 5 {
		t.Fatalf("second statement resolved to %d:%d, want 4:5", line, col)
	}
}
