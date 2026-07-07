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

## Typed Methods with `AddMethod`

A plain `*gad.Function` has one body that must type-check its own arguments. To
give a callable **several typed overloads** that the VM dispatches on by argument
type — and to surface their signatures in `repr(f; indent)` and generated docs —
build each overload with `gad.NewFunction(...)` plus `gad.FunctionWithParams` /
`gad.FunctionWithNamedParams`, then attach them with `gad.AddMethod`.

**Convention:** prefer `AddMethod` (over a single hand-rolled type switch)
whenever a builtin function or type accepts more than one argument shape. Each
overload then carries a declared header, dispatch is handled by the VM, and the
signatures show up in tooling.

```go
// A `clamp` builtin with two typed overloads. The VM picks the int or the
// float method based on the argument types; an unmatched call falls through to
// the function's default body (here, an error).
clamp := gad.NewFunction("clamp",
    func(c gad.Call) (gad.Object, error) {
        return nil, gad.NewArgumentTypeError("0", "int|float", c.Args.Get(0).Type().Name())
    })

method := func(t gad.ObjectType, run func(c gad.Call) (gad.Object, error)) *gad.Function {
    return gad.NewFunction("clamp", run,
        gad.FunctionWithParams(func(p func(name string) *gad.ParamBuilder) {
            p("v").Type(t)
            p("lo").Type(t)
            p("hi").Type(t)
        }))
}

gad.AddMethod(clamp,
    method(gad.TInt, func(c gad.Call) (gad.Object, error) {
        v, lo, hi := c.Args.Get(0).(gad.Int), c.Args.Get(1).(gad.Int), c.Args.Get(2).(gad.Int)
        return max(lo, min(v, hi)), nil
    }),
    method(gad.TFloat, func(c gad.Call) (gad.Object, error) {
        v, lo, hi := c.Args.Get(0).(gad.Float), c.Args.Get(1).(gad.Float), c.Args.Get(2).(gad.Float)
        return max(lo, min(v, hi)), nil
    }),
)
// script: global clamp; return [clamp(5, 0, 3), clamp(0.2, 0.0, 1.0)] // [3, 0.2]
```

`AddMethod` works on any method target: a `*gad.Function` (as above), a builtin
object type (`*gad.BuiltinObjType`), or a registered global builtin. For a
**type**, the overloads become typed constructors/methods:

```go
// Add typed single-argument constructors to a builtin type. Each delegates to
// the type's own constructor; the typed `T(v <kind>)` headers then appear in
// repr(T; indent). (`AddMethod` mutates the type in place.)
gad.AddMethod(MyType,
    gad.NewFunction("MyType", myTypeNew,
        gad.FunctionWithParams(func(p func(name string) *gad.ParamBuilder) {
            p("v").Type(gad.TStr)
        })),
    gad.NewFunction("MyType", myTypeNew,
        gad.FunctionWithParams(func(p func(name string) *gad.ParamBuilder) {
            p("v").Type(gad.TInt)
        })),
)
```

For a **global builtin** registered in `gad.BuiltinObjects`, use
`gad.BuiltinObjects.AddMethod(BuiltinX, methods...)` instead of the package-level
`AddMethod` — it reassigns the entry so the methods survive the static-builtins
snapshot. (See `objects_range.go` for the `Range` type and `module_time_ctors.go`
for the time-module constructors, both built this way.)

`gad.AddMethodOverride(true, target, methods...)` replaces an existing overload
with the same parameter types instead of erroring on the duplicate.

## Operators on custom types

A Go type opts into a binary operator by implementing the matching per-operator
interface from the generated `op_api.go` — one interface per operator, named
`ObjectWith{Op}BinOperator` with a method `BinOp{Op}(vm *VM, right Object)`:

```go
// Vec supports `vec + vec` and `vec < vec`.
type Vec struct{ X, Y int }

func (v Vec) BinOpAdd(_ *gad.VM, right gad.Object) (gad.Object, error) {
    o, ok := right.(Vec)
    if !ok {
        return nil, gad.NewOperandTypeError("+", "Vec", right.Type().Name())
    }
    return Vec{v.X + o.X, v.Y + o.Y}, nil
}

func (v Vec) BinOpLess(_ *gad.VM, right gad.Object) (gad.Object, error) {
    o, ok := right.(Vec)
    if !ok {
        return nil, gad.NewOperandTypeError("<", "Vec", right.Type().Name())
    }
    return gad.Bool(v.X*v.X+v.Y*v.Y < o.X*o.X+o.Y*o.Y), nil
}
```

