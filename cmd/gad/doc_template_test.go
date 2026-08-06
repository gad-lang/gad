package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gad "github.com/gad-lang/gad"
	"github.com/gad-lang/gad/gadconfig"
	"github.com/gad-lang/gad/web/gadbridge"
	cc "github.com/moisespsena-go/command-context"
)

// docSampleSrc is a small .gad module with a root block and two exports.
const docSampleSrc = "/*** greetings module. ***/\n\n" +
	"/** The greeting prefix. **/\nexport hello = \"hi\"\n\n" +
	"/** Adds two numbers. **/\nexport func add(a, b) { return a + b }\n"

// TestRenderDocTemplateMD renders the Markdown template (.gad/doc-templates/md.gadx)
// against the extracted structure and checks the emitted Markdown.
func TestRenderDocTemplateMD(t *testing.T) {
	tmpl := readSampleTemplate(t, gadconfig.DocMDTemplateName)
	doc, err := gadbridge.ExtractDoc(docSampleSrc, "gad")
	if err != nil {
		t.Fatal(err)
	}
	out, err := renderDocTemplate(tmpl, "md.gadx", mustDocDict(t, doc, "greetings.gad", docSampleSrc))
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range []string{"# greetings", "greetings module.", "## Exports", "### hello = \"hi\"", "The greeting prefix.", "### add", "Adds two numbers.", "## Example — `greetings.gad`", "```gad"} {
		if !strings.Contains(out, w) {
			t.Fatalf("md template missing %q:\n%s", w, out)
		}
	}
}

// TestRenderDocTemplateHTML renders the HTML template (.gad/doc-templates/html.gadx)
// and checks the emitted HTML (structure + data-* source positions).
func TestRenderDocTemplateHTML(t *testing.T) {
	tmpl := readSampleTemplate(t, gadconfig.DocHTMLTemplateName)
	doc, err := gadbridge.ExtractDoc(docSampleSrc, "gad")
	if err != nil {
		t.Fatal(err)
	}
	out, err := renderDocTemplate(tmpl, "html.gadx", mustDocDict(t, doc, "greetings.gad", docSampleSrc))
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range []string{
		"<article", "class=\"prose\"", "greetings module.",
		"<h2>Exports</h2>", "<code>hello", "The greeting prefix.",
		"data-line=", "data-column=",
	} {
		if !strings.Contains(out, w) {
			t.Fatalf("html template missing %q:\n%s", w, out)
		}
	}
}

// TestDocCommandUsesTemplates runs `gad doc` end-to-end in a workspace whose
// .gad/doc-templates holds both templates, and checks that md.gadx drives the
// .md output while html.gadx additionally emits a .html file.
func TestDocCommandUsesTemplates(t *testing.T) {
	dir := t.TempDir()
	tdir := gadconfig.DocTemplatesDir(dir)
	if err := os.MkdirAll(tdir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{gadconfig.DocMDTemplateName, gadconfig.DocHTMLTemplateName} {
		if err := os.WriteFile(filepath.Join(tdir, name), readSampleTemplate(t, name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "m.gad"), []byte(docSampleSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	var out, errBuf bytes.Buffer
	inCtx := &cc.CommandContext{Out: &out, Err: &errBuf, InputArgs: cc.Args{"--no-doctest", "m.gad"}}
	runCtx, err := docCommand().Parse(inCtx)
	if err != nil {
		t.Fatal(err)
	}
	if err := runCtx.Run(); err != nil {
		t.Fatal(err)
	}

	md, err := os.ReadFile(filepath.Join(dir, "doc", "m.md"))
	if err != nil {
		t.Fatalf("read doc/m.md: %v", err)
	}
	if !strings.Contains(string(md), "## Exports") || !strings.Contains(string(md), "### hello = \"hi\"") {
		t.Fatalf("md output not from template:\n%s", md)
	}
	html, err := os.ReadFile(filepath.Join(dir, "doc", "m.html"))
	if err != nil {
		t.Fatalf("read doc/m.html: %v", err)
	}
	if !strings.Contains(string(html), "<article") || !strings.Contains(string(html), "<code>hello") {
		t.Fatalf("html output not from template:\n%s", html)
	}
}

// mustDocDict builds the template input dict for doc, failing the test on error.
// Snippet results are not run here (run=false).
func mustDocDict(t *testing.T, doc *gadbridge.DocData, path, src string) gad.Dict {
	t.Helper()
	d, err := buildDocDict(doc, path, []byte(src), sourceTypeFor(path), false)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// readSampleTemplate returns the embedded default doc-template bytes for name
// (md.gadx or html.gadx) — the same templates baked into the binary.
func readSampleTemplate(t *testing.T, name string) []byte {
	t.Helper()
	switch name {
	case gadconfig.DocMDTemplateName:
		return defaultDocTemplateMD
	case gadconfig.DocHTMLTemplateName:
		return defaultDocTemplateHTML
	}
	t.Fatalf("unknown template %s", name)
	return nil
}
