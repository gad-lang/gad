
# Operators

This is the operators chapter. Focused runnable tours live in nearby samples:
[ranges `..`](ranges.gad), [membership `in`](in_operator.gad),
[unary](unary_operators.gad) and
[absent-coalescing `!?`](absent_coalescing.gad). This sample demonstrates the
**user operators** and `met gad.*` handlers (Example below).

## Unary

`+` (identity), `-` (negation), `^` (bitwise complement), `!` (logical NOT, on
any value's truthiness), and the prefix `++x` / `--x` which mutate a variable and
yield the new value. On temporal types `++`/`--` step by the least-significant
non-zero unit. Falsy values: `0`, `0u`, `0.0`, `""`, empty collections, `nil`,
`no`, `false`.

## Binary

Arithmetic `+ - * / % **`, bitwise `& | ^ &^ << >>`, comparison
`== != < <= > >=`, logical `&& ||` (short-circuit, return an operand). `+`
concatenates strings and appends to arrays. `===`/`!==` are **strict** (same
concrete type; object identity for non-primitives), customizable via
`met gad.binOpSame`.

## Ranges, ternary, nullish

`from .. to` builds an inclusive iterable `Range` (step with `/` or the `step`
arg; `..` binds tighter than `/`). `cond ? a : b` is the ternary. `??` returns
the right operand only when the left is `nil`; `?.` is a nullish selector and
`?.(args)` a nullish call (evaluate/call only when non-nil).

## Absent-coalescing (`!?` / `!?=`)

The **existence-based** counterparts of `??`/`??=`: they test whether a *key is
present* (a present `nil` still counts). `a.b !? d` yields `a.b` when `b` exists,
else `d`; `a.b !?= d` assigns only when absent (auto-creating missing
intermediate dicts). See [absent_coalescing.gad](absent_coalescing.gad).

## Assignment, increment & array append

`+= -= *= /= %= **= &= |= ^= &^= <<= >>= ??=`, postfix `x++`/`x--` (statements).
Array append has four forms: `arr + x` (append one, concatenating an iterable),
`arr ++ it` (extend with an iterable), `arr += x` (append x as **one** element,
in place), `arr ++= it` (extend in place). A single target with a comma list is
an array literal (`x := 1, 2, 3`).

## Operator handlers and the `gad` namespace

Operator behaviour is dispatched through per-operator functions in the global
`gad` namespace (no import needed): `gad.binOp{Op}(l, r)`,
`gad.selfAssignOp{Op}(l, r)` and `gad.unOp{Op}(v)`. A type customizes an operator
by adding a typed method (`met gad.binOpAdd(a Vec, b Vec) { … }`); `met ~` with
an override marker replaces an overload. They are callable directly
(`gad.binOpAdd(1, 2)`).

## User operators

Three binary operators have **no built-in meaning** and exist purely for types:
`<<<`, `>>>`, `%%` (self-assign `<<<=`, `>>>=`, `%%=`), at multiplicative
precedence. Give them semantics with `gad.binOpTripleLess` /
`gad.binOpTripleGreater` / `gad.binOpDoubleMod`. Using one without a handler is a
runtime error.

```gad
/**
Define `<<<` and `>>>` on ints as "push"/"pop"-ish helpers.
**/
met gad.binOpTripleLess(a int, b int) {
    return a * 1000 + b           // pack two small ints
}
met gad.binOpTripleGreater(a int, b int) {
    return [a / 1000, a % 1000]   // unpack
}

packed := 12 <<< 345
println(packed)                   // 12345
println(packed >>> 0)             // [12, 345]

/**
`%%` as a "clamp into range" operator.
**/
met gad.binOpDoubleMod(v int, hi int) {
    return v < 0 ? 0 : (v > hi ? hi : v)
}
println(50 %% 10, -3 %% 10, 7 %% 10)   // 10 0 7

/**
The self-assign forms reuse the binary handler via gad.selfAssignOp's
fallback: `x <<<= y` is `x = x <<< y`.
**/
acc := 1
acc <<<= 2
println(acc)                      // 1002

/**
A dedicated gad.selfAssignOp handler can differ from the binary one.
**/
met gad.selfAssignOpDoubleMod(a int, b int) {
    return a + b
}
n := 7
n %%= 5
println(n)                        // 12
```

## Membership, assign-to-type, precedence

`A in B` tests membership (value/key/substring; also the for-in separator —
parenthesize to use as an operator); `A ain B` is "all in". `obj :: Type` checks
assignability and returns `obj` (else raises `ErrIncompatibleAssign`), chaining
`obj::T1::T2`. Precedence (higher binds tighter): `::` (11); `* ** / % << >> & &^
<<< >>> %%` (5); `+ - | ^` (4); comparisons + `in`/`ain` (3); `&&` (2); `||` (1);
unary tightest, ternary loosest. Read/write with `.name` (literal) and `[expr]`
(computed); slice with `[start:end]` (negative indices count from the end).

## Example — `user_operators.gad`

```gad
/**
Define `<<<` and `>>>` on ints as "push"/"pop"-ish helpers.
**/
met gad.binOpTripleLess(a int, b int) {
    return a * 1000 + b           // pack two small ints
}
met gad.binOpTripleGreater(a int, b int) {
    return [a / 1000, a % 1000]   // unpack
}

packed := 12 <<< 345
println(packed)                   // 12345
println(packed >>> 0)             // [12, 345]

/**
`%%` as a "clamp into range" operator.
**/
met gad.binOpDoubleMod(v int, hi int) {
    return v < 0 ? 0 : (v > hi ? hi : v)
}
println(50 %% 10, -3 %% 10, 7 %% 10)   // 10 0 7

/**
The self-assign forms reuse the binary handler via gad.selfAssignOp's
fallback: `x <<<= y` is `x = x <<< y`.
**/
acc := 1
acc <<<= 2
println(acc)                      // 1002

/**
A dedicated gad.selfAssignOp handler can differ from the binary one.
**/
met gad.selfAssignOpDoubleMod(a int, b int) {
    return a + b
}
n := 7
n %%= 5
println(n)                        // 12

return packed
```
