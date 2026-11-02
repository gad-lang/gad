package ide

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// The stashes of the repository, as an IDE manages them:
//
//	GET  git/stashes                     the stashes, newest first
//	GET  git/stash?hash=                 one: its message and its files — the
//	                                     changes it keeps (from the commit it
//	                                     was made on) and its untracked files
//	GET  git/stash/diff?hash=&path=[&from=][&untracked=1]  a file of it
//	POST git/stash/new {message, untracked, keepIndex}  the changes of the
//	                                     workspace stashed — {ref}
//	POST git/stash/apply {ref, pop, index}  put back (pop: dropped then;
//	                                     index: the index too)
//	POST git/stash/branch {ref, branch}  put back in a new branch, from the
//	                                     commit it was made on (dropped)
//	POST git/stash/drop {ref}            dropped
//	POST git/stash/clear                 all dropped
//
// Those that make one (new) take the author of X-Git-Author, as a commit.

// GitStash is a stash: its ref (stash@{n}), commit, message, the branch it
// was made on, its date.
type GitStash struct {
	Index   int    `json:"index"`
	Ref     string `json:"ref"`
	Hash    string `json:"hash"`
	Message string `json:"message"`
	Branch  string `json:"branch,omitempty"`
	Date    string `json:"date"`
}

var (
	stashRefRe        = regexp.MustCompile(`^stash@\{(\d+)\}$`)
	stashSubjectRe    = regexp.MustCompile(`^(?:WIP on|On) ([^:]+): (.*)$`)
	errNothingToStash = errors.New("no local changes to stash")
	errStashRef       = errors.New("invalid stash")
	errBranchName     = errors.New("invalid branch name")
)

// GitStashes are the stashes, the newest first.
func (s *Server) GitStashes(ctx context.Context) ([]GitStash, error) {
	if _, err := s.gitPrefix(ctx); err != nil {
		return nil, err
	}
	out, err := s.gitCmd(ctx, "stash", "list", "--format=%gd%x00%H%x00%gs%x00%aI")
	if err != nil {
		return nil, err
	}
	stashes := []GitStash{}
	for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		f := strings.Split(l, "\x00")
		if len(f) != 4 {
			continue
		}
		st := GitStash{Ref: f[0], Hash: f[1], Message: f[2], Date: f[3]}
		if m := stashRefRe.FindStringSubmatch(f[0]); m != nil {
			st.Index, _ = strconv.Atoi(m[1])
		}
		if m := stashSubjectRe.FindStringSubmatch(f[2]); m != nil {
			st.Branch, st.Message = m[1], m[2]
		}
		stashes = append(stashes, st)
	}
	return stashes, nil
}

// untrackedOf is the commit of the untracked files of the stash hash (its
// third parent), "" when it has none.
func (s *Server) untrackedOf(ctx context.Context, hash string) string {
	out, err := s.gitCmd(ctx, "rev-list", "--parents", "-n", "1", hash)
	if err != nil {
		return ""
	}
	if f := strings.Fields(string(out)); len(f) > 3 {
		return f[3]
	}
	return ""
}

// GitStashOf is the stash hash: as a commit (its message, the files it
// changed from the commit it was made on — of the workspace's folder) and
// its untracked files ("?").
func (s *Server) GitStashOf(ctx context.Context, hash string) (*GitCommitDetail, error) {
	d, err := s.GitCommitOf(ctx, hash)
	if err != nil {
		return nil, err
	}
	if u := s.untrackedOf(ctx, hash); u != "" {
		out, err := s.gitCmd(ctx, "ls-tree", "-r", "-z", "--name-only", u, "--", ".")
		if err == nil {
			for _, p := range strings.Split(string(out), "\x00") {
				if p != "" {
					d.Files = append(d.Files, GitFile{Path: p, Status: "?"})
				}
			}
		}
	}
	return d, nil
}

// GitStashDiff is the diff of the file p of the stash hash: an untracked
// one's content alone, else as a commit's.
func (s *Server) GitStashDiff(ctx context.Context, hash, p, from string, untracked bool) (*GitFile, error) {
	if !hashRe.MatchString(hash) {
		return nil, errStashRef
	}
	if !untracked {
		return s.GitCommitDiff(ctx, hash, p, from)
	}
	u := s.untrackedOf(ctx, hash)
	if u == "" {
		return nil, errStashRef
	}
	f := &GitFile{Path: p, Status: "?"}
	f.New, f.Binary, _ = s.gitShow(ctx, u, p)
	if f.Binary {
		f.New = ""
	}
	return f, nil
}

// authorEnv is the environment of author ("Name <email>"): the author and
// the committer; none when it is not one.
func authorEnv(author string) []string {
	m := authorRe.FindStringSubmatch(author)
	if m == nil {
		return nil
	}
	return []string{"GIT_AUTHOR_NAME=" + m[1], "GIT_AUTHOR_EMAIL=" + m[2], "GIT_COMMITTER_NAME=" + m[1], "GIT_COMMITTER_EMAIL=" + m[2]}
}

