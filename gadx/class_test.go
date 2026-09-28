package gadx

import (
	"strings"
	"testing"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/gadx/node"
	"github.com/gad-lang/gad/gadx/parser"
	gnode "github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
)

// `@class NAME { … }` declares a class — its body a Gad class body, spanning
// lines up to the balanced `}` —, `@export class` exports it too; a field may
// be typed by an anonymous class, named by its path.
func TestClassDirective(t *testing.T) {
	src := "@class listLayout { comp? enum { index_list } }\n" +
		"@export class PageOptions {\n" +
		"    [label=\"Tipo\"] title str = \"x\"\n" +
		"    posts class {\n" +
		"        layout? listLayout\n" +
		"    }\n" +
		"}\n" +
		"@main\n" +
		"    ~ o := PageOptions(; posts={layout: {}})\n" +
		"    p {=o.title} {=typeName(o.posts)} {=typeName(o.posts.layout)}\n"
	out := renderGadx(t, src, gad.Dict{})
	if want := "<p>x PageOptions.posts listLayout</p>"; !strings.Contains(out, want) {
		t.Fatalf("want %q in %q", want, out)
	}
}

// An @class is written back as it was declared — a member per line, behind
// its `@` — and the writing is stable.
func TestClassDirectiveFormat(t *testing.T) {
	src := "@export class PageOptions {\n\t[label=\"Tipo\"]\n\ttitle str = \"x\"\n}\n"
	format := func(s string) string {
		f, err := parser.NewParser(source.NewFileSet().AppendFileData("x.gadx", []byte(s))).ParseFile()
		if err != nil {
			t.Fatal(err)
		}
		var b strings.Builder
		(&node.File{Stmts: f.Stmts}).WriteGadx(node.NewGadxCodeContext(&b))
		return b.String()
	}
	once := format(src)
	if !strings.Contains(once, "@export class PageOptions {\n\t[label=\"Tipo\"]\n\ttitle str = \"x\"\n}") {
		t.Errorf("written back:\n%s", once)
	}
	if again := format(once); again != once {
		t.Errorf("not stable:\n%s\n---\n%s", once, again)
	}
}

// The gadx parser reads `@class` and `@export class` into a ClassStmt: its
// name, whether it is exported, and the Gad class declaration of its body —
// parsed at the body's place in the source, so positions inside it are the
// template's.
func TestClassDirectiveParse(t *testing.T) {
	src := "@class A { x int = 1 }\n" +
		"@export class B {\n" +
		"    a class { b str }\n" +
		"    c = f(1)\n" +
		"}\n"
	f, err := parser.NewParser(source.NewFileSet().AppendFileData("x.gadx", []byte(src))).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	var classes []*node.ClassStmt
	for _, s := range f.Stmts {
		if c, ok := s.(*node.ClassStmt); ok {
			classes = append(classes, c)
		}
	}
	if len(classes) != 2 {
		t.Fatalf("classes: %d", len(classes))
	}
	a, b := classes[0], classes[1]
	if a.Name != "A" || a.Exported || a.Decl == nil || len(a.Decl.Fields) != 1 {
		t.Errorf("A: %+v", a)
	}
	if b.Name != "B" || !b.Exported || b.Decl == nil || len(b.Decl.Fields) != 2 {
		t.Fatalf("B: %+v", b)
	}
	// the call of c's default is where it is written: line 4, column 9
	var call gnode.Expr
	for _, fl := range b.Decl.Fields {
		if fl.Name.Ident.Name == "c" {
			call = fl.Value
		}
	}
	if call == nil {
		t.Fatal("c has no default")
	}
	p := f.InputFile.Set().Position(call.Pos())
	if p.Line != 4 || p.Column != 9 {
		t.Errorf("f(1) at %d:%d, want 4:9", p.Line, p.Column)
	}
}

// A `@class` whose body does not close is not a class.
func TestClassDirectiveUnclosed(t *testing.T) {
	_, err := parser.NewParser(source.NewFileSet().AppendFileData("x.gadx", []byte("@class A {\n    x int\n"))).ParseFile()
	if err == nil {
		t.Error("an unclosed @class: want an error")
	}
}
