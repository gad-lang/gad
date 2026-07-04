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
a, b, c := func() { return 1, 2, 3 }()   // multiple return values
```

Because functions return a single value, "multiple return values" is really an
array being destructured. With `=`, you can assign into dict/array elements too:

```go
m := {}
var z
m.y, z = [1, 2]   // m == {y: 1}, z == 2
```

### Dicts

A `(; …)` pattern destructures a dict by key. Each entry is `name`, or
`target:key` to bind a key to a differently named variable, with optional
defaults and a trailing `**rest` to capture everything else.

```go
(;a, x:b, r=2, **other) := {a: 2, x: 3, q: 4}
// a == 2          (key "a")
// b == 3          (target b ← key "x")
// r == 2          (default; key "r" missing)
// other == {q: 4} (the rest)
```

### Mixed (positional + named)

A full `( positional ; named )` pattern destructures a `MixedParams`-shaped
value — the kind produced by the `(values...; name=value...)` literal. Both
sides accept a trailing `**rest`.

```go
x := (1, 2, *[3, 4]; c=5, **{d: 6})
(a, b, **pos; c, d2:d, r=2, **named) := x
// a == 1, b == 2, pos == [3, 4]
// c == 5, d2 == 6 (target d2 ← key "d"), r == 2 (default), named == {}
```

The naming rule is the same everywhere: `target:key` binds the value under
`key` to the variable `target`.
