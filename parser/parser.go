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
	"strconv"
	"strings"

	"github.com/gad-lang/gad/internal"
	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/token"
	"github.com/shopspring/decimal"
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

// Parser parses the Tengo source files. It's based on ToInterface's parser
// implementation.
type Parser struct {
	File             *source.File
	Errors           ErrorList
	Scanner          ScannerInterface
	Token            PToken
	tokenBuffer      []PToken
	PrevToken        PToken
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
	ScanFunc         func() PToken
	pipes            int
	postScan         func(p *Parser)

	inHeader bool
}

// NewParser creates a Parser.
func NewParser(file *source.File, trace io.Writer) *Parser {
	return NewParserWithOptions(file, &ParserOptions{Trace: trace}, nil)
}

type ParserOptions struct {
	Trace io.Writer
	Mode  Mode
}

// NewParserWithOptions creates a Parser with parser mode flags.
func NewParserWithOptions(
	file *source.File,
	opts *ParserOptions,
	scannerOptions *ScannerOptions,
) *Parser {
	if opts == nil {
		opts = &ParserOptions{}
	}

	if scannerOptions == nil {
		scannerOptions = &ScannerOptions{}
	}
	if scannerOptions.Mode == 0 {
		if opts.Mode.Has(ParseComments) {
			scannerOptions.Mode.Set(ScanComments)
		}
		if opts.Mode.Has(ParseFloatAsDecimal) {
			scannerOptions.Mode.Set(ScanFloatAsDecimal)
		}
		if opts.Mode.Has(ParseMixed) {
			scannerOptions.Mode.Set(ScanMixed)
		}
		if opts.Mode.Has(ParseConfigDisabled) {
			scannerOptions.Mode.Set(ScanConfigDisabled)
		}
		if opts.Mode.Has(ParseMixedExprAsValue) {
			scannerOptions.Mode.Set(ScanMixedExprAsValue)
		}
	}
	return NewParserWithScanner(NewScanner(file, scannerOptions), opts)
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
		postScan:   func(p *Parser) {},
	}
	p.ParseStmtHandler = p.DefaultParseStmt
	var m ScanMode
	if opts.Mode.Has(ParseComments) {
		m.Set(ScanComments)
	}
	if opts.Mode.Has(ParseFloatAsDecimal) {
		m.Set(ScanFloatAsDecimal)
	}
	if opts.Mode.Has(ParseMixed) {
		m.Set(ScanMixed)
	}
	if opts.Mode.Has(ParseMixedExprAsValue) {
		m.Set(ScanMixedExprAsValue)
	}
	if opts.Mode.Has(ParseConfigDisabled) {
		m.Set(ScanConfigDisabled)
	}
	if opts.Mode.Has(ParseCharAsString) {
		m.Set(ScanCharAsString)
	}
	scanner.ErrorHandler(func(pos source.FilePos, msg string) {
		p.Errors.Add(pos, msg)
	})
	return p.Next()
}

func ParseFile(pth string, opts *ParserOptions, scannerOpts *ScannerOptions) (file *File, err error) {
	var (
		fileSet = source.NewFileSet()
		script  []byte
		srcFile *source.File
		f       *os.File
	)

	if f, err = os.Open(pth); err != nil {
		return
	}

	defer f.Close()

	if script, err = io.ReadAll(f); err != nil {
		return
	}

	srcFile = fileSet.AddFileData(pth, -1, script)

	p := NewParserWithOptions(srcFile, opts, scannerOpts)
	return p.ParseFile()
}

