package parser

import (
	"strings"
	"testing"

	giomnode "github.com/gad-lang/gad/giom/node"
	"github.com/gad-lang/gad/token"
)

// TestParamParse checks that @param parses to a Gad `param` GenDecl in each
// supported form (single, positional + variadic, named + named-variadic).
func TestParamParse(t *testing.T) {
	tests := []struct {
		name      string
		src       string
		wantSpecs int
	}{
		{"single", "@param a\n@main\n    p x\n", 1},
		{"positional variadic", "@param (a, b, *rest)\n@main\n    p x\n", 3},
		{"named", "@param (a; b = 1, **named)\n@main\n    p x\n", 3},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			file := parseLine(t, tc.src)
			var ps *giomnode.ParamStmt
			for _, s := range file.Stmts {
				if p, ok := s.(*giomnode.ParamStmt); ok {
					ps = p
					break
				}
			}
			if ps == nil {
				t.Fatal("no ParamStmt in parsed file")
			}
			if ps.Decl == nil || ps.Decl.Tok != token.Param {
				t.Fatalf("expected param GenDecl, got %+v", ps.Decl)
			}
			if got := len(ps.Decl.Specs); got != tc.wantSpecs {
				t.Fatalf("specs = %d, want %d", got, tc.wantSpecs)
			}
		})
	}
}

// TestParamWriteGiom checks that @param round-trips through WriteGiom.
func TestParamWriteGiom(t *testing.T) {
	tests := []struct {
		src  string
		want string
	}{
		{"@param a\n@main\n    p x\n", "@param a"},
		{"@param (a, b, *rest)\n@main\n    p x\n", "@param (a, b, *rest)"},
		{"@param (a; b = 1, **named)\n@main\n    p x\n", "@param (a; b=1, **named)"},
	}
	for _, tc := range tests {
		out := transpileGiom(t, tc.src)
		if !strings.Contains(out, tc.want) {
			t.Fatalf("transpiled giom missing %q:\n%s", tc.want, out)
		}
	}
}
