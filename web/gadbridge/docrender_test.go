package gadbridge

import (
	"strings"
	"testing"
)

func TestDocHTML(t *testing.T) {
	src := "/***\n# Title\n\nSome **bold** and a list:\n\n- one\n- two\n***/\n\nreturn 1\n"
	html, err := DocHTML(src, "gad")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<h1", "Title</h1>", "<strong>bold</strong>", "<ul>", "<li>one</li>"} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in:\n%s", want, html)
		}
	}
}

func TestRenderMarkdownToHTML_GFMTable(t *testing.T) {
	html, err := RenderMarkdownToHTML("| a | b |\n|---|---|\n| 1 | 2 |\n")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "<table>") || !strings.Contains(html, "<td>1</td>") {
		t.Fatalf("GFM table not rendered:\n%s", html)
	}
}
