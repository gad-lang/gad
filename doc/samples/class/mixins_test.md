
# Mixins

A **mixin** is a reusable bundle of members — fields, properties and methods —
that a class pulls in with a `use` clause. A mixin has the basic structure of a
class (parents, fields, props, methods) and parses with the same member syntax,
but it is **not instantiable**: its members become the using class's own.

```
mixin Counter {
    count = 0
    props   { doubled => this.count * 2 }
    methods { inc() { this.count += 1 } }
}

class Widget {
    use Counter        // pulls Counter's count/doubled/inc into Widget
    name = "?"
}
```

The compiler lowers a mixin to a `Mixin("Name", (mx, define) => define(; …))`
call (the `gad.Mixin` builtin), mirroring how a class lowers to `Class(…)`. A
class's `use A, B` becomes the `mixins=[…]` argument of its own `Class(…)` call.

Key rules:

- **Field init order** — a using class initialises its mixin fields **first**, in
  declaration order starting from the parent mixins, before its own fields.
- **No duplicates** — a mixin appearing more than once across the `use` list or a
  mixin hierarchy is merged only once (the first occurrence wins); this is checked
  where the class uses the mixin, silently, with no error.
- **`this` interface** — an optional `this { … }` block (an interface body without
  the `interface` keyword) declares what the `this` of the mixin's props/methods
  must satisfy, so a method can call a member the final class provides.

Reflection attributes mirror a class's: `MyMixin.@fields`, `.@props`, `.@methods`,
`.@parents`, `.@module`, `.@name`, and `.@interface` (a cached `Interface` value
named `Name$interface` reflecting the declared members). A using class exposes
`.@mixins` (like `.@parents`).

Run it: `gad test samples/class`

## Example — `mixins_test.gad`

```gad
/**
`use` pulls a mixin's field, property and method into the class as its own. A
mixin field default can be overridden by a value passed at construction.
**/
test "use merges a mixin's members" {
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
    t.equal("box", w.name)     // Widget's own field
    t.equal(6, w.doubled)      // Counter's property: 3 * 2
    w.inc()                    // Counter's method
    t.equal(4, w.count)        // Counter's field, now incremented
}

/**
Mixin fields initialise before the class's own, in declaration order — observed
here through the order a shared log records each default's evaluation.
**/
test "mixin fields initialise first" {
    log := []
    mark := func(s) { log = log + [s]; return s }

    mixin M { a = mark("a"); b = mark("b") }
    class C { use M; z = mark("z") }

    C()
    t.equal(["a", "b", "z"], log)
}

/**
A mixin can extend parent mixins with `*Parent` spreads. A class that uses the
child gains the parents' members too, and the parents' fields initialise first.
**/
test "mixin parents" {
    log := []
    mark := func(s) { log = log + [s]; return s }

    mixin P { p = mark("p") }
    mixin Q { *P; q = mark("q") }
    class C { use Q; z = mark("z") }

    c := C()
    t.equal(1, len(Q.@parents))            // Q extends one parent (P)
    t.equal(["p", "q", "z"], log)          // parents first, then own
}

/**
A mixin appearing more than once (here via `SubA` and directly) is merged only
once — no duplicate-member error. The first occurrence wins.
**/
test "duplicate mixins are merged once" {
    mixin A { a = 1 }
    mixin SubA { *A }
    class C { use SubA, A }                 // A reachable twice

    t.equal(1, C().a)
    t.equal(2, len(C.@mixins))              // both use-entries recorded
}

/**
An optional `this { … }` block types the `this` of the mixin's props/methods via
an interface, so a method can call a member the final class provides. `this` is
always a class instance of the using class.
**/
test "this interface types the receiver" {
    mixin Described {
        this { label() <str> }              // require a label() method
        methods { describe() => "<" + this.label() + ">" }
    }
    class Tag {
        use Described
        methods { label() => "tag" }
    }

    t.equal("<tag>", Tag().describe())
}

/**
A mixin can be anonymous, bound to a const: `const M = mixin { … }`.
**/
test "anonymous mixin" {
    const Timestamped = mixin { created = 0 }
    class Doc { use Timestamped }

    t.equal(0, Doc().created)
}

/**
`@interface` returns a cached `Interface` instance (same value on each read) named
`Name$interface`, mirroring the mixin's declared members with their types and
accessor kinds (`f int`, `get p`, a method).
**/
test "@interface reflects the declared members" {
    mixin Shape {
        sides int = 0
        props   { area => 0 }
        methods { draw() => nil }
    }

    i := Shape.@interface
    t.true(i :: gad.Interface)              // it is an Interface value
    t.true(i == Shape.@interface)           // cached: same instance
    t.equal("area", collect(keys(Shape.@props))[0])
    t.equal("draw", collect(keys(Shape.@methods))[0])
}

/**
`@this` returns the mixin's declared `this { … }` interface — the contract the
receiver of its methods and properties must satisfy — or nil when the mixin has
no `this` block.
**/
test "@this is the declared this-interface" {
    mixin Movable {
        this { pos() <int> }
        methods { step() => this.pos() + 1 }
    }
    mixin Plain { x = 1 }

    t.true(Movable.@this :: gad.Interface)   // the `this { pos() }` interface
    t.equal(nil, Plain.@this)                // no `this` block -> nil
}

/**
`@interface` **extends** the mixin's `this` interface and each parent mixin's
`@interface`, so a value satisfies it only when it satisfies the whole contract
the mixin contributes: its own members, its `this` requirement, and its parents'.
**/
test "@interface extends this and parents" {
    mixin Named { name = "?" }
    mixin Sized {
        *Named
        this { size() <int> }
        methods { area() => this.size() * this.size() }
    }

    // A class using Sized that provides size() satisfies the whole contract.
    class Box { use Sized; methods { size() => 3 } }
    t.true(Box(; name="b") :: Sized.@interface)

    // Missing size() fails the extended `this` interface.
    class NoSize { use Named }
    t.true(!(bool(NoSize(; name="x") :: Sized.@interface) or false))
}

/**
A `met` declaration adds a method, property or constructor to a class or mixin
after its definition. The receiver — `this` (the instance), `new` (the class
initiator) or `$old` (the overridden implementation) — sees every member the type
offers, **including members pulled in from `use`d mixins, inherited from parents,
and required by a mixin's `this { … }` interface**. Editor auto-completion of
`this.` / `new.` / `$old.` in these contexts resolves exactly this member set
(verified by the CLI's `gad complete` tests).
**/
test "met adds a method seeing merged and inherited members" {
    mixin Counter { count = 0; methods { inc() { this.count += 1 } } }
    class Base { hp = 10 }
    class Hero {
        *Base
        use Counter
        name = "?"
    }

    // `this` inside the met sees own (name), inherited (hp) and mixin (count/inc).
    met Hero.status(this) => [this.name, this.hp, this.count]
    h := Hero(; name="A", hp=5, count=2)
    t.equal(["A", 5, 2], h.status())

    h.inc()                                   // mixin method, added via `use`
    t.equal(3, h.count)
}
```
