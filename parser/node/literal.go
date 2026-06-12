package node

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/quote"
	"github.com/gad-lang/gad/runehelper"
	"github.com/gad-lang/gad/token"
	"github.com/shopspring/decimal"
)

// IntLit represents an integer literal.
type IntLit struct {
	Value    int64
	ValuePos source.Pos
	Literal  string
}

func (e *IntLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *IntLit) Pos() source.Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *IntLit) End() source.Pos {
	return source.Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *IntLit) String() string {
	return e.Literal
}

func (e *IntLit) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(e.Literal)
}

// UintLit represents an unsigned integer literal.
type UintLit struct {
	Value    uint64
	ValuePos source.Pos
	Literal  string
}

func (e *UintLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *UintLit) Pos() source.Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *UintLit) End() source.Pos {
	return source.Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *UintLit) String() string {
	return e.Literal
}

func (e *UintLit) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(e.Literal)
}

// FloatLit represents a floating point literal.
type FloatLit struct {
	Value    float64
	ValuePos source.Pos
	Literal  string
}

func (e *FloatLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *FloatLit) Pos() source.Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *FloatLit) End() source.Pos {
	return source.Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *FloatLit) String() string {
	return e.Literal
}

func (e *FloatLit) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(e.Literal)
}

// DecimalLit represents a floating point literal.
type DecimalLit struct {
	Value    decimal.Decimal
	ValuePos source.Pos
	Literal  string
}

func (e *DecimalLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *DecimalLit) Pos() source.Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *DecimalLit) End() source.Pos {
	return source.Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *DecimalLit) String() string {
	return e.Literal
}

func (e *DecimalLit) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(e.Literal)
}

// BoolLit represents a boolean literal.
type BoolLit struct {
	Value    bool
	ValuePos source.Pos
	Literal  string
}

func (e *BoolLit) ExprNode() {}

func (e *BoolLit) Bool() bool {
	return e.Value
}

// Pos returns the position of first character belonging to the node.
func (e *BoolLit) Pos() source.Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *BoolLit) End() source.Pos {
	return source.Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *BoolLit) String() string {
	return e.Literal
}

func (e *BoolLit) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(e.Literal)
}

// FlagLit represents a yes literal.
type FlagLit struct {
	ValuePos source.Pos
	Literal  string
	Value    bool
}

func (e *FlagLit) ExprNode() {}

func (e *FlagLit) Bool() bool {
	return e.Value
}

// Pos returns the position of first character belonging to the node.
func (e *FlagLit) Pos() source.Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *FlagLit) End() source.Pos {
	return source.Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *FlagLit) String() string {
	if e.Value {
		return "yes"
	}
	return "no"
}

func (e *FlagLit) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(e.Literal)
}

// CharLit represents a character literal.
type CharLit struct {
	Value    rune
	ValuePos source.Pos
	Literal  string
}

func (e *CharLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *CharLit) Pos() source.Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *CharLit) End() source.Pos {
	return source.Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *CharLit) String() string {
	return e.Literal
}

func (e *CharLit) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(e.Literal)
}

// DictElementLit represents a map element.
type DictElementLit struct {
	Key      Expr
	ColonPos source.Pos
	Value    Expr
}

func (e *DictElementLit) Func() (f *FuncDefLit) {
	switch t := e.Value.(type) {
	case *FuncDefLit:
		return t
	case *ClosureExpr:
		return &FuncDefLit{
			Expr: &ClosureExpr{
				Params: t.Params,
				Return: t.Return,
				Lambda: Token{Token: token.Colon},
				Body:   t.Body,
			},
		}
	case *FuncExpr:
		if t.BodyExpr != nil {
			return &FuncDefLit{
				Expr: &ClosureExpr{
					Params: t.Type.Params,
					Return: t.Type.Return,
					Lambda: Token{Token: token.Colon},
					Body:   t.BodyExpr,
				},
			}
		}
		return &FuncDefLit{
			Expr: &FuncExpr{
				Type: &FuncType{
					Params: t.Type.Params,
					Return: t.Type.Return,
				},
				Body:     t.Body,
				BodyExpr: t.BodyExpr,
			},
		}
	}
	return
}

func (e *DictElementLit) IsFunc() (ok bool) {
	return e.Func() != nil
}

func (e *DictElementLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *DictElementLit) Pos() source.Pos {
	return e.Key.Pos()
}

// End returns the position of first character immediately after the node.
func (e *DictElementLit) End() source.Pos {
	return e.Value.End()
}

