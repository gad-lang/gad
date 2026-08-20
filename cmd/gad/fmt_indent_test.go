package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseIndent(t *testing.T) {
	cases := []struct {
		in   string
		want string
		err  bool
	}{
		{"tab", "\t", false},
		{"tabs", "\t", false},
		{"1", " ", false},
		{"2", "  ", false},
		{"4", "    ", false},
		{"0", "", false},
		{"2t", "\t\t", false},
		{"3tabs", "\t\t\t", false},
		{`\t`, "\t", false},
		{"..", "..", false}, // an arbitrary custom leader
		{"  ", "  ", false}, // a literal of spaces
		{"", "", true},      // empty is rejected
		{"-1", "", true},    // negative width
	}
	for _, c := range cases {
		got, err := parseIndent(c.in)
		if c.err {
			require.Error(t, err, "input %q", c.in)
			continue
		}
		require.NoError(t, err, "input %q", c.in)
		require.Equal(t, c.want, got, "input %q", c.in)
	}
}

// TestFormatSourceGadxDispatch checks that a `.gadx` name is formatted with the
// Gadx source formatter (not parsed as plain Gad, which would fail), that the
// indentation unit is applied, and that embedded Gad is formatted with the Gad
// rules.
func TestFormatSourceGadxDispatch(t *testing.T) {
	src := []byte("@main\n    div[data=(a?b:c)]\n        p hi\n")

	// Default: one tab per level, embedded `a?b:c` normalized to `a ? b : c`.
	o := &fmtOptions{codeFlags: fmtFormatFlag()}
	out, err := o.formatSource("x.gadx", src, false)
	require.NoError(t, err)
	require.Contains(t, out, "@comp main()")
	require.Contains(t, out, "\tdiv")
	require.Contains(t, out, "a ? b : c")
	require.NotContains(t, out, "a?b:c")

	// --indent 2 → two spaces per level.
	o2 := &fmtOptions{codeFlags: fmtFormatFlag(), indentPrefix: "  "}
	out2, err := o2.formatSource("x.gadx", src, false)
	require.NoError(t, err)
	require.Contains(t, out2, "\n  div")
	require.False(t, strings.Contains(out2, "\n\tdiv"), "must not use tabs when --indent 2")
}
