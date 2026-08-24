package pluginsync

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTextMateGrammar(t *testing.T) {
	data, err := TextMateGrammar()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	var g map[string]any
	if err := json.Unmarshal(data, &g); err != nil {
		t.Fatalf("grammar is not valid JSON: %v", err)
	}
	if g["scopeName"] != "source.gad" {
		t.Fatalf("scopeName = %v, want source.gad", g["scopeName"])
	}
	// The keyword rule must cover current keywords, including recent ones.
	s := string(data)
	for _, kw := range []string{"with", "ain", "defer_ok", "meti"} {
		if !strings.Contains(s, kw) {
			t.Fatalf("grammar missing keyword %q", kw)
		}
	}
	// Doc-comment scopes must be present.
	for _, scope := range []string{"comment.documentation.block.gad", "comment.documentation.line.gad"} {
		if !strings.Contains(s, scope) {
			t.Fatalf("grammar missing scope %q", scope)
		}
	}
	// Interpolated-string highlighting: the `#`-prefixed forms and the embedded
	// `{ … }` island scope must be present so expressions inside strings colorize.
	for _, scope := range []string{
		"string.quoted.double.interpolated.gad",
		"string.quoted.triple.interpolated.gad",
		"string.quoted.raw.interpolated.gad",
		"meta.interpolation.gad",
	} {
		if !strings.Contains(s, scope) {
			t.Fatalf("grammar missing scope %q", scope)
		}
	}
}
