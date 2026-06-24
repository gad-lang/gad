package ide

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestServer(t *testing.T) (*Server, http.Handler, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.gad"), []byte("println(\"hi\")\nreturn 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	s, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	return s, s.Handler(), dir
}

func do(t *testing.T, h http.Handler, method, url string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body != nil {
		b, _ := json.Marshal(body)
		r = httptest.NewRequest(method, url, bytes.NewReader(b))
	} else {
		r = httptest.NewRequest(method, url, nil)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func decode[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode %T: %v (%s)", v, err, w.Body.String())
	}
	return v
}

func TestNewWithFileOpensIt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.gad")
	os.WriteFile(path, []byte("return 1\n"), 0o644)
	s, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	if s.Root != dir {
		t.Errorf("root = %q, want %q", s.Root, dir)
	}
	if s.OpenFile != "x.gad" {
		t.Errorf("openFile = %q, want x.gad", s.OpenFile)
	}
}

func TestTree(t *testing.T) {
	_, h, dir := newTestServer(t)
	os.WriteFile(filepath.Join(dir, ".hidden"), []byte("x"), 0o644)
	os.MkdirAll(filepath.Join(dir, "node_modules"), 0o755)
	w := do(t, h, "GET", "/api/ide/tree", nil)
	root := decode[treeNode](t, w)
	names := map[string]bool{}
	for _, c := range root.Children {
		names[c.Name] = true
	}
	if !names["main.gad"] || !names["sub"] {
		t.Fatalf("tree missing entries: %+v", names)
	}
	if names[".hidden"] || names["node_modules"] {
		t.Fatalf("tree should skip hidden/ignored: %+v", names)
	}

	// ?hidden=true reveals dot entries but still hides always-ignored dirs.
	w = do(t, h, "GET", "/api/ide/tree?hidden=true", nil)
	root = decode[treeNode](t, w)
	names = map[string]bool{}
	for _, c := range root.Children {
		names[c.Name] = true
	}
	if !names[".hidden"] {
		t.Fatalf("tree?hidden=true should include .hidden: %+v", names)
	}
	if names["node_modules"] {
		t.Fatalf("tree?hidden=true must still skip node_modules: %+v", names)
	}
}

func TestRunSaveOutputs(t *testing.T) {
	_, h, dir := newTestServer(t)
	// stdout via print, stderr via an uncaught error.
	src := `print("hello"); throw "boom"`

	// Separate stdout/stderr files.
	do(t, h, "POST", "/api/ide/run", runRequest{
		Source: src, SaveStdout: "o.txt", SaveStderr: "e.txt",
	})
	if b, _ := os.ReadFile(filepath.Join(dir, "o.txt")); string(b) != "hello" {
		t.Fatalf("stdout file = %q, want hello", b)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "e.txt")); !strings.Contains(string(b), "boom") {
		t.Fatalf("stderr file = %q, want boom", b)
	}

	// Combined into the stdout file only.
	do(t, h, "POST", "/api/ide/run", runRequest{
		Source: src, SaveStdout: "c.txt", Combine: true,
	})
	c, _ := os.ReadFile(filepath.Join(dir, "c.txt"))
	if !strings.HasPrefix(string(c), "hello") || !strings.Contains(string(c), "boom") {
		t.Fatalf("combined file = %q, want hello+boom", c)
	}

	// Legacy saveOut still works (combined).
	do(t, h, "POST", "/api/ide/run", runRequest{Source: src, SaveOut: "legacy.txt"})
	if b, _ := os.ReadFile(filepath.Join(dir, "legacy.txt")); !strings.Contains(string(b), "boom") {
		t.Fatalf("legacy saveOut = %q", b)
	}
}

