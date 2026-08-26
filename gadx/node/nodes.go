package node

import (
	"fmt"
	"strings"

	"github.com/gad-lang/gad/parser/ast"
	gnode "github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
)

// =============================================================================
// File — top-level AST root
// =============================================================================

type File struct {
	Stmts     gnode.Stmts
	Comps     []*CompDecl
	InputFile *source.File
}

func (f *File) Pos() source.Pos {
	if len(f.Stmts) > 0 {
		return f.Stmts[0].Pos()
	}
	return source.NoPos
}

func (f *File) End() source.Pos {
	if len(f.Stmts) > 0 {
		return f.Stmts[len(f.Stmts)-1].End()
	}
	return source.NoPos
}

func (f *File) String() string { return "gadx.File" }

func (f *File) WriteCode(ctx *gnode.CodeWriteContext) {
	ctx.WriteStmts(f.Stmts...)
}

// =============================================================================
// TextStmt — literal text with embedded GAD expressions
// =============================================================================

type TextStmt struct {
	ast.NodeData
	NodePos source.Pos
	NodeEnd source.Pos
	Stmts   gnode.Stmts
}

func (t *TextStmt) Pos() source.Pos { return t.NodePos }
func (t *TextStmt) End() source.Pos { return t.NodeEnd }
func (t *TextStmt) StmtNode()       {}
func (t *TextStmt) String() string  { return "gadx.Text" }

func (t *TextStmt) WriteCode(ctx *gnode.CodeWriteContext) {
	ctx.WriteStmts(convertText(t)...)
}

// =============================================================================
// TextBlockStmt — `@text` literal-text block (children joined by newlines)
// =============================================================================

type TextBlockStmt struct {
	ast.NodeData
	NodePos source.Pos
	NodeEnd source.Pos
	Body    gnode.Stmts // one TextStmt per source line (position-preserving)
	// Pipe marks the YAML-style `|` block form (a bare `|` opening the block)
	// rather than the `@text` directive; both share this node and body.
	Pipe bool
	// Fold marks the YAML folded style `|>`: body line breaks become spaces
	// (YAML `>`), instead of being preserved (`|`). Only meaningful with Pipe.
	Fold bool
}

func (t *TextBlockStmt) Pos() source.Pos { return t.NodePos }
func (t *TextBlockStmt) End() source.Pos { return t.NodeEnd }
func (t *TextBlockStmt) StmtNode()       {}
func (t *TextBlockStmt) String() string  { return "gadx.TextBlock" }

func (t *TextBlockStmt) WriteCode(ctx *gnode.CodeWriteContext) {
	ctx.WriteStmts(convertTextBlock(t)...)
}

// =============================================================================
// ParaBlockStmt — `@p` paragraph block (blank lines split <p> paragraphs)
// =============================================================================

type ParaBlockStmt struct {
	ast.NodeData
	NodePos source.Pos
	NodeEnd source.Pos
	Body    gnode.Stmts // one TextStmt per source line (empty = blank line)
}

func (t *ParaBlockStmt) Pos() source.Pos { return t.NodePos }
func (t *ParaBlockStmt) End() source.Pos { return t.NodeEnd }
func (t *ParaBlockStmt) StmtNode()       {}
func (t *ParaBlockStmt) String() string  { return "gadx.ParaBlock" }

func (t *ParaBlockStmt) WriteCode(ctx *gnode.CodeWriteContext) {
	ctx.WriteStmts(convertParaBlock(t)...)
}

// =============================================================================
// MdBlockStmt — `@md` Markdown block (rendered to HTML via goldmark)
// =============================================================================

type MdBlockStmt struct {
	ast.NodeData
	NodePos source.Pos
	NodeEnd source.Pos
	// Body mixes literal Markdown text lines (TextStmt) with nested `@` directives;
	// on render the whole thing is Markdown source converted to HTML.
	Body gnode.Stmts
}

func (t *MdBlockStmt) Pos() source.Pos { return t.NodePos }
func (t *MdBlockStmt) End() source.Pos { return t.NodeEnd }
func (t *MdBlockStmt) StmtNode()       {}
func (t *MdBlockStmt) String() string  { return "gadx.MdBlock" }

func (t *MdBlockStmt) WriteCode(ctx *gnode.CodeWriteContext) {
	ctx.WriteStmts(convertMdBlock(t)...)
}

// =============================================================================
// TagStmt — HTML/XML tag
// =============================================================================