func (e *DictElementLit) BuildKeyExpr() Expr {
	switch t := e.Key.(type) {
	case *RawStrLit:
		return Str(t.Value(), e.Key.Pos())
	case *RawHeredocLit:
		return Str(t.Value(), e.Key.Pos())
	case *HeredocLit:
		return Str(t.Value(), e.Key.Pos())
	case *SymbolLit:
		return Str(t.Value(), e.Key.Pos())
	case *DecimalLit:
		return Str(t.Value.String(), e.Key.Pos())
	case *IdentExpr, *IntLit, *UintLit, *FloatLit, *NilLit, *BoolLit, *FlagLit:
		return Str(t.String(), e.Key.Pos())
	default:
		return t
	}
}

func (e *DictElementLit) String() string {
	var (
		f = e.Func()
		v = e.Value
		b strings.Builder
	)

	switch t := e.Key.(type) {
	case *IdentExpr, *IntLit, *UintLit, *FloatLit, *DecimalLit, *SymbolLit,
		*ParenExpr:
		b.WriteString(e.Key.String())
	case *StrLit:
		if v := t.Value(); runehelper.IsIdentifierOrDigitRunes([]rune(v)) {
			b.WriteString(v)
		} else {
			b.WriteString(t.String())
		}
	case *RawStrLit:
		if v := t.Value(); runehelper.IsIdentifierOrDigitRunes([]rune(v)) {
			b.WriteString(v)
		} else {
			b.WriteString(t.String())
		}
	case *RawHeredocLit:
		if v := t.Value(); runehelper.IsIdentifierOrDigitRunes([]rune(v)) {
			b.WriteString(v)
		} else {
			b.WriteString(t.String())
		}
	case *HeredocLit:
		if v := t.Value(); runehelper.IsIdentifierOrDigitRunes([]rune(v)) {
			b.WriteString(v)
		} else {
			b.WriteString(t.String())
		}
	case *NilLit, *BoolLit, *FlagLit:
		b.WriteString(t.String())
	}

	if f == nil {
		b.WriteString(": ")
		b.WriteString(v.String())
	} else {
		b.WriteString(f.String())
	}

	return b.String()
}

func (e *DictElementLit) WriteCode(ctx *CodeWriteContext) {
	var (
		fun = e.Func()
		sep string
	)
	if fun == nil {
		sep = ": "
	}

	e.Key.WriteCode(ctx)
	ctx.WriteString(sep)

	if fun != nil {
		fun.WriteCode(ctx)
	} else {
		e.Value.WriteCode(ctx)
	}
}

type DictElementFuncExprs []*FuncDefLit

func (l DictElementFuncExprs) Sort() {
	sort.Slice(l, func(i, j int) bool {
		mi := l[i]
		mj := l[j]

		ti := mi.Params()
		tj := mj.Params()

		if li, lj := len(ti.Args.Values), len(tj.Args.Values); li < lj {
			return true
		} else if li > lj {
			return false
		}

		for i, value := range ti.Args.Values {
			if value.Ident.Name < tj.Args.Values[i].Ident.Name {
				return true
			}
		}

		return false
	})
}

type FuncDefLit struct {
	Expr
}

func (e *FuncDefLit) Closure() (c *ClosureExpr) {
	c, _ = e.Expr.(*ClosureExpr)
	return
}

func (e *FuncDefLit) Params() *FuncParams {
	switch t := e.Expr.(type) {
	case *ClosureExpr:
		return &t.Params
	case *FuncExpr:
		return &t.Type.Params
	default:
		return nil
	}
}

func (e *FuncDefLit) Func() (f *FuncExpr) {
	f, _ = e.Expr.(*FuncExpr)
	return
}

// StrLit represents a string literal.
type StrLit struct {
	ValuePos source.Pos
	Literal  string
}

func (e *StrLit) Value() string {
	var lit = e.Literal
	if len(lit) > 0 && lit[0] == '\'' {
		lit = `"` + strings.ReplaceAll(strings.ReplaceAll(lit[1:len(lit)-1], "\\'", "'"), `"`, `\"`) + `"`
	}
	v, err := strconv.Unquote(lit)
	if err != nil {
		panic(fmt.Sprintf("StrLit can not unquote: %v", err))
	}
	return v
}

func (e *StrLit) CanIdent() bool {
	return runehelper.IsIdentifierRunes([]rune(e.Value()))
}

func (e *StrLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *StrLit) Pos() source.Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *StrLit) End() source.Pos {
	return source.Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *StrLit) String() string {
	return e.Literal
}

func (e *StrLit) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(e.Literal)
}

type RawStrLit struct {
	Literal    string
	LiteralPos source.Pos
	Quoted     bool
}

func (e *RawStrLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *RawStrLit) Pos() source.Pos {
	return e.LiteralPos
}

// End returns the position of first character immediately after the node.
func (e *RawStrLit) End() source.Pos {
	return source.Pos(int(e.LiteralPos) + len(e.Literal))
}

