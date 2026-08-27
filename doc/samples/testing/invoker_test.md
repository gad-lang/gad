# invoker_test

invoker_test.gad — gad.invoker, a repeatable, pre-resolved call.

    gad.invoker(fn, args array; **nargs) <ret callable>

`gad.invoker` builds a callable that is bound, up front, to:

  * the overload of `fn` resolved for the parameter TYPES held in `args` (the
    initial elements of `args` are the types — `[int]`, `[str, str]`, …), and
  * that same `args` array.

Calling the returned invoker runs the resolved overload with the array's CURRENT
values — reusing the same backing array (no per-call allocation), without
re-dispatching on the argument types and without re-validating them (the overload
was resolved once). Mutate the array in place between calls to feed new values:

    var (
        args = [int]                 // the initial element is the parameter type
        inv  = gad.invoker(str, args)
    )
    for i := 0; i < 1000; i++ { args[0] = i; buf.write(inv()) }

It is the fast path for a hot loop that calls the same function shape many times.
Because it skips validation, the caller is trusted to keep the array's values
compatible with the resolved signature (feeding a different type is undefined).

A function with no overloads is bound directly; `**nargs` are captured once and
forwarded to every call.

The invoker resolves the overload once and reuses the args array, so it beats the
equivalent direct call — clearest for a typed function, where the per-call
dispatch and parameter-type validation it skips is the dominant cost. Benchmark
(representative ns per call; run with `gad test -bench=. samples/testing`):

| typed f(a int, b int)      | ns/op |
|----------------------------|-------|
| via invoker (args reused)  | ~555  |
| direct f(i, i)             | ~712  |

In a Go benchmark over 100_000 calls the invoker also allocated ~46% less
(~698k vs ~1.30M allocs/op).

Run:
    gad test samples/testing
    gad test -bench=. samples/testing      # invoker vs the direct call

## Example — `invoker_test.gad`

```gad
/**
The invoker resolves str's int overload and reuses the args array; the result
matches calling str() directly.
**/
test "matches the direct call" {
	args := [int]
	inv := gad.invoker(str, args)
	for i := 0; i < 10; i++ {
		args[0] = i
		t.equal(str(i), inv(); msg=str(i))
	}
}

/**
The parameter types in args select the overload: [str,str] -> the str method,
[int,int] -> the int one.
**/
test "resolves the overload by arg types" {
	f(a int, b int) => a + b
	met f(a str, b str) => a + "-" + b

	sargs := [str, str]
	iargs := [int, int]
	sinv := gad.invoker(f, sargs)
	iinv := gad.invoker(f, iargs)

	sargs[0] = "x"; sargs[1] = "y"
	iargs[0] = 3; iargs[1] = 4
	t.equal("x-y", sinv())
	t.equal(7, iinv())
}

/**
A plain function (no overloads) is bound directly and reuses its args array.
**/
test "plain function" {
	add := func(a, b) => a + b
	args := [0, 0]
	inv := gad.invoker(add, args)
	args[0] = 20; args[1] = 22
	t.equal(42, inv())
}

/**
**nargs captured at construction are forwarded to every invocation.
**/
test "captures named args" {
	f(v; sep="-") => str(v) + sep
	args := [int]
	inv := gad.invoker(f, args; sep="|")
	args[0] = 7
	t.equal("7|", inv())
}

/**
The returned value is an `invoker`, and its signature advertises a callable.
**/
test "type and signature" {
	inv := gad.invoker(str, [int])
	t.equal("invoker", typeName(inv))
	t.true(bool("<ret callable>" in str(gad.invoker)) or false)
}

// Benchmarks — a typed two-arg function through the invoker vs. called directly
// (see the header for representative numbers).

/**
`inv()` — typed two-arg function through the invoker.
**/
bench "typed ::: invoker" {
	f(a int, b int) => a + b
	args := [int, int]
	inv := gad.invoker(f, args)
	for i := 0; i < t.n; i++ { args[0] = i; args[1] = i; _ := inv() }
}

/**
`f(i, i)` — the same typed function called directly.
**/
bench "typed ::: direct" {
	f(a int, b int) => a + b
	for i := 0; i < t.n; i++ { _ := f(i, i) }
}
```
