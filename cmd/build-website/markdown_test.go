package main

import "strings"

import "testing"

// TestRenderNestedList guards against the infinite loop that previously caused
// the renderer to allocate unbounded memory (OOM / exit 137) when a list item
// had a deeper-indented child: the nested renderList call must advance past the
// last consumed line.
func TestRenderNestedList(t *testing.T) {
	src := "- a\n  - b\n- c\n"
	out, _ := renderMarkdown(src)
	for _, want := range []string{
		"<li>a</li>", "<li>b</li>", "<li>c</li>",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	// The nested item must produce exactly two <ul> openings (outer + nested).
	if got := strings.Count(out, "<ul>"); got != 2 {
		t.Fatalf("want 2 <ul>, got %d:\n%s", got, out)
	}
	if got := strings.Count(out, "<li>"); got != 3 {
		t.Fatalf("want 3 <li>, got %d:\n%s", got, out)
	}
}

func TestRenderBlocks(t *testing.T) {
	cases := map[string]string{
		"# Title\n":                     `<h1 id="title">Title</h1>`,
		"text with `code` span\n":       "<code>code</code>",
		"**bold** and _em_\n":           "<strong>bold</strong>",
		"> quoted\n":                    "<blockquote>",
		"```gad\nx := 1\n```\n":         `<pre><code class="language-gad">`,
		"| a | b |\n|---|---|\n|1|2|\n": "<table>",
		"[doc](README.md)\n":            `href="index.html"`,
		"[other](modules.md)\n":         `href="modules.html"`,
	}
	for src, want := range cases {
		out, _ := renderMarkdown(src)
		if !strings.Contains(out, want) {
			t.Errorf("render(%q) missing %q:\n%s", src, want, out)
		}
	}
}

// TestRenderNestedFence guards the CommonMark fence-width rule: a code block
// opened with a wider fence (```` ) must not be closed by a shorter inner ```
// run — that inner run (e.g. a doctest fence embedded in a doc comment) is
// literal content. Regression for the corrupted "forms" section of
// lang-16_doc_comments.html.
func TestRenderNestedFence(t *testing.T) {
	src := "````gad\n" +
		"/**\n" +
		"```gad\n" +
		"sum(2, 3)\n" +
		">>> 5\n" +
		"```\n" +
		"**/\n" +
		"func sum(a, b) { a + b }\n" +
		"````\n" +
		"\nAfter the block.\n"
	out, _ := renderMarkdown(src)

	// Exactly one code block: one <pre><code ...> ... </code></pre>.
	if got := strings.Count(out, "<pre><code"); got != 1 {
		t.Fatalf("want 1 code block, got %d:\n%s", got, out)
	}
	// The inner ```gad doctest survives as escaped literal content inside it.
	if !strings.Contains(out, "```gad") {
		t.Fatalf("inner fence not preserved as content:\n%s", out)
	}
	if !strings.Contains(out, "func sum(a, b)") {
		t.Fatalf("code after inner fence was lost (closed early):\n%s", out)
	}
	// The trailing prose must render as a paragraph, not code.
	if !strings.Contains(out, "<p>After the block.</p>") {
		t.Fatalf("trailing prose not rendered after the block:\n%s", out)
	}
}

// TestRenderTerminates ensures the renderer never loops forever on assorted
// block combinations (each rendered under the test's own timeout).
func TestRenderTerminates(t *testing.T) {
	srcs := []string{
		"- a\n    - deep\n      - deeper\n- back\n",
		"1. one\n2. two\n   1. nested\n",
		"> a\n> b\n\npara\n\n## h2\n",
		"para line 1\npara line 2\n\n- list\n",
	}
	for _, s := range srcs {
		if out, _ := renderMarkdown(s); out == "" {
			t.Errorf("empty render for %q", s)
		}
	}
}
