package main

import (
	"strings"
	"testing"
)

func TestHTMLToGadxWrapsInMain(t *testing.T) {
	got := htmlToGadx("<!DOCTYPE html>\n<html>\n<body>hi</body>\n</html>\n")

	if !strings.HasPrefix(got, "@main\n") {
		t.Fatalf("output does not open with @main:\n%s", got)
	}
	// The doctype is a declaration, which Gadx drops; the statement renders it.
	if !strings.Contains(got, "    !!! 5\n") {
		t.Errorf("doctype was not lifted to `!!! 5`:\n%s", got)
	}
	if strings.Contains(got, "<!DOCTYPE") {
		t.Errorf("doctype left as markup, where it would be dropped:\n%s", got)
	}
	// The gaps between the tags were only the source's indentation, so the
	// markup comes out as one run — the page reads the same either way.
	if !strings.Contains(got, "    <html><body>hi</body></html>\n") {
		t.Errorf("markup was not lifted whole:\n%s", got)
	}
}

// The lift is only half of what `gad transpile page.html` writes: the file is
// then formatted, which is what turns the markup into the tag syntax Gadx is
// written in.
func TestHTMLToGadxFormattedWritesTagSyntax(t *testing.T) {
	got := htmlToGadxFormatted(
		"<!DOCTYPE html>\n<html>\n<body>\n<div class=\"card\">\n<h1 id=\"t\">Hi</h1>\n" +
			"<p>Some <b>bold</b> text</p>\n<select><option value=\"\">Any</option></select>\n" +
			"</div>\n</body>\n</html>\n")

	for _, want := range []string{
		"@main",
		"\t!!! 5",
		"\thtml",
		"\t\tbody",
		"\t\t\tdiv.card",
		"\t\t\t\th1#t Hi",
		"\t\t\t\t\t{= \"Some \" }",
		"\t\t\t\t\tb bold",
		"\t\t\t\t\t{= \" text\" }",
		"\t\t\t\t\toption[value=@empty] Any",
	} {
		if !strings.Contains(got, want+"\n") {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "<div") || strings.Contains(got, "</") {
		t.Errorf("markup left inline:\n%s", got)
	}
}

// A single space between two elements is content — it is what keeps the two
// words apart — and `*` is how a line carries it.
func TestHTMLToGadxFormattedWritesSpaceMarker(t *testing.T) {
	got := htmlToGadxFormatted("<p><a>one</a> <a>two</a></p>\n")

	if !strings.Contains(got, "\t\t*\n") {
		t.Errorf("the space between the links was not written as `*`:\n%s", got)
	}
	if strings.Contains(got, `{= " " }`) {
		t.Errorf("space written the long way:\n%s", got)
	}
}

// A brace in HTML text is a brace; in Gadx it opens an interpolation.
func TestHTMLToGadxEscapesBraces(t *testing.T) {
	got := htmlToGadx("<p>a {b} c</p>\n")

	if !strings.Contains(got, `<p>a \{b\} c</p>`) {
		t.Errorf("braces not escaped:\n%s", got)
	}
}

// Script and stylesheet content is code: its braces are content and its own
// nesting is written out verbatim. What goes is the indentation the page put
// around the element, which is not part of the language inside — without that,
// the body cannot be written under its tag.
func TestHTMLToGadxLeavesRawTextAlone(t *testing.T) {
	got := htmlToGadx(
		"<style>\n    .a {\n        color: red;\n    }\n</style>\n" +
			"<script>\n    if (x) { y(); }\n</script>\n")

	for _, want := range []string{".a {\n    color: red;\n}", "if (x) { y(); }"} {
		if !strings.Contains(got, want) {
			t.Errorf("raw content was rewritten, missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "\n        color") {
		t.Errorf("the page's own indentation was kept:\n%s", got)
	}
	if strings.Contains(got, `\{`) {
		t.Errorf("braces escaped inside raw text:\n%s", got)
	}
}

// An entity left as it is would go out with its `&` escaped, so the reader
// would see the entity instead of the character.
func TestHTMLToGadxDecodesEntities(t *testing.T) {
	got := htmlToGadx("<p>&copy; 2024 &amp; more</p>\n")

	if !strings.Contains(got, "© 2024 & more") {
		t.Errorf("entities not decoded:\n%s", got)
	}
}

// `&lt;` and `&gt;` stay: decoded, they would become markup.
func TestHTMLToGadxKeepsAngleEntities(t *testing.T) {
	got := htmlToGadx("<p>&lt;div&gt; is a tag</p>\n")

	if !strings.Contains(got, "&lt;div&gt;") {
		t.Errorf("angle entities were decoded into markup:\n%s", got)
	}
}

func TestIsHTMLFile(t *testing.T) {
	for name, want := range map[string]bool{
		"a.html": true, "a.htm": true, "A.HTML": true,
		"a.gadx": false, "a.gad": false, "a.htmlx": false,
	} {
		if got := isHTMLFile(name); got != want {
			t.Errorf("isHTMLFile(%q) = %v, want %v", name, got, want)
		}
	}
}

// Whitespace just inside a block is trimmed: a browser trims it too, and kept
// it would turn a whole paragraph into a quoted literal.
func TestHTMLToGadxTrimsBlockEdges(t *testing.T) {
	got := htmlToGadxFormatted(
		"<div>\n  <p>\n    Hello there\n  </p>\n" +
			"  <span> kept </span>\n" +
			"  <pre>\n  as written\n  </pre>\n</div>\n")

	if !strings.Contains(got, "p Hello there\n") {
		t.Errorf("the paragraph's edges were not trimmed:\n%s", got)
	}
	// an inline element's own edges are content
	if !strings.Contains(got, `span {= " kept " }`) {
		t.Errorf("an inline element lost its edges:\n%s", got)
	}
	// `pre` is laid out as written, so nothing in it is touched
	if !strings.Contains(got, "as written") {
		t.Errorf("pre content was lost:\n%s", got)
	}
}

// The trim is on the element's inner edges only — the space between two inline
// elements is still content.
func TestHTMLToGadxKeepsInlineSpacing(t *testing.T) {
	got := htmlToGadxFormatted("<p>\n  Some <b>bold</b> <i>and</i> more\n</p>\n")

	if !strings.Contains(got, `{= "Some " }`) {
		t.Errorf("leading edge not trimmed, or the word run broken:\n%s", got)
	}
	if !strings.Contains(got, "*\n") {
		t.Errorf("the space between the two inline elements was dropped:\n%s", got)
	}
	if !strings.Contains(got, `{= " more" }`) {
		t.Errorf("trailing edge wrongly kept or word run broken:\n%s", got)
	}
}

// A `<pre>` is laid out as written: the lift may not shift its lines, and the
// formatter has to write the content back in a form that carries it.
func TestHTMLToGadxKeepsPreVerbatim(t *testing.T) {
	got := htmlToGadxFormatted("<div>\n  <pre>  a   b\n  c    d\n</pre>\n</div>\n")

	if !strings.Contains(got, `{= "  a   b\n  c    d\n" }`) {
		t.Errorf("pre content was collapsed or shifted:\n%s", got)
	}
	// short content with nothing to lose still reads plainly
	if plain := htmlToGadxFormatted("<pre>hello</pre>\n"); !strings.Contains(plain, "pre hello") {
		t.Errorf("a plain pre was written the long way:\n%s", plain)
	}
}
