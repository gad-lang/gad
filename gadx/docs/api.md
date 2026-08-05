# API Reference

This is a practical user-level API reference. For exact signatures, use `go doc`.

## Package

```go
import "github.com/gad-lang/gad/gadx"
```

Gadx ships inside the Gad module. Its runtime (the render tree and the `gadx`
builtins) lives in this package; Gadx **compilation** is built into Gad itself
(`gad.CompileOptions.GadxOptions`), so there is no separate Gadx compiler.

## `AppendBuiltins`

```go
func AppendBuiltins(b *gad.Builtins) *gad.Builtins
```

Registers the `gadx` module as a non-loadable builtin namespace. After this
call the following names are available globally in every compiled template
without any `@import`:

| Name | Description |
|------|-------------|
| `gadx.Tag` | Construct a tag element: `gadx.Tag([parent,] name, *children; **attrs)`. Omit the name (`gadx.Tag()` / `gadx.Tag(parent)`) for a nameless fragment. |
| `gadx.Text` | Construct a text node: `gadx.Text([parent,] v1, v2, …)` |
| `gadx.escape` | Return its argument as a raw (unescaped) string |
| `gadx.attr` | Render a single `name="value"` attribute fragment |
| `gadx.attrs` | Render multiple attributes from named arguments |
| `gadx.write` | Write a value with the tree's text semantics (raw for `RawStr`) |

Use it before compiling and before constructing the VM.

```go
builtins := gadx.AppendBuiltins(gad.NewBuiltins())
```

### Render tree

A compiled template does not stream HTML directly. Instead it builds a **render
tree** of `Element` values and returns its root; the caller (or `Render`) walks
the tree, writing HTML via `Element.WriteTo`. The tree types are:

- `gadx.Tag` — a tag element with a name, ordered attributes (regular
  attributes, a class list and styles) and child elements. Constructed without a
  name (`gadx.Tag()` / `gadx.Tag(parent)`) it is an *anonymous fragment* that
  renders only its children.
- `gadx.Text` — a leaf node holding a sequence of values written in order.

Each constructor optionally takes the **parent tag as its first argument** and
links the new element into it. Both forms are accepted:

```
gadx.Tag(parent, "div"; class="a")   // linked to parent
gadx.Tag("div"; class="a")           // standalone (append it yourself)
```

The first argument is treated as the parent only when it is a tag or `nil`;
otherwise it is the first content argument. The tag-building operators are
`tag += child` (append one), `tag ++= children` (append many),
`tag[name] = value` (set one attribute) and `tag.attrs += kva` (merge
attributes).

## Compilation (`gad.Compile` with `GadxOptions`)

Gadx source is compiled with Gad's own compiler by setting `GadxOptions` on the
compile options — the Gadx front-end parses the indentation-based syntax and
lowers it to Gad before compilation:

```go
opts := gad.CompileOptions{}
opts.GadxOptions = &gad.GadxOptions{}
_, bc, err := gad.Compile(st, src, opts)
```

`bc` is executable Gad bytecode. Give each independent template its own symbol
table: a compiled template binds a root `tag` at the module top level, so a
symbol table cannot be reused across separate compiles.

Register the Gadx builtins on the symbol table's builtins first
(`gadx.AppendBuiltins`), since the lowered code references `gadx.Tag`,
`gadx.write`, and friends.

Related Gad helpers:

