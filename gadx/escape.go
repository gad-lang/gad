package gadx

import (
	"html"
	"strings"
)

// AttrValueQuoter renders an attribute value as a quoted, escaped string,
// including the surrounding quotes. It is used when a tag is rendered to HTML.
type AttrValueQuoter func(value string) string

// AttrQuoteHTML double-quotes the value and HTML-entity-escapes it (`"`→`&#34;`,
// `&`→`&amp;`, `<`→`&lt;`, `>`→`&gt;`, `'`→`&#39;`), so an interpolated value
// cannot break out of the attribute. It is the safe default.
func AttrQuoteHTML(v string) string { return `"` + html.EscapeString(v) + `"` }

// AttrQuoteSingleQuote single-quotes the value and escapes only `'` as `&#39;`,
// leaving `"`, `<`, `>` and `&` intact. This suits templates whose attribute
// values are framework expressions that must keep their double quotes and
// operators — e.g. VueJS bindings like `:class="{ active: a && b }"`.
func AttrQuoteSingleQuote(v string) string {
	return "'" + strings.ReplaceAll(v, "'", "&#39;") + "'"
}

// AttrValueQuote selects how attribute values are quoted/escaped when a tag is
// rendered to HTML. It defaults to the safe AttrQuoteHTML; an embedding program
// (or a gadx-targeting compiler) may set AttrQuoteSingleQuote for VueJS-style
// output, or a custom quoter.
var AttrValueQuote AttrValueQuoter = AttrQuoteHTML

// escapeText HTML-escapes text content so an interpolated (non-RawStr) value
// cannot inject markup. Use a RawStr (or gadx.escape) for trusted HTML.
func escapeText(s string) string { return html.EscapeString(s) }
