// A modified version Go and Tengo parsers.

// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

// Copyright (c) 2019 Daniel Kang.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE.tengo file.

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.golang file.

package parser

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

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
	Pos SourceFilePos
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
func (p *ErrorList) Add(pos SourceFilePos, msg string) {
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

// Parser parses the Tengo source files. It's based on Go's parser
// implementation.
type Parser struct {
	file      *SourceFile
	errors    ErrorList
	scanner   *Scanner
	token     Token
	prevToken token.Token
	exprLevel int // < 0: in control clause, >= 0: in expression
	syncPos   Pos // last sync position
	syncCount int // number of advance calls without progress
	trace     bool
	indent    int
	mode      Mode
	traceOut  io.Writer
	comments  []*CommentGroup
}

// NewParser creates a Parser.
func NewParser(file *SourceFile, src []byte, trace io.Writer) *Parser {
	return NewParserWithMode(file, src, trace, 0)
}

// NewParserWithMode creates a Parser with parser mode flags.
func NewParserWithMode(
	file *SourceFile,
	src []byte,
	trace io.Writer,
	mode Mode,
) *Parser {
	p := &Parser{
		file:     file,
		trace:    trace != nil,
		traceOut: trace,
		mode:     mode,
	}
	var m ScanMode
	if mode.Has(ParseComments) {
		m.Set(ScanComments)
	}
	if mode.Has(ParseMixed) {
		m.Set(Mixed)
	}
	if mode.Has(ParseConfigDisabled) {
		m.Set(ConfigDisabled)
	}
	p.scanner = NewScanner(p.file, src,
		func(pos SourceFilePos, msg string) {
			p.errors.Add(pos, msg)
		}, m)
	p.next()
	return p
}

// ParseFile parses the source and returns an AST file unit.
func (p *Parser) ParseFile() (file *File, err error) {
	defer func() {
		if e := recover(); e != nil {
			if _, ok := e.(bailout); !ok {
				panic(e)
			}
		}

		p.errors.Sort()
		err = p.errors.Err()
	}()

	if p.trace {
		defer untracep(tracep(p, "File"))
	}

	if p.errors.Len() > 0 {
		return nil, p.errors.Err()
	}

	stmts, _ := p.parseStmtList(0)
	p.expect(token.EOF)
	if p.errors.Len() > 0 {
		return nil, p.errors.Err()
	}

	file = &File{
		InputFile: p.file,
		Stmts:     stmts,
		Comments:  p.comments,
	}
	return
}

func (p *Parser) parseExpr() Expr {
	if p.trace {
		defer untracep(tracep(p, "Expression"))
	}

	expr := p.parseBinaryExpr(token.LowestPrec + 1)

	// ternary conditional expression
	if p.token.Token == token.Question {
		return p.parseCondExpr(expr)
	}
	return expr
}

func (p *Parser) parseBinaryExpr(prec1 int) Expr {
	if p.trace {
		defer untracep(tracep(p, "BinaryExpression"))
	}

	x := p.parseUnaryExpr()

	for {
		op, prec := p.token.Token, p.token.Token.Precedence()
		if prec < prec1 {
			return x
		}

		pos := p.expect(op)

		y := p.parseBinaryExpr(prec + 1)

		if op == token.Equal || op == token.NotEqual {
			if _, ok := x.(*NilLit); ok {
				if op == token.Equal {
					op = token.Null
				} else {
					op = token.NotNull
				}
				x = &UnaryExpr{
					Expr:     y,
					Token:    op,
					TokenPos: pos,
				}
				continue
			} else if _, ok := y.(*NilLit); ok {
				if op == token.Equal {
					op = token.Null
				} else {
					op = token.NotNull
				}
				x = &UnaryExpr{
					Expr:     x,
					Token:    op,
					TokenPos: pos,
				}
				continue
			}
		}

		x = &BinaryExpr{
			LHS:      x,
			RHS:      y,
			Token:    op,
			TokenPos: pos,
		}
	}
}

func (p *Parser) parseCondExpr(cond Expr) Expr {
	questionPos := p.expect(token.Question)
	trueExpr := p.parseExpr()
	colonPos := p.expect(token.Colon)
	falseExpr := p.parseExpr()

	return &CondExpr{
		Cond:        cond,
		True:        trueExpr,
		False:       falseExpr,
		QuestionPos: questionPos,
		ColonPos:    colonPos,
	}
}

func (p *Parser) parseUnaryExpr() Expr {
	if p.trace {
		defer untracep(tracep(p, "UnaryExpression"))
	}

	switch p.token.Token {
	case token.Add, token.Sub, token.Not, token.Xor:
		pos, op := p.token.Pos, p.token.Token
		p.next()
		x := p.parseUnaryExpr()
		return &UnaryExpr{
			Token:    op,
			TokenPos: pos,
			Expr:     x,
		}
	}
	return p.parsePrimaryExpr()
}

func (p *Parser) parsePrimaryExpr() Expr {
	if p.trace {
		defer untracep(tracep(p, "PrimaryExpression"))
	}

	x := p.parseOperand()

L:
	for {
		switch p.token.Token {
		case token.NullishSelector:
			p.next()

			switch p.token.Token {
			case token.Ident, token.LParen:
				x = p.parseNullishSelector(x)
			default:
				pos := p.token.Pos
				p.errorExpected(pos, "nullish selector")
				p.advance(stmtStart)
				return &BadExpr{From: pos, To: p.token.Pos}
			}
		case token.Period:
			p.next()

			switch p.token.Token {
			case token.Ident, token.LParen:
				x = p.parseSelector(x)
			default:
				pos := p.token.Pos
				p.errorExpected(pos, "selector")
				p.advance(stmtStart)
				return &BadExpr{From: pos, To: p.token.Pos}
			}
		case token.LBrack:
			x = p.parseIndexOrSlice(x)
		case token.LParen:
			x = p.parseCall(x)
		default:
			break L
		}
	}
	return x
}

func (p *Parser) parseCall(x Expr) *CallExpr {
	if p.trace {
		defer untracep(tracep(p, "Call"))
	}

	lparen := p.expect(token.LParen)
	p.exprLevel++

	var (
		args      CallExprArgs
		namedArgs CallExprNamedArgs
	)

	for p.token.Token != token.RParen && p.token.Token != token.EOF && p.token.Token != token.Semicolon {
		if p.token.Token == token.Ellipsis {
			elipsis := &EllipsisValue{Pos: p.token.Pos}
			p.next()
			elipsis.Value = p.parseExpr()
			if _, ok := elipsis.Value.(*MapLit); ok {
				namedArgs.Ellipsis = elipsis
				goto done
			} else {
				args.Ellipsis = elipsis
			}
			goto kw
		}
		args.Values = append(args.Values, p.parseExpr())
		switch p.token.Token {
		case token.Assign:
			val := args.Values[len(args.Values)-1]
			args.Values = args.Values[:len(args.Values)-1]
			switch t := val.(type) {
			case *Ident:
				namedArgs.Names = append(namedArgs.Names, NamedArgExpr{Ident: t})
			case *StringLit:
				namedArgs.Names = append(namedArgs.Names, NamedArgExpr{String: t})
			default:
				p.errorExpected(val.Pos(), "string|ident")
			}
			p.next()
			namedArgs.Values = append(namedArgs.Values, p.parseExpr())
			goto kw
		case token.Semicolon:
			goto kw
		}
		if !p.atComma("call argument", token.RParen) {
			break
		}
		p.next()
	}

kw:
	if (p.token.Token == token.Semicolon && p.token.Literal == ";") ||
		(p.token.Token == token.Comma && (len(namedArgs.Names) == 1 || args.Ellipsis != nil)) {
		p.next()

		for {
			switch p.token.Token {
			case token.Ellipsis:
				namedArgs.Ellipsis = &EllipsisValue{Pos: p.token.Pos}
				p.next()
				namedArgs.Ellipsis.Value = p.parseExpr()
				goto done
			case token.RParen, token.EOF:
				goto done
			default:
				expr := p.parsePrimaryExpr()
				switch t := expr.(type) {
				case *Ident:
					namedArgs.Names = append(namedArgs.Names, NamedArgExpr{Ident: t})
				case *StringLit:
					namedArgs.Names = append(namedArgs.Names, NamedArgExpr{String: t})
				case *CallExpr, *SelectorExpr, *MapLit:
					namedArgs.Ellipsis = &EllipsisValue{p.token.Pos, t}
					p.expect(token.Ellipsis)
					if !p.atComma("call argument", token.RParen) {
						goto done
					}
				default:
					pos := p.token.Pos
					p.errorExpected(pos, "string|ident|selector|call")
					p.advance(stmtStart)
					goto done
				}

				// check if is flag
				switch p.token.Token {
				case token.Comma, token.RParen:
					namedArgs.Values = append(namedArgs.Values, nil)
				// is flag
				default:
					p.expect(token.Assign)
					namedArgs.Values = append(namedArgs.Values, p.parseExpr())
				}

				if !p.atComma("call argument", token.RParen) {
					break
				}

				p.next()
			}
		}
	}

done:
	p.exprLevel--
	rparen := p.expect(token.RParen)
	return &CallExpr{
		Func:      x,
		LParen:    lparen,
		RParen:    rparen,
		Args:      args,
		NamedArgs: namedArgs,
	}
}

func (p *Parser) atComma(context string, follow token.Token) bool {
	if p.token.Token == token.Comma {
		return true
	}
	if p.token.Token != follow {
		msg := "missing ','"
		if p.token.Token == token.Semicolon && p.token.Literal == "\n" {
			msg += " before newline"
		}
		p.error(p.token.Pos, msg+" in "+context)
		return true // "insert" comma and continue
	}
	return false
}

func (p *Parser) parseIndexOrSlice(x Expr) Expr {
	if p.trace {
		defer untracep(tracep(p, "IndexOrSlice"))
	}

	lbrack := p.expect(token.LBrack)
	p.exprLevel++

	var index [2]Expr
	if p.token.Token != token.Colon {
		index[0] = p.parseExpr()
	}
	numColons := 0
	if p.token.Token == token.Colon {
		numColons++
		p.next()

		if p.token.Token != token.RBrack && p.token.Token != token.EOF {
			index[1] = p.parseExpr()
		}
	}

	p.exprLevel--
	rbrack := p.expect(token.RBrack)

	if numColons > 0 {
		// slice expression
		return &SliceExpr{
			Expr:   x,
			LBrack: lbrack,
			RBrack: rbrack,
			Low:    index[0],
			High:   index[1],
		}
	}
	return &IndexExpr{
		Expr:   x,
		LBrack: lbrack,
		RBrack: rbrack,
		Index:  index[0],
	}
}

func (p *Parser) parseSelector(x Expr) Expr {
	if p.trace {
		defer untracep(tracep(p, "Selector"))
	}

	var sel Expr
	if p.token.Token == token.LParen {
		lparen := p.token.Pos
		p.next()
		sel = p.parseExpr()
		rparen := p.expect(token.RParen)
		sel = &ParenExpr{sel, lparen, rparen}
	} else {
		ident := p.parseIdent()
		sel = &StringLit{
			Value:    ident.Name,
			ValuePos: ident.NamePos,
			Literal:  ident.Name,
		}
	}
	return &SelectorExpr{Expr: x, Sel: sel}
}

func (p *Parser) parseNullishSelector(x Expr) Expr {
	if p.trace {
		defer untracep(tracep(p, "NullishSelector"))
	}

	var sel Expr
	if p.token.Token == token.LParen {
		lparen := p.token.Pos
		p.next()
		sel = p.parseExpr()
		rparen := p.expect(token.RParen)
		sel = &ParenExpr{sel, lparen, rparen}
	} else {
		ident := p.parseIdent()
		sel = &StringLit{
			Value:    ident.Name,
			ValuePos: ident.NamePos,
			Literal:  ident.Name,
		}
	}

	return &NullishSelectorExpr{Expr: x, Sel: sel}
}

func (p *Parser) parsePrimitiveOperand() Expr {
	switch p.token.Token {
	case token.Ident:
		return p.parseIdent()
	case token.Int:
		v, _ := strconv.ParseInt(p.token.Literal, 0, 64)
		x := &IntLit{
			Value:    v,
			ValuePos: p.token.Pos,
			Literal:  p.token.Literal,
		}
		p.next()
		return x
	case token.Uint:
		v, _ := strconv.ParseUint(strings.TrimSuffix(p.token.Literal, "u"), 0, 64)
		x := &UintLit{
			Value:    v,
			ValuePos: p.token.Pos,
			Literal:  p.token.Literal,
		}
		p.next()
		return x
	case token.Float:
		v, _ := strconv.ParseFloat(p.token.Literal, 64)
		x := &FloatLit{
			Value:    v,
			ValuePos: p.token.Pos,
			Literal:  p.token.Literal,
		}
		p.next()
		return x
	case token.Char:
		return p.parseCharLit()
	case token.String:
		v, _ := strconv.Unquote(p.token.Literal)
		x := &StringLit{
			Value:    v,
			ValuePos: p.token.Pos,
			Literal:  p.token.Literal,
		}
		p.next()
		return x
	case token.True:
		x := &BoolLit{
			Value:    true,
			ValuePos: p.token.Pos,
			Literal:  p.token.Literal,
		}
		p.next()
		return x
	case token.False:
		x := &BoolLit{
			Value:    false,
			ValuePos: p.token.Pos,
			Literal:  p.token.Literal,
		}
		p.next()
		return x
	case token.Nil:
		x := &NilLit{TokenPos: p.token.Pos}
		p.next()
		return x
	case token.Callee:
		x := &CalleeKeyword{TokenPos: p.token.Pos, Literal: p.token.Literal}
		p.next()
		return x
	case token.Args:
		x := &ArgsKeyword{TokenPos: p.token.Pos, Literal: p.token.Literal}
		p.next()
		return x
	case token.NamedArgs:
		x := &NamedArgsKeyword{TokenPos: p.token.Pos, Literal: p.token.Literal}
		p.next()
		return x
	case token.StdIn:
		x := &StdInLit{TokenPos: p.token.Pos}
		p.next()
		return x
	case token.StdOut:
		x := &StdOutLit{TokenPos: p.token.Pos}
		p.next()
		return x
	case token.StdErr:
		x := &StdErrLit{TokenPos: p.token.Pos}
		p.next()
		return x
	}

	pos := p.token.Pos
	p.errorExpected(pos, "primitive operand")
	p.advance(stmtStart)
	return &BadExpr{From: pos, To: p.token.Pos}
}

func (p *Parser) parseOperand() Expr {
	if p.trace {
		defer untracep(tracep(p, "Operand"))
	}

	switch p.token.Token {
	case token.Ident:
		return p.parseIdent()
	case token.Int:
		v, _ := strconv.ParseInt(p.token.Literal, 0, 64)
		x := &IntLit{
			Value:    v,
			ValuePos: p.token.Pos,
			Literal:  p.token.Literal,
		}
		p.next()
		return x
	case token.Uint:
		v, _ := strconv.ParseUint(strings.TrimSuffix(p.token.Literal, "u"), 0, 64)
		x := &UintLit{
			Value:    v,
			ValuePos: p.token.Pos,
			Literal:  p.token.Literal,
		}
		p.next()
		return x
	case token.Float:
		v, _ := strconv.ParseFloat(p.token.Literal, 64)
		x := &FloatLit{
			Value:    v,
			ValuePos: p.token.Pos,
			Literal:  p.token.Literal,
		}
		p.next()
		return x
	case token.Decimal:
		v, err := decimal.NewFromString(strings.TrimSuffix(p.token.Literal, "d"))
		if err != nil {
			p.error(p.token.Pos, err.Error())
		}
		x := &DecimalLit{
			Value:    v,
			ValuePos: p.token.Pos,
			Literal:  p.token.Literal,
		}
		p.next()
		return x
	case token.Char:
		return p.parseCharLit()
	case token.String:
		v, _ := strconv.Unquote(p.token.Literal)
		x := &StringLit{
			Value:    v,
			ValuePos: p.token.Pos,
			Literal:  p.token.Literal,
		}
		p.next()
		return x
	case token.True:
		x := &BoolLit{
			Value:    true,
			ValuePos: p.token.Pos,
			Literal:  p.token.Literal,
		}
		p.next()
		return x
	case token.False:
		x := &BoolLit{
			Value:    false,
			ValuePos: p.token.Pos,
			Literal:  p.token.Literal,
		}
		p.next()
		return x
	case token.Nil:
		x := &NilLit{TokenPos: p.token.Pos}
		p.next()
		return x
	case token.StdIn:
		x := &StdInLit{TokenPos: p.token.Pos}
		p.next()
		return x
	case token.StdOut:
		x := &StdOutLit{TokenPos: p.token.Pos}
		p.next()
		return x
	case token.StdErr:
		x := &StdErrLit{TokenPos: p.token.Pos}
		p.next()
		return x
	case token.Callee:
		x := &CalleeKeyword{TokenPos: p.token.Pos, Literal: p.token.Literal}
		p.next()
		return x
	case token.Args:
		x := &ArgsKeyword{TokenPos: p.token.Pos, Literal: p.token.Literal}
		p.next()
		return x
	case token.NamedArgs:
		x := &NamedArgsKeyword{TokenPos: p.token.Pos, Literal: p.token.Literal}
		p.next()
		return x
	case token.Import:
		return p.parseImportExpr()
	case token.LParen, token.Begin:
		return p.parseParemExpr()
	case token.LBrack: // array literal
		return p.parseArrayLit()
	case token.LBrace: // map literal
		return p.parseMapLit()
	case token.Func: // function literal
		return p.parseFuncLit()
	case token.Text:
		return p.parseTextStmt()
	}

	pos := p.token.Pos
	p.errorExpected(pos, "operand")
	p.advance(stmtStart)
	return &BadExpr{From: pos, To: p.token.Pos}
}

func (p *Parser) parseImportExpr() Expr {
	pos := p.token.Pos
	p.next()
	p.expect(token.LParen)
	if p.token.Token != token.String {
		p.errorExpected(p.token.Pos, "module name")
		p.advance(stmtStart)
		return &BadExpr{From: pos, To: p.token.Pos}
	}

	// module name
	moduleName, _ := strconv.Unquote(p.token.Literal)
	expr := &ImportExpr{
		ModuleName: moduleName,
		Token:      token.Import,
		TokenPos:   pos,
	}

	p.next()
	p.expect(token.RParen)
	return expr
}

func (p *Parser) parseParemExpr() Expr {
	if p.trace {
		defer untracep(tracep(p, "ParemExpr"))
	}

	lparen := p.token.Pos
	end := token.RParen
	switch p.token.Token {
	case token.LParen:
	case token.Begin:
		end = token.End
	default:
		p.errorExpected(lparen, "'"+token.LParen.String()+"' or '"+token.Begin.String()+"'")
		return nil
	}
	p.next()
	if end == token.End && p.token.Token == token.Semicolon {
		p.next()
	}
	if p.token.Token == token.Semicolon && p.token.Literal == ";" {
		return p.parseKeyValueArrayLit(lparen)
	}
	p.exprLevel++
	x := p.parseExpr()
	p.exprLevel--
	rparen := p.expect(end)

	return &ParenExpr{
		LParen: lparen,
		Expr:   x,
		RParen: rparen,
	}
}
func (p *Parser) parseCharLit() Expr {
	if n := len(p.token.Literal); n >= 3 {
		code, _, _, err := strconv.UnquoteChar(p.token.Literal[1:n-1], '\'')
		if err == nil {
			x := &CharLit{
				Value:    code,
				ValuePos: p.token.Pos,
				Literal:  p.token.Literal,
			}
			p.next()
			return x
		}
	}

	pos := p.token.Pos
	p.error(pos, "illegal char literal")
	p.next()
	return &BadExpr{
		From: pos,
		To:   p.token.Pos,
	}
}

func (p *Parser) parseFuncLit() Expr {
	if p.trace {
		defer untracep(tracep(p, "FuncLit"))
	}

	typ := p.parseFuncType()
	p.exprLevel++
	body, closure := p.parseBody()
	p.exprLevel--
	if closure != nil {
		return &ClosureLit{
			Type: typ,
			Body: closure,
		}
	}
	return &FuncLit{
		Type: typ,
		Body: body,
	}
}

func (p *Parser) parseArrayLit() Expr {
	if p.trace {
		defer untracep(tracep(p, "ArrayLit"))
	}

	lbrack := p.expect(token.LBrack)
	p.exprLevel++

	var elements []Expr
	for p.token.Token != token.RBrack && p.token.Token != token.EOF {
		elements = append(elements, p.parseExpr())

		if !p.atComma("array literal", token.RBrack) {
			break
		}
		p.next()
	}

	p.exprLevel--
	rbrack := p.expect(token.RBrack)
	return &ArrayLit{
		Elements: elements,
		LBrack:   lbrack,
		RBrack:   rbrack,
	}
}

func (p *Parser) parseFuncType() *FuncType {
	if p.trace {
		defer untracep(tracep(p, "FuncType"))
	}

	pos := p.expect(token.Func)
	params := p.parseFuncParams()
	return &FuncType{
		FuncPos: pos,
		Params:  *params,
	}
}

func (p *Parser) parseFuncParam(prev Spec) (spec Spec) {
	p.skipSpace()

	var (
		pos = p.token.Pos

		ident    *Ident
		variadic bool
		named    bool
		value    Expr
	)

	if p.token.Token == token.Semicolon && p.token.Literal == ";" {
		p.next()
		p.skipSpace()
		named = true
	} else if prev != nil {
		switch t := prev.(type) {
		case *NamedParamSpec:
			if t.Value == nil {
				p.error(pos, "unexpected func param declaration")
				p.expectSemi()
			}
			named = true
		case *ParamSpec:
			if t.Variadic {
				named = true
			}
		}
	}

	if p.token.Token == token.Ident {
		ident = p.parseIdent()
		p.skipSpace()
		if named {
			p.expect(token.Assign)
			value = p.parseExpr()
		} else if p.token.Literal == ";" {
			goto done
		}
	} else if p.token.Token == token.Ellipsis {
		variadic = true
		p.next()
		ident = p.parseIdent()
		p.skipSpace()
	}

	if p.token.Token == token.Comma {
		p.next()
		p.skipSpace()
	} else if p.token.Token == token.Semicolon {
		if p.token.Token == token.Assign {
			named = true
			p.next()
			value = p.parseExpr()
			p.skipSpace()
			if p.token.Token == token.Comma {
				p.next()
				p.skipSpace()
			}
		} else if !named {
			p.expectSemi()
		}
	}

	if ident == nil {
		p.error(pos, "wrong func params declaration")
		p.expectSemi()
	}

	if named {
		if value == nil && !variadic {
			p.error(pos, "wrong func params declaration")
		}
		return &NamedParamSpec{
			Ident: ident,
			Value: value,
		}
	}
done:
	return &ParamSpec{
		Ident:    ident,
		Variadic: variadic,
	}
}

func (p *Parser) parseFuncParams() *FuncParams {
	if p.trace {
		defer untracep(tracep(p, "FuncParams"))
	}

	var (
		args      ArgsList
		namedArgs NamedArgsList
		lparen    = p.token.Pos
		spec      Spec
	)

	p.next()

	for i := 0; p.token.Token != token.RParen && p.token.Token != token.EOF; i++ { //nolint:predeclared
		spec = p.parseFuncParam(spec)
		if p, _ := spec.(*ParamSpec); p != nil {
			if p.Variadic {
				args.Var = p.Ident
			} else {
				args.Values = append(args.Values, p.Ident)
			}
		} else {
			p := spec.(*NamedParamSpec)
			if p.Value == nil {
				namedArgs.Var = p.Ident
			} else {
				namedArgs.Names = append(namedArgs.Names, p.Ident)
				namedArgs.Values = append(namedArgs.Values, p.Value)
			}
		}
	}

	rparen := p.expect(token.RParen)

	return &FuncParams{
		LParen:    lparen,
		RParen:    rparen,
		Args:      args,
		NamedArgs: namedArgs,
	}
}

func (p *Parser) parseBody() (b *BlockStmt, closure Expr) {
	if p.trace {
		defer untracep(tracep(p, "Body"))
	}

	p.skipSpace()

	if p.token.Token == token.Assign {
		p.next()
		p.expect(token.Greater)

		if p.token.Token.IsBlockStart() {
			closure = &BlockExpr{p.parseBlockStmt()}
		} else {
			closure = p.parseExpr()
		}
	} else {
		b = p.parseBlockStmt(BlockWrap{
			Start: token.Do,
			Ends: []BlockEnd{
				{token.End, true},
			},
		})
	}
	return
}

func (p *Parser) parseStmtList(start token.Token, ends ...BlockWrap) (list []Stmt, end *BlockEnd) {
	if p.trace {
		defer untracep(tracep(p, "StatementList"))
	}

	var s Stmt

	for {
		switch p.token.Token {
		case token.EOF, token.RBrace:
			return
		case token.Semicolon:
			p.next()
		default:
			if start != 0 {
				for _, end_ := range ends {
					if start == end_.Start {
						for _, e := range end_.Ends {
							if p.token.Token == e.Token {
								if e.Next {
									p.next()
								}
								end = &e
								return
							}
						}
					}
				}
			}
			if s = p.parseStmt(); s != nil {
				if _, ok := s.(*EmptyStmt); ok {
					continue
				}
				list = append(list, s)
			}
		}
	}
}

func (p *Parser) parseIdent() *Ident {
	pos := p.token.Pos
	name := "_"

	if p.token.Token == token.Ident {
		name = p.token.Literal
		p.next()
	} else {
		p.expect(token.Ident)
	}
	return &Ident{
		NamePos: pos,
		Name:    name,
	}
}

func (p *Parser) parseStmt() (stmt Stmt) {
	if p.trace {
		defer untracep(tracep(p, "Statement"))
	}

do:
	switch p.token.Token {
	case token.Config:
		return p.parseConfigStmt()
	case token.Text:
		return p.parseTextStmt()
	case token.ToTextBegin:
		return p.parseExprToTextStmt()
	case token.Var, token.Const, token.Global, token.Param:
		return &DeclStmt{Decl: p.parseDecl()}
	case // simple statements
		token.Func, token.Ident, token.Int, token.Uint, token.Float,
		token.Char, token.String, token.True, token.False, token.Nil,
		token.LParen, token.LBrace, token.LBrack, token.Add, token.Sub,
		token.Mul, token.And, token.Xor, token.Not, token.Import,
		token.Callee, token.Args, token.NamedArgs,
		token.StdIn, token.StdOut, token.StdErr,
		token.Then:
		s := p.parseSimpleStmt(false)
		p.expectSemi()
		return s
	case token.Return:
		return p.parseReturnStmt()
	case token.If:
		return p.parseIfStmt()
	case token.For:
		return p.parseForStmt()
	case token.Try:
		return p.parseTryStmt()
	case token.Throw:
		return p.parseThrowStmt()
	case token.Break, token.Continue:
		return p.parseBranchStmt(p.token.Token)
	case token.Semicolon:
		p.next()
		goto do
	case token.RBrace, token.End:
		// semicolon may be omitted before a closing "}"
		return &EmptyStmt{Semicolon: p.token.Pos, Implicit: true}
	default:
		pos := p.token.Pos
		p.errorExpected(pos, "statement")
		p.advance(stmtStart)
		return &BadStmt{From: pos, To: p.token.Pos}
	}
}

func (p *Parser) parseConfigStmt() (c *ConfigStmt) {
	if p.trace {
		defer untracep(tracep(p, "ConfigStmt"))
	}
	c = &ConfigStmt{
		ConfigPos: p.token.Pos,
		EndPos:    p.token.Data.(Pos),
		Literal:   p.token.Literal,
	}

	p.next()

	testFileSet := NewFileSet()
	configLit := "(;" + c.Literal + ")"
	testFile := testFileSet.AddFile("config", -1, len(configLit))
	p2 := NewParserWithMode(testFile, []byte(configLit), nil, ParseConfigDisabled)
	f, err := p2.ParseFile()
	if err != nil {
		p2.error(c.ConfigPos, err.Error())
	} else {
		cfg := f.Stmts[0].(*ExprStmt).Expr.(*KeyValueArrayLit).Elements
		for _, k := range cfg {
			switch k.Key.String() {
			case "mixed":
				if k.Value == nil {
					c.Options.Mixed = true
				} else if b, ok := k.Value.(*BoolLit); ok {
					if b.Value {
						c.Options.Mixed = true
					} else {
						c.Options.NoMixed = true
					}
				}
			case "writer":
				if k.Value != nil {
					c.Options.WriteFunc = k.Value
				}
			}
		}
	}

	if c.Options.Mixed {
		p.scanner.mode.Set(Mixed)
	} else if c.Options.NoMixed {
		p.scanner.mode.Clear(Mixed)
	}

	if !p.scanner.mode.Has(Mixed) {
		p.expect(token.Semicolon)
	}
	return
}

func (p *Parser) parseExprToTextStmt() (ett *ExprToTextStmt) {
	if p.trace {
		defer untracep(tracep(p, "ExprToText"))
	}
	ett = &ExprToTextStmt{
		StartLit: Literal{p.token.Literal, p.token.Pos},
	}
	p.next()
	ett.Expr = p.parseExpr()
	if p.token.Token == token.ToTextEnd {
		ett.EndLit = Literal{p.token.Literal, p.token.Pos}
		p.next()
	} else {
		p.expect(token.ToTextEnd)
	}
	return
}

func (p *Parser) parseTextStmt() (t *TextStmt) {
	if p.trace {
		defer untracep(tracep(p, "TextStmt"))
	}
	t = &TextStmt{p.token.Literal, p.token.Pos}
	p.next()
	return
}

func (p *Parser) parseDecl() Decl {
	if p.trace {
		defer untracep(tracep(p, "DeclStmt"))
	}
	switch p.token.Token {
	case token.Global, token.Param:
		return p.parseGenDecl(p.token.Token, p.parseParamSpec)
	case token.Var, token.Const:
		return p.parseGenDecl(p.token.Token, p.parseValueSpec)
	default:
		p.error(p.token.Pos, "only \"param, global, var\" declarations supported")
		return &BadDecl{From: p.token.Pos, To: p.token.Pos}
	}
}

func (p *Parser) parseGenDecl(
	keyword token.Token,
	fn func(token.Token, bool, []Spec, int) Spec,
) *GenDecl {
	if p.trace {
		defer untracep(tracep(p, "GenDecl("+keyword.String()+")"))
	}
	pos := p.expect(keyword)
	var lparen, rparen Pos
	var list []Spec
	if p.token.Token == token.LParen {
		lparen = p.token.Pos
		p.next()
		for i := 0; p.token.Token != token.RParen && p.token.Token != token.EOF; i++ { //nolint:predeclared
			list = append(list, fn(keyword, true, list, i))
		}
		rparen = p.expect(token.RParen)
		p.expectSemi()
	} else {
		list = append(list, fn(keyword, false, list, 0))
		p.expectSemi()
	}
	return &GenDecl{
		TokPos: pos,
		Tok:    keyword,
		Lparen: lparen,
		Specs:  list,
		Rparen: rparen,
	}
}

func (p *Parser) parseParamSpec(keyword token.Token, multi bool, prev []Spec, i int) (spec Spec) {
	if p.trace {
		defer untracep(tracep(p, keyword.String()+"Spec"))
	}

	if multi {
		p.skipSpace()
	}

	var (
		pos = p.token.Pos

		ident    *Ident
		variadic bool
		named    bool
		value    Expr
	)

	if p.token.Token == token.Semicolon && p.token.Literal == ";" {
		p.next()
		if multi {
			p.skipSpace()
		}
		named = true
	} else if i > 0 {
		switch t := prev[i-1].(type) {
		case *NamedParamSpec:
			if t.Value == nil {
				p.error(pos, "unexpected arg declaration")
				p.expectSemi()
			}
			named = true
		case *ParamSpec:
			if t.Variadic {
				named = true
			}
		}
	}

	if p.token.Token == token.Ident {
		ident = p.parseIdent()
	} else if keyword == token.Param && p.token.Token == token.Ellipsis {
		variadic = true
		p.next()
		ident = p.parseIdent()
		if multi {
			p.skipSpace()
		}
	}

	if multi && p.token.Token == token.Comma {
		p.next()
		p.skipSpace()
	} else if multi {
		if p.token.Token == token.Assign {
			named = true
			p.next()
			value = p.parseExpr()
			if p.token.Token == token.Comma || (p.token.Token == token.Semicolon && p.token.Literal == "\n") {
				p.next()
				p.skipSpace()
			}
		} else if !named {
			p.expectSemi()
		}
	}

	if ident == nil {
		p.error(pos, fmt.Sprintf("wrong %s declaration", keyword.String()))
		p.expectSemi()
	}

	if named {
		if value == nil && !variadic {
			p.error(pos, fmt.Sprintf("wrong %s declaration", keyword.String()))
		}
		return &NamedParamSpec{
			Ident: ident,
			Value: value,
		}
	}

	return &ParamSpec{
		Ident:    ident,
		Variadic: variadic,
	}
}

func (p *Parser) parseValueSpec(keyword token.Token, multi bool, _ []Spec, i int) Spec {
	if p.trace {
		defer untracep(tracep(p, keyword.String()+"Spec"))
	}
	pos := p.token.Pos
	var idents []*Ident
	var values []Expr
	if p.token.Token == token.Ident {
		ident := p.parseIdent()
		var expr Expr
		if p.token.Token == token.Assign {
			p.next()
			expr = p.parseExpr()
		}
		if keyword == token.Const && expr == nil {
			if i == 0 {
				p.error(p.token.Pos, "missing initializer in const declaration")
			}
		}
		idents = append(idents, ident)
		values = append(values, expr)
		if multi && p.token.Token == token.Comma {
			p.next()
		} else if multi {
			p.expectSemi()
		}
	}
	if len(idents) == 0 {
		p.error(pos, "wrong var declaration")
		p.expectSemi()
	}
	spec := &ValueSpec{
		Idents: idents,
		Values: values,
		Data:   i,
	}
	return spec
}

func (p *Parser) parseForStmt() Stmt {
	if p.trace {
		defer untracep(tracep(p, "ForStmt"))
	}

	pos := p.expect(token.For)

	// for {}
	if p.token.Token.IsBlockStart() {
		body := p.parseBlockStmt(BlockWrap{token.Do, []BlockEnd{{token.End, true}}})
		p.expectSemi()

		return &ForStmt{
			ForPos: pos,
			Body:   body,
		}
	}

	prevLevel := p.exprLevel
	p.exprLevel = -1

	var s1 Stmt
	if p.token.Token != token.Semicolon { // skipping init
		s1 = p.parseSimpleStmt(true)
	}

	// for _ in seq {}            or
	// for value in seq {}        or
	// for key, value in seq {}
	if forInStmt, isForIn := s1.(*ForInStmt); isForIn {
		forInStmt.ForPos = pos
		p.exprLevel = prevLevel
		forInStmt.Body = p.parseBlockStmt(BlockWrap{token.Do, []BlockEnd{{token.End, true}}})
		p.expectSemi()
		return forInStmt
	}

	// for init; cond; post {}
	var s2, s3 Stmt
	if p.token.Token == token.Semicolon {
		p.next()
		if p.token.Token != token.Semicolon {
			s2 = p.parseSimpleStmt(false) // cond
		}
		p.expect(token.Semicolon)
		if !p.token.Token.IsBlockStart() {
			s3 = p.parseSimpleStmt(false) // post
		}
	} else {
		// for cond {}
		s2 = s1
		s1 = nil
	}

	// body
	p.exprLevel = prevLevel
	body := p.parseBlockStmt()
	p.expectSemi()
	cond := p.makeExpr(s2, "condition expression")
	return &ForStmt{
		ForPos: pos,
		Init:   s1,
		Cond:   cond,
		Post:   s3,
		Body:   body,
	}
}

func (p *Parser) parseBranchStmt(tok token.Token) Stmt {
	if p.trace {
		defer untracep(tracep(p, "BranchStmt"))
	}

	pos := p.expect(tok)

	var label *Ident
	if p.token.Token == token.Ident {
		label = p.parseIdent()
	}
	p.expectSemi()
	return &BranchStmt{
		Token:    tok,
		TokenPos: pos,
		Label:    label,
	}
}

func (p *Parser) parseIfStmt() Stmt {
	if p.trace {
		defer untracep(tracep(p, "IfStmt"))
	}

	pos := p.expect(token.If)
	init, cond, starts := p.parseIfHeader()

	var body *BlockStmt
	if p.token.Token == token.Colon {
		p.next()
		expr := p.parseExpr()
		body = &BlockStmt{
			Stmts:  []Stmt{&ExprStmt{expr}},
			LBrace: expr.Pos(),
			RBrace: expr.End(),
		}
	} else if starts == token.Then {
		body = p.parseBlockStmt(BlockWrap{
			token.Then,
			[]BlockEnd{
				{token.End, true},
				{token.Else, false},
			},
		})
	} else {
		body = p.parseBlockStmt()
	}

	var elseStmt Stmt
	if p.token.Token == token.Else {
		p.next()

		switch p.token.Token {
		case token.If:
			elseStmt = p.parseIfStmt()
		case token.LBrace:
			elseStmt = p.parseBlockStmt()
			p.expectSemi()
		case token.Then:
			elseStmt = p.parseBlockStmt(BlockWrap{
				token.Then,
				[]BlockEnd{
					{token.End, true},
					{token.Else, false},
					{token.End, true},
				},
			})
		case token.Colon:
			p.next()
			expr := p.parseExpr()
			elseStmt = &BlockStmt{
				Stmts:  []Stmt{&ExprStmt{expr}},
				LBrace: expr.Pos(),
				RBrace: expr.End(),
			}
			p.expectSemi()
		default:
			p.errorExpected(p.token.Pos, "if or {")
			elseStmt = &BadStmt{From: p.token.Pos, To: p.token.Pos}
		}
	} else {
		p.expectSemi()
	}
	return &IfStmt{
		IfPos: pos,
		Init:  init,
		Cond:  cond,
		Body:  body,
		Else:  elseStmt,
	}
}

func (p *Parser) parseTryStmt() Stmt {
	if p.trace {
		defer untracep(tracep(p, "TryStmt"))
	}
	pos := p.expect(token.Try)
	body := p.parseBlockStmt(BlockWrap{
		Start: token.Then,
		Ends: []BlockEnd{
			{token.Catch, false},
			{token.Finally, false},
			{token.End, true},
		},
	})
	var catchStmt *CatchStmt
	var finallyStmt *FinallyStmt
	if p.token.Token == token.Catch {
		catchStmt = p.parseCatchStmt()
	}
	if p.token.Token == token.Finally || catchStmt == nil {
		finallyStmt = p.parseFinallyStmt()
	}
	p.expectSemi()
	return &TryStmt{
		TryPos:  pos,
		Catch:   catchStmt,
		Finally: finallyStmt,
		Body:    body,
	}
}

func (p *Parser) parseCatchStmt() *CatchStmt {
	if p.trace {
		defer untracep(tracep(p, "CatchStmt"))
	}
	pos := p.expect(token.Catch)
	var ident *Ident
	if p.token.Token == token.Ident {
		ident = p.parseIdent()
	}
	body := p.parseBlockStmt(BlockWrap{
		token.Then,
		[]BlockEnd{
			{token.Finally, false},
			{token.End, true},
		},
	})
	return &CatchStmt{
		CatchPos: pos,
		Ident:    ident,
		Body:     body,
	}
}

func (p *Parser) parseFinallyStmt() *FinallyStmt {
	if p.trace {
		defer untracep(tracep(p, "FinallyStmt"))
	}
	pos := p.expect(token.Finally)
	body := p.parseBlockStmt(BlockWrap{
		token.Then,
		[]BlockEnd{
			{token.End, true},
		},
	})
	return &FinallyStmt{
		FinallyPos: pos,
		Body:       body,
	}
}

func (p *Parser) parseThrowStmt() Stmt {
	if p.trace {
		defer untracep(tracep(p, "ThrowStmt"))
	}
	pos := p.expect(token.Throw)
	expr := p.parseExpr()
	p.expectSemi()
	return &ThrowStmt{
		ThrowPos: pos,
		Expr:     expr,
	}
}

func (p *Parser) parseBlockStmt(ends ...BlockWrap) *BlockStmt {
	if p.trace {
		defer untracep(tracep(p, "BlockStmt"))
	}

	var (
		lbrace = p.token.Pos
		start  = p.token.Token
	)

	for _, e := range ends {
		if p.token.Token == e.Start {
			p.next()
			goto parse_list
		}
	}

	p.expect(token.LBrace)

parse_list:

	list, endb := p.parseStmtList(start, ends...)
	var rbrace Pos
	if endb != nil {
		rbrace = p.token.Pos
	} else {
		switch start {
		case token.Then, token.Do:
			rbrace = p.expect(token.End)
		default:
			rbrace = p.expect(token.RBrace)
		}
	}

	return &BlockStmt{
		LBrace: lbrace,
		RBrace: rbrace,
		Stmts:  list,
	}
}

func (p *Parser) parseIfHeader() (init Stmt, cond Expr, starts token.Token) {
	if p.token.Token.IsBlockStart() {
		p.error(p.token.Pos, "missing condition in if statement")
		cond = &BadExpr{From: p.token.Pos, To: p.token.Pos}
		return
	}

	outer := p.exprLevel
	p.exprLevel = -1
	if p.token.Token == token.Semicolon {
		p.error(p.token.Pos, "missing init in if statement")
		return
	}
	init = p.parseSimpleStmt(false)

	var condStmt Stmt
	switch p.token.Token {
	case token.LBrace, token.Then, token.Colon:
		condStmt = init
		init = nil
		if p.token.Token == token.Then {
			starts = token.Then
		}
	case token.Semicolon:
		p.next()
		condStmt = p.parseSimpleStmt(false)
		if p.token.Token == token.Then {
			starts = token.Then
		}
	default:
		p.error(p.token.Pos, "missing condition in if statement")
	}

	if condStmt != nil {
		cond = p.makeExpr(condStmt, "boolean expression")
	}
	if cond == nil {
		cond = &BadExpr{From: p.token.Pos, To: p.token.Pos}
	}
	p.exprLevel = outer
	return
}

func (p *Parser) makeExpr(s Stmt, want string) Expr {
	if s == nil {
		return nil
	}

	if es, isExpr := s.(*ExprStmt); isExpr {
		return es.Expr
	}

	found := "simple statement"
	if _, isAss := s.(*AssignStmt); isAss {
		found = "assignment"
	}
	p.error(s.Pos(), fmt.Sprintf("expected %s, found %s", want, found))
	return &BadExpr{From: s.Pos(), To: p.safePos(s.End())}
}

func (p *Parser) parseReturnStmt() Stmt {
	if p.trace {
		defer untracep(tracep(p, "ReturnStmt"))
	}

	pos := p.token.Pos
	p.expect(token.Return)

	var x Expr
	if p.token.Token != token.Semicolon && p.token.Token != token.RBrace {
		lbpos := p.token.Pos
		x = p.parseExpr()
		if p.token.Token != token.Comma {
			goto done
		}
		// if the next token is a comma, treat it as multi return so put
		// expressions into a slice and replace x expression with an ArrayLit.
		elements := make([]Expr, 1, 2)
		elements[0] = x
		for p.token.Token == token.Comma {
			p.next()
			x = p.parseExpr()
			elements = append(elements, x)
		}
		x = &ArrayLit{
			Elements: elements,
			LBrack:   lbpos,
			RBrack:   x.End(),
		}
	}
done:
	p.expectSemi()
	return &ReturnStmt{
		ReturnPos: pos,
		Result:    x,
	}
}

func (p *Parser) parseSimpleStmt(forIn bool) Stmt {
	if p.trace {
		defer untracep(tracep(p, "SimpleStmt"))
	}

	x := p.parseExprList()

	switch p.token.Token {
	case token.Assign, token.Define: // assignment statement
		pos, tok := p.token.Pos, p.token.Token
		p.next()
		y := p.parseExprList()
		return &AssignStmt{
			LHS:      x,
			RHS:      y,
			Token:    tok,
			TokenPos: pos,
		}
	case token.In:
		if forIn {
			p.next()
			y := p.parseExpr()

			var key, value *Ident
			var ok bool
			switch len(x) {
			case 1:
				key = &Ident{Name: "_", NamePos: x[0].Pos()}

				value, ok = x[0].(*Ident)
				if !ok {
					p.errorExpected(x[0].Pos(), "identifier")
					value = &Ident{Name: "_", NamePos: x[0].Pos()}
				}
			case 2:
				key, ok = x[0].(*Ident)
				if !ok {
					p.errorExpected(x[0].Pos(), "identifier")
					key = &Ident{Name: "_", NamePos: x[0].Pos()}
				}
				value, ok = x[1].(*Ident)
				if !ok {
					p.errorExpected(x[1].Pos(), "identifier")
					value = &Ident{Name: "_", NamePos: x[1].Pos()}
				}
				// TODO: no more than 2 idents
			}
			return &ForInStmt{
				Key:      key,
				Value:    value,
				Iterable: y,
			}
		}
	}

	if len(x) > 1 {
		p.errorExpected(x[0].Pos(), "1 expression")
		// continue with first expression
	}

	switch p.token.Token {
	case token.Define,
		token.AddAssign, token.SubAssign, token.MulAssign, token.QuoAssign,
		token.RemAssign, token.AndAssign, token.OrAssign, token.XorAssign,
		token.ShlAssign, token.ShrAssign, token.AndNotAssign,
		token.NullichAssign, token.LOrAssign:
		pos, tok := p.token.Pos, p.token.Token
		p.next()
		y := p.parseExpr()
		return &AssignStmt{
			LHS:      []Expr{x[0]},
			RHS:      []Expr{y},
			Token:    tok,
			TokenPos: pos,
		}
	case token.Inc, token.Dec:
		// increment or decrement statement
		s := &IncDecStmt{Expr: x[0], Token: p.token.Token, TokenPos: p.token.Pos}
		p.next()
		return s
	}
	return &ExprStmt{Expr: x[0]}
}

func (p *Parser) parseExprList() (list []Expr) {
	if p.trace {
		defer untracep(tracep(p, "ExpressionList"))
	}

	list = append(list, p.parseExpr())
	for p.token.Token == token.Comma {
		p.next()
		list = append(list, p.parseExpr())
	}
	return
}

func (p *Parser) parseMapElementLit() *MapElementLit {
	if p.trace {
		defer untracep(tracep(p, "MapElementLit"))
	}

	pos := p.token.Pos
	name := "_"
	if p.token.Token == token.Ident || p.token.Token.IsKeyword() {
		name = p.token.Literal
	} else if p.token.Token == token.String {
		v, _ := strconv.Unquote(p.token.Literal)
		name = v
	} else {
		p.errorExpected(pos, "map key")
	}
	p.next()
	colonPos := p.expect(token.Colon)
	valueExpr := p.parseExpr()
	return &MapElementLit{
		Key:      name,
		KeyPos:   pos,
		ColonPos: colonPos,
		Value:    valueExpr,
	}
}

func (p *Parser) parseMapLit() *MapLit {
	if p.trace {
		defer untracep(tracep(p, "MapLit"))
	}

	lbrace := p.expect(token.LBrace)
	p.exprLevel++

	var elements []*MapElementLit
	for p.token.Token != token.RBrace && p.token.Token != token.EOF {
		elements = append(elements, p.parseMapElementLit())

		if !p.atComma("map literal", token.RBrace) {
			break
		}
		p.next()
	}

	p.exprLevel--
	rbrace := p.expect(token.RBrace)
	return &MapLit{
		LBrace:   lbrace,
		RBrace:   rbrace,
		Elements: elements,
	}
}

func (p *Parser) parseKeyValueLit() *KeyValueLit {
	if p.trace {
		defer untracep(tracep(p, "KeyValueLit"))
	}

	p.skipSpace()

	var (
		pos       = p.token.Pos
		keyExpr   = p.parsePrimitiveOperand()
		colonPos  Pos
		valueExpr Expr
	)

	p.skipSpace()

	switch p.token.Token {
	case token.Comma, token.RParen:
	default:
		colonPos = p.expect(token.Assign)
		valueExpr = p.parseExpr()
		p.skipSpace()
	}
	return &KeyValueLit{
		Key:      keyExpr,
		KeyPos:   pos,
		ColonPos: colonPos,
		Value:    valueExpr,
	}
}

func (p *Parser) parseKeyValueArrayLit(lbrace Pos) *KeyValueArrayLit {
	if p.trace {
		defer untracep(tracep(p, "parseKeyValueArrayLit"))
	}

	p.exprLevel++
	p.expect(token.Semicolon)

	var elements []*KeyValueLit
	for p.token.Token != token.RParen && p.token.Token != token.EOF {
		elements = append(elements, p.parseKeyValueLit())

		if !p.atComma("keyValueArray literal", token.RParen) {
			break
		}
		p.next()
	}

	p.exprLevel--
	rbrace := p.expect(token.RParen)
	return &KeyValueArrayLit{
		LBrace:   lbrace,
		RBrace:   rbrace,
		Elements: elements,
	}
}

func (p *Parser) expect(token token.Token) Pos {
	pos := p.token.Pos

	if p.token.Token != token {
		p.errorExpected(pos, "'"+token.String()+"'")
	}
	p.next()
	return pos
}

func (p *Parser) expectSemi() {
	switch p.token.Token {
	case token.RParen, token.RBrace, token.End:
		// semicolon is optional before a closing ')' or '}'
	case token.Comma:
		// permit a ',' instead of a ';' but complain
		p.errorExpected(p.token.Pos, "';'")
		fallthrough
	case token.Semicolon:
		p.next()
	default:
		if p.prevToken == token.End {
			return
		}
		p.errorExpected(p.token.Pos, "';'")
		p.advance(stmtStart)
	}
}

func (p *Parser) advance(to map[token.Token]bool) {
	for ; p.token.Token != token.EOF; p.next() {
		if to[p.token.Token] {
			if p.token.Pos == p.syncPos && p.syncCount < 10 {
				p.syncCount++
				return
			}
			if p.token.Pos > p.syncPos {
				p.syncPos = p.token.Pos
				p.syncCount = 0
				return
			}
		}
	}
}

func (p *Parser) error(pos Pos, msg string) {
	filePos := p.file.Position(pos)

	n := len(p.errors)
	if n > 0 && p.errors[n-1].Pos.Line == filePos.Line {
		// discard errors reported on the same line
		return
	}
	if n > 10 {
		// too many errors; terminate early
		panic(bailout{})
	}
	p.errors.Add(filePos, msg)
}

func (p *Parser) errorExpected(pos Pos, msg string) {
	msg = "expected " + msg
	if pos == p.token.Pos {
		// error happened at the current position: provide more specific
		switch {
		case p.token.Token == token.Semicolon && p.token.Literal == "\n":
			msg += ", found newline"
		case p.token.Token.IsLiteral():
			msg += ", found " + p.token.Literal
		default:
			msg += ", found '" + p.token.Token.String() + "'"
		}
	}
	p.error(pos, msg)
}

func (p *Parser) consumeComment() (comment *Comment, endline int) {
	// /*-style comments may end on a different line than where they start.
	// Scan the comment for '\n' chars and adjust endline accordingly.
	endline = p.file.Line(p.token.Pos)
	if p.token.Literal[1] == '*' {
		// don't use range here - no need to decode Unicode code points
		for i := 0; i < len(p.token.Literal); i++ {
			if p.token.Literal[i] == '\n' {
				endline++
			}
		}
	}

	comment = &Comment{Slash: p.token.Pos, Text: p.token.Literal}
	p.next0()
	return
}

func (p *Parser) consumeCommentGroup(n int) (comments *CommentGroup) {
	var list []*Comment
	endline := p.file.Line(p.token.Pos)
	for p.token.Token == token.Comment && p.file.Line(p.token.Pos) <= endline+n {
		var comment *Comment
		comment, endline = p.consumeComment()
		list = append(list, comment)
	}

	comments = &CommentGroup{List: list}
	p.comments = append(p.comments, comments)
	return
}

func (p *Parser) next0() {
	if p.trace && p.token.Pos.IsValid() {
		s := p.token.Token.String()
		switch {
		case p.token.Token.IsLiteral():
			p.printTrace(s, p.token.Literal)
		case p.token.Token.IsOperator(), p.token.Token.IsKeyword():
			p.printTrace(`"` + s + `"`)
		default:
			p.printTrace(s)
		}
	}
	p.token = p.scanner.Scan()
}

func (p *Parser) next() {
	prev := p.token.Pos
	p.prevToken = p.token.Token

next:
	p.next0()
	switch p.token.Token {
	case token.CodeBegin, token.CodeEnd:
		goto next
	case token.Comment:
		if p.file.Line(p.token.Pos) == p.file.Line(prev) {
			// line comment of prev token
			_ = p.consumeCommentGroup(0)
		}
		// consume successor comments, if any
		for p.token.Token == token.Comment {
			// lead comment of next token
			_ = p.consumeCommentGroup(1)
		}
	}
}

func (p *Parser) skipSpace() {
	for p.token.Token == token.Semicolon && p.token.Literal == "\n" {
		p.next()
	}
}

func (p *Parser) printTrace(a ...any) {
	const (
		dots = ". . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . "
		n    = len(dots)
	)

	filePos := p.file.Position(p.token.Pos)
	_, _ = fmt.Fprintf(p.traceOut, "%5d: %5d:%3d: ", p.token.Pos, filePos.Line,
		filePos.Column)
	i := 2 * p.indent
	for i > n {
		_, _ = fmt.Fprint(p.traceOut, dots)
		i -= n
	}
	_, _ = fmt.Fprint(p.traceOut, dots[0:i])
	_, _ = fmt.Fprintln(p.traceOut, a...)
}

func (p *Parser) safePos(pos Pos) Pos {
	fileBase := p.file.Base
	fileSize := p.file.Size

	if int(pos) < fileBase || int(pos) > fileBase+fileSize {
		return Pos(fileBase + fileSize)
	}
	return pos
}

func tracep(p *Parser, msg string) *Parser {
	p.printTrace(msg, "(")
	p.indent++
	return p
}

func untracep(p *Parser) {
	p.indent--
	p.printTrace(")")
}

type BlockEnd struct {
	Token token.Token
	Next  bool
}

type BlockWrap struct {
	Start token.Token
	Ends  []BlockEnd
}
