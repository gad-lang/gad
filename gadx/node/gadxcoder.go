package node

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	gnode "github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/token"
)

// =============================================================================
// GadxCoder — regenerate formatted gadx source (like gofmt)
// =============================================================================

// GadxCodeWriteContext holds state for writing gadx template source.
type GadxCodeWriteContext struct {
	Writer io.Writer
	Depth  int
	Prefix string // indentation string per depth level (default "\t")
	// EmbedFlags are the GAD formatting flags applied to embedded GAD code
	// (attribute expressions, interpolations, conditions, `~` code, declaration
	// values), so embedded code is formatted like `gad fmt`. Defaults to the
	// column-aware NEW_LINE_CALC mode.
	EmbedFlags gnode.CodeWriteContextFlag
	// MaxColumns is the line-width budget for the embedded GAD (0 uses the GAD
	// formatter's default).
	MaxColumns int
	// raw is set while emitting a literal-text block body (@text / @p / @md):
	// its text lines are written verbatim (no `| ` prefix) and blank lines are
	// preserved.
	raw bool
	// preserve is set while emitting the body of a `<pre>` / `<textarea>`,
	// whose whitespace is content: a text run there is written as one quoted
	// literal, the only form that carries it back through the parse.
	preserve bool
}

// NewGadxCodeContext creates a new context writing to w, with 1-tab indentation
// and the column-aware GAD formatting rules for embedded code.
func NewGadxCodeContext(w io.Writer) *GadxCodeWriteContext {
	return &GadxCodeWriteContext{
		Writer:     w,
		Prefix:     "\t",
		EmbedFlags: gnode.CodeWriteContextFlagFormatNewLineCalc,
	}
}

// gadxInterp renders a text interpolation (MixedValueStmt) as the canonical
// gadx output form `{[mark] = expr [mark]}`. It always uses the output `=` (a
// MixedValueStmt always outputs — it lowers to a write() call), and preserves
// the trim markers: `-` strips adjacent spaces but keeps a line break, `--`
// strips all adjacent whitespace (newlines included). Example: `{-=v--}` →
// `{- = v --}`.
func (c *GadxCodeWriteContext) gadxInterp(s *gnode.MixedValueStmt) string {
	leftMark, rightMark := "", ""
	switch {
	case s.RemoveLeftAll:
		leftMark = "--"
	case s.RemoveLeftSpace:
		leftMark = "-"
	}
	switch {
	case s.RemoveRightAll:
		rightMark = "--"
	case s.RemoveRightSpace:
		rightMark = "-"
	}
	var b strings.Builder
	b.WriteString("{")
	b.WriteString(leftMark)
	if leftMark != "" {
		b.WriteString(" ")
	}
	b.WriteString("= ")
	b.WriteString(c.gadExpr(s.Expr))
	b.WriteString(" ")
	b.WriteString(rightMark)
	b.WriteString("}")
	return b.String()
}

// gadExpr renders an embedded GAD expression with the configured formatting
// rules. It returns "" for a nil expression.
func (c *GadxCodeWriteContext) gadExpr(e gnode.Expr) string {
	if e == nil {
		return ""
	}
	return c.gadCode(e)
}

// forCond renders a `@for` condition in its canonical gadx form. The gadx
// parser parses the header as a plain expression, so `@for k, v in it` is stored
// as an array `[k, v in it]` (and `@for v in it` as a binary `v in it`); the
// lowering (convertFor) recognizes those shapes. Rendering the stored expression
// verbatim would parenthesize the `in` (`[k, (v in it)]`), which the lowering no
// longer recognizes — breaking semantics. So reconstruct the header explicitly.
func (c *GadxCodeWriteContext) forCond(cond gnode.Expr) string {
	if k, v, iter, ok := forInPair(cond); ok {
		return c.gadExpr(k) + ", " + c.gadExpr(v) + " in " + c.gadExpr(iter)
	}
	if bin, ok := cond.(*gnode.BinaryExpr); ok && bin.Token == token.In {
		if _, ok := bin.LHS.(*gnode.IdentExpr); ok {
			return c.gadExpr(bin.LHS) + " in " + c.gadExpr(bin.RHS)
		}
	}
	return c.gadCond(cond) // C-style `i := 0; i < n; i++` or any other form
}

// forInPair extracts (key, value, iterable) from the `[key, value in iterable]`
// shape the parser produces for `@for key, value in iterable` — an ArrayExpr or
// MultiParenExpr of two elements whose second is `value in iterable`.
func forInPair(cond gnode.Expr) (key, val, iter gnode.Expr, ok bool) {
	var elems []gnode.Expr
	switch c := cond.(type) {
	case *gnode.ArrayExpr:
		elems = c.Elements
	case *gnode.MultiParenExpr:
		elems = c.PositionalElements
	default:
		return nil, nil, nil, false
	}
	if len(elems) != 2 {
		return nil, nil, nil, false
	}
	if _, ok := elems[0].(*gnode.IdentExpr); !ok {
		return nil, nil, nil, false
	}
	bin := unwrapParen(elems[1])
	b, isBin := bin.(*gnode.BinaryExpr)
	if !isBin || b.Token != token.In {
		return nil, nil, nil, false
	}
	if _, ok := b.LHS.(*gnode.IdentExpr); !ok {
		return nil, nil, nil, false
	}
	return elems[0], b.LHS, b.RHS, true
}

// unwrapParen strips a single enclosing ParenExpr.
func unwrapParen(e gnode.Expr) gnode.Expr {
	if p, ok := e.(*gnode.ParenExpr); ok && p.Expr != nil {
		return p.Expr
	}
	return e
}

// gadCond renders a directive condition (`@if` / `@for` / `@match` / `@case`).
// It unwraps a single enclosing ParenExpr first: the formatter renders a
// condition parenthesised (e.g. a unary `!x` as `(!x)`), so re-parsing the
// emitted `(cond)` yields a ParenExpr that would otherwise gain another paren
// layer on every format pass.
func (c *GadxCodeWriteContext) gadCond(e gnode.Expr) string {
	if p, ok := e.(*gnode.ParenExpr); ok && p.Expr != nil {
		e = p.Expr
	}
	return c.gadExpr(e)
}

