package filepath

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gad-lang/gad"
)

func TestWalk(t *testing.T) {
	var (
		cwd, _ = os.Getwd()
		opts   = gad.NewVMTestOpts().Args(gad.Str(cwd))
		p      = func(s string) gad.Str {
			return gad.Str(filepath.Join(cwd, s))
		}
	)
	expectRun(t, `cwd`, `v := []; fp.walk(cwd, func(pth, info, err) { v = append(v, str(pth))}); return v`, opts,
		gad.Array{p("."), p("filepath.go"), p("filepath_test.go"), p("walk.go"),
			p("walk_test.go")})
	expectRun(t, `cwd`, `v := []; fp.walk(cwd, func(pth, info, err) { v = append(v, str(pth))};relative); return v`, opts,
		gad.Array{gad.Str("."), gad.Str("filepath.go"), gad.Str("filepath_test.go"), gad.Str("walk.go"),
			gad.Str("walk_test.go")})
	expectRun(t, `cwd`, `v := []; fp.walk(cwd, func(pth, info, err) { v = append(v, str(pth))}; dotSkip); return v`, opts,
		gad.Array{p("filepath.go"), p("filepath_test.go"), p("walk.go"),
			p("walk_test.go")})
	expectRun(t, `cwd`, `
v := []
fp.walk(cwd, func(pth, info, err) {
	len(v) == 3 && return fp.WalkSkip.Dir
	v = append(v, str(pth))
}; relative); return v`, opts,
		gad.Array{gad.Str("."), gad.Str("filepath.go"), gad.Str("filepath_test.go")})
	expectRun(t, `cwd`, `
v := []
fp.walk(cwd, func(pth, info, err) {
	len(v) == 3 && return fp.WalkSkip.All
	v = append(v, str(pth))
}; relative); return v`, opts,
		gad.Array{gad.Str("."), gad.Str("filepath.go"), gad.Str("filepath_test.go")})
}
