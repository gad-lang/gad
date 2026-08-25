# counter

counter.gad — a module exporting properties (managed field + live binding).

`export prop name { … }` exposes a getter/setter behind a plain module field:
member access on the imported module delegates to the prop.

`export prop name = init` is the concise live-binding form: it declares a
module-local `var name = init` and exports a read/write property over it, so the
field is a live binding shared with the module's own functions.

Imported by main.gad. See doc/modules.md and doc/properties.md.

## Public API

### count

```gad
count = prop count {() => value; (n) { value = ((n < 0) ? 0 : n) }; }
```

A managed counter: reads return the current value; writes clamp at 0.

### total

```gad
total = prop total {() => total; ($value) { total = $value }; }
```

A live binding: `total` is a real module var; both the module and importers
read/write the same value.

### inc

```gad
inc()
```

A plain exported function that bumps both, showing the shared state.

## Example — `counter.gad`

```gad
var value = 0

/// A managed counter: reads return the current value; writes clamp at 0.
export prop count {
    () => value                              // getter
    (n) { value = n < 0 ? 0 : n }            // setter (never goes negative)
}

/// A live binding: `total` is a real module var; both the module and importers
/// read/write the same value.
export prop total = 0

/// A plain exported function that bumps both, showing the shared state.
export inc() {
    value = value + 1
    total = total + 1
}
```
