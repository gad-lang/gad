// A modified version ToInterface and Tengo parsers.

// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

// Copyright (c) 2019 Daniel Kang.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE.tengo file.

// Copyright 2009 The ToInterface Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.golang file.

package parser

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/token"
	"github.com/shopspring/decimal"
)

// Mode value is a set of flags for parser.
type Mode int

func (b *Mode) Set(flag Mode) *Mode    { *b = *b | flag; return b }
func (b *Mode) Clear(flag Mode) *Mode  { *b = *b &^ flag; return b }
func (b *Mode) Toggle(flag Mode) *Mode { *b = *b ^ flag; return b }
func (b Mode) Has(flag Mode) bool      { return b&flag != 0 }

const (
	// ParseComments parses comments and add them to AST
	ParseComments Mode = 1 << iota
	ParseMixed
	ParseConfigDisabled
	ParseMixedExprAsValue
)

type bailout struct{}

var stmtStart = map[token.Token]bool{
	token.Param:    true,
	token.Global:   true,
	token.Var:      true,
	token.Const:    true,
	token.Break:    true,
	token.Continue: true,
	token.For:      true,
	token.If:       true,
	token.Return:   true,
	token.Try:      true,
	token.Throw:    true,
}

// Error represents a parser error.
type Error struct {
	Pos source.SourceFilePos
	Msg string
}

func (e Error) Error() string {
	if e.Pos.Filename != "" || e.Pos.IsValid() {
		return fmt.Sprintf("Parse Error: %s\n\tat %s", e.Msg, e.Pos)
	}
	return fmt.Sprintf("Parse Error: %s", e.Msg)
}

// ErrorList is a collection of parser errors.
type ErrorList []*Error

// Add adds a new parser error to the collection.
func (p *ErrorList) Add(pos source.SourceFilePos, msg string) {
	*p = append(*p, &Error{pos, msg})
}

// Len returns the number of elements in the collection.
func (p ErrorList) Len() int {
	return len(p)
}

func (p ErrorList) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p ErrorList) Less(i, j int) bool {
	e := &p[i].Pos
	f := &p[j].Pos

	if e.Filename != f.Filename {
		return e.Filename < f.Filename
	}
	if e.Line != f.Line {
		return e.Line < f.Line
	}
	if e.Column != f.Column {
		return e.Column < f.Column
	}
	return p[i].Msg < p[j].Msg
}

// Sort sorts the collection.
func (p ErrorList) Sort() {
	sort.Sort(p)
}

func (p ErrorList) Error() string {
	switch len(p) {
	case 0:
		return "no errors"
	case 1:
		return p[0].Error()
	}
	return fmt.Sprintf("%s (and %d more errors)", p[0], len(p)-1)
}

// Err returns an error.
func (p ErrorList) Err() error {
	if len(p) == 0 {
		return nil
	}
	return p
}

// Parser parses the Tengo source files. It's based on ToInterface's parser
// implementation.
type Parser struct {
	File             *source.SourceFile
	Errors           ErrorList
	Scanner          ScannerInterface
	Token            Token
	PrevToken        Token
	ExprLevel        int        // < 0: in control clause, >= 0: in expression
	syncPos          source.Pos // last sync position
	syncCount        int        // number of advance calls without progress
	Trace            bool
	indent           int
	mode             Mode
	TraceOut         io.Writer
	comments         []*ast.CommentGroup
	ParseStmtHandler func() node.Stmt
	InCode           bool
	BlockStart       token.Token
	BlockEnd         token.Token
	ScanFunc         func() Token
	pipes            int
}

// NewParser creates a Parser.
func NewParser(file *source.SourceFile, src []byte, trace io.Writer) *Parser {
	return NewParserWithOptions(file, src, &ParserOptions{Trace: trace}, nil)
}

type ParserOptions struct {
	Trace io.Writer
	Mode  Mode
}

// NewParserWithOptions creates a Parser with parser mode flags.
func NewParserWithOptions(
	file *source.SourceFile,
	src []byte,
	opts *ParserOptions,
	scannerOptions *ScannerOptions,
) *Parser {
	if scannerOptions == nil {
		scannerOptions = &ScannerOptions{}
	}
	if scannerOptions.Mode == 0 {
		if opts.Mode.Has(ParseComments) {
			scannerOptions.Mode.Set(ScanComments)
		}
		if opts.Mode.Has(ParseMixed) {
			scannerOptions.Mode.Set(Mixed)
		}
		if opts.Mode.Has(ParseConfigDisabled) {
			scannerOptions.Mode.Set(ConfigDisabled)
		}
		if opts.Mode.Has(ParseMixedExprAsValue) {
			scannerOptions.Mode.Set(MixedExprAsValue)
		}
	}
	return NewParserWithScanner(NewScanner(file, src, scannerOptions), opts)
}

// NewParserWithScanner creates a Parser with parser mode flags.
func NewParserWithScanner(
	scanner ScannerInterface,
	opts *ParserOptions,
) *Parser {
	p := &Parser{
		Scanner:    scanner,
		File:       scanner.SourceFile(),
		Trace:      opts.Trace != nil,
		TraceOut:   opts.Trace,
		mode:       opts.Mode,
		BlockStart: token.LBrace,
		BlockEnd:   token.RBrace,
	}
	p.ParseStmtHandler = p.DefaultParseStmt
	var m ScanMode
	if opts.Mode.Has(ParseComments) {
		m.Set(ScanComments)
	}
	if opts.Mode.Has(ParseMixed) {
		m.Set(Mixed)
	}
	if opts.Mode.Has(ParseConfigDisabled) {
		m.Set(ConfigDisabled)
	}
	scanner.ErrorHandler(func(pos source.SourceFilePos, msg string) {
		p.Errors.Add(pos, msg)
	})
	p.Next()
	return p
}

func ParseFile(pth string, opts *ParserOptions, scannerOpts *ScannerOptions) (file *File, err error) {
	var (
		fileSet = source.NewFileSet()
		script  []byte
		srcFile *source.SourceFile
		f       *os.File
	)

	if f, err = os.Open(pth); err != nil {
		return
	}

	defer f.Close()

	if script, err = io.ReadAll(f); err != nil {
		return
	}

	srcFile = fileSet.AddFile(pth, -1, len(script))

	p := NewParserWithOptions(srcFile, script, opts, scannerOpts)
	return p.ParseFile()
}

// ParseFile parses the source and returns an AST file unit.
func (p *Parser) ParseFile() (file *File, err error) {
	defer func() {
		if e := recover(); e != nil {
			if _, ok := e.(bailout); !ok {
				panic(e)
			}
		}

		p.Errors.Sort()
		err = p.Errors.Err()
	}()

	if p.Trace {
		defer untracep(tracep(p, "File"))
	}

	if p.Errors.Len() > 0 {
		return nil, p.Errors.Err()
	}

	stmts := p.ParseStmtList()
	p.Expect(token.EOF)
	if p.Errors.Len() > 0 {
		return nil, p.Errors.Err()
	}

	file = &File{
		InputFile: p.File,
		Stmts:     stmts,
		Comments:  p.comments,
	}
	return
}

// ParseFileH parses the source and returns an AST file unit.
func (p *Parser) ParseFileH(listHandler ParseListHandler) (file *File, err error) {
	defer func() {
		if e := recover(); e != nil {
			if _, ok := e.(bailout); !ok {
				panic(e)
			}
		}

		p.Errors.Sort()
		err = p.Errors.Err()
	}()

	if p.Trace {
		defer untracep(tracep(p, "File"))
	}

	if p.Errors.Len() > 0 {
		return nil, p.Errors.Err()
	}

	stmts, _ := listHandler(0)
	p.Expect(token.EOF)
	if p.Errors.Len() > 0 {
		return nil, p.Errors.Err()
	}

	file = &File{
		InputFile: p.File,
		Stmts:     stmts,
		Comments:  p.comments,
	}
	return
}

func (p *Parser) ParseExpr() node.Expr {
	if p.Trace {
		defer untracep(tracep(p, "Expression"))
	}

	expr := p.ParseBinaryExpr(token.LowestPrec + 1)

	// ternary conditional expression
	if p.Token.Token == token.Question {
		return p.ParseCondExpr(expr)
	}
	return expr
}

func (p *Parser) ParseBinaryExpr(prec1 int) node.Expr {
	if p.Trace {
		defer untracep(tracep(p, "BinaryExpression"))
	}

	x := p.ParseUnaryExpr()

	for {
		op, prec := p.Token.Token, p.Token.Token.Precedence()
		if prec < prec1 {
			return x
		}

		pos := p.Expect(op)

		y := p.ParseBinaryExpr(prec + 1)

		if op == token.Equal || op == token.NotEqual {
			if _, ok := x.(*node.NilLit); ok {
				if op == token.Equal {
					op = token.Null
				} else {
					op = token.NotNull
				}
				x = &node.UnaryExpr{
					Expr:     y,
					Token:    op,
					TokenPos: pos,
				}
				continue
			} else if _, ok := y.(*node.NilLit); ok {
				if op == token.Equal {
					op = token.Null
				} else {
					op = token.NotNull
				}
				x = &node.UnaryExpr{
					Expr:     x,
					Token:    op,
					TokenPos: pos,
				}
				continue
			}
		}

		x = &node.BinaryExpr{
			LHS:      x,
			RHS:      y,
			Token:    op,
			TokenPos: pos,
		}
	}
}