type TagAttribute struct {
	Name      string
	Value     gnode.Expr
	IsRaw     bool
	IsFlag    bool
	Condition gnode.Expr
	Elements  *gnode.KeyValueArrayLit
	// Spread, when set, is a `**expr` attribute spread (a dict/keyValueArray of
	// name→value). It carries an interpolated-name attribute from an HTML region
	// (`data-{key}=v` → `**{[ "data-"+key ]: v}`), which cannot be a static named
	// argument.
	Spread gnode.Expr
}

type TagStmt struct {
	ast.NodeData
	NodePos     source.Pos
	NodeEnd     source.Pos
	Name        string
	Attributes  []*TagAttribute
	Body        gnode.Stmts
	SelfClosing bool
}

func (t *TagStmt) Pos() source.Pos { return t.NodePos }
func setParens(call *gnode.CallExpr, lparen, rparen source.Pos) {
	if !call.LParen.IsValid() {
		call.LParen = lparen
	}
	if !call.RParen.IsValid() {
		call.RParen = rparen
	}
}

func (t *TagStmt) End() source.Pos { return t.NodeEnd }
func (t *TagStmt) StmtNode()       {}
func (t *TagStmt) String() string  { return fmt.Sprintf("gadx.Tag(%s)", t.Name) }

func gadxCallExpr(method string, pos source.Pos) *gnode.CallExpr {
	return gnode.ECall(gnode.ESelector(gnode.EIdent("gadx", pos), gnode.Str(method, 0)), 0, 0)
}

func (t *TagStmt) WriteCode(ctx *gnode.CodeWriteContext) {
	ctx.WriteStmts(convertTag(t)...)
}

func addNamedArg(call *gnode.CallExpr, name string, value gnode.Expr) {
	if isIdentName(name) {
		call.NamedArgs.AppendS(name, value)
		return
	}
	// A non-identifier attribute name (e.g. `data-line`, `aria-label`) must be a
	// quoted string key; emitted unquoted it re-parses as an expression
	// (`data - line`), so the transpiled Gad fails to parse.
	call.NamedArgs.Names = append(call.NamedArgs.Names, &gnode.NamedArgExpr{Lit: gnode.Str(name, 0)})
	call.NamedArgs.Values = append(call.NamedArgs.Values, value)
}

// isIdentName reports whether name is a valid Gad identifier (letter/underscore
// start, then letters/digits/underscores), so it can be a named-argument key
// without quoting.
func isIdentName(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		switch {
		case r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
		case i > 0 && r >= '0' && r <= '9':
		default:
			return false
		}
	}
	return true
}

func exprKeyName(e gnode.Expr) string {
	switch t := e.(type) {
	case *gnode.IdentExpr:
		return t.Name
	case *gnode.StrLit:
		return t.Value()
	default:
		return t.String()
	}
}

// =============================================================================
// HTMLStmt — a raw HTML region (`<tag …>…</tag>` or `<>…</>` fragment)
// =============================================================================

type HTMLStmt struct {
	ast.NodeData
	NodePos source.Pos
	NodeEnd source.Pos
	// Children are the gadx nodes the HTML region parsed into (TagStmt / TextStmt
	// / fragments), so the region compiles to gadx.Tag / gadx.Text elements — like
	// pug-style tags — rather than raw HTML markup, and transpiles back to
	// pug-style gadx via WriteGadx.
	Children gnode.Stmts
}

func (h *HTMLStmt) Pos() source.Pos { return h.NodePos }
func (h *HTMLStmt) End() source.Pos { return h.NodeEnd }
func (h *HTMLStmt) StmtNode()       {}
func (h *HTMLStmt) String() string  { return "gadx.HTML" }

func (h *HTMLStmt) WriteCode(ctx *gnode.CodeWriteContext) {
	ctx.WriteStmts(convertHTML(h)...)
}

// =============================================================================
// DoctypeStmt — DOCTYPE declaration
// =============================================================================

type DoctypeStmt struct {
	ast.NodeData
	NodePos source.Pos
	NodeEnd source.Pos
	Value   string
}

func (d *DoctypeStmt) Pos() source.Pos { return d.NodePos }
func (d *DoctypeStmt) End() source.Pos { return d.NodeEnd }
func (d *DoctypeStmt) StmtNode()       {}
func (d *DoctypeStmt) String() string  { return fmt.Sprintf("gadx.Doctype(%s)", d.Value) }

