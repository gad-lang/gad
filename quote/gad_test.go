package quote

import "testing"

// roundTrip values exercised across every form.
var roundTripValues = []string{
	"hello",
	"a\tb",
	"a\nb",
	"  indented\nsecond",
	"has \" quote",
	"has ` backtick",
	"three ``` backticks",
	`C:\path\{x}`,
	"",
	"line1\nline2\nline3",
}

func TestQuoteStringRoundTrip(t *testing.T) {
	for _, s := range roundTripValues {
		q := QuoteString(s)
		got, err := UnquoteString(q)
		if err != nil || got != s {
			t.Errorf("cooked %q -> %q -> %q, err=%v", s, q, got, err)
		}
	}
}

func TestQuoteRawRoundTrip(t *testing.T) {
	for _, s := range roundTripValues {
		q := QuoteRaw(s)
		got, err := UnquoteRaw(q)
		if err != nil || got != s {
			t.Errorf("raw %q -> %q -> %q, err=%v", s, q, got, err)
		}
	}
}

func TestQuoteHeredocRoundTrip(t *testing.T) {
	for _, s := range roundTripValues {
		q := QuoteHeredoc(s)
		got, err := UnquoteHeredoc(q)
		if err != nil || got != s {
			t.Errorf("heredoc %q -> %q -> %q, err=%v", s, q, got, err)
		}
		qr := QuoteRawHeredoc(s)
		gotr, errr := UnquoteRawHeredoc(qr)
		if errr != nil || gotr != s {
			t.Errorf("raw heredoc %q -> %q -> %q, err=%v", s, qr, gotr, errr)
		}
	}
}

func TestQuoteAutoRoundTrip(t *testing.T) {
	for _, s := range roundTripValues {
		for _, o := range []Options{{}, {Raw: true}, {MaxLineWidth: 5}, {MaxLineWidth: -1}} {
			q := Quote(s, o)
			got, err := Unquote(q)
			if err != nil || got != s {
				t.Errorf("Quote(%q, %+v)=%q -> %q, err=%v", s, o, q, got, err)
			}
		}
	}
}

func TestQuoteRaw_UsesHeredocForBacktick(t *testing.T) {
	// no backtick -> single backtick form
	if got := QuoteRaw("plain"); got != "`plain`" {
		t.Fatalf("QuoteRaw(plain) = %q", got)
	}
	// backtick present -> a wider fence that does not appear in the body
	q := QuoteRaw("a`b")
	if leadingRun(q, '`') < 3 {
		t.Fatalf("QuoteRaw(a`b) = %q, expected a >=3 backtick fence", q)
	}
}

func TestQuoteWidthGoesMultiline(t *testing.T) {
	// A multi-line value longer than the width becomes a heredoc (real newlines).
	q := Quote("aaaa\nbbbb\ncccc", Options{MaxLineWidth: 8})
	if leadingRun(q, '"') != 3 {
		t.Fatalf("expected a heredoc, got %q", q)
	}
	// A short multi-line value stays a single-line "…\n…" literal.
	if q := Quote("a\nb"); q != `"a\nb"` {
		t.Fatalf("expected single-line literal, got %q", q)
	}
}

func TestUnquoteMultilineDoubleQuoted(t *testing.T) {
	// A "…" literal with a raw newline decodes to a value with a real newline.
	got, err := UnquoteString("\"line1\nline2\"")
	if err != nil || got != "line1\nline2" {
		t.Fatalf("got %q err %v", got, err)
	}
}
