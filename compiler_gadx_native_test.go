package gad_test

import (
	"bytes"
	"strings"
	"testing"

	gad "github.com/gad-lang/gad"
	"github.com/gad-lang/gad/gadx"
)

// runScript compiles and runs src (Gad by default; Gadx when gadx is set) with
// the gadx builtins registered, returning the return value and captured stdout.
// When the return value is a gadx.Element (a rendered template tree), its HTML is
// written to the captured stdout so callers can assert on the rendered output.
func runScript(t *testing.T, src string, gadxMode bool) (gad.Object, string) {
	t.Helper()
	builtins := gadx.AppendBuiltins(gad.NewBuiltins())
	st := gad.NewSymbolTable(builtins.NameSet)
	opts := gad.CompileOptions{}
	if gadxMode {
		opts.GadxOptions = &gad.GadxOptions{}
	}
	cr1, err := gad.Compile(st, []byte(src), opts)
	bc := cr1.BC()
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	var out bytes.Buffer
	vm := gad.NewVM(builtins.Build(), bc).SetRecover(true)
	ret, err := vm.RunOpts(&gad.RunOpts{StdOut: &out})
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	if el, ok := ret.(gadx.Element); ok {
		if _, err := el.WriteTo(vm, &out); err != nil {
			t.Fatalf("render element: %v", err)
		}
	}
	return ret, out.String()
}

func TestNativeGadxCompile(t *testing.T) {
	// A .gadx template compiled through gad.CompileOptions.GadxOptions returns a
	// gadx.Tag; runScript writes its rendered HTML into the captured output.
	_, html := runScript(t, `p Hello {= 1 + 2 }`, true)
	if !strings.Contains(html, "Hello") || !strings.Contains(html, "3") {
		t.Fatalf("gadx render = %q, want it to contain Hello and 3", html)
	}
}

func TestGadParseEval(t *testing.T) {
	// gad.parse returns a SourceFileObject carrying .stmts; gad.eval accepts the
	// source string, the SourceFileObject, its .stmts, or a single StmtObject.
	ret, _ := runScript(t, `
		src := gad.parse("return 6 * 7")
		return [gad.eval("return 6 * 7"), gad.eval(src), gad.eval(src.stmts), gad.eval(src.stmts[0])]
	`, false)
	arr, ok := ret.(gad.Array)
	if !ok || len(arr) != 4 {
		t.Fatalf("result = %v (%T), want array of 4", ret, ret)
	}
	for i, v := range arr {
		if v.ToString() != "42" {
			t.Fatalf("eval overload %d = %q, want 42", i, v.ToString())
		}
	}
}

func TestGadParseSourceFile(t *testing.T) {
	// The returned SourceFileObject exposes path/type, indexing (char), slicing
	// (bytes) and bytes() conversion.
	ret, _ := runScript(t, `
		src := gad.parse("abc"; name="x.gadt")
		return [src.path, src.type.name, int(src[0]), str(src[0:2]), str(bytes(src)), len(src)]
	`, false)
	arr, ok := ret.(gad.Array)
	if !ok || len(arr) != 6 {
		t.Fatalf("result = %v (%T), want array of 6", ret, ret)
	}
	if arr[0].ToString() != "x.gadt" {
		t.Fatalf("path = %q, want x.gadt", arr[0].ToString())
	}
	if !strings.Contains(arr[1].ToString(), "TEMPLATE") {
		t.Fatalf("type = %q, want it to contain TEMPLATE (inferred from .gadt)", arr[1].ToString())
	}
	if arr[2].ToString() != "97" { // 'a'
		t.Fatalf("src[0] = %q, want 97", arr[2].ToString())
	}
	if arr[3].ToString() != "ab" {
		t.Fatalf("src[0:2] = %q, want ab", arr[3].ToString())
	}
	if arr[4].ToString() != "abc" || arr[5].ToString() != "3" {
		t.Fatalf("bytes/len = %s/%s, want abc/3", arr[4].ToString(), arr[5].ToString())
	}
}

func TestGadParseGadxEval(t *testing.T) {
	// gad.parse with type=gad.SourceType.GADX returns lowered statements
	// referencing gadx builtins; gad.eval runs them (the VM has the gadx
	// builtins registered) and returns the gadx element, which runScript renders.
	_, html := runScript(t, `
		src := gad.parse("span Hi"; type=gad.SourceType.GADX)
		return gad.eval(src.stmts)
	`, false)
	if !strings.Contains(html, "Hi") {
		t.Fatalf("gadx eval = %q, want it to contain Hi", html)
	}
}

func TestStmtsObjectIterIndex(t *testing.T) {
	// StmtsObject supports len(), iteration and index get.
	ret, _ := runScript(t, `
		src := gad.parse("a := 1\nb := 2\nc := 3")
		stmts := src.stmts
		n := len(stmts)
		first := str(stmts[0])
		count := 0
		for _ in stmts {
			count += 1
		}
		return [n, count, first]
	`, false)
	arr, ok := ret.(gad.Array)
	if !ok || len(arr) != 3 {
		t.Fatalf("result = %v (%T), want array of 3", ret, ret)
	}
	if arr[0].ToString() != "3" || arr[1].ToString() != "3" {
		t.Fatalf("len/iter = %s/%s, want 3/3", arr[0].ToString(), arr[1].ToString())
	}
	if !strings.Contains(arr[2].ToString(), "a") {
		t.Fatalf("stmts[0] = %q, want it to contain 'a'", arr[2].ToString())
	}
}
