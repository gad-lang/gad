package gadx_test

import (
	"strings"
	"testing"

	gadxnode "github.com/gad-lang/gad/gadx/node"
	gadxparser "github.com/gad-lang/gad/gadx/parser"
	gnode "github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
)

// codeOf parses gadx source and returns the transpiled Gad code.
func codeOf(t *testing.T, src string) string {
	t.Helper()
	fs := source.NewFileSet()
	f := fs.AddFileData("t.gadx", -1, []byte(src))
	parsed, err := gadxparser.NewParser(f).ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return gnode.Code(gadxnode.Convert(parsed.Stmts))
}

// TestTextMergeBlock: a folded/literal text block coalesces its lines (literals
// and their separators) into a single gadx.Text(tag, …) call.
func TestTextMergeBlock(t *testing.T) {
	code := codeOf(t, "@main\n    p\n        |>\n            one\n            two\n            three\n")
	if want := `gadx.Text(tag, "one", " ", "two", " ", "three")`; !strings.Contains(code, want) {
		t.Fatalf("folded block should merge into one Text call\n got: %s\nwant substring: %s", code, want)
	}
}

// TestTextMergeExprAndText: a `| ` line merges literal text AND `{= expr }`
// interpolations into one gadx.Text call (multiple args).
func TestTextMergeExprAndText(t *testing.T) {
	code := codeOf(t, "@global a\n@global b\n@main\n    p\n        | x {= a } y {= b } z\n")
	if want := `gadx.Text(tag, "x ", a, " y ", b, " z")`; !strings.Contains(code, want) {
		t.Fatalf("text+expr should merge into one Text call\n got: %s\nwant substring: %s", code, want)
	}
}

// TestTextMergeControlSplits: a bare `{ expr }` control statement flushes the
// merged Text call and is emitted on its own, then merging resumes.
func TestTextMergeControlSplits(t *testing.T) {
	code := codeOf(t, "@global a\n@main\n    p\n        | x {= a }\n        { a }\n        | z {= a }\n")
	for _, want := range []string{`gadx.Text(tag, "x ", a`, `gadx.Text(tag, "z ", a)`} {
		if !strings.Contains(code, want) {
			t.Fatalf("control should split the merged Text calls\n got: %s\nwant substring: %s", code, want)
		}
	}
}
