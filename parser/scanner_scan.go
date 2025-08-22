package parser

import (
	"fmt"

	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/runehelper"
	"github.com/gad-lang/gad/token"
)

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

	if s.mode.Has(ScanMixed) && s.Ch != -1 {
		start := s.Offset
		if s.InCode {
			var removeLeftSpace bool
			s.SkipWhitespace()
			switch s.Ch {
			case '\'', '"', '`':
				// ignore quotes
				goto do
			case '-':
				// test if remove spaces before end delimiter `-END_DELIMITER`
				if s.MixedCodeEnds(1) {
					s.Next()
					removeLeftSpace = true
				}
			}

			if s.MixedCodeEnds(0) {
				s.InCode = false

				t.Token = token.MixedCodeEnd
				t.Literal = string(s.MixedDelimiter.End)
				t.Pos = s.File.FileSetPos(s.Offset)
				t.Set("remove-spaces", removeLeftSpace)

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
				t.Pos = s.File.FileSetPos(start)

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
		case "do", "then":
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
			t.Literal = s.ScanString()
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
			t.Token = s.Switch3(token.Add, token.AddAssign, '+', token.Inc)
			if t.Token == token.Inc {
				insertSemi = true
			}
		case '-':
			t.Token = s.Switch3(token.Sub, token.SubAssign, '-', token.Dec)
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
			if !s.mode.Has(ScanConfigDisabled) {
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
			if ch != source.BOM {
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
