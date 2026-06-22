package node

import (
	"bytes"
	"strings"

	"github.com/gad-lang/gad/parser/source"
)

// ArgsList represents a list of identifiers.
type ArgsList struct {
	Var    *TypedIdentExpr
	Values []*TypedIdentExpr
}

// Pos returns the position of first character belonging to the node.
func (n *ArgsList) Pos() source.Pos {
	if len(n.Values) > 0 {
		return n.Values[0].Pos()
	} else if n.Var != nil {
		return n.Var.Pos()
	}
	return source.NoPos
}

// End returns the position of first character immediately after the node.
func (n *ArgsList) End() source.Pos {
	if n.Var != nil {
		return n.Var.End()
	} else if l := len(n.Values); l > 0 {
		return n.Values[l-1].End()
	}
	return source.NoPos
}

func (n *ArgsList) PrependValue(v ...*TypedIdentExpr) {
	n.Values = append(v, n.Values...)
}

// NumFields returns the number of fields.
func (n *ArgsList) NumFields() int {
	if n == nil {
		return 0
	}
	return len(n.Values)
}

func (n *ArgsList) String() string {
	var list []string
	for _, e := range n.Values {
		list = append(list, e.String())
	}
	if n.Var != nil {
		list = append(list, "*"+n.Var.String())
	}
	return strings.Join(list, ", ")
}

// NamedArgsList represents a list of identifier with value pairs.
type NamedArgsList struct {
	Var    *IdentExpr
	Names  []*TypedIdentExpr
	Values []Expr
}

func (n *NamedArgsList) Add(name *TypedIdentExpr, value Expr) *NamedArgsList {
	n.Names = append(n.Names, name)
	n.Values = append(n.Values, value)
	return n
}

// Pos returns the position of first character belonging to the node.
func (n *NamedArgsList) Pos() source.Pos {
	if len(n.Names) > 0 {
		return n.Names[0].Pos()
	} else if n.Var != nil {
		return n.Var.Pos()
	}
	return source.NoPos
}

// End returns the position of first character immediately after the node.
func (n *NamedArgsList) End() source.Pos {
	if n.Var != nil {
		return n.Var.End()
	}
	if l := len(n.Names); l > 0 {
		if n.Var != nil {
			return n.Var.End()
		}
		return n.Values[l-1].End()
	}
	return source.NoPos
}

// NumFields returns the number of fields.
func (n *NamedArgsList) NumFields() int {
	if n == nil {
		return 0
	}
	i := len(n.Names)
	if n.Var != nil {
		i++
	}
	return i
}

func (n *NamedArgsList) IsZero() bool {
	return n.NumFields() == 0
}

func (n *NamedArgsList) String() string {
	var list []string
	for i, e := range n.Names {
		v := n.Values[i]
		if v == nil {
			list = append(list, e.String())
		} else {
			list = append(list, e.String()+"="+v.String())
		}
	}
	if n.Var != nil {
		list = append(list, "**"+n.Var.String())
	}
	return strings.Join(list, ", ")
}

// FuncParams represents a function paramsw.
type FuncParams struct {
	LParen    source.Pos
	Args      ArgsList
	NamedArgs NamedArgsList
	RParen    source.Pos
}

// Pos returns the position of first character belonging to the node.
func (n *FuncParams) Pos() (pos source.Pos) {
	if n.LParen.IsValid() {
		return n.LParen
	}
	if pos = n.Args.Pos(); pos != source.NoPos {
		return pos
	}
	if pos = n.NamedArgs.Pos(); pos != source.NoPos {
		return pos
	}
	return source.NoPos
}

// End returns the position of first character immediately after the node.
func (n *FuncParams) End() (pos source.Pos) {
	if n.RParen.IsValid() {
		return n.RParen + 1
	}
	if pos = n.NamedArgs.End(); pos != source.NoPos {
		return pos
	}
	if pos = n.Args.End(); pos != source.NoPos {
		return pos
	}
	return source.NoPos
}

func (n *FuncParams) String() string {
	buf := bytes.NewBufferString("(")
	buf.WriteString(n.Args.String())
	if !n.NamedArgs.IsZero() {
		buf.WriteString("; ")
	}
	buf.WriteString(n.NamedArgs.String())
	buf.WriteString(")")
	return buf.String()
}

// funcParamItem is one parameter: a column-aware writer plus whether it belongs
// to the named (`;`) section.
type funcParamItem struct {
	write func(ctx *CodeWriteContext)
	named bool
}