A type implements only the operators it supports; unsupported ones fall back to
an "unsupported operand types" error. The membership operator `a in b` is
special: it is dispatched on the **right** operand via
`ObjectWithInBinOperator.BinOpIn(vm, value)` (the container reports whether
`value` is a member). The array-membership operator `A ain B` ("all in") is
dispatched the same way via `ObjectWithAinBinOperator.BinOpAin(vm, values)`;
without it, `ain` falls back to testing each value with `in`. To run an operator
generically from Go, call `gad.BinaryOp(vm, tok, left, right)`.

The same operators are also overridable from Gad with
`met gad.binOp{Op}(left T, right U)` (see [Operators](operators.md)).

Unary and self-assign operators follow the same pattern with their own generated
interfaces:

- Unary (`!`, `-`, `+`, `^`, `++`, `--`): `ObjectWith{Op}UnaryOperator` with
  `UnOp{Op}(vm *gad.VM) (gad.Object, error)`. Logical NOT is universal (any value
  falls back to its truthiness), so only override `UnOpNot` for a custom result.
  Dispatch is per-operator `gad.unOp{Op}` / `met gad.unOp{Op}(operand T)`.
- Self-assign (`x op= y`): `ObjectWith{Op}SelfAssignOperator` with
  `SelfAssignOp{Op}(vm *gad.VM, value gad.Object) (gad.Object, error)`. An
  operator a type does not implement falls back to the binary operator (so
  `x op= y` runs as `x = x op y`). Dispatch is per-operator
  `gad.selfAssignOp{Op}` / `met gad.selfAssignOp{Op}(left T, right U)`.

```go
// Array += v appends one value; ++= v extends.
func (o Array) SelfAssignOpAdd(_ *gad.VM, value gad.Object) (gad.Object, error) {
    return append(o, value), nil
}
```

## Reusing a VM

A `VM` is reusable. After a run you can run again; `Clear` releases references
and resets the stack and module cache between unrelated runs.

## Calling gad functions from Go

Use an `Invoker` to call a gad `CompiledFunction` (or any callable) from Go. It
forks a child VM for a compiled function and reuses the caller's globals and I/O:

```go
inv := gad.NewInvoker(vm, fn)
ret, err := inv.Invoke(gad.Args{args}, nil)
```

`Acquire`/`Release` reuse a pooled VM across several calls (about 3× faster than
a fresh VM per call); call `Release` when done.

### Cancellation and timeouts

Bind a context with `WithContext` to abort a long-running or infinite invoked
function. When the context is cancelled (a `context.WithTimeout` deadline or a
`context.WithCancel` cancel) before the call returns, the whole VM tree is
aborted from the root — so the invoked function **and every VM it spawned through
nested invokers** stop — and `Invoke` returns the context's error
(`context.DeadlineExceeded` / `context.Canceled`):

```go
ctx, cancel := context.WithTimeout(context.Background(), time.Second)
defer cancel()

ret, err := gad.NewInvoker(vm, fn).WithContext(ctx).Invoke(gad.Args{args}, nil)
if errors.Is(err, context.DeadlineExceeded) {
    // the function ran past the deadline and was aborted
}
```

Because the abort cascades from the root, a deadline stops nested invocations at
every level. A nil or non-cancellable context (the default) adds no goroutine and
no overhead. The `Caller` path (`inv.Caller(...)`, a reusable `VMCaller`) honours
the context the same way. For running a whole *script* under a deadline,
`Eval.RunScript(ctx, …)` wires the same context-to-abort guard.

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

To bound execution time — the main defense against infinite loops and runaway
scripts — run under a cancellable context: `Eval.RunScript(ctx, …)` for a whole
script, or `Invoker.WithContext(ctx)` for a single call (see
[Calling gad functions from Go](#calling-gad-functions-from-go)). A cancelled
context aborts the VM tree at the next instruction.
