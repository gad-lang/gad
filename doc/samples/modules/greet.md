# greet

greet.gad — a parameterised module. Import-time params are passed as
import("./greet.gad"; lang="pt").
See doc/modules.md for detailed documentation.

## Public API

### greet

```gad
greet(name)
```

### language

```gad
language = lang
```

## Example — `greet.gad`

```gad
param (;lang="en")

hello := match lang {
    "pt": "Olá"
    "es": "Hola"
    else: "Hello"
}

export greet(name) => #"{hello}, {name}!"
export language = lang
```
