
# Typed dicts (`DictType[K, V]`, `type NAME dict[K, V]`)

A dict's keys are strings and its values anything. A **typed dict** says
which: `DictType[K, V]` is the type of a dict whose keys are of `K` and whose
values are of `V`.

- **K** is a type of strings: `str`, an **enum** (a key is a member's name), a
  union of them. Any other K is an error.
- **V** is any type a field may have.
- `DictType[V]` is `DictType[str, V]`, and **`dict` is `DictType[str, any]`** —
  the same object: `dict === DictType[str, any]`, `typeof({}) === dict`.
- `DictType[K, V]` with the same arguments is the same type (`===`); its
  reflection: `@key`, `@value`, `@name`; its type is `DictType`.

## The generic: structural

`DictType[K, V]` is **structural**, as an array type is: a plain dict of those
keys and values is of it — a param, a field, `::` take it —, nothing is bound
to the value. Called, it builds a dict, its entries checked (not converted:
`::>` converts each key and value):

- named args are entries: `D(; a=1)`;
- positional args are entries: a `[k=v]`, a `[key, value]` pair, a dict, a
  key-value array;
- **one array alone** is its indexes as keys: `D([10, 20])` is `{"0": 10,
  "1": 20}` (as `dict([…])`).

```gad
D := DictType[str, int]
f := func(scores DictType[str, int]) => scores.ana
[f({ana: 7}), D(; a=1), D([a=1], ["b", 2], {c: 3}), D([10, 20])]
= [7, {a: 1}, {a: 1, b: 2, c: 3}, {"0": 10, "1": 20}]
[dict === DictType[str, any], DictType[int] === D, str(D), D.@value == int]
= [true, true, "DictType[str, int]", true]
({a: "1"} ::> D)
= {a: 1}
```

## `type NAME dict[K, V]`: a type of its own

`type NAME dict[K, V]` declares a **new object type**, a child of the generic
— nominal: its instances are bound to it, **every write is checked**. It is a
dict too (`s :: dict`, a param `x dict`, `s :: DictType[str, int]`), and its
constructors and overloads are its own (`met NAME(x T) => …`). `type NAME dict`
is `dict[str, any]`.

Written where a value goes, it is a value: `type dict[K, V]` anonymous,
`type NAME dict[K, V]` named (NAME is its name, not declared).

```gad
type scores dict[str, int]
met scores(n int) => scores(; total=n)     // an overload of its own
s := scores(; ana=1)
s.bia = 2
[typeName(s), len(s), s :: DictType[str, int] == s, scores(5).total]
= ["scores", 2, true, 5]
refused := func(f) { try { f(); return false } catch { return true } }
refused(() => { s.caio = "x" })            // a write is checked
= true
const flags = type dict[str, bool]
typeName(flags(; on=true))
= "dict[str, bool]"
```

## Enum keys

An enum as K: a key is a member's name; the constructor takes the member or its
name, `::>` its value too.

```gad
enum Svc { Kitchen, Bath }
E := DictType[Svc, bool]
[E(; Kitchen=true), E([Svc.Bath, false], [Svc.Kitchen, true]), {Kitchen: true} :: E]
= [{Kitchen: true}, {Bath: false, Kitchen: true}, {Kitchen: true}]
```

## Example — `typed_dicts.gad`

```gad
example generic {
    D := DictType[str, int]
    f := func(scores DictType[str, int]) => scores.ana
    [f({ana: 7}), D(; a=1), D([a=1], ["b", 2], {c: 3}), D([10, 20])]
    = [7, {a: 1}, {a: 1, b: 2, c: 3}, {"0": 10, "1": 20}]
    [dict === DictType[str, any], DictType[int] === D, str(D), D.@value == int]
    = [true, true, "DictType[str, int]", true]
    ({a: "1"} ::> D)
    = {a: 1}
}

example typed {
    type scores dict[str, int]
    met scores(n int) => scores(; total=n)     // an overload of its own
    s := scores(; ana=1)
    s.bia = 2
    [typeName(s), len(s), s :: DictType[str, int] == s, scores(5).total]
    = ["scores", 2, true, 5]
    refused := func(f) { try { f(); return false } catch { return true } }
    refused(() => { s.caio = "x" })            // a write is checked
    = true
    const flags = type dict[str, bool]
    typeName(flags(; on=true))
    = "dict[str, bool]"
}

example enum_keys {
    enum Svc { Kitchen, Bath }
    E := DictType[Svc, bool]
    [E(; Kitchen=true), E([Svc.Bath, false], [Svc.Kitchen, true]), {Kitchen: true} :: E]
    = [{Kitchen: true}, {Bath: false, Kitchen: true}, {Kitchen: true}]
}
```
