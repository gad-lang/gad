package gad

import (
	"fmt"
	"strconv"
	"strings"
)

func ArrayToString(len int, get func(i int) Object) string {
	var (
		sb   strings.Builder
		last = len - 1
	)
	sb.WriteString("[")

	for i := 0; i <= last; i++ {
		switch v := get(i).(type) {
		case String:
			sb.WriteString(strconv.Quote(v.String()))
		case Char:
			sb.WriteString(strconv.QuoteRune(rune(v)))
		case Bytes:
			sb.WriteString(fmt.Sprint([]byte(v)))
		default:
			sb.WriteString(v.String())
		}
		if i != last {
			sb.WriteString(", ")
		}
	}

	sb.WriteString("]")
	return sb.String()
}
