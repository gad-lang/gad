
# Ranges (`..`)

`from .. to` builds an inclusive, iterable `Range` (sugar for the `Range(from,
to)` builtin) over the numeric kinds, `char` and the temporal types; it runs
ascending or descending depending on its bounds. Part of the
[Operators](user_operators.gad) chapter.

```gad
[
    collect(1 .. 5),        // ascending
    collect(5 .. 1),        // descending
    collect('a' .. 'e'),    // chars
]
// => [[1, 2, 3, 4, 5], [5, 4, 3, 2, 1], ['a', 'b', 'c', 'd', 'e']]
```

## Step

The step is set with `/` (note `..` binds tighter, so `1 .. 10 / 2` is
`(1 .. 10) / 2`), with the `step` named argument of the `Range` constructor, or
with the `r.step(n)` method. `collect` gathers an iterable's values into an
array.

```gad
stepped := (1 .. 100).step(25)
[
    collect(1 .. 10 / 2),               // `..` binds tighter than `/`
    collect(Range(0, 10; step=3)),      // the named `step`
    [stepped.from, stepped.to, stepped.step()],
]
// => [[1, 3, 5, 7, 9], [0, 3, 6, 9], [1, 100, 25]]
```

## Temporal ranges

A range of dates steps by a duration — one day by default.

```gad
days := []
for d in 2026-01-30D .. 2026-02-02D {           // one day at a time
    days += str(d)
}
everyOther := []
for d in 2026-01-30D .. 2026-02-05D / (dur 48h) {
    everyOther += str(d)
}
[days, everyOther]
// => [["2026-01-30", "2026-01-31", "2026-02-01", "2026-02-02"], ["2026-01-30", "2026-02-01", "2026-02-03", "2026-02-05"]]
```

## Example — `ranges.gad`

```gad
[
    collect(1 .. 5),        // ascending
    collect(5 .. 1),        // descending
    collect('a' .. 'e'),    // chars
]

stepped := (1 .. 100).step(25)
[
    collect(1 .. 10 / 2),               // `..` binds tighter than `/`
    collect(Range(0, 10; step=3)),      // the named `step`
    [stepped.from, stepped.to, stepped.step()],
]

days := []
for d in 2026-01-30D .. 2026-02-02D {           // one day at a time
    days += str(d)
}
everyOther := []
for d in 2026-01-30D .. 2026-02-05D / (dur 48h) {
    everyOther += str(d)
}
[days, everyOther]
```
