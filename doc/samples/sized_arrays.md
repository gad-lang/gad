
# Sized arrays (`[N]T`, `[A..B]T`)

A level of an array type may have a **size**: how many items it holds.

| Level | Items |
|-------|-------|
| `[]` | any number |
| `[2]` | exactly 2 |
| `[..3]` | at most 3 |
| `[2..]` | at least 2 |
| `[2..5]` | from 2 to 5 |

Each level has its own: `[][2]str` is any number of pairs, `[2..3][2]int` two or
three pairs. The sizes go wherever an array type goes — a parameter, a class
field, a union (`[]str|[][2]str`), a cast, a `type` declaration and a
`type […]T` value.

A size is an int: a literal, arithmetic of literals (`[1+2]`) — both checked
when the code compiles —, or the **name of a variable**, read when a value is
checked (`[n]`, `[min..max]`). Any other expression is a compile error: name
it first (`n := len(x)`, then `[n]`).

```gad
pair := [1, 2] :: [2]int                 // exactly 2
some := [1, 2, 3] :: [2..3]int           // 2 to 3
rows := [[1, 2], [3, 4]] :: [][2]int     // any number of pairs

n := 3
three := [1, 2, 3] :: [n]int             // a variable, read now

refused := func(f) { try { f(); return false } catch { return true } }
[len(pair), len(some), len(rows), len(three),
 refused(() => [1, 2, 3] :: [2]int),
 refused(() => [[1, 2], [3]] :: [][2]int)]
= [2, 3, 2, 3, true, true]
```

## Parameters and fields

A parameter or a field of a sized type takes only an array of its sizes; a
variable the sizes name is read as a closure reads it.

```gad
first := (p [2]str) => p[0]
class Address {
    city []str|[][2]str      // names, or [name, state] pairs
}

maker := (size) => (v [size]int) => len(v)    // the size of a closure
three := maker(3)

refused := func(f) { try { f(); return false } catch { return true } }
a := Address(;city=[["Viçosa", "MG"]])
[first(["a", "b"]), a.city[0][1], three([1, 2, 3]),
 refused(() => first(["a"])),
 refused(() => three([1]))]
= ["a", "MG", 3, true, true]
```

## Overloads

Overloads told apart by the sizes alone are allowed: the call takes the one
whose sizes the argument has, trying them in order — **exactly** a size first,
then the **narrowest range**, then **no size**, then the order of declaration.
Two overloads of the same types are still an error.

```gad
kind := func {
    (a [1..9]int) => "1 to 9"
    (a [2..3]int) => "2 or 3"
    (a [3]int) => "3"
    (a []int) => "any"
}
[kind([1, 2, 3]), kind([1, 2]), kind([1, 2, 3, 4]), kind([])]
= ["3", "2 or 3", "1 to 9", "any"]
```

## Converting: `::>`

`::>` into an array type checks the sizes first, then converts each leaf into
an element type (a leaf an element type already accepts is kept).

```gad
[[1, 2] ::> []str, [[1, 2], [3, 4]] ::> [][2]str, [1, "a"] ::> []<int|str>]
= [["1", "2"], [["1", "2"], ["3", "4"]], [1, "a"]]
```

## Typed arrays

In a `type NAME […]T` declaration the sizes are kept by the value: the first
level is its number of items, checked when it is built (`pair("a", "b")`, `arr
::> pair`) and by every append and delete — a write that would leave it is an
error. A slice of fewer items is a plain array.

```gad
type pair [2]str
type options [][2]str
type upTo3 [..3]int

p := pair("a", "b")
o := options(["a", "A"], ["b", "B"])
u := upTo3(1)
u += 2

refused := func(f) { try { f(); return false } catch { return true } }
[str(p), str([1, 2] ::> pair), len(o), len(u), typeName(p[:1]),
 refused(() => pair("a")),             // 1 item
 refused(func() { p += "c" }),         // 3 items
 refused(func() { delete p[0] }),      // 1 item
 refused(() => options(["a"]))]        // a pair of 1
= ["pair[\"a\", \"b\"]", "pair[\"1\", \"2\"]", 2, 2, "array", true, true, true, true]
```

## Reflection: `@size`, `@elem`, `@depth`

An array type — `type […]T` or a typed array type — answers:

| Key | Value |
|-----|-------|
| `t.@size` | an `[eq, min, max]` a level, `nil` where unset; also `.eq`, `.min`, `.max` |
| `t.@elem` | the element types (an array type: their array; a typed array type: the type) |
| `t.@depth` | the number of levels |

