# Conventions

[← Back to index](README.md)

This document covers two things: how identifiers are **named** in Gad's builtins
and standard library, and the **code layout** the formatter (`gad fmt` / the
`node.Code` writer) applies. See [Formatting](formatting.md) for the `gad fmt`
command, its flags and the `.gad.yaml` config.

## Naming

These conventions describe how identifiers are named in Gad's builtins and
standard library, so that the API reads consistently. "Specific names" that are
established acronyms (e.g. `URL`, `ID`, `HTTP`) keep their conventional upper
casing, following the Go convention.

| Kind | Case | Examples |
|------|------|----------|
| **Primitive type names** | lowerCamelCase (never PascalCase) | `int`, `uint`, `float`, `str`, `rawStr`, `bytes`, `char`, `bool`, `time`, `date`, `duration` |
| **Other (non-primitive) type names** | PascalCase (or an upper acronym) | `Location` |
| **Constant names** | PascalCase (or an upper acronym) | `time.Hour`, `time.January`, `time.RFC3339`, `time.Type` |
| **Module names** | snake_case | `time`, `strings`, `fmt`, `encoding/base64` |
| **Function / method / property names** | lowerCamelCase (or an upper acronym) | `time.now`, `time.durationString`, `t.year()`, `t.unixNano()` |

### Notes

* A **primitive type** is a built-in value type (`int`, `str`, `time`, `date`,
  `duration`, …); its name is always lowercase. A non-primitive wrapper type
  such as `Location` is PascalCase.
* **Constants** are PascalCase even inside a module whose functions are
  lowerCamelCase — e.g. the `time` module exposes `time.now()` (function) and
  `time.Hour` / `time.RFC3339` (constants).
* **Acronyms** keep their conventional casing as a unit: `URL`, not `Url`;
  `RFC3339`, not `Rfc3339`.

## Code Layout

These rules describe the source layout the formatter produces.

### Declarations

Declaration statements (`var`, `const`, `global`, `param`) share one layout.

#### Single declaration: no parentheses

A declaration of a single spec is written **without** parentheses:

```gad
var x
const Pi = 3.14159
global state
```

Never `var (x)` — the formatter rewrites it to `var x`. This applies to every
declaration keyword.

#### Group declaration: parentheses

Two or more specs are grouped in parentheses. Short groups stay on one line:

```gad
var (x, y)
var (x = 1, y = 2)
const (Min = 0, Max = 10)
```

#### Grouping order

When a group mixes specs with and without an initial value, put the
value-less declarations **first**:

```gad
var (
    // group declarations without value as first
    a, b, c
    d, e
    f = 1
    g = 2
    r = 1, s = 2
    t, u = 3, 4
    v, w, x, y = expr      // destructuring
    (a1, a2; a3, **r) = expr
)
```

Avoid mixing alignment and irregular spacing:

```gad
// bad
var ( a, b, c
    d, e
    f = 1,  g = 2
)
```

### Splitting to new lines

List-like constructs — declaration specs, call arguments, array items, dict
items, key-value arrays and named parameters — are either kept inline or split
one-per-line. The formatter chooses between two modes:

- **Force all to new lines** — when the corresponding
  `CodeWriteContextFlagFormat*InNewLine` flag is set (this is what `gad fmt`
  uses via `CodeWriteContextFlagFormat`), every item goes on its own line:

  ```gad
  var (
      x
      y
  )
  ```

- **Column-aware (`NEW_LINE_CALC`)** — items are wrapped only when the rendered
  line would exceed `ctx.MaxColumns`; otherwise they stay inline (`var (x, y)`,
  `[1, 2, 3]`). Short constructs are left compact.

When items are split one per line, the newline **is** the separator: no comma is
written between items (and none trails the last). Inline lists keep the `, `
separator.

```gad
// inline: comma-separated
x := [1, 2, 3]
d := {a: 1, b: 2}

// wrapped: newline-separated, no commas
x := [
    1
    2
    3
]
d := {
    a: 1
    b: 2
}
```

### Function and call parameters

Function declaration parameters and call arguments may also be written one per
line (a comma is optional; the newline separates them). Two extra rules apply:

- A **typed parameter keeps its ident and type on the same line** (`a int`).
  `a` and `int` on separate lines are two parameters, not a typed one.
- A **type union is spaced around each `|`** when it stays on one line:
  `a int | bool | string`. A single space always precedes the `|`; a trailing
  space follows it **only when the next type is on the same line**.
- When a parameter's **type union is too wide** for the line, continue the type
  on the next line **after a `|`** (the ident stays with the start of the type),
  and put the next parameter on its own following line. Because the next type
  starts a new line, the `|` has no trailing space:

  ```gad
  func(
      a int |
          bool |
          string
      b int
  )
  ```

### Function return types

A function's return-type list is written in angle brackets after the
parameters: ` <T1, T2, ...>`. Each return type follows the same **union spacing**
rule as parameters — a space around each `|` when it stays on one line:

```gad
func() <x int | bool, y str> {
    return 1
}
```

### Match arms

A `match` stays inline while it fits the line budget. When it overflows
(column-aware `NEW_LINE_CALC`), or when the force flag is set, it switches to the
multi-line layout:

- **One arm per line**, separated by the newline — **no comma between arms**.
- Within an arm, the **conditions wrap greedily**: they are packed onto the line
  and continue on a new line only when the next condition would overflow
  `ctx.MaxColumns`. A continuation line is indented **one extra level** (`\t`)
  and is **not** preceded by a comma; conditions sharing a line keep the `, `
  separator. The arm's `: result` / `{ body }` follows the last condition.

```gad
match i {
    1, 2, 3
        4, 5, 6
        7, 8 {}
}
```

When **every** non-else arm's conditions are primitive literals of one comparable
kind (all numeric, or all string), the formatter **sorts** them ascending — the
conditions within each arm and the arms themselves — keeping the `else` arm last:

```gad
// match n { 3: "c", 1: "a", 2: "b", else: "z" }  formats as:
match n { 1: "a", 2: "b", 3: "c", else: "z" }
```

Arms with non-primitive conditions (identifiers, expressions) keep their source
order.
