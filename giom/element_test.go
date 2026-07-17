package giom

import (
	"bytes"
	"testing"

	"github.com/gad-lang/gad"
)

// buildTagTreeVM returns a VM usable for Element.WriteTo (it only needs the
// object writer, which NewVM initialises).
func newElementVM() *gad.VM {
	return gad.NewVM(AppendBuiltins(gad.NewBuiltins()).Build(), nil)
}

func writeElement(t *testing.T, el Element) string {
	t.Helper()
	var buf bytes.Buffer
	if _, err := el.WriteTo(newElementVM(), &buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	return buf.String()
}

func TestTagWriteTo(t *testing.T) {
	// A named tag with an attribute and a text child.
	tag := NewTag(nil, "div", []Element{Text{gad.RawStr("hi")}},
		gad.KeyValueArray{{K: gad.Str("class"), V: gad.Str("a")}})
	if got, want := writeElement(t, tag), `<div class="a">hi</div>`; got != want {
		t.Fatalf("named tag\n got: %s\nwant: %s", got, want)
	}

	// A void element self-closes and ignores children.
	img := NewTag(nil, "img", nil, gad.KeyValueArray{{K: gad.Str("src"), V: gad.Str("/x")}})
	if got, want := writeElement(t, img), `<img src="/x" />`; got != want {
		t.Fatalf("void tag\n got: %s\nwant: %s", got, want)
	}

	// An anonymous tag writes only its children.
	frag := NewTag(nil, "", []Element{
		Text{gad.RawStr("<b>")},
		Text{gad.RawStr("hi")},
		Text{gad.RawStr("</b>")},
	}, nil)
	if got, want := writeElement(t, frag), `<b>hi</b>`; got != want {
		t.Fatalf("anon tag\n got: %s\nwant: %s", got, want)
	}
}

// TestNewTagParentLink verifies the constructor links a new tag to its parent.
func TestNewTagParentLink(t *testing.T) {
	root := NewTag(nil, "ul", nil, nil)
	NewTag(root, "li", []Element{Text{gad.RawStr("a")}}, nil)
	NewTag(root, "li", []Element{Text{gad.RawStr("b")}}, nil)
	if got, want := writeElement(t, root), `<ul><li>a</li><li>b</li></ul>`; got != want {
		t.Fatalf("parent link\n got: %s\nwant: %s", got, want)
	}
}

func TestTagOperators(t *testing.T) {
	vm := newElementVM()
	tag := NewTag(nil, "ul", nil, nil)

	// tag += child (single, via self-assign).
	if _, err := tag.SelfAssignOpAdd(vm, NewTag(nil, "li", []Element{Text{gad.RawStr("a")}}, nil)); err != nil {
		t.Fatal(err)
	}
	// tag ++= [c1, c2] (many).
	if _, err := tag.SelfAssignOpInc(vm, gad.Array{
		NewTag(nil, "li", []Element{Text{gad.RawStr("b")}}, nil),
		NewTag(nil, "li", []Element{Text{gad.RawStr("c")}}, nil),
	}); err != nil {
		t.Fatal(err)
	}
	if got, want := writeElement(t, tag), `<ul><li>a</li><li>b</li><li>c</li></ul>`; got != want {
		t.Fatalf("append\n got: %s\nwant: %s", got, want)
	}

	// tag[attr] = value (single attribute).
	div := NewTag(nil, "div", nil, nil)
	if err := div.IndexSet(vm, gad.Str("id"), gad.Str("main")); err != nil {
		t.Fatal(err)
	}
	// tag.attrs += kva (merge collection).
	if err := div.IndexSet(vm, gad.Str("attrs"), gad.KeyValueArray{
		{K: gad.Str("id"), V: gad.Str("main")},
		{K: gad.Str("class"), V: gad.Str("box")},
	}); err != nil {
		t.Fatal(err)
	}
	if got, want := writeElement(t, div), `<div id="main" class="box"></div>`; got != want {
		t.Fatalf("attrs\n got: %s\nwant: %s", got, want)
	}
}
