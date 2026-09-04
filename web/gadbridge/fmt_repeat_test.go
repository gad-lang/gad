package gadbridge

import (
	"strings"
	"testing"
)

// A run of siblings that write out the same is folded into one `(N)`, and the
// fold has to leave the template itself untouched — the lowering before and
// after must match, or `gad fmt` would be rewriting what the page renders.
func TestFormatFoldsIdenticalSiblings(t *testing.T) {
	src := "@main\n\tdiv\n\t\ta link\n\t\ta link\n\t\ta link\n\t\tp other\n"

	res := FormatGadx(src, GadxFormatOptions{Indent: "\t"})
	if !res.OK {
		t.Fatalf("format failed:\n%s", src)
	}
	if !strings.Contains(res.Source, "a(3) link") {
		t.Errorf("the three links were not folded:\n%s", res.Source)
	}
	if strings.Contains(res.Source, "p(") {
		t.Errorf("a lone sibling was folded:\n%s", res.Source)
	}

	before, ok1 := GadxLowered(src)
	after, ok2 := GadxLowered(res.Source)
	if !ok1 || !ok2 || before != after {
		t.Errorf("folding changed the template (ok1=%v ok2=%v)", ok1, ok2)
	}
}

// Two identical text lines are two lines of text: joining them would change
// what the page says.
func TestFormatKeepsIdenticalTextLines(t *testing.T) {
	src := "@main\n\tdiv\n\t\t| word\n\t\t| word\n"

	res := FormatGadx(src, GadxFormatOptions{Indent: "\t"})
	if !res.OK {
		t.Fatalf("format failed:\n%s", src)
	}
	if strings.Count(res.Source, "| word") != 2 {
		t.Errorf("text lines were folded:\n%s", res.Source)
	}
}