// gadCode renders any GAD coder (expression or statement) with the context's
// formatting flags, matching `gad fmt` but always on a single line. Gadx is
// line-oriented (`~` code, `{= … }`, attribute values, `@if` conditions), so
// embedded GAD must never wrap: column-aware wrapping would split it across
// lines that the gadx grammar cannot represent, and with no indentation prefix
// the wrapped separators collapse (e.g. `f(a, raw "b")` → `f(araw "b")`). An
// effectively unbounded column budget keeps it inline. A trailing newline is
// trimmed so the result can be embedded.
func (c *GadxCodeWriteContext) gadCode(n gnode.Coder) string {
	opts := []gnode.CodeOption{
		gnode.CodeWithFlags(c.EmbedFlags),
		gnode.CodeWithMaxColumns(1 << 30),
	}
	return strings.TrimRight(gnode.Code(n, opts...), "\n")
}

func (c *GadxCodeWriteContext) indent() string {
	return strings.Repeat(c.Prefix, c.Depth)
}

func (c *GadxCodeWriteContext) write(s string) {
	io.WriteString(c.Writer, s)
}

// WriteLine writes an indented line followed by newline.
func (c *GadxCodeWriteContext) WriteLine(s string) {
	c.write(c.indent() + s + "\n")
}

// WriteStmts writes a list of gadx statements at the current depth.
func (c *GadxCodeWriteContext) WriteStmts(stmts gnode.Stmts) {
	// Each statement is written once, into a string, so that a run of siblings
	// that come out the same can be folded into one `tag(N)` — comparing what
	// they *write* is what makes two subtrees the same, not walking them.
	forms := make([]string, len(stmts))
	for i, stmt := range stmts {
		forms[i] = c.render(stmt)
	}

	for i := 0; i < len(stmts); {
		n := c.repeatRun(stmts, forms, i)
		if n > 1 {
			folded := *stmts[i].(*TagStmt)
			folded.Repeat = &gnode.IntLit{
				Value: int64(n), ValuePos: folded.NodePos,
				Literal: strconv.Itoa(n),
			}
			c.write(c.render(&folded))
			i += n
			continue
		}
		c.write(forms[i])
		i++
	}
}

// render writes one statement to a string, at the current depth, so it can be
// compared with its siblings before going out.
func (c *GadxCodeWriteContext) render(stmt gnode.Stmt) string {
	gc, ok := stmt.(GadxCoder)
	if !ok {
		return ""
	}
	var b strings.Builder
	sub := *c
	sub.Writer = &b
	gc.WriteGadx(&sub)
	return b.String()
}

// repeatRun returns how many statements from i on are the same tag written
// again, which `tag(N)` says in one line.
//
// The run is capped at what the lowering writes out as copies: past that a
// `(N)` becomes a loop, and folding would then change the template rather than
// only how it is written. A tag that already carries its own `(N)` is left
// alone, and so is anything that is not a tag — two identical `| ` lines are
// two lines of text, and joining them would change what the page says.
func (c *GadxCodeWriteContext) repeatRun(stmts gnode.Stmts, forms []string, i int) int {
	tag, ok := stmts[i].(*TagStmt)
	if !ok || tag.Repeat != nil || tag.Fragment || forms[i] == "" {
		return 1
	}
	n := 1
	for i+n < len(stmts) && n < maxUnrolledRepeat {
		next, ok := stmts[i+n].(*TagStmt)
		if !ok || next.Repeat != nil || next.Fragment || forms[i+n] != forms[i] {
			break
		}
		n++
	}
	return n
}

// GadxCoder is implemented by nodes that can write formatted gadx source.
type GadxCoder interface {
	WriteGadx(ctx *GadxCodeWriteContext)
}

// =============================================================================
// WriteGadx implementations
// =============================================================================

func (f *File) WriteGadx(ctx *GadxCodeWriteContext) {
	// Separate top-level declaration directives (`@comp`, `@func`, `@param`,
	// `@test`, …) with a blank line, like Go separates top-level decls. When a
	// directive is preceded by leading `//-`/`//` comments, the blank line goes
	// before the first of those comments so the comment stays attached to the
	// directive it documents.
	blank := blankBefore(f.Stmts)
	for i, stmt := range f.Stmts {
		gc, ok := stmt.(GadxCoder)
		if !ok {
			continue
		}
		if blank[i] {
			ctx.write("\n")
		}
		gc.WriteGadx(ctx)
	}
}

// blankBefore marks the statements that should be preceded by a blank line:
//   - each top-level block directive that is not the first, credited to the
//     start of its leading `//`/`//-` comment run so the blank lands before the
//     comment (which documents the directive), not between them;
//   - a directive (or any statement) immediately following a standalone
//     `/** … **/` block-doc comment, so the doc stays a file/section doc rather
//     than re-attaching to the directive as its own doc (which changes the
//     parse: a blank line is what keeps a leading block doc standalone).
func blankBefore(stmts gnode.Stmts) []bool {
	blank := make([]bool, len(stmts))
	for i, s := range stmts {
		if isBlockDirective(s) {
			start := i
			for start > 0 {
				// A standalone block-doc comment is a section/file doc, not part
				// of the directive's attached `//-` run — stop before it.
				if c, ok := stmts[start-1].(*CommentStmt); !ok || isStandaloneDoc(c) {
					break
				}
				start--
			}
			if start > 0 {
				blank[start] = true
			}
		}
		// Keep a blank line after a standalone block-doc comment.
		if i > 0 {
			if c, ok := stmts[i-1].(*CommentStmt); ok && isStandaloneDoc(c) {
				blank[i] = true
			}
		}
	}
	return blank
}

// isStandaloneDoc reports whether a comment is a `/** … **/` block-doc comment
// (which, when it appears as its own top-level statement, is a file/section doc).
func isStandaloneDoc(c *CommentStmt) bool { return c.Block && c.Doc }

// isBlockDirective reports whether a top-level statement is a declaration
// directive that should be preceded by a blank line when it is not the first.
func isBlockDirective(s gnode.Stmt) bool {
	switch s.(type) {
	case *CompDecl, *FuncDecl, *ParamStmt, *GlobalStmt, *VarStmt, *ConstStmt,
		*EnumStmt, *ExportStmt, *TestDecl, *SlotDecl:
		return true
	}
	return false
}

