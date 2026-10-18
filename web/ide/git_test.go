package ide

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// gitRepo is a repository whose workspace is its folder src: two commits on
// main — the second changes a.gad and a file out of src —, a branch feature.
func gitRepo(t *testing.T) (repo string, h http.Handler) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	repo = t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		cmd.Env = append(gitEnv(), "GIT_AUTHOR_NAME=Ann", "GIT_AUTHOR_EMAIL=ann@example.com",
			"GIT_COMMITTER_NAME=Ann", "GIT_COMMITTER_EMAIL=ann@example.com")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return string(out)
	}
	write := func(p, s string) {
		t.Helper()
		abs := filepath.Join(repo, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run("init", "-q", "-b", "main")
	write("src/a.gad", "x := 1\ny := 2\nz := 3\nreturn x\n")
	write("src/old.gad", "one\ntwo\nthree\nfour\n")
	write("README", "out of the workspace\n")
	run("add", ".")
	run("commit", "-q", "-m", "first")
	run("branch", "feature")
	write("src/a.gad", "x := 10\ny := 2\nz := 3\nreturn x\n")
	write("README", "changed\n")
	run("add", ".")
	run("commit", "-q", "-m", "second\n\nThe body of the message.")
	s, err := New(filepath.Join(repo, "src"))
	if err != nil {
		t.Fatal(err)
	}
	return repo, s.Handler()
}

func TestGitWorkspace(t *testing.T) {
	_, h := gitRepo(t)
	ws := decode[map[string]any](t, do(t, h, "GET", "/api/ide/workspace", nil))
	if ws["git"] != true {
		t.Fatalf("workspace git = %v", ws["git"])
	}
	_, plain, _ := newTestServer(t)
	if ws := decode[map[string]any](t, do(t, plain, "GET", "/api/ide/workspace", nil)); ws["git"] == true {
		t.Fatal("a workspace in no repository says git")
	}
}

func TestGitChangesRenameAndSave(t *testing.T) {
	repo, h := gitRepo(t)
	// the IDE renames on the disk: one deleted, one new, alike
	if err := os.Rename(filepath.Join(repo, "src/old.gad"), filepath.Join(repo, "src/new.gad")); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(repo, "src/a.gad"), []byte("x := 11\ny := 2\nz := 3\nreturn x\n"), 0o644)
	os.WriteFile(filepath.Join(repo, "README"), []byte("out\n"), 0o644)

	files := decode[struct{ Files []GitFile }](t, do(t, h, "GET", "/api/ide/git/changes", nil)).Files
	got := map[string]GitFile{}
	for _, f := range files {
		got[f.Path] = f
	}
	if len(files) != 2 || got["a.gad"].Status != "M" || got["new.gad"].Status != "R" || got["new.gad"].From != "old.gad" {
		t.Fatalf("changes = %+v", files)
	}

	d := decode[GitFile](t, do(t, h, "GET", "/api/ide/git/diff?path=a.gad", nil))
	if !strings.HasPrefix(d.Old, "x := 10\n") || !strings.HasPrefix(d.New, "x := 11\n") {
		t.Fatalf("diff = %+v", d)
	}
	r := decode[GitFile](t, do(t, h, "GET", "/api/ide/git/diff?path=new.gad", nil))
	if r.From != "old.gad" || r.Old != "one\ntwo\nthree\nfour\n" || r.Old != r.New {
		t.Fatalf("rename diff = %+v", r)
	}
	if w := do(t, h, "GET", "/api/ide/git/diff?path=README", nil); w.Code != http.StatusNotFound {
		t.Fatalf("a file out of the workspace: %d %s", w.Code, w.Body)
	}

	// saved: CRLF as \n, the change gone when it is HEAD's again
	if w := do(t, h, "POST", "/api/ide/git/save", map[string]string{"path": "a.gad", "content": "x := 10\r\ny := 2\r\nz := 3\r\nreturn x\r\n"}); w.Code != 200 {
		t.Fatalf("save: %d %s", w.Code, w.Body)
	}
	b, _ := os.ReadFile(filepath.Join(repo, "src/a.gad"))
	if string(b) != "x := 10\ny := 2\nz := 3\nreturn x\n" {
		t.Fatalf("saved %q", b)
	}
	files = decode[struct{ Files []GitFile }](t, do(t, h, "GET", "/api/ide/git/changes", nil)).Files
	if len(files) != 1 || files[0].Path != "new.gad" {
		t.Fatalf("changes after save = %+v", files)
	}
	for _, p := range []string{"../README", ".git/config", "/etc/passwd"} {
		if w := do(t, h, "POST", "/api/ide/git/save", map[string]string{"path": p, "content": "x"}); w.Code != http.StatusBadRequest {
			t.Fatalf("save %s: %d", p, w.Code)
		}
	}
}

