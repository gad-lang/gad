package json_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/gad-lang/gad"
	. "github.com/gad-lang/gad/stdlib/json"
)

func TestModuleTypes(t *testing.T) {
	ret, err := ToObject(json.RawMessage(nil))
	require.NoError(t, err)
	require.Equal(t, &RawMessage{Value: Bytes{}}, ret)

	ret, err = ToObject(json.RawMessage([]byte("null")))
	require.NoError(t, err)
	require.Equal(t, &RawMessage{Value: Bytes([]byte("null"))}, ret)

	iface := ToInterface(ret)
	require.Equal(t, json.RawMessage([]byte("null")), iface)
}

func TestScript(t *testing.T) {
	catchf := func(s string, args ...any) string {
		return fmt.Sprintf(`
		json := import("json")
		try {
			return %s
		} catch err {
			return str(err)
		}
		`, fmt.Sprintf(s, args...))
	}
	scriptf := func(s string, args ...any) string {
		return fmt.Sprintf(`
		json := import("json")
		return %s
		`, fmt.Sprintf(s, args...))
	}
	errnarg := func(want, got int) Str {
		return Str(ErrWrongNumArguments.NewError(
			fmt.Sprintf("want=%d got=%d", want, got),
		).ToString())
	}

	expectRun(t, scriptf(""), nil, Nil)

	for key, val := range Module {
		expectRun(t, scriptf("typeName(json.%s)", key), nil, Str("function"))
		expectRun(t, scriptf("str(json.%s)", key), nil, Str(fmt.Sprintf(ReprQuote("function:%s"), key)))
		require.NotNil(t, val)
		require.NotNil(t, val.(*Function).Value)
	}

	expectRun(t, catchf(`json.Marshal()`), nil, errnarg(1, 0))
	expectRun(t, catchf(`typeName(json.Marshal(nil))`), nil, Str("bytes"))
	expectRun(t, catchf(`str(json.Marshal(nil))`), nil, Str("null"))
	expectRun(t, catchf(`str(json.Marshal(error("test")))`), nil, Str("")) // ignore error
	expectRun(t, catchf(`str(json.Marshal(true))`), nil, Str("true"))
	expectRun(t, catchf(`str(json.Marshal(false))`), nil, Str("false"))
	expectRun(t, catchf(`str(json.Marshal(1))`), nil, Str("1"))
	expectRun(t, catchf(`str(json.Marshal(2u))`), nil, Str("2"))
	expectRun(t, catchf(`str(json.Marshal(3.4))`), nil, Str("3.4"))
	expectRun(t, catchf(`str(json.Marshal(3.4d))`), nil, Str("3.4"))
	expectRun(t, catchf(`str(json.Marshal('x'))`), nil, Str("120"))
	expectRun(t, catchf(`str(json.Marshal("test"))`), nil, Str(`"test"`))
	expectRun(t, catchf(`str(json.Marshal(bytes(0,1)))`), nil, Str(`"AAE="`))
	expectRun(t, catchf(`str(json.Marshal([]))`), nil, Str("[]"))
	expectRun(t, catchf(`str(json.Marshal([1, "a", 2u, 'x',3.4,3.4d,true,false,
	{a:[],"b":0,ç:nil},bytes(0,1),
	]))`), nil, Str(`[1,"a",2,120,3.4,3.4,true,false,{"a":[],"b":0,"ç":null},"AAE="]`))
	expectRun(t, catchf(`str(json.Marshal({}))`), nil, Str("{}"))
	expectRun(t, catchf(`str(json.Marshal({_: 1, k2:[3,true,"a"]}))`),
		nil, Str(`{"_":1,"k2":[3,true,"a"]}`))

	expectRun(t, catchf(`json.IndentCount()`), nil, errnarg(3, 0))
	expectRun(t, catchf(`str(json.IndentCount("[1,2]", "", " "))`), nil, Str("[\n 1,\n 2\n]"))

	expectRun(t, catchf(`json.MarshalIndent()`), nil, errnarg(3, 0))
	expectRun(t, catchf(`str(json.MarshalIndent({a: 1, b: [2, true, "<"]},"", " "))`),
		nil, Str("{\n \"a\": 1,\n \"b\": [\n  2,\n  true,\n  \"\\u003c\"\n ]\n}"))

	expectRun(t, catchf(`json.Compact()`), nil, errnarg(2, 0))
	expectRun(t, catchf(`str(json.Compact(json.Marshal(json.NoEscape(["<",">"])), true))`),
		nil, Str(`["\u003c","\u003e"]`))
	expectRun(t, catchf(`str(json.Compact(json.MarshalIndent({a: 1, b: [2, true, "<"]},"", " "), true))`),
		nil, Str(`{"a":1,"b":[2,true,"\u003c"]}`))
	expectRun(t, catchf(`str(json.Compact(json.MarshalIndent(json.NoEscape(["<",">"]), "", " "), false))`),
		nil, Str(`["<",">"]`))

	expectRun(t, catchf(`json.RawMessage()`), nil, errnarg(1, 0))
	expectRun(t, catchf(`str(json.RawMessage(json.Marshal([1, 2])))`),
		nil, Str("[1,2]"))
	expectRun(t, catchf(`str(json.Marshal(json.RawMessage(json.Marshal([1, 2]))))`),
		nil, Str("[1,2]"))

	expectRun(t, catchf(`json.Quote()`), nil, errnarg(1, 0))
	expectRun(t, catchf(`json.NoQuote()`), nil, errnarg(1, 0))
	expectRun(t, catchf(`json.NoEscape()`), nil, errnarg(1, 0))
	expectRun(t, catchf(`str(json.Marshal(json.Quote([1,2,"a"])))`),
		nil, Str(`["1","2","\"a\""]`))
	expectRun(t, catchf(`str(json.Marshal(json.Quote([1,2,{a:"x"}])))`),
		nil, Str(`["1","2",{"a":"\"x\""}]`))
	expectRun(t, catchf(`str(json.Marshal(json.Quote([1,2,{a:json.NoQuote("x")}])))`),
		nil, Str(`["1","2",{"a":"x"}]`))

	expectRun(t, catchf(`json.Unmarshal()`), nil, errnarg(1, 0))
	expectRun(t, catchf(`json.Unmarshal("[1,1.5,true,false,\"x\",{\"a\":\"b\"}]")`),
		nil, Array{Int(1), Float(1.5), True, False, Str("x"), Dict{"a": Str("b")}})
	expectRun(t, catchf(`json.Unmarshal("[1,1.5,true,false,\"x\",{\"a\":\"b\"}]";intAsDecimal)`),
		nil, Array{DecimalFromFloat(1), Float(1.5), True, False, Str("x"), Dict{"a": Str("b")}})
	expectRun(t, catchf(`json.Unmarshal("[1,1.5,true,false,\"x\",{\"a\":\"b\"}]";floatAsDecimal)`),
		nil, Array{Int(1), DecimalFromFloat(1.5), True, False, Str("x"), Dict{"a": Str("b")}})
	expectRun(t, catchf(`json.Unmarshal("[1,1.5,true,false,\"x\",{\"a\":\"b\"}]";numberAsDecimal)`),
		nil, Array{DecimalFromFloat(1), DecimalFromFloat(1.5), True, False, Str("x"), Dict{"a": Str("b")}})

	expectRun(t, catchf(`json.Valid()`), nil, errnarg(1, 0))
	expectRun(t, catchf(`json.Valid("{}")`), nil, True)
	expectRun(t, catchf(`json.Valid("{")`), nil, False)

	expectRun(t, catchf(`str(json.Marshal(json.NoEscape(json.Quote("<"))))`), nil, Str(`"\"<\""`))
	expectRun(t, catchf(`str(json.Marshal(json.NoQuote(json.NoEscape("<"))))`), nil, Str(`"<"`))
	expectRun(t, catchf(`str(json.Marshal(json.Quote(json.NoEscape("<"))))`), nil, Str(`"\"<\""`))

	expectRun(t, catchf(`str(json.Unmarshal(bytes(0)))`),
		nil, Str(`error: invalid character '\x00' looking for beginning of value`))
	expectRun(t, catchf(`str(json.IndentCount(bytes(0), "", " "))`),
		nil, Str(`error: invalid character '\x00' looking for beginning of value`))
	expectRun(t, catchf(`str(json.Compact(bytes(0), true))`),
		nil, Str(`error: invalid character '\x00' looking for beginning of value`))
}