func (d *DoctypeStmt) WriteCode(ctx *gnode.CodeWriteContext) {
	ctx.WriteStmts(convertDoctype(d)...)
}

var doctypes = map[string]string{
	"5":            `<!DOCTYPE html>`,
	"default":      `<!DOCTYPE html>`,
	"xml":          `<?xml version="1.0" encoding="utf-8" ?>`,
	"transitional": `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">`,
	"strict":       `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd">`,
	"frameset":     `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Frameset//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-frameset.dtd">`,
	"1.1":          `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">`,
	"basic":        `<!DOCTYPE html PUBLIC "-//WAPFORUM//DTD XHTML Basic 1.1//EN" "http://www.w3.org/TR/xhtml-basic/xhtml-basic11.dtd">`,
	"mobile":       `<!DOCTYPE html PUBLIC "-//WAPFORUM//DTD XHTML Mobile 1.2//EN" "http://www.openmobilealliance.org/tech/DTD/xhtml-mobile12.dtd">`,
}

func doctypeValue(key string) string {
	if v, ok := doctypes[key]; ok {
		return v
	}
	return "<!DOCTYPE " + key + ">"
}

// =============================================================================
// CommentStmt — template comment
// =============================================================================

type CommentStmt struct {
	ast.NodeData
	NodePos source.Pos
	NodeEnd source.Pos
	Text    string
	// Block is true for a `/* … */` block comment (may be multi-line), as opposed
	// to a `//` line comment. All gadx comments are silent (never rendered).
	Block bool
	// Doc is true for a `/** … */` doc block comment. When it immediately
	// precedes a @comp/@func the parser moves its text to the decl's Doc and
	// drops the CommentStmt; otherwise it stays as a file-level comment.
	Doc  bool
	Body gnode.Stmts
}

func (c *CommentStmt) Pos() source.Pos { return c.NodePos }
func (c *CommentStmt) End() source.Pos { return c.NodeEnd }
func (c *CommentStmt) StmtNode()       {}
func (c *CommentStmt) String() string  { return "gadx.Comment" }

// WriteCode emits nothing: gadx comments (`//`, `///`, `/* … */`) are silent.
// Use an inline HTML region (`<!-- … -->`) to emit an HTML comment into the output.
func (c *CommentStmt) WriteCode(ctx *gnode.CodeWriteContext) {}

// =============================================================================
// IfStmt — conditional block
// =============================================================================

type ElseIfClause struct {
	Cond gnode.Expr
	Body gnode.Stmts
}

type IfStmt struct {
	ast.NodeData
	NodePos source.Pos
	NodeEnd source.Pos
	Init    gnode.Stmt
	Cond    gnode.Expr
	Body    gnode.Stmts
	ElseIfs []*ElseIfClause
	Else    gnode.Stmts
}

func (s *IfStmt) Pos() source.Pos { return s.NodePos }
func (s *IfStmt) End() source.Pos { return s.NodeEnd }
func (s *IfStmt) StmtNode()       {}
func (s *IfStmt) String() string  { return "gadx.If" }

func (s *IfStmt) WriteCode(ctx *gnode.CodeWriteContext) {
	ctx.WriteString("if ")
	if s.Init != nil {
		ctx.WithoutPrefix().WriteStmts(s.Init)
		ctx.WriteString("; ")
	}
	s.Cond.WriteCode(ctx)
	ctx.WriteString(" {")
	ctx.WriteSemi()
	ctx.Depth++
	ctx.WriteStmts(s.Body...)
	ctx.Depth--
	for _, eif := range s.ElseIfs {
		ctx.WriteSemi()
		ctx.WriteString("} else if ")
		eif.Cond.WriteCode(ctx)
		ctx.WriteString(" {")
		ctx.WriteSemi()
		ctx.Depth++
		ctx.WriteStmts(eif.Body...)
		ctx.Depth--
	}
	if len(s.Else) > 0 {
		ctx.WriteSemi()
		ctx.WriteString("} else {")
		ctx.WriteSemi()
		ctx.Depth++
		ctx.WriteStmts(s.Else...)
		ctx.Depth--
	}
	ctx.WriteSemi()
	ctx.WriteString("}")
}

// =============================================================================
// ForStmt — loop block
// =============================================================================

type ForStmt struct {
	ast.NodeData
	NodePos source.Pos
	NodeEnd source.Pos
	Init    gnode.Stmt
	Cond    gnode.Expr
	Post    gnode.Stmt
	Body    gnode.Stmts
	Else    gnode.Stmts
}

