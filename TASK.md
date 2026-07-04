# IDE epic (`web/ide` backend + `web/app` React frontend)


# web/js projects

# Language

- [x] create builtin module `test` (like Go `testing` + testify/require) and a
      `gad test` subcommand to run `*_test.gad` files with reports + benchmarks.
      DONE. Modeled on the `time` module.
      Module: `stdlib/test/module.go` — `ModuleName`/`ModuleInit`; `T` context
      (`gad.Object` + `NameCallerObject` + `IndexGetter`) with require-style
      assertions (equal, notEqual, true, false, nil, notNil, contains, error,
      noError) that record a failure and abort via a `*FailError`; controls
      log/fail/fatal/skip (`*SkipError`)/name/failed; fields `.name/.failed/.n`.
      `Module` Dict exposes `T` + testify-style free helpers `test.equal(t,…)`
      delegating to the t method. Registered in `stdlib/helper/helper.go`.
      Command: `cmd/gad/test_cmd.go` — `testCommand()` discovers `*_test.gad`,
      runs each file via `NewEval(...).RunScript`, then finds top-level
      `test*`/`bench*` functions from `SymbolTable().LocalNames()` + `eval.Locals`
      and invokes each with a fresh `T` via `gad.NewInvoker(eval.VM, fn)`. Flags
      `-v -run -bench -benchtime -timeout`; benchmarks auto-scale `t.n` to reach
      `-benchtime` and report ns/op. Registered in `buildRootCommand()`.
      Convention learned (used in sample): named call args need `;` (`f(a; k=v)`);
      no multi-value `:=`/`=` (`a, b := 0, 1` unsupported); arrow funcs can't
      self-recurse by name.
      Evidence:
        - `go build ./...` -> exit 0.
        - `go test ./stdlib/test/` -> `ok  github.com/gad-lang/gad/stdlib/test`.
        - `go test ./cmd/gad/ -run 'TestIsPrefixFunc|TestTestFiles|TestRunFile'`
          -> all PASS (7 tests: discovery, reports, -run filter, bench, compile
          error, runtime error).
        - `go test ./...` -> no failures (grep -v ok/no-test empty).
        - `gad test -v -bench=. samples/testing` -> 3 passed, 0 failed, 1
          skipped, `BENCH …/benchFib 63715 17831.5 ns/op`, exit 0.
      Docs: `doc/stdlib-test.md` (guide) linked from `doc/README.md`.
      Sample: `samples/testing/math_test.gad` (runnable).

