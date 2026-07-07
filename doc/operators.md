# Operators

[← Back to index](README.md)

## Unary Operators

| Operator | Meaning             | Notes                                  |
|:--------:|---------------------|----------------------------------------|
| `+`      | identity (`0 + x`)  | numeric / char / bool                  |
| `-`      | negation (`0 - x`)  | numeric / char / bool                  |
| `^`      | bitwise complement  | integer / char / bool                  |
| `!`      | logical NOT         | any value (truthy/falsy)               |
| `++x`    | pre-increment       | variable of int/uint/float/decimal/char, or a temporal value |
| `--x`    | pre-decrement       | variable of int/uint/float/decimal/char, or a temporal value |

The prefix `++x` / `--x` operators mutate the variable **and** evaluate to its
new value, so they can be used inside expressions:

```go
x := 5
y := ++x        // x is 6, y is 6
arr := [0, 0, 0]
i := 0
arr[++i] = 9    // assigns arr[1]
```

On the temporal types (`time`, `calendarDate`, `calendarTime`) `++` **increases**
and `--` **decreases** by the *least-significant non-zero unit* of the value. A
plain `calendarDate` has no time-of-day, so it always steps by a day; a value
with a clock steps by the smallest non-zero component (minute when the seconds
are zero, second otherwise, hour when only the hour is set, and a day at
midnight):

```go
d := 2026-01-31D
++d                                   // 2026-02-01  (one day)

t := time.CalendarTime("2026-01-31 08:05:00")
++t                                   // 08:06:00     (a minute; seconds are 0)
t2 := time.CalendarTime("2026-01-31 08:05:30")
++t2                                  // 08:05:31     (a second)
```

Every value is either truthy or falsy. `0`, `0u`, `0.0`, `""`, empty
collections, `nil`, `no` and `false` are falsy; everything else is truthy.

```go
println(!0, !"", ![], !nil)   // true true true true
println(!1, !"x", ![1])       // false false false
```

## Binary Operators

| Op  | Meaning            | Op   | Meaning                  |
|:---:|--------------------|:----:|--------------------------|
| `+` | add / concat       | `==` | equal                    |
| `-` | subtract           | `!=` | not equal                |
| `===`| strict same       | `!==`| not strict same          |
| `*` | multiply           | `<`  | less than                |
| `/` | divide             | `<=` | less or equal            |
| `%` | modulo             | `>`  | greater than             |
| `**`| power              | `>=` | greater or equal         |
| `&` | bitwise AND        | `\|` | bitwise OR               |
| `^` | bitwise XOR        | `&^` | bit clear (AND NOT)      |
| `<<`| shift left         | `>>` | shift right              |
| `&&`| logical AND        | `\|\|`| logical OR              |

```go
println(7 / 2, 7 % 2, 2 ** 10)   // 3 1 1024
println("foo" + "bar")           // foobar
println([1, 2] + [3])            // [1, 2, 3]
println(6 & 3, 6 | 1, 6 ^ 3)     // 2 7 5
```

`&&` and `||` short-circuit and return one of their operands (not necessarily a
bool):

```go
println(0 || "fallback")   // fallback
println("a" && "b")        // b
```

### Strict same (`===` / `!==`)

`==` compares values and **coerces** across numeric kinds; `===` is **strict**:
the operands must be the same concrete type (and equal value). `a !== b` is just
`!(a === b)`.

```go
println(1 == 1u)    // true   (coerced)
println(1 === 1u)   // false  (int vs uint)
println(1 === 1)    // true
println(1.0 === 1)  // false  (float vs int)
println(1 !== 1u)   // true
```

For non-primitive values (arrays, dicts, class instances, …), `===` is **object
identity**, not deep equality. Every array/dict literal evaluates to a *fresh*
object, so two equal-looking literals are never the same:

```go
a := [1, 2]
println(a === a)         // true   (same object)
println(a === [1, 2])    // false  (a fresh array)
println([1, 2] === [1, 2]) // false
```

