
# Values and Types

In Gad, everything is a value and every value has a type. Use `typeof` (or
`typeName`) to inspect a value's type at runtime.

## Type overview

| Type            | Description                        | Go equivalent       |
|-----------------|-----------------------------------|---------------------|
| `int` / `uint`  | signed / unsigned 64-bit integer  | `int64` / `uint64`  |
| `float`         | 64-bit floating point             | `float64`           |
| `decimal`       | arbitrary-precision decimal (`2d`)| shopspring/decimal  |
| `bool`          | `true` / `false`                  | `bool`              |
| `flag`          | `yes` / `no` (prints `on`/`off`)  | `bool`              |
| `char`          | a single unicode code point (`'A'`)| `rune`             |
| `str` / `rawStr`| unicode / raw (un-escaped) string; a symbol `#name` is a `str` | `string` |
| `bytes`         | byte array (`b"…"`, `h"…"`)        | `[]byte`            |
| `array` / `dict`| ordered list / string-keyed map   | `[]Object` / map    |
| `keyValue` / `keyValueArray` | `k=v` pair / ordered pairs | —              |
| `error` / `nil` | error value / absence of a value  | —                   |

Related chapters: [Collections](collections.gad),
[KeyValue arrays](key_value_array.gad),
[Strings, bytes & regex](strings_bytes_regex.gad),
[Functions](functions.gad), [Properties](properties.gad).

## Type constructors

Every value type is **callable as a constructor** that converts a compatible
value (`int("42")`, `float(3)`, `str(7)`, `char(65)`, `bool(0)`, …). Each is
built from typed methods — one overload per input kind — so the conversion is
chosen by the argument's type. List them with `repr(T; indent)`. New typed
methods can be added with `met` (or from Go with `AddMethod`).

```gad
[int("42"), int("0x1F"), float(-51), str(1984), char(88), int('A')]
// => [42, 31, -51, "1984", 'X', 65]
```

The same holds for the other kinds — `uint`, `decimal` (arbitrary precision),
`rawstr` (an uninterpreted string) — and `array(…)` collects its arguments.

```gad
[uint(3), decimal("1.10") + decimal("2.2"), rawstr("a\\b"), typeName(rawstr("x")), array(1, "a"), bool(0), bool("x")]
// => [3, 3.3, a\b, "rawstr", [1, "a"], false, true]
```

## Type checks

`typeName(v)` names a value's type. Each type has a predicate — `isInt`,
`isUint`, `isFloat`, `isChar`, `isBool`, `isStr`, `isRawStr`, `isBytes`,
`isArray`, `isDict`, `isNil`, `isIterator`, … — plus `isCallable` (anything that
can be called) and `isFunction` (a function value). `is(type, v…)` reports
whether **every** value is of the type; `type` may be an array of types, any of
which matches. For a checked conversion that also accepts interfaces and type
unions use the `::` operator; `cast(T, v)` converts a class instance or a
reflected Go value to the object type `T`.

```gad
class Point { x = 0 }
[
    [isInt(1), isUint(1u), isFloat(1.0), isChar('a'), isBool(true)],
    [isArray([]), isDict({}), isBytes(bytes("")), isRawStr(`r`)],
    [isCallable(len), isFunction(len), isFunction(() => 1), isCallable(1)],
    [isIterator(iterate([1])), isIterator([1])],
    [is(int, 1, 2), is(int, 1, "x"), is([int, str], 1, "x")],
    typeName(cast(Point, Point())),
]
// => [[true, true, true, true, true], [true, true, true, true], [true, true, true, false], [true, false], [true, false, true], "Point"]
```

## Numbers, booleans & flags

Numeric literals: `int` (`19`, `0x1F`=31, `017`=15), `uint` (`5u`), `float`
(`1e10`), `decimal` (`2d`). `bool` is `true`/`false`; `flag` is a distinct on/off
type written `yes`/`no` and printed `on`/`off`.

```gad
[typeName(19), typeName(5u), typeName(1e10), typeName(2d), 0x1F, 017]
// => ["int", "uint", "float", "decimal", 31, 15]
```

## Characters

A `char` is a single unicode code point in single quotes. Adding an int shifts
the code point and keeps the `char` type. `chars(s)` splits a string into its
chars — `len` counts bytes, so it differs from `len(chars(s))` for non-ASCII
text.

```gad
['A' + 1, char(88), int('A'), 'ç' > '9', chars("héy"), len("héy"), len(chars("héy"))]
// => ['B', 'X', 65, true, ['h', 'é', 'y'], 4, 3]
```

## Equality

`==` compares values and coerces between numeric kinds; `===` is strict (same
type and value) and `!==` its negation. For `array`/`dict`, `===` is object
identity — every literal is a fresh object.

```gad
a := [1, 2]
[1 == 1u, 1 === 1u, 1.0 === 1, a === a, a === [1, 2]]
// => [true, false, false, true, false]
```

## Nil

`nil` represents a missing or undefined value: a function with no explicit
`return`, a missing dict key, and some builtins yield `nil`.

```gad
x := func() { y := 4 }() // no explicit return -> nil
[isNil(x), {a: "foo"}["b"] == nil]
// => [true, true]
```

## Copy semantics

Assignment copies values, except the reference types `array`, `dict` and `bytes`,
which share their backing storage (as in Go). Use `copy` for a shallow copy and
`dcopy` for a deep copy — a shallow copy still shares the nested arrays/dicts.
`cap` reports an array's (or bytes') capacity.

```gad
orig := [1, 2, 3]
alias := orig  // shares storage
alias[0] = 99
indep := copy(orig) // independent shallow copy
indep[1] = 0
nested := [[1]]
shallow := copy(nested)
deep := dcopy(nested)
nested[0][0] = 9              // the shallow copy shares the inner array
[orig[0], orig[1], shallow[0], deep[0], cap([1, 2]) >= 2]
// => [99, 2, [9], [1], true]
```

## Example — `values_and_types.gad`

```gad
[int("42"), int("0x1F"), float(-51), str(1984), char(88), int('A')]

[uint(3), decimal("1.10") + decimal("2.2"), rawstr("a\\b"), typeName(rawstr("x")), array(1, "a"), bool(0), bool("x")]

class Point { x = 0 }
[
    [isInt(1), isUint(1u), isFloat(1.0), isChar('a'), isBool(true)],
    [isArray([]), isDict({}), isBytes(bytes("")), isRawStr(`r`)],
    [isCallable(len), isFunction(len), isFunction(() => 1), isCallable(1)],
    [isIterator(iterate([1])), isIterator([1])],
    [is(int, 1, 2), is(int, 1, "x"), is([int, str], 1, "x")],
    typeName(cast(Point, Point())),
]

[typeName(19), typeName(5u), typeName(1e10), typeName(2d), 0x1F, 017]

['A' + 1, char(88), int('A'), 'ç' > '9', chars("héy"), len("héy"), len(chars("héy"))]

a := [1, 2]
[1 == 1u, 1 === 1u, 1.0 === 1, a === a, a === [1, 2]]

x := func() { y := 4 }() // no explicit return -> nil
[isNil(x), {a: "foo"}["b"] == nil]

orig := [1, 2, 3]
alias := orig  // shares storage
alias[0] = 99
indep := copy(orig) // independent shallow copy
indep[1] = 0
nested := [[1]]
shallow := copy(nested)
deep := dcopy(nested)
nested[0][0] = 9              // the shallow copy shares the inner array
[orig[0], orig[1], shallow[0], deep[0], cap([1, 2]) >= 2]

return [typeof(42), typeof("s"), typeof([1]), typeof({a: 1})]
```