func (p *Parser) ParseCondExpr(cond node.Expr) node.Expr {
	questionPos := p.Expect(token.Question)
	trueExpr := p.ParseExpr()
	falseExpr := trueExpr
	colonPos := questionPos

	if p.Token.Token == token.Colon {
		colonPos = p.Expect(token.Colon)
		falseExpr = p.ParseExpr()
	}

	return &node.CondExpr{
		Cond:        cond,
		True:        trueExpr,
		False:       falseExpr,
		QuestionPos: questionPos,
		ColonPos:    colonPos,
	}
}

func (p *Parser) ParseUnaryExpr() node.Expr {
	if p.Trace {
		defer untracep(tracep(p, "UnaryExpression"))
	}

	switch p.Token.Token {
	case token.Add, token.Sub, token.Not, token.Xor:
		pos, op := p.Token.Pos, p.Token.Token
		p.Next()
		x := p.ParseUnaryExpr()
		return &node.UnaryExpr{
			Token:    op,
			TokenPos: pos,
			Expr:     x,
		}
	}
	return p.ParsePrimaryExpr()
}

func (p *Parser) ParsePrimaryExpr() node.Expr {
	if p.Trace {
		defer untracep(tracep(p, "PrimaryExpression"))
	}

	x := p.ParseOperand()

	switch t := x.(type) {
	case *node.ParenExpr:
		if ti, _ := t.Expr.(*node.TypedIdent); ti != nil {
			if len(ti.Type) > 0 {
				p.Error(ti.Type[0].Pos(), "unexpected COLON")
				return x
			}
			t.Expr = ti.Ident
		}
	case *node.MultiParenExpr:
		for i, expr := range t.Exprs {
			if ti, _ := expr.(*node.TypedIdent); ti != nil {
				if len(ti.Type) > 0 {
					p.Error(ti.Type[0].Pos(), "unexpected COLON")
					return x
				}
				t.Exprs[i] = ti.Ident
			}
		}
	}

L:
	for {
		switch p.Token.Token {
		case token.NullishSelector:
			p.Next()

			switch p.Token.Token {
			case token.Ident, token.LParen, token.Else:
				x = p.ParseNullishSelector(x)
			default:
				pos := p.Token.Pos
				p.ErrorExpected(pos, "nullish selector")
				p.advance(stmtStart)
				return &node.BadExpr{From: pos, To: p.Token.Pos}
			}
		case token.Period:
			p.Next()

			switch p.Token.Token {
			case token.Ident, token.LParen, token.Else:
				x = p.ParseSelector(x)
			default:
				pos := p.Token.Pos
				p.ErrorExpected(pos, "selector")
				p.advance(stmtStart)
				return &node.BadExpr{From: pos, To: p.Token.Pos}
			}
		case token.LBrack:
			x = p.ParseIndexOrSlice(x)
		case token.LParen:
			x = p.ParseCall(x)
			if p.Token.Token == token.Period && p.pipes == 1 {
				return x
			}
		default:
			break L
		}
	}
	return x
}

func (p *Parser) ParseCall(x node.Expr) *node.CallExpr {
	if p.Trace {
		defer untracep(tracep(p, "Call"))
	}

	return &node.CallExpr{
		Func:     x,
		CallArgs: *p.ParseCallArgs(token.LParen, token.RParen),
	}
}

func (p *Parser) ParsePipe(x node.Expr) node.Expr {
	if p.Trace {
		defer untracep(tracep(p, "Pipe"))
	}
	name := p.ParseIdent()
	call := p.ParseCall(name)
	call.Args.Values = append([]node.Expr{x}, call.Args.Values...)
	if p.Token.Token == token.Period {
		p.Next()
		y := p.ParseSelector(call)
		return y
	}
	return call
}

func (p *Parser) CallArgsOf(lparen, rparen source.Pos, exprs ...node.Expr) (params *node.CallArgs) {
	params = &node.CallArgs{
		LParen: lparen,
		RParen: rparen,
	}

	var (
		i int
		n node.Expr
	)

exps:
	for _, n = range exprs {
		switch t := n.(type) {
		case *node.ArgVarLit:
			params.Args.Var = t
		case *node.KeyValueLit, *node.NamedArgVarLit:
			break exps
		default:
			params.Args.Values = append(params.Args.Values, t)
		}
		i++
	}

	if i < len(exprs) {
	nexps:
		for _, n = range exprs[i:] {
			switch t := n.(type) {
			case *node.KeyValueLit:
				switch t2 := t.Key.(type) {
				case *node.Ident:
					params.NamedArgs.Names = append(params.NamedArgs.Names, node.NamedArgExpr{Ident: t2})
				case *node.StringLit:
					params.NamedArgs.Names = append(params.NamedArgs.Names, node.NamedArgExpr{Lit: t2})
				default:
					p.ErrorExpected(t2.Pos(), "expected Ident | StringLit")
					return
				}
				params.NamedArgs.Values = append(params.NamedArgs.Values, t.Value)
			case *node.NamedArgVarLit:
				params.NamedArgs.Var = t
				i++
				break nexps
			default:
				p.ErrorExpected(t.Pos(), "expected KeyValueLit | NamedArgVarLit")
				return
			}
			i++
		}

		if i < len(exprs) {
			p.Error(exprs[i].Pos(), fmt.Sprintf("unexpected expr %s %[1]T", exprs[1]))
		}
	}

	return
}

func (p *Parser) ParseCallArgs(start, end token.Token) *node.CallArgs {
	if p.Trace {
		defer untracep(tracep(p, "CallArgs"))
	}

	paren := p.ParseParemExpr(start, end, false, true, false)
	switch t := paren.(type) {
	case *node.ParenExpr:
		return p.CallArgsOf(t.LParen, t.RParen, t.Expr)
	case *node.MultiParenExpr:
		return p.CallArgsOf(t.LParen, t.RParen, t.Exprs...)
	case *node.KeyValueArrayLit:
		var exprs = make([]node.Expr, len(t.Elements))
		for i, el := range t.Elements {
			exprs[i] = el
		}
		return p.CallArgsOf(t.LBrace, t.RBrace, exprs...)
	default:
		if t == nil {
			return &node.CallArgs{}
		}
		return &node.CallArgs{
			LParen: t.Pos(),
			RParen: t.End(),
		}
	}
}

func (p *Parser) AtComma(context string, follow token.Token) bool {
	if p.Token.Token == token.Comma {
		return true
	}
	if p.Token.Token != follow {
		msg := "missing ','"
		if p.Token.Token == token.Semicolon && p.Token.Literal == "\n" {
			msg += " before newline"
		}
		p.Error(p.Token.Pos, msg+" in "+context)
		return true // "insert" comma and continue
	}
	return false
}

func (p *Parser) ParseIndexOrSlice(x node.Expr) node.Expr {
	if p.Trace {
		defer untracep(tracep(p, "IndexOrSlice"))
	}

	lbrack := p.Expect(token.LBrack)
	p.ExprLevel++

	var index [2]node.Expr
	if p.Token.Token != token.Colon {
		index[0] = p.ParseExpr()
	}
	numColons := 0
	if p.Token.Token == token.Colon {
		numColons++
		p.Next()

		if p.Token.Token != token.RBrack && p.Token.Token != token.EOF {
			index[1] = p.ParseExpr()
		}
	}

	p.ExprLevel--
	rbrack := p.Expect(token.RBrack)

	if numColons > 0 {
		// slice expression
		return &node.SliceExpr{
			Expr:   x,
			LBrack: lbrack,
			RBrack: rbrack,
			Low:    index[0],
			High:   index[1],
		}
	}
	return &node.IndexExpr{
		Expr:   x,
		LBrack: lbrack,
		RBrack: rbrack,
		Index:  index[0],
	}
}

func (p *Parser) ParseSelectorNode(x node.Expr) (expr, sel node.Expr) {
	switch p.Token.Token {
	case token.LParen:
		lparen := p.Token.Pos
		p.Next()
		sel = p.ParseExpr()
		rparen := p.Expect(token.RParen)
		sel = &node.ParenExpr{Expr: sel, LParen: lparen, RParen: rparen}
	case token.Else:
		name := p.Token.Token.String()
		sel = &node.StringLit{
			Value:    name,
			ValuePos: p.Token.Pos,
			Literal:  name,
		}
		p.Next()
	default:
		ident := p.ParseIdent()
		sel = &node.StringLit{
			Value:    ident.Name,
			ValuePos: ident.NamePos,
			Literal:  ident.Name,
		}
	}
	expr = x
	return
}

func (p *Parser) ParseSelector(x node.Expr) (sel node.Expr) {
	if p.Trace {
		defer untracep(tracep(p, "Selector"))
	}

	x, sel = p.ParseSelectorNode(x)
	return &node.SelectorExpr{Expr: x, Sel: sel}
}

func (p *Parser) ParseNullishSelector(x node.Expr) (sel node.Expr) {
	if p.Trace {
		defer untracep(tracep(p, "NullishSelector"))
	}

	x, sel = p.ParseSelectorNode(x)
	return &node.NullishSelectorExpr{Expr: x, Sel: sel}
}

func (p *Parser) ParseStringLit() *node.StringLit {
	v, _ := Unquote(p.Token.Literal)
	x := &node.StringLit{
		Value:    v,
		ValuePos: p.Token.Pos,
		Literal:  p.Token.Literal,
	}
	p.Next()
	return x
}