func (e *RawStrLit) String() string {
	return e.QuotedValue()
}

func (e *RawStrLit) Value() string {
	if e.Quoted {
		s, _ := strconv.Unquote(e.Literal)
		return s
	} else {
		return e.Literal
	}
}

func (e *RawStrLit) QuotedValue() string {
	if e.Quoted {
		return e.Literal
	} else {
		return quote.Quote(e.Literal, "`")
	}
}

func (e *RawStrLit) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(e.QuotedValue())
}

// NilLit represents an nil literal.
type NilLit struct {
	TokenPos source.Pos
}

func (e *NilLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *NilLit) Pos() source.Pos {
	return e.TokenPos
}

// End returns the position of first character immediately after the node.
func (e *NilLit) End() source.Pos {
	return e.TokenPos + 9 // len(nil) == 9
}

func (e *NilLit) String() string {
	return "nil"
}

func (e *NilLit) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString("nil")
}

// KeyValueLit represents a key value element.
type KeyValueLit struct {
	Key   Expr
	Value Expr
}

func (e *KeyValueLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *KeyValueLit) Pos() source.Pos {
	return e.Key.Pos()
}

// End returns the position of first character immediately after the node.
func (e *KeyValueLit) End() source.Pos {
	if e.Value == nil {
		return e.Key.End()
	}
	return e.Value.End()
}

func (e *KeyValueLit) ElementString() string {
	if e.Value == nil {
		return e.Key.String()
	}
	return e.Key.String() + "=" + e.Value.String()
}

func (e *KeyValueLit) String() string {
	if e.Value == nil {
		return e.Key.String()
	}
	return "[" + e.ElementString() + "]"
}

func (e *KeyValueLit) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteSingleByte('[')
	e.Key.WriteCode(ctx)
	ctx.WriteSingleByte('=')
	e.Value.WriteCode(ctx)
	ctx.WriteSingleByte(']')
}

// KeyValuePairLit represents a key value pair element.
type KeyValuePairLit struct {
	Key   Expr
	Value Expr
}

func (e *KeyValuePairLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *KeyValuePairLit) Pos() source.Pos {
	return e.Key.Pos()
}

// End returns the position of first character immediately after the node.
func (e *KeyValuePairLit) End() source.Pos {
	if e.Value == nil {
		return e.Key.End()
	}
	return e.Value.End()
}

func (e *KeyValuePairLit) Func() (f *FuncDefLit) {
	switch t := e.Value.(type) {
	case *FuncDefLit:
		return t
	case *ClosureExpr:
		return &FuncDefLit{
			Expr: &ClosureExpr{
				Params: t.Params,
				Return: t.Return,
				Lambda: Token{Token: token.Colon},
				Body:   t.Body,
			},
		}
	case *FuncExpr:
		if t.BodyExpr != nil {
			return &FuncDefLit{
				Expr: &ClosureExpr{
					Params: t.Type.Params,
					Return: t.Type.Return,
					Lambda: Token{Token: token.Colon},
					Body:   t.BodyExpr,
				},
			}
		}
		return &FuncDefLit{
			Expr: &FuncExpr{
				Type: &FuncType{
					Params: t.Type.Params,
					Return: t.Type.Return,
				},
				Body:     t.Body,
				BodyExpr: t.BodyExpr,
			},
		}

	}
	return
}

func (e *KeyValuePairLit) IsFunc() (ok bool) {
	return e.Func() != nil
}

func (e *KeyValuePairLit) String() string {
	if e.Value == nil {
		return e.Key.String()
	}

	var (
		sep string
		f   = e.Func()
		v   = e.Value
	)

	if f == nil {
		sep = "="
	} else {
		v = f
	}

	return e.Key.String() + sep + v.String()
}

func (e *KeyValuePairLit) WriteCode(ctx *CodeWriteContext) {
	e.Key.WriteCode(ctx)
	if e.Value == nil {
		return
	}

	fun := e.Func()

	if fun == nil {
		if fwm, _ := e.Value.(*FuncWithMethodsExpr); fwm != nil {
			ctx.WriteString(" ")
			fwm.WriteCode(ctx)
			return
		} else {
			ctx.WriteString("=")
		}
	}

	if fun != nil {
		fun.WriteCode(ctx)
	} else {
		e.Value.WriteCode(ctx)
	}
}

// KeyValueArrayLit represents a key value array literal.
type KeyValueArrayLit struct {
	LParen   source.Pos
	Elements Exprs
	RParen   source.Pos
}

func (e *KeyValueArrayLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *KeyValueArrayLit) Pos() source.Pos {
	return e.LParen
}

