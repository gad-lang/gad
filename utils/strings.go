package utils

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

var asciiSpace = [256]uint8{'\t': 1, '\n': 1, '\v': 1, '\f': 1, '\r': 1, ' ': 1}

// TrimStringSpace returns a slice of the string s, with all leading
// and trailing white space removed, as defined by Unicode.
func TrimStringSpace(s string, starts, ends bool) (start int, _ string) {
	// Fast path for ASCII: look for the first ASCII non-space byte
	if starts {
		for ; start < len(s); start++ {
			c := s[start]
			if c >= utf8.RuneSelf {
				// If we run into a non-ASCII byte, fall back to the
				// slower unicode-aware method on the remaining bytes
				return start, strings.TrimFunc(s[start:], unicode.IsSpace)
			}
			if asciiSpace[c] == 0 {
				break
			}
		}
	}

	// Now look for the first ASCII non-space byte from the end
	stop := len(s)
	if ends {
		for ; stop > start; stop-- {
			c := s[stop-1]
			if c >= utf8.RuneSelf {
				// start has been already trimmed above, should trim end only
				return start, strings.TrimRightFunc(s[start:stop], unicode.IsSpace)
			}
			if asciiSpace[c] == 0 {
				break
			}
		}
	}

	// At this point s[start:stop] starts and ends with an ASCII
	// non-space bytes, so we're done. Non-ASCII cases have already
	// been handled above.
	return start, s[start:stop]
}
