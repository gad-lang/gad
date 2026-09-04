package gadx

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gad-lang/gad"
)

func newTestRender(t *testing.T, workDir string) *Render {
	t.Helper()
	r := NewRender(workDir)
	r.TemplateDelay = 10 * time.Millisecond
	return r
}

func TestNewRender(t *testing.T) {
	dir := t.TempDir()
	r := NewRender(dir)
	if r.WorkDir() != dir {
		t.Fatalf("expected %q, got %q", dir, r.WorkDir())
	}
	if r.TemplateDelay != 15*time.Second {
		t.Fatalf("expected 15s default TemplateDelay, got %v", r.TemplateDelay)
	}
}

func TestNewRenderRelativePath(t *testing.T) {
	wd, _ := os.Getwd()
	r := NewRender(".")
	if r.WorkDir() != wd {
		t.Fatalf("expected abs %q, got %q", wd, r.WorkDir())
	}
}

func TestNewRenderEmpty(t *testing.T) {
	r := NewRender("")
	if r.WorkDir() != "" {
		t.Fatalf("expected empty, got %q", r.WorkDir())
	}
}

func TestRenderRunOptsFunc(t *testing.T) {
	dir := t.TempDir()
	src := "@main\n    p Hello"
	srcPath := filepath.Join(dir, "opts.gadx")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	r := newTestRender(t, dir)
	var (
		called    int
		gotFlags  gad.RunFlags
		gotStdOut bool
	)
	r.RunOptsFunc = func(opts *gad.RunOpts) {
		called++
		gotFlags = opts.Flags
		gotStdOut = opts.StdOut != nil
		// Add a VM-scoped env to prove the callback can customise the opts.
		opts.Env = gad.NewEnv(nil)
	}

	out, err := renderString(r, srcPath, gad.Dict{})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if out != "<p>Hello</p>" {
		t.Fatalf("unexpected output: %q", out)
	}
	if called != 1 {
		t.Fatalf("RunOptsFunc called %d times, want 1", called)
	}
	// The opts are pre-initialised with RunFlagSkipReceiverTypeCheck and StdOut.
	if !gotFlags.Has(gad.RunFlagSkipReceiverTypeCheck) {
		t.Fatalf("RunFlagSkipReceiverTypeCheck not set by default (flags=%d)", gotFlags)
	}
	if !gotStdOut {
		t.Fatal("StdOut not set on the opts passed to RunOptsFunc")
	}
}

func renderString(r *Render, filePath string, globals gad.Dict) (string, error) {
	var buf bytes.Buffer
	err := r.Render(&buf, filePath, globals)
	return buf.String(), err
}

func TestRenderBasic(t *testing.T) {
	dir := t.TempDir()
	src := `@main
    p Hello`
	srcPath := filepath.Join(dir, "basic.gadx")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	r := newTestRender(t, dir)
	out, err := renderString(r, srcPath, gad.Dict{})
	if err != nil {
		t.Fatal(err)
	}
	if out != "<p>Hello</p>" {
		t.Fatalf("expected `<p>Hello</p>`, got %q", out)
	}
}

