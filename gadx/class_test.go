package gadx

import (
	"strings"
	"testing"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/gadx/node"
	"github.com/gad-lang/gad/gadx/parser"
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
