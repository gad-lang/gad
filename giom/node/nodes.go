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

func (f *File) String() string { return "giom.File" }

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
func (t *TextStmt) String() string  { return "giom.Text" }

func (t *TextStmt) WriteCode(ctx *gnode.CodeWriteContext) {
	ctx.WriteStmts(convertText(t)...)
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
func (t *TagStmt) String() string  { return fmt.Sprintf("giom.Tag(%s)", t.Name) }

func giomCallExpr(method string, pos source.Pos) *gnode.CallExpr {
	return gnode.ECall(gnode.ESelector(gnode.EIdent("giom", pos), gnode.Str(method, 0)), 0, 0)
}

func writeCallExpr(s string) *gnode.CallExpr {
	return writeCallExprs(rawStrExpr(s))
}

func rawStrExpr(s string) gnode.Expr {
	return gnode.EToRaw(0, gnode.Str(s, 0))
}

func writeCallExprs(expr ...gnode.Expr) *gnode.CallExpr {
	call := &gnode.CallExpr{Func: gnode.EIdent("write", 0)}
	call.Args.Values = expr
	return call
}

func (t *TagStmt) WriteCode(ctx *gnode.CodeWriteContext) {
	ctx.WriteStmts(convertTag(t)...)
}

func buildAttrsCall(attrs []*TagAttribute, pos ...source.Pos) gnode.Expr {
	call := giomCallExpr("attrs", 0)
	if len(pos) > 0 {
		if !call.LParen.IsValid() {
			call.LParen = pos[0]
		}
	}
	if len(pos) > 1 {
		if !call.RParen.IsValid() {
			call.RParen = pos[1]
		}
	}
	for _, attr := range attrs {
		if attr == nil {
			continue
		}
		if attr.Elements != nil {
			for _, el := range attr.Elements.Elements {
				if kv, ok := el.(*gnode.KeyValuePairLit); ok {
					addNamedArg(call, exprKeyName(kv.Key), kv.Value)
				}
			}
			continue
		}
		value := attr.Value
		if value == nil {
			if attr.IsFlag {
				value = gnode.Str(attr.Name, 0)
			} else {
				value = gnode.Str("", 0)
			}
		}
		addNamedArg(call, attr.Name, value)
	}
	writeCall := &gnode.CallExpr{Func: gnode.EIdent("write", 0)}
	writeCall.Args.Values = append(writeCall.Args.Values, call)
	return writeCall
}

func addNamedArg(call *gnode.CallExpr, name string, value gnode.Expr) {
	call.NamedArgs.AppendS(name, value)
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
// HtmlStmt — a raw HTML region (`<tag …>…</tag>` or `<>…</>` fragment)
// =============================================================================

type HtmlStmt struct {
	ast.NodeData
	NodePos source.Pos
	NodeEnd source.Pos
	// Stmts are the lowered Gad statements (write/giom.write/giom.attr calls)
	// produced from the HTML region.
	Stmts gnode.Stmts
}

func (h *HtmlStmt) Pos() source.Pos { return h.NodePos }
func (h *HtmlStmt) End() source.Pos { return h.NodeEnd }
func (h *HtmlStmt) StmtNode()       {}
func (h *HtmlStmt) String() string  { return "giom.Html" }

func (h *HtmlStmt) WriteCode(ctx *gnode.CodeWriteContext) {
	ctx.WriteStmts(convertHtml(h)...)
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
func (d *DoctypeStmt) String() string  { return fmt.Sprintf("giom.Doctype(%s)", d.Value) }

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
	Silent  bool
	Body    gnode.Stmts
}

func (c *CommentStmt) Pos() source.Pos { return c.NodePos }
func (c *CommentStmt) End() source.Pos { return c.NodeEnd }
func (c *CommentStmt) StmtNode()       {}
func (c *CommentStmt) String() string  { return "giom.Comment" }

func (c *CommentStmt) WriteCode(ctx *gnode.CodeWriteContext) {
	if c.Silent {
		return
	}
	writeRaw(ctx, "<!-- "+c.Text+" -->")
}

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
func (s *IfStmt) String() string  { return "giom.If" }

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
func (s *ForStmt) String() string  { return "giom.For" }

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
func (s *AssignStmt) String() string  { return "giom.Assign" }

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
func (c *CodeStmt) String() string  { return "giom.Code" }

func (c *CodeStmt) WriteCode(ctx *gnode.CodeWriteContext) {
	ctx.WriteStmts(c.Stmts...)
}

// =============================================================================
// FuncDecl — function definition
// =============================================================================

type FuncDecl struct {
	ast.NodeData
	NodePos   source.Pos
	NodeEnd   source.Pos
	Name      string
	Params    *gnode.FuncParams
	ParamsRaw string
	Body      gnode.Stmts
	Exported  bool
}

func (f *FuncDecl) Pos() source.Pos { return f.NodePos }
func (f *FuncDecl) End() source.Pos { return f.NodeEnd }
func (f *FuncDecl) StmtNode()       {}
func (f *FuncDecl) String() string  { return fmt.Sprintf("giom.Func(%s)", f.Name) }

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
	NodePos   source.Pos
	NodeEnd   source.Pos
	Name      string
	ID        string
	Params    *gnode.FuncParams
	ParamsRaw string
	Body      gnode.Stmts
	Slots     []*SlotDecl
	Comps     []*CompDecl
	Exported  bool
	Main      bool
}

func (c *CompDecl) Pos() source.Pos { return c.NodePos }
func (c *CompDecl) End() source.Pos { return c.NodeEnd }
func (c *CompDecl) StmtNode()       {}
func (c *CompDecl) String() string  { return fmt.Sprintf("giom.Comp(%s)", c.Name) }

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
	// interpolated name (e.g. `@slot #(line[{index}])`) and slot bodies can
	// reference values they declare.
	InitStmts gnode.Stmts
}

