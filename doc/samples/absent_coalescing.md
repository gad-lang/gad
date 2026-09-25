
# Absent-Coalescing (`!?` / `!?=`)

`!?` / `!?=` are the existence-based counterparts of the nullish `??` / `??=`:
they test whether a KEY is present in its container, where `??` tests whether a
VALUE is nil (a key present with a `nil` value still counts as present). Part of
the [Operators](user_operators.gad) chapter. See doc/operators.md for detailed
documentation.

## Value form

`a.b !? v` is the member when the key exists (even if nil), otherwise `v`.

```gad
[
    ({b: 5}).b !? 9,                         // present
    ({b: nil}).b !? 9,                       // present with a nil value
    ({b: nil}).b ?? 9,                       // contrast: `??` is nil-based
    ({}).b !? 9,                             // absent
]
// => [5, nil, 9, 9]
```

## Assignment form

`a.b !?= v` assigns only when the key is absent; `v` is evaluated lazily.

```gad
a := {b: nil}
a.b !?= 9                // no-op — the key is already present
e := {}
e.b !?= 9                // assigns — the key is absent
[a.b, e.b]
// => [nil, 9]
```

## Deep paths

`!?` reads a deep path safely (creating nothing); `!?=` auto-creates the
intermediate dicts.

```gad
box := {}
missing := box.x.y.z !? "missing"   // nothing is created
box.x.y.z !?= 42
[missing, box]
// => ["missing", {x: {y: {z: 42}}}]
```

## Class instances

Class instances are containers too — their fields are the keys.

```gad
class Point { x = 0 }
p := Point()
before := ["x" in p, "y" in p, p.y !? -1]  // field y is absent
p.y !?= 7
[before, p.y]
// => [[true, false, -1], 7]
```

## `global` defaults

In a `global (…)` declaration, `= v` fills a name when it is nil-or-absent and
`!?= v` only when it is absent. With no host globals here every name is
absent, so both apply.

```gad
global (page = 1, user !?= "guest")
#"page={page} user={user}"
// => page=1 user=guest
```

## Example — `absent_coalescing.gad`

```gad
[
    ({b: 5}).b !? 9,                         // present
    ({b: nil}).b !? 9,                       // present with a nil value
    ({b: nil}).b ?? 9,                       // contrast: `??` is nil-based
    ({}).b !? 9,                             // absent
]

a := {b: nil}
a.b !?= 9                // no-op — the key is already present
e := {}
e.b !?= 9                // assigns — the key is absent
[a.b, e.b]

box := {}
missing := box.x.y.z !? "missing"   // nothing is created
box.x.y.z !?= 42
[missing, box]

class Point { x = 0 }
p := Point()
before := ["x" in p, "y" in p, p.y !? -1]  // field y is absent
p.y !?= 7
[before, p.y]

global (page = 1, user !?= "guest")
#"page={page} user={user}"
```
