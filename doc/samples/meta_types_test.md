
# Meta types (`type<X>`)

A `type<X>` parameter matches a **type value** — the class `X` itself — rather
than an *instance* of `X`. It is how a function dispatches on which type it was
handed:

    func d {
        (t type<Rect>)  => …   // called as d(Rect)
        (t type<Point>) => …   // called as d(Point)
    }

Inside the body the bound parameter is the type value, so it is usable *as* a
type — you can construct it (`t()`), read its name (`typeName(t)`), or compare it
(`t == Rect`).

## Instances vs. type values

A type value and an instance of that type dispatch **separately** — a plain
`(t X)` parameter matches an instance of `X`, `(t type<X>)` matches the type value
`X`. The two overloads coexist in one function, and passing an instance to a
`type<X>` parameter is rejected: the path a value takes is decided by whether it
*is* a type or is *of* a type.

## `type<X>` vs. `type <X>`

Mind the space. In a **parameter** position `type<X>` is this meta type. At an
**expression** position `type <A|B>` is instead a first-class
[type union](type_unions.gad) value — a different feature. The meta form lives
only where a parameter type is written.

Run it: `gad test samples/meta_types_test.gad`

## Example — `meta_types_test.gad`

```gad
class Rect { w = 0; h = 0 }
class Point { x = 0; y = 0 }

/**
Dispatch on the type value: `d(Rect)` selects the `type<Rect>` overload and binds
`t` to the class `Rect` itself.
**/
test "dispatch on the type value" {
    func d {
        (t type<Rect>)  => [typeName(t), 1]
        (t type<Point>) => [typeName(t), 2]
    }

    t.equal(["Class", 1], d(Rect))
    t.equal(["Class", 2], d(Point))
    t.equal(true, d(Rect)[0] == "Class")
}

/**
The bound type value is usable as a type — here to construct an instance.
**/
test "the bound value is a usable type" {
    func make(t type<Rect>) => t()
    r := make(Rect)
    t.equal("Rect", typeName(r))
    t.equal(0, r.w)
}

/**
An instance overload `(t X)` and a type-value overload `(t type<X>)` coexist and
dispatch distinctly.
**/
test "instances and type values are separate" {
    func k {
        (t Rect)       => ["instance", t.w]
        (t type<Rect>) => ["type", t == Rect]
    }

    t.equal(["instance", 0], k(Rect()))   // an instance of Rect
    t.equal(["type", true], k(Rect))      // the type value Rect
}

/**
Passing an instance to a `type<X>` parameter is rejected — a value that is *of*
type Rect is not the *type* Rect.
**/
test "an instance is not a type value" {
    func d { (t type<Rect>) => t }
    // an instance of Rect does not satisfy `type<Rect>`.
    t.raises(() => d(Rect()))
}

/**
`met f(_ type<X>)` adds a type-value overload to an existing callable; other
arguments fall through to the base function.
**/
test "met adds a type-value overload" {
    func describe(x) => "value"
    met describe(_ type<Rect>) => "the Rect type"

    t.equal("the Rect type", describe(Rect))
    t.equal("value", describe(123))
}

/**
A union `type<X|Y>` matches the type value X or Y — the same `|` syntax as any
parameter-type union.
**/
test "a union meta type matches any listed type value" {
    func accept(t type<int|bool>) => typeName(t)

    t.equal("Base", accept(int))    // int and bool are accepted
    t.equal("Base", accept(bool))

    t.raises(() => accept(str))     // str is not in the union
}
```
