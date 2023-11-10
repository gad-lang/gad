package utils

import (
	"unicode"
	"unicode/utf8"
)

func IsLetter(ch rune) bool {
	return ch == '$' || 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_' ||
		ch >= utf8.RuneSelf && unicode.IsLetter(ch)
}

func IsDigit(ch rune) bool {
	return '0' <= ch && ch <= '9' ||
		ch >= utf8.RuneSelf && unicode.IsDigit(ch)
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
