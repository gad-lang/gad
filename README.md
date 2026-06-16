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
return fib(int(arg0))
```

## Features

* Written in native Go (no cgo).
* Supports Go 1.15 and above.
* `if else` statements.
* `for` and `for in` statements.
* `try catch finally` statements.
* `match` expression and statement (PHP 8 like).
* `defer` / `defer_ok` / `defer_err` (function) and `deferb*` (block-scoped)
  deferred handlers, with `$ret` / `$err` access and recovery.
* `or` error-fallback operator (`z := x() or fallback`).
* Array and dict comprehensions (`[e for x in it if c]`, `{k: v for ...}`).
* Array and dict spread/merge literals (`[1, *a, *b]`, `{a: 1, *b}`).
* Dict and `MixedParams` destructuring assignment.
* `/regex/` literals backed by a built-in `regexp` type (match / find / replace).
* `param`, `global`, `var` and `const` declarations.
* Rich builtins.
* Pure Gad and Go Module support.
* Go like syntax with additions.
* Call Gad functions from Go.
* Import Gad modules from any source (file system, HTTP, etc.).
* Create wrapper functions for Go functions using code generation.

## Language additions

A few of the syntax additions over a Go-like base:

```go
// or: error fallback (re-throws if the fallback is itself an error)
z := mayThrow() or 2
y := 1 + (mayThrow() or 10)

// match (PHP 8 like) — expression and statement forms
label := match (n) {
    1: "one"
    2: "two",
    else: "other"
}
match (n) { 1 { return "one" }, else { return "other" } }

// comprehensions; dict keys are static by default, computed with [..],
// and `_` is the dict being built
squares := [i * i for i in [1, 2, 3] if i > 1]
m := {x: 10, [i]: i * i, z: (_.z ?? 0) + i for i in [1, 2, 3]}

// spread / merge literals (a leading spread yields a fresh copy)
all := [1, *a, 4, *b]
merged := {a: 1, *b, c: 2, *d}

// destructuring: dict and MixedParams
(;a, _b:b, r=2, **other) := {a: 2, b: 3, x: 4}     // a=2, _b=3, r=2, other={x:4}
x := (1, 2, *[3, 4]; c=5, **{d:6, e:7})
(a, b, **pos_rest; c, p:d, r=2, **named_rest) := x

// defer with $ret/$err (recover by clearing $err); deferb runs at block exit
f := func() {
    defer_err { $ret = "recovered: " + str($err); $err = nil }
    throw "boom"
}

// regexp literals and replacement (| yields a replacer; composes with .|)
re := /(\d+)-(\d+)/
re.match("12-34")                 // true
re.replace("12-34", "$2/$1")      // "34/12"
"hello world".|(/o/ | "0")        // "hell0 w0rld"
```

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

### CLI tools

The `gad` binary is organised as subcommands:

| Command     | Purpose                                                           |
|-------------|------------------------------------------------------------------|
| `gad run`   | Run a script/stdin (or a `.gadt`/`--template` template), or the REPL. |
| `gad fmt`   | Format Gad source files in place.                                |
| `gad debug` | Debug a script — interactive REPL or `--dap` for editors.        |
| `gad ide`   | Start a local web IDE (file tree, tabs, format/run/debug).       |

Gad also has a **template / mixed mode** (`{% … %}` code, `{%= … %}` values,
`begin … end` blocks, whitespace trim markers) — see
[doc/templates.md](doc/templates.md).

### Samples & the web IDE

The [`samples/`](samples) directory is a guided tour of the language and the
standard library. Open it in the bundled web IDE:

```sh
make ide                 # serves the samples workspace in your browser
# or: gad ide samples    # or: gad ide path/to/your/project
```

The IDE offers multi-file tabs, formatting, running and full debugging
(breakpoints, stepping, call stack and locals), with per-file run/debug dialogs
(arguments, builtin-module toggles, output capture) and settings stored in the
workspace `.gad.yaml`. See [samples/README.md](samples/README.md).

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
param *args

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
if err != nil {
    return err
}
return v
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

## TODO

- [x] Dollar as valid ident char
- [x] Nullisch Coalescing
- [x] Named arguments
- [x] Array Expansion
- [x] Array Comprehensions
- [x] Map Expansion
- [x] Map Comprehensions
- [ ] Examples for best practices
- [ ] Better Playground
- [x] Configurable Stdin, Stdout and Stderr per Virtual Machine
- [x] Deferring function calls
- [ ] Concurrency support
- [ ] `?` and `!` as valid post 2nd ident char
  - `valid?() = true`
  - `if valid? {}`
  - `push!(values array) { arr += values }`
- [x] Parse bytes from hex string (`const data = h"ffccf1c2" // typeof data == bytes`)
- [x] Parse bytes from string (`const data = b"Hello" // typeof data == bytes`)
- [ ] Parse string from unquoted form:
  - `const data = :hello // data == str("hello")`)
  - `const data = :(Hello World!) // data == str("Hello World!")`

## Documentation

* **[User Guide](doc/README.md)** — hands-on, example-driven docs for every
  language feature (start here).
* [Tutorial](https://github.com/gad-lang/gad/blob/main/docs/tutorial.md)
* [Runtime Types](https://github.com/gad-lang/gad/blob/main/docs/runtime-types.md)
* [Builtins](https://github.com/gad-lang/gad/blob/main/docs/builtins.md)
* [Operators](https://github.com/gad-lang/gad/blob/main/docs/operators.md)
* [Error Handling](https://github.com/gad-lang/gad/blob/main/docs/error-handling.md)
* [Standard Library](https://github.com/gad-lang/gad/blob/main/docs/stdlib.md)
* [Optimizer](https://github.com/gad-lang/gad/blob/main/docs/optimizer.md)
* [Destructuring](https://github.com/gad-lang/gad/blob/main/docs/destructuring.md)

## LICENSE

Gad is licensed under the **MIT License** — see [LICENSE](LICENSE) for the full
text. You are free to use, copy, modify and distribute it, including in
commercial and closed-source projects, provided the copyright and license
notice is retained.

Gad includes code derived from third-party projects, each under its own license:

* [LICENSE.golang](LICENSE.golang) — the Go standard library (BSD-3-Clause).
* [LICENSE.tengo](LICENSE.tengo) — the Tengo language (MIT).

## Acknowledgements

Gad is inspired by script language [uGo](https://github.com/ozanh/gad)
by Ozan Hacıbekiroğlu. A special thanks to uGo's creater and contributors.
