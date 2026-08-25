package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// labelsOf returns the member labels reported at the caret just after the `.`.
func labelsOf(t *testing.T, src string) []string {
	t.Helper()
	caret := strings.Index(src, "u.") + len("u.")
	require.GreaterOrEqual(t, caret, len("u."), "source must contain a `u.` receiver")
	items, ok := memberCompletions(src, caret)
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
