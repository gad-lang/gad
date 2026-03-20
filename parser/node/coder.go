package node

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/gad-lang/gad/parser/ast"
)

type Coder interface {
	WriteCode(ctx *CodeWriteContext)
}

type CodeWriter interface {
	io.Writer
	WriteString(s ...string)
	WriteSingleByte(b byte)
	WriteRune(b rune)
	WriteLine(s ...string)
	WriteLines(l ...string)
}

type cw struct {
	io.Writer
}

func (w *cw) WriteString(s ...string) {
	for _, s := range s {
		w.Write([]byte(s))
	}
}

func (w *cw) WriteRune(r rune) {
	w.WriteString(string([]rune{r}))
}

func (w *cw) WriteSingleByte(c byte) {
	w.Write([]byte{c})
}

func (w *cw) WriteLine(s ...string) {
	w.WriteString(s...)
	w.Write([]byte{'\n'})
}

func (w *cw) WriteLines(l ...string) {
	for _, s := range l {
		w.WriteLine(s)
	}
}

func NewCodeWriter(w io.Writer) CodeWriter {
	return &cw{Writer: w}
}

type TranspileOptions struct {
	RawStrFuncStart string
	RawStrFuncEnd   string
	WriteFunc       string
}

type CodeWriteContextFlag uint8

func (b *CodeWriteContextFlag) Set(flag CodeWriteContextFlag) *CodeWriteContextFlag {
	*b = *b | flag
	return b
}
func (b *CodeWriteContextFlag) Clear(flag CodeWriteContextFlag) *CodeWriteContextFlag {
	*b = *b &^ flag
	return b
}
func (b *CodeWriteContextFlag) Toggle(flag CodeWriteContextFlag) *CodeWriteContextFlag {
	*b = *b ^ flag
	return b
}
func (b CodeWriteContextFlag) Has(flag CodeWriteContextFlag) bool { return b&flag != 0 }

const (
	CodeWriteContextFlagNone CodeWriteContextFlag = 1 << iota
	CodeWriteContextFlagFormatArrayItemInNewLine
	CodeWriteContextFlagFormatDictItemInNewLine
	CodeWriteContextFlagFormatKeyValueArrayItemInNewLine
	CodeWriteContextFlagFormatCallParamsInNewLine
	CodeWriteContextFlagFormatParemValuesInNewLine

	CodeWriteContextFlagFormat = CodeWriteContextFlagFormatArrayItemInNewLine |
		CodeWriteContextFlagFormatDictItemInNewLine |
		CodeWriteContextFlagFormatKeyValueArrayItemInNewLine |
		CodeWriteContextFlagFormatCallParamsInNewLine |
		CodeWriteContextFlagFormatParemValuesInNewLine
)

type CodeWriteContext struct {
	Stack     []ast.Node
	Depth     int
	Prefix    string
	Flags     CodeWriteContextFlag
	Transpile *TranspileOptions
	CodeWriter
}

type CodeOption func(ctx *CodeWriteContext)

func CodeWithPrefix(prefix string) CodeOption {
	return func(ctx *CodeWriteContext) {
		ctx.Prefix = prefix
	}
}

func CodeWithFlags(flag CodeWriteContextFlag) CodeOption {
	return func(ctx *CodeWriteContext) {
		ctx.Flags.Set(flag)
	}
}

func CodeFormat() CodeOption {
	return func(ctx *CodeWriteContext) {
		ctx.Flags |= CodeWriteContextFlagFormat
	}
}

func CodeTranspile(v *TranspileOptions) CodeOption {
	return func(ctx *CodeWriteContext) {
		ctx.Transpile = v
	}
}

func NewCodeWriteContext(codeWriter CodeWriter, opt ...CodeOption) *CodeWriteContext {
	ctx := &CodeWriteContext{CodeWriter: codeWriter}
	for _, opt := range opt {
		opt(ctx)
	}
	return ctx
}

func (c CodeWriteContext) WithoutPrefix() *CodeWriteContext {
	c.Prefix = ""
	return &c
}

