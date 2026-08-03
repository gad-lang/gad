# Template Syntax

Giom uses indentation to describe HTML, components, and Gad control flow.

## Comments

Line comments start a line:

```giom
// rendered as an HTML comment: <!-- … -->
//- silent: not emitted
```

Block comments `/* … */` are silent and may span multiple lines. They are only
recognized at the start of a line (a `/*` mid-line stays literal text):

```giom
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

```giom
/** Renders a greeting for the given name. **/
@comp greeting(name)
    p {= "Hello, " + name }
```

## Document Type

```giom
!!! 5
```

Output:

```html
<!DOCTYPE html>
```

## Tags

```giom
section.hero
    h1 Welcome
    p Ship templates with less noise.
```

Output:

```html
<section class="hero"><h1>Welcome</h1><p>Ship templates with less noise.</p></section>
```

## Ids And Classes

```giom
main#content.page.shell
    h1.title Hello
```

Output:

```html
<main id="content" class="page shell"><h1 class="title">Hello</h1></main>
```

## Attributes

```giom
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

```giom
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

```giom
div[title=join(items, ", "), data-ids=[1, 2, 3]]
```

A trailing `? condition` applies to every attribute in the group.

## Inline HTML

A line starting with `<` is parsed as an inline HTML region. It runs from the
opening tag to its matching close tag (spanning multiple lines if needed), and
runs of whitespace collapse to a single space. `<> … </>` is a fragment: it
renders its children with no wrapping element.

```giom
@main
    <a href="/" class="link">Home</a>

    <ul>
        <li>one</li>
        <li>two</li>
    </ul>
```

The region is **compiled to the same `giom.Tag` / `giom.Text` elements as the
pug-style tag syntax** — it is not emitted as a raw string. So it shares the
tag rendering rules: void elements self-close (`<br>` → `<br />`), boolean
attributes expand (`<input disabled>` → `<input disabled="disabled" />`), and
attributes are classified (regular attributes first, then the class list).
Because the AST carries real giom tag nodes, an inline HTML region also
transpiles back to pug-style giom (`gofmt`-style, via `WriteGiom`):
`<a href="/x">Home</a>` becomes `a[href="/x"]` with an indented `| Home`.

Attributes may be interpolated with `{expr}` — both the value and the name. An
interpolated value is auto-quoted and HTML-escaped (a falsy value drops the
attribute); an interpolated name builds the attribute name from the expression
(lowered to a computed `**{[name]: value}` spread). `{expr}` also interpolates
text content. Interpolation source positions are preserved.

```giom
@main
    <a href={post.URL} data-{key}={value}>
        {post.Title}
    </a>
```

### Interleaving giom statements

Block-level giom statements (`@if`, `@for`, `@match`, `+comp`, `~code`) may be
interleaved inside an HTML region by indentation. The directive line and every
line indented deeper than it form the block — which may itself contain HTML —
and it renders as a child of the enclosing element, in source order alongside
sibling HTML. `@else` / `@else if` clauses continue the block at the same
indentation.

```giom
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

The interleaved block is parsed by the full giom parser, so any giom construct
is available; source lines resolve back to the original `.giom` file (line
accurate). Runnable examples are in `samples/giom/html.giom` and
`samples/giom/html_control_flow.giom`. Use the pug-style `tag[attr=…]` syntax
(below) when you prefer giom's indentation-based nesting throughout.

## Text

Inline text:

```giom
p Hello world
```

Text block:

```giom
p
    | This is plain text.
    | It can span multiple lines.
```

## Expressions

```giom
h1 {= Model.Title}
p {= "Hello " + User.Name}
```

Use Gad expressions inside `{= ...}`.

## Raw HTML Values

If the application passes a `gad.RawStr`, Giom writes it without escaping.

```giom
article {= Post.Body}
```

Use raw values only for trusted HTML.

## Main Block

```giom
@main
    h1 Home
    p This template body is executed.
```

## Code Block

Use `~~` for Gad source sections.

```giom
~~
const title = "Hello"
~~

@main
    h1 {= title}
```

## Variables And Assignment

```giom
@main
    @assign total = len(Items)
    p {= total + " items"}
```

Depending on parser form, assignment can also be represented by Gad code inside
`~~` blocks.

## Conditions

```giom
@if User
    p Welcome {= User.Name}
@else
    p Welcome guest
```

## Loops

```giom
ul
    @for item in Items
        li {= item.Title}
```

## Empty States

```giom
@if Posts
    div.grid
        @for post in Posts
            article.card {= post.Title}
@else
    p No posts yet.
```

## Match

Match a value against `@case` clauses; the default clause is written `@else`.

```giom
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

```giom
@import "components.giom"
```

Imports the module for its side effects.

### Named Import

```giom
@import "components.giom" as comps
```

Makes the module available as the variable `comps`. Components or values from
the module are accessed via `+comps.name(...)`.

### Destructured Import

```giom
@import { page_wrapper, hero } from "components.giom"
```

Extracts specific named exports directly into scope. Components are then
available as `+page_wrapper(...)` and `+hero(...)` without a module prefix.

Supports Gad destructuring syntax including:

- Rename: `@import { page_wrapper: pw } from "components.giom"`
- Default value: `@import { page_wrapper = fallback } from "components.giom"`
- Rest pattern: `@import { page_wrapper, **rest } from "components.giom"`
- Mixed: `@import { a, b: bb, c = nil, **rest } from "modules.giom"`

All forms compile to Gad `import()` calls. Destructured imports generate a
curly-destructure assignment (`{...} := import("...")`), which is handled by
Gad's built-in destructuring compiler.

## Variable Declarations

Declare mutable variables with `@var`. A single name, a comma-separated list
(with optional initializers), or a parenthesized group that may span multiple
lines are all accepted. Indentation inside the parentheses is ignored.

```giom
@var a                          // single
@var a, b                       // multiple, no initializers
@var a, b = {name: "test"}, x   // with an initializer

@var (
    width = 320
    height, depth = 0
)
```

Each form compiles to a Gad grouped declaration, e.g. `var (a, b={name: "test"}, x)`.

## Constant Declarations

Declare immutable constants with `@const`. It accepts the same single,
comma-separated and multi-line parenthesized forms as `@var`, but **every
constant must have an initializer** (a value-less `@const x` is a compile error).

```giom
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

```giom
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

```giom
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

## Params

Declare the parameters the compiled template receives with `@param`, the giom
analog of Gad's top-level [`param`](../../gad/doc/variables-and-scopes.md#parameters)
declaration. It accepts the same forms as Gad's `param`: positional names, a
trailing variadic `*rest`, and — after a `;` — named parameters (which may carry
defaults) and a named-variadic `**named`.

```giom
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
arguments supplied when the template is invoked. See `samples/giom/param.giom`.
