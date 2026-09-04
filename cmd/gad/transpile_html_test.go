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
	for _, want := range []string{"    <html>", "    <body>hi</body>", "    </html>"} {
		if !strings.Contains(got, want+"\n") {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

// A brace in HTML text is a brace; in Gadx it opens an interpolation.
func TestHTMLToGadxEscapesBraces(t *testing.T) {
	got := htmlToGadx("<p>a {b} c</p>\n")

	if !strings.Contains(got, `<p>a \{b\} c</p>`) {
		t.Errorf("braces not escaped:\n%s", got)
	}
}

// Script and stylesheet content is code: its braces are content, and its
// indentation is written out verbatim, so neither may be touched.
func TestHTMLToGadxLeavesRawTextAlone(t *testing.T) {
	got := htmlToGadx("<style>\n    .a { color: red; }\n</style>\n<script>\n    if (x) { y(); }\n</script>\n")

	for _, want := range []string{"    .a { color: red; }", "    if (x) { y(); }"} {
		if !strings.Contains(got, want+"\n") {
			t.Errorf("raw content was rewritten or shifted, missing %q:\n%s", want, got)
		}
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
