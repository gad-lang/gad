// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

//go:build !js
// +build !js

package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/runehelper"
	"github.com/gad-lang/gad/stdlib/helper"
	"github.com/peterh/liner"

	"github.com/gad-lang/gad/importers"
	"github.com/gad-lang/gad/token"
)

const (
	title         = "Gad"
	promptPrefix  = ">>> "
	promptPrefix2 = "... "
)

var (
	noOptimizer     bool
	traceEnabled    bool
	traceParser     bool
	traceOptimizer  bool
	traceCompiler   bool
	safe            bool
	disabledModules map[string]bool
)

var suggestions []suggest
var initialSuggLen int

// Sentinel errors for repl.
var (
	errExit  = errors.New("exit")
	errReset = errors.New("reset")
)

var (
	sourcePath    = importers.PathList(filepath.SplitList(os.Getenv("GADPATH")))
	scriptGlobals = &gad.SyncDict{
		Value: gad.Dict{
			"Gosched": &gad.Function{
				Name: "Gosched",
				Value: func(gad.Call) (gad.Object, error) {
					runtime.Gosched()
					return gad.Nil, nil
				},
			},
			"SOURCE_PATH": func() gad.Object {
				v := gad.MustNewReflectValue(&sourcePath).(*gad.ReflectSlice)
				v.Options.ToStr = func() string {
					arr := make(gad.Array, len(sourcePath))
					for i, v := range sourcePath {
						arr[i] = gad.Str(v)
					}
					return arr.ToString()
				}
				return v
			}(),
		},
	}
)

type suggest struct {
	text        string
	description string
	typ         string
}

type repl struct {
	ctx          context.Context
	eval         *gad.Eval
	out          io.Writer
	commands     map[string]func(string) error
	script       *bytes.Buffer
	lastBytecode *gad.Bytecode
	lastResult   gad.Object
	isMultiline  bool
}

func newREPL(ctx context.Context, stdout io.Writer) *repl {
	opts := gad.CompileOptions{CompilerOptions: gad.CompilerOptions{
		Module: &gad.ModuleInfo{
			Name: "(repl)",
		},
		ModuleMap:         DefaultModuleMap(".", &sourcePath),
		SymbolTable:       defaultSymbolTable(),
		OptimizerMaxCycle: gad.TraceCompilerOptions.OptimizerMaxCycle,
		TraceParser:       traceParser,
		TraceOptimizer:    traceOptimizer,
		TraceCompiler:     traceCompiler,
		OptimizeConst:     !noOptimizer,
		OptimizeExpr:      !noOptimizer,
	}}

	if stdout == nil {
		stdout = os.Stdout
	}

	if traceEnabled {
		opts.Trace = stdout
	}

	r := &repl{
		ctx:    ctx,
		eval:   gad.NewEval(opts, &gad.RunOpts{Globals: scriptGlobals}),
		out:    stdout,
		script: bytes.NewBuffer(nil),
	}
	r.setSymbolSuggestions()

	r.commands = map[string]func(string) error{
		".commands":      r.cmdCommands,
		".builtins":      r.cmdBuiltins,
		".keywords":      r.cmdKeywords,
		".bytecode":      r.cmdBytecode,
		".gc":            r.cmdGC,
		".globals":       r.cmdGlobals,
		".globals+":      r.cmdGlobalsVerbose,
		".locals":        r.cmdLocals,
		".locals+":       r.cmdLocalsVerbose,
		".return":        r.cmdReturn,
		".return+":       r.cmdReturnVerbose,
		".symbols":       r.cmdSymbols,
		".symbols+":      r.cmdSymbolsVerbose,
		".modules_cache": r.cmdModulesCache,
		".memory_stats":  r.cmdMemoryStats,
		".reset":         func(string) error { return errReset },
		".exit":          func(string) error { return errExit },
	}
	return r
}

func (r *repl) cmdCommands(_ string) error {
	suggs, pad := r.rangeSuggestions(
		func(s suggest) bool { return s.typ == "" },
	)
	r.printSuggestions(suggs, pad)
	return nil
}

