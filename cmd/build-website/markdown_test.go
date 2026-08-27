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
	// CommonMark nests the child list inside its parent <li> (so the parent item
	// is not closed before the nested <ul>); assert the structure loosely.
	for _, want := range []string{"<li>b</li>", "<li>c</li>"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	if got := strings.Count(out, "<ul>"); got != 2 {
		t.Fatalf("want 2 <ul> (outer + nested), got %d:\n%s", got, out)
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
// lang-doc_comments.html.
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

// TestRenderLinkUnderscores guards the "Language chapters" table: links whose
// text and destination contain intra-word underscores (hello,
// lang-values_and_types.html) must render as clean anchors, never mangled
// into <em> spans. Regression for the corrupted sample links on the site.
func TestRenderLinkUnderscores(t *testing.T) {
	cases := map[string]string{
		"- [hello](lang-hello.html)\n":                       `<a href="lang-hello.html">hello</a>`,
		"- [values_and_types](lang-values_and_types.html)\n": `<a href="lang-values_and_types.html">values_and_types</a>`,
		"[interfaces](interfaces.md)\n":                      `<a href="interfaces.html">interfaces</a>`,
	}
	for src, want := range cases {
		out, _ := renderMarkdown(src)
		if !strings.Contains(out, want) {
			t.Errorf("render(%q)\n  missing %q\n  got:     %s", src, want, out)
		}
		if strings.Contains(out, "<em>") {
			t.Errorf("render(%q) wrongly emphasized underscores:\n%s", src, out)
		}
	}
	// A genuine _emphasis_ flanked by non-word chars still renders.
	if out, _ := renderMarkdown("a real _word_ here\n"); !strings.Contains(out, "<em>word</em>") {
		t.Errorf("real underscore emphasis lost:\n%s", out)
	}
}

// TestRenderCodeSpanLink guards the "Sample source" column of doc/README's
// Language-chapters table: a link whose text is a code span
// [`samples/NN.gad`](../samples/NN.gad) must render as one anchor with a <code>
// label, not a broken `[` + code + `](url)` literal. The raw-source destination
// resolves to the published chapter page.
func TestRenderCodeSpanLink(t *testing.T) {
	out, _ := renderMarkdown("[`samples/values_and_types.gad`](../samples/values_and_types.gad)\n")
	want := `<a href="lang-values_and_types.html"><code>samples/values_and_types.gad</code></a>`
	if !strings.Contains(out, want) {
		t.Fatalf("code-span link mangled:\n want %s\n got  %s", want, out)
	}
	if strings.Contains(out, "](") || strings.Contains(out, "../samples/") {
		t.Fatalf("raw link syntax / unpublished source leaked:\n%s", out)
	}
	// Rendered-doc column: samples/NN.md -> lang-NN.html.
	if o, _ := renderMarkdown("[02](samples/values_and_types.md)\n"); !strings.Contains(o, `href="lang-values_and_types.html"`) {
		t.Fatalf("rendered-doc link not mapped:\n%s", o)
	}
	// A code span outside any link still renders.
	if o, _ := renderMarkdown("use `gad fmt` here\n"); !strings.Contains(o, "<code>gad fmt</code>") {
		t.Fatalf("plain code span lost:\n%s", o)
	}
}

// TestRenderStarEmphasis guards single-star `*emphasis*` rendering (the
// getting-started "*raw*" showed literally) while leaving asterisks glued to
// words — arithmetic and spreads in prose — untouched.
func TestRenderStarEmphasis(t *testing.T) {
	yes := map[string]string{
		"it opts into *raw* argument\n": "into <em>raw</em> argument",
		"a *b c* d\n":                   "a <em>b c</em> d",
		"lead *directly* here\n":        "lead <em>directly</em> here",
	}
	for src, want := range yes {
		if out, _ := renderMarkdown(src); !strings.Contains(out, want) {
			t.Errorf("render(%q)\n  missing %q\n  got: %s", src, want, out)
		}
	}
	// Arithmetic like `x*this.x + this.y*t` in real docs lives inside code fences,
	// so goldmark never emphasizes it; a fenced sample keeps its stars literal.
	if out, _ := renderMarkdown("```gad\nv.x*v.x + v.y*v.y\n```\n"); strings.Contains(out, "<em>") {
		t.Errorf("code fence stars wrongly emphasized:\n%s", out)
	}
	// `**bold**` must still be bold, not eaten by the single-star rule.
	if out, _ := renderMarkdown("**strong** stuff\n"); !strings.Contains(out, "<strong>strong</strong>") {
		t.Errorf("bold broken:\n%s", out)
	}
}

// TestRenderEmphasisLinkNesting guards emphasis and links nesting either way:
// bold wrapping a link (the getting-started "See **[Templates](…)**"), emphasis
// inside link text, and a code span as link text.
func TestRenderEmphasisLinkNesting(t *testing.T) {
	cases := map[string]string{
		// Bold wrapping a link — the reported bug.
		"See **[Templates](samples/template.md)** now\n": `<strong><a href="lang-template.html">Templates</a></strong>`,
		// Emphasis inside link text.
		"[**bold**](x.md)\n": `<a href="x.html"><strong>bold</strong></a>`,
		"[_em_](x.md)\n":     `<a href="x.html"><em>em</em></a>`,
		// Code span as link text (Sample source column) still works.
		"[`samples/x.gad`](../samples/x.gad)\n": `<a href="lang-x.html"><code>samples/x.gad</code></a>`,
	}
	for src, want := range cases {
		out, _ := renderMarkdown(src)
		if !strings.Contains(out, want) {
			t.Errorf("render(%q)\n  missing %q\n  got: %s", src, want, out)
		}
		if strings.Contains(out, "**") || strings.Contains(out, "](") {
			t.Errorf("render(%q) left literal markup:\n%s", src, out)
		}
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
