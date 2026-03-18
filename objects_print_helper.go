package gad

import (
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/gad-lang/gad/repr"
	"github.com/gad-lang/gad/runehelper"
)

func (s *PrinterState) WrapRepr(o Object) func() {
	switch o.(type) {
	case *CompiledFunction, *ReflectType, *Func:
		return func() {}
	}

	var name string
	if t, _ := o.(ObjectType); t != nil {
		name = t.FullName()
	} else {
		name = ReprTypeName(o)

		switch t := o.(type) {
		case *Buffer:
			name += "(" + humanize.Bytes(uint64(t.Len())) + ")"
		}
	}
	return s.WrapReprString(name)
}

func (s *PrinterState) WrapReprString(str string) func() {
	if runes := []rune(str); runes[0] == '‹' {
		if runes[len(runes)-1] == '›' {
			str = string(runes[1 : len(runes)-1])
		}
	}
	s.WriteString(repr.QuotePrefix + str + ": ")
	return func() {
		s.WriteString(repr.QuoteSufix)
	}
}

func (s *PrinterState) WrapIndentedReprString(str string) func() {
	if s.Indented() {
		if runes := []rune(str); runes[0] == '‹' {
			if runes[len(runes)-1] == '›' {
				str = string(runes[1 : len(runes)-1])
			}
		}
		s.WriteString(repr.QuotePrefix + str + ": ")
		return func() {
			s.PrintLineIndent()
			s.WriteString(repr.QuoteSufix)
		}
	}
	return s.WrapReprString(str)
}

func (s *PrinterState) Repr(o Object) error {
	defer s.WrapRepr(o)()
	return s.Print(o)
}

func (s *PrinterState) WithRepr(cb func(s *PrinterState) error) error {
	if !s.IsRepr {
		s.IsRepr = true
		defer func() { s.IsRepr = false }()
	}
	return cb(s)
}

func (s *PrinterState) WithoutRepr(cb func(s *PrinterState) error) error {
	if s.IsRepr {
		s.IsRepr = false
		defer func() { s.IsRepr = true }()
	}
	return cb(s)
}

func (s *PrinterState) PrintKey(o Object) (err error) {
	return s.PrintKeySafe(false, o)
}
func (s *PrinterState) PrintKeySafe(safe bool, o Object) (err error) {
	return s.DoVisit(o, func() (err error) {
		switch t := o.(type) {
		case Str:
			if !safe && s.IsRepr {
				defer s.WrapRepr(o)()
			}
			var str = string(t)
			if !runehelper.IsIdentifierOrDigitRunes([]rune(str)) {
				str = strconv.Quote(str)
			}
			_, err = s.Write([]byte(str))
		case RawStr:
			if !safe && s.IsRepr {
				defer s.WrapRepr(o)()
			}
			_, err = s.Write([]byte(t.Quoted()))
		case Int, Uint, Float, Decimal, Bool, Flag:
			if !safe && s.IsRepr {
				defer s.WrapRepr(o)()
			}
			_, err = s.Write([]byte(t.ToString()))
		case Bytes:
			if !safe && s.IsRepr {
				defer s.WrapRepr(o)()
			}

			err = t.ToStringF(s)
		case Printer:
			err = t.Print(s)
		default:
			if s.VM == nil {
				if s.IsRepr {
					defer s.WrapRepr(o)()
				}

				err = s.WriteString(o.ToString())
			} else {
				s.stack.fallback = true
				err = Print(s, t)
			}
		}
		return
	})
}

func (s *PrinterState) Print(o Object) (err error) {
	return s.DoVisit(o, func() (err error) {
	try:
		switch t := o.(type) {
		case Str:
			if s.IsRepr {
				defer s.WrapRepr(o)()
				_, err = s.Write([]byte(t.Quoted()))
			} else if s.options.IsQuoteStr() {
				_, err = s.Write([]byte(t.Quoted()))
			} else {
				_, err = s.Write([]byte(t))
			}
		case RawStr:
			if s.IsRepr {
				defer s.WrapRepr(o)()
				_, err = s.Write([]byte(t.Quoted()))
			} else if s.options.IsQuoteStr() {
				_, err = s.Write([]byte(t.Quoted()))
			} else {
				_, err = s.Write([]byte(t))
			}
		case Char, Int, Uint, Float, Decimal, Bool, Flag, *NilType:
			if s.IsRepr {
				defer s.WrapRepr(o)()
			}
			_, err = s.Write([]byte(t.ToString()))
		case Bytes:
			if s.IsRepr {
				defer s.WrapRepr(o)()
				err = t.ToStringF(s)
			} else if s.options.IsBytesToHex() {
				err = t.ToStringF(s)
			} else {
				_, err = s.Write(t)
			}
		case *Buffer:
			if s.IsRepr {
				defer s.WrapRepr(o)()
				s.WriteString("h\"")
				if _, err = hex.NewEncoder(s).Write(t.Bytes()); err == nil {
					err = s.WriteString("\"")
				}
			} else {
				_, err = s.Write(t.Bytes())
			}
		case Printer:
			err = t.Print(s)
		case *Error:
			o = Str(o.ToString())
			goto try
		default:
			if s.VM == nil {
				if s.IsRepr {
					defer s.WrapRepr(o)()
				}
				err = s.WriteString(o.ToString())
			} else {
				s.stack.fallback = true
				err = Print(s, t)
			}
		}
		return
	})
}

