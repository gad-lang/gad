package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

func TestBuiltinTypes(t *testing.T) {
	for k, v := range BuiltinsMap {
		if v > BuiltinConstantsBegin_ {
			continue
		}

		if v < BuiltinTypesEnd_ {
			if _, ok := BuiltinObjects[v].(ObjectType); !ok {
				t.Fatalf("builtin '%s' is not ObjectType type", k)
			}
		} else if v < BuiltinFunctionsEnd_ {
			if _, ok := BuiltinObjects[v].(*BuiltinFunction); !ok {
				t.Fatalf("builtin '%s' is not *BuiltinFunction type", k)
			}
		} else if v < BuiltinErrorsEnd_ {
			if _, ok := BuiltinObjects[v].(*Error); !ok {
				t.Fatalf("builtin '%s' is not *Error type", k)
			}
		}
	}

	if _, ok := BuiltinObjects[BuiltinGlobals].(*BuiltinFunction); !ok {
		t.Fatal("builtin 'global' is not *BuiltinFunction type")
	}
}
