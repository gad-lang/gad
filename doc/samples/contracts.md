
# Contracts (`contract NAME { … }`)

A **contract** says two things of a type: what it is — its **target** — and
what its **metadata** may say — the keys of `meta { … }`, the body of an
[interface](interfaces.gad): each key typed, optional (`?`) or required, with a
[default](interfaces.gad#field-defaults-name-t--expr).

```gad ignore
contract TSelectField {
    *TField                      // parent contracts: target and meta merged
    target type<str|int|enum>    // the types the target may be (none: any)
    meta {
        options []str            // required: what makes it a select
        layout? str = "table"    // a key not given takes its default
    }
}
```

The target's types are written as a field's are — `type<str>` the type `str`
itself, `type<enum>` any enum (a [category](meta_types_test.gad)). A child
contract (`*Parent`) narrows its parents: its target must fit theirs too, its
meta is theirs and its own, a key of its own over a parent's of the same name.

```gad
contract TField {
    meta { label? str; hint? str }
}

contract TSelectField {
    *TField
    target type<str|int|enum>
    meta {
        options []str|[]int
        layout? str = "table"
    }
}

[options=["a", "b"], label="City"]
type city <str>

inst := city :> TSelectField
[typeof(inst) == TSelectField, inst.target == str, inst.meta.options, inst.meta.layout,
 inst.meta.label, inst.meta.hint, inst.given("layout"), inst.given("label")]
// => [true, true, ["a", "b"], "table", "City", off, false, true]
```

## The subject

A contract is checked against a **subject**:

- a **type** — a class, an enum, a typed array, a declared union of one type
  (`[options=[…]] type city <str>`, which is `str` with that metadata) —: the
  target is the type, the meta its `@meta`;
- a **field** of a class or an interface (`Form.city`): the target is its type,
  the meta its type's `@meta` and then the field's own, the field's over the
  type's;
- an **instance** of a contract: its target and meta.

| Cast | Does |
|---|---|
| `subject :: C` | checks: the subject itself, or an error saying what does not fit |
| `subject :> C` | the **instance** of the contract |
| `C(subject; k=v, …)` | the instance, the named arguments the strongest layer of the meta |
| `C.check(subject)` | `{errors: […], warnings: […]}` — every problem, not only the first |

The instance: `inst.target`, `inst.meta` (a dict of **every** key of the
contract: the values checked and transformed — an enum's member by its name
or value —, the defaults applied, a key not given `no`, as a named parameter
not given), `inst.rawMeta` (the metadata as written, the keys of other
plugins too), `inst.given(key)` (given, with or without a value — a default
does not count, a reset `no` is not given) and `typeof(inst) == C`.

A key the contract does not declare is a **warning** of `check`, not an
error: another plugin's, maybe.

```gad
contract TField {
    meta { label? str; hint? str }
}

contract TSelectField {
    *TField
    target type<str|int|enum>
    meta {
        options []str|[]int
        layout? str = "table"
    }
}

[options=["a", "b"], label="City"]
type city <str>

class Form {
    pick city                       // the type's meta
    [label="Billing"] billing? city // the field's over the type's
    name str                        // no options: not a select
}

[(Form.pick :> TSelectField).meta.label, (Form.billing :> TSelectField).meta.label,
 TSelectField(city; layout="cards").meta.layout,   // the call: the strongest layer
 TSelectField(city; layout=no).meta.layout,        // a reset: the default
 city :: TSelectField == city,
 TSelectField.check(Form.name; color="red")]
// => ["City", "Billing", "cards", "table", true, {errors: ["meta \"options\" is required"], warnings: ["meta \"color\": not declared by ‹TSelectField›"]}]
```

## `target` in the meta

Inside the contract `target` is a type and a value — the target given: the
meta of each target is built for it. An enum's options are its members:

```gad
contract TSelectField {
    target type<str|int|enum>
    meta { options []str|[]int }
}

// an enum: a select of its members — `options` redeclared, its items of the
// enum (`target` as a type), its default the members (`target` as a value)
contract TEnumField {
    *TSelectField
    target type<enum>
    meta { options []target = target.@members }
}

enum Service { Kitchen, Bathroom, Addition }

class Form {
    service Service                          // its members
    [options=["Kitchen", 2]] svc2 Service    // by name, by value
    [options=["x"]] bad Service
}

[(Form.service :> TEnumField).meta.options == Service.@members,
 (Form.svc2 :> TEnumField).meta.options == [Service.Kitchen, Service.Bathroom],
 (Form.service :> TEnumField) :: TSelectField != nil,   // it is a TSelectField too
 len(TEnumField.check(Form.bad).errors),
 TEnumField.check(Form.service).errors]
// => [true, true, true, 1, []]
```

## Reflection

`C.@name`, `C.@meta` (the `[k=v]` before the declaration), `C.@parents`,
`C.@target` (the types the target may be), `C.@depth` (how deep in its
parents) and `C.metaIface(target)` (the interface of the meta of a target).

```gad
contract TField {
    meta { label? str }
}

[since=2]
contract TSelectField {
    *TField
    target type<str|int>
    meta { options []str }
}

[TSelectField.@name, TSelectField.@meta, TSelectField.@parents[0] == TField,
 str(TSelectField.@target), TSelectField.@depth,
 [f.name for f in TSelectField.metaIface(str).fields]]
// => ["TSelectField", (;since=2), true, "[type<str>, type<int>]", 1, ["label", "options"]]
```

## Example — `contracts.gad`

```gad
contract TField {
    meta { label? str; hint? str }
}

contract TSelectField {
    *TField
    target type<str|int|enum>
    meta {
        options []str|[]int
        layout? str = "table"
    }
}

[options=["a", "b"], label="City"]
type city <str>

inst := city :> TSelectField
[typeof(inst) == TSelectField, inst.target == str, inst.meta.options, inst.meta.layout,
 inst.meta.label, inst.meta.hint, inst.given("layout"), inst.given("label")]

contract TField {
    meta { label? str; hint? str }
}

contract TSelectField {
    *TField
    target type<str|int|enum>
    meta {
        options []str|[]int
        layout? str = "table"
    }
}

[options=["a", "b"], label="City"]
type city <str>

class Form {
    pick city                       // the type's meta
    [label="Billing"] billing? city // the field's over the type's
    name str                        // no options: not a select
}

[(Form.pick :> TSelectField).meta.label, (Form.billing :> TSelectField).meta.label,
 TSelectField(city; layout="cards").meta.layout,   // the call: the strongest layer
 TSelectField(city; layout=no).meta.layout,        // a reset: the default
 city :: TSelectField == city,
 TSelectField.check(Form.name; color="red")]

contract TSelectField {
    target type<str|int|enum>
    meta { options []str|[]int }
}

// an enum: a select of its members — `options` redeclared, its items of the
// enum (`target` as a type), its default the members (`target` as a value)
contract TEnumField {
    *TSelectField
    target type<enum>
    meta { options []target = target.@members }
}

enum Service { Kitchen, Bathroom, Addition }

class Form {
    service Service                          // its members
    [options=["Kitchen", 2]] svc2 Service    // by name, by value
    [options=["x"]] bad Service
}

[(Form.service :> TEnumField).meta.options == Service.@members,
 (Form.svc2 :> TEnumField).meta.options == [Service.Kitchen, Service.Bathroom],
 (Form.service :> TEnumField) :: TSelectField != nil,   // it is a TSelectField too
 len(TEnumField.check(Form.bad).errors),
 TEnumField.check(Form.service).errors]

contract TField {
    meta { label? str }
}

[since=2]
contract TSelectField {
    *TField
    target type<str|int>
    meta { options []str }
}

[TSelectField.@name, TSelectField.@meta, TSelectField.@parents[0] == TField,
 str(TSelectField.@target), TSelectField.@depth,
 [f.name for f in TSelectField.metaIface(str).fields]]
```
