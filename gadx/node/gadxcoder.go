package node

import (
	"fmt"
	"io"
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

// gadExpr renders an embedded GAD expression with the configured formatting
// rules. It returns "" for a nil expression.
func (c *GadxCodeWriteContext) gadExpr(e gnode.Expr) string {
	if e == nil {
		return ""
	}
	return c.gadCode(e)
}

// gadCode renders any GAD coder (expression or statement) with the context's
// formatting flags and column budget, matching `gad fmt`. A trailing newline is
// trimmed so the result can be embedded inline.
func (c *GadxCodeWriteContext) gadCode(n gnode.Coder) string {
	opts := []gnode.CodeOption{gnode.CodeWithFlags(c.EmbedFlags)}
	if c.MaxColumns > 0 {
		opts = append(opts, gnode.CodeWithMaxColumns(c.MaxColumns))
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
	// Reconstruct text from mixed GAD statements
	for _, stmt := range t.Stmts {
		switch s := stmt.(type) {
		case *gnode.MixedTextStmt:
			ctx.WriteLine("| " + s.String())
		case *gnode.MixedValueStmt:
			ctx.WriteLine("| {" + s.String() + "}")
		default:
			ctx.WriteLine("| " + s.String())
		}
	}
}

func (t *TagStmt) WriteGadx(ctx *GadxCodeWriteContext) {
	ctx.WriteLine(t.Name)
	ctx.Depth++
	for _, attr := range t.Attributes {
		attr.writeGadx(ctx)
	}
	ctx.WriteStmts(t.Body)
	ctx.Depth--
}

func (a *TagAttribute) writeGadx(ctx *GadxCodeWriteContext) {
	cond := ""
	if a.Condition != nil {
		cond = " ? " + ctx.gadExpr(a.Condition)
	}
	if a.Spread != nil {
		ctx.WriteLine("[**" + ctx.gadExpr(a.Spread) + cond + "]")
		return
	}
	switch a.Name {
	case "id":
		ctx.WriteLine("#" + ctx.gadExpr(a.Value) + cond)
	case "class":
		ctx.WriteLine("." + ctx.gadExpr(a.Value) + cond)
	default:
		s := "[" + a.Name
		if a.IsFlag {
			s += cond
		} else if a.Value != nil {
			if a.IsRaw {
				s += "=\"" + ctx.gadExpr(a.Value) + "\""
			} else {
				s += "=" + ctx.gadExpr(a.Value)
			}
			s += cond
		}
		s += "]"
		ctx.WriteLine(s)
	}
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
	ctx.WriteLine("@if " + ctx.gadExpr(s.Cond))
	ctx.Depth++
	ctx.WriteStmts(s.Body)
	ctx.Depth--
	for _, eif := range s.ElseIfs {
		ctx.WriteLine("@else if " + ctx.gadExpr(eif.Cond))
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
	ctx.WriteLine("@for " + ctx.gadExpr(s.Cond))
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
	line := "@slot " + s.Name
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
	if s.Name != nil {
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

func (s *MatchStmt) WriteGadx(ctx *GadxCodeWriteContext) {
	ctx.WriteLine("@match " + ctx.gadExpr(s.Tag))
	ctx.Depth++
	for _, c := range s.Cases {
		ctx.WriteLine("@case " + ctx.gadExpr(c.Expr))
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
	ctx.WriteLine("@global " + strings.Join(s.Names, " "))
}

func (s *ParamStmt) WriteGadx(ctx *GadxCodeWriteContext) {
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
)
