# Properties (`prop`)

A **property** is a named, callable value introduced by the `prop` keyword. It
holds a *getter* (a method with no parameters) and optional *setters* (a method
with one parameter, dispatched by the argument type). A property behaves like a
closure with an assignable value: you can call it, read/write its virtual `.v`
field, or store it in a container where indexing delegates to it.

Properties are also available through the
[`Prop`](values-and-types.md#properties) constructor for building them
programmatically.

## Declaring properties

```go
var value
prop x {
  ()      => value                  // getter:  x()
  (v)     { value = v }             // setter:  x(v)
  (v int) { value = "int= " + v }   // typed setter, dispatched by argument type
}

x("a")   // setter runs
x()      // "a"
x(1)     // typed (int) setter
x()      // "int= 1"
```

A single-accessor property may drop the braces: `prop pi() => 3.14`.

### `prop => expr` — a getter-only property (prop as a closure)

`prop => expr` is a read-only property whose getter is `expr`. It reads live
(the expression is evaluated on each access) and has no setter, so writing it is
an error. It works anonymously or named:

```go
var _x = 5
x := prop => _x        // anonymous, assigned to a variable
prop y => _x           // named statement

x.v   // 5   — getter
_x = 9
x.v   // 9   — live
y()   // 9
```

## The virtual `.v` field

A property exposes a virtual `v` field for value access without an explicit
call: `x.v` runs the getter (like `x()`) and `x.v = value` runs the matching
setter (like `x(value)`). Calling the property directly still works.

```go
var stored
prop x { () => stored; (n) { stored = n } }

x.v = 10   // setter — same as x(10)
x.v        // 10   — getter, same as x()
```

Reading `.v` on a getter-only property works; writing it errors (no setter).

**Prefer `.v` over an explicit call.** `.v` invokes the same accessor as `x()` /
`x(n)`, so it adds no function-call overhead — and a read through `.v` is even a
touch cheaper than a call, because the VM's index-get dispatch is lighter than
its call dispatch. Reads via `.v` win slightly; writes are on par. Combined with
being clearer to read, `.v` is the recommended default.

Benchmark — 50 000 get/set operations on a `prop` (lower is better; see
`BenchmarkProp*`):

```
get   x()   █████████████████████████████████████  17.5 ms
      x.v   ████████████████████████████████████    17.0 ms   (~3% faster)

set   x(n)  █████████████████████████████████████   25.5 ms
      x.v=  ██████████████████████████████████████  26.6 ms   (~4% more, 1 alloc/op)
```

The difference is within a few percent either way — `.v` and a direct call are
effectively equal in cost, so choose `.v` for readability. (Note: `x.v = n` runs
only the setter — it does not read the getter first.)

## Properties as container members (computed properties)

When a `Prop` is stored at a container key, indexing that key **delegates** to
the prop — reading runs its getter, assigning runs its matching setter. This is
the computed-property (accessor) pattern, like a JavaScript getter/setter:

```go
var (
  v = 1,
  d = { x: prop { () => v; (val) { v = val } } }
)

d.x        // 1     — runs the getter
d.x = 2    // v = 2 — runs the setter
d.x        // 2
```

A getter-only prop stored at a key is read-only through it:

```go
var (_x = 10, d = { x: prop => _x })
d.x        // 10  — getter (live)
_x = 20
d.x        // 20
// d.x = 5 // error: no setter
```

Delegation works on any container (dict, module, custom index getters) **except
`array` and class instances**, whose element/field access always returns the
stored value verbatim (a class instance already resolves its own class
properties). Chained access delegates at the leaf: `d.a.x`.

## Exporting properties (module live bindings)

A module can [export](modules.md#exporting-properties) a property so member
access on the imported module delegates to it.

`export prop name = init` declares a real module local `var name = init` and
exports a read/write property over it, giving a **live binding**: writing
`mod.name` from outside changes the module's `name`, and module functions
closing over `name` observe the change (and vice versa).

```go
// counter.gad
export prop x = 10
export getX() => x
```

```go
c := import("./counter.gad")
c.x         // 10  — getter
c.getX()    // 10
c.x = 12    // setter mutates the module's x
c.getX()    // 12  — the change is observed
```

`export prop name => expr` exports a **read-only** live binding (getter only):

```go
var _v = 7
export prop x => _v      // read-only view of _v
export bump() { _v = _v + 1 }
```

## Raw access with `reflect`

The [`reflect`](reflect.md) module reads and writes a key **without** delegating
to a stored prop — the functional analog of JavaScript `Reflect.get` /
`Reflect.set`:

```go
reflect.get(d, "x")    // the Prop itself (getter not run)
reflect.set(d, "x", 3) // overwrites the key with 3, removing the prop
```

## Runnable example

See `samples/31_properties.gad` for a runnable walk-through, and
`samples/modules/counter.gad` for a module live binding.
