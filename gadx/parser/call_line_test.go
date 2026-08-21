package parser

import (
	"strings"
	"testing"
)

// TestCallLine checks the `! callee arg1 arg2 …` fluent call statement: the
// space-separated parts (respecting balanced brackets/quotes) become the callee
// and its arguments, and it round-trips.
func TestCallLine(t *testing.T) {
	src := "@test t\n" +
		"    ! t.equal render(list([\"a\", \"b\"])) \"<ul>\"\n" +
		"    ! t.nil nil\n"

	out := transpileGadx(t, src)
	for _, want := range []string{
		`! t.equal render(list(["a", "b"])) "<ul>"`, // both args stay balanced
		`! t.nil nil`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	// The `! ` prefix is not emitted as literal text.
	if strings.Contains(out, "| !") {
		t.Fatalf("call line emitted as text:\n%s", out)
	}
	// Idempotent.
	if again := transpileGadx(t, out); again != out {
		t.Fatalf("not idempotent:\n--- first ---\n%s\n--- second ---\n%s", out, again)
	}
}