func (s *ForStmt) Pos() source.Pos { return s.NodePos }
func (s *ForStmt) End() source.Pos { return s.NodeEnd }
func (s *ForStmt) StmtNode()       {}
func (s *ForStmt) String() string  { return "gadx.For" }

func (s *ForStmt) WriteCode(ctx *gnode.CodeWriteContext) {
	ctx.WriteString("for ")
	if s.Init != nil {
		ctx.WithoutPrefix().WriteStmts(s.Init)
		ctx.WriteString("; ")
	}
	s.Cond.WriteCode(ctx)
	if s.Post != nil {
		ctx.WriteString("; ")
		ctx.WithoutPrefix().WriteStmts(s.Post)
	}
	ctx.WriteString(" {")
	ctx.WriteSemi()
	ctx.Depth++
	ctx.WriteStmts(s.Body...)
	ctx.Depth--
	if len(s.Else) > 0 {
		ctx.WriteSemi()
		ctx.WriteString("} else {")
		ctx.WriteSemi()
		ctx.Depth++
		ctx.WriteStmts(s.Else...)
		ctx.Depth--
	}
	ctx.WriteSemi()
	ctx.WriteString("}")
}

// =============================================================================
// AssignStmt — variable assignment
// =============================================================================

type AssignStmt struct {
	ast.NodeData
	NodePos source.Pos
	NodeEnd source.Pos
	Op      string
	LHS     gnode.Expr
	RHS     gnode.Expr
}

func (s *AssignStmt) Pos() source.Pos { return s.NodePos }
func (s *AssignStmt) End() source.Pos { return s.NodeEnd }
func (s *AssignStmt) StmtNode()       {}
func (s *AssignStmt) String() string  { return "gadx.Assign" }

func (s *AssignStmt) WriteCode(ctx *gnode.CodeWriteContext) {
	s.LHS.WriteCode(ctx)
	ctx.WriteString(" " + s.Op + " ")
	s.RHS.WriteCode(ctx)
}

// =============================================================================
// CodeStmt — raw GAD code block
// =============================================================================

type CodeStmt struct {
	ast.NodeData
	NodePos   source.Pos
	NodeEnd   source.Pos
	Stmts     gnode.Stmts
	TrimLeft  bool
	TrimRight bool
}

func (c *CodeStmt) Pos() source.Pos { return c.NodePos }
func (c *CodeStmt) End() source.Pos { return c.NodeEnd }
func (c *CodeStmt) StmtNode()       {}
func (c *CodeStmt) String() string  { return "gadx.Code" }

func (c *CodeStmt) WriteCode(ctx *gnode.CodeWriteContext) {
	ctx.WriteStmts(c.Stmts...)
}

// =============================================================================
// FuncDecl — function definition
// =============================================================================

type FuncDecl struct {
	ast.NodeData
	NodePos source.Pos
	NodeEnd source.Pos
	Name    string
	// TypeParams and Return carry the optional `[T constraint, …]` type-parameter
	// list and the `<ret>` return type from the directive signature; nil when
	// absent. Params holds the positional/named parameters (with their types).
	TypeParams []*gnode.TypedIdentExpr
	Params     *gnode.FuncParams
	Return     []*gnode.TypedIdentExpr
	ParamsRaw  string
	Body       gnode.Stmts
	Exported   bool
	// Doc is the text of a `/** … */` doc comment immediately preceding the
	// declaration, or "".
	Doc string
}

func (f *FuncDecl) Pos() source.Pos { return f.NodePos }
func (f *FuncDecl) End() source.Pos { return f.NodeEnd }
func (f *FuncDecl) StmtNode()       {}
func (f *FuncDecl) String() string  { return fmt.Sprintf("gadx.Func(%s)", f.Name) }

func (f *FuncDecl) WriteCode(ctx *gnode.CodeWriteContext) {
	ctx.WriteString("const " + f.Name + " = func")
	ctx.WriteString(renderFuncParams(f.ParamsRaw, f.Params))
	ctx.WriteString(" {")
	ctx.WriteSemi()
	ctx.Depth++
	ctx.WriteStmts(f.Body...)
	ctx.Depth--
	ctx.WriteSemi()
	ctx.WriteString("}")
}

// =============================================================================
// CompDecl — component definition
// =============================================================================

