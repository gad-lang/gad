# Operators

[← Back to index](README.md)

## Unary Operators

| Operator | Meaning             | Notes                                  |
|:--------:|---------------------|----------------------------------------|
| `+`      | identity (`0 + x`)  | numeric / char / bool                  |
| `-`      | negation (`0 - x`)  | numeric / char / bool                  |
| `^`      | bitwise complement  | integer / char / bool                  |
| `!`      | logical NOT         | any value (truthy/falsy)               |

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

## Assignment and Increment

| Op    | Equivalent           | Op     | Equivalent            |
|:-----:|----------------------|:------:|-----------------------|
| `+=`  | `lhs = lhs + rhs`    | `&=`   | `lhs = lhs & rhs`     |
| `-=`  | `lhs = lhs - rhs`    | `\|=`  | `lhs = lhs \| rhs`    |
| `*=`  | `lhs = lhs * rhs`    | `^=`   | `lhs = lhs ^ rhs`     |
| `/=`  | `lhs = lhs / rhs`    | `&^=`  | `lhs = lhs &^ rhs`    |
| `%=`  | `lhs = lhs % rhs`    | `<<=`  | `lhs = lhs << rhs`    |
| `**=` | `lhs = lhs ** rhs`   | `>>=`  | `lhs = lhs >> rhs`    |
| `??=` | assign if `nil`      | `++`   | `lhs = lhs + 1`       |
|       |                      | `--`   | `lhs = lhs - 1`       |

`++` and `--` are statements, not expressions.

## Precedence

Unary operators bind tightest; the ternary operator binds loosest. Binary
operators have five levels:

| Level | Operators                                  |
|:-----:|--------------------------------------------|
| 5     | `*` `**` `/` `%` `<<` `>>` `&` `&^`         |
| 4     | `+` `-` `\|` `^`                            |
| 3     | `==` `!=` `<` `<=` `>` `>=`                 |
| 2     | `&&`                                        |
| 1     | `\|\|`                                      |

## Selectors, Indexers and Slicing

Use `.` (selector) and `[]` (indexer) to read or write elements of arrays,
dicts, strings and bytes. A computed selector uses `.(expr)`.

```go
["one", "two", "three"][1]   // "two"
"foobarbaz"[4]               // 97  (a byte, as int)

m := {a: 1, b: [2, 3, 4]}
println(m.a, m["b"][1])      // 1 3

key := "b"
println(m.(key)[0])          // 2
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
