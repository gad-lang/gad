package filepath

import (
	"testing"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/test_helper"
)

func expectRun(t *testing.T, param, script string, opts *testhelper.VMTestOpts, expect gad.Object) {
	t.Helper()
	if opts == nil {
		opts = testhelper.NewVMTestOpts()
	}
	opts = opts.Module("filepath", Module)
	if param != "" {
		param = "param(" + param + ")"
	}
	script = param + `;const fp = import("filepath");` + script
	testhelper.VMTestExpectRun(t, script, opts, expect)
}