// items flattens the positional and named parameters into their writers. A
// typed param is written via TypedIdentExpr.WriteCode so its type union can wrap
// when it overflows the line.
func (n *FuncParams) items() []funcParamItem {
	var items []funcParamItem
	for _, e := range n.Args.Values {
		e := e
		items = append(items, funcParamItem{write: e.WriteCode})
	}
	if n.Args.Var != nil {
		e := n.Args.Var
		items = append(items, funcParamItem{write: func(ctx *CodeWriteContext) {
			ctx.WriteString("*")
			e.WriteCode(ctx)
		}})
	}
	for i, e := range n.NamedArgs.Names {
		e, v := e, n.NamedArgs.Values[i]
		items = append(items, funcParamItem{named: true, write: func(ctx *CodeWriteContext) {
			e.WriteCode(ctx)
			if v != nil {
				ctx.WriteString("=", v.String())
			}
		}})
	}
	if n.NamedArgs.Var != nil {
		v := n.NamedArgs.Var.String()
		items = append(items, funcParamItem{named: true, write: func(ctx *CodeWriteContext) {
			ctx.WriteString("**", v)
		}})
	}
	return items
}

// WriteCode renders the parameter list, wrapping one parameter per line when
// NEW_LINE_CALC decides it overflows (no comma between wrapped items). The
// named section is introduced by `;`; a typed param keeps its ident and type on
// one line (each item is a single rendered string).
func (n *FuncParams) WriteCode(ctx *CodeWriteContext) {
	items := n.items()

	ctx.WriteSingleByte('(')

	if len(items) <= 1 {
		for _, it := range items {
			if it.named {
				ctx.WriteString("; ")
			}
			it.write(ctx)
		}
		ctx.WriteSingleByte(')')
		return
	}

	firstNamed := -1
	for i, it := range items {
		if it.named {
			firstNamed = i
			break
		}
	}

	sep := func(i int) {
		switch {
		case i == 0:
			if i == firstNamed {
				ctx.WriteString("; ")
			}
		case i == firstNamed:
			ctx.WriteString("; ")
		default:
			ctx.WriteString(", ")
		}
	}

	inNewLine := ctx.DecideNewLineFunc(
		CodeWriteContextFlagFormatCallParamsInNewLine, len(items), 1, func() {
			for i, it := range items {
				sep(i)
				it.write(ctx)
			}
		})

	switch {
	case inNewLine && ctx.Flags.Has(CodeWriteContextFlagFormatNewLineCalc):
		// greedy: pack params per line, no comma at a break, no extra indent.
		writeGreedyParams(ctx, items)
	case inNewLine:
		ctx.Depth++
		for i, it := range items {
			ctx.WriteSecondLine()
			ctx.WritePrefix()
			if it.named && i == firstNamed {
				ctx.WriteString("; ")
			}
			it.write(ctx)
		}
		ctx.WriteSecondLine()
		ctx.Depth--
		ctx.WritePrefix()
	default:
		for i, it := range items {
			sep(i)
			it.write(ctx)
		}
	}
	ctx.WriteSingleByte(')')
}

// writeGreedyParams renders parameter/argument items greedily inside an
// already-open paren (the caller wrote `(`): it opens an indented line, packs
// items onto each line and breaks to a new line only when the next item would
// overflow (no comma at the break, no extra indent), introducing the named
// section with `; `, then returns the cursor to the closing-paren prefix.
func writeGreedyParams(ctx *CodeWriteContext, items []funcParamItem) {
	firstNamed := -1
	for i, it := range items {
		if it.named {
			firstNamed = i
			break
		}
	}

	ctx.Depth++
	ctx.WriteSecondLine()
	ctx.WritePrefix()
	for i, it := range items {
		boundary := i == firstNamed
		if i == 0 {
			if boundary {
				ctx.WriteString("; ")
			}
			it.write(ctx)
			continue
		}
		inlineSep := ", "
		if boundary {
			inlineSep = "; "
		}
		w, _ := ctx.measure(0, func() { it.write(ctx) })
		if ctx.Column()+len(inlineSep)+w > ctx.maxColumns() {
			ctx.WriteSecondLine()
			ctx.WritePrefix()
			if boundary {
				ctx.WriteString("; ")
			}
			it.write(ctx)
		} else {
			ctx.WriteString(inlineSep)
			it.write(ctx)
		}
	}
	ctx.WriteSecondLine()
	ctx.Depth--
	ctx.WritePrefix()
}

func (n *FuncParams) Caller() (c *CallArgs) {
	c = &CallArgs{}
	for _, value := range n.Args.Values {
		c.Args.Values = append(c.Args.Values, value.Ident)
	}
	if n.Args.Var != nil {
		c.Args.Var = &ArgVarLit{Value: n.Args.Var}
	}

	for i, name := range n.NamedArgs.Names {
		c.NamedArgs.Names = append(c.NamedArgs.Names, &NamedArgExpr{Ident: name.Ident})
		c.NamedArgs.Values = append(c.NamedArgs.Values, n.NamedArgs.Values[i])
	}

	if n.NamedArgs.Var != nil {
		c.NamedArgs.Names = append(c.NamedArgs.Names, &NamedArgExpr{Ident: n.NamedArgs.Var, Var: true})
		c.NamedArgs.Values = append(c.NamedArgs.Values, nil)
	}
	return
}

