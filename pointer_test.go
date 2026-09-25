package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
	"github.com/stretchr/testify/require"
)

// TestPointerVariables covers `&x` for locals, captured variables and globals:
// `p.v` reads and writes the variable itself, and the pointer shares the cell a
// closure captures and outlives the declaring function.
func TestPointerVariables(t *testing.T) {
	for _, c := range []struct {
		src  string
		want Object
	}{
		{`x := 10; p := &x; a := p.v; p.v = 20; return [a, x, typeName(p), str(p)]`,
			Array{Int(10), Int(20), Str("ptr"), Str("&20")}},
		{`inc := func(n) { n.v += 1 }; c := 1; inc(&c); inc(&c); return c`, Int(3)},
		{`x := 1; p := &x; p.v++; return x`, Int(2)},
		// an array/bytes may be reallocated when it grows: the pointer writes the
		// new one back into the variable
		{`push := func(xs, v) { xs.v += v }; items := [1]; push(&items, 2); return items`,
			Array{Int(1), Int(2)}},
		{`d := bytes("ab"); p := &d; p.v += bytes(99); return str(d)`, Str("abc")},
		// the same cell as a closure
		{`total := 0; add := (n) => { total += n }; p := &total; add(5); a := p.v; p.v = 100; add(1); return [a, total]`,
			Array{Int(5), Int(101)}},
		// a local that outlives its function
		{`counter := func() { n := 0; return &n }; c := counter(); c.v += 1; c.v += 1; return c.v`, Int(2)},
		// a captured variable, from an inner closure
		{`g := func() { y := 1; h := () => { q := &y; q.v += 10 }; h(); return y }; return g()`, Int(11)},
		// equality is by variable
		{`a := 1; p1 := &a; p2 := &a; b := 1; return [p1 == p2, p1 == &b, bool(p1)]`,
			Array{True, False, True}},
	} {
		testExpectRun(t, c.src, nil, c.want)
	}

	g := Dict{"hits": Int(0)}
	testExpectRun(t, `global hits; track := func(h) { h.v++ }; track(&hits); track(&hits); return hits`,
		newOpts().Globals(g).Skip2Pass(), Int(2))
}

// TestPointerMembers covers `&obj.field` / `&arr[i]`: a pointer into any
// IndexGetter, reading and writing through it (class accessors apply).
func TestPointerMembers(t *testing.T) {
	for _, c := range []struct {
		src  string
		want Object
	}{
		{`u := {name: "ann"}; p := &u.name; p.v = "bo"; return u.name`, Str("bo")},
		{`xs := [1, 2, 3]; q := &xs[1]; q.v *= 10; return xs`, Array{Int(1), Int(20), Int(3)}},
		{`class C { x = 0; props { double { () => this.x * 2; (v) { this.x = v / 2 } } } }
		  c := C(); p := &c.double; p.v = 10; return [c.x, p.v]`, Array{Int(5), Int(10)}},
		{`x := 1; p := &x; q := &p.v; q.v = 9; return x`, Int(9)},
		{`u := {a: 1}; return &u.a == &u.a`, True},
	} {
		testExpectRun(t, c.src, nil, c.want)
	}
}

// TestPointerTypes covers `*T` in type positions: a parameter/field accepts a
// ptr whose current value is a T, envelopes for several types, and the
// multiplication `a * b` still parsing as one outside parameter lists.
func TestPointerTypes(t *testing.T) {
	for _, c := range []struct {
		src  string
		want Object
	}{
		{`func total(xs *array) => len(xs.v); nums := [1, 2]; return total(&nums)`, Int(2)},
		{`func d(v *<int|str>) => typeName(v.v); n := 1; s := "a"; return [d(&n), d(&s)]`,
			Array{Str("int"), Str("str")}},
		{`func sum(xs *[]int) => len(xs.v); a := [1, 2, 3]; return sum(&a)`, Int(3)},
		{`f := (p *int) => { p.v += 1 }; n := 1; f(&n); return n`, Int(2)},
		{`g := func(; c *int = nil) => c.v; n := 4; return g(; c = &n)`, Int(4)},
		{`h := func(*ps *int) => [p.v for p in ps]; a := 1; b := 2; return h(&a, &b)`, Array{Int(1), Int(2)}},
		{`class Cur { pos *int; methods { adv() { this.pos.v++ } } }
		  off := 0; c := Cur(; pos = &off); c.adv(); c.adv(); return off`, Int(2)},
		{`interface I { value *int }; n := 1; return ({value: &n} :: I) != nil`, True},
		{`a := 3; b := 4; m := func(x) => x; return [(a * b), m(a *b), a * b]`,
			Array{Int(12), Int(12), Int(12)}},
		{`func f(p *int) => 1; return str(f.@methods[0].@args)`, Str(`["p"]`)},
	} {
		testExpectRun(t, c.src, nil, c.want)
	}

	for _, src := range []string{
		`func total(xs *array) => 1; nums := [1]; return total(nums)`, // not a ptr
		`func total(xs *array) => 1; s := "x"; return total(&s)`,      // wrong pointee
		`func d(v *<int|str>) => 1; f := 1.5; return d(&f)`,           // not in the union
		`class Cur { pos *int }; return Cur(; pos = 5)`,               // field type
	} {
		expectErrHas(t, src, newOpts(), "TypeError")
	}
}

