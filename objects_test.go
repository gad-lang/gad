package gad_test

import (
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/gad-lang/gad"
)

func TestObjects(t *testing.T) {
	// ensure basic type's ToInterface equality and comparison
	require.True(t, True == true)
	require.True(t, False == false)
	require.True(t, True != False)
	// comparable objects
	comparables := []Object{
		True,
		False,
		Nil,
		Int(-1),
		Int(0),
		Int(1),
		Uint(0),
		Uint(1),
		Char(0),
		Char(1),
		Char('x'),
		Float(0),
		Float(1),
		String(""),
		String("x"),
	}
	for i := range comparables {
		for j := range comparables {
			if i != j {
				require.True(t, comparables[i] != comparables[j],
					"%T and %T must be not equal", comparables[i], comparables[j])
			} else {
				require.True(t, comparables[i] == comparables[j],
					"%T and %T must be equal", comparables[i], comparables[j])
			}
		}
	}
}

func TestObjectIterable(t *testing.T) {
	require.NotNil(t, String("").Iterate(nil))
	require.NotNil(t, Array{}.Iterate(nil))
	require.NotNil(t, Bytes{}.Iterate(nil))
	require.NotNil(t, Map{}.Iterate(nil))
	require.NotNil(t, (&SyncMap{}).Iterate(nil))
}

func TestObjectCallable(t *testing.T) {
	require.False(t, Callable(Int(0)))
	require.False(t, Callable(Uint(0)))
	require.False(t, Callable(Char(0)))
	require.False(t, Callable(Float(0)))
	require.False(t, Callable(Bool(true)))
	require.False(t, Callable(Nil))
	require.False(t, Callable(&Error{}))
	require.False(t, Callable(&RuntimeError{}))
	require.False(t, Callable(String("")))
	require.False(t, Callable(Array{}))
	require.False(t, Callable(Bytes{}))
	require.False(t, Callable(Map{}))
	require.False(t, Callable(&SyncMap{}))

	require.True(t, Callable(&Function{}))
	require.True(t, Callable(&BuiltinFunction{}))
	require.True(t, Callable(&CompiledFunction{}))
	require.True(t, Callable(MustToObject(func() {})))
}

func TestObjectString(t *testing.T) {
	require.Equal(t, "0", Int(0).ToString())
	require.Equal(t, "0", Uint(0).ToString())
	require.Equal(t, "\x00", Char(0).ToString())
	require.Equal(t, "0", Float(0).ToString())
	require.Equal(t, "true", Bool(true).ToString())
	require.Equal(t, "false", Bool(false).ToString())
	require.Equal(t, "nil", Nil.ToString())

	require.Equal(t, "error: ", (&Error{}).ToString())
	require.Equal(t, "error: message", (&Error{Message: "message"}).ToString())
	require.Equal(t, "name: message", (&Error{Name: "name", Message: "message"}).ToString())

	require.Equal(t, "<nil>", (&RuntimeError{}).ToString())

	require.Equal(t, "", String("").ToString())
	require.Equal(t, "xyz", String("xyz").ToString())

	require.Equal(t, "[]", Array{}.ToString())
	require.Equal(t, `[1, "x", 1.1]`, Array{Int(1), String("x"), Float(1.1)}.ToString())

	require.Equal(t, "", Bytes{}.ToString())
	require.Equal(t, "\x00\x01", Bytes{0, 1}.ToString())
	require.Equal(t, "xyz", Bytes(String("xyz")).ToString())
	require.Equal(t, String("xyz").ToString(), Bytes(String("xyz")).ToString())

	require.Equal(t, "{}", Map{}.ToString())
	m := Map{"a": Int(1)}
	require.Equal(t, `{"a": 1}`, m.ToString())
	require.Equal(t, "{}", (&SyncMap{}).ToString())
	require.Equal(t, m.ToString(), (&SyncMap{Value: m}).ToString())
	require.Equal(t, "{}", (&SyncMap{Value: Map{}}).ToString())

	require.Equal(t, "<function:>", (&Function{}).ToString())
	require.Equal(t, "<function:xyz>", (&Function{Name: "xyz"}).ToString())
	require.Equal(t, "<builtinFunction:>", (&BuiltinFunction{}).ToString())
	require.Equal(t, "<builtinFunction:abc>", (&BuiltinFunction{Name: "abc"}).ToString())
	require.Equal(t, "<compiledFunction>", (&CompiledFunction{}).ToString())
	require.Equal(t, "<reflectFunc: func()>", MustToObject(func() {}).ToString())
	require.Equal(t, "<reflectFunc: func(int)>", MustToObject(func(int) {}).ToString())
	require.Equal(t, "<reflectSlice:slice<[]int: []>>", MustToObject([]int{}).ToString())
	var arr [2]int
	arr[1] = 60
	require.Equal(t, "<reflectArray:array<[2]int: [0 60]>>", MustToObject(arr).ToString())
	require.Equal(t, "<reflectMap:map<map[string]int: map[a:2]>>", MustToObject(map[string]int{"a": 2}).ToString())
	require.Equal(t, "<reflectValue:github.com/gad-lang/gad_test.t1<100>>", MustToObject(t1(100)).ToString())
	require.Equal(t, "<reflectValue:github.com/gad-lang/gad_test.t2<@100>>", MustToObject(t2(100)).ToString())
	require.Equal(t, "<reflectValue:github.com/gad-lang/gad_test.t3<#100>>", MustToObject(t3(100)).ToString())
}