func (r *repl) cmdBuiltins(_ string) error {
	suggs, pad := r.rangeSuggestions(
		func(s suggest) bool {
			return s.typ == "builtin" && !strings.HasPrefix(s.text, ":")
		},
	)
	sort.Slice(suggs, func(i, j int) bool {
		return suggs[i].description < suggs[j].description ||
			suggs[i].text < suggs[j].text
	})
	r.printSuggestions(suggs, pad)
	return nil
}

func (r *repl) cmdKeywords(_ string) error {
	suggs, pad := r.rangeSuggestions(
		func(s suggest) bool { return s.typ == "keyword" },
	)
	r.printSuggestions(suggs, pad)
	return nil
}

func (r *repl) cmdSymbols(_ string) error {
	suggs, pad := r.rangeSuggestions(
		func(s suggest) bool { return s.typ == "symbol" },
	)
	r.printSuggestions(suggs, pad)
	return nil
}

func (*repl) rangeSuggestions(filter func(suggest) bool) ([]suggest, int) {
	var suggs []suggest
	var maxtext int
	for _, v := range suggestions {
		if !filter(v) {
			continue
		}
		suggs = append(suggs, v)
		if maxtext < len(v.text) {
			maxtext = len(v.text)
		}
	}
	return suggs, maxtext
}

func (r *repl) printSuggestions(suggs []suggest, maxtext int) {
	const spaces = "                                                           "
	for _, cmd := range suggs {
		_, _ = fmt.Fprintf(r.out, "%s", cmd.text)
		if len(cmd.description) > 0 {
			_, _ = fmt.Fprintf(r.out, "%s", spaces[:maxtext-len(cmd.text)])
			_, _ = fmt.Fprintf(r.out, "\t%v", cmd.description)
		}
		_, _ = fmt.Fprintln(r.out)
	}
}

func (r *repl) cmdBytecode(_ string) error {
	_, _ = fmt.Fprintf(r.out, "%s\n", r.lastBytecode)
	return nil
}

func (*repl) cmdGC(_ string) error {
	runtime.GC()
	return nil
}

func (r *repl) cmdGlobals(_ string) error {
	_, _ = fmt.Fprintf(r.out, "%+v\n", r.eval.Globals)
	return nil
}

func (r *repl) cmdGlobalsVerbose(_ string) error {
	_, _ = fmt.Fprintf(r.out, "%#v\n", r.eval.Globals)
	return nil
}

func (r *repl) cmdLocals(_ string) error {
	_, _ = fmt.Fprintf(r.out, "%+v\n", r.eval.Locals)
	return nil
}

func (r *repl) cmdLocalsVerbose(_ string) error {
	fmt.Fprintf(r.out, "%#v\n", r.eval.Locals)
	return nil
}

func (r *repl) cmdReturn(_ string) error {
	_, _ = fmt.Fprintf(r.out, "%#v\n", r.lastResult)
	return nil
}

func (r *repl) cmdReturnVerbose(_ string) error {
	if r.lastResult != nil {
		_, _ = fmt.Fprintf(r.out,
			"GoType:%[1]T, OpDotName:%[2]s, Value:%#[1]v\n",
			r.lastResult, r.lastResult.Type().Name())
	} else {
		_, _ = fmt.Fprintln(r.out, gad.ReprQuote("nil"))
	}
	return nil
}

func (r *repl) cmdSymbolsVerbose(_ string) error {
	_, _ = fmt.Fprintf(r.out, "%v\n", r.eval.Opts.SymbolTable.Symbols())
	return nil
}

func (r *repl) cmdMemoryStats(_ string) error {
	// writeMemStats writes the formatted current, total and OS memory
	// being used. As well as the number of garbage collection cycles completed.
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	_, _ = fmt.Fprintf(r.out, "ToInterface Memory Stats see: "+
		"https://golang.org/pkg/runtime/#MemStats\n\n")
	_, _ = fmt.Fprintf(r.out, "HeapAlloc = %s", humanFriendlySize(m.HeapAlloc))
	_, _ = fmt.Fprintf(r.out, "\tHeapObjects = %v", m.HeapObjects)
	_, _ = fmt.Fprintf(r.out, "\tSys = %s", humanFriendlySize(m.Sys))
	_, _ = fmt.Fprintf(r.out, "\tNumGC = %v\n", m.NumGC)
	return nil
}

