package gadbridge

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// The Gadx formatter writes the same source again (it is idempotent) and the
// result lowers to the same Gad code: a blank line of a literal block once, a
// second `class` attribute kept, an interpolated run not quoted as text.
func TestFormatGadxStable(t *testing.T) {
	for _, src := range []string{
		"@main\n\t@text\n\t\tDear {= name },\n\n\t\tThank you.\n",
		"@param title\n\n@main\n\th1 {=title}\n",
		// a fragment stays the HTML it is (the indentation form has none)
		"@main\n\tnav\n\t\t<>\n\t\t    <span>one</span>\n\t\t    <span>two</span>\n\t\t</>\n",
		// a value with blank space around it too (that form cannot write it)
		"@param title\n\n@main\n\tnav\n\t\t<a href=\"/\">\n\t\t    {=title}\n\t\t</a>\n",
		// classes merged into one attribute mean the same
		"@main\n\tdiv.card.wide[class=\"active\"] merged\n",
	} {
		opts := GadxFormatOptions{Indent: "\t"}
		res := FormatGadx(src, opts)
		require.True(t, res.OK, src)
		again := FormatGadx(res.Source, opts)
		require.True(t, again.OK)
		require.Equal(t, res.Source, again.Source, "idempotent: %s", src)
		before, ok1 := GadxLowered(src)
		after, ok2 := GadxLowered(res.Source)
		require.True(t, ok1 && ok2)
		require.Equal(t, before, after, "the same meaning: %s", src)
	}
}

// A fragment and an element whose text has blank space around a value are
// written as the HTML they are, indented with the code around them.
func TestFormatGadxKeepsHTMLRegions(t *testing.T) {
	src := "@param title\n\n@main\n    nav\n        <a href=\"/\">\n            {=title}\n        </a>\n        <>\n            <span>one</span>\n        </>\n"
	res := FormatGadx(src, GadxFormatOptions{Indent: "\t"})
	require.True(t, res.OK)
	require.Equal(t, "@param title\n\n@main\n\tnav\n\t\t<a href=\"/\">\n\t\t    {=title}\n\t\t</a>\n\t\t<>\n\t\t    <span>one</span>\n\t\t</>\n", res.Source)
}
