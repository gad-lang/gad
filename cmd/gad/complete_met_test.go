package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// metLabels returns the completion labels at the caret marked by `‸` in src (the
// marker is stripped before completion). It exercises the `met` special-receiver
// path (this/$old/new).
func metLabels(t *testing.T, src string) []string {
	t.Helper()
	caret := strings.Index(src, "‸")
	require.GreaterOrEqual(t, caret, 0, "source must contain the caret marker ‸")
	clean := strings.Replace(src, "‸", "", 1)
	items, ok := memberCompletions("buffer.gad", clean, caret)
	require.True(t, ok, "should be a member/met context")
	var labels []string
	for _, it := range items {
		labels = append(labels, it.Label)
	}
	return labels
}

// TestMetThisMethodMembers: `this.` in `met Class.method(this)` lists the class's
// fields, properties and methods — including members pulled in from a `use`d
// mixin (merged into the class).
func TestMetThisMethodMembers(t *testing.T) {
	src := "mixin Sized { methods { area() => 1 } }\n" +
		"class Shape { legs = 4; use Sized; props { kind => \"s\" }; methods { walk() => 1 } }\n" +
		"met Shape.run(this) { this.‸ }\n"
	labels := metLabels(t, src)
	require.Contains(t, labels, "legs") // own field
	require.Contains(t, labels, "kind") // own property
	require.Contains(t, labels, "walk") // own method
	require.Contains(t, labels, "area") // pulled in from mixin Sized (use)
}

// TestMetThisParentMembers: `this.` includes members inherited from a parent class
// (`*Base`).
func TestMetThisParentMembers(t *testing.T) {
	src := "class Base { hp = 10; methods { heal() => 1 } }\n" +
		"class Hero { *Base; name = \"?\" }\n" +
		"met Hero.act(this) { this.‸ }\n"
	labels := metLabels(t, src)
	require.Contains(t, labels, "name") // own
	require.Contains(t, labels, "hp")   // inherited field
	require.Contains(t, labels, "heal") // inherited method
}

// TestMetNewConstructorMembers: `new.` in a constructor `met Class(new)` lists the
// class instance members (what construction populates), parents included.
func TestMetNewConstructorMembers(t *testing.T) {
	src := "class Base { hp = 10 }\n" +
		"class Hero { *Base; name = \"?\"; methods { attack() => 1 } }\n" +
		"met Hero(new) { new.‸ }\n"
	labels := metLabels(t, src)
	require.Contains(t, labels, "name")
	require.Contains(t, labels, "attack")
	require.Contains(t, labels, "hp")
}

// TestMetOldOverrideMembers: `this.` in an override `met ~Class.method($old, this)`
// resolves through the sibling `this` parameter to the class members.
func TestMetOldOverrideMembers(t *testing.T) {
	src := "class Hero { name = \"?\"; methods { greet() => 1 } }\n" +
		"met ~Hero.greet($old, this) { this.‸ }\n"
	labels := metLabels(t, src)
	require.Contains(t, labels, "name")
	require.Contains(t, labels, "greet")
}

// TestMetThisMixinInterface: for a mixin target, `this.` lists the mixin's own
// members and the members required by its `this { … }` interface.
func TestMetThisMixinInterface(t *testing.T) {
	src := "mixin Movable { this { pos() <int> }; methods { step() => 1 } }\n" +
		"met Movable.jump(this) { this.‸ }\n"
	labels := metLabels(t, src)
	require.Contains(t, labels, "step") // mixin method
	require.Contains(t, labels, "pos")  // required by the this-interface
}

// TestMetThisMixinDeep: a mixin extending a parent mixin exposes the parent's
// members and this-interface too.
func TestMetThisMixinDeep(t *testing.T) {
	src := "mixin Base { this { id() <int> }; methods { tag() => 1 } }\n" +
		"mixin Sub { *Base; this { pos() <int> }; methods { step() => 1 } }\n" +
		"met Sub.jump(this) { this.‸ }\n"
	labels := metLabels(t, src)
	require.Contains(t, labels, "step") // own
	require.Contains(t, labels, "pos")  // own this-interface
	require.Contains(t, labels, "tag")  // parent mixin method
	require.Contains(t, labels, "id")   // parent this-interface
}

// litLabels returns completion labels at the ‸ caret marked in src (stripped
// before completion), exercising the in-literal `this.` path.
func litLabels(t *testing.T, src string) []string {
	t.Helper()
	caret := strings.Index(src, "‸")
	require.GreaterOrEqual(t, caret, 0, "source must contain the caret marker ‸")
	clean := strings.Replace(src, "‸", "", 1)
	items, ok := memberCompletions("buffer.gad", clean, caret)
	require.True(t, ok, "should be a member context")
	var labels []string
	for _, it := range items {
		labels = append(labels, it.Label)
	}
	return labels
}

// TestLiteralThisMixinOwnAndInterface: `this.` inside a mixin literal being edited
// lists the mixin's own members and its `this { … }` interface requirements.
func TestLiteralThisMixinOwnAndInterface(t *testing.T) {
	src := "mixin M {\n" +
		"  count = 0\n" +
		"  this { size() <int> }\n" +
		"  props { doubled => this.count }\n" +
		"  methods { area() => this.‸ }\n" +
		"}\n"
	labels := litLabels(t, src)
	require.Contains(t, labels, "count")   // own field
	require.Contains(t, labels, "doubled") // own property
	require.Contains(t, labels, "area")    // own method
	require.Contains(t, labels, "size")    // required by the this-interface
}

// TestLiteralThisMixinParent: `this.` in a mixin literal includes members of a
// `*Parent` mixin resolved from the same file's AST (no evaluation).
func TestLiteralThisMixinParent(t *testing.T) {
	src := "mixin Base { hp = 10; methods { heal() => 1 } }\n" +
		"mixin M {\n" +
		"  *Base\n" +
		"  name = \"?\"\n" +
		"  methods { go() => this.‸ }\n" +
		"}\n"
	labels := litLabels(t, src)
	require.Contains(t, labels, "name") // own
	require.Contains(t, labels, "go")   // own
	require.Contains(t, labels, "hp")   // parent field
	require.Contains(t, labels, "heal") // parent method
}

// TestLiteralThisClass: `this.` inside a plain class literal being edited lists
// the class's own members.
func TestLiteralThisClass(t *testing.T) {
	src := "class C {\n  x = 1\n  props { p => 1 }\n  methods { m() => this.‸ }\n}\n"
	labels := litLabels(t, src)
	require.Contains(t, labels, "x")
	require.Contains(t, labels, "p")
	require.Contains(t, labels, "m")
}
