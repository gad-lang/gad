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
	"reflect"

	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/parser/utils"
	"github.com/gad-lang/gad/runehelper"
	"github.com/gad-lang/gad/token"
)

// TextFlag represents a text flag.
type TextFlag uint8

func (b *TextFlag) Set(flag TextFlag) *TextFlag    { *b = *b | flag; return b }
func (b *TextFlag) Clear(flag TextFlag) *TextFlag  { *b = *b &^ flag; return b }
func (b *TextFlag) Toggle(flag TextFlag) *TextFlag { *b = *b ^ flag; return b }
func (b TextFlag) Has(flag TextFlag) bool          { return b&flag != 0 }
func (b TextFlag) String() (s string) {
	if b.Has(TrimLeft) {
		s += "<"
	}
	if b.Has(TrimRight) {
		s += ">"
	}
	return
}

// List of scanner modes.
const (
	TrimLeft TextFlag = 1 << iota
	TrimRight
)

type ScannerInterface interface {
	Scan() (t Token)
	Mode() ScanMode
	SetMode(m ScanMode)
	SourceFile() *source.File
	Source() []byte
	ErrorHandler(h ...source.ScannerErrorHandler)
	GetMixedDelimiter() *MixedDelimiter
}

type TokenPool []*Token

func (p *TokenPool) Shift() (t *Token) {
	t = (*p)[0]
	*p = (*p)[1:]
	return
}

func (p TokenPool) Last() (t *Token) {
	return p[len(p)-1]
}

func (p TokenPool) Empty() bool {
	return len(p) == 0
}

func (p *TokenPool) Add(t ...*Token) {
	*p = append(*p, t...)
}

func (p *TokenPool) Semi() {
	*p = append(*p, &Token{Token: token.Semicolon, Literal: ";"})
}

func (s *Handlers) TokenHandler(f func(t *Token)) {
	s.TokenHandlers = append(s.TokenHandlers, f)
}

func (s *Handlers) CallTokenHandlers(t *Token) {
	if t.handled {
		return
	}
	t.handled = true
	for _, handler := range s.TokenHandlers {
		handler(t)
	}
}

type TokenHandler func(t *Token)

type TokenHandlers []TokenHandler

func (th *TokenHandlers) Remove(h TokenHandler) {
	addr := reflect.ValueOf(h).Pointer()
	for i, handler := range *th {
		if reflect.ValueOf(handler).Pointer() == addr {
			defer func() {
				*th = append((*th)[:i], (*th)[i+1:]...)
			}()
			break
		}
	}
}

type Handlers struct {
	source.NextHandlers
	ScanHandler   func(ch rune) (t Token, insertSemi, ok bool)
	TokenHandlers TokenHandlers
}

type MixedDelimiter = source.StartEndDelimiter

var DefaultMixedDelimiter = MixedDelimiter{
	Start: []rune("{%"),
	End:   []rune("%}"),
}

type ScannerOptions struct {
	Mode           ScanMode
	MixedDelimiter MixedDelimiter
}

// Scanner reads the Gad source text. It's based on ToInterface's scanner
// implementation.
type Scanner struct {
	Handlers
	source.Reader
	MixedDelimiter MixedDelimiter // the mixed delimiters

	InsertSemi         bool // insert a semicolon before next newline
	mode               ScanMode
	InCode             bool
	ToText             bool
	BraceCount         int
	BreacksCount       int
	ParenCount         int
	TokenPool          TokenPool
	SkipWhitespaceFunc func(s *Scanner)
	HandleMixed        func(textStart *int, rt func() *Token)
	EOF                *Token
}

// NewScanner creates a Scanner.
func NewScanner(
	file *source.File,
	opts *ScannerOptions,
) *Scanner {
	if opts == nil {
		opts = &ScannerOptions{}
	}

	if opts.MixedDelimiter.IsZero() {
		opts.MixedDelimiter = DefaultMixedDelimiter
	}

	s := &Scanner{
		MixedDelimiter: opts.MixedDelimiter,
		mode:           opts.Mode,
	}

	s.Reader = *source.NewFileReader(
		file,
		source.FileReaderWithData(s),
		source.FileReaderWithSkipWhitespaceFunc(func(fr *source.Reader) {
			fr.Data.(*Scanner).skipWithSpace()
		}),
	)

	s.Start()

	return s
}

func (s *Scanner) Clone() (c *Scanner) {
	clone := *s
	c = &clone
	c.Reader.Data = c
	return c
}

func (s *Scanner) skipWithSpace() {
	for s.Ch == ' ' || s.Ch == '\t' || s.Ch == '\n' && !s.InsertSemi {
		s.Next()
	}
}

func (s *Scanner) GetMixedDelimiter() *MixedDelimiter {
	return &s.MixedDelimiter
}

func (s *Scanner) List() (ret []Token) {
	var t Token
	for {
		t = s.Scan()
		if t.Token == token.EOF {
			return
		}
		ret = append(ret, t)
	}
}

func (s *Scanner) AddNextToken(n ...Token) (r *Token) {
	for _, t := range n {
		t2 := t
		r = s.AddNextTokenPtr(&t2)
	}
	return
}

func (s *Scanner) AddNextTokenPtr(n ...*Token) (r *Token) {
	var newN []*Token
	for _, t := range n {
		if t.Prev != nil {
			for _, p := range t.Prev {
				p2 := p
				newN = append(newN, &p2)
			}
		}
		t.Prev = nil
		newN = append(newN, t)
	}
	n = newN
	for i := range n {
		if n[i].Token == token.EOF {
			if l := len(s.TokenPool); l > 0 {
				if s.TokenPool[l-1].Token == token.EOF {
					if i == 0 {
						r = n[i]
					}
					n = n[:i]
					if len(n) == 0 {
						return
					}
					break
				}
			}
		}
		s.CallTokenHandlers(n[i])
	}
	s.TokenPool.Add(n...)
	return n[len(n)-1]
}

