//go:build !js
// +build !js

package main

import (
	"bytes"
	"context"
	"flag"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/repr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	initSuggestions()
}

func TestREPL(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cw := &console{buf: bytes.NewBuffer(nil)}

	r := newREPL(ctx, cw)

	t.Run("commands", func(t *testing.T) {
		require.NoError(t, r.execute(".commands"))
		testHasPrefix(t, string(cw.consume()),
			".commands     \tPrint REPL commands\n")
	})
	t.Run("builtins", func(t *testing.T) {
		require.NoError(t, r.execute(".builtins"))
		testHasPrefix(t, string(cw.consume()),
			"DISCARD_WRITER         \tBuiltin\n")
	})
	t.Run("keywords", func(t *testing.T) {
		require.NoError(t, r.execute(".keywords"))
		testHasPrefix(t, string(cw.consume()),
			"break\ncontinue\nelse\nfor\nfunc\nif\nreturn\ntrue\nfalse\nyes\nno\nin\nnil\n"+
				"import\nparam\nglobal\nvar\nconst\ntry\ncatch\nfinally\nthrow\n__callee__\n__named_args__\n__args__\n"+
				"STDIN\nSTDOUT\nSTDERR\n__name__\n__file__\n__is_module__\n",
		)
	})
	t.Run("unresolved reference", func(t *testing.T) {
		require.NoError(t, r.execute("test"))
		testHasPrefix(t, string(cw.consume()),
			"\n!   Compile Error: unresolved reference \"test\"")
	})
	t.Run("assignment", func(t *testing.T) {
		require.NoError(t, r.execute("test := 1"))
		testHasPrefix(t, string(cw.consume()), "\n⇦   nil\n")
	})
	t.Run("bytecode", func(t *testing.T) {
		require.NoError(t, r.execute("func(){}"))
		testHasPrefix(t, string(cw.consume()), "\n⇦   "+repr.Quote("compiledFunction #1()")+"\n")
		require.NoError(t, r.execute(".bytecode"))
		testHasPrefix(t, string(cw.consume()), "Bytecode\n")
	})
	t.Run("gc", func(t *testing.T) {
		require.NoError(t, r.execute(".gc"))
		require.Equal(t, "", string(cw.consume()))
	})
	t.Run("globals", func(t *testing.T) {
		require.NoError(t, r.execute(".globals"))
		testHasPrefix(t, string(cw.consume()), `{Gosched: `+repr.Quote("function:Gosched")+`, SOURCE_PATH: `+
			repr.Quote("reflectSlice:github.com/gad-lang/gad/importers.PathList"+repr.Quote("&[]"))+`}`)
	})
	t.Run("globals plus", func(t *testing.T) {
		require.NoError(t, r.execute(".globals+"))
		testHasPrefix(t, string(cw.consume()), "&gad.SyncDict{")
	})
	t.Run("locals", func(t *testing.T) {
		r := newREPL(ctx, cw)
		require.NoError(t, r.execute("test := 1"))
		cw.consume()
		require.NoError(t, r.execute(".locals"))
		require.Equal(t, string(cw.consume()), "[1]\n")
	})
	t.Run("locals plus", func(t *testing.T) {
		r := newREPL(ctx, cw)
		require.NoError(t, r.execute("test := 1"))
		cw.consume()
		require.NoError(t, r.execute(".locals+"))
		require.Equal(t, string(cw.consume()), "[]gad.Object{1}\n")
	})
	t.Run("return 1", func(t *testing.T) {
		r := newREPL(ctx, cw)
		require.NoError(t, r.execute("test := 1"))
		cw.consume()
		require.NoError(t, r.execute("return test"))
		testHasPrefix(t, string(cw.consume()), "\n⇦   1\n")
	})
	t.Run("return", func(t *testing.T) {
		r := newREPL(ctx, cw)
		require.NoError(t, r.execute("return 1"))
		cw.consume()
		require.NoError(t, r.execute(".return"))
		require.Equal(t, string(cw.consume()), "1\n")
	})
	t.Run("return plus", func(t *testing.T) {
		r := newREPL(ctx, cw)
		require.NoError(t, r.execute("return 1"))
		cw.consume()
		require.NoError(t, r.execute(".return+"))
		require.Equal(t, string(cw.consume()),
			"GoType:gad.Int, OpDotName:int, Value:1\n")
	})
	t.Run("symbols", func(t *testing.T) {
		r := newREPL(ctx, cw)
		require.NoError(t, r.execute("test := 1"))
		cw.consume()
		require.NoError(t, r.execute(".symbols"))
		symout := string(cw.consume())
		require.Regexp(t, `test\s+LOCAL`, symout)
	})
	t.Run("symbols+", func(t *testing.T) {
		r := newREPL(ctx, cw)
		require.NoError(t, r.execute("test := 1"))
		cw.consume()
		require.NoError(t, r.execute(".symbols+"))
		symout := string(cw.consume())
		testHasPrefix(t, symout, "[Symbol{Literal:")
		require.Contains(t, symout,
			"Symbol{Literal:Gosched Index:0 Scope:GLOBAL Assigned:false Original:<nil> Constant:false}")
		require.Contains(t, symout,
			"Symbol{Literal:SOURCE_PATH Index:1 Scope:GLOBAL Assigned:false Original:<nil> Constant:false}")
		require.Contains(t, symout,
			"Symbol{Literal:test Index:0 Scope:LOCAL Assigned:true Original:<nil> Constant:false}")
	})
	t.Run("modules_cache", func(t *testing.T) {
		r := newREPL(ctx, cw)
		require.NoError(t, r.execute("test := 1"))
		cw.consume()
		require.NoError(t, r.execute(".modules_cache"))
		require.Equal(t, string(cw.consume()), "[]\n")
	})
	t.Run("import time", func(t *testing.T) {
		r := newREPL(ctx, cw)
		require.NoError(t, r.execute(`import("time")`))
		testHasPrefix(t, string(cw.consume()), "\n⇦   {")
		require.NoError(t, r.execute(".modules_cache"))
		testHasPrefix(t, string(cw.consume()), "[{")
	})
	t.Run("import strings", func(t *testing.T) {
		r := newREPL(ctx, cw)
		require.NoError(t, r.execute(`import("strings")`))
		testHasPrefix(t, string(cw.consume()), "\n⇦   {")
		require.NoError(t, r.execute(".modules_cache"))
		testHasPrefix(t, string(cw.consume()), "[{")
	})
	t.Run("import fmt", func(t *testing.T) {
		r := newREPL(ctx, cw)
		require.NoError(t, r.execute(`import("fmt")`))
		testHasPrefix(t, string(cw.consume()), "\n⇦   {")
		require.NoError(t, r.execute(".modules_cache"))
		testHasPrefix(t, string(cw.consume()), "[{")
	})
	t.Run("import json", func(t *testing.T) {
		r := newREPL(ctx, cw)
		require.NoError(t, r.execute(`import("json")`))
		testHasPrefix(t, string(cw.consume()), "\n⇦   {")
		require.NoError(t, r.execute(".modules_cache"))
		testHasPrefix(t, string(cw.consume()), "[{")
	})
	t.Run("import encoding/base64", func(t *testing.T) {
		r := newREPL(ctx, cw)
		require.NoError(t, r.execute(`import("encoding/base64")`))
		testHasPrefix(t, string(cw.consume()), "\n⇦   {")
		require.NoError(t, r.execute(".modules_cache"))
		testHasPrefix(t, string(cw.consume()), "[{")
	})
	t.Run("memory_stats", func(t *testing.T) {
		require.NoError(t, r.execute(".memory_stats"))
		testHasPrefix(t, string(cw.consume()), "ToInterface Memory Stats")
	})
	t.Run("reset", func(t *testing.T) {
		r := newREPL(ctx, cw)
		require.NoError(t, r.execute("test := 1"))
		cw.consume()
		require.Same(t, errReset, r.execute(".reset"))
		require.Empty(t, cw.consume())
	})
	t.Run("type_method_constructor", func(t *testing.T) {
		r := newREPL(ctx, cw)
		require.NoError(t, r.execute(`Point := struct("Point",fields={x:0, y:0});func Point(x,y)=>Point(x=x,y=y)`))
		cw.consume()
		require.NoError(t, r.execute("func int(p Point) => p.x * p.y"))
		cw.consume()
		require.NoError(t, r.execute("str(int)"))
		require.Equal(t, "⇦   \""+repr.Quote("builtinType int")+" with 1 methods:\\n  "+
			"1. "+repr.Quote("compiledFunction #9(p Point)")+"\"",
			strings.TrimSpace(string(cw.consume())))
		require.NoError(t, r.execute("int(Point(2,8))"))
		require.Equal(t, "⇦   16", strings.TrimSpace(string(cw.consume())))
	})
	t.Run("exit", func(t *testing.T) {
		require.Same(t, errExit, r.execute(".exit"))
		require.Empty(t, cw.consume())
	})
}

