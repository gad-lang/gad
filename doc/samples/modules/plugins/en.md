# en

en.gad — an English greeting plugin, loaded by main.gad's glob import
`import("./plugins/*.gad")`.

## Public API

### lang

```gad
lang = "en"
```

The plugin's language code.

### greet

```gad
greet(name)
```

Greet name in English.

## Example — `en.gad`

```gad
param (; punct = "!")

/// The plugin's language code.
export lang = "en"

/// Greet name in English.
export greet(name) => "Hello, " + name + punct
```
