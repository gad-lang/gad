# attribute_name

Attribute names, bare and quoted.

A name is written as it stands while it holds only what a name is made of —
letters, digits, `_`, `-`, `:`, `@` and `.`. That covers HTML and the framework
punctuation that goes with it: `data-id`, `xlink:href`, `@click`, `v-model`,
`:disabled`, and an event with a modifier such as `@submit.prevent`.

Anything else is written in quotes. Without them the name would end at the
first character the bare rules stop at, and the attribute would not be the one
that was meant — a `/` or a `%` in the name is the usual reason.

The quotes are punctuation, not part of the name: `"v-model"` and `v-model` are
the same attribute. The formatter writes back the form the name needs, so a
quoted name that reads whole on its own comes back bare.

A name may also be computed, with `data-{key}=v` or a `**dict` spread; see
[html](html.gadx).

## Components

### main

## Parameters

### @param

```gadx
@param(; label="Name")
```

## Example — `attribute_name.gadx`

```gadx
@param (; label="Name")

@main
    // Bare: the punctuation a framework attribute usually carries.
    input[type="text", v-model="form.name", @input.trim="touch"]
    // Quoted: names the bare rules would cut short.
    div["x/y"="1", "a%b"] both
    label[for="name"] {= label }
```
