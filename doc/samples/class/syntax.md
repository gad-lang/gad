
# The `class` keyword

The `class` keyword is a readable block syntax that lowers to the
[`Class(...)` builtin](classes.gad). A `class` block reads top to bottom:
optional parent spreads (`*Parent`), bare fields, then `props {}`, `new` and
`methods {}` groups (items separated by newlines or commas).

The first parameter is inserted automatically — you do not write it: `this` for
methods and property accessors, and `new` (the class initiator) for
constructors. A method takes a typed `this cls` (so overloads dispatch on
argument types); a `name = expr` entry in `props`/`methods` is shorthand for a
zero-argument accessor `() => expr`. The statement form `class Name { … }`
defines a const; there are also anonymous expression and `export class` forms.
Everything else — field defaults, typed fields, inheritance, overloaded
methods/constructors — works exactly as in the `Class(...)` builtin.

A typed field is enforced on construction (a value of the wrong type is
rejected); a `?` after the field name (`x? int`, `x? int|str`) makes it
**nullable**, so it also accepts `nil`. See
[typed & nullable fields](field_types.gad).

Doc comments attach to the class and its members (`///`, `/** … **/`,
`/*** … ***/`). The Example below is a runnable tour.

## Example — `syntax.gad`

```gad
/**
A 2D point. Shows the statement form `class Name { … }`, which defines a
constant. `this` is auto-inserted as the first parameter of methods, property
accessors and constructors.
**/
class Point {
    /// horizontal coordinate
    x = 0
    /// vertical coordinate
    y = 0

    new {
        (; **f)  => new(; x=0, y=0, **f)   // named fields (defaults + extra)
        (x, y)   => new(; x=x, y=y)         // positional
    }

    props {
        /// distance from the origin
        mag() => (this.x ** 2 + this.y ** 2) ** 0.5
    }

    methods {
        /// dot product with another point
        dot(o) => this.x * o.x + this.y * o.y
    }
}

p := Point(3, 4)
println("Point(3, 4).mag =", p.mag)          // 5
println("Point().x       =", Point().x)      // 0
println("p.dot(p)        =", p.dot(p))       // 25

/// Tagger shows a typed field, a computed default and a method overloaded by
/// argument type (a method takes a typed `this`, so overloads dispatch).
class Tagger {
    /// the tag prefix
    name str = "?"
    /// a fresh value per instance (here always 0)
    seq = (= 0)

    methods {
        /// tag an int
        tag(n int) => this.name + ":int:" + str(n)
        /// tag a string
        tag(s str) => this.name + ":str:" + s
    }
}
t := Tagger(; name="t")
println("tag(7)   =", t.tag(7))     // t:int:7
println("tag(\"x\") =", t.tag("x"))   // t:str:x

/// Box shows a property with a getter and typed setters, plus the `name = expr`
/// getter shortcut.
class Box {
    /// the stored value
    v
    props {
        /// the value, coerced on set by argument type
        val {
            ()        => this.v
            (x)       { this.v = "any:" + str(x) }
            (x int)   { this.v = "int:" + str(x) }
        }
        /// whether no value has been set (getter shortcut for `() => expr`)
        empty = this.v == nil
    }
}
b := Box()
println("empty   =", b.empty)        // true
b.val = "hi"
println("box any =", b.val)          // any:hi
b.val = 7
println("box int =", b.val)          // int:7

/// Animal is a base class with two methods.
class Animal {
    /// the animal's name
    name str = "?"
    methods {
        /// the sound the animal makes
        speak()    => this.name + " makes a sound"
        /// a human-readable description
        describe() => "I am " + this.name
    }
}

/// Dog inherits Animal and overrides speak().
class Dog {
    *Animal
    methods {
        /// dogs bark
        speak() => this.name + " barks"
    }
}
d := Dog(; name="Rex")
println("dog speak:   ", d.speak())     // Rex barks (override)
println("dog describe:", d.describe())  // I am Rex (inherited)

/// The expression form `X := class { … }` yields an anonymous, first-class
/// class value.
Counter := class {
    /// current count
    n = 0
    methods {
        /// advance and return the new value
        next() { this.n++; return this.n }
    }
}
c := Counter()
c.next(); c.next()
println("counter:", c.next())           // 3

return p.mag
```
