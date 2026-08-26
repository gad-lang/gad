package parser

import (
	"strings"
	"testing"
)

// TestCallLine checks the `! callee arg1 arg2 …` fluent call statement: the
// space-separated parts (respecting balanced brackets/quotes) become the callee
// and its arguments, and it round-trips.
func TestCallLine(t *testing.T) {
	src := "@test t\n" +
		"    ! t.equal render(list([\"a\", \"b\"])) \"<ul>\"\n" +
		"    ! t.nil nil\n"

	out := transpileGadx(t, src)
	for _, want := range []string{
		`! t.equal render(list(["a", "b"])) "<ul>"`, // both args stay balanced
		`! t.nil nil`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	// The `! ` prefix is not emitted as literal text.
	if strings.Contains(out, "| !") {
		t.Fatalf("call line emitted as text:\n%s", out)
	}
	// Idempotent.
	if again := transpileGadx(t, out); again != out {
		t.Fatalf("not idempotent:\n--- first ---\n%s\n--- second ---\n%s", out, again)
	}
}

// TestFormatInlineText: a tag whose whole body is one short text run is inlined
// as `tag text` (from ERROS.md).
func TestFormatInlineText(t *testing.T) {
	out := transpileGadx(t, "@main\n    <span>one</span>\n")
	if !strings.Contains(out, "span one") || strings.Contains(out, "| one") {
		t.Fatalf("expected inline `span one`:\n%s", out)
	}
}

// TestFormatMergeAttrs: separate attribute groups merge into one, and a long
// group wraps one item per line (from ERROS.md).
func TestFormatMergeAttrs(t *testing.T) {
	merged := transpileGadx(t, "@main\n    div[a=\"v\"][b=\"x\"]\n")
	if !strings.Contains(merged, `div[a="v", b="x"]`) {
		t.Fatalf("expected merged group:\n%s", merged)
	}
	long := transpileGadx(t, "@main\n    div[alpha=\"1\"][beta=\"2\"][gamma=\"3\"][delta=\"4\"][epsilon=\"5\"][zeta=\"6\"][eta=\"7\"][theta=\"8\"]\n")
	if !strings.Contains(long, "div[\n") {
		t.Fatalf("expected wrapped group on overflow:\n%s", long)
	}
}

// TestFormatBlankLineBeforeDirective: a blank line separates top-level directive
// declarations (from ERROS.md).
func TestFormatBlankLineBeforeDirective(t *testing.T) {
	out := transpileGadx(t, "@param (; a = 1)\n@comp main()\n    p x\n")
	if !strings.Contains(out, ")\n\n@comp") {
		t.Fatalf("expected blank line before @comp:\n%s", out)
	}
}

// TestFormatMultilineDoc: a multi-line doc comment keeps `/**` and `**/` on
// their own lines (the trailing newline before `**/` survives), for both a
// declaration doc and a standalone block comment.
func TestFormatMultilineDoc(t *testing.T) {
	out := transpileGadx(t, "/**\nline one\nline two\n**/\n@comp main()\n    p x\n")
	if !strings.Contains(out, "/**\nline one\nline two\n**/") {
		t.Fatalf("expected block doc form:\n%s", out)
	}
}

// TestFormatEnumPreserved: an `@enum` declaration is emitted (not silently
// dropped) as `@enum Name (fields)`.
func TestFormatEnumPreserved(t *testing.T) {
	out := transpileGadx(t, "@enum Perm (Read, Write, Exec = 10, Delete)\n@main\n    p x\n")
	if !strings.Contains(out, "@enum Perm (Read, Write, Exec = 10, Delete)") {
		t.Fatalf("@enum not preserved:\n%s", out)
	}
}

// TestFormatBlankBeforeLeadingComment: the blank line before a directive lands
// before its leading `//-` comment, not between the comment and the directive.
func TestFormatBlankBeforeLeadingComment(t *testing.T) {
	out := transpileGadx(t, "@comp a()\n    p x\n// doc for b\n@comp b()\n    p y\n")
	if !strings.Contains(out, "\n\n// doc for b\n@comp b()") {
		t.Fatalf("blank line should precede the leading comment:\n%s", out)
	}
}

// TestFormatSlotScope: a slot declaration/pass keeps its `(scope)` (the LParen
// position is synthetic, so emission gates on the rendered params) and a
// component call keeps its call-scope `~` InitStmts.
func TestFormatSlotScope(t *testing.T) {
	src := "@export comp list(items)\n" +
		"    @for it in items\n" +
		"        @slot row(it)\n" +
		"            span {= it }\n" +
		"@main\n" +
		"    +list([1])\n" +
		"        ~ const target = 1\n" +
		"        @slot #row(it)\n" +
		"            b {= it }\n"
	out := transpileGadx(t, src)
	for _, want := range []string{"@slot row(it)", "@slot #row(it)", "~ const target = 1"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	// An empty scope is not emitted as `()`.
	if strings.Contains(out, "@slot row()") {
		t.Fatalf("empty scope should be suppressed:\n%s", out)
	}
}

// TestFormatFileDocStandalone: a `/** … **/` block doc separated from the first
// directive by a blank line is a file/section doc and must keep that blank line
// (otherwise it re-attaches to the directive as its own doc — changing the
// parse). A doc immediately before a directive (no blank) stays attached.
func TestFormatFileDocStandalone(t *testing.T) {
	standalone := transpileGadx(t, "/** file doc **/\n\n@comp a()\n    p x\n")
	if !strings.Contains(standalone, "/** file doc **/\n\n@comp a()") {
		t.Fatalf("file doc should keep its blank line:\n%s", standalone)
	}
	attached := transpileGadx(t, "/** comp doc **/\n@comp a()\n    p x\n")
	if !strings.Contains(attached, "/** comp doc **/\n@comp a()") {
		t.Fatalf("attached doc should stay attached:\n%s", attached)
	}
}
