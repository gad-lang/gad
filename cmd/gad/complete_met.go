package main

import (
	"math"
	"strings"

	gad "github.com/gad-lang/gad"
	"github.com/gad-lang/gad/langsym"
	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
)

// metReceiverCompletions handles member completion for the special receivers of a
// `met` declaration on a class or mixin:
//
//	met MyClass(new, …) { new. }              // constructor: new is the initiator
//	met ~MyClass($old, new) { new. / $old. }  // override constructor
//	met MyClass.method(this, …) { this. }     // method: this is the instance
//	met ~MyClass.method($old, this) { this. } // override method
//
// `this`/`$old`/`new` are injected parameters with no run-time value in the
// prefix, so they cannot be evaluated like an ordinary receiver. Instead the met
// target (the class or mixin) is resolved and its members are listed — including
// members pulled in from `use`d mixins, inherited from parent classes/mixins, and
// required by a mixin's `this { … }` interface. ok is false when the caret is not
// such a context (so the caller falls back to normal member/scope completion).
func metReceiverCompletions(name, src string, offset int) (items []langsym.Symbol, ok bool) {
	root, recvStart, isMet := metReceiverContext(src, offset)
	if !isMet {
		return nil, false
	}

	classExpr, ok := metTargetClassExpr(src, recvStart, root)
	if !ok || classExpr == "" {
		return nil, false
	}

	val, ok := evalReceiver(name, src, recvStart, recvStart, classExpr)
	if !ok {
		return nil, true // it IS a met context; we just could not resolve the target
	}

	for _, m := range metMembers(val) {
		items = append(items, langsym.Symbol{Label: m.Name, Kind: m.Kind, Doc: m.Doc})
	}
	return items, true
}

// metReceiverContext reports whether the caret is at `this.`/`$old.`/`new.` and
// returns the special receiver name and the offset where it begins.
func metReceiverContext(src string, caret int) (root string, recvStart int, ok bool) {
	if caret < 0 || caret > len(src) {
		return "", 0, false
	}
	i := caret
	for i > 0 && isIdentChar(src[i-1]) { // skip the partial member typed after the dot
		i--
	}
	if i == 0 || src[i-1] != '.' {
		return "", 0, false
	}
	dot := i - 1
	// The receiver identifier immediately left of the dot.
	j := dot
	for j > 0 && isIdentChar(src[j-1]) {
		j--
	}
	ident := src[j:dot]
	if j > 0 && src[j-1] == '$' { // `$old`
		ident = "$" + ident
		j--
	}
	switch ident {
	case "this", "new", "$old":
		return ident, j, true
	}
	return "", 0, false
}

// metTargetClassExpr finds the `met` header enclosing the caret (the nearest
// `met` keyword before recvStart) and returns the source of its target CLASS
// expression. For a method form (`met Class.method(this, …)`) that is the
// selector receiver; for a constructor form (`met Class(new, …)`) it is the whole
// target. The choice is driven by which special receiver is being completed.
func metTargetClassExpr(src string, recvStart int, root string) (string, bool) {
	metPos := lastMetKeyword(src, recvStart)
	if metPos < 0 {
		return "", false
	}
	// Header runs from after `met` (and an optional `~`) to the `(` of the params.
	i := metPos + len("met")
	for i < len(src) && (src[i] == ' ' || src[i] == '\t') {
		i++
	}
	if i < len(src) && src[i] == '~' { // override form `met ~Class…`
		i++
		for i < len(src) && (src[i] == ' ' || src[i] == '\t') {
			i++
		}
	}
	paren := indexAtDepthZero(src[i:recvStart], '(')
	if paren < 0 {
		return "", false
	}
	target := strings.TrimSpace(src[i : i+paren])
	if target == "" {
		return "", false
	}
	// A constructor receiver (`new`) targets the whole class; a `this` receiver is
	// a method, so the class is the target minus its trailing `.method` / `["m"]`.
	// `$old` may be either — decide from the sibling params.
	methodForm := root == "this" || (root == "$old" && metParamsHave(src, i+paren, "this"))
	if methodForm {
		if base := stripLastSegment(target); base != "" {
			return base, true
		}
	}
	return target, true
}

// lastMetKeyword returns the offset of the `met` keyword nearest before end that
// starts a token (not part of a longer identifier), or -1.
func lastMetKeyword(src string, end int) int {
	for pos := strings.LastIndex(src[:end], "met"); pos >= 0; pos = strings.LastIndex(src[:pos], "met") {
		before := pos == 0 || !isIdentChar(src[pos-1])
		afterIdx := pos + 3
		after := afterIdx >= len(src) || !isIdentChar(src[afterIdx])
		if before && after {
			return pos
		}
	}
	return -1
}