func TestGitBranchesLogCommit(t *testing.T) {
	_, h := gitRepo(t)
	bs := decode[struct{ Branches []GitBranch }](t, do(t, h, "GET", "/api/ide/git/branches", nil)).Branches
	if len(bs) != 2 || bs[0].Name != "feature" || bs[1].Name != "main" || !bs[1].Current || bs[1].Subject != "second" {
		t.Fatalf("branches = %+v", bs)
	}
	cs := decode[struct{ Commits []GitCommit }](t, do(t, h, "GET", "/api/ide/git/log?ref=main", nil)).Commits
	if len(cs) != 2 || cs[0].Subject != "second" || cs[0].Parents[0] != cs[1].Hash || cs[0].Author != "Ann" {
		t.Fatalf("log = %+v", cs)
	}
	if one := decode[struct{ Commits []GitCommit }](t, do(t, h, "GET", "/api/ide/git/log?ref=main&skip=1&limit=5", nil)).Commits; len(one) != 1 || one[0].Hash != cs[1].Hash {
		t.Fatalf("log skip = %+v", one)
	}
	if w := do(t, h, "GET", "/api/ide/git/log?ref=--output=/tmp/x", nil); w.Code != http.StatusInternalServerError || !strings.Contains(w.Body.String(), "invalid ref") {
		t.Fatalf("an option as ref: %d %s", w.Code, w.Body)
	}

	// the commit: its message whole, its files of the workspace (README not)
	d := decode[GitCommitDetail](t, do(t, h, "GET", "/api/ide/git/commit?hash="+cs[0].Hash, nil))
	if d.Message != "second\n\nThe body of the message." || len(d.Files) != 1 || d.Files[0].Path != "a.gad" || d.Files[0].Status != "M" {
		t.Fatalf("commit = %+v", d)
	}
	root := decode[GitCommitDetail](t, do(t, h, "GET", "/api/ide/git/commit?hash="+cs[1].Hash, nil))
	if len(root.Files) != 2 || root.Files[0].Status != "A" {
		t.Fatalf("root commit = %+v", root)
	}
	f := decode[GitFile](t, do(t, h, "GET", "/api/ide/git/commit/diff?hash="+cs[0].Hash+"&path=a.gad", nil))
	if !strings.HasPrefix(f.Old, "x := 1\n") || !strings.HasPrefix(f.New, "x := 10\n") {
		t.Fatalf("commit diff = %+v", f)
	}

	// the downloads
	w := do(t, h, "GET", "/api/ide/git/file?hash="+cs[1].Hash+"&path=a.gad", nil)
	if w.Body.String() != "x := 1\ny := 2\nz := 3\nreturn x\n" || !strings.Contains(w.Header().Get("Content-Disposition"), `filename="a.gad"`) {
		t.Fatalf("file: %q %v", w.Body, w.Header())
	}
	w = do(t, h, "GET", "/api/ide/git/patch?hash="+cs[0].Hash+"&path=a.gad", nil)
	if p := w.Body.String(); !strings.Contains(p, "-x := 1\n+x := 10") || strings.Contains(p, "README") || !strings.Contains(p, "Subject: [PATCH] second") {
		t.Fatalf("patch: %s", p)
	}
	if w = do(t, h, "GET", "/api/ide/git/patch?hash="+cs[1].Hash, nil); !strings.Contains(w.Body.String(), "+one") {
		t.Fatalf("root patch: %s", w.Body)
	}
	if w = do(t, h, "GET", "/api/ide/git/file?hash="+cs[0].Hash+"&path=../README", nil); w.Code != http.StatusBadRequest {
		t.Fatalf("a file out of the workspace: %d", w.Code)
	}
}