func TestObjectTypeName(t *testing.T) {
	require.Equal(t, "int", Int(0).Type().Name())
	require.Equal(t, "uint", Uint(0).Type().Name())
	require.Equal(t, "char", Char(0).Type().Name())
	require.Equal(t, "float", Float(0).Type().Name())
	require.Equal(t, "bool", Bool(true).Type().Name())
	require.Equal(t, "nil", Nil.Type().Name())
	require.Equal(t, "error", (&Error{}).Type().Name())
	require.Equal(t, "error", (&RuntimeError{}).Type().Name())
	require.Equal(t, "string", String("").Type().Name())
	require.Equal(t, "array", Array{}.Type().Name())
	require.Equal(t, "bytes", Bytes{}.Type().Name())
	require.Equal(t, "map", Map{}.Type().Name())
	require.Equal(t, "syncMap", (&SyncMap{}).Type().Name())
	require.Equal(t, "function", (&Function{}).Type().Name())
	require.Equal(t, "builtinFunction", (&BuiltinFunction{}).Type().Name())
	require.Equal(t, "compiledFunction", (&CompiledFunction{}).Type().Name())
	require.Equal(t, "reflect:func", MustToObject(func(int) {}).Type().Name())
	require.Equal(t, "reflect:slice", MustToObject([]int{}).Type().Name())
	var arr [2]int
	arr[1] = 60
	require.Equal(t, "reflect:array", MustToObject(arr).Type().Name())
	require.Equal(t, "reflect:map", MustToObject(map[string]int{"a": 2}).Type().Name())
	require.Equal(t, "reflect:github.com/gad-lang/gad_test.t1", MustToObject(t1(10)).Type().Name())
}

type t1 int

type t2 int

func (v t2) IsZero() bool {
	return v == 1
}

func (v *t2) Format(s fmt.State, verb rune) {
	s.Write([]byte("@"))
	fmt.Fprintf(s, "%"+string(verb), int(*v))
}

type t3 int

func (v *t3) IsZero() bool {
	return (*v) == 2
}

func (v t3) Format(s fmt.State, verb rune) {
	s.Write([]byte("#"))
	fmt.Fprintf(s, "%"+string(verb), int(v))
}

