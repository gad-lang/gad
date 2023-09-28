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
	"unicode"
	"unicode/utf8"

	"github.com/gad-lang/gad/token"
)

// byte order mark
const bom = 0xFEFF

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

type Token struct {
	Pos        Pos
	Token      token.Token
	Literal    string
	InsertSemi bool
	Data       any
}

var _ fmt.Stringer = Token{}

func (t Token) String() string {
	return t.Token.String() + ": " + t.Literal
}

// Scanner reads the Gad source text. It's based on Go's scanner
// implementation.
type Scanner struct {
	file         *SourceFile         // source file handle
	src          []byte              // source
	ch           rune                // current character
	offset       int                 // character offset
	readOffset   int                 // reading offset (position after current character)
	lineOffset   int                 // current line offset
	insertSemi   bool                // insert a semicolon before next newline
	errorHandler ScannerErrorHandler // error reporting; or nil
	errorCount   int                 // number of errors encountered
	mode         ScanMode
	inCode       bool
	toText       bool
	braceCount   int
	tokenPool    []Token
	textTrimLeft bool
}

// NewScanner creates a Scanner.
func NewScanner(
	file *SourceFile,
	src []byte,
	errorHandler ScannerErrorHandler,
	mode ScanMode,
) *Scanner {
	if file.Size != len(src) {
		panic(fmt.Sprintf("file size (%d) does not match src len (%d)",
			file.Size, len(src)))
	}

	s := &Scanner{
		file:         file,
		src:          src,
		errorHandler: errorHandler,
		ch:           ' ',
		mode:         mode,
	}

	s.next()
	if s.ch == bom {
		s.next() // ignore BOM at file beginning
	}

	return s
}

// ErrorCount returns the number of errors.
func (s *Scanner) ErrorCount() int {
	return s.errorCount
}

func (s *Scanner) AddNextToken(n ...Token) {
	s.tokenPool = append(s.tokenPool, n...)
}

