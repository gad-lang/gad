# Go modules

[← Back to Get Started](getting-started.md) · [Embedding Gad in Go](embedding.md)

How to make modules for Gad in Go, in one place: the kinds of modules, the
conventions of their Go code, how they are documented, and the modules written
in Gad that are shipped as Go packages (`gad gomodule`).

- [The kinds of modules](#the-kinds-of-modules)
- [Native Go modules (`StdModuleData`)](#native-go-modules-stdmoduledata)
- [Conventions for the Go code of a module](#conventions-for-the-go-code-of-a-module)
- [Go values in gad (`ReflectValue`)](#go-values-in-gad-reflectvalue)
- [Documenting a module (`gad:doc`)](#documenting-a-module-gaddoc)
- [Source modules and dialects](#source-modules-and-dialects)
- [Modules written in Gad, shipped in Go (`gad gomodule`)](#modules-written-in-gad-shipped-in-go-gad-gomodule)
  - [Transpiled to Go (`gad gomodule --go`)](#transpiled-to-go-gad-gomodule---go)

## The kinds of modules

To make modules importable, build a module map and pass it through the compile
options. Builtin modules are `map[string]Object`; any value implementing
`Importable` can serve as a custom module. The CLI wires a file-system importer
(`importers.FileImporter`) that resolves module files along `GADPATH` — see
[Modules](samples/26_embed.md) for the script-side view.

| Kind | Its members | Registered with |
| ---- | ----------- | --------------- |
| native Go module | Go values (`StdModuleData`) | `mm.AddBuiltinModuleInit(name, ModuleInit)` |
| source module | Gad source (`.gad`/`.gadt`/`.gadx`), compiled when imported | `mm.AddSourceModule(Kind)`, `importers.FileImporter` |
| compiled module | Gad bytecode (`.gadb`), linked when imported | `mm.Add(name, encoder.BytecodeModule(data))` |
| Gad shipped in Go | a `.gadb` and a Go init, a package made by `gad gomodule` | `pkg.AddTo(mm)` |

`gad gomod init NAME` scaffolds a native module: `NAME/module.go` (a documented
`ModuleInit`) and `NAME/samples.gad` (its examples, verified by `gad doc`).

## Native Go modules (`StdModuleData`)

A **native module** is one whose members live in Go. Register a
`gad.ModuleInitFunc` — called the first time the module is imported — and set the
module's `Data`:

```go
func(module *gad.Module, c gad.Call) (err error) { module.Data = …; return }
```

`Data` is any value implementing `gad.ModuleData`. The standard implementation,
**`gad.StdModuleData`**, keeps a module's members in three disjoint buckets:

| Field    | Kind      | `module.name = x` (from a script) |
| -------- | --------- | --------------------------------- |
| `Vars`   | variables | allowed (reassigns the variable)  |
| `Consts` | constants | rejected — read-only              |
| `Funcs`  | functions | rejected — mutate via `@funcs`    |

Reads (`m.name`, `len(m)`, `for k, v in m`) see the merged view; only `Vars` is
writable through `m.name = x`. Assign the data **by value** — `StdModuleData`
satisfies `ModuleData` without a pointer, so no `&` is needed.

Because the buckets are plain `gad.Dict`s (Go maps, i.e. reference values), an
exported function can **close over the very dict the module exposes** and mutate
shared state the rest of the module observes:

```go
package counter

import "github.com/gad-lang/gad"

// ModuleInit builds a `counter` module: a read-only Start, a mutable n, and
// functions that operate on the shared n.
var ModuleInit gad.ModuleInitFunc = func(module *gad.Module, c gad.Call) (err error) {
    vars := gad.Dict{"n": gad.Int(0)} // exported, mutable

    funcs := gad.Dict{
        // inc() closes over `vars`, so it mutates the exported `n`.
        "inc": &gad.Function{FuncName: "inc", Value: func(gad.Call) (gad.Object, error) {
            vars["n"] = vars["n"].(gad.Int) + 1
            return vars["n"], nil
        }},
        "get": &gad.Function{FuncName: "get", Value: func(gad.Call) (gad.Object, error) {
            return vars["n"], nil
        }},
    }

    module.Data = gad.StdModuleData{
        Vars:   vars,
        Consts: gad.Dict{"Start": gad.Int(0)}, // read-only
        Funcs:  funcs,
    }
    return
}
```

Register it in the module map and import it from a script:

```go
mm := gad.NewModuleMap()
mm.AddBuiltinModuleInit("counter", counter.ModuleInit)
opts := gad.CompileOptions{CompilerOptions: gad.CompilerOptions{ModuleMap: mm}}
```

```gad ignore
c := import("counter")
c.inc(); c.inc()
c.get()        // 2 — the function sees the exported var
c.n            // 2 — read the variable directly
c.n = 10       // ok: `n` is a variable
c.Start = 1    // error: cannot assign to constant Start (use .@consts)
```

The three buckets are also reachable as live dicts for reflection or controlled
mutation: `c.@vars`, `c.@consts`, `c.@funcs`. Writing through them bypasses the
read-only guard (`c.@consts["Start"] = 1`), so treat them as the deliberate
escape hatch.

To populate incrementally instead of building the dicts up front, start from
`gad.NewStdModuleData()` and call `Set(name, value)` (a variable, or a function
when the value is one) and `SetConst(name, value)` (a constant); each keeps the
buckets disjoint.

> Prefer `StdModuleData` for new modules. A bare `gad.Dict` still works as
> `ModuleData` (every member mutable, no const/func distinction) and remains the
> form some builtin modules use.

## Conventions for the Go code of a module

- **Names.** The module's name is a lowercase snake_case identifier, its Go
  package's name too; the package declares `const ModuleName` and the
  `ModuleInit` (or, made by `gad gomodule`, `Module()` and `AddTo`).
- **Errors are returned**, as Go's convention is: a function is
  `func(c gad.Call) (gad.Object, error)` and fails by returning an error —
  gad's kinds (`gad.ErrType.NewErrorf`, `gad.NewArgumentTypeError`,
  `gad.ErrWrongNumArguments`, …), which a script catches with `try`. Never
  `panic` for an error of a call.
- **Arguments** are checked by the helpers of `gad.Call`: `c.Args.CheckLen(n)`,
  `c.Args.Destructure…`, `c.NamedArgs.Get(…)`; a typed function declares its
  `Header` (`gad.NewFunctionHeader()…`), from which its docs' signature comes.
- **The builtins are called through the VM** — `c.VM.CallBuiltin(gad.BuiltinLen,
  nil, v)` (named arguments as the second argument) —, never their Go
  functions directly: dispatched so, a call gets what a script's call gets, the
  overloads registered on the builtin (`met len(b Box) => …`, a host's
  `AddMethod`) and the host's own builtin when it replaced gad's.
- **Gad values are called through the VM too**: `c.VM.Call(fn, gad.Args{args},
  namedArgs)` — a compiled function, a class, a method, as a script calls it.
- **Overloads** of a function are added as a script adds them (`AddMethod`, see
  [Typed methods](embedding.md#typed-methods-with-addmethod)): a module's
  function is open to the overloads of the code that uses it.
- **State is per VM.** A module's members are built by its init, once per
  import per VM — never a mutable global of the Go package, which every VM (and
  every goroutine) would share. A cache keyed by builtin types compares them by
  key (a VM holds copies of the builtins).
- **Documented** by `gad:doc` comments (below), with examples verified by
  `gad doc` (`gad:samples`); the API stub and the Markdown are generated by
  `go generate` (`gaddoc`).

## Go values in gad (`ReflectValue`)

Any Go value becomes a gad value by `gad.MustNewReflectValue(v)` (or
`gad.NewReflectValue`, with an error): given to a script as a global, an
argument, a module's member. It is the Go value itself, reflected — not a
copy rewritten in gad —: a Go function is called through `reflect` (as
`text/template` calls its functions), a struct's field written in gad is
written in Go.

| Go | In gad |
| -- | ------ |
| `struct` (`ReflectStruct`) | fields (`a.Owner`, `a.Owner = …`), methods of the value and of the pointer (`a.Deposit(5)`), a method as a value (`f := a.Total`) |
| `func` (`ReflectFunc`) | called (`f(1, 2)`); variadic; a last `error` result is a gad error (`try` catches it); several results an array |
| `[]T`, `[n]T` (`ReflectSlice`, `ReflectArray`) | `s[i]`, `s[i] = v`, `len(s)`, `for i, v in s`; `s += v`, `s ++= values` (of `&slice`, the Go slice itself) |
| `map[K]V` (`ReflectMap`) | `m[k]`, `m.k`, `m[k] = v`, `len(m)`, `for k, v in m` |
| `chan T` (`ReflectChan`) | `for v in ch` (receives until closed), `ch.send(v)`, `ch.recv()` (`[value, ok]`), `ch.close()`, `len(ch)`, `ch.cap()` |
| `*int`, `*string`, … (`GoPtr`) | a gad pointer (`p.v`) |

What gad gives a Go parameter, field or item is converted to its Go type:
an `int` of gad to `int`/`int32`/…, an array to a slice or an array, a dict to a
map, and **a function of gad to a Go function type** — a callback, called
through the VM while the Go call runs (synchronously: not from another
goroutine). Its error is returned when the function type's last result is an
`error`, and fails the Go call otherwise. A Go function that gad holds goes
back to Go as itself. A value that does not convert is an error of the script.

The examples below are `reflect_example_test.go`, verified by `go test`; they
run with this helper:

```go
// run compiles src and runs it with globals — Go values made gad values by
// gad.MustNewReflectValue —, printing what it returns, or its error.
func run(src string, globals gad.Dict) {
	st := gad.NewSymbolTable(gad.NewBuiltins().NameSet)
	for name := range globals {
		st.DefineGlobal(name)
	}
	res, err := gad.Compile(st, []byte(src), gad.CompileOptions{})
	if err != nil {
		fmt.Println("compile:", err)
		return
	}
	ret, err := gad.NewVM(gad.NewBuiltins().Build(), res.Bytecode).RunOpts(&gad.RunOpts{Globals: globals})
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(ret)
}
```

### Structs: fields, methods, errors

```go
type Account struct {
	Owner   string
	Balance int
	Tags    []string
	// OnChange is a callback: a gad function may be assigned to it.
	OnChange func(balance int)
}

// Total is a method of the value.
func (a Account) Total(extra int) int { return a.Balance + extra }

// Deposit is a method of the pointer: it changes the account.
func (a *Account) Deposit(n int) {
	a.Balance += n
	if a.OnChange != nil {
		a.OnChange(a.Balance)
	}
}

// Withdraw fails with an error: in gad, an error a `try` catches.
func (a *Account) Withdraw(n int) (int, error) {
	if n > a.Balance {
		return a.Balance, errors.New("insufficient funds")
	}
	a.Balance -= n
	return a.Balance, nil
}

// A struct: its fields read and written, its methods called (of the value
// and of the pointer), a method as a value, an error returned by Go caught.
func Example_reflectStruct() {
	acc := &Account{Owner: "ana", Balance: 10, Tags: []string{"vip"}}
	run(`global acc
acc.Deposit(5)
acc.Owner = "Ana"
total := acc.Total
r := [acc.Owner, acc.Balance, total(100), acc.Tags[0], len(acc.Tags)]
try { acc.Withdraw(1000) } catch e { r += str(e) }
return r`, gad.Dict{"acc": gad.MustNewReflectValue(acc)})
	fmt.Println(acc.Owner, acc.Balance) // the Go value changed
	// Output:
	// ["Ana", 15, 115, "vip", 1, "error: insufficient funds"]
	// Ana 15
}
```

### Slices and maps

```go
// Slices and maps: indexed, iterated, appended to; gad's values converted to
// the Go types (an int of gad to int, an array to []int, a dict to a map).
func Example_reflectSlicesAndMaps() {
	scores := map[string]int{"ana": 1}
	ids := []int{10, 20}
	sum := func(xs []int) (s int) {
		for _, x := range xs {
			s += x
		}
		return
	}
	merge := func(m map[string]int) int { return len(m) }
	run(`global (scores, ids, sum, merge)
scores["bia"] = 2
names := []
for k, v in scores { names += k + "=" + str(v) }
ids += 30                          // appended: the Go slice too (a pointer)
ids ++= [40, 50]
return [sort(names), ids[2], len(ids), sum([1, 2, 3]), merge({a: 1, b: 2})]`, gad.Dict{
		"scores": gad.MustNewReflectValue(scores),
		"ids":    gad.MustNewReflectValue(&ids),
		"sum":    gad.MustNewReflectValue(sum),
		"merge":  gad.MustNewReflectValue(merge),
	})
	fmt.Println(scores, ids)
	// Output:
	// [["ana=1", "bia=2"], 30, 5, 6, 2]
	// map[ana:1 bia:2] [10 20 30 40 50]
}
```

### Callbacks: gad functions given to Go

```go
// Callbacks: a gad function given to a Go function whose parameter is a
// function type (called through the VM, synchronously); its error returned
// when the type returns one; a Go function returned by Go stays itself.
func Example_reflectCallbacks() {
	mapInts := func(xs []int, f func(int) int) []int {
		out := make([]int, len(xs))
		for i, x := range xs {
			out[i] = f(x)
		}
		return out
	}
	attempt := func(f func(string) (string, error)) string {
		s, err := f("go")
		if err != nil {
			return "failed: " + err.Error()
		}
		return s
	}
	multiplier := func(n int) func(int) int { return func(x int) int { return x * n } }
	acc := &Account{Balance: 1}
	run(`global (mapInts, attempt, multiplier, acc, log)
times3 := multiplier(3)           // a Go function, returned as it is
acc.OnChange = (b) => { log += b }
acc.Deposit(2); acc.Deposit(3)
return [
	mapInts([1, 2, 3], (x) => x * 10),
	mapInts([1, 2], times3),       // given back to Go: the same function
	attempt((s) => s + "!"),
	attempt((s) => { throw "no" }),
	log,
]`, gad.Dict{
		"mapInts":    gad.MustNewReflectValue(mapInts),
		"attempt":    gad.MustNewReflectValue(attempt),
		"multiplier": gad.MustNewReflectValue(multiplier),
		"acc":        gad.MustNewReflectValue(acc),
		"log":        gad.Array{},
	})
	// Output:
	// [[10, 20, 30], [3, 6], "go!", "failed: error: no", [3, 6]]
}
```

### Channels

```go
// Channels: received by `for v in ch` until closed (a Go goroutine sends),
// sent to by `ch.send(v)` (Go receives), `ch.recv()` a [value, ok] pair.
func Example_reflectChannels() {
	jobs := make(chan string)
	go func() {
		for _, j := range []string{"a", "b", "c"} {
			jobs <- j
		}
		close(jobs)
	}()
	results := make(chan int, 3)
	run(`global (jobs, results)
seen := []
for i, job in jobs {
	seen += job
	results.send(i * 10)
}
```

### Gad's arrays and dicts in Go: copies, or the collection itself

A Go parameter of a slice or map type gets a **copy** of a gad array or dict —
`[]Object` and `map[string]Object` cannot share memory with a `[]int` or a
`map[string]int` —, **synchronized back** when the Go function returns, as
Go's semantics are: the items of a slice it changed are seen (its growth is
not: a callee does not grow its caller's slice), the entries of a map it
changed, added or deleted are. A Go function that keeps the copy to change it
later is not seen; a typed array or dict checks what is synchronized.

A parameter of type **`gad.ArrayOf[T]`** or **`gad.DictOf[K, V]`** (or a
pointer to one) gets the **gad collection itself**: what it changes, gad sees
at once. `T`, `K` and `V` are Go types — `int`, `uint`, `string`, a struct,
`gad.Object` —, each item converted when read and written; a typed array or
dict (`type ints []int`, `type scores dict[str, int]`) checks what is written,
as its type does. `Key` and `Value` are its gad types (`int` and the items'
type; `str` and `any` for a dict, a typed dict's `K` and `V`).

| `ArrayOf[T]` | `DictOf[K, V]` |
| ------------ | -------------- |
| `Len()`, `Get(i)`, `Set(i, v)`, `Values()` | `Len()`, `Get(k)` (`v, ok, err`), `Has(k)`, `Set(k, v)`, `Delete(k)`, `Keys()`, `Range(fn)` |
| `Append(vs...)`, `Delete(i)`: of a typed array, or of an array given by its pointer (`&a`, `&obj.list`) — one given by value cannot grow, an error | |
| `Object()`: the gad value | `Object()` |

```go
double := func(a *gad.ArrayOf[int]) error {
	for i := 0; i < a.Len(); i++ {
		v, err := a.Get(i)
		if err != nil {
			return err
		}
		if err = a.Set(i, v*2); err != nil {
			return err
		}
	}
	return nil
}
push := func(a gad.ArrayOf[int], v int) error { return a.Append(v) }
bump := func(d *gad.DictOf[string, uint]) error {
	n, _, err := d.Get("n")
	if err == nil {
		err = d.Set("n", n+1)
	}
	return err
}
```

```gad ignore
a := [1, 2]; double(a)        // a is [2, 4]
b := [1]; push(&b, 5)         // b is [1, 5]: given by its pointer, it grows
push(b, 6)                    // an error: an array given by value cannot grow
d := {n: 1u}; bump(d)         // d.n is 2u
type ints []int
x := ints(3); push(x, 4)      // a typed array grows (and checks: push(strs("a"), …) fails)
```

From Go, `gad.NewArrayOf[T](vm, o)` and `gad.NewDictOf[K, V](vm, o)` wrap a gad
value; `gad.ObjectOf(vm, v)` and `gad.ValueOf[T](vm, o)` convert one value.
`objects_go_collections_test.go` tests them.

Other hooks of a `ReflectStruct`: `FieldHandler` (what a field gives),
`FalbackIndexHandler` (a name it has no field for), and a `SetX(v)` method is
the setter of `x` when there is no field `x`.

## Documenting a module (`gad:doc`)

A module's documentation is in comments of its Go source, each starting
`// gad:doc`. The first is headed `# NAME module`, its prose following; the
sections are level-2 headings:

| Section | Its entries |
| ------- | ----------- |
| `## Functions` | `name(params) <ret>` — the signature line — then its description; consecutive lines of one name are overloads; `Example:` and a fenced `gad` block (its checks, `=`/`<<<`, verified) |
| `## Constants` | a member's name, its value read from the module |
| `## Types`, `## Classes`, `## Enums`, `## Methods`, `## Properties` | prose of the module's types and members |
| `## Converters`, `## Method Overrides` | the Go-level conversions and overrides |
| `## Declarations` | Gad declarations (classes, interfaces, enums, constants), each a fenced `gad` block, emitted as written into the API stub |

- A module may be documented over several files: each part headed by
  `# NAME module`, merged — a generated file's part first.
- `gad:samples [module,auto] PATH` links a file of examples (`example` blocks,
  one per member) merged into each member's doc; `auto` scaffolds the missing
  ones.
- `gaddoc DIR OUT.md NAME` renders the Markdown, `gaddoc api DIR OUT.gad NAME`
  the API stub — `export name(params) <ret> => nil` with their docs —, which
  `gad doc` renders and the IDE completes from. Both are `//go:generate` lines
  of the package (`gaddoc_generate.go` in gad's root).

## Source modules and dialects

A source module carries the dialect it must be compiled as. An importer returns
a `gad.SourceCode` pairing the bytes with a `gad.SourceKind`, and the compiler
parses each module with the matching front-end:

| `SourceKind`     | `String()` | `Ext()` | Parsed as          |
| ---------------- | ---------- | ------- | ------------------ |
| `SourceKindGad`  | `GAD`      | `.gad`  | plain Gad          |
| `SourceKindGadt` | `GADT`     | `.gadt` | mixed template     |
| `SourceKindGadx` | `GADX`     | `.gadx` | Gadx (indentation) |

`importers.FileImporter` derives the kind from the file extension
(`gad.SourceKindForExt`), so `.gadt` and `.gadx` modules import correctly
alongside `.gad`. In-memory modules pick their dialect explicitly; a bare
`[]byte` returned by an importer is compiled as `SourceKindGad` for backward
compatibility:

```go
opts.ModuleMap = gad.NewModuleMap().
    AddSourceModule("util", []byte(`export add = func(a, b) => a + b`)).
    AddSourceModuleKind("page", []byte(`{% export title = "Hi" %}`), gad.SourceKindGadt)
```

## Modules written in Gad, shipped in Go (`gad gomodule`)

A module written in Gad can be made a builtin module of a Go package: imported
by name (`import("shapes")`) like `strings` or `json`, already compiled, with
its API documented as gad's own builtin modules are.

```sh
gad gomodule --out shapes [--pkg NAME] [--module NAME] [--api FILE] [--import a,b] shapes.gad
```

writes into the package directory:

- `shapes.gadb` — the module compiled once (`encoder.CompileModuleWith`, with
  the default builtins and the builtin modules: a module that imports a file
  is refused, the host could not resolve it). Imported, it is linked into the
  program, not compiled again.
- `shapes.go` — the package: `ModuleName`, the bytecode (`//go:embed`),
  `Module()` (the importable, an `encoder.GoModule`), `AddTo(mm)`, `Imports`,
  the `Init` hook, a `//go:generate` line that makes the package again
  (`go generate`) and the module's docs as `gad:doc` comments: the file's
  leading `/** … **/`, each exported function's signature and doc
  (`## Functions`) and every other export — classes, interfaces, enums,
  constants — as written, its bodies `nil` (`## Declarations`).
- `shapes_test.go` — links the module in a program of the default builtins.
- the API stub (`--api`, default `shapes/shapes.gad`): what `gaddoc api` makes
  of the package's `gad:doc`, so `gad doc` renders it.

```go
mm := shapes.AddTo(helper.NewModuleMap())
```

**As an API.** The command is the package `github.com/gad-lang/gad/gomodule`,
in memory — no file system —:

```go
r, err := gomodule.Make(src, gomodule.Options{
	Module:     "shapes",
	SourceType: "gad", // as the formatter takes it: "gad", "gadTemplate", "gadx"
	Imports:    []string{"strings"},
	Files:      map[string][]byte{"init.go": initGo}, // the hand-written Go files
})
// r.Bytecode, r.Go, r.Test, r.API, r.Imports, r.Warnings; r.Files() by name
```

Its default module map is `gomodule.BuiltinModules()` — gad's builtin modules
that need no system (the command passes every one, `os` and `http` too).
`gomodule.MakeSource(src, sourceType, gomodule.Request{…})` is the same as JSON
says it, and gad's wasm installs it as `gadGomodule(source, sourceType,
requestJSON)`, beside `gadFormat(source, sourceType)`: it answers a
`gomodule.Response` (`{ok, files, bytecode (base64), imports, warnings, error}`).

**The Go init.** A hand-written file of the package (never overwritten) may set
`Init`: it runs once per import, in the VM that imports the module, **after** the
gad part. Its `*gad.ModuleInitContext` gives the VM (`VM()`), the module's spec
(`Spec()`), the gad part's exports (`Self()`, `Get(name)`) and `Import(name)` —
a module, as `import(name)` gives it: one the gad part imports, or one declared
(`--import`, or a `//gad:import NAME[,NAME]` line in a Go file of the package);
any other is a `NotImportableError`. What it returns is exported beside the gad
part's exports; a name of both is an error, unless the init replaces it
(`ctx.Replace(name, obj)`). Its objects are documented by `gad:doc` comments of
its own file (a `# shapes module` header, `## Declarations`), merged into the
API stub.

```go
//gad:import strings

func init() {
	Init = func(ctx *gad.ModuleInitContext) (gad.Dict, error) {
		base, err := ctx.Get("Shape") // a class of the gad part
		if err != nil {
			return nil, err
		}
		strs, err := ctx.Import("strings")
		if err != nil {
			return nil, err
		}
		cls := gad.NewClass("GoShape", ctx.Spec())
		cls.Extends(base.(*gad.Class), "")
		err = cls.AddGoMethod("label", func(c gad.Call) (gad.Object, error) {
			name, err := c.Args.Get(0).(gad.IndexGetter).IndexGet(c.VM, gad.Str("name"))
			if err != nil {
				return nil, err
			}
			return strs.Call(c.VM, "toUpper", name)
		})
		return gad.Dict{"GoShape": cls}, err
	}
}
```

The whole example: `samples/modules/gomodule/` (`shapes.gad`, the package
`shapes/` and its `init.go`). A program compiled with such a module keeps its
Go init only in the process: it is not written into a `.gadb` of the program.

### Transpiled to Go (`gad gomodule --go`)

*Planned — [#24](https://github.com/gad-lang/gad/issues/24).* The same package
as `gad gomodule`'s, its code Go instead of an embedded `.gadb`: no dispatch
loop, typed Go where the types are known. What is decided:

- **Typed from the first version**: a local or a parameter of a known type
  (`int`, `float`, `str`, `bool`, `[]T` bound) is held as the Go type, its
  arithmetic Go's; values are boxed (`gad.Object`) at the boundaries — calls
  into the runtime, returns, captures.
- **Errors are returned** (`error`), the Go convention: `throw` returns one,
  `try`/`catch` tests it.
- **Builtins through the VM** (`vm.CallBuiltin`): the overloads registered on
  them apply, as above.
- The runtime of gad stays the semantics: a transpiled module behaves as the
  bytecode one (a differential test of the samples), its Go stack traces point
  to the Gad lines (`//line`).
- **Classes, contracts, interfaces are built by the runtime** at the package's
  init, as the bytecode builds them (`gad.NewClass`, its fields and methods —
  the methods Go functions): `:>`, `::`, inheritance, mixins and a Go init's
  class (`GoShape`) work as they do. A Go struct is already a gad value as it
  is (`ReflectValue`, above): no struct is generated for a class.
- **Every exported function is a Go function of the package too**: `export
  area(w int, h int) <int> => w * h` (or `export func area { … }`, its
  overloads) is `func Area(…)` — typed by its signature, `(T, error)`, the VM
  its first parameter when its body needs one (a call of gad, a builtin):
  `func Area(vm *gad.VM, w, h int64) (int64, error)`. A Go module calls it
  directly, without a script.