type CompDecl struct {
	ast.NodeData
	NodePos source.Pos
	NodeEnd source.Pos
	Name    string
	ID      string
	// TypeParams and Return carry the optional `[T constraint, …]` type-parameter
	// list and the `<ret>` return type from the directive signature; nil when
	// absent.
	TypeParams []*gnode.TypedIdentExpr
	Params     *gnode.FuncParams
	Return     []*gnode.TypedIdentExpr
	ParamsRaw  string
	Body       gnode.Stmts
	Slots      []*SlotDecl
	Comps      []*CompDecl
	Exported   bool
	Main       bool
	// Doc is the text of a `/** … */` doc comment immediately preceding the
	// declaration, or "".
	Doc string
}

func (c *CompDecl) Pos() source.Pos { return c.NodePos }
func (c *CompDecl) End() source.Pos { return c.NodeEnd }
func (c *CompDecl) StmtNode()       {}
func (c *CompDecl) String() string  { return fmt.Sprintf("gadx.Comp(%s)", c.Name) }

func (c *CompDecl) WriteCode(ctx *gnode.CodeWriteContext) {
	ctx.WriteString("const " + c.ID + " = func")
	ctx.WriteString(renderFuncParams(c.ParamsRaw, c.Params, "slots={}"))
	ctx.WriteString(" {")
	ctx.WriteSemi()
	ctx.Depth++
	for _, comp := range c.Comps {
		ctx.WriteStmts(comp)
	}
	for _, slot := range c.Slots {
		ctx.WriteStmts(slot)
	}
	ctx.WriteStmts(c.Body...)
	ctx.Depth--
	ctx.WriteSemi()
	ctx.WriteString("}")
	if c.Exported {
		ctx.WriteSemi()
		ctx.WriteString("return {" + c.Name + ": " + c.ID + "}")
	}
}

// =============================================================================
// CompCallStmt — component call/invocation
// =============================================================================

type CompCallStmt struct {
	ast.NodeData
	NodePos  source.Pos
	NodeEnd  source.Pos
	Name     string
	Func     gnode.Expr
	Args     gnode.CallArgs
	SlotPass []*SlotPassStmt
	// InitStmts are call-scope `~` / `~~ … ~~` code statements from the call
	// block. They are emitted before the slot-pass declarations so a slot's
	// interpolated name (e.g. `@slot #"line[{index}]"`) and slot bodies can
	// reference values they declare.
	InitStmts gnode.Stmts
}

func (c *CompCallStmt) Pos() source.Pos { return c.NodePos }
func (c *CompCallStmt) End() source.Pos { return c.NodeEnd }
func (c *CompCallStmt) StmtNode()       {}
func (c *CompCallStmt) String() string  { return fmt.Sprintf("gadx.CompCall(%s)", c.Name) }

func (c *CompCallStmt) WriteCode(ctx *gnode.CodeWriteContext) {
	if c.Func != nil {
		c.Func.WriteCode(ctx)
	} else {
		ctx.WriteString(c.Name)
	}
	c.Args.WriteCode(ctx)
}

// =============================================================================
// SlotDecl — slot definition within a component
// =============================================================================

type SlotDecl struct {
	ast.NodeData
	NodePos source.Pos
	NodeEnd source.Pos
	Name    string
	ID      string
	// NameExpr, when set, is the interpolated slot name from `@slot "…"`. The
	// `slots` lookup then uses `slots[NameExpr]` and ID is a synthetic id for
	// the generated local variables.
	NameExpr gnode.Expr
	Scope    *gnode.FuncParams
	ScopeRaw string
	Body     gnode.Stmts
	Wrap     *WrapStmt
}

func (s *SlotDecl) Pos() source.Pos { return s.NodePos }
func (s *SlotDecl) End() source.Pos { return s.NodeEnd }
func (s *SlotDecl) StmtNode()       {}
func (s *SlotDecl) String() string  { return fmt.Sprintf("gadx.Slot(%s)", s.Name) }

func (s *SlotDecl) WriteCode(ctx *gnode.CodeWriteContext) {
	ctx.WriteString("const $slot$" + s.ID + "$ = func")
	ctx.WriteString(renderFuncParams(s.ScopeRaw, s.Scope))
	ctx.WriteString(" {")
	ctx.WriteSemi()
	ctx.Depth++
	ctx.WriteStmts(s.Body...)
	ctx.Depth--
	ctx.WriteSemi()
	ctx.WriteString("}")
	ctx.WriteSemi()
	ctx.WriteString("var $slot$" + s.ID + " = slots." + s.ID + " ?? $slot$" + s.ID + "$")
}

