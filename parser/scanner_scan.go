package parser

import (
	"fmt"

	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/runehelper"
	"github.com/gad-lang/gad/token"
)

// bytesLitPrefixKey is the PToken data key under which a bytes literal prefix
// ("b" or "h") is stored when the scanner recognises a b"..."/h"..." literal.
const bytesLitPrefixKey = "bytesLitPrefix"

// dateTimeLitKey is the PToken data key set when the scanner recognises a
// digit-suffix date/time literal (20260131D, 235955T, 1781609136U). Its value
// is the suffix rune ('D', 'T' or 'U'); the token's Literal holds the numeric
// body without the suffix.
const dateTimeLitKey = "dateTimeLit"

// durationLitKey is the PToken data flag set when the scanner recognises a
// d"..."/d`...` duration literal (its string body is a Go duration string).
const durationLitKey = "durationLit"

// isDateTimeBodyByte reports whether b may appear in the body of a date/time
// literal: a digit, the `-` calendar separator or the `.` unix fraction dot.
func isDateTimeBodyByte(b byte) bool {
	return b >= '0' && b <= '9' || b == '-' || b == '.'
}

// isDurationByte reports whether b may appear in a bare duration body: a digit,
// a `.` fraction or an ASCII unit letter (h, m, s, ms, us, ns).
func isDurationByte(b byte) bool {
	return b >= '0' && b <= '9' || b == '.' || b >= 'a' && b <= 'z'
}

// scanDurationLit looks ahead (without consuming) from just after the `dur`
// keyword for a bare duration body: optional inline whitespace, an optional
// sign and then a duration run starting with a digit (e.g. ` 1h25s`). On
// success it returns the body (without the leading gap) and the total byte span
// from the current position through the body; otherwise ok is false and the
// reader is untouched so `dur` stays a plain identifier.
func (s *Scanner) scanDurationLit() (body string, span int, ok bool) {
	src := s.Src
	k := s.Offset
	for k < len(src) && (src[k] == ' ' || src[k] == '\t') {
		k++
	}
	start := k
	if k < len(src) && (src[k] == '-' || src[k] == '+') {
		k++
	}
	if k >= len(src) || src[k] < '0' || src[k] > '9' {
		return "", 0, false
	}
	for k < len(src) && isDurationByte(src[k]) {
		k++
	}
	return string(src[start:k]), k - s.Offset, true
}

// scanDateTimeLit looks ahead (without consuming) from the current position —
// the first digit — for a digit-suffix date/time literal: a dashed date / unix
// body followed by a D/T/U suffix letter that must not be the first rune of an
// identifier (so `0xABCD` and `123Drive` are left alone). On success it returns
// the body (the literal without the suffix), the suffix byte and the total byte
// span of body+suffix; otherwise ok is false and the reader is untouched so the
// caller falls back to plain number scanning.
func (s *Scanner) scanDateTimeLit() (body string, suffix byte, span int, ok bool) {
	src := s.Src
	i := s.Offset
	j := i
	for j < len(src) && isDateTimeBodyByte(src[j]) {
		j++
	}
	if j >= len(src) || j == i {
		return "", 0, 0, false
	}
	switch src[j] {
	case 'D', 'T', 't', 'U':
	default:
		return "", 0, 0, false
	}
	if j+1 < len(src) && runehelper.IsIdentifier(rune(src[j+1])) {
		return "", 0, 0, false
	}
	return string(src[i:j]), src[j], j + 1 - i, true
}

