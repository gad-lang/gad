package gadbridge

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDocEncodeSnippets(t *testing.T) {
	src := "/***\n# Demo\n\n@snippet greet\n***/\n\n" +
		"//snippet base\nname := \"Gad\"\n//endsnippet\n\n" +
		"//snippet greet uses base\ngreet := \"hi \" + name\ngreet\n/**= \"hi Gad\" **/\n//endsnippet\n"

	js, err := DocEncode(src, "gad", "json")
	if err != nil {
		t.Fatal(err)
	}
	var got DocEncoded
	if err := json.Unmarshal([]byte(js), &got); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, js)
	}
	if !strings.Contains(got.Prose, "# Demo") {
		t.Fatalf("prose missing: %q", got.Prose)
	}
	if len(got.Snippets) != 2 {
		t.Fatalf("want 2 snippets, got %d", len(got.Snippets))
	}
	by := map[string]DocSnippet{}
	for _, s := range got.Snippets {
		by[s.Name] = s
	}
	if g := by["greet"]; len(g.Uses) != 1 || g.Uses[0] != "base" || g.Kind != "value" ||
		g.Expected != `"hi Gad"` || g.Result != "hi Gad" {
		t.Fatalf("greet snippet: %+v", g)
	}

	yml, err := DocEncode(src, "gad", "yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(yml, "uses:") || !strings.Contains(yml, "greet") {
		t.Fatalf("yaml missing snippet fields:\n%s", yml)
	}
}
