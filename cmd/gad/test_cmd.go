// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	gad "github.com/gad-lang/gad"
	gadtest "github.com/gad-lang/gad/stdlib/test"
	cc "github.com/moisespsena-go/command-context"
)

// testOptions holds the parsed `gad test` flags.
type testOptions struct {
	verbose   bool
	runPat    string // regex matched against test names
	benchPat  string // regex matched against benchmark names ("" = none, "." = all)
	benchtime time.Duration
	timeout   time.Duration
	runRe     *regexp.Regexp
	benchRe   *regexp.Regexp
}

var testOptionsKey = struct{ name string }{"testOptions"}

// testCommand is the `gad test [PATH...]` subcommand: it discovers *_test.gad
// files, runs their top-level `test*` functions with a fresh test context (T)
// and reports pass/fail; with -bench it also runs `bench*` functions.
func testCommand() *cc.Command {
	return &cc.Command{
		Name:  "test",
		Usage: "[flags] [PATH...]",
		Description: "Run Gad tests found in *_test.gad files.\n" +
			"\nEach file is executed, then its top-level functions whose name starts with\n" +
			"\"test\" are run with a fresh test context: func testFoo(t) { t.equal(1, 1) }.\n" +
			"A failed assertion (t.equal, t.true, …) records the failure and aborts that\n" +
			"test. Functions starting with \"bench\" are benchmarks run with -bench: they\n" +
			"loop t.n times, e.g. func benchFoo(t) { for i := 0; i < t.n; i++ { work() } }.\n" +
			"\nPATH may be a file or a directory; write DIR/... to recurse. With no PATH the\n" +
			"current directory is scanned recursively. Exits non-zero if any test fails.",
		New: func(ctx *cc.CommandContext) error {
			o := &testOptions{}
			o.registerFlags(ctx.Flags())
			ctx.WithValue(testOptionsKey, o)
			return nil
		},
		ParseArgs: func(ctx *cc.CommandContext) error {
			o := ctx.Value(testOptionsKey).(*testOptions)
			return o.compile()
		},
		Run: func(ctx *cc.CommandContext) error {
			o := ctx.Value(testOptionsKey).(*testOptions)
			return o.run(ctx)
		},
	}
}

func (o *testOptions) registerFlags(fs *flag.FlagSet) {
	fs.BoolVar(&o.verbose, "v", false, "verbose: log every test, not only failures")
	fs.StringVar(&o.runPat, "run", "", "run only tests whose name matches this regex")
	fs.StringVar(&o.benchPat, "bench", "", "run benchmarks whose name matches this regex (e.g. . for all)")
	fs.DurationVar(&o.benchtime, "benchtime", time.Second, "minimum run time per benchmark")
	fs.DurationVar(&o.timeout, "timeout", 0, "per-file timeout (0 = none)")
}

// compile pre-compiles the -run/-bench regexes.
func (o *testOptions) compile() (err error) {
	if o.runPat != "" {
		if o.runRe, err = regexp.Compile(o.runPat); err != nil {
			return fmt.Errorf("invalid -run regex: %w", err)
		}
	}
	if o.benchPat != "" {
		if o.benchRe, err = regexp.Compile(o.benchPat); err != nil {
			return fmt.Errorf("invalid -bench regex: %w", err)
		}
	}
	return nil
}

// run discovers and runs the test files under the given paths (or ".").
func (o *testOptions) run(ctx *cc.CommandContext) error {
	args := ctx.Args
	if len(args) == 0 {
		args = []string{"./..."}
	}
	var files []string
	seen := map[string]bool{}
	for _, arg := range args {
		fs, err := testFiles(arg)
		if err != nil {
			return err
		}
		for _, f := range fs {
			if !seen[f] {
				seen[f], files = true, append(files, f)
			}
		}
	}
	if len(files) == 0 {
		fmt.Fprintln(ctx.Out, "no *_test.gad files found")
		return nil
	}

	var totalPass, totalFail, totalSkip int
	for _, path := range files {
		pass, fail, skip := o.runFile(ctx, path)
		totalPass, totalFail, totalSkip = totalPass+pass, totalFail+fail, totalSkip+skip
	}

	fmt.Fprintf(ctx.Out, "\ntest: %d passed, %d failed, %d skipped\n",
		totalPass, totalFail, totalSkip)
	if totalFail > 0 {
		return fmt.Errorf("test: %d test(s) failed", totalFail)
	}
	return nil
}

