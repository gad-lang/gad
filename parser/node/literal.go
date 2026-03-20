package node

import (
	"bytes"
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
				Lambda: Token{Token: token.Colon},
				Body:   t.Body,
			},
		}
	case *FuncExpr:
		if t.BodyExpr != nil {
			return &FuncDefLit{
				Expr: &ClosureExpr{
					Params: t.Type.Params,
					Lambda: Token{Token: token.Colon},
					Body:   t.BodyExpr,
				},
			}
		}
		return &FuncDefLit{
			Expr: &FuncExpr{
				Type: &FuncType{
					Params: t.Type.Params,
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
	case *RawStringLit:
		return String(t.Value(), e.Key.Pos())
	case *SymbolLit:
		return String(t.Value(), e.Key.Pos())
	case *DecimalLit:
		return String(t.Value.String(), e.Key.Pos())
	case *IdentExpr, *IntLit, *UintLit, *FloatLit:
		return String(t.String(), e.Key.Pos())
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
	case *IdentExpr, *IntLit, *UintLit, *FloatLit, *DecimalLit, *SymbolLit, *ParenExpr:
		b.WriteString(e.Key.String())
	case *StringLit:
		if v := t.Value(); runehelper.IsIdentifierOrDigitRunes([]rune(v)) {
			b.WriteString(v)
		} else {
			b.WriteString(t.String())
		}
	case *RawStringLit:
		if v := t.Value(); runehelper.IsIdentifierOrDigitRunes([]rune(v)) {
			b.WriteString(v)
		} else {
			b.WriteString(t.String())
		}
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

// StringLit represents a string literal.
type StringLit struct {
	ValuePos source.Pos
	Literal  string
}

func (e *StringLit) Value() string {
	var lit = e.Literal
	if len(lit) > 0 && lit[0] == '\'' {
		lit = `"` + strings.ReplaceAll(strings.ReplaceAll(lit[1:len(lit)-1], "\\'", "'"), `"`, `\"`) + `"`
	}
	v, err := strconv.Unquote(lit)
	if err != nil {
		panic(fmt.Sprintf("StringLit can not unquote: %v", err))
	}
	return v
}

func (e *StringLit) CanIdent() bool {
	return runehelper.IsIdentifierRunes([]rune(e.Value()))
}

func (e *StringLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *StringLit) Pos() source.Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *StringLit) End() source.Pos {
	return source.Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *StringLit) String() string {
	return e.Literal
}

func (e *StringLit) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(e.Literal)
}

type RawStringLit struct {
	Literal    string
	LiteralPos source.Pos
	Quoted     bool
}

func (e *RawStringLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *RawStringLit) Pos() source.Pos {
	return e.LiteralPos
}

// End returns the position of first character immediately after the node.
func (e *RawStringLit) End() source.Pos {
	return source.Pos(int(e.LiteralPos) + len(e.Literal))
}

func (e *RawStringLit) String() string {
	return e.QuotedValue()
}

func (e *RawStringLit) Value() string {
	if e.Quoted {
		s, _ := strconv.Unquote(e.Literal)
		return s
	} else {
		return e.Literal
	}
}

func (e *RawStringLit) QuotedValue() string {
	if e.Quoted {
		return e.Literal
	} else {
		return quote.Quote(e.Literal, "`")
	}
}

func (e *RawStringLit) WriteCode(ctx *CodeWriteContext) {
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
				Lambda: Token{Token: token.Colon},
				Body:   t.Body,
			},
		}
	case *FuncExpr:
		if t.BodyExpr != nil {
			return &FuncDefLit{
				Expr: &ClosureExpr{
					Params: t.Type.Params,
					Lambda: Token{Token: token.Colon},
					Body:   t.BodyExpr,
				},
			}
		}
		return &FuncDefLit{
			Expr: &FuncExpr{
				Type: &FuncType{
					Params: t.Type.Params,
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
	ctx.WriteItems(
		ctx.Flags.Has(CodeWriteContextFlagFormatKeyValueArrayItemInNewLine),
		len(e.Elements),
		func(i int) {
			e.Elements[i].WriteCode(ctx)
		},
		func(nl bool) {
			if nl {
				if len(ctx.Prefix) > 0 {
					ctx.WriteSecondLine()
				} else {
					ctx.WriteLine(", ")
				}
			}
		})
	ctx.WriteSingleByte(')')
}

func (e *KeyValueArrayLit) ToMultiParenExpr() *MultiParenExpr {
	return &MultiParenExpr{
		LParen:        e.LParen,
		RParen:        e.RParen,
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

func (e *RawHeredocLit) String() string {
	return e.Literal
}

func (e *RawHeredocLit) Value() string {
	var bts = []byte(e.Literal)

	for i, r := range bts {
		if r != '`' {
			if r == '\n' {
				bts = bts[i+1 : len(bts)-i]
				// remove Last Line
				bts = bts[:bytes.LastIndexByte(bts, '\n')]
				var stripCount int
			l2:
				for j, r := range bts {
					switch r {
					case ' ', '\t':
					default:
						stripCount = j
						break l2
					}
				}

				if stripCount > 0 {
					var (
						lines = bytes.Split(bts, []byte{'\n'})
						out   strings.Builder
					)
					for j, line := range lines {
						var i int
					l3:
						for ; i < stripCount && i < len(line); i++ {
							switch line[i] {
							case '\t', ' ':
							default:
								break l3
							}
						}

						if j > 0 {
							out.WriteByte('\n')
						}

						out.Write(line[i:])
					}

					return out.String()
				}
			} else {
				bts = bts[i : len(bts)-i]
			}
			break
		}
	}
	return string(bts)
}

func (e *RawHeredocLit) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(e.Literal)
}

// TemplateLit represents an variadic of argument.
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

func (e *TemplateLit) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteSingleByte('#')
	e.Value.WriteCode(ctx)
}

type CallArgs struct {
	LParen    source.Pos
	Args      CallExprPositionalArgs
	NamedArgs CallExprNamedArgs
	RParen    source.Pos
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
	ctx.WriteString(lbrace)
	if c.Args.Valid() {
		c.Args.WriteCodeWithNamedSep(ctx, c.NamedArgs.Valid())
	}
	if c.NamedArgs.Valid() {
		if !c.Args.Valid() {
			ctx.WriteString("; ")
		}
		c.NamedArgs.WriteCode(ctx)
	}
	if c.Args.Valid() || c.NamedArgs.Valid() {
		ctx.WritePrefix()
	}
	ctx.WriteString(rbrace)
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
