package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// labelsOf returns the member labels reported at the caret just after the `.`.
func labelsOf(t *testing.T, src string) []string {
	t.Helper()
	return labelsOfName(t, "t.gad", src)
}

func labelsOfName(t *testing.T, name, src string) []string {
	t.Helper()
	caret := strings.Index(src, "u.") + len("u.")
	require.GreaterOrEqual(t, caret, len("u."), "source must contain a `u.` receiver")
	items, ok := memberCompletions(name, src, caret)
	require.True(t, ok, "should be a member-access context")
	var labels []string
	for _, it := range items {
		labels = append(labels, it.Label)
	}
	return labels
}

// TestMemberCompletionDict checks dict-key completion for a plain receiver.
func TestMemberCompletionDict(t *testing.T) {
	labels := labelsOf(t, "u := {name: \"joe\", admin: true}\nx := u.\n")
	require.Contains(t, labels, "name")
	require.Contains(t, labels, "admin")
}

// TestMemberCompletionLoopVar checks that a receiver bound inside a loop resolves
// its members: the eval must keep the enclosing `for` block balanced (replacing
// the caret line in place) so the loop variable is in scope on the first
// iteration, instead of cutting the source and leaving the block open.
func TestMemberCompletionLoopVar(t *testing.T) {
	src := "users := [{name: \"joe\", admin: true}]\nfor i, u in users {\n  x := u.\n}\n"
	labels := labelsOf(t, src)
	require.Contains(t, labels, "name")
	require.Contains(t, labels, "admin")
}

// TestMemberCompletionGadtLoopVar checks the mixed-template (`.gadt`) path: the
// loop variable's dict keys resolve even though the code lives in `{% … %}`
// islands interleaved with literal text (and a leading doc-comment island whose
// prose contains `{% … %}`, which must not corrupt the extraction).
func TestMemberCompletionGadtLoopVar(t *testing.T) {
	src := "{%--\n/** doc mentioning `{%= x %}` here **/\n--%}\n" +
		"{% users := [{name: \"joe\", admin: true}] %}\n" +
		"{%-- for i, u in users begin %}\n<li>{%= u. %}</li>\n{%-- end %}\n"
	labels := labelsOfName(t, "t.gadt", src)
	require.Contains(t, labels, "name")
	require.Contains(t, labels, "admin")
}

// TestMemberCompletionGadxLoopVar checks the `.gadx` path: the front-end lowers
// to Gad with synthetic nodes whose positions do not slice back to source, so the
// loop header is rebuilt from the variables' names (not source spans).
func TestMemberCompletionGadxLoopVar(t *testing.T) {
	src := "~~\nusers := [{name: \"joe\", admin: true}]\n~~\n" +
		"ul\n\t@for i, u in users\n\t\tli {u.}\n"
	labels := labelsOfName(t, "t.gadx", src)
	require.Contains(t, labels, "name")
	require.Contains(t, labels, "admin")
}

// TestMemberCompletionGadxComplexIterable checks a non-identifier iterable
// (`users[:]`): the header is rendered from the AST's String() (not source
// spans), so complex iterables resolve in the position-lossy `.gadx` lowering.
func TestMemberCompletionGadxComplexIterable(t *testing.T) {
	src := "~~\nusers := [{name: \"joe\", admin: true}]\n~~\n" +
		"ul\n\t@for i, u in users[:]\n\t\tli {u.}\n"
	labels := labelsOfName(t, "t.gadx", src)
	require.Contains(t, labels, "name")
	require.Contains(t, labels, "admin")
}

// TestMemberCompletionTypedArray checks member completion on a typed array value
// and on its type: the body members (fields, properties, methods) with their
// source docs; an int index reaches the items, so `u.` lists only members.
func TestMemberCompletionTypedArray(t *testing.T) {
	decl := "type nums []int {\n" +
		"  /// the label\n" +
		"  label = \"x\"\n" +
		"  props { total => 1 }\n" +
		"  methods {\n" +
		"    /// sums the items\n" +
		"    sum() => 0\n" +
		"  }\n" +
		"}\n"
	check := func(src string) {
		caret := strings.Index(src, "u.") + len("u.")
		items, ok := memberCompletions("t.gad", src, caret)
		require.True(t, ok)
		got := map[string][2]string{}
		for _, it := range items {
			got[it.Label] = [2]string{it.Kind, it.Doc}
		}
		require.Equal(t, map[string][2]string{
			"label": {"field", "the label"},
			"total": {"property", ""},
			"sum":   {"method", "sums the items"},
		}, got)
	}
	check(decl + "u := nums(1, 2)\nx := u.\n") // a value
	check(decl + "u := nums\nx := u.\n")       // the type

	// Without a body a typed array has no members (its items are reached by index).
	src := "type nums []int\nu := nums(1, 2)\nx := u.\n"
	caret := strings.Index(src, "u.") + len("u.")
	items, ok := memberCompletions("t.gad", src, caret)
	require.True(t, ok)
	require.Empty(t, items)
}

// TestMemberCompletionTypedArrayThis checks `this.` inside a typed array type's
// method lists the body members, like inside a class.
func TestMemberCompletionTypedArrayThis(t *testing.T) {
	src := "type nums []int {\n  label = \"x\"\n  methods {\n    sum() {\n      this.\n    }\n  }\n}\n"
	caret := strings.Index(src, "this.") + len("this.")
	items, ok := memberCompletions("t.gad", src, caret)
	require.True(t, ok)
	var labels []string
	for _, it := range items {
		labels = append(labels, it.Label)
	}
	require.ElementsMatch(t, []string{"label", "sum"}, labels)
}
