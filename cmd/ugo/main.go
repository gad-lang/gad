// Copyright (c) 2020 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

// +build !js

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/c-bata/go-prompt"

	"github.com/ozanh/ugo"
	"github.com/ozanh/ugo/stdlib/time"
	"github.com/ozanh/ugo/token"
)

const logo = `
            /$$$$$$   /$$$$$$ 
           /$$__  $$ /$$__  $$
 /$$   /$$| $$  \__/| $$  \ $$
| $$  | $$| $$ /$$$$| $$  | $$
| $$  | $$| $$|_  $$| $$  | $$
| $$  | $$| $$  \ $$| $$  | $$
|  $$$$$$/|  $$$$$$/|  $$$$$$/
 \______/  \______/  \______/ 
                                       
`

const (
	title         = "uGO"
	promptPrefix  = ">>> "
	promptPrefix2 = "... "
)

var (
	isMultiline    bool
	noOptimizer    bool
	traceEnabled   bool
	traceParser    bool
	traceOptimizer bool
	traceCompiler  bool
)

var (
	initialSuggLen int
)

var grepl *repl

func init() {
	var trace string
	flag.StringVar(&trace, "trace", "", `comma separated units: -trace parser,optimizer,compiler`)
	flag.BoolVar(&noOptimizer, "no-optimizer", false, `disable optimization`)
	flag.Parse()
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
}

type repl struct {
	ctx          context.Context
	eval         *ugo.Eval
	lastBytecode *ugo.Bytecode
	lastResult   ugo.Object
	multiline    string
	werr         prompt.ConsoleWriter
	wout         prompt.ConsoleWriter
	commands     map[string]func()
}

func newREPL(ctx context.Context) *repl {
	moduleMap := ugo.NewModuleMap()
	moduleMap.AddBuiltinModule("time", time.Module)
	opts := ugo.CompilerOptions{
		ModulePath:        "(repl)",
		ModuleMap:         moduleMap,
		SymbolTable:       ugo.NewSymbolTable(),
		OptimizerMaxCycle: ugo.TraceCompilerOptions.OptimizerMaxCycle,
		TraceParser:       traceParser,
		TraceOptimizer:    traceOptimizer,
		TraceCompiler:     traceCompiler,
		OptimizeConst:     !noOptimizer,
		OptimizeExpr:      !noOptimizer,
	}
	if traceEnabled {
		opts.Trace = os.Stdout
	}
	r := &repl{
		ctx:  ctx,
		eval: ugo.NewEval(opts, nil),
		werr: prompt.NewStdoutWriter(),
		wout: prompt.NewStdoutWriter(),
	}
	r.commands = map[string]func(){
		".bytecode":      r.cmdBytecode,
		".builtins":      r.cmdBuiltins,
		".gc":            r.cmdGC,
		".globals":       r.cmdGlobals,
		".globals+":      r.cmdGlobalsVerbose,
		".locals":        r.cmdLocals,
		".locals+":       r.cmdLocalsVerbose,
		".return":        r.cmdReturn,
		".return+":       r.cmdReturnVerbose,
		".reset":         r.cmdReset,
		".symbols":       r.cmdSymbols,
		".memory_stats":  r.cmdMemoryStats,
		".modules_cache": r.cmdModulesCache,
		".exit":          r.cmdExit,
	}
	return r
}

func (r *repl) cmdBytecode() {
	fmt.Printf("%s\n", r.lastBytecode)
}

func (r *repl) cmdBuiltins() {
	builtins := make([]string, len(ugo.BuiltinsMap))
	for k, v := range ugo.BuiltinsMap {
		builtins[v] = fmt.Sprint(ugo.BuiltinObjects[v].TypeName(), ":", k)
	}
	fmt.Println(strings.Join(builtins, "\n"))
}

func (*repl) cmdGC() { runtime.GC() }

func (r *repl) cmdGlobals() {
	fmt.Printf("%+v\n", r.eval.Globals)
}

func (r *repl) cmdGlobalsVerbose() {
	fmt.Printf("%#v\n", r.eval.Globals)
}

func (r *repl) cmdLocals() {
	fmt.Printf("%+v\n", r.eval.Locals)
}

func (r *repl) cmdLocalsVerbose() {
	fmt.Printf("%#v\n", r.eval.Locals)
}

func (r *repl) cmdReturn() {
	fmt.Printf("%#v\n", r.lastResult)
}

func (r *repl) cmdReturnVerbose() {
	if r.lastResult != nil {
		fmt.Printf("GoType:%[1]T, TypeName:%[2]s, Value:%#[1]v\n",
			r.lastResult, r.lastResult.TypeName())
	} else {
		fmt.Println("<nil>")
	}
}

func (r *repl) cmdReset() {
	grepl = newREPL(r.ctx)
}

func (r *repl) cmdSymbols() {
	fmt.Printf("%v\n", r.eval.Opts.SymbolTable.Symbols())
}

func (r *repl) cmdMemoryStats() {
	// writeMemStats writes the formatted current, total and OS memory
	// being used. As well as the number of garbage collection cycles completed.
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	_, _ = fmt.Fprintf(os.Stdout, "Go Memory Stats see: "+
		"https://golang.org/pkg/runtime/#MemStats\n\n")
	_, _ = fmt.Fprintf(os.Stdout, "HeapAlloc = %s", humanFriendlySize(m.HeapAlloc))
	_, _ = fmt.Fprintf(os.Stdout, "\tHeapObjects = %v", m.HeapObjects)
	_, _ = fmt.Fprintf(os.Stdout, "\tSys = %s", humanFriendlySize(m.Sys))
	_, _ = fmt.Fprintf(os.Stdout, "\tNumGC = %v\n", m.NumGC)
}