// End returns the position of first character immediately after the node.
func (e *KeyValueArrayLit) End() source.Pos {
	return e.RParen + 1
}

func (e *KeyValueArrayLit) String() string {
	var elements []string
	for _, m := range e.Elements {
		elements = append(elements, m.String())
	}
	return "(;" + strings.Join(elements, ", ") + ")"
}

func (e *KeyValueArrayLit) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString("(;")
	if l := len(e.Elements); l > 0 {
		if l == 1 || !ctx.HasPrefix() {
			ctx.WriteSingleByte(' ')
		}
		if l == 1 {
			e.Elements[0].WriteCode(ctx)
		} else {
			ctx.WriteItems(
				ctx.Flags.Has(CodeWriteContextFlagFormatKeyValueArrayItemInNewLine),
				len(e.Elements),
				func(i int) {
					e.Elements[i].WriteCode(ctx)
				},
				func(nl bool) {
					if nl {
						if ctx.HasPrefix() {
							ctx.WriteSecondLine()
						} else {
							ctx.WriteLine(", ")
						}
					}
				})
		}
	}
	ctx.WriteSingleByte(')')
}

func (e *KeyValueArrayLit) ToMultiParenExpr() *MultiParenExpr {
	return &MultiParenExpr{
		LParen:        Token{Token: token.LParen, Pos: e.LParen},
		RParen:        Token{Token: token.RParen, Pos: e.RParen},
		NamedElements: e.Elements,
	}
}

// StdInLit represents an STDIN literal.
type StdInLit struct {
	TokenPos source.Pos
}

func (e *StdInLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *StdInLit) Pos() source.Pos {
	return e.TokenPos
}

// End returns the position of first character immediately after the node.
func (e *StdInLit) End() source.Pos {
	return e.TokenPos + 5 // len(STDIN) == 5
}

func (e *StdInLit) String() string {
	return "STDIN"
}

func (e *StdInLit) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(e.String())
}

// StdOutLit represents an STDOUT literal.
type StdOutLit struct {
	TokenPos source.Pos
}

func (e *StdOutLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *StdOutLit) Pos() source.Pos {
	return e.TokenPos
}

// End returns the position of first character immediately after the node.
func (e *StdOutLit) End() source.Pos {
	return e.TokenPos + 6 // len(STDOUT) == 6
}

func (e *StdOutLit) String() string {
	return "STDOUT"
}

func (e *StdOutLit) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(e.String())
}

// StdErrLit represents an STDERR literal.
type StdErrLit struct {
	TokenPos source.Pos
}

func (e *StdErrLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *StdErrLit) Pos() source.Pos {
	return e.TokenPos
}

// End StdErrLit the position of first character immediately after the node.
func (e *StdErrLit) End() source.Pos {
	return e.TokenPos + 6 // len(STDERR) == 6
}

func (e *StdErrLit) String() string {
	return "STDERR"
}

func (e *StdErrLit) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(e.String())
}

// ArgVarLit represents an variadic of argument.
type ArgVarLit struct {
	TokenPos source.Pos
	Value    Expr
}

func (e *ArgVarLit) ExprNode() {}

func (e *ArgVarLit) Pos() source.Pos {
	return e.TokenPos
}

func (e *ArgVarLit) End() source.Pos {
	return e.Value.End() + 6
}

func (e *ArgVarLit) String() string {
	return "*" + e.Value.String()
}

func (e *ArgVarLit) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteSingleByte('*')
	e.Value.WriteCode(ctx)
}

// NamedArgVarLit represents an variadic of named argument.
type NamedArgVarLit struct {
	TokenPos source.Pos
	Value    Expr
}

func (e *NamedArgVarLit) ExprNode() {}

func (e *NamedArgVarLit) Pos() source.Pos {
	return e.TokenPos
}

func (e *NamedArgVarLit) End() source.Pos {
	return e.Value.End() + 6
}

func (e *NamedArgVarLit) String() string {
	return "**" + e.Value.String()
}

func (e *NamedArgVarLit) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString("**")
	e.Value.WriteCode(ctx)
}

// DotFileNameLit represents an @name literal.
type DotFileNameLit struct {
	TokenPos source.Pos
}

func (e *DotFileNameLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *DotFileNameLit) Pos() source.Pos {
	return e.TokenPos
}

// End DotFileNameLit the position of first character immediately after the node.
func (e *DotFileNameLit) End() source.Pos {
	return e.TokenPos + +source.Pos(len(e.String()))
}

func (e *DotFileNameLit) String() string {
	return token.DotName.String()
}

func (e *DotFileNameLit) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(e.String())
}

// DotFileLit represents an @name literal.
type DotFileLit struct {
	TokenPos source.Pos
}

