# Metaprogramming: parse, eval and Gadx

Gad can parse Gad (and Gadx) source into an AST value at run time, inspect or
transform it, and compile-and-run it — all from within a script, through the
`gad` builtin namespace. It can also compile Gadx (`.gadx`) templates natively:
Gadx's front-end is built into the Gad compiler, so `.gadx` files run, debug and
import like any other module.

## The `gad` namespace

| Member                  | Kind     | Purpose                                             |
|-------------------------|----------|-----------------------------------------------------|
| `gad.parse`             | function | Parse a source string into a `SourceFileObject`.    |
| `gad.parseFile`         | function | Read and parse a file into a `SourceFileObject`.    |
| `gad.eval`              | function | Compile and run a source / `SourceFileObject` / `StmtsObject` / `StmtObject`. |
| `gad.SourceType`        | enum     | `GAD`, `TEMPLATE`, `GADX` — selects the parse mode. |
| `gad.SourceFileObject`  | type     | A parsed source file (raw bytes + `path`/`type`).   |
| `gad.StmtsObject`       | type     | A sequence of parsed statements (an AST fragment).  |
| `gad.StmtObject`        | type     | A single statement (element of `StmtsObject`).      |

## `gad.parse`

```
gad.parse(script str; type gad.SourceType = gad.SourceType.GAD, name str = "")
    -> sourceFile gad.SourceFileObject
```

Parses `script` and returns the parsed source file; its statements are available
as `sourceFile.stmts`.

The parse mode is chosen as follows:

1. If `type` is given, it wins.
2. Otherwise, if `name` is given, the mode is inferred from its extension
   (`.gadx` → Gadx, `.gadt` → template, else Gad).
3. Otherwise the mode is `GAD`.

The returned source file's `path` is `name` when given, otherwise a generated
name carrying the selected mode's extension (`.gad` / `.gadt` / `.gadx`).

```gad
src := gad.parse("return 6 * 7")
gad.eval(src)                       // 42 (eval accepts the source file directly)

// Template mode, inferred from the name extension:
src := gad.parse("Hi {%= name %}"; name = "greeting.gadt")
src.type                            // gad.SourceType.TEMPLATE

// Gadx mode, selected explicitly:
src := gad.parse("span Hi"; type = gad.SourceType.GADX)
```

## `gad.parseFile`

```
gad.parseFile(pth str) -> sourceFile gad.SourceFileObject
```

Reads the file at `pth` and parses it, selecting the mode from the file
extension. It errors when the file cannot be read or parsed.

```gad
src := gad.parseFile("page.gadx")   // Gadx, from the extension
src.path                            // "page.gadx"
src.stmts                           // the parsed statements
```

## `gad.eval`

`gad.eval` is a type-dispatched function with four overloads:

```
gad.eval(source str; type gad.SourceType = gad.SourceType.GAD) -> res
gad.eval(sourceFile gad.SourceFileObject)                      -> res
gad.eval(stmts gad.StmtsObject)                                -> res
gad.eval(stmt gad.StmtObject)                                  -> res
```

Each compiles the given source / statements against the running VM's builtins
and runs them like [`Eval.Run`](embedding.md), returning the last value produced.
The `source` overload parses first, in the selected mode.

> Statements parsed in Gadx mode reference the `gadx` builtins (`gadx.Tag`,
> `gadx.write`, …). `gad.eval` uses the running VM's builtins, so those must be
> registered in the VM (the `gad` CLI and `gad ide` register them
> automatically). A returned Gadx element is not rendered — write it (`write(el)`)
> to emit HTML.

```gad
gad.eval("return 6 * 7")            // 42 (parse + eval a source string)

src := gad.parse("a := 2\nreturn a * a")
gad.eval(src)                       // 4  (a SourceFileObject)
gad.eval(src.stmts)                 // 4  (its StmtsObject)
gad.eval(src.stmts[0])              // nil (a single StmtObject)
```

## `gad.SourceFileObject`

A parsed source file. It behaves like its raw source bytes and carries two
attributes:

| Access               | Result                                             |
|----------------------|----------------------------------------------------|
| `src[i]`             | the byte at offset `i` as a `char`                 |
| `src[a:b]`           | the byte range as `bytes`                          |
| `bytes(src)`         | the whole source as `bytes`                        |
| `len(src)`           | the source length in bytes                         |
| `src.path`           | the file path / generated name (`str`)             |
| `src.type`           | the `gad.SourceType` member for the parse mode     |
| `src.stmts`          | the parsed statements (`gad.StmtsObject`)          |

```gad
src := gad.parse("abc"; name = "x.gad")
src.path        // "x.gad"
src.type        // gad.SourceType.GAD
src[0]          // 'a'
src[0:2]        // bytes "ab"
bytes(src)      // bytes "abc"
src.stmts       // its StmtsObject
```

## `gad.StmtsObject` and `gad.StmtObject`

`gad.StmtsObject` is an ordered, indexable, iterable collection of statements.
Each element is a `gad.StmtObject`.

```gad
stmts := gad.parse("a := 1\nb := 2\nc := 3").stmts
len(stmts)                          // 3
str(stmts[0])                       // "a := 1"

for s in stmts {
    println(str(s))
}

// Replace a statement in place:
one := gad.parse("a := 99").stmts
stmts[0] = one[0]
```

## Native Gadx compilation

The `.gadx` template engine is compiled by Gad itself — no separate compiler is
involved:

- `gad page.gadx` runs a Gadx template; `gad debug page.gadx` debugs it.
- A plain `.gad` script may `import("./partial.gadx")`; the imported template is
  compiled with the Gadx front-end automatically.
- Embedding: set `GadxOptions` on `gad.CompileOptions` to compile a `.gadx`
  source, and register the Gadx builtins in the VM.

```go
import (
    gad "github.com/gad-lang/gad"
    "github.com/gad-lang/gad/gadx"
)

builtins := gadx.AppendBuiltins(gad.NewBuiltins())
st := gad.NewSymbolTable(builtins.NameSet)

opts := gad.CompileOptions{}
opts.GadxOptions = &gad.GadxOptions{} // parse & lower as Gadx
_, bc, err := gad.Compile(st, []byte("p Hello {= name }"), opts)
```

The file importer (`github.com/gad-lang/gad/importers.FileImporter`) recognises
`.gadx` modules and compiles them natively, so `import("...")` of a `.gadx` file
works with no extra wiring. See the [Gadx documentation](../gadx/docs) for the
template syntax.
