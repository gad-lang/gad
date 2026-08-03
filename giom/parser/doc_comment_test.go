package parser

import (
	"bytes"
	"strings"
	"testing"

	giomnode "github.com/gad-lang/gad/giom/node"
)

func findComp(f *giomnode.File, name string) *giomnode.CompDecl {
	for _, c := range f.Comps {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// TestDocCommentAttach verifies that a `/** … **/` doc comment immediately before
// a @comp/@func attaches to its Doc, while a blank-separated or non-adjacent one
// stays as a file-level comment (like gad).
func TestDocCommentAttach(t *testing.T) {
	// Immediately before @comp -> attaches.
	f := parseLine(t, "/** greets a user **/\n@comp greeting(name)\n    p hi\n")
	c := findComp(f, "greeting")
	if c == nil {
		t.Fatal("comp greeting not found")
	}
	if c.Doc != "greets a user" {
		t.Fatalf("comp.Doc = %q, want %q", c.Doc, "greets a user")
	}

	// Immediately before @func -> attaches.
	f = parseLine(t, "/** adds two **/\n@func add(a, b)\n    p x\n")
	var fd *giomnode.FuncDecl
	for _, s := range f.Stmts {
		if d, ok := s.(*giomnode.FuncDecl); ok {
			fd = d
		}
	}
	if fd == nil || fd.Doc != "adds two" {
		t.Fatalf("func.Doc = %q, want %q", docOf(fd), "adds two")
	}

	// Blank line between -> not attached; stays a file-level comment.
	f = parseLine(t, "/** floating **/\n\n@comp foo()\n    p x\n")
	if c := findComp(f, "foo"); c != nil && c.Doc != "" {
		t.Fatalf("blank-separated comp.Doc = %q, want empty", c.Doc)
	}
	comments := 0
	for _, s := range f.Stmts {
		if _, ok := s.(*giomnode.CommentStmt); ok {
			comments++
		}
	}
	if comments != 1 {
		t.Fatalf("file-level comments = %d, want 1", comments)
	}
}

func docOf(fd *giomnode.FuncDecl) string {
	if fd == nil {
		return "<nil>"
	}
	return fd.Doc
}

// TestBlockCommentWriteGiom checks block/doc comments round-trip through WriteGiom.
func TestBlockCommentWriteGiom(t *testing.T) {
	f := parseLine(t, "/** greets **/\n@comp greeting(name)\n    p hi\n/* plain block */\n@main\n    p x\n")
	var buf bytes.Buffer
	f.WriteGiom(giomnode.NewGiomCodeContext(&buf))
	out := buf.String()
	for _, want := range []string{"/** greets **/", "@comp greeting", "/* plain block */"} {
		if !strings.Contains(out, want) {
			t.Fatalf("WriteGiom missing %q:\n%s", want, out)
		}
	}
}
