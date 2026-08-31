package gad

import (
	"sort"
	"strconv"
	"strings"
)

// gadTransformFn is the callable registered as `gad.transform` (built in
// buildGadNamespaceFuncs, exposed in GadModule).
var gadTransformFn Object

// buildGadTransformFn constructs `gad.transform(value; ".path" = fn, …)`.
//
// It rewrites a JSON-like value (nested dicts/arrays) bottom-up: every named
// argument maps a yq-style path pattern to a transformer function. The walk is a
// single post-order pass — each node's children are transformed and spliced back
// in before the node's own matcher runs — so a container matcher sees its
// already-transformed children (the whole point of the feature). The transformed
// value is RETURNED (reassign it: `v = gad.transform(v; …)`); a matcher may
// replace a node with a different type (a dict → a class instance), which cannot
// be done in place.
func buildGadTransformFn() {
	gadTransformFn = NewFunction("transform", gadTransform,
		FunctionWithModule(gadModuleSpec),
		FunctionWithParams(func(p func(name string) *ParamBuilder) { p("value") }),
		// The path→fn matchers arrive as arbitrary named args; a var named-param
		// lets any name through (they stay unread, read via UnreadPairs below).
		FunctionWithNamedParams(func(np func(name string) *NamedParamBuilder) { np("paths").Var() }),
		FunctionWithReturnVars(func(ret func(name string, typ ...TypeAssigner)) { ret("_", TAny) }),
	)
}

// gadTransform is the `gad.transform` handler. See buildGadTransformFn.
func gadTransform(c Call) (Object, error) {
	if c.VM == nil {
		return nil, ErrNotCallable.NewError("gad.transform requires a VM to invoke its matchers")
	}
	value := c.Args.Get(0)

	// Compile a matcher per named arg, in declaration order, then sort by
	// specificity (most specific first) so the most specific pattern wins at any
	// node regardless of the order the caller listed them.
	var matchers []*transformMatcher
	for _, kv := range c.NamedArgs.UnreadPairs() {
		raw := kv.K.ToString()
		fn, ok := kv.V.(CallerObject)
		if !ok || !Callable(kv.V) {
			return nil, ErrNotCallable.NewError("transform " + strconv.Quote(raw) + " value must be callable")
		}
		segs, err := compilePath(raw)
		if err != nil {
			return nil, err
		}
		matchers = append(matchers, &transformMatcher{
			raw:     raw,
			pattern: segs,
			fn:      fn,
			score:   specificity(segs),
		})
	}
	// Stable so equal-specificity matchers keep declaration order.
	sort.SliceStable(matchers, func(i, j int) bool { return matchers[i].score > matchers[j].score })

	// One invoker per matcher, acquired once and reused across every matched node.
	for _, m := range matchers {
		m.inv = NewInvoker(c.VM, m.fn)
		m.inv.Acquire()
		defer m.inv.Release()
	}

	return applyTransform(value, nil, matchers)
}

// applyTransform is the bottom-up walk: it transforms node's children first
// (splicing each result back into its container), then applies the most-specific
// matcher whose pattern matches this node's path, returning the (possibly
// replaced) node.
func applyTransform(node Object, path []pathStep, matchers []*transformMatcher) (Object, error) {
	switch n := node.(type) {
	case Dict:
		// Sorted keys for a deterministic traversal order (siblings transform
		// independently, but a stable order keeps behaviour reproducible).
		keys := make([]string, 0, len(n))
		for k := range n {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			child, err := applyTransform(n[k], childPath(path, pathStep{key: k}), matchers)
			if err != nil {
				return nil, err
			}
			n[k] = child
		}
	case Array:
		for i := range n {
			child, err := applyTransform(n[i], childPath(path, pathStep{index: true, idx: i}), matchers)
			if err != nil {
				return nil, err
			}
			n[i] = child
		}
	}

	for _, m := range matchers {
		if matchPath(m.pattern, path) {
			// The matcher's typed first param (e.g. `(d dict)`) is enforced by the
			// normal call validation — a type mismatch is an error, not a skip.
			return m.inv.Invoke(Args{Array{node}}, nil)
		}
	}
	return node, nil
}

// transformMatcher is one compiled path→fn rule.
type transformMatcher struct {
	raw     string    // the original path string, for error messages
	pattern []pathSeg // compiled segments
	fn      CallerObject
	score   int      // specificity, higher = more specific
	inv     *Invoker // acquired for the duration of one transform call
}

// childPath returns a fresh copy of path with step appended, so recursive calls
// never alias a shared backing array between siblings.
func childPath(path []pathStep, step pathStep) []pathStep {
	out := make([]pathStep, len(path)+1)
	copy(out, path)
	out[len(path)] = step
	return out
}

