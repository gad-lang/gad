package gad_test

import (
	"bytes"
	"testing"

	"github.com/gad-lang/gad"
)

// The metadata of a func of several methods and of a declared union is
// kept: read as `f.@meta`, `T.@meta` — exported too —; a union's member
// types as `T.@types`. A declared union with metadata is made by
// OpMakeTypeUnionMeta (one instruction), one without by OpMakeTypeUnion.
func TestDeclarationMetadataKept(t *testing.T) {
	mm := gad.NewModuleMap().AddSourceModule("m", []byte(`[kind="sum"] export func add { (a int, b int) => a + b; (a str, b str) => a + b }
[label="Number"] export type Num <int|float>
[doc="twice"] export func double(x) => x * 2`))
	run := func(src string) (gad.Object, *gad.Bytecode) {
		t.Helper()
		bi := gad.NewBuiltins()
		res, err := gad.Compile(gad.NewSymbolTable(bi.NameSet), []byte(src), gad.CompileOptions{
			CompilerOptions: gad.CompilerOptions{ModuleMap: mm},
		})
		if err != nil {
			t.Fatal(err)
		}
		out, err := gad.NewVM(bi.Build(), res.Bytecode).Run()
		if err != nil {
			t.Fatal(err)
		}
		return out, res.Bytecode
	}
	for src, want := range map[string]string{
		`[m=1, doc="x"] func f { (x) => x; (x, y) => y }
return [str(f.@meta), f(1), f(1, 2)]`: `["(;m=1, doc=\"x\")", 1, 2]`,
		`[label="Size", options=["S", "M"]] type T <str|int>
return [str(T.@meta), len(T.@types), "a" :: T]`: `["(;label=\"Size\", options=[\"S\", \"M\"])", 2, "a"]`,
		`type U <str|int>
return [U.@meta, len(U.@types)]`: `[nil, 2]`,
		`m := import("m")
return [str(m.add.@meta), m.add(1, 2), str(m.Num.@meta), str(m.double.@meta)]`: `["(;kind=\"sum\")", 3, "(;label=\"Number\")", "(;doc=\"twice\")"]`,
	} {
		got, _ := run(src)
		if got.ToString() != want {
			t.Errorf("%s:\n got %s\nwant %s", src, got.ToString(), want)
		}
	}

	// the opcode: with metadata, OpMakeTypeUnionMeta; without, OpMakeTypeUnion
	has := func(bc *gad.Bytecode, op gad.Opcode) bool {
		var buf bytes.Buffer
		bc.Fprint(gad.NewBuiltins(), &buf)
		return bytes.Contains(buf.Bytes(), []byte(gad.OpcodeNames[op]+" "))
	}
	_, withMeta := run(`[m=1] type T <str|int>
return T`)
	if !has(withMeta, gad.OpMakeTypeUnionMeta) || has(withMeta, gad.OpMakeTypeUnion) {
		t.Error("a union with metadata not made by OpMakeTypeUnionMeta")
	}
	_, without := run(`type T <str|int>
return T`)
	if has(without, gad.OpMakeTypeUnionMeta) || !has(without, gad.OpMakeTypeUnion) {
		t.Error("a union without metadata not made by OpMakeTypeUnion")
	}
}