func TestFlags(t *testing.T) {
	defer resetGlobals()

	testCases1 := []struct {
		args            []string
		expectEnabled   bool
		expectParser    bool
		expectOptimizer bool
		expectCompiler  bool
	}{
		{[]string{"-trace", "parser"}, true, true, false, false},
		{[]string{"-trace", "optimizer"}, true, false, true, false},
		{[]string{"-trace", "compiler"}, true, false, false, true},

		{[]string{"-trace", "parser,optimizer"}, true, true, true, false},
		{[]string{"-trace", "parser,compiler"}, true, true, false, true},
		{[]string{"-trace", "compiler,optimizer"}, true, false, true, true},
	}
	for _, tC := range testCases1 {
		t.Run("", func(t *testing.T) {
			// trace flags are global variables, set to defaults after each run
			defer resetGlobals()

			fs := flag.NewFlagSet("test tracers", flag.ExitOnError)
			fp, to, _, err := parseFlags(fs, tC.args)
			require.NoError(t, err)
			require.Empty(t, fp)
			require.Empty(t, to)
			require.Equal(t, tC.expectEnabled, traceEnabled)
			require.Equal(t, tC.expectParser, traceParser)
			require.Equal(t, tC.expectOptimizer, traceOptimizer)
			require.Equal(t, tC.expectCompiler, traceCompiler)
		})
	}

	fs := flag.NewFlagSet("script file", flag.ExitOnError)
	fp, to, _, err := parseFlags(fs, []string{"testdata/fibtc.gad"})
	require.NoError(t, err)
	require.Empty(t, to)
	require.Equal(t, "testdata/fibtc.gad", fp)

	resetGlobals()

	fs = flag.NewFlagSet("stdin", flag.ExitOnError)
	fp, to, _, err = parseFlags(fs, []string{"-"})
	require.NoError(t, err)
	require.Empty(t, to)
	require.Equal(t, "-", fp)

	resetGlobals()

	fs = flag.NewFlagSet("file does not exist", flag.ExitOnError)
	_, _, _, err = parseFlags(fs, []string{"testdata/doesnotexist"})
	require.Error(t, err)
}