// indexAtDepthZero returns the index of the first occurrence of ch in s that is
// not nested inside () or [] brackets, or -1.
func indexAtDepthZero(s string, ch byte) int {
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(', '[':
			if s[i] == ch && depth == 0 {
				return i
			}
			depth++
		case ')', ']':
			depth--
		default:
			if s[i] == ch && depth == 0 {
				return i
			}
		}
	}
	return -1
}

// stripLastSegment removes a trailing `.member` or `["member"]` from a target
// expression, yielding the receiver class expression. Empty if there is none.
func stripLastSegment(target string) string {
	if strings.HasSuffix(target, "]") {
		if open := strings.LastIndexByte(target, '['); open > 0 {
			return strings.TrimSpace(target[:open])
		}
		return ""
	}
	if dot := strings.LastIndexByte(target, '.'); dot > 0 {
		return strings.TrimSpace(target[:dot])
	}
	return ""
}

// metParamsHave reports whether the parameter list starting at the `(` offset
// contains a parameter with the given name (a whole-word match within the
// balanced `(...)`).
func metParamsHave(src string, paren int, param string) bool {
	if paren >= len(src) || src[paren] != '(' {
		return false
	}
	depth := 0
	end := paren
	for ; end < len(src); end++ {
		if src[end] == '(' {
			depth++
		} else if src[end] == ')' {
			depth--
			if depth == 0 {
				end++
				break
			}
		}
	}
	return wordIn(src[paren:end], param)
}

// wordIn reports whether s contains word as a whole identifier token.
func wordIn(s, word string) bool {
	for idx := strings.Index(s, word); idx >= 0; {
		before := idx == 0 || !isIdentChar(s[idx-1])
		afterIdx := idx + len(word)
		after := afterIdx >= len(s) || !isIdentChar(s[afterIdx])
		if before && after {
			return true
		}
		next := strings.Index(s[idx+1:], word)
		if next < 0 {
			return false
		}
		idx += 1 + next
	}
	return false
}

// metMembers lists the members of a met target — a class or mixin instance's
// fields, properties and methods, walking used mixins (already merged into a
// class), parent classes/mixins, and a mixin's `this { … }` interface.
func metMembers(val gad.Object) []gad.Member {
	seen := map[string]bool{}
	var out []gad.Member
	add := func(name, kind string) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		out = append(out, gad.Member{Name: name, Kind: kind})
	}
	addIface := func(i *gad.Interface) {
		if i == nil {
			return
		}
		for _, f := range i.Fields {
			add(f.Name, "field")
		}
		for _, p := range i.Props {
			add(p.Name, "property")
		}
		for _, m := range i.Methods {
			add(m.Name, "method")
		}
	}
	var walkClass func(c *gad.Class)
	walkClass = func(c *gad.Class) {
		if c == nil {
			return
		}
		for name := range c.Fields() {
			add(name, "field")
		}
		for name := range c.Properties() {
			add(name, "property")
		}
		for name := range c.Methods() {
			add(name, "method")
		}
		for _, p := range c.RawParents() {
			walkClass(p.Type)
		}
	}
	var walkMixin func(m *gad.Mixin)
	walkMixin = func(m *gad.Mixin) {
		if m == nil {
			return
		}
		for name := range m.Fields() {
			add(name, "field")
		}
		for name := range m.Properties() {
			add(name, "property")
		}
		for name := range m.Methods() {
			add(name, "method")
		}
		addIface(m.This())
		for _, p := range m.RawParents() {
			walkMixin(p)
		}
	}
	switch t := val.(type) {
	case *gad.Class:
		walkClass(t)
	case *gad.Mixin:
		walkMixin(t)
	}
	return out
}

// literalReceiverCompletions handles `this.` inside a class or mixin LITERAL that
// is being EDITED — e.g. `mixin M { methods { m() => this.‸ } }`. The type is not
// a runtime value yet (its members and `this { … }` block are being written), so
// `this` is resolved from the current AST: the enclosing class/mixin's own fields,
// properties and methods, its `this { … }` interface (a mixin), and its parents
// (evaluated, since a parent is defined before the literal). ok is false when the
// caret is not `this.` inside such a literal.
func literalReceiverCompletions(name, src string, offset int) (items []langsym.Symbol, ok bool) {
	root, recvStart, isRecv := metReceiverContext(src, offset)
	if !isRecv || root != "this" {
		return nil, false
	}
	// The `this.` at the caret is a dangling selector that fails to parse, so the
	// enclosing class/mixin literal is absent from the tree. Splice a sentinel
	// identifier at the caret (`this.gadCompletionCaret`) so the literal parses,
	// then locate it. Fall back to the raw parse if the splice does not help.
	data := []byte(src)
	var (
		file *parser.File
		cls  *node.ClassExpr
	)
	if patched, ok2 := spliceIdent(data, offset); ok2 {
		if f, s, _ := langsymParse(name, patched); f != nil {
			file, cls = f, enclosingClassLiteral(f, s, recvStart)
		}
	}
	if cls == nil {
		if f, s, _ := langsymParse(name, data); f != nil {
			file, cls = f, enclosingClassLiteral(f, s, recvStart)
		}
	}
	if cls == nil {
		return nil, false
	}

	// byName indexes the file's top-level class/mixin literals so a `*Parent`
	// spread can be resolved to its literal for recursive member collection —
	// robust while the file is mid-edit (no evaluation of a broken literal).
	byName := classLiteralsByName(file)

	seen := map[string]bool{}
	add := func(n, kind string) {
		if n != "" && !seen[n] {
			seen[n] = true
			items = append(items, langsym.Symbol{Label: n, Kind: kind})
		}
	}
	collectLiteralMembers(cls, byName, map[*node.ClassExpr]bool{}, add)
	return items, true
}

