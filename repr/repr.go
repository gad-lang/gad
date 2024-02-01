package repr

const (
	QuotePrefix = "‹"
	QuoteSufix  = "›"
)

func Quote(s string) string {
	return QuotePrefix + s + QuoteSufix
}

func QuoteTyped(typ, s string) string {
	return QuotePrefix + typ + ":" + s + QuoteSufix
}
