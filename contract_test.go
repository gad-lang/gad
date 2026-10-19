package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

const contractDecls = `
contract TField { meta { label? str; hint? str } }
contract TSelectField {
    *TField
    target type<str|int|enum>
    meta {
        options []str|[]int
        layout? str = "table"
    }
}
contract TEnumField {
    *TSelectField
    target type<enum>
    meta { options []target = target.@members }
}
[options=["a", "b"], label="City"] type city <str>
enum Service { Kitchen, Bathroom, Addition }
class Form {
    pick city
    [label="Service"] service Service
    [options=["Kitchen", 2]] svc2 Service
    [options=["x", 9]] bad Service
    [label="Billing"] billing? city
    name str
}
`

func TestContract(t *testing.T) {
	// a type with meta: the instance — the target, the meta checked, the defaults
	testExpectRun(t, contractDecls+`inst := city :> TSelectField
	return [typeof(inst) == TSelectField, inst.target == str, inst.meta.options, inst.meta.layout, inst.meta.label, inst.meta.hint, inst.given("layout"), inst.given("label")]`,
		nil, Array{True, True, Array{Str("a"), Str("b")}, Str("table"), Str("City"), No, False, True})
	// a field: the type's meta, then the field's
	testExpectRun(t, contractDecls+`return [(Form.pick :> TSelectField).meta.label, (Form.billing :> TSelectField).meta.label]`,
		nil, Array{Str("City"), Str("Billing")})
	// the call's meta: the strongest layer; `no` back to the default
	testExpectRun(t, contractDecls+`return [TSelectField(city; layout="cards").meta.layout, TSelectField(city; layout=no).meta.layout]`,
		nil, Array{Str("cards"), Str("table")})
	// `::` checks: the subject itself
	testExpectRun(t, contractDecls+`return [city :: TSelectField == city, city :: TField == city]`, nil, Array{True, True})
	expectErrHas(t, contractDecls+`return Form.name :: TSelectField`, newOpts(), `meta "options" is required`)

	// an enum: a select of its members (the default computed by the target),
	// by name or value
	testExpectRun(t, contractDecls+`return [(Form.service :> TEnumField).meta.options == Service.@members,
		(Form.svc2 :> TEnumField).meta.options == [Service.Kitchen, Service.Bathroom],
		(Form.service :> TEnumField) :: TSelectField != nil]`,
		nil, Array{True, True, True})

	// check: every problem
	testExpectRun(t, contractDecls+`return TEnumField.check(Form.bad).errors`, nil,
		Array{Str(`meta "options": item 0: ErrIncompatibleAssign: ‹x› is not a member of ‹Service›; item 1: ErrIncompatibleAssign: ‹9› is not a member of ‹Service›`)})
	testExpectRun(t, contractDecls+`return TSelectField.check(Form.name; color="red")`, nil,
		Dict{"errors": Array{Str(`meta "options" is required`)}, "warnings": Array{Str(`meta "color": not declared by ‹TSelectField›`)}})
	testExpectRun(t, contractDecls+`return TEnumField.check(Form.pick).errors`, nil,
		Array{Str(`target: ‹str› is not assignable to ‹type<enum>› (TEnumField)`)})

	// reflection
	testExpectRun(t, contractDecls+`return [TEnumField.@parents[0] == TSelectField, TEnumField.@depth, TField.@depth, str(TSelectField.@target), TSelectField.@name]`,
		nil, Array{True, Int(2), Int(0), Str("[type<str>, type<int>, type<enum>]"), Str("TSelectField")})
	testExpectRun(t, `[since=2] contract T { target any }; return T.@meta`, nil, KeyValueArray{&KeyValue{K: Str("since"), V: Int(2)}})
	testExpectRun(t, contractDecls+`return [f.name for f in TSelectField.metaIface(str).fields]`, nil,
		Array{Str("label"), Str("hint"), Str("options"), Str("layout")})

	// contract as a name elsewhere
	testExpectRun(t, `contract := 1; return contract + 1`, nil, Int(2))
}
