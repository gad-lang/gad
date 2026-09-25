
# Destructuring

Bind several variables at once from arrays and named data. Gad has three
destructuring shapes:

  1. arrays        `x, y := [1, 2]`
  2. named `(; …)` `(; key: target, r = 2, **rest) := source`  (key on the left)
  3. named `{ … }` `{ key: target, r = 2, **rest } := source`  (TypeScript order)

Forms 2 and 3 are the same feature with different brackets — both key-on-the-
left. They work on any named data — dicts, modules, key-value arrays, named
args — because the source is run through the default `dict()` constructor first.
Part of the [Collections](collections.gad) chapter.

## 1. Arrays

`x, y := [1, 2]` — a missing element binds `nil`. The bracket form `[a, b]`
works too (even with one element), a trailing `*rest` (the last target) collects
the remaining elements, and several values on the right need no brackets.
`const` / `var` accept any pattern (immutable / mutable bindings).

```gad
x, y := [10, 20]
a, b, c := [1, 2]                   // missing element -> nil
[first, second] := [100, 200]       // bracket form
head, next, *rest := [1, 2, 3, 4, 5] // trailing *rest
p, q, *tail := 1, 2, 3, 4           // parallel: no brackets on the right
const {tls} = {host: "h", tls: true}
const [lo, hi, *more] = [1, 9, 10, 11]
[[x, y], [a, b, c], [first, second], [head, next, rest], [p, q, tail], [tls, lo, hi, more]]
// => [[10, 20], [1, 2, nil], [100, 200], [1, 2, [3, 4, 5]], [1, 2, [3, 4]], [true, 1, 9, [10, 11]]]
```

## 2. Named, `(; … )` form — key on the left

`key: target` renames, `key = default` fills an absent key, and `**rest`
collects the remaining keys.

```gad
d := {host: "localhost", port: 8080, tls: true}
(; host, port:pn, timeout = 30, **extra) := d
// host    <- key "host"
// pn      <- key "port"    — rename, `key: target`
// timeout <- key "timeout" — default; the key is absent
// extra   <- the remaining keys
[host, pn, timeout, extra]
// => ["localhost", 8080, 30, {tls: true}]
```

## 3. Named, `{ … }` form — TypeScript order, key on the left

The same feature with braces; `**rest` is collected as a dict. `=` instead of
`:=` assigns to variables that already exist.

```gad
cfg := {host: "localhost", port: 8080, tls: true}
{ host, port: pt, timeout = 30, **others } := cfg
var (h, po)
{ host: h, port: po } = cfg         // `=` assigns existing variables
[host, pt, timeout, typeName(others), others, [h, po]]
// => ["localhost", 8080, 30, "dict", {tls: true}, ["localhost", 8080]]
```

## Any named data

The source is run through the default `dict()` constructor, so modules,
key-value arrays (`ToDictConverter`) and named args all destructure.

```gad
{ toUpper, hasPrefix } := import("strings")   // a module
{ a: ka, b: kb, **krest } := (; a = 1, b = 2, c = 3) // a key-value array
connect := func(; **opts) {                   // named args
    { host, port = 80 } := opts
    return host + ":" + str(port)
}
[toUpper("hi"), hasPrefix("hello", "he"), [ka, kb, krest], connect(; host = "example.com")]
// => ["HI", true, [1, 2, {c: 3}], "example.com:80"]
```

## Mixed positional + named

A **MixedParams** value is written like a call's argument list in parentheses —
`(values… ; name=value…)`, with `*` / `**` spreads and `(,)` for an empty one.
It carries BOTH parts: `.positional` (an array) and `.named` (a key-value
array), so it can be stored and later spread into a call.

```gad
args := (1, *[2, 3]; mode = "fast", **{debug: yes})
run := func(*xs; **opts) => [xs, dict(opts)]
[
    typeName(args),
    args.positional,
    str(args.named),
    run(*args.positional; **args.named),        // spread it into a call
    [len((,).positional), len((,).named)],      // the empty one
]
// => ["MixedParams", [1, 2, 3], "(;mode=\"fast\", debug)", [[1, 2, 3], {debug: on, mode: "fast"}], [0, 0]]
```

It destructures with the full `( positional ; named )` pattern — the positional
side takes a trailing `*rest` (a single star, like a variadic parameter
`func(a, *rest)`) and the named side a trailing `**rest`. For just the named side
of any value, use `(; … )` or `{ … }` on it.

```gad
mp := (1, 2, *[3]; user = "ann", role = "admin", team = "core")
(first, second, *restPos; user: who, role: r, **restNamed) := mp
[first, second, restPos, who, r, restNamed]
// => [1, 2, [3], "ann", "admin", {team: "core"}]
```

## Example — `destructuring.gad`

```gad
x, y := [10, 20]
a, b, c := [1, 2]                   // missing element -> nil
[first, second] := [100, 200]       // bracket form
head, next, *rest := [1, 2, 3, 4, 5] // trailing *rest
p, q, *tail := 1, 2, 3, 4           // parallel: no brackets on the right
const {tls} = {host: "h", tls: true}
const [lo, hi, *more] = [1, 9, 10, 11]
[[x, y], [a, b, c], [first, second], [head, next, rest], [p, q, tail], [tls, lo, hi, more]]

d := {host: "localhost", port: 8080, tls: true}
(; host, port:pn, timeout = 30, **extra) := d
// host    <- key "host"
// pn      <- key "port"    — rename, `key: target`
// timeout <- key "timeout" — default; the key is absent
// extra   <- the remaining keys
[host, pn, timeout, extra]

cfg := {host: "localhost", port: 8080, tls: true}
{ host, port: pt, timeout = 30, **others } := cfg
var (h, po)
{ host: h, port: po } = cfg         // `=` assigns existing variables
[host, pt, timeout, typeName(others), others, [h, po]]

{ toUpper, hasPrefix } := import("strings")   // a module
{ a: ka, b: kb, **krest } := (; a = 1, b = 2, c = 3) // a key-value array
connect := func(; **opts) {                   // named args
    { host, port = 80 } := opts
    return host + ":" + str(port)
}
[toUpper("hi"), hasPrefix("hello", "he"), [ka, kb, krest], connect(; host = "example.com")]

args := (1, *[2, 3]; mode = "fast", **{debug: yes})
run := func(*xs; **opts) => [xs, dict(opts)]
[
    typeName(args),
    args.positional,
    str(args.named),
    run(*args.positional; **args.named),        // spread it into a call
    [len((,).positional), len((,).named)],      // the empty one
]

mp := (1, 2, *[3]; user = "ann", role = "admin", team = "core")
(first, second, *restPos; user: who, role: r, **restNamed) := mp
[first, second, restPos, who, r, restNamed]
```
