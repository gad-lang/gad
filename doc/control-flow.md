# Control Flow

[← Back to index](README.md)

## If

`if` works like Go's, including an optional init statement before the condition.

```go
if a < 0 {
    println("negative")
} else if a == 0 {
    println("zero")
} else {
    println("positive")
}

if v := compute(); v > 10 {
    println("big", v)
}
```

The body braces can also be written with `begin` … `end`:

```go
if a > 0 begin println("yes") end
```

## For

The three-clause, condition-only and infinite forms all exist.

```go
for i := 0; i < 3; i++ {   // classic
    println(i)
}

for x < 10 {               // while-style
    x++
}

for {                      // infinite; use break to stop
    if done() { break }
}
```

`break` and `continue` behave as in Go.

## For-In

`for in` iterates any iterable: arrays, dicts, strings, bytes and lazy
iterators (such as the result of `map`/`filter`). Bind one variable for the
value, or two for key/index and value.

```go
for v in [10, 20, 30] {
    println(v)                 // 10, 20, 30
}
for i, v in [10, 20, 30] {
    println(i, v)              // 0 10, 1 20, 2 30
}
for k, v in {a: 1, b: 2} {
    println(k, v)              // a 1, b 2
}
for i, c in "ab" {
    println(i, c)              // 0 'a', 1 'b'
}
```

## Match

`match` (PHP 8-style) compares a subject against arms and yields the first
matching result. The subject needs no parentheses. Each arm lists one or more
conditions (matched against the subject with OR), followed by either `: value`
(expression form) or a `{ … }` block (statement form). An optional `else` arm is
the default; when nothing matches and there is no `else`, the match yields nil.

Expression form — arms are separated by commas or newlines, and an arm may carry
several comma-separated conditions:

```go
label := match n {
    1, 2: "one or two"
    3:    "three",
    else: "other"
}
```

Statement form — arms run a block:

```go
match n {
    1 { return "one" }
    2, 3 { return "few" }
    else { return "other" }
}
```

Arm conditions are arbitrary expressions, so `match` doubles as a clean
multi-branch conditional:

```go
size := match true {
    n < 10:   "small"
    n < 100:  "medium"
    else:     "large"
}
```

An empty match — or one with no matching arm and no `else` — yields nil. An
`else` arm may not be the only arm.

```go
x := match n {}            // nil
y := match 1 { 2: "ok" }   // nil (no match)
```

The formatter keeps a match inline while it fits the line budget and splits the
arms one-per-line only when they overflow (column-aware `NEW_LINE_CALC`), or
always when the corresponding force flag is set. When split, the newline
separates the arms and the `else` arm carries no leading comma.

## Try / Catch / Finally

Gad handles runtime errors (and Go panics surfaced as errors) with
`try`/`catch`/`finally`, similar to ECMAScript. `catch` may bind the error;
`finally` always runs.

```go
try {
    throw "boom"
} catch err {
    println("caught:", err)   // caught: error: boom
} finally {
    println("cleanup")        // always runs
}
```

`catch` and `finally` are each optional, but at least one must be present.
`throw` raises any value as an error. For error values, fallbacks and the `or`
operator, see [Error Handling](error-handling.md).
