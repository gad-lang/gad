package runehelper

import (
	"unicode"
	"unicode/utf8"
)

func IsLetter(ch rune) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' ||
		ch >= utf8.RuneSelf && unicode.IsLetter(ch)
}

func IsIdentifier(ch rune) bool {
	return '0' <= ch && ch <= '9' || IsIdentifierLetter(ch) || unicode.IsDigit(ch)
}

func IsLetterOrDigitRunes(chs []rune) bool {
	for _, r := range chs {
		if !IsIdentifier(r) {
			return false
		}
	}
	return true
}

func IsIdentifierLetter(ch rune) bool {
	return ch == '$' || ch == '_' || IsLetter(ch)
}

func IsDigit(ch rune) bool {
	return '0' <= ch && ch <= '9' ||
		ch >= utf8.RuneSelf && unicode.IsDigit(ch)
}

func IsIdentifierRunes(s []rune) bool {
	if !IsIdentifierLetter(s[0]) {
		return false
	}
	s = s[1:]
	for _, r := range s {
		if !IsIdentifierLetter(r) && !IsDigit(r) {
			return false
		}
	}
	return true
}

func IsSpace(ch rune) bool {
	return ch == '\n' || IsSingleSpace(ch)
}

func IsSingleSpace(ch rune) bool {
	return ch == ' ' || ch == '\t'
}

func DigitVal(ch rune) int {
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