// GitStashNew stashes the changes of the workspace (untracked: its new
// files too; keepIndex: what is staged left there) with message: its ref.
func (s *Server) GitStashNew(ctx context.Context, message string, untracked, keepIndex bool, author string) (string, error) {
	if _, err := s.gitPrefix(ctx); err != nil {
		return "", err
	}
	before, _ := s.gitCmd(ctx, "rev-parse", "-q", "--verify", "refs/stash")
	args := []string{"stash", "push", "-q"}
	if untracked {
		args = append(args, "--include-untracked")
	}
	if keepIndex {
		args = append(args, "--keep-index")
	}
	if strings.TrimSpace(message) != "" {
		args = append(args, "-m", message)
	}
	args = append(args, "--", ".")
	out, err := s.gitCmdEnv(ctx, authorEnv(author), args...)
	if err != nil {
		if strings.Contains(err.Error(), "No local changes") || strings.Contains(err.Error(), "did not match any file") {
			return "", errNothingToStash
		}
		return "", err
	}
	after, _ := s.gitCmd(ctx, "rev-parse", "-q", "--verify", "refs/stash")
	if strings.Contains(string(out), "No local changes") || string(after) == string(before) {
		return "", errNothingToStash
	}
	return "stash@{0}", nil
}

// stashRef is ref checked: stash@{n}.
func stashRef(ref string) (string, error) {
	if !stashRefRe.MatchString(ref) {
		return "", errStashRef
	}
	return ref, nil
}

// GitStashApply puts the stash ref back (pop: dropped then; index: the
// index too).
func (s *Server) GitStashApply(ctx context.Context, ref string, pop, index bool) error {
	ref, err := stashRef(ref)
	if err != nil {
		return err
	}
	args := []string{"stash", "apply", "-q"}
	if pop {
		args[1] = "pop"
	}
	if index {
		args = append(args, "--index")
	}
	_, err = s.gitCmd(ctx, append(args, ref)...)
	return err
}

// GitStashBranch puts the stash ref back in the new branch name, made from
// the commit it was made on (the stash dropped when it applies).
func (s *Server) GitStashBranch(ctx context.Context, ref, name string) error {
	ref, err := stashRef(ref)
	if err != nil {
		return err
	}
	if name == "" || strings.HasPrefix(name, "-") {
		return errBranchName
	}
	if _, err := s.gitCmd(ctx, "check-ref-format", "--branch", name); err != nil {
		return errBranchName
	}
	_, err = s.gitCmd(ctx, "stash", "branch", name, ref)
	return err
}

// GitStashDrop drops the stash ref.
func (s *Server) GitStashDrop(ctx context.Context, ref string) error {
	ref, err := stashRef(ref)
	if err != nil {
		return err
	}
	_, err = s.gitCmd(ctx, "stash", "drop", "-q", ref)
	return err
}

// GitStashClear drops all the stashes.
func (s *Server) GitStashClear(ctx context.Context) error {
	_, err := s.gitCmd(ctx, "stash", "clear")
	return err
}

// stashPosts are the stash's routes that change (POST).
var stashPosts = map[string]bool{
	"git/stash/new": true, "git/stash/apply": true, "git/stash/branch": true,
	"git/stash/drop": true, "git/stash/clear": true,
}

// serveStash serves git/stash… (op).
func (s *Server) serveStash(w http.ResponseWriter, r *http.Request, op string) {
	ctx, q := r.Context(), r.URL.Query()
	post := stashPosts[op]
	if post != (r.Method == http.MethodPost) || (!post && r.Method != http.MethodGet) {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	answer := func(v any, err error) {
		switch {
		case errors.Is(err, errNothingToStash), errors.Is(err, errStashRef), errors.Is(err, errBranchName), errors.Is(err, errOutsideRoot):
			writeError(w, http.StatusBadRequest, err.Error())
		case err != nil:
			// git's own refusals (a conflict, the branch there) said as they are
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeJSON(w, v)
		}
	}
	var req struct {
		Message   string `json:"message"`
		Untracked bool   `json:"untracked"`
		KeepIndex bool   `json:"keepIndex"`
		Ref       string `json:"ref"`
		Pop       bool   `json:"pop"`
		Index     bool   `json:"index"`
		Branch    string `json:"branch"`
	}
	if post && op != "git/stash/clear" {
		if err := decodeBody(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	ok := map[string]bool{"ok": true}
	switch op {
	case "git/stashes":
		st, err := s.GitStashes(ctx)
		answer(map[string]any{"stashes": st}, err)
	case "git/stash":
		d, err := s.GitStashOf(ctx, q.Get("hash"))
		answer(d, err)
	case "git/stash/diff":
		p, err := gitPath(q.Get("path"))
		from := ""
		if err == nil && q.Get("from") != "" {
			from, err = gitPath(q.Get("from"))
		}
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid path")
			return
		}
		f, err := s.GitStashDiff(ctx, q.Get("hash"), p, from, q.Get("untracked") == "1")
		answer(f, err)
	case "git/stash/new":
		ref, err := s.GitStashNew(ctx, req.Message, req.Untracked, req.KeepIndex, r.Header.Get("X-Git-Author"))
		answer(map[string]string{"ref": ref}, err)
	case "git/stash/apply":
		answer(ok, s.GitStashApply(ctx, req.Ref, req.Pop, req.Index))
	case "git/stash/branch":
		answer(ok, s.GitStashBranch(ctx, req.Ref, req.Branch))
	case "git/stash/drop":
		answer(ok, s.GitStashDrop(ctx, req.Ref))
	case "git/stash/clear":
		answer(ok, s.GitStashClear(ctx))
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}
