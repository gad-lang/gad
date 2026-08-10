
# Strings, Bytes and Regex

## String forms

| Form                    | Example            | Type     | Escapes? | Interp. |
|-------------------------|--------------------|----------|----------|---------|
| string                  | `"a\tb"`           | `str`    | yes      | no      |
| raw string              | `` `a\tb` ``       | `rawStr` | no       | no      |
| `raw` prefix            | `raw "x"`          | `rawStr` | n/a      | no      |
| interpolated string     | `#"hi {name}"`     | `str`    | yes      | `{…}`   |
| interpolated raw string | `` #`hi {name}` `` | `rawStr` | no       | `{…}`   |

`str` and `rawStr` interoperate; mixing them in `+` yields a string. `raw EXPR`
produces a `rawStr` from any expression (folded at compile time for a string
literal, converted at run time otherwise). Heredocs (`"""…"""`, `` ```…``` ``)
and `code … end` literals have their own tour in [21_heredocs.gad](21_heredocs.gad).

```gad
name := "Gad"
a := "tab\there"    // str with escapes
b := `no\tescape`   // rawStr, verbatim
c := raw "x" + "y"  // raw (x + y) -> rawStr "xy"
d := #"hi {name}"   // interpolated string -> str "hi Gad"
e := #`hi {name}`   // interpolated raw string -> rawStr "hi Gad" (\ verbatim)
[a, b, c, d, e, typeName(b), typeName(e)]
// => ["tab\there", no\tescape, xy, "hi Gad", hi Gad, "rawstr", "rawstr"]
```

## Raw strings

A `` `…` `` literal is a **raw** string (`rawStr`): the body is verbatim — no
escape sequences and no interpolation, so `\t`, `\`, `{` and `}` are all literal
text. Backtick literals may also **span multiple lines** (the newlines are kept),
which is the raw multiline form; the `` ```…``` `` raw heredoc in
[21_heredocs.gad](21_heredocs.gad) additionally strips common indentation.

Prefix a raw form with `#` for its **template** variant (`` #`…` ``): `{expr}`
interpolation is enabled, but the text stays verbatim — the backslash is *not* an
escape, so there is no `\{` escape (write `` #"…" `` when you need one). A raw
template is handy for paths and regex-like text: `` #`C:\Users\{name}` ``.

```gad
name := "Gad"
verbatim := `C:\tmp\{x}`     // rawStr: \ and {…} are literal (no escapes/interp)
lines := `first
second`                      // backtick literals span lines verbatim
tmpl := #`path C:\{name}`    // template raw str: \ stays literal, {name} interpolates
[verbatim, lines, tmpl, typeName(verbatim), typeName(tmpl)]
// => [C:\tmp\{x}, first
second, path C:\Gad, "rawstr", "rawstr"]
```

## Interpolated strings

`#"…"` (or `` #`…` ``) interpolates `{expr}` and evaluates to a normal string.
The `#` prefix works on the heredoc forms too.

```gad
name := "Gad"
[#"Hello {name}!", #"sum = {2 + 3}", #`raw {name}`]
// => ["Hello Gad!", "sum = 5", raw Gad]
```

## Escaping the delimiters

A literal brace in interpolated text is written `\{` or `\}`. Braces inside an
expression's **own** string literal (`{ "…{…}…" }`) need no escaping — the
interpolation scanner is brace-balanced and string-aware, so it stops at the
matching `}`. The `\{` / `\}` escape is consistent across every text form: `#"…"`
and `#"""…"""`, ordinary `"…"` strings (where it collapses to a literal brace),
`.gadt` templates (`\{%`) and gadx `@text` / `@md` blocks.

```gad
name := "Gad"
lit := #"literal \{ and \}"          // escaped braces -> literal text
mix := #"{name} in \{ braces }"      // escape + real interpolation
js := #"json: \{ {str({a: 1})} }"    // braces in the expr's own literal are fine
plain := "regular \{ ok"             // works in ordinary strings too
[lit, mix, js, plain]
// => ["literal { and }", "Gad in { braces }", "json: { {a: 1} }", "regular { ok"]
```

## Quoting at run time

`gad.quote` encodes a value as a Gad string literal and `gad.unquote` decodes one
back. Each has a `str` and a `rawstr` overload: the argument type selects the
cooked (`"…"`) or raw (`` `…` ``) form. `gad.quote` takes named arguments
`maxCols` (default 120) — switching to a multi-line heredoc when a single line
would exceed it — and `fence` (default 3) — the heredoc start/end delimiter count
(an odd number, widened past any longer run in the body). These map to the
`github.com/gad-lang/gad/quote` package.

```gad
[
    gad.quote("hi"),                          // -> "hi"    (cooked literal text)
    gad.quote(`c:\x`),                        // -> `c:\x`  (raw literal text)
    gad.unquote("\"a\\tb\"") == "a\tb",       // decode -> value with a real tab
    gad.unquote(gad.quote("x\ny")) == "x\ny", // round-trip
    gad.quote("a\nb\nc"; maxCols=4),          // long line -> multiline heredoc
]
// => ["\"hi\"", "`c:\\x`", true, true, "\"\"\"a\nb\nc\"\"\""]
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
name := "Gad"
a := "tab\there"    // str with escapes
b := `no\tescape`   // rawStr, verbatim
c := raw "x" + "y"  // raw (x + y) -> rawStr "xy"
d := #"hi {name}"   // interpolated string -> str "hi Gad"
e := #`hi {name}`   // interpolated raw string -> rawStr "hi Gad" (\ verbatim)
[a, b, c, d, e, typeName(b), typeName(e)]

name := "Gad"
verbatim := `C:\tmp\{x}`     // rawStr: \ and {…} are literal (no escapes/interp)
lines := `first
second`                      // backtick literals span lines verbatim
tmpl := #`path C:\{name}`    // template raw str: \ stays literal, {name} interpolates
[verbatim, lines, tmpl, typeName(verbatim), typeName(tmpl)]

name := "Gad"
[#"Hello {name}!", #"sum = {2 + 3}", #`raw {name}`]

name := "Gad"
lit := #"literal \{ and \}"          // escaped braces -> literal text
mix := #"{name} in \{ braces }"      // escape + real interpolation
js := #"json: \{ {str({a: 1})} }"    // braces in the expr's own literal are fine
plain := "regular \{ ok"             // works in ordinary strings too
[lit, mix, js, plain]

[
    gad.quote("hi"),                          // -> "hi"    (cooked literal text)
    gad.quote(`c:\x`),                        // -> `c:\x`  (raw literal text)
    gad.unquote("\"a\\tb\"") == "a\tb",       // decode -> value with a real tab
    gad.unquote(gad.quote("x\ny")) == "x\ny", // round-trip
    gad.quote("a\nb\nc"; maxCols=4),          // long line -> multiline heredoc
]

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
