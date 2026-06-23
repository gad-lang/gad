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
		} else if v < BuiltinStaticTypesEnd_ {
			if v2, ok := BuiltinObjects[v].(*BuiltinObjType); !ok {
				t.Fatalf("builtin '%s' (%T) is not *BuiltinObjType type", k, v2)
			}
		} else if v < BuiltinFunctionsEnd_ {
			// @binaryOperator / @selfAssignOperator carry typed operator methods
			// (registerOperatorMethods), so they are wrapped as
			// *BuiltinFunctionWithMethods rather than a bare *BuiltinFunction.
			switch BuiltinObjects[v].(type) {
			case *BuiltinFunction, *BuiltinFunctionWithMethods:
			default:
				t.Fatalf("builtin '%s' (%T) is not a function type", k, BuiltinObjects[v])
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