func (e *DotFileLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *DotFileLit) Pos() source.Pos {
	return e.TokenPos
}

// End DotFileLit the position of first character immediately after the node.
func (e *DotFileLit) End() source.Pos {
	return e.TokenPos + +source.Pos(len(e.String()))
}

func (e *DotFileLit) String() string {
	return token.DotFile.String()
}

func (e *DotFileLit) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(e.String())
}

// IsMainLit represents an @main literal.
type IsMainLit struct {
	TokenPos source.Pos
}

func (e *IsMainLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *IsMainLit) Pos() source.Pos {
	return e.TokenPos
}

// End IsMainLit the position of first character immediately after the node.
func (e *IsMainLit) End() source.Pos {
	return e.TokenPos + source.Pos(len(e.String()))
}

func (e *IsMainLit) String() string {
	return token.IsMain.String()
}

func (e *IsMainLit) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(e.String())
}

// ModuleLit represents an @module literal.
type ModuleLit struct {
	TokenPos source.Pos
}

func (e *ModuleLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *ModuleLit) Pos() source.Pos {
	return e.TokenPos
}

// End IsMainLit the position of first character immediately after the node.
func (e *ModuleLit) End() source.Pos {
	return e.TokenPos + source.Pos(len(e.String()))
}

func (e *ModuleLit) String() string {
	return token.Module.String()
}

func (e *ModuleLit) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(e.String())
}

type RawHeredocLit struct {
	Literal    string
	LiteralPos source.Pos
}

func (e *RawHeredocLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *RawHeredocLit) Pos() source.Pos {
	return e.LiteralPos
}

// End returns the position of first character immediately after the node.
func (e *RawHeredocLit) End() source.Pos {
	return source.Pos(int(e.LiteralPos) + len(e.Literal))
}

// backticks returns the number of leading backtick fence characters.
func (e *RawHeredocLit) backticks() int {
	n := 0
	for n < len(e.Literal) && e.Literal[n] == '`' {
		n++
	}
	return n
}

// contentOffset returns the byte offset within Literal at which RawContent
// begins: past the leading backticks and, for the common multiline form, the
// newline that ends the opening fence line.
func (e *RawHeredocLit) contentOffset() int {
	n := e.backticks()
	if n < len(e.Literal) && e.Literal[n] == '\n' {
		return n + 1
	}
	return n
}

// ContentPos returns the source position of the first byte of RawContent.
func (e *RawHeredocLit) ContentPos() source.Pos {
	return source.Pos(int(e.LiteralPos) + e.contentOffset())
}

// RawContent returns the heredoc body with the surrounding backtick fences, the
// opening fence line and the closing line removed, but with interior
// indentation preserved. Unlike Value it keeps a 1:1 byte correspondence with
// the original source starting at ContentPos, so it is what gets parsed for
// template interpolation so positions map back to the source.
func (e *RawHeredocLit) RawContent() string {
	n := e.backticks()
	body := e.Literal[n : len(e.Literal)-n]
	if len(body) > 0 && body[0] == '\n' {
		body = body[1:]
		if i := strings.LastIndexByte(body, '\n'); i >= 0 {
			body = body[:i]
		}
	}
	return body
}

// StripCount returns the common leading indentation (spaces/tabs) removed from
// each content line by Value. It is zero for the single-line form (no newline
// after the opening fence), which is not indentation-stripped.
func (e *RawHeredocLit) StripCount() int {
	n := e.backticks()
	if n >= len(e.Literal) || e.Literal[n] != '\n' {
		return 0
	}
	c := e.RawContent()
	i := 0
	for i < len(c) && (c[i] == ' ' || c[i] == '\t') {
		i++
	}
	return i
}

// stripHeredocIndent removes up to n leading spaces/tabs from every line of s.
func stripHeredocIndent(s string, n int) string {
	if n <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		j := 0
		for j < n && j < len(line) && (line[j] == ' ' || line[j] == '\t') {
			j++
		}
		lines[i] = line[j:]
	}
	return strings.Join(lines, "\n")
}

// stripIndentAfterNewlines removes up to n leading spaces/tabs after every
// newline in s, and also at the start when atStart is true. It applies the same
// per-line heredoc indentation stripping as stripHeredocIndent to a single text
// segment, without touching the segment's source position.
func stripIndentAfterNewlines(s string, n int, atStart bool) string {
	if n <= 0 {
		return s
	}
	var out strings.Builder
	out.Grow(len(s))
	i := 0
	skip := func() {
		for c := 0; c < n && i < len(s) && (s[i] == ' ' || s[i] == '\t'); c++ {
			i++
		}
	}
	if atStart {
		skip()
	}
	for i < len(s) {
		c := s[i]
		out.WriteByte(c)
		i++
		if c == '\n' {
			skip()
		}
	}
	return out.String()
}