func (s *Scanner) Mode() ScanMode {
	return s.mode
}

func (s *Scanner) ModeP() *ScanMode {
	return &s.mode
}

func (s *Scanner) SetMode(m ScanMode) {
	s.mode = m
}

func (s *Scanner) Scan() (t Token) {
	if !s.TokenPool.Empty() {
		return *s.TokenPool.Shift()
	} else if s.EOF != nil {
		return *s.EOF
	}

	t = s.ScanNow()
	if t.Token == token.EOF {
		s.EOF = &t
		s.CallEOFHandlers()
		return t
	}
	s.AddNextToken(t)
	return *s.TokenPool.Shift()
}

func (s Scanner) PeekScan() (t Token) {
	return s.ScanNow()
}

func (s *Scanner) MixedCodeEnds(offsetInc int) bool {
	data := s.Src[s.Offset+offsetInc:]
	if s.MixedDelimiter.Ends(data) {
		if len(s.MixedDelimiter.End) == 1 {
			switch data[0] {
			case ']':
				if s.BreacksCount > 0 {
					return false
				}
			case '}':
				if s.BraceCount > 0 {
					return false
				}
			case ')':
				if s.ParenCount > 0 {
					return false
				}
			}
		}
		return true
	}
	return false
}

func (s *Scanner) scanConfig(pos source.Pos, skip int) (t Token) {
	s.NextC(skip)
	eol := s.NextPosOf('\n')
	s2 := s.Clone()
	s2.Src = s2.Src[:eol]
	s2.mode = 0
	s2.NextNoSpace()
	s2.TokenPool = nil
	t.Token = token.ConfigEnd
	t.Pos = s.File.FileSetPos(eol)
	t.Prev = append([]Token{{
		Token: token.ConfigStart,
		Pos:   pos,
	}}, s2.List()...)
	s.NextC(eol - s.Offset + 1)
	return
}

func (s *Scanner) ScanComment() string {
	// initial '/' already consumed; s.ch == '/' || s.ch == '*'
	offs := s.Offset - 1 // position of initial '/'

	if s.Ch == '/' {
		// -style comment
		// (the final '\n' is not considered part of the comment)
		s.Next()
		for s.Ch != '\n' && s.Ch >= 0 {
			s.Next()
		}
		goto exit
	}

	/*-style comment */
	s.Next()
	for s.Ch >= 0 {
		ch := s.Ch
		s.Next()
		if ch == '*' && s.Ch == '/' {
			s.Next()
			goto exit
		}
	}

	s.Error(offs, "comment not terminated")

exit:
	lit := s.Src[offs:s.Offset]
	return string(lit)
}

func (s *Scanner) ScanNumber(seenDecimalPoint bool) (tok token.Token, lit string) {
	var t source.NumberType
	t, lit = s.Reader.ScanNumber(seenDecimalPoint)
	switch t {
	case source.Int:
		tok = token.Int
	case source.Uint:
		tok = token.Uint
	case source.Float:
		if s.mode.Has(ScanFloatAsDecimal) {
			tok = token.Decimal
		} else {
			tok = token.Float
		}
	case source.Decimal:
		tok = token.Decimal
	}
	return
}

func (s *Scanner) ScanIdentifier() string {
	offs := s.Offset
	for runehelper.IsIdentifier(s.Ch) {
		s.Next()
	}
	return string(s.Src[offs:s.Offset])
}

func (s *Scanner) Switch2(tok0, tok1 token.Token) token.Token {
	if s.Ch == '=' {
		s.Next()
		return tok1
	}
	return tok0
}

func (s *Scanner) Switch3(
	tok0, tok1 token.Token,
	ch2 rune,
	tok2 token.Token,
) token.Token {
	if s.Ch == '=' {
		s.Next()
		return tok1
	}
	if s.Ch == ch2 {
		s.Next()
		return tok2
	}
	return tok0
}

func (s *Scanner) Switch4(
	tok0, tok1 token.Token,
	ch2 rune,
	tok2, tok3 token.Token,
) token.Token {
	if s.Ch == '=' {
		s.Next()
		return tok1
	}
	if s.Ch == ch2 {
		s.Next()
		if s.Ch == '=' {
			s.Next()
			return tok3
		}
		return tok2
	}
	return tok0
}

func (s *Scanner) ScanCodeBlock(leftText *Token) (code Token) {
	var (
		end  = s.Offset
		lit  = string(s.MixedDelimiter.Start)
		data utils.Data
	)

	s.InCode = true
	s.NextC(len(s.MixedDelimiter.Start))

	if leftText != nil {
		switch s.Ch {
		case '-':
			s.NextNoSpace()
			data.Set("remove-spaces", true)
		}
		if leftText.Literal == "" {
			leftText = nil
		}
	}

	code = Token{
		Token:   token.MixedCodeStart,
		Pos:     s.File.FileSetPos(end),
		Literal: lit,
		Data:    data,
	}

	if s.Ch == '=' {
		s.ToText = true
		s.NextNoSpace()
		code.Token = token.MixedValueStart
		code.Data.Set("eq", true)
	} else if s.mode.Has(ScanMixedExprAsValue) {
		s.ToText = true
		code.Token = token.MixedValueStart
	}

	if leftText != nil {
		code.Prev = append(code.Prev, *leftText)
	}

	return
}
