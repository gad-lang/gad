
# Generic classes, interfaces and mixins

A `class`, an `interface` or a `mixin` may declare **type parameters** after its
name, as a [function does](type_parameters.gad): `class Pair[K str, V int] { … }`.
Each is a name and, optionally, the type it stands for — its **alias**. A space
before the brackets is allowed: `class P [X int, Y float] { … }`.

- **`Name[A, B]`** is the declaration with its parameters replaced by the
  arguments — K by A, V by B —: a class (interface, mixin) of its own, named so
  (`Pair[float, bool]`). It is a type where a type goes (`p Pair[str, int]`) and
  a value where a value does (`Pair[str, int](; key="k", value=1)`).
- **An argument left out is its parameter's alias**: `Pair[float]` is
  `Pair[float, int]`. A parameter without an alias is `any`.
- **The name alone is the declaration with every alias**: `Pair` is written as
  declared, K `str` and V `int` — `class P [X int, Y float] { x X; y Y }` is
  the class `{ x int; y float }`.
- An argument may be a union, `Box[int|str]`, or another generic's instance,
  `Box[Pair[str, int]]`.

## Aliases and instances

```gad
// X is int and Y is float, unless P is given other types
class P [X int, Y float] { x X; y Y }

// Pair's aliases: K is str, V is int
class Pair[K str, V int] { key K; value V }

p := P(; x=1, y=2.5)                    // P alone: the aliases
a := Pair[float, bool](; key=1.5, value=true)
b := Pair[float](; key=2.5, value=3)    // V left out: its alias, int
[p.x + p.y, typeName(a), typeName(b)]
// => [3.5, "Pair[float, bool]", "Pair[float, int]"]
```

## How it works: one declaration per instance

The compiler makes the instances (monomorphization): each `Name[args]` the block
uses is declared once, beside the generic, its parameters replaced where they
name a type — a field's, a param's, a return's, a parent's, the argument of
another generic (`next? Node[T]`) — and where a value names one (`T(v)`); not as
the name of a field or of a param, nor after a `.`. The instances are consts of
the block, as the generic: used before it is declared, or by a declaration that
comes before, they are there (see [const hoisting](variables_and_scopes.gad)).
`Box[int]` written twice is the same class.

An instance is of the block — the module, the function body — the generic is
declared in: a module that imports a generic gets it by its name alone (its
aliases), and makes no instance of it.

```gad
// a use before the declaration, and in a declaration before it
one := Box[int](; item=1)
class Holder { b Box[int]; many? Box[int|str] }
class Box[T] { item T }                 // T has no alias: any

[
    Holder(; b=one).b.item,
    Box[int] == Box[int],               // one class per instance
    Box(; item="anything").item,        // Box alone: T is any
    Box[int](; item="x") or "rejected", // T is int
]
// => [1, true, "anything", "rejected"]
```

## Referring to itself

A class's body refers to the class by its name — `define`'s first parameter is
the class being made, and it is named as the class —, so a field may be typed by
it: `class Node { v int; next? Node }`. A generic refers to its instance the same
way: `next? List[T]` in `List[str]` is `List[str]`. An interface (`interface N {
next? N }`) and a mixin (`mixin M { next? M }`) may name themselves too.

```gad
class Node { v int; next? Node }
class List[T] { v T; next? List[T] }

n := Node(; v=1, next=Node(; v=2))
l := List[str](; v="a", next=List[str](; v="b"))
[n.next.v, l.next.v, typeName(l.next)]
// => [2, "b", "List[str]"]
```

## Generic interfaces

An interface's instance is checked like any interface — with `::`, or as the type
of a parameter or of a field.

```gad
interface Named[T str] { name T }
interface Chain { v int; next? Chain }  // an interface naming itself

[
    {name: "a"} :: Named or "no",      // Named alone: T is str
    {name: 1} :: Named or "no",
    {name: 1} :: Named[int] or "no",
    {v: 1, next: {v: 2}} :: Chain or "no",
    {v: 1, next: {v: "x"}} :: Chain or "no",
]
// => [{name: "a"}, "no", {name: 1}, {next: {v: 2}, v: 1}, "no"]
```

## Generic mixins, and a mixin as a type

A mixin is used by its instance, `use M[str]`, or by its name alone (its
aliases). A mixin also types a value: by the interface it makes, its
`@interface` — its `this`, its parents and its own members. A field `m? M` takes
an instance of a class that uses M, or anything else that says what M does.

