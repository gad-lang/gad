
# dict

A **dict** maps str keys to values of any types. Its literal is
`{a: 1, "b c": 2}`; a typed dict (`type scores dict[str, int]`) checks its
keys and values (see [typed dicts](../typed_dicts.md)).

## Constructors

`dict(; **named)` of named arguments; `dict(o)` of a dict-like value (a
key-value array, a dict).

```gad
[dict(; a=1), dict((;b=2)), dict({c: 3})]
= [{a: 1}, {b: 2}, {c: 3}]
```

## Operators

`+` merges (a new dict: the right side's entries win; a `keyValue` sets one
entry); `in` tests a key. A dict indexes by key (`d.a`, `d["b c"]`) and is
iterated (`for k, v in d`).

```gad
d := {a: 1} + {b: 2}
d += keyValue("c", 3)
[d, "a" in d, d.b, d["c"], len(d)]
= [{a: 1, b: 2, c: 3}, true, 2, 3, 3]
```

## See also

- [Collections](../collections.md), [key-value arrays](../key_value_array.md).

## Type

### dict

```gad
dict type
```

#### Constructors

```gad
dict(…)
```

#### Operators

```gad
a + b
a ++ b
a - b
a < b
a <= b
a > b
a >= b
a in b
```

## Example — `dict.gad`

```gad
example constructors {
    [dict(; a=1), dict((;b=2)), dict({c: 3})]
    = [{a: 1}, {b: 2}, {c: 3}]
}

example operators {
    d := {a: 1} + {b: 2}
    d += keyValue("c", 3)
    [d, "a" in d, d.b, d["c"], len(d)]
    = [{a: 1, b: 2, c: 3}, true, 2, 3, 3]
}
```
