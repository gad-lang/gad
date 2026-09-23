
# Array interfaces (`interface P [] { … }`)

Writing one or more `[]` after the interface's name makes the
[interface](interfaces.gad) match an **array** — a *array interface*:

- `interface P [] { … }` matches a flat array whose elements each satisfy the
  body; `interface P [][][] { … }` an array nested three deep (the number of `[]`
  is the required depth). `interface P [] interface { … }` is the long form of
  the same.
- `interface P []<int|uint>` (or `interface P []int`) is a **array of types**:
  the leaves must match one of the element types. It is the named analogue of
  the anonymous `[]<int|uint>` type written in a field or parameter
  (`interface form { field []int }`).
- Anonymous: `interface [] { … }` / `interface []<int|str>`.

The `[]` always follows the name — the former `interface[] P { … }` is rejected.

With the checked cast [`::`](user_operators.gad) the (nested) array must already
hold satisfying elements and is returned unchanged. With the transforming cast
[`:::`](transform_cast_test.gad) each leaf is coerced into the declared shape (its
class-typed fields are built), at any depth; a [typed array](typed_arrays.gad)
converts too. An array interface is also a parameter type, carries
[metadata](metadata.gad) and answers the reflection keys `@depth` (the number of
`[]`), `@elem` (the element types of an array of types, else nil) and `@meta`.
An empty array satisfies any array interface vacuously.

Run it: `gad test samples/interface_arrays_test.gad`

## Example — `interface_arrays_test.gad`

```gad
class Point { x int; y int }

/**
`interface P [] { … }` matches a flat array whose every element satisfies the
body. `::` checks and returns the array unchanged; a non-array, a missing field
or a wrong nesting depth is rejected.
**/
test "flat array interface" {
    interface points [] { x int, y int }

    ps := [{x: 1, y: 2}, {x: 3, y: 4}] :: points
    t.equal(2, len(ps))
    t.equal(1, ps[0].x)
    t.equal(4, ps[1].y)

    // An empty array satisfies vacuously.
    t.equal(0, len([] :: points))

    // A non-array is rejected: the cast throws, so `bool(…) or false` is false.
    d := {x: 1, y: 2}
    t.true(!(bool(d :: points) or false))

    // An element missing a required field is rejected.
    t.true(!(bool([{x: 1}] :: points) or false))
}

/**
Each extra `[]` adds one level of array nesting: `interface P [][][] { … }`
matches an array of arrays of arrays whose leaves satisfy the body.
**/
test "nested array interface" {
    interface nestedPoints [][][] { x int, y int }

    r := [[[{x: 1, y: 2}, {x: 3, y: 4}]]] :: nestedPoints
    t.equal(4, r[0][0][1].y)

    // The wrong depth (too shallow) is rejected.
    t.true(!(bool([{x: 1, y: 2}] :: nestedPoints) or false))
}

/**
`interface P [] interface { … }` is the long form of `interface P [] { … }`.
**/
test "long form" {
    interface users [] interface { name; id }

    t.equal(1, len([{name: "a", id: 1}] :: users))
    t.true(!(bool([{name: "a"}] :: users) or false))
}

/**
`interface P []<T1|T2>` is an array of **types**: every leaf must match one of the
element types (one type needs no envelope: `interface P []int`).
**/
test "array of types" {
    interface numerics []<int|uint|float>
    interface grid [][]int

    t.equal(3, len([1, 2u, 3.5] :: numerics))
    t.true(!(bool([1, "x"] :: numerics) or false))

    t.equal(2, len([[1], [2, 3]] :: grid))
    t.true(!(bool([1, 2] :: grid) or false))
}

/**
An array interface is a type like any other, so it works as a parameter type: the
argument must be an array of satisfying elements.
**/
test "as a parameter type" {
    interface points [] { x int, y int }

    func total(ps points) {
        sum := 0
        for p in ps {
            sum += p.x + p.y
        }
        return sum
    }

    t.equal(10, total([{x: 1, y: 2}, {x: 3, y: 4}]))
}

/**
The transforming cast `:::` coerces each leaf element into the interface's shape —
here every element's `p` dict becomes a real `Point` — recursively through nested
arrays. A typed array converts to an array interface (and back) the same way.
**/
test "transforming an array" {
    // Depth 1: each element's `p` is built into a Point.
    wrapped := [{p: {x: 1, y: 2}}, {p: {x: 3, y: 4}}] ::: interface [] { p Point }
    t.equal("Point", typeName(wrapped[0].p))
    t.equal(4, wrapped[1].p.y)

    // Depth 3: leaves inside nested arrays are coerced too.
    deep := [[[{p: {x: 9, y: 8}}]]] ::: interface [][][] { p Point }
    t.equal("Point", typeName(deep[0][0][0].p))
    t.equal(9, deep[0][0][0].p.x)

    // A typed array converts to an array interface value (a plain array), each
    // leaf checked; and back.
    type nums []int
    interface ints []int
    a := nums(1, 2) ::: ints
    t.equal("array", typeName(a))
    t.equal("nums", typeName(a ::: nums))
}

/**
Reflection: `@depth` is the number of `[]`, `@elem` the element types of an array
of types (nil for a member body) and `@meta` the declaration's metadata.
**/
test "reflection and metadata" {
    [kind="numbers"]
    interface numerics []<int|float>
    interface points [] { x int }

    t.equal(1, numerics.@depth)
    t.equal(2, len(numerics.@elem))
    t.equal(nil, points.@elem)
    t.equal("(;kind=\"numbers\")", str(numerics.@meta))
}
```
