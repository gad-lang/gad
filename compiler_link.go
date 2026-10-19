package gad

import (
	"fmt"

	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/node"
)

// A module may be imported already compiled: an Importable whose Import
// gives a *Bytecode (encoder.BytecodeModule decodes one, `.gadb`). It is not
// compiled again — LINKED: its constants are appended to the program's, and
// every index of a constant it holds is moved by where they start (link).
//
// A function of gad indexes the constants of the bytecode it runs in, so a
// compiled module cannot run as it is in another program: linked, it is part
// of the program's bytecode, run by its VM like a module compiled from source.
//
// What a compiled module may hold: its own code, gad's builtins (compiled
// with the default ones — their indexes are fixed —), imports of modules by
// their names (a builtin module, a file by its path): each imported by the
// program, as `import(name)` would, its index in the module moved to the
// program's.

// linkModule links bc, a module compiled on its own, into the program as the
// module spec: its main function is what initializes the module.
func (c *Compiler) linkModule(nd ast.Node, spec *ModuleSpec, bc *Bytecode) error {
	if bc == nil || bc.Main == nil {
		return c.Errorf(nd, "module %s: an empty bytecode", spec.Name)
	}
	// compiled with the builtins of the program: a builtin is an index
	for name, idx := range bc.Builtins {
		if got, ok := c.symbolTable.builtins[name]; !ok || int(got) != idx {
			return c.Errorf(nd, "module %s: compiled with other builtins (%s is #%d there, #%d here): compile it again",
				spec.Name, name, idx, int(got))
		}
	}
	l := &linker{off: len(c.constants), spec: spec, from: bc.Main.module, syms: map[*SymbolInfo]bool{}, seen: map[Object]bool{},
		modules: map[int]int{}}
	// the modules it imports: the program's, by their names; itself, spec
	for i, ms := range bc.Modules {
		if ms == nil {
			continue
		}
		if l.from != nil && (ms == l.from || ms.Name == l.from.Name) {
			l.modules[i] = spec.Index
			continue
		}
		idx, err := c.linkImport(nd, ms.Name)
		if err != nil {
			return err
		}
		l.modules[i] = idx
	}
	// where its constants start: after what its imports added
	l.off = len(c.constants)
	if l.off+len(bc.Constants) > 1<<16 {
		return c.Errorf(nd, "module %s: too many constants once linked (%d)", spec.Name, l.off+len(bc.Constants))
	}
	for _, o := range bc.Constants {
		if err := l.object(o); err != nil {
			return c.Errorf(nd, "module %s: %v", spec.Name, err)
		}
	}
	if err := l.function(bc.Main); err != nil {
		return c.Errorf(nd, "module %s: %v", spec.Name, err)
	}
	c.constants = append(c.constants, bc.Constants...)
	spec.InitCompiledFunc = bc.Main
	return nil
}

// linker moves the constant indexes of a module by off, gives its functions
// the module they are now of (spec, for the one they were of: from), and moves
// the indexes of the modules it imports (modules: its index to the program's).
type linker struct {
	off     int
	spec    *ModuleSpec
	from    *ModuleSpec
	syms    map[*SymbolInfo]bool
	seen    map[Object]bool
	modules map[int]int
}

// linkImport is the index in the program of the module name, which a linked
// module imports: imported by the program now — as `import(name)`, the
// instructions it compiles to dropped: only the module is wanted — unless it
// is already.
func (c *Compiler) linkImport(nd ast.Node, name string) (int, error) {
	if item, ok := c.getModule(name); ok {
		return item.storeIndex, nil
	}
	n := len(c.instructions)
	if err := c.compileImportExpr(node.EImport(nd.Pos(), name, nd.Pos(), nd.Pos(), nd.Pos())); err != nil {
		return 0, err
	}
	c.instructions = c.instructions[:n]
	for pos := range c.sourceMap {
		if pos >= n {
			delete(c.sourceMap, pos)
		}
	}
	item, ok := c.getModule(name)
	if !ok {
		return 0, c.Errorf(nd, "module %s: not imported", name)
	}
	return item.storeIndex, nil
}

// constOperands are the opcodes whose first operand is the index of a
// constant.
var constOperands = map[Opcode]bool{
	OpConstant: true, OpClosure: true, OpGetGlobal: true, OpSetGlobal: true,
	OpEnvSet: true, OpPushSource: true,
}

