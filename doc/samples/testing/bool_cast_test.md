# bool_cast_test

bool_cast_test.gad — `expr ::: bool`, a fast truthiness conversion.

`bool(x)` converts any value to a boolean by *truthiness*: the falsy values
(`0`, `""`, an empty array/dict, `nil`, `false`, off/no flags) become `false`
and everything else becomes `true`. (Floats follow IEEE-754: only `NaN` is
falsy, so `0.0` is *truthy* — see `floatCases` below.)

`x ::: bool` produces the **same result** — but where `bool(x)` is a builtin
*function call*, `:::` to `bool` is a transforming cast the VM resolves to a
direct truthiness check (a single `IsFalsy` test, no call frame). So it is a
touch simpler to read and slightly faster on hot paths:

    x ::: bool   ==   bool(x)      // same value, for every x

Note the difference from the *checked* cast `::`: `5 :: bool` fails (an `int` is
not a `bool`), while `5 ::: bool` converts and yields `true`.

Run:
    gad test samples/testing
    gad test -v samples/testing            # log every test
    gad test -bench=. samples/testing      # compare `::: bool` vs `bool()`

Benchmark (100_000 conversions per op; representative, hardware-dependent):

| conversion   | time       | vs bool()     | allocations    |
|--------------|------------|---------------|----------------|
| `x ::: bool` |  9.8 ms/op | ~3.9x faster  |  99k allocs/op |
| `bool(x)`    | 38.6 ms/op | baseline      | 699k allocs/op |

`::: bool` wins on both axes because it skips the call: no frame, and no boxing
of the argument / NamedArgs the builtin path allocates. Same result, less work.

See samples/transform_cast_test.gad for the general `:::` transforming cast.

## Example — `bool_cast_test.gad`

```gad
// The two groups of values, one of each Gad "shape", used by the tests below.
truthyCases := [5, "x", [1], {a: 1}, true, 3.14, -1, 0.0]
falsyCases := [0, "", [], {}, nil, false]

// Floats are falsy only for NaN (IEEE-754), so 0.0 above sits in truthyCases.
nan := float("nan")
floatCases := [[0.0, true], [3.14, true], [nan, false]]

/**
For every value, `x ::: bool` equals `bool(x)` — full parity with the builtin.
**/
test "parity with bool()" {
	for v in truthyCases + falsyCases {
		t.equal(bool(v), (v ::: bool); msg=repr(v))
	}
}

/**
Truthy values cast to `true`.
**/
test "truthy -> true" {
	for v in truthyCases {
		t.true((v ::: bool); msg=repr(v))
	}
}

/**
Falsy values cast to `false`.
**/
test "falsy -> false" {
	for v in falsyCases {
		t.false((v ::: bool); msg=repr(v))
	}
}

/**
Floats follow IEEE-754: `0.0` is truthy; only `NaN` casts to `false`.
**/
test "float truthiness (only NaN is falsy)" {
	for c in floatCases {
		t.equal(c[1], (c[0] ::: bool); msg=repr(c[0]))
		t.equal(bool(c[0]), (c[0] ::: bool); msg=repr(c[0])) // still matches bool()
	}
}

/**
`:::` converts (it never raises), unlike the checked cast `::` which would
reject a non-bool. Here a plain `int` casts straight to a boolean.
**/
test "converts, does not check" {
	t.equal(true, 5 ::: bool)
	t.equal(false, 0 ::: bool)
}

// Benchmarks — the cast skips the builtin call, so it should edge out `bool()`.
// Run with `gad test -bench=. samples/testing` and compare the ns/op.

/**
`x ::: bool` — the direct truthiness cast (no function call).
**/
bench "::: bool" {
	acc := false
	for i := 0; i < t.n; i++ {
		acc = i ::: bool
	}
}

/**
`bool(x)` — the builtin, for comparison.
**/
bench "bool()" {
	acc := false
	for i := 0; i < t.n; i++ {
		acc = bool(i)
	}
}
```
