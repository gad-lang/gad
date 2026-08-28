package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	gad "github.com/gad-lang/gad"
	"github.com/gad-lang/gad/langsym"
	"github.com/gad-lang/gad/parser/node"
)

// memberCompletions handles `x.` / `x.partial` / `x[` completion by runtime
// introspection: it isolates the receiver expression left of the caret, evaluates
// the source up to it (in a sandbox with a timeout and panic recovery), and lists
// the resulting value's members — dict keys, class fields/properties/methods,
// module exports. ok is false when the caret is not a member-access context.
func memberCompletions(name, src string, offset int) (items []langsym.Symbol, ok bool) {
	// A `met` special receiver (`this`/`$old`/`new`) is resolved from the met
	// target's type, not by evaluating the (unbound) parameter.
	if items, ok := metReceiverCompletions(name, src, offset); ok {
		return items, true
	}

	recv, recvStart, dot, ok := memberContext(src, offset)
	if !ok {
		return nil, false
	}

	val, ok := evalReceiver(name, src, recvStart, dot, recv)
	if !ok {
		return nil, true // it IS a member context; we just could not evaluate it
	}

	members := gad.Members(val)
	// Attach class member docs from the source. Parse only the complete lines
	// before the caret (the caret line is mid-edit and would not parse), which
	// still contain the class declaration.
	clean := src[:strings.LastIndexByte(src[:recvStart], '\n')+1]
	docs := classMemberDocs(clean, val)
	for _, m := range members {
		doc := m.Doc
		if doc == "" {
			doc = docs[m.Name]
		}
		items = append(items, langsym.Symbol{Label: m.Name, Kind: m.Kind, Doc: doc})
	}
	return items, true
}

// memberContext finds a member access at the caret. It returns the receiver
// expression text, the offset where it starts, and the offset of the `.` (or `[`)
// that begins the access.
func memberContext(src string, caret int) (recv string, recvStart, dot int, ok bool) {
	if caret < 0 || caret > len(src) {
		return "", 0, 0, false
	}
	i := caret
	// Skip the partial member name already typed after the dot.
	for i > 0 && isIdentChar(src[i-1]) {
		i--
	}
	if i == 0 || (src[i-1] != '.' && src[i-1] != '[') {
		return "", 0, 0, false
	}
	dot = i - 1
	recvStart = scanReceiverBack(src, dot)
	recv = strings.TrimSpace(src[recvStart:dot])
	if recv == "" {
		return "", 0, 0, false
	}
	return recv, recvStart, dot, true
}

// scanReceiverBack returns the start offset of the receiver expression ending at
// end (exclusive): a chain of identifiers, dots and balanced () / [] groups.
func scanReceiverBack(src string, end int) int {
	i := end
	for i > 0 {
		c := src[i-1]
		switch {
		case isIdentChar(c) || c == '.':
			i--
		case c == ')' || c == ']':
			i = matchOpen(src, i-1)
		default:
			return i
		}
	}
	return i
}