func (t *TextStmt) WriteGadx(ctx *GadxCodeWriteContext) {
	text := ctx.buildMixed(t.Stmts)

	// Inside a literal-text block (@text / @p / @md) each line is emitted
	// verbatim, with no `| ` prefix, and blank lines are preserved.
	if ctx.raw {
		for _, line := range strings.Split(text, "\n") {
			if line == "" {
				ctx.write("\n")
			} else {
				ctx.WriteLine(line)
			}
		}
		return
	}

	// Inside a `<pre>` the whitespace is content, down to the line breaks, and
	// no line-based form carries it: a `| ` line strips its own edges and the
	// break between two of them is not the same as a break in the text. One
	// quoted literal is, so the run goes out whole.
	if ctx.preserve && preserveNeeded(t.Stmts) {
		ctx.WriteLine(preservedRun(ctx, t.Stmts))
		return
	}

	// A run that renders to a single space is the space between two elements,
	// and it has a marker of its own: a lone `*`. It reaches here in either
	// form — literal text, from HTML, or the `{= " " }` this writer used to
	// emit — so the test is on what it renders, not on how it is written.
	if plain, ok := runPlainText(t.Stmts); ok && plain == " " {
		ctx.WriteLine("*")
		return
	}

	// Reconstruct the mixed run (literal text interleaved with interpolations)
	// as one string, then emit each source line as its own `| ` line. Keeping a
	// run like `x = {= v } (y)` on a single line preserves the spaces around the
	// interpolations — a bare `| ` line strips only trailing whitespace, which
	// would otherwise be lost when the segments are split across lines.
	if strings.TrimSpace(text) == "" {
		if text == "" {
			return
		}
		// A run of pure whitespace still renders — it is the space between two
		// inline elements — so dropping it changes the page. A `| ` line cannot
		// carry it (parsing strips the edges). A single space has its own
		// marker; anything else goes out as an interpolated literal, which
		// lowers to the same text node.
		if text == " " {
			ctx.WriteLine("*")
			return
		}
		ctx.WriteLine("{= " + strconv.Quote(text) + " }")
		return
	}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// A `| ` line strips its edges on parse, so whitespace that matters is
		// written as an interpolated literal instead. It matters at the very
		// edges of the run, where it is the space between this text and the
		// element beside it; between lines the HTML rule collapses it anyway.
		keepLead := i == 0 && line != strings.TrimLeft(line, " \t")
		keepTail := i == len(lines)-1 && line != strings.TrimRight(line, " \t")
		if keepLead || keepTail {
			// The whole line goes out as one interpolated literal, edges and
			// all. Writing only the edges that way would split one text node
			// into three: the page reads the same, but the lowered code does
			// not, and that is what the formatter is checked against.
			edged := line
			if !keepLead {
				edged = strings.TrimLeft(edged, " \t")
			}
			if !keepTail {
				edged = strings.TrimRight(edged, " \t")
			}
			ctx.WriteLine("{= " + strconv.Quote(edged) + " }")
			continue
		}
		ctx.WriteLine(ctx.textLine(t.Stmts, trimmed))
	}
}

// textLine returns the source line for a run's text. A run that is exactly one
// interpolation is written bare — a line opening with `{=` reads back as that
// value — which is what keeps a spacer line down to `{= " " }`. Everything else
// takes the `| ` prefix, without which it would be read as a directive.
func (c *GadxCodeWriteContext) textLine(stmts gnode.Stmts, trimmed string) string {
	if len(stmts) == 1 {
		if _, ok := stmts[0].(*gnode.MixedValueStmt); ok && strings.HasPrefix(trimmed, "{=") {
			return trimmed
		}
	}
	return "| " + trimmed
}

// buildMixed reconstructs a run of mixed statements (literal text interleaved
// with `{= expr }` interpolations) into a single string.
func (c *GadxCodeWriteContext) buildMixed(stmts gnode.Stmts) string {
	var b strings.Builder
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *gnode.MixedTextStmt:
			b.WriteString(s.Lit.Value)
		case *gnode.MixedValueStmt:
			b.WriteString(c.gadxInterp(s))
		default:
			b.WriteString(s.String())
		}
	}
	return b.String()
}

// writeRawBlock emits a literal-text block directive (`@text` / `@p` / `@md`)
// and its body verbatim (text lines with no `| ` prefix, blank lines preserved).
// Non-text nodes in the body (e.g. `@if` inside `@md`) render normally.
func (c *GadxCodeWriteContext) writeRawBlock(directive string, body gnode.Stmts) {
	c.WriteLine(directive)
	c.Depth++
	prev := c.raw
	c.raw = true
	c.WriteStmts(body)
	c.raw = prev
	c.Depth--
}

func (t *TextBlockStmt) WriteGadx(ctx *GadxCodeWriteContext) {
	// A YAML-style block emits `|` (literal) or `|>` (folded); the `@text`
	// directive emits `@text`.
	directive := "@text"
	if t.Pipe {
		directive = "|"
		if t.Fold {
			directive = "|>"
		}
	}
	ctx.writeRawBlock(directive, t.Body)
}

// WriteGadx writes a `@raw_text` block with its body verbatim, one line per
// line. The body is content, so it is not reflowed; the block's indentation is
// re-applied at the current depth, which is what the parser stripped.
func (t *RawTextBlockStmt) WriteGadx(ctx *GadxCodeWriteContext) {
	ctx.WriteLine("@raw_text")
	ctx.Depth++
	for _, line := range t.Lines {
		if line == "" {
			ctx.write("\n")
			continue
		}
		ctx.WriteLine(line)
	}
	ctx.Depth--
}

func (t *ParaBlockStmt) WriteGadx(ctx *GadxCodeWriteContext) { ctx.writeRawBlock("@p", t.Body) }
func (t *MdBlockStmt) WriteGadx(ctx *GadxCodeWriteContext)   { ctx.writeRawBlock("@md", t.Body) }

