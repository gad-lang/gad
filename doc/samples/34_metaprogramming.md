
# Metaprogramming: parse, eval and Gadx

Gad can parse Gad (and Gadx) source into an AST value at run time, inspect or
transform it, and compile-and-run it — all from a script, through the `gad`
builtin namespace. It can also compile Gadx (`.gadx`) templates natively.

## The `gad` namespace

| Member                 | Purpose                                                |
|------------------------|--------------------------------------------------------|
| `gad.parse`            | parse a source string into a `SourceFileObject`         |
| `gad.parseFile`        | read and parse a file into a `SourceFileObject`         |
| `gad.eval`             | compile and run a source / source file / stmts / stmt   |
| `gad.SourceType`       | enum `GAD` / `TEMPLATE` / `GADX` — the parse mode       |
| `gad.SourceFileObject` | a parsed source file (raw bytes + `path` / `type`)      |
| `gad.StmtsObject`      | an ordered, indexable, iterable sequence of statements  |

## parse & eval

`gad.parse(script; type = gad.SourceType.GAD, name = "")` returns a
`SourceFileObject` whose statements are `src.stmts`. The mode is `type` if given,
else inferred from `name`'s extension, else `GAD`. `gad.eval` is type-dispatched:
it accepts a source string, a `SourceFileObject`, a `StmtsObject` or a single
`StmtObject`, compiles it against the running VM's builtins and returns the last
value.

```gad
fromString := gad.eval("return 6 * 7") // parse + eval a source string
src := gad.parse("a := 2\nreturn a * a")
[fromString, gad.eval(src), gad.eval(src.stmts)] // source file / its stmts
// => [42, 4, 4]
```

## Inspecting and transforming statements

A `SourceFileObject` behaves like its raw bytes (`src[i]`, `bytes(src)`,
`len(src)`) and carries `src.path`, `src.type` and `src.stmts`. A `StmtsObject`
is indexable and iterable, and its elements can be replaced in place.

```gad
stmts := gad.parse("a := 1\nb := 2\nc := 3").stmts
one := gad.parse("a := 99").stmts
stmts[0] = one[0] // replace a statement in place
[len(stmts), str(stmts[0]), str(stmts[1])]
// => [3, "a := 99", "b := 2"]
```

## Native Gadx compilation

The `.gadx` template engine is compiled by Gad itself — no separate compiler:
`gad page.gadx` runs a Gadx template, a `.gad` script may `import("./p.gadx")`,
and a Go host sets `GadxOptions` on `gad.CompileOptions` and registers the Gadx
builtins.

```
builtins := gadx.AppendBuiltins(gad.NewBuiltins())
opts := gad.CompileOptions{}
opts.GadxOptions = &gad.GadxOptions{} // parse & lower as Gadx
cr, err := gad.Compile(st, []byte("p Hello {= name }"), opts)
```

## Example — `34_metaprogramming.gad`

```gad
fromString := gad.eval("return 6 * 7") // parse + eval a source string
src := gad.parse("a := 2\nreturn a * a")
[fromString, gad.eval(src), gad.eval(src.stmts)] // source file / its stmts

stmts := gad.parse("a := 1\nb := 2\nc := 3").stmts
one := gad.parse("a := 99").stmts
stmts[0] = one[0] // replace a statement in place
[len(stmts), str(stmts[0]), str(stmts[1])]

return "metaprogramming"
```
