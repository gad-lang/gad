package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

func benchBytecode(b *testing.B, src string) *Bytecode {
	b.Helper()
	cr1, err := Compile(NewSymbolTable(NewBuiltins().NameSet), []byte(src), DefaultCompileOptions)
	bc := cr1.BC()
	if err != nil {
		b.Fatal(err)
	}
	return bc
}

// BenchmarkVMFib measures recursive-call dispatch and arithmetic in the VM loop.
func BenchmarkVMFib(b *testing.B) {
	bc := benchBytecode(b, `
	var fib
	fib = func(n) => n < 2 ? n : fib(n-1) + fib(n-2)
	return fib(25)`)
	builtins := NewBuiltins().Build()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := NewVM(builtins, bc).Run(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkVMSmallInts measures a loop whose values stay small (the common case:
// counters, indices, modulo), which the small-int box cache should keep alloc-free.
func BenchmarkVMSmallInts(b *testing.B) {
	bc := benchBytecode(b, `
	acc := 0
	for i := 0; i < 100000; i++ { acc = i % 100 - 50 }
	return acc`)
	builtins := NewBuiltins().Build()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := NewVM(builtins, bc).Run(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkTransformCastBool and BenchmarkBoolBuiltin are a pair: both convert a
// value to a boolean by truthiness in a tight loop, the first via the `::: bool`
// transforming cast (a direct IsFalsy check in the VM) and the second via the
// bool() builtin (a function call). The cast should be the faster of the two.
func BenchmarkTransformCastBool(b *testing.B) {
	bc := benchBytecode(b, `
	acc := false
	for i := 0; i < 100000; i++ { acc = i ::: bool }
	return acc`)
	builtins := NewBuiltins().Build()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := NewVM(builtins, bc).Run(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBoolBuiltin(b *testing.B) {
	bc := benchBytecode(b, `
	acc := false
	for i := 0; i < 100000; i++ { acc = bool(i) }
	return acc`)
	builtins := NewBuiltins().Build()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := NewVM(builtins, bc).Run(); err != nil {
			b.Fatal(err)
		}
	}
}

// runVMBench compiles src once and runs it b.N times, reporting allocations.
func runVMBench(b *testing.B, src string) {
	b.Helper()
	bc := benchBytecode(b, src)
	builtins := NewBuiltins().Build()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := NewVM(builtins, bc).Run(); err != nil {
			b.Fatal(err)
		}
	}
}

// A `::: T` conversion (non-bool builtin type) or a `::: fn` transformer calls,
// but the VM routes it through the same stack-based call path as a direct call —
// reusing the stack operand as the argument, no argument array allocated — so each
// pair below runs on par (equal ns/op and allocs/op). The arguments are the loop
// variable, not a literal, so the constant-folder cannot pre-evaluate `T(literal)`
// and skew the comparison.

func BenchmarkTransformCastStr(b *testing.B) {
	runVMBench(b, `acc := ""
	for i := 0; i < 100000; i++ { acc = i ::: str }
	return acc`)
}

func BenchmarkStrBuiltin(b *testing.B) {
	runVMBench(b, `acc := ""
	for i := 0; i < 100000; i++ { acc = str(i) }
	return acc`)
}

func BenchmarkTransformCastInt(b *testing.B) {
	runVMBench(b, `acc := 0
	for i := 0; i < 100000; i++ { acc = i ::: int }
	return acc`)
}

func BenchmarkIntBuiltin(b *testing.B) {
	runVMBench(b, `acc := 0
	for i := 0; i < 100000; i++ { acc = int(i) }
	return acc`)
}

func BenchmarkTransformCastFunc(b *testing.B) {
	runVMBench(b, `f := (v) => v * 2
	acc := 0
	for i := 0; i < 100000; i++ { acc = i ::: f }
	return acc`)
}

func BenchmarkFuncCall(b *testing.B) {
	runVMBench(b, `f := (v) => v * 2
	acc := 0
	for i := 0; i < 100000; i++ { acc = f(i) }
	return acc`)
}

// BenchmarkInvoker* compare a hot loop driven by a gad.invoker (overload resolved
// once, args array reused, validation skipped) against the same loop calling the
// function directly. The Str pair converts i via str() — dominated by string
// building, so the win shows mostly in allocations; the Typed pair calls a typed
// two-arg function whose per-call dispatch + validation is the cost, where the
// invoker is both faster and lighter.
func BenchmarkInvokerReused(b *testing.B) {
	runVMBench(b, `
	args := [int]
	inv := gad.invoker(str, args)
	acc := ""
	for i := 0; i < 100000; i++ { args[0] = i; acc = inv() }
	return acc`)
}

func BenchmarkInvokerDirectCall(b *testing.B) {
	runVMBench(b, `
	acc := ""
	for i := 0; i < 100000; i++ { acc = str(i) }
	return acc`)
}

func BenchmarkInvokerTypedReused(b *testing.B) {
	runVMBench(b, `
	f(a int, b int) => a + b
	args := [int, int]
	inv := gad.invoker(f, args)
	acc := 0
	for i := 0; i < 100000; i++ { args[0] = i; args[1] = i; acc = inv() }
	return acc`)
}

func BenchmarkInvokerTypedDirectCall(b *testing.B) {
	runVMBench(b, `
	f(a int, b int) => a + b
	acc := 0
	for i := 0; i < 100000; i++ { acc = f(i, i) }
	return acc`)
}

// BenchmarkVMDictAccess measures dict index/selector reads in a loop.
func BenchmarkVMDictAccess(b *testing.B) {
	bc := benchBytecode(b, `
	m := {a: 1, b: 2, c: 3}
	s := 0
	for i := 0; i < 50000; i++ { s = m.a + m["b"] + m.c }
	return s`)
	builtins := NewBuiltins().Build()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := NewVM(builtins, bc).Run(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkVMIterate measures for-in iteration over an array.
func BenchmarkVMIterate(b *testing.B) {
	bc := benchBytecode(b, `
	arr := [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
	s := 0
	for k := 0; k < 5000; k++ { for _, v in arr { s = v } }
	return s`)
	builtins := NewBuiltins().Build()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := NewVM(builtins, bc).Run(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkVMDictIterate measures for-in iteration over a dict (`for k, v in m`).
func BenchmarkVMDictIterate(b *testing.B) {
	bc := benchBytecode(b, `
	m := {a: 1, b: 2, c: 3, d: 4, e: 5}
	s := 0
	for k := 0; k < 5000; k++ { for _, v in m { s = v } }
	return s`)
	builtins := NewBuiltins().Build()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := NewVM(builtins, bc).Run(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkVMKVArrayIterate measures for-in iteration over a key-value array.
func BenchmarkVMKVArrayIterate(b *testing.B) {
	bc := benchBytecode(b, `
	kva := (;a: 1, b: 2, c: 3, d: 4, e: 5)
	s := 0
	for _, v in kva { s = v }
	for k := 0; k < 5000; k++ { for _, v in kva { s = v } }
	return s`)
	builtins := NewBuiltins().Build()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := NewVM(builtins, bc).Run(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkVMLoop measures a tight arithmetic loop (jumps, locals, binary ops).
func BenchmarkVMLoop(b *testing.B) {
	bc := benchBytecode(b, `
	s := 0
	for i := 0; i < 100000; i++ { s += i }
	return s`)
	builtins := NewBuiltins().Build()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := NewVM(builtins, bc).Run(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGadTransform measures the mapped bottom-up rewrite over a moderately
// wide/deep tree, applying a matcher at every array element and every dict child.
func BenchmarkGadTransform(b *testing.B) {
	runVMBench(b, `
	// build a tree: 200 rows of [x, y] pairs under a dict.
	rows := []
	for i := 0; i < 200; i++ { rows = rows + [[i, i+1]] }
	base := {rows: rows, meta: {a: 1, b: 2, c: 3}}
	acc := 0
	for k := 0; k < 50; k++ {
		d := gad.transform(base;
			".rows[]" = (p array) => [p[1], p[0]]
			".meta.*" = (n int) => n + 1
		)
		acc = acc + len(d.rows)
	}
	return acc`)
}

// BenchmarkGadTransformOverloaded measures a matched call whose callback is an
// overloaded (multi-method) function: the per-type resolved-overload cache should
// let repeat node types skip the dispatch (and validation).
func BenchmarkGadTransformOverloaded(b *testing.B) {
	runVMBench(b, `
	items := []
	for i := 0; i < 200; i++ { items = items + [i] }
	base := {items: items}
	func f(n int) => n + 1
	met f(s str) => s
	acc := 0
	for k := 0; k < 50; k++ {
		d := gad.transform(base; ".items[]" = f)
		acc = acc + len(d.items)
	}
	return acc`)
}
