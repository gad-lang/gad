package gadbridge

import (
	"strings"
	"testing"
)

// A tag whose attributes do not fit on its line is written with them
// wrapped, one group a line — its static classes and its repeat (`(N)`) kept
// on its head, as when it fits: they were dropped.
func TestWrappedTagKeepsClassesAndRepeat(t *testing.T) {
	long := strings.Repeat("x", 60)
	src := "@main\n\tdiv.card.mt-2[title=\"" + long + "\", data-a=\"" + long + "\"](3) text\n"
	res := FormatGadx(src, GadxFormatOptions{Indent: "\t"})
	if !res.OK {
		t.Fatalf("format failed: %v", res)
	}
	if !strings.Contains(res.Source, "div.card.mt-2[\n") {
		t.Errorf("the static classes were dropped:\n%s", res.Source)
	}
	if !strings.Contains(res.Source, "](3)") {
		t.Errorf("the repeat was dropped:\n%s", res.Source)
	}
	if strings.Count(res.Source, "\n") < 4 {
		t.Errorf("the attributes were not wrapped:\n%s", res.Source)
	}
	again := FormatGadx(res.Source, GadxFormatOptions{Indent: "\t"})
	if !again.OK || again.Source != res.Source {
		t.Errorf("formatting must be idempotent:\n%s\n---\n%s", res.Source, again.Source)
	}
}
