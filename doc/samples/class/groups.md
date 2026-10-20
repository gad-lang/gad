
# Groups (`{ … }` in a body)

A block written where a member goes — `{ … }` — is a **group**: fields kept
together, the way a form keeps fields side by side. It is sugar for a field
typed by an anonymous class, named for its place: the N-th group of a body is
the field **`$N`**.

```text
class Contact {
    name str
    { phone str; email str }       // the field `$1`
    ? { note str }                 // the field `$2`, nullable
}

// is the same class as
class Contact {
    name str
    $1 class { phone str; email str }
    $2? class { note str }
}
```

`$1` is an identifier as any other: the field is `c.$1`, given as `$1=…`,
and the second form may be written as well — it is the same field:

```gad
class Grouped { a str; { b int } }
class Written { a str; $1 class { b int } }
g := Grouped(; $1={b: 1})
w := Written(; $1={b: 2})
[g.$1.b, w.$1.b, len(Grouped.@groups), len(Written.@groups)]
= [1, 2, 1, 1]
```

- `$N` counts the groups of **that** body, in the order written, from 1. A
  group's own body counts its own: groups nest without limit (`$1` inside
  `$1`).
- The field is **where the block is**, among the others: the order of the
  fields is the order written (a class keeps it: its instances, its
  `@flat`, the formatter).
- **`? { … }`** makes the field nullable, as `name? Type` does.
- The **metadata** written before the block (`[width=2] { … }`) is the
  field's.
