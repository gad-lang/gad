# Modules

[← Back to index](README.md)

A module is a unit of Gad code that exposes values to other modules via the
`export` keyword. Import a module with the `import(...)` expression, which runs
the module once and returns a dict of its exports.

There are three kinds of modules:

* **Source modules** — Gad code (the focus of this chapter).
* **Builtin modules** — provided by the host as a `map[string]Object`
  (e.g. the [standard library](builtins.md)).
* **Custom modules** — any Go value implementing the `Importable` interface.

> A few stdlib modules — `time`, `strings`, `fmt` and `base64` — are also
> exposed as **builtin namespaces** that work without an `import` (e.g.
> `strings.toUpper("hi")`). `import(...)` still works for them. See
> [Builtins → Builtin Modules](builtins.md#builtin-modules).

## Importing

```go
strings := import("strings")    // a builtin module
println(strings.toUpper("hi"))  // HI
```

A source module is referenced by path; the file importer resolves `.gad`
files relative to the importing file (and along `GADPATH`):

```go
m := import("./greet.gad")
```

A module's code runs **once**; importing the same module again returns the same
exports object, so module state is preserved across imports.

## Exporting

Use `export` to expose values from a module. Several forms are supported:

```go
// greet.gad
hello := "Hello"

export hello                       // export an existing binding
export add(a, b) { return a + b }  // export a function
export greet(name) => hello + ", " + name   // arrow form
export pi = 3.14                   // export a new binding
export {e: 2.71, phi: 1.61}        // export several keys at once
```

Importing it:

```go
g := import("./greet.gad")
println(g.hello)         // Hello
println(g.add(2, 3))     // 5
println(g.greet("Gad"))  // Hello, Gad
println(g.pi, g.e)       // 3.14 2.71
```

## Module Parameters

A module may declare `param` just like the main script. Parameters are supplied
as **named arguments** to `import` and are interpreted **only on the first
import** — later imports reuse the already-loaded module and ignore any
arguments.

```go
// greet.gad
param (;lang="en")
const msgs = {en: "Hello", br: "Olá"}
hello := msgs[lang]
export hello
export greet(name) => hello + ", " + name
```

```go
g := import("./greet.gad"; lang="br")
println(g.hello)          // Olá
println(g.greet("Gad"))   // Olá, Gad
```

## Embedding files with `embed`

`embed("path")` pulls a file (or a whole directory) into the program **at compile
time**. It evaluates to an `Embedded` value:

```go
f := embed("data/greeting.txt")
f.name        // "data/greeting.txt" — the reference name
f.path        // the resolved path
f.size        // byte count
f.data        // the file contents, as bytes
str(f.data)   // "Hello!"
f.isDir       // false
```

Paths resolve against the running script's directory (the CLI wires this up; a Go
host configures it via an embed importer — see
[Embedding in Go](embedding.md)). `sources=[…]` lists directories to look the
name up in, so the reference need not spell out the full path:

```go
embed("greeting.txt"; sources=["data"])   // finds data/greeting.txt
```

### Directories: indexing, iteration and walking

Embedding a directory yields an `Embedded` whose entries are indexed by name, and
whose `.fs` is iterable (`for name, entry in dir.fs`, or `iterator(dir.fs;
sorted)` to order by name). Each entry is itself an `Embedded`; `.isDir` tells a
sub-directory from a file:

```go
dir := embed("data")
str(dir["greeting.txt"].data)             // index an entry by name

for name, e in iterator(dir.fs; sorted) {
    println(e.isDir ? "dir " : "file", name)
}
```

Recurse on `.isDir` to walk the whole tree (bind the function name first so it can
call itself):

```go
var walk
walk = func(node, indent) {
    for name, e in iterator(node.fs; sorted) {
        if e.isDir {
            println(indent + name + "/")
            walk(e, indent + "  ")
        } else {
            println(indent + name + " (" + str(e.size) + " bytes)")
        }
    }
}
walk(embed("data"), "")
```

See [`samples/26_embed.gad`](../samples/26_embed.gad). A full multi-module example (importing a Greeter module and a math module with parameters) is available in [`samples/modules/`](../samples/modules/) — run with `gad run samples/modules/main.gad`.

## Shared State via globals

Modules can use `global` to reach the same host-provided globals object as the
main script — a simple way to share mutable state across modules. See
[Variables → global](variables-and-scopes.md#global). For goroutine-safe shared
maps, the host can provide a `syncDict`.

To configure which modules are importable from Go, see
[Embedding in Go](embedding.md).
