package gadx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	gad "github.com/gad-lang/gad"
)

// renderSrc writes src to a temp .gadx file and renders it with globals.
func renderSrc(t *testing.T, src string, globals gad.Dict) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "e.gadx")
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := renderString(newTestRender(t, dir), p, globals)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	return out
}

// TestEscapeTextInterpolation checks that an interpolated value in text content
// is HTML-escaped, so untrusted data cannot inject markup (XSS). A RawStr (via
// `raw`) is written verbatim.
func TestEscapeTextInterpolation(t *testing.T) {
	// Untrusted value with the HTML metacharacters.
	got := renderSrc(t, "@global x\n@main\n    p {= x }\n", gad.Dict{"x": gad.Str(`<b>&"'z`)})
	want := `<p>&lt;b&gt;&amp;&#34;&#39;z</p>`
	if got != want {
		t.Fatalf("text not escaped:\n got %q\nwant %q", got, want)
	}
	if strings.Contains(got, "<b>") {
		t.Fatalf("unescaped markup leaked into text: %q", got)
	}

	// `raw` opts out: trusted HTML is written verbatim.
	rawGot := renderSrc(t, "@global x\n@main\n    p {= raw x }\n", gad.Dict{"x": gad.Str(`<b>ok</b>`)})
	if rawGot != `<p><b>ok</b></p>` {
		t.Fatalf("raw value should be verbatim, got %q", rawGot)
	}
}

// TestEscapeAttributeValue checks that an interpolated attribute value is
// HTML-entity-escaped (default AttrQuoteHTML), so it cannot break out of the
// quoted attribute.
func TestEscapeAttributeValue(t *testing.T) {
	got := renderSrc(t, "@global x\n@main\n    a[title=x] hi\n", gad.Dict{"x": gad.Str(`"><script>`)})
	want := `<a title="&#34;&gt;&lt;script&gt;">hi</a>`
	if got != want {
		t.Fatalf("attribute not escaped:\n got %q\nwant %q", got, want)
	}
	if strings.Contains(got, "<script>") {
		t.Fatalf("attribute value broke out: %q", got)
	}
}

// TestAttrQuoteSingleQuote checks the VueJS-friendly attribute quoter: values are
// single-quoted and only `'` is escaped, so double quotes and operators in
// framework expressions survive.
func TestAttrQuoteSingleQuote(t *testing.T) {
	prev := AttrValueQuote
	AttrValueQuote = AttrQuoteSingleQuote
	defer func() { AttrValueQuote = prev }()

	got := renderSrc(t, "@global x\n@main\n    div[class=x] hi\n", gad.Dict{"x": gad.Str(`{ active: ok && ready }`)})
	want := `<div class='{ active: ok && ready }'>hi</div>`
	if got != want {
		t.Fatalf("single-quote attr:\n got %q\nwant %q", got, want)
	}
	// A single quote in the value is escaped so it cannot close the attribute.
	got2 := renderSrc(t, "@global x\n@main\n    div[title=x] hi\n", gad.Dict{"x": gad.Str(`it's`)})
	if !strings.Contains(got2, `title='it&#39;s'`) {
		t.Fatalf("single quote not escaped: %q", got2)
	}
}
