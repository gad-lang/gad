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
			e := &RawHeredocLit{
				Literal: tt.Literal,
			}
			assert.Equal(t, tt.want, e.Value())
		})
	}
}
