package gad_test

import (
	"os"
	"testing"

	. "github.com/gad-lang/gad"
)

// Type parameters of functions, closures, class methods and properties (issue
// #23): testdata/type_params_test.gad, its checks on.
func TestVMTypeParams(t *testing.T) {
	src, err := os.ReadFile("testdata/type_params_test.gad")
	if err != nil {
		t.Fatal(err)
	}
	testExpectRun(t, string(src), checksOn(), True)
	expectErrHas(t, `func f[T number](v T) => v; f[str]`, newOpts(), `f: type argument T: str does not satisfy number`)
	expectErrHas(t, `func f[T](v T) => v; f[int]("a")`, newOpts(), `expected int, found str`)
	expectErrHas(t, `func f[T](a T, b T) => a; f(1, "a")`, newOpts(), `expected int, found str`)
	expectErrHas(t, `func f[T number](v T) => v; f("a")`, newOpts(), `expected number, found str`)
	expectErrHas(t, `func f[T](v T) => v; f[int, str]`, newOpts(), `has 1 type parameters, given 2 type arguments`)
	expectErrHas(t, `f := func[T](v T) => v; f[1]`, newOpts(), `type argument T: ‹1› is not a type`)
	testExpectRun(t, `return int|float`, newOpts(), NewTypeUnion(TInt, TFloat))
	testExpectRun(t, `return any(5)`, newOpts(), Int(5))
}
