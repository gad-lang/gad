package gadx

import (
	"testing"

	"github.com/gad-lang/gad"
)

// EMPTY must not be falsy: the attribute renderer drops falsy values, which is
// what makes a conditional attribute disappear, and an explicitly empty value
// has to survive that rule.
func TestEmptyIsNotFalsy(t *testing.T) {
	if EMPTY.IsFalsy() {
		t.Error("EMPTY is falsy, so the renderer would drop it")
	}
	if gad.Str("").IsFalsy() != true {
		t.Error("a plain empty string should still be falsy")
	}
}

func TestEmptyHasItsOwnType(t *testing.T) {
	if EMPTY.Type() != EmptyType {
		t.Errorf("Type() = %v, want %v", EMPTY.Type(), EmptyType)
	}
	if EMPTY.Type() == gad.Str("").Type() {
		t.Error("EMPTY shares the string type, so the renderer cannot tell them apart")
	}
}

// `@empty` is present and empty; a plain "" is still dropped.
func TestEmptyRendersPresentAndEmpty(t *testing.T) {
	got := renderGadx(t, "@main\n    option[value=@empty] x\n", gad.Dict{})
	if got != `<option value="">x</option>` {
		t.Errorf(`got %s, want <option value="">x</option>`, got)
	}

	got = renderGadx(t, "@main\n    option[value=\"\"] x\n", gad.Dict{})
	if got != "<option>x</option>" {
		t.Errorf("an empty string should still be dropped, got %s", got)
	}
}

// Inline HTML means what HTML means: `value=""` is an attribute that is there
// with nothing in it, which is what a placeholder <option> relies on.
func TestEmptyFromInlineHTML(t *testing.T) {
	got := renderGadx(t, "@main\n    <option value=\"\">x</option>\n", gad.Dict{})
	if got != `<option value="">x</option>` {
		t.Errorf(`got %s, want <option value="">x</option>`, got)
	}
}
