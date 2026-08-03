# `reflect` module

Raw, delegation-free index access — the functional analog of JavaScript
`Reflect.get` / `Reflect.set`.

Unlike the `target[key]` / `target[key] = value` operators, which
[delegate](properties.md#properties-as-container-members-computed-properties) to a
stored [property](properties.md)'s getter/setter, the `reflect`
functions operate on the **stored value verbatim**.

`reflect` is available directly (like `strings`/`fmt`), and also via
`import("reflect")`.

## Functions

`get(target indexGetter, key str|int) -> any`

Returns the value stored at `key` verbatim. If a `Prop` is stored there, the
`Prop` itself is returned — its getter is **not** run.

```go
var (v = 1, d = { x: prop { () => v; (val) { v = val } } })

d.x                   // 1        — the operator runs the getter
reflect.get(d, "x")   // prop {…}  — the Prop object itself
```

---

`set(target indexSetter, key str|int, value any)`

Writes `value` at `key` verbatim, overwriting (and thus removing) any `Prop`
stored there **without** running its setter.

```go
reflect.set(d, "x", 3)   // d becomes { x: 3 } — the prop is replaced
d.x                      // 3
```

## When to use

- Introspect or serialize a container's raw contents (getters would otherwise
  run side effects or compute values).
- Replace a computed property with a plain value, or install/remove a `Prop`.
- Bypass a managed field's setter validation when you deliberately need the raw
  slot.
