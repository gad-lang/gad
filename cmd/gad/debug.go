// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

//go:build !js
// +build !js

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/debug"
	cc "github.com/moisespsena-go/command-context"
)

// debugOptions holds the parsed flags of the `delve` subcommand.
type debugOptions struct {
	breaks      breakList
	stopOnEntry bool
}

const debugOptionsKey ctxKey = "debugOptions"

// breakList is a repeatable/comma-separated int flag for --break.
type breakList []int

func (b *breakList) String() string {
	parts := make([]string, len(*b))
	for i, n := range *b {
		parts[i] = strconv.Itoa(n)
	}
	return strings.Join(parts, ",")
}

func (b *breakList) Set(v string) error {
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			n, err := strconv.Atoi(p)
			if err != nil {
				return fmt.Errorf("invalid line number %q", p)
			}
			*b = append(*b, n)
		}
	}
	return nil
}

// debugCommand is `gad debug [flags] SCRIPT_FILE`: an interactive, delve-style
// debugger for Gad scripts (breakpoints, stepping, stack and locals).
func debugCommand() *cc.Command {
	return &cc.Command{
		Name:        "debug",
		Usage:       "[flags] SCRIPT_FILE",
		Description: "Debug a Gad script: breakpoints, stepping, stack and locals.",
		New: func(ctx *cc.CommandContext) error {
			o := &debugOptions{}
			ctx.Flags().Var(&o.breaks, "break", "breakpoint line (repeatable, comma-separated)")
			ctx.Flags().BoolVar(&o.stopOnEntry, "stop-on-entry", false, "pause before the first instruction")
			ctx.WithValue(debugOptionsKey, o)
			return nil
		},
		ParseArgs: func(ctx *cc.CommandContext) error {
			return ctx.Args.Eq(1)
		},
		Run: func(ctx *cc.CommandContext) error {
			o := ctx.Value(debugOptionsKey).(*debugOptions)
			return runDebug(ctx, ctx.Args[0], o)
		},
	}
}

// runDebug compiles the script, attaches a debug engine, runs the VM in a
// goroutine and drives an interactive command loop.
func runDebug(ctx *cc.CommandContext, file string, o *debugOptions) error {
	src, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	lines := strings.Split(string(src), "\n")

	builtins := gad.NewBuiltins()
	st := defaultSymbolTable(builtins.NameSet)
	opts := gad.CompileOptions{CompilerOptions: gad.DefaultCompilerOptions}
	opts.ModuleMap = DefaultModuleMap(filepath.Dir(file), &sourcePath)

	_, bc, err := gad.Compile(st, src, opts)
	if err != nil {
		return err
	}

	eng := debug.New(o.stopOnEntry)
	bps := map[int]bool{}
	for _, l := range o.breaks {
		bps[l] = true
	}
	eng.SetBreakpoints(sortedKeys(bps))

	vm := gad.NewVM(builtins.Build(), bc).SetRecover(true)
	vm.SetDebugger(eng)

	done := make(chan debugResult, 1)
	go func() {
		ret, rerr := vm.RunOpts(&gad.RunOpts{StdOut: ctx.Out, StdErr: ctx.Err})
		done <- debugResult{ret, rerr}
	}()

	out := ctx.Out
	fmt.Fprintf(out, "Debugging %s. Type 'help' for commands.\n", file)
	in := bufio.NewScanner(os.Stdin)

	for {
		select {
		case ev := <-eng.Stops():
			printStop(out, ev, lines)
		case r := <-done:
			if r.err != nil {
				fmt.Fprintf(out, "program exited with error: %v\n", r.err)
			} else {
				fmt.Fprintf(out, "program returned: %s\n", objString(r.ret))
			}
			return nil
		}

		// Command loop: inspect freely, resume with a step/continue command.
		resumed := false
		for !resumed {
			fmt.Fprint(out, "(gad-debug) ")
			if !in.Scan() {
				vm.Abort()
				eng.Continue()
				<-done
				return nil
			}
			resumed = handleDebugCmd(out, strings.TrimSpace(in.Text()), eng, vm, bps, done)
		}
	}
}

// debugResult carries the VM's exit value/error from the run goroutine.
type debugResult struct {
	ret gad.Object
	err error
}

// handleDebugCmd executes one debugger command. It returns true when the command
// resumes execution (so the outer loop waits for the next stop).
func handleDebugCmd(out io.Writer, line string, eng *debug.Engine, vm *gad.VM, bps map[int]bool, done chan debugResult) bool {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return false
	}
	switch fields[0] {
	case "c", "continue":
		eng.Continue()
		return true
	case "n", "next":
		eng.StepOver()
		return true
	case "s", "step":
		eng.StepInto()
		return true
	case "o", "out":
		eng.StepOut()
		return true
	case "bt", "backtrace", "where":
		printFrames(out, eng)
	case "l", "locals":
		printLocals(out, eng)
	case "b", "break":
		if len(fields) == 2 {
			if n, err := strconv.Atoi(fields[1]); err == nil {
				bps[n] = true
				eng.SetBreakpoints(sortedKeys(bps))
				fmt.Fprintf(out, "breakpoint set at line %d\n", n)
			} else {
				fmt.Fprintln(out, "usage: break LINE")
			}
		} else {
			fmt.Fprintln(out, "usage: break LINE")
		}
	case "clear":
		if len(fields) == 2 {
			if n, err := strconv.Atoi(fields[1]); err == nil {
				delete(bps, n)
				eng.SetBreakpoints(sortedKeys(bps))
				fmt.Fprintf(out, "breakpoint cleared at line %d\n", n)
			}
		}
	case "q", "quit":
		vm.Abort()
		eng.Continue()
		<-done
		fmt.Fprintln(out, "quit")
		os.Exit(0)
	case "help", "h", "?":
		fmt.Fprint(out, debugHelp)
	default:
		fmt.Fprintf(out, "unknown command %q (try 'help')\n", fields[0])
	}
	return false
}

const debugHelp = `commands:
  c, continue     run until the next breakpoint
  n, next         step over (same depth)
  s, step         step into
  o, out          step out
  b, break LINE   set a breakpoint
  clear LINE      remove a breakpoint
  bt, backtrace   print the call stack
  l, locals       print local variables
  q, quit         abort and exit
  help            this help
`

func printStop(out io.Writer, ev debug.StopEvent, lines []string) {
	fmt.Fprintf(out, "\nstopped (%s) at line %d:%d\n", ev.Reason, ev.Line, ev.Column)
	if ev.Line >= 1 && ev.Line <= len(lines) {
		fmt.Fprintf(out, "  %d| %s\n", ev.Line, lines[ev.Line-1])
	}
}

func printFrames(out io.Writer, eng *debug.Engine) {
	frames := eng.Frames()
	for i := len(frames) - 1; i >= 0; i-- {
		f := frames[i]
		fmt.Fprintf(out, "  #%d %s at %d:%d\n", len(frames)-1-i, f.FuncName, f.Line, f.Column)
	}
}

func printLocals(out io.Writer, eng *debug.Engine) {
	locals := eng.Locals()
	if len(locals) == 0 {
		fmt.Fprintln(out, "  (no locals)")
		return
	}
	for _, v := range locals {
		fmt.Fprintf(out, "  %s = %s (%s)\n", v.Name, v.Value, v.Type)
	}
}

func sortedKeys(m map[int]bool) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

func objString(o gad.Object) string {
	if o == nil || o == gad.Nil {
		return "nil"
	}
	return o.ToString()
}
