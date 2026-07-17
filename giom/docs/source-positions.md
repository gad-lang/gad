# Source Positions

Giom preserves the position (file, line, column) of the original `.giom` source
through parsing and compilation, so runtime error stack traces, compile errors,
and AST node positions point at the real template location.

## Why it matters

A giom template is transpiled to Gad and then compiled and executed. When a
template hits a runtime error — for example calling a value that is `nil` — the
Gad `RuntimeError` carries a stack trace resolved against the source file:

```
NotCallableError: nil
    at layouts/index.giom:10:5
       layouts/default.giom:50:32
```

Without correct positions every frame collapses to `line 1`, making template
errors nearly impossible to locate.

## How it works

Two things are required for a position to resolve correctly:

1. **The source file's line table must be populated.** Gad resolves a byte
   offset to a line/column using `source.File.Lines`. The giom scanner reads the
   source through its own buffered reader and never advances Gad's
   `source.Reader` (which is what normally records line starts), so the scanner
   registers every line start up front when it is created.

2. **Embedded Gad fragments must be parsed in the source coordinate space.**
   Giom extracts Gad code from directives and parses it with Gad's parser:

   - single-line code — `~ expr`
   - multi-line code — `~~ … ~~`
   - interpolations — `{= expr }` / `{ expr }`
   - directive expressions — `@if`, `@for`, `@match`, attribute values, comp
     call arguments

   Each fragment is a verbatim slice of the original source, so it is parsed
   with the fragment file's base offset set to the fragment's absolute position.
   Gad then assigns every node a position of `base + localOffset`, i.e. directly
   in the original file's coordinate space. A constant wrapper prefix such as the
   `return ` used to coerce expression parsing only shifts the base uniformly and
   is compensated automatically.

Multi-line `~~` blocks keep their original indentation when parsed. Leading
whitespace is insignificant to Gad, so preserving it keeps byte offsets — and
therefore line and column numbers — faithful to the source.

## Guarantees and limits

- Line numbers are accurate for all executable constructs listed above.
- Column numbers are accurate for verbatim fragments (code, interpolation).
  For directive expressions whose extracted value strips a leading keyword
  (e.g. `@if `), the line is exact and the column may be offset by the directive
  prefix length.
- Fragments whose absolute position is unknown fall back to fragment-local
  positions rather than producing a wrong location.
