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