func (r *repl) cmdModulesCache() {
	fmt.Printf("%v\n", r.eval.ModulesCache)
}

func (r *repl) cmdExit() {
	os.Exit(0)
}

func (r *repl) writeErrorStr(msg string) {
	r.werr.SetColor(prompt.Red, prompt.DefaultColor, true)
	r.werr.WriteStr(msg)
	r.werr.Flush()
}

func (r *repl) writeStr(msg string) {
	r.wout.SetColor(prompt.Green, prompt.DefaultColor, false)
	r.wout.WriteStr(msg)
	r.wout.Flush()
}

func (r *repl) executor(line string) {
	switch {
	case line == "":
		if !isMultiline {
			return
		}
	case line[0] == '.':
		if f, ok := r.commands[line]; ok {
			f()
			return
		}
	case strings.HasSuffix(line, "\\"):
		isMultiline = true
		r.multiline += line[:len(line)-1] + "\n"
		return
	}
	r.executeScript(line)
}

func (r *repl) executeScript(line string) {
	defer func() {
		isMultiline = false
		r.multiline = ""
	}()
	var err error
	r.lastResult, r.lastBytecode, err = r.eval.Run(r.ctx, []byte(r.multiline+line))
	if err != nil {
		r.writeErrorStr(fmt.Sprintf("\n%+v\n", err))
		return
	}
	if err != nil {
		r.writeErrorStr(fmt.Sprintf("VM:\n     %+v\n", err))
		return
	}
	switch v := r.lastResult.(type) {
	case ugo.String:
		r.writeStr(fmt.Sprintf("%q\n", string(v)))
	case ugo.Char:
		r.writeStr(fmt.Sprintf("%q\n", rune(v)))
	case ugo.Bytes:
		r.writeStr(fmt.Sprintf("%v\n", []byte(v)))
	default:
		r.writeStr(fmt.Sprintf("%v\n", r.lastResult))
	}

	symbols := r.eval.Opts.SymbolTable.Symbols()
	suggestions = suggestions[:initialSuggLen]
	for _, s := range symbols {
		if s.Scope != ugo.ScopeBuiltin {
			suggestions = append(suggestions,
				prompt.Suggest{
					Text:        s.Name,
					Description: string(s.Scope) + " variable",
				},
			)
		}
	}
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

func completer(in prompt.Document) []prompt.Suggest {
	w := in.GetWordBeforeCursorWithSpace()
	return prompt.FilterHasPrefix(suggestions, w, true)
}

var suggestions = []prompt.Suggest{
	// Commands
	{Text: ".bytecode", Description: "Print Bytecode"},
	{Text: ".builtins", Description: "Print Builtins"},
	{Text: ".reset", Description: "Reset"},
	{Text: ".locals", Description: "Print Locals"},
	{Text: ".locals+", Description: "Print Locals (verbose)"},
	{Text: ".globals", Description: "Print Globals"},
	{Text: ".globals+", Description: "Print Globals (verbose)"},
	{Text: ".return", Description: "Print Last Return Result"},
	{Text: ".return+", Description: "Print Last Return Result (verbose)"},
	{Text: ".modules_cache", Description: "Print Modules Cache"},
	{Text: ".memory_stats", Description: "Print Memory Stats"},
	{Text: ".gc", Description: "Run Go GC"},
	{Text: ".symbols", Description: "Print Symbols"},
	{Text: ".exit", Description: "Exit"},
}

func init() {
	// add builtins to suggestions
	for k := range ugo.BuiltinsMap {
		suggestions = append(suggestions,
			prompt.Suggest{
				Text:        k,
				Description: "Builtin " + k,
			},
		)
	}
	for tok := token.Question + 3; tok.IsKeyword(); tok++ {
		s := tok.String()
		suggestions = append(suggestions, prompt.Suggest{
			Text:        s,
			Description: "keyword " + s,
		})
	}
	initialSuggLen = len(suggestions)
}

func livePrefix() (string, bool) {
	if isMultiline {
		return promptPrefix2, true
	}
	return "", false
}

func main() {
	fmt.Println("Copyright (c) 2020 Ozan Hacıbekiroğlu")
	fmt.Println("License: MIT")
	fmt.Println("Press Ctrl+D to exit or use .exit command")
	fmt.Println(logo)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	grepl = newREPL(ctx)
	p := prompt.New(
		func(s string) { grepl.executor(s) },
		completer,
		prompt.OptionPrefix(promptPrefix),
		prompt.OptionHistory([]string{
			"a := 1",
			"sum := func(a...) { total:=0; for v in a { total+=v }; return total }",
			"func(a, b){ return a*b }(2, 3)",
			`println("")`,
			`var (x, y, z); if x { y } else { z }`,
			`var (x, y, z); x ? y : z`,
			`for i := 0; i < 3; i++ { }`,
			`m := {}; for k,v in m { printf("%s:%v\n", k, v) }`,
			`try { } catch err { } finally { }`,
		}),
		prompt.OptionLivePrefix(livePrefix),
		prompt.OptionTitle(title),
		prompt.OptionPrefixTextColor(prompt.Yellow),
		prompt.OptionPreviewSuggestionTextColor(prompt.Blue),
		prompt.OptionSelectedSuggestionBGColor(prompt.LightGray),
		prompt.OptionSuggestionBGColor(prompt.DarkGray),
	)
	p.Run()
}