// Scan returns a token, token literal and its position.
func (s *Scanner) Scan() (t Token) {
	if len(s.tokenPool) > 0 {
		t = s.tokenPool[0]
		s.tokenPool = s.tokenPool[1:]
		return
	}

	t.Pos = s.file.FileSetPos(s.offset)

	if s.mode.Has(Mixed) && !s.inCode && s.ch != -1 {
		start := s.offset
		for {
			var scape bool
			switch s.ch {
			case '\\':
				if scape {
					scape = false
				}
			case -1:
				t.Token = token.Text
				t.Literal = string(s.src[start:s.offset])
				if s.textTrimLeft {
					t.Literal = trimSpace(true, false, t.Literal)
				}
				s.textTrimLeft = false
				s.tokenPool = append(s.tokenPool, Token{Pos: s.file.FileSetPos(s.offset), Token: token.EOF})
				return
			case '#':
				if !scape {
					if s.peek() == '{' {
						var (
							end = s.offset
							lit = "#{"
						)
						s.inCode = true
						s.next()
						s.next()
						s.braceCount++

						t.Literal = string(s.src[start:end])
						t.Token = token.Text

						switch s.ch {
						case '-':
							s.nextNoSpace()
							t.Literal = trimSpace(s.textTrimLeft, true, t.Literal)
						}

						s.textTrimLeft = false

						if s.ch == '=' {
							s.toText = true
							s.nextNoSpace()
							next := Token{Token: token.ToTextBegin, Pos: s.file.FileSetPos(end), Literal: lit + "="}
							if t.Literal == "" {
								t = next
							} else {
								s.AddNextToken(next)
							}
						} else {
							next := Token{Token: token.CodeBegin, Pos: s.file.FileSetPos(end), Literal: lit}
							if t.Literal == "" {
								t = next
							} else {
								s.AddNextToken(next)
							}
						}
						return
					}

					if !s.mode.Has(ConfigDisabled) && string(s.peekNoSpaceN(4)) == "gad:" {
						t, ok := s.scanConfig()
						if ok {
							return t
						}
					}

					goto do
				}
			}
			s.next()
		}
	}

do:
	s.skipWhitespace()
	t.Pos = s.file.FileSetPos(s.offset)

	insertSemi := false

	// determine token value
	switch ch := s.ch; {
	case isLetter(ch):
		t.Literal = s.scanIdentifier()
		t.Token = token.Lookup(t.Literal)
		switch t.Token {
		case token.Ident, token.Break, token.Continue, token.Return,
			token.True, token.False, token.Nil,
			token.Callee, token.Args, token.NamedArgs,
			token.StdIn, token.StdOut, token.StdErr:
			insertSemi = true
		}
	case '0' <= ch && ch <= '9':
		insertSemi = true
		t.Token, t.Literal = s.scanNumber(false)
	default:
		s.next() // always make progress

		switch ch {
		case -1: // EOF
			if s.insertSemi {
				s.insertSemi = false // EOF consumed
				t.Data = "\n"
				t.Token = token.Semicolon
				return
			}
			t.Token = token.EOF
		case '\n':
			// we only reach here if s.insertSemi was set in the first place
			s.insertSemi = false // newline consumed
			t.Literal = "\n"
			t.Token = token.Semicolon
			return
		case '"':
			insertSemi = true
			t.Token = token.String
			t.Literal = s.scanString()
		case '\'':
			insertSemi = true
			t.Token = token.Char
			t.Literal = s.scanRune()
		case '`':
			insertSemi = true
			t.Token = token.String
			t.Literal = s.scanRawString()
		case ':':
			t.Token = s.switch2(token.Colon, token.Define)
		case '.':
			if '0' <= s.ch && s.ch <= '9' {
				insertSemi = true
				t.Token, t.Literal = s.scanNumber(true)
			} else {
				t.Token = token.Period
				if s.ch == '.' && s.peek() == '.' {
					s.next()
					s.next() // consume last '.'
					t.Token = token.Ellipsis
				}
			}
		case ',':
			t.Token = token.Comma
		case '?':
			switch s.ch {
			case '.':
				s.next()
				t.Token = token.NullishSelector
			case '?':
				if s.peek() == '=' {
					s.next()
					s.next()
					t.Token = token.NullichAssign
				} else {
					s.next()
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
		case ')':
			insertSemi = true
			t.Token = token.RParen
		case '[':
			t.Token = token.LBrack
		case ']':
			insertSemi = true
			t.Token = token.RBrack
		case '{':
			t.Token = token.LBrace
			if s.inCode {
				s.braceCount++
			}
		case '}':
			insertSemi = true
			t.Token = token.RBrace
			if s.inCode {
				s.braceCount--
				if s.braceCount == 0 {
					s.inCode = false
					insertSemi = false
					t.Token = token.Semicolon
					t.Literal = "\n"
					if s.toText {
						t.Token = token.ToTextEnd
						s.toText = false
						t.Literal = "}"
						t.Data = s.textTrimLeft
					} else {
						next := Token{Token: token.CodeEnd, Literal: "}", Pos: t.Pos, Data: s.textTrimLeft}
						if !s.mode.Has(DontInsertSemis) {
							s.AddNextToken(t)
							t = next
						} else {
							return next
						}
					}
				}
			}
		case '+':
			t.Token = s.switch3(token.Add, token.AddAssign, '+', token.Inc)
			if t.Token == token.Inc {
				insertSemi = true
			}
		case '-':
			if s.ch == '}' {
				s.textTrimLeft = true
				goto do
			}

			t.Token = s.switch3(token.Sub, token.SubAssign, '-', token.Dec)
			if t.Token == token.Dec {
				insertSemi = true
			}
		case '*':
			t.Token = s.switch2(token.Mul, token.MulAssign)
		case '/':
			if s.ch == '/' || s.ch == '*' {
				// comment
				if s.insertSemi && s.findLineEnd() {
					// reset position to the beginning of the comment
					s.ch = '/'
					s.offset = s.file.Offset(t.Pos)
					s.readOffset = s.offset + 1
					s.insertSemi = false // newline consumed
					t.Data = "\n"
					t.Token = token.Semicolon
					return
				}
				comment := s.scanComment()
				if !s.mode.Has(ScanComments) {
					// skip comment
					s.insertSemi = false // newline consumed
					return s.Scan()
				}
				t.Token = token.Comment
				t.Literal = comment
			} else {
				t.Token = s.switch2(token.Quo, token.QuoAssign)
			}
		case '%':
			t.Token = s.switch2(token.Rem, token.RemAssign)
		case '^':
			t.Token = s.switch2(token.Xor, token.XorAssign)
		case '<':
			t.Token = s.switch4(token.Less, token.LessEq, '<',
				token.Shl, token.ShlAssign)
		case '>':
			t.Token = s.switch4(token.Greater, token.GreaterEq, '>',
				token.Shr, token.ShrAssign)
		case '=':
			t.Token = s.switch2(token.Assign, token.Equal)
		case '!':
			t.Token = s.switch2(token.Not, token.NotEqual)
		case '&':
			if s.ch == '^' {
				s.next()
				t.Token = s.switch2(token.AndNot, token.AndNotAssign)
			} else {
				t.Token = s.switch3(token.And, token.AndAssign, '&', token.LAnd)
			}
		case '|':
			if s.ch == '=' {
				s.next()
				t.Token = token.OrAssign
			} else if s.ch == '|' {
				if s.peek() == '=' {
					s.next()
					s.next()
					t.Token = token.LOrAssign
				} else {
					s.next()
					t.Token = token.LOr
				}
			} else {
				t.Token = token.Or
			}
		case '#':
			if !s.mode.Has(ConfigDisabled) && string(s.peekNoSpaceN(4)) == "gad:" {
				t, ok := s.scanConfig()
				if ok {
					return t
				}
			}
			fallthrough
		default:
			// next reports unexpected BOMs - don't repeat
			if ch != bom {
				s.error(s.file.Offset(t.Pos),
					fmt.Sprintf("illegal character %#U", ch))
			}
			insertSemi = s.insertSemi // preserve insertSemi info
			t.Token = token.Illegal
			t.Literal = string(ch)
		}
	}
	if !s.mode.Has(DontInsertSemis) {
		s.insertSemi = insertSemi
	}
	return
}

func (s *Scanner) next() {
	if s.readOffset < len(s.src) {
		s.offset = s.readOffset
		if s.ch == '\n' {
			s.lineOffset = s.offset
			s.file.AddLine(s.offset)
		}
		r, w := rune(s.src[s.readOffset]), 1
		switch {
		case r == 0:
			s.error(s.offset, "illegal character NUL")
		case r >= utf8.RuneSelf:
			// not ASCII
			r, w = utf8.DecodeRune(s.src[s.readOffset:])
			if r == utf8.RuneError && w == 1 {
				s.error(s.offset, "illegal UTF-8 encoding")
			} else if r == bom && s.offset > 0 {
				s.error(s.offset, "illegal byte order mark")
			}
		}
		s.readOffset += w
		s.ch = r
	} else {
		s.offset = len(s.src)
		if s.ch == '\n' {
			s.lineOffset = s.offset
			s.file.AddLine(s.offset)
		}
		s.ch = -1 // eof
	}
}

func (s *Scanner) nextNoSpace() {
	s.next()
	s.skipWhitespace()
}

func (s *Scanner) peek() byte {
	if s.readOffset < len(s.src) {
		return s.src[s.readOffset]
	}
	return 0
}

func (s *Scanner) peekNoSpaceN(n int) []byte {
	off := s.readOffset
	for off < len(s.src) {
		switch s.src[off] {
		case ' ', '\r', '\n', '\t':
			off++
		default:
			part := s.src[off:]
			if len(part) >= n {
				return part[:n]
			}
			return nil
		}
	}
	return nil
}

func (s *Scanner) error(offset int, msg string) {
	if s.errorHandler != nil {
		s.errorHandler(s.file.Position(s.file.FileSetPos(offset)), msg)
	}
	s.errorCount++
}

func (s *Scanner) scanConfig() (t Token, ok bool) {
	off, roff, loff := s.offset, s.readOffset, s.lineOffset
	pos := s.file.FileSetPos(off - 1)
	p := s.file.position(pos)
	if p.Column != 1 {
		return
	}
	s.nextNoSpace()
	if isLetter(s.ch) {
		name := s.scanIdentifier()
		if name == "gad" {
			s.skipWhitespace()
			if s.ch == ':' {
				ok = true
				s.nextNoSpace()
				var (
					start = s.offset
					end   int
					semi  string
				)

			cfg_line:
				for {
					switch s.ch {
					case '\n', ';', -1:
						end = s.offset - 1
						if s.src[end] == '\r' {
							end--
						}
						t.Data = s.file.FileSetPos(end)
						if s.ch != -1 {
							semi = string(s.ch)
						}
						s.next()
						break cfg_line
					}
					s.next()
				}

				t.Literal = string(s.src[start : end+1])
				t.Token = token.Config
				t.Pos = pos
				s.tokenPool = append(s.tokenPool, Token{Pos: s.file.FileSetPos(end), Token: token.Semicolon, Literal: semi})
				return
			}
		}
	}
	s.offset, s.readOffset, s.lineOffset = off, roff, loff
	return
}

func (s *Scanner) scanComment() string {
	// initial '/' already consumed; s.ch == '/' || s.ch == '*'
	offs := s.offset - 1 // position of initial '/'
	var numCR int

	if s.ch == '/' {
		// -style comment
		// (the final '\n' is not considered part of the comment)
		s.next()
		for s.ch != '\n' && s.ch >= 0 {
			if s.ch == '\r' {
				numCR++
			}
			s.next()
		}
		goto exit
	}

	/*-style comment */
	s.next()
	for s.ch >= 0 {
		ch := s.ch
		if ch == '\r' {
			numCR++
		}
		s.next()
		if ch == '*' && s.ch == '/' {
			s.next()
			goto exit
		}
	}

	s.error(offs, "comment not terminated")

exit:
	lit := s.src[offs:s.offset]

	// On Windows, a (//-comment) line may end in "\r\n".
	// Remove the final '\r' before analyzing the text for line directives (matching the compiler).
	// Remove any other '\r' afterwards (matching the pre-existing behavior of the scanner).
	if numCR > 0 && len(lit) >= 2 && lit[1] == '/' && lit[len(lit)-1] == '\r' {
		lit = lit[:len(lit)-1]
		numCR--
	}
	if numCR > 0 {
		lit = StripCR(lit, lit[1] == '*')
	}
	return string(lit)
}

func (s *Scanner) findLineEnd() bool {
	// initial '/' already consumed

	defer func(offs int) {
		// reset scanner state to where it was upon calling findLineEnd
		s.ch = '/'
		s.offset = offs
		s.readOffset = offs + 1
		s.next() // consume initial '/' again
	}(s.offset - 1)

	// read ahead until a newline, EOF, or non-comment tok is found
	for s.ch == '/' || s.ch == '*' {
		if s.ch == '/' {
			// -style comment always contains a newline
			return true
		}
		/*-style comment: look for newline */
		s.next()
		for s.ch >= 0 {
			ch := s.ch
			if ch == '\n' {
				return true
			}
			s.next()
			if ch == '*' && s.ch == '/' {
				s.next()
				break
			}
		}
		s.skipWhitespace() // s.insertSemi is set
		if s.ch < 0 || s.ch == '\n' {
			return true
		}
		if s.ch != '/' {
			// non-comment Token
			return false
		}
		s.next() // consume '/'
	}
	return false
}

func (s *Scanner) scanIdentifier() string {
	offs := s.offset
	for isLetter(s.ch) || isDigit(s.ch) {
		s.next()
	}
	return string(s.src[offs:s.offset])
}

func (s *Scanner) scanMantissa(base int) {
	for digitVal(s.ch) < base {
		s.next()
	}
}

func (s *Scanner) scanNumber(seenDecimalPoint bool) (tok token.Token, lit string) {
	// digitVal(s.ch) < 10
	offs := s.offset
	tok = token.Int

	defer func() {
		lit = string(s.src[offs:s.offset])
	}()

	if seenDecimalPoint {
		offs--
		if tok != token.Decimal {
			tok = token.Float
		}
		s.scanMantissa(10)
		goto exponent
	}

	if s.ch == '0' {
		// int or float
		offs := s.offset
		s.next()
		if s.ch == 'x' || s.ch == 'X' {
			// hexadecimal int
			s.next()
			s.scanMantissa(16)
			if s.offset-offs <= 2 {
				// only scanned "0x" or "0X"
				s.error(offs, "illegal hexadecimal number")
			}
		} else {
			// octal int or float
			seenDecimalDigit := false
			s.scanMantissa(8)
			if s.ch == '8' || s.ch == '9' {
				// illegal octal int or float
				seenDecimalDigit = true
				s.scanMantissa(10)
			}
			if s.ch == '.' || s.ch == 'e' || s.ch == 'E' || s.ch == 'i' {
				goto fraction
			}
			// octal int
			if seenDecimalDigit {
				s.error(offs, "illegal octal number")
			}
			// check if unsigned
			if s.ch == 'u' {
				s.next()
				tok = token.Uint
			} else if s.ch == 'd' {
				s.next()
				tok = token.Decimal
			}
		}
		return
	}
	// decimal int or float
	s.scanMantissa(10)

	// check if unsigned
	if s.ch == 'u' {
		s.next()
		tok = token.Uint
	} else if s.ch == 'd' {
		s.next()
		tok = token.Decimal
	}

fraction:
	if s.ch == '.' {
		tok = token.Float
		s.next()
		s.scanMantissa(10)
	}

exponent:
	if s.ch == 'e' || s.ch == 'E' {
		if tok != token.Decimal {
			tok = token.Float
		}
		s.next()
		if s.ch == '-' || s.ch == '+' {
			s.next()
		}
		if digitVal(s.ch) < 10 {
			s.scanMantissa(10)
		} else {
			s.error(offs, "illegal floating-point exponent")
		}
	}

	if s.ch == 'd' && tok != token.Decimal && tok != token.Uint {
		tok = token.Decimal
		s.next()
	}

	return
}

func (s *Scanner) scanEscape(quote rune) bool {
	offs := s.offset

	var n int
	var base, max uint32
	switch s.ch {
	case 'a', 'b', 'f', 'n', 'r', 't', 'v', '\\', quote:
		s.next()
		return true
	case '0', '1', '2', '3', '4', '5', '6', '7':
		n, base, max = 3, 8, 255
	case 'x':
		s.next()
		n, base, max = 2, 16, 255
	case 'u':
		s.next()
		n, base, max = 4, 16, unicode.MaxRune
	case 'U':
		s.next()
		n, base, max = 8, 16, unicode.MaxRune
	default:
		msg := "unknown escape sequence"
		if s.ch < 0 {
			msg = "escape sequence not terminated"
		}
		s.error(offs, msg)
		return false
	}

	var x uint32
	for n > 0 {
		d := uint32(digitVal(s.ch))
		if d >= base {
			msg := fmt.Sprintf(
				"illegal character %#U in escape sequence", s.ch)
			if s.ch < 0 {
				msg = "escape sequence not terminated"
			}
			s.error(s.offset, msg)
			return false
		}
		x = x*base + d
		s.next()
		n--
	}

	if x > max || 0xD800 <= x && x < 0xE000 {
		s.error(offs, "escape sequence is invalid Unicode code point")
		return false
	}
	return true
}

func (s *Scanner) scanRune() string {
	offs := s.offset - 1 // '\'' opening already consumed

	valid := true
	n := 0
	for {
		ch := s.ch
		if ch == '\n' || ch < 0 {
			// only report error if we don't have one already
			if valid {
				s.error(offs, "rune literal not terminated")
				valid = false
			}
			break
		}
		s.next()
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
		s.error(offs, "illegal rune literal")
	}
	return string(s.src[offs:s.offset])
}

func (s *Scanner) scanString() string {
	offs := s.offset - 1 // '"' opening already consumed

	for {
		ch := s.ch
		if ch == '\n' || ch < 0 {
			s.error(offs, "string literal not terminated")
			break
		}
		s.next()
		if ch == '"' {
			break
		}
		if ch == '\\' {
			s.scanEscape('"')
		}
	}
	return string(s.src[offs:s.offset])
}

func (s *Scanner) scanRawString() string {
	offs := s.offset - 1 // '`' opening already consumed

	hasCR := false
	for {
		ch := s.ch
		if ch < 0 {
			s.error(offs, "raw string literal not terminated")
			break
		}

		s.next()

		if ch == '`' {
			break
		}

		if ch == '\r' {
			hasCR = true
		}
	}

	lit := s.src[offs:s.offset]
	if hasCR {
		lit = StripCR(lit, false)
	}
	return string(lit)
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

func (s *Scanner) skipWhitespace() {
	for s.ch == ' ' || s.ch == '\t' || s.ch == '\n' && !s.insertSemi ||
		s.ch == '\r' {
		s.next()
	}
}

func (s *Scanner) switch2(tok0, tok1 token.Token) token.Token {
	if s.ch == '=' {
		s.next()
		return tok1
	}
	return tok0
}

func (s *Scanner) switch3(
	tok0, tok1 token.Token,
	ch2 rune,
	tok2 token.Token,
) token.Token {
	if s.ch == '=' {
		s.next()
		return tok1
	}
	if s.ch == ch2 {
		s.next()
		return tok2
	}
	return tok0
}

func (s *Scanner) switch4(
	tok0, tok1 token.Token,
	ch2 rune,
	tok2, tok3 token.Token,
) token.Token {
	if s.ch == '=' {
		s.next()
		return tok1
	}
	if s.ch == ch2 {
		s.next()
		if s.ch == '=' {
			s.next()
			return tok3
		}
		return tok2
	}
	return tok0
}

func isLetter(ch rune) bool {
	return ch == '$' || 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_' ||
		ch >= utf8.RuneSelf && unicode.IsLetter(ch)
}

func isDigit(ch rune) bool {
	return '0' <= ch && ch <= '9' ||
		ch >= utf8.RuneSelf && unicode.IsDigit(ch)
}

func digitVal(ch rune) int {
	switch {
	case '0' <= ch && ch <= '9':
		return int(ch - '0')
	case 'a' <= ch && ch <= 'f':
		return int(ch - 'a' + 10)
	case 'A' <= ch && ch <= 'F':
		return int(ch - 'A' + 10)
	}
	return 16 // larger than any legal digit val
}
