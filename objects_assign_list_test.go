package gad_test

import (
	"testing"

	gad "github.com/gad-lang/gad"
)

// runSrc compiles and runs src, returning the result string.
func runSrc(t *testing.T, src string) string {
	t.Helper()
	b := gad.NewBuiltins()
	st := gad.NewSymbolTable(b.NameSet)
	_, bc, err := gad.Compile(st, []byte(src), gad.CompileOptions{})
	if err != nil {
		t.Fatalf("compile %q: %v", src, err)
	}
	ret, err := gad.NewVM(b.Build(), bc).SetRecover(true).RunOpts(&gad.RunOpts{})
	if err != nil {
		t.Fatalf("run %q: %v", src, err)
	}
	if ret == nil {
		return "nil"
	}
	return ret.ToString()
}

// TestAssignCommaList covers `Target OP a, b, *rest` == `Target OP [a, b, *rest]`
// for every assignment operator (single target), while multi-target
// destructuring is preserved.
func TestAssignCommaList(t *testing.T) {
	cases := []struct{ src, want string }{
		// Single target, comma-separated values become an array literal.
		{`x := 1, 2, 3; return x`, "[1, 2, 3]"},
		{`o := [9]; x := 1, 2, *o; return x`, "[1, 2, 9]"},
		{`x := 0; x = 1, 2, 3; return x`, "[1, 2, 3]"},
		// Compound operators wrap the comma list too.
		{`a := []; a ++= 2, 3; return a`, "[2, 3]"},
		{`o := [4, 5]; a := [1]; a ++= 2, 3, *o; return a`, "[1, 2, 3, 4, 5]"},
		// Multi-target destructuring is untouched.
		{`a, b := 1, 2; return [a, b]`, "[1, 2]"},
		{`a, b := [10, 20]; return [a, b]`, "[10, 20]"},
		{`o := [3, 4]; a, *rest := 1, 2, *o; return [a, rest]`, "[1, [2, 3, 4]]"},
	}
	for _, c := range cases {
		if got := runSrc(t, c.src); got != c.want {
			t.Errorf("%s => %s, want %s", c.src, got, c.want)
		}
	}
}

// TestArrayAppendOps covers the array append operators: `+`/`+=` (append a
// single element), `++`/`++=` (spread-extend with an iterable).
func TestArrayAppendOps(t *testing.T) {
	cases := []struct{ src, want string }{
		// Binary + appends a single element; a non-iterable joins as one item,
		// an iterable is concatenated.
		{`return [] + 1`, "[1]"},
		{`return [1] + [2]`, "[1, 2]"},
		// Binary ++ requires an iterable and flattens it.
		{`return [] ++ [1, 2]`, "[1, 2]"},
		{`return [1] ++ [2, 3]`, "[1, 2, 3]"},
		{`o := [3, 4]; return [1] ++ o`, "[1, 3, 4]"},
		// += appends one element (an array joins as a single nested element).
		{`a := []; a += 1; return a`, "[1]"},
		{`a := []; a += [1, 2]; return a`, "[[1, 2]]"},
		// ++= spread-extends.
		{`a := []; a ++= [2, 3]; return a`, "[2, 3]"},
		// The requested combined example.
		{`a := []; a += 1; a ++= [2, 3]; return a == [1, 2, 3]`, "true"},
	}
	for _, c := range cases {
		if got := runSrc(t, c.src); got != c.want {
			t.Errorf("%s => %s, want %s", c.src, got, c.want)
		}
	}
}