func (t *TagStmt) WriteGadx(ctx *GadxCodeWriteContext) {
	if t.Fragment {
		// A `<>…</>` fragment has no wrapper element and pug-style gadx has no
		// fragment token, so it transpiles back as its children at this level.
		ctx.WriteStmts(t.Body)
		return
	}
	// Attributes merge into a single `[a=v, b=x]` group (canonical gadx); spread
	// and conditional attributes keep their own group (their `? cond` is
	// group-scoped). The result is written inline on the tag line.
	shorthand, rest := shorthandAttrs(t.Attributes)
	groups := ctx.attrGroups(rest)
	inline := t.Name + shorthand
	for _, g := range groups {
		inline += g.inline()
	}
	// `(N)` closes the head: after everything that describes the tag, before
	// the text that is inside it.
	if t.Repeat != nil {
		inline += "(" + ctx.gadExpr(t.Repeat) + ")"
	}

	if IsRawText(t.Name) && len(t.Body) > 0 && !rawTextBlockSafe(ctx.rawTextContent(t.Body)) {
		// The body cannot be written under the tag and read back unchanged, so
		// it stays the HTML region it came from. A body opening with a blank
		// line has no indentation for the block to be read from, and one
		// ending in blank space would have that space stripped as the line's
		// own trailing whitespace.
		ctx.writeRawTextRegion(t)
		return
	}

	if IsRawText(t.Name) && len(t.Body) > 0 {
		// A script or a stylesheet holds text, not markup, so its body is
		// written the way `@raw_text` writes one: verbatim under the tag,
		// carrying only the block's own indentation, which the parser strips
		// again. That keeps the content byte for byte — the language depends
		// on it — while the tag itself reads like every other tag.
		ctx.writeRawTextTag(t, inline)
		return
	}

	if !ctx.overflows(inline) {
		// A tag whose whole body is a single short text run is written inline as
		// `tag text` (so `<span>one</span>` → `span one`, not `span` + `| one`).
		// Inside a `<pre>` the whitespace is content for every element under it,
		// not only the `<pre>` itself: a `<code>` nested in one carries the line
		// breaks just the same.
		preserve := ctx.preserve || IsPreserveWhitespace(t.Name)
		if text, ok := ctx.inlineTagText(t.Body, preserve); ok && !ctx.overflows(inline+" "+text) {
			ctx.WriteLine(inline + " " + text)
			return
		}
		ctx.WriteLine(inline)
	} else {
		// Overflow: wrap the merged attribute group one item per line.
		ctx.writeWrappedTag(t.Name, groups)
	}
	ctx.Depth++
	if IsPreserveWhitespace(t.Name) {
		was := ctx.preserve
		ctx.preserve = true
		ctx.WriteStmts(t.Body)
		ctx.preserve = was
	} else {
		ctx.WriteStmts(t.Body)
	}
	ctx.Depth--
}

// attrGroup is either a merged group (comma-joined items) or a raw `[…]` group
// (a spread or conditional attribute that must stay on its own).
type attrGroup struct {
	items  []string
	raw    string
	merged bool
}

func (g attrGroup) inline() string {
	if g.merged {
		return "[" + strings.Join(g.items, ", ") + "]"
	}
	return g.raw
}

// mergeable reports whether the attribute can be combined into a shared bracket
// group: it has no group-scoped condition and is not a spread.
func (a *TagAttribute) mergeable() bool { return a.Condition == nil && a.Spread == nil }

// inner renders the attribute as a bracket-group item (`name`, `name=value`),
// without the surrounding `[ ]`.
func (a *TagAttribute) inner(ctx *GadxCodeWriteContext) string {
	s := a.Name
	if !a.IsFlag && a.Value != nil {
		if IsEmptyAttrValue(a.Value) {
			// `@empty` is how the source says "present, with an empty value".
			// Writing the gadx.EMPTY selector back out would be correct but
			// unreadable, and a bare "" means the opposite: dropped.
			return s + "=@empty"
		}
		s += "=" + ctx.gadExpr(a.Value)
	}
	return s
}

// IsEmptyAttrValue reports whether e is the gadx.EMPTY selector — the value an
// attribute carries when it is present and empty.
func IsEmptyAttrValue(e gnode.Expr) bool {
	sel, ok := e.(*gnode.SelectorExpr)
	if !ok {
		return false
	}
	x, ok := sel.X.(*gnode.IdentExpr)
	if !ok || x.Name != "gadx" {
		return false
	}
	name, ok := sel.Sel.(*gnode.StrLit)
	return ok && name.Value() == "EMPTY"
}

// attrGroups builds the tag's attribute groups: consecutive mergeable
// attributes fold into one group (order preserved), each spread/conditional
// attribute is its own group.
// shorthandAttrs peels the leading `id` and `class` attributes off a tag and
// returns them in their shorthand form — `#title.card.shadow` — along with the
// attributes that are left.
//
// Only a literal, unconditional value is written this way: `[class=cond ? …]`
// or a computed one is an expression, and there is no shorthand for an
// expression. Only a *leading* run is taken, so nothing is reordered — the
// parser has already put a literal id first, which is where it renders from.
func shorthandAttrs(attrs []*TagAttribute) (shorthand string, rest []*TagAttribute) {
	var b strings.Builder
	i := 0
	for ; i < len(attrs); i++ {
		a := attrs[i]
		if a.Condition != nil || a.Spread != nil {
			break
		}
		lit, ok := a.Value.(*gnode.StrLit)
		if !ok {
			break
		}
		switch a.Name {
		case "id":
			if b.Len() > 0 || lit.Value() == "" {
				// A second id is not a shorthand, and an empty one would read
				// back as no name at all.
				return b.String(), attrs[i:]
			}
			b.WriteString("#" + shorthandToken(lit.Value()))
		case "class":
			names := strings.Fields(lit.Value())
			if len(names) == 0 {
				return b.String(), attrs[i:]
			}
			for _, n := range names {
				b.WriteString("." + shorthandToken(n))
			}
		default:
			return b.String(), attrs[i:]
		}
	}
	return b.String(), attrs[i:]
}

// shorthandToken writes one shorthand name: bare when it is only letters,
// digits, `_` and `-`, quoted otherwise — a Tailwind variant such as
// `group-hover:bg-black/60` ends at its `:` unless it is quoted.
func shorthandToken(name string) string {
	for _, r := range name {
		if r == '-' || r == '_' || r >= '0' && r <= '9' ||
			r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' {
			continue
		}
		return strconv.Quote(name)
	}
	return name
}

func (c *GadxCodeWriteContext) attrGroups(attrs []*TagAttribute) []attrGroup {
	var groups []attrGroup
	var run []string
	flush := func() {
		if len(run) > 0 {
			groups = append(groups, attrGroup{items: run, merged: true})
			run = nil
		}
	}
	for _, a := range attrs {
		if a.mergeable() {
			run = append(run, a.inner(c))
			continue
		}
		flush()
		groups = append(groups, attrGroup{raw: a.fragment(c)})
	}
	flush()
	return groups
}

