
# Doc Comments

A **doc comment** documents the identifier, declaration, or
`func` / `met` / `meti` / `prop` statement it is attached to. Unlike ordinary
`//` and `/* … */` comments — which the formatter only preserves as-is — a doc
comment is *linked to the node it documents*, so `gad fmt` keeps it with that
node (even across declaration merges) and reflows its contents. Doc-comment
contents are **Markdown** (safe inline HTML is allowed).

## Forms

| Form           | Syntax                       | Example                              |
|----------------|------------------------------|--------------------------------------|
| `SINGLE`       | `/// text` on its own line   | `/// the pi value` above a decl       |
| `INLINE`       | `IDENT /// text` (no value)  | `var pi /// the pi value`            |
| `INLINE_VALUE` | `IDENT = EXPR /// text`      | `const Pi = 3.14 /// the pi value`   |
| `BLOCK`        | `/**` … `**/` fenced block   | a `/**`-fenced block on its own lines |

The fence of a `BLOCK` must be on its **own line**; `/** text **/` on a single
line is not a block.

**File- and section-level docs use the same `/** … **/` form**, distinguished
only by context: a block **directly above** a statement documents that statement,
while a block **followed by a blank line (or at the end of the file)** documents
the **module/section** — like this header. (The older three-star `/*** … ***/`
root block is still accepted but deprecated; `gad fmt` rewrites it to `/** … **/`.)

````gad
/// the service listen address (SINGLE form, on its own line)
const (
    ServerAddr = ":8080"
    /// the greeting prefix (SINGLE, linked to the spec ident)
    Greeting = "hello"
    Retries = 3 /// how many times to retry (INLINE_VALUE, trailing)
)

/**
`sum` returns the sum of `a` and `b`.

A `/**` … `**/` block is the BLOCK form; its lines are reflowed as Markdown. It
can embed a runnable example that `gad doctest` checks against the `>>>` line:

```gad
sum := func(a, b) { return a + b }
sum(2, 3)
>>> 5
```
**/
func sum(a, b) {
    return a + b
}
````

## What can be documented

Declarations (`var` / `const`, and each spec inside a `( … )` group), functions
(`func` / `met`, including the func-with-methods form and each method), `prop`
statements and their accessors, and `meti` headers.

```gad
/// a tiny calculator dispatching on argument types
func calc {
    /// add two ints
    (a int, b int) => a + b

    /// add two floats
    (a float, b float) => a + b
}

/// a difference contract
meti differ {
    /// difference of two ints
    (a int, b int) <int>
}
```

## Attachment rules

- A `SINGLE` / `BLOCK` on the line **directly above** a target is a *lead* doc and
  links to it. A **blank line** in between **detaches** it — a detached block (or
  a block at end of file) is a module/section doc.
- `INLINE` / `INLINE_VALUE` docs trail their target on the **same line** and link
  to its identifier; they apply only when there is no lead doc.
- A doc trailing a comma-separated, value-less identifier (`f, g /// …`) is
  ambiguous and is a **parse error**.

## Formatting

`gad fmt` reflows attached doc comments: a `SINGLE` that grows past the width
budget becomes a `BLOCK`, and an **attached** `BLOCK` that fits on one line
collapses back to `SINGLE`. A **detached** (module/section) block always stays a
`/** … **/` block, and its blank-line separation is preserved so it keeps
documenting the module. Paragraphs are re-wrapped while fenced code, list items,
headings, blockquotes and table rows are preserved line-for-line.

### Code fences

Always tag a code fence with its language so it highlights correctly: `` ```gad ``
for Gad, `` ```gadt `` for a template, `` ```gadx `` for a Gadx template, or
`` ```go `` for a Go-embedding example. A bare `` ``` `` fence is left unhighlighted.

When the fenced code itself contains a run of three or more backticks — a raw
string or heredoc like `` ```…``` `` — open the fence with **more** backticks than
the longest run inside it, and close it with the same count (Markdown's escape for
nested fences). For example, code that uses a five-backtick heredoc goes inside a
six-backtick fence:

``````gad
raw := `````he said ``` here`````   // a 5-backtick raw heredoc, verbatim
``````

## Runnable examples (doctest)

A `BLOCK` may embed a runnable example in a ```` ```gad ````
fence. Examples are self-contained; a line beginning with `>>> ` asserts that the
value produced so far equals the expression after it. `gad doctest PATH…` runs
every embedded example, and `gad doc` runs them while generating (unless
`--no-doctest`). See also [Conventions](conventions.md).

Only an **exactly-three-backtick** `` ```gad `` fence is executed. To show an
**illustrative** Gad fragment that must *not* run — one that references files that
do not exist, repeats a once-only statement, or is otherwise not self-contained —
tag the fence `` ```gad ignore `` (or `` ```gad no-run ``). It still highlights as
Gad on the docs site, but the doctest runner skips it. A wider fence (`` ``````gad ``)
is likewise never executed.

## Example — `doc_comments.gad`

````gad
/// the service listen address (SINGLE form, on its own line)
const (
    ServerAddr = ":8080"
    /// the greeting prefix (SINGLE, linked to the spec ident)
    Greeting = "hello"
    Retries = 3 /// how many times to retry (INLINE_VALUE, trailing)
)

/**
`sum` returns the sum of `a` and `b`.

A `/**` … `**/` block is the BLOCK form; its lines are reflowed as Markdown. It
can embed a runnable example that `gad doctest` checks against the `>>>` line:

```gad
sum := func(a, b) { return a + b }
sum(2, 3)
>>> 5
```
**/
func sum(a, b) {
    return a + b
}

/// a tiny calculator dispatching on argument types
func calc {
    /// add two ints
    (a int, b int) => a + b

    /// add two floats
    (a float, b float) => a + b
}

/// a difference contract
meti differ {
    /// difference of two ints
    (a int, b int) <int>
}

println(Greeting, ServerAddr, "retries:", Retries)
println("sum:", sum(2, 3))            // sum: 5
println("calc ints:", calc(2, 3))     // calc ints: 5
println("calc floats:", calc(2.5, 0.5)) // calc floats: 3

return sum(2, 3)
````