func (p *Parser) ParsePrimitiveOperand() node.Expr {
	switch p.Token.Token {
	case token.Ident:
		return p.ParseIdent()
	case token.Int:
		v, _ := strconv.ParseInt(p.Token.Literal, 0, 64)
		x := &node.IntLit{
			Value:    v,
			ValuePos: p.Token.Pos,
			Literal:  p.Token.Literal,
		}
		p.Next()
		return x
	case token.Uint:
		v, _ := strconv.ParseUint(strings.TrimSuffix(p.Token.Literal, "u"), 0, 64)
		x := &node.UintLit{
			Value:    v,
			ValuePos: p.Token.Pos,
			Literal:  p.Token.Literal,
		}
		p.Next()
		return x
	case token.Float:
		v, _ := strconv.ParseFloat(p.Token.Literal, 64)
		x := &node.FloatLit{
			Value:    v,
			ValuePos: p.Token.Pos,
			Literal:  p.Token.Literal,
		}
		p.Next()
		return x
	case token.Char:
		return p.ParseCharLit()
	case token.String:
		return p.ParseStringLit()
	case token.True, token.False:
		x := &node.BoolLit{
			Value:    p.Token.Token == token.True,
			ValuePos: p.Token.Pos,
			Literal:  p.Token.Literal,
		}
		p.Next()
		return x
	case token.Yes, token.No:
		x := &node.FlagLit{
			ValuePos: p.Token.Pos,
			Literal:  p.Token.Literal,
			Value:    p.Token.Token == token.Yes,
		}
		p.Next()
		return x
	case token.Nil:
		x := &node.NilLit{TokenPos: p.Token.Pos}
		p.Next()
		return x
	case token.Callee:
		x := &node.CalleeKeyword{TokenPos: p.Token.Pos, Literal: p.Token.Literal}
		p.Next()
		return x
	case token.Args:
		x := &node.ArgsKeyword{TokenPos: p.Token.Pos, Literal: p.Token.Literal}
		p.Next()
		return x
	case token.NamedArgs:
		x := &node.NamedArgsKeyword{TokenPos: p.Token.Pos, Literal: p.Token.Literal}
		p.Next()
		return x
	case token.StdIn:
		x := &node.StdInLit{TokenPos: p.Token.Pos}
		p.Next()
		return x
	case token.StdOut:
		x := &node.StdOutLit{TokenPos: p.Token.Pos}
		p.Next()
		return x
	case token.StdErr:
		x := &node.StdErrLit{TokenPos: p.Token.Pos}
		p.Next()
		return x
	case token.DotName:
		x := &node.DotFileNameLit{TokenPos: p.Token.Pos}
		p.Next()
		return x
	case token.DotFile:
		x := &node.DotFileLit{TokenPos: p.Token.Pos}
		p.Next()
		return x
	case token.IsModule:
		x := &node.IsModuleLit{TokenPos: p.Token.Pos}
		p.Next()
		return x
	}

	pos := p.Token.Pos
	p.ErrorExpected(pos, "primitive operand")
	p.advance(stmtStart)
	return &node.BadExpr{From: pos, To: p.Token.Pos}
}

func (p *Parser) ParseOperand() node.Expr {
	if p.Trace {
		defer untracep(tracep(p, "Operand"))
	}

	switch p.Token.Token {
	case token.Ident:
		return p.ParseIdent()
	case token.Int:
		v, _ := strconv.ParseInt(p.Token.Literal, 0, 64)
		x := &node.IntLit{
			Value:    v,
			ValuePos: p.Token.Pos,
			Literal:  p.Token.Literal,
		}
		p.Next()
		return x
	case token.Uint:
		v, _ := strconv.ParseUint(strings.TrimSuffix(p.Token.Literal, "u"), 0, 64)
		x := &node.UintLit{
			Value:    v,
			ValuePos: p.Token.Pos,
			Literal:  p.Token.Literal,
		}
		p.Next()
		return x
	case token.Float:
		v, _ := strconv.ParseFloat(p.Token.Literal, 64)
		x := &node.FloatLit{
			Value:    v,
			ValuePos: p.Token.Pos,
			Literal:  p.Token.Literal,
		}
		p.Next()
		return x
	case token.Decimal:
		v, err := decimal.NewFromString(strings.TrimSuffix(p.Token.Literal, "d"))
		if err != nil {
			p.Error(p.Token.Pos, err.Error())
		}
		x := &node.DecimalLit{
			Value:    v,
			ValuePos: p.Token.Pos,
			Literal:  p.Token.Literal,
		}
		p.Next()
		return x
	case token.Char:
		return p.ParseCharLit()
	case token.String:
		return p.ParseStringLit()
	case token.True, token.False:
		x := &node.BoolLit{
			Value:    p.Token.Token == token.True,
			ValuePos: p.Token.Pos,
			Literal:  p.Token.Literal,
		}
		p.Next()
		return x
	case token.Yes, token.No:
		x := &node.FlagLit{
			ValuePos: p.Token.Pos,
			Literal:  p.Token.Literal,
			Value:    p.Token.Token == token.Yes,
		}
		p.Next()
		return x
	case token.Nil:
		x := &node.NilLit{TokenPos: p.Token.Pos}
		p.Next()
		return x
	case token.StdIn:
		x := &node.StdInLit{TokenPos: p.Token.Pos}
		p.Next()
		return x
	case token.StdOut:
		x := &node.StdOutLit{TokenPos: p.Token.Pos}
		p.Next()
		return x
	case token.StdErr:
		x := &node.StdErrLit{TokenPos: p.Token.Pos}
		p.Next()
		return x
	case token.DotName:
		x := &node.DotFileNameLit{TokenPos: p.Token.Pos}
		p.Next()
		return x
	case token.DotFile:
		x := &node.DotFileLit{TokenPos: p.Token.Pos}
		p.Next()
		return x
	case token.IsModule:
		x := &node.IsModuleLit{TokenPos: p.Token.Pos}
		p.Next()
		return x
	case token.Callee:
		x := &node.CalleeKeyword{TokenPos: p.Token.Pos, Literal: p.Token.Literal}
		p.Next()
		return x
	case token.Args:
		x := &node.ArgsKeyword{TokenPos: p.Token.Pos, Literal: p.Token.Literal}
		p.Next()
		return x
	case token.NamedArgs:
		x := &node.NamedArgsKeyword{TokenPos: p.Token.Pos, Literal: p.Token.Literal}
		p.Next()
		return x
	case token.Import:
		return p.ParseImportExpr()
	case token.LParen:
		return p.ParseParemExpr(token.LParen, token.RParen, true, true, true)
	case token.LBrack: // array literal
		return p.ParseArrayLitOrKeyValue()
	case token.LBrace: // dict literal
		return p.ParseDictLit()
	case token.Func: // function literal
		return p.ParseFuncLit()
	case token.RawString:
		return p.ParseRawStringLit()
	case token.RawHeredoc:
		return p.ParseRawHeredocLit()
	case token.Throw:
		return p.ParseThrowExpr()
	case token.Return:
		return p.ParseReturnExpr()
	case token.Template:
		pos := p.Token.Pos
		p.Next()
		switch p.Token.Token {
		case token.String, token.RawString, token.RawHeredoc:
			return &node.TemplateLit{
				TokenPos: pos,
				Value:    p.ParseOperand(),
			}
		}
	}

	pos := p.Token.Pos
	p.ErrorExpected(pos, "operand")
	p.advance(stmtStart)
	return &node.BadExpr{From: pos, To: p.Token.Pos}
}

func (p *Parser) ParseImportExpr() node.Expr {
	pos := p.Token.Pos
	p.Next()
	p.Expect(token.LParen)
	if p.Token.Token != token.String {
		p.ErrorExpected(p.Token.Pos, "module name")
		p.advance(stmtStart)
		return &node.BadExpr{From: pos, To: p.Token.Pos}
	}

	// module name
	moduleName, _ := strconv.Unquote(p.Token.Literal)
	expr := &node.ImportExpr{
		ModuleName: moduleName,
		Token:      token.Import,
		TokenPos:   pos,
	}

	p.Next()
	p.Expect(token.RParen)
	return expr
}

