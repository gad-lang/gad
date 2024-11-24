package parser

import (
	"bytes"
	"fmt"
	"github.com/gad-lang/gad/runehelper"
	"strconv"
	"unicode"
	"unicode/utf8"
)

type FileReader struct {
	File               *SourceFile
	Ch                 rune // current character
	Offset             int  // character offset
	ReadOffset         int  // reading offset (position after current character)
	lineOffset         int  // current line offset
	NewLineEscaped     bool
	Src                []byte
	NewLineEscape      func() bool
	SkipWhitespaceFunc func(s *FileReader)
	errorHandler       []ScannerErrorHandler // error reporting; or nil
	errorCount         int                   // number of errors encountered
	Handlers
}

func NewFileReader(file *SourceFile, src []byte) *FileReader {
	if file.Size != len(src) {
		panic(fmt.Sprintf("file size (%d) does not match Src len (%d)",
			file.Size, len(src)))
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

	fr := &FileReader{
		File: file,
		Src:  src,
		Ch:   ' ',

		SkipWhitespaceFunc: func(s *FileReader) {
			for s.IsSpace() {
				s.Next()
			}
		},
	}

	fr.Next()
	if fr.Ch == BOM {
		fr.Next() // ignore BOM at file beginning
	}

	return fr
}

func (s *FileReader) ErrorHandler(h ...ScannerErrorHandler) {
	s.errorHandler = append(s.errorHandler, h...)
}
func (s *FileReader) SourceFile() *SourceFile {
	return s.File
}

func (s *FileReader) Source() []byte {
	return s.Src
}

// ErrorCount returns the number of errors.
func (s *FileReader) ErrorCount() int {
	return s.errorCount
}

func (s *FileReader) HasError() bool {
	return len(s.errorHandler) > 0
}

func (s *FileReader) SkipWhitespace() {
	s.SkipWhitespaceFunc(s)
}

func (s *FileReader) IsSpace() bool {
	return s.Ch == ' ' || s.Ch == '\t' || s.Ch == '\n'
}

func (s *FileReader) NextC(count int) {
	for i := 0; i < count; i++ {
		s.Next()
	}
}

func (s *FileReader) Skip(str string) {
	for _, r := range str {
		if s.Ch != r {
			break
		}
		s.Next()
	}
}

func (s *FileReader) NextTo(v string) {
	for _, r := range v {
		s.Expect(r, "next to: "+strconv.Quote(v))
		s.Next()
	}
}

func (s *FileReader) Next() {
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

func (s *FileReader) PeekAtEndLine() (start, end int) {
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

func (s *FileReader) NextNoSpace() {
	s.Next()
	s.SkipWhitespace()
}

func (s *FileReader) Peek() byte {
	if s.ReadOffset < len(s.Src) {
		return s.Src[s.ReadOffset]
	}
	return 0
}

func (s *FileReader) NextPosOf(b byte) (end int) {
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

func (s *FileReader) ReadAt(b rune) []byte {
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

func (s *FileReader) ReadCount(q int) []byte {
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

func (s *FileReader) ReadWhen(b rune) []byte {
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

func (s *FileReader) PeekNoSpace() byte {
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

func (s *FileReader) PeekInlineNoSpace() byte {
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

func (s *FileReader) IndexOfInlineNoSpace() int {
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

func (s *FileReader) PeekNoSpaceN(n int) []byte {
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

func (s *FileReader) PeekN(n int) []byte {
	if (s.ReadOffset + n) <= len(s.Src) {
		return s.Src[s.ReadOffset : s.ReadOffset+n]
	}
	return nil
}

func (s *FileReader) PeekEq(str string) bool {
	b := s.PeekN(len(str))
	if b != nil {
		return string(b) == str
	}
	return true
}

func (s *FileReader) PeekNoSpaceEq(to string, skip int) bool {
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

func (s *FileReader) PeekNoSingleSpaceEq(to string, skip int) (length int) {
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

func (s *FileReader) PeekIdentEq(to string, skip int) bool {
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

func (s *FileReader) Error(offset int, msg string) {
	pos := s.File.Position(s.File.FileSetPos(offset))
	for _, h := range s.errorHandler {
		h(pos, msg)
	}
	s.errorCount++
}

func (s *FileReader) Expect(ch rune, msg string) bool {
	if s.Ch != ch {
		s.ExpectError(msg + fmt.Sprintf(", but got %s", string(s.Ch)))
		s.Ch = -1
		return false
	}
	return true
}

func (s *FileReader) ExpectError(msg string) {
	s.Error(s.Offset, "Expect: "+msg)
}

func (s *FileReader) FindLineEnd() bool {
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

func (s *Scanner) ReadEscape(quote rune) bool {
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

func (s *FileReader) ReadAtMany(quote []byte) []byte {
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
