package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

func benchBytecode(b *testing.B, src string) *Bytecode {
	b.Helper()
	_, bc, err := Compile(NewSymbolTable(NewBuiltins().NameSet), []byte(src), DefaultCompileOptions)
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