func (r *repl) cmdModulesCache(_ string) error {
	_, _ = fmt.Fprintf(r.out, "%v\n", r.eval.ModulesCache)
	return nil
}

func (r *repl) writeString(msg string) {
	_, _ = fmt.Fprint(r.out, msg)
	_, _ = fmt.Fprintln(r.out)
}

func (r *repl) execute(line string) error {
	switch {
	case !r.isMultiline && line == "":
		return nil
	case !r.isMultiline && len(line) > 0 && line[0] == '.':
		cmd := strings.Fields(line)[0]
		if fn, ok := r.commands[cmd]; ok {
			return fn(line)
		}
	case strings.HasSuffix(line, "\\"):
		r.isMultiline = true
		r.script.WriteString(line[:len(line)-1])
		r.script.WriteString("\n")
		return nil
	}

	r.script.WriteString(line)

	r.executeScript()

	r.isMultiline = false
	r.setSymbolSuggestions()
	r.script.Reset()
	return nil
}

func (r *repl) executeScript() {
	var err error

	r.lastResult, r.lastBytecode, err = r.eval.Run(r.ctx, r.script.Bytes())
	if err != nil {
		r.writeString(fmt.Sprintf("\n!   %+v", err))
		return
	}

	switch v := r.lastResult.(type) {
	case gad.Str:
		r.writeString(fmt.Sprintf("\n⇦   %q", string(v)))
	case gad.Char:
		r.writeString(fmt.Sprintf("\n⇦   %q", rune(v)))
	case gad.Bytes:
		r.writeString(fmt.Sprintf("\n⇦   %v", []byte(v)))
	default:
		r.writeString(fmt.Sprintf("\n⇦   %v", r.lastResult))
	}
}

func (r *repl) setSymbolSuggestions() {
	symbols := r.eval.Opts.SymbolTable.Symbols()
	suggestions = suggestions[:initialSuggLen]

	for _, s := range symbols {
		if s.Scope != gad.ScopeBuiltin {
			suggestions = append(suggestions,
				suggest{
					text:        s.Name,
					description: s.Scope.String(),
					typ:         "symbol",
				},
			)
		}
	}
}

func (r *repl) prefix() string {
	if r.isMultiline {
		return promptPrefix2
	}
	return promptPrefix
}

