package filepath

import (
	"testing"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/test_helper"
)

func expectRun(t *testing.T, param, script string, opts *test_helper.VMTestOpts, expect gad.Object) {
	t.Helper()
	if opts == nil {
		opts = test_helper.NewVMTestOpts()
	}
	opts = opts.Module("filepath", Module)
	if param != "" {
		param = "param(" + param + ")"
	}
	script = param + `;const fp = import("filepath");` + script
	test_helper.VMTestExpectRun(t, script, opts, expect)
}
