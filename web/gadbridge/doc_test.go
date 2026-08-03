package gadbridge

import (
	"strings"
	"testing"
)

// TestDocGad documents a .gad file: module heading, /*** root block, and the
// exported symbols with their doc comments.
func TestDocGad(t *testing.T) {
	src := "/*** greetings module. ***/\n\n" +
		"/** The greeting prefix. **/\nexport hello = \"hi\"\n\n" +
		"/** Adds two numbers. **/\nexport func add(a, b) { return a + b }\n"
	md, err := Doc(src, "gad")
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range []string{
		"greetings module.", "## Exports",
		"hello = \"hi\"", "The greeting prefix.",
		"add", "Adds two numbers.",
	} {
		if !strings.Contains(md, w) {
			t.Fatalf("gad doc missing %q:\n%s", w, md)
		}
	}
	if strings.Contains(md, "**/") || strings.Contains(md, "/**") {
		t.Fatalf("gad doc leaked comment markers:\n%s", md)
	}
}

// TestDocGiom documents a .giom template: components with doc text and a
// data-source-pos anchor.
func TestDocGiom(t *testing.T) {
	src := "/** Reusable widgets. **/\n@comp greeting(name)\n    p hi\n"
	md, err := Doc(src, "giom")
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range []string{
		"## Components", "+greeting",
		"Reusable widgets.", `data-source-pos="2,1"`,
	} {
		if !strings.Contains(md, w) {
			t.Fatalf("giom doc missing %q:\n%s", w, md)
		}
	}
}
