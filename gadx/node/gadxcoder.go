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
	for _, stmt := range stmts {
		if gc, ok := stmt.(GadxCoder); ok {
			gc.WriteGadx(c)
		}
	}
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

	// Reconstruct the mixed run (literal text interleaved with interpolations)
	// as one string, then emit each source line as its own `| ` line. Keeping a
	// run like `x = {= v } (y)` on a single line preserves the spaces around the
	// interpolations — a bare `| ` line strips only trailing whitespace, which
	// would otherwise be lost when the segments are split across lines.
	if strings.TrimSpace(text) == "" {
		return // drop a whitespace-only run: `| ` strips it and it does not survive
	}
	for _, line := range strings.Split(text, "\n") {
		// Trim edge whitespace: a `| ` line strips trailing space on parse and a
		// leading run collapses, so `|  x ` would not round-trip. This matches the
		// HTML rule that whitespace at a text run's edges collapses.
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		ctx.WriteLine("| " + line)
	}
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
	groups := ctx.attrGroups(t.Attributes)
	inline := t.Name
	for _, g := range groups {
		inline += g.inline()
	}

	if !ctx.overflows(inline) {
		// A tag whose whole body is a single short text run is written inline as
		// `tag text` (so `<span>one</span>` → `span one`, not `span` + `| one`).
		if text, ok := ctx.inlineTagText(t.Body); ok && !ctx.overflows(inline+" "+text) {
			ctx.WriteLine(inline + " " + text)
			return
		}
		ctx.WriteLine(inline)
	} else {
		// Overflow: wrap the merged attribute group one item per line.
		ctx.writeWrappedTag(t.Name, groups)
	}
	ctx.Depth++
	ctx.WriteStmts(t.Body)
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
		s += "=" + ctx.gadExpr(a.Value)
	}
	return s
}

// attrGroups builds the tag's attribute groups: consecutive mergeable
// attributes fold into one group (order preserved), each spread/conditional
// attribute is its own group.
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
func (c *GadxCodeWriteContext) inlineTagText(body gnode.Stmts) (string, bool) {
	if len(body) != 1 {
		return "", false
	}
	ts, ok := body[0].(*TextStmt)
	if !ok {
		return "", false
	}
	// Trim edge whitespace exactly as the `| ` text path does, so a tag body that
	// came from an HTML region (with surrounding whitespace text nodes) inlines to
	// the same result on every pass (idempotent).
	text := strings.TrimSpace(c.buildMixed(ts.Stmts))
	if text == "" || strings.ContainsAny(text, "\n") {
		return "", false
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
