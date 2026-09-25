
# Modules and Embedding

A module exposes values via `export` and is imported with `import(...)`, which
runs the module **once** and returns a dict of its exports (later imports reuse
it). This sample demonstrates `embed(...)`; a full multi-module example (import,
export, module parameters) is in `samples/modules/` — run
`gad run samples/modules/main.gad`.

## Importing and exporting

```gad ignore
strings := import("strings")   // a builtin module
m := import("./greet.gad")      // a source module, resolved relative to the file

// greet.gad:
export hello = "Hello"                     // export a new binding
export greet(name) => hello + ", " + name  // export a function
export {e: 2.71, phi: 1.61}                // export several keys at once
```

`export prop name = init` exports a **live binding** — a module-local `var`
behind a read/write property, so external writes change the module's variable
(see [Properties](properties.gad)).

## Module parameters

A module may declare `param` like the main script; parameters are supplied as
**named arguments** to `import` and interpreted **only on the first import**:

```gad ignore
// greet.gad:  param (; lang = "en"); const msgs = {en: "Hello", br: "Olá"}
g := import("./greet.gad"; lang = "br")
```

## Embedding files with `embed`

`embed("path")` pulls a file (or a whole directory) into the program **at
compile time**, evaluating to an `Embedded` value: `.name`, `.path`, `.size`,
`.data` (bytes), `.isDir`, and (for a directory) name-indexed entries plus an
iterable `.fs`. Paths resolve against the running script's directory;
`sources=[…]` lists directories to look the name up in.

```gad
greeting := embed("embed/greeting.txt")
[greeting.name, greeting.size, str(greeting.data)]
// => ["embed/greeting.txt", 29, "Hello from an embedded file!\n"]
```

Embedding a directory gives a node indexed by entry name; each entry is itself
an Embedded (a file or a nested directory). `sources=[...]` lists directories to
look the name up in, so the reference need not spell out the full path.

```gad
dir := embed("embed")
cfg := embed("config.json"; sources = ["embed"])   // found via the search path
[dir.name, dir.isDir, str(dir["config.json"].data), str(cfg.data) == str(dir["config.json"].data)]
// => ["embed", true, "{\"name\": \"gad\", \"stars\": 3}\n", true]
```

A directory's `.fs` is iterable: `for name, entry in dir.fs` yields each child
(`iterator(dir.fs; sorted)` orders them by name), and `.isDir` tells a
sub-directory from a file — recurse with it to walk the whole tree (`var f; f =
func …` binds the name for the recursive call).

```gad
root := embed("embed")
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
walk(root, "")
```

Output:

```text
config.json (28 bytes)
greeting.txt (29 bytes)
nested/
  note.txt (14 bytes)
```

## Example — `embed.gad`

```gad
greeting := embed("embed/greeting.txt")
[greeting.name, greeting.size, str(greeting.data)]

dir := embed("embed")
cfg := embed("config.json"; sources = ["embed"])   // found via the search path
[dir.name, dir.isDir, str(dir["config.json"].data), str(cfg.data) == str(dir["config.json"].data)]

root := embed("embed")
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
walk(root, "")

return greeting.name
```