func (p *Parser) ParseParemExpr(lparenToken, rparenToken token.Token, acceptKv, parseLambda, parseTypes bool) node.Expr {
	if p.Trace {
		defer untracep(tracep(p, "ParemExpr"))
	}

	var (
		lparen = p.Token.Pos
		end    = rparenToken
		kv     bool
	)
	switch p.Token.Token {
	case lparenToken:
	default:
		p.ErrorExpected(lparen, "'"+lparenToken.String()+"'")
		return nil
	}
	p.Next()

	var (
		exprs []node.Expr
		expr  node.Expr
	)

	switch p.Token.Token {
	case token.Semicolon:
		if p.Token.Literal == ";" {
			if acceptKv {
				return p.ParseKeyValueArrayLit(lparen)
			}
			kv = true
			p.Next()
		}
	case token.MixedCodeEnd:
		mte := &node.MixedTextExpr{
			StartLit: ast.Literal{Value: p.Token.Literal, Pos: p.Token.Pos},
		}
		p.Next()
		stmt := p.ParseMixedTextStmt()
		p.ExprLevel++
		mcs := p.ExpectToken(token.MixedCodeStart)
		mte.EndLit = ast.Literal{Value: mcs.Literal, Pos: mcs.Pos}
		mte.Stmt = *stmt
		exprs = append(exprs, mte)
		p.ExprLevel--
		goto end
	}

	p.SkipSpace()

	for p.Token.Token != end {
		var (
			pos = p.Token.Pos
			mul int
		)

		if p.Token.Token == token.Mul {
			mul++
			p.Next()

			if p.Token.Token == token.Mul {
				mul++
				p.Next()
			}
			p.SkipSpace()
		}

		p.ExprLevel++
		expr = p.ParseExpr()
		p.ExprLevel--
		p.SkipSpace()

		if ident, _ := expr.(*node.Ident); ident != nil && parseTypes {
			expr = &node.TypedIdent{
				Ident: ident,
				Type:  p.ParseType(),
			}
		}

		if kv {
			if mul != 2 {
				switch expr.(type) {
				case *node.TypedIdent, *node.Ident, *node.StringLit:
					kv := &node.KeyValueLit{
						Key: expr,
					}
					if p.Token.Token == token.Assign {
						p.Next()
						kv.Value = p.ParseExpr()
					}
					expr = kv
					goto add
				}
			}
		}

		switch mul {
		case 1:
			expr = &node.ArgVarLit{
				TokenPos: pos,
				Value:    expr,
			}
		case 2:
			expr = &node.NamedArgVarLit{
				TokenPos: pos,
				Value:    expr,
			}
		default:
			if p.Token.Token == token.Assign {
				p.Next()
				p.ExprLevel++
				expr = &node.KeyValueLit{
					Key:   expr,
					Value: p.ParseExpr(),
				}
				p.ExprLevel--
			}
		}

	add:

		exprs = append(exprs, expr)

		if p.Token.Token == token.Comma {
			p.Next()
		} else if p.Token.Token == token.Semicolon && p.Token.Literal == ";" && !kv {
			kv = true
			p.Next()
		} else {
			break
		}
	}

end:

	rparen := p.Expect(end)

	if p.Token.Token == token.Lambda && parseLambda {
		p.Next()

		var body node.Expr
		if p.Token.Token.IsBlockStart() {
			body = &node.BlockExpr{BlockStmt: p.ParseBlockStmt()}
		} else {
			body = p.ParseExpr()
		}

		expr = &node.ClosureLit{
			Type: &node.FuncType{
				FuncPos: lparen,
				Params:  *p.FuncParamsOf(lparen, rparen, exprs...),
			},
			Body: body,
		}
	} else if len(exprs) == 1 {
		expr = &node.ParenExpr{
			LParen: lparen,
			Expr:   exprs[0],
			RParen: rparen,
		}
	} else {
		expr = &node.MultiParenExpr{
			LParen: lparen,
			Exprs:  exprs,
			RParen: rparen,
		}
	}

	return expr
}

func (p *Parser) FuncParamsOf(lparen, rparen source.Pos, exprs ...node.Expr) (params *node.FuncParams) {
	params = &node.FuncParams{
		LParen: lparen,
		RParen: rparen,
	}

	var (
		i int
		n node.Expr
	)

exps:
	for _, n = range exprs {
		switch t := n.(type) {
		case *node.Ident:
			params.Args.Values = append(params.Args.Values, &node.TypedIdent{Ident: t})
		case *node.TypedIdent:
			params.Args.Values = append(params.Args.Values, t)
		case *node.ArgVarLit:
			switch t2 := t.Value.(type) {
			case *node.Ident:
				params.Args.Var = &node.TypedIdent{
					Ident: t2,
				}
				i++
				break exps
			case *node.TypedIdent:
				params.Args.Var = t2
				i++
				break exps
			default:
				p.ErrorExpectedExpr(&node.Ident{}, t.Value)
			}
		case *node.KeyValueLit, *node.NamedArgVarLit:
			break exps
		default:
			p.ErrorExpected(t.Pos(), fmt.Sprintf("Ident|keyValueLit, but got %T", n))
			return
		}
		i++
	}

	if i < len(exprs) {
	nexps:
		for _, n = range exprs[i:] {
			switch t := n.(type) {
			case *node.KeyValueLit:
				switch t2 := t.Key.(type) {
				case *node.Ident:
					params.NamedArgs.Names = append(params.NamedArgs.Names, &node.TypedIdent{Ident: t2})
					params.NamedArgs.Values = append(params.NamedArgs.Values, t.Value)
				case *node.TypedIdent:
					params.NamedArgs.Names = append(params.NamedArgs.Names, t2)
					params.NamedArgs.Values = append(params.NamedArgs.Values, t.Value)
				default:
					p.ErrorExpected(t2.Pos(), "expected Ident")
					return
				}
			case *node.NamedArgVarLit:
				switch t2 := t.Value.(type) {
				case *node.Ident:
					params.NamedArgs.Var = &node.TypedIdent{Ident: t2}
					i++
					break nexps
				case *node.TypedIdent:
					params.NamedArgs.Var = t2
					i++
					break nexps
				default:
					p.ErrorExpectedExpr(&node.Ident{}, t.Value)
					return
				}
			default:
				p.ErrorExpected(t.Pos(), "expected Ident or keyValueLit")
				return
			}
			i++
		}

		if i < len(exprs) {
			p.Error(exprs[i].Pos(), fmt.Sprintf("unexpected expr %s %[1]T", exprs[1]))
		}
	}

	return
}

func (p *Parser) ParseCharLit() node.Expr {
	if n := len(p.Token.Literal); n >= 3 {
		code, _, _, err := strconv.UnquoteChar(p.Token.Literal[1:n-1], '\'')
		if err == nil {
			x := &node.CharLit{
				Value:    code,
				ValuePos: p.Token.Pos,
				Literal:  p.Token.Literal,
			}
			p.Next()
			return x
		}
	}

	pos := p.Token.Pos
	p.Error(pos, "illegal char literal")
	p.Next()
	return &node.BadExpr{
		From: pos,
		To:   p.Token.Pos,
	}
}

func (p *Parser) ParseFuncLit() node.Expr {
	if p.Trace {
		defer untracep(tracep(p, "FuncLit"))
	}

	typ := p.ParseFuncType(false)
	p.ExprLevel++

	body, closure := p.ParseBody()
	p.ExprLevel--
	if closure != nil {
		body = &node.BlockStmt{
			Stmts: []node.Stmt{&node.ReturnStmt{
				Return: node.Return{
					Result: closure,
				},
			}},
		}
	}
	return &node.FuncLit{
		Type: typ,
		Body: body,
	}
}

func (p *Parser) ParseArrayLitOrKeyValue() node.Expr {
	if p.Trace {
		defer untracep(tracep(p, "ArrayLitOrKeyValue"))
	}

	lbrack := p.Expect(token.LBrack)
	p.ExprLevel++

	var (
		elements []node.Expr
	)

	if p.Token.Token != token.RBrack && p.Token.Token != token.EOF {
		elements = append(elements, p.ParseExpr())
		if p.Token.Token == token.Assign {
			p.Next()
			expr := &node.KeyValueLit{
				Key:   elements[0],
				Value: p.ParseExpr(),
			}
			p.Expect(token.RBrack)
			return expr
		}

		if p.AtComma("array literal", token.RBrack) {
			p.Next()
		}
	}

	for p.Token.Token != token.RBrack && p.Token.Token != token.EOF {
		elements = append(elements, p.ParseExpr())

		if !p.AtComma("array literal", token.RBrack) {
			break
		}
		p.Next()
	}

	p.ExprLevel--
	rbrack := p.Expect(token.RBrack)
	return &node.ArrayLit{
		Elements: elements,
		LBrack:   lbrack,
		RBrack:   rbrack,
	}
}

func (p *Parser) ParseFuncType(parseLambda bool) *node.FuncType {
	if p.Trace {
		defer untracep(tracep(p, "FuncType"))
	}

	tok := p.Token.Token

	var (
		pos          = p.Expect(token.Func)
		ident        *node.Ident
		allowMethods bool
	)

	if p.Token.Token == token.Ident {
		ident = p.ParseIdent()
		allowMethods = true
	}

	params := p.ParseFuncParams(parseLambda)
	return &node.FuncType{
		Token:        tok,
		FuncPos:      pos,
		Ident:        ident,
		Params:       *params,
		AllowMethods: allowMethods,
	}
}

func (p *Parser) ParseFuncParams(parseLambda bool) *node.FuncParams {
	if p.Trace {
		defer untracep(tracep(p, "FuncParams"))
	}

	paren := p.ParseParemExpr(token.LParen, token.RParen, false, parseLambda, true)
	switch t := paren.(type) {
	case *node.ParenExpr:
		return p.FuncParamsOf(t.LParen, t.RParen, t.Expr)
	case *node.MultiParenExpr:
		return p.FuncParamsOf(t.LParen, t.RParen, t.Exprs...)
	default:
		return &node.FuncParams{
			LParen: t.Pos(),
			RParen: t.End(),
		}
	}
}

func (p *Parser) ParseBody() (b *node.BlockStmt, closure node.Expr) {
	if p.Trace {
		defer untracep(tracep(p, "Body"))
	}

	p.SkipSpace()

	if p.Token.Token == token.Lambda {
		p.Next()
		if p.Token.Token.IsBlockStart() {
			closure = &node.BlockExpr{BlockStmt: p.ParseBlockStmt()}
		} else {
			closure = p.ParseExpr()
		}
	} else {
		b = p.ParseBlockStmt()
	}
	return
}

func (p *Parser) ParseStmtList(end ...token.Token) (list []node.Stmt) {
	if p.Trace {
		defer untracep(tracep(p, "StatementList"))
	}

	var s node.Stmt

	for {
		switch p.Token.Token {
		case token.EOF, token.RBrace:
			return
		case token.Semicolon:
			p.Next()
		default:
			for _, t := range end {
				if t == p.Token.Token {
					return
				}
			}
			if s = p.ParseStmt(); s != nil {
				if _, ok := s.(*node.EmptyStmt); ok {
					continue
				}
				list = append(list, s)
			}
		}
	}
}

