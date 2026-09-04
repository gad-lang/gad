# html_comments

Comments and declarations in inline HTML.

An inline HTML region may carry the markup a real document does, and Gadx reads
each for what it is rather than as an element:

- **`<!-- … -->`** is a comment. It nests nothing and renders nothing, so it is
  dropped: a `/` inside it (`js/main.js`) is text, not a self-closing tag.
- **`<!DOCTYPE …>`** and **`<![CDATA[ … ]]>`** are declarations, and are dropped
  the same way. To emit a doctype, use the `!!! 5` statement, which renders
  `<!DOCTYPE html>`.
- **Void elements** — `<meta>`, `<br>`, `<img>`, `<input>`, `<embed>` and the
  rest — have no closing tag and so no children: what follows one is its
  sibling.

A `gad transpile page.html` writes a `.gadx` whose `@main` renders the page,
applying these rules: the doctype becomes `!!! 5`, HTML entities become the
characters they name, and a `{` outside a script or a stylesheet is escaped.

## Components

### main

## Example — `html_comments.gadx`

```gadx
@main
    !!! 5
    <html lang="en">
    <head>
    <!-- A comment renders nothing, whatever it contains: a/b.js, {braces}. -->
    <meta charset="UTF-8">
    <title>Voids and comments</title>
    </head>
    <body>
    <p>Text<br>after a void element</p>
    <img alt="" src="/logo.png">
    </body>
    </html>
```
