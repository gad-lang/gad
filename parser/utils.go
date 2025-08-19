package parser

import (
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"
)

var asciiSpace = [256]uint8{'\t': 1, '\n': 1, '\v': 1, '\f': 1, '\r': 1, ' ': 1}

// TrimSpace returns a slice of the string s, with all leading
// and trailing white space removed, as defined by Unicode.
func TrimSpace(left, right bool, s string) string {
	start := 0
	if left {
		// Fast path for ASCII: look for the first ASCII non-space byte
		for ; start < len(s); start++ {
			c := s[start]
			if c >= utf8.RuneSelf {
				// If we run into a non-ASCII byte, fall back to the
				// slower unicode-aware method on the remaining bytes
				return strings.TrimFunc(s[start:], unicode.IsSpace)
			}
			if asciiSpace[c] == 0 {
				break
			}
		}
	}

	stop := len(s)
	if right {
		// Now look for the first ASCII non-space byte from the end
		for ; stop > start; stop-- {
			c := s[stop-1]
			if c >= utf8.RuneSelf {
				// start has been already trimmed above, should trim end only
				return strings.TrimRightFunc(s[start:stop], unicode.IsSpace)
			}
			if asciiSpace[c] == 0 {
				break
			}
		}
	}

	// At this point s[start:stop] starts and ends with an ASCII
	// non-space bytes, so we're done. Non-ASCII cases have already
	// been handled above.
	return s[start:stop]
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

func isType(v any, typ ...any) (ok bool) {
	ctyp := reflect.TypeOf(v)
	for _, t := range typ {
		if reflect.TypeOf(t) == ctyp {
			return true
		}
	}
	return false
}
