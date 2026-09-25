
# Assignment and Append

Assignment lists, and the append operators for arrays and dicts.

## Assignment lists

A comma-separated right-hand side is an **array** when there is a single target
(`*x` splices an iterable into it); with several targets it **destructures**,
and a `*name` target collects the rest.

```gad
x := 1, 2, 3            // one target: the list is an array
rest := [3, 4]
y := 1, 2, *rest        // `*` splices an iterable in
a, b := 10, 20          // several targets destructure
head, *tail := 1, 2, 3, 4
[x, y, [a, b], head, tail]
// => [[1, 2, 3], [1, 2, 3, 4], [10, 20], 1, [2, 3, 4]]
```

## Array append

As expressions, `+` appends **one** element (an array on the right is
concatenated), while `++` **extends** with any iterable. The in-place forms are
`+=` (append one) and `++=` (extend); `++= a, b` is `++= [a, b]`.

```gad
exprs := [
    [1] + 2,            // append one
    [1] + [2, 3],       // + concatenates an array
    [1] ++ [2, 3],      // ++ extends with an iterable
]

acc := []
acc += 1                // append one element
acc ++= [2, 3]          // extend with an iterable
acc ++= 4, 5            // ++= a, b  ==  ++= [a, b]
[exprs, acc]
// => [[[1, 2], [1, 2, 3], [1, 2, 3]], [1, 2, 3, 4, 5]]
```

## Dict entries

Dicts follow the same rule: `+` takes **one entry** — a `keyValue(k, v)`, or a
dict, which reads as a sequence of entries and merges — and `++` spreads a
sequence of entries. `d ++ keyValue(…)` is an error: `++` always spreads, and
one entry is not a sequence of them.

```gad
d := {a: 1}
d += keyValue("b", 2)   // put one entry
d += {c: 3}             // a dict reads as a sequence of entries, and merges
d ++= {e: 5, f: 6}      // extend with a sequence
d
// => {a: 1, b: 2, c: 3, e: 5, f: 6}
```

## Example — `assign_and_append.gad`

```gad
x := 1, 2, 3            // one target: the list is an array
rest := [3, 4]
y := 1, 2, *rest        // `*` splices an iterable in
a, b := 10, 20          // several targets destructure
head, *tail := 1, 2, 3, 4
[x, y, [a, b], head, tail]

exprs := [
    [1] + 2,            // append one
    [1] + [2, 3],       // + concatenates an array
    [1] ++ [2, 3],      // ++ extends with an iterable
]

acc := []
acc += 1                // append one element
acc ++= [2, 3]          // extend with an iterable
acc ++= 4, 5            // ++= a, b  ==  ++= [a, b]
[exprs, acc]

d := {a: 1}
d += keyValue("b", 2)   // put one entry
d += {c: 3}             // a dict reads as a sequence of entries, and merges
d ++= {e: 5, f: 6}      // extend with a sequence
d

return acc == [1, 2, 3, 4, 5] && d == {a: 1, b: 2, c: 3, e: 5, f: 6}
```
