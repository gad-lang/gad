# Collections

[← Back to index](README.md)

Gad's composite types are **arrays** (ordered lists) and **dicts** (string-keyed
maps). This chapter also covers comprehensions, spread/merge literals and
destructuring.

## Arrays

```go
a := [1, 2, 3]
println(a[0], a[2])   // 1 3
a[1] = 20             // mutate in place
println(len(a))       // 3
a += [4, 5]           // concatenate
println(a)            // [1, 20, 3, 4, 5]
```

Slicing returns a sub-array; negative indices count from the end:

```go
[1, 2, 3, 4, 5][1:3]   // [2, 3]
[1, 2, 3, 4, 5][:-1]   // [1, 2, 3, 4]
```

## Dicts

A dict maps string keys to values. Items are separated by commas or newlines.

```go
m := {a: 1, "b": 2, c: 3}
println(m.a, m["b"])   // 1 2
m.d = 4                // add a key
delete(m, "a")         // remove a key
for k, v in m { println(k, v) }
```

Nested dicts can be written with block-style nesting:

```go
config := ({
    server {
        host: "localhost"
        port: 8080
    }
})
println(config.server.port)   // 8080
```

(The outer parentheses are required because a bare `{` at statement position
starts a block, not a dict.)

### Keyword keys

A keyword (`class`, `if`, `else`, `func`, `met`, `meti`, `false`, `nil`, …) may
be used as a **bare name** in any key position — it is taken as the string of its
spelling, not as the keyword. This holds for the selector `.name`, a dict key, a
key-value `[name=value]`, and a key-value-array key `(;name=value)` (including a
bare-flag key with no value):

```go
m := {class: 1, else: 2, func: 3}   // dict keys
m.class                             // 1   (selector)
m["else"]                           // 2

[class = 1]                         // key-value with key "class"
(;class = 1, if, else, false, nil)  // key-value-array; all are string keys
```

Only the **key** position is affected; on the value side keywords keep their
meaning, so in `(;x = false)` the value is the boolean `false`.

## Comprehensions

Array comprehensions build an array from an iterable, with an optional filter:

```go
[i * i for i in [1, 2, 3]]            // [1, 4, 9]
[i * i for i in [1, 2, 3] if i > 1]   // [4, 9]
[i + j for i in [1, 2] for j in [10, 20]]  // [11, 21, 12, 22]
```

Dict comprehensions build a dict. **Keys are static (literal) by default**; wrap
the key in `[ ]` to compute it. The special name `_` refers to the dict being
built, enabling accumulation.

```go
// computed key, computed value
{[v]: k for k, v in {a: 1, b: 2}}     // {1: "a", 2: "b"}

// static keys mixed with computed ones; `_` accumulates
{x: 10, [i]: i * i, z: (_.z ?? 0) + i for i in [1, 2, 3]}
// {1: 1, 2: 4, 3: 9, x: 10, z: 6}
```

## Spread and Merge Literals

`*expr` inside an array literal splices the elements of `expr` in place. A
leading spread produces a fresh copy.

```go
a := [1, 2]
b := [3, 4]
[0, *a, *b, 5]   // [0, 1, 2, 3, 4, 5]
```

`*expr` inside a dict literal merges another dict's entries; later entries win.

```go
{a: 1, *{b: 2, c: 3}, d: 4}     // {a: 1, b: 2, c: 3, d: 4}
{a: 1, *{a: 9, b: 2}}           // {a: 9, b: 2}   (merge overrides)
```

## Destructuring

### Arrays

Assign array elements to several variables at once. Missing elements become
`nil`. Use `:=` to declare or `=` to assign existing variables.

```go
x, y := [1, 2]        // x == 1, y == 2
x, y, z := [1, 2]     // z == nil
[a, b] := [1, 2]      // bracket form (also destructures a single [x])
c, d, e := func() { return 1, 2, 3 }()   // multiple return values
```

A trailing `*rest` (single star, last target only) collects the remaining
elements into a fresh array; the fixed targets before it still pad with `nil`:

