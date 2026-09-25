
# Output and Standard Streams

Printing, formatting, capturing output and the standard streams. Part of the
[Strings, bytes & regex](strings_bytes_regex.gad) chapter; the `buffer` as a
`with` resource is in [With](with.gad).

## Printing and formatting

`print(…)` writes its arguments and `println(…)` adds spaces between them and a
trailing newline; both return the byte count. `sprintf(format, …)` returns a
formatted `str` and `printf(format, …)` writes it — Go's verbs: `%s`, `%d`,
`%5d`/`%05d`, `%.2f`, `%v` (any value), `%q` (quoted), `%x`, …

```gad
n := println("total:", 3, [1, 2])     // "total: 3 [1, 2]\n" — 16 bytes
printf("%-6s|%05d|%.2f\n", "id", 42, 3.14159)
[
    n,
    sprintf("%s=%d", "x", 7),
    sprintf("%v %q %x", [1], "hi", 255),
]
// => [16, "x=7", "[1] \"hi\" ff"]
```

Output:

```text
total: 3 [1, 2]
id    |00042|3.14
```

## Capturing output

`obstart()` starts capturing everything printed into a fresh `buffer` (and
returns it); `obend()` stops the most recent capture. Captures nest — output
goes to the innermost one. `with buffer() { … }` is the block-scoped form.

```gad
outer := obstart()
print("a")
inner := obstart()          // nested: output now goes here
print("b")
obend()
print("c")                  // back to the outer capture
obend()
viaWith := with buffer() { print("block") }
[str(outer), str(inner), str(viaWith)]
// => ["ac", "b", "block"]
```

## Standard streams

`STDIN`, `STDOUT` and `STDERR` are the running VM's standard streams — a
`reader` and two `writer`s (the host may redirect them). `stdio(which)` returns
the same streams by name (`"IN"`, `"OUT"`, `"ERR"`) or number (`0`, `1`, `2`).
`write(w, …)` writes each value to any writable — a stream or a `buffer` — and
returns the byte count; `flush(w)` pushes out buffered output and `close(o)`
closes a closable reader/writer.

```gad
buf := buffer()
written := write(buf, "héllo", 1)    // bytes, not chars
[
    [typeName(STDIN), typeName(STDOUT), typeName(STDERR)],
    stdio("OUT") == STDOUT, stdio(2) == STDERR,
    written, str(buf),
]
// => [["reader", "writer", "writer"], true, true, 7, "héllo1"]
```

## Example — `io.gad`

```gad
n := println("total:", 3, [1, 2])     // "total: 3 [1, 2]\n" — 16 bytes
printf("%-6s|%05d|%.2f\n", "id", 42, 3.14159)
[
    n,
    sprintf("%s=%d", "x", 7),
    sprintf("%v %q %x", [1], "hi", 255),
]
//= [16, "x=7", "[1] \"hi\" ff"]

outer := obstart()
print("a")
inner := obstart()          // nested: output now goes here
print("b")
obend()
print("c")                  // back to the outer capture
obend()
viaWith := with buffer() { print("block") }
[str(outer), str(inner), str(viaWith)]

buf := buffer()
written := write(buf, "héllo", 1)    // bytes, not chars
[
    [typeName(STDIN), typeName(STDOUT), typeName(STDERR)],
    stdio("OUT") == STDOUT, stdio(2) == STDERR,
    written, str(buf),
]
```
