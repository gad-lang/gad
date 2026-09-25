
# Classes (the `Class(...)` builtin)

Gad's class system is built on the `Class(name, define)` builtin. A class
describes **fields**, **methods**, **properties** and one or more
**constructors**, and can **extend** other classes; calling a class creates an
instance. The high-level [`class` keyword](syntax.gad) lowers to this
builtin.

## Defining

`Class(name, define)` takes the name positionally and an optional **define
handler** `(cls, define) => …` that receives the in-construction class and a
`define` function; `define(; …)` registers members as named args: `fields`,
`methods`, `properties`, `new` (the constructor) and `extends`. All are optional.

## Fields

Fields are a `(; … )` group; each may have a type (annotation, not enforced) and
a default. A default may be a **computed value** `(= … )`, evaluated fresh per
instance — handy for per-instance mutable defaults.

## Constructors

Without a `new`, a class is constructed by passing fields as named args
(`Point(; x=3)`). Define `new` (func-with-methods) for positional overloads; the
first parameter is the `new` **initiator** — `new(; field=value, …)` initialises
and returns the instance, and a recursive `new` falls to the default initiator so
construction terminates.

## Methods & properties

Methods take a first `this` param and may overload by arity/type. Properties are
computed members with a getter (no extra param) and typed setters (one extra
param), accessed like fields.

```gad
Point := Class("Point", (cls, define) => define(;
    new {
        (new; **f)  => new(; x=0, y=0, **f)   // defaults + extra named fields
        (new, x, y) => new(; x=x, y=y)        // positional
    },
    methods = [
        dist(this) => (this.x ** 2 + this.y ** 2) ** 0.5
    ]
))
p := Point(3, 4)
[p.dist(), Point().x, Point(; y=2).y]
// => [5, 0, 2]
```

```gad
Box := Class("Box", (cls, define) => define(; fields = (; v), props = {
    val: func {
        (this)        => this.v                        // getter
        (this, x)     { this.v = "any:" + str(x) }     // setters, by type
        (this, x int) { this.v = "int:" + str(x) }
    }
}))
b := Box()
b.val = "hi"
first := b.val
b.val = 7
[first, b.val]
// => ["any:hi", "int:7"]
```

## Inheritance

`extends = [Parent, …]` embeds parents (Go-style anonymous fields): their fields,
methods and properties are **promoted** and a same-named child method overrides.
A promoted field is **shared** with the embedded parent instance.

```gad
Animal := Class("Animal", (cls, define) => define(;
    fields  = (; name str = "?"),
    methods = [
        speak(this)    => this.name + " makes a sound"
        describe(this) => "I am " + this.name
    ]
))
Dog := Class("Dog", (cls, define) => define(;
    extends = [Animal],
    methods = [ speak(this) => this.name + " barks" ]   // override
))
d := Dog(; name="Rex")
[d.speak(), d.describe()]   // overridden, inherited
// => ["Rex barks", "I am Rex"]
```

## Extending with `met` / `met ~` / `$old`

`met` attaches behaviour to an existing class from outside — extra methods,
operator overloads (`met gad.binOpAdd`), type conversions (`met str(v Vec)`),
custom printing. `met ~Class.name(...)` **overrides** an existing member (method,
constructor or property setter); a `$old` first parameter captures the previous
implementation for super/around wrapping. It works for methods, constructors
and property setters alike.

```gad
// method: wrap the overridden speak()
met ~Dog.speak($old, this) => $old(this) + " loudly!"

// constructor: scale coordinates, delegating to the previous constructor
Point3 := Class("Point3", (cls, define) => define(;
    fields = (; x int = 0, y int = 0),
    new { (new, x, y) => new(; x=x, y=y) }
))
met ~Point3($old, new, x, y) => $old(new, x * 10, y * 10)
q := Point3(3, 4)

// property setter: validate on top of the previous setter
met ~Box.val($old, this, x int) { $old(this, x); this.v = this.v + " (checked)" }
b.val = 9

[d.speak(), [q.x, q.y], b.val]
// => ["Rex barks loudly!", [30, 40], "int:9 (checked)"]
```

```gad
Vec := Class("Vec", (cls, define) => define(; fields = (; x int = 0, y int = 0)))
met gad.binOpAdd(a Vec, b Vec) {           // an operator overload
    return Vec(; x=a.x+b.x, y=a.y+b.y)
}
met str(v Vec) => "(" + v.x + ", " + v.y + ")"   // a conversion
met Vec.len2(this) => this.x*this.x + this.y*this.y  // an extra method

a := Vec(; x=1, y=2)
[str(a + Vec(; x=10, y=20)), a.len2()]
// => ["(11, 22)", 5]
```

## Example — `classes.gad`

```gad
Point := Class("Point", (cls, define) => define(;
    new {
        (new; **f)  => new(; x=0, y=0, **f)   // defaults + extra named fields
        (new, x, y) => new(; x=x, y=y)        // positional
    },
    methods = [
        dist(this) => (this.x ** 2 + this.y ** 2) ** 0.5
    ]
))
p := Point(3, 4)
[p.dist(), Point().x, Point(; y=2).y]

Box := Class("Box", (cls, define) => define(; fields = (; v), props = {
    val: func {
        (this)        => this.v                        // getter
        (this, x)     { this.v = "any:" + str(x) }     // setters, by type
        (this, x int) { this.v = "int:" + str(x) }
    }
}))
b := Box()
b.val = "hi"
first := b.val
b.val = 7
[first, b.val]

Animal := Class("Animal", (cls, define) => define(;
    fields  = (; name str = "?"),
    methods = [
        speak(this)    => this.name + " makes a sound"
        describe(this) => "I am " + this.name
    ]
))
Dog := Class("Dog", (cls, define) => define(;
    extends = [Animal],
    methods = [ speak(this) => this.name + " barks" ]   // override
))
d := Dog(; name="Rex")
[d.speak(), d.describe()]   // overridden, inherited

// method: wrap the overridden speak()
met ~Dog.speak($old, this) => $old(this) + " loudly!"

// constructor: scale coordinates, delegating to the previous constructor
Point3 := Class("Point3", (cls, define) => define(;
    fields = (; x int = 0, y int = 0),
    new { (new, x, y) => new(; x=x, y=y) }
))
met ~Point3($old, new, x, y) => $old(new, x * 10, y * 10)
q := Point3(3, 4)

// property setter: validate on top of the previous setter
met ~Box.val($old, this, x int) { $old(this, x); this.v = this.v + " (checked)" }
b.val = 9

[d.speak(), [q.x, q.y], b.val]

Vec := Class("Vec", (cls, define) => define(; fields = (; x int = 0, y int = 0)))
met gad.binOpAdd(a Vec, b Vec) {           // an operator overload
    return Vec(; x=a.x+b.x, y=a.y+b.y)
}
met str(v Vec) => "(" + v.x + ", " + v.y + ")"   // a conversion
met Vec.len2(this) => this.x*this.x + this.y*this.y  // an extra method

a := Vec(; x=1, y=2)
[str(a + Vec(; x=10, y=20)), a.len2()]

return p.dist()
```