func (e *RawHeredocLit) String() string {
	return e.Literal
}

func (e *RawHeredocLit) Value() string {
	return stripHeredocIndent(e.RawContent(), e.StripCount())
}

func (e *RawHeredocLit) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(e.Literal)
}

// HeredocLit represents a non-raw heredoc string literal delimited by a fence
// of three or more double quotes (`"""`). It shares RawHeredocLit's fencing,
// opening-line and common-indentation handling, but, like a double-quoted
// StrLit, it interprets escape sequences (e.g. \n, \t, \", \xFF, \uXXXX) in
// its content.
type HeredocLit struct {
	Literal    string
	LiteralPos source.Pos
}

func (e *HeredocLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *HeredocLit) Pos() source.Pos {
	return e.LiteralPos
}

// End returns the position of first character immediately after the node.
func (e *HeredocLit) End() source.Pos {
	return source.Pos(int(e.LiteralPos) + len(e.Literal))
}

// quotes returns the number of leading double-quote fence characters.
func (e *HeredocLit) quotes() int {
	n := 0
	for n < len(e.Literal) && e.Literal[n] == '"' {
		n++
	}
	return n
}

// contentOffset returns the byte offset within Literal at which RawContent
// begins: past the leading quotes and, for the common multiline form, the
// newline that ends the opening fence line.
func (e *HeredocLit) contentOffset() int {
	n := e.quotes()
	if n < len(e.Literal) && e.Literal[n] == '\n' {
		return n + 1
	}
	return n
}

// ContentPos returns the source position of the first byte of RawContent.
func (e *HeredocLit) ContentPos() source.Pos {
	return source.Pos(int(e.LiteralPos) + e.contentOffset())
}

// RawContent returns the heredoc body with the surrounding quote fences, the
// opening fence line and the closing line removed, but with interior
// indentation preserved and escape sequences left unprocessed. It keeps a 1:1
// byte correspondence with the original source starting at ContentPos.
func (e *HeredocLit) RawContent() string {
	n := e.quotes()
	body := e.Literal[n : len(e.Literal)-n]
	if len(body) > 0 && body[0] == '\n' {
		body = body[1:]
		if i := strings.LastIndexByte(body, '\n'); i >= 0 {
			body = body[:i]
		}
	}
	return body
}

// StripCount returns the common leading indentation (spaces/tabs) removed from
// each content line by Value. It is zero for the single-line form (no newline
// after the opening fence), which is not indentation-stripped.
func (e *HeredocLit) StripCount() int {
	n := e.quotes()
	if n >= len(e.Literal) || e.Literal[n] != '\n' {
		return 0
	}
	c := e.RawContent()
	i := 0
	for i < len(c) && (c[i] == ' ' || c[i] == '\t') {
		i++
	}
	return i
}

func (e *HeredocLit) String() string {
	return e.Literal
}

func (e *HeredocLit) Value() string {
	return unescapeHeredoc(stripHeredocIndent(e.RawContent(), e.StripCount()))
}

func (e *HeredocLit) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(e.Literal)
}

// unescapeHeredoc interprets the escape sequences in a non-raw heredoc body the
// same way a double-quoted string literal does. Literal newlines and unescaped
// double quotes are preserved as-is, and any unrecognized escape is kept
// verbatim rather than reported as an error.
func unescapeHeredoc(s string) string {
	if !strings.ContainsRune(s, '\\') {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		if s[i] != '\\' {
			b.WriteByte(s[i])
			i++
			continue
		}
		value, multibyte, tail, err := strconv.UnquoteChar(s[i:], '"')
		if err != nil {
			// keep the backslash verbatim on an unrecognized escape sequence
			b.WriteByte(s[i])
			i++
			continue
		}
		if multibyte {
			b.WriteRune(value)
		} else {
			b.WriteByte(byte(value))
		}
		i = len(s) - len(tail)
	}
	return b.String()
}

// TemplateLit represents a template string literal prefixed with `#`, such as
// `#"text"` or `#'symbol'`. It is parsed in ParseOperand when a token.Template
// is followed by a string, raw string, heredoc, or symbol token.
// The Value field holds the string/symbol expression that follows the `#` token.
type TemplateLit struct {
	TokenPos source.Pos
	Value    Expr
}

func (e *TemplateLit) ExprNode() {}

func (e *TemplateLit) Pos() source.Pos {
	return e.TokenPos
}

func (e *TemplateLit) End() source.Pos {
	return e.Value.End() + 6
}

func (e *TemplateLit) String() string {
	return "#" + e.Value.String()
}

