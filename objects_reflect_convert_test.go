package gad_test

import (
	"strings"
	"testing"

	"github.com/gad-lang/gad"
)

func runReflect(t *testing.T, src string, globals gad.Dict) (gad.Object, error) {
	t.Helper()
	st := gad.NewSymbolTable(gad.NewBuiltins().NameSet)
	for name := range globals {
		st.DefineGlobal(name)
	}
	res, err := gad.Compile(st, []byte(src), gad.CompileOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return gad.NewVM(gad.NewBuiltins().Build(), res.Bytecode).RunOpts(&gad.RunOpts{Globals: globals})
}

// A map's keys and values converted to its Go types (an int of gad set in a
// map[string]int panicked the host); a value not convertible is an error of
// the script, not a panic.
func TestReflectMapConversions(t *testing.T) {
	byID := map[int]string{1: "a"}
	counts := map[string]int{}
	ret, err := runReflect(t, `global (byID, counts)
byID[2] = "b"
counts["x"] = 3
return [byID[1], byID[2], byID["no"], counts.x]`, gad.Dict{
		"byID":   gad.MustNewReflectValue(byID),
		"counts": gad.MustNewReflectValue(counts),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := ret.ToString(); got != `["a", "b", nil, 3]` {
		t.Fatalf("got %s", got)
	}
	if byID[2] != "b" || counts["x"] != 3 {
		t.Fatalf("the Go maps: %v %v", byID, counts)
	}
	_, err = runReflect(t, `global counts; counts["y"] = [1]`, gad.Dict{"counts": gad.MustNewReflectValue(counts)})
	if err == nil || !strings.Contains(err.Error(), "should be int") {
		t.Fatalf("want a conversion error, got %v", err)
	}
}

// A gad function for a Go function type with no error result: its error is
// the Go call's (recovered), not a crash.
func TestReflectCallbackErrorWithoutErrorResult(t *testing.T) {
	each := func(xs []int, f func(int)) {
		for _, x := range xs {
			f(x)
		}
	}
	_, err := runReflect(t, `global each; each([1, 2], (x) => { throw "stop" })`,
		gad.Dict{"each": gad.MustNewReflectValue(each)})
	if err == nil || !strings.Contains(err.Error(), "stop") {
		t.Fatalf("want the callback's error, got %v", err)
	}
}
