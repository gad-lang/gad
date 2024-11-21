package source

import "bytes"

type StartEndDelimiter struct {
	Start []rune
	End   []rune
}

func (m *StartEndDelimiter) String() string {
	return string(m.Start) + " - " + string(m.End)
}

func (m *StartEndDelimiter) IsZero() bool {
	return len(m.Start) == 0 && len(m.End) == 0
}

func (m *StartEndDelimiter) Strings() (start, end string) {
	return string(m.Start), string(m.End)
}

func (m *StartEndDelimiter) Starts(r rune, b []byte) bool {

	if m.Start[0] == r {
		if len(m.Start) > 1 {
			return bytes.HasPrefix(b, []byte(string(m.Start[1:])))
		}
		return true
	}
	return false
}

func (m *StartEndDelimiter) Ends(b []byte) bool {
	return bytes.HasPrefix(b, []byte(string(m.End)))
}

func (m *StartEndDelimiter) EndsR(r rune, b []byte) bool {
	if m.End[0] == r {
		if len(m.End) > 1 {
			return bytes.HasPrefix(b, []byte(string(m.End[1:])))
		}
		return true
	}
	return false
}

func (m *StartEndDelimiter) WrapString(s string) string {
	return string(m.Start) + s + string(m.End)
}
