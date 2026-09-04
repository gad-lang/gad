package gadx

import "github.com/gad-lang/gad"

// EmptyValue is an attribute value that is present and empty.
//
// An attribute whose value is falsy is left out — that is how `[hidden=cond]`
// disappears when cond is false — which means a plain `""` cannot express an
// attribute that is there with nothing in it. Some must be: `<option value="">`
// is the placeholder entry of a select, and dropping its value changes what the
// form submits. EmptyValue is never falsy, so it survives that rule; the
// attribute renderer writes it as `name=""`.
type EmptyValue string

func (EmptyValue) Type() gad.ObjectType { return EmptyType }
func (o EmptyValue) ToString() string   { return string(o) }
func (EmptyValue) IsFalsy() bool        { return false }

func (o EmptyValue) Equal(right gad.Object) bool {
	switch v := right.(type) {
	case EmptyValue:
		return o == v
	case gad.Str:
		return string(o) == string(v)
	}
	return false
}

// EmptyType is the Gad object type of EmptyValue. It is its own type rather
// than a string one so that the attribute renderer can tell an explicitly empty
// value from an ordinary "" — which is falsy, and dropped.
var EmptyType = gad.NewBuiltinObjType("Empty")

// EMPTY is the empty attribute value, written `@empty` in a template.
const EMPTY EmptyValue = ""
