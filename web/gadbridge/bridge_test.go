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

func TestFormatSourceTemplate(t *testing.T) {
	// A mixed template must format as a template, keeping the {% %}/{%= %} islands.
	r := FormatSource("{%    var name=\"Gad\" %}\n<h1>Hello, {%=name%}!</h1>\n", "gadTemplate")
	if !r.OK {
		t.Fatalf("not ok: %v", r.Diagnostics)
	}
	for _, want := range []string{`{% var name = "Gad" %}`, `{%= name %}`, "<h1>Hello,"} {
		if !strings.Contains(r.Source, want) {
			t.Fatalf("template format missing %q:\n%s", want, r.Source)
		}
	}
}

func TestFormatSourceGadx(t *testing.T) {
	// Gadx must format back to Gadx syntax (tags/components), not lowered Gad.
	r := FormatSource("@main\n    h1 Hello Gadx\n    ul\n        @for i in [1, 2, 3]\n            li item {= i }\n", "gadx")
	if !r.OK {
		t.Fatalf("not ok: %v", r.Diagnostics)
	}
	for _, want := range []string{"@comp main()", "h1", "@for (i in [1, 2, 3])"} {
		if !strings.Contains(r.Source, want) {
			t.Fatalf("gadx format missing %q:\n%s", want, r.Source)
		}
	}
}

func TestFormatSourceGadxParseError(t *testing.T) {
	// Inconsistent indentation (spaces then tabs) is rejected by the Gadx parser.
	r := FormatSource("    \tbad\n\t\tmix\n", "gadx")
	if r.OK {
		t.Fatal("expected not ok for invalid gadx")
	}
	if len(r.Diagnostics) == 0 {
		t.Fatal("expected diagnostics")
	}
	if r.Diagnostics[0].Line < 1 || r.Diagnostics[0].Column < 1 {
		t.Fatalf("expected positioned diagnostic, got %+v", r.Diagnostics[0])
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

// TestRunSourceGadxTagEncode covers the gadx tag output modes: default renders
// HTML, "json"/"yaml" encode the returned tag's structured data instead.
func TestRunSourceGadxTagEncode(t *testing.T) {
	src := "@main\n    h1 Hello\n    ul\n        @for i in [1, 2]\n            li item {= i }"

	// Default: HTML render.
	if r := RunSourceArgs(src, "gadx", nil, ""); !r.OK || !strings.Contains(r.Stdout, "<h1>Hello</h1>") {
		t.Fatalf("html render: ok=%v stdout=%q", r.OK, r.Stdout)
	}

	// JSON: structured data, no HTML tags.
	rj := RunSourceArgs(src, "gadx", nil, "json")
	if !rj.OK {
		t.Fatalf("json run failed: %q", rj.Stderr)
	}
	for _, want := range []string{`"tag"`, `"h1"`, `"children"`} {
		if !strings.Contains(rj.Stdout, want) {
			t.Fatalf("json output missing %q:\n%s", want, rj.Stdout)
		}
	}
	if strings.Contains(rj.Stdout, "<h1>") {
		t.Fatalf("json output should not contain HTML tags:\n%s", rj.Stdout)
	}

	// YAML: structured data, YAML syntax.
	ry := RunSourceArgs(src, "gadx", nil, "yaml")
	if !ry.OK || !strings.Contains(ry.Stdout, "tag: h1") {
		t.Fatalf("yaml output = %q", ry.Stdout)
	}
}

// TestRunSourceArgs passes command-line arguments to a script's `param *args`.
func TestRunSourceArgs(t *testing.T) {
	src := "param *args\nreturn args\n"
	r := RunSourceArgs(src, "gad", []string{"a", "b", "c"}, "")
	if !r.OK {
		t.Fatalf("run failed: stderr=%q diags=%+v", r.Stderr, r.Diagnostics)
	}
	if !strings.Contains(r.Result, "a") || !strings.Contains(r.Result, "c") {
		t.Fatalf("result = %q, want it to contain the args", r.Result)
	}
	// No args → the param list is empty.
	if r := RunSourceArgs(src, "gad", nil, ""); !r.OK || r.Result != "[]" {
		t.Fatalf("no-args result = %q (ok=%v), want []", r.Result, r.OK)
	}
}

// TestFormatSourceShebang verifies a leading `#!…` line round-trips through the
// formatter in every dialect and is ignored at run time.
func TestFormatSourceShebang(t *testing.T) {
	cases := []struct {
		st, src, wantRun string
	}{
		{"gad", "#!/usr/bin/env gad\nx:=1\nreturn x\n", ""},
		{"gadTemplate", "#!/usr/bin/env gad\n{% x := 1 %}v={%= x %}\n", "v=1\n"},
		{"gadx", "#!/usr/bin/env gad\ndiv Hello\n", "<div>Hello</div>"},
	}
	for _, c := range cases {
		f := FormatSource(c.src, c.st)
		if !f.OK {
			t.Fatalf("%s: format failed: %v", c.st, f.Diagnostics)
		}
		if want := "#!/usr/bin/env gad\n"; f.Source[:len(want)] != want {
			t.Fatalf("%s: shebang not preserved: %q", c.st, f.Source)
		}
		// Formatting is idempotent (re-formatting keeps the shebang).
		if f2 := FormatSource(f.Source, c.st); f2.Source != f.Source {
			t.Fatalf("%s: format not idempotent:\n%q\n%q", c.st, f.Source, f2.Source)
		}
		// The shebang does not leak into run output.
		if r := RunSource(c.src, c.st); r.Stdout != c.wantRun {
			t.Fatalf("%s: run output %q, want %q", c.st, r.Stdout, c.wantRun)
		}
	}

	// A source without a shebang is unaffected.
	if f := FormatSource("x:=1\n", "gad"); f.Source != "x := 1\n" {
		t.Fatalf("no-shebang changed: %q", f.Source)
	}
}

// TestFormatDetachedModuleDoc verifies the formatter round-trips the unified doc
// convention: a detached `/** … **/` (module/section) doc keeps its blank-line
// separation, an attached block collapses to `///`, and a three-star `/*** … ***/`
// block is normalized to two stars.
func TestFormatDetachedModuleDoc(t *testing.T) {
	// Detached block: blank line preserved, stays a block.
	if f := FormatSource("/**\nModule doc.\n**/\n\nconst x = 1\n", "gad"); f.Source != "/**\nModule doc.\n**/\n\nconst x = 1\n" {
		t.Fatalf("detached block: %q", f.Source)
	}
	// Three-star normalized to two-star (and blank line kept).
	if f := FormatSource("/***\nModule doc.\n***/\n\nconst x = 1\n", "gad"); f.Source != "/**\nModule doc.\n**/\n\nconst x = 1\n" {
		t.Fatalf("root normalize: %q", f.Source)
	}
	// Attached single-line block collapses to a `///` statement doc.
	if f := FormatSource("/**\nDoc of x.\n**/\nconst x = 1\n", "gad"); f.Source != "/// Doc of x.\nconst x = 1\n" {
		t.Fatalf("attached collapse: %q", f.Source)
	}
}
