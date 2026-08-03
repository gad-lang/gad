package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// propDict is a dict literal whose key `x` holds a getter/setter Prop backed by
// the local `v`. Reused across the delegation tests.
const propDict = `var (v = 1, d = {x: prop { () => v; (val) { v = val } }})`

// TestVMPropIndexDelegation covers computed-property delegation: reading/writing
// a dict/module key that holds a Prop runs its getter/setter (see vm_prop.go).
func TestVMPropIndexDelegation(t *testing.T) {
	// Read delegates to the getter.
	testExpectRun(t, propDict+"; return d.x", nil, Int(1))
	testExpectRun(t, propDict+`; return d["x"]`, nil, Int(1))

	// Write delegates to the setter (mutating the backing local).
	testExpectRun(t, propDict+"; d.x = 2; return v", nil, Int(2))
	testExpectRun(t, propDict+"; d.x = 5; return d.x", nil, Int(5))
	testExpectRun(t, propDict+`; d["x"] = 7; return d.x`, nil, Int(7))

	// A plain (non-prop) key is unaffected: normal get/set, and a new key can be
	// added without delegation.
	testExpectRun(t, propDict+"; d.y = 3; return [d.x, d.y]", nil, Array{Int(1), Int(3)})

	// Nested containers delegate at the leaf.
	testExpectRun(t, `var (v = 9, d = {a: {x: prop { () => v; (val) { v = val } }}}); return d.a.x`,
		nil, Int(9))
	testExpectRun(t, `var (v = 0, d = {a: {x: prop { () => v; (val) { v = val } }}}); d.a.x = 4; return v`,
		nil, Int(4))

	// A getter-only property errors on assignment (no matching setter).
	expectErrHas(t, `var d = {x: prop { () => 1 }}; d.x = 2; return d.x`, newOpts(),
		"no have method")
}

// TestVMPropValueField covers a Prop's virtual `v` field: x.v reads via the
// getter and x.v = value writes via the setter, without an explicit call.
func TestVMPropValueField(t *testing.T) {
	const p = "var v = 1; prop x { () => v; (val) { v = val } }"

	// x.v reads through the getter; x.v = n writes through the setter.
	testExpectRun(t, p+"; return x.v", nil, Int(1))
	testExpectRun(t, p+"; x.v = 10; return v", nil, Int(10))
	testExpectRun(t, p+"; x.v = 10; return x.v", nil, Int(10))
	// Bracket indexing works too.
	testExpectRun(t, p+`; return x["v"]`, nil, Int(1))

	// Calling the prop directly still works and shares the same backing value.
	testExpectRun(t, p+"; x(8); return x.v", nil, Int(8))
	testExpectRun(t, p+"; x.v = 5; return x()", nil, Int(5))

	// A getter-only prop errors on x.v = … (no setter); an unknown field errors.
	expectErrHas(t, `prop y { () => 1 }; y.v = 3`, newOpts(), "no have method")
	expectErrHas(t, `prop y { () => 1 }; return y.w`, newOpts(), "InvalidIndexError")
}

// TestVMPropReadonly covers the getter-only `prop => expr` form (prop as a
// closure): anonymous, named, live reads, and the read-only error on write.
func TestVMPropReadonly(t *testing.T) {
	// Anonymous read-only prop assigned to a variable, read via .v; live.
	testExpectRun(t, `var (_x = 5, x = prop => _x); return x.v`, nil, Int(5))
	testExpectRun(t, `var (_x = 5, x = prop => _x); _x = 9; return x.v`, nil, Int(9))
	// Named read-only prop statement; read via call and via .v.
	testExpectRun(t, `var _x = 3; prop y => _x; return y()`, nil, Int(3))
	testExpectRun(t, `var _x = 3; prop y => _x; return y.v`, nil, Int(3))
	// Writing a read-only prop errors (no setter).
	expectErrHas(t, `var _x = 1; prop y => _x; y.v = 2`, newOpts(), "no have method")

	// An anonymous read-only prop stored at a dict key: reading delegates to its
	// getter (live), and writing errors (no setter).
	testExpectRun(t, `var (_x = 10, d = {x: prop => _x}); return d.x`, nil, Int(10))
	testExpectRun(t, `var (_x = 10, d = {x: prop => _x}); _x = 20; return d.x`, nil, Int(20))
	expectErrHas(t, `var (_x = 10, d = {x: prop => _x}); d.x = 5`, newOpts(), "no have method")
}

