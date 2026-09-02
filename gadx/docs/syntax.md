# Template Syntax

Gadx uses indentation to describe HTML, components, and Gad control flow.

## Comments

Line comments start a line:

```gadx
// a silent comment — never rendered
/// a single-line doc comment (attaches to the next declaration)
```

A `//` line is a plain, silent comment. A `///` line is a documentation comment
(see below). To emit an **HTML comment** into the output, write it inline as an
HTML region instead — `<!-- … -->` is preserved verbatim:

```gadx
<!-- rendered into the output as an HTML comment -->
```

Block comments `/* … */` are silent and may span multiple lines. They are only
recognized at the start of a line (a `/*` mid-line stays literal text):

```gadx
/* a silent note */

/*
  a multi-line
  block comment
*/
```

A doc comment uses the gad convention `/** … **/` (opened with `/**`, closed with
`**/`). When it immediately precedes a `@comp`, `@func`, `@param`, `@var`,
`@const`, `@enum` or `@export`, it documents that declaration (surfaced by
`gad doc`); a blank line or any other statement in between leaves it as a plain
silent comment.

```gadx
/** Renders a greeting for the given name. **/
@comp greeting(name)
    p {= "Hello, " + name }
```

## Document Type

```gadx
!!! 5
```

Output:

```html
<!DOCTYPE html>
```

## Tags

```gadx
section.hero
    h1 Welcome
    p Ship templates with less noise.
```

Output:

```html
<section class="hero"><h1>Welcome</h1><p>Ship templates with less noise.</p></section>
```

## Ids And Classes

```gadx
main#content.page.shell
    h1.title Hello
```

Output:

```html
<main id="content" class="page shell"><h1 class="title">Hello</h1></main>
```

## Attributes

```gadx
a.button[href="/docs"][target="_blank"] Read docs
img.cover[src=Post.CoverImage][alt=Post.Title]
input[type="email"][name="email"][required]
```

Attributes can use string literals, variables, or expressions.

### Attribute groups

A single `[ … ]` group may hold multiple attributes, separated by commas or
newlines — like a GAD `KeyValueArray (; … )`. A group may span several lines up
to its closing `]`; indentation inside is ignored. Repeated attributes (such as
`class`) are merged.

```gadx
// one attribute per group (still valid)
div[class="a"][title="hello"]

// many attributes in one group, comma-separated
div[class="a", title="hello"]

// mix group forms
div[class="a"][class="b", title="hello"]

// span multiple lines up to the closing ]
div[
    class="a"
    class="b"
    title="hello"
]
```

Commas and newlines inside strings, parentheses, brackets or braces do not split
a group, so call arguments and array/dict literals work as attribute values:

```gadx
div[title=join(items, ", "), data-ids=[1, 2, 3]]
```

A trailing `? condition` applies to every attribute in the group.

### The `class` attribute

`class` accepts several value shapes, all merged with any static `.token`
classes on the tag into a single space-separated list:

```gadx
// string
div[class="card"]                          // class="card"

// static tokens merged with a class attribute
div.card.wide[class="active"]              // class="card wide active"

// array — falsy entries (nil, false, "") are dropped
div[class=[cond ? "on" : nil, "base"]]     // class="base" (or "on base")

// dict — JSX/Vue object form: keep each key whose value is truthy.
// Keys are emitted in sorted order, so output is deterministic.
button[class={primary: true, active: on, muted: off}]  // class="active primary"
```

Multiple `[class=…]` groups accumulate, so static tokens, arrays and dicts can
be combined: `div.base[class=["x"]][class={hl: on}]` → `class="base x hl"`.

### Boolean and flag attributes

Presence is controlled by the **flag** type (`yes` / `no`), and is separate
from a boolean **value** (`true` / `false`):

```gadx
div[data-value]            // <div data-value>          — valueless ≡ =yes
div[data-value=yes]        // <div data-value>          — bare boolean attribute
div[data-value=no]         //                           — attribute omitted
div[data-value=true]       // <div data-value="true">   — literal value
div[data-value=false]      // <div data-value="false">  — literal value
```

So a valueless attribute `[x]` is the flag `yes` (like a named param
`fn(;x)`): it renders bare, with no value, and `[x=no]` drops it entirely.
A `bool` (`true`/`false`) always renders its literal value. Other falsy values
(`nil`, `""`) omit the attribute.

## Inline HTML