func (r *repl) printInfo() {
	_, _ = fmt.Fprintln(r.out, "Copyright (c) 2020-2023 Ozan Hacıbekiroğlu")
	_, _ = fmt.Fprintln(r.out, "https://github.com/gad-lang/gad License: MIT",
		"Build:", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	_, _ = fmt.Fprintln(r.out, "Write .commands to list available commands")
	_, _ = fmt.Fprintln(r.out, "Press Ctrl+D or write .exit command to exit")
	_, _ = fmt.Fprintln(r.out)
}

func (r *repl) run(history io.Reader) error {
	line := liner.NewLiner()
	defer line.Close()

	line.SetMultiLineMode(true)
	line.SetCompleter(complete)
	_, err := line.ReadHistory(history)
	if err != nil {
		err = &gad.Error{Message: "failed history read", Cause: err}
		return err
	}
	r.printInfo()

	var str string

	for err == nil {
		str, err = line.Prompt(r.prefix())
		if err != nil {
			if err == io.EOF {
				err = nil
				break
			}
			err = &gad.Error{Message: "prompt error", Cause: err}
			break
		}
		err = r.execute(str)
		if err == nil {
			if !r.isMultiline && len(str) > 0 {
				if v := strings.TrimSpace(str); len(v) > 0 {
					line.AppendHistory(v)
				}
			}
		}
	}
	return err
}

func complete(line string) (completions []string) {
	var contains []string
	for _, v := range suggestions {
		if strings.HasPrefix(v.text, line) {
			completions = append(completions, v.text)
		} else if strings.Contains(v.text, line) {
			contains = append(contains, v.text)
		}
	}
	completions = append(completions, contains...)
	return
}

func defaultSymbolTable() *gad.SymbolTable {
	table := gad.NewSymbolTable(gad.NewBuiltins())
	_, err := table.DefineGlobals([]string{"Gosched", "SOURCE_PATH"})
	if err != nil {
		panic(&gad.Error{Message: "global symbol define error", Cause: err})
	}
	return table
}

func DefaultModuleMap(workdir string, sourcePath *importers.PathList) *gad.ModuleMap {
	mb := helper.NewModuleMapBuilder()
	mb.Safe = safe
	mb.Disabled = disabledModules
	return mb.Build().
		SetExtImporter(&importers.FileImporter{
			WorkDir:      workdir,
			FileReader:   importers.ShebangReadFile,
			NameResolver: importers.OsDirsNameResolverPtr(sourcePath),
		})
}

func humanFriendlySize(b uint64) string {
	if b < 1024 {
		return fmt.Sprint(strconv.FormatUint(b, 10), " bytes")
	}

	if b >= 1024 && b < 1024*1024 {
		return fmt.Sprint(strconv.FormatFloat(
			float64(b)/1024, 'f', 1, 64), " KiB")
	}

	return fmt.Sprint(strconv.FormatFloat(
		float64(b)/1024/1024, 'f', 1, 64), " MiB")
}

func initSuggestions() {
	suggestions = []suggest{
		// Commands
		{text: ".commands", description: "Print REPL commands"},
		{text: ".builtins", description: "Print Builtins"},
		{text: ".keywords", description: "Print Keywords"},
		{text: ".bytecode", description: "Print Bytecode"},
		{text: ".locals", description: "Print Locals"},
		{text: ".locals+", description: "Print Locals (verbose)"},
		{text: ".globals", description: "Print Globals"},
		{text: ".globals+", description: "Print Globals (verbose)"},
		{text: ".return", description: "Print Last Return Result"},
		{text: ".return+", description: "Print Last Return Result (verbose)"},
		{text: ".modules_cache", description: "Print Modules Cache"},
		{text: ".memory_stats", description: "Print Memory Stats"},
		{text: ".gc", description: "Run Garbage Collector"},
		{text: ".symbols", description: "Print Symbols"},
		{text: ".symbols+", description: "Print Symbols (verbose)"},
		{text: ".reset", description: "Reset"},
		{text: ".exit", description: "Exit"},
	}

	// add builtins to suggestions
	for k, id := range gad.BuiltinsMap {
		var desc string
		o := gad.BuiltinObjects[id]
		switch o.(type) {
		case *gad.BuiltinFunction:
			desc = "Builtin Function"
		case *gad.BuiltinObjType:
			desc = "Builtin Object Type"
		case *gad.Error:
			desc = "Builtin Error"
		default:
			desc = "Builtin"
		}
		suggestions = append(suggestions,
			suggest{
				text:        k,
				description: desc,
				typ:         "builtin",
			},
		)
	}

	// add keywords to suggestions
	for tok := token.KeyworkBegin_ + 1; tok.IsKeyword(); tok++ {
		s := tok.String()
		suggestions = append(suggestions, suggest{
			text: s,
			typ:  "keyword",
		})
	}
	initialSuggLen = len(suggestions)
}

func parseFlags(
	flagset *flag.FlagSet,
	args []string,
) (filePath string, timeout time.Duration, params []string, err error) {
	var (
		trace    string
		disabled string
		module   bool
	)
	flagset.StringVar(&trace, "trace", "",
		`Comma separated units: -trace parser,optimizer,compiler`)
	flagset.BoolVar(&noOptimizer, "no-optimizer", false, `Disable optimization`)
	flagset.BoolVar(&safe, "safe", false, `Disable al external acess modules: "http", "os" and "filepath"`)
	flagset.BoolVar(&module, "module", false, `if SCRIPT_FILE does not exists, check exists in GADPATH`)
	flagset.StringVar(&disabled, "disabled-modules", "", `Disable external acess modules by comma separated units: -disabled-modules http,os`)
	flagset.DurationVar(&timeout, "timeout", 0,
		"Program timeout. It is applicable if a script file is provided and "+
			"must be non-zero duration")

	disabledModules = map[string]bool{}

	for _, v := range strings.Split(disabled, ",") {
		if v = strings.TrimSpace(v); v != "" {
			disabledModules[v] = true
		}
	}

	flagset.Usage = func() {
		_, _ = fmt.Fprint(flagset.Output(),
			"Usage: gad [flags] [SCRIPT_FILE [ARGS...]]\n\n",
			"If script file is not provided, REPL terminal application is started.\n\n",
			"If script file is provided, pass named params with '--NAME=VALUE' named flags '--NAME'.\n",
			"  Script example for join arguments:\n\n",
			"    // usages: 1) SCRIPT.gad a b c (result: a,b,c)\n",
			"    //         2) SCRIPT.gad a b c --sep + (result: a+b+c)\n",
			"    //         3) SCRIPT.gad a b c --ln (result: a,b,c\\n)\n",
			"    //         4) SCRIPT.gad a b c --sep + --ln (result: a+b+c\\n)\n",
			"    param (*args, sep=\",\", ln=no)\n",
			"    if !args { return }\n    for _, arg in args[:-1] { print(arg, sep) }\n    print(args[-1])\n    if ln { println() }\n\n",
			"Use - to read from stdin\n\n",
			"\nFlags:\n",
		)
		flagset.PrintDefaults()
	}

	if err = flagset.Parse(args); err != nil {
		return
	}

	if trace != "" {
		traceEnabled = true
		trace = "," + trace + ","
		if strings.Contains(trace, ",parser,") {
			traceParser = true
		}
		if strings.Contains(trace, ",optimizer,") {
			traceOptimizer = true
		}
		if strings.Contains(trace, ",compiler,") {
			traceCompiler = true
		}
	}

	if flagset.NArg() < 1 {
		return
	}

	filePath = flagset.Arg(0)

	params = flagset.Args()[1:]

	if filePath == "-" {
		return
	}

	_, err = os.Stat(filePath)

	if os.IsNotExist(err) && module {
		var err2 error
		for _, p := range sourcePath {
			if _, err2 = os.Stat(p); err2 == nil {
				filePath = filepath.Join(p, filePath)
				break
			}
		}
	}

	return
}

type Script struct {
	ctx        context.Context
	modulePath string
	workdir    string
	script     []byte
	traceOut   io.Writer
	args       []string
	sourcePath *importers.PathList
}

func newScript(ctx context.Context, modulePath string, workdir string, script []byte, traceOut io.Writer) *Script {
	return &Script{ctx: ctx, modulePath: modulePath, workdir: workdir, script: script, traceOut: traceOut, sourcePath: &sourcePath}
}

func (s *Script) execute() error {
	opts := gad.CompileOptions{
		CompilerOptions: gad.DefaultCompilerOptions,
	}
	opts.SymbolTable = defaultSymbolTable()
	opts.ModuleMap = DefaultModuleMap(s.workdir, s.sourcePath)
	opts.Module = &gad.ModuleInfo{
		Name: path.Clean(s.modulePath),
		File: "file:" + s.modulePath,
	}

	if traceEnabled {
		opts.Trace = s.traceOut
		opts.TraceParser = traceParser
		opts.TraceCompiler = traceCompiler
		opts.TraceOptimizer = traceOptimizer
	}

	bc, err := gad.Compile(s.script, opts)
	if err != nil {
		return err
	}

	namedArgs := make(gad.Dict)
	args := make(gad.Array, 0)

	if numArgs := len(s.args); numArgs > 0 {
		var newArgs []gad.Object
	args:
		for i := 0; i < numArgs; i++ {
			arg := s.args[i]
			if strings.HasPrefix(arg, "--") && len(arg) > 2 {
				arg := []rune(arg[2:])
				lastIndex := len(arg) - 1
				for i := 0; i <= lastIndex; i++ {
					if arg[i] == '=' {
						namedArgs[string(arg[:i])] = gad.Str(arg[i+1:])
						continue args
					} else if runehelper.IsIdentifier(arg[i]) {
						if i == lastIndex {
							namedArgs[string(arg)] = gad.Yes
						}
					} else {
						continue args
					}
				}
				if i+1 < numArgs {
					continue
				}
			}
			newArgs = append(newArgs, gad.Str(arg))
		}
		args = newArgs
	}

	if requiredParams := bc.Main.Params.RequiredCount(); requiredParams > 0 {
		if len(args) < requiredParams {
			return gad.ErrWrongNumArguments.NewError(fmt.Sprintf("want=%d got=%d", requiredParams, len(s.args)))
		}
	}

	if len(namedArgs) > 0 && !bc.Main.NamedParams.Variadic() {
		np := bc.Main.NamedParams.ToMap()
		for name := range namedArgs {
			if np[name] == nil {
				return gad.ErrUnexpectedNamedArg.NewError(name)
			}
		}
	}

	vm := gad.NewVM(bc).SetRecover(true)

	done := make(chan struct{})

	go func() {
		defer close(done)
		_, err = vm.RunOpts(&gad.RunOpts{
			Globals:   scriptGlobals,
			Args:      gad.Args{args},
			NamedArgs: gad.NewNamedArgs(namedArgs.ToKeyValueArray()),
		})
	}()

	select {
	case <-done:
	case <-s.ctx.Done():
		vm.Abort()
		<-done
		if err == nil {
			err = s.ctx.Err()
		}
	}
	return err
}

func hasMode(f *os.File, m os.FileMode) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&m == m
}

