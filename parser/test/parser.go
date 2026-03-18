package test

import (
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Parser struct {
	t           *testing.T
	input       string
	actual      *parser.File
	codeOptions []node.CodeOption

	options        *parser.ParserOptions
	scannerOptions *parser.ScannerOptions
}

func New(t *testing.T, input string) *Parser {
	return &Parser{
		t:       t,
		input:   input,
		options: &parser.ParserOptions{},
		scannerOptions: &parser.ScannerOptions{
			MixedDelimiter: mixedDelimiter,
		},
	}
}

func (p *Parser) WithMode(m parser.Mode) *Parser {
	p.options.Mode |= m
	return p
}

func (p *Parser) GetParserOptions() *parser.ParserOptions {
	return p.options
}

func (p *Parser) WithParserOptions(f func(opt *parser.ParserOptions)) *Parser {
	f(p.options)
	return p
}

func (p *Parser) GetScannerOptions() *parser.ScannerOptions {
	return p.scannerOptions
}

func (p *Parser) WithScannerOptions(f func(opt *parser.ScannerOptions)) *Parser {
	f(p.scannerOptions)
	return p
}

func (p *Parser) WithCodeOptions(opt ...node.CodeOption) *Parser {
	p.codeOptions = append(p.codeOptions, opt...)
	return p
}

func (p *Parser) WithMixed() *Parser {
	return p.WithMode(parser.ParseMixed)
}

func (p *Parser) File() *File {
	if p.actual == nil {
		p.parse()
	}
	return NewFile(p.t, p.actual)
}

func (p *Parser) Stmts(fn ExpectedFn, post ...PostFileCallback) *Parser {
	f := p.File()
	f.Expect(fn, post...)
	return p
}

func (p *Parser) parse() {
	var ok bool
	defer func() {
		if !ok {
			// print Trace
			tr := &Tracer{}
			_, _ = p.ParseSource(tr)
			p.t.Logf("Trace:\n%s", strings.Join(tr.Out, ""))
		}
	}()

	var err error

	p.actual, err = p.ParseSource(nil)
	require.NoError(p.t, err)
	ok = true
}

func (p *Parser) Error(e ...[2]string) {
	var ok bool
	defer func() {
		if !ok {
			// print Trace
			tr := &Tracer{}
			_, _ = p.ParseSource(tr)
			p.t.Logf("Trace:\n%s", strings.Join(tr.Out, ""))
		}
	}()

	_, err := p.ParseSource(nil)
	require.Error(p.t, err)

	if len(e) > 0 {
		for i, ev := range e {
			s := fmt.Sprintf(ev[0], err)
			require.Equal(p.t, ev[1], s, "formatted error "+strconv.Itoa(i))
		}
	}
	ok = true
}

func (p *Parser) String(s string) *Parser {
	p.Run(func(pt *Parser) {
		require.Equal(p.t, s, p.actual.String(), "EXPECT STRING")
	})
	return p
}

func (p *Parser) Code(s string, opt ...node.CodeOption) *Parser {
	p.Run(func(pt *Parser) {
		require.Equal(p.t, s, node.Code(p.actual.Stmts, append(p.codeOptions, opt...)...), "EXPECT CODE")
	})
	return p
}

func (p *Parser) IndentedCode(s string, opt ...node.CodeOption) *Parser {
	p.Run(func(pt *Parser) {
		opts := append(append(p.codeOptions, opt...), node.CodeWithPrefix("\t"))
		require.Equal(p.t, s, node.Code(p.actual.Stmts, opts...), "EXPECT INDENTED CODE")
	})
	return p
}

func (p *Parser) FormattedCode(s string, opt ...node.CodeOption) *Parser {
	p.Run(func(pt *Parser) {
		opts := append(append(p.codeOptions, append(opt, node.CodeWithFlags(node.CodeWriteContextFlagFormat))...),
			node.CodeWithPrefix("\t"))
		code := node.Code(p.actual.Stmts, opts...)
		require.Equal(p.t, s, code, "EXPECT FORMATTED CODE")
	})
	return p
}

func (p *Parser) Type(v any) *Parser {
	if v != nil {
		typ := reflect.TypeOf(v)
		p.Run(func(pt *Parser) {
			assert.Equal(p.t, typ.String(), reflect.TypeOf(p.actual.Stmts[0].(*node.ExprStmt).Expr).String())
		})
	}
	return p
}

func (p *Parser) Run(cb ...func(pt *Parser)) *Parser {
	if p.actual == nil {
		p.parse()
	}

	if len(cb) == 0 {
		cb = append(cb, nil)
	}

	for _, f := range cb {
		cp := *p
		if f != nil {
			f(&cp)
		}
	}
	return p
}

func (p *Parser) ParseSource(
	trace io.Writer,
) (*parser.File, error) {
	fileSet := source.NewFileSet()
	file := fileSet.AddFileData("test", -1, []byte(p.input))
	po := *p.options
	po.Trace = trace

	pr := parser.NewParserWithOptions(file, &po, p.scannerOptions)
	return pr.ParseFile()
}

var mixedDelimiter = parser.MixedDelimiter{
	Start: []rune("‹"),
	End:   []rune("›"),
}