A type can customise `===` from Gad with
`met gad.binOpSame(…)`, or from Go via
`ObjectWithSameBinOperator`. When the left operand defines neither, the right
operand's is tried, then primitives fall back to a reflect type+value check and
other objects to address identity.

## Range Operator

`from .. to` builds an inclusive, iterable `Range` (sugar for the `Range(from,
to)` builtin). It supports the numeric kinds (`int`, `uint`, `float`,
`decimal`), `char`, and the temporal types (`time`, `calendarDate`,
`calendarTime`). A range runs ascending or descending depending on its bounds.

```go
for v in 1 .. 5 { print(v) }        // 1 2 3 4 5
for v in 5 .. 1 { print(v) }        // 5 4 3 2 1
for c in 'a' .. 'e' { print(c) }    // a b c d e
```

The step is set with `/` (note `..` binds *tighter* than `/`, so `1 .. 10 / 2`
is `(1 .. 10) / 2`), with the `Range` constructor's `step` named argument, or
with the `r.step(n)` method. For numeric/char ranges the step is a number
(default `1`); for temporal ranges it is a `duration` (default one day).

```go
for v in 1 .. 10 / 2 { print(v) }              // 1 3 5 7 9
for v in Range(0, 10; step=3) { print(v) }     // 0 3 6 9
r := (1 .. 100).step(25)                       // 1, 26, 51, 76
for d in 2026-01-30D .. 2026-02-05D / (dur 48h) { } // every 2 days

r.from   // 1
r.to     // 100
r.step() // 25
```

## Ternary Operator

`cond ? a : b` evaluates to `a` when `cond` is truthy, otherwise `b`.

```go
a := true ? 1 : -1            // 1
min := (x, y) => x < y ? x : y
println(min(5, 10))           // 5
```

## Nullish Operators

`??` returns its right operand only when the left is `nil`. `??=` assigns only
when the current value is `nil`.

```go
println(2 ?? 3)     // 2
println(nil ?? 3)   // 3

a := nil
a ??= 5             // assigns, a == 5
a ??= 9             // no-op,   a == 5
```

`?.` is a nullish selector: it stops and yields `nil` as soon as the receiver is
`nil`, instead of raising an error.

```go
m := {}
println(m.x?.y.z)   // nil  (no error)
m.x = {y: {z: 1}}
println(m.x?.y.z)   // 1
```

`?.(args)` is a nullish call: the callee is evaluated once and called only when
it is not `nil`; otherwise the expression is `nil` and the arguments are not
evaluated. It is the call counterpart of `?.`, shorthand for
`callee != nil ? callee(args) : nil`.

```go
f := nil
println(f?.())          // nil  (no call)
f = (x) => x * 2
println(f?.(21))        // 42

m := {cb: nil}
m.cb?.(1, 2)            // no-op while cb is nil

// chain with ?. at each step to keep the whole chain optional
handlers := {}
handlers.onClick?.()    // nil
```

## Assignment and Increment

| Op    | Equivalent           | Op     | Equivalent            |
|:-----:|----------------------|:------:|-----------------------|
| `+=`  | `lhs = lhs + rhs`    | `&=`   | `lhs = lhs & rhs`     |
| `-=`  | `lhs = lhs - rhs`    | `\|=`  | `lhs = lhs \| rhs`    |
| `*=`  | `lhs = lhs * rhs`    | `^=`   | `lhs = lhs ^ rhs`     |
| `/=`  | `lhs = lhs / rhs`    | `&^=`  | `lhs = lhs &^ rhs`    |
| `%=`  | `lhs = lhs % rhs`    | `<<=`  | `lhs = lhs << rhs`    |
| `**=` | `lhs = lhs ** rhs`   | `>>=`  | `lhs = lhs >> rhs`    |
| `??=` | assign if `nil`      | `x++`  | `x = x + 1`           |
|       |                      | `x--`  | `x = x - 1`           |

