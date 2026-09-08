package gadbridge

import (
	"strings"
	"testing"
)

// The formatter writes a tag's classes in order. Which classes an element
// carries is what the attribute means; the order they are written in is not,
// since CSS resolves by stylesheet order and never by the attribute's.
func TestFormatterSortsClasses(t *testing.T) {
	for name, c := range map[string]struct{ src, want string }{
		"plain": {
			"@main\n\tdiv.b.a x\n",
			"div.a.b x",
		},
		"id keeps its place ahead of the classes": {
			"@main\n\tp#x.z.a.m x\n",
			"p#x.a.m.z x",
		},
		"quoted names sort by the name, not by the quote": {
			"@main\n\tspan.\"md:flex\".flex.\"hover:x\" y\n",
			"span.flex.\"hover:x\".\"md:flex\" y",
		},
		"several class attributes are one run": {
			"@main\n\tdiv[class=\"b\", class=\"a\"] x\n",
			"div.a.b x",
		},
	} {
		res := FormatGadx(c.src, GadxFormatOptions{Indent: "\t"})
		if !res.OK {
			t.Errorf("%s: %v", name, res.Diagnostics)
			continue
		}
		if !strings.Contains(res.Source, c.want) {
			t.Errorf("%s:\n got %q\nwant %q", name, res.Source, c.want)
		}
	}
}

// Sorting must not make the file look changed to the safety net that compares
// what the two versions lower to, or `gad fmt` would refuse to write it.
func TestSortedClassesLowerTheSame(t *testing.T) {
	before, ok1 := GadxLowered("@main\n\tdiv.b.a x\n")
	after, ok2 := GadxLowered("@main\n\tdiv.a.b x\n")
	if !ok1 || !ok2 {
		t.Fatal("lowering failed")
	}
	if before != after {
		t.Errorf("class order changed the lowered code:\n%s\n---\n%s", before, after)
	}
}

// The normalization is about order alone: a class that is not there is a
// different element, and must still be reported as one.
func TestDifferentClassesStillDiffer(t *testing.T) {
	before, _ := GadxLowered("@main\n\tdiv.a.b x\n")
	after, _ := GadxLowered("@main\n\tdiv.a.c x\n")
	if before == after {
		t.Error("a different class set was normalized away")
	}
}
