package gadbridge

import (
	"strings"
	"testing"

	"github.com/gad-lang/gad"
)

// `@type NAME` in a file's prose is the builtin type NAME as a symbol of the
// doc object: its members discovered, each in its group; the line leaves the
// prose.
func TestDocTypeDirective(t *testing.T) {
	d, err := ExtractDoc("/**\n# uint\n\nAn unsigned integer.\n\n@type uint\n**/\n\nx := 1\n", "gad")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(d.Prose, "@type") || !strings.Contains(d.Prose, "An unsigned integer.") {
		t.Fatalf("prose %q", d.Prose)
	}
	if len(d.Sections) == 0 || d.Sections[0].Title != "Type" || d.Sections[0].Symbols[0].Name != "uint" {
		t.Fatalf("sections %+v", d.Sections)
	}
	groups := map[string]int{}
	for _, m := range d.Sections[0].Symbols[0].Members {
		groups[m.Group]++
	}
	for _, g := range []string{"Constructors", "Operators", "Conversions"} {
		if groups[g] == 0 {
			t.Errorf("no %s: %v", g, groups)
		}
	}
	// in the doc dict the templates take: the members with their group
	sec := d.GadDict()["sections"].(gad.Array)[0].(gad.Dict)
	sym := sec["symbols"].(gad.Array)[0].(gad.Dict)
	m := sym["members"].(gad.Array)[0].(gad.Dict)
	if m["group"] != gad.Str("Constructors") {
		t.Fatalf("first member %v", m)
	}
}
