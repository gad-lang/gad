package parser_test

import (
	"testing"

	. "github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	"github.com/stretchr/testify/require"

	. "github.com/gad-lang/gad/parser"
)

// parseDoc parses src with comment+doc attachment enabled and returns the File.
func parseDoc(t *testing.T, src string) *File {
	t.Helper()
	fs := source.NewFileSet()
	f := fs.AddFileData("doc", -1, []byte(src))
	file, err := NewParserWithOptions(f,
		&ParserOptions{Mode: ParseComments}, nil).ParseFile()
	require.NoError(t, err)
	return file
}

func TestDocCommentAttachGenDecl(t *testing.T) {
	file := parseDoc(t, "/// this is the server addr\nconst ServerAddr = \":0\"\n")
	require.Len(t, file.Stmts, 1)
	ds, ok := file.Stmts[0].(*DeclStmt)
	require.True(t, ok, "want *DeclStmt, got %T", file.Stmts[0])
	gd, ok := ds.Decl.(*GenDecl)
	require.True(t, ok)
	require.NotNil(t, gd.Doc, "GenDecl.Doc should be attached")
	require.Equal(t, "/// this is the server addr", gd.Doc.List[0].Text)
}

func TestDocCommentAttachValueSpecSingle(t *testing.T) {
	// SINGLE form inside a paren group: linked to the spec/ident.
	file := parseDoc(t, "const (\n\t/// the pi value\n\tpi = 3.14\n)\n")
	gd := file.Stmts[0].(*DeclStmt).Decl.(*GenDecl)
	require.Len(t, gd.Specs, 1)
	vs := gd.Specs[0].(*ValueSpec)
	require.NotNil(t, vs.Doc, "ValueSpec.Doc should be attached")
	require.Equal(t, "/// the pi value", vs.Doc.List[0].Text)
}

func TestDocCommentAttachValueSpecInline(t *testing.T) {
	// INLINE_VALUE form: trailing doc linked to the ident.
	file := parseDoc(t, "const (\n\tpi = 3.14 /// the pi value\n)\n")
	gd := file.Stmts[0].(*DeclStmt).Decl.(*GenDecl)
	vs := gd.Specs[0].(*ValueSpec)
	require.NotNil(t, vs.Doc, "inline ValueSpec.Doc should be attached")
	require.Equal(t, "/// the pi value", vs.Doc.List[0].Text)
}

func TestDocCommentAttachBlock(t *testing.T) {
	src := "/**\nthe server addr value\n**/\nconst ServerAddr = \":0\"\n"
	file := parseDoc(t, src)
	gd := file.Stmts[0].(*DeclStmt).Decl.(*GenDecl)
	require.NotNil(t, gd.Doc)
	require.Contains(t, gd.Doc.List[0].Text, "the server addr value")
}

func TestDocCommentDetachedRootBlock(t *testing.T) {
	// A ROOT_BLOCK separated by a blank line is NOT a lead doc for the stmt.
	src := "/***\nthis is a root doc\n***/\n\nconst pi = 3.14\n"
	file := parseDoc(t, src)
	gd := file.Stmts[0].(*DeclStmt).Decl.(*GenDecl)
	require.Nil(t, gd.Doc, "blank line should detach the root block")
}

func TestDocCommentAttachFuncStmt(t *testing.T) {
	file := parseDoc(t, "/// sum values\nfunc sum(a, b) { return a + b }\n")
	require.Len(t, file.Stmts, 1)
	var fe *FuncExpr
	switch s := file.Stmts[0].(type) {
	case *FuncStmt:
		fe = s.Func
	default:
		t.Fatalf("unexpected stmt %T", s)
	}
	require.NotNil(t, fe.Doc, "FuncExpr.Doc should be attached")
	require.Equal(t, "/// sum values", fe.Doc.List[0].Text)
}
