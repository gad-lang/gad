package gad

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// The following Go types back the reflected-value structural-contract tests: one
// per ReflectValuer kind (struct, slice, array, map), each exposing Go fields
// and/or methods that a Gad `interface { … }` can check.

type rfPerson struct {
	Name string
	Age  int
}

func (p rfPerson) Greet() string { return "hi " + p.Name }

type rfTags []string

func (t rfTags) Count() int { return len(t) }

type rfVec [3]int

func (v rfVec) Dim() int { return len(v) }

type rfEnv map[string]string

func (e rfEnv) Has(k string) bool { _, ok := e[k]; return ok }

// runReflect wraps v as a reflected Go value, binds it to the global `v`, and
// runs src, returning the result.
func runReflect(t *testing.T, v any, src string) Object {
	t.Helper()
	rv, err := NewReflectValue(v)
	require.NoError(t, err)

	builtins := NewBuiltins()
	st := NewSymbolTable(builtins.NameSet)
	_, err = st.DefineGlobals([]string{"v"})
	require.NoError(t, err)

	_, bc, err := Compile(st, []byte(src), CompileOptions{})
	require.NoErrorf(t, err, "compile: %s", src)

	vm := NewVM(builtins.Build(), bc)
	ret, err := vm.RunOpts(&RunOpts{Globals: Dict{"v": rv}})
	require.NoErrorf(t, err, "run: %s", src)
	return ret
}

// TestReflectStructStructuralContract checks that a reflected Go struct exposes
// its exported fields and methods to Gad, and satisfies an interface that
// requires them (and is rejected when it does not).
func TestReflectStructStructuralContract(t *testing.T) {
	p := rfPerson{Name: "Ann", Age: 30}

	require.Equal(t, Str("Ann"), runReflect(t, p, `return v.Name`))
	require.Equal(t, Int(30), runReflect(t, p, `return v.Age`))
	require.Equal(t, Str("hi Ann"), runReflect(t, p, `return v.Greet()`))

	// satisfies an interface requiring a field...
	require.Equal(t, True, runReflect(t, p,
		`interface Named { Name str }; return v :: Named != nil`))
	// ...and one requiring a method
	require.Equal(t, True, runReflect(t, p,
		`return v :: interface{ Greet() <str> } != nil`))
	// ...and both together
	require.Equal(t, True, runReflect(t, p,
		`return v :: interface{ Name str; Greet() <str> } != nil`))

	// rejected when a required member is missing
	require.Equal(t, Str("rejected"), runReflect(t, p,
		`interface HasZ { Z str }; try { v :: HasZ; return "ok" } catch { return "rejected" }`))
}

// TestReflectSliceStructuralContract checks a named-slice type's Go method
// satisfies a Gad method-interface.
func TestReflectSliceStructuralContract(t *testing.T) {
	tags := rfTags{"a", "b", "c"}
	require.Equal(t, Int(3), runReflect(t, tags, `return v.Count()`))
	require.Equal(t, True, runReflect(t, tags,
		`return v :: interface{ Count() <int> } != nil`))
	// A required *field* the slice lacks rejects it (methods, by contrast, are
	// accepted optimistically — see TestReflectDuckTypedMethods).
	require.Equal(t, Str("rejected"), runReflect(t, tags,
		`try { v :: interface{ label str } ; return "ok" } catch { return "rejected" }`))
}

// TestReflectDuckTypedMethods documents that a required *method* the value does
// not actually define is accepted optimistically (duck typing) at the interface
// check — the mismatch only surfaces if the method is later called.
func TestReflectDuckTypedMethods(t *testing.T) {
	tags := rfTags{"a", "b"}
	require.Equal(t, True, runReflect(t, tags,
		`return v :: interface{ Missing() } != nil`))
}

// TestReflectArrayStructuralContract checks a named-array type's Go method.
func TestReflectArrayStructuralContract(t *testing.T) {
	vec := rfVec{1, 2, 3}
	require.Equal(t, Int(3), runReflect(t, vec, `return v.Dim()`))
	require.Equal(t, True, runReflect(t, vec,
		`return v :: interface{ Dim() <int> } != nil`))
}

// TestReflectMapStructuralContract checks a named-map (dict) type's Go method.
func TestReflectMapStructuralContract(t *testing.T) {
	env := rfEnv{"PATH": "/bin"}
	require.Equal(t, True, runReflect(t, env, `return v.Has("PATH")`))
	require.Equal(t, False, runReflect(t, env, `return v.Has("HOME")`))
	require.Equal(t, True, runReflect(t, env,
		`return v :: interface{ Has(str) <bool> } != nil`))
}

// TestReflectTypeAssigner checks the ReflectType type-assigner: a value of an
// assignable Go type satisfies CanAssign, an incompatible one does not.
func TestReflectTypeAssigner(t *testing.T) {
	rv, err := NewReflectValue(rfPerson{Name: "Ann"})
	require.NoError(t, err)
	rt := rv.GetRType()

	// the same reflected value is assignable to its own type
	ok, err := rt.CanAssign(rv)
	require.NoError(t, err)
	require.True(t, ok)

	// an unrelated Gad value is not
	ok, _ = rt.CanAssign(Int(1))
	require.False(t, ok)
}

// ExampleNewReflectValue_structuralContract embeds a Go value in a script and
// checks it against a Gad interface built from its Go fields and methods.
func ExampleNewReflectValue_structuralContract() {
	// rfPerson is a Go struct with a Name/Age field and a Greet() method.
	rv, _ := NewReflectValue(rfPerson{Name: "Ann", Age: 30})

	builtins := NewBuiltins()
	st := NewSymbolTable(builtins.NameSet)
	_, _ = st.DefineGlobals([]string{"p"})

	src := `
		interface Greeter { Name str; Greet() <str> }
		p :: Greeter          // rejected unless p has the Name field and Greet()
		println(p.Greet())
	`
	_, bc, err := Compile(st, []byte(src), CompileOptions{})
	if err != nil {
		panic(err)
	}
	vm := NewVM(builtins.Build(), bc)
	if _, err = vm.RunOpts(&RunOpts{StdOut: os.Stdout, Globals: Dict{"p": rv}}); err != nil {
		panic(err)
	}
	// Output: hi Ann
}
