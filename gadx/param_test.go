package gadx

import (
	"strings"
	"testing"
)

// TestParamRender covers @param — the gadx analog of Gad's top-level `param`
// declaration. Named params (after `;`) may carry defaults, which apply when no
// argument is supplied, so they render deterministically here.
func TestParamRender(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "named default in text",
			src:  "@param (; a = 5)\n@main\n    p {a}\n",
			want: `<p>5</p>`,
		},
		{
			name: "named default string",
			src:  "@param (; greet = \"hi\")\n@main\n    p {greet}\n",
			want: `<p>hi</p>`,
		},
		{
			name: "param used in attribute",
			src:  "@param (; cls = \"box\")\n@main\n    div[class=cls] x\n",
			want: `<div class="box">x</div>`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := strings.TrimSpace(renderGadx(t, tc.src, nil))
			if got != tc.want {
				t.Fatalf("render mismatch\n got: %q\nwant: %q", got, tc.want)
			}
		})
	}
}