func TestRenderWithModel(t *testing.T) {
	dir := t.TempDir()
	src := `@main
    h1 {= Model.Title}`
	srcPath := filepath.Join(dir, "model.gadx")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	r := newTestRender(t, dir)
	out, err := renderString(r, srcPath, gad.Dict{
		"Model": gad.Dict{"Title": gad.Str("Home")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != "<h1>Home</h1>" {
		t.Fatalf("expected `<h1>Home</h1>`, got %q", out)
	}
}

func TestRenderCachesBytecode(t *testing.T) {
	dir := t.TempDir()
	src := `@main
    p Hello`
	srcPath := filepath.Join(dir, "cache.gadx")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	var compileCount atomic.Int32
	r := newTestRender(t, dir)
	r.OnRender(func(first bool, mainFile string, files []string, lastTime time.Time, err error) {
		compileCount.Add(1)
	})

	// First render compiles.
	out1, err := renderString(r, srcPath, gad.Dict{})
	if err != nil {
		t.Fatal(err)
	}
	if out1 != "<p>Hello</p>" {
		t.Fatalf("expected `<p>Hello</p>`, got %q", out1)
	}
	if n := compileCount.Load(); n != 1 {
		t.Fatalf("expected 1 compile, got %d", n)
	}

	// Second render uses cache — no compile.
	out2, err := renderString(r, srcPath, gad.Dict{})
	if err != nil {
		t.Fatal(err)
	}
	if out2 != "<p>Hello</p>" {
		t.Fatalf("expected `<p>Hello</p>`, got %q", out2)
	}
	if n := compileCount.Load(); n != 1 {
		t.Fatalf("expected still 1 compile, got %d", n)
	}
}

func writeFileWithMtime(t *testing.T, path, content string, mtime time.Time) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatal(err)
	}
}

func TestRenderRecompilesOnFileChange(t *testing.T) {
	dir := t.TempDir()
	baseTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	writeFileWithMtime(t, filepath.Join(dir, "change.gadx"), `@main
    p Hello`, baseTime)

	srcPath := filepath.Join(dir, "change.gadx")

	var (
		compileCount atomic.Int32
		lastChanged  []string
		lastFirst    bool
	)
	r := newTestRender(t, dir)
	r.OnRender(func(first bool, mainFile string, files []string, lastTime time.Time, err error) {
		compileCount.Add(1)
		lastFirst = first
		lastChanged = files
	})

	// First render.
	if _, err := renderString(r, srcPath, gad.Dict{}); err != nil {
		t.Fatal(err)
	}
	if !lastFirst {
		t.Fatal("expected first=true on first render")
	}

	// Modify the file, with the mtime a save produces: the debounce is measured
	// from it, so a backdated one would read as an edit that settled long ago
	// and compile at once.
	writeFileWithMtime(t, srcPath, `@main
    p Changed`, time.Now())

	// Render immediately — change is detected but not yet compiled (debounce).
	r.TemplateDelay = 50 * time.Millisecond
	out, err := renderString(r, srcPath, gad.Dict{})
	if err != nil {
		t.Fatal(err)
	}
	if out != "<p>Hello</p>" {
		t.Fatalf("expected stale `<p>Hello</p>`, got %q", out)
	}
	if n := compileCount.Load(); n != 1 {
		t.Fatalf("expected still 1 compile before debounce, got %d", n)
	}

	// Wait for debounce, then render again to trigger recompile.
	time.Sleep(60 * time.Millisecond)
	out, err = renderString(r, srcPath, gad.Dict{})
	if err != nil {
		t.Fatal(err)
	}
	if out != "<p>Changed</p>" {
		t.Fatalf("expected `<p>Changed</p>`, got %q", out)
	}
	if n := compileCount.Load(); n != 2 {
		t.Fatalf("expected 2 compiles, got %d", n)
	}
	if lastFirst {
		t.Fatal("expected first=false on recompile")
	}
	if len(lastChanged) == 0 {
		t.Fatal("expected non-empty changed files on recompile")
	}
}

func TestRenderFileChangeDetectsImportedFile(t *testing.T) {
	dir := t.TempDir()
	baseTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	// Write an empty imported file (import tracking only needs it to exist).
	compPath := filepath.Join(dir, "comps.gadx")
	writeFileWithMtime(t, compPath, ``, baseTime)

	// Write a template that imports it.
	tplSrc := `@import "comps.gadx"
@main
    p Hello`
	tplPath := filepath.Join(dir, "imported.gadx")
	writeFileWithMtime(t, tplPath, tplSrc, baseTime)

	var (
		compileCount atomic.Int32
		lastChanged  []string
	)
	r := newTestRender(t, dir)
	r.TemplateDelay = 10 * time.Millisecond
	r.OnRender(func(first bool, mainFile string, files []string, lastTime time.Time, err error) {
		compileCount.Add(1)
		lastChanged = files
	})

	// First render.
	if _, err := renderString(r, tplPath, gad.Dict{}); err != nil {
		t.Fatal(err)
	}

	// Modify the imported component's mtime.
	// A save's own mtime: the debounce is measured from it (see
	// TestRenderRecompilesOnFileChange).
	writeFileWithMtime(t, compPath, ``, time.Now())

	// First post-change render — stamps changedAt, no recompile yet.
	if _, err := renderString(r, tplPath, gad.Dict{}); err != nil {
		t.Fatal(err)
	}
	if n := compileCount.Load(); n != 1 {
		t.Fatalf("expected still 1 compile after change detection, got %d", n)
	}

	// Wait for debounce, then render again to trigger recompile.
	time.Sleep(15 * time.Millisecond)
	if _, err := renderString(r, tplPath, gad.Dict{}); err != nil {
		t.Fatal(err)
	}
	if n := compileCount.Load(); n != 2 {
		t.Fatalf("expected 2 compiles, got %d", n)
	}
	if len(lastChanged) == 0 {
		t.Fatal("expected changed files on recompile")
	}
	found := false
	for _, f := range lastChanged {
		if f == "comps.gadx" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected comps.gadx in changed files, got %v", lastChanged)
	}
}

func TestRenderCallbackReceivesRelativePaths(t *testing.T) {
	dir := t.TempDir()

	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}

	src := `@main
    p Hello`
	srcPath := filepath.Join(dir, "sub", "nested.gadx")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	var (
		mainFile string
		changed  []string
	)
	r := newTestRender(t, dir)
	r.OnRender(func(first bool, mf string, f []string, lastTime time.Time, err error) {
		mainFile = mf
		changed = f
	})

	if _, err := renderString(r, srcPath, gad.Dict{}); err != nil {
		t.Fatal(err)
	}
	if mainFile != "sub/nested.gadx" {
		t.Fatalf("expected relative path `sub/nested.gadx`, got %q", mainFile)
	}
	if len(changed) != 0 {
		t.Fatalf("expected no changed files on first render, got %v", changed)
	}
}

