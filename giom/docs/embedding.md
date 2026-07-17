# Embedding In Go

Giom is embedded through Gad bytecode compilation and VM execution.

## Minimal Renderer

```go
package render

import (
    "bytes"

    "github.com/gad-lang/gad"
    "github.com/gad-lang/giom"
)

func Render(src []byte, globals gad.Dict) (string, error) {
    builtins := giom.AppendBuiltins(gad.NewBuiltins())
    st := gad.NewSymbolTable(builtins.NameSet)

    names := make([]string, 0, len(globals))
    for name := range globals {
        names = append(names, name)
    }
    if _, err := st.DefineGlobals(names); err != nil {
        return "", err
    }

    _, bc, err := giom.Compile(st, src, gad.CompileOptions{})
    if err != nil {
        return "", err
    }

    var out bytes.Buffer
    vm := gad.NewVM(builtins.Build(), bc)
    ret, err := vm.RunOpts(&gad.RunOpts{StdOut: &out, Globals: globals})
    if err != nil {
        return "", err
    }
    // A compiled template builds a render tree and returns its root element;
    // walk it to write the HTML output.
    if el, ok := ret.(giom.Element); ok {
        if _, err := el.WriteTo(vm, &out); err != nil {
            return "", err
        }
    }
    return out.String(), nil
}
```

The template does not stream HTML to `StdOut`; it returns the root of a render
tree (a `giom.Element`) that you write with `WriteTo`. The `Render` struct below
does this for you.

## Compiling Several Templates

`giom.Compile` is shorthand for `giom.NewCompiler(st, opts).Compile(input)`. Give
each template its own symbol table: a compiled template binds a root `tag` at the
module top level, so a symbol table is not shared across independent compiles.

```go
for _, src := range sources {
    st := gad.NewSymbolTable(builtins.NameSet)
    _, bc, err := giom.Compile(st, src, gad.CompileOptions{})
    if err != nil {
        return err
    }
    // run bc on a Gad VM and write the returned root element…
}
```

Each `Compile` call returns the parsed Giom AST and executable Gad bytecode. For
automatic bytecode caching and file-change detection, prefer the `Render` struct
below.

## Render Struct (Cached Compilation)

`giom.Render` provides a ready-to-use template engine with bytecode caching and
automatic recompilation on file changes. It is safe for concurrent use.

### Construction

Use the `NewRender` constructor to set the work directory (resolved to an
absolute path automatically):

```go
import "github.com/gad-lang/giom"

r := giom.NewRender("./templates")
r.TemplateDelay = 5 * time.Second
r.TranspilePath = func(src string) string {
    return strings.TrimSuffix(src, ".giom") + ".gad"
}
```

### Rendering

```go
var out bytes.Buffer
err := r.Render(&out, "template.giom", gad.Dict{
    "Model": gad.Dict{"Title": gad.Str("Home")},
})
```

The `TemplateDelay` (default 15s) prevents recompilation on rapid file saves.
`WorkDir` is the base for resolving `@import` lines via `FileImporter`.
`TranspilePath` is optional — when set, transpiled `.gad` files are written
for inspection.

### File Change Detection

`Render` tracks all files accessed during compilation (the template and its
imports). If any change is detected, compilation is deferred until
`TemplateDelay` elapses, then the bytecode is rebuilt.

If compilation fails, the old bytecode remains in the cache and continues to
be served. This ensures that broken edits never cause a blank page.

### Callbacks

Use `OnRender` to hook into the compilation lifecycle:

```go
r.OnRender(func(first bool, mainFile string, files []string, lastTime time.Time, err error) {
    if err != nil {
        log.Printf("compile error for %s: %v", mainFile, err)
        return
    }
    if first {
        log.Printf("first render: %s", mainFile)
    } else {
        log.Printf("recompile: %s (changed: %v)", mainFile, files)
    }
})
```

Multiple callbacks can be added:

```go
r.OnRender(loggingCallback).OnRender(metricsCallback)
```

## Builtins Rule

Use the same `*gad.Builtins` value for symbol-table creation and VM creation:

```go
builtins := giom.AppendBuiltins(gad.NewBuiltins())
st := gad.NewSymbolTable(builtins.NameSet)
vm := gad.NewVM(builtins.Build(), bc)
```

Do not call `giom.AppendBuiltins(gad.NewBuiltins())` separately for compile and
run steps.

## Globals

Global variables are defined from the keys of the `gad.Dict` passed to
`Render`. Both the compile-time symbol table and runtime globals use the
same names:

```go
globals := gad.Dict{
    "Model": gad.Dict{
        "Title": gad.Str("Home"),
    },
    "User": gad.Dict{
        "Name": gad.Str("Alice"),
    },
}

names := make([]string, 0, len(globals))
for name := range globals {
    names = append(names, name)
}
st.DefineGlobals(names)
```

The cache entry records the global names at compile time so that symbol
indices are consistent across cache hits.

Template:

```giom
@main
    h1 {= Model.Title}
```

## Trusted HTML

```go
model := gad.Dict{
    "Body": gad.RawStr("<p>Already sanitized HTML.</p>"),
}
```

Template:

```giom
article {= Model.Body}
```

Only use `gad.RawStr` for trusted content.

## Import Resolution

Giom uses `giom.FileImporter` to resolve `@import` lines during compilation.
The importer is set via `gad.ModuleMap.SetExtImporter`:

```go
imports := gad.NewModuleMap().SetExtImporter(&giom.FileImporter{
    WorkDir:       "./templates",
    FileReader:    os.ReadFile,
    TranspilePath: nil, // optional: write .gad files for inspection
})
```

`FileImporter` also handles named imports (`@import "file.giom" as name`)
and compiles imported Giom files to Gad bytecode transparently.

The CMS example in `examples/cms` demonstrates this in production.

## Transpiling For Inspection

You can parse and convert a Giom file to Gad statements:

```go
fs := source.NewFileSet()
f := fs.AddFileData("template.giom", -1, src)
p := parser.NewParser(f)
file, err := p.ParseFile()
if err != nil {
    return err
}

stmts := node.Convert(file.Stmts)
var buf bytes.Buffer
gnode.CodeW(&buf, stmts, gnode.CodeWithPrefix("\t"), gnode.CodeFormat())
```

This is useful for debugging, teaching, and generated-template review.