- The class of a group is named as an [anonymous field's](inline_classes.gad),
  by its path: `Contact.$1`, `Contact.$1.$1`.
- The field is reached as any other — `c.$1` —, and given as any other —
  `Contact(; $1={…})`.

`@groups` lists a class's groups, in order — its fields `$1`, `$2`, … —;
`C.@fields["$1"].@types` is the class of the group, `.@nullable` whether it
may be nil.

```gad
class Contact {
    name str
    { phone str; email str }
    ? { note str }
}
c := Contact(; name = "Ana", $1 = {phone: "555-0100", email: "ana@example.com"})
names := []
for f in Contact.@groups { names += f.@name }
[
    names,
    typeName(c.$1),
    c.$1.phone,
    c.$2,                           // nullable, not given: nil
]
= [["$1", "$2"], "Contact.$1", "555-0100", nil]
```

## A group's keys: under `$N`, or `[autoGroupKeys]`

By default a group's keys are **required under it**: a dict that builds the
class — the constructor, `:>` — gives the group whole, `$1: {…}`, and a
field of the group is read through it, `c.$1.phone`. A key of a group given
flat is no field of the class.

```gad
class Contact {
    name str
    { phone str }
}
c := {name: "Ana", phone: "555-0100"} :> Contact
d := {name: "Ana", $1: {phone: "555-0100"}} :> Contact
[c.$1, d.$1.phone]                  // flat keys are no group's
= [nil, "555-0100"]
```

`[autoGroupKeys]` before the class resolves them **by themselves**, as the
record's own keys — the way a form (a schema form) posts a group's fields,
with no name of the group in their path. Both forms are taken then:

- a key of a group given flat goes to it — `Contact(; phone="…")`,
  `{phone: "…"} :> Contact` —, and so does the group given whole, `$1: {…}`;
- a field of a group is read flat, `c.phone`, and through it, `c.$1.phone`;
- each group is an instance of its class even when none of its fields is
  given (its defaults); an optional one, `? { … }`, stays nil then;
- the classes of its groups have it as well (a group in a group).

The metadata is written before the class, or set by an application that
reads a class as a form of flat paths (`Class.SetAutoGroupKeys`, the
metadata `autoGroupKeys`).

```gad
[autoGroupKeys]
class Contact {
    name str
    { phone str; email? str }
    ? { note str }
}
c := {name: "Ana", phone: "555-0100"} :> Contact
d := {name: "Bob", $1: {phone: "555-0101"}} :> Contact
e := Contact(; name="Eve", phone="555-0102")
[c.phone, c.$1.phone, c.$2, d.phone, e.phone]
= ["555-0100", "555-0100", nil, "555-0101", "555-0102"]
```

## Nested groups, and their names

```gad
class Address {
    { street str; { city str; zip str } }
}
inner := Address.@fields["$1"].@types[0]
deepest := inner.@fields["$1"].@types[0]
[inner.@name, deepest.@name, len(inner.@groups)]
= ["Address.$1", "Address.$1.$1", 1]
```

## Metadata, nullable

```gad
class Row {
    [width=2] { a str; b str }
    ? { c str }
}
g1 := Row.@fields["$1"]
g2 := Row.@fields["$2"]
[str(g1.@meta), g1.@nullable, g2.@nullable]
= ["(;width=2)", false, true]
```

## In a mixin: classes

A group of a **mixin** is a class too — never a mixin —: the class that uses
the mixin has the field `$N` of it.

```gad
mixin Contactable { { phone str } }
class Person { use Contactable; name str }
[Contactable.@groups[0].@name, typeName(Person.@fields["$1"].@types[0])]
= ["$1", "Class"]
```

## In an interface: interfaces

A group of an **interface** is an interface, `$N interface { … }`: a value
satisfies it when its `$N` does — a nullable one may be nil. The interface
lists them in `groups` (as it does `fields`, with no `@`).

```gad
interface Contactable { name str; { phone str }; ? { note str } }
ok := func(v) { try { v :: Contactable; return true } catch { return false } }
[
    len(Contactable.groups),
    ok({name: "Ana", "$1": {phone: "555-0100"}}),
    ok({name: "Ana"}),              // no $1: it is not
]
= [2, true, false]
```

## Example — `groups.gad`

```gad
example groups_basic {
    class Contact {
        name str
        { phone str; email str }
        ? { note str }
    }
    c := Contact(; name = "Ana", $1 = {phone: "555-0100", email: "ana@example.com"})
    names := []
    for f in Contact.@groups { names += f.@name }
    [
        names,
        typeName(c.$1),
        c.$1.phone,
        c.$2,                           // nullable, not given: nil
    ]
    = [["$1", "$2"], "Contact.$1", "555-0100", nil]
}

example groups_same {
    class Grouped { a str; { b int } }
    class Written { a str; $1 class { b int } }
    g := Grouped(; $1={b: 1})
    w := Written(; $1={b: 2})
    [g.$1.b, w.$1.b, len(Grouped.@groups), len(Written.@groups)]
    = [1, 2, 1, 1]
}

example groups_keys_required {
    class Contact {
        name str
        { phone str }
    }
    c := {name: "Ana", phone: "555-0100"} :> Contact
    d := {name: "Ana", $1: {phone: "555-0100"}} :> Contact
    [c.$1, d.$1.phone]                  // flat keys are no group's
    = [nil, "555-0100"]
}

example groups_keys_auto {
    [autoGroupKeys]
    class Contact {
        name str
        { phone str; email? str }
        ? { note str }
    }
    c := {name: "Ana", phone: "555-0100"} :> Contact
    d := {name: "Bob", $1: {phone: "555-0101"}} :> Contact
    e := Contact(; name="Eve", phone="555-0102")
    [c.phone, c.$1.phone, c.$2, d.phone, e.phone]
    = ["555-0100", "555-0100", nil, "555-0101", "555-0102"]
}

example groups_nested {
    class Address {
        { street str; { city str; zip str } }
    }
    inner := Address.@fields["$1"].@types[0]
    deepest := inner.@fields["$1"].@types[0]
    [inner.@name, deepest.@name, len(inner.@groups)]
    = ["Address.$1", "Address.$1.$1", 1]
}

example groups_meta {
    class Row {
        [width=2] { a str; b str }
        ? { c str }
    }
    g1 := Row.@fields["$1"]
    g2 := Row.@fields["$2"]
    [str(g1.@meta), g1.@nullable, g2.@nullable]
    = ["(;width=2)", false, true]
}

example groups_mixin {
    mixin Contactable { { phone str } }
    class Person { use Contactable; name str }
    [Contactable.@groups[0].@name, typeName(Person.@fields["$1"].@types[0])]
    = ["$1", "Class"]
}

example groups_interface {
    interface Contactable { name str; { phone str }; ? { note str } }
    ok := func(v) { try { v :: Contactable; return true } catch { return false } }
    [
        len(Contactable.groups),
        ok({name: "Ana", "$1": {phone: "555-0100"}}),
        ok({name: "Ana"}),              // no $1: it is not
    ]
    = [2, true, false]
}
```