// writeWrappedTag emits a tag whose attributes overflow the column budget with
// the merged group expanded one item per line:
//
//	div[
//	    a="v"
//	    b="x"
//	]
//
// A tag that also has non-mergeable groups is emitted inline (accepting the
// overflow) rather than reordered.
func (c *GadxCodeWriteContext) writeWrappedTag(name string, groups []attrGroup) {
	if len(groups) == 1 && groups[0].merged {
		c.WriteLine(name + "[")
		c.Depth++
		for _, it := range groups[0].items {
			c.WriteLine(it)
		}
		c.Depth--
		c.WriteLine("]")
		return
	}
	line := name
	for _, g := range groups {
		line += g.inline()
	}
	c.WriteLine(line)
}

// overflows reports whether the line (at the current indent) exceeds the column
// budget.
func (c *GadxCodeWriteContext) overflows(line string) bool {
	max := c.MaxColumns
	if max <= 0 {
		max = gnode.DefaultMaxColumns
	}
	return len(c.indent())+len(line) > max
}

// inlineTagText returns the inline text for a tag body that is a single
// single-line text run (no interpolation newlines), and whether it qualifies.
// Such a body is emitted as `tag text` on the tag line rather than an indented
// `| text`.
func (c *GadxCodeWriteContext) inlineTagText(body gnode.Stmts, preserve bool) (string, bool) {
	if len(body) != 1 {
		return "", false
	}
	if preserve {
		ts, ok := body[0].(*TextStmt)
		if !ok {
			return "", false
		}
		if !preserveNeeded(ts.Stmts) {
			// Nothing an ordinary inline text would lose, so it reads better
			// written plainly.
			preserve = false
		} else {
			if plain, ok := runPlainText(ts.Stmts); !ok || strings.Contains(plain, "\n") {
				return "", false
			}
			return preservedRun(c, ts.Stmts), true
		}
	}
	ts, ok := body[0].(*TextStmt)
	if !ok {
		return "", false
	}
	raw := c.buildMixed(ts.Stmts)
	text := strings.TrimSpace(raw)
	if text == "" || strings.ContainsAny(text, "\n") {
		return "", false
	}
	// Inlining a plain body drops the whitespace around it, and that whitespace
	// renders — it is the space between this text and whatever sits beside the
	// element. A literal run keeps it by going out interpolated (`tag {= " x " }`),
	// which carries the edges through the parse. A run with a computed part
	// cannot be quoted whole, so it keeps its own line.
	if runHasEdgeSpace(ts.Stmts) {
		plain, ok := runPlainText(ts.Stmts)
		if !ok {
			return "", false
		}
		return "{= " + strconv.Quote(plain) + " }", true
	}
	// A leading `|`/`<`/`@`/`!`/`+`/`~` would be re-parsed as a directive rather
	// than inline text, so keep those on their own `| ` line.
	switch text[0] {
	case '|', '<', '@', '!', '+', '~', '.', '#':
		return "", false
	}
	return text, true
}

// fragment returns the attribute's inline gadx form, to be appended to the tag
// line, as a `[name=value]` (or `[**spread]`) bracket group. The bracket form is
// used for every attribute — including class/id — because it re-parses reliably,
// whereas the `.class` / `#id` shorthands do not round-trip for all values.
func (a *TagAttribute) fragment(ctx *GadxCodeWriteContext) string {
	cond := ""
	if a.Condition != nil {
		cond = " ? " + ctx.gadExpr(a.Condition)
	}
	if a.Spread != nil {
		return "[**" + ctx.gadExpr(a.Spread) + cond + "]"
	}
	s := "[" + a.Name
	if a.IsFlag {
		s += cond
	} else if a.Value != nil {
		// The Gad formatter already renders a string literal with its quotes and
		// an expression without, so IsRaw needs no extra quoting here.
		s += "=" + ctx.gadExpr(a.Value) + cond
	}
	return s + "]"
}

// WriteGadx emits the HTML region as pug-style gadx: its parsed children
// (TagStmt / TextStmt / …) are written with their own WriteGadx, so
// `<a href="/">Home</a>` transpiles to `a[href="/"]` with an indented `| Home`.
func (h *HTMLStmt) WriteGadx(ctx *GadxCodeWriteContext) {
	ctx.WriteStmts(h.Children)
}

func (d *DoctypeStmt) WriteGadx(ctx *GadxCodeWriteContext) {
	ctx.WriteLine("!!! " + d.Value)
}

// WriteGadx writes an HTML comment back as it was written. It is content, so it
// keeps its own delimiters; a `//` line would make it a note in the template
// and it would stop reaching the page.
func (c *HTMLCommentStmt) WriteGadx(ctx *GadxCodeWriteContext) {
	lines := strings.Split(c.Source(), "\n")
	for _, line := range lines {
		ctx.WriteLine(line)
	}
}

func (c *CommentStmt) WriteGadx(ctx *GadxCodeWriteContext) {
	if c.Block {
		// A `/* … */` block comment (silent); a `/** … **/` doc comment (gad
		// convention) when it carries doc.
		if c.Doc {
			ctx.writeBlockComment("/**", c.Text, "**/")
		} else {
			ctx.writeBlockComment("/*", c.Text, "*/")
		}
		return
	}
	// `///` is a single-line doc comment; `//` is a plain (silent) comment.
	prefix := "//"
	if c.Doc {
		prefix = "///"
	}
	ctx.WriteLine(prefix + " " + c.Text)
	if len(c.Body) > 0 {
		ctx.Depth++
		ctx.WriteStmts(c.Body)
		ctx.Depth--
	}
}

// writeDoc emits a decl's `/** … **/` doc comment line (gad convention), if any.
func writeDoc(ctx *GadxCodeWriteContext, doc string) {
	if doc != "" {
		ctx.writeBlockComment("/**", doc, "**/")
	}
}

// writeBlockComment emits a block comment/doc. A single-line body is written
// compactly as `open text close`; a multi-line body keeps open and close on
// their own lines with the text between, so the opening/closing line breaks
// survive a round-trip.
func (c *GadxCodeWriteContext) writeBlockComment(open, text, close string) {
	if !strings.Contains(text, "\n") {
		c.WriteLine(open + " " + text + " " + close)
		return
	}
	c.WriteLine(open)
	for _, line := range strings.Split(text, "\n") {
		c.WriteLine(line)
	}
	c.WriteLine(close)
}

