
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
// => [1, 2, 1, 1]
```

- `$N` counts the groups of **that** body, in the order written, from 1. A
  group's own body counts its own: groups nest without limit (`$1` inside
  `$1`).
- The field is **where the block is**, among the others: the order of the
  fields is the order written (`[ordered]` instances keep it, and the
  formatter does not reorder a class that has groups).
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
// => [["$1", "$2"], "Contact.$1", "555-0100", nil]
```

## Nested groups, and their names

```gad
class Address {
    { street str; { city str; zip str } }
}
inner := Address.@fields["$1"].@types[0]
deepest := inner.@fields["$1"].@types[0]
[inner.@name, deepest.@name, len(inner.@groups)]
// => ["Address.$1", "Address.$1.$1", 1]
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
// => ["(;width=2)", false, true]
```

## In a mixin: classes

A group of a **mixin** is a class too — never a mixin —: the class that uses
the mixin has the field `$N` of it.

```gad
mixin Contactable { { phone str } }
class Person { use Contactable; name str }
[Contactable.@groups[0].@name, typeName(Person.@fields["$1"].@types[0])]
// => ["$1", "Class"]
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
// => [2, true, false]
```

## Example — `groups.gad`

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

class Grouped { a str; { b int } }
class Written { a str; $1 class { b int } }
g := Grouped(; $1={b: 1})
w := Written(; $1={b: 2})
[g.$1.b, w.$1.b, len(Grouped.@groups), len(Written.@groups)]

class Address {
    { street str; { city str; zip str } }
}
inner := Address.@fields["$1"].@types[0]
deepest := inner.@fields["$1"].@types[0]
[inner.@name, deepest.@name, len(inner.@groups)]

class Row {
    [width=2] { a str; b str }
    ? { c str }
}
g1 := Row.@fields["$1"]
g2 := Row.@fields["$2"]
[str(g1.@meta), g1.@nullable, g2.@nullable]

mixin Contactable { { phone str } }
class Person { use Contactable; name str }
[Contactable.@groups[0].@name, typeName(Person.@fields["$1"].@types[0])]

interface Contactable { name str; { phone str }; ? { note str } }
ok := func(v) { try { v :: Contactable; return true } catch { return false } }
[
    len(Contactable.groups),
    ok({name: "Ana", "$1": {phone: "555-0100"}}),
    ok({name: "Ana"}),              // no $1: it is not
]
```
