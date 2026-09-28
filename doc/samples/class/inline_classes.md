
# Anonymous classes as field types (`a class { … }`)

A class field may be typed by a class declared right where the field is:
`a class { … }`. It is a record of its own inside the record — a settings
object reads as the tree it is, with no name to come up with for each level.
It nests to any depth, and behaves exactly as a field typed by a named class
([typed fields](field_types.gad)):

- a **dict** given for the field is constructed into an instance of the class,
  recursively;
- its fields keep their **types**, **defaults** and `[k=v]` **metadata**;
- a field not given is **nil** (the class's defaults apply once a dict is
  given);
- a value of the wrong type is **rejected**, naming the field.

## Names

An anonymous class is named by its **path from the outermost class**, that
class's name first: `Options.page`, `Options.page.header`. The names are given
by the compiler; they are not written back by the formatter, which writes the
class as it was declared. An outermost class that is itself anonymous has a
generated name, `#N`, as an anonymous function does.

```gad
class Options {
    page class {
        header class {
            title str = "Home"
            sticky bool = false
        }
    }
    theme str = "light"
}
o := Options(; page = {header: {sticky: true}})
[
    typeName(o.page),
    typeName(o.page.header),
    o.page.header.title,     // the default, the dict did not give it
    o.page.header.sticky,
    o.theme,
]
// => ["Options.page", "Options.page.header", "Home", true, "light"]
```

## Defaults, metadata and types

```gad
[
    Options().page,                                  // not given: nil
    Options(; page = {}).page.header,                // an inner class not given: nil
    Options(; page = {header: {}}).page.header.title, // given empty: its defaults
]
// => [nil, nil, "Home"]
```

## Rejection

```gad
rejects := func(f) { try { f(); return false } catch { return true } }
[
    rejects(() => Options(; page = {header: {title: 1}})),  // field "title" expects str
    rejects(() => Options(; page = {header: {title: "Hi"}})),
]
// => [true, false]
```

## An anonymous outermost class

```gad
Settings := class { db class { host str = "localhost"; port int = 5432 } }
s := Settings(; db = {port: 5445})
[strings.hasPrefix(typeName(s.db), "#"), strings.hasSuffix(typeName(s.db), ".db"), s.db.host, s.db.port]
// => [true, true, "localhost", 5445]
```

## Example — `inline_classes.gad`

```gad
class Options {
    page class {
        header class {
            title str = "Home"
            sticky bool = false
        }
    }
    theme str = "light"
}
o := Options(; page = {header: {sticky: true}})
[
    typeName(o.page),
    typeName(o.page.header),
    o.page.header.title,     // the default, the dict did not give it
    o.page.header.sticky,
    o.theme,
]

[
    Options().page,                                  // not given: nil
    Options(; page = {}).page.header,                // an inner class not given: nil
    Options(; page = {header: {}}).page.header.title, // given empty: its defaults
]

rejects := func(f) { try { f(); return false } catch { return true } }
[
    rejects(() => Options(; page = {header: {title: 1}})),  // field "title" expects str
    rejects(() => Options(; page = {header: {title: "Hi"}})),
]

Settings := class { db class { host str = "localhost"; port int = 5432 } }
s := Settings(; db = {port: 5445})
[strings.hasPrefix(typeName(s.db), "#"), strings.hasSuffix(typeName(s.db), ".db"), s.db.host, s.db.port]
```