func TestTranspile(t *testing.T) {
	_, h, dir := newTestServer(t)

	// Mixed inferred from a .gadt path; default write/rawstr funcs.
	w := do(t, h, "POST", "/api/ide/transpile", transpileRequest{
		Path: "p.gadt", Source: "Hi {%= name %}!\n",
	})
	res := decode[map[string]any](t, w)
	if res["ok"] != true {
		t.Fatalf("transpile not ok: %v", res)
	}
	src, _ := res["source"].(string)
	if !strings.Contains(src, `write(raw "`) || !strings.Contains(src, "write(name)") {
		t.Fatalf("transpiled source unexpected:\n%s", src)
	}

	// transpile config in .gad.yaml overrides the write function name.
	os.WriteFile(filepath.Join(dir, configFile),
		[]byte("transpile:\n  writeFunc: emit\n"), 0o644)
	w = do(t, h, "POST", "/api/ide/transpile", transpileRequest{
		Mixed: true, Source: "{%= x %}",
	})
	src, _ = decode[map[string]any](t, w)["source"].(string)
	if !strings.Contains(src, "emit(x)") {
		t.Fatalf("config writeFunc not applied:\n%s", src)
	}

	// A per-request override beats the config.
	w = do(t, h, "POST", "/api/ide/transpile", transpileRequest{
		Mixed: true, Source: "{%= x %}", WriteFunc: "put",
	})
	src, _ = decode[map[string]any](t, w)["source"].(string)
	if !strings.Contains(src, "put(x)") {
		t.Fatalf("request override not applied:\n%s", src)
	}
}

func TestEval(t *testing.T) {
	_, h, _ := newTestServer(t)

	// Plain expression, str() rendering.
	w := do(t, h, "POST", "/api/ide/eval", evalRequest{Expr: "2 + 3 * 4"})
	res := decode[evalResult](t, w)
	if !res.OK || res.Value != "14" {
		t.Fatalf("eval 2+3*4 = %+v", res)
	}

	// repr() rendering uses Gad's repr form (type-tagged), unlike str().
	w = do(t, h, "POST", "/api/ide/eval", evalRequest{Expr: `"hi"`, Repr: true})
	res = decode[evalResult](t, w)
	if !res.OK || !strings.Contains(res.Value, `"hi"`) || res.Value == "hi" {
		t.Fatalf("repr eval = %+v (want a repr form containing \"hi\")", res)
	}

	// Source prelude provides context.
	w = do(t, h, "POST", "/api/ide/eval", evalRequest{Source: "x := 21", Expr: "x * 2"})
	res = decode[evalResult](t, w)
	if !res.OK || res.Value != "42" {
		t.Fatalf("eval with prelude = %+v", res)
	}

	// A prelude that ends with a top-level `return` must not mask the expression
	// result (regression: the file's return short-circuited the eval).
	w = do(t, h, "POST", "/api/ide/eval", evalRequest{
		Source: "i := 42\nreturn [i]\n", Expr: "1 + 1",
	})
	res = decode[evalResult](t, w)
	if !res.OK || res.Value != "2" {
		t.Fatalf("eval past prelude return = %+v (want 2)", res)
	}
	// Definitions from such a prelude stay in scope.
	w = do(t, h, "POST", "/api/ide/eval", evalRequest{
		Source: "i := 42\nreturn [i]\n", Expr: "i",
	})
	res = decode[evalResult](t, w)
	if !res.OK || res.Value != "42" {
		t.Fatalf("eval prelude def in scope = %+v (want 42)", res)
	}

	// An error expression reports !ok with a message.
	w = do(t, h, "POST", "/api/ide/eval", evalRequest{Expr: "1 +"})
	res = decode[evalResult](t, w)
	if res.OK || res.Error == "" {
		t.Fatalf("eval of bad expr should fail: %+v", res)
	}

	// Empty expression is rejected.
	w = do(t, h, "POST", "/api/ide/eval", evalRequest{Expr: "  "})
	res = decode[evalResult](t, w)
	if res.OK {
		t.Fatalf("empty expr should not be ok: %+v", res)
	}
}