func (n FuncParams) WithNamedValuesNil() (c *FuncParams) {
	c = &n

	c.NamedArgs.Values = append([]Expr{}, n.NamedArgs.Values...)
	for i := range c.NamedArgs.Values {
		c.NamedArgs.Values[i] = &NilLit{}
	}
	return
}

// FuncHeader is the shared shape of a function/method signature: an optional
// name, a parameter list and an optional return-type list. It is embedded in
// FuncType and FuncHeaderExpr.
type FuncHeader struct {
	NameExpr Expr
	Params   FuncParams
	Return   []*TypedIdentExpr
}

// Pos returns the position of first character belonging to the node.
func (e *FuncHeader) Pos() source.Pos {
	if e.NameExpr != nil {
		return e.NameExpr.Pos()
	}
	if p := e.Params.Pos(); p != source.NoPos {
		return p
	}
	if len(e.Return) > 0 {
		return e.Return[0].Pos()
	}
	return source.NoPos
}

// End returns the position of first character immediately after the node.
func (e *FuncHeader) End() source.Pos {
	if n := len(e.Return); n > 0 {
		return e.Return[n-1].End()
	}
	if p := e.Params.End(); p != source.NoPos {
		return p
	}
	if e.NameExpr != nil {
		return e.NameExpr.End()
	}
	return source.NoPos
}

func (e *FuncHeader) NameIdent() *IdentExpr {
	if e.NameExpr == nil {
		return nil
	}
	return IdentOfSelector(e.NameExpr)
}

func (e *FuncHeader) Name() string {
	if e.NameExpr == nil {
		return ""
	}
	switch t := e.NameExpr.(type) {
	case *IdentExpr:
		return t.Name
	case *IndexExpr:
		switch it := t.Index.(type) {
		case *StrLit:
			return it.Value()
		}
	case *SelectorExpr:
		switch it := t.Sel.(type) {
		case *IdentExpr:
			return it.Name
		}
	}

	return ""
}

func (e *FuncHeader) String() string {
	var s string
	if e.NameExpr != nil {
		s = e.NameExpr.String()
	}
	s += e.Params.String()
	s += FormatFuncReturn(e.Return)
	return s
}

// WriteCode renders the header (name + params + return) routing the parameter
// list through FuncParams.WriteCode so it participates in formatting.
func (e *FuncHeader) WriteCode(ctx *CodeWriteContext) {
	if e.NameExpr != nil {
		ctx.WriteString(e.NameExpr.String())
	}
	e.Params.WriteCode(ctx)
	WriteFuncReturn(ctx, e.Return)
}

// FuncType represents a function type definition.
type FuncType struct {
	Token   TokenLit
	FuncPos source.Pos
	FuncHeader
}

func (e *FuncType) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *FuncType) Pos() source.Pos {
	return e.FuncPos
}

// End returns the position of first character immediately after the node.
func (e *FuncType) End() source.Pos {
	return e.Params.End()
}

// FuncHeaderExpr is a function-header value expression written between angle
// brackets: `<()>`, `<(v int)>`, `<(v int) <x uint|int>>`. It evaluates to a
// FunctionHeader value describing a signature.
type FuncHeaderExpr struct {
	OpenPos  source.Pos // `<`
	ClosePos source.Pos // `>`
	FuncHeader
}

func (e *FuncHeaderExpr) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *FuncHeaderExpr) Pos() source.Pos {
	if e.OpenPos != source.NoPos {
		return e.OpenPos
	}
	return e.FuncHeader.Pos()
}

// End returns the position of first character immediately after the node.
func (e *FuncHeaderExpr) End() source.Pos {
	if e.ClosePos != source.NoPos {
		return e.ClosePos + 1
	}
	return e.FuncHeader.End()
}

func (e *FuncHeaderExpr) String() string {
	return "<" + e.FuncHeader.String() + ">"
}

func (e *FuncHeaderExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(e.String())
}

// FormatFuncReturn renders an optional function return-type list as
// " <T1, T2, ...>". It returns an empty string when there are no return types.
func FormatFuncReturn(ret []*TypedIdentExpr) string {
	if len(ret) == 0 {
		return ""
	}
	s := make([]string, len(ret))
	for i, t := range ret {
		s[i] = t.String()
	}
	return " <" + strings.Join(s, ", ") + ">"
}

// WriteFuncReturn is the WriteCode counterpart of FormatFuncReturn: it renders
// the return-type list applying the union spacing rule (` | ` around each `|`),
// inline within the `< >`.
func WriteFuncReturn(ctx *CodeWriteContext, ret []*TypedIdentExpr) {
	if len(ret) == 0 {
		return
	}
	ctx.WriteString(" <")
	for i, t := range ret {
		if i > 0 {
			ctx.WriteString(", ")
		}
		if len(t.Type) == 0 {
			ctx.WriteString(t.String())
		} else {
			t.writeInlineUnion(ctx)
		}
	}
	ctx.WriteString(">")
}
