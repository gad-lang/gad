package repr

const (
	QuotePrefix = "‹"
	QuoteSufix  = "›"
)

func Quote(s string) string {
	return QuotePrefix + s + QuoteSufix
}
