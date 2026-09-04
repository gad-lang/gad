package parser

import (
	"strings"
	"testing"

	gadxnode "github.com/gad-lang/gad/gadx/node"
)

// A void element has no closing tag, so it can hold no children. Taking a
// following text run as its body swallowed the text — the element renders
// nothing — and left the space between two of them unable to round-trip.
func TestVoidElementTakesNoBody(t *testing.T) {
	for _, name := range []string{"meta", "br", "img", "input", "hr", "embed", "param", "track", "wbr"} {
		if !gadxnode.IsSelfClosing(name) {
			t.Errorf("%s is a void element but IsSelfClosing says otherwise", name)
		}
	}
	if gadxnode.IsSelfClosing("div") {
		t.Error("div is not a void element")
	}
	// HTML tag names are case-insensitive.
	if !gadxnode.IsSelfClosing("META") {
		t.Error("IsSelfClosing should fold case")
	}
}

// The scanner and the node builder ask the same list, so they cannot disagree
// about whether an element closes itself.
func TestVoidElementRegionAndNodesAgree(t *testing.T) {
	src := "<div><embed src=\"a\"><span>x</span></div>"
	end, ok := htmlRegionEnd(src, 0)
	if !ok || end != len(src) {
		t.Fatalf("htmlRegionEnd = %d, %v; want %d, true", end, ok, len(src))
	}
	b := &htmlBuilder{src: src}
	nodes, _ := b.parseNodes(0)
	if len(nodes) != 1 {
		t.Fatalf("got %d root nodes, want 1", len(nodes))
	}
	div, okd := nodes[0].(*gadxnode.TagStmt)
	if !okd {
		t.Fatalf("root is %T, want a TagStmt", nodes[0])
	}
	var names []string
	for _, c := range div.Body {
		if tag, ok := c.(*gadxnode.TagStmt); ok {
			names = append(names, tag.Name)
		}
	}
	// embed is void: it must be a sibling of span, not its parent.
	if strings.Join(names, ",") != "embed,span" {
		t.Errorf("children = %v, want embed and span as siblings", names)
	}
}
