package gadbridge_test

import (
	"strings"
	"testing"

	"github.com/gad-lang/gad/web/gadbridge"
)

// formatTwice formats src and then formats the result, reporting both, so a
// test can check that formatting settles and that it says the same thing.
func formatTwice(t *testing.T, src string) (once, twice string) {
	t.Helper()
	o := gadbridge.GadxFormatOptions{Indent: "    "}
	r1 := gadbridge.FormatGadx(src, o)
	if !r1.OK {
		t.Fatalf("format failed: %v", r1.Diagnostics)
	}
	r2 := gadbridge.FormatGadx(r1.Source, o)
	if !r2.OK {
		t.Fatalf("second format failed: %v", r2.Diagnostics)
	}
	return r1.Source, r2.Source
}

// assertFormatSafe checks the two guards `gad fmt` applies before rewriting a
// file: formatting settles, and it does not change what the file means.
func assertFormatSafe(t *testing.T, name, src string) {
	t.Helper()
	once, twice := formatTwice(t, src)
	if once != twice {
		t.Errorf("%s: formatting does not settle\nfirst:\n%s\nsecond:\n%s", name, once, twice)
	}
	before, ok1 := gadbridge.GadxLowered(src)
	after, ok2 := gadbridge.GadxLowered(once)
	if !ok1 || !ok2 {
		t.Fatalf("%s: could not lower", name)
	}
	if before != after {
		t.Errorf("%s: formatting changed the meaning\nformatted:\n%s", name, once)
	}
}

// A script or a stylesheet holds text whose whitespace and braces are part of
// the language, so the formatter writes it back as the HTML region it is.
func TestFormatKeepsRawTextElements(t *testing.T) {
	for name, src := range map[string]string{
		"style":  "@main\n    <style>\n        .a { color: red; }\n    </style>\n",
		"script": "@main\n    <script>\n        if (x) { y(); }\n    </script>\n",
		"src":    "@main\n    <script src=\"a.js\"></script>\n",
	} {
		assertFormatSafe(t, name, src)
	}

	once, _ := formatTwice(t, "@main\n    <style>\n        .a { color: red; }\n    </style>\n")
	if !strings.Contains(once, ".a { color: red; }") {
		t.Errorf("stylesheet text was rewritten:\n%s", once)
	}
}

// Whitespace between elements renders, so it has to survive formatting — as
// must whitespace at the edges of a text run.
func TestFormatKeepsMeaningfulWhitespace(t *testing.T) {
	assertFormatSafe(t, "between elements", "@main\n    <div>\n    <span>x</span>\n    </div>\n")
	assertFormatSafe(t, "around text", "@main\n    <a href=\"#x\">\n        Get Free Quote\n    </a>\n")
}

// A `| …` line under a tag is a sibling, not the tag's inline body: only text
// on the tag's own line is inline. An attribute group may span lines, so what
// counts is where it ends.
func TestFormatSiblingTextLine(t *testing.T) {
	assertFormatSafe(t, "sibling pipe", "@main\n    div\n        i[class=\"a\"]\n        | {= \" \" }\n")
	assertFormatSafe(t, "wrapped attrs then inline text",
		"@main\n    a[\n        href=\"/x\"\n        title=\"go\"\n    ] link\n")
}

// An attribute that is present and empty comes back as `@empty`, the form that
// says so; a bare "" would mean the opposite, since falsy values are dropped.
func TestFormatWritesEmptyAttr(t *testing.T) {
	once, _ := formatTwice(t, "@main\n    <option value=\"\">x</option>\n")
	if !strings.Contains(once, "value=@empty") {
		t.Errorf("empty attribute did not come back as @empty:\n%s", once)
	}
	assertFormatSafe(t, "empty attr", "@main\n    <option value=\"\">x</option>\n")
}

// A `@raw_text` block is content: it survives formatting unchanged, and its
// indentation is re-applied rather than reflowed.
func TestFormatKeepsRawTextBlock(t *testing.T) {
	src := "@main\n    @raw_text\n        .a {\n            color: red;\n        }\n"
	assertFormatSafe(t, "raw_text block", src)

	once, _ := formatTwice(t, src)
	if !strings.Contains(once, "@raw_text") {
		t.Errorf("the block lost its directive:\n%s", once)
	}
	if !strings.Contains(once, "color: red;") {
		t.Errorf("the body was rewritten:\n%s", once)
	}
}
