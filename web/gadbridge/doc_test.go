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
		"greetings module.", "## Public API",
		"### hello", "= \"hi\"", "The greeting prefix.",
		"### add", "Adds two numbers.",
	} {
		if !strings.Contains(md, w) {
			t.Fatalf("gad doc missing %q:\n%s", w, md)
		}
	}
	// The rendered Markdown must be clean — no raw data-source-pos span or comment
	// markers.
	if strings.Contains(md, "data-source-pos") {
		t.Fatalf("gad doc leaked a raw data-source-pos span:\n%s", md)
	}
	if strings.Contains(md, "**/") || strings.Contains(md, "/**") {
		t.Fatalf("gad doc leaked comment markers:\n%s", md)
	}
}

// TestDocGadx documents a .gadx template: components with doc text.
func TestDocGadx(t *testing.T) {
	src := "/** Reusable widgets. **/\n@comp greeting(name)\n    p hi\n"
	md, err := Doc(src, "gadx")
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range []string{
		"## Components", "### +greeting",
		"Reusable widgets.",
	} {
		if !strings.Contains(md, w) {
			t.Fatalf("gadx doc missing %q:\n%s", w, md)
		}
	}
}

// TestExtractDocGadtModuleProse verifies that a `.gadt` module doc placed inside
// the leading code island — a `{%-- … --%}` block wrapping a `/*** … ***/` root
// comment, a `/** … **/` block or a normal `/* … */` comment — is captured as
// prose and never leaks into the rendered template output.
func TestExtractDocGadtModuleProse(t *testing.T) {
	cases := []struct{ name, src, wantProse string }{
		{"root", "{%--\n/***\nRoot doc.\n***/\n--%}\n<h1>hi</h1>\n", "Root doc."},
		{"block", "{%--\n/**\nBlock doc.\n**/\n--%}\n<h1>hi</h1>\n", "Block doc."},
		{"normal", "{%-- /* Normal doc. */ --%}\n<h1>hi</h1>\n", "Normal doc."},
		{"shebang+root", "#!/usr/bin/env gad\n{%--\n/*** After shebang. ***/\n--%}\n<h1>hi</h1>\n", "After shebang."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d, err := ExtractDoc(c.src, "gadTemplate")
			if err != nil {
				t.Fatal(err)
			}
			if d.Prose != c.wantProse {
				t.Fatalf("prose = %q, want %q", d.Prose, c.wantProse)
			}
			// The doc comment must not be emitted as template text.
			if r := RunSource(c.src, "gadTemplate"); r.Stdout != "<h1>hi</h1>\n" {
				t.Fatalf("run leaked the doc: %q", r.Stdout)
			}
		})
	}

	// A plain .gad file's module prose is a DETACHED leading block (blank line
	// after); a block glued to a statement documents the statement instead.
	if d, _ := ExtractDoc("/** Plain gad. **/\n\nexport a = 1\n", "gad"); d.Prose != "Plain gad." {
		t.Fatalf("gad prose regressed: %q", d.Prose)
	}
}

// TestExtractDocModuleProseBlank verifies the module prose is a leading `/** … **/`
// block DETACHED from the code — followed by a blank line or at end of file — and
// that a block glued to a statement documents the statement, not the module.
func TestExtractDocModuleProseBlank(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"detached", "/**\n# Mod\nDoc.\n**/\n\nconst x = 1\n", "# Mod\nDoc."},
		{"eof", "/**\n# Mod\n**/\n", "# Mod"},
		{"three-star still works", "/***\n# Mod\n***/\n\nconst x = 1\n", "# Mod"},
		{"attached is not module", "/**\nDoc of x.\n**/\nconst x = 1\n", ""},
		{"none", "const x = 1\n", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d, err := ExtractDoc(c.src, "gad")
			if err != nil {
				t.Fatal(err)
			}
			if d.Prose != c.want {
				t.Fatalf("prose = %q, want %q", d.Prose, c.want)
			}
		})
	}
}
