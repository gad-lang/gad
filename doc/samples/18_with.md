
# With

`with` runs a resource's enter/exit hooks around a block, so cleanup always
happens — even on an early return or an error. A resource is any value that
provides the hooks: a Gad object with `enter()` / `exit(err)` methods, or a Go
type implementing the `ObjectEnter` / `ObjectExit` interfaces. A value with
neither is a silent no-op. `exit` receives any error raised in the block (`nil`
on normal exit) and the error still propagates after it runs; resources nest and
their `exit` hooks run in reverse order.

## Binding forms

```
with resource { … }         // use an existing value
with mk() as f { … }        // bind the resource to a block-local `f`
with x := mk() { … }        // define `x` (visible after the block)
var x
with x = mk() { … }         // assign to an existing variable
```

```gad
// `as` binds the resource to a block-local name.
println("as:")
with Res(; name = "a") as f {
    println("  use  ", f.name, "open =", f.open)
}

// `:=` defines a variable visible after the block.
println("define:")
with r := Res(; name = "b") { println("  use  ", r.name) }
println("  after, open =", r.open) // false (exit ran)

// An error still runs exit, then propagates; resources nest (reverse-order exit).
println("error+nested:")
try {
    with Res(; name = "outer") {
        with Res(; name = "inner") { throw "boom" }
    }
} catch e {
    println("  caught:", e)
}
```

## Expression forms

The **colon** variant, `with resource [as name]: value`, enters the resource,
evaluates `value`, runs `exit`, and yields `value`:

```gad
// Colon form: enter, evaluate, exit, yield the value.
contents := with Res(; name = "d") as g: g.read()
println("expr:", contents)
```

The **block** variant, `with resource [as name] { body }`, is an expression too:
it enters the resource, runs `body`, runs `exit`, and yields **the resource
itself** — a build-and-return primitive. `as name` is optional; bind one only
when the body needs to reference the resource.

```gad
// Block form: enter, run the body, exit, yield the RESOURCE (built inside).
built := with Res(; name = "e") as it {
    it.add(1)
    it.add(2)
}
println("expr-block:", built.name, "open =", built.open, "items =", built.items)

// Being an expression, a function can build-and-return with it.
make := func() => with Res(; name = "f") as it { it.add(9) }
println("make():", make().items)

// A non-resource value is a silent no-op (the body still runs).
with 42 as n { println("noop:", n) }
```

`with` introduces no new opcode: it desugars to a block that registers
`gad.exit(resource, $err)` as a [`deferb`](03_functions.md) and then calls
`gad.enter(resource)`, dispatched through the `gad.enter` / `gad.exit` functions
in the global `gad` namespace.

## Capturing output with a buffer

A `buffer` is a `with` resource: entering it captures everything `print`/`write`
emits inside the block (the same output-buffering `obstart`/`obend` do), and —
since `with` yields the resource — the block evaluates to the buffer holding the
captured text. So `content := with buffer() { print("hello") }` gives a buffer
whose `str(content)` is `"hello"`. It is the block-scoped, auto-closed form of the
`obstart` … `obend` pair.

```gad
// `buffer()` captures the block's output; `with` yields the buffer with it.
content := with buffer() { print("hello, "); print("world") }
[str(content), typeName(content)]
// => ["hello, world", "buffer"]
```

## Example — `18_with.gad`

```gad
/// A resource: a class with enter()/exit(err) hooks (plus a little state).
Res := Class("Res", (cls, define) => define(; fields = (; name = (= ""), open = (= false), items = (= nil)),
    methods = [
        enter(this) { this.open = true; this.items = []; println("  open ", this.name); return this }
        exit(this, err) { this.open = false; println("  close", this.name, "err =", err) }
        read(this) => this.name + "-data"
        add(this, v) { this.items += v }
    ]))

// `as` binds the resource to a block-local name.
println("as:")
with Res(; name = "a") as f {
    println("  use  ", f.name, "open =", f.open)
}

// `:=` defines a variable visible after the block.
println("define:")
with r := Res(; name = "b") { println("  use  ", r.name) }
println("  after, open =", r.open) // false (exit ran)

// An error still runs exit, then propagates; resources nest (reverse-order exit).
println("error+nested:")
try {
    with Res(; name = "outer") {
        with Res(; name = "inner") { throw "boom" }
    }
} catch e {
    println("  caught:", e)
}

// Colon form: enter, evaluate, exit, yield the value.
contents := with Res(; name = "d") as g: g.read()
println("expr:", contents)

// Block form: enter, run the body, exit, yield the RESOURCE (built inside).
built := with Res(; name = "e") as it {
    it.add(1)
    it.add(2)
}
println("expr-block:", built.name, "open =", built.open, "items =", built.items)

// Being an expression, a function can build-and-return with it.
make := func() => with Res(; name = "f") as it { it.add(9) }
println("make():", make().items)

// A non-resource value is a silent no-op (the body still runs).
with 42 as n { println("noop:", n) }

// `buffer()` captures the block's output; `with` yields the buffer with it.
content := with buffer() { print("hello, "); print("world") }
[str(content), typeName(content)]
```
