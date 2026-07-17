package node

import (
	"fmt"
	"io"
	"strings"

	gnode "github.com/gad-lang/gad/parser/node"
)

// =============================================================================
// GiomCoder — regenerate formatted giom source (like gofmt)
// =============================================================================

// GiomCodeWriteContext holds state for writing giom template source.
type GiomCodeWriteContext struct {
	Writer io.Writer
	Depth  int
	Prefix string // indentation string (default "\t")
}

// NewGiomCodeContext creates a new context writing to w.
func NewGiomCodeContext(w io.Writer) *GiomCodeWriteContext {
	return &GiomCodeWriteContext{Writer: w, Prefix: "\t"}
}

func (c *GiomCodeWriteContext) indent() string {
	return strings.Repeat(c.Prefix, c.Depth)
}

func (c *GiomCodeWriteContext) write(s string) {
	io.WriteString(c.Writer, s)
}

// WriteLine writes an indented line followed by newline.
func (c *GiomCodeWriteContext) WriteLine(s string) {
	c.write(c.indent() + s + "\n")
}

// WriteStmts writes a list of giom statements at the current depth.
func (c *GiomCodeWriteContext) WriteStmts(stmts gnode.Stmts) {
	for _, stmt := range stmts {
		if gc, ok := stmt.(GiomCoder); ok {
			gc.WriteGiom(c)
		}
	}
}

// GiomCoder is implemented by nodes that can write formatted giom source.
type GiomCoder interface {
	WriteGiom(ctx *GiomCodeWriteContext)
}

// =============================================================================
// WriteGiom implementations
// =============================================================================

func (f *File) WriteGiom(ctx *GiomCodeWriteContext) {
	ctx.WriteStmts(f.Stmts)
}

