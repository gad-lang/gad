# Getting Started

This guide shows the shortest path from a Giom template to rendered HTML.

## Install

Giom ships inside the Gad module, so add Gad as a dependency:

```sh
go get github.com/gad-lang/gad
```

Applications usually import both the `gad` package and the `giom` sub-package:

```go
import (
    "github.com/gad-lang/gad"
    "github.com/gad-lang/gad/giom"
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

// GiomOptions selects Gad's native Giom front-end.
opts := gad.CompileOptions{}
opts.GiomOptions = &gad.GiomOptions{}
_, bc, err := gad.Compile(st, src, opts)
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

`@import` lines are resolved automatically during compilation by the file
importer (`github.com/gad-lang/gad/importers.FileImporter`), which compiles
imported `.giom` modules natively — see [Embedding in Go](embedding.md).

## First Concepts

- A line like `div.card` emits an HTML tag.
- Indentation defines tag bodies and control-flow bodies.
- `{= expr}` writes a Gad expression.
- `@main` marks the executable template body.
- `@export comp name(...)` defines a reusable component.
- `+name(...)` calls a component.
- `@slot main` declares where child content is rendered.

Continue with [Template Syntax](syntax.md).