func (e *TemplateLit) StringValue() string {
	switch vt := e.Value.(type) {
	case *StrLit:
		return vt.Value()
	case *RawStrLit:
		return vt.Value()
	case *RawHeredocLit:
		// Parse the untrimmed body so interpolation positions map 1:1 to the
		// source; Build re-applies the heredoc indentation stripping to the
		// rendered text segments.
		return vt.RawContent()
	case *HeredocLit:
		// Parse the untrimmed, un-escaped body so interpolation positions map
		// 1:1 to the source; Build re-applies indentation stripping and escape
		// processing to the rendered text segments.
		return vt.RawContent()
	case *SymbolLit:
		return vt.Value()
	default:
		return ""
	}
}

// StringValuePos returns the source position to pass to
// parser.ParseTemplateString: the position of the byte immediately before the
// first content byte of StringValue (the template content begins one byte
// after it). For single-delimiter values that is the opening delimiter; for a
// heredoc it is the newline that ends the opening backtick line, since the
// surrounding backticks and the opening line are stripped from the content.
func (e *TemplateLit) StringValuePos() source.Pos {
	switch h := e.Value.(type) {
	case *RawHeredocLit:
		return h.ContentPos() - 1
	case *HeredocLit:
		return h.ContentPos() - 1
	}
	return e.Value.Pos()
}

func (e *TemplateLit) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteSingleByte('#')
	e.Value.WriteCode(ctx)
}

func (e *TemplateLit) Build(sourceStmts Stmts) (expr Expr, err error) {
	var raw bool
	// unescape marks a non-raw heredoc whose rendered text segments must have
	// their escape sequences interpreted (like a double-quoted string), even
	// though they were parsed un-escaped to keep interpolation positions mapped
	// to the original source.
	var unescape bool
	// stripCount > 0 marks a heredoc whose rendered text segments must be
	// indentation-stripped, even though they were parsed untrimmed (to keep
	// interpolation positions mapped to the original source).
	var stripCount int
	switch v := e.Value.(type) {
	case *RawHeredocLit:
		raw = true
		stripCount = v.StripCount()
	case *HeredocLit:
		unescape = true
		stripCount = v.StripCount()
	case *RawStrLit:
		raw = true
	case *StrLit, *SymbolLit:
	default:
		return nil, errors.New("template literal must be a string, raw string, heredoc, or symbol")
	}

	// The first text segment begins a source line, so its leading indentation
	// is stripped too; later segments follow an interpolation mid-line.
	atLineStart := true
	var exprs Exprs
	for _, stmt := range sourceStmts {
		var exp Expr
		switch lit := stmt.(type) {
		case *MixedTextStmt:
			val := lit.Value()
			if stripCount > 0 {
				trimmed := stripIndentAfterNewlines(val, stripCount, atLineStart)
				atLineStart = strings.HasSuffix(val, "\n")
				val = trimmed
			}
			if raw {
				// Raw text is kept verbatim; a RawStrLit emits it as-is,
				// whereas a StrLit would later be unquoted by the compiler.
				exp = &RawStrLit{
					Literal:    val,
					LiteralPos: lit.Pos(),
				}
			} else {
				if unescape {
					// Interpret escape sequences after indentation stripping;
					// String re-quotes the result so the compiler reproduces it
					// verbatim instead of re-processing the escapes.
					val = unescapeHeredoc(val)
				}
				exp = Str(val, lit.Pos())
			}
		case *ExprStmt:
			exp = lit.Expr
			atLineStart = false
		case *MixedValueStmt:
			exp = lit.Expr
			atLineStart = false
		default:
			continue
		}
		exprs = append(exprs, exp)
	}

	call := &CallExpr{
		Func:     &IdentExpr{Name: "str", NamePos: e.Pos()},
		CallArgs: CallArgs{Args: CallExprPositionalArgs{Values: exprs}},
	}

	if raw {
		return &ToRaw{Expr: call, TokenPos: e.Pos()}, nil
	}

	return call, nil
}

type CallArgs struct {
	LParen    source.Pos
	RParen    source.Pos
	Args      CallExprPositionalArgs
	NamedArgs CallExprNamedArgs
}

func NewCallArgs(LParen source.Pos, RParen source.Pos) *CallArgs {
	ca := &CallArgs{LParen: LParen, RParen: RParen}
	return ca
}

// Pos returns the position of first character belonging to the node.
func (c *CallArgs) Pos() source.Pos {
	return c.LParen
}

// End returns the position of first character immediately after the node.
func (c *CallArgs) End() source.Pos {
	return c.RParen + 1
}

func (c *CallArgs) String() string {
	var buf strings.Builder
	c.StringW(&buf)
	return buf.String()
}