```gad
mixin Valued[T int] { v T }
class Label { use Valued[str] }
class Count { use Valued }              // Valued alone: T is int

mixin Linked { v int = 1; next? Linked }
class Item { use Linked }
class Other { v str = "x" }
class Slot { l? Linked }              // typed by Linked's interface

[
    Label(; v="a").v,
    Count(; v=2).v,
    Item(; next=Item(; v=3)).next.v,
    Slot(; l=Item()).l.v,
    Slot(; l=Other()) or "rejected",
]
// => ["a", 2, 3, 1, "rejected"]
```

## Reflection: `@tparams`

`Type.@tparams` is a key-value array of the type parameters, each with the type
it is there — an array of them for a union —, in their declaration order: the
aliases for the name alone, the arguments for an instance. A class, interface or
mixin that is not generic has none (an empty key-value array).

```gad
class Point [X int, Y float] { x X; y Y }
class Cell[T] { item T }
interface Titled[T str] { title T }
mixin Tagged[T int] { tag T }
class Plain {}

t := Point.@tparams
[
    t[0].k,                                     // the parameters, in order
    dict(t).X == int,                           // the name alone: the aliases
    dict(Cell.@tparams).T == any,                // no alias: any
    dict(Cell[int|str].@tparams).T == [int, str], // a union: an array
    dict(Titled[bool].@tparams).T == bool,
    dict(Tagged[str].@tparams).T == str,
    len(Plain.@tparams),                        // not generic: none
]
// => ["X", true, true, true, true, true, 0]
```

## Limits

- The alias is the type a parameter is when no argument is given; an argument
  is not checked against it (`Pair[float]` is fine though K's alias is `str`).
- An instance is made in the block of the generic's declaration: a module
  importing a generic cannot instantiate it.
- A generic whose instances need ever more instances (`class L[T] { next?
  L[L[T]] }`) stops with a compile error after 1000 of them.

## Example — `generics.gad`

```gad
// X is int and Y is float, unless P is given other types
class P [X int, Y float] { x X; y Y }

// Pair's aliases: K is str, V is int
class Pair[K str, V int] { key K; value V }

p := P(; x=1, y=2.5)                    // P alone: the aliases
a := Pair[float, bool](; key=1.5, value=true)
b := Pair[float](; key=2.5, value=3)    // V left out: its alias, int
[p.x + p.y, typeName(a), typeName(b)]

// a use before the declaration, and in a declaration before it
one := Box[int](; item=1)
class Holder { b Box[int]; many? Box[int|str] }
class Box[T] { item T }                 // T has no alias: any

[
    Holder(; b=one).b.item,
    Box[int] == Box[int],               // one class per instance
    Box(; item="anything").item,        // Box alone: T is any
    Box[int](; item="x") or "rejected", // T is int
]

class Node { v int; next? Node }
class List[T] { v T; next? List[T] }

n := Node(; v=1, next=Node(; v=2))
l := List[str](; v="a", next=List[str](; v="b"))
[n.next.v, l.next.v, typeName(l.next)]

interface Named[T str] { name T }
interface Chain { v int; next? Chain }  // an interface naming itself

[
    {name: "a"} :: Named or "no",      // Named alone: T is str
    {name: 1} :: Named or "no",
    {name: 1} :: Named[int] or "no",
    {v: 1, next: {v: 2}} :: Chain or "no",
    {v: 1, next: {v: "x"}} :: Chain or "no",
]

mixin Valued[T int] { v T }
class Label { use Valued[str] }
class Count { use Valued }              // Valued alone: T is int

mixin Linked { v int = 1; next? Linked }
class Item { use Linked }
class Other { v str = "x" }
class Slot { l? Linked }              // typed by Linked's interface

[
    Label(; v="a").v,
    Count(; v=2).v,
    Item(; next=Item(; v=3)).next.v,
    Slot(; l=Item()).l.v,
    Slot(; l=Other()) or "rejected",
]

class Point [X int, Y float] { x X; y Y }
class Cell[T] { item T }
interface Titled[T str] { title T }
mixin Tagged[T int] { tag T }
class Plain {}

t := Point.@tparams
[
    t[0].k,                                     // the parameters, in order
    dict(t).X == int,                           // the name alone: the aliases
    dict(Cell.@tparams).T == any,                // no alias: any
    dict(Cell[int|str].@tparams).T == [int, str], // a union: an array
    dict(Titled[bool].@tparams).T == bool,
    dict(Tagged[str].@tparams).T == str,
    len(Plain.@tparams),                        // not generic: none
]

return "generics"
```