- [~] check if possible to improve bytecode performance after optimizer
      First pass done: operator dispatch was the top hotspot. `callBinaryOp` /
      `callSelfAssignOp` always allocated an Args+Call and walked the method tree
      (Args.Types + GetMethod + Methods.get) even for `int+int`. Added a fast path
      (vm.hasOpMethods): when the operator builtin has no user `met gad.…Op{Op}`
      overload, dispatch natively (BinaryOp / selfAssignOpDispatch → binOpObject),
      skipping the allocation and tree walk. New bench_test.go (BenchmarkVMFib,
      BenchmarkVMLoop): Fib(25) 251ms→61ms (5.3M→486K allocs); loop(100k)
      172ms→20ms (4.2M→200K allocs, 141MB→1.75MB). Side effect: a built-in
      operator runtime error is now clean (e.g. `ZeroDivisionError`) instead of
      `ErrCall: ‹binOpQuo…›; caused by …` (tests updated).
      Second pass done: xOpCallCompiled allocated the args slice (Args{nil,nil})
      and a *NamedArgs on every compiled-function call (both stored in the frame,
      so they escaped). Added inline per-frame buffers (frame.argsBuf/namedBuf) —
      frames are pooled, so &frame.buf does not allocate per call — and isolated
      the two remaining `&namedParams` stores (named-variadic + tail-call) so the
      locals stop escaping. Verified via `-gcflags=-m` (no escape) and behaviour
      (positional / named / `**kwargs` / tail-call recursion all correct, race
      clean). BenchmarkVMFib 486K→243K allocs (~half), 61ms→57ms; combined with
      the operator pass Fib is 251ms→57ms and 5.3M→243K allocs vs the original.
      Third pass done: the loop path's allocations were Int→Object boxing (each
      arithmetic result heap-allocs an interface box). Added a shared small-int
      box cache (smallInts, -256..1024) returned by intObject, applied to the
      Int+Int Add/Sub/Mul/Quo/Rem results — safe since Int is immutable and Go
      compares interface values by (type, value). BenchmarkVMSmallInts 149K→99K
      allocs (~34%, 1.45MB→1.05MB). Large ints (running sums, fib values) stay
      boxed — fundamental to the boxed-Object model; a bigger win would need a
      tagged/union value representation (major refactor).
      Fourth pass done: the remaining ~1 alloc/call was the frame's args slice —
      storeArgs's `return args` fallback flowed to frame.args, so escape analysis
      escaped the local always. args is at most 3 slots (positional, var-args tail,
      one merge spare), so argsBuf is now [3]Array and storeArgs always copies into
      it (never returns the caller's args). BenchmarkVMFib 243K→108 allocs (call
      path is now essentially alloc-free), 11.9MB→280KB, 57ms→49ms. Verified via
      -gcflags=-m (no args escape) and behaviour (positional / `f(*arr)` spread /
      `func(a, *b)` rest+merge / tail-call / named args all correct; race clean).
      Combined across all passes vs the original baseline: Fib 251ms→49ms (~5x),
      5.3M→108 allocs, 193MB→280KB.
      Fifth pass done: a plain `for x in it` (no options) still allocated two
      dicts per loop in NamedArgs.check to materialise an empty option set.
      check() now leaves o.m/o.ready nil when there are no passed names (reads on
      a nil map return nil; keys are consumed only when found, so o.ready is never
      written) — this also fixes a latent Add-after-check staleness. New
      BenchmarkVMIterate/BenchmarkVMDictAccess; iterate 54K→44K allocs (~18%),
      2.59MB→2.11MB. Verified: sorted/reversed iteration and named args still
      correct; full suite + -race ok.
      Sixth pass done: the for-in-over-array path allocated a RangeIteration
      plus the several closures SliceIteration/NewRangeIteration capture (valid,
      readTo, get) on every loop. Replaced Array.Iterate's generic machinery
      with a concrete closure-free `arrayIterator` (iterator.go): one struct
      allocation, honouring `step`/`reversed` (the valid range is just
      0<=i<len in either direction, so no per-call closure is needed). Semantics
      verified against the existing forward/reversed/step/step+reversed cases in
      TestVMIterator (vm_test.go:960-965). BenchmarkVMIterate 43994->23994
      allocs/op (~45%), 2.11MB->1.43MB, 4.2ms->3.6ms (~14%); `go build ./...`,
      `go test ./...`, and `go test -race -run TestVMIterator$|TestVMArray$` all
      clean; Fib/Loop benches unchanged (108 / 198947 allocs). The 4 remaining
      allocs/loop are the three iterator objects (IteratorState, arrayIterator,
      StateIteratorObject) plus large-index int boxing.
      REMAINING: pooling the three per-loop iterator objects (needs a reliable
      release point at loop end and must not pool user-visible iterator(...)
      values — high-risk lifecycle change), the same closure-free treatment for
      non-array iterators (KeyValueArray/Dict/etc.), and large-int boxing
      (tagged-value representation — major refactor).
- [x] check if allow keywords in `x.KEYWORD`, `[KEYWORD=...]` (keyvalue), `{KEYWORD:...}` (dict key), `(;KEYWORD=...)` (keyvalue array key).
      examples `x.class`, `[class=1]`, `{class:1}`, `(;class=1,class,false,nil,met,meti,func,if,else)`, all is single key. add doc for describe this rule
      DONE. `.name` selector and `{class: 1}` dict keys already accepted keywords;
      added it to `[keyword=value]` (ParseArrayLitOrKeyValue + ParseKeyValueLit)
      and `(;keyword=value)` / bare `(;keyword)` (ParseKeyValuePairLit) via a
      shared keywordStrLit helper (a keyword in a key position becomes a Str of
      its spelling). Only the key position changes — values keep their meaning
      (`(;x=false)` value is boolean false). Behaviour change: `true`/`false`/`nil`
      as keyValueArray keys are now the strings "true"/"false"/"nil" (updated
      TestParseKeyValueArray). Tests: TestParseKeywordKeys + cases in
      TestParseKeyValueArray. Docs: collections.md "Keyword keys". go test ./... ok.
- [~] replace class/interface extends syntaxe from `extends { A, m.B }` to `class { *A, *m.B, ... x = 1 ... }` (using `*Expr`). update samples, tests and docs.