func (s *PrinterState) PrintDict(l int, key, value func(i int) (Object, error)) (err error) {
	return s.PrintPairs(l, true, []byte{'{'}, []byte{'}'}, []byte{':', ' '}, []byte{','}, key, value)
}

func (s *PrinterState) PrintPairs(l int, safeKey bool, open, close, keySep, itemSep []byte, getKey, getValue func(i int) (Object, error)) (err error) {
	if _, err = s.Write(open); err != nil {
		return
	}

	defer func() {
		if err == nil {
			if s.Indented() && !s.SkipDepth() {
				if l > 0 {
					s.PrintLine()
					s.PrintIndent()
				}
			}
			_, err = s.Write(close)
		}
	}()

	if s.SkipDepth() {
		_, err = s.Write([]byte("…"))
		return
	}

	if l == 0 {
		return
	}

	defer s.Enter()()

	var key, value Object

	do := func(i int) {
		if key, err = getKey(i); err != nil {
			return
		}

		if err = s.PrintKeySafe(safeKey, key); err != nil {
			return
		}

		if value, err = getValue(i); value != nil && err == nil {
			_, _ = s.Write(keySep)
			err = s.Print(value)
		}
		if err != nil {
			return
		}
	}

	if s.Indented() {
		itemSep = append(itemSep, '\n')
		s.PrintLine()

		old := do
		do = func(i int) {
			s.PrintIndent()
			old(i)
		}
	} else {
		itemSep = append(itemSep, ' ')
		old := do
		do = func(i int) {
			old(i)
		}
	}

	for i, last := 0, l-1; i <= last; i++ {
		if i > 0 {
			if _, err = s.Write(itemSep); err != nil {
				return
			}
		}
		do(i)
		if err != nil {
			return
		}
	}

	return
}

func (s *PrinterState) PrintArray(l int, get func(i int) (Object, error)) (err error) {
	return s.PrintValues(l, []byte{'['}, []byte{']'}, []byte{','}, get)
}

func (s *PrinterState) PrintValues(l int, open, close, itemSep []byte, get func(i int) (Object, error)) (err error) {
	if _, err = s.Write(open); err != nil {
		return
	}

	if l == 0 {
		_, err = s.Write(close)
		return
	}

	defer func() {
		if err == nil {
			if s.Indented() && !s.SkipDepth() {
				s.PrintLine()
				if l > 0 {
					s.PrintIndent()
				}
			}
			_, err = s.Write(close)
		}
	}()

	if s.SkipDepth() {
		_, err = s.Write([]byte("…"))
		return
	}

	defer s.Enter()()

	var (
		item Object

		indexes, _ = s.options.Indexes()
		doItem     = func(_ int, item Object) {
			err = s.Print(item)
		}
	)

	if indexes {
		old := doItem
		doItem = func(i int, item Object) {
			_, _ = fmt.Fprintf(s, "%d 🠆 ", i)
			old(i, item)
		}
	}

	do := func(i int) {
		if item, err = get(i); err == nil {
			doItem(i, item)
		}
	}

	if s.Indented() {
		itemSep = append(itemSep, '\n')
		s.PrintLine()

		old := do
		do = func(i int) {
			s.PrintIndent()
			old(i)
		}
	} else {
		itemSep = append(itemSep, ' ')
		old := do
		do = func(i int) {
			old(i)
		}
	}

	for i, last := 0, l-1; i <= last; i++ {
		if i > 0 {
			if _, err = s.Write(itemSep); err != nil {
				return
			}
		}
		do(i)
		if err != nil {
			return
		}
	}

	return
}

type PrintBuilder struct {
	state *PrinterState
}

func NewPrintBuilder(vm *VM, o ...PrinterStateOption) *PrintBuilder {
	return &PrintBuilder{state: NewPrinterState(vm, nil, o...)}
}

func (b *PrintBuilder) Options(f func(opts PrinterStateOptions)) *PrintBuilder {
	f(b.state.options)
	b.state.Update()
	return b
}

func (b *PrintBuilder) State(f func(s *PrinterState)) *PrintBuilder {
	f(b.state)
	return b
}

func (b *PrintBuilder) Print(w io.Writer, o ...Object) (err error) {
	b.state.writer = w
	return Print(b.state, o...)
}

func (b *PrintBuilder) String(o ...Object) (_ string, err error) {
	var w strings.Builder
	err = b.Print(&w, o...)
	return w.String(), err
}

func (b *PrintBuilder) MustString(o ...Object) string {
	s, err := b.String(o...)
	if err != nil {
		panic(err)
	}
	return s
}
