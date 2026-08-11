package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestEmitAPIOverloads checks that consecutive same-name signatures become a
// single `export func NAME { … }` with one method per overload, while a
// single-signature function keeps the flat `export name(params) => nil` form.
func TestEmitAPIOverloads(t *testing.T) {
	dg := &docgroup{
		docs: []string{"# `probe` module", "", "Probe."},
		api: []apiSym{
			{kind: "func", name: "area", overloads: []apiOverload{
				{sig: "area(r float) <float>", desc: []string{"circle"}},
				{sig: "area(w float, h float) <float>", desc: []string{"rectangle"}},
			}},
			{kind: "func", name: "id", overloads: []apiOverload{
				{sig: "id(x int) <int>", desc: []string{"identity"}},
			}},
			{kind: "const", name: "Pi", sig: "3.14159", desc: []string{"pi"}},
		},
	}

	src, warnings := emitAPIGad(dg, nil)
	require.Empty(t, warnings, src)

	for _, want := range []string{
		"export func area {",
		"/// circle",
		"(r float) <float> => nil",
		"/// rectangle",
		"(w float, h float) <float> => nil",
		"export id(x int) <int> => nil", // single sig stays flat
		"export const Pi = 3.14159",
	} {
		require.Contains(t, src, want)
	}

	// The generated file must be valid Gad.
	require.True(t, parsesRaw(src), src)
	// area is emitted once (as a func-with-methods), not two separate exports.
	require.Equal(t, 1, strings.Count(src, "export func area"))
}

// TestGroupOverloadsCapture verifies the docgroup groups consecutive same-name
// signature lines into one func entry with multiple overloads.
func TestGroupOverloadsCapture(t *testing.T) {
	dg := &docgroup{curFuncIdx: -1}
	// Simulate two consecutive signatures for `area` then a different func.
	dg.captureFuncSig("area", "area(r float) <float>")
	dg.appendFuncDesc("circle")
	dg.captureFuncSig("area", "area(w float, h float) <float>")
	dg.appendFuncDesc("rectangle")
	dg.captureFuncSig("id", "id(x int) <int>")

	require.Len(t, dg.api, 2)
	require.Equal(t, "area", dg.api[0].name)
	require.Len(t, dg.api[0].overloads, 2)
	require.Equal(t, []string{"circle"}, dg.api[0].overloads[0].desc)
	require.Equal(t, []string{"rectangle"}, dg.api[0].overloads[1].desc)
	require.Equal(t, "id", dg.api[1].name)
	require.Len(t, dg.api[1].overloads, 1)
}
