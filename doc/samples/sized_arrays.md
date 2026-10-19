
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
// => [2, 3, 2, 3, true, true]
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
// => ["a", "MG", 3, true, true]
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
// => ["3", "2 or 3", "1 to 9", "any"]
```

## Converting: `::>`

`::>` into an array type checks the sizes first, then converts each leaf into
an element type (a leaf an element type already accepts is kept).

```gad
[[1, 2] ::> []str, [[1, 2], [3, 4]] ::> [][2]str, [1, "a"] ::> []<int|str>]
// => [["1", "2"], [["1", "2"], ["3", "4"]], [1, "a"]]
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
// => ["pair[\"a\", \"b\"]", "pair[\"1\", \"2\"]", 2, 2, "array", true, true, true, true]
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
// => [true, 2, nil, 2, true, true, 5, 2, true, true, "[2..3][2]str", "options"]
```

## Example — `sized_arrays.gad`

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

kind := func {
    (a [1..9]int) => "1 to 9"
    (a [2..3]int) => "2 or 3"
    (a [3]int) => "3"
    (a []int) => "any"
}
[kind([1, 2, 3]), kind([1, 2]), kind([1, 2, 3, 4]), kind([])]

[[1, 2] ::> []str, [[1, 2], [3, 4]] ::> [][2]str, [1, "a"] ::> []<int|str>]

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

const t = type [2..3][2]str
type options [..5][2]str
type ints []int

[t.@size[0] == [nil, 2, 3], t.@size[1].eq, t.@size[1].min, t.@depth,
 t.@elem[0] == str, (type []<int|str>).@elem == [int, str],
 options.@size[0].max, options.@size[1].eq,
 ints.@size[0] == [nil, nil, nil], ints.@elem == int,
 str(t), options.@name]
```
