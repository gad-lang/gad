# main

main.gad — imports the sibling source modules in this directory.

Run from the repo root:  gad run samples/modules/main.gad
See doc/modules.md for detailed documentation.

## Importing a module

`import("./mathx.gad")` runs the module (resolved relative to this file) and
returns its exports: plain bindings, functions and the entries of an exported
dict.

```gad
mathx := import("./mathx.gad")
[mathx.pi, mathx.square(7), mathx.cube(3), mathx.circleArea(2), mathx.rectArea(3, 4)]
// => [3.141592653589793, 49, 27, 12.566370614359172, 12]
```

## Module parameters

Named import args set the module's `param (…)`. A module runs once and is then
cached, so the params apply on its first import.

```gad
pt := import("./exports.gad"; lang="pt")
[pt.language, pt.greet("Gad")]
// => ["pt", "Olá, Gad!"]
```

## Exported properties

Member access on a module that exports a property delegates to its
getter/setter (`count` clamps at 0). A live binding (`export prop total = 0`)
shares one module variable between external and internal writes.
`reflect.get` reads the exported `Prop` object itself (the getter is not run).

```gad
counter := import("./exports.gad")
seen := [counter.count]           // 0 (getter)
counter.count = 5                 // setter
counter.inc()                     // bumps count and total
seen += counter.count             // 6
counter.count = -3                // the setter clamps at 0
seen += counter.count             // 0

counter.total = 10                // external write to the live binding
counter.inc()                     // internal write (+1)
[seen, counter.total, typeName(reflect.get(counter, "count"))]
// => [[0, 6, 0], 11, "Prop"]
```

## Importing many modules with a glob

A module name holding a glob meta character (`*`, `?`, `[…]`) imports **every
matching module** and yields an **array** of them, sorted by path. `*` matches
within one directory level and a `**` segment any number of them
(`./plugins/**/*.gad`). The pattern resolves like any import name (a `./`
pattern against this file's directory), a pattern matching nothing yields `[]`,
and the importing file itself is never matched. Other named args are the
module params, passed to each module.

```gad
// en.gad and pt.gad, sorted; `punct` is their module param (the first import of
// a module applies its params, like any import)
plugins := import("./plugins/*.gad"; punct=".")
[
    [p.lang for p in plugins],
    [p.greet("Gad") for p in plugins],
    import("./no_such_dir/*.gad"),                    // no match: []
]
// => [["en", "pt"], ["Hello, Gad.", "Olá, Gad."], []]
```

The matches can be narrowed by the same filters `embed` takes, prefixed with `@`
because an import's named args are otherwise module params:

| Named arg      | Keeps / drops the files whose…                        |
|----------------|-------------------------------------------------------|
| `@includes`    | base name or relative path matches one of the globs   |
| `@excludes`    | … drops those matching one of the globs               |
| `@includes_re` | relative path matches one of the regular expressions  |
| `@excludes_re` | … drops those matching one of the regular expressions |

Each takes a string or an array of strings (literals — they are applied at
compile time); the relative path is the one below the pattern's static
directory (`./plugins/`).

**Test files are skipped by default**: a `*_test` file (`plugins_test.gad`) only
matches when an include that names `_test` itself selects it —
`@includes=["*_test.gad"]` or `@includes_re=["_test"]`. A generic include such
as `*.gad` does not bring it back.

```gad
langs := func(mods) => [m.lang for m in mods]
[
    langs(import("./plugins/*.gad"; @excludes=["pt.gad"])),
    langs(import("./plugins/*.gad"; @includes_re=["^p"])),     // pt.gad (not the test file)
    langs(import("./plugins/*.gad"; @includes=["*_test.gad"])), // only the test file
    langs(import("./plugins/*.gad"; @includes=["*.gad", "*_test.gad"])),
]
// => [["en"], ["pt"], ["test"], ["en", "test", "pt"]]
```

## Example — `main.gad`

```gad
mathx := import("./mathx.gad")
[mathx.pi, mathx.square(7), mathx.cube(3), mathx.circleArea(2), mathx.rectArea(3, 4)]

pt := import("./exports.gad"; lang="pt")
[pt.language, pt.greet("Gad")]

counter := import("./exports.gad")
seen := [counter.count]           // 0 (getter)
counter.count = 5                 // setter
counter.inc()                     // bumps count and total
seen += counter.count             // 6
counter.count = -3                // the setter clamps at 0
seen += counter.count             // 0

counter.total = 10                // external write to the live binding
counter.inc()                     // internal write (+1)
[seen, counter.total, typeName(reflect.get(counter, "count"))]

// en.gad and pt.gad, sorted; `punct` is their module param (the first import of
// a module applies its params, like any import)
plugins := import("./plugins/*.gad"; punct=".")
[
    [p.lang for p in plugins],
    [p.greet("Gad") for p in plugins],
    import("./no_such_dir/*.gad"),                    // no match: []
]

langs := func(mods) => [m.lang for m in mods]
[
    langs(import("./plugins/*.gad"; @excludes=["pt.gad"])),
    langs(import("./plugins/*.gad"; @includes_re=["^p"])),     // pt.gad (not the test file)
    langs(import("./plugins/*.gad"; @includes=["*_test.gad"])), // only the test file
    langs(import("./plugins/*.gad"; @includes=["*.gad", "*_test.gad"])),
]

return mathx.square(7)
```
