# API Reference

This is a practical user-level API reference. For exact signatures, use `go doc`.

## Package

```go
import "github.com/gad-lang/giom"
```

## `AppendBuiltins`

```go
func AppendBuiltins(b *gad.Builtins) *gad.Builtins
```

Registers the `giom` module as a non-loadable builtin namespace. After this
call the following names are available globally in every compiled template
without any `@import`:

| Name | Description |
|------|-------------|
| `giom.Tag` | Construct a tag element: `giom.Tag([parent,] name, *children; **attrs)`. Omit the name (`giom.Tag()` / `giom.Tag(parent)`) for a nameless fragment. |
| `giom.Text` | Construct a text node: `giom.Text([parent,] v1, v2, …)` |
| `giom.escape` | Return its argument as a raw (unescaped) string |
| `giom.attr` | Render a single `name="value"` attribute fragment |
| `giom.attrs` | Render multiple attributes from named arguments |
| `giom.write` | Write a value with the tree's text semantics (raw for `RawStr`) |

Use it before compiling and before constructing the VM.

```go
builtins := giom.AppendBuiltins(gad.NewBuiltins())
```

### Render tree

A compiled template does not stream HTML directly. Instead it builds a **render
tree** of `Element` values and returns its root; the caller (or `Render`) walks
the tree, writing HTML via `Element.WriteTo`. The tree types are:

- `giom.Tag` — a tag element with a name, ordered attributes (regular
  attributes, a class list and styles) and child elements. Constructed without a
  name (`giom.Tag()` / `giom.Tag(parent)`) it is an *anonymous fragment* that
  renders only its children.
- `giom.Text` — a leaf node holding a sequence of values written in order.

Each constructor optionally takes the **parent tag as its first argument** and
links the new element into it. Both forms are accepted:

```
giom.Tag(parent, "div"; class="a")   // linked to parent
giom.Tag("div"; class="a")           // standalone (append it yourself)
```

The first argument is treated as the parent only when it is a tag or `nil`;
otherwise it is the first content argument. The tag-building operators are
`tag += child` (append one), `tag ++= children` (append many),
`tag[name] = value` (set one attribute) and `tag.attrs += kva` (merge
attributes).

## `Compile`

```go
func Compile(st *gad.SymbolTable, input []byte, opts gad.CompileOptions) (*node.File, *gad.Bytecode, error)
```

Parses Giom source and compiles it to Gad bytecode. It is shorthand for
`NewCompiler(st, opts).Compile(input)`.

```go
file, bc, err := giom.Compile(st, src, gad.CompileOptions{})
```

`file` is the parsed Giom AST. `bc` is executable Gad bytecode.

## `Compiler`

```go
type Compiler struct { /* unexported */ }

func NewCompiler(st *gad.SymbolTable, opts gad.CompileOptions) *Compiler
func (c *Compiler) Compile(input []byte) (*node.File, *gad.Bytecode, error)
```

A `Compiler` binds a symbol table and compile options. Construct one with
`NewCompiler` and call `Compile` for each source; each call returns the parsed
Giom AST and executable Gad bytecode, exactly like the package-level
[`Compile`](#compile).

```go
_, bcHome, err := giom.NewCompiler(stHome, gad.CompileOptions{}).
    Compile([]byte("@main\n    p Home\n"))
// ...
_, bcAbout, err := giom.NewCompiler(stAbout, gad.CompileOptions{}).
    Compile([]byte("@main\n    p About\n"))
```

Give each independent template its own symbol table: a compiled template binds a
root `tag` at the module top level, so a symbol table cannot be reused across
separate compiles. A nil symbol table is created on demand at compile time. The
package-level `Compile` is simply `NewCompiler(st, opts).Compile(input)`.

## `CompileFile`

```go
func CompileFile(st *gad.SymbolTable, module *gad.ModuleSpec, file *node.File, opts gad.CompileOptions) (*gad.Bytecode, error)
```

Compiles an already parsed Giom file. Use this when you need parser access before
compilation.

## `Render` Struct

`giom.Render` is a ready-to-use template engine with bytecode caching and
automatic recompilation on file changes. Safe for concurrent use.

### `NewRender`

```go
func NewRender(workDir string) *Render
```

Creates a `Render` with the given work directory. Non-empty paths are
resolved to an absolute path. Default `TemplateDelay` is 15 seconds.

```go
r := giom.NewRender("./templates")
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
  successful compile. Receives the source `.giom` path, returns output path.
- `BuiltinsFunc` — factory for Gad builtins. Called once (and cached) on the
  first compile. If nil, defaults to `gad.NewBuiltins()` with Giom builtins.

### `(*Render) Render`

```go
func (r *Render) Render(out io.Writer, filePath string, globals gad.Dict) error
```

Reads `filePath`, compiles (or retrieves cached bytecode), and executes the
template with the keys of `globals` available as global variables. The output is
written to `out`.

```go
err := r.Render(&out, "post.giom", gad.Dict{
    "Model": gad.Dict{"Title": gad.Str("Hello")},
})
```

Caching tracks all files accessed during compilation (template + imports).
When a file change is detected, recompilation is deferred by `TemplateDelay`.

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

## `Transpile`

```go
func Transpile(name string, src []byte, outPath string) error
```

Parses Giom source, converts it to Gad statements, and writes the result
to `outPath`. Useful for inspection and debugging.

```go
giom.Transpile("template.giom", src, "template.gad")
```

## `FileImporter`

```go
type FileImporter struct {
    WorkDir       string
    FileReader    func(path string) ([]byte, string, error)
    TranspilePath func(srcPath string) string
}
```

Implements `gad.ExtImporter` for resolving `@import` lines in Giom
templates. It reads imported files via `FileReader`, compiles them to
Gad bytecode, and optionally writes transpiled `.gad` output.

Used automatically by `Render` when `WorkDir` is set. Can also be wired
manually:

```go
mm := gad.NewModuleMap().SetExtImporter(&giom.FileImporter{
    WorkDir:    "./templates",
    FileReader: os.ReadFile,
})
```

## Parser Package

```go
import "github.com/gad-lang/giom/parser"
```

```go
fs := source.NewFileSet()
f := fs.AddFileData("template.giom", -1, src)
p := parser.NewParser(f)
file, err := p.ParseFile()
```

## Node Package

```go
import giomnode "github.com/gad-lang/giom/node"
```

Convert Giom nodes to Gad nodes:

```go
stmts := giomnode.Convert(file.Stmts)
```

## Token Package

```go
import "github.com/gad-lang/giom/token"
```

Contains Giom token definitions used by the parser and scanner.
