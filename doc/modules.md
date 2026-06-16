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
> `strings.ToUpper("hi")`). `import(...)` still works for them. See
> [Builtins → Builtin Modules](builtins.md#builtin-modules).

## Importing

```go
strings := import("strings")    // a builtin module
println(strings.ToUpper("hi"))  // HI
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

## Shared State via globals

Modules can use `global` to reach the same host-provided globals object as the
main script — a simple way to share mutable state across modules. See
[Variables → global](variables-and-scopes.md#global). For goroutine-safe shared
maps, the host can provide a `syncDict`.

To configure which modules are importable from Go, see
[Embedding in Go](embedding.md).
