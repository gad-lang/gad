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

// A plugin: overloads whose params are contracts. The call takes the contract
// with more required keys of its meta, then the deeper in its parents —
// whatever the order of declaration —; two of the same rank that fit are an
// ambiguity.
func TestContractOverloads(t *testing.T) {
	const decls = `
	contract TField { meta { label? str } }
	contract TTextField { *TField; target type<str>; meta { maxlength? int } }
	contract TSelectField { *TField; target type<str|int|enum>; meta { options []str|[]int } }
	contract TEnumField { *TSelectField; target type<enum>; meta { options []target = target.@members } }
	contract TNumberField { *TField; target type<int|float>; meta { min? int } }
	enum Service { Kitchen, Bathroom }
	class Form {
	    [options=["a", "b"]] pick str
	    service Service
	    name str
	    [min=1] rooms int
	    flag bool
	}
	`
	want := Array{Str("select"), Str("enum"), Str("text"), Str("number"), Str("field")}
	testExpectRun(t, decls+`draw := func {
	    (f TField) => "field"
	    (f TTextField) => "text"
	    (f TNumberField) => "number"
	    (f TSelectField) => "select"
	    (f TEnumField) => "enum"
	}
	return [draw(Form.pick), draw(Form.service), draw(Form.name), draw(Form.rooms), draw(Form.flag)]`, nil, want)
	// a param of a contract alone: a subject that fits, or an error
	testExpectRun(t, decls+`one := (f TSelectField) => (f :> TSelectField).meta.options; return one(Form.pick)`,
		nil, Array{Str("a"), Str("b")})
	expectErrHas(t, decls+`one := (f TSelectField) => 1; return one(Form.name)`, newOpts(), "expected TSelectField")
	// an instance fits its contract and its parents'
	testExpectRun(t, decls+`d := func { (f TEnumField) => "enum"; (f TField) => "field" }; return d(Form.service :> TEnumField)`,
		nil, Str("enum"))
	// two of the same rank that fit: ambiguous
	expectErrHas(t, `contract TA { target type<str> }; contract TB { target type<str> }; class F { name str }
	d := func { (f TA) => 1; (f TB) => 2 }; return d(F.name)`, newOpts(), "ambiguous")
	// the candidates are the function's methods
	testExpectRun(t, decls+`d := func { (f TTextField) => 1; (f TNumberField) => 2 }; return len(d.@methods)`, nil, Int(2))
}

// A contract is a value too: anonymous (`contract { … }`), made by a function,
// written where a type goes.
func TestContractExpr(t *testing.T) {
	testExpectRun(t, `const TSel = contract { target type<str>; meta { options []str } }
	[options=["a"]] type city <str>
	return [(city :> TSel).meta.options, TSel.@name, typeName(TSel)]`,
		nil, Array{Array{Str("a")}, Str("contract"), Str("contract")})
	testExpectRun(t, `mk := (n) => contract { meta { n? int = n } }; return [(str :> mk(5)).meta.n, (str :> mk(7)).meta.n]`,
		nil, Array{Int(5), Int(7)})
	testExpectRun(t, `x := contract Named { target any }; return x.@name`, nil, Str("Named"))
	testExpectRun(t, `class F { a str; b int }
	d := func { (f contract { target type<str> }) => "s"; (f contract { target type<int> }) => "i" }
	return [d(F.a), d(F.b)]`, nil, Array{Str("s"), Str("i")})
}
