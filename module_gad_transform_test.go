package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// TestGadTransformIssueExample is the issue's own case: a nested value is rewritten
// bottom-up — the inner arrays become Point instances first, then the root dict
// becomes a Points whose pts already hold the transformed Points.
func TestGadTransformIssueExample(t *testing.T) {
	testExpectRun(t, `
		class Points {pts array}
		class Point {x;y}
		d := {points: [[1,2],[3,4]]}
		d = gad.transform(d;
			"." = (d dict) => Points(;pts=d.points)
			".points[]" = (d array) => Point(;x=d[0], y=d[1])
		)
		return [typeName(d), typeName(d.pts[0]), d.pts[0].x, d.pts[1].y]`,
		nil, Array{Str("Points"), Str("Point"), Int(1), Int(4)})
}

// TestGadTransformAllArrayIndices: `.[]` matches every element of the root array.
func TestGadTransformAllArrayIndices(t *testing.T) {
	testExpectRun(t, `return gad.transform([1,2,3]; ".[]" = (n int) => n*10)`,
		nil, Array{Int(10), Int(20), Int(30)})
}

// TestGadTransformAllDictKeys: `.*` matches every non-array (dict) child.
func TestGadTransformAllDictKeys(t *testing.T) {
	testExpectRun(t, `
		m := gad.transform({a:1, b:2}; ".*" = (n int) => n+100)
		return [m.a, m.b]`,
		nil, Array{Int(101), Int(102)})
}

// TestGadTransformAnyChild: `.**` matches any child — array index and dict key.
func TestGadTransformAnyChild(t *testing.T) {
	testExpectRun(t, `
		// double every immediate child, whether reached by key or by index.
		m := gad.transform({xs:[1,2], y:5}; ".**" = (v) => v)
		return [typeName(m.xs), m.y]`,
		nil, Array{Str("array"), Int(5)})
}

// TestGadTransformSpecificIndex: `.key[N]` matches only that one index.
func TestGadTransformSpecificIndex(t *testing.T) {
	testExpectRun(t, `
		p := gad.transform({v:[10,20,30]}; ".v[1]" = (n int) => n*2)
		return p.v`,
		nil, Array{Int(10), Int(40), Int(30)})
}

// TestGadTransformEscapedKey: a `."quoted key"` segment matches a key with spaces.
func TestGadTransformEscapedKey(t *testing.T) {
	testExpectRun(t, `
		e := gad.transform({"po in": 7}; ".\"po in\"" = (n int) => n+1)
		return e["po in"]`,
		nil, Int(8))
}

// TestGadTransformSpecificityWins: at a node matched by both a literal and a
// wildcard path, the more specific (literal) matcher wins — regardless of the
// order the two were listed.
func TestGadTransformSpecificityWins(t *testing.T) {
	testExpectRun(t, `
		s := gad.transform({a:{b:1, c:1}};
			".a.*" = (n int) => 9
			".a.b" = (n int) => 100
		)
		return [s.a.b, s.a.c]`,
		// .a.b beats .a.* on b; c only matches the wildcard.
		nil, Array{Int(100), Int(9)})
}

// TestGadTransformTypedParamEnforced: a matcher's typed first param is enforced —
// an array node hitting a `(d dict)` matcher raises, it does not silently skip.
func TestGadTransformTypedParamEnforced(t *testing.T) {
	testExpectRun(t, `
		ok := false
		try {
			gad.transform([1,2]; ".[]" = (d dict) => d)
		} catch {
			ok = true
		}
		return ok`,
		nil, True)
}

// TestGadTransformBadPath: a path not starting with '.' is a type error.
func TestGadTransformBadPath(t *testing.T) {
	expectErrHas(t, `gad.transform(1; "points" = (v) => v)`,
		newOpts().Skip2Pass(), "transform path must start with '.'")
}
