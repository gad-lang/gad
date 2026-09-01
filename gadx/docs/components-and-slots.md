# Components And Slots

Components are reusable template functions. They can receive positional
arguments, named arguments, and slot content.

## Define A Component

```gadx
@export comp button(label; href="#", kind="primary")
    a.btn[href=href][class="btn--" + kind]
        {= label}
```

Call it:

```gadx
@main
    +button("Read more" ; href="/docs", kind="secondary")
```

## Layout Component

```gadx
@export comp page(title)
    !!! 5
    html[lang="en"]
        head
            title {= title}
        body
            header.site-header
                a[href="/"] Home
            main
                @slot main
            footer Site footer
```

Use it:

```gadx
@main
    +page("About")
        h1 About
        p This content is passed into the main slot.
```

## Named Slots

Component:

```gadx
@export comp shell(title)
    section.shell
        aside
            @slot sidebar
                p Default sidebar
        main
            h1 {= title}
            @slot main
```

Caller:

```gadx
@main
    +shell("Dashboard")
        @slot #sidebar
            nav
                a[href="/reports"] Reports
                a[href="/settings"] Settings
        p Main dashboard content.
```

## Slot Defaults

```gadx
@export comp empty_state(title)
    section.empty
        h2 {= title}
        @slot main
            p Nothing to show yet.
```

If the caller does not pass content, the default slot body is rendered.

## Optional Slots

A slot with no default body is optional: it renders only when the caller
provides content, and nothing otherwise (it compiles to a nullish call
`slots.name?.(super)`). Even an optional slot's override receives a usable
`super` — an empty function — so calling `super` there is always safe and
simply renders nothing.

```gadx
@export comp panel
    section.panel
        @slot header      // optional — omitted when not provided
        @slot main
            p Body
```

## Rendering The Default With `super`

When a caller overrides a slot, its content can render the component's default
by calling `super`. **`super` is auto-injected as the override's first
parameter** — you do not declare or bind it. Just call `super(…)`:

```gadx
@export comp button(label)
    button.btn
        @slot main
            span {= label}

@main
    +button("Save")
        @slot #main
            em ★
            +super          // renders the default <span>Save</span>
```

You may name the first parameter `super` explicitly (for example when you also
declare scope parameters) — it will not be injected twice:

```gadx
        @slot #main(super)
            em ★
            +super
```

For a slot **with** default content, `super` renders that content; for an
**optional** slot (no default), `super` is an empty function, so `super`
renders nothing. Because a scoped slot's default expects its scope arguments,
forward them when rendering the default via `super`, e.g. `+super(item)`.

