package parser

import (
	"strings"
	"testing"

	gadxnode "github.com/gad-lang/gad/gadx/node"
	"github.com/gad-lang/gad/parser/source"
)

func parseRawTextSrc(t *testing.T, src string) *gadxnode.RawTextBlockStmt {
	t.Helper()
	f := source.NewFileSet().AddFileData("t.gadx", -1, []byte(src))
	file, err := NewParser(f).ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var found *gadxnode.RawTextBlockStmt
	for _, c := range file.Stmts {
		if comp, ok := c.(*gadxnode.CompDecl); ok {
			for _, b := range comp.Body {
				if rt, ok := b.(*gadxnode.RawTextBlockStmt); ok {
					found = rt
				}
			}
		}
	}
	if found == nil {
		t.Fatalf("no @raw_text block parsed from:\n%s", src)
	}
	return found
}

// The block's own indentation is not part of its content; the indentation
// inside it is, so a nested line stays nested.
func TestRawTextBlockDedents(t *testing.T) {
	rt := parseRawTextSrc(t, "@main\n    @raw_text\n        a {\n            b: 1;\n        }\n")

	want := []string{"a {", "    b: 1;", "}"}
	if strings.Join(rt.Lines, "|") != strings.Join(want, "|") {
		t.Errorf("lines = %q, want %q", rt.Lines, want)
	}
}

// Braces are content: only `#{ … }#` interpolates.
func TestRawTextBlockBracesAreLiteral(t *testing.T) {
	rt := parseRawTextSrc(t, "@main\n    @raw_text\n        .a { color: red }\n")

	if len(rt.Body) != 1 {
		t.Fatalf("got %d body parts, want the whole line as one literal", len(rt.Body))
	}
	if strings.Join(rt.Lines, "") != ".a { color: red }" {
		t.Errorf("lines = %q", rt.Lines)
	}
}

// A blank line inside the block is kept: it is content too.
func TestRawTextBlockKeepsBlankLines(t *testing.T) {
	rt := parseRawTextSrc(t, "@main\n    @raw_text\n        a\n\n        b\n")

	if len(rt.Lines) != 3 || rt.Lines[1] != "" {
		t.Errorf("lines = %q, want a blank line between a and b", rt.Lines)
	}
}
