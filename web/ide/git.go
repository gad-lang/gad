package ide

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// The workspace's git repository, when it is in one: the IDE's Changes panel
// (the files changed and not committed, compared with HEAD's, edited) and its
// Git panel (the branches, their commits, what each changed — read only).
// git is the binary; paths are the workspace's, Root possibly a folder of
// the repository (only its files are shown).
//
//	GET  git/changes                  {"files": [summary]}
//	GET  git/diff?path&from           a change's diff ({path, from, old, new, binary})
//	POST git/save {path, content}     the file written
//	GET  git/branches                 {"branches": [{name, hash, remote, current, subject}]}
//	GET  git/log?ref&skip&limit       {"commits": [{hash, parents, author, email, date, subject, refs}]}
//	GET  git/commit?hash              the commit, its full message and files (from its first parent)
//	GET  git/commit/diff?hash&path&from  a file's diff in the commit
//	GET  git/file?hash&path           the file as the commit has it (a download)
//	GET  git/patch?hash[&path]        the commit's patch, of a file or whole (a download)

// GitFile is a file of a diff, as the IDE's diff browser has it: its
// summary — path, origin of a renamed one, status (M, A, D, R, ?) — and,
// asked for alone, its contents.
type GitFile struct {
	Path   string `json:"path"`
	From   string `json:"from,omitempty"`
	Status string `json:"status,omitempty"`
	Old    string `json:"old,omitempty"`
	New    string `json:"new,omitempty"`
	Binary bool   `json:"binary,omitempty"`
}

// GitBranch is a branch of the repository.
type GitBranch struct {
	Name    string `json:"name"`
	Hash    string `json:"hash"`
	Remote  bool   `json:"remote,omitempty"`
	Current bool   `json:"current,omitempty"`
	Subject string `json:"subject,omitempty"`
}

// GitCommit is a commit of a log.
type GitCommit struct {
	Hash    string   `json:"hash"`
	Parents []string `json:"parents"`
	Author  string   `json:"author"`
	Email   string   `json:"email"`
	Date    string   `json:"date"`
	Subject string   `json:"subject"`
	Refs    []string `json:"refs,omitempty"`
}

// GitCommitDetail is a commit, its full message and the files it changed.
type GitCommitDetail struct {
	GitCommit
	Message string    `json:"message"`
	Files   []GitFile `json:"files"`
}

// maxDiffSize is the largest content compared: a larger one is not (binary).
const maxDiffSize = 2 << 20

// gitTimeout bounds a git command.
const gitTimeout = 30 * time.Second

var (
	hashRe   = regexp.MustCompile(`^[0-9a-fA-F]{4,64}$`)
	errNoGit = errors.New("the workspace is not in a git repository")
)

// gitCmd runs git in the workspace; its error carries what git said.
func (s *Server) gitCmd(ctx context.Context, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, gitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = s.Root
	cmd.Env = gitEnv()
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return out.Bytes(), fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(errb.String()))
	}
	return out.Bytes(), nil
}

// gitEnv is the process's environment without git's own variables (a hook's
// GIT_DIR would point every command elsewhere), no prompt, no optional lock
// (status would take the index's).
func gitEnv() (env []string) {
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "GIT_") {
			env = append(env, e)
		}
	}
	return append(env, "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0", "LC_ALL=C")
}

