package filepath

import (
	"testing"

	"github.com/gad-lang/gad"
)

func expectRun(t *testing.T, param, script string, opts *gad.TestOpts, expect gad.Object) {
	t.Helper()
	if opts == nil {
		opts = gad.NewTestOpts()
	}
	opts = opts.Module("filepath", Module)
	if param != "" {
		param = "param(" + param + ")"
	}
	script = param + `;const fp = import("filepath");` + script
	gad.TestExpectRun(t, script, opts, expect)
}
