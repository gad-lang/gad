package source

import "testing"

// TestReaderTrailingCR guards against an index-out-of-range panic when the source
// ends with a carriage return: the CRLF-skipping loop must stop at the end of the
// buffer. Windows (CRLF) checkouts, or a line split off a CRLF stream, can leave a
// trailing '\r'.
func TestReaderTrailingCR(t *testing.T) {
	cases := []string{
		"abc\r",      // lone trailing CR (the crash case)
		"a\r\nb\r",   // CRLF then trailing CR
		"\r",         // just a CR
		"x\r\r",      // doubled trailing CR
		"a\r\nb\r\n", // well-formed CRLF (no trailing bare CR)
	}
	for _, src := range cases {
		func(s string) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("panic reading %q: %v", s, r)
				}
			}()
			fs := NewFileSet()
			f := fs.AppendFileData("t", []byte(s))
			r := NewFileReader(f)
			for r.Ch != -1 {
				r.Next()
			}
		}(src)
	}
}