```gad
const t = type [2..3][2]str
type options [..5][2]str
type ints []int

[t.@size[0] == [nil, 2, 3], t.@size[1].eq, t.@size[1].min, t.@depth,
 t.@elem[0] == str, (type []<int|str>).@elem == [int, str],
 options.@size[0].max, options.@size[1].eq,
 ints.@size[0] == [nil, nil, nil], ints.@elem == int,
 str(t), options.@name]
= [true, 2, nil, 2, true, true, 5, 2, true, true, "[2..3][2]str", "options"]
```

## Building: calling an array type

An array type is a **constructor**. Its items: `(type [2]str)("a", "b")` —
each level checked, as `::` checks it (the leaves are not converted: `::>`
converts) —; `(type […]T)()` is the empty array. The value is a **plain
array**: an array type is structural, not bound to its values (a declared
[typed array](typed_arrays.gad) is).

```gad
pair := type [2]str
refused := func(f) { try { f(); return false } catch { return true } }
[pair("a", "b"), (type [][]int)([1], [2, 3]), (type []int)(),
 refused(() => pair("a")),                  // level 1: 1 item, wants 2
 refused(() => (type []int)("x"))]          // item 0: not an int
= [["a", "b"], [[1], [2, 3]], [], true, true]
```

### By its dimensions: `size`, `fill`

With no items, the named arguments `size` and `fill` build it by its
dimensions:

- `size=[n1, n2, …]` — a size a level, or a size a level **with no exact
  size** (`(type [2][]int)(; size=[3])`): a level of a range takes a size in
  it. A type whose levels are all exact needs none.
- `fill` — every leaf: a value of the element types, or a **function of the
  indexes** of the leaf (as many as the levels). With no `fill` a leaf is
  `nil`, when the element types take it (`[2]any`) — else it is an error.

```gad
n := 2
[(type [2][3]int)(; fill=0),
 (type [][]int)(; size=[2, 2], fill=(i, j) => i + j),
 (type [2][]str)(; size=[1], fill="x"),     // the level with no size
 (type [n]any)(; size=[n])]                 // nil leaves: any takes nil
= [[[0, 0, 0], [0, 0, 0]], [[0, 1], [1, 2]], [["x"], ["x"]], [nil, nil]]
```

## `arrayType`: the type of the array types

An array type's type is `arrayType` (`typeof(type []int) === arrayType`), a
type with a constructor: `arrayType(elem; depth, size)` builds an array type
at run time — `elem` a type or an array of types (a union), `depth` the levels
(1 when not given), `size` a level each: an int (exactly), or the `[eq, min,
max]` of `@size` (`nil` where unset). It is the inverse of the reflection
keys; as any type with a constructor it takes **overloads** (`met
arrayType(x MyType) => …`).

Two array types of the same depth, sizes and element types are `==`.

```gad
T := type [2][1..3]int
grid := arrayType(int; size=[3, 3])
[str(arrayType(int; depth=2)), str(arrayType([int, str])), str(grid),
 grid(; fill=0)[2],
 arrayType(T.@elem; depth=T.@depth, size=T.@size) == T,   // the round trip
 typeof(T) === arrayType]
= ["[][]int", "[]<int|str>", "[3][3]int", [0, 0, 0], true, true]
```

## `array` is `type []any`

The builtin `array` is the array type of `any` items with no size — the
**same object**: `array === type []any`, `typeof([1, 2]) === type []any`, and
`array.@elem`, `@depth`, `@size` answer as an array type's. `type []any`
written anywhere — a value, a parameter, a field — is `array`; `array(1, 2)`
builds as it always did, and by its dimensions too.

```gad
f := func(x []any) => len(x)       // the same param as `x array`
[array === type []any, typeof([1, 2]) === type []any, arrayType(any) === array,
 array.@depth, f([1, 2]), array(; size=[2], fill=0)]
= [true, true, true, 1, 2, [0, 0]]
```

## Example — `sized_arrays.gad`

