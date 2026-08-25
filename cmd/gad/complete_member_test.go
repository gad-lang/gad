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
