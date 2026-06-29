package gadbridge

import (
	gad "github.com/gad-lang/gad"
	"strings"
	"testing"
)

func TestFormat(t *testing.T) {
	r := Format("x:=1\nif x>0{println(x)}\n")
	if !r.OK {
		t.Fatalf("expected ok, got diagnostics: %v", r.Diagnostics)
	}
	if !strings.Contains(r.Source, "if (x > 0) {\n") {
		t.Fatalf("unexpected format output:\n%s", r.Source)
	}
}

func TestFormatPreservesComments(t *testing.T) {
	// Regression: formatting dropped // comments, trailing comments and doc
	// comments because Format parsed without ParseComments / CodeWithComments.
	r := Format("// header\nx := 1 // trailing\n/// doc for Y\nconst Y = 2\n")
	if !r.OK {
		t.Fatalf("not ok: %v", r.Diagnostics)
	}
	for _, want := range []string{"// header", "// trailing", "/// doc for Y"} {
		if !strings.Contains(r.Source, want) {
			t.Fatalf("format dropped %q:\n%s", want, r.Source)
		}
	}
}

func TestFormatParseError(t *testing.T) {
	r := Format("x :=\n")
	if r.OK {
		t.Fatal("expected not ok for invalid source")
	}
	if len(r.Diagnostics) == 0 {
		t.Fatal("expected diagnostics")
	}
	if r.Diagnostics[0].Line < 1 || r.Diagnostics[0].Column < 1 {
		t.Fatalf("expected positioned diagnostic, got %+v", r.Diagnostics[0])
	}
}

func TestDiagnoseValid(t *testing.T) {
	if d := Diagnose("a := 1\nreturn a\n"); len(d) != 0 {
		t.Fatalf("expected no diagnostics, got %v", d)
	}
}

func TestDiagnoseCompileError(t *testing.T) {
	d := Diagnose("return missing\n")
	if len(d) == 0 {
		t.Fatal("expected a compile diagnostic for an unresolved reference")
	}
	if !strings.Contains(d[0].Message, "unresolved") {
		t.Fatalf("unexpected diagnostic: %+v", d[0])
	}
}

func TestRun(t *testing.T) {
	r := Run(`println("hello"); return 1 + 2`)
	if !r.OK {
		t.Fatalf("expected ok, got %v", r.Diagnostics)
	}
	if !strings.Contains(r.Stdout, "hello") {
		t.Fatalf("expected stdout to contain hello, got %q", r.Stdout)
	}
	if r.Result != "3" {
		t.Fatalf("expected result 3, got %q", r.Result)
	}
}

func TestInspectObject(t *testing.T) {
	// Array: indexed children; a nested dict child is expandable.
	arr := gad.Array{gad.Int(10), gad.Dict{"k": gad.Str("v")}}
	r := InspectObject(nil, arr)
	if r.Type != "array" || !r.Expandable || len(r.Entries) != 2 {
		t.Fatalf("array inspect = %+v", r)
	}
	if r.Entries[0].Key != "0" || r.Entries[0].Accessor != "[0]" || r.Entries[0].Value != "10" {
		t.Fatalf("array entry 0 = %+v", r.Entries[0])
	}
	if r.Entries[1].Accessor != "[1]" || !r.Entries[1].Expandable {
		t.Fatalf("nested dict entry should be expandable: %+v", r.Entries[1])
	}

	// Dict: string keys quoted in the accessor.
	d := gad.Dict{"name": gad.Str("gad"), "n": gad.Int(3)}
	r = InspectObject(nil, d)
	if !r.Expandable || len(r.Entries) != 2 {
		t.Fatalf("dict inspect = %+v", r)
	}
	for _, e := range r.Entries {
		if e.Key == "name" && e.Accessor != `["name"]` {
			t.Fatalf("string-key accessor = %q, want [\"name\"]", e.Accessor)
		}
	}

	// A scalar is not expandable and has no entries.
	r = InspectObject(nil, gad.Int(42))
	if r.Expandable || len(r.Entries) != 0 || r.Value != "42" {
		t.Fatalf("scalar inspect = %+v", r)
	}
}
