package gadbridge

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDocEncodeSnippets(t *testing.T) {
	src := "/***\n# Demo\n\n@snippet greet\n***/\n\n" +
		"example base {\n    name := \"Gad\"\n}\n\n" +
		"example greet {\n    greet := \"hi \" + name\n    println(greet)\n}\n<<< hi Gad\n"

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
	if g := by["greet"].Code; g != "greet := \"hi \" + name\nprintln(greet)\n<<< hi Gad" {
		t.Fatalf("greet snippet: %q", g)
	}

	yml, err := DocEncode(src, "gad", "yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(yml, "greet") || !strings.Contains(yml, "code:") {
		t.Fatalf("yaml missing snippet fields:\n%s", yml)
	}
}

func TestSnippetsGadx(t *testing.T) {
	src := "@example hello\n\tdiv hi\n\t~ x := 2\n\n<<< <div>hi</div>\n\n@example list\n\tul\n\t\tli a\n\n<<<\n\t<ul><li>a</li></ul>\n\np end\n"
	got := Snippets([]byte(src))
	if len(got) != 2 {
		t.Fatalf("want 2 snippets, got %+v", got)
	}
	if got[0].Name != "hello" || got[0].Code != "div hi\n~ x := 2\n\n<<< <div>hi</div>" {
		t.Fatalf("hello: %q", got[0].Code)
	}
	if got[1].Name != "list" || got[1].Code != "ul\n\tli a\n\n<<<\n\t<ul><li>a</li></ul>" {
		t.Fatalf("list: %q", got[1].Code)
	}
}
