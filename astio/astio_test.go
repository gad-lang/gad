package astio_test

import (
	"testing"

	"github.com/gad-lang/gad/astio"
	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	"github.com/stretchr/testify/require"
)

func parseFirstStmt(t *testing.T, src string) node.Stmt {
	t.Helper()
	fs := source.NewFileSet()
	f := fs.AddFileData("t.gad", -1, []byte(src))
	pf, err := parser.NewParser(f, nil).ParseFile()
	require.NoError(t, err, src)
	require.NotEmpty(t, pf.Stmts, src)
	return pf.Stmts[0]
}

func render(n ast.Node) string {
	if c, ok := n.(node.Coder); ok {
		return node.Code(c)
	}
	return ""
}

// TestRoundTrip parses a range of constructs, exports them to JSON and YAML,
// imports them back, and requires that the re-rendered source is identical —
// proving the AST survives the round-trip.
func TestRoundTrip(t *testing.T) {
	for _, src := range []string{
		"a := [1, 2, 3]",
		"x := a + b*2 - c",
		`d := {"k": v, e: f}`,
		"g := func(x, y) { return x + y }",
		"if ok { doThis() } else { doThat() }",
		"for i := 0; i < n; i++ { work(i) }",
		"h := #\"a { x + 1 } b\"",
	} {
		stmt := parseFirstStmt(t, src)
		want := render(stmt)

		jb, err := astio.MarshalJSON(stmt)
		require.NoError(t, err, src)
		jback, err := astio.UnmarshalJSON(jb)
		require.NoError(t, err, src)
		require.Equal(t, want, render(jback), "JSON round-trip: %s", src)

		yb, err := astio.MarshalYAML(stmt)
		require.NoError(t, err, src)
		yback, err := astio.UnmarshalYAML(yb)
		require.NoError(t, err, src)
		require.Equal(t, want, render(yback), "YAML round-trip: %s", src)
	}
}

// customStmt is a user-defined node the registry does not know about.
type customStmt struct {
	ast.NodeData
	Label string
}

func (customStmt) StmtNode()                        {}
func (customStmt) Pos() source.Pos                  { return 0 }
func (customStmt) End() source.Pos                  { return 0 }
func (customStmt) String() string                   { return "custom" }
func (customStmt) WriteCode(*node.CodeWriteContext) {}

// TestFallback checks an unregistered node type exports and re-imports via the
// RawNode fallback (data preserved), and works once registered.
func TestFallback(t *testing.T) {
	orig := &customStmt{Label: "hi"}
	jb, err := astio.MarshalJSON(orig)
	require.NoError(t, err)

	back, err := astio.UnmarshalJSON(jb)
	require.NoError(t, err)
	raw, ok := back.(*astio.RawNode)
	require.True(t, ok, "unregistered type should fall back to RawNode")
	require.Equal(t, "hi", raw.Tree["Label"])

	// Registered, it reconstructs to the concrete type.
	astio.Register((*customStmt)(nil))
	back2, err := astio.UnmarshalJSON(jb)
	require.NoError(t, err)
	cs, ok := back2.(*customStmt)
	require.True(t, ok, "registered type should reconstruct")
	require.Equal(t, "hi", cs.Label)
}