// function links f: the constant operands of its instructions, the symbols of
// the types of its params and returns, its module.
func (l *linker) function(f *CompiledFunction) error {
	if f.module == nil || f.module == l.from {
		f.module = l.spec
	}
	ins := f.Instructions
	operands := make([]int, 0, 4)
	for i := 0; i < len(ins); {
		op := Opcode(ins[i])
		if int(op) >= len(OpcodeOperands) {
			return fmt.Errorf("an unknown opcode %d", op)
		}
		widths := OpcodeOperands[op]
		var read int
		operands, read = ReadOperands(widths, ins[i+1:], operands)
		switch {
		case op == OpLoadModule:
			n, ok := l.modules[operands[0]]
			if !ok {
				return fmt.Errorf("it loads a module it does not list (%d)", operands[0])
			}
			ins[i+1], ins[i+2] = byte(n>>8), byte(n)
		case constOperands[op]:
			if len(widths) == 0 || widths[0] != 2 {
				return fmt.Errorf("%s: not a constant of two bytes", OpcodeNames[op])
			}
			n := operands[0] + l.off
			if n >= 1<<16 {
				return fmt.Errorf("too many constants once linked")
			}
			ins[i+1], ins[i+2] = byte(n>>8), byte(n)
		}
		i += 1 + read
	}
	for _, p := range f.Params.Items {
		l.symbols(p.TypesSymbols)
		l.symbol(p.Symbol)
	}
	for _, p := range f.NamedParams.Items {
		l.symbols(p.TypesSymbols)
		l.symbol(p.Symbol)
	}
	for _, r := range f.ReturnVars {
		l.symbols(r.TypesSymbols)
	}
	return nil
}

// symbol moves a symbol of a constant or a global (whose index is the
// constant of its name), once.
func (l *linker) symbol(s *SymbolInfo) {
	if s == nil || l.syms[s] {
		return
	}
	l.syms[s] = true
	if s.Scope == ScopeConstant || s.Scope == ScopeGlobal {
		s.Index += l.off
	}
}

func (l *linker) symbols(syms ParamType) {
	for _, s := range syms {
		l.symbol(s)
	}
}

// object links a constant: the functions it holds, the symbols of its types.
func (l *linker) object(o Object) error {
	if o == nil {
		return nil
	}
	switch o.(type) {
	case *CompiledFunction, *Interface, *ArrayType, *PtrType, *FuncHeaderObject, *TypedIdent, *InterfaceField:
		if l.seen[o] {
			return nil
		}
		l.seen[o] = true
	}
	switch t := o.(type) {
	case *CompiledFunction:
		return l.function(t)
	case MetaType:
		l.symbol(t.TargetSym)
	case *ArrayType:
		l.symbols(t.Elem)
		l.symbols(t.SizeSyms)
	case *PtrType:
		l.symbols(t.Elem)
	case *TypedIdent:
		l.symbols(t.TypesSymbols)
	case *FuncHeaderObject:
		if t.Module == l.from {
			t.Module = l.spec
		}
		for _, list := range []Array{t.Params, t.NamedParams, t.Return} {
			for _, p := range list {
				if err := l.object(p); err != nil {
					return err
				}
			}
		}
	case *InterfaceField:
		l.symbols(t.TypesSymbols)
		return l.object(t.Default)
	case *Interface:
		if t.Module == l.from {
			t.Module = l.spec
		}
		l.symbols(t.Extends)
		for _, p := range t.ExtendsIface {
			if err := l.object(p); err != nil {
				return err
			}
		}
		for _, f := range t.Fields {
			if err := l.object(f); err != nil {
				return err
			}
		}
		if t.Elem != nil {
			if err := l.object(t.Elem); err != nil {
				return err
			}
		}
		for _, p := range t.Props {
			if err := l.object(p.Getter); err != nil {
				return err
			}
			for _, s := range p.Setters {
				if err := l.object(s); err != nil {
					return err
				}
			}
		}
		for _, m := range t.Methods {
			for _, h := range m.Headers {
				if err := l.object(h); err != nil {
					return err
				}
			}
		}
		for _, cf := range t.ContextFuncs {
			for _, h := range cf.Headers {
				if err := l.object(h); err != nil {
					return err
				}
			}
		}
	case Array:
		for _, e := range t {
			if err := l.object(e); err != nil {
				return err
			}
		}
	case KeyValueArray:
		for _, kv := range t {
			if err := l.object(kv.K); err != nil {
				return err
			}
			if err := l.object(kv.V); err != nil {
				return err
			}
		}
	case Dict:
		for _, v := range t {
			if err := l.object(v); err != nil {
				return err
			}
		}
	}
	return nil
}

// BytecodeImports reports whether bc imports a module (OpLoadModule): the
// program that links it imports them too (linkModule).
func BytecodeImports(bc *Bytecode) bool {
	fns := []*CompiledFunction{bc.Main}
	for _, o := range bc.Constants {
		if f, ok := o.(*CompiledFunction); ok {
			fns = append(fns, f)
		}
	}
	operands := make([]int, 0, 4)
	for _, f := range fns {
		ins := f.Instructions
		for i := 0; i < len(ins); {
			op := Opcode(ins[i])
			if op == OpLoadModule {
				return true
			}
			if int(op) >= len(OpcodeOperands) {
				return false
			}
			var read int
			operands, read = ReadOperands(OpcodeOperands[op], ins[i+1:], operands)
			i += 1 + read
		}
	}
	return false
}
