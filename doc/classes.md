# Classes and Objects

[← Back to index](README.md)

Gad has a class system built on the `Class(...)` builtin. A class describes
**fields**, **methods**, **properties** and one or more **constructors**, and
can **extend** other classes. Calling a class creates an instance.

## Defining a class

`Class(name; …)` takes the class name positionally and everything else as named
arguments: `fields`, `methods`, `properties`, `new` (the constructor) and
`extends`. All are optional.

```go
Point := Class("Point";
    fields = (;
        x int = 0
        y int = 0
    ),
    methods = [
        dist(this) => float(this.x ** 2 + this.y ** 2) ** 0.5
    ]
)

p := Point(; x=3, y=4)
println(p.dist())     // 5
```

## Fields

Fields are declared in a `(; … )` group. Each field may have a type and a
default value:

```go
Class("P"; fields = (;
    a              // any, default nil
    b int          // type annotation (not enforced), default nil
    c = "x"        // default value
    d str = "y"    // type + default
))
```

A field's default may be a **computed value** `(= … )`, which is evaluated
*fresh for each instance* — handy for per-instance mutable defaults:

```go
n := 0
C := Class("C"; fields = (; id = (= n++)))
[C().id, C().id, C().id]    // [1, 2, 3]
```

Instances expose fields with `inst.field` (read) and `inst.field = v` (write).

## Constructors

Without a `new`, a class is constructed by passing field values as named
arguments: `Point(; x=3, y=4)`. To accept positional arguments, define `new`
with one or more overloads (the func-with-methods syntax). The first parameter
is always `this`; calling `this(; field=value, …)` initialises the instance:

```go
Point := Class("Point"; new {
    (this; **f)      => this(; x=0, y=0, **f)   // defaults + extra named fields
    (this, x, y)     => this(; x=x, y=y)        // positional
    (this, x)        => this.@new(; x=x)        // chain to another overload
})

Point()         // x=0, y=0
Point(3, 4)     // x=3, y=4
Point(; x=7)    // x=7, y=0
```

`this.@new(…)` chains to the default constructor (the one that just assigns the
named fields).

## Methods

Methods live in the `methods` list. Each is a function whose first parameter is
`this`. A method may be written in shorthand (`name(this, …) => expr`) or as a
func-with-methods block to overload it by arity/type:

```go
Class("Calc"; methods = [
    add(this, a, b) => a + b
    add(this, a)    => a + a       // overload
    label(this) => "calc",
])
```

## Properties

Properties are computed members with a getter (no extra parameters) and one or
more setters (one extra parameter, optionally typed). They are accessed like
fields — reading runs the getter, assigning runs the matching setter:

```go
Box := Class("Box"; fields = (; v), properties = {
    val: func {
        (this)        => this.v               // getter
        (this, x)     { this.v = "any:" + str(x) }   // setter
        (this, x int) { this.v = "int:" + str(x) }   // typed setter
    }
})

b := Box()
b.val = "a"; b.val    // "any:a"
b.val = 5;   b.val    // "int:5"
```

## Inheritance

`extends = [Parent, …]` embeds one or more parent classes — like Go's anonymous
fields. Parent fields, methods and properties are **promoted**: an instance of
the child can use them directly, and a child method of the same name overrides
the parent's.

```go
Animal := Class("Animal";
    fields  = (; name str = "?"),
    methods = [
        speak(this)    => this.name + " makes a sound"
        describe(this) => "I am " + this.name
    ]
)

Dog := Class("Dog";
    extends = [Animal],
    methods = [ speak(this) => this.name + " barks" ]   // override
)

d := Dog(; name="Rex")
d.speak()       // "Rex barks"   (override)
d.describe()    // "I am Rex"    (inherited)
d.name          // "Rex"         (promoted field)
```

A promoted field is **shared** with the embedded parent: writing `d.name`
routes to the parent instance, so the parent's inherited methods see the same
value.

Multiple parents are embedded left to right:

```go
A := Class("A"; methods = [ a(this) => "a" ])
B := Class("B"; methods = [ b(this) => "b" ])
C := Class("C"; extends = [A, B])
o := C()
[o.a(), o.b()]    // ["a", "b"]
```

## Extending a class with `met`

The `met` statement attaches behaviour to an existing class from the outside —
extra methods, operator overloads, type conversions and custom printing.

```go
Vec := Class("Vec"; fields = (; x int = 0, y int = 0))

// add a method
met Vec.len2(this) => this.x*this.x + this.y*this.y

// overload a binary operator
met @binaryOperator(_ TBinaryOperatorAdd, a Vec, b Vec) {
    return Vec(; x=a.x+b.x, y=a.y+b.y)
}

// type conversions (str(v), int(v), …)
met str(v Vec) => "(" + v.x + ", " + v.y + ")"

// custom printing (str vs repr via state.isRepr)
met print(state PrinterState, v Vec) { write(state, "Vec" + str(v)) }

a := Vec(; x=1, y=2)
b := Vec(; x=10, y=20)
str(a + b)      // "(11, 22)"
a.len2()        // 5
```
