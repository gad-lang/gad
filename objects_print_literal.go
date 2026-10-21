package gad

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// The forms a value prints in (#25), each by its Print:
//
//   - its text — `str(v)`, a value printed alone: a str as it is, `2` for 2u;
//   - its literal — inside a collection (an array's, a dict's, a key-value
//     array's items: the state's depth), or asked for (`quoteStr`): what reads
//     back as an equal value of the same type — `"a"`, `2u`, `2.0`, `'a'`;
//   - its repr — `repr(v)`: the literal wrapped with its type, `‹uint: 2u›`.

// IsLiteral reports whether a value prints as its literal: inside a
// collection, or asked for (quoteStr).
func (s *PrinterState) IsLiteral() bool {
	return s.depth > 0 || s.options.IsQuoteStr()
}

// literalOr writes lit as the repr or the literal, else text.
func (s *PrinterState) literalOr(o Object, lit func() string, text string) error {
	if s.IsRepr {
		defer s.WrapRepr(o)()
		return s.WriteString(lit())
	}
	if s.IsLiteral() {
		return s.WriteString(lit())
	}
	return s.WriteString(text)
}

func (o Str) Print(state *PrinterState) error {
	return state.literalOr(o, func() string { return strconv.Quote(string(o)) }, string(o))
}

func (o RawStr) Print(state *PrinterState) error {
	return state.literalOr(o, o.Quoted, string(o))
}

func (o Char) Print(state *PrinterState) error {
	return state.literalOr(o, func() string { return strconv.QuoteRune(rune(o)) }, string(rune(o)))
}

func (o Int) Print(state *PrinterState) error {
	s := o.ToString()
	return state.literalOr(o, func() string { return s }, s)
}

func (o Uint) Print(state *PrinterState) error {
	s := o.ToString()
	return state.literalOr(o, func() string { return s + "u" }, s)
}

func (o Float) Print(state *PrinterState) error {
	s := o.ToString()
	return state.literalOr(o, func() string { return floatLiteral(s) }, s)
}

// floatLiteral is a float's text as a literal of a float: `2.0`, not `2`
// (an int).
func floatLiteral(s string) string {
	if strings.ContainsAny(s, ".eEnN") { // a point, an exponent, NaN, Inf
		return s
	}
	return s + ".0"
}

func (o Decimal) Print(state *PrinterState) error {
	return state.literalOr(o, func() string { return decimalLiteral(o) }, o.ToString())
}

// decimalLiteral is a decimal's literal, its scale kept: `1.50d`.
func decimalLiteral(o Decimal) string {
	d := o.ToGo()
	if e := d.Exponent(); e < 0 {
		return d.StringFixed(-e) + "d"
	}
	return d.String() + "d"
}

func (o Bool) Print(state *PrinterState) error {
	s := o.ToString()
	return state.literalOr(o, func() string { return s }, s)
}

func (o Flag) Print(state *PrinterState) error {
	return state.literalOr(o, func() string {
		if o {
			return "yes"
		}
		return "no"
	}, o.ToString())
}

func (o Bytes) Print(state *PrinterState) (err error) {
	if state.options.IsBytesToHex() {
		if state.IsRepr {
			defer state.WrapRepr(o)()
		}
		return o.ToStringF(state)
	}
	return state.literalOr(o, func() string { return bytesLiteral(o) }, string(o))
}

// bytesLiteral is `bytes("…")`: its content as a str literal (its invalid
// UTF-8 escaped).
func bytesLiteral(o Bytes) string {
	if utf8.Valid(o) {
		return "bytes(" + strconv.Quote(string(o)) + ")"
	}
	return "bytes(" + strconv.QuoteToASCII(string(o)) + ")"
}

// Print writes an error: its text (`error: x`); its literal `error("x")`;
// its repr with its name, message and cause, in detail.
func (o *Error) Print(state *PrinterState) error {
	if !state.IsRepr {
		if state.IsLiteral() && o.Name == "error" {
			return state.WriteString("error(" + strconv.Quote(o.Message) + ")")
		}
		return state.WriteString(o.ToString())
	}
	defer state.WrapReprString("error")()
	if o.Name != "error" {
		state.WriteString(o.Name + " ")
	}
	state.WriteString(strconv.Quote(o.Message))
	if o.Cause != nil {
		state.WriteString("; cause: ")
		if co, ok := o.Cause.(Object); ok {
			return state.Print(co)
		}
		return state.Print(WrapError(o.Cause))
	}
	return nil
}

// Print writes a regexp as its literal, `/a+/` — its text, its literal and
// its repr alike.
func (o *Regexp) Print(state *PrinterState) error {
	if state.IsRepr {
		defer state.WrapRepr(o)()
	}
	return state.WriteString(regexpLiteral(o.Go().String()))
}

// regexpLiteral is `/pattern/`, a `/` of the pattern escaped.
func regexpLiteral(p string) string {
	var b strings.Builder
	b.WriteByte('/')
	for i := 0; i < len(p); i++ {
		switch p[i] {
		case '\\':
			b.WriteByte('\\')
			if i+1 < len(p) {
				i++
				b.WriteByte(p[i])
			}
			continue
		case '/':
			b.WriteByte('\\')
		}
		b.WriteByte(p[i])
	}
	b.WriteByte('/')
	return b.String()
}
