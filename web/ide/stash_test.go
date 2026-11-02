package ide

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitStashes(t *testing.T) {
	repo, h := gitRepo(t)
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		cmd.Env = append(gitEnv(), "GIT_AUTHOR_NAME=Ann", "GIT_AUTHOR_EMAIL=ann@example.com", "GIT_COMMITTER_NAME=Ann", "GIT_COMMITTER_EMAIL=ann@example.com")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	post := func(op string, body any) *httptest.ResponseRecorder {
		t.Helper()
		b, _ := json.Marshal(body)
		r := httptest.NewRequest("POST", "/api/ide/"+op, bytes.NewReader(b))
		r.Header.Set("X-Git-Author", "Bob <bob@example.com>")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	list := func() []GitStash {
		t.Helper()
		return decode[struct{ Stashes []GitStash }](t, do(t, h, "GET", "/api/ide/git/stashes", nil)).Stashes
	}

	if st := list(); len(st) != 0 {
		t.Fatalf("stashes at first: %v", st)
	}
	if w := post("git/stash/new", map[string]any{"message": "nada"}); w.Code != http.StatusBadRequest {
		t.Fatalf("nothing to stash: %d %s", w.Code, w.Body)
	}
	// a change and a new file: stashed with the untracked
	os.WriteFile(filepath.Join(repo, "src/a.gad"), []byte("x := 99\n"), 0o644)
	os.WriteFile(filepath.Join(repo, "src/novo.gad"), []byte("novo\n"), 0o644)
	if w := post("git/stash/new", map[string]any{"message": "primeira", "untracked": true}); w.Code != 200 {
		t.Fatalf("stash: %d %s", w.Code, w.Body)
	}
	if s := git("status", "--porcelain"); s != "" {
		t.Fatalf("after the stash: %q", s)
	}
	st := list()
	if len(st) != 1 || st[0].Ref != "stash@{0}" || st[0].Message != "primeira" || st[0].Branch != "main" {
		t.Fatalf("stashes: %+v", st)
	}
	if a := git("log", "-1", "--format=%an", "stash@{0}"); a != "Bob" {
		t.Fatalf("stash author %q", a)
	}
	d := decode[GitCommitDetail](t, do(t, h, "GET", "/api/ide/git/stash?hash="+st[0].Hash, nil))
	got := map[string]string{}
	for _, f := range d.Files {
		got[f.Path] = f.Status
	}
	if got["a.gad"] != "M" || got["novo.gad"] != "?" {
		t.Fatalf("stash files: %+v", d.Files)
	}
	if f := decode[GitFile](t, do(t, h, "GET", "/api/ide/git/stash/diff?hash="+st[0].Hash+"&path=a.gad", nil)); !strings.HasPrefix(f.Old, "x := 10") || f.New != "x := 99\n" {
		t.Fatalf("diff: %+v", f)
	}
	if f := decode[GitFile](t, do(t, h, "GET", "/api/ide/git/stash/diff?hash="+st[0].Hash+"&path=novo.gad&untracked=1", nil)); f.New != "novo\n" || f.Old != "" {
		t.Fatalf("untracked diff: %+v", f)
	}

	// a second; apply the older (kept), pop it (gone)
	os.WriteFile(filepath.Join(repo, "src/old.gad"), []byte("mudou\n"), 0o644)
	if w := post("git/stash/new", map[string]any{}); w.Code != 200 {
		t.Fatalf("stash 2: %d %s", w.Code, w.Body)
	}
	if st = list(); len(st) != 2 || st[1].Message != "primeira" {
		t.Fatalf("two: %+v", st)
	}
	if w := post("git/stash/apply", map[string]any{"ref": "stash@{1}"}); w.Code != 200 {
		t.Fatalf("apply: %d %s", w.Code, w.Body)
	}
	if b, _ := os.ReadFile(filepath.Join(repo, "src/novo.gad")); string(b) != "novo\n" || len(list()) != 2 {
		t.Fatalf("applied: %q", b)
	}
	git("checkout", "--", ".")
	os.Remove(filepath.Join(repo, "src/novo.gad"))
	if w := post("git/stash/apply", map[string]any{"ref": "stash@{1}", "pop": true}); w.Code != 200 || len(list()) != 1 {
		t.Fatalf("pop: %d %s", w.Code, w.Body)
	}
	git("checkout", "--", ".")
	os.Remove(filepath.Join(repo, "src/novo.gad"))

	// a branch of the one left: there, dropped
	if w := post("git/stash/branch", map[string]any{"ref": "stash@{0}", "branch": "-x"}); w.Code != http.StatusBadRequest {
		t.Fatalf("bad branch: %d", w.Code)
	}
	if w := post("git/stash/branch", map[string]any{"ref": "stash@{0}", "branch": "do-stash"}); w.Code != 200 || len(list()) != 0 {
		t.Fatalf("branch: %d %s", w.Code, w.Body)
	}
	if b := git("rev-parse", "--abbrev-ref", "HEAD"); b != "do-stash" {
		t.Fatalf("on %q", b)
	}

	// drop, clear; refs checked
	git("checkout", "--", ".")
	git("checkout", "-q", "main")
	for i := 0; i < 2; i++ {
		os.WriteFile(filepath.Join(repo, "src/a.gad"), []byte(strings.Repeat("y", i+1)), 0o644)
		post("git/stash/new", map[string]any{})
	}
	if w := post("git/stash/drop", map[string]any{"ref": "stash@{0}; rm -rf /"}); w.Code != http.StatusBadRequest {
		t.Fatalf("bad ref: %d", w.Code)
	}
	if w := post("git/stash/drop", map[string]any{"ref": "stash@{0}"}); w.Code != 200 || len(list()) != 1 {
		t.Fatalf("drop: %d %s", w.Code, w.Body)
	}
	if w := post("git/stash/clear", nil); w.Code != 200 || len(list()) != 0 {
		t.Fatalf("clear: %d %s", w.Code, w.Body)
	}
	if w := do(t, h, "GET", "/api/ide/git/stash/clear", nil); w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET clear: %d", w.Code)
	}
}
