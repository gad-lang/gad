package node

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRawHeredocLit_Value(t *testing.T) {
	tests := []struct {
		name    string
		Literal string
		want    string
	}{
		{"single", "```abc```", "abc"},
		{"single", "```  abc```", "  abc"},
		{"single", "```  a\nbc  ```", "  a\nbc  "},
		{"single", "```  a\nb\n\tc  ```", "  a\nb\n\tc  "},
		{"multiline", "```\nabc\n```", "abc"},
		{"multiline", "```\n\tabc\n\t```", "abc"},
		{"multiline", "```\n\ta\n\tbc\n\t```", "a\nbc"},
		{"multiline", "```\n\ta\n\t bc\n\t```", "a\n bc"},
		{"multiline", "```\n\ta\n\t b\nc\n\t```", "a\n b\nc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := RawHeredoc(tt.Literal, 0)
			assert.Equal(t, tt.want, e.Value())
		})
	}
}

func TestHeredocLit_Value(t *testing.T) {
	tests := []struct {
		name    string
		Literal string
		want    string
	}{
		{"single", `"""abc"""`, "abc"},
		{"single-indent", `"""  abc"""`, "  abc"},
		{"single-newline", `"""  a` + "\nbc  " + `"""`, "  a\nbc  "},
		{"multiline", `"""` + "\nabc\n" + `"""`, "abc"},
		{"multiline-tab", `"""` + "\n\tabc\n\t" + `"""`, "abc"},
		{"multiline-strip", `"""` + "\n\ta\n\tbc\n\t" + `"""`, "a\nbc"},
		{"multiline-partial", `"""` + "\n\ta\n\t bc\n\t" + `"""`, "a\n bc"},
		{"escape-newline", `"""a\nb"""`, "a\nb"},
		{"escape-tab", `"""a\tb"""`, "a\tb"},
		{"escape-quote", `"""a\"b"""`, "a\"b"},
		{"escape-backslash", `"""a\\b"""`, "a\\b"},
		{"escape-hex", `"""\x41"""`, "A"},
		{"escape-unicode-seq", `"""\u00e9"""`, "\u00e9"},
		{"escape-unicode", `"""é"""`, "é"},
		{"escape-unknown-kept", `"""a\qb"""`, "a\\qb"},
		{"multiline-escape", `"""` + "\n\ta\\tb\n\t" + `"""`, "a\tb"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := Heredoc(tt.Literal, 0)
			assert.Equal(t, tt.want, e.Value())
		})
	}
}
