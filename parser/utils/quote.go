package utils

import (
	_ "strconv"
	_ "unsafe"
)

func Quote(s string, quote byte) string {
	return strconvQuoteWith(s, quote, false, false)
}

//go:linkname strconvQuoteWith strconv.quoteWith
func strconvQuoteWith(s string, quote byte, ASCIIonly, graphicOnly bool) string
