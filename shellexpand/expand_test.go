package shellexpand

import (
	"reflect"
	"testing"
)

// envFrom builds an Env whose Get reads from a map; Set writes back to it.
func envFrom(vars map[string]string, cfg any) Env {
	return Env{
		Get:    func(name string) (string, bool) { v, ok := vars[name]; return v, ok },
		Set:    func(name, value string) { vars[name] = value },
		Config: cfg,
	}
}

func TestExpandBasic(t *testing.T) {
	vars := map[string]string{"HOME": "/home/u", "EMPTY": "", "PATH": "/bin"}
	cases := []struct{ in, want string }{
		{`$HOME`, "/home/u"},
		{`${HOME}`, "/home/u"},
		{`${HOME}/bin`, "/home/u/bin"},
		{`x/bin:$PATH`, "x/bin:/bin"},
		{`$UNSET`, ""},
		{`\$HOME`, "$HOME"}, // escaped
		{`$$`, "$$"},        // lone $ (not a name start) stays literal
		{`a${HOME}b${PATH}c`, "a/home/ub/binc"},
	}
	for _, c := range cases {
		if got := Expand(c.in, envFrom(vars, nil)); got != c.want {
			t.Errorf("Expand(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestExpandDefaults(t *testing.T) {
	vars := map[string]string{"SET": "v", "EMPTY": ""}
	cases := []struct{ in, want string }{
		{`${SET:-d}`, "v"},
		{`${EMPTY:-d}`, "d"},
		{`${UNSET:-d}`, "d"},
		{`${UNSET:-$SET}`, "v"}, // default is itself expanded
		{`${SET:+alt}`, "alt"},
		{`${EMPTY:+alt}`, ""},
		{`${UNSET:+alt}`, ""},
	}
	for _, c := range cases {
		if got := Expand(c.in, envFrom(vars, nil)); got != c.want {
			t.Errorf("Expand(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestExpandAssignDefault(t *testing.T) {
	vars := map[string]string{}
	if got := Expand(`${NEW:=fallback}`, envFrom(vars, nil)); got != "fallback" {
		t.Fatalf("assign result = %q, want fallback", got)
	}
	if vars["NEW"] != "fallback" {
		t.Fatalf("NEW not assigned: %q", vars["NEW"])
	}
	// A subsequent reference sees the assigned value.
	if got := Expand(`$NEW`, envFrom(vars, nil)); got != "fallback" {
		t.Fatalf("after assign = %q, want fallback", got)
	}
}

func TestExpandSubstring(t *testing.T) {
	vars := map[string]string{"V": "abcdef"}
	cases := []struct{ in, want string }{
		{`${V:2}`, "cdef"},
		{`${V:2:2}`, "cd"},
		{`${V:0:3}`, "abc"},
		{`${V:-1}`, "abcdef"}, // ":-1" is the default operator, V is set → V
		{`${V: -2}`, "ef"},    // negative offset (space avoids ':-')
		{`${V:1:-1}`, "bcde"}, // negative length = offset from end
		{`${V:10}`, ""},
	}
	for _, c := range cases {
		if got := Expand(c.in, envFrom(vars, nil)); got != c.want {
			t.Errorf("Expand(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestExpandStripPrefixSuffix(t *testing.T) {
	// Values and expected results verified against GNU bash 5.2.
	vars := map[string]string{"F": "src/main.test.gad", "P": "a.b.c.d", "W": "aXbXc"}
	cases := []struct{ in, want string }{
		{`${F#*/}`, "main.test.gad"},   // shortest prefix up to a '/'
		{`${F##*/}`, "main.test.gad"},  // only one '/', same
		{`${F%.gad}`, "src/main.test"}, // strip literal suffix
		{`${F%.*}`, "src/main.test"},   // shortest suffix from last '.'
		{`${F%%.*}`, "src/main"},       // longest suffix from first '.'
		{`${P#*.}`, "b.c.d"},           // shortest prefix ending in '.'
		{`${P##*.}`, "d"},              // longest prefix ending in '.'
		{`${P%.*}`, "a.b.c"},
		{`${P%%.*}`, "a"},
		// Repeated pattern char: the shortest/longest distinction really matters.
		{`${W#*X}`, "bXc"}, // shortest prefix
		{`${W##*X}`, "c"},  // longest prefix
		{`${W%X*}`, "aXb"}, // shortest suffix
		{`${W%%X*}`, "a"},  // longest suffix
	}
	for _, c := range cases {
		if got := Expand(c.in, envFrom(vars, nil)); got != c.want {
			t.Errorf("Expand(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestExpandReplace(t *testing.T) {
	vars := map[string]string{"V": "a/b/a/b", "X": "hello"}
	cases := []struct{ in, want string }{
		{`${V/b/Z}`, "a/Z/a/b"},  // first
		{`${V//b/Z}`, "a/Z/a/Z"}, // all
		{`${V/#a/Z}`, "Z/b/a/b"}, // front only
		{`${V/%b/Z}`, "a/b/a/Z"}, // end only
		{`${V//a\/b/Z}`, "Z/Z"},  // pattern with escaped '/'
		{`${X/l*o/Y}`, "heY"},    // glob in pattern
		{`${X/xyz/Q}`, "hello"},  // no match
	}
	for _, c := range cases {
		if got := Expand(c.in, envFrom(vars, nil)); got != c.want {
			t.Errorf("Expand(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestExpandConfigPath(t *testing.T) {
	cfg := map[string]any{
		"ide": map[string]any{
			"a": []any{
				[]any{"x", "y"},
				[]any{"z", map[string]any{"g": 42}},
			},
			"name": "proj",
			"on":   true,
		},
	}
	env := envFrom(map[string]string{"H": "/h"}, cfg)
	cases := []struct{ in, want string }{
		{`${.ide.name}`, "proj"},
		{`${.ide.a[0][1]}`, "y"},
		{`${.ide.a[1][1].g}`, "42"}, // int leaf → "42"
		{`${.ide.on}`, "true"},      // bool leaf → "true"
		{`${.ide.missing:-none}`, "none"},
		{`${.ide.name}-$H`, "proj-/h"}, // mixed with env
	}
	for _, c := range cases {
		if got := Expand(c.in, env); got != c.want {
			t.Errorf("Expand(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestWalkExpand(t *testing.T) {
	vars := map[string]string{"PORT": "8080", "NAME": "svc", "DBG": "yes"}
	doc := map[string]any{
		"name":    "${NAME}",
		"port":    "${PORT:-1}", // expands → coerced to int
		"debug":   "${DBG}",     // "yes" → bool true
		"literal": "8080",       // no reference → stays string
		"n":       5,            // non-string scalar untouched
		"nested": map[string]any{
			"path": "x/bin:$NAME",
			"list": []any{"${PORT}", "plain"},
		},
	}
	got := WalkExpand(doc, envFrom(vars, nil)).(map[string]any)
	want := map[string]any{
		"name":    "svc",
		"port":    8080,   // int
		"debug":   true,   // bool
		"literal": "8080", // string
		"n":       5,
		"nested": map[string]any{
			"path": "x/bin:svc",
			"list": []any{8080, "plain"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("WalkExpand mismatch:\n got  %#v\n want %#v", got, want)
	}
}
