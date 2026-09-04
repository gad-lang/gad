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
