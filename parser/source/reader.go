package source

import (
	"bytes"
	"fmt"
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/gad-lang/gad/runehelper"
)

// BOM byte order mark
const BOM = 0xFEFF

type Reader struct {
	Data               any
	File               *File
	Ch                 rune // current character
	Offset             int  // character offset
	ReadOffset         int  // reading offset (position after current character)
	lineOffset         int  // current line offset
	NewLineEscaped     bool
	Src                []byte
	NewLineEscape      func() bool
	SkipWhitespaceFunc func(r *Reader)
	errorHandler       []ScannerErrorHandler // error reporting; or nil
	errorCount         int                   // number of errors encountered
	NextHandlers
}

type FileReaderOption func(r *Reader)

func FileReaderWithSkipWhitespaceFunc(f func(fr *Reader)) FileReaderOption {
	return func(r *Reader) {
		r.SkipWhitespaceFunc = f
	}
}
func FileReaderWithData(data any) FileReaderOption {
	return func(r *Reader) {
		r.Data = data
	}
}

func NewFileReader(file *File, option ...FileReaderOption) (fr *Reader) {
	src := file.Data
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

	fr = &Reader{
		File: file,
		Src:  src,
		Ch:   ' ',

		SkipWhitespaceFunc: func(r *Reader) {
			for r.IsSpace() {
				r.Next()
			}
		},
	}

	for _, opt := range option {
		opt(fr)
	}

	return fr
}

func (s *Reader) Read(b []byte) (n int, err error) {
	v := s.ReadCount(len(b))
	for i := range v {
		b[i] = v[i]
	}
	n = len(v)
	return
}

func (s *Reader) Start() {
	s.Next()
	if s.Ch == BOM {
		s.Next() // ignore BOM at file beginning
	}
}

func (s *Reader) ErrorHandler(h ...ScannerErrorHandler) {
	s.errorHandler = append(s.errorHandler, h...)
}
func (s *Reader) SourceFile() *File {
	return s.File
}

func (s *Reader) Source() []byte {
	return s.Src
}

// ErrorCount returns the number of errors.
func (s *Reader) ErrorCount() int {
	return s.errorCount
}

func (s *Reader) HasError() bool {
	return len(s.errorHandler) > 0
}

func (s *Reader) SkipWhitespace() {
	s.SkipWhitespaceFunc(s)
}

func (s *Reader) IsSpace() bool {
	return s.Ch == ' ' || s.Ch == '\t' || s.Ch == '\n'
}

func (s *Reader) NextC(count int) {
	for i := 0; i < count; i++ {
		s.Next()
	}
}

func (s *Reader) Skip(str string) {
	for _, r := range str {
		if s.Ch != r {
			break
		}
		s.Next()
	}
}

func (s *Reader) NextTo(v string) {
	for _, r := range v {
		s.Expect(r, "next to: "+strconv.Quote(v))
		s.Next()
	}
}

