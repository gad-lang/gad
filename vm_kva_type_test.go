package gad_test

import (
	"os"
	"testing"

	. "github.com/gad-lang/gad"
)

// KeyValueArrayType[K, V], `keyValueArray === KeyValueArrayType[any, any]`,
// typed key-value arrays (issue #20): testdata/kva_type_test.gad, its checks on.
func TestVMKeyValueArrayType(t *testing.T) {
	src, err := os.ReadFile("testdata/kva_type_test.gad")
	if err != nil {
		t.Fatal(err)
	}
	testExpectRun(t, string(src), checksOn(), True)
	expectErrHas(t, `KeyValueArrayType[str, int](; a="x")`, newOpts(), `pair 0 (a): ‹str› is not of ‹int›`)
	expectErrHas(t, `type o keyValueArray[str, int]; x := o(; a=1) + [b="x"]`, newOpts(), `o: pair 1 (b): ‹str› is not of ‹int›`)
	expectErrHas(t, `x := (;a="x") :: KeyValueArrayType[str, int]`, newOpts(), `is not assignable to ‹KeyValueArrayType[str, int]›`)
}