// =============================================================================
// SlotPassStmt — passing content to a component slot
// =============================================================================

type SlotPassStmt struct {
	ast.NodeData
	NodePos  source.Pos
	NodeEnd  source.Pos
	FuncType *gnode.FuncType
	Name     gnode.Expr
	// NameExpr, when set, is the interpolated slot name from `@slot #"…"`. It is
	// used as the `$$slots[NameExpr]` index in place of a static string.
	NameExpr gnode.Expr
	Body     gnode.Stmts
}

func (s *SlotPassStmt) Pos() source.Pos { return s.NodePos }
func (s *SlotPassStmt) End() source.Pos { return s.NodeEnd }
func (s *SlotPassStmt) StmtNode()       {}
func (s *SlotPassStmt) String() string  { return "gadx.SlotPass" }

func (s *SlotPassStmt) WriteCode(ctx *gnode.CodeWriteContext) {
	ctx.WriteString("const $slot = func")
	if s.FuncType != nil {
		ctx.WriteString(s.FuncType.Params.String())
	} else {
		ctx.WriteString("()")
	}
	ctx.WriteString(" {")
	ctx.WriteSemi()
	ctx.Depth++
	ctx.WriteStmts(s.Body...)
	ctx.Depth--
	ctx.WriteSemi()
	ctx.WriteString("}")
}

// =============================================================================
// WrapStmt — wraps slot content
// =============================================================================

type WrapStmt struct {
	ast.NodeData
	NodePos source.Pos
	NodeEnd source.Pos
	Body    gnode.Stmts
}

func (w *WrapStmt) Pos() source.Pos { return w.NodePos }
func (w *WrapStmt) End() source.Pos { return w.NodeEnd }
func (w *WrapStmt) StmtNode()       {}
func (w *WrapStmt) String() string  { return "gadx.Wrap" }

func (w *WrapStmt) WriteCode(ctx *gnode.CodeWriteContext) {
	ctx.WriteStmts(w.Body...)
}

// =============================================================================
// MatchStmt — match/case block (compiles to GAD match expression)
// =============================================================================

type CaseClause struct {
	Expr gnode.Expr
	Body gnode.Stmts
}

type MatchStmt struct {
	ast.NodeData
	NodePos source.Pos
	NodeEnd source.Pos
	Tag     gnode.Expr
	Cases   []*CaseClause
	Default gnode.Stmts
}

func (s *MatchStmt) Pos() source.Pos { return s.NodePos }
func (s *MatchStmt) End() source.Pos { return s.NodeEnd }
func (s *MatchStmt) StmtNode()       {}
func (s *MatchStmt) String() string  { return "gadx.Match" }

func (s *MatchStmt) WriteCode(ctx *gnode.CodeWriteContext) {
	switchMatchExpr(s).WriteCode(ctx)
}

// =============================================================================
// VarDecl — single variable declaration within @var
// =============================================================================

type VarDecl struct {
	Name string
	Init gnode.Expr
}

// VarStmt — @var declaration (compiles to Gad `var (...)` statement)
// =============================================================================

type VarStmt struct {
	ast.NodeData
	NodePos source.Pos
	NodeEnd source.Pos
	Decl    *gnode.GenDecl
	Decls   []VarDecl
	// Doc is the text of a `/** … */` doc comment immediately preceding the
	// declaration, or "".
	Doc string
}

func (s *VarStmt) Pos() source.Pos { return s.NodePos }
func (s *VarStmt) End() source.Pos { return s.NodeEnd }
func (s *VarStmt) StmtNode()       {}
func (s *VarStmt) String() string  { return "gadx.Var" }

func (s *VarStmt) WriteCode(ctx *gnode.CodeWriteContext) {
	if s.Decl != nil {
		s.Decl.WriteCode(ctx)
		return
	}
	ctx.WriteString("var (")
	for i, d := range s.Decls {
		if i > 0 {
			ctx.WriteString(", ")
		}
		ctx.WriteString(d.Name)
		if d.Init != nil {
			ctx.WriteString(" = ")
			d.Init.WriteCode(ctx)
		}
	}
	ctx.WriteString(")")
}

// ConstStmt — @const declaration (compiles to Gad `const (...)` statement)
// =============================================================================

type ConstStmt struct {
	ast.NodeData
	NodePos source.Pos
	NodeEnd source.Pos
	Decl    *gnode.GenDecl
	Decls   []VarDecl
	// Doc is the text of a `/** … */` doc comment immediately preceding the
	// declaration, or "".
	Doc string
}

