package gad

import "testing"

// TestStdModuleData covers the three-bucket module data: reads merge all
// members, direct assignment (`module.name = x`) touches only variables, and
// constants/functions are reached (and mutated) through the @consts/@funcs
// escape hatches.
func TestStdModuleData(t *testing.T) {
	d := NewStdModuleData()
	d.Vars["page"] = Int(1)
	d.SetConst("Pi", Float(3.14))
	d.Set("add", &BuiltinFunction{FuncName: "add"}) // routed to Funcs by type

	if _, ok := d.Funcs["add"]; !ok {
		t.Fatalf("Set should route a function to Funcs: %v", d.Funcs)
	}
	if _, ok := d.Vars["add"]; ok {
		t.Fatal("a function must not also land in Vars")
	}

	// Reads see every bucket; Length/Keys reflect the union.
	for _, k := range []string{"page", "Pi", "add"} {
		if v, _ := d.IndexGet(nil, Str(k)); v == nil || v == Nil {
			t.Fatalf("IndexGet(%q) = %v", k, v)
		}
	}
	if d.Length() != 3 {
		t.Fatalf("Length = %d, want 3", d.Length())
	}

	// A variable is assignable; a constant and a function are not.
	if err := d.IndexSet(nil, Str("page"), Int(2)); err != nil {
		t.Fatalf("var IndexSet: %v", err)
	}
	if v, _ := d.IndexGet(nil, Str("page")); !v.Equal(Int(2)) {
		t.Fatalf("page after set = %v", v)
	}
	if err := d.IndexSet(nil, Str("Pi"), Float(9)); err == nil {
		t.Fatal("assigning to a constant should error")
	}
	if err := d.IndexSet(nil, Str("add"), Nil); err == nil {
		t.Fatal("assigning to a function should error")
	}

	// @vars/@consts/@funcs expose the live dicts (the mutation escape hatch).
	consts, _ := d.IndexGet(nil, Str("@consts"))
	cd, ok := consts.(Dict)
	if !ok || !cd["Pi"].Equal(Float(3.14)) {
		t.Fatalf("@consts = %v", consts)
	}
	cd["Pi"] = Float(3.14159) // mutate through the live dict
	if v, _ := d.IndexGet(nil, Str("Pi")); !v.Equal(Float(3.14159)) {
		t.Fatalf("Pi after @consts mutation = %v", v)
	}
	if funcs, _ := d.IndexGet(nil, Str("@funcs")); func() bool { _, ok := funcs.(Dict); return !ok }() {
		t.Fatalf("@funcs is not a dict: %v", funcs)
	}
}

// TestStdModuleDataClosure documents the intended Go-module pattern: an exported
// function closes over the same `vars` dict the module exposes, so calling it
// mutates state the module (and other exports) observe. The data is assigned by
// value — StdModuleData satisfies ModuleData without a pointer.
func TestStdModuleDataClosure(t *testing.T) {
	vars := Dict{"x": Int(2)}
	incX := &Function{FuncName: "incX", Value: func(Call) (Object, error) {
		vars["x"] = vars["x"].(Int) + 1 // mutate the shared exported variable
		return vars["x"], nil
	}}

	var data ModuleData = StdModuleData{Vars: vars, Funcs: Dict{"incX": incX}}

	if v, _ := data.IndexGet(nil, Str("x")); !v.Equal(Int(2)) {
		t.Fatalf("x before = %v", v)
	}
	if _, err := incX.Value(Call{}); err != nil {
		t.Fatal(err)
	}
	if v, _ := data.IndexGet(nil, Str("x")); !v.Equal(Int(3)) {
		t.Fatalf("x after incX = %v (function should see the exported var)", v)
	}
}

// TestStdModuleDataDisjoint verifies a key never lands in two buckets at once as
// it is redeclared across kinds.
func TestStdModuleDataDisjoint(t *testing.T) {
	d := NewStdModuleData()
	d.Set("x", Int(1)) // var
	d.SetConst("x", Int(2))
	if _, ok := d.Vars["x"]; ok {
		t.Fatal("SetConst must remove the variable of the same name")
	}
	if !d.Consts["x"].Equal(Int(2)) {
		t.Fatalf("x const = %v", d.Consts["x"])
	}
	d.Set("x", &BuiltinFunction{FuncName: "x"}) // now a function
	if _, ok := d.Consts["x"]; ok {
		t.Fatal("Set must remove the constant of the same name")
	}
	if _, ok := d.Funcs["x"]; !ok {
		t.Fatal("x should be a function now")
	}
	if d.Length() != 1 {
		t.Fatalf("Length = %d, want 1 (disjoint)", d.Length())
	}
}