A line starting with `<` is parsed as an inline HTML region. It runs from the
opening tag to its matching close tag (spanning multiple lines if needed), and
runs of whitespace collapse to a single space. `<> … </>` is a fragment: it
lowers to a [`gadx.Elements()`](./api.md#render-tree) node that renders its
children with no wrapping element (spliced into the enclosing parent).

```gadx
@main
    <a href="/" class="link">Home</a>

    <ul>
        <li>one</li>
        <li>two</li>
    </ul>
```

The region is **compiled to the same `gadx.Tag` / `gadx.Text` elements as the
pug-style tag syntax** — it is not emitted as a raw string. So it shares the
tag rendering rules: void elements self-close (`<br>` → `<br />`), valueless
boolean attributes stay bare (`<input disabled>` → `<input disabled />`), and
attributes are classified (regular attributes first, then the class list).
Because the AST carries real gadx tag nodes, an inline HTML region also
transpiles back to pug-style gadx (`gofmt`-style, via `WriteGadx`):
`<a href="/x">Home</a>` becomes `a[href="/x"]` with an indented `| Home`.

Attributes may be interpolated with `{expr}` — both the value and the name. An
interpolated value is auto-quoted and HTML-escaped (a falsy value drops the
attribute); an interpolated name builds the attribute name from the expression
(lowered to a computed `**{[name]: value}` spread). In **text content** the rule
is `{= expr }` to emit and `{ expr }` for control (see below), so a value inside a
tag is `{= expr }`. Interpolation source positions are preserved.

```gadx
@main
    <a href={post.URL} data-{key}={value}>
        {= post.Title }
    </a>
```

### Interleaving gadx statements

Block-level gadx statements (`@if`, `@for`, `@match`, `+comp`, `~code`) may be
interleaved inside an HTML region by indentation. The directive line and every
line indented deeper than it form the block — which may itself contain HTML —
and it renders as a child of the enclosing element, in source order alongside
sibling HTML. `@else` / `@else if` clauses continue the block at the same
indentation.

```gadx
@main
    <ul class="menu">
        <li>Home</li>
        @if user
            <li>{user.name}</li>
        @else
            <li>Sign in</li>
        @for item in items
            <li>{item}</li>
    </ul>
```

The interleaved block is parsed by the full gadx parser, so any gadx construct
is available; source lines resolve back to the original `.gadx` file (line
accurate). Runnable examples are in `samples/gadx/html.gadx` and
`samples/gadx/html_control_flow.gadx`. Use the pug-style `tag[attr=…]` syntax
(below) when you prefer gadx's indentation-based nesting throughout.

## Output vs control — `{= expr }` vs `{ expr }`

In text content (a tag body, `| ` text, or an inline HTML region) an interpolation
comes in two forms:

- `{= expr }` **emits** the value (interpolation / output).
- `{ expr }` is a **control** statement: it is evaluated but emits nothing — use it
  for side effects (`{ log(x) }`) or, more usually, just prefer the `@if` / `@for`
  directives.

```gadx
p
    | Hello {= name }        # emits: "Hello Ada"
    { track("seen") }        # runs, emits nothing
```

`{= }` accepts any Gad expression, including interpolated strings, so
`{= #"Hi {name}!" }` works. (Attribute values are always the expression form
`attr={expr}`; the `{=}`/`{}` distinction is about text content.)

## Text

Inline text — right after a tag on the same line:

```gadx
p Hello world
```

A tag may carry inline text **and** an indented block of children: the inline
text and the child tags are all children of the tag, in order.

```gadx
h2 Example —
    code file.gad
```

renders `<h2>Example —<code>file.gad</code></h2>`.

Per-line `| ` text. Consecutive `| ` lines are joined with **no** separator:

```gadx
p
    | first
    | second
```

renders `<p>firstsecond</p>`. `{= expr }` interpolation works in every text form.

### Literal and folded blocks (`|` / `|>`)

A bare `|` (or `|>`) on its own line opens a **YAML-style text block** whose
indented lines carry no `| ` prefix — the two styles differ in how line breaks
are handled, exactly like YAML’s `|` and `>`:

| Marker | YAML  | Line breaks                    |
|--------|-------|--------------------------------|
| `\|`   | `\|`  | **kept** (literal)             |
| `\|>`  | `>`   | become **spaces** (folded)     |

```gadx
pre
    |
        one
        two          # -> "one\ntwo"

p
    |>
        one
        two
        three        # -> "one two three"
```

The `|` / `|>` may sit **inline** right after the tag — `p |>` is the same as `p`
with an indented `|>` block:

```gadx
p |>
    one
    two
    three            # -> <p>one two three</p>
```

### `@text` — verbatim literal block

`@text` emits its indented body verbatim: no tag/directive parsing (bare words
stay words), line breaks and blank lines preserved. `{= expr }` still
interpolates. Use it for preformatted content (license headers, ASCII art,
`<pre>` bodies). `@p` is the paragraph variant (blank lines split `<p>` tags).

```gadx
@text
    License (c) {= year }
      indented lines keep their spaces
```

## Markdown block (`@md`)

`@md` writes its indented body as Markdown. At compile (and transpile) time the
Markdown is rendered to HTML by goldmark — GFM tables/strikethrough/task
lists/autolinks, typographer, definition lists, footnotes and auto heading ids —
and that HTML is parsed into the **same `gadx.Tag` / `gadx.Text` elements** as the
inline-HTML and pug-style syntax. So `@md` produces a real tag tree, and
`gad transpile` emits it as plain gadx tag code (not a runtime Markdown string).

```gadx
@param (; title = "Release notes")
@main
    @md
        # {= title }

        A **Markdown** paragraph with a <span class="tag">mixed {= title }</span> HTML span.

        @p
            A nested @p directive renders inline as its own tag.
```

- `{= expr }` interpolation becomes a **dynamic value inserted into the fixed HTML
  structure** — the Markdown layout is fixed at compile time and the value is
  evaluated at render time, so an interpolated value is not re-parsed as Markdown.
- **Inline and block HTML** mixed into the Markdown is supported; it flows through
  goldmark and is parsed into tags like everything else.
- **Nested `@` directives** render inline as their own tags; Markdown text before
  and after a directive is its own section.
- Interpolation **source positions are preserved** (AST and bytecode source map),
  so runtime errors and debug stepping map back to the `.gadx`. Write a literal
  brace as `\{` / `\}`.

The renderer is the package-level `gadx.Markdown` variable; an embedding Go
program can replace or extend it before compiling. See `samples/gadx/markdown.gadx`.

## Expressions

```gadx
h1 {= Model.Title}
p {= "Hello " + User.Name}
```

Use Gad expressions inside `{= ...}`.

## Escaping and raw HTML values

Interpolated values are **HTML-escaped by default**, in both text content and
attribute values, so untrusted data cannot inject markup (XSS-safe): a `<b>` in
`userInput` becomes `&lt;b&gt;`, and a `"` in an attribute value becomes `&#34;`
so it cannot break out of the quoted attribute.

```gadx
p {= userInput}
a[title=userInput] link
```

Opt out for **trusted** HTML with `raw` (a `gad.RawStr` is written verbatim):

```gadx
article {= raw Post.Body}
```

`@md`, inline HTML regions and pug-style tags produce their own markup (they are
not affected by text escaping). Attribute-value quoting is configurable via the
package-level `gadx.AttrValueQuote`: the default `gadx.AttrQuoteHTML`
double-quotes and entity-escapes; `gadx.AttrQuoteSingleQuote` single-quotes and
escapes only `'`, so framework expressions keep their double quotes and operators
(e.g. VueJS `:class="{ active: a && b }"`).

## Main Block

`@main` is the template's entry point. Its body is the content rendered when the
template runs.

```gadx
@main
    h1 Home
    p This template body is executed.
```

### `@main` is sugar for `@comp main`

`@main` is exactly a component named `main` that the module invokes for you.
Everything true of `@comp` is true of `@main`: it may declare a **signature**,
type parameters and a return type, and its body builds and returns an
[`Elements` fragment](./api.md#render-tree).

So `@main`'s parameters are the `main` component's **own parameters** — they are
**not** module globals or module parameters. Because the module calls `main()`
with no arguments, a parameter is only usable if it has a **default** (and a
defaulted parameter is *named*, so it needs the leading `;`):

```gadx
@main(; title = "Home", theme = "light")
    h1 {= title }
    body[class={theme}]
```

| You want… | Use | Not |
|-----------|-----|-----|
| a value with a default, local to `main` | `@main(; x = 1)` | — |
| a **module global** (injected by the host) | `@global x` | `@main(x)` |
| a **module parameter** (CLI/args) | `@param (x)` | `@main(x)` |

A `@global` (or `@param`) is visible inside `@main`'s body as a free variable:

```gadx
@global user
@main
    p Welcome, {= user.name }
```

### Return model

Every `@main` / `@comp` / `@slot` body builds a fresh `gadx.Elements()` fragment
and returns it; nested tags append into it, and the module returns `main()`.
Appending one fragment to another **splices** its children in (no extra
wrapper), so components compose cleanly — see
[the render tree](./api.md#render-tree).

## Code Block

A single `~` line is one Gad statement. It is **not** limited to one physical
line: it continues across lines until its brackets `()`/`[]`/`{}` balance, so a
call or func literal can span several lines:

```gadx
@main
    ~ items := ["a", "b", "c"]
    ~ t.run("group", func(t) {
        t.equal(1, 1)
    })
```

Use `~~ … ~~` for a block of several statements (or when you prefer an explicit
block):

```gadx
~~
const title = "Hello"
const items = ["a", "b"]
~~

@main
    h1 {= title}
```

A `+comp(…)` component call likewise reads across lines until its parentheses
close, so its arguments may be written one per line:

```gadx
+card(;
    title = "Hello",
    body = "…",
)
```

## Variables And Assignment

```gadx
@main
    @assign total = len(Items)
    p {= total + " items"}
```

Depending on parser form, assignment can also be represented by Gad code inside
`~~` blocks.

## Conditions

```gadx
@if User
    p Welcome {= User.Name}
@else
    p Welcome guest
```

## Loops

```gadx
ul
    @for item in Items
        li {= item.Title}
```

## Empty States

```gadx
@if Posts
    div.grid
        @for post in Posts
            article.card {= post.Title}
@else
    p No posts yet.
```

## Match

Match a value against `@case` clauses; the default clause is written `@else`.

```gadx
@match Status
    @case "draft"
        span.badge Draft
    @case "published"
        span.badge Published
    @else
        span.badge Unknown
```

## Imports

### Bare Import

```gadx
@import "components.gadx"
```

Imports the module for its side effects.

### Named Import

```gadx
@import "components.gadx" as comps
```

Makes the module available as the variable `comps`. Components or values from
the module are accessed via `+comps.name(...)`.

### Destructured Import

```gadx
@import { page_wrapper, hero } from "components.gadx"
```

Extracts specific named exports directly into scope. Components are then
available as `+page_wrapper(...)` and `+hero(...)` without a module prefix.

Supports Gad destructuring syntax including:

- Rename: `@import { page_wrapper: pw } from "components.gadx"`
- Default value: `@import { page_wrapper = fallback } from "components.gadx"`
- Rest pattern: `@import { page_wrapper, **rest } from "components.gadx"`
- Mixed: `@import { a, b: bb, c = nil, **rest } from "modules.gadx"`

All forms compile to Gad `import()` calls. Destructured imports generate a
curly-destructure assignment (`{...} := import("...")`), which is handled by
Gad's built-in destructuring compiler.

## Variable Declarations

Declare mutable variables with `@var`. A single name, a comma-separated list
(with optional initializers), or a parenthesized group that may span multiple
lines are all accepted. Indentation inside the parentheses is ignored.

```gadx
@var a                          // single
@var a, b                       // multiple, no initializers
@var a, b = {name: "test"}, x   // with an initializer

@var (
    width = 320
    height, depth = 0
)
```

Each form compiles to a Gad grouped declaration, e.g. `var (a, b={name: "test"}, x)`.

`@var` names are **untyped** (Gad's `var`/`const` have no type annotation — only
`@param`/`@global` do); `@var x int` is a parse error, not a typed variable.

## Constant Declarations

Declare immutable constants with `@const`. It accepts the same single,
comma-separated and multi-line parenthesized forms as `@var`, but **every
constant must have an initializer** (a value-less `@const x` is a compile error).

```gadx
@const pi = 3.14
@const a = 1, b = 2

@const (
    min = 0
    max = 100
)
```

Each form compiles to a Gad grouped declaration, e.g. `const (a=1, b=2)`.

## Enums

Declare an enum with `@enum IDENT ( … )`. The parenthesized body holds the
fields; its syntax mirrors a `@var` declaration — fields are separated by commas
or newlines, each an optional value (`Name` or `Name = value`) — and it also
accepts Gad's enum extras `bit` (power-of-two values) and a leading `+`/`-`
sign. Defaulted fields auto-increment from the previous one.

```gadx
@enum Perm (Read, Write, Exec = 10, Delete)   // 1, 2, 10, 11

@enum Color (
    Red
    Green
    Blue
)

@enum Flags (bit List, Detail, Create, Read = List | Detail)   // 1, 2, 4, 3
@enum Signed (-Low, Lower, +High, Higher)                      // -1, -2, 3, 4
```

Each form compiles to a Gad `enum IDENT { … }` statement, so a member exposes
`.value`, `.name`, `.index` and its owning enum, the enum is indexable by member
name (`Perm["Write"]`) and iterable in declaration order. See the Gad
[enum documentation](../../gad/doc/enums.md) for the value-computation rules.

## Globals

Declare globals with `@global`. Names may be space-separated (legacy) or
comma-separated, may carry a default, and may be grouped in parentheses spanning
multiple lines (indentation inside is ignored).

```gadx
@global Model User            // space-separated
@global t, Req, Context       // comma-separated
@global page = 1, limit = 20  // `= v` default: applied when nil OR absent
@global user !?= "guest"      // `!?= v` default: applied only when absent

@global (
    a
    b, c = 2
)
```

Each form compiles to a Gad grouped declaration, e.g. `global (t, Req, Context)`.
The `= v` / `!?= v` defaults lower onto Gad's [`global` defaults](../../gad/doc/variables-and-scopes.md#defaults):
`= v` fills a nil-or-absent global, `!?= v` fills only an absent one. Globals can
also be provided through the Go symbol table — the CMS example passes one global
named `Model`.

A global may also carry a **type** — like a `@param`, and unlike `@var`/`@const`,
which are untyped. Because the bare form keeps its legacy space-separated-names
meaning (`@global a b` declares two globals), a typed global uses the
**parenthesized** form:

```gadx
@global (x int)              // typed
@global (x int, y str)       // several typed globals
@global (id int|uint)        // a type union
@global (page int = 1)       // typed, `= v` nil-or-absent default
@global (user str !?= "guest") // typed, `!?= v` absent-only default
```

The type is a Gad type expression, so it may be a single type, a `|` union, a
named interface or an inline `interface { … }`.

## Params

Declare the parameters the compiled template receives with `@param`, the gadx
analog of Gad's top-level [`param`](../../gad/doc/variables-and-scopes.md#parameters)
declaration. It accepts the same forms as Gad's `param`: positional names, a
trailing variadic `*rest`, and — after a `;` — named parameters (which may carry
defaults) and a named-variadic `**named`.

```gadx
@param a                        // single positional
@param (a, b, *rest)            // positional + variadic
@param (a; b = 1, **named)      // positional; named (with default) + named-variadic

@param (
    title
    items, *rest
)
```

Each form compiles to a Gad grouped `param (…)` declaration at the template's top
scope. Positional parameters have no defaults (a default requires the named
section after `;`); a named parameter's default applies when its argument is
absent. Unlike `@global` (which reads ambient values), `@param` values are the
arguments supplied when the template is invoked. See `samples/gadx/param.gadx`.

Parameters may be **typed** — again like Gad's `param`, and unlike `@var`/`@const`:

```gadx
@param a int                    // typed positional
@param (a int, b str)           // several typed positionals
@param (id int|uint)            // a type union
@param (a int; b int = 2)       // positional typed; named typed with default
```

The type is a Gad type expression (a single type, a `|` union, a named interface
or an inline `interface { … }`), enforced when the template is invoked.

## Testing (`@test`)

`@test NAME` is the Gadx form of Gad’s `test NAME { … }`. `gad test` discovers
every `@test` in a `*_test.gadx` file and runs it with an injected `t` test
context (the same assertions as `_test.gad`: `t.equal`, `t.true`, `t.false`,
`t.nil`, `t.error`, `t.run`, …).

To assert a component’s HTML, render it with the `gadx.render(el)` builtin — it
renders a tag / component result to its string.

### Calling the test context

There are two ways to invoke `t` (or any callable):

- `! callee arg1 arg2 …` — the **fluent call statement** (idiomatic in tests).
  It lowers to `callee(arg1, arg2, …)`. Each space-separated part is one
  argument; whitespace inside `()`/`[]`/`{}`/quotes does not split a part, so an
  argument that itself contains spaces (operators) is **parenthesized**:
  `! t.true (a == b)`. It works with any callable — `! t.equal …`, `! myAssert …`.
- `~ callee(arg1, arg2, …)` — a plain `~` code line with the explicit call. Use
  it for anything the fluent form does not cover: locals, richer expressions, or
  a function-literal argument. A `~` line spans several lines until its brackets
  balance (so `~ t.run("x", func(t) { … })` works); use a `~~` block for several
  independent statements.

```gadx
@comp greeting(; name = "world")
    span Hello {= name }

// fluent `!` form — the idiomatic style
@test renders_with_name
    ! t.equal gadx.render(greeting(; name = "Gad")) "<span>Hello Gad</span>"

@test boolean_and_nil
    ! t.true (1 + 1 == 2)
    ! t.false (1 == 2)
    ! t.nil nil

// explicit `~` form — for locals or several steps
@test explicit_form
    ~ html := gadx.render(greeting())
    ~ t.equal(html, "<span>Hello world</span>")
    ~ t.true(len(html) > 0)
```

Run them with:

```sh
gad test greeting_test.gadx      # a file
gad test ./...                   # recurse, running *_test.gad and *_test.gadx
```

Each `@test` reports as `FILE/NAME`; a failed assertion records the failure and
aborts that test. See `samples/gadx/greeting_test.gadx`.
