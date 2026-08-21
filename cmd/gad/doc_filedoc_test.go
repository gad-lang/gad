package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDocFileLevelDoubleStarProse: a leading, detached `/** … **/` block is the
// module prose (not just `/*** … ***/`), and when it leads with a `# Title` the
// synthesized `# name` module heading is suppressed (no double title).
func TestDocFileLevelDoubleStarProse(t *testing.T) {
	// Detached `/** … **/` with its own title.
	md, err := generateDoc("01_hello.gad", []byte("/**\n# Hello, Gad\n\nThe basics.\n**/\n\nexport A = 1\n"), true)
	require.NoError(t, err)
	require.Contains(t, md, "# Hello, Gad")
	require.Contains(t, md, "The basics.")
	require.NotContains(t, md, "# 01_hello", "no double title when prose has a heading")

	// Detached `/** … **/` without a title → synthesize `# name`.
	md2, err := generateDoc("mod.gad", []byte("/**\nplain prose.\n**/\n\nexport A = 1\n"), true)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(md2, "# mod"), "module heading synthesized:\n"+md2)
	require.Contains(t, md2, "plain prose.")

	// An ATTACHED `/** … **/` (no blank line) documents its declaration, not the
	// module, so it is not module prose.
	md3, err := generateDoc("mod.gad", []byte("/** attached doc **/\nexport A = 1\n"), true)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(md3, "# mod"), "attached doc is not module prose:\n"+md3)
}