func (s *IfStmt) WriteGadx(ctx *GadxCodeWriteContext) {
	ctx.WriteLine("@if " + ctx.gadCond(s.Cond))
	ctx.Depth++
	ctx.WriteStmts(s.Body)
	ctx.Depth--
	for _, eif := range s.ElseIfs {
		ctx.WriteLine("@else if " + ctx.gadCond(eif.Cond))
		ctx.Depth++
		ctx.WriteStmts(eif.Body)
		ctx.Depth--
	}
	if len(s.Else) > 0 {
		ctx.WriteLine("@else")
		ctx.Depth++
		ctx.WriteStmts(s.Else)
		ctx.Depth--
	}
}

func (s *ForStmt) WriteGadx(ctx *GadxCodeWriteContext) {
	ctx.WriteLine("@for " + ctx.forCond(s.Cond))
	ctx.Depth++
	ctx.WriteStmts(s.Body)
	ctx.Depth--
	if len(s.Else) > 0 {
		ctx.WriteLine("@else")
		ctx.Depth++
		ctx.WriteStmts(s.Else)
		ctx.Depth--
	}
}

func (s *AssignStmt) WriteGadx(ctx *GadxCodeWriteContext) {
	ctx.WriteLine(ctx.gadExpr(s.LHS) + " " + s.Op + " " + ctx.gadExpr(s.RHS))
}

func (c *CodeStmt) WriteGadx(ctx *GadxCodeWriteContext) {
	if len(c.Stmts) == 1 {
		ctx.WriteLine("~ " + ctx.gadCode(c.Stmts[0]))
	} else if len(c.Stmts) > 1 {
		ctx.WriteLine("~~")
		for _, stmt := range c.Stmts {
			ctx.WriteLine(ctx.gadCode(stmt))
		}
		ctx.WriteLine("~~")
	}
}

func (f *FuncDecl) WriteGadx(ctx *GadxCodeWriteContext) {
	writeDoc(ctx, f.Doc)
	line := "@func " + f.Name
	if f.Params != nil {
		line += f.Params.String()
	}
	ctx.WriteLine(line)
	ctx.Depth++
	ctx.WriteStmts(f.Body)
	ctx.Depth--
}

func (c *CompDecl) WriteGadx(ctx *GadxCodeWriteContext) {
	writeDoc(ctx, c.Doc)
	// `@main` is the anonymous entry component (auto-rendered): it must not be
	// re-emitted as `@comp main()`, which would define an unused component and
	// render nothing. `@export comp` marks an exported component.
	var line string
	switch {
	case c.Main:
		line = "@main"
	case c.Exported:
		line = "@export comp " + c.Name
	default:
		line = "@comp " + c.Name
	}
	if c.Params != nil {
		// Suppress an empty `()` on `@main` (the entry block reads globals, not
		// params) so it stays the canonical bare `@main`.
		if p := c.Params.String(); !(c.Main && p == "()") {
			line += p
		}
	}
	ctx.WriteLine(line)
	ctx.Depth++
	ctx.WriteStmts(c.Body)
	ctx.Depth--
}

func (c *CompCallStmt) WriteGadx(ctx *GadxCodeWriteContext) {
	line := "+" + c.Name
	if c.Args.Args.Valid() || c.Args.NamedArgs.Valid() {
		line += c.Args.String()
	}
	ctx.WriteLine(line)
	ctx.Depth++
	// Call-scope `~` code (InitStmts) is emitted first — a slot's interpolated
	// name / body may reference the values it declares — then the slot passes.
	ctx.WriteStmts(c.InitStmts)
	for _, sp := range c.SlotPass {
		sp.WriteGadx(ctx)
	}
	ctx.Depth--
}

func (s *SlotDecl) WriteGadx(ctx *GadxCodeWriteContext) {
	name := s.Name
	if s.NameExpr != nil {
		// A dynamic (interpolated) name — `@slot "item[{i}]"` — must stay quoted;
		// emitting the bare name does not re-parse as a dynamic slot.
		name = `"` + s.Name + `"`
	}
	line := "@slot " + name
	if s.Scope != nil {
		// Emit the scope from its rendered params, suppressing an empty `()` (the
		// LParen position is synthetic, so it cannot gate emission).
		if sc := s.Scope.String(); sc != "" && sc != "()" {
			line += sc
		}
	}
	ctx.WriteLine(line)
	ctx.Depth++
	ctx.WriteStmts(s.Body)
	ctx.Depth--
}

func (s *SlotPassStmt) WriteGadx(ctx *GadxCodeWriteContext) {
	line := "@slot #"
	if s.NameExpr != nil && s.Name != nil {
		line += `"` + s.Name.String() + `"` // dynamic pass name stays quoted
	} else if s.Name != nil {
		line += s.Name.String()
	}
	if s.FuncType != nil {
		// The scope `(it)` must survive; LParen.IsValid() is false for a parsed
		// slot-pass scope, so gate on the rendered params (suppress empty `()`).
		if p := s.FuncType.Params.String(); p != "" && p != "()" {
			line += p
		}
	}
	ctx.WriteLine(line)
	ctx.Depth++
	ctx.WriteStmts(s.Body)
	ctx.Depth--
}

func (w *WrapStmt) WriteGadx(ctx *GadxCodeWriteContext) {
	ctx.WriteLine("@wrap")
	ctx.Depth++
	ctx.WriteStmts(w.Body)
	ctx.Depth--
}

func (t *TestDecl) WriteGadx(ctx *GadxCodeWriteContext) {
	writeDoc(ctx, t.Doc)
	name := t.Name
	if t.Quoted {
		name = strconv.Quote(t.Name)
	}
	ctx.WriteLine("@test " + name)
	ctx.Depth++
	ctx.WriteStmts(t.Body)
	ctx.Depth--
}

func (s *EnumStmt) WriteGadx(ctx *GadxCodeWriteContext) {
	writeDoc(ctx, s.Doc)
	if s.Decl == nil {
		ctx.WriteLine("@enum " + s.Name)
		return
	}
	// The Decl renders as `enum Name { fields }`; re-emit it as the `@enum`
	// directive `@enum Name (fields)`.
	code := ctx.gadCode(s.Decl)
	open := strings.IndexByte(code, '{')
	closeB := strings.LastIndexByte(code, '}')
	if open >= 0 && closeB > open {
		fields := strings.TrimSpace(code[open+1 : closeB])
		ctx.WriteLine("@enum " + s.Name + " (" + fields + ")")
		return
	}
	ctx.WriteLine("@enum " + s.Name)
}

