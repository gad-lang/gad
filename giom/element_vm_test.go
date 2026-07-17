package giom

import (
	"bytes"
	"context"
	"testing"

	"github.com/gad-lang/gad"
)

// runGadReturningTag compiles and runs a gad program with the giom builtins and
// returns the root *Tag it yields, rendered to a string.
func runGadReturningTag(t *testing.T, src string) string {
	t.Helper()
	builtins := AppendBuiltins(gad.NewBuiltins())
	st := gad.NewSymbolTable(builtins.NameSet)
	_, bc, err := gad.Compile(st, []byte(src), gad.CompileOptions{})
	if err != nil {
		t.Fatalf("compile: %v\nsrc:\n%s", err, src)
	}
	e := gad.NewEval(builtins.Build(), st, gad.CompileOptions{})
	ret, err := e.Run(context.Background(), bc)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	el, ok := ret.(Element)
	if !ok {
		t.Fatalf("expected Element, got %T (%v)", ret, ret)
	}
	var buf bytes.Buffer
	if _, err := el.WriteTo(e.VM, &buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	return buf.String()
}

// TestTreeBuildBlocks exercises the whole tree-building surface from gad source
// in the block/parent model the converter emits: a tag is `tag := giom.Tag(tag,
// name; **attrs)` inside a nested `{ }` block (so `tag` shadows the parent), the
// constructor links each tag to its parent, text is `giom.Text(tag, …)`, and the
// component/root returns its root tag.
func TestTreeBuildBlocks(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "nested tags and text",
			src: `
				tag := giom.Tag(nil)
				{
					tag := giom.Tag(tag, "div"; class="a")
					{
						giom.Text(tag, raw "hi")
						{
							tag := giom.Tag(tag, "span")
							{ giom.Text(tag, raw "x") }
						}
					}
				}
				return tag`,
			want: `<div class="a">hi<span>x</span></div>`,
		},
		{
			// Parentless constructor forms: giom.Tag() / giom.Tag(name) /
			// giom.Text(values) — no leading parent argument. Children are wired by
			// explicit appends instead of self-linking.
			name: "parentless constructor forms",
			src: `
				tag := giom.Tag()
				{
					div := giom.Tag("div"; id="x")
					div += giom.Text(raw "hi")
					tag += div
				}
				return tag`,
			want: `<div id="x">hi</div>`,
		},
		{
			// A nil first argument means "no parent" and is skipped, so
			// giom.Tag(nil, name) and giom.Tag(name) are equivalent.
			name: "nil parent equals parentless",
			src: `
				tag := giom.Tag(nil)
				{
					a := giom.Tag(nil, "b")
					giom.Text(a, raw "x")
					tag += a
				}
				return tag`,
			want: `<b>x</b>`,
		},
		{
			name: "many children",
			src: `
				tag := giom.Tag(nil)
				{
					tag := giom.Tag(tag, "ul")
					{
						{ tag := giom.Tag(tag, "li"); { giom.Text(tag, raw "a") } }
						{ tag := giom.Tag(tag, "li"); { giom.Text(tag, raw "b") } }
					}
				}
				return tag`,
			want: `<ul><li>a</li><li>b</li></ul>`,
		},
		{
			name: "dynamic attrs and single attr",
			src: `
				tag := giom.Tag(nil)
				{
					tag := giom.Tag(tag, "div")
					{
						tag["id"] = "main"
						tag.attrs += (; "data-x"="1")
						giom.Text(tag, raw "y")
					}
				}
				return tag`,
			want: `<div id="main" data-x="1">y</div>`,
		},
		{
			// class array + style string classified at build time, rendered
			// directly (regular attrs first, then class, then style).
			name: "class array and style",
			src: `
				tag := giom.Tag(nil)
				{
					tag := giom.Tag(tag, "div"; id="x", class=["a", "b"], style="color:red")
					{ giom.Text(tag, raw "y") }
				}
				return tag`,
			want: `<div id="x" class="a b" style="color:red">y</div>`,
		},
		{
			// Incremental single-attribute writes: class tokens accumulate.
			name: "incremental class and attr",
			src: `
				tag := giom.Tag(nil)
				{
					tag := giom.Tag(tag, "div")
					{
						tag["class"] = "a"
						tag["class"] = "b"
						tag["data-x"] = "1"
					}
				}
				return tag`,
			want: `<div data-x="1" class="a b"></div>`,
		},
		{
			name: "multi-value text node",
			src: `
				tag := giom.Tag(nil)
				{
					tag := giom.Tag(tag, "p")
					{ giom.Text(tag, raw "a", raw "b", raw "c") }
				}
				return tag`,
			want: `<p>abc</p>`,
		},
		{
			name: "anonymous fragment writes only children",
			src: `
				tag := giom.Tag(nil)
				{
					{ tag := giom.Tag(tag, "p"); { giom.Text(tag, raw "1") } }
					{ tag := giom.Tag(tag, "p"); { giom.Text(tag, raw "2") } }
				}
				return tag`,
			want: `<p>1</p><p>2</p>`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := runGadReturningTag(t, tc.src); got != tc.want {
				t.Fatalf("mismatch\n got: %s\nwant: %s", got, tc.want)
			}
		})
	}
}