func (t *TextStmt) WriteGiom(ctx *GiomCodeWriteContext) {
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

func (t *TagStmt) WriteGiom(ctx *GiomCodeWriteContext) {
	ctx.WriteLine(t.Name)
	ctx.Depth++
	for _, attr := range t.Attributes {
		attr.writeGiom(ctx)
	}
	ctx.WriteStmts(t.Body)
	ctx.Depth--
}

func (a *TagAttribute) writeGiom(ctx *GiomCodeWriteContext) {
	cond := ""
	if a.Condition != nil {
		cond = " ? " + a.Condition.String()
	}
	switch a.Name {
	case "id":
		ctx.WriteLine("#" + exprStr(a.Value) + cond)
	case "class":
		ctx.WriteLine("." + exprStr(a.Value) + cond)
	default:
		s := "[" + a.Name
		if a.IsFlag {
			s += cond
		} else if a.Value != nil {
			if a.IsRaw {
				s += "=\"" + exprStr(a.Value) + "\""
			} else {
				s += "=" + exprStr(a.Value)
			}
			s += cond
		}
		s += "]"
		ctx.WriteLine(s)
	}
}

func (d *DoctypeStmt) WriteGiom(ctx *GiomCodeWriteContext) {
	ctx.WriteLine("!!! " + d.Value)
}

func (c *CommentStmt) WriteGiom(ctx *GiomCodeWriteContext) {
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

func (s *IfStmt) WriteGiom(ctx *GiomCodeWriteContext) {
	ctx.WriteLine("@if " + exprStr(s.Cond))
	ctx.Depth++
	ctx.WriteStmts(s.Body)
	ctx.Depth--
	for _, eif := range s.ElseIfs {
		ctx.WriteLine("@else if " + exprStr(eif.Cond))
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

func (s *ForStmt) WriteGiom(ctx *GiomCodeWriteContext) {
	ctx.WriteLine("@for " + exprStr(s.Cond))
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

func (s *AssignStmt) WriteGiom(ctx *GiomCodeWriteContext) {
	ctx.WriteLine(exprStr(s.LHS) + " " + s.Op + " " + exprStr(s.RHS))
}

func (c *CodeStmt) WriteGiom(ctx *GiomCodeWriteContext) {
	if len(c.Stmts) == 1 {
		ctx.WriteLine("~ " + c.Stmts[0].String())
	} else if len(c.Stmts) > 1 {
		ctx.WriteLine("~~")
		for _, stmt := range c.Stmts {
			ctx.WriteLine(stmt.String())
		}
		ctx.WriteLine("~~")
	}
}

func (f *FuncDecl) WriteGiom(ctx *GiomCodeWriteContext) {
	line := "@func " + f.Name
	if f.Params != nil {
		line += f.Params.String()
	}
	ctx.WriteLine(line)
	ctx.Depth++
	ctx.WriteStmts(f.Body)
	ctx.Depth--
}

func (c *CompDecl) WriteGiom(ctx *GiomCodeWriteContext) {
	line := "@comp " + c.Name
	if c.Params != nil {
		line += c.Params.String()
	}
	ctx.WriteLine(line)
	ctx.Depth++
	ctx.WriteStmts(c.Body)
	ctx.Depth--
}

func (c *CompCallStmt) WriteGiom(ctx *GiomCodeWriteContext) {
	line := "+" + c.Name
	if c.Args.Args.Valid() || c.Args.NamedArgs.Valid() {
		line += c.Args.String()
	}
	ctx.WriteLine(line)
	ctx.Depth++
	for _, sp := range c.SlotPass {
		sp.WriteGiom(ctx)
	}
	ctx.Depth--
}

func (s *SlotDecl) WriteGiom(ctx *GiomCodeWriteContext) {
	line := "@slot " + s.Name
	if s.Scope != nil {
		line += s.Scope.String()
	}
	ctx.WriteLine(line)
	ctx.Depth++
	ctx.WriteStmts(s.Body)
	ctx.Depth--
}

func (s *SlotPassStmt) WriteGiom(ctx *GiomCodeWriteContext) {
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

func (w *WrapStmt) WriteGiom(ctx *GiomCodeWriteContext) {
	ctx.WriteLine("@wrap")
	ctx.Depth++
	ctx.WriteStmts(w.Body)
	ctx.Depth--
}

func (s *MatchStmt) WriteGiom(ctx *GiomCodeWriteContext) {
	ctx.WriteLine("@match " + exprStr(s.Tag))
	ctx.Depth++
	for _, c := range s.Cases {
		ctx.WriteLine("@case " + exprStr(c.Expr))
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

func (s *VarStmt) WriteGiom(ctx *GiomCodeWriteContext) {
	var parts []string
	for _, d := range s.Decls {
		if d.Init != nil {
			parts = append(parts, fmt.Sprintf("%s = %s", d.Name, exprStr(d.Init)))
		} else {
			parts = append(parts, d.Name)
		}
	}
	ctx.WriteLine("@var (" + strings.Join(parts, ", ") + ")")
}

func (s *ConstStmt) WriteGiom(ctx *GiomCodeWriteContext) {
	var parts []string
	for _, d := range s.Decls {
		if d.Init != nil {
			parts = append(parts, fmt.Sprintf("%s = %s", d.Name, exprStr(d.Init)))
		} else {
			parts = append(parts, d.Name)
		}
	}
	ctx.WriteLine("@const (" + strings.Join(parts, ", ") + ")")
}

func (s *GlobalStmt) WriteGiom(ctx *GiomCodeWriteContext) {
	ctx.WriteLine("@global " + strings.Join(s.Names, " "))
}

func (e *ExportStmt) WriteGiom(ctx *GiomCodeWriteContext) {
	line := "@export " + e.Name
	if e.Value != nil {
		line += " = " + exprStr(e.Value)
	}
	ctx.WriteLine(line)
}

// =============================================================================
// GiomCoder interface check
// =============================================================================

var (
	_ GiomCoder = (*TextStmt)(nil)
	_ GiomCoder = (*TagStmt)(nil)
	_ GiomCoder = (*DoctypeStmt)(nil)
	_ GiomCoder = (*CommentStmt)(nil)
	_ GiomCoder = (*IfStmt)(nil)
	_ GiomCoder = (*ForStmt)(nil)
	_ GiomCoder = (*AssignStmt)(nil)
	_ GiomCoder = (*CodeStmt)(nil)
	_ GiomCoder = (*FuncDecl)(nil)
	_ GiomCoder = (*CompDecl)(nil)
	_ GiomCoder = (*CompCallStmt)(nil)
	_ GiomCoder = (*SlotDecl)(nil)
	_ GiomCoder = (*SlotPassStmt)(nil)
	_ GiomCoder = (*WrapStmt)(nil)
	_ GiomCoder = (*MatchStmt)(nil)
	_ GiomCoder = (*VarStmt)(nil)
	_ GiomCoder = (*ConstStmt)(nil)
	_ GiomCoder = (*GlobalStmt)(nil)
	_ GiomCoder = (*ExportStmt)(nil)
)

// exprStr returns the string representation of a GAD expression.
func exprStr(e gnode.Expr) string {
	if e == nil {
		return ""
	}
	return e.String()
}