func TestRenderCallbackCompileError(t *testing.T) {
	dir := t.TempDir()

	// Gadx source that fails at the Gad compilation stage.
	srcPath := filepath.Join(dir, "bad.gadx")
	if err := os.WriteFile(srcPath, []byte("@main\n    p {= undefinedVar }"), 0644); err != nil {
		t.Fatal(err)
	}

	var callbackErr error
	r := newTestRender(t, dir)
	r.OnRender(func(first bool, mainFile string, files []string, lastTime time.Time, err error) {
		callbackErr = err
	})

	_, err := renderString(r, srcPath, gad.Dict{})
	if err == nil {
		t.Fatal("expected compile error for undefined variable")
	}
	if callbackErr == nil {
		t.Fatal("expected callback to receive error")
	}
}

func TestRenderOldCachePreservedOnCompileError(t *testing.T) {
	dir := t.TempDir()
	baseTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	srcPath := filepath.Join(dir, "err_cache.gadx")
	writeFileWithMtime(t, srcPath, `@main
    p Hello`, baseTime)

	r := newTestRender(t, dir)

	// First successful render.
	out, err := renderString(r, srcPath, gad.Dict{})
	if err != nil {
		t.Fatal(err)
	}
	if out != "<p>Hello</p>" {
		t.Fatalf("expected `<p>Hello</p>`, got %q", out)
	}

	// Write invalid source (undefined var), with a save's own mtime so the
	// debounce below is what governs when it is compiled.
	writeFileWithMtime(t, srcPath, `@main
    p {= undefinedVar}`, time.Now())

	// First post-change render — stamps changedAt, serves stale cache.
	if _, err := renderString(r, srcPath, gad.Dict{}); err != nil {
		t.Fatal(err) // should succeed with stale cache
	}

	// Wait for debounce, then render again — should trigger compile error.
	r.TemplateDelay = 10 * time.Millisecond
	time.Sleep(15 * time.Millisecond)
	_, err = renderString(r, srcPath, gad.Dict{})
	if err == nil {
		t.Fatal("expected compile error")
	}

	// After failed compile, old cache was NOT replaced.
	// Fix the file back to original content.
	writeFileWithMtime(t, srcPath, `@main
    p Hello`, baseTime.Add(2*time.Hour))
	time.Sleep(15 * time.Millisecond)

	out, err = renderString(r, srcPath, gad.Dict{})
	if err != nil {
		t.Fatal(err)
	}
	if out != "<p>Hello</p>" {
		t.Fatalf("expected `<p>Hello</p>`, got %q", out)
	}
}

