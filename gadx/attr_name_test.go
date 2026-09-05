package gadx

import (
	"strings"
	"testing"
)

// An attribute name that the bare rules would cut short is written in quotes.
// It used to be dropped: the entry parsed to nothing, so the attribute never
// reached the page and nothing was said about it.
func TestQuotedAttributeName(t *testing.T) {
	for name, c := range map[string]struct{ tpl, want string }{
		"punctuation the bare form stops at": {
			"@main\n\tdiv[\"x/y\"=\"1\", title=\"t\"] a\n",
			`<div x/y="1" title="t">a</div>`,
		},
		"quoted flag": {
			"@main\n\tdiv[\"x/y\"] a\n",
			`<div x/y>a</div>`,
		},
		// A name the bare rules read whole means the same either way.
		"quoted name that needs no quotes": {
			"@main\n\tdiv[\"v-model\"=\"form.name\"] a\n",
			`<div v-model="form.name">a</div>`,
		},
	} {
		out, err := portRun(t, c.tpl, nil, nil)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if !strings.Contains(out, c.want) {
			t.Errorf("%s:\n got %s\nwant %s", name, out, c.want)
		}
	}
}

// `tag[…]="x"` — the value written outside the group — is a mistake, and it
// used to reach the code writer as an assignment with no target and crash it.
// It reads as text now, which is what the line says once the group has closed.
func TestAssignmentNeedsATarget(t *testing.T) {
	out, err := portRun(t, "@main\n\tspan[\"v-text\"]=\"x\"\n", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "=&#34;x&#34;") {
		t.Errorf("the stray value should read as text, got: %s", out)
	}
}
