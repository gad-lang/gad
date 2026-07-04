# `test` module and `gad test`

The `test` module and the `gad test` command bring Go-style testing to Gad:
write `*_test.gad` files with `test*` functions, run them with `gad test`, and
get pass/fail reports and benchmarks — much like `go test`.

- [Quick start](#quick-start)
- [Writing tests](#writing-tests)
- [The test context `t`](#the-test-context-t)
- [require-style helpers](#require-style-helpers)
- [Benchmarks](#benchmarks)
- [The `gad test` command](#the-gad-test-command)

## Quick start

Create `math_test.gad`:

```gad
test := import("test")

func add(a, b) => a + b

func testAdd(t) {
	t.equal(3, add(1, 2))
	t.true(add(2, 2) == 4)
}
```

Run it:

```
$ gad test math_test.gad
test: 1 passed, 0 failed, 0 skipped
```

A full runnable example is in [`samples/testing`](../samples/testing/math_test.gad):

```
$ gad test -v -bench=. samples/testing
PASS samples/testing/math_test.gad/testAdd
PASS samples/testing/math_test.gad/testHelpers
PASS samples/testing/math_test.gad/testFib
SKIP samples/testing/math_test.gad/testNotReady: pending feature
PASS samples/testing/math_test.gad/addCommutes
PASS samples/testing/math_test.gad/fib of ten is 55
BENCH samples/testing/math_test.gad/benchFib	60843	18084.0 ns/op
BENCH samples/testing/math_test.gad/fib 15	99582	19924.6 ns/op

test: 5 passed, 0 failed, 1 skipped
```

## Writing tests

`gad test` discovers files whose name ends in `_test.gad`. In each file, every
**top-level function** is inspected:

- a name starting with `test` (case-insensitive, e.g. `testAdd`, `TestAdd`) is a
  **test**;
- a name starting with `bench` (e.g. `benchFib`) is a **benchmark**;
- everything else (helpers, imported modules) is left alone.

A test function takes a single parameter, the test context `t`:

```gad
func testSomething(t) {
	// assertions on t …
}
```

Assertions are **require-style**: the first failing assertion records a message
and aborts that test (the rest of the function does not run). Other tests in the
file still run. A test with no failure passes.

### Statement form: `test NAME { … }` and `bench NAME { … }`

The `test` / `bench` **statements** are shorthand for the functions above: `t` is
available directly (no parameter to write), and NAME may be a string, so a test
can carry a spaces-and-all description. A `///` doc comment is allowed.

```gad
/// add is commutative
test addCommutes {
	t.equal(add(2, 3), add(3, 2))
}

test "fib of ten is 55" {
	t.equal(55, fib(10))
}

bench "fib 15" {
	for i := 0; i < t.n; i++ { fib(15) }
}
```

`test` and `bench` are **contextual keywords**: they introduce a statement only
when followed by a NAME and `{`, so they stay ordinary identifiers everywhere
else — `test := import("test")`, `test.equal(t, …)` and `bench()` all keep
working. The two forms may be freely mixed in one file and run in source order.

## The test context `t`

`t` carries the assertions and controls. Assertion methods fail-and-abort on
mismatch; every one accepts a trailing named `msg` argument that prefixes the
failure text.

| Method | Passes when |
|--------|-------------|
| `t.equal(a, b)` | `a == b` |
| `t.notEqual(a, b)` | `a != b` |
| `t.true(x)` | `x` is truthy |
| `t.false(x)` | `x` is falsy |
| `t.nil(x)` | `x` is `nil` |
| `t.notNil(x)` | `x` is not `nil` |
| `t.contains(s, sub)` | string `s` contains `sub` |
| `t.error(fn)` | calling `fn()` returns an error |
| `t.noError(fn)` | calling `fn()` returns no error |

Controls:

| Method | Effect |
|--------|--------|
| `t.log(args…)` | record a message (shown under the test with `-v`) |
| `t.fail(msg…)` | record a failure and abort |
| `t.fatal(msg…)` | alias of `t.fail` |
| `t.skip(msg…)` | stop the test and mark it skipped (not failed) |
| `t.name()` | the test's name |
| `t.failed()` | whether a failure was recorded so far |

Read-only fields via indexing: `t.name`, `t.failed`, and `t.n` (benchmark count).

The named `msg` argument (note the `;` that begins named arguments in Gad):

```gad
func testFib(t) {
	t.equal(55, fib(10); msg="fib(10) should be 55")
}
```

## require-style helpers

The module mirrors Go's testify/require: the same assertions as free functions
that take `t` first. `test.equal(t, a, b)` is exactly `t.equal(a, b)`.

```gad
test := import("test")

func testHelpers(t) {
	test.equal(t, 5, add(2, 3))
	test.contains(t, "gad language", "language")
	test.noError(t, () => add(1, 1))
}
```

Available: `test.equal`, `test.notEqual`, `test.true`, `test.false`, `test.nil`,
`test.notNil`, `test.contains`, `test.error`, `test.noError`, `test.fail`,
`test.fatal`. `test.T` is the context type.

## Benchmarks

A benchmark function loops `t.n` times. The runner scales `t.n` up until the run
lasts at least `-benchtime` (default 1s), then reports the iteration count and
nanoseconds per iteration:

```gad
func benchFib(t) {
	for i := 0; i < t.n; i++ {
		fib(15)
	}
}
```

Benchmarks run only when a `-bench` pattern is given (`-bench=.` runs all).

## The `gad test` command

```
gad test [flags] [PATH...]
```

`PATH` may be a file or a directory; write `DIR/...` to recurse. With no `PATH`
the current directory is scanned recursively. The command exits non-zero if any
test fails.

| Flag | Meaning |
|------|---------|
| `-v` | verbose: log every test (and its `t.log` output), not only failures |
| `-run REGEX` | run only tests whose name matches `REGEX` |
| `-bench REGEX` | run benchmarks whose name matches `REGEX` (e.g. `.` for all) |
| `-benchtime DUR` | minimum run time per benchmark (default `1s`) |
| `-timeout DUR` | per-file timeout (`0` = none) |

Examples:

```
gad test                       # every *_test.gad under the current dir
gad test ./mypkg/...           # recurse into mypkg
gad test -run Add math_test.gad
gad test -bench=. -benchtime=200ms ./...
```