func (s *ConstStmt) Pos() source.Pos { return s.NodePos }
func (s *ConstStmt) End() source.Pos { return s.NodeEnd }
func (s *ConstStmt) StmtNode()       {}
func (s *ConstStmt) String() string  { return "gadx.Const" }

func (s *ConstStmt) WriteCode(ctx *gnode.CodeWriteContext) {
	if s.Decl != nil {
		s.Decl.WriteCode(ctx)
		return
	}
	ctx.WriteString("const (")
	for i, d := range s.Decls {
		if i > 0 {
			ctx.WriteString(", ")
		}
		ctx.WriteString(d.Name)
		if d.Init != nil {
			ctx.WriteString(" = ")
			d.Init.WriteCode(ctx)
		}
	}
	ctx.WriteString(")")
}

// GlobalStmt — @global declaration (compiles to Gad `global (...)` statements)
// =============================================================================

type GlobalStmt struct {
	ast.NodeData
	NodePos source.Pos
	NodeEnd source.Pos
	Names   []string
	// Decl, when set, is a fully-formed Gad `global (…)` declaration (with
	// optional `= v` / `!?= v` defaults). It takes precedence over Names.
	Decl *gnode.GenDecl
}

func (s *GlobalStmt) Pos() source.Pos { return s.NodePos }
func (s *GlobalStmt) End() source.Pos { return s.NodeEnd }
func (s *GlobalStmt) StmtNode()       {}
func (s *GlobalStmt) String() string  { return "gadx.Global" }

func (s *GlobalStmt) WriteCode(ctx *gnode.CodeWriteContext) {
	ctx.WriteString("global (" + strings.Join(s.Names, ", ") + ")")
}

// ParamStmt — @param declaration (compiles to Gad `param (...)` statement),
// declaring the parameters the compiled template receives.
// =============================================================================

type ParamStmt struct {
	ast.NodeData
	NodePos source.Pos
	NodeEnd source.Pos
	// Decl is the fully-formed Gad `param (…)` declaration (positional, variadic
	// `*rest`, named after `;`, and named-variadic `**named`), parsed from the
	// directive body.
	Decl *gnode.GenDecl
	// Doc is the text of a `/** … */` doc comment immediately preceding the
	// declaration, or "".
	Doc string
}

func (s *ParamStmt) Pos() source.Pos { return s.NodePos }
func (s *ParamStmt) End() source.Pos { return s.NodeEnd }
func (s *ParamStmt) StmtNode()       {}
func (s *ParamStmt) String() string  { return "gadx.Param" }

func (s *ParamStmt) WriteCode(ctx *gnode.CodeWriteContext) {
	if s.Decl != nil {
		s.Decl.WriteCode(ctx)
	}
}

// EnumStmt — @enum declaration (compiles to Gad `enum IDENT { ... }` statement)
// =============================================================================

type EnumStmt struct {
	ast.NodeData
	NodePos source.Pos
	NodeEnd source.Pos
	Name    string
	// Decl is the fully-formed Gad enum statement (`enum Name { … }`), parsed
	// from the directive body.
	Decl *gnode.EnumStmt
	// Doc is the text of a `/** … */` doc comment immediately preceding the
	// declaration, or "".
	Doc string
}

func (s *EnumStmt) Pos() source.Pos { return s.NodePos }
func (s *EnumStmt) End() source.Pos { return s.NodeEnd }
func (s *EnumStmt) StmtNode()       {}
func (s *EnumStmt) String() string  { return fmt.Sprintf("gadx.Enum(%s)", s.Name) }

func (s *EnumStmt) WriteCode(ctx *gnode.CodeWriteContext) {
	if s.Decl != nil {
		s.Decl.WriteCode(ctx)
	}
}

// ExportStmt — export declaration
// =============================================================================

type ExportStmt struct {
	ast.NodeData
	NodePos source.Pos
	NodeEnd source.Pos
	Name    string
	Value   gnode.Expr
	// Doc is the text of a `/** … */` doc comment immediately preceding the
	// export, or "".
	Doc string
}

func (e *ExportStmt) Pos() source.Pos { return e.NodePos }
func (e *ExportStmt) End() source.Pos { return e.NodeEnd }
func (e *ExportStmt) StmtNode()       {}
func (e *ExportStmt) String() string  { return fmt.Sprintf("gadx.Export(%s)", e.Name) }