func resetGlobals() {
	noOptimizer = false
	traceEnabled = false
	traceParser = false
	traceOptimizer = false
	traceCompiler = false
}

func TestExecuteScript(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const workdir = "./testdata"
	scr, err := ioutil.ReadFile("./testdata/fibtc.gad")
	require.NoError(t, err)
	require.NoError(t, newScript(ctx, "(test1)", workdir, scr, nil).execute())

	traceEnabled = true
	require.NoError(t, newScript(ctx, "(test2)", workdir, scr, ioutil.Discard).execute())
	resetGlobals()

	// FIXME: Following is a flaky test which compromise CI
	// Although runtime.Gosched() is called in script, scheduler may not switch
	// to goroutine started VM goroutine in time. Find a better way to test
	// canceled/timed out context error. A script with a long execution time can
	// fix this issue but it will extend the test duration.

	cancel()
	err = newScript(ctx, "(test3)", workdir, scr, nil).execute()
	if err != nil {
		if err != context.Canceled && err != gad.ErrVMAborted {
			t.Fatalf("unexpected error: %+v", err)
		}
	}
}

func testHasPrefix(t *testing.T, s, pref string) {
	t.Helper()
	v := strings.HasPrefix(s, pref)
	if !assert.True(t, v) {
		t.Fatalf("input: %q\nprefix: %q", s, pref)
	}
}

type console struct {
	buf *bytes.Buffer
}

func (c *console) consume() []byte {
	p := c.buf.Bytes()
	c.buf.Reset()
	return p
}

func (c *console) Write(p []byte) (int, error) {
	return c.buf.Write(p)
}