func (c *CallArgs) StringW(w io.Writer) {
	c.StringArg(w, "(", ")")
}

func (c *CallArgs) StringArg(w io.Writer, lbrace, rbrace string) {
	io.WriteString(w, lbrace)
	if c.Args.Valid() {
		io.WriteString(w, c.Args.String())
	}
	if c.NamedArgs.Valid() {
		io.WriteString(w, "; ")
		io.WriteString(w, c.NamedArgs.String())
	}
	io.WriteString(w, rbrace)
}

func (c *CallArgs) WriteCode(ctx *CodeWriteContext) {
	c.WriteCodeBrace(ctx, "(", ")")
}

func (c *CallArgs) WriteCodeBrace(ctx *CodeWriteContext, lbrace, rbrace string) {
	inNewLine := ctx.Flags.Has(CodeWriteContextFlagFormatCallParamsInNewLine)

	ctx.WriteString(lbrace)
	if c.Args.Valid() {
		c.Args.WriteCodeWithNamed(ctx, inNewLine, c.NamedArgs.Valid())
	}
	if c.NamedArgs.Valid() {
		c.NamedArgs.WriteCode(ctx, inNewLine, c.Args.Valid())
	}
	if c.Args.Valid() || c.NamedArgs.Valid() {
		ctx.WritePrefix()
	}
	ctx.WriteString(rbrace)
}

func (c *CallArgs) Arg(e ...Expr) *CallArgs {
	c.Args.AppendValues(e...)
	return c
}

func (c *CallArgs) ArgVar(pos source.Pos, e Expr) *CallArgs {
	c.Args.Var = &ArgVarLit{Value: e, TokenPos: pos}
	return c
}

func (c *CallArgs) NamedFlag(e ...Expr) *CallArgs {
	for _, flagE := range e {
		c.NamedArgs.AppendFlagE(flagE)
	}
	return c
}

func (c *CallArgs) NamedValue(e ...Expr) *CallArgs {
	for i := 0; i < len(e); i += 2 {
		c.NamedArgs.AppendE(e[i], e[i+1])
	}
	return c
}

func (c *CallArgs) ToFuncParams() (fp *FuncParams, err error) {
	fp = &FuncParams{
		LParen: c.LParen,
		RParen: c.RParen,
	}

	for i, v := range c.Args.Values {
		switch t := v.(type) {
		case *IdentExpr:
			fp.Args.Values = append(fp.Args.Values, &TypedIdentExpr{Ident: t})
		case *TypedIdentExpr:
			fp.Args.Values = append(fp.Args.Values, t)
		default:
			return nil, fmt.Errorf("arg[%d] expected arg type as *Ident, but got %T", i, v)
		}
	}

	if c.Args.Var != nil {
		switch t := c.Args.Var.Value.(type) {
		case *IdentExpr:
			fp.Args.Var = &TypedIdentExpr{Ident: t}
		case *TypedIdentExpr:
			fp.Args.Var = t
		default:
			return nil, fmt.Errorf("expected arg var type as *Ident|*TypedIdent, but got %T", c.Args.Var.Value)
		}
	}

	for i, n := range c.NamedArgs.Names {
		if n.Var {
			if fp.NamedArgs.Var != nil {
				return nil, fmt.Errorf("multiple named args var")
			}

			if n.Ident != nil {
				fp.NamedArgs.Var = n.Ident
			} else {
				if ident, _ := n.Exp.(*IdentExpr); ident != nil {
					fp.NamedArgs.Var = ident
					continue
				} else {
					return nil, fmt.Errorf("named arg var %s isn't *Ident", n)
				}
			}
		} else {
			if n.Ident == nil {
				return nil, fmt.Errorf("named arg[%d] expected *Ident, but got %T", i, n.Lit)
			}
			fp.NamedArgs.Names = append(fp.NamedArgs.Names, ETypedIdent(n.Ident))
			fp.NamedArgs.Values = append(fp.NamedArgs.Values, nil)
		}
	}
	return
}

type SymbolLit struct {
	Lit TokenLit
}

func (s *SymbolLit) Pos() source.Pos {
	return s.Lit.Pos
}

func (s *SymbolLit) End() source.Pos {
	return s.Lit.Pos + source.Pos(len(s.Lit.Literal))
}

func (s *SymbolLit) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(s.Lit.Literal)
}

func (s *SymbolLit) ExprNode() {
}

func (s *SymbolLit) String() string {
	return "#(" + s.Value() + ")"
}

func (s *SymbolLit) Value() string {
	v := s.Lit.Literal[1:]
	if v[0] == '(' {
		v = quote.Unquote(v, ")")
	}
	return v
}
