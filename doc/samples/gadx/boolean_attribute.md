# boolean_attribute

Boolean and flag attributes.

Attribute *presence* is controlled by the flag type (`yes` / `no`), which is
separate from a boolean *value* (`true` / `false`):

- **valueless `[x]`** ≡ **`[x=yes]`** — a bare boolean attribute (like a named
  param `fn(;x)`): rendered as `x` with no value.
- **`[x=no]`** — the attribute is omitted.
- **`[x=true]` / `[x=false]`** — a `bool` renders its literal value:
  `x="true"` / `x="false"`.
- other falsy values (`nil`, `""`) omit the attribute; any other value renders
  as `x="value"`.

## Components

### main

## Example — `boolean_attribute.gadx`

```gadx
@main
    // valueless flag ≡ =yes → bare attribute, no value
    input[type="checkbox", checked] valueless

    // explicit yes / no
    input[type="checkbox", checked=yes] flag-yes
    input[type="checkbox", checked=no] flag-no

    // bool value renders literally (both true and false)
    div[aria-expanded=true] bool-true
    div[aria-expanded=false] bool-false

    // other values: string/number render as-is, nil/"" omit
    div[data-count=42] number
    div[data-x=nil] omitted-nil
```