// matchOpen returns the index of the bracket matching the closer at close,
// scanning backwards; on imbalance it returns close (giving up).
func matchOpen(src string, close int) int {
	open, shut := byte('('), byte(')')
	if src[close] == ']' {
		open, shut = '[', ']'
	}
	depth := 0
	for i := close; i >= 0; i-- {
		switch src[i] {
		case shut:
			depth++
		case open:
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return close
}

// evalReceiver evaluates the source preceding the receiver plus `return RECEIVER`
// and returns the resulting value. It runs in a goroutine-safe Eval with a short
// timeout and recovers from any panic, so a malformed or side-effecting prefix
// can never hang or crash the command.
func evalReceiver(name, src string, recvStart, dot int, recv string) (val gad.Object, ok bool) {
	defer func() {
		if recover() != nil {
			val, ok = nil, false
		}
	}()

	prelude, ok := receiverPrelude(name, src, recvStart, recv)
	if !ok {
		return nil, false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Enable module resolution so `import("time").` / local-module members can be
	// introspected, mirroring how the CLI runs a script.
	opts := gad.CompileOptions{
		CompilerOptions: gad.CompilerOptions{ModuleMap: DefaultModuleMap(".", &sourcePath)},
	}
	e := gad.NewEval(nil, nil, opts)
	ret, _, err := e.RunScript(ctx, []byte(prelude))
	if err != nil || ret == nil {
		return nil, false
	}
	return ret, true
}

// receiverPrelude builds a runnable plain-Gad program whose result is the value
// of recv, evaluated in the scope it has at the caret. The strategy depends on
// the dialect (chosen by name's extension):
//
//   - plain .gad: replace the caret line in place with `return recv`, keeping the
//     untouched tail so all enclosing blocks stay balanced — a receiver inside a
//     `for { … }` / `if { … }` block resolves because `return` fires with the
//     block variables bound.
//   - .gadt (mixed template): the code lives in `{% … %}` islands interleaved with
//     literal text, which RunScript cannot execute directly. Extract the code
//     islands before the caret (skipping `{%= … %}` output islands, which do not
//     bind names), append `return recv`, and close any block still open at the
//     caret (a `for … begin` whose `end` is in a later island).
func receiverPrelude(name, src string, recvStart int, recv string) (string, bool) {
	// Template dialects hold their code in constructs (`{% … %}` islands in
	// `.gadt`, indentation blocks in `.gadx`) that RunScript cannot execute as-is,
	// so rebuild the caret's scope as runnable Gad from the parsed AST.
	if strings.HasSuffix(name, ".gadt") || strings.HasSuffix(name, ".gadx") {
		return astReceiverPrelude(name, src, recvStart, recv)
	}
	lineStart := strings.LastIndexByte(src[:recvStart], '\n') + 1
	lineEnd := lineStart + strings.IndexByte(src[lineStart:], '\n')
	if lineEnd < lineStart {
		lineEnd = len(src)
	}
	return src[:lineStart] + "return " + recv + "\n" + src[lineEnd:], true
}

// astReceiverPrelude assembles a runnable plain-Gad program from a template
// (`.gadt` mixed source or `.gadx` indentation source) so recv's value can be
// introspected. It parses the source through the dialect front-end (langsymParse,
// which lowers `.gadx` to Gad and parses `.gadt` in mixed mode, both preserving
// positions), then rebuilds the scope at the caret as real Gad: every enclosing
// declaration before the caret, plus the header of each `for … in …` that
// contains the caret (as `for k, v in it {` opening a block). `return recv` then
// runs on the first loop iteration with the loop variables bound, and the opened
// blocks are closed. Working from the AST (not `{%`/`%}` string matching) means
// island strings or a leading doc comment whose prose contains `{% … %}` cannot
// corrupt the extraction.
func astReceiverPrelude(name, src string, caret int, recv string) (string, bool) {
	file, sf, _ := langsymParse(name, []byte(src))
	if file == nil {
		return "", false
	}
	base := int(sf.Base)

	var b strings.Builder
	opens := 0
	pos := func(n node.Node) int { return int(n.Pos()) - base }
	end := func(n node.Node) int { return int(n.End()) - base }
	// identName returns a loop variable's name, or "_" when absent/blank.
	identName := func(id *node.IdentExpr) string {
		if id == nil || id.Empty || id.Name == "" {
			return "_"
		}
		return id.Name
	}
	// iterableSrc returns runnable source for a `for … in` iterable via the AST's
	// own String() rendering. This is dialect-independent and handles complex
	// iterables (`items.filter(f)`), unlike slicing the source by position — the
	// `.gadx` front-end lowers to synthetic nodes whose positions do not slice
	// back to clean source.
	iterableSrc := func(e node.Expr) string {
		if s, ok := e.(fmt.Stringer); ok {
			return strings.TrimSpace(s.String())
		}
		return ""
	}

	var walk func(stmts []node.Stmt)
	walk = func(stmts []node.Stmt) {
		for _, s := range stmts {
			sp, se := pos(s), end(s)
			if se <= caret {
				// A declaration fully before the caret binds names in scope.
				switch s.(type) {
				case *node.DeclStmt, *node.AssignStmt:
					if sp >= 0 && se <= len(src) {
						b.WriteString(src[sp:se])
						b.WriteByte('\n')
					}
				}
				continue
			}
			if sp > caret {
				continue // starts after the caret — not in scope yet
			}
			// This statement contains the caret; descend, opening any binding block.
			switch st := s.(type) {
			case *node.ForInStmt:
				it := iterableSrc(st.Iterable)
				if it == "" {
					return // iterable not yet typed; cannot evaluate
				}
				if st.Value != nil && !st.Value.Empty {
					b.WriteString("for " + identName(st.Key) + ", " + identName(st.Value) + " in " + it + " {\n")
				} else {
					b.WriteString("for " + identName(st.Key) + " in " + it + " {\n")
				}
				opens++
				if st.Body != nil {
					walk(st.Body.Stmts)
				}
			case *node.IfStmt:
				if st.Body != nil {
					walk(st.Body.Stmts)
				}
				if b2, ok := st.Else.(*node.BlockStmt); ok {
					walk(b2.Stmts)
				}
			case *node.BlockStmt:
				walk(st.Stmts)
			}
		}
	}
	walk(file.Stmts)

	b.WriteString("return ")
	b.WriteString(recv)
	b.WriteByte('\n')
	for ; opens > 0; opens-- {
		b.WriteString("}\n")
	}
	return b.String(), true
}

// classMemberDocs returns the source doc comments for the members of val, when
// val is a class or class instance whose class is declared in src.
func classMemberDocs(src string, val gad.Object) map[string]string {
	var name string
	switch v := val.(type) {
	case *gad.ClassInstance:
		if c, ok := v.Type().(*gad.Class); ok {
			name = c.Name()
		}
	case *gad.Class:
		name = v.Name()
	default:
		return nil
	}
	return langsym.ClassMemberDocs([]byte(src), name)
}

func isIdentChar(c byte) bool {
	return c == '_' ||
		(c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9')
}
