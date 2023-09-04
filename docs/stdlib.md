# Standard Library

## Module List

* [fmt](stdlib-fmt.md) module at `github.com/gad-lang/gad/stdlib/fmt`
* [strings](stdlib-strings.md) module at `github.com/gad-lang/gad/stdlib/strings`
* [time](stdlib-time.md) module at `github.com/gad-lang/gad/stdlib/time`
* [json](stdlib-json.md) module at `github.com/gad-lang/gad/stdlib/json`

## How-To

### Import Module

Each standard library module is imported separately. `Module` variable as
`map[string]Object` in modules holds module values to pass to module map which
is deeply copied then.

**Example**

```go
package main

import (
    "github.com/gad-lang/gad"
    "github.com/gad-lang/gad/stdlib/fmt"
    "github.com/gad-lang/gad/stdlib/json"
    "github.com/gad-lang/gad/stdlib/strings"
    "github.com/gad-lang/gad/stdlib/time"
)

func main() {
    script := `
    const fmt = import("fmt")
    const strings = import("strings")
    const time = import("time")
    const json = import("json")

    total := 0
    fn := func() {
        start := time.Now()
        try {
            /* ... */
        } finally {
            total += time.Since(start)
        }
    }
    fn()
    /* ... */
    `
    moduleMap := gad.NewModuleMap()
    moduleMap.AddBuiltinModule("fmt", fmt.Module)
    moduleMap.AddBuiltinModule("strings", strings.Module)
    moduleMap.AddBuiltinModule("time", time.Module)
    moduleMap.AddBuiltinModule("json", json.Module)

    opts := gad.DefaultCompilerOptions
    opts.ModuleMap = moduleMap

    byteCode, err := gad.Compile([]byte(script), opts)
    ret, err := gad.NewVM(byteCode).Run(nil)
    /* ... */
}
```
