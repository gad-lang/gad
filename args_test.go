package gad

import "testing"

// TestParseArgs covers typed coercion, named args, boolean flags and the `--`
// option terminator.
func TestParseArgs(t *testing.T) {
	pos, named := ParseArgs([]string{
		"5", `{a: 1}`, `"hi"`, "bare", "--count=10", `--cfg={x: 2}`, "--on", "--", "--not-a-flag",
	})

	// Positional: typed values, with a string fallback for the unparseable ones.
	if len(pos) != 5 {
		t.Fatalf("positional count = %d (%v), want 5", len(pos), pos)
	}
	if _, ok := pos[0].(Int); !ok || pos[0].ToString() != "5" {
		t.Fatalf("pos[0] = %v (%T), want Int 5", pos[0], pos[0])
	}
	if _, ok := pos[1].(Dict); !ok {
		t.Fatalf("pos[1] type = %T, want Dict", pos[1])
	}
	if pos[2].ToString() != "hi" {
		t.Fatalf("pos[2] = %q, want hi", pos[2].ToString())
	}
	if pos[3].ToString() != "bare" { // unparseable identifier → string
		t.Fatalf("pos[3] = %q, want bare", pos[3].ToString())
	}
	// After `--`, `--not-a-flag` is positional (string), not a named flag.
	if pos[4].ToString() != "--not-a-flag" {
		t.Fatalf("pos[4] = %q, want --not-a-flag", pos[4].ToString())
	}

	// Named: typed value, dict value and boolean flag.
	if v, ok := named["count"].(Int); !ok || v != 10 {
		t.Fatalf("named[count] = %v (%T), want Int 10", named["count"], named["count"])
	}
	if _, ok := named["cfg"].(Dict); !ok {
		t.Fatalf("named[cfg] type = %T, want Dict", named["cfg"])
	}
	if named["on"] != Yes {
		t.Fatalf("named[on] = %v, want Yes", named["on"])
	}
	if _, hasNot := named["not-a-flag"]; hasNot {
		t.Fatal("--not-a-flag after -- must not be a named arg")
	}
}
