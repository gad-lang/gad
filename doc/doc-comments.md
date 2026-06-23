# Doc Comments

[← Back to index](README.md)

A **doc comment** documents the identifier, declaration, or
`func` / `met` / `meti` / `prop` statement it is attached to. Unlike ordinary
`//` and `/* … */` comments — which are only preserved as-is by the formatter —
a doc comment is *linked to the node it documents*, so `gad fmt` keeps it with
that node (even across declaration merges) and reflows its contents.

Doc-comment contents are **Markdown** (safe inline HTML is allowed).

## Forms

| Form           | Syntax                       | Example                                   |
|----------------|------------------------------|-------------------------------------------|
| `SINGLE`       | `/? text` on its own line    | `/? the pi value`<br>`const Pi = 3.14`    |
| `INLINE`       | `IDENT /? text` (no value)   | `var pi /? the pi value`                  |
| `INLINE_VALUE` | `IDENT = EXPR /? text`       | `const Pi = 3.14 /? the pi value`         |
| `BLOCK`        | `/??` … `??` fenced block    | `/??`<br>`the pi value`<br>`??`           |
| `ROOT_BLOCK`   | `/???` … `???` fenced block  | `/???`<br>`module overview`<br>`???`      |

The fence of a `BLOCK` / `ROOT_BLOCK` must be on its **own line**; `/?? text ??`
on a single line is not a block.

```gad
/? the service listen address
const ServerAddr = ":8080"

const (
    /? the greeting prefix (linked to the spec ident)
    Greeting = "hello"

    Retries = 3 /? how many times to retry (inline, trailing)
)

/??
`sum` returns the sum of `a` and `b`.

A block doc's lines are reflowed as Markdown.
??
func sum(a, b) {
    return a + b
}
```

## What can be documented

- **Declarations** — a `var` / `const` declaration (`GenDecl`) and each value
  spec inside a `( … )` group.
- **Functions** — `func` / `met` statements, including the **func-with-methods**
  form and each method inside it.
- **Properties** — `prop` statements and each accessor method.
- **Method interfaces** — `meti` statements and each header inside them.

```gad
/? a tiny calculator dispatching on argument types
func calc {
    /? add two ints
    (a int, b int) => a + b

    /? add two floats
    (a float, b float) => a + b
}

/? a difference contract
meti differ {
    /? difference of two ints
    (a int, b int) <int>
}
```

A `ROOT_BLOCK` separated from the next statement by a blank line documents the
**module/section**, not that statement:

```gad
/???
This module provides arithmetic helpers.
???

/? add two values
func add(a, b) { return a + b }
```

## Attachment rules

- A `SINGLE` / `BLOCK` / `ROOT_BLOCK` doc on the line **directly above** a target
  is a *lead* doc and links to that target. A **blank line** between the doc and
  the next statement **detaches** it.
- `INLINE` / `INLINE_VALUE` docs trail their target on the **same line** and link
  to its identifier; they apply only when there is no lead doc.
- A doc trailing a comma-separated, value-less identifier (`f, g /? …`) is
  ambiguous and is a **parse error**.

## Formatting

`gad fmt` reflows attached doc comments:

- A `SINGLE` doc that grows past the line-width budget is rewritten as a `BLOCK`;
  a `BLOCK` doc whose content fits on one line collapses back to `SINGLE`. A
  `ROOT_BLOCK` always stays a block.
- The Markdown content is reflowed: soft-wrapped paragraph lines are joined and
  re-wrapped, while fenced code blocks, list items, headings, blockquotes and
  table rows are preserved line-for-line.

```gad
// before
/??
the pi value
??
const Pi = 3.14

// after `gad fmt`
/? the pi value
const Pi = 3.14
```

See also the runnable [`samples/16_doc_comments.gad`](../samples/16_doc_comments.gad)
and the doc-comment layout rules in [Conventions](conventions.md).
