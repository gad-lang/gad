# Embedding Gad in Go

[← Back to index](README.md)

Gad is built to be embedded. A script is compiled to a `Bytecode` object and
then run on a `VM`. This chapter shows the minimal flow and the common
extension points.

## Minimal Example

```go
package main

import (
	"fmt"

	"github.com/gad-lang/gad"
)

func main() {
	script := `
param *args
global multiplier
return [x * multiplier for x in args]
`
	builtins := gad.NewBuiltins()
	st := gad.NewSymbolTable(builtins.NameSet)

	_, bc, err := gad.Compile(st, []byte(script), gad.CompileOptions{})
	if err != nil {
		panic(err)
	}

	ret, err := gad.NewVM(builtins.Build(), bc).RunOpts(&gad.RunOpts{
		Globals: gad.Dict{"multiplier": gad.Int(2)},
		Args:    gad.Args{gad.Array{gad.Int(1), gad.Int(2), gad.Int(3), gad.Int(4)}},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(ret) // [2, 4, 6, 8]
}
```

The flow is:

1. **Builtins** — `gad.NewBuiltins()` creates the builtin set; `.NameSet` seeds
   the symbol table and `.Build()` produces the runtime objects.
2. **Symbol table** — `gad.NewSymbolTable(builtins.NameSet)` tracks declared
   names at compile time.
3. **Compile** — `gad.Compile(st, src, opts)` returns the bytecode (`bc`).
4. **Run** — `gad.NewVM(builtins.Build(), bc).RunOpts(&gad.RunOpts{…})` executes
   it and returns the script's `return` value as a `gad.Object`.

## Passing Globals and Arguments

* `Globals` is any map-like `gad.Object` (e.g. `gad.Dict`). Script `global`
  declarations read and write through it, so it is also how the script returns
  data back to Go by mutation.
* `Args` provides the positional/variadic values consumed by the script's
  `param` statement.

```go
RunOpts{
    Globals: gad.Dict{"multiplier": gad.Int(2)},
    Args:    gad.Args{gad.Array{gad.Int(1), gad.Int(2)}},
}
```

## Exposing Go Functions

A Go function exposed as a `*gad.Function` becomes callable from a script.
Returning a `gad.Array` lets the script destructure multiple values:

```go
globals := gad.Dict{
    "goAdd": &gad.Function{
        FuncName: "goAdd",
        Value: func(c gad.Call) (gad.Object, error) {
            a := c.Args.Get(0).(gad.Int)
            b := c.Args.Get(1).(gad.Int)
            return a + b, nil
        },
    },
}
// script: global goAdd; return goAdd(2, 3)
```

For wrapping existing Go functions with less boilerplate, the repository
provides the `cmd/mkcallable` code generator.

## Reusing a VM

A `VM` is reusable. After a run you can run again; `Clear` releases references
and resets the stack and module cache between unrelated runs.

## Modules from Go

To make modules importable, build a module map and pass it through the compile
options. Source modules are added as bytes; builtin modules are
`map[string]Object`; any value implementing `Importable` can serve as a custom
module. The CLI wires a file-system importer that resolves `.gad` files along
`GADPATH` — see [Modules](modules.md) for the script-side view.

## Safety

Gad does not impose allocation limits and relies on Go's garbage collector. To
run untrusted scripts, disable risky builtins/modules before compilation (the
CLI's `-safe` and `-disabled-modules` flags do this) and run inside whatever
sandbox your application provides. Compilation is deterministic, and bytecode
can be serialized for transport and executed later.
