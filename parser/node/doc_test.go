package node

import (
	"strings"
	"testing"

	"github.com/gad-lang/gad/parser/ast"
)

func docGroup(texts ...string) *ast.CommentGroup {
	g := &ast.CommentGroup{}
	for _, t := range texts {
		g.List = append(g.List, &ast.Comment{Text: t})
	}
	return g
}

func TestParseDocComment(t *testing.T) {
	cases := []struct {
		name    string
		group   *ast.CommentGroup
		kind    docKind
		content string
	}{
		{"single", docGroup("/? the pi value"), docSingle, "the pi value"},
		{"single multi", docGroup("/? line one", "/? line two"), docSingle, "line one\nline two"},
		{"block", docGroup("/??\nthe pi value\nmore\n??"), docBlock, "the pi value\nmore"},
		{"root", docGroup("/???\nmodule overview\n???"), docRoot, "module overview"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d, ok := parseDocComment(c.group)
			if !ok {
				t.Fatalf("parseDocComment failed")
			}
			if d.kind != c.kind {
				t.Errorf("kind = %d, want %d", d.kind, c.kind)
			}
			if d.content != c.content {
				t.Errorf("content = %q, want %q", d.content, c.content)
			}
		})
	}
}

func TestRenderDocLines(t *testing.T) {
	// short single stays single
	d := docComment{docSingle, "the pi value"}
	got := strings.Join(renderDocLines(d, 80), "\n")
	if got != "/? the pi value" {
		t.Errorf("short single = %q", got)
	}

	// short block collapses to single
	d = docComment{docBlock, "the pi value"}
	got = strings.Join(renderDocLines(d, 80), "\n")
	if got != "/? the pi value" {
		t.Errorf("short block = %q", got)
	}

	// long single becomes a wrapped block
	long := "aaaa bbbb cccc dddd eeee ffff gggg hhhh iiii jjjj kkkk llll mmmm"
	d = docComment{docSingle, long}
	lines := renderDocLines(d, 24)
	if lines[0] != "/??" || lines[len(lines)-1] != "??" {
		t.Errorf("long single not blocked: %v", lines)
	}
	for _, ln := range lines[1 : len(lines)-1] {
		if len(ln) > 24 {
			t.Errorf("line exceeds width: %q (%d)", ln, len(ln))
		}
	}

	// root stays a root block
	d = docComment{docRoot, "overview"}
	lines = renderDocLines(d, 80)
	if lines[0] != "/???" || lines[len(lines)-1] != "???" {
		t.Errorf("root not blocked: %v", lines)
	}
}

func TestReflowMarkdownPreservesStructure(t *testing.T) {
	content := "a calculator\nspanning lines\n\n- item one\n- item two\n\n```\ncode here\n```"
	lines := reflowMarkdown(content, 80)
	out := strings.Join(lines, "\n")
	// paragraph joined
	if !strings.Contains(out, "a calculator spanning lines") {
		t.Errorf("paragraph not joined: %q", out)
	}
	// list items preserved line-for-line
	if !strings.Contains(out, "- item one\n- item two") {
		t.Errorf("list not preserved: %q", out)
	}
	// code fence preserved
	if !strings.Contains(out, "```\ncode here\n```") {
		t.Errorf("code fence not preserved: %q", out)
	}
}
