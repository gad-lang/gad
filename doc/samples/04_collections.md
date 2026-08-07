
# Collections

Gad's composite types are **arrays** (ordered lists) and **dicts** (string-keyed
maps). Comprehensions live in [05_comprehensions.gad](05_comprehensions.gad),
keyValue arrays in [22_key_value_array.gad](22_key_value_array.gad), and
destructuring in [27_destructuring.gad](27_destructuring.gad).

## Arrays

Indexable and mutable; `len` reports the length, `+=` concatenates, and slicing
returns a sub-array (negative indices count from the end).

```gad
a := [1, 2, 3]
a[1] = 20      // mutate in place
a = a + [4, 5] // concatenate (`a += x` appends x as a single element)
[a, len(a), a[0], a[-1], a[1:3], a[:-1]]
// => [[1, 20, 3, 4, 5], 5, 1, 5, [20, 3], [1, 20, 3, 4]]
```

## Dicts

A dict maps string keys to values (items separated by commas or newlines). Read
with `.name` or `["name"]`, add/remove keys, and iterate key/value pairs. Nested
dicts may use block-style nesting (the outer parentheses are required because a
bare `{` at statement position starts a block).

```gad
m := {a: 1, "b": 2, c: 3}
m.d = 4     // add a key
delete m.a  // remove a key (delete is a statement)
config := ({
    server {
        host: "localhost"
        port: 8080
    }
})
[m.b, m["d"], config.server.port]
// => [2, 4, 8080]
```

### Keyword keys

A keyword (`class`, `if`, `else`, `func`, `false`, `nil`, …) may be used as a
**bare name** in any key position — it is taken as the string of its spelling.
This holds for `.name`, a dict key, `[name=value]` and `(;name=value)`. Only the
key position is affected; on the value side keywords keep their meaning.

```gad
mk := {class: 1, else: 2, func: 3} // keywords as bare string keys
[mk.class, mk["else"], mk.func]    // read via selector and index
// => [1, 2, 3]
```

## Spread and merge literals

`*expr` inside an array literal splices its elements in place (a leading spread
makes a fresh copy); inside a dict literal it merges another dict's entries, with
later entries winning.

```gad
sa := [1, 2]
sb := [3, 4]
arr := [0, *sa, *sb, 5]              // splice
dd := {a: 1, *{a: 9, b: 2}, c: 3}    // merge (later wins)
[arr, dd]
// => [[0, 1, 2, 3, 4, 5], {a: 9, b: 2, c: 3}]
```

## Iteration

`for … in` yields (index, value) for arrays and (key, value) for dicts.

```gad
nums := [10, 20, 30]
squares := []
for _, n in nums { // arrays: (index, value)
    squares += n * n
}
person := {name: "Ada", age: 36}
for k, v in person { // dicts: (key, value)
    println(#"  {k} -> {v}")
}
squares
// => [100, 400, 900]
```

## Example — `04_collections.gad`

```gad
a := [1, 2, 3]
a[1] = 20      // mutate in place
a = a + [4, 5] // concatenate (`a += x` appends x as a single element)
[a, len(a), a[0], a[-1], a[1:3], a[:-1]]

m := {a: 1, "b": 2, c: 3}
m.d = 4     // add a key
delete m.a  // remove a key (delete is a statement)
config := ({
    server {
        host: "localhost"
        port: 8080
    }
})
[m.b, m["d"], config.server.port]

mk := {class: 1, else: 2, func: 3} // keywords as bare string keys
[mk.class, mk["else"], mk.func]    // read via selector and index

sa := [1, 2]
sb := [3, 4]
arr := [0, *sa, *sb, 5]              // splice
dd := {a: 1, *{a: 9, b: 2}, c: 3}    // merge (later wins)
[arr, dd]

nums := [10, 20, 30]
squares := []
for _, n in nums { // arrays: (index, value)
    squares += n * n
}
person := {name: "Ada", age: 36}
for k, v in person { // dicts: (key, value)
    println(#"  {k} -> {v}")
}
squares

return [1, 20, 3, 4, 5]
```