// scanCodeStr scans a `code … end` code-string literal whose `code` keyword
// starts at codeStart (already consumed: s.Ch is the first character after it).
// Two forms are accepted:
//
//	code
//	    <body lines>
//	end
//
// and the single-line form `code <body> end`. For the block form the closing
// fence is the first line sharing the opening line's indentation whose only word
// is `end`; deeper-indented `end`s belong to the body. It returns the full
// source literal (`code … end`), advancing the reader past the closing `end`.
// ok is false (reader untouched) when no fence is found, so a bare `code`
// identifier is left alone.
func (s *Scanner) scanCodeStr(codeStart int) (lit string, ok bool) {
	const kw, end = "code", "end"
	src := s.Src

	// Leading whitespace of the line that contains `code`.
	lineStart := codeStart
	for lineStart > 0 && src[lineStart-1] != '\n' {
		lineStart--
	}
	indentEnd := lineStart
	for indentEnd < codeStart && isInlineSpace(src[indentEnd]) {
		indentEnd++
	}
	indentStr := string(src[lineStart:indentEnd])

	// Skip the inline whitespace after `code` to classify block vs inline.
	i := codeStart + len(kw)
	j := i
	for j < len(src) && isInlineSpace(src[j]) {
		j++
	}

	var closeEnd int
	if j < len(src) && src[j] == '\n' {
		// Block form: a closing fence line == indentStr + "end".
		for off := j + 1; off <= len(src); {
			lineEnd := off
			for lineEnd < len(src) && src[lineEnd] != '\n' {
				lineEnd++
			}
			if isCodeStrFence(src, off, lineEnd, indentStr) {
				closeEnd = off + len(indentStr) + len(end)
				ok = true
				break
			}
			if lineEnd >= len(src) {
				break
			}
			off = lineEnd + 1
		}
	} else {
		// Inline form: code <body> end on a single line.
		for k := i; k < len(src) && src[k] != '\n'; k++ {
			if k > i && isInlineSpace(src[k-1]) &&
				k+len(end) <= len(src) && string(src[k:k+len(end)]) == end &&
				(k+len(end) == len(src) || isWordBoundary(src[k+len(end)])) {
				closeEnd = k + len(end)
				ok = true
				break
			}
		}
	}
	if !ok {
		return "", false
	}

	lit = string(src[codeStart:closeEnd])
	for s.Offset < closeEnd && s.Ch >= 0 {
		s.Next()
	}
	return lit, true
}

// isCodeStrFence reports whether src[off:lineEnd] is a `code … end` closing
// fence line: the opening indentation followed by `end` and only trailing
// inline whitespace.
func isCodeStrFence(src []byte, off, lineEnd int, indentStr string) bool {
	line := string(src[off:lineEnd])
	if len(line) < len(indentStr)+len("end") || line[:len(indentStr)] != indentStr {
		return false
	}
	rest := line[len(indentStr):]
	for len(rest) > 0 && isInlineSpace(rest[len(rest)-1]) {
		rest = rest[:len(rest)-1]
	}
	return rest == "end"
}

func isInlineSpace(b byte) bool { return b == ' ' || b == '\t' }

// isWordBoundary reports whether b ends the `end` fence word: anything that is
// not an identifier continuation character (so a trailing space, newline, `)`,
// etc. all close the inline form).
func isWordBoundary(b byte) bool { return !runehelper.IsIdentifier(rune(b)) }