func (e *ExportStmt) WriteCode(ctx *gnode.CodeWriteContext) {
	ctx.WriteString("export " + e.Name)
	if e.Value != nil {
		ctx.WriteString(" = ")
		e.Value.WriteCode(ctx)
	}
}

// TestDecl — `@test NAME` block (lowers to Gad `test NAME { … }`)
// =============================================================================

type TestDecl struct {
	ast.NodeData
	NodePos source.Pos
	NodeEnd source.Pos
	Name    string // the test name (identifier spelling or string value)
	Quoted  bool   // NAME was written as a quoted string
	Body    gnode.Stmts
	Doc     string // lead `/** … */` doc comment text, or ""
}

func (t *TestDecl) Pos() source.Pos { return t.NodePos }
func (t *TestDecl) End() source.Pos { return t.NodeEnd }
func (t *TestDecl) StmtNode()       {}
func (t *TestDecl) String() string  { return fmt.Sprintf("gadx.Test(%s)", t.Name) }

func (t *TestDecl) WriteCode(ctx *gnode.CodeWriteContext) {
	ctx.WriteStmts(convertTestDecl(t)...)
}

// CallLineStmt — `! recv.method arg1 arg2 …` fluent call statement, lowering to
// `recv.method(arg1, arg2, …)`.
// =============================================================================

type CallLineStmt struct {
	ast.NodeData
	NodePos source.Pos
	NodeEnd source.Pos
	Callee  gnode.Expr   // the callable expression (e.g. `t.equal`, `myfunc`)
	Args    []gnode.Expr // the space-separated arguments
}

func (s *CallLineStmt) Pos() source.Pos { return s.NodePos }
func (s *CallLineStmt) End() source.Pos { return s.NodeEnd }
func (s *CallLineStmt) StmtNode()       {}
func (s *CallLineStmt) String() string  { return "gadx.Call" }

func (s *CallLineStmt) WriteCode(ctx *gnode.CodeWriteContext) {
	ctx.WriteStmts(convertCallLine(s)...)
}

// =============================================================================
// Helpers
// =============================================================================

func renderFuncParams(raw string, params *gnode.FuncParams, extraNamed ...string) string {
	parts := []string{}
	if raw = strings.TrimSpace(raw); raw != "" {
		parts = append(parts, raw)
	} else if params != nil {
		if rendered := strings.TrimSpace(strings.TrimPrefix(strings.TrimSuffix(params.String(), ")"), "(")); rendered != "" {
			parts = append(parts, rendered)
		}
	}
	parts = append(parts, extraNamed...)
	return "(" + strings.Join(parts, ", ") + ")"
}

func Quote(s string) string {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\', '"', '\n', '\r', '\t':
			return "`" + s + "`"
		}
	}
	return `"` + s + `"`
}

var (
	_ gnode.Stmt = (*TextStmt)(nil)
	_ gnode.Stmt = (*TagStmt)(nil)
	_ gnode.Stmt = (*DoctypeStmt)(nil)
	_ gnode.Stmt = (*CommentStmt)(nil)
	_ gnode.Stmt = (*IfStmt)(nil)
	_ gnode.Stmt = (*ForStmt)(nil)
	_ gnode.Stmt = (*AssignStmt)(nil)
	_ gnode.Stmt = (*CodeStmt)(nil)
	_ gnode.Stmt = (*FuncDecl)(nil)
	_ gnode.Stmt = (*CompDecl)(nil)
	_ gnode.Stmt = (*CompCallStmt)(nil)
	_ gnode.Stmt = (*SlotDecl)(nil)
	_ gnode.Stmt = (*SlotPassStmt)(nil)
	_ gnode.Stmt = (*WrapStmt)(nil)
	_ gnode.Stmt = (*MatchStmt)(nil)
	_ gnode.Stmt = (*VarStmt)(nil)
	_ gnode.Stmt = (*ConstStmt)(nil)
	_ gnode.Stmt = (*GlobalStmt)(nil)
	_ gnode.Stmt = (*ExportStmt)(nil)
)

var selfClosingTags = map[string]bool{
	"meta":   true,
	"img":    true,
	"link":   true,
	"input":  true,
	"source": true,
	"area":   true,
	"base":   true,
	"col":    true,
	"br":     true,
	"hr":     true,
}

func IsSelfClosing(name string) bool {
	return selfClosingTags[name]
}

func IsRawText(name string) bool {
	return name == "style" || name == "script"
}
