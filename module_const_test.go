package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// TestExportConstModule verifies that a compiled module's `export const` is
// declared as a read-only constant at runtime (StdModuleData), its exported
// function lands in the Funcs bucket, and only variables are directly assignable.
func TestExportConstModule(t *testing.T) {
	const mod = "\nexport const Pi = 3\nexport n = 1\nexport func add(a, b) { return a + b }\n"

	// Reads see every member regardless of bucket.
	testExpectRun(t, `m := import("m"); return [m.Pi, m.n, m.add(2, 3)]`,
		newOpts().Module("m", mod), Array{Int(3), Int(1), Int(5)})

	// Members are split into the three buckets by kind.
	testExpectRun(t, `m := import("m"); return [len(m.@consts), len(m.@funcs), len(m.@vars)]`,
		newOpts().Module("m", mod), Array{Int(1), Int(1), Int(1)})

	// A variable is assignable through the module.
	testExpectRun(t, `m := import("m"); m.n = 5; return m.n`,
		newOpts().Module("m", mod), Int(5))

	// Assigning to a constant is rejected at runtime.
	testExpectRun(t, `m := import("m"); try { m.Pi = 9; return "no" } catch e { return "blocked" }`,
		newOpts().Module("m", mod), Str("blocked"))

	// The @consts escape hatch mutates the live constants dict.
	testExpectRun(t, `m := import("m"); m.@consts["Pi"] = 4; return m.Pi`,
		newOpts().Module("m", mod), Int(4))
}