// ScanNow returns a token, token literal and its position.
func (s *Scanner) ScanNow() (t PToken) {
	t.Pos = source.MustFileSetPos(s.File, s.Offset)

	if s.Ch == -1 {
		if s.InsertSemi {
			s.InsertSemi = false // EOF consumed
			t.Literal = "\n"
			t.Token = token.Semicolon
			return t
		}
		return PToken{TokenLit: node.TokenLit{Token: token.EOF, Pos: t.Pos}}
	}

	if s.mode.Has(ScanMixed) && s.Ch != -1 {
		start := s.Offset
		if s.InCode {
			var removeLeftSpace, removeAllSpace bool
			s.SkipWhitespace()
			switch s.Ch {
			case '\'', '"', '`':
				// ignore quotes
				goto do
			case '-':
				// test if remove spaces before end delimiter: `--END` (strip all)
				// or `-END` (strip blanks, keep a boundary newline)
				if s.Offset+1 < len(s.Src) && s.Src[s.Offset+1] == '-' && s.MixedCodeEnds(2) {
					s.Next()
					s.Next()
					removeAllSpace = true
				} else if s.MixedCodeEnds(1) {
					s.Next()
					removeLeftSpace = true
				}
			}

			if s.MixedCodeEnds(0) {
				s.InCode = false

				t.Token = token.MixedCodeEnd
				t.Literal = string(s.MixedDelimiter.End)
				t.Pos = source.MustFileSetPos(s.File, s.Offset)
				t.Set("remove-spaces", removeLeftSpace)
				t.Set("remove-spaces-all", removeAllSpace)

				s.Next()
				s.NextC(len(s.MixedDelimiter.End) - 1)

				if s.ToText {
					t.Token = token.MixedValueEnd
					s.ToText = false
				}
				return
			}
		} else {
			readText := func() {
				t.Token = token.MixedText
				t.Pos = source.MustFileSetPos(s.File, start)

				if s.Offset > start {
					t.Literal = string(s.Src[start:s.Offset])
				}
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

						if !s.mode.Has(ScanConfigDisabled) {
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
					s.HandleMixed(&start, func() *PToken {
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
	t.Pos = source.MustFileSetPos(s.File, s.Offset)

	insertSemi := false

	// determine token value
	switch ch := s.Ch; {
	case runehelper.IsIdentifierLetter(ch):
		identStart := s.Offset
		t.Literal = s.ScanIdentifier()
		t.Token = token.Lookup(t.Literal)
		// A `code` keyword immediately followed by whitespace opens a `code … end`
		// code-string literal (its body becomes a Str and is NOT parsed). It only
		// triggers when a matching `end` fence is present, so a plain `code`
		// identifier without one is unaffected.
		if t.Token == token.Ident && t.Literal == "code" &&
			(s.Ch == ' ' || s.Ch == '\t' || s.Ch == '\n') {
			if lit, ok := s.scanCodeStr(identStart); ok {
				t.Token = token.CodeStr
				t.Literal = lit
				insertSemi = true
				break
			}
		}
		// A single-letter b/h identifier immediately followed by a string
		// delimiter is a bytes literal prefix: b"..." (raw bytes) or h"..."
		// (hex). The underlying string may be a regular string, raw string,
		// heredoc or raw heredoc.
		if t.Token == token.Ident && (t.Literal == "b" || t.Literal == "h") &&
			(s.Ch == '"' || s.Ch == '`') {
			prefix := t.Literal
			delim := s.Ch
			s.Next() // consume the opening delimiter
			var ishd bool
			switch delim {
			case '"':
				t.Token = token.String
				if t.Literal, ishd = s.ScanString(); ishd {
					t.Token = token.Heredoc
				}
			case '`':
				t.Token = token.RawString
				if t.Literal, ishd = s.ScanRawString(); ishd {
					t.Token = token.RawHeredoc
				}
			}
			t.Set(bytesLitPrefixKey, prefix)
			insertSemi = true
			break
		}
		// The `dur` keyword followed by a bare Go duration string is a duration
		// literal: `dur 1h25s`, `dur 500ms`, `dur 1.5h`. It only triggers when a
		// duration body (a digit) follows, so a `dur` variable is unaffected
		// unless it is immediately followed by a number.
		if t.Token == token.Ident && t.Literal == "dur" {
			if body, span, ok := s.scanDurationLit(); ok {
				s.NextC(span) // consume the gap + duration body
				t.Token = token.String
				t.Literal = body
				t.Set(durationLitKey, true)
				insertSemi = true
				break
			}
		}
		switch t.Literal {
		case "begin":
			t.Token = token.LBrace
		case "end":
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
		// digit-suffix date/time literals: 20260131D (date), 235955T (time),
		// 1781609136U (unix time), optionally with a `_` date/time separator,
		// a `.` fraction and a `Z<location>` zone. The suffix letter must end
		// the run and not be the first rune of an identifier, so `123Drive`
		// stays a number followed by an identifier.
		if body, suffix, span, ok := s.scanDateTimeLit(); ok {
			t.Token = token.String
			t.Literal = body
			s.NextC(span) // consume the body and the suffix letter
			t.Set(dateTimeLitKey, rune(suffix))
		} else {
			t.Token, t.Literal = s.ScanNumber(false)
		}
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
			s.CallEOFHandlers()
		case '\n':
			// we only reach here if s.InsertSemi was set in the first place
			s.InsertSemi = false // newline consumed
			t.Literal = "\n"
			t.Token = token.Semicolon
			return
		case '"':
			insertSemi = true
			t.Token = token.String
			var ishd bool
			if t.Literal, ishd = s.ScanString(); ishd {
				t.Token = token.Heredoc
			}
		case '\'':
			insertSemi = true
			if s.mode.Has(ScanCharAsString) {
				t.Token = token.String
				t.Literal = s.ScanStringDelimiter('\'')
			} else {
				t.Token = token.Char
				t.Literal = s.ScanRune()
			}
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
			if s.Ch == '.' {
				s.Next()
				t.Token = token.DotDot
			} else if s.Ch == '|' {
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
					t.Token = token.Nullich
				}
			default:
				t.Token = token.Question
			}
		case ';':
			t.Token = token.Semicolon
			t.Literal = ";"
		case '(':
			t.Token = token.LParen
			t.Literal = string(ch)
			s.ParenCount++
		case ')':
			insertSemi = true
			t.Token = token.RParen
			t.Literal = string(ch)
			s.ParenCount--
		case '[':
			t.Token = token.LBrack
			t.Literal = string(ch)
			s.BreacksCount++
		case ']':
			insertSemi = true
			t.Token = token.RBrack
			t.Literal = string(ch)
			s.BreacksCount--
		case '{':
			t.Token = token.LBrace
			t.Literal = string(ch)
			s.BraceCount++
		case '}':
			insertSemi = true
			t.Token = token.RBrace
			t.Literal = string(ch)
			s.BraceCount--
		case '+':
			t.Token = s.Switch4(token.Add, token.AddAssign, '+', token.Inc, token.IncAssign)
			if t.Token == token.Inc {
				insertSemi = true
			}
		case '-':
			t.Token = s.Switch4(token.Sub, token.SubAssign, '-', token.Dec, token.DecAssign)
			if t.Token == token.Dec {
				insertSemi = true
			}
		case '*':
			if s.Ch == '*' {
				s.Next()
				t.Token = s.Switch2(token.Pow, token.PowAssign)
			} else {
				t.Token = s.Switch2(token.Mul, token.MulAssign)
			}
		case '/':
			if s.Ch == '/' || s.Ch == '*' || s.Ch == '?' {
				// comment (`//`, `/* */`) or the single doc comment `/?`
				if s.InsertSemi && s.FindLineEnd() {
					// reset position to the beginning of the comment
					s.Ch = '/'
					s.Offset = source.MustFileOffset(s.File, t.Pos)
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
			} else if !s.InsertSemi && s.LooksLikeRegex() {
				// operand position with a closing '/' on the same line: regex
				insertSemi = true
				t.Token = token.Regex
				t.Literal = s.ScanRegex()
			} else {
				t.Token = s.Switch2(token.Quo, token.QuoAssign)
			}
		case '%':
			if s.Ch == '%' {
				s.Next()
				t.Token = s.Switch2(token.DoubleMod, token.DoubleModAssign)
			} else {
				t.Token = s.Switch2(token.Rem, token.RemAssign)
			}
		case '^':
			t.Token = s.Switch2(token.Xor, token.XorAssign)
		case '<':
			if s.Ch == '<' {
				s.Next()
				if s.Ch == '<' {
					s.Next()
					t.Token = s.Switch2(token.TripleLess, token.TripleLessAssign)
				} else {
					t.Token = s.Switch2(token.Shl, token.ShlAssign)
				}
			} else {
				t.Token = s.Switch2(token.Less, token.LessEq)
			}
		case '>':
			if s.Ch == '>' {
				s.Next()
				if s.Ch == '>' {
					s.Next()
					t.Token = s.Switch2(token.TripleGreater, token.TripleGreaterAssign)
				} else {
					t.Token = s.Switch2(token.Shr, token.ShrAssign)
				}
			} else {
				t.Token = s.Switch2(token.Greater, token.GreaterEq)
			}
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
			if !s.mode.Has(ScanConfigDisabled) {
				// at line start
				if s.Offset == 1 || s.Src[s.Offset-2] == '\n' {
					if l := s.PeekNoSingleSpaceEq("gad:", 0); l > 0 {
						return s.scanConfig(t.Pos, l+1)
					}
				}
			}

			if s.Ch == '(' {
				insertSemi = true
				s.Next()
				t.Literal = "#" + s.ScanStringDelimiter(')')
				t.Token = token.Symbol
			} else if runehelper.IsIdentifier(s.Ch) {
				insertSemi = true
				t.Literal = "#" + s.ScanIdentifier()
				t.Token = token.Symbol
			} else {
				t.Token = token.Template
			}
		default:
			// next reports unexpected BOMs - don't repeat
			if ch != source.BOM {
				if s.ScanHandler != nil {
					var ok bool
					if t, insertSemi, ok = s.ScanHandler(ch); ok {
						goto done
					}
				}
				s.Error(source.MustFileOffset(s.File, t.Pos),
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
