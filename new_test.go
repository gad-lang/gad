// Put relatively new features' tests in this test file.

package gad_test

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	. "github.com/gad-lang/gad"
	gadtime "github.com/gad-lang/gad/stdlib/time"
)

// compileFunc compiles src (which must `return func(){…}`) and returns the root
// VM plus the produced CompiledFunction, ready to be driven by an Invoker.
func compileFunc(t *testing.T, src string) (*VM, *CompiledFunction) {
	t.Helper()
	_, bc, err := Compile(NewSymbolTable(NewBuiltins().NameSet), []byte(src), DefaultCompileOptions)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	vm := NewVM(NewBuiltins().Build(), bc)
	ret, err := vm.Run()
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	fn, ok := ret.(*CompiledFunction)
	if !ok {
		t.Fatalf("want *CompiledFunction, got %T", ret)
	}
	return vm, fn
}

func TestInvokerContextCancel(t *testing.T) {
	// A deadline aborts an infinite loop in the invoked function and Invoke
	// returns the context's error promptly.
	t.Run("timeout aborts infinite loop", func(t *testing.T) {
		vm, fn := compileFunc(t, `return func() { x := 0; for { x++ } }`)
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		type result struct {
			err error
			d   time.Duration
		}
		res := make(chan result, 1)
		go func() {
			start := time.Now()
			_, err := NewInvoker(vm, fn).WithContext(ctx).Invoke(Args{}, nil)
			res <- result{err, time.Since(start)}
		}()

		select {
		case r := <-res:
			if !errors.Is(r.err, context.DeadlineExceeded) {
				t.Fatalf("want DeadlineExceeded, got %v", r.err)
			}
			if r.d > 2*time.Second {
				t.Fatalf("abort took too long: %v", r.d)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("Invoke did not abort within 5s — context cancel not wired")
		}
	})

	// An already-cancelled context returns immediately without running.
	t.Run("pre-cancelled context", func(t *testing.T) {
		vm, fn := compileFunc(t, `return func() { x := 0; for { x++ } }`)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := NewInvoker(vm, fn).WithContext(ctx).Invoke(Args{}, nil); !errors.Is(err, context.Canceled) {
			t.Fatalf("want Canceled, got %v", err)
		}
	})

	// The abort cascades through nested invokers: `outer` calls a Go builtin
	// that invokes `inner` (an infinite loop) on a grandchild VM. A deadline on
	// the top invoker aborts from the root, stopping the grandchild too.
	t.Run("timeout propagates to nested invoker", func(t *testing.T) {
		spawn := &Function{FuncName: "spawn", Value: func(c Call) (Object, error) {
			return NewInvoker(c.VM, c.Args.Get(0)).Invoke(Args{}, nil)
		}}
		src := `
global spawn
inner := func() { x := 0; for { x++ } }
return func() { return spawn(inner) }`
		_, bc, err := Compile(NewSymbolTable(NewBuiltins().NameSet), []byte(src), DefaultCompileOptions)
		if err != nil {
			t.Fatalf("compile: %v", err)
		}
		vm := NewVM(NewBuiltins().Build(), bc)
		ret, err := vm.RunOpts(&RunOpts{Globals: Dict{"spawn": spawn}})
		if err != nil {
			t.Fatalf("run: %v", err)
		}
		outer := ret.(*CompiledFunction)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		res := make(chan error, 1)
		go func() {
			_, err := NewInvoker(vm, outer).WithContext(ctx).Invoke(Args{}, nil)
			res <- err
		}()
		select {
		case err := <-res:
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("want DeadlineExceeded, got %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("nested invoke did not abort within 5s")
		}
	})

	// The Caller() path honours the context too: a VMCaller from a context-bound
	// Invoker aborts its infinite loop on the deadline.
	t.Run("caller path timeout", func(t *testing.T) {
		vm, fn := compileFunc(t, `return func() { x := 0; for { x++ } }`)
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		caller, err := NewInvoker(vm, fn).WithContext(ctx).Caller(Args{}, nil)
		if err != nil {
			t.Fatalf("Caller: %v", err)
		}
		res := make(chan error, 1)
		go func() {
			_, err := caller.Call()
			res <- err
		}()
		select {
		case err := <-res:
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("want DeadlineExceeded, got %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("caller did not abort within 5s")
		}
	})

	// A fast call with a live deadline returns its result, unaffected.
	t.Run("fast call unaffected", func(t *testing.T) {
		vm, fn := compileFunc(t, `return func() { return 1 + 2 }`)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		ret, err := NewInvoker(vm, fn).WithContext(ctx).Invoke(Args{}, nil)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if !ret.Equal(Int(3)) {
			t.Fatalf("want 3, got %v", ret)
		}
	})
}

func TestVMPrefixIncDec(t *testing.T) {
	// prefix ++/-- mutate the variable and yield the new value
	testExpectRun(t, `x := 5; return ++x`, nil, Int(6))
	testExpectRun(t, `x := 5; return --x`, nil, Int(4))
	testExpectRun(t, `x := 5; ++x; return x`, nil, Int(6))
	testExpectRun(t, `x := 5; r := ++x; return [r, x]`, nil, Array{Int(6), Int(6)})

	// all numeric types
	testExpectRun(t, `y := 3u; return ++y`, nil, Uint(4))
	testExpectRun(t, `f := 1.5; return ++f`, nil, Float(2.5))
	testExpectRun(t, `d := decimal(2); return --d`, nil, DecimalFromInt(1))

	// usable inside expressions; evaluated left to right
	testExpectRun(t, `i := 0; return ++i + ++i`, nil, Int(3)) // 1 + 2
	testExpectRun(t, `arr := [10, 20, 30]; i := 0; arr[++i] = 99; return arr`,
		nil, Array{Int(10), Int(99), Int(30)})

	// statement form, repeated
	testExpectRun(t, `n := 0; ++n; ++n; --n; return n`, nil, Int(1))

	// the operand must be a variable
	expectErrHas(t, `return ++5`, newOpts().CompilerError(), "requires a variable operand")
	// a non-numeric operand is a runtime type error
	expectErrHas(t, `s := "a"; return ++s`, newOpts(), "invalid type for unary")
}

func TestVMFuncHeaderExpr(t *testing.T) {
	// a `<…>` value is a FunctionHeader describing a signature
	testExpectRun(t, `return typeName(<()>)`, nil, Str("FunctionHeader"))
	testExpectRun(t, `return len((<()>).params)`, nil, Int(0))
	// positional params (each a typedIdent)
	testExpectRun(t, `h := <(a int, b str)>
	return [len(h.params), h.params[0].name, h.params[1].name]`,
		nil, Array{Int(2), Str("a"), Str("b")})
	// param type captured
	testExpectRun(t, `h := <(v int)>
	return h.params[0].types[0] == int`, nil, True)
	// an anonymous header gets an incremented `fh#N` name at compile time
	testExpectRun(t, `h := <(v int) <r bool>>
	return [h.name, len(h.return), h.return[0].name]`,
		nil, Array{Str("fh#1"), Int(1), Str("r")})
	// str renders the module-qualified FullName (MODULE.Name + name)
	testExpectRun(t, `return str(<(a int) <r str>>)`, nil, Str("<(main).fh#1(a int) <r str>>"))
}

func TestVMMethodInterface(t *testing.T) {
	// a `meti { … }` value is a MethodInterface of required headers
	testExpectRun(t, `return typeName(meti { () })`, nil, Str("MethodInterface"))
	// `met <header>` is the single-method shortcut -> a MethodInterface
	testExpectRun(t, `mi := met<(int) <str>>
	return [typeName(mi), len(mi.headers), mi.headers[0].params[0].name, mi.headers[0].return[0].name]`,
		nil, Array{Str("MethodInterface"), Int(1), Str("_"), Str("str")})
	testExpectRun(t, `mi := meti { (), (v int) <int> }
	return [len(mi.headers), mi.headers[1].params[0].name]`, nil, Array{Int(2), Str("v")})
	// a bare positional entry is a type: `(int)` is the unnamed typed param `(_ int)`
	testExpectRun(t, `mi := meti { (int) }
	return [mi.headers[0].params[0].name, mi.headers[0].params[0].types[0] == int]`,
		nil, Array{Str("_"), True})
	// the named statement form binds a const
	testExpectRun(t, `meti S { () <str> }
	return [typeName(S), S.name, len(S.headers)]`, nil, Array{Str("MethodInterface"), Str("S"), Int(1)})

	// implements: matched by parameter arity and assignable types
	testExpectRun(t, `St := meti { () <str> }; f := func() => "x"; return implements(f, St)`,
		nil, True)
	testExpectRun(t, `St := meti { () }; g := func(a) => a; return implements(g, St)`,
		nil, False) // arity mismatch
	testExpectRun(t, `Add := meti { (a int) }; g := func(a int) => a; return implements(g, Add)`,
		nil, True)
	// a single function rarely satisfies two distinct headers
	testExpectRun(t, `St := meti { () }; Ad := meti { (a int) }; g := func(a int) => a
	return implements(g, St, Ad)`, nil, False)

	// merge with `+` and append
	testExpectRun(t, `a := meti { () }; b := meti { (x int) }; return len((a + b).headers)`,
		nil, Int(2))
	testExpectRun(t, `a := meti { () }; b := meti { (x int) }; return len(append(a, b).headers)`,
		nil, Int(2))

	// a func with methods can satisfy several interfaces at once
	testExpectRun(t, `
	St := meti { () <str> }
	Ad := meti { (a int) }
	func m() => "x"
	met m(a int) => a
	return [implements(m, St, Ad), implements(m, St + Ad)]`, nil, Array{True, True})

	// a structural (meti) parameter type dispatches by value: a callable that
	// implements the interface is accepted, a non-callable is rejected even
	// though the param keys as TAny in the dispatch tree.
	testExpectRun(t, `func x(cb met<(int) <int>>) => 1; return x(func(a int) => a)`,
		nil, Int(1))
	expectErrHas(t, `func x(cb met<(int) <int>>) => 1; return x(42)`,
		newOpts(), "invalid type for argument")
}

func TestVMBinaryIncDec(t *testing.T) {
	// the binary form does not disturb the postfix/prefix forms
	testExpectRun(t, `x := 5; x++; return x`, nil, Int(6))
	testExpectRun(t, `x := 5; return ++x`, nil, Int(6))
	// the for-loop post statement `i++` stays postfix (followed by `{`)
	testExpectRun(t, `s := 0; for i := 0; i < 5; i++ { s += i }; return s`, nil, Int(10))

	// `a ++ b` / `a -- b` are binary operators an object can override
	const stack = `
	Stack := Class("Stack"; fields=(; items=(= [])))
	met gad.binOpInc(s Stack, v) { s.items = append(s.items, v); return s }
	met gad.binOpDec(s Stack, i) { return s.items[i] }
	`
	// left-associative chaining: ((s ++ 1) ++ 2) ++ 3
	testExpectRun(t, stack+`s := Stack(); s ++ 1; s ++ 2 ++ 3; return s.items`,
		nil, Array{Int(1), Int(2), Int(3)})
	testExpectRun(t, stack+`s := Stack(); s ++ 10; return s -- 0`, nil, Int(10))

	// numeric types do not handle the binary form (there is no fallback)
	expectErrHas(t, `f := func(a, b) { return a ++ b }; return f(2, 3)`,
		newOpts(), "unsupported operand")
}

func TestVMRange(t *testing.T) {
	collect := func(src string) string { return `return collect(` + src + `)` }

	// `..` ascending and descending integer ranges (inclusive).
	testExpectRun(t, collect(`1 .. 5`), nil, Array{Int(1), Int(2), Int(3), Int(4), Int(5)})
	testExpectRun(t, collect(`5 .. 1`), nil, Array{Int(5), Int(4), Int(3), Int(2), Int(1)})
	// `1 .. 10 / 2` parses as `(1 .. 10) / 2` -> Range(1, 10; step=2).
	testExpectRun(t, collect(`1 .. 10 / 2`), nil, Array{Int(1), Int(3), Int(5), Int(7), Int(9)})
	testExpectRun(t, collect(`(1 .. 10) / 3`), nil, Array{Int(1), Int(4), Int(7), Int(10)})
	// char ranges.
	testExpectRun(t, collect(`'a' .. 'e'`), nil, Array{Char('a'), Char('b'), Char('c'), Char('d'), Char('e')})
	// the Range constructor and its named step.
	testExpectRun(t, collect(`Range(0, 6; step=2)`), nil, Array{Int(0), Int(2), Int(4), Int(6)})
	// `.step()` reads the magnitude; `.step(n)` returns a new range.
	testExpectRun(t, `return (1 .. 10).step()`, nil, Int(1))
	testExpectRun(t, `return (1 .. 10 / 4).step()`, nil, Int(4))
	testExpectRun(t, collect(`(1 .. 10).step(5)`), nil, Array{Int(1), Int(6)})
	// from/to fields.
	testExpectRun(t, `r := 3 .. 9; return [r.from, r.to]`, nil, Array{Int(3), Int(9)})

	// other numeric element kinds.
	testExpectRun(t, collect(`uint(1) .. uint(4)`), nil,
		Array{Uint(1), Uint(2), Uint(3), Uint(4)})
	testExpectRun(t, collect(`1.0 .. 3.0`), nil, Array{Float(1), Float(2), Float(3)})
	testExpectRun(t, collect(`decimal(1) .. decimal(3)`), nil,
		Array{DecimalFromInt(1), DecimalFromInt(2), DecimalFromInt(3)})

	// temporal ranges step by a duration (default one day).
	testExpectRun(t, collect(`2026-01-30D .. 2026-02-02D`), nil, Array{
		NewCalendarDate(2026, 1, 30), NewCalendarDate(2026, 1, 31),
		NewCalendarDate(2026, 2, 1), NewCalendarDate(2026, 2, 2)})
	testExpectRun(t, collect(`(2026-01-30D .. 2026-02-05D) / (dur 48h)`), nil, Array{
		NewCalendarDate(2026, 1, 30), NewCalendarDate(2026, 2, 1),
		NewCalendarDate(2026, 2, 3), NewCalendarDate(2026, 2, 5)})
}

func TestVMRangeRepr(t *testing.T) {
	// repr(Range; indent) dumps the headers of every typed constructor method
	// (one per element kind), tab-indented and sorted by parameter types.
	want := "‹Range: ‹builtin type ‹Range› with 8 methods: [\n" +
		"\t⨍(calendarDate, calendarDate) 🠆 ‹function Range(from calendarDate, to calendarDate; step)›,\n" +
		"\t⨍(calendarTime, calendarTime) 🠆 ‹function Range(from calendarTime, to calendarTime; step)›,\n" +
		"\t⨍(char, char) 🠆 ‹function Range(from char, to char; step)›,\n" +
		"\t⨍(decimal, decimal) 🠆 ‹function Range(from decimal, to decimal; step)›,\n" +
		"\t⨍(float, float) 🠆 ‹function Range(from float, to float; step)›,\n" +
		"\t⨍(int, int) 🠆 ‹function Range(from int, to int; step)›,\n" +
		"\t⨍(time, time) 🠆 ‹function Range(from time, to time; step)›,\n" +
		"\t⨍(uint, uint) 🠆 ‹function Range(from uint, to uint; step)›\n" +
		"]››"
	testExpectRun(t, `return repr(Range; indent)`, nil, Str(want))
}

func TestVMTimeModuleTypesRepr(t *testing.T) {
	// repr(time.<Type>; indent) of every type exported by the time module dumps
	// the headers of its typed single-argument constructor methods (one per
	// accepted input kind), sorted by parameter type.
	dump := func(typ string, kinds ...string) Str {
		lines := make([]string, len(kinds))
		for i, k := range kinds {
			lines[i] = "\t⨍(" + k + ") 🠆 ‹function " + typ + "(v " + k + ")›"
		}
		return Str("‹time." + typ + ": ‹builtin type ‹time." + typ + "› with " +
			strconv.Itoa(len(kinds)) + " methods: [\n" + strings.Join(lines, ",\n") + "\n]››")
	}

	for _, c := range []struct {
		member string
		want   Str
	}{
		{"Type", dump("time", "calendarDate", "calendarTime", "int", "rawstr", "str", "time", "uint")},
		{"CalendarDate", dump("calendarDate", "calendarDate", "calendarTime", "int", "str", "time", "uint")},
		{"CalendarTime", dump("calendarTime", "calendarDate", "calendarTime", "int", "str", "time", "uint")},
		{"Duration", dump("duration", "duration", "int", "str", "uint")},
		{"Location", dump("Location", "Location", "int", "rawstr", "str")},
	} {
		src := `time := import("time"); return repr(time.` + c.member + `; indent)`
		testExpectRun(t, src, newOpts().Module("time", gadtime.ModuleInit), c.want)
	}
}

func TestCoreNamespace(t *testing.T) {
	// `core` is a global namespace (no import needed).
	testExpectRun(t, `return gad.binOpAdd(2, 3)`, nil, Int(5))
	testExpectRun(t, `return gad.binOpMul(4, 5)`, nil, Int(20))

	// `met gad.binOp` adds a typed operator method that the VM dispatches.
	testExpectRun(t, `
	met gad.binOpMul(p str, n int) {
		s := ""
		for i in 1..n { s += p }
		return s
	}
	p := "ab"
	return p * 3`, nil, Str("ababab"))

	// `met gad.selfAssignOp` extends the self-assign fallback.
	testExpectRun(t, `
	met gad.selfAssignOpAdd(a str, b str) { return a + "-" + b }
	x := "a"; y := "b"; x += y
	return x`, nil, Str("a-b"))

	// the old global @binaryOperator / @selfAssignOperator names no longer resolve.
	expectErrHas(t, `return @binaryOperator(1, 1, 1)`,
		newOpts().CompilerError(), `unresolved reference "@binaryOperator"`)
	expectErrHas(t, `return @selfAssignOperator(1, 1, 1)`,
		newOpts().CompilerError(), `unresolved reference "@selfAssignOperator"`)
}

func TestVMOperatorMethods(t *testing.T) {
	// Binary operators dispatch through the gad.binOp methods (which run
	// each type's BinaryOp); results are unchanged.
	testExpectRun(t, `return 1 + 2`, nil, Int(3))
	testExpectRun(t, `return 7 / 2`, nil, Int(3))
	testExpectRun(t, `return 2 ** 8`, nil, DecimalFromInt(256))
	testExpectRun(t, `return "a" + "b"`, nil, Str("ab"))
	testExpectRun(t, `return [1, 2] + [3]`, nil, Array{Int(1), Int(2), Int(3)})
	testExpectRun(t, `return 3.5 * 2.0`, nil, Float(7))
	testExpectRun(t, `return 'a' < 'b'`, nil, True)
	testExpectRun(t, `return uint(6) - uint(2)`, nil, Uint(4))

	// The operator can be invoked directly via the gad.binOp builtin.
	testExpectRun(t, `return gad.binOpAdd(1, 1)`, nil, Int(2))
	testExpectRun(t, `return gad.binOpMul(2, 10)`, nil, Int(20))

	// Self-assign operators dispatch through gad.selfAssignOp: `+=` appends
	// the element, `++=` extends.
	testExpectRun(t, `a := [1, 2]; a += [3, 4]; return a`, nil,
		Array{Int(1), Int(2), Array{Int(3), Int(4)}})
	testExpectRun(t, `a := [1, 2]; a ++= [3, 4]; return a`, nil,
		Array{Int(1), Int(2), Int(3), Int(4)})
	testExpectRun(t, `s := 0; s += 5; return s`, nil, Int(5))

	// A user-defined operator method takes precedence over the built-in one.
	testExpectRun(t, `
	met gad.binOpMul(p str, n int) {
		out := ""
		for i in 1 .. n { out += p }
		return out
	}
	return "ab" * 3`, nil, Str("ababab"))
}

func TestVMUserOperators(t *testing.T) {
	// `<<<`, `>>>` and `%%` have no built-in semantics; they are defined per
	// type with `met gad.binOp`.
	testExpectRun(t, `
	met gad.binOpTripleLess(a int, b int) { return a + b * 100 }
	return 1 <<< 2`, nil, Int(201))
	testExpectRun(t, `
	met gad.binOpTripleGreater(a int, b int) { return a - b }
	x := 9; y := 4; return x >>> y`, nil, Int(5))
	testExpectRun(t, `
	met gad.binOpDoubleMod(a int, b int) { return [a, b] }
	return 5 %% 3`, nil, Array{Int(5), Int(3)})

	// Without a handler the operator is a runtime error (not constant-folded).
	expectErrIs(t, `return 1 <<< 2`, nil, ErrType)

	// The self-assign forms `<<<=` / `>>>=` / `%%=` reuse the binary handler via
	// the gad.selfAssignOp fallback.
	testExpectRun(t, `
	met gad.binOpTripleGreater(a int, b int) { return a * b }
	x := 10; x >>>= 3; return x`, nil, Int(30))
	// ...or a dedicated gad.selfAssignOp handler.
	testExpectRun(t, `
	met gad.selfAssignOpDoubleMod(a int, b int) { return a + b }
	x := 7; x %%= 5; return x`, nil, Int(12))
}

func TestVMMethodOverride(t *testing.T) {
	// `met ~name(…)` re-adds an existing method signature by replacing it
	// instead of erroring; the last definition wins. (Variables defeat the
	// constant-folder so the operator builtin is actually dispatched.)
	testExpectRun(t, `
	a := 1; b := 2
	met ~gad.binOpTripleLess(l int, r int) { return 100 }
	met ~gad.binOpTripleLess(l int, r int) { return 200 }
	return a <<< b`, nil, Int(200))

	// The block form takes `~` per method.
	testExpectRun(t, `
	a := 1; b := 2
	met gad.binOpTripleLess { ~(l int, r int) { return 1 } }
	met gad.binOpTripleLess { ~(l int, r int) { return 2 } }
	return a <<< b`, nil, Int(2))

	// `met ~name { … }` applies the override to every method in the block.
	testExpectRun(t, `
	a := 1; b := 2
	met gad.binOpTripleLess(l int, r int) { return 1 }
	met ~gad.binOpTripleLess { (l int, r int) { return 9 } }
	return a <<< b`, nil, Int(9))

	// Without `~`, re-adding the same signature is a runtime error.
	expectErrIs(t, `
	met gad.binOpTripleLess(l int, r int) { return 1 }
	met gad.binOpTripleLess(l int, r int) { return 2 }
	return 0`, nil, ErrNotIndexable)
}

func TestVMOldOverrideParam(t *testing.T) {
	// A `$old` first parameter captures the method being overridden, so the new
	// method can call the previous implementation (around advice / super).
	testExpectRun(t, `
	func x(i int) => i * 10
	met ~x($old, i int) => $old(i) + 1
	return x(3)`, nil, Int(31))

	// `$old` calls the previous method, not itself (no infinite recursion).
	testExpectRun(t, `
	func f(n int) => n
	met ~f($old, n int) => $old(n) + 10
	met ~f($old, n int) => $old(n) + 100
	return f(1)`, nil, Int(111))

	// Multi-parameter signature: `$old` is resolved by the remaining params.
	testExpectRun(t, `
	func g(a int, b int) => a + b
	met ~g($old, a int, b int) => $old(a, b) * 2
	return g(3, 4)`, nil, Int(14))

	// An untyped `$old` method resolves the previous untyped method.
	testExpectRun(t, `
	func k(v) => "base"
	met ~k($old, v) => $old(v) + "+wrap"
	return k(0)`, nil, Str("base+wrap"))

	// When no previous method exists for the signature, `$old` is nil.
	testExpectRun(t, `
	func h { (s str) => s }
	met h { ~($old, n int) => $old == nil }
	return h(9)`, nil, True)

	// gad.methodFromArgs resolves a method by example value or by type name.
	testExpectRun(t, `
	func p(i int) => i
	met p(s str) => s
	return [gad.methodFromArgs(p, 1)(5), gad.methodFromArgs(p, str)("z")]`,
		nil, Array{Int(5), Str("z")})
}

func TestVMClassOldOverride(t *testing.T) {
	// $old rewrites a class method, wrapping the previous implementation.
	testExpectRun(t, `
	Animal := Class("Animal", (cls, define) => define(;
		fields  = (; name str = "?"),
		methods = [ speak(this) => this.name + " barks" ]))
	d := Animal(; name="Rex")
	met ~Animal.speak($old, this) => $old(this) + " loudly"
	return d.speak()`, nil, Str("Rex barks loudly"))

	// $old rewrites a constructor, delegating to the previous one (a Class is a
	// MethodCaller through its constructor).
	testExpectRun(t, `
	Point := Class("Point", (cls, define) => define(;
		fields = (; x int = 0, y int = 0),
		new { (new, x, y) => new(; x=x, y=y) }))
	met ~Point($old, new, x, y) => $old(new, x * 10, y * 10)
	p := Point(3, 4)
	return [p.x, p.y]`, nil, Array{Int(30), Int(40)})

	// $old rewrites a property setter (met Class.prop routes to the property's
	// getter/setter, not to a shadowing method).
	testExpectRun(t, `
	Box := Class("Box", (cls, define) => define(; fields = (; v), properties = {
		val: func { (this) => this.v; (this, x int) { this.v = "int:" + str(x) } }
	}))
	b := Box()
	met ~Box.val($old, this, x int) { $old(this, x); this.v = this.v + " ok" }
	b.val = 9
	return b.v`, nil, Str("int:9 ok"))
}

func TestVMAssignOperator(t *testing.T) {
	// `obj :: Type` yields obj when assignable to Type.
	testExpectRun(t, `return 5 :: int`, nil, Int(5))
	testExpectRun(t, `return "hi" :: str`, nil, Str("hi"))
	// chains left-to-right, each step returns the same value.
	testExpectRun(t, `return 5 :: int :: any`, nil, Int(5))
	// binds tighter than arithmetic: `2 + 3 :: int` is `2 + (3 :: int)`.
	testExpectRun(t, `return 2 + 3 :: int`, nil, Int(5))

	// a subclass instance is assignable to a parent class.
	testExpectRun(t, `
	Animal := Class("Animal", (cls, define) => define(; fields = (; name str = "")))
	Dog := Class("Dog", (cls, define) => define(; extends = [Animal]))
	d := Dog(; name = "rex")
	return (d :: Animal).name`, nil, Str("rex"))

	// a callable is assignable to a matching method interface (structural).
	testExpectRun(t, `
	f := func(a int) => a
	return (f :: met<(int) <int>>)(6)`, nil, Int(6))

	// not assignable -> ErrIncompatibleAssign (catchable at runtime).
	expectErrIs(t, `return "hi" :: int`, nil, ErrIncompatibleAssign)
	expectErrIs(t, `return 42 :: met<(int) <int>>`, nil, ErrIncompatibleAssign)

	// the RHS must be a type.
	expectErrIs(t, `return 1 :: 2`, nil, ErrType)
}

func TestVMParamTypeErrorPosition(t *testing.T) {
	// An unresolved param type points at the type identifier, not at the
	// enclosing func/param declaration.
	expectErrHas(t, `func x(l V) { }`,
		newOpts().CompilerError(), `unresolved reference "V"`)
	expectErrHas(t, `func x(l V) { }`, newOpts().CompilerError(), `:1:10`)
	expectErrHas(t, `func x { (l V) { } }`, newOpts().CompilerError(), `:1:13`)
	expectErrHas(t, `param (a V)`, newOpts().CompilerError(), `:1:10`)
	expectErrHas(t, `param (;a V)`, newOpts().CompilerError(), `:1:11`)
}

func TestVMSameOperator(t *testing.T) {
	// `===` is strict: no numeric coercion, unlike `==`.
	testExpectRun(t, `return 1 == 1u`, nil, True)
	testExpectRun(t, `return 1 === 1u`, nil, False)
	testExpectRun(t, `return 1 === 1`, nil, True)
	testExpectRun(t, `return 1.0 === 1`, nil, False)
	testExpectRun(t, `return 1.5 === 1.5`, nil, True)
	testExpectRun(t, `return "a" === "a"`, nil, True)
	testExpectRun(t, `return "a" === "b"`, nil, False)
	testExpectRun(t, `return nil === nil`, nil, True)
	testExpectRun(t, `return nil === 0`, nil, False)
	testExpectRun(t, `return 'a' === 'a'`, nil, True)

	// `!==` is `!(a === b)`.
	testExpectRun(t, `return 1 !== 1u`, nil, True)
	testExpectRun(t, `return 1 !== 1`, nil, False)
	testExpectRun(t, `return "a" !== "b"`, nil, True)

	// non-primitive objects compare by identity; each array/dict literal is a
	// fresh object, so equal-looking literals are not the same.
	testExpectRun(t, `a := [1, 2]; return a === a`, nil, True)
	testExpectRun(t, `a := [1, 2]; return a === [1, 2]`, nil, False)
	testExpectRun(t, `a := [1, 2]; b := [1, 2]; return a === b`, nil, False)
	testExpectRun(t, `return [1, 2] === [1, 2]`, nil, False)
	testExpectRun(t, `d := {x: 1}; return d === d`, nil, True)
	testExpectRun(t, `return {x: 1} === {x: 1}`, nil, False)

	// a type can define `===` via met gad.binOp.
	testExpectRun(t, `
	met gad.binOpSame(a str, b int) { return true }
	p := "x"
	return p === 1`, nil, True)
}

func TestVMInOperator(t *testing.T) {
	// Array: value membership.
	testExpectRun(t, `return 2 in [1, 2, 3]`, nil, True)
	testExpectRun(t, `return 9 in [1, 2, 3]`, nil, False)
	// Dict / SyncDict: key membership.
	testExpectRun(t, `return "a" in {a: 1, b: 2}`, nil, True)
	testExpectRun(t, `return "z" in {a: 1}`, nil, False)
	testExpectRun(t, `return "x" in syncDict({x: 1})`, nil, True)
	// KeyValueArray: key membership.
	testExpectRun(t, `return "b" in (;a=1, b=2)`, nil, True)
	// Bytes: byte-value membership ('h' == 104).
	testExpectRun(t, `return [104 in bytes("hi"), 65 in bytes("hi")]`, nil, Array{True, False})
	// Str / RawStr: substring membership; a char needle matches its rune.
	testExpectRun(t, `return 'e' in "hello"`, nil, True)
	testExpectRun(t, `return 'z' in "hello"`, nil, False)
	testExpectRun(t, `return [("ell" in "hello"), ("xyz" in "hello")]`, nil, Array{True, False})
	testExpectRun(t, "return \"ll\" in `hello`", nil, True)

	// Precedence and use as a condition.
	testExpectRun(t, `return 1 + 1 in [2, 3]`, nil, True)
	testExpectRun(t, `out := []; for x in [1, 2, 3] { if x in [2] { out = append(out, x) } }; return out`,
		nil, Array{Int(2)})

	// Fallback: when the right operand is not a Container, `in` goes through the
	// binary-operator handlers, so a type can define it.
	testExpectRun(t, `
	met gad.binOpIn(a int, b str) { return a > 0 }
	return 5 in "anything"`, nil, True)
	// No Container and no handler -> error.
	expectErrIs(t, `return 1 in 2`, nil, ErrType)
}

func TestVMAinOperator(t *testing.T) {
	// `A ain B` is true when every value of A is a member of B.
	testExpectRun(t, `return [1, 2] ain [1, 2, 3]`, nil, True)
	testExpectRun(t, `return [1, 4] ain [1, 2, 3]`, nil, False)
	// An empty left array is vacuously true.
	testExpectRun(t, `return [] ain [1, 2, 3]`, nil, True)
	// A non-array left is treated as a single value (matches `in`).
	testExpectRun(t, `return 2 ain [1, 2, 3]`, nil, True)
	testExpectRun(t, `return 5 ain [1, 2, 3]`, nil, False)
	// Dict key membership and bytes value membership.
	testExpectRun(t, `return ["a", "b"] ain {a: 1, b: 2}`, nil, True)
	testExpectRun(t, `return ["a", "z"] ain {a: 1, b: 2}`, nil, False)
	testExpectRun(t, `return [104, 105] ain bytes("hi")`, nil, True)
	// String: every left value must be a substring (chars match their rune).
	testExpectRun(t, `return ["ell", "hel"] ain "hello"`, nil, True)
	testExpectRun(t, `return ["ell", "xyz"] ain "hello"`, nil, False)
	testExpectRun(t, `return ['h', 'o'] ain "hello"`, nil, True)

	// Precedence: comparison level, like `in`.
	testExpectRun(t, `return [1 + 1] ain [2, 3]`, nil, True)

	// Falls back through `in`, so a Gad type that defines only `in` works.
	testExpectRun(t, `
	met gad.binOpIn(v, b str) { return v > 0 }
	return [1, 2, 3] ain "anything"`, nil, True)

	// A type can intercept `ain` directly, taking precedence over the fallback.
	testExpectRun(t, `
	met gad.binOpAin(a array, b str) { return "custom" }
	return [1, 2] ain "x"`, nil, Str("custom"))

	// gad.binOp is callable directly with the ain operator type.
	testExpectRun(t, `return gad.binOpAin([1, 2], [1, 2, 3])`, nil, True)

	// No `in` support on the right operand -> error.
	expectErrIs(t, `return [1] ain 2`, nil, ErrType)
}

func TestVMWith(t *testing.T) {
	// A resource whose enter/exit hooks record into its own `log` field (the
	// with-protocol: a Gad object provides `enter()` / `exit(err)` methods).
	prelude := `
	Res := Class("Res"; fields = (; log = (= [])), methods = [
		enter(this) { this.log = append(this.log, "enter"); return this }
		exit(this, err) { this.log = append(this.log, "exit") }
		read(this) { return "data" }
	])
	`
	str := func(s string) string { return prelude + s }

	// Bare identifier resource.
	testExpectRun(t, str(`r := Res(); with r { r.log = append(r.log, "body") }; return r.log`),
		nil, Array{Str("enter"), Str("body"), Str("exit")})

	// `as` binding (f aliases the resource).
	testExpectRun(t, str(`r := Res(); with r as f { f.log = append(f.log, "body") }; return r.log`),
		nil, Array{Str("enter"), Str("body"), Str("exit")})

	// `:=` define: the variable is visible after the block.
	testExpectRun(t, str(`with r := Res() { r.log = append(r.log, "body") }; return r.log`),
		nil, Array{Str("enter"), Str("body"), Str("exit")})

	// `=` assign onto a pre-declared variable.
	testExpectRun(t, str(`var r; with r = Res() { r.log = append(r.log, "body") }; return r.log`),
		nil, Array{Str("enter"), Str("body"), Str("exit")})

	// Bare non-identifier resource (bound to an internal temp).
	testExpectRun(t, str(`a := [Res()]; with a[0] { a[0].log = append(a[0].log, "body") }; return a[0].log`),
		nil, Array{Str("enter"), Str("body"), Str("exit")})

	// Nesting: exits run in LIFO order.
	testExpectRun(t, str(`
		o := Res(); i := Res()
		with o { with i { i.log = append(i.log, "body") } }
		return [o.log, i.log]`),
		nil, Array{Array{Str("enter"), Str("exit")}, Array{Str("enter"), Str("body"), Str("exit")}})

	// An error in the body still runs exit and propagates.
	testExpectRun(t, str(`
		r := Res()
		try { with r { throw "boom" } } catch e { }
		return r.log`),
		nil, Array{Str("enter"), Str("exit")})

	// exit receives the block error (nil on success).
	testExpectRun(t, `
		Res := Class("Res"; fields = (; got = (= "?")), methods = [
			exit(this, err) { this.got = str(err) }
		])
		r := Res()
		try { with r { throw "boom" } } catch e { }
		return r.got`,
		nil, Str("error: boom"))

	// Expression form yields the value; exit runs around it.
	testExpectRun(t, str(`r := Res(); v := with r as h: h.read(); return [r.log, v]`),
		nil, Array{Array{Str("enter"), Str("exit")}, Str("data")})

	// A non-resource value is a no-op (enter/exit do nothing; the body still runs).
	testExpectRun(t, str(`a := [1]; with a { a = append(a, 2) }; return a`),
		nil, Array{Int(1), Int(2)})
}

func TestVMToArray(t *testing.T) {
	// toArray yields index=value KeyValue pairs; entries from a custom iterator
	// must be distinct copies, not aliases of the iterator's shared state.
	testExpectRun(t, `return toArray("abc")`, nil, Array{
		&KeyValue{K: Int(0), V: Char('a')},
		&KeyValue{K: Int(1), V: Char('b')},
		&KeyValue{K: Int(2), V: Char('c')},
	})
	testExpectRun(t, `return toArray(1 .. 4)`, nil, Array{
		&KeyValue{K: Int(0), V: Int(1)},
		&KeyValue{K: Int(1), V: Int(2)},
		&KeyValue{K: Int(2), V: Int(3)},
		&KeyValue{K: Int(3), V: Int(4)},
	})
	// arrays pass through unchanged.
	testExpectRun(t, `return toArray([1, 2, 3])`, nil, Array{Int(1), Int(2), Int(3)})
}

func TestVMPowFractional(t *testing.T) {
	// integer powers are unchanged (int**int and decimal yield a decimal)
	testExpectRun(t, `return 2 ** 10`, nil, DecimalFromInt(1024))
	testExpectRun(t, `return decimal(2) ** 10`, nil, DecimalFromInt(1024))
	// a fractional exponent on a decimal base falls back to float (regression:
	// decimal.Pow truncated the exponent, so `** 0.5` wrongly yielded 1)
	testExpectRun(t, `return decimal(25) ** 0.5`, nil, Float(5))
	testExpectRun(t, `return 25.0 ** 0.5`, nil, Float(5))
	// int**int produces a decimal, so `(a**2 + b**2) ** 0.5` (a hypotenuse) must
	// still compute the square root
	testExpectRun(t, `return (3 ** 2 + 4 ** 2) ** 0.5`, nil, Float(5))
}

func TestVMClassFeatures(t *testing.T) {
	// --- fields: declarations, types (not enforced) and defaults ---
	testExpectRun(t, `
	P := Class("P"; fields=(; a, b int, c = "x", d str = "y"))
	p := P()
	return [p.a, p.b, p.c, p.d]`, nil, Array{Nil, Nil, Str("x"), Str("y")})

	// a computed field default is evaluated fresh for every instance
	testExpectRun(t, `
	n := 0
	C := Class("C"; fields=(; id = (= n++)))
	return [C().id, C().id, C().id]`, nil, Array{Int(1), Int(2), Int(3)})

	// --- methods (overloaded by arity/type) ---
	testExpectRun(t, `
	Calc := Class("Calc"; methods=[
		add(this, a, b) => a + b
		add(this, a) => a + a
	])
	c := Calc()
	return [c.add(2, 3), c.add(5)]`, nil, Array{Int(5), Int(10)})

	// --- constructor `new` with overloads; new(;..) sets fields ---
	testExpectRun(t, `
	Point := Class("Point", (cls, define) => define(; new {
		(new; **f) => new(; x=0, y=0, **f)
		(new, x, y) => new(; x=x, y=y)
	}))
	a := Point()
	b := Point(3, 4)
	c := Point(; x=7)
	return [a.x, a.y, b.x, b.y, c.x, c.y]`,
		nil, Array{Int(0), Int(0), Int(3), Int(4), Int(7), Int(0)})

	// --- properties: getter + (typed) setters ---
	testExpectRun(t, `
	Box := Class("Box", (cls, define) => define(; fields=(; v), properties={
		val: func {
			(this) => this.v
			(this, x) { this.v = "any:" + str(x) }
			(this, x int) { this.v = "int:" + str(x) }
		}
	}))
	b := Box()
	out := []
	b.val = "a"; out = append(out, b.val)
	b.val = 5;   out = append(out, b.val)
	return out`, nil, Array{Str("any:a"), Str("int:5")})

	// --- inheritance: override, inherited method, promoted (anonymous) field ---
	testExpectRun(t, `
	Animal := Class("Animal"; fields=(; name str = "?"), methods=[
		speak(this) => this.name + " makes a sound"
		describe(this) => "I am " + this.name
	])
	Dog := Class("Dog"; extends=[Animal], methods=[ speak(this) => this.name + " barks" ])
	d := Dog(; name="Rex")
	return [d.speak(), d.describe(), d.name]`,
		nil, Array{Str("Rex barks"), Str("I am Rex"), Str("Rex")})

	// promoted field is shared with the embedded parent (set routes to parent)
	testExpectRun(t, `
	Animal := Class("Animal"; fields=(; name str = "?"), methods=[ describe(this) => "I am " + this.name ])
	Dog := Class("Dog"; extends=[Animal])
	d := Dog(; name="Rex")
	d.name = "Buddy"
	return [d.name, d.describe()]`, nil, Array{Str("Buddy"), Str("I am Buddy")})

	// multiple inheritance: methods from several parents are promoted
	testExpectRun(t, `
	A := Class("A"; methods=[ a(this) => "a" ])
	B := Class("B"; methods=[ b(this) => "b" ])
	C := Class("C"; extends=[A, B], methods=[ c(this) => "c" ])
	o := C()
	return [o.a(), o.b(), o.c()]`, nil, Array{Str("a"), Str("b"), Str("c")})

	// --- operator overload + conversion + external method via `met` ---
	testExpectRun(t, `
	Vec := Class("Vec"; fields=(; x int = 0, y int = 0))
	met gad.binOpAdd(a Vec, b Vec) {
		return Vec(; x=a.x+b.x, y=a.y+b.y)
	}
	met str(v Vec) => "(" + v.x + ", " + v.y + ")"
	met Vec.len2(this) => this.x*this.x + this.y*this.y
	a := Vec(; x=1, y=2)
	b := Vec(; x=10, y=20)
	return [str(a + b), a.len2()]`, nil, Array{Str("(11, 22)"), Int(5)})
}

// TestVMClassSyntax exercises the `class` expression/statement syntax, which
// lowers to the Class(...) constructor (see TestVMClassFeatures for the
// equivalent hand-written forms).
func TestVMClassSyntax(t *testing.T) {
	// expression form: fields (incl. typed + default) and a method.
	testExpectRun(t, `
	Point := class {
		x = 0
		y = 0
		methods { dist() => (this.x**2 + this.y**2) ** 0.5 }
	}
	p := Point(; x=3, y=4)
	return [p.x, p.y, p.dist()]`, nil, Array{Int(3), Int(4), Float(5)})

	// statement form desugars to `const Name = class …`.
	testExpectRun(t, `
	class Box {
		v = 1
		methods { get() => this.v }
	}
	return Box(; v=9).get()`, nil, Int(9))

	// fields: defaults and no-default (nil); a typed field is accepted.
	testExpectRun(t, `
	class P { a; b int; c = "x"; d str = "y" }
	p := P()
	return [p.a, p.b, p.c, p.d]`, nil, Array{Nil, Nil, Str("x"), Str("y")})

	// computed field default, evaluated per instance.
	testExpectRun(t, `
	n := 0
	class C { id = (= n++) }
	return [C().id, C().id, C().id]`, nil, Array{Int(1), Int(2), Int(3)})

	// methods: typed `this` enables type/arity overload dispatch.
	testExpectRun(t, `
	class Calc {
		methods {
			add(a, b) => a + b
			add(a) => a + a
			tag(n int) => "int:" + str(n)
			tag(s str) => "str:" + s
		}
	}
	c := Calc()
	return [c.add(2, 3), c.add(5), c.tag(7), c.tag("x")]`,
		nil, Array{Int(5), Int(10), Str("int:7"), Str("str:x")})

	// constructor `new` with overloads; new(;..) sets fields.
	testExpectRun(t, `
	class Point {
		x = 0
		y = 0
		new {
			(; **f) => new(; x=0, y=0, **f)
			(x, y) => new(; x=x, y=y)
		}
	}
	a := Point()
	b := Point(3, 4)
	c := Point(; x=7)
	return [a.x, a.y, b.x, b.y, c.x, c.y]`,
		nil, Array{Int(0), Int(0), Int(3), Int(4), Int(7), Int(0)})

	// properties: getter + (typed) setters.
	testExpectRun(t, `
	class Box {
		v
		props {
			val {
				() => this.v
				(x) { this.v = "any:" + str(x) }
				(x int) { this.v = "int:" + str(x) }
			}
		}
	}
	b := Box()
	out := []
	b.val = "a"; out = append(out, b.val)
	b.val = 5;   out = append(out, b.val)
	return out`, nil, Array{Str("any:a"), Str("int:5")})

	// getter shortcut `name = expr` is a zero-arg accessor.
	testExpectRun(t, `
	class T { props { greeting = "hi" } }
	return T().greeting`, nil, Str("hi"))

	// inheritance: override + inherited method + promoted field.
	testExpectRun(t, `
	class Animal { name str = "?"; methods {
		speak() => this.name + " makes a sound"
		describe() => "I am " + this.name
	} }
	class Dog { *Animal; methods { speak() => this.name + " barks" } }
	d := Dog(; name="Rex")
	return [d.speak(), d.describe(), d.name]`,
		nil, Array{Str("Rex barks"), Str("I am Rex"), Str("Rex")})

	// expression-form class is a first-class value usable inline.
	testExpectRun(t, `
	return (class { v = 1; methods { go() => this.v + 1 } })(; v=4).go()`,
		nil, Int(5))
}

// TestVMEnum exercises the `enum` syntax, which builds a compile-time Enum
// constant. Values are read back via `.value`.
func TestVMEnum(t *testing.T) {
	// default increment (uint); first field is 1.
	testExpectRun(t, `e := enum { Read, Write, Exec }
	return [e.Read.value, e.Write.value, e.Exec.value]`,
		nil, Array{Uint(1), Uint(2), Uint(3)})

	// explicit value, then resume incrementing from it.
	testExpectRun(t, `e := enum { Read = 10, Write }
	return [e.Read.value, e.Write.value]`, nil, Array{Int(10), Int(11)})

	// `+`/`-` signs: signed ints, sign propagates to defaulted fields.
	testExpectRun(t, `e := enum { -Read, Write, +List, Delete }
	return [e.Read.value, e.Write.value, e.List.value, e.Delete.value]`,
		nil, Array{Int(-1), Int(-2), Int(3), Int(4)})

	// bit mode: 1<<n; a later field may combine earlier ones.
	testExpectRun(t, `e := enum { bit List, Detail, Create, Edit, Read = List | Detail }
	return [e.List.value, e.Detail.value, e.Create.value, e.Edit.value, e.Read.value]`,
		nil, Array{Uint(1), Uint(2), Uint(4), Uint(8), Uint(3)})

	// a value expression may reference earlier fields.
	testExpectRun(t, `e := enum { Read, Write, All = Read + Write }
	return [e.Read.value, e.Write.value, e.All.value]`,
		nil, Array{Uint(1), Uint(2), Uint(3)})

	// `_` advances the running value but is not added.
	testExpectRun(t, `e := enum { _, Read, Write }
	return [e.Read.value, e.Write.value]`, nil, Array{Uint(2), Uint(3)})
	testExpectRun(t, `e := enum { _ = 10u, Read, Write }
	return [e.Read.value, e.Write.value]`, nil, Array{Uint(11), Uint(12)})
	testExpectRun(t, `e := enum { Read, _ = 6, Write }
	return [e.Read.value, e.Write.value]`, nil, Array{Uint(1), Int(7)})

	// statement form binds a constant; members carry name/index/value.
	testExpectRun(t, `enum Perm { Read, Write, Exec = 10 }
	return [Perm.Exec.name, Perm.Exec.index, Perm.Exec.value]`,
		nil, Array{Str("Exec"), Int(2), Int(10)})

	// indexGet, len via iteration and str.
	testExpectRun(t, `enum Perm { Read, Write }
	out := []
	for k, v in Perm { out = append(out, k + "=" + str(v.value)) }
	return out`, nil, Array{Str("Read=1"), Str("Write=2")})

	// indexGet of an unknown member errors.
	expectErrIs(t, `enum Perm { Read }
	return Perm["Nope"]`, nil, ErrInvalidIndex)
}

func TestVMDeferStmt(t *testing.T) {
	// runs after the body
	testExpectRun(t, `
	out := ""
	f := func() { defer { out += "d" }; out += "b" }
	f()
	return out`, nil, Str("bd"))

	// $ret can be modified by a defer
	testExpectRun(t, `f := func(x) { defer { $ret = $ret + 100 }; return x }; return f(5)`,
		nil, Int(105))

	// LIFO order, $ret threaded across defers
	testExpectRun(t, `f := func() { defer { $ret += "-A" }; defer { $ret += "-B" }; return "x" }; return f()`,
		nil, Str("x-B-A"))

	// `defer handler` calls the handler; `defer handler(x)` calls with args
	testExpectRun(t, `
	out := ""
	h := func(m) { out += m }
	f := func() { defer h("done"); defer h("step") }
	f()
	return out`, nil, Str("stepdone"))

	// defer_err recovers: sets $ret and clears $err
	testExpectRun(t, `
	f := func() { defer_err { $ret = "recovered:" + str($err); $err = nil }; throw "boom" }
	return f()`, nil, Str("recovered:error: boom"))

	// defer_ok runs only on success
	testExpectRun(t, `
	out := ""
	f := func(fail) {
		defer_ok { out += "ok " }
		defer_err { out += "err " }
		if fail { throw "x" }
	}
	f(false)
	try { f(true) } catch {}
	return out`, nil, Str("ok err "))

	// throw inside a defer becomes the function's error
	expectErrHas(t, `f := func() { defer { throw "from-defer" }; return 1 }; return f()`,
		newOpts(), "from-defer")

	// an error from the body still propagates when no defer suppresses it
	expectErrHas(t, `f := func() { defer { $ret = 1 }; throw "boom" }; return f()`,
		newOpts(), "boom")

	// nested functions have independent defers (inner runs at inner exit)
	testExpectRun(t, `
	out := ""
	g := func() {
		defer { out += "outer " }
		inner := func() { defer { out += "inner " }; out += "in " }
		inner()
		out += "out "
	}
	g()
	return out`, nil, Str("in inner out outer "))

	// conditional defer only registers (and runs) when reached
	testExpectRun(t, `
	f := func(c) { if c { defer { $ret += "y" } }; return "b" }
	return [f(true), f(false)]`, nil, Array{Str("by"), Str("b")})

	// shortcut form: assignment to $ret (no braces)
	testExpectRun(t, `f := func() { defer $ret += 100; return 1 }; return f()`,
		nil, Int(101))

	// shortcut form: a call receiving $ret and $err as arguments
	testExpectRun(t, `
	out := ""
	rec := func(r, e) { out += "r=" + str(r) + " e=" + str(e) }
	f := func() { defer rec($ret, $err); return 7 }
	f()
	return out`, nil, Str("r=7 e=nil"))

	// shortcut form: defer_ok with an assignment runs only on success
	testExpectRun(t, `
	f := func(fail) { defer_ok $ret = "ok"; if fail { throw "x" }; return "raw" }
	return f(false)`, nil, Str("ok"))
}

func TestVMCodeStr(t *testing.T) {
	// inline form: body becomes a plain str
	testExpectRun(t, `return code a + b end`, nil, Str("a + b"))
	testExpectRun(t, `return typeName(code x end)`, nil, Str("str"))

	// block form: the body is captured verbatim (NOT evaluated as code), with the
	// opening line's indentation stripped from every line.
	testExpectRun(t, "x := code\n    a := 1\n    b := 2\nend\nreturn x",
		nil, Str("a := 1\nb := 2"))

	// a deeper-indented `end` belongs to the body; the fence is the `end` at the
	// opening indentation.
	testExpectRun(t,
		"x := code\n    begin\n        y := 1\n    end\nend\nreturn x",
		nil, Str("begin\n    y := 1\nend"))

	// nested in a function: dedent uses the opening line's indentation (4 here)
	testExpectRun(t,
		"f := func() {\n    return code\n        a\n    end\n}\nreturn f()",
		nil, Str("a"))

	// a `code` identifier with no matching `end` fence is unaffected
	testExpectRun(t, "code := 41\nreturn code + 1", nil, Int(42))
}

func TestVMProperty(t *testing.T) {
	// a getter-only property: calling with no args invokes the getter
	testExpectRun(t, `p := Prop("x", () => 42); return p()`, nil, Int(42))
	testExpectRun(t, `return typeName(Prop("x", () => 1))`, nil, Str("Prop"))

	// getter + setter: no args reads, one arg writes
	testExpectRun(t, `
	var v
	p := Prop("x", () => v, (n) => {v = n})
	out := [p()]          // nil (unset)
	p("a")
	out = append(out, p())
	return out`, nil, Array{Nil, Str("a")})

	// a typed setter added with `met` is dispatched by the argument type
	testExpectRun(t, `
	var v
	const p = Prop("x", () => v, (n) => {v = n})
	met p(n int) { v = "int= " + n }
	p("a"); s1 := p()
	p(1);   s2 := p()
	return [s1, s2]`, nil, Array{Str("a"), Str("int= 1")})

	// a property may be created with no methods, but calling it without a
	// matching method is an error
	testExpectRun(t, `return typeName(Prop("x"))`, nil, Str("Prop"))
	expectErrHas(t, `return Prop("x")()`, newOpts(), "no have method without params")
}

func TestVMPropStmt(t *testing.T) {
	// statement form: `prop name { ... }` declares a const property
	testExpectRun(t, `
	var v
	prop x {
		() => v          // getter
		(n) { v = n }    // setter
	}
	x("a")
	return x()`, nil, Str("a"))

	// the declared name carries the Prop type
	testExpectRun(t, `
	prop x { () => 1 }
	return typeName(x)`, nil, Str("Prop"))

	// single-accessor (brace-less) form
	testExpectRun(t, `
	prop pi() => 3.14
	return pi()`, nil, Float(3.14))

	// a typed setter is dispatched by the argument type
	testExpectRun(t, `
	var v
	prop x {
		() => v
		(n) { v = n }
		(n int) { v = "int= " + n }
	}
	x("a"); s1 := x()
	x(2);   s2 := x()
	return [s1, s2]`, nil, Array{Str("a"), Str("int= 2")})

	// a typed setter may also be added afterwards with `met`
	testExpectRun(t, `
	var v
	prop x {
		() => v
		(n) { v = n }
	}
	met x(n int) { v = "int= " + n }
	x(7)
	return x()`, nil, Str("int= 7"))

	// expression form: an anonymous prop assigned to a variable
	testExpectRun(t, `
	var v
	p := prop { () => v
	(n) { v = n } }
	p(9)
	return p()`, nil, Int(9))

	// getter with no matching setter: calling with an unmatched arg is an error
	expectErrHas(t, `
	prop x { () => 1 }
	return x("nope")`, newOpts(), "no have method")
}

func TestVMBuiltinModuleStringsFmt(t *testing.T) {
	// strings and fmt are available as builtin namespaces, without an import
	testExpectRun(t, `return strings.contains("abcd", "bc")`, nil, Bool(true))
	testExpectRun(t, `return strings.toUpper("hi")`, nil, Str("HI"))
	testExpectRun(t, `return strings.join(["a", "b"], "-")`, nil, Str("a-b"))
	testExpectRun(t, `return fmt.sprintf("%d-%s", 7, "x")`, nil, Str("7-x"))
	// members are builtin functions carrying their module spec
	testExpectRun(t, `return typeName(strings.contains)`, nil, Str("builtinFunction"))
	// a shadowing local disables the namespace: indexes the local dict instead
	testExpectRun(t, `strings := {contains: func(a, b) { return "local" }}; return strings.contains(1, 2)`,
		nil, Str("local"))
}

func TestVMBuiltinModuleTime(t *testing.T) {
	// time is a builtin namespace usable without an import; members are camelCase
	testExpectRun(t, `d := time.date(2009, 11, 10, 23, 0, 0, 0, time.utc()); return d.year()`, nil, Int(2009))
	testExpectRun(t, `return typeName(time.now())`, nil, Str("time"))
	// the type renders module-qualified (FullName)
	testExpectRun(t, `return str(typeof(time.now()))`, nil, Str("‹builtin type ‹time.time› with 7 methods›"))
	// the int(time) override is registered globally (no import), converting to a
	// Unix timestamp
	testExpectRun(t, `d := time.date(1970, 1, 1, 0, 0, 1, 0, time.utc()); return int(d)`, nil, Int(1))
	// constants and duration helpers
	testExpectRun(t, `return time.durationString(time.Hour + 30 * time.Minute)`, nil, Str("1h30m0s"))
}

func TestVMTimeDurationDate(t *testing.T) {
	// `dur …` duration literal compiles to a duration value
	testExpectRun(t, `return typeName(dur 1h30m)`, nil, Str("duration"))
	testExpectRun(t, `return str(dur 1h30m)`, nil, Str("1h30m0s"))
	testExpectRun(t, `return (dur 1h30m).minutes()`, nil, Float(90))
	// duration arithmetic and comparison
	testExpectRun(t, `return str(dur 1h + dur 30m)`, nil, Str("1h30m0s"))
	testExpectRun(t, `return dur 1h > dur 30m`, nil, Bool(true))
	// `dur` is still usable as an identifier when not followed by a number
	testExpectRun(t, `dur := 5; return dur`, nil, Int(5))
	// the Duration constructor: from int (ns) or a string
	testExpectRun(t, `return dur 1s == time.Duration("1s")`, nil, Bool(true))
	testExpectRun(t, `return str(time.Duration(1000000000))`, nil, Str("1s"))
	// invalid duration literal is a compile error
	expectErrHas(t, `return dur 1nope`, newOpts().CompilerError(),
		`Compile Error: invalid duration literal`)

	// the Date type: constructor + accessors (gad methods are camelCase)
	testExpectRun(t, `dt := time.CalendarDate(20260131); return [dt.year(), dt.month(), dt.day()]`,
		nil, Array{Int(2026), Int(1), Int(31)})
	testExpectRun(t, `return str(time.CalendarDate("2026-01-31"))`, nil, Str("2026-01-31"))
	testExpectRun(t, `return typeName(time.CalendarDate(20260131))`, nil, Str("calendarDate"))
}

func TestVMTimeArithmetic(t *testing.T) {
	// duration arithmetic: ±, scale, ratio, remainder, unary minus
	testExpectRun(t, `return str(dur 1h + dur 30m)`, nil, Str("1h30m0s"))
	testExpectRun(t, `return str(dur 1h - dur 30m)`, nil, Str("30m0s"))
	testExpectRun(t, `return str(dur 1h * 3)`, nil, Str("3h0m0s"))
	testExpectRun(t, `return str(dur 90m / 2)`, nil, Str("45m0s"))
	testExpectRun(t, `return dur 1h / dur 30m`, nil, Float(2)) // ratio
	testExpectRun(t, `return str(dur 1h % dur 45m)`, nil, Str("15m0s"))
	testExpectRun(t, `return str(-(dur 1h))`, nil, Str("-1h0m0s"))
	testExpectRun(t, `return dur 1h > dur 30m`, nil, True)
	expectErrHas(t, `return dur 1h / dur 0s`, newOpts(), "ZeroDivision")

	// time ± duration -> time
	testExpectRun(t, `return str(2026-01-01T + dur 90m)`,
		nil, Str("2026-01-01 01:30:00 +0000 UTC"))
	testExpectRun(t, `return str(2026-01-01T - dur 1h)`,
		nil, Str("2025-12-31 23:00:00 +0000 UTC"))

	// calendarTime ± duration -> calendarTime; difference -> duration
	testExpectRun(t, `return str(2026-01-31t + dur 25h)`, nil, Str("2026-02-01 01:00:00"))
	testExpectRun(t, `return str(2026-02-01t - 2026-01-31t)`, nil, Str("24h0m0s"))
	testExpectRun(t, `return str(2026-01-31t.add(dur 1h))`, nil, Str("2026-01-31 01:00:00"))

	// calendarDate + duration: stays a date when day-aligned, else calendarTime
	testExpectRun(t, `return typeName(2025-01-01D + dur 1s)`, nil, Str("calendarTime"))
	testExpectRun(t, `return str(2025-01-01D + dur 1s)`, nil, Str("2025-01-01 00:00:01"))
	testExpectRun(t, `return typeName(2025-01-01D + dur 24h)`, nil, Str("calendarDate"))
	testExpectRun(t, `return str(2025-01-01D + dur 24h)`, nil, Str("2025-01-02"))
	testExpectRun(t, `return str(2025-02-01D - 2025-01-01D)`, nil, Str("744h0m0s"))

	// cross-type conversions
	testExpectRun(t, `return typeName(time.CalendarTime(2026-01-31T))`, nil, Str("calendarTime"))
	testExpectRun(t, `return typeName(time.CalendarDate(2026-01-31t))`, nil, Str("calendarDate"))
	testExpectRun(t, `return typeName(time.Type(2026-01-31t))`, nil, Str("time"))
}

func TestVMTimeTruncate(t *testing.T) {
	// .trunc(unit) lower-truncates to a calendar ('y','M','w','d') or Go
	// duration ('h','m','s','ms','us','ns') unit.
	const src = `time.strToTime("2026-08-17T14:37:52.123456789Z")` // 2026-08-17 is a Monday
	testExpectRun(t, `return str(`+src+`.trunc('y'))`, nil, Str("2026-01-01 00:00:00 +0000 UTC"))
	testExpectRun(t, `return str(`+src+`.trunc('M'))`, nil, Str("2026-08-01 00:00:00 +0000 UTC"))
	testExpectRun(t, `return str(`+src+`.trunc('w'))`, nil, Str("2026-08-17 00:00:00 +0000 UTC"))
	testExpectRun(t, `return str(`+src+`.trunc('d'))`, nil, Str("2026-08-17 00:00:00 +0000 UTC"))
	testExpectRun(t, `return str(`+src+`.trunc('h'))`, nil, Str("2026-08-17 14:00:00 +0000 UTC"))
	testExpectRun(t, `return str(`+src+`.trunc("ms"))`, nil, Str("2026-08-17 14:37:52.123 +0000 UTC"))

	// calendarTime and calendarDate keep their own types
	testExpectRun(t, `return str(time.strToCalendarTime("2026-08-17 14:37:52.5").trunc('h'))`,
		nil, Str("2026-08-17 14:00:00"))
	testExpectRun(t, `return typeName((2026-08-17D).trunc('M'))`, nil, Str("calendarDate"))
	testExpectRun(t, `return str((2026-08-17D).trunc('M'))`, nil, Str("2026-08-01"))

	// durations truncate toward zero by a fixed unit
	testExpectRun(t, `return str((dur 1h37m52s).trunc('m'))`, nil, Str("1h37m0s"))
	testExpectRun(t, `return str((dur 200h).trunc('w'))`, nil, Str("168h0m0s"))

	// invalid units error; 'y'/'M' are rejected for durations
	expectErrHas(t, `return (dur 1h).trunc('z')`, newOpts(), "invalid truncate unit")
	expectErrHas(t, `return (dur 1h).trunc('y')`, newOpts(), "invalid truncate unit")

	// .round(unit) rounds to the nearest boundary (a tie rounds up)
	const rsrc = `time.strToTime("2026-08-17T14:37:52Z")`
	testExpectRun(t, `return str(`+rsrc+`.round('h'))`, nil, Str("2026-08-17 15:00:00 +0000 UTC"))
	testExpectRun(t, `return str(`+rsrc+`.round('d'))`, nil, Str("2026-08-18 00:00:00 +0000 UTC"))
	testExpectRun(t, `return str(`+rsrc+`.round('M'))`, nil, Str("2026-09-01 00:00:00 +0000 UTC"))
	testExpectRun(t, `return str((2026-08-17D).round('M'))`, nil, Str("2026-09-01"))
	testExpectRun(t, `return str(time.strToCalendarTime("2026-08-17 14:37:52").round('m'))`,
		nil, Str("2026-08-17 14:38:00"))
	testExpectRun(t, `return str((dur 1h37m52s).round('m'))`, nil, Str("1h38m0s"))
	testExpectRun(t, `return str((dur 1h37m52s).round('h'))`, nil, Str("2h0m0s"))
	expectErrHas(t, `return (dur 1h).round('y')`, newOpts(), "invalid round unit")
}

func TestVMTimeMethods(t *testing.T) {
	// addDate, weekday and format on calendarDate / calendarTime
	testExpectRun(t, `return str((2026-08-17D).addDate(0, 0, 5))`, nil, Str("2026-08-22"))
	testExpectRun(t, `return str((2026-08-17D).addDate(1, 1, 0))`, nil, Str("2027-09-17"))
	testExpectRun(t, `return (2026-08-17D).weekday()`, nil, Int(1)) // Monday
	testExpectRun(t, `return (2026-08-17D).format("Mon 02 Jan 2006")`, nil, Str("Mon 17 Aug 2026"))

	testExpectRun(t, `return str(time.strToCalendarTime("2026-08-17 10:00:00").addDate(0, 1, 0))`,
		nil, Str("2026-09-17 10:00:00"))
	testExpectRun(t, `return time.strToCalendarTime("2026-08-17 14:37:00").format("15:04")`,
		nil, Str("14:37"))
}

func TestVMTimeStrTo(t *testing.T) {
	// strToDate / strToDuration / strToLocation module functions
	testExpectRun(t, `return str(time.strToDate("2026-01-31"))`, nil, Str("2026-01-31"))
	testExpectRun(t, `return str(time.strToDuration("1h30m"))`, nil, Str("1h30m0s"))
	testExpectRun(t, `return str(time.strToLocation("-03:00"))`, nil, Str("-03:00"))
	// strToTime parses RFC3339 timestamps
	testExpectRun(t, `return str(time.strToTime("2026-01-31T23:59:55Z"))`,
		nil, Str("2026-01-31 23:59:55 +0000 UTC"))
	// fractional seconds
	testExpectRun(t, `return time.strToTime("2026-01-31T23:59:55.001Z").ns()`, nil, Int(1000000))
	// an explicit offset is honoured
	testExpectRun(t, `return time.strToTime("2026-01-31T23:59:55-03:00").hour()`, nil, Int(23))
	// a bare calendar date is midnight UTC
	testExpectRun(t, `return str(time.strToTime("2026-01-31"))`,
		nil, Str("2026-01-31 00:00:00 +0000 UTC"))
	// the time and Location constructors accept strings / unix ints
	testExpectRun(t, `return str(time.Type("2026-01-31T00:00:00Z"))`,
		nil, Str("2026-01-31 00:00:00 +0000 UTC"))
	testExpectRun(t, `return time.Type(1781609136).year()`, nil, Int(2026))
	testExpectRun(t, `return str(time.Location("America/Sao_Paulo"))`, nil, Str("America/Sao_Paulo"))
	// invalid input is an error
	expectErrHas(t, `return time.strToTime("nope")`, newOpts(), "invalid time")
}

func TestVMDateTimeLit(t *testing.T) {
	// digit-suffix literals fold to constants at compile time:
	// 2006-01-02D -> date, 2006-01-02T -> time (midnight UTC), 123U -> unix.
	testExpectRun(t, `return typeName(2026-01-31D)`, nil, Str("calendarDate"))
	testExpectRun(t, `return typeName(2026-01-31T)`, nil, Str("time"))
	testExpectRun(t, `return typeName(1781609136U)`, nil, Str("time"))

	testExpectRun(t, `return str(2026-01-31D)`, nil, Str("2026-01-31"))
	testExpectRun(t, `return str(2026-01-31T)`, nil, Str("2026-01-31 00:00:00 +0000 UTC"))
	testExpectRun(t, `return str(1781609136U)`, nil, Str("2026-06-16 11:25:36 +0000 UTC"))

	// lowercase `t` is the zone-less calendarTime
	testExpectRun(t, `return typeName(2026-01-31t)`, nil, Str("calendarTime"))
	testExpectRun(t, `return str(2026-01-31t)`, nil, Str("2026-01-31 00:00:00"))
	testExpectRun(t, `return 2026-01-31t.day()`, nil, Int(31))
	testExpectRun(t, `return 2026-01-31t < 2026-02-01t`, nil, True)
	// calendarTime stores nanoseconds (via the string parser / constructor)
	testExpectRun(t, `return time.strToCalendarTime("2026-01-31 23:59:55.001").ns()`,
		nil, Int(1000000))

	// the compact YYYYMMDD form is still accepted for dates
	testExpectRun(t, `return str(20260131D)`, nil, Str("2026-01-31"))

	// unix fractional seconds
	testExpectRun(t, `return 1781609136.001U.ns()`, nil, Int(1000000))

	// method dispatch on the folded values
	testExpectRun(t, `return 2026-01-31D.year()`, nil, Int(2026))
	testExpectRun(t, `return 2026-01-31T.day()`, nil, Int(31))

	// arithmetic is unaffected: a dashed run without a D/T/U suffix is just
	// subtraction, and a suffix glued to an identifier stays number + ident.
	testExpectRun(t, `return 2026 - 1`, nil, Int(2025))
	testExpectRun(t, `Drive := 5; return 123 * Drive`, nil, Int(615))
	testExpectRun(t, `return 0xABCD`, nil, Int(0xABCD))

	// invalid literal bodies fail at compile time
	expectErrHas(t, `return 2026D`, newOpts().CompilerError(), "invalid date")
}

func TestVMBuiltinModuleBase64(t *testing.T) {
	// the base64 module is available as a builtin namespace, without an import
	testExpectRun(t, `return base64.StdEncoding.EncodeToString(bytes("hi"))`, nil, Str("aGk="))
	testExpectRun(t, `return base64.RawStdEncoding.EncodeToString(bytes("hi"))`, nil, Str("aGk"))
	testExpectRun(t, `return base64.URLEncoding.EncodeToString(bytes("hi"))`, nil, Str("aGk="))
}

func TestVMBytesLit(t *testing.T) {
	// h"..." decodes a hexadecimal sequence to bytes
	testExpectRun(t, `return h"ffccf1c2"`, nil,
		Bytes{0xff, 0xcc, 0xf1, 0xc2})
	testExpectRun(t, `return typeName(h"ffccf1c2")`, nil, Str("bytes"))
	// uppercase hex digits work too
	testExpectRun(t, `return h"FFCC"`, nil, Bytes{0xff, 0xcc})
	// whitespace inside hex is ignored
	testExpectRun(t, `return h"ff cc f1 c2"`, nil,
		Bytes{0xff, 0xcc, 0xf1, 0xc2})
	// empty hex literal yields empty bytes
	testExpectRun(t, `return len(h"")`, nil, Int(0))

	// b"..." uses the UTF-8 bytes of the string content
	testExpectRun(t, `return b"Hello"`, nil, Bytes("Hello"))
	testExpectRun(t, `return typeName(b"Hello")`, nil, Str("bytes"))
	testExpectRun(t, `return str(b"Hello")`, nil, Str("Hello"))
	// escapes are processed in the regular string form
	testExpectRun(t, `return b"a\nb"`, nil, Bytes("a\nb"))
	// raw string form keeps escapes literal
	testExpectRun(t, "return b`a\\nb`", nil, Bytes(`a\nb`))
	// heredoc form
	testExpectRun(t, "return b\"\"\"\nab\ncd\n\"\"\"", nil, Bytes("ab\ncd"))
	// hex from a raw string
	testExpectRun(t, "return h`ffcc`", nil, Bytes{0xff, 0xcc})

	// usable in larger expressions
	testExpectRun(t, `return b"ab" + b"cd"`, nil, Bytes("abcd"))
	testExpectRun(t, `b := b"Hi"; return b[0]`, nil, Int('H'))

	// invalid hex content is a compile error
	expectErrHas(t, `return h"xy"`, newOpts().CompilerError(),
		`Compile Error: invalid bytes literal`)
	expectErrHas(t, `return h"abc"`, newOpts().CompilerError(),
		`Compile Error: invalid bytes literal`)
}

func TestVMRawStr(t *testing.T) {
	// `raw "..."` of a string literal yields a rawStr at compile time (the
	// double-quoted string's escapes are still processed before conversion)
	testExpectRun(t, `return raw "a\nb"`, nil, RawStr("a\nb"))
	testExpectRun(t, "return raw `a\\nb`", nil, RawStr(`a\nb`))
	testExpectRun(t, `return typeName(raw "x")`, nil, Str("rawstr"))
	// `raw EXPR` of a runtime expression converts at evaluation time. This also
	// exercises the constant-folding optimizer over OpToRawStr (regression for a
	// panic when the opcode was outside the optimizer's allowed-ops table).
	testExpectRun(t, `return raw str(100)`, nil, RawStr("100"))
	testExpectRun(t, `x := raw ("a" + str(1)); return x`, nil, RawStr("a1"))
	testExpectRun(t, `return typeName(raw str(100))`, nil, Str("rawstr"))
}

func TestVMRegexLit(t *testing.T) {
	// the literal evaluates to a regexp object
	testExpectRun(t, `return typeName(/ab+/)`, nil, Str("regexp"))
	testExpectRun(t, `r := /ab+/; return r.match("abbb")`, nil, True)
	testExpectRun(t, `r := /ab+/; return r.match("xyz")`, nil, False)
	// equivalent to the regexp() constructor
	testExpectRun(t, `return (/ab+/).match("abbb") == regexp("ab+").match("abbb")`,
		nil, True)
	// escapes and character classes
	testExpectRun(t, `r := /[0-9]+\/[0-9]+/; return r.match("12/34")`, nil, True)
	// POSIX flag compiles and matches
	testExpectRun(t, `return typeName(/a+/p)`, nil, Str("regexp"))
	testExpectRun(t, `r := /a+/p; return r.match("aaa")`, nil, True)
	// in operand positions
	testExpectRun(t, `f := func(re) { return re.match("ab") }; return f(/ab/)`, nil, True)
	// division still works (after a value `/` is the operator)
	testExpectRun(t, `return 10 / 2`, nil, Int(5))
	testExpectRun(t, `a := 12; b := 3; return a / b`, nil, Int(4))
	testExpectRun(t, `return [6/2, 8/4]`, nil, Array{Int(3), Int(2)})
}

func TestVMRegexpOperators(t *testing.T) {
	// ~ tests for a match (regexp on the left)
	testExpectRun(t, `return (/\d+/) ~ "abc123"`, nil, True)
	testExpectRun(t, `return (/\d+/) ~ "abc"`, nil, False)
	// ~~ returns the first match as a submatch result (full match + groups);
	// the submatch result is indexable (0 = whole match, 1.. = capture groups)
	// and supports negative indices and len()
	testExpectRun(t, `m := /(\w+)@(\w+)/ ~~ "user@host"; return [m[0], m[1], m[2], m[-1], len(m)]`,
		nil, Array{Str("user@host"), Str("user"), Str("host"), Str("host"), Int(3)})
	// ... and iterable
	testExpectRun(t, `
	re := /(\w+)@(\w+)/
	out := []
	for i, g in re ~~ "user@host" { out = append(out, [i, g]) }
	return out`, nil, Array{
		Array{Int(0), Str("user@host")},
		Array{Int(1), Str("user")},
		Array{Int(2), Str("host")},
	})
	// ~~~ returns all matches; each is itself an indexable submatch result
	testExpectRun(t, `all := /\d+/ ~~~ "a1 b22 c333"; return [len(all), all[0][0], all[2][0]]`,
		nil, Array{Int(3), Str("1"), Str("333")})
	// an out-of-range group index errors
	expectErrHas(t, `m := /(\w+)/ ~~ "ab"; return m[5]`, newOpts(), "IndexOutOfBounds")
}

func TestVMRegexpReplace(t *testing.T) {
	// replace method with a string template
	testExpectRun(t, `r := /o/; return r.replace("hello world", "0")`,
		nil, Str("hell0 w0rld"))
	// $1/$2 numbered group expansion
	testExpectRun(t, `r := /(\d+)-(\d+)/; return r.replace("12-34", "$2/$1")`,
		nil, Str("34/12"))
	// ${name} named group expansion
	testExpectRun(t, `r := /(?P<y>\d+)-(?P<m>\d+)/; return r.replace("2024-06", "${m}/${y}")`,
		nil, Str("06/2024"))
	// $$ is a literal dollar sign
	testExpectRun(t, `r := /x/; return r.replace("ax", "$$")`, nil, Str("a$"))
	// groups can be reused/duplicated in the template
	testExpectRun(t, `r := /(\w)/; return r.replace("ab", "$1$1")`, nil, Str("aabb"))
	// callable replacement (invoked per whole match)
	testExpectRun(t, `r := /[a-z]+/; return r.replace("ab cd", func(m) { return "<" + m + ">" })`,
		nil, Str("<ab> <cd>"))
	// callable replacement using a builtin-module function
	testExpectRun(t, `r := /[a-z]+/; return r.replace("hi bye", strings.toUpper)`,
		nil, Str("HI BYE"))
	// callable receives capture groups via the named arg `m` (m[0] is the whole
	// match, m[1].. are the groups)
	testExpectRun(t, `r := /(\w+)@(\w+)/; return r.replace("user@host", func(whole; m) { return m[2] + "@" + m[1] })`,
		nil, Str("host@user"))
	// the whole match is also available positionally alongside the groups
	testExpectRun(t, `r := /(\w)(\w)/; return r.replace("ab cd", func(whole; m) { return whole + ":" + m[2] + m[1] })`,
		nil, Str("ab:ba cd:dc"))
	// the named arg `re` is the regexp itself
	testExpectRun(t, `r := /\d+/; return r.replace("a12 b3", func(whole; re) { return "<" + str(re) + ">" })`,
		nil, Str("a<\\d+> b<\\d+>"))
	// bytes subject -> bytes result
	testExpectRun(t, `r := /o/; return str(r.replace(bytes("foo"), "0"))`,
		nil, Str("f00"))

	// `|` operator yields a unary replacer function
	testExpectRun(t, `r := /o/; f := r | "0"; return f("hello world")`,
		nil, Str("hell0 w0rld"))
	testExpectRun(t, `r := /[a-z]+/; f := r | func(m) { return m + "!" }; return f("ab cd")`,
		nil, Str("ab! cd!"))
	// `|` with a group-using callable
	testExpectRun(t, `
	redact := /(\d{2})(\d+)/ | func(whole; m) { return m[1] + strings.repeat("*", len(m[2])) }
	return "card 1234567890".|redact`, nil, Str("card 12********"))
	// composes with the pipe operator
	testExpectRun(t, `r := /o/; return "hello world".|(r | "0")`,
		nil, Str("hell0 w0rld"))
}

func TestVMDeferbStmt(t *testing.T) {
	// runs at block exit, LIFO
	testExpectRun(t, `
	out := ""
	{ deferb { out += "d1 " }; deferb { out += "d2 " }; out += "b " }
	out += "after"
	return out`, nil, Str("b d2 d1 after"))

	// deferb_err recovers within the block; execution continues after it
	testExpectRun(t, `
	out := ""
	{
		deferb_err { out += "caught:" + str($err) + " "; $err = nil }
		out += "before "
		throw "boom"
		out += "unreached "
	}
	out += "after"
	return out`, nil, Str("before caught:error: boom after"))

	// deferb_ok runs only on success
	testExpectRun(t, `
	g := func(fail) {
		res := ""
		{
			deferb_ok { res += "ok " }
			deferb_err { res += "err " }
			if fail { throw "x" }
		}
		return res
	}
	return g(false)`, nil, Str("ok "))

	// unsuppressed block error still propagates
	expectErrHas(t, `
	g := func() { { deferb_err { } ; throw "x" } }
	return g()`, newOpts(), "x")

	// a throw inside a deferb handler is captured into $err (recoverable)
	testExpectRun(t, `
	out := ""
	f := func() {
		{
			deferb_err { out += "recovered:" + str($err) + " "; $err = nil }
			deferb { throw "from-handler" }
			out += "body "
		}
		out += "after"
		return out
	}
	return f()`, nil, Str("body recovered:error: from-handler after"))

	// $ret is inappropriate in a block: it is a shadowed nil local and does not
	// reach an enclosing function's return value
	testExpectRun(t, `
	f := func() {
		{ deferb { } }
		return "base"
	}
	return f()`, nil, Str("base"))

	// `return` inside the block runs the handlers AND preserves the value
	testExpectRun(t, `
	out := []
	h := func() {
		{
			deferb { out = append(out, "D") }
			out = append(out, "B")
			return "RET"
		}
	}
	r := h()
	return [r, out]`, nil, Array{Str("RET"), Array{Str("B"), Str("D")}})

	// each block has its own deferb scope (nested blocks are independent)
	testExpectRun(t, `
	out := ""
	{
		deferb { out += "outer " }
		{ deferb { out += "inner " }; out += "in " }
		out += "out "
	}
	return out`, nil, Str("in inner out outer "))

	// shortcut form: assignment at block exit, LIFO
	testExpectRun(t, `
	out := ""
	{ deferb out += "b1 "; deferb out += "b2 "; out += "body " }
	out += "after"
	return out`, nil, Str("body b2 b1 after"))

	// shortcut form: increment at block exit
	testExpectRun(t, `
	n := 0
	{ deferb n++; deferb n++ }
	return n`, nil, Int(2))

	// shortcut form: a call receiving $err at block exit
	testExpectRun(t, `
	out := ""
	rec := func(e) { out += "e=" + str(e) }
	{ deferb rec($err) }
	return out`, nil, Str("e=nil"))
}

func TestVMComprehension(t *testing.T) {
	// array comprehension
	testExpectRun(t, `return [i * 2 for i in [1, 2, 3]]`,
		nil, Array{Int(2), Int(4), Int(6)})
	// with filter
	testExpectRun(t, `return [i for i in [1, 2, 3, 4, 5] if i % 2 == 0]`,
		nil, Array{Int(2), Int(4)})
	// nested generators
	testExpectRun(t, `return [i + j for i in [1, 2] for j in [10, 20]]`,
		nil, Array{Int(11), Int(21), Int(12), Int(22)})
	// empty source
	testExpectRun(t, `return [i for i in []]`, nil, Array{})
	// captures outer variable
	testExpectRun(t, `n := 10; return [i + n for i in [1, 2]]`, nil, Array{Int(11), Int(12)})
	// nested comprehension
	testExpectRun(t, `return [[j for j in [1, 2]] for i in [0, 0]]`,
		nil, Array{Array{Int(1), Int(2)}, Array{Int(1), Int(2)}})

	// dict comprehension: `[expr]` is a computed key (the value of i)
	testExpectRun(t, `return {[i]: i * i for i in [1, 2, 3]}`,
		nil, Dict{"1": Int(1), "2": Int(4), "3": Int(9)})
	// k, v iteration with filter and computed key
	testExpectRun(t, `return {[k]: v * 10 for k, v in {a: 1, b: 2} if v == 2}`,
		nil, Dict{"b": Int(20)})
	// a static key keeps its literal name (last write wins)
	testExpectRun(t, `return {n: i for i in [1, 2, 3]}`, nil, Dict{"n": Int(3)})
	// multiple keys (static + computed) and `_` reads the dict being built
	testExpectRun(t,
		`return {i: 100, x: 10 + i, z: (_.z ?? 20) + i, [i]: i * i for i in [1, 2, 3]}`,
		nil, Dict{
			"i": Int(100), "x": Int(13), "z": Int(26),
			"1": Int(1), "2": Int(4), "3": Int(9),
		})

	// loop variable does not leak into the surrounding scope
	expectErrHas(t, `x := [i for i in [1, 2]]; return i`,
		newOpts().CompilerError(), `Compile Error: unresolved reference "i"`)
}

func TestVMMatchExpr(t *testing.T) {
	// expression form: first matching arm wins, else is the default
	const f = `f := func(x) { return match (x) { 1: "one", 2: "two", else: "other" } }; `
	testExpectRun(t, f+`return f(1)`, nil, Str("one"))
	testExpectRun(t, f+`return f(2)`, nil, Str("two"))
	testExpectRun(t, f+`return f(9)`, nil, Str("other"))

	// single-line comma separators
	testExpectRun(t, `return match (2) { 1: "a", 2: "b", else: "z" }`, nil, Str("b"))
	// newline separators
	testExpectRun(t, "return match (3) {\n1: \"a\"\n3: \"c\"\nelse: \"z\"\n}", nil, Str("c"))

	// non-literal conditions evaluated against the subject
	testExpectRun(t, `a := 10; b := 20; return match (20) { a: "x", b: "y" }`,
		nil, Str("y"))
	// string subject
	testExpectRun(t, `return match ("hi") { "hi": 1, "bye": 2, else: 0 }`, nil, Int(1))

	// no matching arm and no else => yields nil
	testExpectRun(t, `return match (7) { 1: "a" }`, nil, Nil)
	// an empty match yields nil
	testExpectRun(t, `x := match 7 {}; return x`, nil, Nil)

	// the subject no longer requires parentheses
	testExpectRun(t, `return match 2 { 1: "a", 2: "b", else: "z" }`, nil, Str("b"))

	// multiple conditions per arm (OR), comma- and newline-separated
	testExpectRun(t, `return match 3 { 1, 2, 3: "low", else: "hi" }`, nil, Str("low"))
	testExpectRun(t, "return match 4 {\n1, 2\n3, 4: \"x\"\nelse: \"y\"\n}", nil, Str("x"))
	// multi-condition statement-form arm
	testExpectRun(t, `
	var out = 0
	match 3 { 1, 2 { out = 12 } 3, 4 { out = 34 } }
	return out`, nil, Int(34))

	// statement form: runs the matching block; returns from the enclosing func
	const g = `g := func(x) { match (x) { 1 { return "ONE" }, 2 { return "TWO" }, else { return "OTHER" } } }; `
	testExpectRun(t, g+`return g(1)`, nil, Str("ONE"))
	testExpectRun(t, g+`return g(2)`, nil, Str("TWO"))
	testExpectRun(t, g+`return g(5)`, nil, Str("OTHER"))

	// statement form, no else, no match => falls through (no effect)
	testExpectRun(t, `
	var out = 0
	k := func(x) { match (x) { 1 { out = 100 } } }
	k(5)
	return out`, nil, Int(0))
	testExpectRun(t, `
	var out = 0
	k := func(x) { match (x) { 1 { out = 100 } } }
	k(1)
	return out`, nil, Int(100))

	// match is usable inline as an expression value
	testExpectRun(t, `return 1 + match (2) { 2: 40, else: 0 }`, nil, Int(41))

	// `match` keyword does not break selector method names (e.g. regexp.match)
	testExpectRun(t, `re := regexp("ab"); return re.match("ab")`, nil, True)
}

func TestVMSpreadLiterals(t *testing.T) {
	// array merge
	testExpectRun(t, `a := [2, 3]; b := [5, 6]; return [1, *a, 4, *b]`,
		nil, Array{Int(1), Int(2), Int(3), Int(4), Int(5), Int(6)})
	testExpectRun(t, `a := [2, 3]; return [*a]`, nil, Array{Int(2), Int(3)})
	testExpectRun(t, `a := [2]; b := [3]; return [*a, *b]`, nil, Array{Int(2), Int(3)})
	testExpectRun(t, `return []`, nil, Array{})
	// spread yields a copy, source not mutated/aliased
	testExpectRun(t, `a := [1, 2]; b := [*a]; b[0] = 9; return [a, b]`,
		nil, Array{Array{Int(1), Int(2)}, Array{Int(9), Int(2)}})

	// dict merge (later keys win)
	testExpectRun(t, `b := {x:9}; d := {y:8, x:100}; return {a:1, *b, e:2, *d}`,
		nil, Dict{"a": Int(1), "e": Int(2), "x": Int(100), "y": Int(8)})
	testExpectRun(t, `b := {x:9}; return {*b}`, nil, Dict{"x": Int(9)})
	testExpectRun(t, `b := {x:9}; d := {y:8}; return {*b, *d}`,
		nil, Dict{"x": Int(9), "y": Int(8)})
	// spread yields a copy, source not mutated/aliased
	testExpectRun(t, `c := {x:1}; d := {*c}; d.x = 9; return [c, d]`,
		nil, Array{Dict{"x": Int(1)}, Dict{"x": Int(9)}})
}

func TestVMMixedParamsDestructure(t *testing.T) {
	const x = `x := (1, 2, *[3, 4]; c=5, **{d:6, e:7}); `

	// full destructure: positional rest is a single `*rest` (like a variadic
	// param); named side uses rename/default/rest.
	testExpectRun(t, x+`(a, b, *pos_rest; c, d:p, r=2, **named_rest) := x; return [a, b, pos_rest]`,
		nil, Array{Int(1), Int(2), Array{Int(3), Int(4)}})
	testExpectRun(t, x+`(a, b, *pos_rest; c, d:p, r=2, **named_rest) := x; return [c, p, r, named_rest]`,
		nil, Array{Int(5), Int(6), Int(2), Dict{"e": Int(7)}})
	// `**rest` is accepted as a lenient alias for the positional rest
	testExpectRun(t, x+`(a, b, **pos_rest; c) := x; return [a, b, pos_rest]`,
		nil, Array{Int(1), Int(2), Array{Int(3), Int(4)}})

	// positional only
	testExpectRun(t, x+`(m, n) := x; return [m, n]`, nil, Array{Int(1), Int(2)})
	// positional rest collects the remainder
	testExpectRun(t, x+`(m, *rest) := x; return [m, rest]`,
		nil, Array{Int(1), Array{Int(2), Int(3), Int(4)}})

	// `=` assigns to pre-defined variables
	testExpectRun(t, x+`var (p, q, rest); (p, q, *rest; ) = x; return [p, q, rest]`,
		nil, Array{Int(1), Int(2), Array{Int(3), Int(4)}})

	// the MixedParams value round-trips through .positional/.named
	testExpectRun(t, x+`return x.positional`, nil, Array{Int(1), Int(2), Int(3), Int(4)})
	testExpectRun(t, x+`return dict(x.named)`, nil,
		Dict{"c": Int(5), "d": Int(6), "e": Int(7)})
}

func TestVMDictDestructure(t *testing.T) {
	const d = `d := {a:2, b:3, x:4, y:5}; `

	// `:=` defines new locals; plain name reads same-named key
	testExpectRun(t, d+`(;a) := d; return a`, nil, Int(2))
	// rename: variable `_b` from dict key `b` (key-on-the-left: `key: target`)
	testExpectRun(t, d+`(;b:_b) := d; return _b`, nil, Int(3))
	// fallback default used only when the key is absent
	testExpectRun(t, d+`(;r=9) := d; return r`, nil, Int(9))
	testExpectRun(t, d+`(;a=9) := d; return a`, nil, Int(2))
	// **rest collects the keys not consumed
	testExpectRun(t, d+`(;a, b:_b, **other) := d; return other`,
		nil, Dict{"x": Int(4), "y": Int(5)})
	testExpectRun(t, d+`(;a, b:_b, **other) := d; return [a, _b]`,
		nil, Array{Int(2), Int(3)})
	// rest with all keys consumed -> empty dict; source dict is not mutated
	testExpectRun(t, d+`(;a:a, b:b2, x:x2, y:y2, **other) := d; return [other, d]`,
		nil, Array{Dict{}, Dict{"a": Int(2), "b": Int(3), "x": Int(4), "y": Int(5)}})

	// `=` assigns to predefined variables (all must already exist)
	testExpectRun(t, d+`var (p, q, rest); (;a:p, b:q, **rest) = d; return [p, q, rest]`,
		nil, Array{Int(2), Int(3), Dict{"x": Int(4), "y": Int(5)}})

	// errors
	expectErrHas(t, `d := {a:1}; (;a) = d; return a`,
		newOpts().CompilerError(), `Compile Error: unresolved reference "a"`)
}

func TestVMIteratorPooling(t *testing.T) {
	// A pooled for-in iterator must never recycle a user-held iterator() value:
	// after it.next consumes 100, the loop continues from 200/300.
	testExpectRun(t, `it := iterator([100, 200, 300]); it.next; u := []; for k, v in it { u += [v] }; return u`,
		nil, Array{Array{Int(200)}, Array{Int(300)}})
	// nested loops (LIFO release)
	testExpectRun(t, `s := 0; for i in [1,2,3] { for j in [10,20] { s += i*j } }; return s`, nil, Int(180))
	// break / continue do not corrupt the pool
	testExpectRun(t, `t := 0; for v in [1,2,3,4,5] { if v==2 {continue}; if v==4 {break}; t += v }; return t`, nil, Int(4))
	// sequential loops reuse the pooled iterator
	testExpectRun(t, `a := 0; for v in [1,2,3] {a+=v}; for v in [4,5,6] {a+=v}; return a`, nil, Int(21))
	// empty iterable -> else, then a following loop still works
	testExpectRun(t, `x := 0; for v in [] {x=1} else {x=2}; for v in [7] {x+=v}; return x`, nil, Int(9))

	// the concrete iterator is pooled too: a user-held dict iterator survives.
	testExpectRun(t, `it := iterator({x:1, y:2, z:3}; sorted); it.next; d := []; for k, v in it { d += [v] }; return d`,
		nil, Array{Array{Int(2)}, Array{Int(3)}})
	// iterator consumers (collect/values) stay correct while loops churn the pool.
	testExpectRun(t, `s := 0; for n := 0; n < 3; n++ { for v in [1,2,3] { s += v } }; return collect(values([s]))`,
		nil, Array{Int(18)})

	// the rarer pooled iterators: key-value array and args.
	testExpectRun(t, `s := 0; for k, v in (;a=1, b=2, c=3) { s += v }; return s`, nil, Int(6))
	testExpectRun(t, `it := iterator((;x=10, y=20, z=30)); it.next; r := []; for k, v in it { r += [v] }; return r`,
		nil, Array{Array{Int(20)}, Array{Int(30)}})
	testExpectRun(t, `f := func(*args) { t := 0; for i, v in args { t += v }; return t }; return f(5, 15, 25)`,
		nil, Int(45))
}

func TestVMConstDestructure(t *testing.T) {
	// const/var with `{ … }` (dict) and `[ … ]` (bracket array) patterns.
	testExpectRun(t, `const {x} = {a:1, x:2}; return x`, nil, Int(2))
	testExpectRun(t, `const {k: v} = {k: 42}; return v`, nil, Int(42))
	testExpectRun(t, `var {y, z, **rest} = {y:5, z:6, w:7}; return [y, z, rest]`,
		nil, Array{Int(5), Int(6), Dict{"w": Int(7)}})
	testExpectRun(t, `const [a, b] = [1, 2]; return [a, b]`, nil, Array{Int(1), Int(2)})
	testExpectRun(t, `var [p, q, r] = [8, 9]; return [p, q, r]`, nil, Array{Int(8), Int(9), Nil})

	// `[a, b]` bracket destructure at statement level (`:=` and `=`).
	testExpectRun(t, `[a, b] := [3, 4]; return [a, b]`, nil, Array{Int(3), Int(4)})
	testExpectRun(t, `[x] := [9]; return x`, nil, Int(9))
	testExpectRun(t, `var m; var n; [m, n] = [5, 6]; return [m, n]`, nil, Array{Int(5), Int(6)})

	// the comma forms (no brackets) remain valid alongside the bracket form.
	testExpectRun(t, `a, b := [1, 2]; return [a, b]`, nil, Array{Int(1), Int(2)})
	testExpectRun(t, `var a; var b; a, b = [1, 2]; return [a, b]`, nil, Array{Int(1), Int(2)})
	testExpectRun(t, `a, b, c := [1, 2]; return [a, b, c]`, nil, Array{Int(1), Int(2), Nil})

	// trailing `*rest` collects the remaining elements as a fresh array.
	testExpectRun(t, `a, b, *rest := [1, 2, 3, 4]; return [a, b, rest]`,
		nil, Array{Int(1), Int(2), Array{Int(3), Int(4)}})
	testExpectRun(t, `var a; var b; var rest; a, b, *rest = [1, 2, 3, 4]; return [a, b, rest]`,
		nil, Array{Int(1), Int(2), Array{Int(3), Int(4)}})
	// short source: fixed targets pad with nil, rest is empty.
	testExpectRun(t, `a, b, *rest := [1]; return [a, b, rest]`,
		nil, Array{Int(1), Nil, Array{}})
	// bracket and const/var forms accept the rest too.
	testExpectRun(t, `[a, b, *rest] := [1, 2, 3, 4]; return [a, b, rest]`,
		nil, Array{Int(1), Int(2), Array{Int(3), Int(4)}})
	testExpectRun(t, `const [a, b, *rest] = [1, 2, 3, 4]; return [a, b, rest]`,
		nil, Array{Int(1), Int(2), Array{Int(3), Int(4)}})
	// `*rest` is only valid as the last target.
	expectErrHas(t, `a, *rest, b := [1, 2, 3, 4]; return a`, newOpts().CompilerError(),
		`rest element must be last in destructuring`)

	// const produces immutable bindings.
	expectErrHas(t, `const {x} = {x:1}; x = 2`, newOpts().CompilerError(),
		`assignment to constant variable "x"`)
	expectErrHas(t, `const [a] = [1]; a = 2`, newOpts().CompilerError(),
		`assignment to constant variable "a"`)
	expectErrHas(t, `const [a, *rest] = [1, 2, 3]; rest = []`, newOpts().CompilerError(),
		`assignment to constant variable "rest"`)
}

func TestVMCurlyDestructure(t *testing.T) {
	const d = `d := {a:2, b:3, x:4, y:5}; `

	// TypeScript order: key-on-the-left. `a` reads key "a"; `b: v` binds key "b".
	testExpectRun(t, d+`{ a } := d; return a`, nil, Int(2))
	testExpectRun(t, d+`{ b: v } := d; return v`, nil, Int(3))
	// fallback default via `=`
	testExpectRun(t, d+`{ r = 9 } := d; return r`, nil, Int(9))
	testExpectRun(t, d+`{ a = 9 } := d; return a`, nil, Int(2))
	// **rest collects the remaining keys as a dict
	testExpectRun(t, d+`{ a, b: v, **rest } := d; return rest`,
		nil, Dict{"x": Int(4), "y": Int(5)})
	testExpectRun(t, d+`{ a, b: v, **rest } := d; return [a, v]`,
		nil, Array{Int(2), Int(3)})
	// `=` assigns to predefined variables
	testExpectRun(t, d+`var (p, q); { a: p, b: q } = d; return [p, q]`,
		nil, Array{Int(2), Int(3)})
	// any ToDictConverter source: a KeyValueArray works via the dict() constructor
	testExpectRun(t, `{ a, b: v, **rest } := (;a=1, b=2, c=3); return [a, v, rest]`,
		nil, Array{Int(1), Int(2), Dict{"c": Int(3)}})
	// empty pattern is a no-op
	testExpectRun(t, d+`{} := d; return d.a`, nil, Int(2))
}

func TestVMDestructuring(t *testing.T) {
	expectErrHas(t, `x, y = nil; return x`,
		newOpts().CompilerError(), `Compile Error: unresolved reference "x"`)
	expectErrHas(t, `var (x, y); x, y := nil; return x`,
		newOpts().CompilerError(), `Compile Error: no new variable on left side`)
	// a single target with several right-side expressions is still meaningless.
	expectErrHas(t, `x := 1, 2`, newOpts().CompilerError(),
		`Compile Error: multiple expressions on the right side not supported`)

	// parallel multi-value assignment: several targets, several values.
	testExpectRun(t, `a, b := 1, 2; return [a, b]`, nil, Array{Int(1), Int(2)})
	testExpectRun(t, `var (a, b); a, b = 1, 2; return [a, b]`, nil, Array{Int(1), Int(2)})
	testExpectRun(t, `a, b, *rest := 1, 2, 3, 4; return [a, b, rest]`,
		nil, Array{Int(1), Int(2), Array{Int(3), Int(4)}})
	// leniency matches array destructuring: extra values drop, missing pad nil.
	testExpectRun(t, `a, b := 1, 2, 3; return [a, b]`, nil, Array{Int(1), Int(2)})
	testExpectRun(t, `a, b, c := 1, 2; return [a, b, c]`, nil, Array{Int(1), Int(2), Nil})
	// spreads inside the right side flatten.
	testExpectRun(t, `a, b, *rest := 1, *[2, 3, 4]; return [a, b, rest]`,
		nil, Array{Int(1), Int(2), Array{Int(3), Int(4)}})

	testExpectRun(t, `x, y := nil; return x`, nil, Nil)
	testExpectRun(t, `x, y := nil; return y`, nil, Nil)
	testExpectRun(t, `x, y := 1; return x`, nil, Int(1))
	testExpectRun(t, `x, y := 1; return y`, nil, Nil)
	testExpectRun(t, `x, y := []; return x`, nil, Nil)
	testExpectRun(t, `x, y := []; return y`, nil, Nil)
	testExpectRun(t, `x, y := [1]; return x`, nil, Int(1))
	testExpectRun(t, `x, y := [1]; return y`, nil, Nil)
	testExpectRun(t, `x, y := [1, 2]; return x`, nil, Int(1))
	testExpectRun(t, `x, y := [1, 2]; return y`, nil, Int(2))
	testExpectRun(t, `x, y := [1, 2, 3]; return x`, nil, Int(1))
	testExpectRun(t, `x, y := [1, 2, 3]; return y`, nil, Int(2))
	testExpectRun(t, `var x; x, y := [1]; return x`, nil, Int(1))
	testExpectRun(t, `var x; x, y := [1]; return y`, nil, Nil)

	testExpectRun(t, `x, y, z := nil; return x`, nil, Nil)
	testExpectRun(t, `x, y, z := nil; return y`, nil, Nil)
	testExpectRun(t, `x, y, z := nil; return z`, nil, Nil)
	testExpectRun(t, `x, y, z := 1; return x`, nil, Int(1))
	testExpectRun(t, `x, y, z := 1; return y`, nil, Nil)
	testExpectRun(t, `x, y, z := 1; return z`, nil, Nil)
	testExpectRun(t, `x, y, z := []; return x`, nil, Nil)
	testExpectRun(t, `x, y, z := []; return y`, nil, Nil)
	testExpectRun(t, `x, y, z := []; return z`, nil, Nil)
	testExpectRun(t, `x, y, z := [1]; return x`, nil, Int(1))
	testExpectRun(t, `x, y, z := [1]; return y`, nil, Nil)
	testExpectRun(t, `x, y, z := [1]; return z`, nil, Nil)
	testExpectRun(t, `x, y, z := [1, 2]; return x`, nil, Int(1))
	testExpectRun(t, `x, y, z := [1, 2]; return y`, nil, Int(2))
	testExpectRun(t, `x, y, z := [1, 2]; return z`, nil, Nil)
	testExpectRun(t, `x, y, z := [1, 2, 3]; return x`, nil, Int(1))
	testExpectRun(t, `x, y, z := [1, 2, 3]; return y`, nil, Int(2))
	testExpectRun(t, `x, y, z := [1, 2, 3]; return z`, nil, Int(3))
	testExpectRun(t, `x, y, z := [1, 2, 3, 4]; return z`, nil, Int(3))

	// test index assignments
	testExpectRun(t, `
	var (x = {}, y, z)
	x.a, y, z = [1, 2, 3, 4]; return x`, nil, Dict{"a": Int(1)})
	testExpectRun(t, `
	var (x = {}, y, z)
	x.a, y, z = [1, 2, 3, 4]; return y`, nil, Int(2))
	testExpectRun(t, `
	var (x = {}, y, z)
	x.a, y, z = [1, 2, 3, 4]; return z`, nil, Int(3))

	testExpectRun(t, `
	var (x = {}, y, z)
	y, x.a, z = [1, 2, 3, 4]; return x`, nil, Dict{"a": Int(2)})
	testExpectRun(t, `
	var (x = {}, y, z)
	y, x.a, z = [1, 2, 3, 4]; return y`, nil, Int(1))
	testExpectRun(t, `
	var (x = {}, y, z)
	y, x.a, z = [1, 2, 3, 4]; return z`, nil, Int(3))

	testExpectRun(t, `
	var (x = [0], y, z)
	x[0], y, z = [1, 2, 3, 4]; return x`, nil, Array{Int(1)})
	testExpectRun(t, `
	var (x = [0], y, z)
	x[0], y, z = [1, 2, 3, 4]; return y`, nil, Int(2))
	testExpectRun(t, `
	var (x = [0], y, z)
	x[0], y, z = [1, 2, 3, 4]; return z`, nil, Int(3))

	testExpectRun(t, `
	var (x = [0], y, z)
	y, x[0], z = [1, 2, 3, 4]; return x`, nil, Array{Int(2)})
	testExpectRun(t, `
	var (x = [0], y, z)
	y, x[0], z = [1, 2, 3, 4]; return y`, nil, Int(1))
	testExpectRun(t, `
	var (x = [0], y, z)
	y, x[0], z = [1, 2, 3, 4]; return z`, nil, Int(3))

	// test function calls
	testExpectRun(t, `
	fn := func() { 
		return [1, error("abc")]
	}
	x, y := fn()
	return [x, str(y)]`, nil, Array{Int(1), Str("error: abc")})

	testExpectRun(t, `
	fn := func() { 
		return [1]
	}
	x, y := fn()
	return [x, y]`, nil, Array{Int(1), Nil})
	testExpectRun(t, `
	fn := func() { 
		return
	}
	x, y := fn()
	return [x, y]`, nil, Array{Nil, Nil})
	testExpectRun(t, `
	fn := func() { 
		return [1, 2, 3]
	}
	x, y := fn()
	t := {a: x}
	return [x, y, t]`, nil, Array{Int(1), Int(2), Dict{"a": Int(1)}})
	testExpectRun(t, `
	fn := func() { 
		return {}
	}
	x, y := fn()
	return [x, y]`, nil, Array{Dict{}, Nil})
	testExpectRun(t, `
	fn := func(v) { 
		return [1, v, 3]
	}
	var x = 10
	x, y := fn(x)
	t := {a: x}
	return [x, y, t]`, nil, Array{Int(1), Int(10), Dict{"a": Int(1)}})

	// test any expression
	testExpectRun(t, `x, y :=  {}; return [x, y]`, nil, Array{Dict{}, Nil})
	testExpectRun(t, `
	var x = 2
	if x > 0 {
		fn := func(v) { 
			return [3*v, 4*v]
		}
		var y
		x, y = fn(x)
		if y != 8 {
			throw sprintf("y value expected: %s, got: %s", 8, y)
		}
	}
	return x
	`, nil, Int(6))
	testExpectRun(t, `
	var x = 2
	if x > 0 {
		fn := func(v) { 
			return [3*v, 4*v]
		}
		// new x symbol is created within if block
		// x in upper block is not affected
		x, y := fn(x)
		if y != 8 {
			throw sprintf("y value expected: %s, got: %s", 8, y)
		}
	}
	return x
	`, nil, Int(2))

	testExpectRun(t, `
	var x = 2
	if x > 0 {
		fn := func(v) {
			try {
				ret := v/2
			} catch err {
				return [0, err]
			} finally {
				if err == nil {
					return ret
				}
			}
		}
		a, err := fn("str")
		if !isError(err) {
			throw err
		}
		if a != 0 {
			throw "a is not 0"
		}
		a, err = fn(6)
		if err != nil {
			throw sprintf("unexpected error: %s", err)
		}
		if a != 3 {
			throw "a is not 3"
		}
		x = a
	}
	// return map to check stack pointer is correct
	return {x: x}
	`, nil, Dict{"x": Int(3)})
	testExpectRun(t, `
	for x,y := [1, 2]; true; x++ {
		if x == 10 {
			return [x, y]
		}
	}
	`, nil, Array{Int(10), Int(2)})
	testExpectRun(t, `
	if x,y := [1, 2]; true {
		return [x, y]
	}
	`, nil, Array{Int(1), Int(2)})
	testExpectRun(t, `
	var x = 0
	for true {
		x, y := [x]
		x++
		break
	}
	return x`, nil, Int(0))
	testExpectRun(t, `
	x, y := func(n) {
		return repeat([n], n)
	}(3)
	return [x, y]`, nil, Array{Int(3), Int(3)})
	// closures
	testExpectRun(t, `
	var x = 10
	a, b := func(n) {
		x = n
	}(3)
	return [x, a, b]`, nil, Array{Int(3), Nil, Nil})
	testExpectRun(t, `
	var x = 10
	a, b := func(*args) {
		x, y := args
		return [x, y]
	}(1, 2)
	return [x, a, b]`, nil, Array{Int(10), Int(1), Int(2)})
	testExpectRun(t, `
	var x = 10
	a, b := func(*args) {
		var y
		x, y = args
		return [x, y]
	}(1, 2)
	return [x, a, b]`, nil, Array{Int(1), Int(1), Int(2)})

	// return implicit array if return statement's expressions are comma
	// separated which is a part of destructuring implementation to mimic multi
	// return values.
	parseErr := `Parse Error: expected operand, found 'EOF'`
	expectErrHas(t, `return 1,`,
		newOpts().CompilerError(), parseErr)
	expectErrHas(t, `return 1, 2,`,
		newOpts().CompilerError(), parseErr)
	expectErrHas(t, `var a; return a,`,
		newOpts().CompilerError(), parseErr)
	expectErrHas(t, `var (a, b); return a, b,`,
		newOpts().CompilerError(), parseErr)
	expectErrHas(t, `return 1,`,
		newOpts().CompilerError(), parseErr)
	expectErrHas(t, `return 1, 2,`,
		newOpts().CompilerError(), parseErr)
	expectErrHas(t, `var a; return a,`,
		newOpts().CompilerError(), parseErr)
	expectErrHas(t, `var (a, b); return a, b,`,
		newOpts().CompilerError(), parseErr)

	parseErr = `Parse Error: expected operand, found '}'`
	expectErrHas(t, `func(){ return 1, }`,
		newOpts().CompilerError(), parseErr)
	expectErrHas(t, `func(){ return 1, 2,}`,
		newOpts().CompilerError(), parseErr)

	expectErrHas(t, `func(){ var a; return a,}`,
		newOpts().CompilerError(), parseErr)
	expectErrHas(t, `func(){ var (a, b); return a, b,}`,
		newOpts().CompilerError(), parseErr)
	expectErrHas(t, `func(){ return 1,}`,
		newOpts().CompilerError(), parseErr)
	expectErrHas(t, `func(){ return 1, 2,}`,
		newOpts().CompilerError(), parseErr)
	expectErrHas(t, `func(){ var a; return a,}`,
		newOpts().CompilerError(), parseErr)
	expectErrHas(t, `func(){ var (a, b); return a, b,}`,
		newOpts().CompilerError(), parseErr)

	testExpectRun(t, `return 1, 2`, nil, Array{Int(1), Int(2)})
	testExpectRun(t, `a := 1; return a, a`, nil, Array{Int(1), Int(1)})
	testExpectRun(t, `a := 1; return a, 2`, nil, Array{Int(1), Int(2)})
	testExpectRun(t, `a := 1; return 2, a`, nil, Array{Int(2), Int(1)})
	testExpectRun(t, `a := 1; return 2, a, [3]`, nil,
		Array{Int(2), Int(1), Array{Int(3)}})
	testExpectRun(t, `a := 1; return [2, a], [3]`, nil,
		Array{Array{Int(2), Int(1)}, Array{Int(3)}})
	testExpectRun(t, `return {}, []`, nil, Array{Dict{}, Array{}})
	testExpectRun(t, `return func(){ return 1}(), []`, nil, Array{Int(1), Array{}})
	testExpectRun(t, `return func(){ return 1}(), [2]`, nil,
		Array{Int(1), Array{Int(2)}})
	testExpectRun(t, `
	f := func() {
		return 1, 2
	}
	a, b := f()
	return a, b`, nil, Array{Int(1), Int(2)})
	testExpectRun(t, `
	a, b := func() {
		return 1, error("x")
	}()
	return a, "" + b`, nil, Array{Int(1), Str("error: x")})
	testExpectRun(t, `
	a, b := func(a, b) {
		return a + 1, b + 1
	}(1, 2)
	return a, b, a*2, 3/b`, nil, Array{Int(2), Int(3), Int(4), Int(1)})
	testExpectRun(t, `
	return func(a, b) {
		return a + 1, b + 1
	}(1, 2), 4`, nil, Array{Array{Int(2), Int(3)}, Int(4)})

	testExpectRun(t, `
	param *args

	mapEach := func(seq, fn) {
	
		if !isArray(seq) {
			return error("want array, got " + typeName(seq))
		}
	
		var out = []
	
		if sz := len(seq); sz > 0 {
			out = repeat([0], sz)
		} else {
			return out
		}
	
		try {
			for i, v in seq {
				out[i] = fn(v)
			}
		} catch err {
			println(err)
		} finally {
			return out, err
		}
	}
	
	global multiplier
	
	v, err := mapEach(args, func(x) { return x*multiplier })
	if err != nil {
		return err
	}
	return v
	`, newOpts().
		Globals(Dict{"multiplier": Int(2)}).
		Args(Int(1), Int(2), Int(3), Int(4)),
		Array{Int(2), Int(4), Int(6), Int(8)})

	testExpectRun(t, `
	global goFunc
	// ...
	v, err := goFunc(2)
	if err != nil {
		return str(err)
	}
	`, newOpts().
		Globals(Dict{"goFunc": &Function{
			Value: func(Call) (Object, error) {
				// ...
				return Array{
					Nil,
					ErrIndexOutOfBounds.NewError("message"),
				}, nil
			},
		}}),
		Str("IndexOutOfBoundsError: message"))
}

func TestVMConst(t *testing.T) {
	expectErrHas(t, `const x = 1; x = 2`, newOpts().CompilerError(),
		`Compile Error: assignment to constant variable "x"`)
	expectErrHas(t, `const x = 1; x := 2`, newOpts().CompilerError(),
		`Compile Error: "x" redeclared in this block`)
	expectErrHas(t, `const (x = 1, x = 2)`, newOpts().CompilerError(),
		`Compile Error: "x" redeclared in this block`)
	expectErrHas(t, `const x`, newOpts().CompilerError(),
		`Parse Error: missing initializer in const declaration`)
	expectErrHas(t, `const (x, y = 2)`, newOpts().CompilerError(),
		`Parse Error: missing initializer in const declaration`)

	// After iota support, `const (x=1,y)` does not throw error, like ToInterface. It
	// uses last expression as initializer.
	testExpectRun(t, `const (x = 1, y)`, nil, Nil)

	expectErrHas(t, `const (x, y)`, newOpts().CompilerError(),
		`Parse Error: missing initializer in const declaration`)
	expectErrHas(t, `
	const x = 1
	func f() {
		x = 2
	}`, newOpts().CompilerError(),
		`Compile Error: assignment to constant variable "x"`)
	expectErrHas(t, `
	const x = 1
	if x > 0 {
		x = 2
	}`, newOpts().CompilerError(),
		`Compile Error: assignment to constant variable "x"`)
	expectErrHas(t, `
	const x = 1
	if x > 0 {
		return func() {
			x = 2
		}
	}`, newOpts().CompilerError(),
		`Compile Error: assignment to constant variable "x"`)
	expectErrHas(t, `
	const x = 1
	if x = 2; x > 0 {
		return
	}`, newOpts().CompilerError(),
		`Compile Error: assignment to constant variable "x"`)
	expectErrHas(t, `
	const x = 1
	for x = 1; x < 10; x++ {
		return
	}`, newOpts().CompilerError(),
		`Compile Error: assignment to constant variable "x"`)
	expectErrHas(t, `
	const x = 1
	func f() {
		var y
		x, y = [1, 2]
	}`, newOpts().CompilerError(),
		`Compile Error: assignment to constant variable "x"`)
	expectErrHas(t, `
	x := 1
	func f() {
		const y = 2
		x, y = [1, 2]
	}`, newOpts().CompilerError(),
		`Compile Error: assignment to constant variable "y"`)
	expectErrHas(t, `const x = 1;global x`, newOpts().CompilerError(),
		`Compile Error: "x" redeclared in this block`)
	expectErrHas(t, `const x = 1;param x`, newOpts().CompilerError(),
		`Compile Error: "x" redeclared in this block`)
	expectErrHas(t, `global x; const x = 1`, newOpts().CompilerError(),
		`Compile Error: "x" redeclared in this block`)
	expectErrHas(t, `param x; const x = 1`, newOpts().CompilerError(),
		`Compile Error: "x" redeclared in this block`)
	expectErrHas(t, `
	const x = 1
	if [2] { // not optimized
		x = 2
	}`, newOpts().CompilerError(),
		`Compile Error: assignment to constant variable "x"`)
	expectErrHas(t, `
	const x = 1
	if [2] { // not optimized
		func f() {
			x = 2
		}
	}`, newOpts().CompilerError(),
		`Compile Error: assignment to constant variable "x"`)
	expectErrHas(t, `
	const x = 1
	if x {
		func f() {
			func f2() {
				for {
					x = 2
				}
			}
		}
	}`, newOpts().CompilerError(),
		`Compile Error: assignment to constant variable "x"`)

	// FIXME: Compiler does not compile if or else blocks if condition is
	// a *BoolLit (which may be reduced by optimizer). So compiler does not
	// check whether a constant is reassigned in block to throw an error.
	// A few examples for this issue.
	testExpectRun(t, `
	const x = 1
	if true {
		
	} else {
		// block is not compiled
		x = 2
	}
	return x
	`, nil, Int(1))
	testExpectRun(t, `
	const x = 1
	if false {
		// block is not compiled
		x = 2
	}
	return x
	`, nil, Int(1))

	testExpectRun(t, `const x = 1; return x`, nil, Int(1))
	testExpectRun(t, `const x = "1"; return x`, nil, Str("1"))
	testExpectRun(t, `const x = []; return x`, nil, Array{})
	testExpectRun(t, `const x = []; return x`, nil, Array{})
	testExpectRun(t, `const x = nil; return x`, nil, Nil)
	testExpectRun(t, `const (x = 1, y = "2"); return x, y`, nil,
		Array{Int(1), Str("2")})
	testExpectRun(t, `
	const (
		x = 1
		y = "2"
	)
	return x, y`, nil, Array{Int(1), Str("2")})
	testExpectRun(t, `
	const (
		x = 1
		y = x + 1
	)
	return x, y`, nil, Array{Int(1), Int(2)})
	testExpectRun(t, `
	const x = 1
	return func() {
		const x = x + 1
		return x
	}()`, nil, Int(2))
	testExpectRun(t, `
	const x = 1
	return func() {
		x := x + 1
		return x
	}()`, nil, Int(2))
	testExpectRun(t, `
	const x = 1
	return func() {
		return func() {
			return x + 1
		}()
	}()`, nil, Int(2))
	testExpectRun(t, `
	const x = 1
	for x := 10; x < 100; x++{
		return x
	}`, nil, Int(10))
	testExpectRun(t, `
	const (i = 1, v = 2)
	for i,v in [10] {
		v = -1
		return i
	}`, nil, Int(0))
	testExpectRun(t, `
	const x = 1
	return func() {
		const y = 2
		const x = y
		return x
	}() + x
	`, nil, Int(3))
	testExpectRun(t, `
	const x = 1
	return func() {
		const y = 2
		var x = y
		return x
	}() + x
	`, nil, Int(3))
	testExpectRun(t, `
	const x = 1
	func() {
		x, y := [2, 3]
	}()
	return x
	`, nil, Int(1))
	testExpectRun(t, `
	const x = 1
	for i := 0; i < 1; i++ {
		x, y := [2, 3]
		break
	}
	return x
	`, nil, Int(1))
	testExpectRun(t, `
	const x = 1
	if [1] {
		x, y := [2, 3]
	}
	return x
	`, nil, Int(1))

	testExpectRun(t, `
	return func() {
		const x = 1
		func() {
			x, y := [2, 3]
		}()
		return x
	}()
	`, nil, Int(1))
	testExpectRun(t, `
	return func() {
		const x = 1
		for i := 0; i < 1; i++ {
			x, y := [2, 3]
			break
		}
		return x
	}()
	`, nil, Int(1))
	testExpectRun(t, `
	return func(){
		const x = 1
		if [1] {
			x, y := [2, 3]
		}
		return x
	}()
	`, nil, Int(1))
	testExpectRun(t, `
	return func(){
		const x = 1
		if [1] {
			var y
			x, y := [2, 3]
		}
		return x
	}()
	`, nil, Int(1))
}

func TestConstIota(t *testing.T) {
	testExpectRun(t, `const x = iota; return x`, nil, Int(0))
	testExpectRun(t, `const x = iota; const y = iota; return x, y`, nil, Array{Int(0), Int(0)})
	testExpectRun(t, `const(x = iota, y = iota); return x, y`, nil, Array{Int(0), Int(1)})
	testExpectRun(t, `const(x = iota, y); return x, y`, nil, Array{Int(0), Int(1)})

	testExpectRun(t, `const(x = 1+iota, y); return x, y`, nil, Array{Int(1), Int(2)})
	testExpectRun(t, `const(x = 1+iota, y=iota); return x, y`, nil, Array{Int(1), Int(1)})
	testExpectRun(t, `const(x = 1+iota, y, z); return x, y, z`, nil, Array{Int(1), Int(2), Int(3)})
	testExpectRun(t, `const(x = iota+1, y, z); return x, y, z`, nil, Array{Int(1), Int(2), Int(3)})

	testExpectRun(t, `const(_ = iota+1, y, z); return y, z`, nil, Array{Int(2), Int(3)})

	testExpectRun(t, `
	const (
		x = [iota]
	)
	return x`, nil, Array{Int(0)})

	testExpectRun(t, `
	const (
		x = []
	)
	return x`, nil, Array{})

	testExpectRun(t, `
	const (
		x = [iota, iota]
	)
	return x`, nil, Array{Int(0), Int(0)})

	testExpectRun(t, `
	const (
		x = [iota, iota]
		y
	)
	return x, y`, nil, Array{Array{Int(0), Int(0)}, Array{Int(1), Int(1)}})

	testExpectRun(t, `
	const (
		x = [iota, iota]
		y
		z
	)
	return x, y, z`, nil,
		Array{Array{Int(0), Int(0)}, Array{Int(1), Int(1)}, Array{Int(2), Int(2)}})

	testExpectRun(t, `
	const (
		x = [iota, iota]
		y
	)
	x[0] = 2
	return x, y`, nil, Array{Array{Int(2), Int(0)}, Array{Int(1), Int(1)}})

	testExpectRun(t, `
	const (
		x = {}
	)
	return x`, nil, Dict{})

	testExpectRun(t, `
	const (
		x = {iota: 1}
	)
	return x`, nil, Dict{"iota": Int(1)})

	testExpectRun(t, `
	const (
		x = {k: iota}
	)
	return x`, nil, Dict{"k": Int(0)})

	testExpectRun(t, `
	const (
		x = {k: iota}
		y
	)
	return x, y`, nil, Array{Dict{"k": Int(0)}, Dict{"k": Int(1)}})

	testExpectRun(t, `
	const (
		x = {k: iota}
		y
	)
	x["k"] = 2
	return x, y`, nil, Array{Dict{"k": Int(2)}, Dict{"k": Int(1)}})

	testExpectRun(t, `
	const (
		x = {k: iota}
		y
		z
	)
	return x, y, z`, nil,
		Array{Dict{"k": Int(0)}, Dict{"k": Int(1)}, Dict{"k": Int(2)}})

	testExpectRun(t, `
	const (
		_ = 1 << iota
		x
		y
	)
	return x, y`, nil, Array{Int(2), Int(4)})

	testExpectRun(t, `
	const (
		x = 1 << iota
		_
		y
	)
	return x, y`, nil, Array{Int(1), Int(4)})

	testExpectRun(t, `
	const (
		x = 1 << iota
		a
		y = a
		z
	)
	return x, y, z`, nil, Array{Int(1), Int(2), Int(2)})

	testExpectRun(t, `
	const (
		x = 1 << iota
		_
		_
		z
	)
	return x, z`, nil, Array{Int(1), Int(8)})

	testExpectRun(t, `
	iota := 1
	const (
		x = 1 << iota
	)
	return x, iota`, nil, Array{Int(2), Int(1)})

	testExpectRun(t, `
	iota := 1
	const (
		x = 1 << iota
		y
	)
	return x, y`, nil, Array{Int(2), Int(2)})

	expectErrHas(t, `const iota = 1`,
		newOpts().CompilerError(), "Compile Error: assignment to iota")

	expectErrHas(t, `const iota = iota + 1`,
		newOpts().CompilerError(), "Compile Error: assignment to iota")

	expectErrHas(t, `
	const (
		x = 1 << iota
		iota
		y
	)
	return x, iota, y`,
		newOpts().CompilerError(), "Compile Error: assignment to iota")

	expectErrHas(t, `const x = iota; return iota`,
		newOpts().CompilerError(), `Compile Error: unresolved reference "iota"`)

	testExpectRun(t, `
	const (
		x = iota
		y
	)
	iota := 3
	return x, y, iota`, nil, Array{Int(0), Int(1), Int(3)})

	testExpectRun(t, `
	const (
		x = iota
		y
	)
	iota := 3
	const (
		a = 10+iota
		b
	)
	return x, y, iota, a, b`, nil, Array{Int(0), Int(1), Int(3), Int(13), Int(13)})

	testExpectRun(t, `
	const (
		x = iota
		y
	)
	const (
		a = 10+iota
		b
	)
	return x, y, a, b`, nil, Array{Int(0), Int(1), Int(10), Int(11)})

	testExpectRun(t, `
	const (
		x = func() { return 1 }()
		y
		z
	)
	return x, y, z`, nil, Array{Int(1), Int(1), Int(1)})

	testExpectRun(t, `
	const (
		x = func(x) { return x }(iota)
		y
		z
	)
	return x, y, z`, nil, Array{Int(0), Int(1), Int(2)})

	testExpectRun(t, `
	a:=0
	const (
		x = func() { a++; return a }()
		y
		z
	)
	return x, y, z`, nil, Array{Int(1), Int(2), Int(3)})

	testExpectRun(t, `
	const (
		x = 1+iota
		y = func() { return 1+x }()
		z
	)
	return x, y, z`, nil, Array{Int(1), Int(2), Int(2)})

	testExpectRun(t, `
	const (
		x = func() { return 1 }
		y
		z
	)
	return x(), y(), z()`, nil, Array{Int(1), Int(1), Int(1)})

	testExpectRun(t, `
	const (
		x = func() { return 1 }
		y
		z
	)
	return str([x, y, z])`, nil,
		Str("[‹compiledFunction: (main).#1()›, ‹compiledFunction: (main).#2()›, ‹compiledFunction: (main).#3()›]"))

	testExpectRun(t, `
	var a
	const (
		x = func() { return a }
		y
		z
	)
	return x != y && y != z`, nil, True)

	testExpectRun(t, `
	return func() {
		const (
			x = 1 << iota
			_
			y
		)
		return x, y
	}()`, nil, Array{Int(1), Int(4)})

	testExpectRun(t, `
	iota := 2
	return func() {
		const (
			x = 1 << iota
			_
			y
		)
		return x, y
	}()`, nil, Array{Int(4), Int(4)})

	testExpectRun(t, `
	const (
		x = 1 + iota + func() { 
			const (
				_ = iota
				r
			)
			return r
		}()
		y
		_
	)
	return x,y`, nil, Array{Int(2), Int(3)})

	testExpectRun(t, `
	const (x = iota%2?"odd":"even", y, z)
	return x,y,z`, nil, Array{Str("even"), Str("odd"), Str("even")})
}

func TestVM_Invoke(t *testing.T) {
	applyPool := &Function{
		FuncName: "applyPool",
		Value: func(c Call) (Object, error) {
			inv := NewInvoker(c.VM, c.Args.Shift())
			inv.Acquire()
			defer inv.Release()
			return inv.Invoke(c.Args, &c.NamedArgs)
		},
	}
	applyNoPool := &Function{
		FuncName: "applyNoPool",
		Value: func(c Call) (Object, error) {
			args := make([]Object, 0, c.Args.Length()-1)
			for i := 1; i < c.Args.Length(); i++ {
				args = append(args, c.Args.Get(i))
			}
			inv := NewInvoker(c.VM, c.Args.Get(0))
			return inv.Invoke(Args{args}, &c.NamedArgs)
		},
	}
	for _, apply := range []*Function{applyPool, applyNoPool} {
		t.Run(apply.FuncName, func(t *testing.T) {
			t.Run("apply", func(t *testing.T) {
				scr := `
global apply
sum := func(*args) {
	println("called f", args)
	s := 0
	for v in args {
		println("v", v)
		s += v
	}
	return s
}
return apply(sum, 1, 2, 3)
`
				testExpectRun(t, scr,
					newOpts().Globals(Dict{"apply": apply}),
					Int(6),
				)
			})

			t.Run("apply indirect", func(t *testing.T) {
				scr := `
global apply
sum := func(*args) {
	println("sum args", args)
	s := 0
	for v in args {
		println("v", v)
		s += v
	}
	return s
}
f := func(fn, *args) {
	return fn(*args)
}
return apply(f, sum, 1, 2, 3)
`
				testExpectRun(t, scr,
					newOpts().Globals(Dict{"apply": apply}),
					Int(6),
				)
			})

			t.Run("apply indirect 2", func(t *testing.T) {
				scr := `
global apply
sum := func(*args) {
	println("sum args", args)
	s := 0
	for v in args {
		println("v", v)
		s += v
	}
	return s
}
f := func(fn, *args) {
	return apply(fn, *args)
}
return apply(f, sum, 1, 2, 3)
`
				testExpectRun(t, scr,
					newOpts().Globals(Dict{"apply": apply}),
					Int(6),
				)
			})

			t.Run("apply go func", func(t *testing.T) {
				sum := &Function{
					Value: func(c Call) (Object, error) {
						s := Int(0)
						for i := 0; i < c.Args.Length(); i++ {
							s += c.Args.Get(i).(Int)
						}
						return s, nil
					},
				}
				scr := `
global (apply, sum)
return apply(sum, 1, 2, 3)
`
				testExpectRun(t, scr,
					newOpts().Globals(Dict{"apply": apply, "sum": sum}),
					Int(6),
				)
			})

			t.Run("module state", func(t *testing.T) {
				scr := `
module := import("module")
module.counter = 1

global apply

inc := func(a) {
	module := import("module")
	module.counter += a
}
apply(inc, 3)
return module.counter
`
				t.Run("builtin", func(t *testing.T) {
					testExpectRun(t, scr,
						newOpts().
							Globals(Dict{"apply": apply}).
							Module("module", Dict{}),
						Int(4),
					)
				})
				t.Run("source", func(t *testing.T) {
					testExpectRun(t, scr,
						newOpts().
							Globals(Dict{"apply": apply}).
							Module("module", `return {}`),
						Int(4),
					)
				})
			})

			t.Run("closure", func(t *testing.T) {
				scr := `
global apply

counter := 1
f1 := func(a) {
	counter += a
}

f2 := func(a) {
	counter += a
}
apply(f1, 3)
apply(f2, 5)
return counter
`
				testExpectRun(t, scr,
					newOpts().Globals(Dict{"apply": apply}),
					Int(9),
				)
			})

			t.Run("global", func(t *testing.T) {
				scr := `
global apply
global counter

f1 := func(a) {
	counter += a
}

f2 := func(a) {
	counter += a
}
apply(f1, 3)
apply(f2, 5)
return counter
`
				expected := Int(9)
				globals := Dict{"apply": apply, "counter": Int(1)}
				testExpectRun(t, scr,
					newOpts().Globals(globals).Skip2Pass(),
					expected,
				)
				if expected != globals["counter"] {
					t.Fatalf("expected %s, got %v", expected, globals["counter"])
				}
			})
		})
	}
}

type nameCaller struct {
	Dict
	counts map[string]int
}

func (n *nameCaller) CallName(name string, c Call) (Object, error) {
	fn := n.Dict[name]
	args := make([]Object, 0, c.Args.Length())
	for i := 0; i < c.Args.Length(); i++ {
		args = append(args, c.Args.Get(i))
	}
	ret, err := NewInvoker(c.VM, fn).Invoke(Args{args}, &c.NamedArgs)
	n.counts[name]++
	return ret, err
}

var _ NameCallerObject = &nameCaller{}

func TestVMCallName(t *testing.T) {
	newobject := func() *nameCaller {
		var f = &Function{
			Value: func(c Call) (Object, error) {
				return c.Args.Get(0).(Int) + 1, nil
			},
		}

		return &nameCaller{Dict: Dict{"add1": f}, counts: map[string]int{}}
	}
	scr := `
global object

object.sub1 = func(a) {
	return a - 1
}

return [object.add1(10), object.sub1(10)]
`

	t.Run("basic", func(t *testing.T) {
		testExpectRun(t, scr,
			newOpts().Globals(Dict{"object": newobject()}),
			Array{Int(11), Int(9)},
		)
	})

	t.Run("counts single pass", func(t *testing.T) {
		object := newobject()
		testExpectRun(t, scr,
			newOpts().Globals(Dict{"object": object}).Skip2Pass(),
			Array{Int(11), Int(9)},
		)
		if object.counts["add1"] != 1 {
			t.Fatalf("expected 1, got %d", object.counts["add1"])
		}
		if object.counts["sub1"] != 1 {
			t.Fatalf("expected 1, got %d", object.counts["sub1"])
		}
	})

	t.Run("counts all pass", func(t *testing.T) {
		object := newobject()
		testExpectRun(t, scr,
			newOpts().Globals(Dict{"object": object}),
			Array{Int(11), Int(9)},
		)
		if object.counts["add1"] <= 0 {
			t.Fatalf("expected >0, got %d", object.counts["add1"])
		}
		if object.counts["sub1"] <= 0 {
			t.Fatalf("expected >0, got %d", object.counts["sub1"])
		}
	})
}

func TestInterfaceObject(t *testing.T) {
	fh := &FuncHeaderObject{FuncName: "g"}
	i := (&Interface{IName: "Shape", Module: NewModuleSpecFromName("mymod")}).
		WithField("id", TInt).
		WithGetter("area", fh).
		WithSetter("scale", fh).
		WithMethod("draw", &FuncHeaderObject{FuncName: "draw#1"})

	if got := i.FullName(); got != "mymod.Shape" {
		t.Fatalf("FullName = %q", got)
	}
	if got := i.Type().Name(); got != "Interface" {
		t.Fatalf("type name = %q", got)
	}
	if len(i.Fields) != 1 || len(i.Props) != 2 || len(i.Methods) != 1 {
		t.Fatalf("members: fields=%d props=%d methods=%d", len(i.Fields), len(i.Props), len(i.Methods))
	}
	// field types resolve; member back-refs are set
	if i.Fields[0].Iface != i {
		t.Fatal("field Iface back-ref not set")
	}
	if got := i.Fields[0].ToString(); got != "id int" {
		t.Fatalf("field str = %q", got)
	}
	// index access
	fields, _ := i.IndexGet(nil, Str("fields"))
	if arr, _ := fields.(Array); len(arr) != 1 {
		t.Fatalf("i.fields = %v", fields)
	}
	// a copy is Equal
	j := (&Interface{IName: "Shape"}).WithField("id", TInt).
		WithGetter("area", fh).WithSetter("scale", fh).
		WithMethod("draw", &FuncHeaderObject{FuncName: "draw#1"})
	if !i.Equal(j) {
		t.Fatalf("Equal: %s != %s", i, j)
	}
}

func TestVMInterface(t *testing.T) {
	// interface compiles to an Interface constant; members are readable
	testExpectRun(t, `return typeName(interface {})`, nil, Str("Interface"))
	testExpectRun(t, `I := interface Shape { id int; get area; draw() }
	return [I.name, len(I.fields), len(I.props), len(I.methods)]`,
		nil, Array{Str("Shape"), Int(1), Int(1), Int(1)})
	// a bare field entry is typed; field type resolves
	testExpectRun(t, `I := interface { n int|uint }
	return [I.fields[0].name, I.fields[0].types[0] == int, I.fields[0].types[1] == uint]`,
		nil, Array{Str("n"), True, True})
	// a block method (`parse`-style) groups several signatures
	testExpectRun(t, `I := interface { parse { (str), (v int) <bool> } }
	return [I.methods[0].name, len(I.methods[0].headers)]`,
		nil, Array{Str("parse"), Int(2)})
	// statement form binds a const
	testExpectRun(t, `interface S { m() }
	return [typeName(S), S.name]`, nil, Array{Str("Interface"), Str("S")})
}

func TestVMInterfaceSatisfaction(t *testing.T) {
	// `obj :: Interface` checks structural satisfaction: the object must have the
	// required fields (assignable type) and methods (matching signatures).
	testExpectRun(t, `
	interface Named { name str }
	class Person { name = ""; methods { greet() => "hi " + this.name } }
	return (Person(; name = "Ann") :: Named).name`, nil, Str("Ann"))

	testExpectRun(t, `
	interface Greeter { greet() <str> }
	class Person { name = ""; methods { greet() => "hi " + this.name } }
	return (Person(; name = "Bo") :: Greeter).greet()`, nil, Str("hi Bo"))

	// a missing method / field is rejected.
	expectErrIs(t, `
	interface Greeter { greet() <str> }
	class Anon { name = "x" }
	return Anon() :: Greeter`, nil, ErrIncompatibleAssign)
	expectErrIs(t, `
	interface HasAge { age int }
	class Person { name = "" }
	return Person() :: HasAge`, nil, ErrIncompatibleAssign)

	// an inline `interface{…}` parameter type dispatches by structural
	// satisfaction: a satisfying value is accepted, others rejected up front.
	testExpectRun(t, `
	func welcome(g interface{ greet() <str> }) => g.greet()
	class Person { name = ""; methods { greet() => "hi " + this.name } }
	return welcome(Person(; name = "Bo"))`, nil, Str("hi Bo"))
	expectErrHas(t, `
	func welcome(g interface{ greet() <str> }) => g.greet()
	return welcome(42)`, newOpts(), "invalid type for argument")

	// a dict satisfies an interface too: a field matches a key, a method matches
	// a callable key.
	testExpectRun(t, `
	interface Greeter { name str; greet() <str> }
	d := {name: "Ann", greet: func() => "hi"}
	return (d :: Greeter).greet()`, nil, Str("hi"))
	// a dict missing a required method (no callable key) is rejected.
	expectErrIs(t, `
	interface Greeter { greet() <str> }
	return {name: "Ann"} :: Greeter`, nil, ErrIncompatibleAssign)
	// a dict field of the wrong type is rejected.
	expectErrIs(t, `
	interface HasAge { age int }
	return {age: "x"} :: HasAge`, nil, ErrIncompatibleAssign)
	// a property maps to a plain key by presence.
	testExpectRun(t, `
	interface Titled { prop title }
	return ({title: "T"} :: Titled).title`, nil, Str("T"))
	// a class instance satisfies a `prop` requirement via a real property
	// accessor (detected without invoking the getter) or a plain field.
	testExpectRun(t, `
	interface Titled { prop title }
	class Doc { _t = "T"; props { title { () => this._t; (v) { this._t = v } } } }
	return typeName(Doc() :: Titled)`, nil, Str("Doc"))
	testExpectRun(t, `
	interface Titled { prop title }
	class Doc { title = "T" }
	return (Doc() :: Titled).title`, nil, Str("T"))

	// a NameCaller (which dispatches methods by name) optimistically satisfies a
	// method-only interface, but a required field it lacks is still rejected.
	testExpectRun(t, `interface HasFoo { foo() }
	return typeName((1d) :: HasFoo)`, nil, Str("decimal"))
	expectErrIs(t, `interface HasName { name str }
	return (1d) :: HasName`, nil, ErrIncompatibleAssign)
}
