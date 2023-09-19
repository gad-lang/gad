package parser

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

var asciiSpace = [256]uint8{'\t': 1, '\n': 1, '\v': 1, '\f': 1, '\r': 1, ' ': 1}

// TrimSpace returns a slice of the string s, with all leading
// and trailing white space removed, as defined by Unicode.
func trimSpace(left, right bool, s string) string {
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
