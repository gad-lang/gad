package gadx

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gad-lang/gad"
	gadxnode "github.com/gad-lang/gad/gadx/node"
	gadxparser "github.com/gad-lang/gad/gadx/parser"
	"github.com/gad-lang/gad/parser/source"
)

// TestDirectiveSignatureRender covers the full signature grammar the `@func`,
// `@comp` and `@main` directives accept — parameter types, type parameters
// (`[T constraint]`) and return types (`<ret>`) — by rendering templates that
// exercise each shape. The signature is lowered to a Gad function header, so the
// same syntax Gad functions accept works verbatim in the directives.
func TestDirectiveSignatureRender(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		globals gad.Dict
		want    string
	}{
		{
			name: "func param and return types",
			src:  "@func add(a int, b int) <int>\n    | {=a + b}\n@main\n    +add(2, 3)\n",
			want: "5",
		},
		{
			name: "comp param type",
			src:  "@comp box(title str)\n    div {=title}\n@main\n    +box(\"hi\")\n",
			want: "<div>hi</div>",
		},
		{
			name:    "main param type",
			src:     "@main(n int)\n    p {=n}\n",
			globals: gad.Dict{"n": gad.Int(7)},
			want:    "<p>7</p>",
		},
		{
			name: "func type parameter",
			src:  "@func id[T any](v T) <T>\n    | {=v}\n@main\n    +id(\"z\")\n",
			want: "z",
		},
		{
			name: "comp type parameter",
			src:  "@comp cell[T any](v T)\n    td {=v}\n@main\n    +cell(42)\n",
			want: "<td>42</td>",
		},
		{
			name: "positional and named default params",
			src:  "@comp greeting(name; greet = \"Hello\")\n    p {=greet}, {=name}\n@main\n    +greeting(\"Bob\")\n",
			want: "<p>Hello, Bob</p>",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := strings.TrimSpace(renderGadx(t, tc.src, tc.globals))
			if got != tc.want {
				t.Fatalf("render = %q, want %q\nsrc:\n%s", got, tc.want, tc.src)
			}
		})
	}
}

// TestDirectiveParamTypeEnforced verifies a directive parameter's declared type
// is enforced at call time: passing an argument of the wrong type is a TypeError
// naming the parameter and both types.
func TestDirectiveParamTypeEnforced(t *testing.T) {
	_, err := renderGadxErr(t, "@func add(a int, b int) <int>\n    | {=a + b}\n@main\n    +add(\"x\", 3)\n")
	if err == nil {
		t.Fatal("expected a type error, got nil")
	}
	if msg := err.Error(); !strings.Contains(msg, "expected int, found str") {
		t.Fatalf("error = %q, want it to mention `expected int, found str`", msg)
	}
}

// TestDirectiveTypeParamConstraintEnforced verifies a type parameter's
// constraint is enforced: `[T number]` rejects a non-number argument bound to a
// `T`-typed parameter.
func TestDirectiveTypeParamConstraintEnforced(t *testing.T) {
	_, err := renderGadxErr(t, "@func inc[T number](v T) <T>\n    | {=v}\n@main\n    +inc(\"x\")\n")
	if err == nil {
		t.Fatal("expected a type error, got nil")
	}
	if msg := err.Error(); !strings.Contains(msg, "expected number, found str") {
		t.Fatalf("error = %q, want it to mention `expected number, found str`", msg)
	}
}