func TestGitDefaultFileHistoryCompare(t *testing.T) {
	repo, h := gitRepo(t)
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		cmd.Env = append(gitEnv(), "GIT_AUTHOR_NAME=Ann", "GIT_AUTHOR_EMAIL=ann@example.com",
			"GIT_COMMITTER_NAME=Ann", "GIT_COMMITTER_EMAIL=ann@example.com")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	// the default branch: main, though feature is checked out
	run("checkout", "-q", "feature")
	res := decode[struct {
		Branches []GitBranch
		Default  string
	}](t, do(t, h, "GET", "/api/ide/git/branches", nil))
	if res.Default != "main" {
		t.Fatalf("default = %q", res.Default)
	}
	run("checkout", "-q", "main")

	// a.gad renamed to b.gad, then changed: its history follows the rename
	run("mv", "src/a.gad", "src/b.gad")
	run("commit", "-q", "-m", "rename")
	if err := os.WriteFile(filepath.Join(repo, "src", "b.gad"), []byte("x := 10\ny := 2\nz := 3\nreturn y\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("commit", "-q", "-am", "third")
	cs := decode[struct{ Commits []GitCommit }](t, do(t, h, "GET", "/api/ide/git/log?ref=main&path=b.gad", nil)).Commits
	if len(cs) != 4 || cs[0].Subject != "third" || cs[0].Path != "b.gad" ||
		cs[1].Subject != "rename" || cs[1].Path != "b.gad" || cs[1].From != "a.gad" ||
		cs[2].Path != "a.gad" || cs[3].Subject != "first" || cs[3].Path != "a.gad" {
		t.Fatalf("history = %+v", cs)
	}
	if w := do(t, h, "GET", "/api/ide/git/log?ref=main&path=../README", nil); w.Code != http.StatusBadRequest {
		t.Fatalf("a file out of the workspace: %d", w.Code)
	}

	// a branch with another: what changed, of the workspace (README not)
	files := decode[struct{ Files []GitFile }](t, do(t, h, "GET", "/api/ide/git/compare?a=feature&b=main", nil)).Files
	if len(files) != 2 || files[0].Path != "a.gad" || files[0].Status != "D" || files[1].Path != "b.gad" || files[1].Status != "A" {
		t.Fatalf("compare = %+v", files)
	}
	// a commit with another: the rename
	files = decode[struct{ Files []GitFile }](t, do(t, h, "GET", "/api/ide/git/compare?a="+cs[2].Hash+"&b="+cs[1].Hash, nil)).Files
	if len(files) != 1 || files[0].Status != "R" || files[0].From != "a.gad" || files[0].Path != "b.gad" {
		t.Fatalf("compare commits = %+v", files)
	}
	f := decode[GitFile](t, do(t, h, "GET", "/api/ide/git/compare/diff?a=feature&b=main&path=b.gad&from=a.gad", nil))
	if !strings.HasPrefix(f.Old, "x := 1\n") || !strings.HasSuffix(f.New, "return y\n") {
		t.Fatalf("compare diff = %+v", f)
	}
	// a commit with another, by their hashes
	f = decode[GitFile](t, do(t, h, "GET", "/api/ide/git/compare/diff?a="+cs[3].Hash+"&b="+cs[2].Hash+"&path=a.gad", nil))
	if !strings.HasPrefix(f.Old, "x := 1\n") || !strings.HasPrefix(f.New, "x := 10\n") {
		t.Fatalf("commits diff = %+v", f)
	}
	// a ref with the working files
	if err := os.WriteFile(filepath.Join(repo, "src", "b.gad"), []byte("working\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f = decode[GitFile](t, do(t, h, "GET", "/api/ide/git/compare/diff?a=main&path=b.gad", nil))
	if !strings.HasSuffix(f.Old, "return y\n") || f.New != "working\n" {
		t.Fatalf("working diff = %+v", f)
	}
	if w := do(t, h, "GET", "/api/ide/git/compare?a=--output=/tmp/x&b=main", nil); w.Code != http.StatusInternalServerError {
		t.Fatalf("an option as ref: %d", w.Code)
	}
}

func TestGitLogFiltersAuthorsBlame(t *testing.T) {
	repo, h := gitRepo(t)
	commit := func(name, email, date, msg string, files map[string]string) {
		t.Helper()
		for p, c := range files {
			if err := os.WriteFile(filepath.Join(repo, filepath.FromSlash(p)), []byte(c), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		cmd := exec.Command("git", "commit", "-q", "-am", msg)
		cmd.Dir = repo
		cmd.Env = append(gitEnv(), "GIT_AUTHOR_NAME="+name, "GIT_AUTHOR_EMAIL="+email, "GIT_AUTHOR_DATE="+date,
			"GIT_COMMITTER_NAME="+name, "GIT_COMMITTER_EMAIL="+email, "GIT_COMMITTER_DATE="+date)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("commit: %v: %s", err, out)
		}
	}
	// main: first, second (Ann, now); then Bob's two, dated
	commit("Bob", "bob@example.com", "2020-01-10T10:00:00Z", "Fix the Parser\n\nIt broke on tabs.",
		map[string]string{"src/old.gad": "one\ntwo\nthree\nfour\nneedle here\n"})
	commit("Bob", "bob@example.com", "2020-03-01T10:00:00Z", "tune a",
		map[string]string{"src/a.gad": "x := 10\ny := 20\nz := 3\nreturn x\n"})
	logOf := func(query string) []GitCommit {
		t.Helper()
		w := do(t, h, "GET", "/api/ide/git/log?ref=main&"+query, nil)
		if w.Code != http.StatusOK {
			t.Fatalf("log %s: %d %s", query, w.Code, w.Body)
		}
		return decode[struct{ Commits []GitCommit }](t, w).Commits
	}
	subjects := func(cs []GitCommit) string {
		var s []string
		for _, c := range cs {
			s = append(s, c.Subject)
		}
		return strings.Join(s, ",")
	}
	for _, c := range []struct{ query, want string }{
		{"author=Bob+%3Cbob%40example.com%3E", "tune a,Fix the Parser"},
		{"author=bob&author=ann", "tune a,Fix the Parser,second,first"},
		{"q=parser", "Fix the Parser"},
		{"message=BROKE+ON", "Fix the Parser"},
		{"q=fix&message=tabs", "Fix the Parser"},
		{"q=fix&message=nothing", ""},
		{"name=OLD", "Fix the Parser,first"},
		{"name=*.gad&author=Ann", "second,first"},
		{"content=needle", "Fix the Parser"},
		{"content=y+%3A%3D+20", "tune a"},
		{"since=2020-01-01&until=2020-02-01", "Fix the Parser"},
		{"until=2020-02-01", "Fix the Parser"},
		{"author=Bob&since=2020-02-01", "tune a"},
	} {
		if got := subjects(logOf(c.query)); got != c.want {
			t.Errorf("log %s = %q, want %q", c.query, got, c.want)
		}
	}
	// a hash: its commit first
	all := logOf("")
	if cs := logOf("q=" + all[3].Hash[:8]); len(cs) != 1 || cs[0].Hash != all[3].Hash {
		t.Errorf("by hash = %q", subjects(cs))
	}
	// a file's history, filtered
	if got := subjects(logOf("path=a.gad&author=Bob")); got != "tune a" {
		t.Errorf("history by Bob = %q", got)
	}
	if got := subjects(logOf("path=a.gad&content=x+%3A%3D+10")); got != "second" {
		t.Errorf("history by content = %q", got)
	}
	for _, q := range []string{"since=yesterday", "path=a.gad&name=x", "q=a%0Ab"} {
		if w := do(t, h, "GET", "/api/ide/git/log?ref=main&"+q, nil); w.Code == http.StatusOK {
			t.Errorf("log %s: %d, want an error", q, w.Code)
		}
	}

	// the authors, of the workspace and of a file
	as := decode[struct{ Authors []GitAuthor }](t, do(t, h, "GET", "/api/ide/git/authors?ref=main", nil)).Authors
	if len(as) != 2 || as[0].Commits != 2 || as[1].Commits != 2 || as[0].Email == "" {
		t.Errorf("authors = %+v", as)
	}
	as = decode[struct{ Authors []GitAuthor }](t, do(t, h, "GET", "/api/ide/git/authors?ref=main&path=old.gad", nil)).Authors
	if len(as) != 2 {
		t.Errorf("authors of old.gad = %+v", as)
	}

	// the blame: of a commit, of the working file
	b := decode[GitBlame](t, do(t, h, "GET", "/api/ide/git/blame?ref=main&path=a.gad", nil))
	if len(b.Lines) != 4 || b.Lines[1].Content != "y := 20" || b.Commits[b.Lines[1].Hash].Author != "Bob" ||
		b.Commits[b.Lines[0].Hash].Summary != "second" || b.Commits[b.Lines[2].Hash].Summary != "first" ||
		!strings.HasPrefix(b.Commits[b.Lines[1].Hash].Date, "2020-03-01") {
		t.Errorf("blame = %+v", b)
	}
	if err := os.WriteFile(filepath.Join(repo, "src", "a.gad"), []byte("x := 10\nnew\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	b = decode[GitBlame](t, do(t, h, "GET", "/api/ide/git/blame?path=a.gad", nil))
	if len(b.Lines) != 2 || b.Lines[1].Hash != notCommitted || b.Lines[1].Content != "new" {
		t.Errorf("working blame = %+v", b)
	}
	if w := do(t, h, "GET", "/api/ide/git/blame?ref=main&path=../README", nil); w.Code != http.StatusBadRequest {
		t.Errorf("blame out of the workspace: %d", w.Code)
	}
}
