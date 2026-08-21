package parser

import (
	"strings"
	"testing"
)

// TestPipeBlock checks the YAML-style `|` block: a bare `|` opens a literal text
// block whose indented lines need no per-line `| ` prefix, and it round-trips.
func TestPipeBlock(t *testing.T) {
	src := "@main\n" +
		"    div\n" +
		"        |\n" +
		"            Hello {= name }\n" +
		"            second line\n"

	out := transpileGadx(t, src)
	// The block marker is a bare `|`, and its body lines carry no `| ` prefix.
	if !strings.Contains(out, "\t\t|\n") {
		t.Fatalf("expected a bare `|` block opener:\n%s", out)
	}
	if !strings.Contains(out, "Hello {= name }") || strings.Contains(out, "| Hello") {
		t.Fatalf("body lines must be verbatim (no `| ` prefix):\n%s", out)
	}

	// Idempotent: re-formatting the output yields the same text.
	if again := transpileGadx(t, out); again != out {
		t.Fatalf("not idempotent:\n--- first ---\n%s\n--- second ---\n%s", out, again)
	}
}

// TestPipeFoldBlock checks the folded `|>` block: its marker is preserved (so
// literal `|` and folded `|>` stay distinct) and it round-trips.
func TestPipeFoldBlock(t *testing.T) {
	src := "@main\n" +
		"    div\n" +
		"        |>\n" +
		"            line one\n" +
		"            line two\n"

	out := transpileGadx(t, src)
	if !strings.Contains(out, "\t\t|>\n") {
		t.Fatalf("expected a `|>` folded block opener:\n%s", out)
	}
	if again := transpileGadx(t, out); again != out {
		t.Fatalf("not idempotent:\n--- first ---\n%s\n--- second ---\n%s", out, again)
	}
}
