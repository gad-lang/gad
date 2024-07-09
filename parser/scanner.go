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
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/parser/utils"
	"github.com/gad-lang/gad/runehelper"
	"github.com/gad-lang/gad/token"
)

// BOM byte order mark
const BOM = 0xFEFF

// ScanMode represents a scanner mode.
type ScanMode uint8

func (b *ScanMode) Set(flag ScanMode) *ScanMode    { *b = *b | flag; return b }
func (b *ScanMode) Clear(flag ScanMode) *ScanMode  { *b = *b &^ flag; return b }
func (b *ScanMode) Toggle(flag ScanMode) *ScanMode { *b = *b ^ flag; return b }
func (b ScanMode) Has(flag ScanMode) bool          { return b&flag != 0 }

// List of scanner modes.
const (
	ScanComments ScanMode = 1 << iota
	DontInsertSemis
	Mixed
	ConfigDisabled
	MixedExprAsValue
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

// ScannerErrorHandler is an error handler for the scanner.
type ScannerErrorHandler func(pos SourceFilePos, msg string)

type ScannerInterface interface {
	Scan() (t Token)
	Mode() ScanMode
	SetMode(m ScanMode)
	SourceFile() *SourceFile
	Source() []byte
	ErrorHandler(h ...ScannerErrorHandler)
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

type NextHandlers struct {
	LineEndHandlers     []func()
	PostLineEndHandlers []func()
	EOFHandlers         []func(s *Scanner)
}

func (h *NextHandlers) LineEndHandler(f func()) {
	h.LineEndHandlers = append(h.LineEndHandlers, f)
}

func (h *NextHandlers) CallLineEndHandlers() {
	for _, handler := range h.LineEndHandlers {
		handler()
	}
	h.LineEndHandlers = nil
}

func (h *NextHandlers) PostLineEndHandler(f func()) {
	h.PostLineEndHandlers = append(h.PostLineEndHandlers, f)
}

func (h *NextHandlers) CallPostLineEndHandlers() {
	for _, handler := range h.PostLineEndHandlers {
		handler()
	}
	h.PostLineEndHandlers = nil
}

func (h *NextHandlers) EOFHandler(f func(*Scanner)) {
	h.EOFHandlers = append(h.EOFHandlers, f)
}

func (h *NextHandlers) CallEOFHandlers(s *Scanner) {
	handlers := h.EOFHandlers
	h.EOFHandlers = nil

	for _, handler := range handlers {
		handler(s)
	}
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
	NextHandlers
	ScanHandler   func(ch rune) (t Token, insertSemi, ok bool)
	TokenHandlers TokenHandlers
}

type MixedDelimiter = source.StartEndDelimiter

var DefaultMixedDelimiter = MixedDelimiter{
	Start: []rune("#{"),
	End:   []rune("}"),
}

// Scanner reads the Gad source text. It's based on ToInterface's scanner
// implementation.
type Scanner struct {
	Handlers

	File               *SourceFile           // source file handle
	Src                []byte                // source
	MixedDelimiter     *MixedDelimiter       // the mixed delimiters
	Ch                 rune                  // current character
	Offset             int                   // character offset
	ReadOffset         int                   // reading offset (position after current character)
	lineOffset         int                   // current line offset
	InsertSemi         bool                  // insert a semicolon before next newline
	errorHandler       []ScannerErrorHandler // error reporting; or nil
	errorCount         int                   // number of errors encountered
	mode               ScanMode
	InCode             bool
	ToText             bool
	BraceCount         int
	BreacksCount       int
	ParenCount         int
	TokenPool          TokenPool
	TextTrimLeft       bool
	SkipWhitespaceFunc func(s *Scanner)
	NewLineEscape      func() bool
	NewLineEscaped     bool
	HandleMixed        func(textStart *int, rt func() *Token)
	EOF                *Token
}

type ScannerOptions struct {
	Mode           ScanMode
	MixedDelimiter *MixedDelimiter
}

// NewScanner creates a Scanner.
func NewScanner(
	file *SourceFile,
	src []byte,
	opts *ScannerOptions,
) *Scanner {
	if file.Size != len(src) {
		panic(fmt.Sprintf("file size (%d) does not match Src len (%d)",
			file.Size, len(src)))
	}

	isSpace := func(r rune) bool {
		switch r {
		case ' ', '\t', '\n', '\r':
			return true
		default:
			return false
		}
	}

	src = bytes.TrimRightFunc(src, isSpace)

	if opts == nil {
		opts = &ScannerOptions{}
	}

	if opts.MixedDelimiter == nil {
		opts.MixedDelimiter = &DefaultMixedDelimiter
	}

	last := len(src) - 1
	if pos := bytes.IndexByte(src, '\r'); pos >= 0 {
		// if line sep is only CR, replaces to EOL
		if pos < last && src[pos] != '\n' {
			for i, b := range src {
				if b == '\r' && i < last && src[i+1] != '\n' {
					src[i] = '\n'
				}
			}
		}
	}

	s := &Scanner{
		File:           file,
		Src:            src,
		MixedDelimiter: opts.MixedDelimiter,
		Ch:             ' ',
		mode:           opts.Mode,
	}

	s.SkipWhitespaceFunc = func(s *Scanner) {
		for s.Ch == ' ' || s.Ch == '\t' || s.Ch == '\n' && !s.InsertSemi {
			s.Next()
		}
	}

	s.Next()
	if s.Ch == BOM {
		s.Next() // ignore BOM at file beginning
	}

	return s
}

func (s *Scanner) GetMixedDelimiter() *MixedDelimiter {
	return s.MixedDelimiter
}

func (s *Scanner) SkipWhitespace() {
	s.SkipWhitespaceFunc(s)
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

func (s *Scanner) ErrorHandler(h ...ScannerErrorHandler) {
	s.errorHandler = append(s.errorHandler, h...)
}

func (s *Scanner) SourceFile() *SourceFile {
	return s.File
}

func (s *Scanner) Source() []byte {
	return s.Src
}

// ErrorCount returns the number of errors.
func (s *Scanner) ErrorCount() int {
	return s.errorCount
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
		s.CallEOFHandlers(s)
		return t
	}
	s.AddNextToken(t)
	return *s.TokenPool.Shift()
}

func (s Scanner) PeekScan() (t Token) {
	return s.ScanNow()
}

// ScanNow returns a token, token literal and its position.
func (s *Scanner) ScanNow() (t Token) {
	t.Pos = s.File.FileSetPos(s.Offset)

	if s.Ch == -1 {
		if s.InsertSemi {
			s.InsertSemi = false // EOF consumed
			t.Literal = "\n"
			t.Token = token.Semicolon
			return t
		}
		return Token{Token: token.EOF, Pos: t.Pos}
	}

	if s.mode.Has(Mixed) && s.Ch != -1 {
		start := s.Offset
		if s.InCode {
			s.SkipWhitespace()
			switch s.Ch {
			case '\'', '"', '`':
				// ignore quotes
				goto do
			}
			if s.MixedDelimiter.Ends(s.Src[s.Offset:]) {
				s.InCode = false
				t.Token = token.Semicolon
				t.Literal = "\n" // read first end byte
				t.Pos = s.File.FileSetPos(s.Offset)
				s.Next()
				s.NextC(len(s.MixedDelimiter.End) - 1)

				if s.ToText {
					t.Token = token.MixedValueEnd
					s.ToText = false
					t.Literal = string(s.MixedDelimiter.End)
					if s.TextTrimLeft {
						t.Set("trim_left_space", s.TextTrimLeft)
					}
				} else {
					next := Token{Token: token.MixedCodeEnd, Literal: string(s.MixedDelimiter.End), Pos: t.Pos}
					if s.TextTrimLeft {
						next.Set("trim_left_space", s.TextTrimLeft)
					}
					if !s.mode.Has(DontInsertSemis) {
						s.AddNextToken(t)
						t = next
					} else {
						return next
					}
				}
				return
			}
		} else {
			readText := func() {
				t.Token = token.MixedText
				t.Pos = s.File.FileSetPos(start)

				if s.Offset > start {
					t.Literal = string(s.Src[start:s.Offset])
					if s.TextTrimLeft {
						t.Literal = TrimSpace(true, false, t.Literal)
					}
				}
				s.TextTrimLeft = false
			}
			for {
				var scape bool
				switch int(s.Ch) {
				case '\\':
					if scape {
						scape = false
					}
				case -1:
					readText()
					return t
				case int(s.MixedDelimiter.Start[0]):
					if !scape {
						if s.MixedDelimiter.Starts(s.Ch, s.Src[s.ReadOffset:]) {
							readText()
							return s.ScanCodeBlock(&t)
						}

						if !s.mode.Has(ConfigDisabled) {
							// at line start
							if s.Offset == 0 || s.Src[s.Offset-1] == '\n' {
								if l := s.PeekNoSingleSpaceEq("gad:", 0); l > 0 {
									return s.scanConfig(t.Pos, l+1)
								}
							}
						}

						s.Next()
						continue
					}
				}
				if s.HandleMixed != nil {
					s.HandleMixed(&start, func() *Token {
						readText()
						return &t
					})
					if s.Ch == -1 {
						s.Ch = -1
						return
					}
				}
				s.Next()
			}
		}
	}

do:
	s.SkipWhitespace()
	t.Pos = s.File.FileSetPos(s.Offset)

	insertSemi := false

	// determine token value
	switch ch := s.Ch; {
	case runehelper.IsIdentifierLetter(ch):
		t.Literal = s.ScanIdentifier()
		t.Token = token.Lookup(t.Literal)
		switch t.Literal {
		case "do", "then", "begin":
			t.Token = token.LBrace
		case "done", "end":
			t.Token = token.RBrace
		default:
			switch t.Token {
			case token.Ident, token.Break, token.Continue, token.Return,
				token.True, token.False, token.Yes, token.No, token.Nil,
				token.Callee, token.Args, token.NamedArgs,
				token.StdIn, token.StdOut, token.StdErr:
				insertSemi = true
			}
		}
	case '0' <= ch && ch <= '9':
		insertSemi = true
		t.Token, t.Literal = s.ScanNumber(false)
	default:
		s.Next() // always make progress

		switch ch {
		case -1: // EOF
			if s.InsertSemi {
				s.InsertSemi = false // EOF consumed
				t.Literal = "\n"
				t.Token = token.Semicolon
				return
			}
			t.Token = token.EOF
			s.CallEOFHandlers(s)
		case '\n':
			// we only reach here if s.InsertSemi was set in the first place
			s.InsertSemi = false // newline consumed
			t.Literal = "\n"
			t.Token = token.Semicolon
			return
		case '"':
			insertSemi = true
			t.Token = token.String
			t.Literal = s.ScanString()
		case '\'':
			insertSemi = true
			t.Token = token.Char
			t.Literal = s.ScanRune()
		case '`':
			insertSemi = true
			t.Token = token.RawString
			var ishd bool
			if t.Literal, ishd = s.ScanRawString(); ishd {
				t.Token = token.RawHeredoc
			}
		case ':':
			t.Token = s.Switch2(token.Colon, token.Define)
		case '.':
			if s.Ch == '|' {
				s.Next()
				t.Token = token.Pipe
			} else if '0' <= s.Ch && s.Ch <= '9' {
				insertSemi = true
				t.Token, t.Literal = s.ScanNumber(true)
			} else {
				t.Token = token.Period
			}
		case ',':
			t.Token = token.Comma
		case '~':
			t.Token = token.Tilde
			if s.Ch == '~' {
				s.Next()
				t.Token = token.DoubleTilde
				if s.Ch == '~' {
					s.Next()
					t.Token = token.TripleTilde
				}
			}
		case '?':
			switch s.Ch {
			case '.':
				s.Next()
				t.Token = token.NullishSelector
			case '?':
				if s.Peek() == '=' {
					s.Next()
					s.Next()
					t.Token = token.NullichAssign
				} else {
					s.Next()
					t.Token = token.NullichCoalesce
				}
			default:
				t.Token = token.Question
			}
		case ';':
			t.Token = token.Semicolon
			t.Literal = ";"
		case '(':
			t.Token = token.LParen
			s.ParenCount++
		case ')':
			insertSemi = true
			t.Token = token.RParen
			s.ParenCount--
		case '[':
			t.Token = token.LBrack
			s.BreacksCount++
		case ']':
			insertSemi = true
			t.Token = token.RBrack
			s.BreacksCount--
		case '{':
			t.Token = token.LBrace
			s.BraceCount++
		case '}':
			insertSemi = true
			t.Token = token.RBrace
			s.BraceCount--
		case '+':
			t.Token = s.Switch3(token.Add, token.AddAssign, '+', token.Inc)
			if t.Token == token.Inc {
				insertSemi = true
			}
		case '-':
			if s.Ch == '}' {
				s.TextTrimLeft = true
				goto do
			}

			t.Token = s.Switch3(token.Sub, token.SubAssign, '-', token.Dec)
			if t.Token == token.Dec {
				insertSemi = true
			}
		case '*':
			t.Token = s.Switch2(token.Mul, token.MulAssign)
		case '/':
			if s.Ch == '/' || s.Ch == '*' {
				// comment
				if s.InsertSemi && s.FindLineEnd() {
					// reset position to the beginning of the comment
					s.Ch = '/'
					s.Offset = s.File.Offset(t.Pos)
					s.ReadOffset = s.Offset + 1
					s.InsertSemi = false // newline consumed
					t.Literal = "\n"
					t.Token = token.Semicolon
					return
				}
				comment := s.ScanComment()
				if !s.mode.Has(ScanComments) {
					// skip comment
					s.InsertSemi = false // newline consumed
					return s.Scan()
				}
				t.Token = token.Comment
				t.Literal = comment
			} else {
				t.Token = s.Switch2(token.Quo, token.QuoAssign)
			}
		case '%':
			t.Token = s.Switch2(token.Rem, token.RemAssign)
		case '^':
			t.Token = s.Switch2(token.Xor, token.XorAssign)
		case '<':
			t.Token = s.Switch4(token.Less, token.LessEq, '<',
				token.Shl, token.ShlAssign)
		case '>':
			t.Token = s.Switch4(token.Greater, token.GreaterEq, '>',
				token.Shr, token.ShrAssign)
		case '=':
			t.Token = s.Switch3(token.Assign, token.Equal, '>', token.Lambda)
		case '!':
			t.Token = s.Switch2(token.Not, token.NotEqual)
		case '&':
			if s.Ch == '^' {
				s.Next()
				t.Token = s.Switch2(token.AndNot, token.AndNotAssign)
			} else {
				t.Token = s.Switch3(token.And, token.AndAssign, '&', token.LAnd)
			}
		case '|':
			if s.Ch == '=' {
				s.Next()
				t.Token = token.OrAssign
			} else if s.Ch == '|' {
				if s.Peek() == '=' {
					s.Next()
					s.Next()
					t.Token = token.LOrAssign
				} else {
					s.Next()
					t.Token = token.LOr
				}
			} else {
				t.Token = token.Or
			}
		case '#':
			if !s.mode.Has(ConfigDisabled) {
				// at line start
				if s.Offset == 1 || s.Src[s.Offset-2] == '\n' {
					if l := s.PeekNoSingleSpaceEq("gad:", 0); l > 0 {
						return s.scanConfig(t.Pos, l+1)
					}
				}
			}

			t.Token = token.Template
		default:
			// next reports unexpected BOMs - don't repeat
			if ch != BOM {
				if s.ScanHandler != nil {
					var ok bool
					if t, insertSemi, ok = s.ScanHandler(ch); ok {
						goto done
					}
				}
				s.Error(s.File.Offset(t.Pos),
					fmt.Sprintf("illegal character %#U", ch))
			}
			insertSemi = s.InsertSemi // preserve InsertSemi info
			t.Token = token.Illegal
			t.Literal = string(ch)
		}
	}
done:
	if !s.mode.Has(DontInsertSemis) {
		s.InsertSemi = insertSemi
	}
	return
}

func (s *Scanner) NextC(count int) {
	for i := 0; i < count; i++ {
		s.Next()
	}
}

func (s *Scanner) Skip(str string) {
	for _, r := range str {
		if s.Ch != r {
			break
		}
		s.Next()
	}
}

func (s *Scanner) NextTo(v string) {
	for _, r := range v {
		s.Expect(r, "next to: "+strconv.Quote(v))
		s.Next()
	}
}

func (s *Scanner) Next() {
	var (
		newLineEscape bool
		r             rune
		w             int
	)
next:
	if s.ReadOffset < len(s.Src) {
		s.Offset = s.ReadOffset

		if s.Ch == '\n' {
			s.lineOffset = s.Offset
			s.File.AddLine(s.Offset)
			if s.NewLineEscaped {
				s.NewLineEscaped = false
			} else {
				s.CallLineEndHandlers()
			}

			defer s.CallPostLineEndHandlers()
		}

		r, w = rune(s.Src[s.ReadOffset]), 1

		for r == '\r' {
			s.ReadOffset++
			r = rune(s.Src[s.ReadOffset])
		}

		switch {
		case r == 0:
			s.Error(s.Offset, "illegal character NUL")
		case r >= utf8.RuneSelf:
			// not ASCII
			r, w = utf8.DecodeRune(s.Src[s.ReadOffset:])
			if r == utf8.RuneError && w == 1 {
				s.Error(s.Offset, "illegal UTF-8 encoding")
			} else if r == BOM && s.Offset > 0 {
				s.Error(s.Offset, "illegal byte order mark")
			}
		}
		s.ReadOffset += w
		s.Ch = r

		if s.Ch == '\\' && s.Peek() == '\n' {
			newLineEscape = s.NewLineEscape != nil && s.NewLineEscape()
			if newLineEscape {
				goto next
			}
		} else if s.Ch == '\n' {
			s.NewLineEscaped = newLineEscape
			if newLineEscape {
				s.SkipWhitespace()
			}
		}
	} else {
		s.Offset = len(s.Src)
		if s.Ch == '\n' {
			s.lineOffset = s.Offset
			s.File.AddLine(s.Offset)
			s.CallLineEndHandlers()
		}
		s.Ch = -1 // EOF
	}
}

func (s *Scanner) PeekAtEndLine() (start, end int) {
	start = s.Offset
	end = s.Offset
	for end < len(s.Src) {
		switch s.Src[end] {
		case '\n':
			if s.Src[end-1] != '\\' {
				return
			}
		}
		end++
	}
	return
}

func (s *Scanner) NextNoSpace() {
	s.Next()
	s.SkipWhitespace()
}

func (s *Scanner) Peek() byte {
	if s.ReadOffset < len(s.Src) {
		return s.Src[s.ReadOffset]
	}
	return 0
}

func (s *Scanner) NextPosOf(b byte) (end int) {
	end = s.Offset + 1

	var escape bool
	for end < len(s.Src) {
		switch s.Src[end] {
		case '\\':
			escape = !escape
		case b:
			if !escape {
				return
			}
		}
		end++
	}
	return end
}

func (s *Scanner) ReadAt(b rune) []byte {
	var (
		start = s.Offset
		end   = s.Offset
	)

	var escape bool
	for end < len(s.Src) {
		if s.Ch == -1 {
			return nil
		}

		if s.Ch == '\\' {
			escape = !escape
		}
		if s.Ch == b && !escape {
			break
		}
		s.Next()
		end++
	}
	return s.Src[start:end]
}

func (s *Scanner) ReadCount(q int) []byte {
	var (
		start = s.Offset
		end   = s.Offset
	)

	for i := 0; i < q; i++ {
		end++
		s.Next()
	}
	return s.Src[start:end]
}

func (s *Scanner) ReadWhen(b rune) []byte {
	var (
		start = s.Offset
		end   = s.Offset
	)

	for end < len(s.Src) {
		if s.Ch == -1 {
			return nil
		}

		if s.Ch != b {
			break
		}
		s.Next()
		end++
	}
	return s.Src[start:end]
}

func (s *Scanner) PeekNoSpace() byte {
	offs := s.ReadOffset
	for offs < len(s.Src) {
		switch s.Src[offs] {
		case ' ', '\n', '\t':
			offs++
		default:
			return s.Src[offs]
		}
	}
	return 0
}

func (s *Scanner) PeekInlineNoSpace() byte {
	offs := s.ReadOffset
	for offs < len(s.Src) {
		switch s.Src[offs] {
		case ' ', '\t':
			offs++
		default:
			return s.Src[offs]
		}
	}
	return 0
}

func (s *Scanner) IndexOfInlineNoSpace() int {
	var offs int
	for offs < len(s.Src) {
		switch s.Src[offs] {
		case ' ', '\t':
			offs++
		default:
			return offs
		}
	}
	return 0
}

func (s *Scanner) PeekNoSpaceN(n int) []byte {
	off := s.ReadOffset
	for off < len(s.Src) {
		switch s.Src[off] {
		case ' ', '\n', '\t':
			off++
		default:
			part := s.Src[off:]
			if len(part) >= n {
				return part[:n]
			}
			return nil
		}
	}
	return nil
}

func (s *Scanner) PeekN(n int) []byte {
	if (s.ReadOffset + n) <= len(s.Src) {
		return s.Src[s.ReadOffset : s.ReadOffset+n]
	}
	return nil
}

func (s *Scanner) PeekEq(str string) bool {
	b := s.PeekN(len(str))
	if b != nil {
		return string(b) == str
	}
	return true
}

func (s *Scanner) PeekNoSpaceEq(to string, skip int) bool {
	off := s.ReadOffset + skip
	for off < len(s.Src) {
		switch s.Src[off] {
		case ' ', '\n', '\t':
			off++
		default:
			n := len(to)
			if (off + n) <= len(s.Src) {
				b := s.Src[off : off+n]
				return to == string(b)
			}
			return false
		}
	}

	return false
}

func (s *Scanner) PeekNoSingleSpaceEq(to string, skip int) (length int) {
	off := s.ReadOffset + skip
	for off < len(s.Src) {
		switch s.Src[off] {
		case ' ', '\t':
			off++
		default:
			n := len(to)
			if (off + n) <= len(s.Src) {
				b := s.Src[off : off+n]
				if to == string(b) {
					return off + n - s.ReadOffset
				}
				return 0
			}
			return 0
		}
	}

	return 0
}

func (s *Scanner) PeekIdentEq(to string, skip int) bool {
	off := s.ReadOffset + skip
	for off < len(s.Src) {
		switch s.Src[off] {
		case ' ', '\t':
			off++
		default:
			n := len(to)
			if (off + n) <= len(s.Src) {
				b := s.Src[off : off+n]
				return to == string(b)
			}
			return false
		}
	}

	return false
}

func (s *Scanner) Error(offset int, msg string) {
	pos := s.File.Position(s.File.FileSetPos(offset))
	for _, h := range s.errorHandler {
		h(pos, msg)
	}
	s.errorCount++
}

func (s *Scanner) Expect(ch rune, msg string) bool {
	if s.Ch != ch {
		s.ExpectError(msg + fmt.Sprintf(", but got %s", string(s.Ch)))
		s.Ch = -1
		return false
	}
	return true
}

func (s *Scanner) ExpectError(msg string) {
	s.Error(s.Offset, "Expect: "+msg)
}

func (s *Scanner) scanConfig(pos source.Pos, skip int) (t Token) {
	s.NextC(skip)
	eol := s.NextPosOf('\n')
	s2 := *s
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

func (s *Scanner) FindLineEnd() bool {
	// initial '/' already consumed

	defer func(offs int) {
		// reset scanner state to where it was upon calling FindLineEnd
		s.Ch = '/'
		s.Offset = offs
		s.ReadOffset = offs + 1
		s.Next() // consume initial '/' again
	}(s.Offset - 1)

	// read ahead until a newline, EOF, or non-comment tok is found
	for s.Ch == '/' || s.Ch == '*' {
		if s.Ch == '/' {
			// -style comment always contains a newline
			return true
		}
		/*-style comment: look for newline */
		s.Next()
		for s.Ch >= 0 {
			ch := s.Ch
			if ch == '\n' {
				return true
			}
			s.Next()
			if ch == '*' && s.Ch == '/' {
				s.Next()
				break
			}
		}
		s.SkipWhitespace() // s.InsertSemi is set
		if s.Ch < 0 || s.Ch == '\n' {
			return true
		}
		if s.Ch != '/' {
			// non-comment Token
			return false
		}
		s.Next() // consume '/'
	}
	return false
}

func (s *Scanner) ScanIdentifier() string {
	offs := s.Offset
	for runehelper.IsIdentifier(s.Ch) {
		s.Next()
	}
	return string(s.Src[offs:s.Offset])
}

func (s *Scanner) scanMantissa(base int) {
	for runehelper.DigitVal(s.Ch) < base {
		s.Next()
	}
}

func (s *Scanner) ScanNumber(seenDecimalPoint bool) (tok token.Token, lit string) {
	// DigitVal(s.ch) < 10
	offs := s.Offset
	tok = token.Int

	defer func() {
		lit = string(s.Src[offs:s.Offset])
	}()

	if seenDecimalPoint {
		offs--
		if tok != token.Decimal {
			tok = token.Float
		}
		s.scanMantissa(10)
		goto exponent
	}

	if s.Ch == '0' {
		// int or float
		offs := s.Offset
		s.Next()
		if s.Ch == 'x' || s.Ch == 'X' {
			// hexadecimal int
			s.Next()
			s.scanMantissa(16)
			if s.Offset-offs <= 2 {
				// only scanned "0x" or "0X"
				s.Error(offs, "illegal hexadecimal number")
			}
		} else {
			// octal int or float
			seenDecimalDigit := false
			s.scanMantissa(8)
			if s.Ch == '8' || s.Ch == '9' {
				// illegal octal int or float
				seenDecimalDigit = true
				s.scanMantissa(10)
			}
			if s.Ch == '.' || s.Ch == 'e' || s.Ch == 'E' || s.Ch == 'i' {
				goto fraction
			}
			// octal int
			if seenDecimalDigit {
				s.Error(offs, "illegal octal number")
			}
			// check if unsigned
			if s.Ch == 'u' {
				s.Next()
				tok = token.Uint
			} else if s.Ch == 'd' {
				s.Next()
				tok = token.Decimal
			}
		}
		return
	}
	// decimal int or float
	s.scanMantissa(10)

	// check if unsigned
	if s.Ch == 'u' {
		s.Next()
		tok = token.Uint
	} else if s.Ch == 'd' {
		s.Next()
		tok = token.Decimal
	}

fraction:
	if s.Ch == '.' {
		tok = token.Float
		s.Next()
		s.scanMantissa(10)
	}

exponent:
	if s.Ch == 'e' || s.Ch == 'E' {
		if tok != token.Decimal {
			tok = token.Float
		}
		s.Next()
		if s.Ch == '-' || s.Ch == '+' {
			s.Next()
		}
		if runehelper.DigitVal(s.Ch) < 10 {
			s.scanMantissa(10)
		} else {
			s.Error(offs, "illegal floating-point exponent")
		}
	}

	if s.Ch == 'd' && tok != token.Decimal && tok != token.Uint {
		tok = token.Decimal
		s.Next()
	}

	return
}

func (s *Scanner) scanEscape(quote rune) bool {
	offs := s.Offset

	var n int
	var base, max uint32
	switch s.Ch {
	case 'a', 'b', 'f', 'n', 'r', 't', 'v', '\\', quote:
		s.Next()
		return true
	case '0', '1', '2', '3', '4', '5', '6', '7':
		n, base, max = 3, 8, 255
	case 'x':
		s.Next()
		n, base, max = 2, 16, 255
	case 'u':
		s.Next()
		n, base, max = 4, 16, unicode.MaxRune
	case 'U':
		s.Next()
		n, base, max = 8, 16, unicode.MaxRune
	default:
		msg := "unknown escape sequence"
		if s.Ch < 0 {
			msg = "escape sequence not terminated"
		}
		s.Error(offs, msg)
		return false
	}

	var x uint32
	for n > 0 {
		d := uint32(runehelper.DigitVal(s.Ch))
		if d >= base {
			msg := fmt.Sprintf(
				"illegal character %#U in escape sequence", s.Ch)
			if s.Ch < 0 {
				msg = "escape sequence not terminated"
			}
			s.Error(s.Offset, msg)
			return false
		}
		x = x*base + d
		s.Next()
		n--
	}

	if x > max || 0xD800 <= x && x < 0xE000 {
		s.Error(offs, "escape sequence is invalid Unicode code point")
		return false
	}
	return true
}

func (s *Scanner) ScanRune() string {
	offs := s.Offset - 1 // '\'' opening already consumed

	valid := true
	n := 0
	for {
		ch := s.Ch
		if ch == '\n' || ch < 0 {
			// only report error if we don't have one already
			if valid {
				s.Error(offs, "rune literal not terminated")
				valid = false
			}
			break
		}
		s.Next()
		if ch == '\'' {
			break
		}
		n++
		if ch == '\\' {
			if !s.scanEscape('\'') {
				valid = false
			}
			// continue to read to closing quote
		}
	}

	if valid && n != 1 {
		s.Error(offs, "illegal rune literal")
	}
	return string(s.Src[offs:s.Offset])
}

func (s *Scanner) ScanString() string {
	offs := s.Offset - 1 // '"' opening already consumed

	for {
		ch := s.Ch
		if ch < 0 {
			s.Error(offs, "string literal not terminated")
			break
		}
		s.Next()
		if ch == '"' {
			break
		}
		if ch == '\\' {
			s.scanEscape('"')
		}
	}
	return string(s.Src[offs:s.Offset])
}

func (s *Scanner) ReadAtMany(quote []byte) []byte {
	var (
		offs = s.Offset - 1
		next = make([]byte, len(quote))
		w    bytes.Buffer
		r    = rune(quote[0])
	)

	for {
		b := s.ReadAt(r)
		if s.Ch == -1 {
			s.Error(offs, "unexpected EOF")
			break
		}
		w.Write(b)
		next[0] = byte(s.Ch)
		x := s.PeekN(len(next) - 1)
		copy(next[1:], x)

		if bytes.Equal(next, quote) {
			break
		}

		w.WriteRune(s.Ch)
		s.Next()
	}

	return w.Bytes()
}

func (s *Scanner) ScanRawString() (string, bool) {
	offs := s.Offset - 1 // '`' opening already consumed

	// if is raw heredoc, minimal 3 chars (current more 2)
	if s.Ch == '`' && s.Peek() == '`' {
		quote := []byte{byte(s.Ch)}
		quote = append(quote, s.ReadWhen('`')...)
		if len(quote)%2 != 1 {
			s.Error(offs, "raw heredoc literal not open")
		}
		var w bytes.Buffer
		w.Write(quote)
		if s.Ch == '\n' {
			quote = append([]byte{'\n'}, quote...)
		}
		w.Write(s.ReadAtMany(quote))
		w.Write(s.ReadCount(len(quote)))

		return w.String(), true
	}

	for {
		ch := s.Ch
		if ch < 0 {
			s.Error(offs, "raw string literal not terminated")
			break
		}

		s.Next()

		if ch == '`' {
			break
		}
	}

	return string(s.Src[offs:s.Offset]), false
}

// StripCR removes carriage return characters.
func StripCR(b []byte, comment bool) []byte {
	c := make([]byte, len(b))
	i := 0
	for j, ch := range b {
		// In a /*-style comment, don't strip \r from *\r/ (incl. sequences of
		// \r from *\r\r...\r/) since the resulting  */ would terminate the
		// comment too early unless the \r is immediately following the opening
		// /* in which case it's ok because /*/ is not closed yet.
		if ch != '\r' || comment && i > len("/*") && c[i-1] == '*' &&
			j+1 < len(b) && b[j+1] == '/' {
			c[i] = ch
			i++
		}
	}
	return c[:i]
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

	s.TextTrimLeft = false

	code = Token{
		Token:   token.MixedCodeStart,
		Pos:     s.File.FileSetPos(end),
		Literal: lit,
		Data:    data,
	}

	if s.Ch == '=' {
		s.ToText = true
		s.NextNoSpace()
		code.Literal += "="
		code.Token = token.MixedValueStart
	} else if s.mode.Has(MixedExprAsValue) {
		s.ToText = true
		code.Token = token.MixedValueStart
	}

	if leftText != nil {
		code.Prev = append(code.Prev, *leftText)
	}

	return
}
