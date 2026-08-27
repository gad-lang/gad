# main

main.gad — imports the sibling source modules in this directory.

Run from the repo root:  gad run samples/modules/main.gad
See doc/modules.md for detailed documentation.

## Example — `main.gad`

```gad
mathx := import("./mathx.gad")
println("pi          =", mathx.pi)
println("square(7)   =", mathx.square(7))
println("cube(3)     =", mathx.cube(3))
println("circleArea  =", mathx.circleArea(2))
println("rectArea    =", mathx.rectArea(3, 4))

/**
Import with parameters (named import args). A module runs once and is then
cached, so the params apply on its first import.
**/
pt := import("./greet.gad"; lang="pt")
println("language:", pt.language)   // pt
println(pt.greet("Gad"))            // Olá, Gad!

/**
A module exporting a property: member access delegates to its getter/setter.
**/
counter := import("./counter.gad")
println("count       =", counter.count)          // 0  (getter)
counter.count = 5                                 // setter
counter.inc()                                     // 6
println("count       =", counter.count)          // 6
counter.count = -3                                // setter clamps at 0
println("count       =", counter.count)          // 0
/**
A live binding (export prop total = 0): external and internal writes share
the same module variable.
**/
counter.total = 10                                // external write
counter.inc()                                     // internal write (+1)
println("total       =", counter.total)          // 11  (shared live binding)
/**
reflect.get reads the exported Prop object itself (getter not run).
**/
println("raw member  =", typeName(reflect.get(counter, "count")))  // Prop

return mathx.square(7)
```