func (s *Reader) Next() {
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

func (s *Reader) PeekAtEndLine() (start, end int) {
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

func (s *Reader) NextNoSpace() {
	s.Next()
	s.SkipWhitespace()
}

func (s *Reader) Peek() byte {
	if s.ReadOffset < len(s.Src) {
		return s.Src[s.ReadOffset]
	}
	return 0
}

func (s *Reader) NextPosOf(b byte) (end int) {
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

func (s *Reader) ReadAt(b rune) []byte {
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

func (s *Reader) ReadCount(q int) []byte {
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

func (s *Reader) ReadWhen(b rune) []byte {
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

func (s *Reader) PeekNoSpace() byte {
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

func (s *Reader) PeekInlineNoSpace() byte {
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

func (s *Reader) IndexOfInlineNoSpace() int {
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

func (s *Reader) PeekNoSpaceN(n int) []byte {
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

func (s *Reader) PeekN(n int) []byte {
	if (s.ReadOffset + n) <= len(s.Src) {
		return s.Src[s.ReadOffset : s.ReadOffset+n]
	}
	return nil
}

func (s *Reader) PeekEq(str string) bool {
	b := s.PeekN(len(str))
	if b != nil {
		return string(b) == str
	}
	return true
}

func (s *Reader) PeekNoSpaceEq(to string, skip int) bool {
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

func (s *Reader) PeekNoSingleSpaceEq(to string, skip int) (length int) {
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

func (s *Reader) PeekIdentEq(to string, skip int) bool {
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

func (s *Reader) Error(offset int, msg string) {
	pos := s.File.Position(s.File.FileSetPos(offset))
	for _, h := range s.errorHandler {
		h(pos, msg)
	}
	s.errorCount++
}

func (s *Reader) Expect(ch rune, msg string) bool {
	if s.Ch != ch {
		s.ExpectError(msg + fmt.Sprintf(", but got %s", string(s.Ch)))
		s.Ch = -1
		return false
	}
	return true
}

func (s *Reader) ExpectError(msg string) {
	s.Error(s.Offset, "Expect: "+msg)
}

func (s *Reader) FindLineEnd() bool {
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

func (s Reader) ReadEscape(quote rune) bool {
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

func (s *Reader) ReadAtMany(quote []byte) []byte {
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
func (s *Reader) readEscape(quote rune) bool {
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

func (s *Reader) ScanRune() string {
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
			if !s.readEscape('\'') {
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

func (s *Reader) ScanString() string {
	return s.ScanStringDelimiter('"')
}

func (s *Reader) ScanStringDelimiter(delimiter rune) string {
	offs := s.Offset - 1 // delimiter opening already consumed

	for {
		ch := s.Ch
		if ch < 0 {
			s.Error(offs, "string literal not terminated")
			break
		}
		s.Next()
		if ch == delimiter {
			break
		}
		if ch == '\\' {
			s.readEscape(delimiter)
		}
	}
	return string(s.Src[offs:s.Offset])
}

func (s *Reader) ScanRawString() (string, bool) {
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

func (s *Reader) ScanMantissa(base int) {
	for runehelper.DigitVal(s.Ch) < base {
		s.Next()
	}
}

type NumberType uint8

const (
	Int NumberType = iota
	Uint
	Float
	Decimal
)

func (s *Reader) ScanNumber(seenDecimalPoint bool) (tok NumberType, lit string) {
	// DigitVal(s.ch) < 10
	offs := s.Offset
	defer func() {
		lit = string(s.Src[offs:s.Offset])
	}()

	if seenDecimalPoint {
		offs--
		if tok != Decimal {
			tok = Float
		}
		s.ScanMantissa(10)
		goto exponent
	}

	if s.Ch == '0' {
		// int or float
		offs := s.Offset
		s.Next()
		if s.Ch == 'x' || s.Ch == 'X' {
			// hexadecimal int
			s.Next()
			s.ScanMantissa(16)
			if s.Offset-offs <= 2 {
				// only scanned "0x" or "0X"
				s.Error(offs, "illegal hexadecimal number")
			}
		} else {
			// octal int or float
			seenDecimalDigit := false
			s.ScanMantissa(8)
			if s.Ch == '8' || s.Ch == '9' {
				// illegal octal int or float
				seenDecimalDigit = true
				s.ScanMantissa(10)
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
				tok = Uint
			} else if s.Ch == 'd' {
				s.Next()
				tok = Decimal
			}
		}
		return
	}
	// decimal int or float
	s.ScanMantissa(10)

	// check if unsigned
	if s.Ch == 'u' {
		s.Next()
		tok = Uint
	} else if s.Ch == 'd' {
		s.Next()
		tok = Decimal
	}

fraction:
	if s.Ch == '.' {
		tok = Float
		s.Next()
		s.ScanMantissa(10)
	}

exponent:
	if s.Ch == 'e' || s.Ch == 'E' {
		if tok != Decimal {
			tok = Float
		}
		s.Next()
		if s.Ch == '-' || s.Ch == '+' {
			s.Next()
		}
		if runehelper.DigitVal(s.Ch) < 10 {
			s.ScanMantissa(10)
		} else {
			s.Error(offs, "illegal floating-point exponent")
		}
	}

	if s.Ch == 'd' && tok != Decimal && tok != Uint {
		tok = Decimal
		s.Next()
	}

	return
}
