# pt

pt.gad — a Portuguese greeting plugin, loaded by main.gad's glob import
`import("./plugins/*.gad")`.

## Public API

### lang

```gad
lang = "pt"
```

The plugin's language code.

### greet

```gad
greet(name)
```

Greet name in Portuguese.

## Example — `pt.gad`

```gad
param (; punct = "!")

/// The plugin's language code.
export lang = "pt"

/// Greet name in Portuguese.
export greet(name) => "Olá, " + name + punct
```
