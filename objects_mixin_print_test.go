package gad_test

import (
	"strings"
	"testing"
)

// A mixin printed (str, repr, in an array) — its ToString printed through
// itself: a stack overflow, fatal to the process.
func TestMixinPrint(t *testing.T) {
	ret, err := runReflect(t, `mixin A { x = 1 }
mixin M { *A; methods { hi() => 1 } }
return [str(M), repr(M), str([M])]`, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"‹mixin ", "M: {", "hi:", "parents:", "A: {fields: {x:"} {
		if !strings.Contains(ret.ToString(), want) {
			t.Fatalf("no %q in %s", want, ret)
		}
	}
}