> **Convention:** call `super` (and any argument-less component) without empty
> parentheses — `+super`, not `+super()`. See
> [conventions.md](conventions.md#componentfunction-calls--omit-empty-parentheses).

## Slot Parameters

A slot may declare parameters (a *scoped slot*): the component supplies the
values when it renders the slot, and the override receives them.

```gadx
@export comp list(items)
    ul
        @for item in items
            li
                @slot item(item)
                    {= item}

@main
    +list(Posts)
        @slot #item(post)
            a[href=post.URL] {= post.Title}
```

An override of a scoped slot still receives `super` as its auto-injected first
parameter. To render the component's default for that item, forward the scope to
`super`:

```gadx
@main
    +list(Posts)
        @slot #item(super, post)
            span.star ★
            +super(post)      // renders the component's default for this item
```

A runnable version of scoped slots and `super` forwarding is in
`samples/gadx/slot_params.gadx`.

## Passing Slots Programmatically

`@slot #name` and `+super` are sugar. A component compiles to a gad function that
takes a `slots` dict, so you can build that dict yourself in a `~~ … ~~` code
block and call the component directly — useful when the set of slots is dynamic.

Each `slots` entry is a slot function whose **first parameter is `super`** (the
component's default for that slot), followed by the slot's scope parameters. A
slot function builds and **returns an `Elements` fragment** (like a component):
create it with `gadx.Elements()`, append content, and `return` it. Unlike `+super`,
a raw `super(…)` call is not rewritten, so it must pass super's own super (an
empty function) as its first argument; its returned fragment is appended with
`tag += super(…)`. The component call's own result is appended with `tag += …`.

```gadx
@export comp list(items;slots={})
    ul
        @for i, it in items
            li
                @slot row(i, it)
                    {=i}: {=it}

@main
    // render every row bold, ignoring the default
    ~~
    tag += list(["a", "b"]; slots={
        row: func(super, i, it) {
            tag := gadx.Elements()
            gadx.Text(tag, raw "<b>" + it + "</b>")
            return tag
        },
    })
    ~~

    // prefix each row, then render the default via super (scope forwarded)
    ~~
    tag += list(["a", "b"]; slots={
        row: func(super, i, it) {
            tag := gadx.Elements()
            gadx.Text(tag, raw "* ")
            tag += super(func(*_){}, i, it)   // +super(i, it) sugars to this
            return tag
        },
    })
    ~~
```

See `samples/gadx/slots_programmatic.gadx` for a runnable version.

## Dynamic Slot Names

A simple slot name is a bare identifier (`@slot header`, `@slot #header`). A
**dynamic** name is written as a **double-quoted interpolated string**, so a
`{expr}` interpolation is evaluated at render time and used as the `slots[…]`
key. This works for both the declaration and the pass (override):

```gadx
@slot "item[{i}]"        // declaration — one slot per value of i
@slot #"item[{index}]"   // pass — override the slot named item[<index>]
```

(The quoted form is always dynamic — even a constant `@slot "item"` is treated as
an interpolated string, whereas `@slot item` is a plain identifier.) Source
positions inside `{ … }` are preserved, so a runtime error in an interpolated
name reports the correct line.

A component can therefore give each item its own overridable slot:

```gadx
@comp list(items)
    @for i, it in items
        @slot "item[{i}]"(it)
            li {= it }

@main
    +list(Posts)
        @slot #"item[{1}]"(super, it)   // override just the second row
            li.featured {= it }
            +super(it)                  // then render its default
```

### Call-block code and slot names

The `~` / `~~ … ~~` code statements written directly in a component-call block
are **hoisted to the call scope, before the slot-pass declarations**. An
interpolated slot name (and any slot body) can therefore reference the values
they declare:

```gadx
+list(Posts)
    ~ const index = 3
    @slot #"item[{index}]"(super, it)
        li.featured {= it }
    ~ var mark = "★"
    @slot #"item[{4}]"(super, it)
        li {= mark }{= it }
```

A runnable version is in `samples/gadx/slot_dynamic_name.gadx`.

## Card Component

```gadx
@export comp card(title; href="")
    article.card
        h2
            @if href
                a[href=href] {= title}
            @else
                {= title}
        div.card-body
            @slot main
```

Usage:

```gadx
+card(Post.Title ; href=Post.URL)
    p {= Post.Summary}
```

## Component Libraries

Put reusable components in one file:

```text
templates/
├── components.gadx
├── forms.gadx
└── pages/home.gadx
```

Then import what your application resolver supports:

```gadx
@import "components.gadx"
@import "forms.gadx"

@main
    +page("Contact")
        +contact_form
```

## Typed Signatures

The `@comp`, `@func` and `@main` directives accept the full Gad function
signature grammar: parameter types, type parameters (`[T constraint]`) and
return types (`<ret>`). Each is lowered to the corresponding Gad function header,
so the same syntax Gad functions use works verbatim in a template.

### Parameter types

Annotate a parameter with a type; the type is enforced when the component or
function is called.

```gadx
@comp box(title str, count int)
    div.box
        h2 {= title}
        span {= count}

@func add(a int, b int) <int>
    | {= a + b}

@main
    +box("Inbox", 3)
    p Total: {= add(2, 3) }
```

Passing an argument of the wrong type raises a `TypeError` naming the parameter
and both types (`expected int, found str`). Named parameters (after `;`) may
carry both a type and a default:

```gadx
@comp badge(label str; kind str = "info")
    span[class="badge badge--" + kind] {= label}
```

### Type unions

Every typed position accepts a **type union** written with `|` — in a parameter
type, a type-parameter constraint and a return type alike. A union member may be
a concrete type, a named interface or an inline `interface { … }`.

```gadx
@func format(v int|str|float) <str>
    | {= str(v) }

@comp cell[T int|uint](v T)
    td {= v}

@func pick(v int|str) <int|str>
    | {= v}
```

### Type parameters

A `[T constraint, …]` list between the name and the parameters introduces type
parameters. References to a type parameter in a parameter or return type are
substituted by the constraint at compile time, and the constraint is enforced at
call time.

```gadx
@func cell[T any](v T) <T>
    | {= v}

@comp list[T stringer](items array)
    ul
        @for it in items
            li {= it}

@main
    p {= cell("x") }
    +list(["a", "b"])
```

`@func inc[T number](v T)` rejects a non-number argument with
`expected number, found str`.

### Return types

Declare a return type with `<ret>` after the parameter list. It documents the
directive and participates in Gad's type checking where a concrete value is
returned. The return type may itself be a `|` union (`<int|str>`).

```gadx
@func total(items array) <int>
    | {= len(items) }

@func idOrName(v int|str) <int|str>
    | {= v }
```

### The `@main` entry point

`@main` is **sugar for `@comp main`** — a component named `main` that the module
invokes for you. It takes the same typed signature, and its parameters are the
`main` component's **own parameters**, *not* template globals. Since the module
calls `main()` with no arguments, give a parameter a **default** (a defaulted
parameter is named, so it needs the leading `;`):

```gadx
@main(; user = "guest", count = 0)
    p Hello {= user}, you have {= count} messages
```

For values injected by the host, declare a `@global` (or a module `@param`) —
they are visible inside `@main`'s body as free variables:

```gadx
@global user
@param (count = 0)
@main
    p Hello {= user}, you have {= count} messages
```

See [Main Block](./syntax.md#main-block) for the full rules and return model.

Source positions of every signature part — parameter identifiers, parameter
types, type parameters and return types — are preserved back to the exact
offsets in the `.gadx` source, so type errors and editor navigation land on the
right token. See [Source Positions](./source-positions.md).

## Composition Guidelines

- Use components for repeated markup, not one-off tags.
- Use slots for page layout, cards, panels, and custom item rendering.
- Keep data shaping in Go when possible.
- Pass trusted rich HTML as `gad.RawStr`; pass regular strings as `gad.Str`.
