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
      Seventh pass done: applied the same closure-free treatment to Dict
      iteration (`for k, v in m`), the most common non-array case. Dict.Iterate
      built its keys slice then wrapped it in SliceEntryIteration (a
      RangeIteration + valid/readTo/get closures) per loop; replaced with a
      concrete closure-free dictIterator (iterator.go) that walks the keys slice
      directly, honouring step/reversed exactly like arrayIterator (Dict.Iterate
      still builds+sorts the keys). New BenchmarkVMDictIterate: 73994->53994
      allocs/op (~27%), 2.87MB->2.23MB, 4.7ms->3.98ms (~15%). `go build/test
      ./...` and `go test -race -run TestVMIterator$` clean; sorted/reversed dict
      iteration verified (values sorted ->[1,2,3], reversed ->[3,2,1]). The keys
      slice and per-key Str boxing remain (fundamental to a stable order / the
      boxed-Object model).
      Eighth pass done: each for-in still allocated a fresh &IteratorState{} in
      the iterator's Start(). Since the concrete array/dict iterators are already
      one heap allocation per loop, embedded the IteratorState in the iterator
      struct (arrayIterator.state / dictIterator.state) and Start now returns
      &it.state (reset each Start) — folding two allocs into one, no lifecycle
      risk (state lives exactly as long as the iterator). BenchmarkVMIterate
      23994->18994 allocs/op, BenchmarkVMDictIterate 53994->48994 (−1 alloc/loop
      each). `go build/test ./...` and -race clean; stateful iterator objects
      (`it.next` then `for k,v in it`) and repeated dict iteration still correct.
      REMAINING: pooling the remaining two per-loop objects (the concrete
      iterator + StateIteratorObject — needs a release point at loop end and must
      not pool user-visible iterator(...) values, high-risk), the same
      closure-free treatment for the rarer KeyValueArray/KeyValueArrays/Args
      iterators, and large-int/Str boxing (tagged-value representation — major
      refactor).
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
- [x] replace class/interface extends syntaxe from `extends { A, m.B }` to `class { *A, *m.B, ... x = 1 ... }` (using `*Expr`). update samples, tests and docs.
      DONE. The `extends {}` block is gone; parents are now `*Parent` spread body
      items (classes keep the optional `: Alias`, so `*Base`, `*m.B`, `*A: Alias`;
      interfaces have no alias). AST unchanged (ClassExpr.Parents /
      InterfaceExpr.Parents still populated), so the compiler lowering to the
      builder `define(; extends=[…])` is untouched — the `extends=` named arg is
      the runtime API and stays.
      Parser (parser/class.go, parser/interface.go): parseClassBodyItem /
      parseInterfaceBodyItem recognise a leading token.Mul and parse one parent
      (ParsePrimaryExpr, then optional `: Alias` for classes); dropped
      parseClassExtendsBlock / parseInterfaceExtendsBlock and the four
      `Literal != "extends"` name guards (extends is a plain ident again).
      Formatter (parser/node/class.go, interface.go): parents render as one
      `*Parent` item each, ExtendsDoc on the first; removed writeClassParents.
      Evidence: `go build ./...`, `go vet ./parser/... .`, `go test ./...` all
      clean (updated TestParseClass `*A, *B: B2` and TestParseInterface
      `*A, *mod.B`); built binary runs samples 11/19/24/25, a scratch
      `interface {*Base …}` + `class {*Animal: A …}` parses/runs/round-trips, fmt
      idempotent on 24_interfaces.gad. Updated samples 19/24/25, tests
      parser_test.go + new_test.go, docs classes.md/method-interfaces.md/
      samples/README.md. (Builder-API `extends=[…]` in 11_classes.gad and
      classes.md kept — that's the lowered form, not the sugar.)
- [x] create parser for test and bench stmts `test NAME { ... }` (`t` var is available). examples: `test xIs1 { x:=1; t.equal(x, 1) }`,  `test "x Is 2" { x:=2; t.equal(x, 1) }`. bech like test `bench fib { ... }`,  `bench "the fib" { ... }`.
    the test and bench Stmts allow doc comment. create samples, docs and tests.
      DONE. `test NAME { … }` / `bench NAME { … }` where NAME is an identifier or a
      string literal, `t` is injected, and a `///` doc comment is allowed.
      `test`/`bench` are CONTEXTUAL (only a statement when followed by NAME + `{`,
      via isTestStmtStart lookahead), so `test := import("test")`, `test.equal(…)`,
      `bench()` still work.
      AST: parser/node/test.go (TestStmt, TestKind) with WriteCode round-trip
      (bare vs quoted name, doc). Parser: parser/test.go (ParseTestStmt +
      isTestStmtStart) hooked into DefaultParseStmt. Compiler:
      compiler_test_stmt.go lowers each to a top-level
      `const __gadTest_<pos> = [kind, name, func(t){body}, doc]`
      (gad.TestRegistryPrefix). Runner: cmd/gad/test_cmd.go discover() now also
      reads those const bindings (statementTest) alongside func-form tests, so
      both forms mix in source order; -v prints the doc when available.
      Evidence: `go build ./...`, `go vet ./...`, `go test ./...` all clean.
        - parser tests: TestParseTestStmt (ident/string/bench + contextual
          `test :=`/`bench.run()`/`test(x)`), TestParseTestStmtNode.
        - runner tests: TestRunFileStatementForm (pass/fail/skip + string name),
          TestRunFileStatementBench.
        - `gad test samples/testing` -> 5 passed, 0 failed, 1 skipped; fmt
          round-trips the statements + doc comment and it still runs.
      Docs: doc/stdlib-test.md (statement-form section + updated example).
      Sample: samples/testing/math_test.gad (both forms).
- [x] check doc generator for test/bench stmt
      DONE. The `gad doc` generator (cmd/gad/doc_gen.go, doc.go) now groups
      `test NAME { … }` into a **Tests** section and `bench NAME { … }` into a
      **Benchs** section (per request), in both the must-exported and
      exported/internal layouts, with a TOC entry each. New `testEntries(file)`
      walks the `*node.TestStmt`s (in source order) into docEntry values (name +
      `test/bench NAME { … }` code line + doc comment via docContent(ts.Doc));
      names are shown quoted when written as a string literal. They are kept out
      of the Constants/Variables/Types buckets (explicit `case *node.TestStmt` in
      internalStmtEntry). Tests/benches are listed even without a doc comment.
      Evidence: `go build ./...`, `go vet ./cmd/gad/`, `go test ./...` clean;
      TestDocGeneratorGroupsTestsAndBenchs asserts the Tests/Benchs headings +
      TOC bullets, per-entry `### test **name**`/code/doc, quoted string names,
      and that a bench is not listed among the tests. Manually confirmed the
      rendered Markdown on a scratch module.
- [x] nested subtests (`test NAME { test SUB { … } }`, like Go t.Run) + `t.helper()`.
      DONE. A `test`/`bench` nested inside a test body is a subtest: the compiler
      (compiler_test_stmt.go) detects `t` in scope (c.symbolTable.Resolve("t"))
      and lowers it to `t.run("NAME", func(t){body})` instead of the top-level
      const; the T context (stdlib/test/module.go) gains run(name, fn) — runs the
      child synchronously on a forked VM (gad.NewInvoker), records it under subs
      named parent/NAME, and returns whether it passed (a sub failure/skip does
      not abort the parent). Failure() is now recursive (SelfFailed() is own-only)
      so a parent fails when a subtest fails, like Go. Added `t.helper()` (no-op,
      Go parity). Runner (cmd/gad/test_cmd.go) reportNode() walks the subtree,
      reporting each node as path/parent/child. Doc generator (doc_gen.go)
      testEntries() recurses into test bodies so subtests appear with
      parent/child qualified names and their own `///` doc comments.
      Evidence: `go build ./...`, `go vet ./...`, `go test ./...` clean; race
      clean. New tests: TestRunFileNestedSubtests (3 pass / 2 fail incl. deep
      nesting + parent-fails-on-subfail), TestHelperNoOp,
      TestFailurePropagatesFromSubs, TestDocGeneratorNestedTests. Sample
      samples/testing/math_test.gad has a nested `test sum { … }` with subtest
      doc comments; `gad test samples/testing` -> 9 passed, 0 failed, 1 skipped;
      fmt idempotent. Docs: doc/stdlib-test.md (Subtests section + t.helper/t.run).