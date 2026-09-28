package ide

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gad-lang/gad"
	"github.com/stretchr/testify/require"
)

// TestBuildModuleMapSourceType verifies the run/debug module map resolves a
// name written without an extension in the order of the source's dialect —
// the request's sourceType when given, else its path's extension.
func TestBuildModuleMapSourceType(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"m.gad", "m.gadx"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), nil, 0o644))
	}
	resolve := func(kind gad.SourceKind) string {
		imp := buildModuleMap(dir, kind, nil, false).Get("./m").(gad.ExtImporter)
		name, err := imp.Name()
		require.NoError(t, err)
		return filepath.Base(name)
	}
	require.Equal(t, "m.gad", resolve(gad.SourceKindGad))
	require.Equal(t, "m.gad", resolve(gad.SourceKindGadt))
	require.Equal(t, "m.gadx", resolve(gad.SourceKindGadx))

	require.Equal(t, gad.SourceKindGadx, requestSourceType("page.gad", "gadx")) // sourceType wins
	require.Equal(t, gad.SourceKindGadt, requestSourceType("x.gad", "gadTemplate"))
	require.Equal(t, gad.SourceKindGadx, requestSourceType("page.gadx", "")) // else the path
	require.Equal(t, gad.SourceKindGad, requestSourceType("", ""))
}