func (p *Parser) Failed() bool {
	return len(p.Errors) > 0
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

func (p *Parser) WrapPostScan(f func(in func(p *Parser)) func(p *Parser)) {
	p.postScan = f(p.postScan)
}

func (p *Parser) ParseExpr() node.Expr {
	if p.Trace {
		defer untracep(tracep(p, "Expression"))
	}

	expr := p.ParseBinaryExpr(token.LowestPrec + 1)

	// ternary conditional expression
	if p.Token.Is(token.Question) {
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
		op, prec := p.Token.Token, p.Token.Precedence()
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
	case token.And:
		pos := p.Expect(token.And)
		expr := p.ParsePrimaryExpr()
		return &node.Ptr{TokenPos: pos, Expr: expr}
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
		if ti, _ := t.Expr.(*node.TypedIdentExpr); ti != nil {
			if len(ti.Type) > 0 {
				p.Error(ti.Type[0].Pos(), "unexpected COLON")
				return x
			}
			t.Expr = ti.Ident
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
				tk := p.Token.Token
				if tk.IsKeyword() {
					x = p.ParseSelector(x)
				} else {
					pos := p.Token.Pos
					p.ErrorExpected(pos, "selector")
					p.advance(stmtStart)
					return &node.BadExpr{From: pos, To: p.Token.Pos}
				}
			}
		case token.LBrack:
			x = p.ParseIndexOrSlice(x)
		case token.LParen:
			paren := p.ParseParemExpr(token.LParen, token.RParen)
			ih := p.inHeader

			if !ih && p.Token.Token.Is(token.LBrace, token.Lambda) {
				if ident, _ := x.(*node.IdentExpr); ident != nil {
					if params, err := paren.ToMultiParenExpr().ToFuncParams(); err == nil {
						f := &node.FuncExpr{
							Type: &node.FuncType{
								NameExpr: ident,
								Params:   params,
							},
						}
						x = f

						if p.Token.Token.Is(token.LBrace) {
							f.Body = p.ParseBlockStmt()
						} else {
							f.LambdaPos = p.Expect(token.Lambda)
							f.BodyExpr = p.ParseExpr()
						}
					} else {
						p.Error(paren.Pos(), "expected function parameters")
						return &node.BadExpr{From: paren.Pos(), To: p.Token.Pos}
					}
				} else {
					p.Error(x.Pos(), fmt.Sprintf("expected *Ident, but got %T", x))
					return &node.BadExpr{From: paren.Pos(), To: p.Token.Pos}
				}
			} else {
				p.inHeader = false

				args, err := paren.ToMultiParenExpr().ToCallArgs(true)
				if err != nil {
					p.Error(err.Pos(), err.Error())
				}

				x = &node.CallExpr{
					Func:     x,
					CallArgs: *args,
				}

				if p.Token.Token.Is(token.Period) && p.pipes == 1 {
					return x
				}

				p.inHeader = ih
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
		case *node.KeyValuePairLit, *node.NamedArgVarLit:
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
			case *node.KeyValuePairLit:
				switch t2 := t.Key.(type) {
				case *node.IdentExpr:
					params.NamedArgs.Names = append(params.NamedArgs.Names, &node.NamedArgExpr{Ident: t2})
				case *node.StringLit:
					params.NamedArgs.Names = append(params.NamedArgs.Names, &node.NamedArgExpr{Lit: t2})
				default:
					p.ErrorExpected(t2.Pos(), "expected Ident | StringLit")
					return
				}
				params.NamedArgs.Values = append(params.NamedArgs.Values, t.Value)
			case *node.NamedArgVarLit:
				e := &node.NamedArgExpr{Var: true}
				if ident, _ := t.Value.(*node.IdentExpr); ident != nil {
					e.Ident = ident
				} else {
					e.Exp = t.Value
				}
				params.NamedArgs.Names = append(params.NamedArgs.Names, e)
				params.NamedArgs.Values = append(params.NamedArgs.Values, nil)
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
	args, err := p.ParseParemExpr(start, end).ToMultiParenExpr().ToCallArgs(true)
	if err != nil {
		p.Error(err.Pos(), err.Error())
	}
	return args
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

func (p *Parser) AtCommaOrNewLine(context string, follow token.Token) bool {
	if p.Token.Token == token.Comma {
		return true
	}

	if p.Token.IsSpace() {
		return true
	}

	if p.Token.Token != follow {
		msg := "missing ',' or new line"
		p.Error(p.Token.Pos, msg+" in "+context)
		return true // "insert" comma and continue
	}
	return false
}

func (p *Parser) ParseIndexExpr(x node.Expr) *node.IndexExpr {
	if p.Trace {
		defer untracep(tracep(p, "Index"))
	}

	lbrack := p.Expect(token.LBrack)
	p.ExprLevel++

	index := p.ParseExpr()
	p.ExprLevel--
	rbrack := p.Expect(token.RBrack)
	return &node.IndexExpr{
		X:      x,
		LBrack: lbrack,
		RBrack: rbrack,
		Index:  index,
	}
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
		X:      x,
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
		sel = node.EParen(sel, lparen, rparen)
	case token.Else:
		name := p.Token.Token.String()
		sel = node.String(name, p.Token.Pos)
		p.Next()
	default:
		if tk := p.Token.Token; tk.IsKeyword() {
			sel = node.String(tk.String(), p.Token.Pos)
			p.Next()
		} else {
			ident := p.ParseIdent()
			sel = node.String(ident.Name, ident.NamePos)
		}
	}
	expr = x
	return
}

func (p *Parser) ParseSimpleSelector(x node.Expr) (sel *node.SelectorExpr) {
	if p.Trace {
		defer untracep(tracep(p, "SimpleSelector"))
	}

	var s node.Expr
	x, s = p.ParseSimpleSelectorNode(x)
	return &node.SelectorExpr{X: x, Sel: s}
}

func (p *Parser) ParseSimpleSelectorNode(x node.Expr) (expr, sel node.Expr) {
	switch p.Token.Token {
	case token.LParen:
		lparen := p.Token.Pos
		p.Next()
		sel = p.ParseExpr()
		rparen := p.Expect(token.RParen)
		sel = node.EParen(sel, lparen, rparen)
	default:
		ident := p.ParseIdent()
		sel = node.String(ident.Name, ident.NamePos)
	}
	expr = x
	return
}

func (p *Parser) ParseSelector(x node.Expr) (sel *node.SelectorExpr) {
	if p.Trace {
		defer untracep(tracep(p, "Selector"))
	}

	var s node.Expr
	x, s = p.ParseSelectorNode(x)
	return &node.SelectorExpr{X: x, Sel: s}
}

func (p *Parser) ParseNullishSelector(x node.Expr) (sel node.Expr) {
	if p.Trace {
		defer untracep(tracep(p, "NullishSelector"))
	}

	x, sel = p.ParseSelectorNode(x)
	return &node.NullishSelectorExpr{Expr: x, Sel: sel}
}

func (p *Parser) ParseStringLit() *node.StringLit {
	x := &node.StringLit{
		ValuePos: p.Token.Pos,
		Literal:  p.Token.Literal,
	}
	p.Next()
	return x
}

func (p *Parser) ParseSymbolLit() *node.SymbolLit {
	x := &node.SymbolLit{
		Lit: p.Token.TokenLit,
	}
	p.Next()
	return x
}

func (p *Parser) ParseIntLit() *node.IntLit {
	v, _ := strconv.ParseInt(p.Token.Literal, 0, 64)
	x := &node.IntLit{
		Value:    v,
		ValuePos: p.Token.Pos,
		Literal:  p.Token.Literal,
	}
	p.Next()
	return x
}

func (p *Parser) ParseUintLit() *node.UintLit {
	v, _ := strconv.ParseUint(strings.TrimSuffix(p.Token.Literal, "u"), 0, 64)
	x := &node.UintLit{
		Value:    v,
		ValuePos: p.Token.Pos,
		Literal:  p.Token.Literal,
	}
	p.Next()
	return x
}

func (p *Parser) ParseFloatLit() *node.FloatLit {
	v, _ := strconv.ParseFloat(p.Token.Literal, 64)
	x := &node.FloatLit{
		Value:    v,
		ValuePos: p.Token.Pos,
		Literal:  p.Token.Literal,
	}
	p.Next()
	return x
}

func (p *Parser) ParseDecimalLit() *node.DecimalLit {
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
}
func (p *Parser) ParseBoolLit() *node.BoolLit {
	x := &node.BoolLit{
		Value:    p.Token.Token == token.True,
		ValuePos: p.Token.Pos,
		Literal:  p.Token.Literal,
	}
	p.Next()
	return x
}

func (p *Parser) ParseFlagLit() *node.FlagLit {
	x := &node.FlagLit{
		Value:    p.Token.Token == token.Yes,
		ValuePos: p.Token.Pos,
		Literal:  p.Token.Literal,
	}
	p.Next()
	return x
}

func (p *Parser) ParseLiteral() node.Expr {
	if p.Trace {
		defer untracep(tracep(p, "Operand"))
	}
	switch p.Token.Token {
	case token.Nil:
		x := &node.NilLit{TokenPos: p.Token.Pos}
		p.Next()
		return x
	case token.Ident:
		return p.ParseIdent()
	case token.Int:
		return p.ParseIntLit()
	case token.Uint:
		return p.ParseUintLit()
	case token.Float:
		return p.ParseFloatLit()
	case token.Decimal:
		return p.ParseDecimalLit()
	case token.Char:
		return p.ParseCharLit()
	case token.String:
		return p.ParseStringLit()
	case token.RawString:
		return p.ParseRawStringLit()
	case token.Symbol:
		return p.ParseSymbolLit()
	case token.True, token.False:
		return p.ParseBoolLit()
	case token.Yes, token.No:
		return p.ParseFlagLit()
	case token.RawHeredoc:
		return p.ParseRawHeredocLit()
	default:
		pos := p.Token.Pos
		p.ErrorExpected(pos, "literal value")
		p.advance(stmtStart)
		return &node.BadExpr{From: pos, To: p.Token.Pos}
	}
}

func (p *Parser) ParsePrimitiveOperand() node.Expr {
	if isPrimiteValue(p.Token.Token) {
		return p.ParseLiteral()
	} else {
		switch p.Token.Token {
		case token.Nil:
			x := &node.NilLit{TokenPos: p.Token.Pos}
			p.Next()
			return x
		case token.Callee:
			x := &node.CalleeKeywordExpr{TokenPos: p.Token.Pos, Literal: p.Token.Literal}
			p.Next()
			return x
		case token.Args:
			x := &node.ArgsKeywordExpr{TokenPos: p.Token.Pos, Literal: p.Token.Literal}
			p.Next()
			return x
		case token.NamedArgs:
			x := &node.NamedArgsKeywordExpr{TokenPos: p.Token.Pos, Literal: p.Token.Literal}
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
		case token.IsMain:
			x := &node.IsMainLit{TokenPos: p.Token.Pos}
			p.Next()
			return x
		case token.Module:
			x := &node.ModuleLit{TokenPos: p.Token.Pos}
			p.Next()
			return x
		}
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

	if isPrimiteValue(p.Token.Token) {
		return p.ParseLiteral()
	} else {
		switch p.Token.Token {
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
		case token.IsMain:
			x := &node.IsMainLit{TokenPos: p.Token.Pos}
			p.Next()
			return x
		case token.Module:
			x := &node.ModuleLit{TokenPos: p.Token.Pos}
			p.Next()
			return x
		case token.Callee:
			x := &node.CalleeKeywordExpr{TokenPos: p.Token.Pos, Literal: p.Token.Literal}
			p.Next()
			return x
		case token.Args:
			x := &node.ArgsKeywordExpr{TokenPos: p.Token.Pos, Literal: p.Token.Literal}
			p.Next()
			return x
		case token.NamedArgs:
			x := &node.NamedArgsKeywordExpr{TokenPos: p.Token.Pos, Literal: p.Token.Literal}
			p.Next()
			return x
		case token.Import:
			return p.ParseImportExpr()
		case token.Embed:
			return p.ParseEmbedExpr()
		case token.LParen:
			if p.Peek().Token == token.Assign {
				return p.ParseComputedExpr()
			}
			return p.ParseParenOrClosure(token.LParen, token.RParen)
		case token.LBrack: // array literal
			return p.ParseArrayLitOrKeyValue()
		case token.LBrace: // dict literal
			return p.ParseDictLit()
		case token.Func, token.Method: // function literal
			return p.ParseFuncExpr()
		case token.Throw:
			return p.ParseThrowExpr()
		case token.Return:
			return p.ParseReturnExpr()
		case token.Template:
			pos := p.Token.Pos
			p.Next()
			switch p.Token.Token {
			case token.String, token.RawString, token.RawHeredoc, token.Symbol:
				return &node.TemplateLit{
					TokenPos: pos,
					Value:    p.ParseOperand(),
				}
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
	ident := node.EIdent(p.Token.Token.String(), pos)
	p.Next()

	var (
		c     = p.ParseCall(ident)
		valid = len(c.Args.Values) >= 1
	)

	if valid {
		_, valid = c.Args.Values[0].(*node.StringLit)
	}

	if !valid {
		p.ErrorExpected(p.Token.Pos, "module name")
		p.advance(stmtStart)
		return &node.BadExpr{From: pos, To: p.Token.Pos}
	}

	return &node.ImportExpr{CallExpr: *c}
}

func (p *Parser) ParseEmbedExpr() node.Expr {
	pos := p.Token.Pos

	p.Next()
	p.Expect(token.LParen)

	var pth string
	switch p.Token.Token {
	case token.String:
		pth, _ = strconv.Unquote(p.Token.Literal)
	case token.Symbol:
		pth = p.ParseSymbolLit().Value()
	default:
		p.ErrorExpected(p.Token.Pos, "path")
		p.advance(stmtStart)
		return &node.BadExpr{From: pos, To: p.Token.Pos}
	}

	expr := &node.EmbedExpr{
		Path:     pth,
		Token:    token.Embed,
		TokenPos: pos,
	}

	p.Next()
	p.Expect(token.RParen)
	return expr
}

func (p *Parser) ParseClosureExpr(lambdaToken token.Token, paren *node.MultiParenExpr) *node.ClosureExpr {
	if p.Trace {
		defer untracep(tracep(p, "ClosureExpr"))
	}

	lambda := p.Expect(lambdaToken)

	var body node.Expr
	if p.Token.Token.IsBlockStart() {
		body = &node.BlockExpr{BlockStmt: p.ParseBlockStmt()}
	} else {
		body = p.ParseExpr()
	}

	var params, err = paren.ToFuncParams()
	if err != nil {
		p.Error(err.Pos(), err.Error())
	}

	return &node.ClosureExpr{
		Lambda: node.Token{
			Pos:   lambda,
			Token: lambdaToken,
		},
		Params: params,
		Body:   body,
	}
}

func (p *Parser) ParseParenOrClosure(lparenToken, rparenToken token.Token) node.Expr {
	paren := p.ParseParemExpr(lparenToken, rparenToken)
	if p.Failed() {
		return paren
	}
	switch p.Token.Token {
	case token.Lambda:
		return p.ParseClosureExpr(p.Token.Token, paren.ToMultiParenExpr())
	}
	return paren
}

func (p *Parser) ParseSingleParemExpr(lparen, rparen token.Token) node.Expr {
	if p.Trace {
		defer untracep(tracep(p, "SingleParemExpr"))
	}

	n := p.ParseParemExpr(lparen, rparen)

	if paren, _ := n.(*node.ParenExpr); paren == nil && p.Errors.Len() == 0 {
		p.ErrorExpectedExpr(&node.ParenExpr{}, n)
		return n
	} else {
		return paren
	}
}

func (p *Parser) ParseParemExpr(lparenToken, rparenToken token.Token) node.ToMultiParenConverter {
	if p.Trace {
		defer untracep(tracep(p, "ParemExpr"))
	}

	var (
		lparen = p.Token.Pos
		end    = rparenToken
	)

	switch p.Token.Token {
	case lparenToken:
	default:
		p.ErrorExpected(lparen, "'"+lparenToken.String()+"'")
		return &node.ParenExpr{}
	}

	p.Next()

	var (
		exprs, nexprs node.Exprs
		expr          node.Expr
		multi         bool
		rparen        source.Pos
	)

	switch p.Token.Token {
	case token.Semicolon:
		if p.Token.IsSemi() {
			return p.ParseKeyValueArrayLit(lparen)
		}
	case token.Comma:
		multi = true
		p.Next()
		if p.Token.IsSemi() {
			kv := p.ParseKeyValueArrayLit(lparen)
			rparen = kv.RParen
			nexprs = kv.Elements
			goto done
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
		p.ExprLevel--
		p.ExpectToken(token.RParen)
		return &node.ParenExpr{
			LParen: node.Token{Token: lparenToken, Pos: lparen},
			RParen: node.Token{Token: rparenToken, Pos: rparen},
			Expr:   mte,
		}
	}

	p.SkipSpace()

	for p.Token.Token != end {
		var (
			pos = p.Token.Pos
			mul bool
		)

		switch p.Token.Token {
		case token.Mul:
			mul = true
			p.Next()
			p.SkipSpace()
		}

		p.ExprLevel++
		expr = p.ParseExpr()
		p.ExprLevel--
		p.SkipSpace()

		if ident, _ := expr.(*node.IdentExpr); ident != nil {
			if p.Token.Token == token.Ident {
				expr = &node.TypedIdentExpr{
					Ident: ident,
					Type:  p.ParseTypes(),
				}
			}
		}

		if mul {
			expr = &node.ArgVarLit{
				TokenPos: pos,
				Value:    expr,
			}
		}

		exprs = append(exprs, expr)

		if p.Token.Token == token.Comma {
			p.Next()
		} else if p.Token.Token == token.Semicolon {
			if p.Token.IsSemi() {
				kv := p.ParseKeyValueArrayLit(0)
				rparen = kv.RParen
				nexprs = kv.Elements
				goto done
			}
			p.Next()
		} else {
			break
		}
	}

	rparen = p.Expect(end)

done:

	if !multi && len(exprs) == 1 && len(nexprs) == 0 &&
		!internal.TSType(exprs[0],
			&node.KeyValuePairLit{},
			&node.ArgVarLit{},
			&node.NamedArgVarLit{}) {

		for {
			if paren, _ := exprs[0].(*node.ParenExpr); paren != nil {
				exprs[0] = paren.Expr
			} else {
				break
			}
		}

		return &node.ParenExpr{
			LParen: node.Token{Token: lparenToken, Pos: lparen},
			RParen: node.Token{Token: rparenToken, Pos: rparen},
			Expr:   exprs[0],
		}
	}
	return &node.MultiParenExpr{
		LParen:             node.Token{Token: lparenToken, Pos: lparen},
		RParen:             node.Token{Token: rparenToken, Pos: rparen},
		PositionalElements: exprs,
		NamedElements:      nexprs,
	}
}

func (p *Parser) ParseComputedExpr() *node.ComputedExpr {
	lparen := p.Expect(token.LParen)
	p.Next()
	stmts := p.ParseStmtList(token.RParen)
	if len(stmts) == 1 {
		if b, _ := stmts[0].(*node.BlockStmt); b != nil {
			stmts = b.Stmts
		}
	}
	return &node.ComputedExpr{
		StartPos: lparen,
		Stmts:    stmts,
		EndPos:   p.Expect(token.RParen),
	}
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

		if p.AtCommaOrNewLine("array literal", token.RBrack) {
			p.Next()
		}
	}

	for p.Token.Token != token.RBrack && p.Token.Token != token.EOF {
		elements = append(elements, p.ParseExpr())

		if !p.AtCommaOrNewLine("array literal", token.RBrack) {
			break
		}
		p.Next()
	}

	p.ExprLevel--
	rbrack := p.Expect(token.RBrack)
	return &node.ArrayExpr{
		Elements: elements,
		LBrack:   lbrack,
		RBrack:   rbrack,
	}
}

func (p *Parser) ParseFuncStmt() (stmt node.Stmt) {
	if p.Trace {
		defer untracep(tracep(p, "FuncStmt"))
	}

	e := p.ParseFuncExprT(p.ExpectToken(p.Token.Token))

	if p.Token.Token.Is(token.LParen) {
		parem := p.ParseParemExpr(token.LParen, token.RParen)
		params, err := parem.ToMultiParenExpr().ToCallArgs(true)
		if err != nil {
			p.Error(parem.Pos(), err.Error())
			e = &node.BadExpr{From: parem.Pos(), To: parem.End()}
		} else {
			e = &node.CallExpr{
				Func:     e,
				CallArgs: *params,
			}
		}
	}

	switch t := e.(type) {
	case *node.FuncExpr:
		return &node.FuncStmt{Func: t}
	case *node.FuncWithMethodsExpr:
		return &node.FuncWithMethodsStmt{FuncWithMethodsExpr: *t}
	}
	return &node.ExprStmt{Expr: e}
}

func (p *Parser) ParseFuncExpr() (e node.Expr) {
	if p.Trace {
		defer untracep(tracep(p, "FuncExpr"))
	}

	switch p.Token.Token {
	case token.Func, token.Method:
		return p.ParseFuncExprT(p.ExpectToken(p.Token.Token))
	default:
		p.ErrorExpectToken(p.Token, token.Func, token.Method)
		return &node.BadExpr{From: p.Token.Pos, To: p.Token.Pos}
	}
}

func (p *Parser) ParseFuncExprT(tok PToken) (e node.Expr) {
	if p.Trace {
		defer untracep(tracep(p, "FuncExprT"))
	}

	defer func() {
		if tok.Token.Is(token.Method) {
			switch t := e.(type) {
			case *node.FuncWithMethodsExpr, *node.FuncExpr:
				e = &node.MethodExpr{Expr: t}
			}
		}
	}()

	var (
		f = &node.FuncType{
			FuncPos: tok.Pos,
		}
	)

	defer func() {
		if e == nil {
			e = &node.BadExpr{
				From: f.FuncPos,
				To:   p.Token.Pos,
			}
		}
	}()

	if tok.Is(token.Method) || p.Token.Token == token.Ident {
		f.Token = tok.TokenLit
		f.NameExpr = p.ParseIdent()

		switch p.Token.Token {
		case token.Period, token.LBrack:
			f.NameExpr = p.ParseSimpleSelectorExpr(f.NameExpr)
		case token.LParen, token.LBrace:
		default:
			pos := p.Token.Pos
			p.ExpectLits(token.Period, token.LBrack, token.LParen, token.LBrace)
			return &node.BadExpr{From: pos, To: p.Token.Pos}
		}
	}

	switch p.Token.Token {
	case token.LParen:
		if paren := p.ParseParemExpr(token.LParen, token.RParen); paren != nil && p.Errors.Len() == 0 {
			var err *node.NodeError

			if f.Params, err = paren.ToMultiParenExpr().ToFuncParams(); err != nil {
				p.Error(err.Pos(), err.Error())
			} else {
				p.ExprLevel++
				body, lambdaPos, closure := p.ParseBody()
				p.ExprLevel--

				if p.Failed() {
					return
				}

				return &node.FuncExpr{
					Type:      f,
					Body:      body,
					LambdaPos: lambdaPos,
					BodyExpr:  closure,
				}
			}
		}
	case token.LBrace:
		// have methods
		wm := &node.FuncWithMethodsExpr{
			FuncToken: tok.TokenLit,
			NameExpr:  f.NameExpr,
			LBrace:    p.Expect(token.LBrace),
		}

		p.ExprLevel++
		p.SkipSpace()

		for p.Token.Token != token.RBrace {
			p.SkipSpace()

			f := &node.FuncMethod{}

			if paren := p.ParseParemExpr(token.LParen, token.RParen); paren != nil && p.Errors.Len() == 0 {
				var err *node.NodeError

				if f.Params, err = paren.ToMultiParenExpr().ToFuncParams(); err != nil {
					p.Error(err.Pos(), err.Error())
					return
				} else {
					p.ExprLevel++
					f.Body, f.LambdaPos, f.BodyExpr = p.ParseBody()
					p.ExprLevel--

					if p.Failed() {
						return
					}

					wm.Methods = append(wm.Methods, f)
					p.ExpectSemi()
				}
			} else {
				return
			}
		}

		p.ExprLevel--
		p.Expect(token.RBrace)
		e = wm
	default:
		p.ErrorExpectToken(p.Token, token.LParen, token.LBrace)
	}

	return
}

func (p *Parser) ParseBody() (b *node.BlockStmt, lambdaPos source.Pos, closure node.Expr) {
	if p.Trace {
		defer untracep(tracep(p, "Body"))
	}

	p.SkipSpace()

	if p.Token.Token == token.Lambda {
		lambdaPos = p.Token.Pos
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

loop:
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
				switch s.(type) {
				case *node.EmptyStmt:
					continue loop
				case *node.CodeBeginStmt:
					if len(list) > 0 {
						prev := list[len(list)-1]
						if _, ok := prev.(*node.CodeEndStmt); ok {
							list = list[:len(list)-1]
							continue loop
						}
					}
				}
				list = append(list, s)
			}
		}
	}
}

func (p *Parser) ParseIdent() *node.IdentExpr {
	pos := p.Token.Pos
	name := "_"

	if p.Token.Token == token.Ident {
		name = p.Token.Literal
		p.Next()
	} else {
		p.Expect(token.Ident)
	}
	return &node.IdentExpr{
		NamePos: pos,
		Name:    name,
	}
}

func (p *Parser) ParseTypedIdent() *node.TypedIdentExpr {
	return &node.TypedIdentExpr{
		Ident: p.ParseIdent(),
		Type:  p.ParseTypes(),
	}
}

func (p *Parser) ParseSimpleSelectorExpr(x node.Expr) node.Expr {
	if p.Trace {
		defer untracep(tracep(p, "SimpleSelectorExpr"))
	}

L:
	for {
		switch p.Token.Token {
		case token.Period:
			p.Next()

			switch p.Token.Token {
			case token.Ident, token.LParen:
				x = p.ParseSimpleSelector(x)
			default:
				pos := p.Token.Pos
				p.ErrorExpected(pos, "simple selector")
				p.advance(stmtStart)
				return &node.BadExpr{From: pos, To: p.Token.Pos}
			}
		case token.LBrack:
			x = p.ParseIndexExpr(x)
		default:
			break L
		}
	}
	return x
}

func (p *Parser) parseType() (t *node.TypeExpr) {
	return &node.TypeExpr{Expr: p.ParseSimpleSelectorExpr(p.ParseIdent())}
}

func (p *Parser) ParseTypes() (types []*node.TypeExpr) {
	if p.Token.Token == token.Ident {
		var (
			exists = map[string]any{}
			add    = func(t *node.TypeExpr) {
				s := t.String()
				if _, ok := exists[s]; !ok {
					types = append(types, t)
					exists[s] = nil
				}
			}
		)

		add(p.parseType())
		for p.Token.Token == token.Or {
			p.Next()
			add(p.parseType())
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
	case token.LBrace:
		return p.ParseScopedBlockStmt()
	case token.Func:
		return p.ParseFuncStmt()
	case // simple statements
		token.Method, token.Ident, token.Int, token.Uint, token.Float, token.Decimal,
		token.Char, token.String, token.RawString, token.RawHeredoc, token.Symbol,
		token.True, token.False, token.Nil,
		token.LParen, token.LBrack, token.Add, token.Sub,
		token.Mul, token.And, token.Xor, token.Not, token.Import, token.Embed,
		token.Callee, token.Args, token.NamedArgs,
		token.StdIn, token.StdOut, token.StdErr,
		token.Yes, token.No,
		token.DotName, token.DotFile, token.IsMain, token.Module, token.Template:
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
	case token.Export:
		return p.ParseExportStmt()
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

	for _, el := range kva.Elements {
		switch el := el.(type) {
		case *node.KeyValuePairLit:
			c.Elements = append(c.Elements, el)
		default:
			p.Error(el.Pos(), node.NewExpectedError(el, &node.KeyValuePairLit{}).Err)
		}
	}

	c.ParseElements()

	if c.Options.Mixed {
		p.Scanner.SetMode(p.Scanner.Mode() | ScanMixed)
	} else if c.Options.NoMixed {
		p.Scanner.SetMode(p.Scanner.Mode() &^ ScanMixed)
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
	p.SkipSpace()
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
	case token.Pow:
		p.Next()
		d.Specs = append(d.Specs, &node.NamedParamSpec{
			Ident: p.ParseTypedIdent(),
			Var:   true,
		})
	case token.Mul:
		p.Next()
		d.Specs = append(d.Specs, &node.ParamSpec{
			Ident: p.ParseTypedIdent(),
			Var:   true,
		})
	case token.Ident:
		ident := p.ParseIdent()
		types := p.ParseTypes()

		if p.Token.Token == token.Assign {
			p.Next()
			d.Specs = append(d.Specs, &node.NamedParamSpec{
				Ident: &node.TypedIdentExpr{
					Ident: ident,
					Type:  types,
				},
				Value: p.ParseExpr(),
			})
		} else {
			d.Specs = append(d.Specs, &node.ParamSpec{
				Ident: &node.TypedIdentExpr{
					Ident: ident,
					Type:  types,
				},
			})
		}
	case token.LParen:
		paren := p.ParseParemExpr(token.LParen, token.RParen)
		fp, err := paren.ToMultiParenExpr().ToFuncParams()

		if err != nil {
			p.Error(err.Pos(), err.Error())
		}

		d.Lparen = fp.LParen
		d.Rparen = fp.RParen

		for _, value := range fp.Args.Values {
			d.Specs = append(d.Specs, &node.ParamSpec{
				Ident: value,
			})
		}

		if fp.Args.Var != nil {
			d.Specs = append(d.Specs, &node.ParamSpec{
				Ident: fp.Args.Var,
				Var:   true,
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
				Ident: &node.TypedIdentExpr{Ident: fp.NamedArgs.Var},
				Var:   true,
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

		ident    *node.TypedIdentExpr
		variadic bool
		named    bool
		value    node.Expr
	)

	if p.Token.IsSemi() {
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
			if t.Var {
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
		Ident: ident,
		Var:   variadic,
	}
}

func (p *Parser) ParseValueSpec(keyword token.Token, multi bool, _ []node.Spec, i int) node.Spec {
	if p.Trace {
		defer untracep(tracep(p, keyword.String()+"Spec"))
	}
	pos := p.Token.Pos
	var idents []*node.IdentExpr
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
		s1 = p.parseHeaderStmt(true)
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
			s2 = p.parseHeaderStmt(false) // cond
		}
		p.Expect(token.Semicolon)
		if !p.Token.Token.IsBlockStart() {
			s3 = p.parseHeaderStmt(false) // post
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

	var label *node.IdentExpr
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

		if p.Token.Token == token.Finally {
			finallyStmt = p.ParseFinallyStmt()
		}
	} else {
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
	var ident *node.IdentExpr
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

func (p *Parser) ParseScopedBlockStmt() *node.BlockStmt {
	if p.Trace {
		defer untracep(tracep(p, "ScopedBlockStmt"))
	}

	return p.ParseBlockStmt()
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

		defer func() {
			if _, ok := stmt.Stmts[1].(*node.CodeBeginStmt); ok {
				stmt.LBrace.Value = "{"
				stmt.Stmts = stmt.Stmts[2:]
				stmt.RBrace.Value = "}"
			}
		}()
	} else {
		stmt.LBrace = p.ExpectLit(token.LBrace)
	}

	stmt.Stmts.Append(p.ParseStmtList(ends...)...)

	if stmt.LBrace.Value == "{" || !p.IsToken(ends...) {
		stmt.RBrace = p.ExpectLit(token.RBrace)
	}

	return stmt
}

func (p *Parser) parseHeaderStmt(forIn bool) (stmt node.Stmt) {
	p.inHeader = true
	stmt = p.ParseSimpleStmt(forIn)
	p.inHeader = false
	return
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

	init = p.parseHeaderStmt(false)

	var condStmt node.Stmt
	switch p.Token.Token {
	case token.LBrace, token.MixedCodeEnd, p.BlockStart:
		condStmt = init
		init = nil
	case token.Semicolon:
		p.Next()
		condStmt = p.parseHeaderStmt(false)
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

		if ret.Assign = p.Token.Token == token.Assign; ret.Assign {
			p.Next()
		}

		ret.Result = p.ParseExpr()

		if ret.Assign {
			if _, ok := ret.Result.(*node.IdentExpr); !ok {
				p.Error(ret.Result.Pos(), fmt.Sprintf("expected *Ident, found %T", ret.Result))
			}
		}

		if p.Token.Token != token.Comma {
			goto done
		}
		// if the next token is a comma, treat it as multi return so put
		// expressions into a slice and replace x expression with an ArrayExpr.
		elements := make([]node.Expr, 1, 2)
		elements[0] = ret.Result
		for p.Token.Token == token.Comma {
			p.Next()
			ret.Result = p.ParseExpr()
			elements = append(elements, ret.Result)
		}
		ret.Result = &node.ArrayExpr{
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

			var key, value *node.IdentExpr
			var ok bool
			switch len(x) {
			case 1:
				key = node.EEmptyIdent(x[0].Pos())

				value, ok = x[0].(*node.IdentExpr)
				if !ok {
					p.ErrorExpected(x[0].Pos(), "identifier")
					value = node.EEmptyIdent(x[0].Pos())
				}
			case 2:
				key, ok = x[0].(*node.IdentExpr)
				if !ok {
					p.ErrorExpected(x[0].Pos(), "identifier")
					key = node.EEmptyIdent(x[0].Pos())
				}
				value, ok = x[1].(*node.IdentExpr)
				if !ok {
					p.ErrorExpected(x[1].Pos(), "identifier")
					value = node.EEmptyIdent(x[1].Pos())
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
		token.NullichAssign, token.LOrAssign, token.PowAssign,
		token.IncAssign, token.DecAssign:
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
	default:
		if len(x) == 1 {
			if f, _ := x[0].(*node.FuncExpr); f != nil {
				return &node.FuncStmt{Func: f}
			}
		}
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

func (p *Parser) ParseDictElementLit() *node.DictElementLit {
	if p.Trace {
		defer untracep(tracep(p, "DictElementLit"))
	}

	pos := p.Token.Pos
	var key node.Expr

	if isPrimiteValue(p.Token.Token) {
		key = p.ParseLiteral()
	} else {
		switch p.Token.Token {
		case token.LBrack:
			key = p.ParseSingleParemExpr(token.LBrack, token.RBrack)
		default:
			if p.Token.Token.IsKeyword() {
				key = &node.StringLit{ValuePos: pos, Literal: strconv.Quote(p.Token.Literal)}
				p.Next()
			} else {
				p.ErrorExpected(pos, "map key")
			}
		}
	}

	var (
		colonPos  source.Pos
		valueExpr node.Expr
	)

	switch p.Token.Token {
	case token.LParen:
		valueExpr = p.ParseDictElementLitFunc()
	case token.LBrace:
		valueExpr = p.ParseDictLit()
	default:
		colonPos = p.Expect(token.Colon)
		valueExpr = p.ParseExpr()
	}

	return &node.DictElementLit{
		Key:      key,
		ColonPos: colonPos,
		Value:    valueExpr,
	}
}

func (p *Parser) ParseDictElementLitFunc() *node.FuncDefLit {
	return p.ParseFuncDefLit(token.Colon)
}

func (p *Parser) ParseFuncDefLit(colon token.Token) *node.FuncDefLit {
	e := &node.FuncDefLit{}
	paren := p.ParseParemExpr(token.LParen, token.RParen)

	if p.Token.Token == colon {
		e.Expr = p.ParseClosureExpr(colon, paren.ToMultiParenExpr())
	} else {
		params, err := paren.ToMultiParenExpr().ToFuncParams()

		if err != nil {
			p.Error(err.Pos(), err.Error())
		}

		p.ExprLevel++
		body := p.ParseBlockStmt()
		p.ExprLevel--

		e.Expr = &node.FuncExpr{
			Type: &node.FuncType{
				Params: params,
			},
			Body: body,
		}
	}

	return e
}

func (p *Parser) ParseDictLit() *node.DictExpr {
	if p.Trace {
		defer untracep(tracep(p, "DictExpr"))
	}

	lbrace := p.Expect(token.LBrace)
	p.ExprLevel++

	var elements []*node.DictElementLit
	for p.Token.Token != token.RBrace && p.Token.Token != token.EOF {
		elements = append(elements, p.ParseDictElementLit())

		if !p.AtCommaOrNewLine("map literal", token.RBrace) {
			break
		}
		p.Next()
	}

	p.ExprLevel--
	rbrace := p.Expect(token.RBrace)
	return &node.DictExpr{
		LBrace:   lbrace,
		RBrace:   rbrace,
		Elements: elements,
	}
}

func (p *Parser) ParseKeyValueLit() *node.KeyValueLit {
	if p.Trace {
		defer untracep(tracep(p, "ParseKeyValueLit"))
	}

	p.SkipSpace()

	p.Expect(token.LBrack)

	var (
		keyExpr   = p.ParsePrimaryExpr()
		valueExpr node.Expr
	)

	p.SkipSpace()

	switch p.Token.Token {
	case token.Ident:
		if ident, _ := keyExpr.(*node.IdentExpr); ident != nil {
			keyExpr = &node.TypedIdentExpr{
				Ident: ident,
				Type:  p.ParseTypes(),
			}
		}
	case token.RParen:
		// is func or closure
		valueExpr = p.ParseFuncDefLit(token.Lambda)
		goto done
	}

	p.Expect(token.Assign)
	valueExpr = p.ParseExpr()

done:
	p.SkipSpace()
	p.Expect(token.RBrack)

	return &node.KeyValueLit{
		Key:   keyExpr,
		Value: valueExpr,
	}
}

func (p *Parser) ParseKeyValuePairLit(endToken token.Token) *node.KeyValuePairLit {
	if p.Trace {
		defer untracep(tracep(p, "ParseKeyValuePairLit"))
	}

	p.SkipSpace()

	var (
		keyExpr   = p.ParsePrimitiveOperand()
		valueExpr node.Expr
	)

	switch p.Token.Token {
	case token.Comma, endToken:
	case token.LParen:
		valueExpr = p.ParseFuncDefLit(token.Lambda)
	case token.LBrace:
		var tok PToken
		tok.Token = token.Func
		valueExpr = p.ParseFuncExprT(tok)
	case token.Ident:
		if ident, _ := keyExpr.(*node.IdentExpr); ident != nil {
			keyExpr = &node.TypedIdentExpr{
				Ident: ident,
				Type:  p.ParseTypes(),
			}
		}
		switch p.Token.Token {
		case token.Comma, endToken:
			goto done
		}
		fallthrough
	default:
		if p.Token.IsSpace() && p.PeekNoSpace().Token == token.Assign {
			p.SkipSpace()
		}

		if p.Token.Token == token.Assign {
			p.Next()
			p.SkipSpace()
			valueExpr = p.ParseExpr()
		}
	}
done:
	return &node.KeyValuePairLit{
		Key:   keyExpr,
		Value: valueExpr,
	}
}

func (p *Parser) ParseKeyValueArrayLitAt(lbrace source.Pos, rbraceToken token.Token) *node.KeyValueArrayLit {
	p.ExprLevel++
	var elements []node.Expr

l:
	for p.Token.Token != rbraceToken && p.Token.Token != token.EOF {
		switch p.Token.Token {
		case token.Pow:
			pos := p.Expect(token.Pow)
			elements = append(elements, &node.NamedArgVarLit{
				TokenPos: pos,
				Value:    p.ParseExpr(),
			})
			break l
		case token.LBrack:
			elements = append(elements, p.ParseKeyValueLit())
		default:
			elements = append(elements, p.ParseKeyValuePairLit(rbraceToken))
		}

		if !p.AtCommaOrNewLine("keyValueArray literal", rbraceToken) {
			break
		}
		p.Next()
	}

	p.ExprLevel--

	p.SkipSpace()

	if p.Token.Token != rbraceToken {
		p.Expect(rbraceToken)
	}
	rbrace := p.Token.Pos
	return &node.KeyValueArrayLit{
		LParen:   lbrace,
		RParen:   rbrace,
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

func (p *Parser) ParseExportStmt() (stmt *node.ExportStmt) {
	if p.Trace {
		defer untracep(tracep(p, "Export"))
	}

	stmt = &node.ExportStmt{
		TokenPos: p.Expect(token.Export),
	}

	if p.Failed() {
		return
	}

	p.SkipSpace()

	switch p.Token.Token {
	case token.LBrack:
		stmt.KeyExpr = p.ParseSingleParemExpr(token.LBrack, token.RBrack)
	case token.LBrace:
		stmt.ValueExpr = p.ParseDictLit()
	case token.LParen:
		stmt.ValueExpr = p.ParseSingleParemExpr(token.LParen, token.RParen)
	case token.Func:
		s := p.ParseFuncExprT(p.ExpectToken(p.Token.Token))
		if p.Failed() {
			return
		}
		stmt.ValueExpr = s
	default:
		stmt.KeyExpr = p.ParseLiteral()
		if ident, _ := stmt.KeyExpr.(*node.IdentExpr); ident != nil {
			if p.Token.Token == token.LParen {
				exp := p.ParseParenOrClosure(token.LParen, token.RParen)
				switch t := exp.(type) {
				case *node.ClosureExpr:
					stmt.ValueExpr = &node.FuncExpr{
						LambdaPos: t.Lambda.Pos,
						Type: &node.FuncType{
							NameExpr: ident,
							Params:   t.Params,
						},
						BodyExpr: t.Body,
					}
					stmt.KeyExpr = nil
				case node.ToMultiParenConverter:
					if p.Token.Token == token.LBrace {
						if params, err := t.ToMultiParenExpr().ToFuncParams(); err == nil {
							block := p.ParseBlockStmt(token.RBrace)
							stmt.ValueExpr = &node.FuncExpr{
								Type: &node.FuncType{
									Params:   params,
									NameExpr: ident,
								},
								Body: block,
							}
							stmt.KeyExpr = nil
						} else {
							return
						}
					} else {
						p.Error(exp.End(), "expected *FuncExpr | *ClosureExpr")
					}
				default:
					p.Error(exp.End(), "expected *FuncExpr | *ClosureExpr")
				}
			}
		}
	}

	if p.Failed() {
		return
	}

	if p.Token.Token == token.Assign {
		p.Next()
		stmt.ValueExpr = p.ParseExpr()
	}

	return
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

func (p *Parser) ExpectToken(token token.Token) (tok PToken) {
	tok = p.Token
	if tok.Token != token {
		p.ErrorExpected(tok.Pos, "'"+token.String()+"'")
	}
	p.Next()
	return
}

func (p *Parser) ExpectSemi() {
	switch p.Token.Token {
	case token.EOF:
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
		case token.Else, p.BlockEnd, token.DotName, token.DotFile, token.IsMain, token.Module:
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
	filePos := source.MustFilePosition(p.File, pos)
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

func (p *Parser) ErrorExpectToken(got PToken, expected ...token.Token) (tok PToken) {
	names := make([]string, len(expected))
	for i, t := range expected {
		names[i] = fmt.Sprintf("'%s'", t.String())
	}

	msg := "expected " + strings.Join(names, " | ") + ", found "
	switch {
	case got.Token == token.Semicolon && got.Literal == "\n":
		msg += "newline"
	case got.Token.IsLiteral():
		msg += got.Literal
	default:
		msg += "'" + got.Token.String() + "'"
	}

	p.Error(got.Pos, msg)
	p.Next()
	return
}

func (p *Parser) ErrorExpectedExpr(expected, got node.Expr) {
	p.Error(got.Pos(), fmt.Sprintf("expected %T, but got %s (%[2]T)", expected, got))
}

func (p *Parser) consumeComment() (comment *ast.Comment, endline int) {
	// /*-style comments may end on a different line than where they start.
	// Scan the comment for '\n' chars and adjust endline accordingly.
	endline = source.MustFileLine(p.File, p.Token.Pos)
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
	endline := source.MustFileLine(p.File, p.Token.Pos)
	for p.Token.Token == token.Comment && source.MustFileLine(p.File, p.Token.Pos) <= endline+n {
		var comment *ast.Comment
		comment, endline = p.consumeComment()
		list = append(list, comment)
	}

	comments = &ast.CommentGroup{List: list}
	p.comments = append(p.comments, comments)
	return
}

func (p *Parser) scan() (t PToken) {
	if p.ScanFunc != nil {
		return p.ScanFunc()
	}
	return p.Scanner.Scan()
}

func (p *Parser) PeekC(count int) (t []PToken) {
	if l := len(p.tokenBuffer); l > 0 {
		if count <= l {
			return p.tokenBuffer[:count]
		}
		count -= l
	}

	for ; count > 0; count-- {
		p.tokenBuffer = append(p.tokenBuffer, p.scan())
	}

	return p.tokenBuffer
}

func (p *Parser) PeekCb(on func(t PToken) (more bool)) {
	for _, t := range p.tokenBuffer {
		if !on(t) {
			return
		}
	}

	for {
		t := p.scan()
		p.tokenBuffer = append(p.tokenBuffer, t)
		if !on(t) {
			return
		}
	}
}

func (p *Parser) Peek() PToken {
	t := p.PeekC(1)
	return t[0]
}

func (p *Parser) PeekNoSpace() (r PToken) {
	p.PeekCb(func(t PToken) bool {
		if t.IsSpace() {
			return true
		}
		r = t
		return false
	})
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

	for _, t := range p.tokenBuffer {
		p.tokenBuffer = p.tokenBuffer[1:]
		p.Token = t
		return
	}

	p.Token = p.scan()
}

func (p *Parser) Next() *Parser {
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
		if source.MustFileLine(p.File, p.Token.Pos) == source.MustFileLine(p.File, prev) {
			// line comment of prev token
			_ = p.consumeCommentGroup(0)
		}
		// consume successor comments, if any
		for p.Token.Token == token.Comment {
			// lead comment of next token
			_ = p.consumeCommentGroup(1)
		}
	}

	p.postScan(p)
	return p
}

func (p *Parser) SkipSpace() {
	for p.Token.IsSpace() {
		p.Next()
	}
}

func (p *Parser) PrintTrace(a ...any) {
	const (
		dots = ". . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . "
		n    = len(dots)
	)

	filePos := source.MustFilePosition(p.File, p.Token.Pos)
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

func isPrimiteValue(tok token.Token) bool {
	switch tok {
	case token.Ident,
		token.Int,
		token.Uint,
		token.Float,
		token.Decimal,
		token.Char,
		token.String,
		token.RawString,
		token.RawHeredoc,
		token.Symbol,
		token.Nil,
		token.True,
		token.False,
		token.Yes,
		token.No:
		return true
	}
	return false
}
