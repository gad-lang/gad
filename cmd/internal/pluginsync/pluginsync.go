// Package pluginsync extracts the authoritative Gad language vocabulary
// (keywords, atoms, constants, builtin functions) from the gad/token packages
// and keeps the editor plugins (codemirror-gad, prism-gad) in sync with it.
//
// The plugins hand-maintain JS/TS arrays of keywords and builtins; this package
// derives the current sets from the compiler so the cmd/update-*-plugin tools
// can add what is missing and report drift, instead of editing by hand.
package pluginsync

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	gad "github.com/gad-lang/gad"
	"github.com/gad-lang/gad/token"
)

// Lang is the language vocabulary the plugins highlight.
type Lang struct {
	Keywords  []string // `with`, `for`, `defer_ok`, … (word keywords)
	Atoms     []string // true, false, yes, no, nil
	Constants []string // STDIN, STDOUT, STDERR
	Builtins  []string // global builtin function names (no import)
}

var (
	lowerWord = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	upperWord = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)
	funcName  = regexp.MustCompile(`^[a-z][A-Za-z0-9]*$`)
)

// Extract reads the current vocabulary from the compiler packages.
func Extract() Lang {
	atomSet := map[string]bool{"true": true, "false": true, "yes": true, "no": true, "nil": true}

	var l Lang
	// Word keywords come from the token keyword group; atoms are split out and
	// the `@name`/`STDIN` style specials are skipped (they are not plain words).
	for tk := token.GroupKeywordBegin + 1; tk < token.GroupKeywordEnd; tk++ {
		s := tk.String()
		switch {
		case atomSet[s]:
			l.Atoms = append(l.Atoms, s)
		case lowerWord.MatchString(s):
			l.Keywords = append(l.Keywords, s)
		}
	}
	for _, c := range []token.Token{token.StdIn, token.StdOut, token.StdErr} {
		l.Constants = append(l.Constants, c.String())
	}

	// Global builtin functions are the unqualified BuiltinsMap entries (a `.`
	// marks a namespaced member such as `time.now`) that resolve to a function.
	for name, bt := range gad.BuiltinsMap {
		if strings.Contains(name, ".") {
			continue
		}
		if !funcName.MatchString(name) && !upperWord.MatchString(name) {
			continue
		}
		switch gad.BuiltinObjects[bt].(type) {
		case *gad.BuiltinFunction, *gad.BuiltinFunctionWithMethods:
			l.Builtins = append(l.Builtins, name)
		}
	}

	sort.Strings(l.Keywords)
	sort.Strings(l.Atoms)
	sort.Strings(l.Constants)
	sort.Strings(l.Builtins)
	return l
}

// arrayRe matches a JS/TS array assignment `… <name> … = [ … ]` (the name may be
// `export const name: string[]` or a plain `const name`). It is non-greedy so it
// stops at the first closing bracket; the plugin arrays hold only string items.
func arrayRe(name string) *regexp.Regexp {
	// `<name>` then an optional `: type` annotation (which may itself contain
	// `[]`, e.g. `string[]`, so match up to the `=`), then `= [ … ]`.
	return regexp.MustCompile(`(?s)\b` + regexp.QuoteMeta(name) + `\b\s*(?::[^=]*)?=\s*\[(.*?)\]`)
}

var stringItem = regexp.MustCompile(`"([^"]*)"`)

// ParseArray returns the string items of the named array in src, or ok=false
// when the array is absent.
func ParseArray(src, name string) (items []string, ok bool) {
	m := arrayRe(name).FindStringSubmatch(src)
	if m == nil {
		return nil, false
	}
	for _, im := range stringItem.FindAllStringSubmatch(m[1], -1) {
		items = append(items, im[1])
	}
	return items, true
}

// InsertItems adds additions to the named array (just before its closing `]`),
// indented like a fresh trailing line. It returns the new source and the items
// actually added (those not already present).
func InsertItems(src, name string, additions []string) (string, []string) {
	re := arrayRe(name)
	loc := re.FindStringSubmatchIndex(src)
	if loc == nil {
		return src, nil
	}
	existing := map[string]bool{}
	if items, _ := ParseArray(src, name); items != nil {
		for _, it := range items {
			existing[it] = true
		}
	}
	var added []string
	for _, a := range additions {
		if !existing[a] {
			added = append(added, a)
		}
	}
	if len(added) == 0 {
		return src, nil
	}
	// loc[2]:loc[3] is the captured array body; insert before its end.
	end := loc[3]
	quoted := make([]string, len(added))
	for i, a := range added {
		quoted[i] = fmt.Sprintf("%q", a)
	}
	line := "  // added by update plugin\n  " + strings.Join(quoted, ", ") + ",\n"
	return src[:end] + line + src[end:], added
}

