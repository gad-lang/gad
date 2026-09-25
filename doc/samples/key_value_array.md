
# KeyValue Arrays

`keyValue` (`[k = v]`) and `keyValueArray` (`(; … )`) are an ordered list of
`key = value` pairs (duplicate keys allowed) — the literal form behind named
arguments. Part of the [Collections](collections.gad) chapter.

## A single pair

`[k = v]` is a `keyValue`: `.k` is the key, `.v` the value and `.array` the
`[k, v]` pair. The `=` is what makes it one — `["color"]` is a plain array.

```gad
pair := [color = "red"]
[pair.k, pair.v, pair.array, typeName(pair), typeName(["color"])]
// => ["color", "red", ["color", "red"], "keyValue", "array"]
```

## Ordered pairs

`(; … )` is a `keyValueArray`. Keys may be identifiers, strings, numbers,
booleans or a dynamic `[(expr)=v]`. The `keyValueArray(pair…)` builtin builds
one from `keyValue` pairs.

```gad
kva := (; a = 1, b = 2)
anyKeys := (; name = "x", "a b" = 1, 42 = 2, true = 3, [("k" + "1") = 9])
built := keyValueArray([a=1], keyValue("b", 2))
[str(kva), str(anyKeys), str(built)]
// => ["(;a=1, b=2)", "(;name=\"x\", \"a b\"=1, 42=2, true=3, k1=9)", "(;a=1, b=2)"]
```

## Flags

A bare key is a flag (`key` == `key=yes`); `key=no` drops the entry.
`kva.flag(name)` reads one.

```gad
flags := (; debug, verbose = no, level = 3)
[str(flags), flags.flag("debug"), flags.flag("verbose")]
// => ["(;debug, level=3)", true, false]
```

## Function values

A value may be a function or closure, written in method form.

```gad
fns := (; greet() => "hi", add(a, b) => a + b, twice(x) { return x * 2 })
[fns[0].v(), fns[1].v(2, 3), fns[2].v(4)]
// => ["hi", 5, 8]
```

## Typed keys

`name Type` carries the declared type as metadata on the key — the same form
that types a function's named parameters (`func(; n int = 0)`).

```gad
typed := (; id int, label str = "none")
[typed[0].k.name, len(typed[0].k.types), typed[1].v]
// => ["id", 1, "none"]
```

## Duplicates, lookup and mutation

Duplicate keys are preserved; `values(name)` / `delete(name)` act on every entry
of a name. Indexing yields the `keyValue`, and entries are mutable.

```gad
dup := (; tag = 1, tag = 2, name = 3)
found := [dup.values("tag"), dup.values()]
removed := str(dup.delete("tag"))
m := (; x = 1)
m[0].v = 99                          // indexing yields the keyValue
[found, removed, str(m)]
// => [[[1, 2], [1, 2, 3]], "(;name=3)", "(;x=99)"]
```

## Spread, iteration and conversion

`**kva` merges another key-value array; `for k, v in kva` iterates the pairs,
`dict(kva)` converts it, and — being exactly the named-argument form — it
spreads into a call's named arguments.

```gad
base := (; a = 1, b = 2)
merged := (; head = 0, **base)
show := func(; a = 0, b = 0, **rest) => [a, b, dict(rest)]
[str(merged), dict(base), show(; **(; a = 1, b = 2, c = 3))]
// => ["(;head=0, a=1, b=2)", {a: 1, b: 2}, [1, 2, {c: 3}]]

for k, v in base {
    println(k, "=", v)
}
```

Output:

```text
a = 1
b = 2
```

## Example — `key_value_array.gad`

```gad
pair := [color = "red"]
[pair.k, pair.v, pair.array, typeName(pair), typeName(["color"])]

kva := (; a = 1, b = 2)
anyKeys := (; name = "x", "a b" = 1, 42 = 2, true = 3, [("k" + "1") = 9])
built := keyValueArray([a=1], keyValue("b", 2))
[str(kva), str(anyKeys), str(built)]

flags := (; debug, verbose = no, level = 3)
[str(flags), flags.flag("debug"), flags.flag("verbose")]

fns := (; greet() => "hi", add(a, b) => a + b, twice(x) { return x * 2 })
[fns[0].v(), fns[1].v(2, 3), fns[2].v(4)]

typed := (; id int, label str = "none")
[typed[0].k.name, len(typed[0].k.types), typed[1].v]

dup := (; tag = 1, tag = 2, name = 3)
found := [dup.values("tag"), dup.values()]
removed := str(dup.delete("tag"))
m := (; x = 1)
m[0].v = 99                          // indexing yields the keyValue
[found, removed, str(m)]

base := (; a = 1, b = 2)
merged := (; head = 0, **base)
show := func(; a = 0, b = 0, **rest) => [a, b, dict(rest)]
[str(merged), dict(base), show(; **(; a = 1, b = 2, c = 3))]

for k, v in base {
    println(k, "=", v)
}

return str(kva)
```