func TestRenderMultipleCallbacks(t *testing.T) {
	dir := t.TempDir()
	src := `@main
    p Hello`
	srcPath := filepath.Join(dir, "multi.gadx")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	var c1, c2 atomic.Int32
	r := newTestRender(t, dir)
	r.OnRender(func(first bool, mainFile string, files []string, lastTime time.Time, err error) {
		c1.Add(1)
	})
	r.OnRender(func(first bool, mainFile string, files []string, lastTime time.Time, err error) {
		c2.Add(1)
	})

	if _, err := renderString(r, srcPath, gad.Dict{}); err != nil {
		t.Fatal(err)
	}
	if c1.Load() != 1 || c2.Load() != 1 {
		t.Fatalf("expected both callbacks to fire, got c1=%d c2=%d", c1.Load(), c2.Load())
	}
}

func TestRenderCallbackLastTime(t *testing.T) {
	dir := t.TempDir()
	baseTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	srcPath := filepath.Join(dir, "lasttime.gadx")
	writeFileWithMtime(t, srcPath, `@main
    p Hello`, baseTime)

	var firstTime, secondTime time.Time
	r := newTestRender(t, dir)
	r.OnRender(func(first bool, mainFile string, files []string, lastTime time.Time, err error) {
		if first {
			firstTime = lastTime
		} else {
			secondTime = lastTime
		}
	})

	// First render.
	if _, err := renderString(r, srcPath, gad.Dict{}); err != nil {
		t.Fatal(err)
	}
	if !firstTime.IsZero() {
		t.Fatal("expected zero lastTime on first render")
	}

	// Modify file.
	writeFileWithMtime(t, srcPath, `@main
    p Changed`, baseTime.Add(time.Hour))

	// First post-change render — stamps changedAt.
	if _, err := renderString(r, srcPath, gad.Dict{}); err != nil {
		t.Fatal(err)
	}

	// Wait for debounce, then render again to trigger recompile.
	r.TemplateDelay = 10 * time.Millisecond
	time.Sleep(15 * time.Millisecond)
	if _, err := renderString(r, srcPath, gad.Dict{}); err != nil {
		t.Fatal(err)
	}
	if secondTime.IsZero() {
		t.Fatal("expected non-zero lastTime on recompile")
	}
}

