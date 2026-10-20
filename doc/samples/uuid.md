
# UUIDs (`uuid`)

`uuid` is a type of its own — a builtin, with no namespace, as `str` is —:
a universally unique identifier, the 16 bytes of RFC 9562. Its text is the
canonical lower-case `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`.

- **`uuid()`** is a new random UUID (version 4).
- **`uuid("…")`** reads a text: canonical in any case, its 32 hex digits
  alone, in braces (`{…}`) or after `urn:uuid:`. Anything else is refused.
- **`uuid(bytes)`** takes its 16 bytes; `u.bytes` gives them back.
- **`u.version`** is the version its bits say: 4 random, 7 ordered by time.
- A UUID **equals** another of the same bytes, and the text of it.
- The **nil UUID** — every bit 0 — is falsy.
- It types parameters and fields, as any type does: `func(id uuid)`,
  `class C { id? uuid }`.

```gad
u := uuid("01A0A168-6707-748F-B487-C0AB78E25C11")
[
    str(u),                                        // canonical, lower case
    u.version,                                     // 7: ordered by time
    uuid().version,                                // a new one: 4, random
    u == "01a0a168-6707-748f-b487-c0ab78e25c11",   // equal to its text
    uuid(u.bytes) == u,                            // its 16 bytes, and back
    bool(uuid("00000000000000000000000000000000")), // the nil UUID: falsy
]
= ["01a0a168-6707-748f-b487-c0ab78e25c11", 7, 4, true, true, false]
```

## As a type

```gad
class Record { id uuid; name str }
r := Record(; id = uuid("01a0a1686707748fb487c0ab78e25c11"), name = "a")
refused := func() { try { Record(; id = "not one", name = "b"); return false } catch { return true } }()
[typeName(r.id), str(r.id), refused]
= ["uuid", "01a0a168-6707-748f-b487-c0ab78e25c11", true]
```

## A constant

`uuid("…")` — the builtin, a text literal alone — is a **constant**: the
compiler reads the text, once, and the bytecode holds the UUID (its 16
bytes); nothing is called as the code runs, and the same UUID written twice
is one constant. A text that is no UUID stays a call — its error the run's,
which a `try` catches, as ever. Any other call — `uuid()`, a text not literal (`uuid(s)`, an
interpolated one), bytes — runs as it is written, and so does a `uuid` of
the code's own (a variable, a parameter) or of the application's.

```gad
const ADMIN = uuid("01a0a168-6707-748f-b487-c0ab78e25c11")  // read when compiled
text := "01a0a168-6707-748f-b487-c0ab78e25c11"
[
    ADMIN == uuid(text),                               // the same as one read at run time
    uuid("01A0A1686707748FB487C0AB78E25C11") == ADMIN, // any form uuid("…") reads
    func(uuid) { return uuid("x") }(func(s) => s),     // a uuid of its own: a call
]
= [true, true, "x"]
```

## Example — `uuid.gad`

```gad
example uuid_basics {
    u := uuid("01A0A168-6707-748F-B487-C0AB78E25C11")
    [
        str(u),                                        // canonical, lower case
        u.version,                                     // 7: ordered by time
        uuid().version,                                // a new one: 4, random
        u == "01a0a168-6707-748f-b487-c0ab78e25c11",   // equal to its text
        uuid(u.bytes) == u,                            // its 16 bytes, and back
        bool(uuid("00000000000000000000000000000000")), // the nil UUID: falsy
    ]
    = ["01a0a168-6707-748f-b487-c0ab78e25c11", 7, 4, true, true, false]
}

example uuid_type {
    class Record { id uuid; name str }
    r := Record(; id = uuid("01a0a1686707748fb487c0ab78e25c11"), name = "a")
    refused := func() { try { Record(; id = "not one", name = "b"); return false } catch { return true } }()
    [typeName(r.id), str(r.id), refused]
    = ["uuid", "01a0a168-6707-748f-b487-c0ab78e25c11", true]
}

example uuid_constant {
    const ADMIN = uuid("01a0a168-6707-748f-b487-c0ab78e25c11")  // read when compiled
    text := "01a0a168-6707-748f-b487-c0ab78e25c11"
    [
        ADMIN == uuid(text),                               // the same as one read at run time
        uuid("01A0A1686707748FB487C0AB78E25C11") == ADMIN, // any form uuid("…") reads
        func(uuid) { return uuid("x") }(func(s) => s),     // a uuid of its own: a call
    ]
    = [true, true, "x"]
}
```
