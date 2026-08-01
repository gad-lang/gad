package gad_test

import (
	gad "github.com/gad-lang/gad"
	"testing"
)

func TestSelfAssignN(t *testing.T) {
	b := gad.NewBuiltins()
	run := func(src string) gad.Object {
		st := gad.NewSymbolTable(b.NameSet)
		_, bc, err := gad.Compile(st, []byte(src), gad.CompileOptions{CompilerOptions: gad.DefaultCompilerOptions})
		if err != nil {
			t.Fatalf("compile %q: %v", src, err)
		}
		r, err := gad.NewVM(b.Build(), bc).SetRecover(true).RunOpts(&gad.RunOpts{})
		if err != nil {
			t.Fatalf("run %q: %v", src, err)
		}
		return r
	}
	cases := []struct{ src, want string }{
		{`a := []; a ++= 1, 2, 3; return a`, "[1, 2, 3]"},
		{`a := [1]; a ++= 2, 3; return a`, "[1, 2, 3]"},
		{`a := []; a ++= [4, 5]; return a`, "[4, 5]"}, // array literal RHS
		{`a := [1]; a ++= 2, 3; return a == [1,2,3]`, "true"},
		// borrow must not corrupt: after the ++=, allocate more and check a intact
		{`a := []; a ++= 1, 2, 3; b := [9,9,9]; c := [8,8]; return a`, "[1, 2, 3]"},
		// elements survive as independent values (not aliased to reused stack)
		{`a := []; a ++= 10, 20; d := [0,0,0,0,0]; return [a[0], a[1]]`, "[10, 20]"},
		// nested arrays as elements are preserved
		{`a := []; a ++= [1], [2]; return a`, "[[1], [2]]"},
	}
	for _, c := range cases {
		if got := run(c.src).ToString(); got != c.want {
			t.Errorf("%s => %s, want %s", c.src, got, c.want)
		}
	}
}
