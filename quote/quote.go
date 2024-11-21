package quote

import (
	"strings"
)

type scanner struct {
	src []byte
	r   byte
	pos int
}

func newScanner(src []byte) *scanner {
	return &scanner{src: src}
}

func (s *scanner) next() {
	if s.pos >= len(s.src) {
		s.r = 0
		return
	}
	s.r = s.src[s.pos]
	s.pos++
}

func (s *scanner) nextN(n int) {
	for i := 0; i < n; i++ {
		s.next()
	}
}

func (s *scanner) peek() byte {
	if s.pos >= len(s.src) {
		return 0
	}
	return s.src[s.pos]
}

func (s *scanner) peekN(n int) []byte {
	if s.pos+n > len(s.src) {
		return nil
	}
	return s.src[s.pos : s.pos+n]
}

func (s *scanner) peekV(v []byte) bool {
	left := s.peekN(len(v))
	if left == nil {
		return false
	}

	for i, r := range v {
		if r != left[i] {
			return false
		}
	}

	return true
}

func (s *scanner) quoted() (ok bool) {
	for i := s.pos; i >= 0; i-- {
		if s.src[i] != '\\' {
			return
		}
		ok = !ok
	}
	return
}

func Quote(s, quote string) string {
	var (
		scan = newScanner([]byte(s))
		qs   = []byte(quote)
		qs0  = qs[0]
		qs2  = qs[1:]
		out  strings.Builder
	)

	scan.next()

	out.WriteString(quote)
loop:
	for {
		switch scan.r {
		case 0:
			break loop
		case qs0:
			out.WriteByte('\\')
			if len(qs2) > 0 && scan.peekV(qs2) {
				scan.nextN(len(qs2))
			}
			out.WriteString(quote)
		default:
			out.WriteByte(scan.r)
		}
		scan.next()
	}
	out.WriteString(quote)
	return out.String()
}

func Unquote(s, quote string) string {
	var (
		qb   = []byte(quote)
		scan = newScanner([]byte(s[len(quote) : len(s)-len(quote)]))
		out  strings.Builder
	)

	scan.next()
loop:
	for {
		switch scan.r {
		case 0:
			break loop
		case '\\':
			if scan.peekV(qb) {
				scan.nextN(len(qb))
				out.Write(qb)
			} else {
				out.WriteByte(scan.r)
			}
		default:
			out.WriteByte(scan.r)
		}
		scan.next()
	}
	return out.String()
}
