package gad_test

import (
	"context"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	gadtime "github.com/gad-lang/gad/stdlib/time"

	. "github.com/gad-lang/gad"
)

func TestEval(t *testing.T) {
	type scriptResult struct {
		script string
		result Object
	}
	testCases := []struct {
		name      string
		opts      CompileOptions
		global    IndexGetSetter
		args      []Object
		namedArgs *NamedArgs
		ctx       context.Context
		sr        []scriptResult
	}{
		{
			name: "simple",
			sr: []scriptResult{
				{`var a`, Nil},
				{`1`, Int(1)},
				{`return 10`, Int(10)},
				{`a = 10`, Nil},
				{`return a`, Int(10)},
				{`return a*a`, Int(100)},
			},
		},
		{
			name: "import",
			opts: CompileOptions{CompilerOptions: CompilerOptions{
				ModuleMap: NewModuleMap().
					AddBuiltinModule("time", gadtime.Module),
			}},
			sr: []scriptResult{
				{`time := import("time")`, Nil},
				{`time.Second`, gadtime.Module["Second"]},
				{`tmp := time.Second`, Nil},
				{`tmp`, gadtime.Module["Second"]},
				{`time.Second = ""`, Nil},
				{`time.Second`, Str("")},
				{`time.Second = tmp`, Nil},
				{`time.Second`, gadtime.Module["Second"]},
			},
		},
		{
			name:   "globals",
			global: Dict{"g": Str("test")},
			sr: []scriptResult{
				{`global g`, Nil},
				{`return g`, Str("test")},
				{`globals()["g"]`, Str("test")},
			},
		},
		{
			name: "locals",
			args: []Object{Int(1), Int(2)},
			sr: []scriptResult{
				{`var (a, b, c)`, Nil},
				{`a`, Nil},
				{`b`, Nil},
				{`c`, Nil},
			},
		},
		{
			name: "params",
			args: []Object{Int(1), Int(2)},
			sr: []scriptResult{
				{`param (a, b, c)`, Nil},
				{`a`, Int(1)},
				{`b`, Int(2)},
				{`c`, Nil},
			},
		},
		{
			name: "namedParams0",
			sr: []scriptResult{
				{`param (a=1)`, Nil},
				{`a`, Int(1)},
			},
		},
		{
			name: "namedParams1",
			sr: []scriptResult{
				{`param (a=1,b=2)`, Nil},
				{`a`, Int(1)},
				{`b`, Int(2)},
			},
		},
		{
			name:      "namedParams2",
			namedArgs: NewNamedArgs(KeyValueArray{&KeyValue{Str("b"), Int(3)}}),
			sr: []scriptResult{
				{`param (a=1,b=2)`, Nil},
				{`a`, Int(1)},
				{`b`, Int(3)},
			},
		},
		{
			name:      "namedParams3",
			namedArgs: NewNamedArgs(KeyValueArray{&KeyValue{Str("b"), Int(3)}, &KeyValue{Str("c"), Int(4)}}),
			sr: []scriptResult{
				{`param (a=1,b=2,**other)`, Nil},
				{`a`, Int(1)},
				{`b`, Int(3)},
				{`str(other)`, Str("(;c=4)")},
			},
		},
		{
			name: "paramsAndNamedParams0",
			sr: []scriptResult{
				{`param (a;b=1)`, Nil},
				{`a`, Nil},
				{`b`, Int(1)},
			},
		},
		{
			name:      "paramsAndNamedParams1",
			namedArgs: NewNamedArgs(KeyValueArray{&KeyValue{Str("c"), Int(4)}}),
			sr: []scriptResult{
				{`param (a;b=1,**other)`, Nil},
				{`a`, Nil},
				{`b`, Int(1)},
				{`str(other)`, Str("(;c=4)")},
			},
		},
		{
			name:      "paramsAndNamedParams2",
			namedArgs: NewNamedArgs(KeyValueArray{&KeyValue{Str("c"), Int(4)}, &KeyValue{Str("d"), Int(5)}}),
			sr: []scriptResult{
				{`param (a;b=1,c=2,**other)`, Nil},
				{`a`, Nil},
				{`b`, Int(1)},
				{`c`, Int(4)},
				{`str(other)`, Str("(;d=5)")},
			},
		},
		{
			name: "paramsAndNamedParams3",
			args: []Object{Int(1), Int(2)},
			sr: []scriptResult{
				{`param (a, b, c;d=100,e=10,**other)`, Nil},
				{`a`, Int(1)},
				{`b`, Int(2)},
				{`c`, Nil},
				{`d`, Int(100)},
				{`e`, Int(10)},
				{`str(other)`, Str("(;)")},
			},
		},
		{
			name:      "paramsAndNamedParams4",
			args:      []Object{Int(1), Int(2)},
			namedArgs: NewNamedArgs(KeyValueArray{&KeyValue{Str("e"), Int(6)}, &KeyValue{Str("f"), Int(7)}}),
			sr: []scriptResult{
				{`param (a, b, c;d=100,e=10,**other)`, Nil},
				{`a`, Int(1)},
				{`b`, Int(2)},
				{`c`, Nil},
				{`d`, Int(100)},
				{`e`, Int(6)},
				{`str(other)`, Str("(;f=7)")},
			},
		},
		{
			name:      "paramsAndNamedParams5",
			args:      []Object{Int(1), Int(2), Int(3)},
			namedArgs: NewNamedArgs(KeyValueArray{&KeyValue{Str("e"), Int(6)}, &KeyValue{Str("f"), Int(7)}}),
			sr: []scriptResult{
				{`param (a, *otherArgs;**other)`, Nil},
				{`str(otherArgs)`, Str("[2, 3]")},
			},
		},
		{
			name:      "paramsAndNamedParams6",
			args:      []Object{Int(1), Int(2), Int(3)},
			namedArgs: NewNamedArgs(KeyValueArray{&KeyValue{Str("e"), Int(6)}, &KeyValue{Str("f"), Int(7)}}),
			sr: []scriptResult{
				{`param (a, *otherArgs;d=100,e=10,**other)`, Nil},
				{`a`, Int(1)},
				{`str(otherArgs)`, Str("[2, 3]")},
				{`d`, Int(100)},
				{`e`, Int(6)},
				{`str(other)`, Str("(;f=7)")},
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			eval := NewEval(tC.opts, &RunOpts{Globals: tC.global, Args: Args{tC.args}, NamedArgs: tC.namedArgs})
			for _, sr := range tC.sr {
				ret, _, err := eval.Run(tC.ctx, []byte(sr.script))
				require.NoError(t, err, sr.script)
				require.Equal(t, sr.result, ret, sr.script)
			}
		})
	}

	// test context
	t.Run("context", func(t *testing.T) {
		globals := Dict{
			"Gosched": &Function{
				Value: func(Call) (Object, error) {
					runtime.Gosched()
					return Nil, nil
				},
			},
		}
		eval := NewEval(DefaultCompileOptions, &RunOpts{Globals: globals})
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		ret, bc, err := eval.Run(ctx, []byte(`
		global Gosched; Gosched(); foo := "bar"; return foo`))
		require.Nilf(t, ret, "return value:%v", ret)
		require.Equal(t, context.Canceled, err, err)
		require.NotNil(t, bc)
	})

	// test error
	t.Run("parser error", func(t *testing.T) {
		eval := NewEval(DefaultCompileOptions)
		ret, bc, err := eval.Run(context.Background(), []byte(`...`))
		require.Nil(t, ret)
		require.Nil(t, bc)
		require.Contains(t, err.Error(),
			`Parse Error: expected statement, found '.'`)
	})
}