func TestRenderEmptyWorkDirFallsBackToFileDir(t *testing.T) {
	dir := t.TempDir()

	// Write an imported file in a subdir.
	sub := filepath.Join(dir, "inner")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	compPath := filepath.Join(sub, "comps.gadx")
	if err := os.WriteFile(compPath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	tplSrc := `@import "comps.gadx"
@main
    p Hello`
	tplPath := filepath.Join(sub, "main.gadx")
	if err := os.WriteFile(tplPath, []byte(tplSrc), 0644); err != nil {
		t.Fatal(err)
	}

	r := NewRender("") // empty workDir
	r.TemplateDelay = 10 * time.Millisecond
	out, err := renderString(r, tplPath, gad.Dict{})
	if err != nil {
		t.Fatal(err)
	}
	if out != "<p>Hello</p>" {
		t.Fatalf("expected `<p>Hello</p>`, got %q", out)
	}
}

func TestRenderConcurrent(t *testing.T) {
	dir := t.TempDir()
	src := `@main
    p Hello`
	srcPath := filepath.Join(dir, "conc.gadx")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	r := newTestRender(t, dir)
	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			_, err := renderString(r, srcPath, gad.Dict{})
			if err != nil {
				t.Errorf("concurrent render: %v", err)
			}
			done <- struct{}{}
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestRenderWithTranspile(t *testing.T) {
	dir := t.TempDir()
	src := `@main
    p Hello`
	srcPath := filepath.Join(dir, "transpile.gadx")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	r := newTestRender(t, dir)
	transpilePath := filepath.Join(dir, ".transpiled", "transpile.gad")
	r.TranspilePath = func(src string) string {
		return transpilePath
	}

	if _, err := renderString(r, srcPath, gad.Dict{}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(transpilePath); os.IsNotExist(err) {
		t.Fatal("expected transpiled file to exist")
	}
}

func TestRenderWriterOutput(t *testing.T) {
	dir := t.TempDir()
	src := `@main
    p Output test`
	srcPath := filepath.Join(dir, "writer.gadx")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	r := newTestRender(t, dir)

	var buf bytes.Buffer
	if err := r.Render(&buf, srcPath, gad.Dict{}); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "<p>Output test</p>" {
		t.Fatalf("expected `<p>Output test</p>`, got %q", buf.String())
	}
}

func TestRenderWriteToNilWriter(t *testing.T) {
	// Verify that io.Discard works correctly.
	dir := t.TempDir()
	src := `@main
    p Discard me`
	srcPath := filepath.Join(dir, "discard.gadx")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	r := newTestRender(t, dir)
	if err := r.Render(io.Discard, srcPath, gad.Dict{}); err != nil {
		t.Fatal(err)
	}
}

func TestRenderFileNotFound(t *testing.T) {
	r := newTestRender(t, t.TempDir())
	err := r.Render(io.Discard, "/nonexistent/file.gadx", gad.Dict{})
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestRenderCacheIsPerFile(t *testing.T) {
	dir := t.TempDir()

	src1 := `@main
    p File1`
	src2 := `@main
    p File2`

	p1 := filepath.Join(dir, "f1.gadx")
	p2 := filepath.Join(dir, "f2.gadx")
	if err := os.WriteFile(p1, []byte(src1), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p2, []byte(src2), 0644); err != nil {
		t.Fatal(err)
	}

	r := newTestRender(t, dir)

	out1, err := renderString(r, p1, gad.Dict{})
	if err != nil {
		t.Fatal(err)
	}
	out2, err := renderString(r, p2, gad.Dict{})
	if err != nil {
		t.Fatal(err)
	}
	if out1 != "<p>File1</p>" {
		t.Fatalf("expected `<p>File1</p>`, got %q", out1)
	}
	if out2 != "<p>File2</p>" {
		t.Fatalf("expected `<p>File2</p>`, got %q", out2)
	}
}

func TestRenderOnRenderReturnsRender(t *testing.T) {
	r := newTestRender(t, t.TempDir())
	chained := r.OnRender(func(first bool, mainFile string, files []string, lastTime time.Time, err error) {})
	if chained != r {
		t.Fatal("OnRender should return the Render for chaining")
	}
}

// TestRenderInterfaceCacheReused checks that a compiled template's
// interface-satisfaction cache persists across renders and is replaced (reset)
// when the template recompiles after a source change.
func TestRenderInterfaceCacheReused(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "c.gadx")
	if err := os.WriteFile(srcPath, []byte("@main\n    p A"), 0644); err != nil {
		t.Fatal(err)
	}
	r := newTestRender(t, dir)

	if _, err := renderString(r, srcPath, gad.Dict{}); err != nil {
		t.Fatal(err)
	}
	entry1 := r.templateCache[srcPath]
	if entry1 == nil || entry1.ifaceCache == nil {
		t.Fatal("compiled entry should carry an interface cache")
	}

	// A second render reuses the same entry (and its cache).
	if _, err := renderString(r, srcPath, gad.Dict{}); err != nil {
		t.Fatal(err)
	}
	if r.templateCache[srcPath] != entry1 {
		t.Fatal("unchanged template should reuse the same cache entry")
	}

	// Change the source (sleep first so the mtime is strictly newer); the first
	// render after the change detects it and arms the debounce, a render after
	// the delay recompiles → a fresh cache.
	time.Sleep(20 * time.Millisecond)
	if err := os.WriteFile(srcPath, []byte("@main\n    p B"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := renderString(r, srcPath, gad.Dict{}); err != nil { // detect
		t.Fatal(err)
	}
	time.Sleep(r.TemplateDelay + 20*time.Millisecond)
	out, err := renderString(r, srcPath, gad.Dict{}) // recompile
	if err != nil {
		t.Fatal(err)
	}
	if out != "<p>B</p>" {
		t.Fatalf("recompile should render the new source, got %q", out)
	}
	entry2 := r.templateCache[srcPath]
	if entry2 == entry1 {
		t.Fatal("recompile should produce a fresh entry")
	}
	if entry2.ifaceCache == entry1.ifaceCache {
		t.Fatal("recompile should reset the interface cache")
	}
}
