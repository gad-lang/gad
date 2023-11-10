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

// Parser parses the Tengo source files. It's based on ToInterface's parser
// implementation.
type Parser struct {
	File                    *SourceFile
	Errors                  ErrorList
	Scanner                 ScannerInterface
	Token                   Token
	PrevToken               Token
	ExprLevel               int        // < 0: in control clause, >= 0: in expression
	syncPos                 source.Pos // last sync position
	syncCount               int        // number of advance calls without progress
	Trace                   bool
	indent                  int
	mode                    Mode
	TraceOut                io.Writer
	comments                []*ast.CommentGroup
	ParseStmtHandler        func() node.Stmt
	IgnoreCodeBlockDisabled bool
	InCode                  bool
	BlockStart              token.Token
	BlockEnd                token.Token
	ScanFunc                func() Token
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
	return NewParserWithArgs(NewScanner(file, src, m), trace, mode)
}

// NewParserWithArgs creates a Parser with parser mode flags.
func NewParserWithArgs(
	scanner ScannerInterface,
	trace io.Writer,
	mode Mode,
) *Parser {
	p := &Parser{
		Scanner:    scanner,
		File:       scanner.SourceFile(),
		Trace:      trace != nil,
		TraceOut:   trace,
		mode:       mode,
		BlockStart: token.LBrace,
		BlockEnd:   token.RBrace,
	}
	p.ParseStmtHandler = p.DefaultParseStmt
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
	scanner.ErrorHandler(func(pos SourceFilePos, msg string) {
		p.Errors.Add(pos, msg)
	})
	p.Next()
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

		p.Errors.Sort()
		err = p.Errors.Err()
	}()

	if p.Trace {
		defer untracep(tracep(p, "File"))
	}

	if p.Errors.Len() > 0 {
		return nil, p.Errors.Err()
	}

	stmts, _ := p.ParseStmtList(0)
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
	colonPos := p.Expect(token.Colon)
	falseExpr := p.ParseExpr()

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