The **postfix** `x++` / `x--` are statements. The **prefix** `++x` / `--x` are
[unary expressions](#unary-operators) that also yield the new value.

`++` and `--` are also **binary operators** when an operand follows them
(`a ++ b`, `a -- b`); they have additive precedence and are left-associative.
The built-in numeric types do not define them, but a type can — typically a
class via `met gad.binOpInc(…)` — for example to model
a "push":

```go
Stack := Class("Stack", (cls, define) => define(; fields = (; items = (= []))))
met gad.binOpInc(s Stack, v) {
    s.items = append(s.items, v)
    return s
}
s := Stack()
s ++ 1 ++ 2 ++ 3      // s.items == [1, 2, 3]
```

## Operator handlers and the `gad` namespace

Operator behaviour is dispatched through per-operator functions in the global
**`gad`** namespace (available everywhere without `import`, like `strings`):
`gad.binOp{Op}(left, right)` for binary operators,
`gad.selfAssignOp{Op}(left, right)` for the self-assign forms, and
`gad.unOp{Op}(operand)` for the unary operators (`!`, `-`, `+`, `^`, `++`,
`--`) — e.g. `gad.binOpAdd`, `gad.unOpSub`, `gad.selfAssignOpAdd`. A type
customises an operator by adding a typed method to the matching function:

```go
met gad.binOpAdd(a Vec, b Vec) { … }
met gad.unOpSub(v Vec) { return Vec(; x = -v.x) }
```

You can also call them directly, e.g. `gad.binOpAdd(1, 2)` or
`gad.unOpInc(41)`. Logical NOT (`!`) is universal: any value that does not
define a `gad.unOpNot` overload falls back to its truthiness.

Re-adding an existing method signature is an error unless the declaration uses
the override marker `~` — `met ~gad.binOpAdd(a Vec, b Vec) { … }` (or, in the
block form, `met gad.binOpAdd { ~(a Vec, b Vec) { … } }`) replaces the previous
overload instead of failing.

## User Operators

Three binary operators have **no built-in meaning** and exist purely for types to
define: `<<<`, `>>>` and `%%` (with self-assign forms `<<<=`, `>>>=`, `%%=`).
They have multiplicative precedence (level 5). Give them semantics per type with
the matching per-operator function (`gad.binOpTripleLess`,
`gad.binOpTripleGreater`, `gad.binOpDoubleMod`):

```go
met gad.binOpTripleLess(a int, b int) {
    return a * 1000 + b
}
println(12 <<< 345)        // 12345
```

Using one without a handler is a runtime error (these operators are never
constant-folded by the optimizer). The self-assign form `x <<<= y` runs as
`x = x <<< y` via the `gad.selfAssignOp` fallback; a type can also intercept
it directly with `met gad.selfAssignOpTripleLess(…)`.

## Membership (`in`)

`A in B` tests membership and yields a bool — a **value** for arrays and bytes, a
**key** for the dict kinds, a **substring** for strings. Built-in containers:
`array`, `dict`, `syncDict`, `keyValueArray`, `bytes`, `str`, `rawStr`, and
method-interface instances (membership of a function header). It has comparison
precedence.

```go
2 in [1, 2, 3]        // true
"a" in {a: 1}         // true (key)
104 in bytes("hi")    // true ('h')
'e' in "hello"        // true (char in string)
"ell" in "hello"      // true (substring)
```

`in` is also the **for-in loop** separator, so at the top of a for header
`for x in y` is the loop. Parenthesize to use the operator there:
`for (x in y) { … }` is a for-cond loop. Everywhere else (`if x in y`, the for
condition clause, any expression) `in` is the operator.

A Go type implements membership with the `ObjectWithInBinOperator` interface
(`BinOpIn(vm *VM, value Object) (Object, error)`), implemented by the **right**
operand (the container); it returns a `bool`-valued object for the membership of
`value`. In Gad, a type can define `in` with
`met gad.binOpIn(left T, right U)`.

## Array membership (`ain`)

`A ain B` ("all in") is true when **every** value of the left operand is a member
of `B`. It has the same comparison precedence as `in`. The left operand is an
array of values (a non-array value is treated as a single element, so `x ain B`
matches `x in B`); an empty array is vacuously true.

```go
[1, 2] ain [1, 2, 3]        // true
[1, 4] ain [1, 2, 3]        // false
[] ain [1, 2, 3]            // true   (vacuous)
2 ain [1, 2, 3]             // true   (scalar, like `in`)
["a", "b"] ain {a: 1, b: 2} // true   (dict keys)
```

`ain` is dispatched on the **right** operand. A type provides an optimized
all-membership check by implementing `ObjectWithAinBinOperator`
(`BinOpAin(vm *VM, values Object)`) in Go, or `met gad.binOpAin(left T, right U)` in Gad. When the right operand defines neither, `ain` falls back
to testing each value with `in`, so it works for every container that supports
`in`.

## Assign-to-type (`::`)

`obj :: Type` is the **assign-to-type** operator: it checks that `obj` is
assignable to `Type` and returns `obj` unchanged, otherwise it raises a
catchable type error (`ErrIncompatibleAssign`). Assignability follows the same
rules the method dispatcher uses:

```go
5 :: int               // 5
d :: Animal            // a Dog instance is assignable to a parent class
f :: met<(int) <int>>  // a callable that structurally satisfies a method interface
"hi" :: int            // raises ErrIncompatibleAssign
```

The right operand is a type value — a built-in type, a class, or a structural
`met<…>` / `meti{…}` / `interface{…}` literal. `::` is left-associative and
chains, so a value can be narrowed through several types: `obj::T1::T2::T3`. It
binds tighter than the arithmetic operators (`2 + 3 :: int` is `2 + (3 :: int)`);
parenthesise to cast a whole expression (`(2 + 3) :: int`). The formatter writes
it tightly, `a::B`, dropping redundant chain parens but keeping needed ones.

## Precedence

Unary operators bind tightest; the ternary operator binds loosest. Binary
operators have these levels (`::` binds tightest of the binary operators):

| Level | Operators                                       | Example |
|:-----:|-------------------------------------------------|---------|
| 11    | `::`                                            | `v::int::any` |
| 5     | `*` `**` `/` `%` `<<` `>>` `&` `&^` `<<<` `>>>` `%%` | `a * b` |
| 4     | `+` `-` `\|` `^`                                 | `a + b` |
| 3     | `==` `!=` `===` `!==` `<` `<=` `>` `>=` `in` `ain` | `a == b` |
| 2     | `&&`                                        | `a && b` |
| 1     | `\|\|`                                      | `a \|\| b` |

Higher binds tighter, so `a + b :: int` groups as `a + (b::int)` and
`x || y :: T` as `x || (y::T)`; parenthesise to override (`(a + b)::int`).

## Selectors, Indexers and Slicing

Use `.` (selector) and `[]` (indexer) to read or write elements of arrays,
dicts, strings and bytes. The selector `.name` takes a literal name; use the
indexer `[expr]` for a computed key.

```go
["one", "two", "three"][1]   // "two"
"foobarbaz"[4]               // 97  (a byte, as int)

m := {a: 1, b: [2, 3, 4]}
println(m.a, m["b"][1])      // 1 3

key := "b"
println(m[key][0])          // 2
```

Slices use `[start:end]` on arrays, strings and bytes. A negative index counts
from the end.

```go
[1, 2, 3, 4, 5][1:3]   // [2, 3]
[1, 2, 3, 4, 5][3:]    // [4, 5]
[1, 2, 3, 4, 5][:3]    // [1, 2, 3]
"hello world"[:5]      // "hello"
[1, 2, 3][:-1]         // [1, 2]
```

Keywords cannot be used as bare selectors; index with a string instead:

```go
a := {}
a["func"] = 1   // ok
// a.func = 1   // parse error
```
