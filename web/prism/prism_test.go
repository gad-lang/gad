package prism

import (
	"strings"
	"testing"
)

// The bundle has the grammars it says (the Gad family and the common
// languages) and the watcher of what comes in later (GadPrism).
func TestBundle(t *testing.T) {
	src := string(JS())
	if len(src) < 10000 {
		t.Fatalf("the bundle is %d bytes", len(src))
	}
	for _, want := range []string{"GadPrism", "MutationObserver", "gadx", "gadt", "golang", "dockerfile", "sql", "python", "toml", "yaml", "tsx"} {
		if !strings.Contains(src, want) {
			t.Errorf("the bundle has no %q", want)
		}
	}
}
