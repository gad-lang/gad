
# keyValue

A **keyValue** is one key/value pair. Its literal is `[a=1]` (the key a
name, a str, a number or a computed `[(expr)=v]`).

## Constructors

`keyValue(key, value)`.

```gad
[keyValue("a", 1), [b=2], str(keyValue("c", yes))]
= [[a=1], [b=2], "[c]"]
```

## Properties

`kv.k` is its key, `kv.v` its value, `kv.array` the pair `[k, v]`.

```gad
kv := keyValue("a", 1)
[kv.k, kv.v, kv.array]
= ["a", 1, ["a", 1]]
```

## See also

- [Key-value arrays](../key_value_array.md), [dict](dict.md) (`d += kv`).

## Type

### keyValue

```gad
keyValue type
```

#### Constructors

```gad
keyValue(…)
```

#### Operators

```gad
a < b
a <= b
a > b
a >= b
```

## Example — `keyValue.gad`

```gad
example constructors {
    [keyValue("a", 1), [b=2], str(keyValue("c", yes))]
    = [[a=1], [b=2], "[c]"]
}

example properties {
    kv := keyValue("a", 1)
    [kv.k, kv.v, kv.array]
    = ["a", 1, ["a", 1]]
}
```
