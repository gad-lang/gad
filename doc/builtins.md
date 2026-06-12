# Builtin Functions

[← Back to index](README.md)

Gad ships with a library of builtin functions available without an `import`.
This page is a categorised overview; for the full, detailed reference see the
generated [`docs/builtins.md`](../docs/builtins.md).

The host can disable any builtin before compilation, so a sandboxed script may
see a reduced set.

## Type Constructors / Conversions

`int`, `uint`, `float`, `decimal`, `bool`, `flag`, `char`, `string` (alias
`str`), `bytes`, `array`, `chars`, `error`, `keyValue`, `keyValueArray`.

```go
int("42")        // 42
str(42)          // "42"
float(3)         // 3.0
bool(1)          // true
char("X")        // 'X'
decimal(2)       // 2
bytes("hi")      // bytes of "hi"
array(1, 2, 3)   // [1, 2, 3]
```

## Type Inspection

`typeName`, `typeof`, `is`, `isArray`, `isBool`, `isBytes`, `isCallable`,
`isChar`, `isDict`, `isError`, `isFloat`, `isFunction`, `isInt`, `isIterable`,
`isIterator`, `isNil`, `isRawStr`, `isStr`, `isUint`, `isSyncDict`.

```go
typeName([1, 2])   // "array"
isInt(5)           // true
isNil(nil)         // true
isCallable(println)// true
```

## Sequences and Collections

`len`, `append`, `delete`, `copy`, `dcopy`, `repeat`, `contains`, `sort`,
`sortReverse`, `keys`, `values`, `items`, `zip`, `enumerate`.

```go
len([1, 2, 3])             // 3
append([1, 2], 3, 4)       // [1, 2, 3, 4]
contains([1, 2, 3], 2)     // true
repeat([0], 3)             // [0, 0, 0]
sort([3, 1, 2])            // [1, 2, 3]
delete({a: 1, b: 2}, "a")  // mutates → {b: 2}
```

## Iteration

`map`, `filter`, `reduce`, `each`, `iterate`, `iterator`, `collect`, `toArray`,
`zip`, `enumerate`.

Several of these (`map`, `filter`, `keys`, `values`, …) return **lazy
iterators**. Consume them in a `for in` loop or a comprehension, or materialise
them with `collect` / `array`:

```go
// for-in over a lazy map (callback gets value first)
for k, v in map([10, 20, 30], func(v, k) { return v * 2 }) {
    println(k, v)        // 0 20, 1 40, 2 60
}

// reduce eagerly folds
reduce([1, 2, 3, 4], func(acc, v, k) { return acc + v }, 0)   // 10

// collect an iterator into an array
collect(keys({a: 1, b: 2}))   // ["a", "b"]
collect(filter([10, 11, 12, 13], func(v, k, it) { return v % 2 == 0 }))  // [10, 12]
```

Mind the callback argument order: `each` receives `(key, value)`, while
`map` receives `(value, key)`, `filter` receives `(value, key, iterable)` and
`reduce` receives `(accumulator, value, key)`. Comprehensions are often clearer
than `map`/`filter` — see [Collections](collections.md#comprehensions).

## I/O and Formatting

`print`, `println`, `printf`, `sprintf`, `repr`, `read`, `write`, `flush`,
`stdio`.

```go
println("a", 1, [2])     // a 1 [2]
printf("%d-%s\n", 7, "x")// 7-x
s := sprintf("%v", {a: 1})
repr("hi")               // a debug representation
```

## Misc

`globals`, `cast`, `wrap`, `addMethod`, `Class`, `userData`.

`globals()` returns the host-provided globals object (see
[Variables → global](variables-and-scopes.md#global)); `Class` and `addMethod`
support the object/class system.

## Standard Library Modules

Beyond builtins, functionality is grouped into importable modules such as
`strings`, `fmt`, `json`, `time`, `os`, `filepath` and `path`:

```go
strings := import("strings")
println(strings.ToUpper("hi"))   // HI
```

See the generated `docs/stdlib-*.md` files for per-module references.
