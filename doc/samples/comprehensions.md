
# Comprehensions

Python-like comprehensions build a collection from an iterable. Part of the
[Collections](collections.gad) chapter.

## Array comprehensions

`[expr for x in iterable]` with an optional `if` filter; several `for` clauses
nest (the last varies fastest).

```gad
evens := [n for n in [1, 2, 3, 4, 5, 6] if n % 2 == 0]
squares := [n * n for n in [1, 2, 3, 4] if n > 1]
pairs := [i + j for i in [1, 2] for j in [10, 20]] // nested; last varies fastest
[evens, squares, pairs]
// => [[2, 4, 6], [4, 9, 16], [11, 21, 12, 22]]
```

## Dict comprehensions

`{key: value for x in iterable}` builds a dict. **Keys are static (literal) by
default**; wrap the key in `[ ]` to compute it. The special name `_` refers to
the dict being built, so it can accumulate across iterations.

```gad
lengths := {[w]: len(w) for w in ["a", "bb", "ccc"]} // [w] computes the key
// `_` accumulates into the dict being built
counts := {[v]: (_[v] ?? 0) + 1 for v in ["a", "b", "a", "a"]}
[lengths, counts]
// => [{a: 1, bb: 2, ccc: 3}, {a: 3, b: 1}]
```

## Multi-line comprehensions

The `for` clause (and any further `for` / `if` clauses) may start on its own
line after the element, so a long comprehension can span several lines. It reads
as a comprehension, not a multi-line array/dict literal, because `for` can never
be an element.

```gad
// the `for` and `if` clauses may each begin on their own line
labels := [
    "#" + str(n)
    for n in [1, 2, 3, 4]
    if n % 2 == 0]
// the same for a dict comprehension
squared := {
    [n]: n * n
    for n in [2, 3]}
[labels, squared]
// => [["#2", "#4"], {"2": 4, "3": 9}]
```

## Example — `comprehensions.gad`

```gad
evens := [n for n in [1, 2, 3, 4, 5, 6] if n % 2 == 0]
squares := [n * n for n in [1, 2, 3, 4] if n > 1]
pairs := [i + j for i in [1, 2] for j in [10, 20]] // nested; last varies fastest
[evens, squares, pairs]

lengths := {[w]: len(w) for w in ["a", "bb", "ccc"]} // [w] computes the key
// `_` accumulates into the dict being built
counts := {[v]: (_[v] ?? 0) + 1 for v in ["a", "b", "a", "a"]}
[lengths, counts]

// the `for` and `if` clauses may each begin on their own line
labels := [
    "#" + str(n)
    for n in [1, 2, 3, 4]
    if n % 2 == 0]
// the same for a dict comprehension
squared := {
    [n]: n * n
    for n in [2, 3]}
[labels, squared]

return [4, 9, 16]
```
