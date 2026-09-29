package gadx

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gad-lang/gad"
)

// An error in a component a template imports comes out of Render wrapped
// ("render main.gad: …"): humanized, it still says what wraps it, what it is,
// and where — the component's line, the call in the template — with the
// lines around it.
func TestHumanizeWrappedRuntimeError(t *testing.T) {
	dir := t.TempDir()
	for f, src := range map[string]string{
		"main.gad":          "layouts := import(\"layouts/*.gadx\")::dict\nreturn layouts[\"page\"].main(;x=1)",
		"layouts/page.gadx": "@export comp main(;x=nil)\n    ~ a := nil\n    div\n        | ok\n        span {= a.y }\n",
	} {
		p := filepath.Join(dir, f)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	err := NewRender(dir).Render(io.Discard, filepath.Join(dir, "main.gad"), gad.Dict{})
	if err == nil {
		t.Fatal("no error")
	}
	var b strings.Builder
	(&gad.ErrorHumanizing{}).Humanize(&b, err)
	out := b.String()
	for _, want := range []string{
		"render " + filepath.Join(dir, "main.gad") + ":", // what wraps it
		"NotIndexableError: nil",                         // what it is
		"layouts/page.gadx:5:17",                         // where: the component
		"span {= a.y }",                                  // … its line
		"(main):2:28",                                    // the call in the template
	} {
		if !strings.Contains(out, want) {
			t.Errorf("no %q in\n%s", want, out)
		}
	}
	if strings.HasPrefix(out, "ERROR:") {
		t.Errorf("humanized as an unknown error:\n%s", out)
	}
}
