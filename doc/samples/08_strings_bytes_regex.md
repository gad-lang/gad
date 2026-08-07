
# Strings, Bytes and Regex

## String forms

| Form                | Example          | Type     | Escapes? |
|---------------------|------------------|----------|----------|
| string              | `"a\tb"`         | `str`    | yes      |
| raw string          | `` `a\tb` ``     | `rawStr` | no       |
| `raw` prefix        | `raw "x"`        | `rawStr` | n/a      |
| interpolated string | `#"hi {name}"`   | `str`    | yes      |

`str` and `rawStr` interoperate; mixing them in `+` yields a string. `raw EXPR`
produces a `rawStr` from any expression (folded at compile time for a string
literal, converted at run time otherwise). Heredocs (`"""…"""`, `` ```…``` ``)
and `code … end` literals have their own tour in [21_heredocs.gad](21_heredocs.gad).

```gad
a := "tab\there"   // str with escapes
b := `no\tescape`  // rawStr, verbatim
c := raw "x" + "y" // raw (x + y) -> rawStr "xy"
[a, b, c, typeName(b)]
// => ["tab\there", no\tescape, xy, "rawstr"]
```

## Interpolated strings

`#"…"` (or `` #`…` ``) interpolates `{expr}` and evaluates to a normal string.
The `#` prefix works on the heredoc forms too.

```gad
name := "Gad"
[#"Hello {name}!", #"sum = {2 + 3}", #`raw {name}`]
// => ["Hello Gad!", "sum = 5", raw Gad]
```

## Bytes

`bytes` is a mutable byte slice. A single-letter prefix glued to a string literal
is a **bytes literal**: `b"…"` are the UTF-8 bytes of the content, `h"…"` the
bytes decoded from hex (whitespace ignored; invalid hex is a compile error). The
prefix must touch the quote — `b "x"` is the variable `b` then a string.

```gad
fromText := b"Hello"
fromHex := h"48 65 6c 6c 6f" // "Hello" in hex (whitespace ignored)
[typeName(fromText), str(fromText) == str(fromHex), fromText[0], str(fromText[1:3])]
// => ["bytes", true, 72, "el"]
```

## Regular expressions

A `/pattern/` literal compiles to a `regexp` at compile time (an invalid pattern
is a compile error). It is only recognised in operand position (after a value `/`
is division). Test/extract with operators or the equivalent methods: `re ~ s` /
`re.match(s)` (bool), `re ~~ s` / `re.find(s)` (first match — index 0 is the
whole match, 1.. the groups), `re ~~~ s` / `re.findAll(s)` (all matches).

```gad
re := /(\w+)@(\w+)/
m := re ~~ "user@host"           // first match: [whole, group1, group2]
all := /\d+/ ~~~ "a1 b22 c333"   // every match
[re.match("user@host"), m[0], m[1], m[2], len(all), all[0][0], all[2][0]]
// => [true, "user@host", "user", "host", 3, "1", "333"]
```

## Replacing

`re.replace(subject, replacement)` replaces every match. The replacement is a
**template** (`$1`, `${name}`, `$$`) or a **callable** (invoked per match; it
receives the matched text plus named args `m` — the full submatch — and `re`).
The `regexp | replacement` operator builds a reusable replacer that composes with
the pipe `.|`.

```gad
swapped := (/(\d+)-(\d+)/).replace("12-34", "$2/$1")     // numbered groups
named := (/(?P<y>\d+)-(?P<m>\d+)/).replace("2024-06", "${m}/${y}")
upper := (/[a-z]+/).replace("hi bye", strings.toUpper)   // callable per match
redact := /(\d{2})(\d+)/ | func(whole; m) => m[1] + strings.repeat("*", len(m[2]))
[swapped, named, upper, "card 1234567890".|redact]
// => ["34/12", "06/2024", "HI BYE", "card 12********"]
```

## Example — `08_strings_bytes_regex.gad`

```gad
a := "tab\there"   // str with escapes
b := `no\tescape`  // rawStr, verbatim
c := raw "x" + "y" // raw (x + y) -> rawStr "xy"
[a, b, c, typeName(b)]

name := "Gad"
[#"Hello {name}!", #"sum = {2 + 3}", #`raw {name}`]

fromText := b"Hello"
fromHex := h"48 65 6c 6c 6f" // "Hello" in hex (whitespace ignored)
[typeName(fromText), str(fromText) == str(fromHex), fromText[0], str(fromText[1:3])]

re := /(\w+)@(\w+)/
m := re ~~ "user@host"           // first match: [whole, group1, group2]
all := /\d+/ ~~~ "a1 b22 c333"   // every match
[re.match("user@host"), m[0], m[1], m[2], len(all), all[0][0], all[2][0]]

swapped := (/(\d+)-(\d+)/).replace("12-34", "$2/$1")     // numbered groups
named := (/(?P<y>\d+)-(?P<m>\d+)/).replace("2024-06", "${m}/${y}")
upper := (/[a-z]+/).replace("hi bye", strings.toUpper)   // callable per match
redact := /(\d{2})(\d+)/ | func(whole; m) => m[1] + strings.repeat("*", len(m[2]))
[swapped, named, upper, "card 1234567890".|redact]

return "strings"
```
