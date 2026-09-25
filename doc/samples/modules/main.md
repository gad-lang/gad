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

return mathx.square(7)
```