```gad
example casts {
    pair := [1, 2] :: [2]int                 // exactly 2
    some := [1, 2, 3] :: [2..3]int           // 2 to 3
    rows := [[1, 2], [3, 4]] :: [][2]int     // any number of pairs

    n := 3
    three := [1, 2, 3] :: [n]int             // a variable, read now

    refused := func(f) { try { f(); return false } catch { return true } }
    [len(pair), len(some), len(rows), len(three),
     refused(() => [1, 2, 3] :: [2]int),
     refused(() => [[1, 2], [3]] :: [][2]int)]
    = [2, 3, 2, 3, true, true]
}

example params {
    first := (p [2]str) => p[0]
    class Address {
        city []str|[][2]str      // names, or [name, state] pairs
    }

    maker := (size) => (v [size]int) => len(v)    // the size of a closure
    three := maker(3)

    refused := func(f) { try { f(); return false } catch { return true } }
    a := Address(;city=[["Viçosa", "MG"]])
    [first(["a", "b"]), a.city[0][1], three([1, 2, 3]),
     refused(() => first(["a"])),
     refused(() => three([1]))]
    = ["a", "MG", 3, true, true]
}

example overloads {
    kind := func {
        (a [1..9]int) => "1 to 9"
        (a [2..3]int) => "2 or 3"
        (a [3]int) => "3"
        (a []int) => "any"
    }
    [kind([1, 2, 3]), kind([1, 2]), kind([1, 2, 3, 4]), kind([])]
    = ["3", "2 or 3", "1 to 9", "any"]
}

example build_items {
    pair := type [2]str
    refused := func(f) { try { f(); return false } catch { return true } }
    [pair("a", "b"), (type [][]int)([1], [2, 3]), (type []int)(),
     refused(() => pair("a")),                  // level 1: 1 item, wants 2
     refused(() => (type []int)("x"))]          // item 0: not an int
    = [["a", "b"], [[1], [2, 3]], [], true, true]
}

example build_dims {
    n := 2
    [(type [2][3]int)(; fill=0),
     (type [][]int)(; size=[2, 2], fill=(i, j) => i + j),
     (type [2][]str)(; size=[1], fill="x"),     // the level with no size
     (type [n]any)(; size=[n])]                 // nil leaves: any takes nil
    = [[[0, 0, 0], [0, 0, 0]], [[0, 1], [1, 2]], [["x"], ["x"]], [nil, nil]]
}

example array_type {
    T := type [2][1..3]int
    grid := arrayType(int; size=[3, 3])
    [str(arrayType(int; depth=2)), str(arrayType([int, str])), str(grid),
     grid(; fill=0)[2],
     arrayType(T.@elem; depth=T.@depth, size=T.@size) == T,   // the round trip
     typeof(T) === arrayType]
    = ["[][]int", "[]<int|str>", "[3][3]int", [0, 0, 0], true, true]
}

example any_array {
    f := func(x []any) => len(x)       // the same param as `x array`
    [array === type []any, typeof([1, 2]) === type []any, arrayType(any) === array,
     array.@depth, f([1, 2]), array(; size=[2], fill=0)]
    = [true, true, true, 1, 2, [0, 0]]
}

example transform {
    [[1, 2] ::> []str, [[1, 2], [3, 4]] ::> [][2]str, [1, "a"] ::> []<int|str>]
    = [["1", "2"], [["1", "2"], ["3", "4"]], [1, "a"]]
}

example typed {
    type pair [2]str
    type options [][2]str
    type upTo3 [..3]int

    p := pair("a", "b")
    o := options(["a", "A"], ["b", "B"])
    u := upTo3(1)
    u += 2

    refused := func(f) { try { f(); return false } catch { return true } }
    [str(p), str([1, 2] ::> pair), len(o), len(u), typeName(p[:1]),
     refused(() => pair("a")),             // 1 item
     refused(func() { p += "c" }),         // 3 items
     refused(func() { delete p[0] }),      // 1 item
     refused(() => options(["a"]))]        // a pair of 1
    = ["pair[\"a\", \"b\"]", "pair[\"1\", \"2\"]", 2, 2, "array", true, true, true, true]
}

example reflection {
    const t = type [2..3][2]str
    type options [..5][2]str
    type ints []int

    [t.@size[0] == [nil, 2, 3], t.@size[1].eq, t.@size[1].min, t.@depth,
     t.@elem[0] == str, (type []<int|str>).@elem == [int, str],
     options.@size[0].max, options.@size[1].eq,
     ints.@size[0] == [nil, nil, nil], ints.@elem == int,
     str(t), options.@name]
    = [true, 2, nil, 2, true, true, 5, 2, true, true, "[2..3][2]str", "options"]
}
```
