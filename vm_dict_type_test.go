package gad_test

import (
	"os"
	"testing"

	. "github.com/gad-lang/gad"
)

// DictType[K, V], `dict === DictType[str, any]`, typed dict types (issue #19).
// The cases are the gad of testdata/dict_type_test.gad, run with its checks.
func TestVMDictType(t *testing.T) {
	src, err := os.ReadFile("testdata/dict_type_test.gad")
	if err != nil {
		t.Fatal(err)
	}
	testExpectRun(t, string(src), checksOn(), True)
	expectErrHas(t, `type scores dict[str, int]; s := scores(); s.a = "x"`, newOpts(), `scores: value "a": ‹str› is not of ‹int›`)
	expectErrHas(t, `DictType[str, int](; a="x")`, newOpts(), `value "a": ‹str› is not of ‹int›`)
	expectErrHas(t, `enum E {A}; DictType[E, int](; B=1)`, newOpts(), `a key ‹B› is not of ‹E›`)
	expectErrHas(t, `DictType[int, str]`, newOpts(), `a key is a string`)
	expectErrHas(t, `x := {a: "x"} :: DictType[str, int]`, newOpts(), `value "a": ‹str› is not of ‹int›`)
	expectErrHas(t, `DictType[str, int]([1], 2)`, newOpts(), `argument 0: a [key, value] pair, not 1 items`)
}
