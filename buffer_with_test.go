package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// TestWithBufferCapture verifies that a `buffer` is a `with` resource: it captures
// the block's print/write output, and `with` yields the buffer holding it.
func TestWithBufferCapture(t *testing.T) {
	testExpectRun(t, `content := with buffer() { print("hello") }; return str(content)`, nil, Str("hello"))
	testExpectRun(t, `return str(with buffer() { print("a"); println("b") })`, nil, Str("ab\n"))
	testExpectRun(t, `return typeName(with buffer() { print("x") })`, nil, Str("buffer"))
}