// GitRepo reports whether the workspace is in a git repository.
func (s *Server) GitRepo(ctx context.Context) bool {
	out, err := s.gitCmd(ctx, "rev-parse", "--is-inside-work-tree")
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

// gitPrefix is the workspace's folder in the repository ("" at its root,
// "sub/dir/" in one).
func (s *Server) gitPrefix(ctx context.Context) (string, error) {
	out, err := s.gitCmd(ctx, "rev-parse", "--show-prefix")
	if err != nil {
		return "", errNoGit
	}
	return strings.TrimSpace(string(out)), nil
}

// gitPath is p as a path of the workspace, checked: clean, in it, not in .git.
func gitPath(p string) (string, error) {
	c := strings.TrimPrefix(path.Clean("/"+strings.ReplaceAll(p, `\`, "/")), "/")
	if c == "" || c != strings.TrimPrefix(p, "./") {
		return "", errOutsideRoot
	}
	for _, part := range strings.Split(c, "/") {
		if part == ".git" {
			return "", errOutsideRoot
		}
	}
	return c, nil
}

// GitChanges are the files of the workspace changed and not committed, a
// file deleted and one not tracked of a content alike paired as a rename.
func (s *Server) GitChanges(ctx context.Context) ([]GitFile, error) {
	prefix, err := s.gitPrefix(ctx)
	if err != nil {
		return nil, err
	}
	out, err := s.gitCmd(ctx, "status", "--porcelain=v1", "-z", "--untracked-files=all", "--", ".")
	if err != nil {
		return nil, err
	}
	var files []GitFile
	parts := strings.Split(string(out), "\x00")
	for i := 0; i < len(parts); i++ {
		e := parts[i]
		if len(e) < 4 {
			continue
		}
		x, y, p := e[0], e[1], e[3:]
		f := GitFile{Path: strings.TrimPrefix(p, prefix)}
		switch {
		case x == '?':
			f.Status = "?"
		case x == 'R' || y == 'R' || x == 'C':
			f.Status = "R"
			if i+1 < len(parts) {
				i++
				f.From = strings.TrimPrefix(parts[i], prefix)
			}
		case x == 'D' || y == 'D':
			f.Status = "D"
		case x == 'A' || y == 'A':
			f.Status = "A"
		default:
			f.Status = "M"
		}
		files = append(files, f)
	}
	files = s.pairRenames(ctx, files)
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

// pairRenames pairs each file deleted with the file new (not tracked,
// added) most alike — half its lines the same, at least —: renamed, as the
// IDE renames (on the disk: git sees one deleted, one new). Of many, none
// (it would read them all).
func (s *Server) pairRenames(ctx context.Context, files []GitFile) []GitFile {
	var del, add []int
	for i, f := range files {
		switch f.Status {
		case "D":
			del = append(del, i)
		case "?", "A":
			add = append(add, i)
		}
	}
	if len(del) == 0 || len(add) == 0 || len(del)*len(add) > 400 {
		return files
	}
	news := map[int]string{}
	for _, j := range add {
		if b, err := os.ReadFile(filepath.Join(s.Root, filepath.FromSlash(files[j].Path))); err == nil && len(b) <= maxDiffSize {
			news[j] = string(b)
		}
	}
	gone := map[int]bool{}
	for _, i := range del {
		old, _, err := s.gitShow(ctx, "HEAD", files[i].Path)
		if err != nil {
			continue
		}
		best, bestSim := -1, 0.5
		for _, j := range add {
			text, ok := news[j]
			if !ok || gone[j] {
				continue
			}
			sim := similarity(old, text)
			if sim > bestSim || (sim == bestSim && best >= 0 && path.Base(files[j].Path) == path.Base(files[i].Path)) {
				best, bestSim = j, sim
			}
		}
		if best < 0 {
			continue
		}
		gone[best] = true
		files[i] = GitFile{Path: files[best].Path, From: files[i].Path, Status: "R"}
	}
	kept := files[:0]
	for j, f := range files {
		if !gone[j] {
			kept = append(kept, f)
		}
	}
	return kept
}

// similarity is how alike two texts are: the lines they have in common, of
// all (Dice's), 1 for two empty.
func similarity(a, b string) float64 {
	la, lb := strings.Split(a, "\n"), strings.Split(b, "\n")
	if a == b {
		return 1
	}
	count := map[string]int{}
	for _, l := range la {
		count[l]++
	}
	common := 0
	for _, l := range lb {
		if count[l] > 0 {
			count[l]--
			common++
		}
	}
	return 2 * float64(common) / float64(len(la)+len(lb))
}

// gitShow is the content of the file p (of the workspace) at rev; binary
// when it has a NUL byte or is too large to compare.
func (s *Server) gitShow(ctx context.Context, rev, p string) (string, bool, error) {
	out, err := s.gitCmd(ctx, "show", rev+":./"+p)
	if err != nil {
		return "", false, err
	}
	return contentOf(out)
}

func contentOf(b []byte) (string, bool, error) {
	if len(b) > maxDiffSize || bytes.IndexByte(b[:min(len(b), 8000)], 0) >= 0 {
		return "", true, nil
	}
	return string(b), false, nil
}

// GitDiff is the diff of the change of the file p: HEAD's content (of from,
// a renamed one's origin) and the current one. nil when p is no change.
func (s *Server) GitDiff(ctx context.Context, p string) (*GitFile, error) {
	changes, err := s.GitChanges(ctx)
	if err != nil {
		return nil, err
	}
	for _, f := range changes {
		if f.Path != p {
			continue
		}
		var oldBin, newBin bool
		if f.Status != "?" && f.Status != "A" {
			src := f.Path
			if f.From != "" {
				src = f.From
			}
			if f.Old, oldBin, err = s.gitShow(ctx, "HEAD", src); err != nil {
				f.Old = ""
			}
		}
		if f.Status != "D" {
			if b, err := os.ReadFile(filepath.Join(s.Root, filepath.FromSlash(f.Path))); err == nil {
				f.New, newBin, _ = contentOf(b)
			}
		}
		if f.Binary = oldBin || newBin; f.Binary {
			f.Old, f.New = "", ""
		}
		return &f, nil
	}
	return nil, nil
}

// GitSave writes the file p with content, as the compare saves it: its line
// endings kept (a form sends CRLF: \n, unless the file has \r\n).
func (s *Server) GitSave(p, content string) error {
	abs, err := s.resolve(p)
	if err != nil {
		return err
	}
	if was, err := os.ReadFile(abs); err != nil || !bytes.Contains(was, []byte("\r\n")) {
		content = strings.ReplaceAll(content, "\r\n", "\n")
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	return os.WriteFile(abs, []byte(content), 0o644)
}

// GitBranches are the branches, local and remote (a remote's HEAD aside).
func (s *Server) GitBranches(ctx context.Context) ([]GitBranch, error) {
	out, err := s.gitCmd(ctx, "for-each-ref",
		"--format=%(refname)%00%(refname:short)%00%(objectname)%00%(HEAD)%00%(contents:subject)",
		"refs/heads", "refs/remotes")
	if err != nil {
		return nil, err
	}
	branches := []GitBranch{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		f := strings.Split(line, "\x00")
		if len(f) < 5 || strings.HasSuffix(f[0], "/HEAD") {
			continue
		}
		branches = append(branches, GitBranch{Name: f[1], Hash: f[2], Remote: strings.HasPrefix(f[0], "refs/remotes/"),
			Current: f[3] == "*", Subject: f[4]})
	}
	return branches, nil
}

const logFormat = "%H%x00%P%x00%an%x00%ae%x00%aI%x00%s%x00%D"

func parseCommit(fields []string) GitCommit {
	c := GitCommit{Hash: fields[0], Parents: strings.Fields(fields[1]), Author: fields[2], Email: fields[3], Date: fields[4], Subject: fields[5]}
	for _, r := range strings.Split(fields[6], ", ") {
		if r = strings.TrimSpace(r); r != "" {
			c.Refs = append(c.Refs, r)
		}
	}
	return c
}

// validRef says whether ref may be given to git as a revision: no option, no
// space nor control character, no range.
func validRef(ref string) bool {
	if ref == "" || strings.HasPrefix(ref, "-") || strings.Contains(ref, "..") {
		return false
	}
	for _, r := range ref {
		if r <= ' ' || r == 0x7f || r == ':' {
			return false
		}
	}
	return true
}

// GitLog are limit commits of ref after skip, each before its parents; in a
// folder of the repository, those that changed it.
func (s *Server) GitLog(ctx context.Context, ref string, skip, limit int) ([]GitCommit, error) {
	if !validRef(ref) {
		return nil, fmt.Errorf("invalid ref %q", ref)
	}
	prefix, err := s.gitPrefix(ctx)
	if err != nil {
		return nil, err
	}
	args := []string{"log", "--topo-order", "--format=" + logFormat + "%x1e",
		"--skip=" + strconv.Itoa(skip), "-n", strconv.Itoa(limit)}
	if prefix != "" {
		args = append(args, "--parents")
	}
	args = append(args, ref, "--")
	if prefix != "" {
		args = append(args, ".")
	}
	out, err := s.gitCmd(ctx, args...)
	if err != nil {
		return nil, err
	}
	commits := []GitCommit{}
	for _, rec := range strings.Split(string(out), "\x1e") {
		f := strings.Split(strings.TrimLeft(rec, "\n"), "\x00")
		if len(f) < 7 {
			continue
		}
		commits = append(commits, parseCommit(f))
	}
	return commits, nil
}

// GitCommitOf is the commit hash: its full message and the files it changed
// from its first parent (all, of a root commit) — of the workspace's folder.
func (s *Server) GitCommitOf(ctx context.Context, hash string) (*GitCommitDetail, error) {
	if !hashRe.MatchString(hash) {
		return nil, fmt.Errorf("invalid commit %q", hash)
	}
	out, err := s.gitCmd(ctx, "show", "-s", "--format="+logFormat+"%x00%B", hash)
	if err != nil {
		return nil, err
	}
	f := strings.SplitN(string(out), "\x00", 8)
	if len(f) < 8 {
		return nil, fmt.Errorf("commit %s: unexpected output", hash)
	}
	d := &GitCommitDetail{GitCommit: parseCommit(f), Message: strings.TrimRight(f[7], "\n")}
	args := []string{"diff-tree", "-r", "-z", "-M", "--name-status", "--relative", "--no-commit-id"}
	if len(d.Parents) > 0 {
		args = append(args, d.Parents[0], d.Hash)
	} else {
		args = append(args, "--root", d.Hash)
	}
	out, err = s.gitCmd(ctx, args...)
	if err != nil {
		return nil, err
	}
	d.Files = []GitFile{}
	parts := strings.Split(string(out), "\x00")
	for i := 0; i < len(parts); i++ {
		st := parts[i]
		if st == "" || i+1 >= len(parts) {
			continue
		}
		file := GitFile{Status: st[:1]}
		if file.Status == "R" || file.Status == "C" {
			if i+2 >= len(parts) {
				break
			}
			file.From, file.Path = parts[i+1], parts[i+2]
			file.Status = "R"
			i += 2
		} else {
			file.Path = parts[i+1]
			i++
		}
		d.Files = append(d.Files, file)
	}
	return d, nil
}

// firstParent is the first parent of the commit hash ("" for a root one).
func (s *Server) firstParent(ctx context.Context, hash string) (string, error) {
	if !hashRe.MatchString(hash) {
		return "", fmt.Errorf("invalid commit %q", hash)
	}
	out, err := s.gitCmd(ctx, "rev-list", "--parents", "-n", "1", hash)
	if err != nil {
		return "", err
	}
	f := strings.Fields(string(out))
	if len(f) > 1 {
		return f[1], nil
	}
	return "", nil
}

// GitCommitDiff is the diff of the file p in the commit hash: its content in
// the commit before (of from, a renamed one's origin) and in the commit.
func (s *Server) GitCommitDiff(ctx context.Context, hash, p, from string) (*GitFile, error) {
	parent, err := s.firstParent(ctx, hash)
	if err != nil {
		return nil, err
	}
	f := &GitFile{Path: p, From: from}
	var oldBin, newBin bool
	if parent != "" {
		src := p
		if from != "" {
			src = from
		}
		f.Old, oldBin, _ = s.gitShow(ctx, parent, src)
	}
	f.New, newBin, _ = s.gitShow(ctx, hash, p)
	if f.Binary = oldBin || newBin; f.Binary {
		f.Old, f.New = "", ""
	}
	return f, nil
}

// serveGit serves git/… (op): the repository's routes (see above).
func (s *Server) serveGit(w http.ResponseWriter, r *http.Request) {
	op := strings.TrimPrefix(r.URL.Path, "/api/ide/")
	ctx := r.Context()
	q := r.URL.Query()
	if op == "git/save" {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
	} else if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	// a path given: the workspace's, not in .git
	cleanPath := func(name string) (string, bool) {
		p, err := gitPath(q.Get(name))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid "+name)
			return "", false
		}
		return p, true
	}
	fail := func(err error) { writeError(w, http.StatusInternalServerError, err.Error()) }
	switch op {
	case "git/changes":
		files, err := s.GitChanges(ctx)
		if err != nil {
			fail(err)
			return
		}
		writeJSON(w, map[string]any{"files": files})
	case "git/diff":
		p, ok := cleanPath("path")
		if !ok {
			return
		}
		f, err := s.GitDiff(ctx, p)
		switch {
		case err != nil:
			fail(err)
		case f == nil:
			writeError(w, http.StatusNotFound, "this file is no longer changed")
		default:
			writeJSON(w, f)
		}
	case "git/save":
		var req struct{ Path, Content string }
		if err := decodeBody(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		p, err := gitPath(req.Path)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid path")
			return
		}
		if err := s.GitSave(p, req.Content); err != nil {
			fail(err)
			return
		}
		writeJSON(w, map[string]any{"saved": true})
	case "git/branches":
		bs, err := s.GitBranches(ctx)
		if err != nil {
			fail(err)
			return
		}
		writeJSON(w, map[string]any{"branches": bs})
	case "git/log":
		skip, _ := strconv.Atoi(q.Get("skip"))
		limit, _ := strconv.Atoi(q.Get("limit"))
		if limit <= 0 || limit > 1000 {
			limit = 100
		}
		ref := q.Get("ref")
		if ref == "" {
			ref = "HEAD"
		}
		cs, err := s.GitLog(ctx, ref, max(skip, 0), limit)
		if err != nil {
			fail(err)
			return
		}
		writeJSON(w, map[string]any{"commits": cs})
	case "git/commit":
		d, err := s.GitCommitOf(ctx, q.Get("hash"))
		if err != nil {
			fail(err)
			return
		}
		writeJSON(w, d)
	case "git/commit/diff":
		p, ok := cleanPath("path")
		if !ok {
			return
		}
		from := ""
		if q.Get("from") != "" {
			if from, ok = cleanPath("from"); !ok {
				return
			}
		}
		f, err := s.GitCommitDiff(ctx, q.Get("hash"), p, from)
		if err != nil {
			fail(err)
			return
		}
		writeJSON(w, f)
	case "git/file":
		p, ok := cleanPath("path")
		if !ok {
			return
		}
		hash := q.Get("hash")
		if !hashRe.MatchString(hash) {
			writeError(w, http.StatusBadRequest, "invalid hash")
			return
		}
		out, err := s.gitCmd(ctx, "show", hash+":./"+p)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		download(w, path.Base(p), "application/octet-stream", out)
	case "git/patch":
		hash := q.Get("hash")
		if !hashRe.MatchString(hash) {
			writeError(w, http.StatusBadRequest, "invalid hash")
			return
		}
		spec, name := ".", hash[:min(len(hash), 10)]+".patch"
		if q.Get("path") != "" {
			p, ok := cleanPath("path")
			if !ok {
				return
			}
			spec, name = p, hash[:min(len(hash), 10)]+"-"+path.Base(p)+".patch"
		}
		args := []string{"format-patch", "-1", "--stdout", "--relative", hash, "--", spec}
		if parent, err := s.firstParent(ctx, hash); err == nil && parent == "" {
			args = []string{"format-patch", "--root", "-1", "--stdout", "--relative", hash, "--", spec}
		}
		out, err := s.gitCmd(ctx, args...)
		if err != nil {
			fail(err)
			return
		}
		download(w, name, "text/x-patch; charset=utf-8", out)
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

// download sends b as a file named name, to be saved.
func download(w http.ResponseWriter, name, contentType string, b []byte) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", strings.ReplaceAll(name, `"`, "")))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(b)
}
