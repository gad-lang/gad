
# Properties (`prop`)

A **property** is a named, callable value introduced by the `prop` keyword. It
holds a *getter* (a method with no parameters) and optional *setters* (a method
with one parameter, dispatched by the argument type). A property behaves like a
closure with an assignable value: you can call it, read/write its virtual `.v`
field, or store it in a container where indexing delegates to it. Properties are
also available through the `Prop` constructor for building them programmatically.

## Declaring properties

The `prop` statement declares a named property: the getter takes no params,
setters take one and may be typed (dispatched by argument type). A
single-accessor property may drop the braces (`prop pi() => 3.14`).

```gad
var stored
prop value {
    ()      => stored                // getter
    (v)     { stored = v }           // setter (any value)
    (v int) { stored = "int: " + v } // typed setter, dispatched by arg type
}
value("hello"); a := value()         // setter, then getter
value(42);      b := value()         // typed (int) setter
println("value (str):", a)           // hello
println("value (int):", b)           // int: 42

prop pi() => 3.14159                 // read-only, braces dropped
println("pi():      ", pi())         // 3.14159
```

`prop => expr` is a read-only property whose getter is `expr`. It reads **live**
(re-evaluated on each access) and has no setter, so writing it is an error. It
works anonymously or named (`prop y => _x`).

```gad
var _live = 1
ro := prop => _live                  // anonymous, read-only, live
_live = 42
println("ro.v:      ", ro.v)         // 42 (live)
```

## The virtual `.v` field

A property exposes a virtual `v` field for value access without an explicit
call: `x.v` runs the getter (like `x()`) and `x.v = value` runs the matching
setter (like `x(value)`). **Prefer `.v`**: it invokes the same accessor as a
call, adds no call overhead (a read through `.v` is even a touch cheaper than a
call — the VM's index-get dispatch is lighter than its call dispatch), and reads
clearer. `x.v = n` runs only the setter — it does not read the getter first.

```gad
value.v = 3                          // setter via .v (preferred over value(3))
println("value.v:   ", value.v)      // int: 3 (getter via .v)
```

## Properties as container members (computed properties)

When a `prop` is stored at a container key, indexing that key **delegates** to
it — reading runs its getter, assigning runs its matching setter — the
computed-property (accessor) pattern, like a JavaScript getter/setter:

```gad
var (
    celsius = 20,
    temp = {
        c: prop { () => celsius; (v) { celsius = v } },
        f: prop { () => celsius * 9 / 5 + 32; (v) { celsius = (v - 32) * 5 / 9 } },
    },
)
println("c =", temp.c)               // 20 — getter runs
temp.c = 30                          // setter runs
println("f =", temp.f)               // 86 — computed from celsius
temp.f = 212                         // setter converts back
println("c =", temp.c)               // 100
```

A `prop` declared with `prop` is a plain value: storing it at a key shares the
**same** prop (both see one backing var). Delegation works on any container
(dict, module, custom index getters) **except `array` and class instances**,
whose access always returns the stored value verbatim. Chained access delegates
at the leaf (`d.a.x`).

```gad
var n = 1
prop counter { () => n; (v) { n = v } }
d := {x: counter}                    // store the prop by reference
d.x = 8                              // delegates -> runs counter's setter
println("d.x =", d.x, "counter() =", counter()) // 8 8 (shared backing var)

/// Arrays never delegate: a stored prop is the value itself.
p := prop { () => 99 }
println("array holds prop:", typeName([p][0])) // Prop
```

## Raw access with `reflect`

The [`reflect`](reflect.md) module reads and writes a key **without** delegating
to a stored prop — the functional analog of JavaScript `Reflect.get` /
`Reflect.set`.

```gad
println("raw get:", typeName(reflect.get(temp, "c"))) // Prop (getter not run)
reflect.set(temp, "c", 5)                             // replaces the prop
println("after raw set:", temp.c)                     // 5 (plain value now)
```

## Exporting properties (module live bindings)

`export prop name = init` declares a real module local `var name = init` and
exports a read/write property over it — a **live binding**: writing `mod.name`
from outside changes the module's `name`, and module functions closing over it
observe the change. `export prop name => expr` exports a read-only live binding.
See [Modules](26_embed.gad) and `samples/modules/counter.gad`.

```gad
// counter.gad
export prop x = 10
export getX() => x
// caller:  c := import("./counter.gad"); c.x = 12; c.getX()  // 12
```

## Example — `31_properties.gad`

```gad
var stored
prop value {
    ()      => stored                // getter
    (v)     { stored = v }           // setter (any value)
    (v int) { stored = "int: " + v } // typed setter, dispatched by arg type
}
value("hello"); a := value()         // setter, then getter
value(42);      b := value()         // typed (int) setter
println("value (str):", a)           // hello
println("value (int):", b)           // int: 42

prop pi() => 3.14159                 // read-only, braces dropped
println("pi():      ", pi())         // 3.14159

var _live = 1
ro := prop => _live                  // anonymous, read-only, live
_live = 42
println("ro.v:      ", ro.v)         // 42 (live)

value.v = 3                          // setter via .v (preferred over value(3))
println("value.v:   ", value.v)      // int: 3 (getter via .v)

var (
    celsius = 20,
    temp = {
        c: prop { () => celsius; (v) { celsius = v } },
        f: prop { () => celsius * 9 / 5 + 32; (v) { celsius = (v - 32) * 5 / 9 } },
    },
)
println("c =", temp.c)               // 20 — getter runs
temp.c = 30                          // setter runs
println("f =", temp.f)               // 86 — computed from celsius
temp.f = 212                         // setter converts back
println("c =", temp.c)               // 100

var n = 1
prop counter { () => n; (v) { n = v } }
d := {x: counter}                    // store the prop by reference
d.x = 8                              // delegates -> runs counter's setter
println("d.x =", d.x, "counter() =", counter()) // 8 8 (shared backing var)

/// Arrays never delegate: a stored prop is the value itself.
p := prop { () => 99 }
println("array holds prop:", typeName([p][0])) // Prop

println("raw get:", typeName(reflect.get(temp, "c"))) // Prop (getter not run)
reflect.set(temp, "c", 5)                             // replaces the prop
println("after raw set:", temp.c)                     // 5 (plain value now)

return value()
```