func (p *Parser) ParseIdent() *node.Ident {
	pos := p.Token.Pos
	name := "_"

	if p.Token.Token == token.Ident {
		name = p.Token.Literal
		p.Next()
	} else {
		p.Expect(token.Ident)
	}
	return &node.Ident{
		NamePos: pos,
		Name:    name,
	}
}

func (p *Parser) ParseTypedIdent() *node.TypedIdent {
	return &node.TypedIdent{
		Ident: p.ParseIdent(),
		Type:  p.ParseType(),
	}
}

func (p *Parser) ParseType() (idents []*node.Ident) {
	if p.Token.Token == token.Ident {
		var (
			exists = map[string]any{}
			add    = func(ident *node.Ident) {
				if _, ok := exists[ident.Name]; !ok {
					idents = append(idents, ident)
					exists[ident.Name] = nil
				}
			}
		)

		add(p.ParseIdent())
		for p.Token.Token == token.Or {
			p.Next()
			add(p.ParseIdent())
		}
	}
	return
}

func (p *Parser) ParseStmt() (stmt node.Stmt) {
	if p.Trace {
		defer untracep(tracep(p, "Statement"))
	}

	tok := p.Token
	defer func() {
		if p.Token.Token == tok.Token && p.Token.Pos == tok.Pos {
			p.Next()
		}
	}()

	stmt = p.ParseStmtHandler()

	if stmt == nil {
		pos := p.Token.Pos
		p.ErrorExpected(pos, "statement")
		p.advance(stmtStart)
		stmt = &node.BadStmt{From: pos, To: p.Token.Pos}
	}
	return
}

func (p *Parser) DefaultParseStmt() (stmt node.Stmt) {
do:
	switch p.Token.Token {
	case token.ConfigStart:
		return p.ParseConfigStmt()
	case token.MixedText:
		return p.ParseMixedTextStmt()
	case token.MixedCodeStart:
		n := &node.CodeBeginStmt{
			Lit: ast.Literal{
				Value: p.Token.Literal,
				Pos:   p.Token.Pos,
			},
			RemoveSpace: RemoveSpaces(p.Token),
		}
		p.Next()
		return n
	case token.MixedCodeEnd:
		n := &node.CodeEndStmt{
			Lit: ast.Literal{
				Value: p.Token.Literal,
				Pos:   p.Token.Pos,
			},
			RemoveSpace: RemoveSpaces(p.Token),
		}
		p.Next()
		return n
	case token.MixedValueStart:
		return p.ParseMixedValue()
	case token.Var, token.Const, token.Global, token.Param:
		return &node.DeclStmt{Decl: p.ParseDecl()}
	case // simple statements
		token.Func, token.Ident, token.Int, token.Uint, token.Float,
		token.Char, token.String, token.True, token.False, token.Nil,
		token.LParen, token.LBrace, token.LBrack, token.Add, token.Sub,
		token.Mul, token.And, token.Xor, token.Not, token.Import,
		token.Callee, token.Args, token.NamedArgs,
		token.StdIn, token.StdOut, token.StdErr,
		token.Yes, token.No,
		token.DotName, token.DotFile, token.IsModule, token.Template:
		s := p.ParseSimpleStmt(false)
		p.ExpectSemi()
		return s
	case token.Return:
		return p.ParseReturnStmt()
	case token.If:
		return p.ParseIfStmt()
	case token.For:
		return p.ParseForStmt()
	case token.Try:
		return p.ParseTryStmt()
	case token.Throw:
		return p.ParseThrowStmt()
	case token.Break, token.Continue:
		return p.ParseBranchStmt(p.Token.Token)
	case token.Semicolon:
		p.Next()
		goto do
	case token.RBrace:
		// semicolon may be omitted before a closing "}"
		return &node.EmptyStmt{Semicolon: p.Token.Pos, Implicit: true}
	}

	return
}

func (p *Parser) ParseConfigStmt() (c *node.ConfigStmt) {
	if p.Trace {
		defer untracep(tracep(p, "ConfigStmt"))
	}

	c = &node.ConfigStmt{
		ConfigPos: p.Token.Pos,
	}

	p.Next()

	kva := p.ParseKeyValueArrayLitAt(p.Token.Pos, token.ConfigEnd)

	c.Elements = kva.Elements
	c.ParseElements()

	if c.Options.Mixed {
		p.Scanner.SetMode(p.Scanner.Mode() | Mixed)
	} else if c.Options.NoMixed {
		p.Scanner.SetMode(p.Scanner.Mode() &^ Mixed)
	}

	if c.Options.MixedStart != "" {
		p.Scanner.GetMixedDelimiter().Start = []rune(c.Options.MixedStart)
	}
	if c.Options.MixedEnd != "" {
		p.Scanner.GetMixedDelimiter().End = []rune(c.Options.MixedEnd)
	}

	p.Expect(token.ConfigEnd)
	return
}

func (p *Parser) ParseMixedValue() (ett *node.MixedValueStmt) {
	if p.Trace {
		defer untracep(tracep(p, "MixedValueStmt"))
	}
	ett = &node.MixedValueStmt{
		StartLit:        ast.Literal{Value: p.Token.Literal, Pos: p.Token.Pos},
		RemoveLeftSpace: RemoveSpaces(p.Token),
		Eq:              p.Token.Data.Flag("eq"),
	}
	p.Next()
	ett.Expr = p.ParseExpr()
	end := p.ExpectToken(token.MixedValueEnd)
	ett.RemoveRightSpace = RemoveSpaces(end)
	ett.EndLit = ast.Literal{Value: end.Literal, Pos: end.Pos}
	return
}

func (p *Parser) ParseRawStringLit() (t *node.RawStringLit) {
	if p.Trace {
		defer untracep(tracep(p, "RawStringLit"))
	}
	t = &node.RawStringLit{
		Literal:    p.Token.Literal,
		LiteralPos: p.Token.Pos,
		Quoted:     p.Token.Literal[0] == '`',
	}
	p.Next()
	return
}

func (p *Parser) ParseMixedTextStmt() (t *node.MixedTextStmt) {
	if p.Trace {
		defer untracep(tracep(p, "MixedTextStmt"))
	}

	t = &node.MixedTextStmt{
		Lit: ast.Literal{
			Value: p.Token.Literal,
			Pos:   p.Token.Pos,
		},
	}

	switch p.PrevToken.Token {
	case token.MixedValueEnd, token.MixedCodeEnd:
		if RemoveSpaces(p.PrevToken) {
			t.Flags |= node.RemoveLeftSpaces
		}
	}

	p.Next()

	switch p.Token.Token {
	case token.MixedCodeStart, token.MixedValueStart:
		if RemoveSpaces(p.Token) {
			t.Flags |= node.RemoveRightSpaces
		}
	}

	return
}

func (p *Parser) ParseRawHeredocLit() (t *node.RawHeredocLit) {
	if p.Trace {
		defer untracep(tracep(p, "RawHeredocLit"))
	}
	t = &node.RawHeredocLit{
		Literal:    p.Token.Literal,
		LiteralPos: p.Token.Pos,
	}
	p.Next()
	return
}

func (p *Parser) ParseDecl() node.Decl {
	if p.Trace {
		defer untracep(tracep(p, "DeclStmt"))
	}
	switch p.Token.Token {
	case token.Param:
		return p.ParseParamDecl()
	case token.Global:
		return p.ParseGenDecl(p.Token.Token, p.ParseParamSpec)
	case token.Var, token.Const:
		return p.ParseGenDecl(p.Token.Token, p.ParseValueSpec)
	default:
		p.Error(p.Token.Pos, "only \"param, global, var\" declarations supported")
		return &node.BadDecl{From: p.Token.Pos, To: p.Token.Pos}
	}
}

func (p *Parser) ParseParamDecl() (d *node.GenDecl) {
	if p.Trace {
		defer untracep(tracep(p, "ParamDecl"))
	}

	d = &node.GenDecl{
		Tok:    p.Token.Token,
		TokPos: p.Token.Pos,
	}

	p.Next()

	switch p.Token.Token {
	case token.Mul:
		p.Next()
		if p.Token.Token == token.Mul {
			p.Next()
			d.Specs = append(d.Specs, &node.NamedParamSpec{
				Ident: p.ParseTypedIdent(),
			})
		} else {
			d.Specs = append(d.Specs, &node.ParamSpec{
				Ident:    p.ParseTypedIdent(),
				Variadic: true,
			})
		}
	case token.Ident:
		ident := p.ParseIdent()
		types := p.ParseType()

		if p.Token.Token == token.Assign {
			p.Next()
			d.Specs = append(d.Specs, &node.NamedParamSpec{
				Ident: &node.TypedIdent{
					Ident: ident,
					Type:  types,
				},
				Value: p.ParseExpr(),
			})
		} else {
			d.Specs = append(d.Specs, &node.ParamSpec{
				Ident: &node.TypedIdent{
					Ident: ident,
					Type:  types,
				},
			})
		}
	case token.LParen:
		fp := p.ParseFuncParams(false)
		d.Lparen = fp.LParen
		d.Rparen = fp.RParen

		for _, value := range fp.Args.Values {
			d.Specs = append(d.Specs, &node.ParamSpec{
				Ident: value,
			})
		}

		if fp.Args.Var != nil {
			d.Specs = append(d.Specs, &node.ParamSpec{
				Ident:    fp.Args.Var,
				Variadic: true,
			})
		}

		for i, name := range fp.NamedArgs.Names {
			d.Specs = append(d.Specs, &node.NamedParamSpec{
				Ident: name,
				Value: fp.NamedArgs.Values[i],
			})
		}

		if fp.NamedArgs.Var != nil {
			d.Specs = append(d.Specs, &node.NamedParamSpec{
				Ident: fp.NamedArgs.Var,
			})
		}
	}

	return
}

