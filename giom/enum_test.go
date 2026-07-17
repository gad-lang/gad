package giom

import (
	"strings"
	"testing"

	"github.com/gad-lang/gad"
	gnode "github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	giomnode "github.com/gad-lang/gad/giom/node"
	giomparser "github.com/gad-lang/gad/giom/parser"
)

// TestEnumDeclarations covers the `@enum IDENT ( … )` directive: its field body
// mirrors a `@var` declaration (comma- or newline-separated `Name` / `Name =
// value`) and also accepts the Gad enum extras `bit` and `+`/`-`.
func TestEnumDeclarations(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "single-line comma with explicit value",
			src: "@enum Perm (Read, Write, Exec = 10, Delete)\n" +
				"@main\n\t| {=Perm.Read.value} {=Perm.Write.value} {=Perm.Exec.value} {=Perm.Delete.value}\n",
			want: "1 2 10 11",
		},
		{
			name: "multi-line var-like body",
			src: "@enum Color (\n\tRed\n\tGreen\n\tBlue\n)\n" +
				"@main\n\t| {=Color.Red.value} {=Color.Green.value} {=Color.Blue.value}\n",
			want: "1 2 3",
		},
		{
			name: "bit flags with expression value",
			src: "@enum Flags (bit List, Detail, Create, Read = List | Detail)\n" +
				"@main\n\t| {=Flags.List.value} {=Flags.Create.value} {=Flags.Read.value}\n",
			want: "1 4 3",
		},
		{
			name: "signed fields",
			src: "@enum Signed (-Low, Lower, +High, Higher)\n" +
				"@main\n\t| {=Signed.Low.value} {=Signed.Lower.value} {=Signed.High.value} {=Signed.Higher.value}\n",
			want: "-1 -2 3 4",
		},
		{
			name: "index by name and member introspection",
			src: "@enum Perm (Read, Write, Exec = 10)\n" +
				"@main\n\t| {=Perm[\"Write\"].value} {=Perm.Exec.name} {=Perm.Exec.index}\n",
			want: "2 Exec 2",
		},
		{
			name: "iterate in declaration order",
			src: "@enum Perm (Read, Write, Exec = 10)\n" +
				"@main\n\t@for name, v in Perm\n\t\t| {=name}={=v.value};\n",
			want: "Read=1;Write=2;Exec=10;",
		},
		{
			name: "enum scoped inside a component",
			src: "@comp badge(level)\n" +
				"\t@enum Level (info, warn, err)\n" +
				"\tspan {= level == Level.warn.value ? \"!\" : \"-\" }\n" +
				"@main\n\t+badge(2)\n",
			want: "<span>!</span>",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := strings.TrimSpace(renderGiom(t, tc.src, gad.Dict{}))
			if got != tc.want {
				t.Fatalf("render mismatch\n got: %q\nwant: %q", got, tc.want)
			}
		})
	}
}

// TestEnumTranspile checks the scanner/parser lower `@enum IDENT ( … )` to a Gad
// `enum IDENT { … }` statement.
func TestEnumTranspile(t *testing.T) {
	src := "@enum Perm (Read, Write, Exec = 10)\n"
	fs := source.NewFileSet()
	f := fs.AddFileData("t.giom", -1, []byte(src))
	p := giomparser.NewParser(f)
	parsed, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(parsed.Stmts) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(parsed.Stmts))
	}
	es, ok := parsed.Stmts[0].(*giomnode.EnumStmt)
	if !ok {
		t.Fatalf("expected *giomnode.EnumStmt, got %T", parsed.Stmts[0])
	}
	if es.Name != "Perm" {
		t.Fatalf("enum name: got %q want %q", es.Name, "Perm")
	}
	code := gnode.Code(giomnode.Convert(parsed.Stmts))
	if want := "enum Perm {Read, Write, Exec = 10}"; code != want {
		t.Fatalf("transpiled code:\n got: %q\nwant: %q", code, want)
	}
}

// TestEnumInvalidField reports a parse error for a malformed field, resolving to
// the giom source line.
func TestEnumInvalidField(t *testing.T) {
	src := "@enum Bad (1abc)\n@main\n\tp x\n"
	fs := source.NewFileSet()
	f := fs.AddFileData("t.giom", -1, []byte(src))
	p := giomparser.NewParser(f)
	if _, err := p.ParseFile(); err == nil {
		t.Fatal("expected a parse error for an invalid enum field")
	}
}