func TestCycle(t *testing.T) {
	expectRun(t, `json:=import("json");a:=[1,2];a[1]=a;return str(json.Marshal(a))`,
		nil, Str(`error: json: unsupported value: encountered a cycle via array`))
	expectRun(t, `json:=import("json");a:=[1,2];a[1]=a;return str(json.MarshalIndent(a,""," "))`,
		nil, Str(`error: json: unsupported value: encountered a cycle via array`))
	expectRun(t, `json:=import("json");m:={a:1};m.b=m;return str(json.Marshal(m))`,
		nil, Str(`error: json: unsupported value: encountered a cycle via dict`))
	expectRun(t, `param m;json:=import("json");m.b=m;return str(json.Marshal(m))`,
		newOpts().Args(&SyncDict{Value: Dict{}}),
		Str(`error: json: unsupported value: encountered a cycle via syncDict`))

	ptr := &ObjectPtr{}
	var m Object = Dict{}
	m.(Dict)["a"] = ptr
	ptr.Value = &m
	_, err := Marshal(nil, ptr)
	require.Error(t, err)
	require.Contains(t, err.Error(), `json: unsupported value: encountered a cycle via objectPtr`)
}

type Opts struct {
	global IndexGetSetter
	args   []Object
}

func newOpts() *Opts {
	return &Opts{}
}

func (o *Opts) Args(args ...Object) *Opts {
	o.args = args
	return o
}

func (o *Opts) Globals(g IndexGetSetter) *Opts {
	o.global = g
	return o
}

func expectRun(t *testing.T, script string, opts *Opts, expected Object) {
	t.Helper()
	if opts == nil {
		opts = newOpts()
	}
	mm := NewModuleMap()
	mm.AddBuiltinModule("json", Module)
	c := CompileOptions{CompilerOptions: DefaultCompilerOptions}
	c.ModuleMap = mm
	bc, err := Compile([]byte(script), c)
	require.NoError(t, err)
	ret, err := NewVM(bc).RunOpts(&RunOpts{Globals: opts.global, Args: Args{opts.args}})
	require.NoError(t, err)
	require.Equal(t, expected, ret)
}
