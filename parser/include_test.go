package parser_test

import (
	"testing"

	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	"github.com/stretchr/testify/require"
)

// parseInclude parses src and returns its single IncludeStmt.
func parseInclude(t *testing.T, src string) *node.IncludeStmt {
	t.Helper()
	f := source.NewFileSet().AddFileData("t.gad", -1, []byte(src))
	file, err := parser.NewParserWithOptions(f, nil, nil).ParseFile()
	require.NoError(t, err)
	for _, s := range file.Stmts {
		if inc, ok := s.(*node.IncludeStmt); ok {
			return inc
		}
	}
	t.Fatalf("no IncludeStmt in: %s", src)
	return nil
}

// TestParseIncludeStmt covers the parenthesized forms of the `include`
// statement (the parentheses are required, since it is written like a call).
func TestParseIncludeStmt(t *testing.T) {
	t.Run("single", func(t *testing.T) {
		inc := parseInclude(t, `include ("a.gad")`)
		require.True(t, inc.LParen.IsValid())
		require.True(t, inc.RParen.IsValid())
		require.Len(t, inc.Paths, 1)
		require.Equal(t, "a.gad", inc.Paths[0].Value())
	})

	t.Run("multi", func(t *testing.T) {
		inc := parseInclude(t, `include ("a.gad", "b.gad", "c.gad")`)
		require.Len(t, inc.Paths, 3)
		require.Equal(t, "a.gad", inc.Paths[0].Value())
		require.Equal(t, "b.gad", inc.Paths[1].Value())
		require.Equal(t, "c.gad", inc.Paths[2].Value())
	})

	t.Run("round-trips via String", func(t *testing.T) {
		inc := parseInclude(t, `include ("a.gad", "b.gad")`)
		require.Equal(t, `include ("a.gad", "b.gad")`, inc.String())
	})
}

// TestParseIncludeRequiresParens verifies the bare (parenthesis-less) form is a
// parse error: `include` is only a statement when written like a call.
func TestParseIncludeRequiresParens(t *testing.T) {
	f := source.NewFileSet().AddFileData("t.gad", -1, []byte(`include "a.gad"`))
	_, err := parser.NewParserWithOptions(f, nil, nil).ParseFile()
	require.Error(t, err)
}

// TestParseIncludeRejectsNonString verifies a non-string include path is a parse
// error.
func TestParseIncludeRejectsNonString(t *testing.T) {
	f := source.NewFileSet().AddFileData("t.gad", -1, []byte(`include (123)`))
	_, err := parser.NewParserWithOptions(f, nil, nil).ParseFile()
	require.Error(t, err)
}

// TestParseSourceKeywords verifies the `@mod` and `@files` operand keywords parse
// as their literal nodes.
func TestParseSourceKeywords(t *testing.T) {
	f := source.NewFileSet().AddFileData("t.gad", -1, []byte("a := @mod\nb := @files"))
	file, err := parser.NewParserWithOptions(f, nil, nil).ParseFile()
	require.NoError(t, err)

	require.Len(t, file.Stmts, 2)
	a := file.Stmts[0].(*node.AssignStmt)
	require.IsType(t, &node.ModLit{}, a.RHS[0])
	b := file.Stmts[1].(*node.AssignStmt)
	require.IsType(t, &node.FilesLit{}, b.RHS[0])
}