// pathStep is one concrete descent taken during the walk: into a dict by key, or
// into an array by index.
type pathStep struct {
	index bool
	key   string
	idx   int
}

// pathSegKind classifies one segment of a compiled transform path.
type pathSegKind uint8

const (
	segKey      pathSegKind = iota // literal dict key
	segAnyKey                      // `*`  — any dict key (non-array child)
	segAnyChild                    // `**` — any child (dict key or array index)
	segAnyIndex                    // `[]` — any array index
	segIndex                       // `[N]` — a specific array index
)

// pathSeg is one segment of a compiled path pattern.
type pathSeg struct {
	kind pathSegKind
	key  string // segKey
	idx  int    // segIndex
}

// compilePath parses a yq-style path into segments. The path must start with
// `.`; `.` alone is the root (no segments). Segment forms:
//
//	.key            literal dict key (bareword)
//	."key"/.'key'   quoted literal key (spaces/special chars, backslash-escapable)
//	.*              any dict key (non-array child)
//	.**             any child (dict key or array index)
//	.[] / .key[]    any array index
//	.[N] / .key[N]  a specific array index
func compilePath(s string) ([]pathSeg, error) {
	if s == "" || s[0] != '.' {
		return nil, ErrType.NewError("transform path must start with '.': " + strconv.Quote(s))
	}
	var segs []pathSeg
	i, n := 1, len(s)
	for i < n {
		switch c := s[i]; {
		case c == '.':
			i++ // segment separator; a key selector must follow
			if i >= n {
				return nil, ErrType.NewError("transform path has a trailing '.': " + strconv.Quote(s))
			}
		case c == '[':
			j := strings.IndexByte(s[i:], ']')
			if j < 0 {
				return nil, ErrType.NewError("transform path has an unclosed '[': " + strconv.Quote(s))
			}
			inner := s[i+1 : i+j]
			if inner == "" {
				segs = append(segs, pathSeg{kind: segAnyIndex})
			} else {
				idx, err := strconv.Atoi(inner)
				if err != nil || idx < 0 {
					return nil, ErrType.NewError("transform path has an invalid array index " +
						strconv.Quote(inner) + " in " + strconv.Quote(s))
				}
				segs = append(segs, pathSeg{kind: segIndex, idx: idx})
			}
			i += j + 1
		case c == '*':
			if i+1 < n && s[i+1] == '*' {
				segs = append(segs, pathSeg{kind: segAnyChild})
				i += 2
			} else {
				segs = append(segs, pathSeg{kind: segAnyKey})
				i++
			}
		case c == '"' || c == '\'':
			key, ni, err := readQuotedKey(s, i)
			if err != nil {
				return nil, err
			}
			segs = append(segs, pathSeg{kind: segKey, key: key})
			i = ni
		default:
			// bareword key: read until the next separator/subscript.
			j := i
			for j < n && s[j] != '.' && s[j] != '[' {
				j++
			}
			segs = append(segs, pathSeg{kind: segKey, key: s[i:j]})
			i = j
		}
	}
	return segs, nil
}

// readQuotedKey reads a `"…"` / `'…'` key starting at s[i] (the quote), returning
// the unescaped key and the index just past the closing quote. `\` escapes the
// next byte.
func readQuotedKey(s string, i int) (string, int, error) {
	q := s[i]
	i++
	var b strings.Builder
	for i < len(s) {
		c := s[i]
		switch {
		case c == '\\' && i+1 < len(s):
			b.WriteByte(s[i+1])
			i += 2
		case c == q:
			return b.String(), i + 1, nil
		default:
			b.WriteByte(c)
			i++
		}
	}
	return "", 0, ErrType.NewError("transform path has an unclosed quote: " + strconv.Quote(s))
}

// matchPath reports whether a compiled pattern matches a concrete walk path.
func matchPath(pat []pathSeg, path []pathStep) bool {
	if len(pat) != len(path) {
		return false
	}
	for i, seg := range pat {
		st := path[i]
		switch seg.kind {
		case segKey:
			if st.index || st.key != seg.key {
				return false
			}
		case segAnyKey:
			if st.index {
				return false
			}
		case segAnyChild:
			// matches any child
		case segAnyIndex:
			if !st.index {
				return false
			}
		case segIndex:
			if !st.index || st.idx != seg.idx {
				return false
			}
		}
	}
	return true
}

// specificity scores a pattern so the most specific wins at a node: literal keys
// and fixed indices outweigh kind-wildcards (`*`, `[]`), which outweigh the
// any-child wildcard (`**`); deeper paths edge out shallower ones.
func specificity(segs []pathSeg) int {
	score := len(segs)
	for _, s := range segs {
		switch s.kind {
		case segKey, segIndex:
			score += 100
		case segAnyKey, segAnyIndex:
			score += 10
		case segAnyChild:
			score++
		}
	}
	return score
}
