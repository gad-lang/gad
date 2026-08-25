# param

@param declares the parameters the compiled template receives — the gadx
analog of Gad's top-level `param`. It accepts the same forms as Gad's param:
positional names, a trailing variadic `*rest`, and (after `;`) named
parameters with optional defaults plus a named-variadic `**named`.

Positional parameters have no defaults; a default requires the named section
after `;` and applies when the argument is absent. Unlike @global (ambient
values), @param values are the arguments passed when the template is invoked.

## Components

### main

## Parameters

### @param

```gadx
@param(title; subtitle="welcome", theme="light")
```

## Example — `param.gadx`

```gadx
@param (title; subtitle = "welcome", theme = "light")

@main
    <section class={theme}>
        <h1>{title}</h1>
        <p class="sub">{subtitle}</p>
    </section>
```
