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
			if v2, ok := BuiltinObjects[v].(ObjectType); !ok {
				t.Fatalf("builtin '%s' (%T) is not ObjectType type", k, v2)
			}
		} else if v < BuiltinFunctionsEnd_ {
			if v2, ok := BuiltinObjects[v].(*BuiltinFunction); !ok {
				t.Fatalf("builtin '%s' (%T) is not *BuiltinFunction type", k, v2)
			}
		} else if v < BuiltinErrorsEnd_ {
			if v2, ok := BuiltinObjects[v].(*Error); !ok {
				t.Fatalf("builtin '%s' (%T) is not *Error type", k, v2)
			}
		}
	}

	if _, ok := BuiltinObjects[BuiltinGlobals].(*BuiltinFunction); !ok {
		t.Fatal("builtin 'global' is not *BuiltinFunction type")
	}
}
