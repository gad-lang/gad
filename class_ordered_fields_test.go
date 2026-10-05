package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// TestClassOrderedFields covers `[ordered]` before a class: its instances
// print their fields in the order declared — the parents' first —, whatever
// the names; without it, sorted by name.
func TestClassOrderedFields(t *testing.T) {
	testExpectRun(t, `
		class P { pz = 1; pa = 2 }
		[ordered] class C { *P; z = 3; a = 4 }
		class D { *P; z = 3; a = 4 }
		return [str(C(; a=5)), str(D())]`,
		nil, Array{
			Str("‹class instance of ‹(main).C›: {pz: 1, pa: 2, z: 3, a: 5}›"),
			Str("‹class instance of ‹(main).D›: {a: 4, pa: 2, pz: 1, z: 3}›"),
		})
	testExpectRun(t, `[ordered] class C { a = 1 }; class D { a = 1 }; return [len(C.@meta), len(D.@meta)]`,
		nil, Array{Int(1), Int(0)})
}

// TestClassOrderedFieldsGo covers the option from Go: Keys, Values and Items
// of an instance in the order the class declares its fields.
func TestClassOrderedFieldsGo(t *testing.T) {
	c := NewClass("C", nil)
	if err := c.AddField(&ClassField{Name: "z"}, &ClassField{Name: "a"}, &ClassField{Name: "m"}); err != nil {
		t.Fatal(err)
	}
	c.OrderedFields = true
	inst, err := c.NewInstanceWithFields(nil, Dict{"z": Int(1), "a": Int(2), "m": Int(3)})
	if err != nil {
		t.Fatal(err)
	}
	if got := inst.Keys(); !got.Equal(Array{Str("z"), Str("a"), Str("m")}) {
		t.Errorf("keys = %v", got)
	}
	if got := inst.Values(); !got.Equal(Array{Int(1), Int(2), Int(3)}) {
		t.Errorf("values = %v", got)
	}
	var names []string
	_ = inst.Items(nil, func(_ int, kv *KeyValue) error { names = append(names, kv.K.ToString()); return nil })
	if len(names) != 3 || names[0] != "z" || names[1] != "a" || names[2] != "m" {
		t.Errorf("items = %v", names)
	}
}

// TestClassInstanceWalkParents covers the instances of a class that extends
// another, printed: its parents' fields, once (the walk ended).
func TestClassInstanceWalkParents(t *testing.T) {
	testExpectRun(t, `
		class G { g = 0 }
		class P { *G; p = 1 }
		class D { *P; d = 2 }
		return str(D())`,
		nil, Str("‹class instance of ‹(main).D›: {d: 2, g: 0, p: 1}›"))
}
