package quote

import "testing"

func TestQuote(t *testing.T) {
	tests := []struct {
		name  string
		s     string
		quote string
		want  string
	}{
		{"", `a`, `"`, `"a"`},
		{"", `abc`, `"`, `"abc"`},
		{"", `a"`, `"`, `"a\""`},
		{"", `a"b`, `"`, `"a\"b"`},
		{"", `a"""b`, `"""`, `"""a\"""b"""`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Quote(tt.s, tt.quote); got != tt.want {
				t.Errorf("Quote() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnquote(t *testing.T) {
	tests := []struct {
		name  string
		s     string
		quote string
		want  string
	}{
		{"", `"a"`, `"`, `a`},
		{"", `"abc"`, `"`, `abc`},
		{"", `"a\""`, `"`, `a"`},
		{"", `"a\"b"`, `"`, `a"b`},
		{"", `"""a\"""b"""`, `"""`, `a"""b`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Unquote(tt.s, tt.quote); got != tt.want {
				t.Errorf("Unquote() = %v, want %v", got, tt.want)
			}
		})
	}
}