func TestObjectIsFalsy(t *testing.T) {
	require.True(t, Int(0).IsFalsy())
	require.True(t, Uint(0).IsFalsy())
	require.True(t, Char(0).IsFalsy())
	require.False(t, Float(0).IsFalsy())
	require.True(t, Float(math.NaN()).IsFalsy())
	require.False(t, Bool(true).IsFalsy())
	require.True(t, Bool(false).IsFalsy())
	require.True(t, Nil.IsFalsy())
	require.True(t, (&Error{}).IsFalsy())
	require.True(t, (&RuntimeError{}).IsFalsy())
	require.True(t, String("").IsFalsy())
	require.False(t, String("x").IsFalsy())
	require.True(t, Array{}.IsFalsy())
	require.False(t, Array{Int(0)}.IsFalsy())
	require.True(t, Bytes{}.IsFalsy())
	require.False(t, Bytes{0}.IsFalsy())
	require.True(t, Map{}.IsFalsy())
	require.False(t, Map{"a": Int(1)}.IsFalsy())
	require.True(t, (&SyncMap{}).IsFalsy())
	require.False(t, (&SyncMap{Value: Map{"a": Int(1)}}).IsFalsy())
	require.False(t, (&Function{}).IsFalsy())
	require.False(t, (&BuiltinFunction{}).IsFalsy())
	require.False(t, (&CompiledFunction{}).IsFalsy())
	require.True(t, MustToObject([]int{}).IsFalsy())
	require.False(t, MustToObject([]int{1}).IsFalsy())
	var arr [2]int
	arr[1] = 60
	require.False(t, MustToObject(arr).IsFalsy())
	require.False(t, MustToObject(map[string]int{"a": 2}).IsFalsy())
	require.True(t, MustToObject(map[string]int{}).IsFalsy())
	require.False(t, MustToObject(t1(10)).IsFalsy())
	require.True(t, MustToObject(t1(0)).IsFalsy())
	require.False(t, MustToObject(t2(0)).IsFalsy())
	require.True(t, MustToObject(t2(1)).IsFalsy())
	require.False(t, MustToObject(t3(0)).IsFalsy())
	require.True(t, MustToObject(t3(2)).IsFalsy())
}

func TestObjectCopier(t *testing.T) {
	objects := []Object{
		Array{},
		Bytes{},
		Map{},
		&SyncMap{},
	}
	for _, o := range objects {
		if _, ok := o.(Copier); !ok {
			t.Fatalf("%T must implement Copier interface", o)
		}
	}
}

func TestObjectImpl(t *testing.T) {
	var o any = ObjectImpl{}
	if _, ok := o.(Object); !ok {
		t.Fatal("ObjectImpl must implement Object interface")
	}
	impl := ObjectImpl{}
	require.Panics(t, func() { _ = impl.ToString() })
	require.Panics(t, func() { _ = impl.Type().Name() })
	require.False(t, impl.Equal(impl))
	require.True(t, impl.IsFalsy())
	require.False(t, Callable(impl))
	require.False(t, Iterable(impl))
}

