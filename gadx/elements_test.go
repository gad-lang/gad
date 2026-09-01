package gadx

import (
	"bytes"
	"testing"

	"github.com/gad-lang/gad"
)

// TestElementsFlattenOnAppend verifies that appending an *Elements fragment to a
// parent (another fragment or a tag) splices its items in individually, as if
// `parent ++= child.Items` — a fragment is never nested as a single node. This
// is what lets a component/slot/@main return an Elements and have its children
// merge into the caller's tree.
func TestElementsFlattenOnAppend(t *testing.T) {
	child := &Elements{}
	child.append(Text{gad.RawStr("a")})
	child.append(Text{gad.RawStr("b")})

	// Fragment += fragment → items spliced in (not a nested fragment node).
	parent := &Elements{}
	if _, err := parent.SelfAssignOpAdd(nil, child); err != nil {
		t.Fatalf("SelfAssignOpAdd: %v", err)
	}
	if len(parent.Items) != 2 {
		t.Fatalf("fragment += fragment: got %d items, want 2 (flattened)", len(parent.Items))
	}
	for _, it := range parent.Items {
		if _, ok := it.(*Elements); ok {
			t.Fatalf("fragment += fragment nested an *Elements instead of splicing its items")
		}
	}

	// Tag += fragment → the fragment's items become the tag's children.
	tag := &Tag{Name: "ul"}
	tag.append(child)
	if len(tag.Children) != 2 {
		t.Fatalf("tag += fragment: got %d children, want 2 (flattened)", len(tag.Children))
	}

	// Render matches the flattened structure.
	var buf bytes.Buffer
	if _, err := tag.WriteTo(nil, &buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	if got, want := buf.String(), "<ul>ab</ul>"; got != want {
		t.Fatalf("render = %q, want %q", got, want)
	}
}

// TestElementsIncSelfAssign verifies `el ++= other` appends each element of an
// iterable (here another fragment iterated as its items).
func TestElementsIncSelfAssign(t *testing.T) {
	a := &Elements{}
	a.append(Text{gad.Str("x")})
	b := &Elements{}
	b.append(Text{gad.Str("y")})
	b.append(Text{gad.Str("z")})

	if _, err := a.SelfAssignOpInc(nil, b); err != nil {
		t.Fatalf("SelfAssignOpInc: %v", err)
	}
	if len(a.Items) != 3 {
		t.Fatalf("el ++= other: got %d items, want 3", len(a.Items))
	}
}