func TestFetch(t *testing.T) {
	_, h, dir := newTestServer(t)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("downloaded body"))
	}))
	defer upstream.Close()

	w := do(t, h, "POST", "/api/ide/fetch", fetchRequest{URL: upstream.URL, Path: "sub/dl.txt"})
	if w.Code != 200 {
		t.Fatalf("fetch status %d: %s", w.Code, w.Body)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "sub", "dl.txt")); string(b) != "downloaded body" {
		t.Fatalf("downloaded content = %q", b)
	}

	// Non-http schemes are rejected.
	w = do(t, h, "POST", "/api/ide/fetch", fetchRequest{URL: "file:///etc/passwd", Path: "x.txt"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("file:// scheme should be rejected, got %d", w.Code)
	}
}

func TestFileReadWrite(t *testing.T) {
	_, h, dir := newTestServer(t)
	// write
	w := do(t, h, "PUT", "/api/ide/file", fileRequest{Path: "sub/new.gad", Content: "return 42\n"})
	if w.Code != 200 {
		t.Fatalf("write status %d: %s", w.Code, w.Body)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "sub", "new.gad")); string(b) != "return 42\n" {
		t.Fatalf("file content = %q", b)
	}
	// read
	w = do(t, h, "GET", "/api/ide/file?path=sub/new.gad", nil)
	got := decode[map[string]string](t, w)
	if got["content"] != "return 42\n" {
		t.Fatalf("read content = %q", got["content"])
	}
}

func TestPathTraversalRejected(t *testing.T) {
	_, h, _ := newTestServer(t)
	w := do(t, h, "GET", "/api/ide/file?path=../../etc/passwd", nil)
	if w.Code == 200 {
		t.Fatalf("traversal should be rejected, got 200: %s", w.Body)
	}
}

func TestDeleteAndRename(t *testing.T) {
	_, h, dir := newTestServer(t)
	do(t, h, "PUT", "/api/ide/file", fileRequest{Path: "a.gad", Content: "1"})
	w := do(t, h, "POST", "/api/ide/rename", fileRequest{Path: "a.gad", To: "b.gad"})
	if w.Code != 200 {
		t.Fatalf("rename status %d", w.Code)
	}
	if _, err := os.Stat(filepath.Join(dir, "b.gad")); err != nil {
		t.Fatalf("renamed file missing: %v", err)
	}
	w = do(t, h, "POST", "/api/ide/delete", fileRequest{Path: "b.gad"})
	if w.Code != 200 {
		t.Fatalf("delete status %d", w.Code)
	}
	if _, err := os.Stat(filepath.Join(dir, "b.gad")); !os.IsNotExist(err) {
		t.Fatalf("file not deleted")
	}
}

func TestConfigRoundTrip(t *testing.T) {
	_, h, dir := newTestServer(t)
	doc := map[string]any{
		"fmt": map[string]any{"backup": true},
		"ide": map[string]any{"theme": "dark", "sidebarWidth": 200},
	}
	w := do(t, h, "PUT", "/api/ide/config", doc)
	if w.Code != 200 {
		t.Fatalf("config put status %d: %s", w.Code, w.Body)
	}
	if _, err := os.Stat(filepath.Join(dir, configFile)); err != nil {
		t.Fatalf(".gad.yaml not written: %v", err)
	}
	w = do(t, h, "GET", "/api/ide/config", nil)
	got := decode[map[string]any](t, w)
	ide, _ := got["ide"].(map[string]any)
	if ide == nil || ide["theme"] != "dark" {
		t.Fatalf("config ide not preserved: %+v", got)
	}
}

func TestFormatAndDiagnose(t *testing.T) {
	_, h, _ := newTestServer(t)
	w := do(t, h, "POST", "/api/ide/format", formatRequest{Source: "a:=1\n"})
	fr := decode[map[string]any](t, w)
	if fr["ok"] != true {
		t.Fatalf("format not ok: %+v", fr)
	}
	w = do(t, h, "POST", "/api/ide/diagnose", formatRequest{Source: "return missing(\n"})
	dr := decode[map[string]any](t, w)
	if d, _ := dr["diagnostics"].([]any); len(d) == 0 {
		t.Fatalf("expected diagnostics for bad source: %+v", dr)
	}
}

