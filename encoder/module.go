package encoder

import (
	"bytes"
	"context"

	"github.com/gad-lang/gad"
)

// BytecodeModule is a module already compiled — its bytecode encoded
// (EncodeBytecodeTo, a `.gadb`) —: imported, it is decoded and LINKED into
// the program, not compiled again. Each import decodes a copy of its own (the
// link moves the indexes of its constants).
//
//	mm.Add("contracts", encoder.BytecodeModule(compiled))
func BytecodeModule(data []byte) gad.Importable { return bytecodeModule(data) }

type bytecodeModule []byte

func (m bytecodeModule) Import(_ context.Context, module *gad.ModuleSpec) (any, string, error) {
	bc, err := DecodeModule(m)
	if err != nil {
		return nil, "", err
	}
	return bc, "bytecode:" + module.Name, nil
}

// DecodeModule is the bytecode of a module already compiled (a `.gadb`), to
// link: a copy of its own each time.
func DecodeModule(data []byte) (*gad.Bytecode, error) {
	return DecodeBytecodeFrom(NewReadContext(NewReader(bytes.NewReader(data))))
}

// CompileModule compiles src as a module to link (BytecodeModule) — a
// module, not a main: its `export`s are its members — and encodes it: with
// gad's builtins, which a program that imports it has.
func CompileModule(name string, src []byte) ([]byte, error) {
	return CompileModuleKind(name, src, gad.SourceKindGad)
}

// CompileModuleKind is CompileModule of a source of the dialect kind: plain
// Gad, a .gadt template, a .gadx template.
func CompileModuleKind(name string, src []byte, kind gad.SourceKind) ([]byte, error) {
	return CompileModuleWith(name, src, kind, nil, nil)
}

// CompileModuleWith is CompileModuleKind, the modules it imports read from
// mm (the program's: a builtin module, a file by its path), with the builtins
// names (the program's; nil: gad's) — which the bytecode keeps: a program
// links it only when they are its own (Bytecode.Builtins).
func CompileModuleWith(name string, src []byte, kind gad.SourceKind, mm *gad.ModuleMap, names gad.BuiltinsNameSet) ([]byte, error) {
	if names == nil {
		names = gad.NewBuiltins().NameSet
	}
	spec := &gad.ModuleSpec{ModuleInfo: gad.ModuleInfo{Name: name}}
	opts := gad.CompileOptions{CompilerOptions: gad.CompilerOptions{ModuleMap: mm}}
	res, err := gad.CompileModuleKind(gad.NewSymbolTable(names), spec, src, kind, opts)
	if err != nil {
		return nil, err
	}
	res.Bytecode.Builtins = make(map[string]int, len(names))
	for n, i := range names {
		res.Bytecode.Builtins[n] = int(i)
	}
	var buf bytes.Buffer
	if _, err = EncodeBytecodeTo(NewWriteContext(context.Background(), NewWriter(&buf)), res.Bytecode); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
