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

func TestDocCommentAttachClass(t *testing.T) {
	src := "/// a 2D point\n" +
		"class Point {\n" +
		"\t/// x coord\n\tx = 0\n" +
		"\tmethods {\n\t\t/// distance\n\t\tdist() => x\n\t}\n" +
		"}\n"
	file := parseDoc(t, src)
	cs, ok := file.Stmts[0].(*ClassStmt)
	require.True(t, ok, "want *ClassStmt, got %T", file.Stmts[0])
	require.NotNil(t, cs.Doc, "class doc should be attached")
	require.Equal(t, "/// a 2D point", cs.Doc.List[0].Text)
	require.Len(t, cs.Fields, 1)
	require.NotNil(t, cs.Fields[0].Doc, "field doc should be attached")
	require.Equal(t, "/// x coord", cs.Fields[0].Doc.List[0].Text)
	require.Len(t, cs.Methods, 1)
	require.NotNil(t, cs.Methods[0].Doc, "method doc should be attached")
	require.Equal(t, "/// distance", cs.Methods[0].Doc.List[0].Text)
}

func TestDocCommentClassRoundTrip(t *testing.T) {
	// Class body doc comments are emitted in place by the formatter (not flushed
	// at the end of the file), for both the statement and expression forms.
	for _, src := range []string{
		"class P {\n\t/// x doc\n\tx = 0\n\tmethods {\n\t\t/// m doc\n\t\tf() => x\n\t}\n}",
		"P := class {\n\t/// x doc\n\tx = 0\n\tmethods {\n\t\t/// m doc\n\t\tf() => x\n\t}\n}",
	} {
		fs := source.NewFileSet()
		f := fs.AddFileData("doc", -1, []byte(src))
		file, err := NewParserWithOptions(f, &ParserOptions{Mode: ParseComments}, nil).ParseFile()
		require.NoError(t, err)
		out := Code(file.Stmts, CodeWithComments(f, file.Comments), CodeWithPrefix("\t"))
		require.Equal(t, src, out, "class body docs should round-trip in place")
	}
}

func TestDocCommentAttachEnum(t *testing.T) {
	src := "/// permissions\n" +
		"enum Perm {\n\t/// may read\n\tRead\n\tWrite\n}\n"
	file := parseDoc(t, src)
	es, ok := file.Stmts[0].(*EnumStmt)
	require.True(t, ok, "want *EnumStmt, got %T", file.Stmts[0])
	require.NotNil(t, es.Doc)
	require.Equal(t, "/// permissions", es.Doc.List[0].Text)
	require.Len(t, es.Fields, 2)
	require.NotNil(t, es.Fields[0].Doc, "field doc should be attached")
	require.Equal(t, "/// may read", es.Fields[0].Doc.List[0].Text)
	require.Nil(t, es.Fields[1].Doc)
}

func TestDocCommentEnumRoundTrip(t *testing.T) {
	for _, src := range []string{
		"enum P {\n\t/// a doc\n\tA\n\tB\n}",
		"P := enum {\n\t/// a doc\n\tA\n\tB\n}",
	} {
		fs := source.NewFileSet()
		f := fs.AddFileData("doc", -1, []byte(src))
		file, err := NewParserWithOptions(f, &ParserOptions{Mode: ParseComments}, nil).ParseFile()
		require.NoError(t, err)
		out := Code(file.Stmts, CodeWithComments(f, file.Comments), CodeWithPrefix("\t"))
		require.Equal(t, src, out, "enum body docs should round-trip in place")
	}
}

func TestDocCommentBlockDelimitsGroup(t *testing.T) {
	// A fenced block doc must not absorb an immediately following `///` lead doc;
	// the root block stays detached and the `///` documents the statement.
	src := "/***\nmodule doc\n***/\n/// the addr\nconst Addr = \":0\"\n"
	file := parseDoc(t, src)
	gd := file.Stmts[0].(*DeclStmt).Decl.(*GenDecl)
	require.NotNil(t, gd.Doc, "the `///` should attach, not be absorbed by the root block")
	require.Equal(t, "/// the addr", gd.Doc.List[0].Text)
}
