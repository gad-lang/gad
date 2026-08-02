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