func hasInputRedirection() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeNamedPipe == os.ModeNamedPipe ||
		info.Size() > 0
}

func setTerminalTitle(title string) {
	if runtime.GOOS == "windows" {
		return
	}

	titleBytes := bytes.ReplaceAll([]byte(title), []byte{0x13}, []byte{})
	titleBytes = bytes.ReplaceAll(titleBytes, []byte{0x07}, []byte{})

	_, _ = os.Stdout.Write([]byte{0x1b, ']', '2', ';'})
	_, _ = os.Stdout.Write(titleBytes)
	_, _ = os.Stdout.Write([]byte{0x07})
}

func main() {
	filePath, timeout, args, err := parseFlags(flag.CommandLine, os.Args[1:])
	checkErr(err, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if len(filePath) == 0 && hasInputRedirection() {
		filePath = "-"
	}

	if len(filePath) > 0 {
		if timeout > 0 {
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}

		var (
			modulePath = filePath
			workdir    = "."
			script     []byte
		)
		if filePath == "-" {
			modulePath = "(stdin)"
			script, err = ioutil.ReadAll(os.Stdin)
		} else {
			workdir = filepath.Dir(filePath)
			script, err = ioutil.ReadFile(filePath)
		}
		importers.Shebang2Slashes(script)

		checkErr(err, cancel)
		s := newScript(ctx, modulePath, workdir, script, os.Stdout)
		s.args = args
		err = s.execute()
		checkErr(err, cancel)
		return
	}

	if !hasMode(os.Stdout, os.ModeCharDevice) {
		_, _ = fmt.Fprintln(os.Stderr, "not a terminal")
		os.Exit(1)
	}

	initSuggestions()
	setTerminalTitle(title)

	const history = "a := 1\n" +
		"sum := func(...a) { total := 0; for v in a { total += v }; return total }\n" +
		"func(a, b){ return a*b }(2, 3)\n" +
		"println(\"\")\n" +
		"var (x, y, z); if x { y } else { z }\n" +
		"var (x, y, z); x ? y : z\n" +
		"for i := 0; i < 3; i++ { }\n" +
		"m := {}; for k,v in m { printf(\"%s:%v\\n\", k, v) }\n" +
		"try { } catch err { } finally { }\n"

L:
	for {
		hist := strings.NewReader(history)

		err = newREPL(ctx, os.Stdout).run(hist)
		if err != nil {
			switch err {
			case errReset:
				continue
			case errExit:
				break L
			}
			checkErr(err, cancel)
		}
		break
	}
}

func checkErr(err error, fn func()) {
	if err == nil {
		return
	}

	defer os.Exit(1)
	_, _ = fmt.Fprintf(os.Stderr, "%+v\n", err)
	if fn != nil {
		fn()
	}
}
