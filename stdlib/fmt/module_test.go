package fmt_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/gad-lang/gad"
	. "github.com/gad-lang/gad/stdlib/fmt"
)

func Example() {
	exampleRun(`
	fmt := import("fmt")
	fmt.print("print_")
	fmt.println("line")
	fmt.println("a", "b", 3)
	fmt.printf("%v\n", [1, 2])
	fmt.println(fmt.sprint("x", "y", 4))

	a1 := fmt.scanArg("str")
	a2 := fmt.scanArg("int")
	r := fmt.sscanf("abc 123", "%s%d", a1, a2)
	fmt.println(r)
	fmt.println(bool(a1), a1.Value)
	fmt.println(bool(a2), a2.Value)
	`)
	// Output:
	// print_line
	// a b 3
	// [1, 2]
	// xy4
	// 2
	// true abc
	// true 123
}

func TestScript(t *testing.T) {
	testCases := []struct {
		s string
		r Object
	}{
		// scan
		{
			s: `return str(fmt.scanArg())`,
			r: Str(ReprQuote("scanArg")),
		},
		{
			s: `return typeName(fmt.scanArg())`,
			r: Str("scanArg"),
		},
		{
			s: `
		a1 := fmt.scanArg()
		ret := fmt.sscan("abc", a1)
		return ret, bool(a1), a1.Value
			`,
			r: Array{Int(1), True, Str("abc")},
		},
		{
			s: `
		a1 := fmt.scanArg()
		ret := fmt.sscan("abc xyz", a1)
		return ret, bool(a1), a1.Value
			`,
			r: Array{Int(1), True, Str("abc")},
		},
		{
			s: `
		a1 := fmt.scanArg()
		a2 := fmt.scanArg()
		ret := fmt.sscan("abc xyz", a1, a2)
		return [
			ret,
			[bool(a1), a1.Value],
			[bool(a2), a2.Value],
		]
			`,
			r: Array{
				Int(2),
				Array{True, Str("abc")},
				Array{True, Str("xyz")},
			},
		},
		{
			s: `
		a1 := fmt.scanArg("str")
		a2 := fmt.scanArg("int")
		a3 := fmt.scanArg("uint")
		a4 := fmt.scanArg("float")
		a5 := fmt.scanArg("char")
		a6 := fmt.scanArg("bool")
		a7 := fmt.scanArg("bytes")
		ret := fmt.sscan("abc 1 2 3.4 5 t bytes", 
			a1, a2, a3, a4, a5, a6, a7)
		return [
			ret,
			[bool(a1), a1.Value],
			[bool(a2), a2.Value],
			[bool(a3), a3.Value],
			[bool(a4), a4.Value],
			[bool(a5), a5.Value],
			[bool(a6), a6.Value],
			[bool(a7), a7.Value],
		]
			`,
			r: Array{
				Int(7),
				Array{True, Str("abc")},
				Array{True, Int(1)},
				Array{True, Uint(2)},
				Array{True, Float(3.4)},
				Array{True, Char(5)},
				Array{True, True},
				Array{True, Bytes("bytes")},
			},
		},
		{
			s: `
		a1 := fmt.scanArg(str)
		a2 := fmt.scanArg(int)
		a3 := fmt.scanArg(uint)
		a4 := fmt.scanArg(float)
		a5 := fmt.scanArg(char)
		a6 := fmt.scanArg(bool)
		a7 := fmt.scanArg(bytes)
		ret := fmt.sscan("abc 1 2 3.4 5 t bytes", 
			a1, a2, a3, a4, a5, a6, a7)
		return [
			ret,
			[bool(a1), a1.Value],
			[bool(a2), a2.Value],
			[bool(a3), a3.Value],
			[bool(a4), a4.Value],
			[bool(a5), a5.Value],
			[bool(a6), a6.Value],
			[bool(a7), a7.Value],
		]
			`,
			r: Array{
				Int(7),
				Array{True, Str("abc")},
				Array{True, Int(1)},
				Array{True, Uint(2)},
				Array{True, Float(3.4)},
				Array{True, Char(5)},
				Array{True, True},
				Array{True, Bytes("bytes")},
			},
		},
		{
			s: `
		a1 := fmt.scanArg()
		a2 := fmt.scanArg()
		a3 := fmt.scanArg()
		ret := fmt.sscan("abc xyz", a1, a2, a3)
		return [
			str(ret),
			[bool(a1), a1.Value],
			[bool(a2), a2.Value],
			[bool(a3), a3.Value],
		]
			`,
			r: Array{
				Str("error: EOF"),
				Array{True, Str("abc")},
				Array{True, Str("xyz")},
				Array{False, Nil},
			},
		},
		{
			s: `
		a1 := fmt.scanArg("str")
		a2 := fmt.scanArg("int")
		a3 := fmt.scanArg("int")
		ret := fmt.sscanf("abc 3 15", "%s%d", a1, a2, a3)
		return [
			str(ret),
			[bool(a1), a1.Value],
			[bool(a2), a2.Value],
			[bool(a3), a3.Value],
		]
			`,
			r: Array{
				Str("error: too many operands"),
				Array{True, Str("abc")},
				Array{True, Int(3)},
				Array{False, Nil},
			},
		},
		{
			s: `
		a1 := fmt.scanArg("str")
		a2 := fmt.scanArg("int")
		a3 := fmt.scanArg("float")
		ret := fmt.sscanln("abc 3\n1.5", a1, a2, a3)
		return [
			str(ret),
			[bool(a1), a1.Value],
			[bool(a2), a2.Value],
			[bool(a3), a3.Value],
		]
			`,
			r: Array{
				Str("error: unexpected newline"),
				Array{True, Str("abc")},
				Array{True, Int(3)},
				Array{False, Nil},
			},
		},
		// sprint
		{
			s: `return fmt.sprint(1, 2, "c", 'd')`,
			r: Str("1 2c100"),
		},
		{
			s: `return fmt.sprintf("%.1f%s%c%d", 1.2, "abc", 'e', 18u)`,
			r: Str("1.2abce18"),
		},
		{
			s: `return fmt.sprintln(1.2, "abc", 'e', 18u)`,
			r: Str("1.2 abc 101 18\n"),
		},
		// runtime errors
		{
			s: `
		try {
			fmt.printf()
		} catch err {
			return str(str(err.cause))
		}
			`,
			r: Str("WrongNumberOfArgumentsError: want>=1 got=0"),
		},
		{
			s: `
		try {
			fmt.sprintf()
		} catch err {
			return str(err.cause)
		}
			`,
			r: Str("WrongNumberOfArgumentsError: want>=1 got=0"),
		},
		{
			s: `
		try {
			arg := fmt.scanArg("unknown")
		} catch err {
			return str(err.cause)
		}
			`,
			r: Str("TypeError: \"unknown\" not implemented"),
		},
		{
			s: `
		try {
			arg := fmt.sscan()
		} catch err {
			return str(err.cause)
		}
			`,
			r: Str("WrongNumberOfArgumentsError: want>=2 got=0"),
		},
		{
			s: `
		try {
			arg := fmt.sscanf()
		} catch err {
			return str(err.cause)
		}
			`,
			r: Str("WrongNumberOfArgumentsError: want>=3 got=0"),
		},
		{
			s: `
		try {
			arg := fmt.sscanln()
		} catch err {
			return str(err.cause)
		}
			`,
			r: Str("WrongNumberOfArgumentsError: want>=2 got=0"),
		},
		{
			s: `
		try {
			arg := fmt.sscanf("", "", 1)
		} catch err {
			return str(err.cause)
		}
			`,
			r: Str("TypeError: invalid type for argument '2': expected ScanArg interface, found int"),
		},
		{
			s: `
		try {
			arg := fmt.sscanln("", 1)
		} catch err {
			return str(err.cause)
		}
			`,
			r: Str("TypeError: invalid type for argument '1': expected ScanArg interface, found int"),
		},
	}
	for _, tC := range testCases {
		if tC.s == "" {
			return
		}
		expectRun(t, tC.s, tC.r)
	}
}

func expectRun(t *testing.T, script string, expected Object) {
	t.Helper()

	script = `
		fmt := import("fmt")
	` + script

	ret, err := run(script)
	require.NoError(t, err, script)
	require.Equal(t, expected, ret, script)
}

func exampleRun(script string) {
	if _, err := run(script); err != nil {
		panic(err)
	}
}

func run(script string) (ret Object, err error) {
	mm := NewModuleMap()
	mm.AddBuiltinModuleInit("fmt", ModuleInit)
	c := CompileOptions{CompilerOptions: DefaultCompilerOptions}
	c.ModuleMap = mm

	builtins := NewBuiltins().Build()
	_, bc, err := Compile(NewSymbolTable(builtins.Builtins().NameSet), []byte(script), c)
	if err != nil {
		return
	}
	return NewVM(builtins, bc).Run()
}
