package gadx

import (
	"bytes"
	"testing"

	"github.com/gad-lang/gad"
	gadxnode "github.com/gad-lang/gad/gadx/node"
	gadxparser "github.com/gad-lang/gad/gadx/parser"
	gnode "github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
)

// renderGadxInclude compiles and renders a .gadx template whose module map has
// each entry of mods registered as a plain Gad source module (for `@include`).
func renderGadxInclude(t *testing.T, src string, mods map[string]string) string {
	t.Helper()
	builtins := AppendBuiltins(gad.NewBuiltins())
	mm := gad.NewModuleMap()
	for name, s := range mods {
		mm.AddSourceModule(name, []byte(s))
	}
	opts := gad.CompileOptions{CompilerOptions: gad.CompilerOptions{
		FallbackFunc: CompileFallback,
		ModuleMap:    mm,
		ModuleFile:   "main.gadx",
	}}
	st := gad.NewSymbolTable(builtins.NameSet)
	cr, err := gad.Compile(st, []byte(src), opts)
	if err != nil {
		t.Fatalf("compile: %v\nsrc:\n%s", err, src)
	}
	var buf bytes.Buffer
	vm := gad.NewVM(builtins.Build(), cr.Bytecode)
	ret, err := vm.RunOpts(&gad.RunOpts{StdOut: &buf, Globals: gad.Dict{}})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if el, ok := ret.(Element); ok {
		if _, err := el.WriteTo(vm, &buf); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	return buf.String()
}

// TestGadxInclude verifies the `@include` directive compiles the named Gad
// source file(s) inline, so their declarations are visible to the template.
func TestGadxInclude(t *testing.T) {
	tests := []struct {
		name string
		src  string
		mods map[string]string
		want string
	}{
		{
			name: "single",
			src:  "@include \"data.gad\"\np {=title}\n",
			mods: map[string]string{"data.gad": `title := "Hello"`},
			want: "<p>Hello</p>",
		},
		{
			name: "multiple parenthesized",
			src:  "@include (\"a.gad\", \"b.gad\")\np {=x + y}\n",
			mods: map[string]string{"a.gad": "x := 3", "b.gad": "y := 4"},
			want: "<p>7</p>",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := renderGadxInclude(t, tc.src, tc.mods); got != tc.want {
				t.Fatalf("render = %q, want %q", got, tc.want)
			}
		})
	}
}

// findIncludeStmt walks gadx statements for the lowered Gad *IncludeStmt (each
// @include becomes a CodeStmt wrapping it).
func findIncludeStmt(t *testing.T, stmts gnode.Stmts) *gnode.IncludeStmt {
	t.Helper()
	for _, s := range stmts {
		if inc, ok := s.(*gnode.IncludeStmt); ok {
			return inc
		}
		if cs, ok := s.(*gadxnode.CodeStmt); ok {
			if inc := findIncludeStmt(t, cs.Stmts); inc != nil {
				return inc
			}
		}
	}
	return nil
}

// TestGadxIncludePreservesPositions verifies the @include directive lowers to a
// Gad `include` whose path literals keep their ORIGINAL .gadx source positions
// (so a bad path reports the right line/column), for both the parenthesized and
// the bare single-string forms.
func TestGadxIncludePreservesPositions(t *testing.T) {
	cases := []struct {
		src     string
		wantOff int // byte offset of the path string's opening quote in src
	}{
		// @include ("data.gad")
		// `@include ` = 9 bytes, `(` at 9, `"` at 10.
		{"@include (\"data.gad\")\np hi\n", 10},
		// @include "data.gad"
		// `@include ` = 9 bytes, `"` at 9.
		{"@include \"data.gad\"\np hi\n", 9},
	}
	for _, tc := range cases {
		f := source.NewFileSet().AddFileData("t.gadx", -1, []byte(tc.src))
		file, err := gadxparser.NewParser(f).ParseFile()
		if err != nil {
			t.Fatalf("parse: %v\nsrc: %q", err, tc.src)
		}
		inc := findIncludeStmt(t, file.Stmts)
		if inc == nil {
			t.Fatalf("no IncludeStmt in: %q", tc.src)
		}
		got := source.MustFilePosition(f, inc.Paths[0].Pos()).Offset
		if got != tc.wantOff {
			t.Fatalf("path offset = %d, want %d (src %q)", got, tc.wantOff, tc.src)
		}
	}
}
