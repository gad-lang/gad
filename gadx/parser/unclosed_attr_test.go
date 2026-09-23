package parser

import (
	"testing"
	"time"

	"github.com/gad-lang/gad/parser/source"
)

// TestUnclosedAttributeTerminates guards against the scanner hang hit while
// typing `script[t` in an editor: an attribute `[` never closed before EOF made
// Scan return an Illegal token without consuming, so the parser re-scanned the
// same buffer forever (the IDE's `gad complete` spun at 100% CPU). Parsing must
// now finish promptly (with or without an error — the text is incomplete).
func TestUnclosedAttributeTerminates(t *testing.T) {
	for _, src := range []string{
		"script[t",
		"div\n    script[t\n    p hello\n",
		"@main\n    div\n        script[t\n        p x\n",
	} {
		done := make(chan struct{})
		go func() {
			defer close(done)
			fs := source.NewFileSet()
			f := fs.AddFileData("test.gadx", -1, []byte(src))
			_, _ = NewParser(f).ParseFile()
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatalf("parser hung on an unclosed attribute `[` in %q", src)
		}
	}
}