func (c CodeWriteContext) Buffer(do func(ctx *CodeWriteContext)) string {
	var buf bytes.Buffer
	c.CodeWriter = NewCodeWriter(&buf)
	do(&c)
	return buf.String()
}

func (c *CodeWriteContext) CurrentPrefix() string {
	return strings.Repeat(c.Prefix, c.Depth)
}

func (c *CodeWriteContext) WritePrefix() {
	c.WriteString(c.CurrentPrefix())
}

func (c *CodeWriteContext) PrevPrefix() string {
	if c.Depth == 0 {
		return ""
	}
	return strings.Repeat(c.Prefix, c.Depth-1)
}

func (c *CodeWriteContext) WritePrevPrefix() {
	c.WriteString(c.PrevPrefix())
}

func (c *CodeWriteContext) WriteLine(s string) {
	c.WriteString(s)
	c.WriteString("\n")
}

func (c *CodeWriteContext) WritePrefixedLine() {
	c.WriteString("\n", c.CurrentPrefix())
}

func (c *CodeWriteContext) WriteSecondLine() {
	if c.Prefix != "" {
		c.WriteString("\n")
	}
}

func (c *CodeWriteContext) WriteSemi() {
	if c.Prefix == "" {
		c.WriteString("; ")
	} else {
		c.WriteString("\n")
	}
}

func (c *CodeWriteContext) WriteSemiOrDoubleLine() {
	if c.Prefix == "" {
		c.WriteString("; ")
	} else {
		c.WriteString("\n\n")
	}
}

func (c *CodeWriteContext) Printf(format string, args ...interface{}) {
	fmt.Fprintf(c.CodeWriter, format, args...)
}

func (c *CodeWriteContext) Top() ast.Node {
	return c.Stack[len(c.Stack)-1]
}

func (c *CodeWriteContext) Push(n ast.Node) {
	c.Stack = append(c.Stack, n)
}

func (c *CodeWriteContext) Pop() {
	c.Stack = c.Stack[:len(c.Stack)-1]
}

func (c *CodeWriteContext) With(n ast.Node, cb func() error) (err error) {
	c.Push(n)
	err = cb()
	c.Pop()
	return
}

func (c *CodeWriteContext) WriteStmts(smt ...Stmt) {
	Stmts(smt).Each(func(i int, sep bool, s Stmt) {
		if sep {
			c.WriteSemi()
		}
		s.WriteCode(c)
	})
}

func (c *CodeWriteContext) WriteItems(inNewLine bool, count int, do func(i int), done func(newLine bool)) {
	c.WriteItemsSep(inNewLine, count, ", ", ",", do, done)
}

func (c *CodeWriteContext) WriteItemsSep(inNewLine bool, count int, inlineSep, newLineSep string, do func(i int), done func(newLine bool)) {
	if count == 0 {
		return
	}

	last := count - 1

	if inNewLine {
		c.Depth++
		c.WriteSecondLine()
		for i := 0; i < count; i++ {
			c.WritePrefix()
			do(i)
			if i != last {
				c.WriteString(newLineSep)
				c.WriteSecondLine()
			}
		}
		if done != nil {
			done(inNewLine)
		}
		c.Depth--
	} else {
		for i := 0; i < count; i++ {
			do(i)
			if i != last {
				c.WriteString(inlineSep)
			}
		}
		if done != nil {
			done(inNewLine)
		}
	}
}

func (c *CodeWriteContext) WriteExprs(sep string, expr ...Expr) {
	for i, e := range expr {
		if i > 0 {
			c.WriteString(sep)
		}
		e.WriteCode(c)
	}
}

func Code(n Coder, opt ...CodeOption) string {
	var buf bytes.Buffer
	CodeW(&buf, n, opt...)
	return buf.String()
}

func CodeW(w io.Writer, n Coder, opt ...CodeOption) {
	n.WriteCode(NewCodeWriteContext(NewCodeWriter(w), opt...))
}