func (s *CallLineStmt) WriteGadx(ctx *GadxCodeWriteContext) {
	parts := make([]string, 0, len(s.Args)+1)
	parts = append(parts, ctx.gadExpr(s.Callee))
	for _, a := range s.Args {
		parts = append(parts, ctx.gadExpr(a))
	}
	ctx.WriteLine("! " + strings.Join(parts, " "))
}

func (s *MatchStmt) WriteGadx(ctx *GadxCodeWriteContext) {
	ctx.WriteLine("@match " + ctx.gadCond(s.Tag))
	ctx.Depth++
	for _, c := range s.Cases {
		ctx.WriteLine("@case " + ctx.gadCond(c.Expr))
		ctx.Depth++
		ctx.WriteStmts(c.Body)
		ctx.Depth--
	}
	if len(s.Default) > 0 {
		ctx.WriteLine("@else")
		ctx.Depth++
		ctx.WriteStmts(s.Default)
		ctx.Depth--
	}
	ctx.Depth--
}

func (s *VarStmt) WriteGadx(ctx *GadxCodeWriteContext) {
	var parts []string
	for _, d := range s.Decls {
		if d.Init != nil {
			parts = append(parts, fmt.Sprintf("%s = %s", d.Name, ctx.gadExpr(d.Init)))
		} else {
			parts = append(parts, d.Name)
		}
	}
	ctx.WriteLine("@var (" + strings.Join(parts, ", ") + ")")
}

func (s *ConstStmt) WriteGadx(ctx *GadxCodeWriteContext) {
	var parts []string
	for _, d := range s.Decls {
		if d.Init != nil {
			parts = append(parts, fmt.Sprintf("%s = %s", d.Name, ctx.gadExpr(d.Init)))
		} else {
			parts = append(parts, d.Name)
		}
	}
	ctx.WriteLine("@const (" + strings.Join(parts, ", ") + ")")
}

func (s *GlobalStmt) WriteGadx(ctx *GadxCodeWriteContext) {
	if s.Decl != nil {
		// s.Decl renders `global (…)`; re-emit it as the `@global` directive
		// (Decl takes precedence over Names, e.g. `@global Model`).
		ctx.WriteLine("@global " + strings.TrimSpace(strings.TrimPrefix(s.Decl.String(), "global")))
		return
	}
	if len(s.Names) == 0 {
		ctx.WriteLine("@global")
		return
	}
	ctx.WriteLine("@global " + strings.Join(s.Names, " "))
}

func (s *ParamStmt) WriteGadx(ctx *GadxCodeWriteContext) {
	writeDoc(ctx, s.Doc) // a lead `/** … **/` doc is attached to the ParamStmt
	if s.Decl == nil {
		ctx.WriteLine("@param")
		return
	}
	// s.Decl.String() renders `param …`; re-emit it as the `@param` directive.
	ctx.WriteLine("@param " + strings.TrimSpace(strings.TrimPrefix(s.Decl.String(), "param")))
}

func (e *ExportStmt) WriteGadx(ctx *GadxCodeWriteContext) {
	line := "@export " + e.Name
	if e.Value != nil {
		line += " = " + ctx.gadExpr(e.Value)
	}
	ctx.WriteLine(line)
}

// =============================================================================
// GadxCoder interface check
// =============================================================================

var (
	_ GadxCoder = (*TextStmt)(nil)
	_ GadxCoder = (*TagStmt)(nil)
	_ GadxCoder = (*DoctypeStmt)(nil)
	_ GadxCoder = (*CommentStmt)(nil)
	_ GadxCoder = (*IfStmt)(nil)
	_ GadxCoder = (*ForStmt)(nil)
	_ GadxCoder = (*AssignStmt)(nil)
	_ GadxCoder = (*CodeStmt)(nil)
	_ GadxCoder = (*FuncDecl)(nil)
	_ GadxCoder = (*CompDecl)(nil)
	_ GadxCoder = (*CompCallStmt)(nil)
	_ GadxCoder = (*SlotDecl)(nil)
	_ GadxCoder = (*SlotPassStmt)(nil)
	_ GadxCoder = (*WrapStmt)(nil)
	_ GadxCoder = (*MatchStmt)(nil)
	_ GadxCoder = (*VarStmt)(nil)
	_ GadxCoder = (*ConstStmt)(nil)
	_ GadxCoder = (*GlobalStmt)(nil)
	_ GadxCoder = (*ParamStmt)(nil)
	_ GadxCoder = (*ExportStmt)(nil)
	_ GadxCoder = (*TestDecl)(nil)
	_ GadxCoder = (*CallLineStmt)(nil)
	_ GadxCoder = (*EnumStmt)(nil)
)

// writeRawTextTag writes a raw-text element (a script or a stylesheet) in the
// inline-HTML form, so its content survives verbatim. The attributes come out as
// HTML: a literal value quoted, a flag bare, anything computed in `{ … }`.
// rawTextBlockSafe reports whether a raw-text body survives being written as an
// indented block: its first and last lines have to carry content, since the
// block's edges are where indentation is read and trailing space is trimmed.
func rawTextBlockSafe(content string) bool {
	if content == "" {
		return false
	}
	lines := strings.Split(dedentRaw(content), "\n")
	return strings.TrimSpace(lines[0]) != "" && strings.TrimSpace(lines[len(lines)-1]) != ""
}

// writeRawTextRegion writes a raw-text element back as the inline HTML region
// it came from, for a body the block form cannot carry unchanged.
func (c *GadxCodeWriteContext) writeRawTextRegion(t *TagStmt) {
	var b strings.Builder
	b.WriteString("<" + t.Name)
	for _, a := range t.Attributes {
		if a.Name == "" {
			continue
		}
		switch {
		case a.IsFlag || a.Value == nil:
			b.WriteString(" " + a.Name)
		default:
			if lit, ok := a.Value.(*gnode.StrLit); ok {
				b.WriteString(" " + a.Name + "=" + strconv.Quote(lit.Value()))
			} else {
				b.WriteString(" " + a.Name + "={" + c.gadExpr(a.Value) + "}")
			}
		}
	}
	b.WriteString(">")
	c.write(c.indent() + b.String())
	c.write(c.rawTextContent(t.Body))
	c.write("</" + t.Name + ">\n")
}

