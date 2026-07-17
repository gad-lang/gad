# Conventions

## Go ↔ Gad value conversion

**Use `gad.MustNewReflectValue` instead of manually constructing `gad.Dict`.**

Pass Go structs directly through `gad.MustNewReflectValue` — the function uses reflection to convert exported struct fields into Gad values automatically. This eliminates the boilerplate of mapping each field by hand.

```go
// good — reflection-based
gad.MustNewReflectValue(page)

// bad — manual dict construction
gad.Dict{"ID": gad.Uint(p.ID), "Title": gad.Str(p.Title), …}
```

For computed fields (e.g., `URL` derived from `Slug`), pre-populate the struct before reflection rather than building a separate dict.

```go
p.URL = "/posts/" + p.Slug
gad.MustNewReflectValue(p)
```

Use `gorm:"-"` tags on model fields that should not be persisted but are needed during reflection (e.g., computed `URL` fields).

## Template output — raw HTML

**Use `{=raw expr}` for template expressions that contain HTML markup.**

The `raw` directive outputs the value without HTML escaping. Use it for `Body`, `RightBody`, or any field that contains pre-rendered HTML.

| Expression | Escaping | Use case |
|---|---|---|
| `{= expr}` | Escaped | Plain text, user-controlled strings |
| `{=raw expr}` | Unescaped | HTML content (Body, RightBody) |

```giom
section.page-body
    {=raw Model.Page.Body}
```

## Template imports

**Use named imports when accessing exports from another `.giom` file.**

Import exported values with `as` and reference them through the module alias. This avoids relying on side-effect imports and makes component ownership explicit.

```giom
@import "components.giom" as comps

@main
    +comps.page_wrapper("Home")
        +comps.hero("Title", "Summary")
```

Do not call exported symbols from another file as unqualified local names.

```giom
// bad
@import "components.giom"
+page_wrapper("Home")
```

## Component/function calls — omit empty parentheses

**For any component or function call without arguments, write `+name`, not
`+name()`.** Add parentheses only when passing arguments.

```giom
// good
+super
+the_tags
+default_layout

// bad
+super()
+the_tags()
+default_layout()
```

```giom
// with arguments — parentheses required
+super(item)
+card("Fancy")
+default_layout(; config=config)
```

## Slot `super`

**`super` is auto-injected as the first parameter of an overriding `@slot`.** Do
not declare it and do not bind it to a local — just call `super`.

```giom
// good — super is available automatically
+card("Fancy")
    @slot #header
        em ★
        +super
```

- A slot **with** default content passes that default as `super`; an
  **optional** slot (no default body) passes an empty function, so `+super` is
  always safe and renders nothing.
- For a **scoped** slot, forward the scope arguments when rendering the default:
  `+super(item)`.
- You may name the first parameter `super` explicitly (e.g. alongside scope
  parameters) — it is not injected twice: `@slot #item_image(super, item)`.
- When invoking a slot function directly at gad level (bypassing `@slot`), pass
  an empty super: `slots["main"](func(*_){})`.
