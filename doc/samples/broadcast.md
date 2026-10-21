
# Item by item (`.op`) and the pipe (`|>`)

A `.` right before an operator applies it **item by item** — Julia's broadcast —
and yields an array of the results: `[1, 2] .+ 2` is `[3, 4]`. Every binary
operator that has a meaning item by item takes the dot: arithmetic
`.+ .- .* ./ .% .**`, comparison `.== .!= .< .<= .> .>= .=== .!==`, bitwise
`.& .^ .&^ .<< .>>` and the user operators `.<<< .>>> .%%`; called,
`a.op(b; nested)` takes options. Part of the
[Operators](user_operators.gad) chapter.

## The operands

Two arrays (typed ones too) pair their items — they must have the same length;
an array and a value apply the value to every item; two values are the plain
operator (`1 .+ 2` is `3`). The result is an array, a typed array's too. The
item operator is the ordinary one: the user's overloads (`met gad.binOpMul`)
apply to every item, and `.==` compares as `==`.

```gad
a := [1, 2, 3]
[a .+ 1, a .* [10, 100, 1000], 10 .- a, a .< 2, a .== [1, 0, 3], a .<< 1]
= [[2, 3, 4], [10, 200, 3000], [9, 8, 7], [true, false, false], [true, false, true], [2, 4, 6]]
["a", "b"] .+ "!"
= ["a!", "b!"]
```

Each operator keeps its precedence: `a .+ b * c` is `a .+ (b * c)`, and
`a .+ 2 == b` compares the array that `.+` made. A dot spaced from the
operator is no broadcast (`a. + b` is an error), and after a number a space is
needed: `1.+2` is the float `1.` plus `2`.

## Nested arrays and the options

One level, as Julia: an item that is an array takes the operator as a whole —
`[[1, 2]] .< [[1, 3]]` compares the two arrays (`[true]`).

The call form `a.op(b; options)` — a `(` right after the operator — is
postfix, as a call (`a.+(1) .* 2` is `(a .+ 1) .* 2`; spaced, `a .+ (1) * 2`
is `a .+ ((1) * 2)`), and passes its named arguments to the operator's
handler. Without them, `a.op(b)` is `a .op b`. An array's handler takes
`nested`: it descends into the nested arrays, broadcasting at every level.
The options are values: `nested=cond`, `**opts`; an option the handler does
not know is an error.

```gad
[[[1, 2]] .< [[1, 3]], [1, [2, [3]]].*(2; nested), [1, [2]].+([10, 20]; nested)]
= [[true], [2, [4, [6]]], [11, [22]]]
deep := (; nested)
[[1, 2].+(1) .* 2, [1, [2]].+(1; nested=len(deep) > 0), [1, [2]].+(1; **deep)]
= [[4, 6], [2, [3]], [2, [3]]]
```

A type of Go handles `.op` with its options by the method
`BinOp{Op}Options(vm, right, options gad.BinOpOptions)` — the interface
`ObjectWith{Op}BinOperatorOptions` (see [Go modules](../go-modules.md)).

## The pipe `|>`

`x |> f(args)` calls `f(x, args)` (`x |> f` is `f(x)`): a chain reads in the
order it runs. It binds tighter than the arithmetic operators — parenthesize
an operation to pipe its result. `.|`, its older spelling, is still read.

```gad
["b", "c", "a"] |> sort
= ["a", "b", "c"]
([1, 2, 3] .* 2) |> str
= "[2, 4, 6]"
```

## Example — `broadcast.gad`

```gad
example operands {
    a := [1, 2, 3]
    [a .+ 1, a .* [10, 100, 1000], 10 .- a, a .< 2, a .== [1, 0, 3], a .<< 1]
    = [[2, 3, 4], [10, 200, 3000], [9, 8, 7], [true, false, false], [true, false, true], [2, 4, 6]]
    ["a", "b"] .+ "!"
    = ["a!", "b!"]
}

example nested {
    [[[1, 2]] .< [[1, 3]], [1, [2, [3]]].*(2; nested), [1, [2]].+([10, 20]; nested)]
    = [[true], [2, [4, [6]]], [11, [22]]]
    deep := (; nested)
    [[1, 2].+(1) .* 2, [1, [2]].+(1; nested=len(deep) > 0), [1, [2]].+(1; **deep)]
    = [[4, 6], [2, [3]], [2, [3]]]
}

example pipe {
    ["b", "c", "a"] |> sort
    = ["a", "b", "c"]
    ([1, 2, 3] .* 2) |> str
    = "[2, 4, 6]"
}
```
