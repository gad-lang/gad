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
	CodeWriteContextFlagFormatDeclItemInNewLine

	CodeWriteContextFlagFormat = CodeWriteContextFlagFormatArrayItemInNewLine |
		CodeWriteContextFlagFormatDictItemInNewLine |
		CodeWriteContextFlagFormatKeyValueArrayItemInNewLine |
		CodeWriteContextFlagFormatCallParamsInNewLine |
		CodeWriteContextFlagFormatParemValuesInNewLine |
		CodeWriteContextFlagFormatDeclItemInNewLine
)

type CodeWriteSkiper interface {
	SkipCode(ctx *CodeWriteContext) bool
}

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

func (ctx CodeWriteContext) WithoutPrefix() *CodeWriteContext {
	ctx.Prefix = ""
	return &ctx
}

func (ctx CodeWriteContext) Buffer(do func(ctx *CodeWriteContext)) string {
	var buf bytes.Buffer
	ctx.CodeWriter = NewCodeWriter(&buf)
	do(&ctx)
	return buf.String()
}

func (ctx *CodeWriteContext) HasPrefix() bool {
	return ctx.Prefix != ""
}

func (ctx *CodeWriteContext) CurrentPrefix() string {
	return strings.Repeat(ctx.Prefix, ctx.Depth)
}

func (ctx *CodeWriteContext) WritePrefix() {
	ctx.WriteString(ctx.CurrentPrefix())
}

func (ctx *CodeWriteContext) PrevPrefix() string {
	if ctx.Depth == 0 {
		return ""
	}
	return strings.Repeat(ctx.Prefix, ctx.Depth-1)
}

func (ctx *CodeWriteContext) WritePrevPrefix() {
	ctx.WriteString(ctx.PrevPrefix())
}

func (ctx *CodeWriteContext) WriteLine(s string) {
	ctx.WriteString(s)
	ctx.WriteString("\n")
}

func (ctx *CodeWriteContext) WritePrefixedLine() {
	ctx.WriteString("\n", ctx.CurrentPrefix())
}

func (ctx *CodeWriteContext) WriteSecondLine() {
	if ctx.HasPrefix() {
		ctx.WriteString("\n")
	}
}

func (ctx *CodeWriteContext) WriteSemi() {
	if !ctx.HasPrefix() {
		ctx.WriteString("; ")
	} else {
		ctx.WriteString("\n" + ctx.CurrentPrefix())
	}
}

func (ctx *CodeWriteContext) WriteSemiOrDoubleLine() {
	if !ctx.HasPrefix() {
		ctx.WriteString("; ")
	} else {
		ctx.WriteString("\n\n")
	}
}

func (ctx *CodeWriteContext) Printf(format string, args ...interface{}) {
	fmt.Fprintf(ctx.CodeWriter, format, args...)
}

func (ctx *CodeWriteContext) Top() ast.Node {
	return ctx.Stack[len(ctx.Stack)-1]
}

func (ctx *CodeWriteContext) Push(n ast.Node) {
	ctx.Stack = append(ctx.Stack, n)
}

func (ctx *CodeWriteContext) Pop() {
	ctx.Stack = ctx.Stack[:len(ctx.Stack)-1]
}

func (ctx *CodeWriteContext) With(n ast.Node, cb func() error) (err error) {
	ctx.Push(n)
	err = cb()
	ctx.Pop()
	return
}

func (ctx *CodeWriteContext) simplifyStmts(stmt []Stmt) (ret []Stmt) {
	l := len(stmt)

loop:
	for i := 0; i < l; i++ {
		if i > 0 {
			switch e := stmt[i].(type) {
			case *DeclStmt:
				if ge, _ := e.Decl.(*GenDecl); ge != nil {
					if last, _ := ret[len(ret)-1].(*DeclStmt); last != nil {
						if lge, _ := last.Decl.(*GenDecl); lge != nil {
							if ge.Tok == lge.Tok {
								lge.Specs = append(lge.Specs, ge.Specs...)
								continue loop
							}
						}
					}
				}
			}
		}
		ret = append(ret, stmt[i])
	}
	return
}

func (ctx *CodeWriteContext) WriteStmts(stmt ...Stmt) {
	stmt = ctx.simplifyStmts(stmt)

	var (
		i   int
		sep = true
	)

	Stmts(stmt).Each(func(_ int, _ bool, s Stmt) {
		if skiper, _ := s.(CodeWriteSkiper); skiper != nil {
			if skiper.SkipCode(ctx) {
				return
			}
		}

		if sep {
			if i > 0 {
				ctx.WriteSemi()
			}
		}
		s.WriteCode(ctx)
		i++

		switch s.(type) {
		case *CodeBeginStmt:
			sep = true
		case *CodeEndStmt, *ConfigStmt:
			sep = false
		}
	})
}

func (ctx *CodeWriteContext) WriteItems(inNewLine bool, count int, do func(i int), done func(newLine bool)) {
	ctx.WriteItemsSep(inNewLine, count, ", ", ",", do, done)
}

func (ctx *CodeWriteContext) WriteItemsSep(inNewLine bool, count int, inlineSep, newLineSep string, do func(i int), done func(newLine bool)) {
	if count == 0 {
		return
	}

	last := count - 1

	if inNewLine {
		ctx.Depth++
		ctx.WriteSecondLine()
		for i := 0; i < count; i++ {
			ctx.WritePrefix()
			do(i)
			if i != last {
				ctx.WriteString(newLineSep)
				ctx.WriteSecondLine()
			}
		}
		if done != nil {
			done(inNewLine)
		}
		ctx.Depth--
	} else {
		for i := 0; i < count; i++ {
			do(i)
			if i != last {
				ctx.WriteString(inlineSep)
			}
		}
		if done != nil {
			done(inNewLine)
		}
	}
}

func (ctx *CodeWriteContext) WriteExprs(sep string, expr ...Expr) {
	for i, e := range expr {
		if i > 0 {
			ctx.WriteString(sep)
		}
		e.WriteCode(ctx)
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