// TestDirectiveSignaturePositions verifies the source positions of a directive
// signature's parts (parameter identifiers, parameter types, type parameters and
// return types) are preserved: they point back at the exact offsets in the .gadx
// source, so diagnostics and editor navigation land on the right token. Before
// the signature was lowered through the Gad parser, these all collapsed to the
// directive's start.
func TestDirectiveSignaturePositions(t *testing.T) {
	// Offsets are 0-based byte positions within src.
	//  @func add(a int, b int) <int>
	//  0123456789...
	//  a@10, its type int@12, b@17, its type int@19, return int@25.
	src := "@func add(a int, b int) <int>\n    | {=a + b}\n"
	f, decl := parseFuncDecl(t, src)
	off := func(p source.Pos) int { return source.MustFilePosition(f, p).Offset }

	a := decl.Params.Args.Values[0]
	if got := off(a.Ident.Pos()); got != 10 {
		t.Errorf("param a ident offset = %d, want 10", got)
	}
	if got := off(a.Type[0].Pos()); got != 12 {
		t.Errorf("param a type offset = %d, want 12", got)
	}
	b := decl.Params.Args.Values[1]
	if got := off(b.Ident.Pos()); got != 17 {
		t.Errorf("param b ident offset = %d, want 17", got)
	}
	if len(decl.Return) != 1 {
		t.Fatalf("return count = %d, want 1", len(decl.Return))
	}
	if got := off(decl.Return[0].Pos()); got != 25 {
		t.Errorf("return type offset = %d, want 25", got)
	}

	// Type parameters keep their positions too.
	//  @func id[T number](v T) <T>
	//  T@9, its constraint number@11, param v@19, its type T@21.
	src2 := "@func id[T number](v T) <T>\n    | {=v}\n"
	f2, decl2 := parseFuncDecl(t, src2)
	off2 := func(p source.Pos) int { return source.MustFilePosition(f2, p).Offset }
	if len(decl2.TypeParams) != 1 {
		t.Fatalf("type param count = %d, want 1", len(decl2.TypeParams))
	}
	tp := decl2.TypeParams[0]
	if got := off2(tp.Ident.Pos()); got != 9 {
		t.Errorf("type param ident offset = %d, want 9", got)
	}
	if got := off2(tp.Type[0].Pos()); got != 11 {
		t.Errorf("type param constraint offset = %d, want 11", got)
	}
}

// TestCompSignaturePositions verifies @comp signatures preserve positions the
// same way @func does; comp names may contain dashes (not valid Gad
// identifiers), which must not shift parameter positions.
func TestCompSignaturePositions(t *testing.T) {
	//  @comp date-box(when str)
	//  name `date-box`@6, param when@15, its type str@20.
	src := "@comp date-box(when str)\n    div {when}\n"
	f, comps := parseFile(t, src)
	off := func(p source.Pos) int { return source.MustFilePosition(f, p).Offset }
	if len(comps) != 1 {
		t.Fatalf("comp count = %d, want 1", len(comps))
	}
	c := comps[0]
	if c.Name != "date-box" {
		t.Fatalf("comp name = %q, want date-box", c.Name)
	}
	when := c.Params.Args.Values[0]
	if got := off(when.Ident.Pos()); got != 15 {
		t.Errorf("param when ident offset = %d, want 15", got)
	}
	if got := off(when.Type[0].Pos()); got != 20 {
		t.Errorf("param when type offset = %d, want 20", got)
	}
}

// renderGadxErr renders src and returns any render error (instead of failing).
func renderGadxErr(t *testing.T, src string) (string, error) {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "t.gadx")
	if err := os.WriteFile(p, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	err := newTestRender(t, dir).Render(&buf, p, gad.Dict{})
	return buf.String(), err
}

// parseFile parses src as a gadx file and returns the file plus its comps.
func parseFile(t *testing.T, src string) (*source.File, []*gadxnode.CompDecl) {
	t.Helper()
	f := source.NewFileSet().AddFileData("t.gadx", -1, []byte(src))
	file, err := gadxparser.NewParser(f).ParseFile()
	if err != nil {
		t.Fatalf("parse: %v\nsrc:\n%s", err, src)
	}
	return f, file.Comps
}

// parseFuncDecl parses src and returns the file plus its single @func decl.
func parseFuncDecl(t *testing.T, src string) (*source.File, *gadxnode.FuncDecl) {
	t.Helper()
	f := source.NewFileSet().AddFileData("t.gadx", -1, []byte(src))
	file, err := gadxparser.NewParser(f).ParseFile()
	if err != nil {
		t.Fatalf("parse: %v\nsrc:\n%s", err, src)
	}
	for _, s := range file.Stmts {
		if d, ok := s.(*gadxnode.FuncDecl); ok {
			return f, d
		}
	}
	t.Fatalf("no @func decl found in:\n%s", src)
	return nil, nil
}