func (p *Parser) ParseGenDecl(
	keyword token.Token,
	fn func(token.Token, bool, []node.Spec, int) node.Spec,
) *node.GenDecl {
	if p.Trace {
		defer untracep(tracep(p, "GenDecl("+keyword.String()+")"))
	}
	pos := p.Expect(keyword)
	var lparen, rparen source.Pos
	var list []node.Spec
	if p.Token.Token == token.LParen {
		lparen = p.Token.Pos
		p.Next()
		for i := 0; p.Token.Token != token.RParen && p.Token.Token != token.EOF; i++ { //nolint:predeclared
			list = append(list, fn(keyword, true, list, i))
		}
		rparen = p.Expect(token.RParen)
		p.ExpectSemi()
	} else {
		list = append(list, fn(keyword, false, list, 0))
		p.ExpectSemi()
	}

	for _, spec := range list {
		if vs, _ := spec.(*node.ValueSpec); vs != nil {
			if len(vs.Values) == 1 {
				switch fn := vs.Values[0].(type) {
				case *node.ClosureLit:
					fn.Type.Token = keyword
					if keyword == token.Const {
						fn.Type.Ident = vs.Idents[0]
					}
				case *node.FuncLit:
					fn.Type.Token = keyword
					if keyword == token.Const {
						fn.Type.Ident = vs.Idents[0]
					}
				}
			}
		}
	}

	return &node.GenDecl{
		TokPos: pos,
		Tok:    keyword,
		Lparen: lparen,
		Specs:  list,
		Rparen: rparen,
	}
}

func (p *Parser) ParseParamSpec(keyword token.Token, multi bool, prev []node.Spec, i int) (spec node.Spec) {
	if p.Trace {
		defer untracep(tracep(p, keyword.String()+"Spec"))
	}

	if multi {
		p.SkipSpace()
	}

	var (
		pos = p.Token.Pos

		ident    *node.TypedIdent
		variadic bool
		named    bool
		value    node.Expr
	)

	if p.Token.Token == token.Semicolon && p.Token.Literal == ";" {
		p.Next()
		if multi {
			p.SkipSpace()
		}
		named = true
	} else if i > 0 {
		switch t := prev[i-1].(type) {
		case *node.NamedParamSpec:
			if t.Value == nil {
				p.Error(pos, "unexpected arg declaration")
				p.ExpectSemi()
			}
			named = true
		case *node.ParamSpec:
			if t.Variadic {
				named = true
			}
		}
	}

	if p.Token.Token == token.Ident {
		ident = p.ParseTypedIdent()
	} else if keyword == token.Param && p.Token.Token == token.Mul {
		variadic = true
		p.Next()
		if p.Token.Token == token.Mul {
			named = true
			p.Next()
		}
		ident = p.ParseTypedIdent()
		if multi {
			p.SkipSpace()
		}
	}

	if multi && p.Token.Token == token.Comma {
		p.Next()
		p.SkipSpace()
	} else if multi {
		if p.Token.Token == token.Assign {
			named = true
			p.Next()
			value = p.ParseExpr()
			if p.Token.Token == token.Comma || (p.Token.Token == token.Semicolon && p.Token.Literal == "\n") {
				p.Next()
				p.SkipSpace()
			}
		} else if !named {
			p.ExpectSemi()
		}
	}

	if ident == nil {
		p.Error(pos, fmt.Sprintf("wrong %s declaration", keyword.String()))
		p.ExpectSemi()
	}

	if named {
		if value == nil && !variadic {
			p.Error(pos, fmt.Sprintf("wrong %s declaration", keyword.String()))
		}
		return &node.NamedParamSpec{
			Ident: ident,
			Value: value,
		}
	}

	return &node.ParamSpec{
		Ident:    ident,
		Variadic: variadic,
	}
}

func (p *Parser) ParseValueSpec(keyword token.Token, multi bool, _ []node.Spec, i int) node.Spec {
	if p.Trace {
		defer untracep(tracep(p, keyword.String()+"Spec"))
	}
	pos := p.Token.Pos
	var idents []*node.Ident
	var values []node.Expr
	if p.Token.Token == token.Ident {
		ident := p.ParseIdent()
		var expr node.Expr
		if p.Token.Token == token.Assign {
			p.Next()
			expr = p.ParseExpr()
		}
		if keyword == token.Const && expr == nil {
			if i == 0 {
				p.Error(p.Token.Pos, "missing initializer in const declaration")
			}
		}
		idents = append(idents, ident)
		values = append(values, expr)
		if multi && p.Token.Token == token.Comma {
			p.Next()
		} else if multi {
			p.ExpectSemi()
		}
	}
	if len(idents) == 0 {
		p.Error(pos, "wrong var declaration")
		p.ExpectSemi()
	}
	spec := &node.ValueSpec{
		Idents: idents,
		Values: values,
		Data:   i,
	}
	return spec
}

func (p *Parser) ParseElse(ifs bool) node.Stmt {
	if p.Trace {
		defer untracep(tracep(p, "Else"))
	}

	p.Expect(token.Else)
	p.SkipSpace()

	switch p.Token.Token {
	case token.If:
		if !ifs {
			p.Error(p.Token.Pos, "only else")
		}
		return p.ParseIfStmt()
	case token.MixedCodeEnd:
		b := p.ParseBlockStmt()
		p.ExpectSemi()
		return b
	case token.LBrace:
		return p.ParseBlockStmt()
	case token.Semicolon:
		p.ExpectSemi()
		return &node.BlockStmt{}
	default:
		b := &node.BlockStmt{}
		b.LBrace.Pos = p.Token.Pos
		if p.Token.Token != token.RBrace {
			b.Stmts = []node.Stmt{
				p.ParseSimpleStmt(false),
			}
		}
		b.RBrace = p.ExpectLit(token.RBrace)
		return b
	}
}

func (p *Parser) ParseForStmt() node.Stmt {
	if p.Trace {
		defer untracep(tracep(p, "ForStmt"))
	}

	pos := p.Expect(token.For)

	// for {}
	if p.Token.Token.IsBlockStart() {
		body := p.ParseBlockStmt()
		p.ExpectSemi()

		return &node.ForStmt{
			ForPos: pos,
			Body:   body,
		}
	}

	prevLevel := p.ExprLevel
	p.ExprLevel = -1

	var s1 node.Stmt
	if p.Token.Token != token.Semicolon { // skipping init
		s1 = p.ParseSimpleStmt(true)
	}

	// for _ in seq {}            or
	// for value in seq {}        or
	// for key, value in seq {}
	if forInStmt, isForIn := s1.(*node.ForInStmt); isForIn {
		forInStmt.ForPos = pos
		p.ExprLevel = prevLevel
		forInStmt.Body = p.ParseBlockStmt(token.Else)
		if p.Token.Token == token.Else {
			forInStmt.Else = p.ParseElse(false).(*node.BlockStmt)
		} else if p.Token.Token != token.EOF {
			p.ExpectSemi()
		}
		return forInStmt
	}

	// for init; cond; post {}
	var s2, s3 node.Stmt
	if p.Token.Token == token.Semicolon {
		p.Next()
		if p.Token.Token != token.Semicolon {
			s2 = p.ParseSimpleStmt(false) // cond
		}
		p.Expect(token.Semicolon)
		if !p.Token.Token.IsBlockStart() {
			s3 = p.ParseSimpleStmt(false) // post
		}
	} else {
		// for cond {}
		s2 = s1
		s1 = nil
	}

	// body
	p.ExprLevel = prevLevel
	body := p.ParseBlockStmt()
	p.ExpectSemi()
	cond := p.MakeExpr(s2, "condition expression")
	return &node.ForStmt{
		ForPos: pos,
		Init:   s1,
		Cond:   cond,
		Post:   s3,
		Body:   body,
	}
}

func (p *Parser) ParseBranchStmt(tok token.Token) node.Stmt {
	if p.Trace {
		defer untracep(tracep(p, "BranchStmt"))
	}

	pos := p.Expect(tok)

	var label *node.Ident
	if p.Token.Token == token.Ident {
		label = p.ParseIdent()
	}
	p.ExpectSemi()
	return &node.BranchStmt{
		Token:    tok,
		TokenPos: pos,
		Label:    label,
	}
}

func (p *Parser) ParseIfStmt() node.Stmt {
	if p.Trace {
		defer untracep(tracep(p, "IfStmt"))
	}

	pos := p.Expect(token.If)
	init, cond := p.ParseIfHeader()

	body := p.ParseBlockStmt(token.Else)

	var elseStmt node.Stmt
	if p.Token.Token == token.Else {
		elseStmt = p.ParseElse(true)
	} else {
		p.ExpectSemi()
	}
	return &node.IfStmt{
		IfPos: pos,
		Init:  init,
		Cond:  cond,
		Body:  body,
		Else:  elseStmt,
	}
}

func (p *Parser) ParseTryStmt() node.Stmt {
	if p.Trace {
		defer untracep(tracep(p, "TryStmt"))
	}
	pos := p.Expect(token.Try)
	body := p.ParseBlockStmt(token.Catch, token.Finally)
	var catchStmt *node.CatchStmt
	var finallyStmt *node.FinallyStmt
	if p.Token.Token == token.Catch {
		catchStmt = p.ParseCatchStmt()
	}
	if p.Token.Token == token.Finally || catchStmt == nil {
		finallyStmt = p.ParseFinallyStmt()
	}
	p.ExpectSemi()
	return &node.TryStmt{
		TryPos:  pos,
		Catch:   catchStmt,
		Finally: finallyStmt,
		Body:    body,
	}
}

