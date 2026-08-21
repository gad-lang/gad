package node

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	gnode "github.com/gad-lang/gad/parser/node"
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
	ctx.WriteStmts(f.Stmts)
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
func (ctx *GadxCodeWriteContext) writeRawBlock(directive string, body gnode.Stmts) {
	ctx.WriteLine(directive)
	ctx.Depth++
	prev := ctx.raw
	ctx.raw = true
	ctx.WriteStmts(body)
	ctx.raw = prev
	ctx.Depth--
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
	// Attributes are written inline on the tag line (`div.card[href=x]`), the
	// canonical gadx form; emitting them as separate indented lines does not
	// re-parse. The body/text follows indented below.
	line := t.Name
	for _, attr := range t.Attributes {
		line += attr.fragment(ctx)
	}
	ctx.WriteLine(line)
	ctx.Depth++
	ctx.WriteStmts(t.Body)
	ctx.Depth--
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
			ctx.WriteLine("/** " + c.Text + " **/")
		} else {
			ctx.WriteLine("/* " + c.Text + " */")
		}
		return
	}
	prefix := "//"
	if c.Silent {
		prefix = "//-"
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
		ctx.WriteLine("/** " + doc + " **/")
	}
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
	ctx.WriteLine("@for " + ctx.gadCond(s.Cond))
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
	line := "@comp " + c.Name
	if c.Params != nil {
		line += c.Params.String()
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
		line += s.Scope.String()
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
	if s.FuncType != nil && s.FuncType.Params.LParen.IsValid() {
		line += s.FuncType.Params.String()
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
)
