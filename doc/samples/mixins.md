
# Mixins (`mixin`)

A **mixin** is a reusable bundle of members — fields, properties and methods —
that a [class](class/classes.gad) pulls in with a `use` clause. A mixin has the
basic structure of a class (parents, fields, `props`, `methods`) and parses with
the same member syntax, but it is **not instantiable**: its members become the
using class's own.

```gad
mixin Counter {
    count = 0
    props   { doubled => this.count * 2 }
    methods { inc() { this.count += 1 } }
}

class Widget {
    use Counter        // pulls count / doubled / inc into Widget
    name = "?"
}
```

## Lowering

Like a class lowers to a `Class("Name", (cls, define) => define(; …))` call, a
mixin lowers to a `Mixin("Name", (mx, define) => define(; …))` call (the
`gad.Mixin` builtin). A class's `use A, B` clause becomes the `mixins=[…]`
argument of its own `Class(…)` call.

## `use`

`use` lists the mixins a class pulls in. Names are ident, selector (`pkg.M`) or
index (`mods["x"].M`) expressions, separated by commas (a comma may precede a
newline) — the list ends at a newline or `;`. A long list wraps, indented under
`use`. `use` is a contextual identifier, not a reserved word, so `use` stays
usable as an ordinary name elsewhere.

## Rules

- **Field init order** — a using class initialises its mixin fields **first**, in
  declaration order starting from the parent mixins, before its own fields.
- **No duplicates** — a mixin reachable more than once (through the `use` list or
  a mixin hierarchy) is merged only **once**; the first occurrence wins. This is
  resolved where the class uses the mixin, silently, with no error — a mixin
  itself accepts everything, even repeated parents.
- **Name conflicts** — a member the class (or an earlier mixin) already declares
  is kept; the later mixin's same-named member is skipped.

## The `this` interface

An optional `this { … }` block — an [interface](interfaces.gad) body written
without the `interface` keyword — declares what the `this` of the mixin's
properties and methods must satisfy, so a method can call a member the final
class provides. The `this` of a mixin method is always a class instance of the
using class.

## Reflection

A mixin mirrors a class's reflection attributes: `M.@fields`, `M.@props`,
`M.@methods`, `M.@parents`, `M.@module`, `M.@name`. It also exposes cached
`Interface` values: `M.@this` (the `this { … }` block, nil without one),
`M.@membersInterface` (own members), `M.@classInterface` (the using-class
contract: `*@this ; *parent.@interface`) and `M.@interface` (the whole contract,
extending both). Any interface has `iface.@flat` — the extends graph flattened
into one interface. A using class exposes `C.@mixins` (like `C.@parents`). See
[the class-samples mixin tests](class/mixins_test.gad) for the full reflection
surface and the `this`-receiver contract validation.

This whole sample is a runnable tour (see the Example below).

## Example — `mixins.gad`

```gad
// --- Basic use -------------------------------------------------------------
/**
`use` pulls a mixin's field, property and method into the class as its own. A
mixin field default can be overridden by a value passed at construction.
**/
mixin Counter {
    count = 0
    props   { doubled => this.count * 2 }
    methods { inc() { this.count += 1 } }
}

class Widget {
    use Counter
    name = "?"
}

w := Widget(; name = "box", count = 3)
println("name:        ", w.name)      // box  — Widget's own field
println("doubled:     ", w.doubled)   // 6    — Counter's property (3 * 2)
w.inc()                               // Counter's method
println("after inc:   ", w.count)     // 4    — Counter's field

// --- Field init order ------------------------------------------------------
/**
Mixin field defaults initialise before the class's own, in declaration order —
observed here through the order a shared log records each default's evaluation.
**/
log := []
mark := func(s) { log = log + [s]; return s }

mixin M { a = mark("a"); b = mark("b") }
class Ordered { use M; z = mark("z") }

Ordered()
println("init order:  ", log)         // [a, b, z] — mixin fields first

// --- Parent mixins ---------------------------------------------------------
/**
A mixin can extend parent mixins with `*Parent` spreads. A class that uses the
child gains the parents' members too, and the parents' fields initialise first.
**/
mixin Timestamped { created = 0 }
mixin Identified  { id = 0 }
mixin Entity {
    *Timestamped
    *Identified
}

class Record { use Entity }
r := Record(; created = 100, id = 7)
println("parents:     ", len(Entity.@parents))  // 2
println("inherited:   ", r.created, r.id)        // 100 7

// --- Deduplication ---------------------------------------------------------
/**
A mixin reachable more than once (here via `Sub` and directly) is merged only
once — no duplicate-member error. The first occurrence wins.
**/
mixin Base { kind = "base" }
mixin Sub  { *Base }

class Both { use Sub, Base }          // Base reachable twice
println("dedup value: ", Both().kind) // base
println("mixins count:", len(Both.@mixins)) // 2 — both use-entries recorded

// --- The `this` interface --------------------------------------------------
/**
A `this { … }` block types the `this` of the mixin's props/methods via an
interface, so a method can call a member the final class provides. `this` is
always a class instance of the using class.
**/
mixin Described {
    this { label() <str> }            // require the receiver to have label()
    methods { describe() => "<" + this.label() + ">" }
}

class Tag {
    use Described
    methods { label() => "tag" }
}
println("described:   ", Tag().describe())  // <tag>

// --- Anonymous mixin -------------------------------------------------------
/**
A mixin can be anonymous, bound to a const with `const M = mixin { … }`.
**/
const Versioned = mixin { version = 1 }
class Doc { use Versioned }
println("anonymous:   ", Doc().version)  // 1

// --- Reflection ------------------------------------------------------------
/**
Reflection attributes mirror a class's. `@interface` returns a cached `Interface`
value named `Name$interface` reflecting the declared members with their types and
accessor kinds.
**/
mixin Shape {
    sides int = 0
    props   { area => 0 }
    methods { draw() => nil }
}
println("mixin name:  ", Shape.@name)                    // Shape
println("mixin fields:", collect(keys(Shape.@fields)))   // [sides]
println("@interface:  ", bool(Shape.@interface :: gad.Interface)) // true
println("cached:      ", Shape.@interface == Shape.@interface) // true

return w.count
```