func TestRunWithArgs(t *testing.T) {
	_, h, _ := newTestServer(t)
	src := "param (*args)\nvar p\nif len(args) > 0 { p = args[0] } else { p = \"none\" }\nprintln(p)\nreturn p\n"
	w := do(t, h, "POST", "/api/ide/run", runRequest{Source: src, Args: []string{"hello"}})
	res := decode[map[string]any](t, w)
	if res["ok"] != true {
		t.Fatalf("run failed: %+v", res)
	}
	if !strings.Contains(res["stdout"].(string), "hello") {
		t.Fatalf("expected stdout to contain arg, got %q", res["stdout"])
	}
}

func TestRunSavesOutput(t *testing.T) {
	_, h, dir := newTestServer(t)
	w := do(t, h, "POST", "/api/ide/run", runRequest{
		Source: "println(\"saved output\")\n", SaveOut: "out.log",
	})
	if decode[map[string]any](t, w)["ok"] != true {
		t.Fatalf("run failed: %s", w.Body)
	}
	b, err := os.ReadFile(filepath.Join(dir, "out.log"))
	if err != nil || !strings.Contains(string(b), "saved output") {
		t.Fatalf("output file = %q err=%v", b, err)
	}
}

func TestRunDisabledModule(t *testing.T) {
	_, h, _ := newTestServer(t)
	// Disabling the time module should make importing it fail.
	src := "import(\"time\")\nreturn 1\n"
	w := do(t, h, "POST", "/api/ide/run", runRequest{Source: src, Disabled: []string{"time"}})
	res := decode[map[string]any](t, w)
	if res["ok"] == true {
		t.Fatalf("expected failure importing a disabled module, got ok: %+v", res)
	}
}

func TestModulesList(t *testing.T) {
	_, h, _ := newTestServer(t)
	w := do(t, h, "GET", "/api/ide/modules", nil)
	mods := decode[[]moduleInfo](t, w)
	if len(mods) == 0 {
		t.Fatal("expected modules")
	}
	var sawTime, sawUnsafe bool
	for _, m := range mods {
		if m.Name == "time" {
			sawTime = true
		}
		if m.Unsafe {
			sawUnsafe = true
		}
	}
	if !sawTime || !sawUnsafe {
		t.Fatalf("modules incomplete: %+v", mods)
	}
}

func TestDebugFramesCarryLocals(t *testing.T) {
	_, h, _ := newTestServer(t)
	// Breakpoint inside the function: there should be two frames, the innermost
	// carrying its own named locals (a, b, s).
	src := "add := func(a, b) {\n  s := a + b\n  return s\n}\nr := add(10, 20)\nreturn r\n"
	w := do(t, h, "POST", "/api/ide/debug/start", StartRequest{Source: src, Breakpoints: []int{3}})
	resp := decode[DebugResponse](t, w)
	if resp.State != "stopped" {
		t.Fatalf("expected stopped, got %+v", resp)
	}
	if len(resp.Frames) < 2 {
		t.Fatalf("expected at least 2 frames, got %d: %+v", len(resp.Frames), resp.Frames)
	}
	inner := resp.Frames[0]
	if inner.File == "" || inner.Line == 0 {
		t.Errorf("inner frame missing file/line: %+v", inner)
	}
	names := map[string]string{}
	for _, v := range inner.Locals {
		names[v.Name] = v.Value
	}
	if names["a"] != "10" || names["b"] != "20" || names["s"] != "30" {
		t.Fatalf("inner frame locals wrong: %+v", inner.Locals)
	}
}

