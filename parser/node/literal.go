package node

import (
	"bytes"
	"fmt"
	"io"
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
	Key      string
	KeyPos   source.Pos
	ColonPos source.Pos
	Value    Expr
}

func (e *DictElementLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *DictElementLit) Pos() source.Pos {
	return e.KeyPos
}

// End returns the position of first character immediately after the node.
func (e *DictElementLit) End() source.Pos {
	return e.Value.End()
}

func (e *DictElementLit) String() string {
	return e.Key + ": " + e.Value.String()
}

func (e *DictElementLit) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(e.Key)
	ctx.WriteString(": ")
	e.Value.WriteCode(ctx)
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

func (e *RawStringLit) UnquotedValue() string {
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

// KeyValueSepLit represents a key value separator in paren context
type KeyValueSepLit struct {
	TokenPos source.Pos
}

func (e *KeyValueSepLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *KeyValueSepLit) Pos() source.Pos {
	return e.TokenPos
}

// End returns the position of first character immediately after the node.
func (e *KeyValueSepLit) End() source.Pos {
	return e.TokenPos + 9 // len(nil) == 9
}

func (e *KeyValueSepLit) String() string {
	return ";"
}

func (e *KeyValueSepLit) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString("; ")
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
	ctx.WriteByte('[')
	e.Key.WriteCode(ctx)
	ctx.WriteByte('=')
	e.Value.WriteCode(ctx)
	ctx.WriteByte(']')
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

func (e *KeyValuePairLit) String() string {
	if e.Value == nil {
		return e.Key.String()
	}
	return e.Key.String() + "=" + e.Value.String()
}

func (e *KeyValuePairLit) WriteCode(ctx *CodeWriteContext) {
	e.Key.WriteCode(ctx)
	ctx.WriteByte('=')
	e.Value.WriteCode(ctx)
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
	l := len(e.Elements) - 1
	for i, element := range e.Elements {
		switch t := element.(type) {
		case *KeyValuePairLit:
			t.Key.WriteCode(ctx)
			if t.Value != nil {
				ctx.WriteByte('=')
				t.Value.WriteCode(ctx)
			}
		case *KeyValueLit:
			t.Key.WriteCode(ctx)
			if t.Value != nil {
				ctx.WriteByte('=')
				t.Value.WriteCode(ctx)
			}
		case *NamedArgVarLit:
			t.WriteCode(ctx)
		}
		if i < l {
			ctx.WriteString(", ")
		}
	}
	ctx.WriteByte(')')
}

func (e *KeyValueArrayLit) ToMultiParenExpr() *MultiParenExpr {
	r := &MultiParenExpr{
		LParen: e.LParen,
		RParen: e.RParen,
		Exprs:  make([]Expr, len(e.Elements)),
	}

	for i, ele := range e.Elements {
		r.Exprs[i] = ele
	}

	return r
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
	ctx.WriteByte('*')
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

// DotFileNameLit represents an __name__ literal.
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

// DotFileLit represents an __name__ literal.
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

// IsModuleLit represents an __is_module__ literal.
type IsModuleLit struct {
	TokenPos source.Pos
}

func (e *IsModuleLit) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *IsModuleLit) Pos() source.Pos {
	return e.TokenPos
}

// End IsModuleLit the position of first character immediately after the node.
func (e *IsModuleLit) End() source.Pos {
	return e.TokenPos + source.Pos(len(e.String()))
}

func (e *IsModuleLit) String() string {
	return token.IsModule.String()
}

func (e *IsModuleLit) WriteCode(ctx *CodeWriteContext) {
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
	ctx.WriteByte('#')
	e.Value.WriteCode(ctx)
}

type CallArgs struct {
	LParen    source.Pos
	Args      CallExprArgs
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
		if c.Args.Valid() {
			io.WriteString(w, ", ")
		}
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
		c.Args.WriteCode(ctx)
	}
	if c.NamedArgs.Valid() {
		ctx.WriteString("; ")
		c.NamedArgs.WriteCode(ctx)
	}
	ctx.WriteString(rbrace)
}
func (c *CallArgs) ToFuncParams() (fp *FuncParams, err error) {
	fp = &FuncParams{}

	for i, v := range c.Args.Values {
		if s, ok := v.(*IdentExpr); !ok {
			return nil, fmt.Errorf("arg[%d] expected arg type as *Ident, but got %T", i, v)
		} else {
			fp.Args.Values = append(fp.Args.Values, &TypedIdentExpr{Ident: s})
		}
	}

	if c.Args.Var != nil {
		if s, ok := c.Args.Var.Value.(*IdentExpr); !ok {
			return nil, fmt.Errorf("expected arg var type as *Ident, but got %T", c.Args.Var.Value)
		} else {
			fp.Args.Var = &TypedIdentExpr{Ident: s}
		}
	}

	for i, n := range c.NamedArgs.Names {
		if n.Ident == nil {
			return nil, fmt.Errorf("named arg[%d] expected *Ident, but got %T", i, n.Lit)
		}
		fp.NamedArgs.Names = append(fp.NamedArgs.Names, ETypedIdent(n.Ident))
		fp.NamedArgs.Values = append(fp.NamedArgs.Values, Flag(false, n.Ident.NamePos))
	}

	if c.NamedArgs.Var != nil {
		var ident, _ = c.NamedArgs.Var.Value.(*IdentExpr)

		if ident == nil {
			return nil, fmt.Errorf("named arg var expected *Ident, but got %T", c.NamedArgs.Var.Value)
		}
		fp.NamedArgs.Names = append(fp.NamedArgs.Names, ETypedIdent(ident))
	}
	return
}