func (p *Parser) ParseCatchStmt() *node.CatchStmt {
	if p.Trace {
		defer untracep(tracep(p, "CatchStmt"))
	}
	pos := p.Expect(token.Catch)
	var ident *node.Ident
	if p.Token.Token == token.Ident {
		ident = p.ParseIdent()
	}
	body := p.ParseBlockStmt(token.Finally)
	return &node.CatchStmt{
		CatchPos: pos,
		Ident:    ident,
		Body:     body,
	}
}

func (p *Parser) ParseFinallyStmt() *node.FinallyStmt {
	if p.Trace {
		defer untracep(tracep(p, "FinallyStmt"))
	}
	pos := p.Expect(token.Finally)
	body := p.ParseBlockStmt()
	return &node.FinallyStmt{
		FinallyPos: pos,
		Body:       body,
	}
}

func (p *Parser) ParseThrowStmt() node.Stmt {
	if p.Trace {
		defer untracep(tracep(p, "Throw"))
	}
	pos := p.Expect(token.Throw)
	expr := p.ParseExpr()
	p.ExpectSemi()
	return &node.ThrowStmt{
		ThrowPos: pos,
		Expr:     expr,
	}
}

func (p *Parser) ParseThrowExpr() *node.ThrowExpr {
	if p.Trace {
		defer untracep(tracep(p, "ThrowExpr"))
	}
	pos := p.Expect(token.Throw)
	expr := p.ParseExpr()
	return &node.ThrowExpr{
		ThrowPos: pos,
		Expr:     expr,
	}
}

func (p *Parser) ParseBlockStmt(ends ...token.Token) *node.BlockStmt {
	if p.Trace {
		defer untracep(tracep(p, "BlockStmt"))
	}

	stmt := &node.BlockStmt{}

	if p.Token.Token == token.MixedCodeEnd {
		mce := &node.CodeEndStmt{
			Lit:         ast.Literal{Value: p.Token.Literal, Pos: p.Token.Pos},
			RemoveSpace: RemoveSpaces(p.Token),
		}
		p.Next()
		stmt.Stmts.Append(mce)
		stmt.LBrace.Pos = mce.Pos()
	} else {
		stmt.LBrace = p.ExpectLit(token.LBrace)
	}

	stmt.Stmts.Append(p.ParseStmtList(ends...)...)

	if stmt.LBrace.Value == "{" || !p.IsToken(ends...) {
		stmt.RBrace = p.ExpectLit(token.RBrace)
	}

	return stmt
}

func (p *Parser) ParseIfHeader() (init node.Stmt, cond node.Expr) {
	if p.Token.Token.IsBlockStart() {
		p.Error(p.Token.Pos, "missing condition in if statement")
		cond = &node.BadExpr{From: p.Token.Pos, To: p.Token.Pos}
		return
	}

	outer := p.ExprLevel
	p.ExprLevel = -1
	if p.Token.Token == token.Semicolon {
		p.Error(p.Token.Pos, "missing init in if statement")
		return
	}
	init = p.ParseSimpleStmt(false)

	var condStmt node.Stmt
	switch p.Token.Token {
	case token.LBrace, token.MixedCodeEnd, p.BlockStart:
		condStmt = init
		init = nil
	case token.Semicolon:
		p.Next()
		condStmt = p.ParseSimpleStmt(false)
	default:
		p.Error(p.Token.Pos, "missing condition in if statement")
	}

	if condStmt != nil {
		cond = p.MakeExpr(condStmt, "boolean expression")
	}
	if cond == nil {
		cond = &node.BadExpr{From: p.Token.Pos, To: p.Token.Pos}
	}
	p.ExprLevel = outer
	return
}

func (p *Parser) MakeExpr(s node.Stmt, want string) node.Expr {
	if s == nil {
		return nil
	}

	if es, isExpr := s.(*node.ExprStmt); isExpr {
		return es.Expr
	}

	found := "simple statement"
	if _, isAss := s.(*node.AssignStmt); isAss {
		found = "assignment"
	}
	p.Error(s.Pos(), fmt.Sprintf("expected %s, found %s", want, found))
	return &node.BadExpr{From: s.Pos(), To: p.safePos(s.End())}
}

func (p *Parser) ParseReturn() (ret node.Return) {
	ret.ReturnPos = p.Token.Pos
	p.Expect(token.Return)

	if p.Token.Token != token.Semicolon && p.Token.Token != token.RBrace {
		lbpos := p.Token.Pos
		ret.Result = p.ParseExpr()
		if p.Token.Token != token.Comma {
			goto done
		}
		// if the next token is a comma, treat it as multi return so put
		// expressions into a slice and replace x expression with an ArrayLit.
		elements := make([]node.Expr, 1, 2)
		elements[0] = ret.Result
		for p.Token.Token == token.Comma {
			p.Next()
			ret.Result = p.ParseExpr()
			elements = append(elements, ret.Result)
		}
		ret.Result = &node.ArrayLit{
			Elements: elements,
			LBrack:   lbpos,
			RBrack:   ret.Result.End(),
		}
	}
done:
	return
}

func (p *Parser) ParseReturnStmt() node.Stmt {
	if p.Trace {
		defer untracep(tracep(p, "ReturnStmt"))
	}
	defer p.ExpectSemi()
	return &node.ReturnStmt{Return: p.ParseReturn()}
}

func (p *Parser) ParseReturnExpr() node.Expr {
	if p.Trace {
		defer untracep(tracep(p, "ReturnExpr"))
	}
	return &node.ReturnExpr{Return: p.ParseReturn()}
}

func (p *Parser) ParseSimpleStmt(forIn bool) node.Stmt {
	if p.Trace {
		defer untracep(tracep(p, "SimpleStmt"))
	}

	x := p.ParseExprList()

	switch p.Token.Token {
	case token.Assign, token.Define: // assignment statement
		pos, tok := p.Token.Pos, p.Token.Token
		p.Next()
		y := p.ParseExprList()
		return &node.AssignStmt{
			LHS:      x,
			RHS:      y,
			Token:    tok,
			TokenPos: pos,
		}
	case token.In:
		if forIn {
			p.Next()
			y := p.ParseExpr()

			var key, value *node.Ident
			var ok bool
			switch len(x) {
			case 1:
				key = &node.Ident{Name: "_", NamePos: x[0].Pos()}

				value, ok = x[0].(*node.Ident)
				if !ok {
					p.ErrorExpected(x[0].Pos(), "identifier")
					value = &node.Ident{Name: "_", NamePos: x[0].Pos()}
				}
			case 2:
				key, ok = x[0].(*node.Ident)
				if !ok {
					p.ErrorExpected(x[0].Pos(), "identifier")
					key = &node.Ident{Name: "_", NamePos: x[0].Pos()}
				}
				value, ok = x[1].(*node.Ident)
				if !ok {
					p.ErrorExpected(x[1].Pos(), "identifier")
					value = &node.Ident{Name: "_", NamePos: x[1].Pos()}
				}
				// TODO: no more than 2 idents
			}
			return &node.ForInStmt{
				Key:      key,
				Value:    value,
				Iterable: y,
			}
		}
	}

	if len(x) > 1 {
		p.ErrorExpected(x[0].Pos(), "1 expression")
		// continue with first expression
	}

	switch p.Token.Token {
	case token.Define,
		token.AddAssign, token.SubAssign, token.MulAssign, token.QuoAssign,
		token.RemAssign, token.AndAssign, token.OrAssign, token.XorAssign,
		token.ShlAssign, token.ShrAssign, token.AndNotAssign,
		token.NullichAssign, token.LOrAssign:
		pos, tok := p.Token.Pos, p.Token.Token
		p.Next()
		y := p.ParseExpr()
		return &node.AssignStmt{
			LHS:      []node.Expr{x[0]},
			RHS:      []node.Expr{y},
			Token:    tok,
			TokenPos: pos,
		}
	case token.Inc, token.Dec:
		// increment or decrement statement
		s := &node.IncDecStmt{Expr: x[0], Token: p.Token.Token, TokenPos: p.Token.Pos}
		p.Next()
		return s
	}

	return &node.ExprStmt{Expr: x[0]}
}

func (p *Parser) ParseExprList() (list []node.Expr) {
	if p.Trace {
		defer untracep(tracep(p, "ExpressionList"))
	}

	list = append(list, p.ParseExpr())
	for p.Token.Token == token.Comma {
		p.Next()
		list = append(list, p.ParseExpr())
	}
	return
}

func (p *Parser) ParseMapElementLit() *node.DictElementLit {
	if p.Trace {
		defer untracep(tracep(p, "DictElementLit"))
	}

	pos := p.Token.Pos
	name := "_"
	if p.Token.Token == token.Ident || p.Token.Token.IsKeyword() {
		name = p.Token.Literal
	} else if p.Token.Token == token.String {
		v, _ := strconv.Unquote(p.Token.Literal)
		name = v
	} else {
		p.ErrorExpected(pos, "map key")
	}
	p.Next()
	colonPos := p.Expect(token.Colon)
	valueExpr := p.ParseExpr()
	return &node.DictElementLit{
		Key:      name,
		KeyPos:   pos,
		ColonPos: colonPos,
		Value:    valueExpr,
	}
}

