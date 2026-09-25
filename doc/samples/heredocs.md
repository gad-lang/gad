
# Heredocs and Code Strings

Part of [Strings, bytes & regex](strings_bytes_regex.gad).

## Multiline strings and when to use a heredoc

**Every string form already spans multiple lines** — a single `"…"` or `` `…` ``
may contain source line breaks, and each break becomes a real newline in the
value, verbatim. A `"…"` interprets escapes (so `\n` is also a newline and `\t` a
tab); a `` `…` `` is raw, so `\n` stays the two literal characters while an actual
line break is still a newline.

A **heredoc** is not needed just to go multiline. Use it — a fence of **three or
more** quotes/backticks, PostgreSQL-style — when either:

- the body contains the delimiter itself (`"` or `` ` ``): the wider fence lets it
  appear unescaped, or
- you want the **common leading indentation stripped** so the block stays aligned
  with the surrounding code (this stripping is exclusive to the multi-line
  heredoc; single-delimiter strings keep their indentation verbatim).

| Form                | Single delimiter | Heredoc (3+ fence) | Escapes | Interp. | Strips indent |
|---------------------|------------------|--------------------|---------|---------|---------------|
| string              | `"…"`            | `"""…"""`          | yes     | no      | heredoc only  |
| raw string          | `` `…` ``        | `` ```…``` ``      | no      | no      | heredoc only  |
| template string     | `#"…"`           | `#"""…"""`         | yes     | `{…}`   | heredoc only  |
| template raw string | `` #`…` ``       | `` #```…``` ``     | no      | `{…}`   | heredoc only  |

## Heredocs

A heredoc is delimited by a fence of `"` (or `` ` ``):

- `"""…"""` — a `str`: escapes interpreted.
- `` ```…``` `` — a **raw** `rawStr`: verbatim (no escapes, no interpolation).
- `#"""…"""` / `` #```…``` `` — the template forms, adding `{expr}` interpolation
  (raw templates keep the backslash literal — there is no `\{` escape).

### Fence width — an odd count of three or more

The fence is an **odd** number of the delimiter, **three or more**: `"""` (3),
`"""""` (5), `"""""""` (7), … (and the same in backticks). An even-length fence is
not a heredoc. A **wider fence lets the body contain shorter runs of the
delimiter** without closing early — the fence only closes on a run of *exactly*
the opening width. So inside a `"""` heredoc a single or doubled `"` is just text,
inside a `"""""` (five) heredoc a literal `"""` is text, and so on:

``````gad
"""say "hi" now"""            // a single "  → say "hi" now
""""" a triple """ inside """""  // fence 5, body has """ → a triple """ inside
`````raw ``` fence`````        // fence 5 backticks, body has ``` → raw ``` fence
``````

In the multi-line form the opening/closing fence lines are dropped and the
**common leading indentation is stripped**, so a heredoc stays aligned with the
surrounding code.

## Code strings (`code … end`)

A `code … end` literal captures its body **verbatim** as a plain `str` — it is
not parsed, evaluated or interpolated. The `code`/`end` fences signal that the
body is Gad source (editors highlight it), handy for embedding snippets or
generated code. The block form's closing `end` is the line at the opening
statement's indentation whose only word is `end`; the body is dedented to its own
least-indented line. There is also a single-line form `code <body> end`. A bare
`code` identifier (no matching `end`) is unaffected, so `code := 1` still
declares a variable.

## Examples

A `"…"` or `` `…` `` may span lines directly; each source line break is a real
newline and the text is kept verbatim (no indentation stripping — that is
heredoc-only). In a backtick string `\t` stays literal.

```gad
println("first
second")
println(`raw \t here
next`)
```

Output:

```text
first
second
raw \t here
next
```

A single-line heredoc: the fence is three quotes, so a doubled quote inside is
literal text; escape sequences are interpreted, like a normal `"…"` string.

```gad
["""abc""", """abc""de""", """tab\tend""", """quote: \"x\""""]
// => ["abc", "abc\"\"de", "tab\tend", "quote: \"x\""]
```

A wider odd fence closes only on a run of exactly its width, so the body may
hold shorter runs of the delimiter.

``````gad
[
    """say "hi" now""",                 // a single " is text
    """"" a triple """ inside """"",    // fence of 5: """ is text
    str(`````raw ``` fence`````),       // fence of 5 backticks (a rawStr)
]
// => ["say \"hi\" now", " a triple \"\"\" inside ", "raw ``` fence"]
``````

A multi-line heredoc strips the common indentation; the raw form keeps
backslashes and braces verbatim (it is not a template).

````gad
poem := """
    roses are red
    violets are blue
    """
verbatim := ```
    C:\tmp\file {x}
    line two
    ```
println(poem)
println(verbatim)
````

Output:

```text
roses are red
violets are blue
C:\tmp\file {x}
line two
```

Template heredocs interpolate `{expr}`. A literal brace inside `#"""…"""` is
escaped with `\{` / `\}`, exactly as in an `#"…"` string; the raw template forms
(`` #`…` `` and `` #```…``` ``) keep backslashes verbatim.

````gad
name := "Gad"
n := 3
println(#"""
    hello {name}
    {n} + 1 = {n + 1}
    """)
println(#"""set \{ {name} }""")          // escaped braces
println(#`user home: C:\Users\{name}`)   // raw template: backslashes verbatim
println(#```
    dir: C:\Users\{name}
    n+1 = {n + 1}
    ```)
````

Output:

```text
hello Gad
3 + 1 = 4
set { Gad }
user home: C:\Users\Gad
dir: C:\Users\Gad
n+1 = 4
```

A `code … end` literal captures Gad source verbatim, in block or single-line
form.

```gad
src := code
    for x in [1, 2] {
        println(x)
    }
end
println(src)
println(code a + b end)                   // single-line form
```

Output:

```text
for x in [1, 2] {
    println(x)
}
a + b
```

## Example — `heredocs.gad`

``````gad
println("first
second")
println(`raw \t here
next`)

["""abc""", """abc""de""", """tab\tend""", """quote: \"x\""""]

[
    """say "hi" now""",                 // a single " is text
    """"" a triple """ inside """"",    // fence of 5: """ is text
    str(`````raw ``` fence`````),       // fence of 5 backticks (a rawStr)
]

poem := """
    roses are red
    violets are blue
    """
verbatim := ```
    C:\tmp\file {x}
    line two
    ```
println(poem)
println(verbatim)

name := "Gad"
n := 3
println(#"""
    hello {name}
    {n} + 1 = {n + 1}
    """)
println(#"""set \{ {name} }""")          // escaped braces
println(#`user home: C:\Users\{name}`)   // raw template: backslashes verbatim
println(#```
    dir: C:\Users\{name}
    n+1 = {n + 1}
    ```)

src := code
    for x in [1, 2] {
        println(x)
    }
end
println(src)
println(code a + b end)                   // single-line form

return poem
``````