func TestObjectIndexGet(t *testing.T) {
	v, err := (&Error{}).IndexGet(nil, Nil)
	require.NoError(t, err)
	require.Equal(t, Nil, v)

	v, err = (&Error{}).IndexGet(nil, String("Literal"))
	require.NoError(t, err)
	require.Equal(t, String(""), v)

	v, err = (&Error{Name: "x"}).IndexGet(nil, String("Literal"))
	require.NoError(t, err)
	require.Equal(t, String("x"), v)

	v, err = (&Error{}).IndexGet(nil, String("Message"))
	require.NoError(t, err)
	require.Equal(t, String(""), v)

	v, err = (&Error{Message: "x"}).IndexGet(nil, String("Message"))
	require.NoError(t, err)
	require.Equal(t, String("x"), v)

	v, err = (&RuntimeError{}).IndexGet(nil, Nil)
	require.Equal(t, Nil, v)
	require.NoError(t, err)

	v, err = (&RuntimeError{Err: &Error{}}).IndexGet(nil, String("Literal"))
	require.NoError(t, err)
	require.Equal(t, String(""), v)

	v, err = (&RuntimeError{Err: &Error{Name: "x"}}).IndexGet(nil, String("Literal"))
	require.NoError(t, err)
	require.Equal(t, String("x"), v)

	v, err = (&RuntimeError{Err: &Error{}}).IndexGet(nil, String("Message"))
	require.NoError(t, err)
	require.Equal(t, String(""), v)

	v, err = (&RuntimeError{Err: &Error{Message: "x"}}).IndexGet(nil, String("Message"))
	require.NoError(t, err)
	require.Equal(t, String("x"), v)

	v, err = String("").IndexGet(nil, Nil)
	require.Nil(t, v)
	require.NotNil(t, err)
	require.True(t, errors.Is(err, ErrType))

	v, err = String("x").IndexGet(nil, Int(0))
	require.NotNil(t, v)
	require.Nil(t, err)
	require.Equal(t, Int("x"[0]), v)

	v, err = String("x").IndexGet(nil, Int(0))
	require.NotNil(t, v)
	require.Nil(t, err)
	require.Equal(t, Int("x"[0]), v)

	v, err = String("x").IndexGet(nil, Int(1))
	require.Nil(t, v)
	require.Equal(t, ErrIndexOutOfBounds, err)

	v, err = Array{Int(1)}.IndexGet(nil, Nil)
	require.NotNil(t, err)
	require.Nil(t, v)
	require.True(t, errors.Is(err, ErrType))

	v, err = Array{Int(1)}.IndexGet(nil, Int(0))
	require.NotNil(t, v)
	require.Nil(t, err)
	require.Equal(t, Int(1), v)

	v, err = Array{Int(1)}.IndexGet(nil, Int(1))
	require.Nil(t, v)
	require.NotNil(t, err)
	require.Equal(t, ErrIndexOutOfBounds, err)

	v, err = Bytes{1}.IndexGet(nil, Nil)
	require.NotNil(t, err)
	require.Nil(t, v)
	require.True(t, errors.Is(err, ErrType))

	v, err = Bytes{1}.IndexGet(nil, Int(0))
	require.NotNil(t, v)
	require.Nil(t, err)
	require.Equal(t, Int(1), v)

	v, err = Bytes{1}.IndexGet(nil, Int(1))
	require.Nil(t, v)
	require.NotNil(t, err)
	require.Equal(t, ErrIndexOutOfBounds, err)

	v, err = Map{}.IndexGet(nil, Nil)
	require.Nil(t, err)
	require.Equal(t, Nil, v)

	v, err = Map{"a": Int(1)}.IndexGet(nil, Int(0))
	require.Nil(t, err)
	require.Equal(t, Nil, v)

	v, err = Map{"a": Int(1)}.IndexGet(nil, String("a"))
	require.Nil(t, err)
	require.Equal(t, Int(1), v)

	v, err = (&SyncMap{Value: Map{}}).IndexGet(nil, Nil)
	require.Nil(t, err)
	require.Equal(t, Nil, v)

	v, err = (&SyncMap{Value: Map{"a": Int(1)}}).IndexGet(nil, Int(0))
	require.Nil(t, err)
	require.Equal(t, Nil, v)

	v, err = (&SyncMap{Value: Map{"a": Int(1)}}).IndexGet(nil, String("a"))
	require.Nil(t, err)
	require.Equal(t, Int(1), v)
}

func TestObjectIndexSet(t *testing.T) {
	var v IndexGetSetter = Array{Int(1)}
	err := v.IndexSet(nil, Int(0), Int(2))
	require.NoError(t, err)
	require.Equal(t, Int(2), v.(Array)[0])

	v = Array{Int(1)}
	err = v.IndexSet(nil, Int(1), Int(3))
	require.Equal(t, ErrIndexOutOfBounds, err)
	require.Equal(t, Array{Int(1)}, v)

	v = Array{Int(1)}
	err = v.IndexSet(nil, String("x"), Int(3))
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrType))

	v = Bytes{1}
	err = v.IndexSet(nil, Int(0), Int(2))
	require.NoError(t, err)
	require.Equal(t, byte(2), v.(Bytes)[0])

	v = Bytes{1}
	err = v.IndexSet(nil, Int(1), Int(2))
	require.Error(t, err)
	require.Equal(t, ErrIndexOutOfBounds, err)

	v = Bytes{1}
	err = v.IndexSet(nil, Int(0), String(""))
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrType))

	v = Bytes{1}
	err = v.IndexSet(nil, String("x"), Int(1))
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrType))

	v = Map{}
	err = v.IndexSet(nil, Nil, Nil)
	require.Nil(t, err)
	require.Equal(t, Nil, v.(Map)["nil"])

	v = Map{"a": Int(1)}
	err = v.IndexSet(nil, String("a"), Int(2))
	require.Nil(t, err)
	require.Equal(t, Int(2), v.(Map)["a"])

	v = &SyncMap{Value: Map{}}
	err = v.IndexSet(nil, Nil, Nil)
	require.Nil(t, err)
	require.Equal(t, Nil, v.(*SyncMap).Value["nil"])

	v = &SyncMap{Value: Map{"a": Int(1)}}
	err = v.IndexSet(nil, String("a"), Int(2))
	require.Nil(t, err)
	require.Equal(t, Int(2), v.(*SyncMap).Value["a"])
}
