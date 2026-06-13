package gadbridge

import (
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
