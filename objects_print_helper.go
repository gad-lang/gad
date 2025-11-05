package gad

import (
	"fmt"
	"strconv"

	"github.com/gad-lang/gad/runehelper"
)

func PrintDict(state *PrinterState, l int, key, value func(i int) (Object, error)) (err error) {
	return PrintPairs(state, l, []byte{'{'}, []byte{'}'}, []byte{':', ' '}, []byte{','}, key, value)
}

func PrintPairs(state *PrinterState, l int, open, close, keySep, itemSep []byte, getKey, getValue func(i int) (Object, error)) (err error) {
	if _, err = state.Write(open); err != nil {
		return
	}

	defer func() {
		if err == nil {
			if state.Indented() && !state.SkipDepth() {
				state.PrintLine()
				if l > 0 {
					state.PrintIndent()
				}
			}
			_, err = state.Write(close)
		}
	}()

	if state.SkipDepth() {
		_, err = state.Write([]byte("…"))
		return
	}

	defer state.Enter()()

	var key, value Object

	do := func(i int) {
		if key, err = getKey(i); err != nil {
			return
		}

		switch t := key.(type) {
		case Str:
			var s = string(t)
			if !runehelper.IsIdentifierOrDigitRunes([]rune(s)) {
				s = strconv.Quote(s)
			}
			_, err = state.Write([]byte(s))
		case RawStr:
			_, err = state.Write([]byte(t.Quoted()))
		case Int, Uint, Float, Decimal, Bool, Flag:
			_, err = state.Write([]byte(t.ToString()))
		default:
			err = Print(state, t)
		}

		if err != nil {
			return
		}

		if value, err = getValue(i); value != nil {
			_, _ = state.Write(keySep)
			switch t := value.(type) {
			case Str:
				_, err = state.Write([]byte(strconv.Quote(string(t))))
			case RawStr:
				_, err = state.Write([]byte(t.Quoted()))
			default:
				err = Print(state, value)
			}
		}
	}

	if state.Indented() {
		itemSep = append(itemSep, '\n')
		state.PrintLine()

		old := do
		do = func(i int) {
			state.PrintIndent()
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
			if _, err = state.Write(itemSep); err != nil {
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

func PrintArray(state *PrinterState, l int, get func(i int) (Object, error)) (err error) {
	return PrintValues(state, l, []byte{'['}, []byte{']'}, []byte{','}, get)
}

func PrintValues(state *PrinterState, l int, open, close, itemSep []byte, get func(i int) (Object, error)) (err error) {
	if _, err = state.Write(open); err != nil {
		return
	}

	defer func() {
		if err == nil {
			if state.Indented() && !state.SkipDepth() {
				state.PrintLine()
				if l > 0 {
					state.PrintIndent()
				}
			}
			_, err = state.Write(close)
		}
	}()

	if state.SkipDepth() {
		_, err = state.Write([]byte("…"))
		return
	}

	defer state.Enter()()

	var (
		item Object

		indexes = PrintStateOptionsGetIndexes(state)
		doItem  = func(i int, item Object) {
			switch t := item.(type) {
			case Str:
				_, err = state.Write([]byte(strconv.Quote(string(t))))
			case RawStr:
				_, err = state.Write([]byte(t.Quoted()))
			default:
				err = Print(state, item)
			}
		}
	)

	if indexes {
		old := doItem
		doItem = func(i int, item Object) {
			_, _ = fmt.Fprintf(state, "%d 🠆 ", i)
			old(i, item)
		}
	}

	do := func(i int) {
		if item, err = get(i); err == nil {
			doItem(i, item)
		}
	}

	if state.Indented() {
		itemSep = append(itemSep, '\n')
		state.PrintLine()

		old := do
		do = func(i int) {
			state.PrintIndent()
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
			if _, err = state.Write(itemSep); err != nil {
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
