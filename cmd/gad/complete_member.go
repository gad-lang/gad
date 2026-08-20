package main

import (
	"context"
	"strings"
	"time"

	gad "github.com/gad-lang/gad"
	"github.com/gad-lang/gad/langsym"
)

// memberCompletions handles `x.` / `x.partial` / `x[` completion by runtime
// introspection: it isolates the receiver expression left of the caret, evaluates
// the source up to it (in a sandbox with a timeout and panic recovery), and lists
// the resulting value's members — dict keys, class fields/properties/methods,
// module exports. ok is false when the caret is not a member-access context.
func memberCompletions(src string, offset int) (items []langsym.Symbol, ok bool) {
	recv, recvStart, dot, ok := memberContext(src, offset)
	if !ok {
		return nil, false
	}

	val, ok := evalReceiver(src, recvStart, dot, recv)
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
func evalReceiver(src string, recvStart, dot int, recv string) (val gad.Object, ok bool) {
	defer func() {
		if recover() != nil {
			val, ok = nil, false
		}
	}()

	lineStart := strings.LastIndexByte(src[:recvStart], '\n') + 1
	prelude := src[:lineStart] + "return " + recv + "\n"

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
