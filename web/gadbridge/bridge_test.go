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

// TestRunSourceTemplate verifies that `.gadt` template source (mixed `{% … %}`
// tags, including `/* … */` comments) runs in template mode, while the same
// source is (correctly) a parse error under plain Gad. Regression: the web IDE
// ran every file as plain Gad, so templates failed on their first `{%`.
func TestRunSourceTemplate(t *testing.T) {
	src := "{% /* a comment */ %}" +
		"{%-- var (n = 2) --%}" +
		"count={%= n %}\n"

	if d := DiagnoseSource(src, "gadTemplate"); len(d) > 0 {
		t.Fatalf("template diagnose flagged valid source: %+v", d)
	}
	res := RunSource(src, "gadTemplate")
	if !res.OK {
		t.Fatalf("template run failed: stderr=%q diags=%+v", res.Stderr, res.Diagnostics)
	}
	if !strings.Contains(res.Stdout, "count=2") {
		t.Fatalf("unexpected template output: %q", res.Stdout)
	}
	// Plain Gad must still reject the template syntax (the bug being fixed).
	if d := DiagnoseSource(src, "gad"); len(d) == 0 {
		t.Fatal("expected plain-Gad diagnostics for template source")
	}
}

// TestEvalExpr evaluates an expression against a source prelude (the file's
// top-level definitions), the path the IDE's Evaluate panel takes. It covers the
// prelude context, str vs repr rendering, and an error case.
func TestEvalExpr(t *testing.T) {
	// The prelude defines `x`; the expression sees it. str() renders plainly.
	r := EvalExpr("x := 20\n", "x + 22", false)
	if !r.OK || r.Value != "42" {
		t.Fatalf("eval with prelude = %+v, want ok value 42", r)
	}

	// repr() renders the quoted, type-annotated form; str() renders plainly.
	if r := EvalExpr("", `"hi"`, true); !r.OK || !strings.Contains(r.Value, `"hi"`) {
		t.Fatalf("repr eval = %+v, want ok value containing \"hi\"", r)
	}
	if r := EvalExpr("", `"hi"`, false); !r.OK || r.Value != "hi" {
		t.Fatalf("str eval = %+v, want ok value hi", r)
	}

	// A reference to an undefined symbol is an error, not a crash.
	if r := EvalExpr("", "nope", false); r.OK || r.Error == "" {
		t.Fatalf("eval of undefined symbol = %+v, want an error", r)
	}
}

// TestRunSourceArgs passes command-line arguments to a script's `param *args`.
func TestRunSourceArgs(t *testing.T) {
	src := "param *args\nreturn args\n"
	r := RunSourceArgs(src, "gad", []string{"a", "b", "c"})
	if !r.OK {
		t.Fatalf("run failed: stderr=%q diags=%+v", r.Stderr, r.Diagnostics)
	}
	if !strings.Contains(r.Result, "a") || !strings.Contains(r.Result, "c") {
		t.Fatalf("result = %q, want it to contain the args", r.Result)
	}
	// No args → the param list is empty.
	if r := RunSourceArgs(src, "gad", nil); !r.OK || r.Result != "[]" {
		t.Fatalf("no-args result = %q (ok=%v), want []", r.Result, r.OK)
	}
}
