# Getting Started

This guide shows the shortest path from a Giom template to rendered HTML.

## Install

Use Giom as a Go module dependency:

```sh
go get github.com/gad-lang/giom
```

Giom builds on Gad, so applications usually import both packages:

```go
import (
    "github.com/gad-lang/gad"
    "github.com/gad-lang/giom"
)
```

## A Minimal Template

```giom
@main
    p Hello {= Name}
```

The template renders one paragraph. `{= Name}` writes the value of the `Name`
global.

Expected output:

```html
<p>Hello Giom</p>
```

## Render From Go

```go
src := []byte(`@main
    p Hello {= Name}
`)

builtins := giom.AppendBuiltins(gad.NewBuiltins())
st := gad.NewSymbolTable(builtins.NameSet)
_, _ = st.DefineGlobals([]string{"Name"})

_, bc, err := giom.Compile(st, src, gad.CompileOptions{})
if err != nil {
    return err
}

var out bytes.Buffer
vm := gad.NewVM(builtins.Build(), bc)
ret, err := vm.RunOpts(&gad.RunOpts{
    StdOut:  &out,
    Globals: gad.Dict{"Name": gad.Str("Giom")},
})
if err != nil {
    return err
}
// The template returns the root of a render tree; write it to produce HTML.
if el, ok := ret.(giom.Element); ok {
    _, err = el.WriteTo(vm, &out)
}
```

Use the same `builtins` instance for the symbol table and VM. This keeps Gad
builtin indexes consistent.

## File-Based Rendering Pattern

A common application layout is:

```text
templates/
├── components.giom
├── layout.giom
└── index.giom
```

`index.giom`:

```giom
@import "components.giom"

@main
    +page("Home")
        h1 {= Model.Title}
```

Your application can resolve `@import` lines before compilation, then call
`giom.Compile` with the combined source.

## First Concepts

- A line like `div.card` emits an HTML tag.
- Indentation defines tag bodies and control-flow bodies.
- `{= expr}` writes a Gad expression.
- `@main` marks the executable template body.
- `@export comp name(...)` defines a reusable component.
- `+name(...)` calls a component.
- `@slot main` declares where child content is rendered.

Continue with [Template Syntax](syntax.md).