L:
	for {
		switch p.Token.Token {
		case token.NullishSelector:
			p.Next()

			switch p.Token.Token {
			case token.Ident, token.LParen:
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
			case token.Ident, token.LParen:
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

func (p *Parser) ParseCallArgs(tlparen, trparen token.Token) *node.CallArgs {
	if p.Trace {
		defer untracep(tracep(p, "CallArgs"))
	}

	lparen := p.Expect(tlparen)
	p.ExprLevel++

	var (
		args      node.CallExprArgs
		namedArgs node.CallExprNamedArgs
	)

	for p.Token.Token != trparen && p.Token.Token != token.EOF && p.Token.Token != token.Semicolon {
		if p.Token.Token == token.Ellipsis {
			elipsis := &node.EllipsisValue{Pos: p.Token.Pos}
			p.Next()
			elipsis.Value = p.ParseExpr()
			if _, ok := elipsis.Value.(*node.MapLit); ok {
				namedArgs.Ellipsis = elipsis
				goto done
			} else {
				args.Ellipsis = elipsis
			}
			goto kw
		}
		args.Values = append(args.Values, p.ParseExpr())
		switch p.Token.Token {
		case token.Assign:
			val := args.Values[len(args.Values)-1]
			args.Values = args.Values[:len(args.Values)-1]
			switch t := val.(type) {
			case *node.Ident:
				namedArgs.Names = append(namedArgs.Names, node.NamedArgExpr{Ident: t})
			case *node.StringLit:
				namedArgs.Names = append(namedArgs.Names, node.NamedArgExpr{Lit: t})
			default:
				p.ErrorExpected(val.Pos(), "string|ident")
			}
			p.Next()
			namedArgs.Values = append(namedArgs.Values, p.ParseExpr())
			goto kw
		case token.Semicolon:
			goto kw
		}
		if !p.AtComma("call argument", trparen) {
			break
		}
		p.Next()
	}

kw:
	if (p.Token.Token == token.Semicolon && p.Token.Literal == ";") ||
		(p.Token.Token == token.Comma && (len(namedArgs.Names) == 1 || args.Ellipsis != nil)) {
		p.Next()

		for {
			switch p.Token.Token {
			case token.Ellipsis:
				namedArgs.Ellipsis = &node.EllipsisValue{Pos: p.Token.Pos}
				p.Next()
				namedArgs.Ellipsis.Value = p.ParseExpr()
				goto done
			case trparen, token.EOF:
				goto done
			default:
				expr := p.ParsePrimaryExpr()
				switch t := expr.(type) {
				case *node.Ident:
					namedArgs.Names = append(namedArgs.Names, node.NamedArgExpr{Ident: t})
				case *node.StringLit:
					namedArgs.Names = append(namedArgs.Names, node.NamedArgExpr{Lit: t})
				case *node.CallExpr, *node.SelectorExpr, *node.MapLit:
					namedArgs.Ellipsis = &node.EllipsisValue{Pos: p.Token.Pos, Value: t}
					p.Expect(token.Ellipsis)
					if !p.AtComma("call argument", trparen) {
						goto done
					}
				default:
					pos := p.Token.Pos
					p.ErrorExpected(pos, "string|ident|selector|call")
					p.advance(stmtStart)
					goto done
				}

				// check if is flag
				switch p.Token.Token {
				case token.Comma, trparen:
					namedArgs.Values = append(namedArgs.Values, nil)
				// is flag
				default:
					p.Expect(token.Assign)
					namedArgs.Values = append(namedArgs.Values, p.ParseExpr())
				}

				if !p.AtComma("call argument", trparen) {
					break
				}

				p.Next()
			}
		}
	}

done:
	p.ExprLevel--
	rparen := p.Expect(trparen)
	return &node.CallArgs{
		LParen:    lparen,
		RParen:    rparen,
		Args:      args,
		NamedArgs: namedArgs,
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

func (p *Parser) ParseSelector(x node.Expr) node.Expr {
	if p.Trace {
		defer untracep(tracep(p, "Selector"))
	}

	var sel node.Expr
	if p.Token.Token == token.LParen {
		lparen := p.Token.Pos
		p.Next()
		sel = p.ParseExpr()
		rparen := p.Expect(token.RParen)
		sel = &node.ParenExpr{Expr: sel, LParen: lparen, RParen: rparen}
	} else {
		ident := p.ParseIdent()
		sel = &node.StringLit{
			Value:    ident.Name,
			ValuePos: ident.NamePos,
			Literal:  ident.Name,
		}
	}
	return &node.SelectorExpr{Expr: x, Sel: sel}
}

func (p *Parser) ParseNullishSelector(x node.Expr) node.Expr {
	if p.Trace {
		defer untracep(tracep(p, "NullishSelector"))
	}

	var sel node.Expr
	if p.Token.Token == token.LParen {
		lparen := p.Token.Pos
		p.Next()
		sel = p.ParseExpr()
		rparen := p.Expect(token.RParen)
		sel = &node.ParenExpr{Expr: sel, LParen: lparen, RParen: rparen}
	} else {
		ident := p.ParseIdent()
		sel = &node.StringLit{
			Value:    ident.Name,
			ValuePos: ident.NamePos,
			Literal:  ident.Name,
		}
	}

	return &node.NullishSelectorExpr{Expr: x, Sel: sel}
}

func (p *Parser) ParseStringLit() *node.StringLit {
	v, _ := strconv.Unquote(p.Token.Literal)
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
	case token.True:
		x := &node.BoolLit{
			Value:    true,
			ValuePos: p.Token.Pos,
			Literal:  p.Token.Literal,
		}
		p.Next()
		return x
	case token.False:
		x := &node.BoolLit{
			Value:    false,
			ValuePos: p.Token.Pos,
			Literal:  p.Token.Literal,
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
		v, _ := strconv.Unquote(p.Token.Literal)
		x := &node.StringLit{
			Value:    v,
			ValuePos: p.Token.Pos,
			Literal:  p.Token.Literal,
		}
		p.Next()
		return x
	case token.True:
		x := &node.BoolLit{
			Value:    true,
			ValuePos: p.Token.Pos,
			Literal:  p.Token.Literal,
		}
		p.Next()
		return x
	case token.False:
		x := &node.BoolLit{
			Value:    false,
			ValuePos: p.Token.Pos,
			Literal:  p.Token.Literal,
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
	case token.LParen, token.Begin:
		return p.ParseParemExpr()
	case token.LBrack: // array literal
		return p.ParseArrayLit()
	case token.LBrace: // map literal
		return p.ParseMapLit()
	case token.Func: // function literal
		return p.ParseFuncLit()
	case token.Text:
		return p.ParseTextStmt()
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

func (p *Parser) ParseParemExpr() node.Expr {
	if p.Trace {
		defer untracep(tracep(p, "ParemExpr"))
	}

	lparen := p.Token.Pos
	end := token.RParen
	switch p.Token.Token {
	case token.LParen:
	case token.Begin:
		end = token.End
	default:
		p.ErrorExpected(lparen, "'"+token.LParen.String()+"' or '"+token.Begin.String()+"'")
		return nil
	}
	p.Next()
	if end == token.End && p.Token.Token == token.Semicolon {
		p.Next()
	}
	if p.Token.Token == token.Semicolon && p.Token.Literal == ";" {
		return p.ParseKeyValueArrayLit(lparen)
	}
	p.ExprLevel++
	x := p.ParseExpr()
	p.ExprLevel--
	rparen := p.Expect(end)

	return &node.ParenExpr{
		LParen: lparen,
		Expr:   x,
		RParen: rparen,
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

func (p *Parser) ParseFuncLit() node.Expr {
	if p.Trace {
		defer untracep(tracep(p, "FuncLit"))
	}

	typ := p.ParseFuncType()
	p.ExprLevel++
	body, closure := p.ParseBody()
	p.ExprLevel--
	if closure != nil {
		return &node.ClosureLit{
			Type: typ,
			Body: closure,
		}
	}
	return &node.FuncLit{
		Type: typ,
		Body: body,
	}
}

func (p *Parser) ParseArrayLit() node.Expr {
	if p.Trace {
		defer untracep(tracep(p, "ArrayLit"))
	}

	lbrack := p.Expect(token.LBrack)
	p.ExprLevel++

	var elements []node.Expr
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

func (p *Parser) ParseFuncType() *node.FuncType {
	if p.Trace {
		defer untracep(tracep(p, "FuncType"))
	}

	var (
		pos   = p.Expect(token.Func)
		ident *node.Ident
	)

	if p.Token.Token == token.Ident {
		ident = p.ParseIdent()
	}

	params := p.ParseFuncParams()
	return &node.FuncType{
		FuncPos: pos,
		Ident:   ident,
		Params:  *params,
	}
}

func (p *Parser) ParseFuncParam(prev node.Spec) (spec node.Spec) {
	p.SkipSpace()

	var (
		pos = p.Token.Pos

		ident    *node.Ident
		variadic bool
		named    bool
		value    node.Expr
	)

	if p.Token.Token == token.Semicolon && p.Token.Literal == ";" {
		p.Next()
		p.SkipSpace()
		named = true
	} else if prev != nil {
		switch t := prev.(type) {
		case *node.NamedParamSpec:
			if t.Value == nil {
				p.Error(pos, "unexpected func param declaration")
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
		ident = p.ParseIdent()
		p.SkipSpace()
		if named {
			p.Expect(token.Assign)
			value = p.ParseExpr()
		} else if p.Token.Literal == ";" {
			goto done
		}
	} else if p.Token.Token == token.Ellipsis {
		variadic = true
		p.Next()
		ident = p.ParseIdent()
		p.SkipSpace()
	}

	if p.Token.Token == token.Comma {
		p.Next()
		p.SkipSpace()
	} else if p.Token.Token == token.Semicolon {
		if p.Token.Token == token.Assign {
			named = true
			p.Next()
			value = p.ParseExpr()
			p.SkipSpace()
			if p.Token.Token == token.Comma {
				p.Next()
				p.SkipSpace()
			}
		} else if !named {
			p.ExpectSemi()
		}
	}

	if ident == nil {
		p.Error(pos, "wrong func params declaration")
		p.ExpectSemi()
	}

	if named {
		if value == nil && !variadic {
			p.Error(pos, "wrong func params declaration")
		}
		return &node.NamedParamSpec{
			Ident: ident,
			Value: value,
		}
	}
done:
	return &node.ParamSpec{
		Ident:    ident,
		Variadic: variadic,
	}
}

func (p *Parser) ParseFuncParams() *node.FuncParams {
	if p.Trace {
		defer untracep(tracep(p, "FuncParams"))
	}

	var (
		args      node.ArgsList
		namedArgs node.NamedArgsList
		lparen    = p.Token.Pos
		spec      node.Spec
	)

	p.Next()

	for i := 0; p.Token.Token != token.RParen && p.Token.Token != token.EOF; i++ { //nolint:predeclared
		spec = p.ParseFuncParam(spec)
		if p, _ := spec.(*node.ParamSpec); p != nil {
			if p.Variadic {
				args.Var = p.Ident
			} else {
				args.Values = append(args.Values, p.Ident)
			}
		} else {
			p := spec.(*node.NamedParamSpec)
			if p.Value == nil {
				namedArgs.Var = p.Ident
			} else {
				namedArgs.Names = append(namedArgs.Names, p.Ident)
				namedArgs.Values = append(namedArgs.Values, p.Value)
			}
		}
	}

	rparen := p.Expect(token.RParen)

	return &node.FuncParams{
		LParen:    lparen,
		RParen:    rparen,
		Args:      args,
		NamedArgs: namedArgs,
	}
}

func (p *Parser) ParseBody() (b *node.BlockStmt, closure node.Expr) {
	if p.Trace {
		defer untracep(tracep(p, "Body"))
	}

	p.SkipSpace()

	if p.Token.Token == token.Assign {
		p.Next()
		p.Expect(token.Greater)

		if p.Token.Token.IsBlockStart() {
			closure = &node.BlockExpr{BlockStmt: p.ParseBlockStmt()}
		} else {
			closure = p.ParseExpr()
		}
	} else {
		b = p.ParseBlockStmt(BlockWrap{
			Start: token.Do,
			Ends: []BlockEnd{
				{token.End, true},
			},
		})
	}
	return
}

func (p *Parser) ParseStmtList(start token.Token, ends ...BlockWrap) (list []node.Stmt, end *BlockEnd) {
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
			if start != 0 {
				for _, end_ := range ends {
					if start == end_.Start {
						for _, e := range end_.Ends {
							if p.Token.Token == e.Token {
								if e.Next {
									p.Next()
								}
								end = &e
								return
							}
						}
					}
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
	case token.Text:
		return p.ParseTextStmt()
	case token.ToTextBegin:
		return p.ParseExprToTextStmt()
	case token.Var, token.Const, token.Global, token.Param:
		return &node.DeclStmt{Decl: p.ParseDecl()}
	case // simple statements
		token.Func, token.Ident, token.Int, token.Uint, token.Float,
		token.Char, token.String, token.True, token.False, token.Nil,
		token.LParen, token.LBrace, token.LBrack, token.Add, token.Sub,
		token.Mul, token.And, token.Xor, token.Not, token.Import,
		token.Callee, token.Args, token.NamedArgs,
		token.StdIn, token.StdOut, token.StdErr,
		token.Then:
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
	case token.RBrace, token.End:
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

	p.Expect(token.ConfigEnd)
	return
}

func (p *Parser) ParseExprToTextStmt() (ett *node.ExprToTextStmt) {
	if p.Trace {
		defer untracep(tracep(p, "ExprToText"))
	}
	ett = &node.ExprToTextStmt{
		StartLit: ast.Literal{Value: p.Token.Literal, Pos: p.Token.Pos},
	}
	if p.Token.Token == token.CodeBegin {
		p.Next()
		stmts, _ := p.ParseStmtList(token.CodeBegin, BlockWrap{
			Start: token.CodeBegin,
			Ends: []BlockEnd{{
				token.CodeEnd,
				true,
			}},
		})
		if len(stmts) == 1 {
			switch t := stmts[0].(type) {
			case *node.ExprStmt:
				return node.NewExprToTextStmt(t.Expr)
			}
		}
		return node.NewExprToTextStmt(&node.StmtsExpr{Stmts: stmts})
	} else {
		p.Next()
		ett.Expr = p.ParseExpr()
		if p.Token.Token == token.ToTextEnd {
			ett.EndLit = ast.Literal{Value: p.Token.Literal, Pos: p.Token.Pos}
			p.Next()
		} else {
			p.Expect(token.ToTextEnd)
		}
	}
	return
}

func (p *Parser) ParseTextStmt() (t *node.TextStmt) {
	if p.Trace {
		defer untracep(tracep(p, "TextStmt"))
	}
	t = &node.TextStmt{Literal: p.Token.Literal, TextPos: p.Token.Pos, Data: p.Token.Data}
	p.Next()
	for p.Token.Token == token.Text {
		t.Literal += p.Token.Literal
		p.Next()
	}
	return
}

func (p *Parser) ParseDecl() node.Decl {
	if p.Trace {
		defer untracep(tracep(p, "DeclStmt"))
	}
	switch p.Token.Token {
	case token.Global, token.Param:
		return p.ParseGenDecl(p.Token.Token, p.ParseParamSpec)
	case token.Var, token.Const:
		return p.ParseGenDecl(p.Token.Token, p.ParseValueSpec)
	default:
		p.Error(p.Token.Pos, "only \"param, global, var\" declarations supported")
		return &node.BadDecl{From: p.Token.Pos, To: p.Token.Pos}
	}
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

		ident    *node.Ident
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
		ident = p.ParseIdent()
	} else if keyword == token.Param && p.Token.Token == token.Ellipsis {
		variadic = true
		p.Next()
		ident = p.ParseIdent()
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

func (p *Parser) ParseForStmt() node.Stmt {
	if p.Trace {
		defer untracep(tracep(p, "ForStmt"))
	}

	pos := p.Expect(token.For)

	// for {}
	if p.Token.Token.IsBlockStart() {
		body := p.ParseBlockStmt(BlockWrap{token.Do, []BlockEnd{{token.End, true}}})
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
		forInStmt.Body = p.ParseBlockStmt(BlockWrap{token.Do, []BlockEnd{
			{token.End, true},
			{token.Else, false},
		}})
		if p.Token.Token == token.Else {
			lbrace := p.Token.Pos
			p.Next()
			p.SkipSpace()

			switch p.Token.Token {
			case token.End:
				forInStmt.Else = &node.BlockStmt{LBrace: lbrace, RBrace: p.Token.Pos}
				p.Next()
				p.ExpectSemi()
			case token.LBrace:
				forInStmt.Else = p.ParseBlockStmt()
				p.ExpectSemi()
			case token.Then:
				forInStmt.Else = p.ParseBlockStmt(BlockWrap{
					token.Then,
					[]BlockEnd{
						{token.End, true},
					},
				})
			case token.Colon:
				p.Next()
				expr := p.ParseExpr()
				forInStmt.Else = &node.BlockStmt{
					Stmts:  []node.Stmt{&node.ExprStmt{Expr: expr}},
					LBrace: expr.Pos(),
					RBrace: expr.End(),
				}
				p.ExpectSemi()
			case token.Semicolon:
				p.ExpectSemi()
			default:
				forInStmt.Else = &node.BlockStmt{LBrace: lbrace, RBrace: p.Token.Pos}
				if stmt := p.ParseSimpleStmt(false); stmt != nil {
					forInStmt.Else.Stmts = []node.Stmt{stmt}
				}
				p.ExpectSemi()
			}
		} else {
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
	init, cond, starts := p.ParseIfHeader()

	var body *node.BlockStmt
	if p.Token.Token == token.Colon {
		p.Next()
		expr := p.ParseExpr()
		body = &node.BlockStmt{
			Stmts:  []node.Stmt{&node.ExprStmt{Expr: expr}},
			LBrace: expr.Pos(),
			RBrace: expr.End(),
		}
	} else if starts == token.Then {
		body = p.ParseBlockStmt(BlockWrap{
			token.Then,
			[]BlockEnd{
				{token.End, true},
				{token.Else, false},
			},
		})
	} else {
		body = p.ParseBlockStmt()
	}

	var elseStmt node.Stmt
	if p.Token.Token == token.Else {
		p.Next()
		p.SkipSpace()

		switch p.Token.Token {
		case token.If:
			elseStmt = p.ParseIfStmt()
		case token.LBrace, p.BlockStart:
			elseStmt = p.ParseBlockStmt()
			p.ExpectSemi()
		case token.Then:
			elseStmt = p.ParseBlockStmt(BlockWrap{
				token.Then,
				[]BlockEnd{
					{token.End, true},
					{token.Else, false},
					{token.End, true},
				},
			})
		case token.Colon:
			p.Next()
			expr := p.ParseExpr()
			elseStmt = &node.BlockStmt{
				Stmts:  []node.Stmt{&node.ExprStmt{Expr: expr}},
				LBrace: expr.Pos(),
				RBrace: expr.End(),
			}
			p.ExpectSemi()
		default:
			b := &node.BlockStmt{LBrace: p.Token.Pos, RBrace: p.Token.Pos}
			if stmt := p.ParseSimpleStmt(false); stmt != nil {
				b.RBrace = p.Token.Pos
				b.Stmts = []node.Stmt{stmt}
			}
			p.ExpectSemi()
			elseStmt = b
		}
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
	body := p.ParseBlockStmt(BlockWrap{
		Start: token.Then,
		Ends: []BlockEnd{
			{token.Catch, false},
			{token.Finally, false},
			{token.End, true},
		},
	})
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
	body := p.ParseBlockStmt(BlockWrap{
		token.Then,
		[]BlockEnd{
			{token.Finally, false},
			{token.End, true},
		},
	})
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
	body := p.ParseBlockStmt(BlockWrap{
		token.Then,
		[]BlockEnd{
			{token.End, true},
		},
	})
	return &node.FinallyStmt{
		FinallyPos: pos,
		Body:       body,
	}
}

func (p *Parser) ParseThrowStmt() node.Stmt {
	if p.Trace {
		defer untracep(tracep(p, "ThrowStmt"))
	}
	pos := p.Expect(token.Throw)
	expr := p.ParseExpr()
	p.ExpectSemi()
	return &node.ThrowStmt{
		ThrowPos: pos,
		Expr:     expr,
	}
}

func (p *Parser) ParseBlockStmt(ends ...BlockWrap) *node.BlockStmt {
	if p.Trace {
		defer untracep(tracep(p, "BlockStmt"))
	}

	var (
		lbrace = p.Token.Pos
		start  = p.Token.Token
	)

	if p.BlockStart != 0 {
		ends = append(ends, BlockWrap{
			Start: p.BlockStart,
			Ends: []BlockEnd{{
				p.BlockEnd,
				true,
			}},
		})
	}

	for _, e := range ends {
		if p.Token.Token == e.Start {
			p.Next()
			goto parse_list
		}
	}

	p.Expect(token.LBrace)

parse_list:

	list, endb := p.ParseStmtList(start, ends...)
	var rbrace source.Pos
	if endb != nil {
		rbrace = p.Token.Pos
	} else {
		switch start {
		case token.Then, token.Do:
			if p.Token.Token == token.EOF && p.PrevToken.Token == token.End {
				rbrace = p.PrevToken.Pos
			} else {
				rbrace = p.Expect(token.End)
			}
		default:
			rbrace = p.Expect(token.RBrace)
		}
	}

	return &node.BlockStmt{
		LBrace: lbrace,
		RBrace: rbrace,
		Stmts:  list,
	}
}

func (p *Parser) ParseIfHeader() (init node.Stmt, cond node.Expr, starts token.Token) {
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
	case token.LBrace, token.Then, token.Colon, p.BlockStart:
		condStmt = init
		init = nil
		if p.Token.Token == token.Then {
			starts = token.Then
		}
	case token.Semicolon:
		p.Next()
		condStmt = p.ParseSimpleStmt(false)
		if p.Token.Token == token.Then {
			starts = token.Then
		}
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

func (p *Parser) ParseReturnStmt() node.Stmt {
	if p.Trace {
		defer untracep(tracep(p, "ReturnStmt"))
	}

	pos := p.Token.Pos
	p.Expect(token.Return)

	var x node.Expr
	if p.Token.Token != token.Semicolon && p.Token.Token != token.RBrace {
		lbpos := p.Token.Pos
		x = p.ParseExpr()
		if p.Token.Token != token.Comma {
			goto done
		}
		// if the next token is a comma, treat it as multi return so put
		// expressions into a slice and replace x expression with an ArrayLit.
		elements := make([]node.Expr, 1, 2)
		elements[0] = x
		for p.Token.Token == token.Comma {
			p.Next()
			x = p.ParseExpr()
			elements = append(elements, x)
		}
		x = &node.ArrayLit{
			Elements: elements,
			LBrack:   lbpos,
			RBrack:   x.End(),
		}
	}
done:
	p.ExpectSemi()
	return &node.ReturnStmt{
		ReturnPos: pos,
		Result:    x,
	}
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

func (p *Parser) ParseMapElementLit() *node.MapElementLit {
	if p.Trace {
		defer untracep(tracep(p, "MapElementLit"))
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
	return &node.MapElementLit{
		Key:      name,
		KeyPos:   pos,
		ColonPos: colonPos,
		Value:    valueExpr,
	}
}

func (p *Parser) ParseMapLit() *node.MapLit {
	if p.Trace {
		defer untracep(tracep(p, "MapLit"))
	}

	lbrace := p.Expect(token.LBrace)
	p.ExprLevel++

	var elements []*node.MapElementLit
	for p.Token.Token != token.RBrace && p.Token.Token != token.EOF {
		elements = append(elements, p.ParseMapElementLit())

		if !p.AtComma("map literal", token.RBrace) {
			break
		}
		p.Next()
	}

	p.ExprLevel--
	rbrace := p.Expect(token.RBrace)
	return &node.MapLit{
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
	pos := p.Token.Pos

	if p.Token.Token != token {
		p.ErrorExpected(pos, "'"+token.String()+"'")
	}
	p.Next()
	return pos
}

func (p *Parser) ExpectSemi() {
	switch p.Token.Token {
	case token.RParen, token.RBrace, token.Else, token.CodeEnd:
		// semicolon is optional before a closing ')' or '}'
	case token.Comma:
		// permit a ',' instead of a ';' but complain
		p.ErrorExpected(p.Token.Pos, "';'")
		fallthrough
	case token.End:
		p.Next()
		if p.Token.Token == token.Semicolon {
			p.Next()
		}
	case token.Semicolon:
		p.Next()
	default:
		switch p.PrevToken.Token {
		case token.Else, token.End, p.BlockEnd:
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
	case token.CodeBegin:
		p.InCode = true
		if p.IgnoreCodeBlockDisabled {
			return
		}
		goto next
	case token.CodeEnd:
		p.InCode = false
		if p.IgnoreCodeBlockDisabled {
			return
		}
		goto next
	case token.Text:
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
