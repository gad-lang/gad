# Components And Slots

Components are reusable template functions. They can receive positional
arguments, named arguments, and slot content.

## Define A Component

```giom
@export comp button(label; href="#", kind="primary")
    a.btn[href=href][class="btn--" + kind]
        {= label}
```

Call it:

```giom
@main
    +button("Read more" ; href="/docs", kind="secondary")
```

## Layout Component

```giom
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

```giom
@main
    +page("About")
        h1 About
        p This content is passed into the main slot.
```

## Named Slots

Component:

```giom
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

```giom
@main
    +shell("Dashboard")
        @slot #sidebar
            nav
                a[href="/reports"] Reports
                a[href="/settings"] Settings
        p Main dashboard content.
```

## Slot Defaults

```giom
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

```giom
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

```giom
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

```giom
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

```giom
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

```giom
@main
    +list(Posts)
        @slot #item(super, post)
            span.star ★
            +super(post)      // renders the component's default for this item
```

A runnable version of scoped slots and `super` forwarding is in
`samples/slot_params.giom`.

## Passing Slots Programmatically

`@slot #name` and `+super` are sugar. A component compiles to a gad function that
takes a `slots` dict, so you can build that dict yourself in a `~~ … ~~` code
block and call the component directly — useful when the set of slots is dynamic.

Each `slots` entry is a slot function whose **first parameter is `super`** (the
component's default for that slot), followed by the slot's scope parameters. A
slot function builds and **returns a fragment tag** (like a component): create it
with `giom.Tag(nil)`, append content, and `return` it. Unlike `+super`,
a raw `super(…)` call is not rewritten, so it must pass super's own super (an
empty function) as its first argument; its returned fragment is appended with
`tag += super(…)`. The component call's own result is appended with `tag += …`.

```giom
@export comp list(items;slots={})
    ul
        @for i, it in items
            li
                @slot row(i, it)
                    {=i}: {=it}

@main
    //- render every row bold, ignoring the default
    ~~
    tag += list(["a", "b"]; slots={
        row: func(super, i, it) {
            tag := giom.Tag(nil)
            giom.Text(tag, raw "<b>" + it + "</b>")
            return tag
        },
    })
    ~~

    //- prefix each row, then render the default via super (scope forwarded)
    ~~
    tag += list(["a", "b"]; slots={
        row: func(super, i, it) {
            tag := giom.Tag(nil)
            giom.Text(tag, raw "* ")
            tag += super(func(*_){}, i, it)   // +super(i, it) sugars to this
            return tag
        },
    })
    ~~
```

See `samples/slots_programmatic.giom` for a runnable version.

## Dynamic Slot Names

A slot name may be written in parentheses as a Gad template string, so a `{expr}`
interpolation is evaluated at render time and used as the `slots[…]` key. This
works for both the declaration and the pass (override):

```giom
@slot (item[{i}])        // declaration — one slot per value of i
@slot #(item[{index}])   // pass — override the slot named item[<index>]
```

Source positions inside `{ … }` are preserved, so a runtime error in an
interpolated name reports the correct line.

A component can therefore give each item its own overridable slot:

```giom
@comp list(items)
    @for i, it in items
        @slot (item[{i}])(it)
            li {= it }

@main
    +list(Posts)
        @slot #(item[{1}])(super, it)   // override just the second row
            li.featured {= it }
            +super(it)                  // then render its default
```

### Call-block code and slot names

The `~` / `~~ … ~~` code statements written directly in a component-call block
are **hoisted to the call scope, before the slot-pass declarations**. An
interpolated slot name (and any slot body) can therefore reference the values
they declare:

```giom
+list(Posts)
    ~ const index = 3
    @slot #(item[{index}])(super, it)
        li.featured {= it }
    ~ var mark = "★"
    @slot #(item[{4}])(super, it)
        li {= mark }{= it }
```

A runnable version is in `samples/slot_dynamic_name.giom`.

## Card Component

```giom
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

```giom
+card(Post.Title ; href=Post.URL)
    p {= Post.Summary}
```

## Component Libraries

Put reusable components in one file:

```text
templates/
├── components.giom
├── forms.giom
└── pages/home.giom
```

Then import what your application resolver supports:

```giom
@import "components.giom"
@import "forms.giom"

@main
    +page("Contact")
        +contact_form
```

## Composition Guidelines

- Use components for repeated markup, not one-off tags.
- Use slots for page layout, cards, panels, and custom item rendering.
- Keep data shaping in Go when possible.
- Pass trusted rich HTML as `gad.RawStr`; pass regular strings as `gad.Str`.
