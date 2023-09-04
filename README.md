# The Gad Language

[![Go Reference](https://pkg.go.dev/badge/github.com/gad-lang/gad.svg)](https://pkg.go.dev/github.com/gad-lang/gad)
[![Go Report Card](https://goreportcard.com/badge/github.com/gad-lang/gad)](https://goreportcard.com/report/github.com/gad-lang/gad)
[![Gad Test](https://github.com/gad-lang/gad/actions/workflows/workflow.yml/badge.svg)](https://github.com/gad-lang/gad/actions/workflows/workflow.yml)
[![Gad Dev Test](https://github.com/gad-lang/gadedev/workflows/gaddev-test/badge.svg)](https://github.com/gad-lang/gadedev/actions)
[![Maintainability](https://api.codeclimate.com/v1/badges/a358e050217385db8002/maintainability)](https://codeclimate.com/github/gad-lang/gad/maintainability)

Gad is a fast, dynamic scripting language to embed in Go applications.
Gad is compiled and executed as bytecode on stack-based VM that's written
in native Go.

Gad is actively used in production to evaluate Sigma Rules' conditions, and to
perform compromise assessment dynamically.

To see how fast Gad is, please have a look at fibonacci
[benchmarks](https://github.com/gad-lang/gadebenchfib) (not updated frequently).

> Play with Gad via [Playground](https://play.verigraf.com) built for
> WebAssembly.

**Fibonacci Example**

```go
param arg0

var fib

fib = func(x) {
    if x == 0 {
        return 0
    } else if x == 1 {
        return 1
    }
    return fib(x-1) + fib(x-2)
}
return fib(arg0)
```

## Features

* Written in native Go (no cgo).
* Supports Go 1.15 and above.
* `if else` statements.
* `for` and `for in` statements.
* `try catch finally` statements.
* `param`, `global`, `var` and `const` declarations.
* Rich builtins.
* Pure Gad and Go Module support.
* Go like syntax with additions.
* Call Gad functions from Go.
* Import Gad modules from any source (file system, HTTP, etc.).
* Create wrapper functions for Go functions using code generation.

## Why Gad

`Gad` (Hebrew: גָּד‎, Modern: Gad, Tiberian: Gāḏ, "luck") was, according to the Book of Genesis, the first of the two 
sons of Jacob and Zilpah (Jacob's seventh son) and the founder of the Israelite tribe of Gad.[2] 
The text of the Book of Genesis implies that the name of Gad means luck/fortunate, in Hebrew.

## Quick Start

`go get github.com/gad-lang/gad@latest`

Gad has a REPL application to learn and test Gad scripts.

`go install github.com/gad-lang/gad/cmd/gad@latest`

`./gad`

![repl-gif](https://github.com/gad-lang/gad/blob/main/docs/repl.gif)

This example is to show some features of Gad.

<https://play.golang.org/p/1Tj6joRmLiX>

```go
package main

import (
    "fmt"

    "github.com/gad-lang/gad"
)

func main() {
    script := `
param ...args

mapEach := func(seq, fn) {

    if !isArray(seq) {
        return error("want array, got " + typeName(seq))
    }

    var out = []

    if sz := len(seq); sz > 0 {
        out = repeat([0], sz)
    } else {
        return out
    }

    try {
        for i, v in seq {
            out[i] = fn(v)
        }
    } catch err {
        println(err)
    } finally {
        return out, err
    }
}

global multiplier

v, err := mapEach(args, func(x) { return x*multiplier })
if err != undefined {
    return err
}
return v
`

    bytecode, err := gad.Compile([]byte(script), gad.DefaultCompilerOptions)
    if err != nil {
        panic(err)
    }
    globals := gad.Map{"multiplier": gad.Int(2)}
    ret, err := gad.NewVM(bytecode).Run(
        globals,
        gad.Int(1), gad.Int(2), gad.Int(3), gad.Int(4),
    )
    if err != nil {
        panic(err)
    }
    fmt.Println(ret) // [2, 4, 6, 8]
}
```

## TODO

- [ ] Nullisch Coalescing
- [ ] Named arguments
- [ ] Array Expansion
- [ ] Array Comprehensions
- [ ] Map Expansion
- [ ] Map Comprehensions
- [ ] Examples for best practices
- [ ] Better Playground
- [ ] Configurable Stdin, Stdout and Stderr per Virtual Machine
- [ ] Deferring function calls
- [ ] Concurrency support

## Documentation

* [Tutorial](https://github.com/gad-lang/gad/blob/main/docs/tutorial.md)
* [Runtime Types](https://github.com/gad-lang/gad/blob/main/docs/runtime-types.md)
* [Builtins](https://github.com/gad-lang/gad/blob/main/docs/builtins.md)
* [Operators](https://github.com/gad-lang/gad/blob/main/docs/operators.md)
* [Error Handling](https://github.com/gad-lang/gad/blob/main/docs/error-handling.md)
* [Standard Library](https://github.com/gad-lang/gad/blob/main/docs/stdlib.md)
* [Optimizer](https://github.com/gad-lang/gad/blob/main/docs/optimizer.md)
* [Destructuring](https://github.com/gad-lang/gad/blob/main/docs/destructuring.md)

## LICENSE

Gad is licensed under the MIT License.

See [LICENSE](LICENSE) for the full license text.

## Acknowledgements

Gad is inspired by script language [uGo](https://github.com/ozanh/gad)
by Ozan Hacıbekiroğlu. A special thanks to uGo's creater and contributors.