```go
a, b, *rest := [1, 2, 3, 4]     // a == 1, b == 2, rest == [3, 4]
[a, b, *rest] := [1, 2, 3, 4]   // same, bracket form
a, b, *rest := [1]              // a == 1, b == nil, rest == []
```

The right side may also be **several comma-separated expressions** instead of a
single array (parallel multi-value assignment) — they are gathered and
destructured the same way, so spreads flatten and `*rest` still applies. Targets
may be selectors or indexes:

```go
a, b := 1, 2                 // a == 1, b == 2
a, b, *rest := 1, 2, 3, 4    // a == 1, b == 2, rest == [3, 4]
a, b, *rest := 1, *[2, 3]    // a == 1, b == 2, rest == [3]
m.x, y = 1, 2                // m.x == 1, y == 2
```

Leniency matches array destructuring: extra values are dropped and missing ones
become `nil` (`a, b, c := 1, 2` gives `c == nil`). Only the multi-target forms
qualify; a single target with several values (`x = 1, 2`) is an error.

Because functions return a single value, "multiple return values" is really an
array being destructured. With `=`, you can assign into dict/array elements too:

```go
m := {}
var z
m.y, z = [1, 2]   // m == {y: 1}, z == 2
```

Named data is destructured by key, **key on the left** (like TypeScript object
destructuring). Each entry is `key` (bind key `key` to variable `key`),
`key: target` (bind key `key` to a differently named variable), `name = default`
(fallback used when the key is absent), or a trailing `**rest` that collects the
remaining keys into a dict.

There are two interchangeable brackets — `(; … )` and `{ … }` — with identical
semantics:

```go
(;a, x:b, r=2, **other) := {a: 2, x: 3, q: 4}   // parens form
{ a, x: b, r = 2, **rest } := {a: 2, x: 3, q: 4} // curly form (TypeScript-like)
// a == 2          (key "a")
// b == 3          (key "x" → variable b)
// r == 2          (default; key "r" missing)
// other / rest == {q: 4} (the rest, as a dict)
```

Use `:=` to declare new variables or `=` to assign existing ones. A statement
that starts with `{ … } :=`/`=` is a destructuring; a bare `{ … }` is still a
block.

`const` and `var` also accept a destructuring pattern (both the `{ … }` and
`[ … ]` forms), so the bound names — including a `**rest` dict or `*rest` array —
follow the keyword's mutability:

```go
const { x, **rest } = {x: 1, y: 2, z: 3}   // x == 1, rest == {y: 2, z: 3}
const [ a, b, *rest ] = [1, 2, 3, 4]       // a == 1, b == 2, rest == [3, 4]
var { host, port = 80 } = cfg              // mutable bindings
```

### Any named source

Both `(; … )` and `{ … }` run the source through the default `dict()`
constructor first, so they work on **any** named data — a dict, a module, a
key-value array (`(; … )` value), named args, or anything convertible to a dict:

```go
strings := import("strings")
{ toUpper, hasPrefix } := strings          // a module

kva := (; a = 1, b = 2)
{ a, b } := kva                            // a key-value array

serve := func(; **opts) {
	{ host, port = 80 } := opts            // named args
	return host + ":" + str(port)
}
```

### Mixed (positional + named)

## See also

For runnable collection examples, see:
- `samples/04_collections.gad` — arrays, dicts, spread literals, iteration
- `samples/05_comprehensions.gad` — array and dict comprehensions
- `samples/22_key_value_array.gad` — keyValue and keyValueArray
- `samples/27_destructuring.gad` — array and named destructuring

A `MixedParams` value carries both positional and named parts (the kind produced
by the `(values...; name=value...)` literal), so it uses the full
`( positional ; named )` pattern. The positional side takes a trailing `*rest`
(a single star, like a variadic parameter `func(a, *rest)`) and the named side a
trailing `**rest`; the named side is key-on-the-left like the forms above.

```go
x := (1, 2, *[3, 4]; c=5, **{d: 6})
(a, b, *pos; c, d:d2, r=2, **named) := x
// a == 1, b == 2, pos == [3, 4]
// c == 5, d2 == 6 (key "d" → variable d2), r == 2 (default), named == {}
```