// Diff returns the entries present in want but not in have (added) and in have
// but not in want (removed).
func Diff(want, have []string) (added, removed []string) {
	w, h := toSet(want), toSet(have)
	for _, s := range want {
		if !h[s] {
			added = append(added, s)
		}
	}
	for _, s := range have {
		if !w[s] {
			removed = append(removed, s)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	return
}

func toSet(s []string) map[string]bool {
	m := make(map[string]bool, len(s))
	for _, v := range s {
		m[v] = true
	}
	return m
}

// LastCommit returns the short hash of the most recent commit touching dir, or
// "" when the path has no history yet.
func LastCommit(dir string) string {
	out, err := exec.Command("git", "log", "-1", "--format=%h", "--", dir).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// LangCommitsSince lists the one-line commits that touched the language sources
// (token, parser, compiler, builtins) since the given commit. When since is
// empty it returns the most recent such commits.
func LangCommitsSince(since string) []string {
	args := []string{"log", "--oneline"}
	if since != "" {
		args = append(args, since+"..HEAD")
	} else {
		args = append(args, "-20")
	}
	args = append(args, "--",
		"token", "parser", "compiler.go", "compiler_nodes.go",
		"builtins.go", "builtin_operators.go", "op_api.go")
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return nil
	}
	var lines []string
	for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

// Target describes a plugin to keep in sync.
type Target struct {
	Name   string   // display name, e.g. "codemirror-gad"
	Dir    string   // plugin directory (for git history)
	File   string   // source file holding the vocabulary arrays
	Arrays []string // array names to sync: keywords, atoms, constants, builtins
}

// authFor returns the authoritative set for a plugin array name.
func (l Lang) authFor(name string) ([]string, bool) {
	switch name {
	case "keywords":
		return l.Keywords, true
	case "atoms":
		return l.Atoms, true
	case "constants":
		return l.Constants, true
	case "builtins":
		return l.Builtins, true
	}
	return nil, false
}

// Run syncs a target file against the current language vocabulary. Missing
// entries are added (when write is set); plugin-only entries are reported but
// never removed (some are intentional contextual keywords). It also prints the
// language commits since the plugin's last update. Returns whether the file
// changed.
func Run(t Target, src string, write bool) (string, bool, error) {
	lang := Extract()
	fmt.Printf("== %s (%s) ==\n", t.Name, t.File)

	changed := false
	for _, name := range t.Arrays {
		want, _ := lang.authFor(name)
		have, ok := ParseArray(src, name)
		if !ok {
			fmt.Printf("  %-10s array not found, skipping\n", name+":")
			continue
		}
		// The builtins set is advisory: type constructors and a few niche
		// functions are intentionally curated in the plugins, so report drift
		// but never rewrite it. Keywords/atoms/constants are authoritative.
		advisory := name == "builtins"

		added, removed := Diff(want, have)
		if len(added) == 0 && len(removed) == 0 {
			fmt.Printf("  %-10s up to date (%d)\n", name+":", len(have))
			continue
		}
		if len(added) > 0 {
			tag := "+"
			if advisory {
				tag = "+ (advisory)"
			}
			fmt.Printf("  %-10s %s %s\n", name+":", tag, strings.Join(added, " "))
		}
		if len(removed) > 0 {
			fmt.Printf("  %-10s (plugin-only, kept) %s\n", name+":", strings.Join(removed, " "))
		}
		if write && !advisory && len(added) > 0 {
			var ins []string
			src, ins = InsertItems(src, name, added)
			if len(ins) > 0 {
				changed = true
			}
		}
	}

	fmt.Println("  language commits since last plugin update:")
	commits := LangCommitsSince(LastCommit(t.Dir))
	if len(commits) == 0 {
		fmt.Println("    (none)")
	}
	for _, c := range commits {
		fmt.Println("    " + c)
	}
	return src, changed, nil
}

// RunCLI is the shared entry point for the update-*-plugin commands: it parses
// the `-w` flag, runs the sync against t.File and writes the result back when
// `-w` is given and something changed.
func RunCLI(t Target) {
	write := flag.Bool("w", false, "write the additions back to the plugin file")
	flag.Parse()

	src, err := os.ReadFile(t.File)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read:", err)
		os.Exit(1)
	}
	out, changed, err := Run(t, string(src), *write)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sync:", err)
		os.Exit(1)
	}
	switch {
	case *write && changed:
		if err := os.WriteFile(t.File, []byte(out), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "write:", err)
			os.Exit(1)
		}
		fmt.Println("  wrote", t.File)
	case *write:
		fmt.Println("  nothing to write")
	default:
		fmt.Println("  (dry run; pass -w to apply additions)")
	}
}
