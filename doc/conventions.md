# Naming Conventions

[← Back to index](README.md)

These conventions describe how identifiers are named in Gad's builtins and
standard library, so that the API reads consistently. "Specific names" that are
established acronyms (e.g. `URL`, `ID`, `HTTP`) keep their conventional upper
casing, following the Go convention.

| Kind | Case | Examples |
|------|------|----------|
| **Primitive type names** | lowerCamelCase (never PascalCase) | `int`, `uint`, `float`, `str`, `rawStr`, `bytes`, `char`, `bool`, `time`, `date`, `duration` |
| **Other (non-primitive) type names** | PascalCase (or an upper acronym) | `Location` |
| **Constant names** | PascalCase (or an upper acronym) | `time.Hour`, `time.January`, `time.RFC3339`, `time.Type` |
| **Module names** | snake_case | `time`, `strings`, `fmt`, `encoding/base64` |
| **Function / method / property names** | lowerCamelCase (or an upper acronym) | `time.now`, `time.durationString`, `t.year()`, `t.unixNano()` |

## Notes

* A **primitive type** is a built-in value type (`int`, `str`, `time`, `date`,
  `duration`, …); its name is always lowercase. A non-primitive wrapper type
  such as `Location` is PascalCase.
* **Constants** are PascalCase even inside a module whose functions are
  lowerCamelCase — e.g. the `time` module exposes `time.now()` (function) and
  `time.Hour` / `time.RFC3339` (constants).
* **Acronyms** keep their conventional casing as a unit: `URL`, not `Url`;
  `RFC3339`, not `Rfc3339`.
