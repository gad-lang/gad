package encoder_test

import (
	"bytes"
	"testing"

	"github.com/gad-lang/gad"
	. "github.com/gad-lang/gad/encoder"
)

func BenchmarkBytecodeDecode(b *testing.B) {
	b.ReportAllocs()
	script := `
	f := func() {
		return [nil, true, false, "", -1, 0, 1, 2u, 3.0, 'a', bytes(0, 1, 2)]
	}
	f()
	m := {a: 1, b: ["abc"], c: {x: bytes()}, builtins: [append, len]}
	`
	var err error
	_, bc, err := gad.Compile(newSt(), []byte(script), gad.CompileOptions{})
	if err != nil {
		b.Fatal(err)
	}
	// bc.FileSet = nil
	// bc.Main.SourceMap = nil
	d, err := encode(bc)
	if err != nil {
		b.Fatal(err)
	}
	rd := bytes.NewReader(d)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rd.Reset(d)
		_, err := Decode(NewReadContext(rd))
		if err != nil {
			b.Fatal(err)
		}
	}
	b.ReportMetric(float64(len(d)), "Bytes")
}

func BenchmarkBytecodeEncDec(b *testing.B) {
	b.ReportAllocs()
	script := `
	f := func() {
		return [nil, true, false, "", -1, 0, 1, 2u, 3.0, 'a', bytes(0, 1, 2)]
	}
	f()
	m := {a: 1, b: ["abc"], c: {x: bytes()}, builtins: [append, len]}
	`
	var err error
	_, bc, err := gad.Compile(newSt(), []byte(script), gad.CompileOptions{})
	if err != nil {
		b.Fatal(err)
	}

	b.Run("compileUnopt", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _, err := gad.Compile(newSt(), []byte(script), gad.CompileOptions{})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("compileOpt", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _, err := gad.Compile(newSt(), []byte(script), gad.CompileOptions{
				CompilerOptions: gad.CompilerOptions{
					OptimizeConst:     true,
					OptimizeExpr:      true,
					OptimizerMaxCycle: 100,
				},
			})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("encode", func(b *testing.B) {
		var size int
		for i := 0; i < b.N; i++ {
			d, err := encode(bc)
			if err != nil {
				b.Fatal(err)
			}
			if size == 0 {
				size = len(d)
			}
		}
		b.ReportMetric(float64(size), "Bytes")
	})
	b.Run("decode", func(b *testing.B) {
		d, err := encode(bc)
		if err != nil {
			b.Fatal(err)
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := decode[*gad.Bytecode](d)
			if err != nil {
				b.Fatal(err)
			}
		}
		b.ReportMetric(float64(len(d)), "Bytes")
	})
}

func BenchmarkIntEncDec(b *testing.B) {
	b.Run("encode decode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			data, err := encode(gad.Int(i))
			if err != nil {
				b.Fatal(err)
			}
			_, err = decode[gad.Int](data)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func newSt() *gad.SymbolTable {
	return gad.NewSymbolTable(gad.NewBuiltins().NameSet)
}