// runFile executes one *_test.gad file, runs its discovered test/bench
// functions and reports results, returning the pass/fail/skip counts.
func (o *testOptions) runFile(ctx *cc.CommandContext, path string) (pass, fail, skip int) {
	src, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(ctx.Err, "FAIL %s: %s\n", path, err)
		return 0, 1, 0
	}

	builtins := gad.NewBuiltins().Build()
	opts := gad.CompileOptions{CompilerOptions: gad.CompilerOptions{
		ModuleMap: DefaultModuleMap(".", &sourcePath),
	}}
	eval := gad.NewEval(builtins, defaultSymbolTable(builtins.Builtins().NameSet), opts,
		&gad.RunOpts{StdOut: io.Discard, StdErr: io.Discard})
	eval.VM.Builtins = builtins

	runCtx := context.Background()
	if o.timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(runCtx, o.timeout)
		defer cancel()
	}
	if _, _, err := eval.RunScript(runCtx, src); err != nil {
		fmt.Fprintf(ctx.Err, "FAIL %s: %s\n", path, err)
		return 0, 1, 0
	}

	tests, benches := discover(eval)
	for _, fn := range tests {
		if o.runRe != nil && !o.runRe.MatchString(fn.name) {
			continue
		}
		p, f, s := o.runTest(ctx, path, eval, fn)
		pass, fail, skip = pass+p, fail+f, skip+s
	}
	if o.benchRe != nil {
		for _, fn := range benches {
			if !o.benchRe.MatchString(fn.name) {
				continue
			}
			o.runBench(ctx, path, eval, fn)
		}
	}
	return pass, fail, skip
}

// discovered is a top-level test/benchmark function found in a file.
type discovered struct {
	name string
	fn   gad.CallerObject
	doc  string // for statement-form tests (`test NAME { … }`)
}

// discover returns the file's tests and benchmarks in source (index) order: both
// top-level `test*`/`bench*` functions and the const bindings that `test NAME
// { … }` / `bench NAME { … }` statements lower to (gad.TestRegistryPrefix).
func discover(eval *gad.Eval) (tests, benches []discovered) {
	names := eval.SymbolTable().LocalNames()
	type entry struct {
		idx  int
		name string
	}
	var entries []entry
	for idx, name := range names {
		entries = append(entries, entry{idx, name})
	}
	// stable order by local index (definition order)
	for i := 1; i < len(entries); i++ {
		for j := i; j > 0 && entries[j-1].idx > entries[j].idx; j-- {
			entries[j-1], entries[j] = entries[j], entries[j-1]
		}
	}
	for _, e := range entries {
		if e.idx < 0 || e.idx >= len(eval.Locals) {
			continue
		}
		val := eval.Locals[e.idx]

		// Statement form: `const __gadTest_<pos> = [kind, name, fn, doc]`.
		if strings.HasPrefix(e.name, gad.TestRegistryPrefix) {
			if d, isBench, ok := statementTest(val); ok {
				if isBench {
					benches = append(benches, d)
				} else {
					tests = append(tests, d)
				}
			}
			continue
		}

		// Function form: top-level `func test*(t)` / `func bench*(t)`.
		fn, ok := val.(gad.CallerObject)
		if !ok {
			continue
		}
		switch {
		case isPrefixFunc(e.name, "test"):
			tests = append(tests, discovered{name: e.name, fn: fn})
		case isPrefixFunc(e.name, "bench"):
			benches = append(benches, discovered{name: e.name, fn: fn})
		}
	}
	return tests, benches
}

// statementTest decodes a `[kind, name, fn, doc]` registry entry produced by a
// `test`/`bench` statement, reporting whether it is a benchmark.
func statementTest(val gad.Object) (d discovered, isBench, ok bool) {
	arr, _ := val.(gad.Array)
	if len(arr) < 3 {
		return d, false, false
	}
	fn, _ := arr[2].(gad.CallerObject)
	if fn == nil {
		return d, false, false
	}
	d.name, d.fn = arr[1].ToString(), fn
	if len(arr) >= 4 {
		d.doc = strings.TrimSpace(arr[3].ToString())
	}
	return d, arr[0].ToString() == "bench", true
}

