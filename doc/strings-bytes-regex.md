# Strings, Bytes and Regex

[← Back to index](README.md)

## String Forms

Gad has several textual literal forms:

| Form                | Example                | Type     | Escapes? |
|---------------------|------------------------|----------|----------|
| string              | `"a\tb"`               | `str`    | yes      |
| raw string          | `` `a\tb` ``           | `rawStr` | no       |
| `raw` prefix        | `raw "x"`              | `rawStr` | n/a      |
| heredoc             | `"""…"""`              | `str`    | yes      |
| raw heredoc         | `` ```…``` ``          | `rawStr` | no       |
| template string     | `#"hi {name}"`         | `str`    | yes      |

```go
"tab\there"     // tab<TAB>here
`no\tescape`    // literally  no\tescape
raw "x" + "y"   // "xy"  (rawStr concatenates with str)
```

`str` and `rawStr` interoperate; mixing them in `+` yields a string.

## Heredocs

A heredoc is delimited by a fence of three or more `"` (or `` ` `` for the raw
form). Leading indentation is stripped to match the closing fence, so heredocs
stay aligned with surrounding code.

```go
s := """
  line1
  line2
  """
println(s)   // line1\nline2
```

## Template Strings

A `#"…"` (or `` #`…` ``) literal is a template string: `{expr}` is interpolated
and the whole thing evaluates to a normal string.

```go
name := "Gad"
println(#"Hello {name}!")     // Hello Gad!
println(#"sum = {2 + 3}")     // sum = 5
println(#`raw {name}`)        // raw Gad
```

## Bytes

`bytes` is a mutable byte slice. Build one with the `bytes` constructor, or with
a **bytes literal**.

### Bytes literals

A single-letter prefix glued directly to a string literal produces a `bytes`
value:

* `b"…"` — the UTF-8 bytes of the string content.
* `h"…"` — the bytes decoded from a hexadecimal sequence.

```go
b"Hello"        // bytes: H e l l o
h"ffccf1c2"     // bytes: 0xff 0xcc 0xf1 0xc2
typeName(b"x")  // "bytes"
str(h"4869")    // "Hi"
```

Any string form may be used as the body — regular string, raw string, heredoc
or raw heredoc — so escapes follow the body's rules:

```go
b"a\nb"     // 3 bytes: 'a', newline, 'b'  (escape processed)
b`a\nb`     // 4 bytes: 'a', '\', 'n', 'b' (raw, no escape)
b"""
hello
"""         // bytes of the heredoc body
```

For `h"…"`, whitespace inside the literal is ignored, so you can group digits:

```go
h"ff cc f1 c2"   // same as h"ffccf1c2"
```

Invalid hex (odd length or non-hex characters) is reported as a **compile
error**.

> The prefix must touch the opening quote. `b "x"` (with a space) is a plain
> identifier `b` followed by a string, not a bytes literal — so existing
> variables named `b` or `h` keep working.

### Indexing and slicing bytes

```go
data := b"Hello"
println(data[0])     // 72  (the byte value, as int)
println(data[1:3])   // bytes "el"
println(len(data))   // 5
```

## Regular Expressions

A `/pattern/` literal compiles to a `regexp` value at compile time (an invalid
pattern is a compile error). Append `p` for POSIX semantics. The same object can
be created at runtime with the `regexp(...)` constructor.

```go
re := /ab+/
re.match("abbb")          // true
re.match("xyz")           // false

// capture groups and replacement; $1, $2 expand groups
r := /(\d+)-(\d+)/
r.replace("12-34", "$2/$1")   // "34/12"
```

Because `/` is also division, a `/regex/` literal is only recognised in operand
position (where a value is expected); after a value, `/` is the division
operator. Use parentheses after value-position keywords, e.g. `return (/re/)`.

### The replace operator `|`

`regexp | replacement` yields a unary replacer function. The replacement may be
a string (with `$1` group expansion) or a callable invoked per match. It
composes with the pipe operator `.|`:

```go
f := /o/ | "0"
f("hello world")              // "hell0 w0rld"
"hello world".|(/o/ | "0")    // "hell0 w0rld"

/[a-z]+/ | func(m) { return "<" + m + ">" }   // wrap each match
```