func (c *CompCallStmt) Pos() source.Pos { return c.NodePos }
func (c *CompCallStmt) End() source.Pos { return c.NodeEnd }
func (c *CompCallStmt) StmtNode()       {}
func (c *CompCallStmt) String() string  { return fmt.Sprintf("giom.CompCall(%s)", c.Name) }

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
	// NameExpr, when set, is the interpolated slot name from `@slot (…)`. The
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
func (s *SlotDecl) String() string  { return fmt.Sprintf("giom.Slot(%s)", s.Name) }

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
	// NameExpr, when set, is the interpolated slot name from `@slot #(…)`. It is
	// used as the `$$slots[NameExpr]` index in place of a static string.
	NameExpr gnode.Expr
	Body     gnode.Stmts
}

func (s *SlotPassStmt) Pos() source.Pos { return s.NodePos }
func (s *SlotPassStmt) End() source.Pos { return s.NodeEnd }
func (s *SlotPassStmt) StmtNode()       {}
func (s *SlotPassStmt) String() string  { return "giom.SlotPass" }

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
func (w *WrapStmt) String() string  { return "giom.Wrap" }

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
func (s *MatchStmt) String() string  { return "giom.Match" }

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
}

func (s *VarStmt) Pos() source.Pos { return s.NodePos }
func (s *VarStmt) End() source.Pos { return s.NodeEnd }
func (s *VarStmt) StmtNode()       {}
func (s *VarStmt) String() string  { return "giom.Var" }

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
}

func (s *ConstStmt) Pos() source.Pos { return s.NodePos }
func (s *ConstStmt) End() source.Pos { return s.NodeEnd }
func (s *ConstStmt) StmtNode()       {}
func (s *ConstStmt) String() string  { return "giom.Const" }

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
func (s *GlobalStmt) String() string  { return "giom.Global" }

func (s *GlobalStmt) WriteCode(ctx *gnode.CodeWriteContext) {
	ctx.WriteString("global (" + strings.Join(s.Names, ", ") + ")")
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
}

func (s *EnumStmt) Pos() source.Pos { return s.NodePos }
func (s *EnumStmt) End() source.Pos { return s.NodeEnd }
func (s *EnumStmt) StmtNode()       {}
func (s *EnumStmt) String() string  { return fmt.Sprintf("giom.Enum(%s)", s.Name) }

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
}

func (e *ExportStmt) Pos() source.Pos { return e.NodePos }
func (e *ExportStmt) End() source.Pos { return e.NodeEnd }
func (e *ExportStmt) StmtNode()       {}
func (e *ExportStmt) String() string  { return fmt.Sprintf("giom.Export(%s)", e.Name) }

func (e *ExportStmt) WriteCode(ctx *gnode.CodeWriteContext) {
	ctx.WriteString("export " + e.Name)
	if e.Value != nil {
		ctx.WriteString(" = ")
		e.Value.WriteCode(ctx)
	}
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

func writeRaw(ctx *gnode.CodeWriteContext, s string) {
	writeCall := &gnode.CallExpr{Func: gnode.EIdent("write", 0)}
	writeCall.Args.Values = append(writeCall.Args.Values, rawStrExpr(s))
	ctx.WriteStmts(gnode.SExpr(writeCall))
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