func (c *GadxCodeWriteContext) writeRawTextTag(t *TagStmt, inline string) {
	c.WriteLine(inline)
	c.Depth++
	for _, line := range strings.Split(dedentRaw(c.rawTextContent(t.Body)), "\n") {
		if line == "" {
			// A blank line carries no indentation to restore, and writing one
			// would put trailing whitespace in the file.
			c.write("\n")
			continue
		}
		c.WriteLine(line)
	}
	c.Depth--
}

// dedentRaw removes the indentation common to every non-blank line of a
// raw-text body, which is the indentation of the element it came from rather
// than part of its content. What is left is written back under the tag, where
// the parser strips the block's indentation the same way.
func dedentRaw(s string) string {
	lines := strings.Split(s, "\n")
	prefix, first := "", true
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		lead := l[:len(l)-len(strings.TrimLeft(l, " \t"))]
		if first {
			prefix, first = lead, false
			continue
		}
		n := 0
		for n < len(prefix) && n < len(lead) && prefix[n] == lead[n] {
			n++
		}
		prefix = prefix[:n]
	}
	for i, l := range lines {
		lines[i] = strings.TrimPrefix(l, prefix)
	}
	return strings.Join(lines, "\n")
}

// rawTextContent rebuilds the source text of a raw-text element's body: the
// literal chunks as they were, and each interpolation back in its `#{= … }#`
// form.
func (c *GadxCodeWriteContext) rawTextContent(body gnode.Stmts) string {
	var b strings.Builder
	for _, stmt := range body {
		ts, ok := stmt.(*TextStmt)
		if !ok {
			continue
		}
		for _, s := range ts.Stmts {
			switch v := s.(type) {
			case *gnode.MixedTextStmt:
				b.WriteString(v.Lit.Value)
			case *gnode.MixedValueStmt:
				if rs, ok := v.Expr.(*gnode.RawStrLit); ok && v.StartLit.Value == "" {
					b.WriteString(rs.Value())
					continue
				}
				b.WriteString("#{= " + c.gadExpr(v.Expr) + " }#")
			case *gnode.ExprStmt:
				b.WriteString("#{ " + c.gadExpr(v.Expr) + " }#")
			}
		}
	}
	return b.String()
}

// runHasEdgeSpace reports whether a text run begins or ends with whitespace,
// looking through a literal interpolation to the string it carries.
func runHasEdgeSpace(stmts gnode.Stmts) bool {
	first, last, ok := runEdges(stmts)
	if !ok {
		return false
	}
	return first != strings.TrimLeft(first, " \t") || last != strings.TrimRight(last, " \t")
}

// preserveNeeded reports whether a run holds whitespace that the ordinary
// line-based forms would lose: a line break, or an edge that a `| ` line strips.
// Everything else reads the same written plainly, and reads better.
func preserveNeeded(stmts gnode.Stmts) bool {
	for _, stmt := range stmts {
		switch stmt.(type) {
		case *gnode.MixedTextStmt, *gnode.MixedValueStmt:
		default:
			// A statement the quoted form cannot carry; write the run the
			// ordinary way rather than dropping part of it.
			return false
		}
	}
	if runHasEdgeSpace(stmts) {
		return true
	}
	// The test is on the text the run renders, not on how it is written: once
	// the whitespace has moved into a `{= "…" }` literal the source form no
	// longer shows it, and reading only the plain segments would say the run is
	// ordinary and write it back as one — losing what the quotes were carrying.
	if plain, ok := runPlainText(stmts); ok {
		return strings.Contains(plain, "\n")
	}
	for _, stmt := range stmts {
		if mt, ok := stmt.(*gnode.MixedTextStmt); ok && strings.Contains(mt.Lit.Value, "\n") {
			return true
		}
	}
	return false
}

// preservedRun writes a run whose whitespace is content: every literal segment
// goes out as a quoted string, so the line breaks and the edges survive the
// parse, and the interpolations between them keep their own form. It is one
// line, and it lowers to the one text run it came from.
func preservedRun(c *GadxCodeWriteContext, stmts gnode.Stmts) string {
	var b strings.Builder
	for _, stmt := range stmts {
		switch v := stmt.(type) {
		case *gnode.MixedTextStmt:
			b.WriteString("{= " + strconv.Quote(v.Lit.Value) + " }")
		case *gnode.MixedValueStmt:
			b.WriteString(c.gadxInterp(v))
		}
	}
	return b.String()
}

// runPlainText returns the text a run renders to, when every one of its parts
// is a literal — plain text, or an interpolation of a string constant. ok is
// false as soon as a part is computed, whose text cannot be known here.
func runPlainText(stmts gnode.Stmts) (string, bool) {
	var b strings.Builder
	for _, stmt := range stmts {
		switch v := stmt.(type) {
		case *gnode.MixedTextStmt:
			b.WriteString(v.Lit.Value)
		case *gnode.MixedValueStmt:
			switch e := v.Expr.(type) {
			case *gnode.StrLit:
				b.WriteString(e.Value())
			case *gnode.RawStrLit:
				b.WriteString(e.Value())
			default:
				return "", false
			}
		default:
			return "", false
		}
	}
	return b.String(), len(stmts) > 0
}

// runEdges returns the leading and trailing literal text of a run. ok is false
// when an edge is a computed value, whose whitespace cannot be known here.
func runEdges(stmts gnode.Stmts) (first, last string, ok bool) {
	literal := func(stmt gnode.Stmt) (string, bool) {
		switch v := stmt.(type) {
		case *gnode.MixedTextStmt:
			return v.Lit.Value, true
		case *gnode.MixedValueStmt:
			switch e := v.Expr.(type) {
			case *gnode.StrLit:
				return e.Value(), true
			case *gnode.RawStrLit:
				return e.Value(), true
			}
		}
		return "", false
	}
	if len(stmts) == 0 {
		return "", "", false
	}
	if first, ok = literal(stmts[0]); !ok {
		return "", "", false
	}
	if last, ok = literal(stmts[len(stmts)-1]); !ok {
		return "", "", false
	}
	return first, last, true
}
