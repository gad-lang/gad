package gad

import (
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gad-lang/gad/parser/node"
)

// GlobMatch is one module matched by a glob import pattern (see
// GlobExtImporter).
type GlobMatch struct {
	// Name is the module's full name, as ExtImporter.Name would report it for
	// that file (e.g. its absolute path) — the key the compiler caches it by.
	Name string
	// Rel is the match's path relative to the pattern's static base directory
	// (the part before its first glob segment), with forward slashes. Path
	// filters are matched against it.
	Rel string
}

// GlobExtImporter is an ExtImporter that can expand a glob pattern into the
// modules it matches, so `import("./plugins/*.gad")` and
// `include ("parts/**/*.gad")` load every matching file.
type GlobExtImporter interface {
	ExtImporter
	// Glob expands the pattern given to the preceding Get call. The matches are
	// sorted by path; a pattern that matches nothing yields none (not an error).
	Glob() ([]GlobMatch, error)
}

// IsGlobPattern reports whether an import/include path is a glob pattern: it
// holds a `*`, `?` or `[` meta character.
func IsGlobPattern(s string) bool {
	return strings.ContainsAny(s, "*?[")
}

// SplitGlobPattern splits a slash-separated glob pattern into its static base
// directory (the leading segments without meta characters; "" or "." for none)
// and the glob part matched below it.
func SplitGlobPattern(pattern string) (base, glob string) {
	segs := strings.Split(pattern, "/")
	for i, seg := range segs {
		if IsGlobPattern(seg) {
			return strings.Join(segs[:i], "/"), strings.Join(segs[i:], "/")
		}
	}
	// No meta character: the whole pattern is its own base.
	return pattern, ""
}

// MatchGlob reports whether the slash-separated path rel matches the glob
// pattern. Each segment is matched with path.Match (`*`, `?`, `[…]`), and a
// `**` segment matches zero or more whole segments, so `**/*.gad` matches
// `a.gad` and `x/y/a.gad`.
func MatchGlob(pattern, rel string) bool {
	return matchGlobSegs(strings.Split(pattern, "/"), strings.Split(rel, "/"))
}

func matchGlobSegs(pat, segs []string) bool {
	for len(pat) > 0 {
		if pat[0] == "**" {
			// Collapse repeated `**` and try every split point.
			for len(pat) > 0 && pat[0] == "**" {
				pat = pat[1:]
			}
			if len(pat) == 0 {
				return true
			}
			for i := range segs {
				if matchGlobSegs(pat, segs[i:]) {
					return true
				}
			}
			return false
		}
		if len(segs) == 0 {
			return false
		}
		if ok, _ := path.Match(pat[0], segs[0]); !ok {
			return false
		}
		pat, segs = pat[1:], segs[1:]
	}
	return len(segs) == 0
}

// GlobDepth reports how many path segments a glob part spans, or -1 when it
// holds a `**` segment (any depth). A walker can stop descending past it.
func GlobDepth(glob string) int {
	segs := strings.Split(glob, "/")
	for _, s := range segs {
		if s == "**" {
			return -1
		}
	}
	return len(segs)
}

// PathFilters are the file filters shared by `embed` and the glob forms of
// `include` (their `includes` / `excludes` / `includes_re` / `excludes_re`
// named args) and `import` (the same names prefixed with `@`, since its other
// named args are module params).
type PathFilters struct {
	// Includes / Excludes are glob patterns, matched against the file's base
	// name or its relative path (a `**` segment spans directories).
	Includes, Excludes []string
	// IncludesRe / ExcludesRe are regular expressions matched against the
	// relative path.
	IncludesRe, ExcludesRe []string
}

// IsZero reports whether no filter is set.
func (f *PathFilters) IsZero() bool {
	return len(f.Includes) == 0 && len(f.Excludes) == 0 && len(f.IncludesRe) == 0 && len(f.ExcludesRe) == 0
}

// Match reports whether the relative path rel (slash- or OS-separated) passes
// the filters: it matches no exclude, and — for each non-empty include list —
// at least one of its patterns.
func (f *PathFilters) Match(rel string) bool {
	rel = filepath.ToSlash(rel)
	base := path.Base(rel)
	globMatch := func(pattern string) bool {
		if ok, _ := path.Match(pattern, base); ok {
			return true
		}
		if ok, _ := path.Match(pattern, rel); ok {
			return true
		}
		return strings.Contains(pattern, "**") && MatchGlob(pattern, rel)
	}
	reMatch := func(pattern string) bool {
		ok, _ := regexp.MatchString(pattern, rel)
		return ok
	}
	anyOf := func(patterns []string, m func(string) bool) bool {
		for _, p := range patterns {
			if m(p) {
				return true
			}
		}
		return false
	}

	if anyOf(f.Excludes, globMatch) || anyOf(f.ExcludesRe, reMatch) {
		return false
	}
	if len(f.Includes) > 0 && !anyOf(f.Includes, globMatch) {
		return false
	}
	if len(f.IncludesRe) > 0 && !anyOf(f.IncludesRe, reMatch) {
		return false
	}
	return true
}

