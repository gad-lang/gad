package gadbridge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A valueless attribute is the flag `yes`. It renders bare — `<form novalidate>`
// — and the formatter writes it back bare, but the lowered Gad has to name a
// value: a named argument with nothing after its `=` is not Gad, and the code
// came back out unparseable.
func TestFlagAttributeLowersToYes(t *testing.T) {
	src := "@main\n\tform[novalidate, title=\"t\"]\n\t\tspan x\n"

	lowered, ok := GadxLowered(src)
	if !ok {
		t.Fatal("lowering failed")
	}
	if !strings.Contains(lowered, "novalidate=yes") {
		t.Errorf("flag did not lower to a value:\n%s", lowered)
	}

	res := FormatGadx(src, GadxFormatOptions{Indent: "\t"})
	if !res.OK {
		t.Fatal("format failed")
	}
	if !strings.Contains(res.Source, "[novalidate, title=\"t\"]") {
		t.Errorf("the flag should be written back bare:\n%s", res.Source)
	}
}

// `x=yes` says what a bare `x` says, so that is how it is written back. `x=no`
// is not folded: it omits the attribute, the opposite of what bare means.
func TestExplicitFlagIsWrittenBare(t *testing.T) {
	res := FormatGadx("@main\n\tdiv[a=yes] 1\n\tdiv[a=no] 2\n", GadxFormatOptions{Indent: "\t"})
	if !res.OK {
		t.Fatal("format failed")
	}
	if !strings.Contains(res.Source, "div[a] 1") {
		t.Errorf("`a=yes` was not written bare:\n%s", res.Source)
	}
	if !strings.Contains(res.Source, "div[a=no] 2") {
		t.Errorf("`a=no` must keep its value:\n%s", res.Source)
	}
}

// Every sample has to survive being lowered and read back. The flag attribute
// above broke exactly that, and `samples/gadx/boolean_attribute.gadx` — the
// sample documenting the feature — went unparseable with nothing to catch it:
// the samples are run, but nothing transpiled them.
func TestSamplesLowerAndReparse(t *testing.T) {
	dir := filepath.Join("..", "..", "samples", "gadx")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".gadx" {
			continue
		}
		t.Run(e.Name(), func(t *testing.T) {
			b, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := GadxLowered(string(b)); !ok {
				t.Error("does not lower and reparse")
			}
		})
	}
}