- `gad.TranspileGadx(name, src, outPath)` — write the lowered Gad source to a
  file (see [Transpile](#transpile)).
- `gad.CompileGadxModule(ctx, src)` — compile Gadx as an imported module from a
  custom `gad.ExtImporter`.
- `importers.FileImporter` — resolves `@import` of `.gadx` files natively (see
  [FileImporter](#fileimporter)).

## `Render` Struct

`gadx.Render` is a ready-to-use template engine with bytecode caching and
automatic recompilation on file changes. Safe for concurrent use.

### `NewRender`

```go
func NewRender(workDir string) *Render
```

Creates a `Render` with the given work directory. Non-empty paths are
resolved to an absolute path. Default `TemplateDelay` is 15 seconds.

```go
r := gadx.NewRender("./templates")
r.TemplateDelay = 5 * time.Second
```

### `WorkDir`

```go
func (r *Render) WorkDir() string
```

Returns the work directory used for import resolution.

### Exported Fields

```go
type Render struct {
    TemplateDelay time.Duration        // debounce before recompiling (default 15s)
    TranspilePath func(srcPath string) string  // optional .gad output path
    BuiltinsFunc  func() *gad.Builtins        // optional builtins factory
}
```

- `TemplateDelay` — debounce duration before recompiling after a file change.
  Defaults to 15s when zero. Set before the first call to `Render`.
- `TranspilePath` — if set, transpiled `.gad` files are written after each
  successful compile. Receives the source `.gadx` path, returns output path.
- `BuiltinsFunc` — factory for Gad builtins. Called once (and cached) on the
  first compile. If nil, defaults to `gad.NewBuiltins()` with Gadx builtins.

### `(*Render) Render`

```go
func (r *Render) Render(out io.Writer, filePath string, globals gad.Dict) error
```

Reads `filePath`, compiles (or retrieves cached bytecode), and executes the
template with the keys of `globals` available as global variables. The output is
written to `out`.

```go
err := r.Render(&out, "post.gadx", gad.Dict{
    "Model": gad.Dict{"Title": gad.Str("Hello")},
})
```

Caching tracks all files accessed during compilation (template + imports).
When a file change is detected, recompilation is deferred by `TemplateDelay`.

#### Interface-satisfaction cache

Each compiled template carries a `gad.InterfaceSatCache` that `Render` reuses
across renders and injects into the VM. So an `obj :: SomeInterface` check (or an
interface used as a parameter type) inside a template is validated **once per
value type** — a big win when rendering the same interface check in a loop or
across many requests — instead of re-validating every render. The cache is
**reset automatically when the template recompiles** (a source change): the fresh
bytecode gets a fresh cache, so a changed interface never returns a stale result.

Only values whose type fully determines their members are cached (Gad class
instances and reflected Go values); dicts, whose keys vary per value, are always
re-checked. To share or pre-warm a cache across engines yourself, build one with
`gad.NewInterfaceSatCache()` and inject it into a VM via
`(*gad.VM).SetInterfaceSatCache` — the same cache the VM uses on its own (it
otherwise lives on the root VM and is dropped with it).

### `OnRender`

```go
func (r *Render) OnRender(f ...func(first bool, mainFile string, files []string, lastTime time.Time, err error)) *Render
```

Appends callback functions invoked after compilation. Returns the `Render` for
chaining. Multiple callbacks may be added.

Parameters:
- `first` — true on initial compile, false on recompile after file changes.
- `mainFile` — path relative to `WorkDir` of the rendered template.
- `files` — changed file paths (relative to `WorkDir`) that triggered
  recompilation. Empty on first render.
- `lastTime` — timestamp of the previous successful render. Zero on first
  render, non-zero on recompile.
- `err` — non-nil if compilation failed. The cached entry is **not** updated
  on failure, so the previous bytecode continues to be served.

```go
r.OnRender(func(first bool, mainFile string, files []string, lastTime time.Time, err error) {
    if err != nil {
        log.Printf("compile error for %s: %v", mainFile, err)
        return
    }
    if first {
        log.Printf("first render: %s", mainFile)
    } else {
        log.Printf("recompile: %s (changed: %v, last render: %s)",
            mainFile, files, lastTime.Format(time.Stamp))
    }
})
```

### Caching Behavior

- The first call to `Render` for a given file compiles it and caches the
  bytecode along with file modification times for the template and all its
  imports.
- Subsequent calls check all tracked files. If any have changed, the change
  is noted and recompilation is deferred until `TemplateDelay` elapses since
  the first detected change.
- This debounce prevents recompilation during rapid file-save sequences.
- If recompilation fails, the old bytecode remains in the cache and continues
  to be served. Callbacks still fire with the error.

## Transpile

```go
func gad.TranspileGadx(name string, src []byte, outPath string) error
```

Parses Gadx source, converts it to Gad statements, and writes the result to
`outPath` (a `.gad` suffix is appended when missing). Useful for inspection and
debugging.

```go
gad.TranspileGadx("template.gadx", src, "template.gad")
```

## `FileImporter`

```go
import "github.com/gad-lang/gad/importers"

type importers.FileImporter struct {
    WorkDir       string
    FileReader    func(path string) ([]byte, string, error)
    NameResolver  func(cwd, name string) (string, error)
    TranspilePath func(srcPath string) string
}
```

Gad's file importer implements `gad.ExtImporter`. It resolves `@import` lines,
reading imported files via `FileReader`; a `.gadx` module is compiled natively
with the Gadx front-end (and, when `TranspilePath` is set, its transpiled `.gad`
output is written).

Used automatically by `Render` when `WorkDir` is set. Can also be wired
manually:

```go
mm := gad.NewModuleMap().SetExtImporter(&importers.FileImporter{
    WorkDir:    "./templates",
    FileReader: importers.ShebangReadFile,
})
```

## Parser Package

```go
import "github.com/gad-lang/gad/gadx/parser"
```

```go
fs := source.NewFileSet()
f := fs.AddFileData("template.gadx", -1, src)
p := parser.NewParser(f)
file, err := p.ParseFile()
```

## Node Package

```go
import gadxnode "github.com/gad-lang/gad/gadx/node"
```

Convert Gadx nodes to Gad nodes:

```go
stmts := gadxnode.Convert(file.Stmts)
```

## Token Package

```go
import "github.com/gad-lang/gad/gadx/token"
```

Contains Gadx token definitions used by the parser and scanner.
