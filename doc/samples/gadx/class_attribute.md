# class_attribute

Class attribute forms.

The `class` attribute accepts several shapes, all merged into a single
space-separated class list (together with any static `.token` classes on the
tag):

- **string** — `class="a"` adds `a`.
- **array** — `class=["x", "y"]` adds each token; falsy entries
  (`nil`, `false`, `""`) are dropped, so `class=[cond ? "on" : nil, "base"]`
  keeps `base` (and `on` when `cond`).
- **dict (JSX/Vue object form)** — `class={active: cond, muted: !cond}`
  keeps each key whose value is truthy. Keys are emitted in sorted order so
  the output is deterministic.

Static `.token` classes and multiple `[class=…]` groups all accumulate:
`div.base[class=["x"]][class={on: true}]` yields `class="base x on"`.

## Components

### main

## Example — `class_attribute.gadx`

```gadx
~~
active := true
disabled := false
~~

@main
    // plain string
    div[class="card"] plain

    // static tokens merged with a class attribute
    div.card.wide[class="active"] merged

    // array with falsy filtering
    div[class=[active ? "on" : nil, "base"]] array

    // dict / JSX object form — truthy keys only, sorted
    button[class={primary: true, active: active, disabled: disabled}] dict

    // everything at once: static tokens + array + dict, all accumulated
    div.base[class=["x", "y"]][class={highlight: active}] combined
```