// TestPointerCompileErrors covers `&` on what has no address.
func TestPointerCompileErrors(t *testing.T) {
	for src, msg := range map[string]string{
		`const K = 1; p := &K`:        `cannot take the address of constant "K"`,
		`p := &1`:                     `& needs a variable, a field or an index`,
		`p := &len`:                   `cannot take the address of "len"`,
		`f := func() => 1; p := &f()`: `& needs a variable, a field or an index`,
	} {
		expectErrHas(t, src, newOpts().CompilerError(), msg)
	}
}

// customPtr is a Go Pointer handed to a script (host state behind `.v`).
type customPtr struct {
	ObjectImpl
	val Object
}

func (p *customPtr) Type() ObjectType                          { return TPtr }
func (p *customPtr) ToString() string                          { return "custom" }
func (p *customPtr) PtrGet(*VM) (Object, error)                { return p.val, nil }
func (p *customPtr) PtrSet(_ *VM, v Object) error              { p.val = v; return nil }
func (p *customPtr) IndexGet(vm *VM, i Object) (Object, error) { return PtrIndexGet(vm, p, i) }
func (p *customPtr) IndexSet(vm *VM, i, v Object) error        { return PtrIndexSet(vm, p, i, v) }

// TestPointerGoInterop covers the Go side: a Go pointer to a basic value is a
// script ptr writing the Go variable; `&goStruct.Field` writes the Go struct; a
// Go Pointer implementation works in the script, `*T` checks included; and a
// host takes a pointer to a class instance field with AddrOf.
func TestPointerGoInterop(t *testing.T) {
	port, name := 80, "x"
	type Cfg struct{ Port int }
	cfg := &Cfg{Port: 1}
	custom := &customPtr{val: Int(1)}

	g := Dict{"custom": custom}
	g["port"], _ = ToObject(&port)
	g["name"], _ = ToObject(&name)
	g["cfg"], _ = ToObject(cfg)
	testExpectRun(t, `
	global (port, name, cfg, custom)
	func bump(p *int) { p.v += 1 }
	port.v += 8000
	name.v = "gad"
	pp := &cfg.Port
	pp.v = 99
	bump(&custom.v)      // a pointer to the pointee, through the custom ptr
	bump(custom)
	return [typeName(port), str(port), custom.v]`,
		newOpts().Globals(g).Skip2Pass(), Array{Str("ptr"), Str("&8080"), Int(3)})
	require.Equal(t, 8080, port)
	require.Equal(t, "gad", name)
	require.Equal(t, 99, cfg.Port)

	// A host pointer to a class instance field (accessor/type rules apply).
	builtins := NewBuiltins()
	cr, err := Compile(NewSymbolTable(builtins.NameSet),
		[]byte(`class P { x = 1 }; return P()`), CompileOptions{})
	require.NoError(t, err)
	vm := NewVM(builtins.Build(), cr.Bytecode)
	inst, err := vm.Run(nil)
	require.NoError(t, err)
	p, err := AddrOf(vm, inst, Str("x"))
	require.NoError(t, err)
	require.NoError(t, p.PtrSet(vm, Int(7)))
	v, err := inst.(IndexGetter).IndexGet(vm, Str("x"))
	require.NoError(t, err)
	require.Equal(t, Int(7), v)
}

// TestPointerOpcodes checks the bytecode `&` compiles to: a local's slot is
// promoted (GETLOCALPTR) and wrapped (VARPTR); a captured variable uses
// GETFREEPTR; a member takes the target and key then ADDROFINDEX; a global
// points into the globals object.
func TestPointerOpcodes(t *testing.T) {
	ops := func(src string) (out []string) {
		builtins := NewBuiltins()
		cr, err := Compile(NewSymbolTable(builtins.NameSet), []byte(src), CompileOptions{})
		require.NoError(t, err)
		walk := func(insts []byte) {
			IterateInstructions(insts, func(_ int, op Opcode, _ []int, _ int) bool {
				out = append(out, OpcodeNames[op])
				return true
			})
		}
		walk(cr.Bytecode.Main.Instructions)
		for _, c := range cr.Bytecode.Constants {
			if f, ok := c.(*CompiledFunction); ok {
				walk(f.Instructions)
			}
		}
		return
	}
	require.Subset(t, ops(`x := 1; p := &x`), []string{"GETLOCALPTR", "VARPTR"})
	require.Subset(t, ops(`x := 1; f := func() { return &x }`), []string{"GETFREEPTR", "VARPTR"})
	require.Subset(t, ops(`d := {a: 1}; p := &d.a`), []string{"ADDROFINDEX"})
	require.Subset(t, ops(`global g; p := &g`), []string{"GLOBALS", "ADDROFINDEX"})
}
