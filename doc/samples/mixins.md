
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

A mixin field default can be overridden by a value passed at construction.

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

w := Widget(; name = "box", count = 3)
before := [w.name, w.doubled]       // Widget's own field, Counter's property
w.inc()                             // Counter's method
[before, w.count]
// => [["box", 6], 4]
```

A mixin can also be anonymous, bound to a const with `const M = mixin { … }`.

```gad
const Versioned = mixin { version = 1 }
class Doc { use Versioned }
Doc().version
// => 1
```

## Rules

- **Field init order** — a using class initialises its mixin fields **first**, in
  declaration order starting from the parent mixins, before its own fields.
- **No duplicates** — a mixin reachable more than once (through the `use` list or
  a mixin hierarchy) is merged only **once**; the first occurrence wins. This is
  resolved where the class uses the mixin, silently, with no error — a mixin
  itself accepts everything, even repeated parents.
- **Name conflicts** — a member the class (or an earlier mixin) already declares
  is kept; the later mixin's same-named member is skipped.

The init order, observed through a shared log that records each default's
evaluation:

```gad
log := []
mark := func(s) { log = log + [s]; return s }

mixin M { a = mark("a"); b = mark("b") }
class Ordered { use M; z = mark("z") }

Ordered()
log                                 // mixin fields first
// => ["a", "b", "z"]
```

A mixin reachable more than once (here via `Sub` and directly) is merged only
once:

```gad
mixin Base { kind = "base" }
mixin Sub  { *Base }

class Both { use Sub, Base }        // Base reachable twice: merged once
[Both().kind, len(Both.@mixins)]    // both use-entries are recorded
// => ["base", 2]
```

## The `this` interface

An optional `this { … }` block — an [interface](interfaces.gad) body written
without the `interface` keyword — declares what the `this` of the mixin's
properties and methods must satisfy, so a method can call a member the final
class provides. The `this` of a mixin method is always a class instance of the
using class.

```gad
mixin Described {
    this { label() <str> }          // require the receiver to have label()
    methods { describe() => "<" + this.label() + ">" }
}

class Tag {
    use Described
    methods { label() => "tag" }
}
Tag().describe()
// => <tag>
```

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

```gad
mixin Named { name = "?" }
mixin Shape {
    *Named
    sides int = 0
    props   { area => 0 }
    methods { draw() => nil }
}
flat := Shape.@interface.@flat      // the whole contract, flattened
[
    Shape.@name,
    collect(keys(Shape.@fields)),   // own fields only
    [f.name for f in flat.fields],
    [m.name for m in flat.methods],
    Shape.@interface == Shape.@interface,   // cached
]
// => ["Shape", ["sides"], ["name", "sides"], ["draw"], true]
```

A `met` added to a class after the fact sees every member — its own, those
merged from mixins, and inherited ones — on the `this` receiver.

```gad
class Card { use Shape; title = "" }
met Card.summary(this) => this.title + " (" + str(this.sides) + " sides, " + this.name + ")"
Card(; title="Ace", sides=4, name="c").summary()
// => Ace (4 sides, c)
```

## Parent mixins

A mixin extends parent mixins with `*Parent` spreads. A class that uses the
child gains the parents' members too, and the parents' fields initialise first.

```gad
mixin Timestamped { created = 0 }
mixin Identified  { id = 0 }
mixin Entity {
    *Timestamped
    *Identified
}

class Record { use Entity }
r := Record(; created = 100, id = 7)
[len(Entity.@parents), r.created, r.id]
// => [2, 100, 7]
```

### Parent lists

A `*` parent spread takes a mixin or an **array of mixins** (flattened, nesting
allowed), written inline or held in a variable: `*[A, B]`, `*parents`.

```gad
mixin Sized   { size = 1 }
mixin Colored { color = "red" }
mixin Labeled { label = "" }

// a literal list of parent mixins
mixin Styled { *[Sized, Colored] }

// a list held in a variable (nested arrays flatten)
styleParts := [Sized, [Colored, Labeled]]
mixin Styled2 { *styleParts }

class Button { use Styled2 }
b := Button(; label = "ok")
[len(Styled.@parents), len(Styled2.@parents), b.size, b.color, b.label]
// => [2, 3, 1, "red", "ok"]
```

## Example — `mixins.gad`

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

w := Widget(; name = "box", count = 3)
before := [w.name, w.doubled]       // Widget's own field, Counter's property
w.inc()                             // Counter's method
[before, w.count]

const Versioned = mixin { version = 1 }
class Doc { use Versioned }
Doc().version

log := []
mark := func(s) { log = log + [s]; return s }

mixin M { a = mark("a"); b = mark("b") }
class Ordered { use M; z = mark("z") }

Ordered()
log                                 // mixin fields first

mixin Timestamped { created = 0 }
mixin Identified  { id = 0 }
mixin Entity {
    *Timestamped
    *Identified
}

class Record { use Entity }
r := Record(; created = 100, id = 7)
[len(Entity.@parents), r.created, r.id]

mixin Base { kind = "base" }
mixin Sub  { *Base }

class Both { use Sub, Base }        // Base reachable twice: merged once
[Both().kind, len(Both.@mixins)]    // both use-entries are recorded

mixin Described {
    this { label() <str> }          // require the receiver to have label()
    methods { describe() => "<" + this.label() + ">" }
}

class Tag {
    use Described
    methods { label() => "tag" }
}
Tag().describe()

mixin Named { name = "?" }
mixin Shape {
    *Named
    sides int = 0
    props   { area => 0 }
    methods { draw() => nil }
}
flat := Shape.@interface.@flat      // the whole contract, flattened
[
    Shape.@name,
    collect(keys(Shape.@fields)),   // own fields only
    [f.name for f in flat.fields],
    [m.name for m in flat.methods],
    Shape.@interface == Shape.@interface,   // cached
]

class Card { use Shape; title = "" }
met Card.summary(this) => this.title + " (" + str(this.sides) + " sides, " + this.name + ")"
Card(; title="Ace", sides=4, name="c").summary()

mixin Sized   { size = 1 }
mixin Colored { color = "red" }
mixin Labeled { label = "" }

// a literal list of parent mixins
mixin Styled { *[Sized, Colored] }

// a list held in a variable (nested arrays flatten)
styleParts := [Sized, [Colored, Labeled]]
mixin Styled2 { *styleParts }

class Button { use Styled2 }
b := Button(; label = "ok")
[len(Styled.@parents), len(Styled2.@parents), b.size, b.color, b.label]

return w.count
```