func (p *Parser) ParseDictLit() *node.DictLit {
	if p.Trace {
		defer untracep(tracep(p, "DictLit"))
	}

	lbrace := p.Expect(token.LBrace)
	p.ExprLevel++

	var elements []*node.DictElementLit
	for p.Token.Token != token.RBrace && p.Token.Token != token.EOF {
		elements = append(elements, p.ParseMapElementLit())

		if !p.AtComma("map literal", token.RBrace) {
			break
		}
		p.Next()
	}

	p.ExprLevel--
	rbrace := p.Expect(token.RBrace)
	return &node.DictLit{
		LBrace:   lbrace,
		RBrace:   rbrace,
		Elements: elements,
	}
}

func (p *Parser) ParseKeyValueLit(endToken token.Token) *node.KeyValueLit {
	if p.Trace {
		defer untracep(tracep(p, "KeyValueLit"))
	}

	p.SkipSpace()

	var (
		keyExpr   = p.ParsePrimitiveOperand()
		valueExpr node.Expr
	)

	p.SkipSpace()

	switch p.Token.Token {
	case token.Comma, endToken:
	default:
		p.Expect(token.Assign)
		valueExpr = p.ParseExpr()
		p.SkipSpace()
	}
	return &node.KeyValueLit{
		Key:   keyExpr,
		Value: valueExpr,
	}
}

func (p *Parser) ParseKeyValueArrayLitAt(lbrace source.Pos, rbraceToken token.Token) *node.KeyValueArrayLit {
	p.ExprLevel++
	var elements []*node.KeyValueLit

	for p.Token.Token != rbraceToken && p.Token.Token != token.EOF {
		elements = append(elements, p.ParseKeyValueLit(rbraceToken))

		if !p.AtComma("keyValueArray literal", rbraceToken) {
			break
		}
		p.Next()
	}

	p.ExprLevel--

	if p.Token.Token != rbraceToken {
		p.Expect(rbraceToken)
	}
	rbrace := p.Token.Pos
	return &node.KeyValueArrayLit{
		LBrace:   lbrace,
		RBrace:   rbrace,
		Elements: elements,
	}
}

func (p *Parser) ParseKeyValueArrayLit(lbrace source.Pos) *node.KeyValueArrayLit {
	if p.Trace {
		defer untracep(tracep(p, "ParseKeyValueArrayLit"))
	}

	p.Expect(token.Semicolon)

	kva := p.ParseKeyValueArrayLitAt(lbrace, token.RParen)
	p.Expect(token.RParen)
	return kva
}

func (p *Parser) Expect(token token.Token) source.Pos {
	return p.ExpectToken(token).Pos
}

func (p *Parser) ExpectLit(token token.Token) ast.Literal {
	lit := ast.Literal{
		Pos:   p.Token.Pos,
		Value: p.Token.Literal,
	}

	if p.Token.Token != token {
		p.ErrorExpected(lit.Pos, "'"+token.String()+"'")
	}
	p.Next()
	return lit
}

func (p *Parser) IsToken(toks ...token.Token) bool {
	for _, tok := range toks {
		if p.Token.Token == tok {
			return true
		}
	}
	return false
}

func (p *Parser) ExpectLits(toks ...token.Token) ast.Literal {
	lit := ast.Literal{
		Pos:   p.Token.Pos,
		Value: p.Token.Literal,
	}

	for _, tok := range toks {
		if p.Token.Token == tok {
			p.Next()
			return lit
		}
	}

	s := make([]string, len(toks))
	for i, tok := range toks {
		s[i] = tok.String()
	}

	p.ErrorExpected(lit.Pos, strings.Join(s, " | "))
	return lit
}

func (p *Parser) ExpectToken(token token.Token) (tok Token) {
	tok = p.Token
	if tok.Token != token {
		p.ErrorExpected(tok.Pos, "'"+token.String()+"'")
	}
	p.Next()
	return
}

func (p *Parser) ExpectSemi() {
	switch p.Token.Token {
	case token.RParen, token.RBrace, token.Else, token.MixedCodeEnd:
		// semicolon is optional before a closing ')' or '}'
	case token.Comma:
		// permit a ',' instead of a ';' but complain
		p.ErrorExpected(p.Token.Pos, "';'")
		fallthrough
	case token.Semicolon:
		p.Next()
	default:
		switch p.PrevToken.Token {
		case token.Else, p.BlockEnd, token.DotName, token.DotFile, token.IsModule:
			return
		}
		p.ErrorExpected(p.Token.Pos, "';'")
		p.advance(stmtStart)
	}
}

func (p *Parser) advance(to map[token.Token]bool) {
	for ; p.Token.Token != token.EOF; p.Next() {
		if to[p.Token.Token] {
			if p.Token.Pos == p.syncPos && p.syncCount < 10 {
				p.syncCount++
				return
			}
			if p.Token.Pos > p.syncPos {
				p.syncPos = p.Token.Pos
				p.syncCount = 0
				return
			}
		}
	}
}

func (p *Parser) Error(pos source.Pos, msg string) {
	filePos := p.File.Position(pos)

	n := len(p.Errors)
	if n > 0 && p.Errors[n-1].Pos.Line == filePos.Line {
		// discard errors reported on the same line
		return
	}
	if n > 10 {
		// too many errors; terminate early
		panic(bailout{})
	}
	p.Errors.Add(filePos, msg)
}

func (p *Parser) ErrorExpected(pos source.Pos, msg string) {
	msg = "expected " + msg
	if pos == p.Token.Pos {
		// error happened at the current position: provide more specific
		switch {
		case p.Token.Token == token.Semicolon && p.Token.Literal == "\n":
			msg += ", found newline"
		case p.Token.Token.IsLiteral():
			msg += ", found " + p.Token.Literal
		default:
			msg += ", found '" + p.Token.Token.String() + "'"
		}
	}
	p.Error(pos, msg)
}

func (p *Parser) ErrorExpectedExpr(expected, got node.Expr) {
	p.Error(got.Pos(), fmt.Sprintf("expected %T, but got %s (%[2]T)", expected, got))
}

func (p *Parser) consumeComment() (comment *ast.Comment, endline int) {
	// /*-style comments may end on a different line than where they start.
	// Scan the comment for '\n' chars and adjust endline accordingly.
	endline = p.File.Line(p.Token.Pos)
	if p.Token.Literal[1] == '*' {
		// don't use range here - no need to decode Unicode code points
		for i := 0; i < len(p.Token.Literal); i++ {
			if p.Token.Literal[i] == '\n' {
				endline++
			}
		}
	}

	comment = &ast.Comment{Slash: p.Token.Pos, Text: p.Token.Literal}
	p.next0()
	return
}

func (p *Parser) consumeCommentGroup(n int) (comments *ast.CommentGroup) {
	var list []*ast.Comment
	endline := p.File.Line(p.Token.Pos)
	for p.Token.Token == token.Comment && p.File.Line(p.Token.Pos) <= endline+n {
		var comment *ast.Comment
		comment, endline = p.consumeComment()
		list = append(list, comment)
	}

	comments = &ast.CommentGroup{List: list}
	p.comments = append(p.comments, comments)
	return
}

func (p *Parser) next0() {
	if p.Trace && p.Token.Pos.IsValid() {
		s := p.Token.Token.String()
		switch {
		case p.Token.Token.IsLiteral():
			p.PrintTrace(s, p.Token.Literal)
		case p.Token.Token.IsOperator(), p.Token.Token.IsKeyword():
			p.PrintTrace(`"` + s + `"`)
		default:
			p.PrintTrace(s)
		}
	}
	if p.ScanFunc != nil {
		p.Token = p.ScanFunc()
	} else {
		p.Token = p.Scanner.Scan()
	}
}

func (p *Parser) Next() {
	prev := p.Token.Pos
	p.PrevToken = p.Token

next:
	p.next0()
	switch p.Token.Token {
	case token.MixedText:
		if p.Token.Literal == "" {
			goto next
		}
	case token.Comment:
		if p.File.Line(p.Token.Pos) == p.File.Line(prev) {
			// line comment of prev token
			_ = p.consumeCommentGroup(0)
		}
		// consume successor comments, if any
		for p.Token.Token == token.Comment {
			// lead comment of next token
			_ = p.consumeCommentGroup(1)
		}
	}
}

func (p *Parser) SkipSpace() {
	for p.Token.Token == token.Semicolon && p.Token.Literal == "\n" {
		p.Next()
	}
}

func (p *Parser) PrintTrace(a ...any) {
	const (
		dots = ". . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . "
		n    = len(dots)
	)

	filePos := p.File.Position(p.Token.Pos)
	_, _ = fmt.Fprintf(p.TraceOut, "%5d: %5d:%3d: ", p.Token.Pos, filePos.Line,
		filePos.Column)
	i := 2 * p.indent
	for i > n {
		_, _ = fmt.Fprint(p.TraceOut, dots)
		i -= n
	}
	_, _ = fmt.Fprint(p.TraceOut, dots[0:i])
	_, _ = fmt.Fprintln(p.TraceOut, a...)
}

func (p *Parser) safePos(pos source.Pos) source.Pos {
	fileBase := p.File.Base
	fileSize := p.File.Size

	if int(pos) < fileBase || int(pos) > fileBase+fileSize {
		return source.Pos(fileBase + fileSize)
	}
	return pos
}

func tracep(p *Parser, msg string) *Parser {
	p.PrintTrace(msg, "(")
	p.indent++
	return p
}

func untracep(p *Parser) {
	p.indent--
	p.PrintTrace(")")
}

type BlockEnd struct {
	Token token.Token
	Next  bool
}

type BlockWrap struct {
	Start token.Token
	Ends  []BlockEnd
}