// IsTestFile reports whether the base name of rel, without its extension, ends
// in `_test` (`a_test.gad`, `x/b_test.gadx`): a test file, which glob imports
// and includes skip by default (see MatchModule).
func IsTestFile(rel string) bool {
	base := path.Base(filepath.ToSlash(rel))
	return strings.HasSuffix(strings.TrimSuffix(base, path.Ext(base)), "_test")
}

// MatchModule is Match for a module matched by a glob import/include: test
// files (IsTestFile) are skipped by default, and are kept only when an
// include pattern that names `_test` itself matches them
// (`includes=["*_test.gad"]`, `includes_re=["_test"]`) — a generic include such
// as `*.gad` does not bring them back.
func (f *PathFilters) MatchModule(rel string) bool {
	if IsTestFile(rel) {
		rel := filepath.ToSlash(rel)
		named := PathFilters{}
		for _, p := range f.Includes {
			if strings.Contains(p, "_test") {
				named.Includes = append(named.Includes, p)
			}
		}
		for _, p := range f.IncludesRe {
			if strings.Contains(p, "_test") {
				named.IncludesRe = append(named.IncludesRe, p)
			}
		}
		if len(named.Includes) == 0 && len(named.IncludesRe) == 0 {
			return false
		}
		// One of the `_test` patterns must match; the excludes still apply.
		matched := false
		for _, p := range named.Includes {
			if (&PathFilters{Includes: []string{p}}).Match(rel) {
				matched = true
				break
			}
		}
		for _, p := range named.IncludesRe {
			if (&PathFilters{IncludesRe: []string{p}}).Match(rel) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
		return (&PathFilters{Excludes: f.Excludes, ExcludesRe: f.ExcludesRe}).Match(rel)
	}
	return f.Match(rel)
}

// ValidateRe reports the first invalid regular expression among the `_re`
// filters.
func (f *PathFilters) ValidateRe() error {
	for _, list := range [][]string{f.IncludesRe, f.ExcludesRe} {
		for _, p := range list {
			if _, err := regexp.Compile(p); err != nil {
				return err
			}
		}
	}
	return nil
}

// PathFiltersFromArgs splits the glob filter named args — includes, excludes,
// includes_re and excludes_re, each written with prefix (`@` for import, whose
// other named args are module params; "" for include) — out of named. It
// returns the filters and the remaining named args; bad names the first filter
// whose value is not a string/symbol literal or an array of them (the filters
// are read at compile time).
func PathFiltersFromArgs(prefix string, named *node.CallExprNamedArgs) (f PathFilters, rest node.CallExprNamedArgs, bad string) {
	for i, name := range named.Names {
		var key string
		if name.Ident != nil {
			key = name.Ident.Name
		}
		var dst *[]string
		switch key {
		case prefix + "includes":
			dst = &f.Includes
		case prefix + "excludes":
			dst = &f.Excludes
		case prefix + "includes_re":
			dst = &f.IncludesRe
		case prefix + "excludes_re":
			dst = &f.ExcludesRe
		default:
			rest.Names = append(rest.Names, name)
			rest.Values = append(rest.Values, named.Values[i])
			continue
		}
		vals, ok := constStrings(named.Values[i])
		if !ok {
			if bad == "" {
				bad = key
			}
			continue
		}
		*dst = append(*dst, vals...)
	}
	return
}

// constStrings reads a string/symbol literal, or an array of them.
func constStrings(e node.Expr) ([]string, bool) {
	switch t := e.(type) {
	case *node.StrLit:
		return []string{t.Value()}, true
	case *node.RawStrLit:
		return []string{t.Value()}, true
	case *node.SymbolLit:
		return []string{t.Value()}, true
	case *node.ArrayExpr:
		var out []string
		for _, el := range t.Elements {
			v, ok := constStrings(el)
			if !ok || len(v) != 1 {
				return nil, false
			}
			out = append(out, v[0])
		}
		return out, true
	}
	return nil, false
}
