package gadx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gad-lang/gad"
)

// The AttrHandler of a Render says the value of each attribute written: a
// static file's URI versioned, an attribute dropped; the others, and the
// text, as they are.
func TestRenderAttrHandler(t *testing.T) {
	dir := t.TempDir()
	src := "@main\n" +
		"    link[href=\"/static/css/site.css\", rel=\"stylesheet\"]\n" +
		"    script[src=\"/static/js/main.js\"]\n" +
		"    a[href=\"/about\", data-drop=\"x\"] /static/ in a text\n"
	srcPath := filepath.Join(dir, "page.gadx")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	r := newTestRender(t, dir)
	r.AttrHandler = func(tag, name string, value gad.Object) gad.Object {
		if name == "data-drop" {
			return nil
		}
		if s, ok := value.(gad.Str); ok && (name == "href" || name == "src") && strings.HasPrefix(string(s), "/static/") {
			return gad.Str("/static/abc1234/" + strings.TrimPrefix(string(s), "/static/"))
		}
		return value
	}
	out, err := renderString(r, srcPath, gad.Dict{})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`href="/static/abc1234/css/site.css"`, `rel="stylesheet"`,
		`src="/static/abc1234/js/main.js"`, `href="/about"`, `/static/ in a text`} {
		if !strings.Contains(out, want) {
			t.Errorf("no %s in %s", want, out)
		}
	}
	if strings.Contains(out, "data-drop") {
		t.Errorf("an attribute dropped is written: %s", out)
	}

	// without it, as they are
	r.AttrHandler = nil
	if out, _ = renderString(r, srcPath, gad.Dict{}); !strings.Contains(out, `href="/static/css/site.css"`) {
		t.Errorf("without a handler: %s", out)
	}
}
