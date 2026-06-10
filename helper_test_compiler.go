package gad

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	testhelper "github.com/gad-lang/gad/test_helper/teststrings"
	"github.com/gad-lang/gad/tests"
	"github.com/stretchr/testify/require"
)

type TestBytecodesEqualOptions struct {
	NoInsertMainModule bool
}

func TestBytecodesEqual(t *testing.T,
	expected, got *Bytecode, checkSourceMap bool, opt ...*TestBytecodesEqualOptions) {
	var opts *TestBytecodesEqualOptions

	for _, opts = range opt {
	}

	if opts == nil {
		opts = &TestBytecodesEqualOptions{}
	}

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

	if !opts.NoInsertMainModule {
		if len(expected.Modules) == 0 {
			expected.Modules = []*ModuleSpec{{ModuleInfo: ModuleInfo{Name: MainName}}}
			expected.NumModules++
		} else {
			m := expected.Modules[0]
			if !m.Main {
				expected.Modules = append([]*ModuleSpec{{ModuleInfo: ModuleInfo{Name: MainName}}}, expected.Modules...)
				expected.NumModules++
			}
		}
	}

	expected.NumModules = len(expected.Modules)

	if len(expected.Modules) != len(got.Modules) {
		t.Fatalf("Modules len not equal\n"+
			"Expected len(Modules):\n%d\nGot len(Modules):\n%d\n",
			len(expected.Modules), len(got.Modules))
	}

	if len(expected.Constants) != len(got.Constants) {
		t.Fatalf("Constants len not equal\n"+
			"Expected len(Constants):\n%d\nGot len(Constants):\n%d\n"+
			"Expected Dump:\n%s\nGot Dump:\n%s\n",
			len(expected.Constants), len(got.Constants), repr(expected.Constants), repr(got.Constants))
	}

	compareCf(expected.Main, got.Main, false, "Main functions")

	for i, gMod := range got.Modules {
		eMod := expected.Modules[i]
		if gMod.ModuleInfo.Name != eMod.ModuleInfo.Name {
			t.Fatalf("Module not equal at %d\nExpected:\n%v (%[2]T)\nGot:\n%v (%[3]T)\n",
				i, eMod.ModuleInfo.Name, gMod.ModuleInfo.Name)
		}

		if eMod.InitGoFunc != nil {
			require.NotNil(t, gMod.InitGoFunc, "Got module[%d] %s InitGoFunc is nil", i, eMod.ModuleInfo.Name)
			ef := gMod.InitFunc(NewModule(eMod))
			gf := gMod.InitFunc(NewModule(gMod))
			require.Equal(t, ef.ToString(), gf.ToString(), "InitGoFunc not equal at %d", i)
		} else if eMod.InitCompiledFunc != nil {
			require.NotNil(t, gMod.InitCompiledFunc, "Got module[%d] %s InitCompiledFunc is nil", i, eMod.ModuleInfo.Name)
			if !assertCompiledFunctionsEqual(t,
				eMod.InitCompiledFunc, gMod.InitCompiledFunc, checkSourceMap) {
				t.Fatalf("InitCompiledFunc not equal at %d \nExpected:\n%v\nGot:\n%v\n",
					i, eMod.InitCompiledFunc.HeaderString(), gMod.InitCompiledFunc.HeaderString())
			}
		} else {
			require.Nil(t, gMod.InitGoFunc)
			require.Nil(t, gMod.InitCompiledFunc)
		}
	}

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
		case *Embedded:
			ee, ok := expectObj.(*Embedded)
			if !ok {
				t.Fatalf("%T expected at index %d but got %T",
					expectObj, i, gotObj)
			}
			if ee.Path() == g.Path() {
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