// isPrefixFunc reports whether name is a test/bench function name: it begins
// with prefix (case-insensitive) and is longer than the bare prefix.
func isPrefixFunc(name, prefix string) bool {
	if len(name) <= len(prefix) {
		return false
	}
	return strings.EqualFold(name[:len(prefix)], prefix)
}

// runTest runs one test function with a fresh context and reports its result.
func (o *testOptions) runTest(ctx *cc.CommandContext, path string, eval *gad.Eval, d discovered) (pass, fail, skip int) {
	t := gadtest.NewT(d.name)
	_, err := invoke(eval, d.fn, t)

	// A skip (t.skip / SkipError) aborts without recording a failure.
	if skipped, msg := t.Skipped(); skipped {
		if o.verbose {
			fmt.Fprintf(ctx.Out, "SKIP %s/%s: %s\n", path, d.name, msg)
		}
		return 0, 0, 1
	}

	failed := t.Failure()
	// An error with no recorded failure is an unexpected runtime error (a require
	// abort already recorded its failure via t, so we don't double-report it).
	if err != nil && !failed {
		failed = true
		fmt.Fprintf(ctx.Err, "FAIL %s/%s: %s\n", path, d.name, err)
	}

	if failed {
		fmt.Fprintf(ctx.Err, "FAIL %s/%s\n", path, d.name)
		for _, m := range t.Failures() {
			for _, line := range strings.Split(m, "\n") {
				fmt.Fprintf(ctx.Err, "    %s\n", line)
			}
		}
		return 0, 1, 0
	}

	if o.verbose {
		fmt.Fprintf(ctx.Out, "PASS %s/%s\n", path, d.name)
		if d.doc != "" {
			fmt.Fprintf(ctx.Out, "    %s\n", strings.ReplaceAll(d.doc, "\n", "\n    "))
		}
		for _, l := range t.Logs() {
			fmt.Fprintf(ctx.Out, "    %s\n", l)
		}
	}
	return 1, 0, 0
}

// runBench runs one benchmark function, auto-scaling the iteration count until
// it runs at least benchtime, then reports ns/op.
func (o *testOptions) runBench(ctx *cc.CommandContext, path string, eval *gad.Eval, d discovered) {
	n := 1
	var elapsed time.Duration
	for {
		t := gadtest.NewT(d.name)
		t.SetBenchN(n)
		start := time.Now()
		_, err := invoke(eval, d.fn, t)
		elapsed = time.Since(start)
		if t.Failure() {
			fmt.Fprintf(ctx.Err, "FAIL %s/%s\n", path, d.name)
			return
		}
		if err != nil {
			fmt.Fprintf(ctx.Err, "FAIL %s/%s: %s\n", path, d.name, err)
			return
		}
		if elapsed >= o.benchtime || n >= 1e9 {
			break
		}
		// predict the count that reaches benchtime, growing by at most 100x.
		next := n
		if elapsed > 0 {
			next = int(int64(n) * o.benchtime.Nanoseconds() / elapsed.Nanoseconds())
		}
		next += next / 5 // +20% safety margin
		if next <= n {
			next = n + 1
		}
		if next > n*100 {
			next = n * 100
		}
		n = next
	}
	nsPerOp := float64(elapsed.Nanoseconds()) / float64(n)
	fmt.Fprintf(ctx.Out, "BENCH %s/%s\t%d\t%.1f ns/op\n", path, d.name, n, nsPerOp)
}

// invoke calls fn with the test context t using a forked VM.
func invoke(eval *gad.Eval, fn gad.CallerObject, t *gadtest.T) (gad.Object, error) {
	inv := gad.NewInvoker(eval.VM, fn)
	inv.Acquire()
	defer inv.Release()
	return inv.Invoke(gad.Args{gad.Array{t}}, nil)
}

// testFiles resolves a path arg (file, directory or DIR/...) to *_test.gad files.
func testFiles(arg string) ([]string, error) {
	recursive, path := splitRecursive(arg)
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{path}, nil
	}
	all, err := scanDir(path, recursive, &fileFilter{})
	if err != nil {
		return nil, err
	}
	var out []string
	for _, f := range all {
		if strings.HasSuffix(f, "_test.gad") {
			out = append(out, f)
		}
	}
	return out, nil
}
