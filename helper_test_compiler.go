package gad

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	testhelper "github.com/gad-lang/gad/test_helper/teststrings"
	"github.com/gad-lang/gad/tests"
)

func TestBytecodesEqual(t *testing.T,
	expected, got *Bytecode, checkSourceMap bool) {
	repr := func(v Object) string {
		opts := make(PrinterStateOptions)
		opts.WithIndent()

		if o, err := ToRepr(nil, v, opts); err != nil {
			panic(err)
		} else {
			return o.ToString()
		}
	}

	compareCf := func(ef, gf *CompiledFunction, indent bool, msg string) {
		t.Helper()

		doIndent := func(builder *bytes.Buffer) {
			lines := strings.Split(builder.String(), "\n")
			builder.Reset()
			if len(lines) > 0 {
				builder.WriteString("  ")
				builder.WriteString(strings.Join(lines, "\n  "))
			}
		}

		if !assertCompiledFunctionsEqual(t, ef, gf, checkSourceMap) {
			var ebuf bytes.Buffer
			ef.Fprint(nil, &ebuf, expected)

			var gbuf bytes.Buffer
			gf.Fprint(nil, &gbuf, got)

			if indent {
				doIndent(&ebuf)
				doIndent(&gbuf)
			}

			t.Fatalf("%s not equal\n\nExpected: %s\n\nGot: %s\n", msg, ebuf.String(), gbuf.String())
		}
	}

	t.Helper()

	if expected.NumModules != got.NumModules {
		t.Fatalf("NumModules not equal expected %d, got %d\n",
			expected.NumModules, got.NumModules)
	}

	if len(expected.Constants) == 0 {
		expected.Constants = append(expected.Constants, &Module{info: ModuleInfo{Name: MainName}})
	} else {
		m, _ := expected.Constants[0].(*Module)
		if m == nil || m.Name() != MainName {
			expected.Constants = append(Array{&Module{info: ModuleInfo{Name: MainName}}}, expected.Constants...)
		}
	}

	if len(expected.Constants) != len(got.Constants) {
		t.Fatalf("Constants len not equal\n"+
			"Expected len(Constants):\n%d\nGot len(Constants):\n%d\n"+
			"Expected Dump:\n%s\nGot Dump:\n%s\n",
			len(expected.Constants), len(got.Constants), repr(Array(expected.Constants)), repr(Array(got.Constants)))
	}

	compareCf(expected.Main, got.Main, false, "Main functions")

	for i, gotObj := range got.Constants {
		expectObj := expected.Constants[i]

		switch g := gotObj.(type) {
		case *CompiledFunction:
			ef, ok := expectObj.(*CompiledFunction)
			if !ok {
				t.Fatalf("%T expected at index %d but got %T",
					expectObj, i, gotObj)
			}

			compareCf(ef, g, true, fmt.Sprintf("Contants[%d]: CompiledFunctions", i))
			continue
		case *Module:
			if am, _ := gotObj.(*Module); am != nil {
				if !reflect.DeepEqual(g.info, am.info) {
					t.Fatalf("Constants not equal at %d (*Module.info)\nExpected:\n%v (%[2]T)\nGot:\n%v (%[3]T)\n",
						i, g.info, am.info)
				}

				if !reflect.DeepEqual(g.data, am.data) {
					t.Fatalf("Constants not equal at %d (*Module.dict)\nExpected:\n%v (%[2]T)\nGot:\n%v (%[3]T)\n",
						i, g.data, am.data)
				}

				if ei, _ := g.init.(*CompiledFunction); ei != nil {
					if ai, _ := am.init.(*CompiledFunction); ai != nil {
						if !assertCompiledFunctionsEqual(t,
							ei, ei, checkSourceMap) {
							t.Fatalf("Constants not equal at %d (*Module.init)\nExpected:\n%v (%[2]T)\nGot:\n%v (%[3]T)\n",
								i, ei, ai)
						}
					}
				}
				continue
			}
		}

		if !reflect.DeepEqual(expectObj, gotObj) {
			t.Fatalf("Constants not equal at %d\nExpected:\n%v (%[2]T)\nGot:\n%v (%[3]T)\n",
				i, repr(expectObj), repr(gotObj))
		}
	}
}

func assertCompiledFunctionsEqual(t *testing.T,
	expected, got *CompiledFunction, checkSourceMap bool) bool {
	t.Helper()
	if expected.Params.String() != got.Params.String() {
		t.Errorf("Params not equal expected (%s), got (%s)\n",
			expected.Params.String(), got.Params.String())
		return false
	}
	if expected.NamedParams.String() != got.NamedParams.String() {
		t.Errorf("NamedParams not equal expected (;%s), got (;%s)\n",
			expected.NamedParams.String(), got.NamedParams.String())
		return false
	}
	if expected.NumLocals != got.NumLocals {
		t.Errorf("NumLocals not equal expected %d, got %d\n",
			expected.NumLocals, got.NumLocals)
		return false
	}
	if string(expected.Instructions) != string(got.Instructions) {
		var eb, gb bytes.Buffer
		expected.FprintLP(nil, nil, "\t", &eb)
		got.FprintLP(nil, nil, "\t", &gb)
		testhelper.EqualStringf(t, eb.String(), gb.String(), "Instructions not equal")
	}
	if len(expected.Free) != len(got.Free) {
		t.Errorf("Free not equal expected %d, got %d\n",
			len(expected.Free), len(got.Free))
		return false
	}
	if checkSourceMap &&
		!reflect.DeepEqual(got.SourceMap, expected.SourceMap) {
		t.Errorf("sourceMaps not equal\n"+
			"Expected:\n%s\nGot:\n%s\n"+
			"Bytecode dump:\n%s\n",
			tests.Sdump(expected.SourceMap), tests.Sdump(got.SourceMap), got)
		return false
	}
	return true
}
