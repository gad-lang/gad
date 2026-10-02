package parser_test

import (
	"strings"
	"testing"

	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/test"
)

// TestParseMetadata verifies a `[k=v, …]` metadata block parses on a declaration
// and its members and round-trips through the formatter.
func TestParseMetadata(t *testing.T) {
	// interface + a member, each with metadata (the member's is rendered inline
	// before it; both round-trip).
	test.ExpectParseString(t,
		"[a=1]\ninterface I { [b=2]\n x int }",
		`[a=1] interface I {[b=2] x int; }`)

	// flag entry (`k` == `k=true`) and a nested key-value array value.
	test.ExpectParseString(t,
		`[readonly, db=(;primary_key)] interface I { x int }`,
		`[readonly, db=(;primary_key)] interface I {x int; }`)

	// enum + item.
	test.ExpectParseString(t,
		"[cat=\"acl\"]\nenum E { [bit=0]\n Read }",
		`[cat="acl"] enum E {[bit=0] Read}`)

	// a bare array before a NON-declaration is still an ordinary array statement.
	test.ExpectParseString(t, "[1, 2, 3]\nx := 1", `[1, 2, 3]; x := 1`)
}

// TestParseMetadataClass verifies metadata parses on a class and its fields,
// props and methods and round-trips through the formatter.
func TestParseMetadataClass(t *testing.T) {
	test.ExpectParseString(t,
		"[a=1]\nclass C { [b=2]\n x = 0 }",
		`[a=1] class C {[b=2] x = 0}`)

	test.ExpectParseString(t,
		"class C { methods { [route=\"/s\"]\n save() { return 1 } } }",
		`class C {methods {[route="/s"] save() {return 1}}}`)

	// marker type and mixin.
	test.ExpectParseString(t,
		"[kind=\"m\"]\ntype T { [n=1]\n x = 0 }",
		`[kind="m"] type T {[n=1] x = 0}`)
	test.ExpectParseString(t,
		"[role=\"r\"]\nmixin M { x = 0 }",
		`[role="r"] mixin M {x = 0}`)
}

// TestParseMetadataUnbracketed parses a metadata block's entries written on
// their own — no `[ ]` —, with single-quoted strings, keeping the positions of
// the text.
func TestParseMetadataUnbracketed(t *testing.T) {
	src := `type=html, label='Tab: all', hint="Shows %s.", readonly`
	kva, err := parser.ParseMetadata(src, 0)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(kva.Elements); n != 4 {
		t.Fatalf("%d entries, want 4: %s", n, kva)
	}
	want := []struct{ key, value string }{
		{"type", "html"}, {"label", "Tab: all"}, {"hint", "Shows %s."}, {"readonly", ""},
	}
	for i, w := range want {
		kv := kva.Elements[i].(*node.KeyValuePairLit)
		if got := kv.Key.String(); got != w.key {
			t.Errorf("entry %d key %q, want %q", i, got, w.key)
		}
		got := ""
		switch v := kv.Value.(type) {
		case nil:
		case *node.StrLit:
			got = v.Value()
		default:
			got = v.String()
		}
		if got != w.value {
			t.Errorf("entry %d value %q, want %q", i, got, w.value)
		}
	}

	// positions are offsets into the text (+1 for a text on its own)…
	label := kva.Elements[1].(*node.KeyValuePairLit)
	if got, want := int(label.Value.Pos())-1, strings.Index(src, "'Tab"); got != want {
		t.Errorf("the label's value at offset %d, want %d", got, want)
	}
	// …or into whatever holds it, from base.
	kva, err = parser.ParseMetadata(src, 100)
	if err != nil {
		t.Fatal(err)
	}
	hint := kva.Elements[2].(*node.KeyValuePairLit)
	if got, want := int(hint.Key.Pos()), 100+strings.Index(src, "hint"); got != want {
		t.Errorf("the hint's key at %d, want %d", got, want)
	}

	// an error points into the text
	_, err = parser.ParseMetadata(`a=1, b='open`, 0)
	if err == nil {
		t.Fatal("an unclosed string was accepted")
	}
	if !strings.Contains(err.Error(), "1:8") { // the quote that opens it
		t.Errorf("the error does not say where: %v", err)
	}
	if kva, err := parser.ParseMetadata("", 0); err != nil || len(kva.Elements) != 0 {
		t.Errorf("empty metadata: %v %v", kva, err)
	}
}