// TestVMExportPropLiveBinding covers `export prop x = init`: a module exports a
// live read/write binding — reading/writing m.x delegates to a getter/setter
// over the module's own `var x`, so external writes are observed by functions
// closing over x, and vice versa.
func TestVMExportPropLiveBinding(t *testing.T) {
	const mod = "export prop x = 10\nexport getX() => x\nexport bump() { x = x + 1 }\n"
	opt := func() *VMTestOpts { return newOpts().Module("mod", mod) }

	testExpectRun(t, `m := import("mod"); return m.x`, opt(), Int(10))
	testExpectRun(t, `m := import("mod"); return m.getX()`, opt(), Int(10))
	// An external write to m.x is observed by the module's own getX (live).
	testExpectRun(t, `m := import("mod"); m.x = 12; return m.getX()`, opt(), Int(12))
	testExpectRun(t, `m := import("mod"); m.x = 12; return m.x`, opt(), Int(12))
	// An internal mutation (bump) is observed through m.x.
	testExpectRun(t, `m := import("mod"); m.bump(); return m.x`, opt(), Int(11))
	// reflect.get returns the backing Prop verbatim.
	testExpectRun(t, `m := import("mod"); return typeName(reflect.get(m, "x"))`, opt(), Str("Prop"))

	// export prop x => expr exports a read-only binding (writing errors).
	const ro = "var _v = 7\nexport prop x => _v\nexport bump() { _v = _v + 1 }\n"
	testExpectRun(t, `m := import("mod"); return m.x`, newOpts().Module("mod", ro), Int(7))
	testExpectRun(t, `m := import("mod"); m.bump(); return m.x`, newOpts().Module("mod", ro), Int(8))
	expectErrHas(t, `m := import("mod"); m.x = 3`, newOpts().Module("mod", ro), "no have method")
}

// TestVMPropRawContainers verifies Array and ClassInstance never delegate: a
// stored Prop is returned/stored verbatim.
func TestVMPropRawContainers(t *testing.T) {
	// Array element holding a Prop is the value itself (no getter run).
	testExpectRun(t, `var v = 1; p := prop { () => v; (val) { v = val } }; a := [p]; return a[0] == p`,
		nil, True)
	testExpectRun(t, `p := prop { () => 1 }; a := [p]; return typeName(a[0])`, nil, Str("Prop"))

	// A class field holding a Prop is returned verbatim (the instance resolves its
	// own class properties, not stored Prop values).
	testExpectRun(t, `
	class C { p = nil }
	c := C()
	c.p = prop { () => 42 }
	return typeName(c.p)`, nil, Str("Prop"))
}

// TestVMReflectRaw covers the reflect module: raw, delegation-free access.
func TestVMReflectRaw(t *testing.T) {
	// get returns the stored Prop verbatim (getter not run).
	testExpectRun(t, propDict+`; return typeName(reflect.get(d, "x"))`, nil, Str("Prop"))
	// get on a plain key returns the value.
	testExpectRun(t, `d := {y: 42}; return reflect.get(d, "y")`, nil, Int(42))
	// set overwrites (removes) the Prop without running its setter; the key now
	// holds a plain value.
	testExpectRun(t, propDict+`; reflect.set(d, "x", 3); return d.x`, nil, Int(3))
	testExpectRun(t, propDict+`; reflect.set(d, "x", 3); return typeName(reflect.get(d, "x"))`,
		nil, Str("int"))
	// set on an array writes by index (arrays never delegate anyway).
	testExpectRun(t, `a := [1, 2, 3]; reflect.set(a, 1, 9); return a`, nil,
		Array{Int(1), Int(9), Int(3)})
	// get accepts int keys.
	testExpectRun(t, `a := [10, 20]; return reflect.get(a, 1)`, nil, Int(20))
	// non-indexable target errors.
	expectErrHas(t, `return reflect.get(1, "x")`, newOpts(), "NotIndexableError")
}

// TestVMExportProp covers a module exporting a Prop: member access on the
// imported module delegates to the prop, and reflect.get bypasses it.
func TestVMExportProp(t *testing.T) {
	const mod = "var value = 0\nexport prop v { () => value; (val) { value = val } }\n"

	// Reading the exported prop runs its getter (default value).
	testExpectRun(t, `m := import("mod"); return m.v`,
		newOpts().Module("mod", mod), Int(0))
	// Writing runs its setter; a subsequent read reflects it.
	testExpectRun(t, `m := import("mod"); m.v = 5; return m.v`,
		newOpts().Module("mod", mod), Int(5))
	// reflect.get returns the Prop verbatim.
	testExpectRun(t, `m := import("mod"); return typeName(reflect.get(m, "v"))`,
		newOpts().Module("mod", mod), Str("Prop"))

	// An anonymous export prop is a parse error.
	expectErrHas(t, `export prop { () => 1 }`, newOpts().CompilerError(),
		"export prop requires a name")
}