func TestDebugConditionalBreakpoint(t *testing.T) {
	_, h, _ := newTestServer(t)
	src := "a := 1\nb := 5\nc := a + b\nreturn c\n" // lines 1..4

	// Condition true at line 3 -> stops.
	w := do(t, h, "POST", "/api/ide/debug/start", StartRequest{
		Source:          src,
		BreakpointSpecs: []BreakpointSpec{{Line: 3, Condition: "b == 5"}},
	})
	resp := decode[DebugResponse](t, w)
	if resp.State != "stopped" || resp.Line != 3 {
		t.Fatalf("expected stop at line 3, got %+v", resp)
	}

	// Disabled breakpoint -> runs to completion.
	w = do(t, h, "POST", "/api/ide/debug/start", StartRequest{
		Source:          src,
		BreakpointSpecs: []BreakpointSpec{{Line: 3, Disabled: true}},
	})
	resp = decode[DebugResponse](t, w)
	if resp.State != "terminated" {
		t.Fatalf("disabled bp should run to completion, got %+v", resp)
	}
}

func TestDebugResolvesBuiltinModule(t *testing.T) {
	_, h, _ := newTestServer(t)
	// A debug session must resolve stdlib imports (regression: "module time not
	// found"). Stop-on-entry then run to completion.
	src := "time := import(\"time\")\nreturn time.Hour\n"
	w := do(t, h, "POST", "/api/ide/debug/start", StartRequest{Source: src})
	resp := decode[DebugResponse](t, w)
	if resp.State == "error" {
		t.Fatalf("debug start errored (import failed?): %+v", resp)
	}
	// It runs straight through (no breakpoints) to termination.
	if resp.State == "stopped" {
		resp = decode[DebugResponse](t, do(t, h, "POST", "/api/ide/debug/command",
			CommandRequest{Session: resp.Session, Command: "continue"}))
	}
	if resp.State != "terminated" || resp.Error != "" {
		t.Fatalf("expected clean termination, got %+v", resp)
	}
}

func TestDebugRelativeImportSamples(t *testing.T) {
	// Debug the real samples/modules/main.gad, which imports ./mathx.gad and the
	// parameterised ./greet.gad — verifying relative imports resolve relative to
	// the debugged file's directory.
	root := filepath.Join("..", "..", "samples", "modules")
	src, err := os.ReadFile(filepath.Join(root, "main.gad"))
	if err != nil {
		t.Skipf("samples not present: %v", err)
	}
	s, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	h := s.Handler()

	w := do(t, h, "POST", "/api/ide/debug/start", StartRequest{
		Path: "main.gad", Source: string(src), Breakpoints: []int{4},
	})
	resp := decode[DebugResponse](t, w)
	if resp.State == "error" {
		t.Fatalf("debug start errored (relative import failed?): %+v", resp)
	}
	// Drive to completion; relative imports must not raise along the way.
	for i := 0; i < 10 && resp.State == "stopped"; i++ {
		resp = decode[DebugResponse](t, do(t, h, "POST", "/api/ide/debug/command",
			CommandRequest{Session: resp.Session, Command: "continue"}))
	}
	if resp.State != "terminated" {
		t.Fatalf("expected terminated, got %+v", resp)
	}
	if resp.Error != "" {
		t.Fatalf("debug run errored: %s", resp.Error)
	}
	if resp.Result != "49" { // mathx.square(7)
		t.Fatalf("expected result 49, got %q (output=%q)", resp.Result, resp.Output)
	}
}

func TestDebugSessionOverHTTP(t *testing.T) {
	_, h, _ := newTestServer(t)
	w := do(t, h, "POST", "/api/ide/debug/start", StartRequest{
		Source:      "a := 1\nb := 2\nprintln(a + b)\nreturn a + b\n",
		Breakpoints: []int{3},
	})
	start := decode[DebugResponse](t, w)
	if start.State != "stopped" || start.Line != 3 {
		t.Fatalf("expected stop at line 3, got %+v", start)
	}
	w = do(t, h, "POST", "/api/ide/debug/command", CommandRequest{Session: start.Session, Command: "continue"})
	end := decode[DebugResponse](t, w)
	if end.State != "terminated" || end.Result != "3" {
		t.Fatalf("expected terminated result 3, got %+v", end)
	}
}
