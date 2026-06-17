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

### The `raw` prefix

`raw EXPR` produces a `rawStr` from any expression. When `EXPR` is a string
literal the conversion happens at **compile time** (it folds to a constant);
otherwise it converts the evaluated value at **run time**:

```go
raw `a\nb`         // rawStr with a literal backslash-n — folded at compile time
raw "x" + str(1)   // rawStr "x1"
raw str(100)       // rawStr "100" — converted at run time
typeName(raw "x")  // "rawstr"
```

`raw` does not skip a string literal's own escapes: `raw "a\nb"` first turns the
double-quoted `"a\nb"` into a two-line string and then converts it, so use a
raw-string literal (`` raw `a\nb` ``) when you want to keep the backslashes.

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

## Code Strings (`code … end`)

A `code … end` literal is a heredoc-like string whose body is captured
**verbatim** — it is not parsed, evaluated or template-interpolated, it just
becomes a plain `str`. The `code`/`end` fences signal that the body is Gad
source (editors highlight it accordingly), which makes it handy for embedding
snippets, templates or generated code.

The block form spans multiple lines; the closing `end` is the line at the
opening statement's indentation whose only word is `end`. A deeper-indented
`end` (e.g. from an embedded `begin … end`) belongs to the body, and the body is
dedented to its own least-indented line:

```go
src := code
    for x in [1, 2] {
        println(x)
    }
end
println(src)
// for x in [1, 2] {
//     println(x)
// }
```

There is also a single-line form `code <body> end`:

```go
s := code a + b end
println(s)   // a + b
```

A bare `code` identifier (with no matching `end` fence) is unaffected, so
`code := 1` still declares a variable.

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
pattern is a compile error). Append `p` for POSIX semantics, and use Go's inline
flags such as `(?i)` (case-insensitive) inside the pattern. The same object can
be created at runtime with the `regexp(...)` constructor.

```go
re := /ab+/
re.match("abbb")          // true
re.match("xyz")           // false

/a+/p.match("aaa")        // true  — POSIX
(/(?i)hello/).match("HELLO")   // true  — inline flag
```

Because `/` is also division, a `/regex/` literal is only recognised in operand
position (where a value is expected); after a value, `/` is the division
operator. Use parentheses after value-position keywords, e.g. `return (/re/)`.

### Matching and finding

There are operators and equivalent methods for testing and extracting matches:

| Operator        | Method            | Result                                   |
|-----------------|-------------------|------------------------------------------|
| `re ~ s`        | `re.match(s)`     | `bool` — does the pattern match?         |
| `re ~~ s`       | `re.find(s)`      | the first match (full match + groups)    |
| `re ~~~ s`      | `re.findAll(s)`   | all matches                              |

The result of `~~` is a *submatch* value: index `0` is the whole match and
`1, 2, …` are the capture groups. It is indexable (including negative indices),
has a `len`, and is iterable. `~~~` yields a list of such submatch values.

```go
m := /(\w+)@(\w+)/ ~~ "user@host"
m[0]                       // "user@host"  (whole match)
m[1]                       // "user"       (group 1)
m[2]                       // "host"        (group 2)
len(m)                     // 3
for i, g in m { ... }      // 0:user@host, 1:user, 2:host

all := /\d+/ ~~~ "a1 b22 c333"
len(all)                   // 3
all[0][0]                  // "1"
all[2][0]                  // "333"
```

### Replacing

`re.replace(subject, replacement)` returns `subject` with every match replaced.
The replacement is either a **template string** or a **callable**:

- In a template, `$1`, `$2`, … expand numbered capture groups, `${name}` expands
  a named group `(?P<name>…)`, and `$$` is a literal `$`.
- A callable is invoked once per match. It receives the matched text as the
  positional argument, plus two named arguments: `m` — the full submatch
  (`m[0]` is the whole match, `m[1]`, `m[2]`, … are the capture groups) — and
  `re`, the regexp itself. It returns the replacement string.

A `bytes` subject yields a `bytes` result.

```go
// numbered groups
(/(\d+)-(\d+)/).replace("12-34", "$2/$1")              // "34/12"
// named groups
(/(?P<y>\d+)-(?P<m>\d+)/).replace("2024-06", "${m}/${y}")  // "06/2024"
// literal dollar
(/x/).replace("ax", "$$")                              // "a$"
// callable on the whole match
(/[a-z]+/).replace("hi bye", func(m) { return "<" + m + ">" })  // "<hi> <bye>"
(/[a-z]+/).replace("hi bye", strings.toUpper)          // "HI BYE"
// callable using capture groups via the named arg `m`
(/(\w+)@(\w+)/).replace("user@host", func(whole; m) { return m[2] + "@" + m[1] })  // "host@user"
// bytes subject -> bytes result
(/o/).replace(bytes("foo"), "0")                       // bytes("f00")
```

### The replace operator `|`

`regexp | replacement` yields a unary replacer function. The replacement is the
same string or callable accepted by `replace` (a callable still receives the
`m` and `re` named arguments). It composes with the pipe operator `.|`:

```go
f := /o/ | "0"
f("hello world")              // "hell0 w0rld"
"hello world".|(/o/ | "0")    // "hell0 w0rld"

/[a-z]+/ | func(m) { return "<" + m + ">" }   // wrap each match

// a group-using replacer: keep the first 2 digits, mask the rest
redact := /(\d{2})(\d+)/ | func(whole; m) { return m[1] + strings.repeat("*", len(m[2])) }
"card 1234567890".|redact     // "card 12********"
```