// collectLiteralMembers adds the members visible on `this` inside a class/mixin
// literal: its own fields, properties and methods, its `this { … }` interface
// requirements (a mixin), and — recursively — those of each `*Parent` spread it
// can resolve to a same-file literal. seen guards against a cyclic parent graph.
func collectLiteralMembers(cls *node.ClassExpr, byName map[string]*node.ClassExpr, seen map[*node.ClassExpr]bool, add func(string, string)) {
	if cls == nil || seen[cls] {
		return
	}
	seen[cls] = true
	for _, f := range cls.Fields {
		if f.Name != nil && f.Name.Ident != nil {
			add(f.Name.Ident.Name, "field")
		}
	}
	for _, p := range cls.Props {
		if id, _ := p.NameExpr.(*node.IdentExpr); id != nil {
			add(id.Name, "property")
		}
	}
	for _, m := range cls.Methods {
		if id, _ := m.NameExpr.(*node.IdentExpr); id != nil {
			add(id.Name, "method")
		}
	}
	if cls.This != nil {
		for _, mm := range cls.This.Members {
			if mm.Name != nil && mm.Name.Ident != nil {
				add(mm.Name.Ident.Name, "field")
			}
		}
		for _, mm := range cls.This.Methods {
			if mm.NameExpr != nil {
				add(mm.NameExpr.Name, "method")
			}
		}
	}
	for _, p := range cls.Parents {
		if id, _ := p.Type.(*node.IdentExpr); id != nil {
			collectLiteralMembers(byName[id.Name], byName, seen, add)
		}
	}
}

// classLiteralsByName indexes a file's named class/mixin literals by name, so a
// `*Parent` spread can be resolved to its declaration for in-editor completion.
func classLiteralsByName(file *parser.File) map[string]*node.ClassExpr {
	out := map[string]*node.ClassExpr{}
	record := func(cls *node.ClassExpr) {
		if id, _ := cls.NameExpr.(*node.IdentExpr); id != nil {
			out[id.Name] = cls
		}
	}
	for _, stmt := range file.Stmts {
		node.Walk(stmt, func(n ast.Node) bool {
			switch c := n.(type) {
			case *node.ClassExpr:
				record(c)
			case *node.ClassStmt:
				record(&c.ClassExpr)
			}
			return true
		})
	}
	return out
}

// enclosingClassLiteral returns the innermost `class`/`mixin` literal (ClassExpr)
// whose method, property or constructor body contains the caret byte offset, or
// nil. Used to resolve `this.` while the literal is being edited.
func enclosingClassLiteral(file *parser.File, sf *source.File, caret int) *node.ClassExpr {
	pos := source.Pos(sf.Base + caret)
	var best *node.ClassExpr
	bestSpan := math.MaxInt // no literal found yet
	consider := func(cls *node.ClassExpr) {
		inBody := func(n node.Node) bool {
			return n != nil && n.Pos() <= pos && pos <= n.End()
		}
		hit := false
		for _, m := range cls.Methods {
			for _, fm := range m.Methods {
				if inBody(fm) {
					hit = true
				}
			}
		}
		for _, p := range cls.Props {
			for _, fm := range p.Methods {
				if inBody(fm) {
					hit = true
				}
			}
		}
		for _, fm := range cls.New {
			if inBody(fm) {
				hit = true
			}
		}
		if hit {
			if span := int(cls.End() - cls.Pos()); span < bestSpan {
				best, bestSpan = cls, span
			}
		}
	}
	for _, stmt := range file.Stmts {
		node.Walk(stmt, func(n ast.Node) bool {
			switch c := n.(type) {
			case *node.ClassExpr:
				consider(c)
			case *node.ClassStmt:
				consider(&c.ClassExpr)
			}
			return true
		})
	}
	return best
}
