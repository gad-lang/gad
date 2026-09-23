
# Type Unions and Behavioural Interfaces

A **type** in Gad is a first-class value usable wherever a type is expected: a
parameter type, a return type `<T>`, and the `::` checked cast. Besides the
concrete types (`int`, `str`, …), Gad has **type unions** and **behavioural
interfaces**.

## Type unions

A union of types is written with `|` directly wherever a type is expected — a
parameter type (`func(v int|uint)`), a type-parameter constraint (`[T int|uint]`)
and a return type (`func() <int|str>`). To name a union or pass it around as a
value, write `type <T1|T2|…>`:

- **expression** — `const number = type <int|uint|float|decimal>`
- **statement**  — `type number <int|uint|float|decimal>` (sugar for the const)

A value satisfies the union when it matches any member, and unions **nest** — a
named union can appear inside another union (`str|number`). The builtin `number`
(`int|uint|float|decimal`) is one such union.

`type` stays an ordinary identifier everywhere else (`.type`, `type = …`,
`type := …`); it is only a type-union keyword when followed by `<`.

```gad
type number <int|uint|float|decimal> // named union (statement form)
addOne := func(v number) <number> => v + 1
// inline unions need no naming — here in both a parameter and a return type
pick := func(v int|str) <int|str> => v
// nested: a union can contain another named union
label := func(v str|number) => isStr(v) ? "text" : "num"
[addOne(2), addOne(2.5), pick(5), pick("x"), 1 :: number, label("hi"), label(3)]
// => [3, 3.5, 5, "x", 1, "text", "num"]
```

## Named slice types

A slice type — `[]T` (an array of `T`), `[]<T1|T2>` (an array whose elements may
be any of the union), or `[]{ … }` (an array whose elements each satisfy an
inline interface) — can be **named** with the `type NAME []…` statement, the slice
analogue of `type NAME <…>`:

- `type numerics []<int|uint|float|decimal>` — a named slice of a type union.
- `type users []{ name; id }` — a named slice interface; `[][]{ … }` nests.

The name is a first-class type value: use it as a cast target (`v :: numerics`)
or a parameter/field type, and every element is checked (recursively, for a slice
interface).

```gad
type numerics []<int|uint|float>   // a named slice of a type union
type users []{ name; id }          // a named slice interface

f := func(xs numerics) => len(xs)
sat := func(v, T) { try { v :: T; return true } catch { return false } }

[
    f([1, 2.5, 3]),                              // ok: every element is numeric
    sat([1, "x"], numerics),                     // false: "x" is not numeric
    sat([{name: "a", id: 1}], users),            // ok: element satisfies { name; id }
    sat([{name: "a"}], users),                   // false: missing id
]
// => [3, false, true, false]
```

## Behavioural interfaces

The builtin interfaces match values by the behaviour they provide, not by a
named type — so they accept anything (built-in or user-defined) that behaves the
right way:

| Interface         | Matches a value that…                     |
|-------------------|-------------------------------------------|
| `iterable`        | can be iterated (array, dict, str, a class with `iterator`) |
| `callable`        | can be called                             |
| `lengther`        | has a length (`len`)                      |
| `indexable`       | supports `obj[i]`                         |
| `indexAssignable` | supports `obj[i] = v`                     |
| `indexDeletable`  | supports deleting an index                |
| `readable`        | can be read from (a reader/buffer)        |
| `writable`        | can be written to (a writer/buffer)       |
| `classType`       | is a class object (`Class(…)` / `class`)  |
| `classInstance`   | is an instance of a class                 |

```gad
sum := func(xs iterable) { total := 0; for v in xs { total += v }; return total }
apply := func(f callable, x) => f(x)
size := func(x lengther) => len(x)
first := func(x indexable) => x[0]
[sum([1, 2, 3]), apply(func(n) => n * 2, 21), size("abc"), first([7, 8, 9])]
// => [6, 42, 3, 7]
```

## Classes: `classType` and `classInstance`

`Class("Name", …)` (or the `class` keyword) yields a **classType**; calling that
class yields a **classInstance**. The two interfaces let a signature require one
or the other.

```gad
class Point { // the statement form names the class (an expression `class { … }` is anonymous)
    x = 0
    y = 0
    methods { sum() => this.x + this.y }
}
// a classType is a class; calling it makes a classInstance
build := func(t classType) <classInstance> => t(; x = 3, y = 4)
p := build(Point)
[typeName(Point), typeName(p), p.sum()]
// => ["Class", "Point", 7]
```

## Example — `type_unions.gad`

```gad
type number <int|uint|float|decimal> // named union (statement form)
addOne := func(v number) <number> => v + 1
// inline unions need no naming — here in both a parameter and a return type
pick := func(v int|str) <int|str> => v
// nested: a union can contain another named union
label := func(v str|number) => isStr(v) ? "text" : "num"
[addOne(2), addOne(2.5), pick(5), pick("x"), 1 :: number, label("hi"), label(3)]

type numerics []<int|uint|float>   // a named slice of a type union
type users []{ name; id }          // a named slice interface

f := func(xs numerics) => len(xs)
sat := func(v, T) { try { v :: T; return true } catch { return false } }

[
    f([1, 2.5, 3]),                              // ok: every element is numeric
    sat([1, "x"], numerics),                     // false: "x" is not numeric
    sat([{name: "a", id: 1}], users),            // ok: element satisfies { name; id }
    sat([{name: "a"}], users),                   // false: missing id
]

sum := func(xs iterable) { total := 0; for v in xs { total += v }; return total }
apply := func(f callable, x) => f(x)
size := func(x lengther) => len(x)
first := func(x indexable) => x[0]
[sum([1, 2, 3]), apply(func(n) => n * 2, 21), size("abc"), first([7, 8, 9])]

class Point { // the statement form names the class (an expression `class { … }` is anonymous)
    x = 0
    y = 0
    methods { sum() => this.x + this.y }
}
// a classType is a class; calling it makes a classInstance
build := func(t classType) <classInstance> => t(; x = 3, y = 4)
p := build(Point)
[typeName(Point), typeName(p), p.sum()]

return "type unions"
```
