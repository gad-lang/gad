package gad

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReflectFunction_Call(t *testing.T) {
	tests := []struct {
		name    string
		fn      any
		args    Array
		want    any
		wantErr error
	}{
		{"1", func(m map[string]any) any {
			return m
		}, Array{Dict{"a": Int(1)}}, map[string]any{"a": int64(1)}, nil},
		{"2", func(i int) int { return i }, Array{Int(1)}, int64(1), nil},
		{"3", func(i int, args ...int) int {
			for _, arg := range args {
				i += arg
			}
			return i
		}, Array{Int(1), Int(2)}, int64(3), nil},
	}
	vm := (&VM{}).Setup(SetupOpts{})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewReflectValue(tt.fn)
			if !checkError(t, fmt.Sprintf("NewReflectValue(%T)", tt.fn), tt.wantErr, err) {
				return
			}

			c := Call{VM: vm, Args: Args{tt.args}}
			got, err := r.(*ReflectFunc).Call(c)
			if !checkError(t, fmt.Sprintf("Call(%T)", c), tt.wantErr, err) {
				return
			}
			gota := ToInterface(got)
			assert.Equalf(t, tt.want, gota, "Call(%v)", c)
		})
	}
}

func TestReflectStruct_Copy(t *testing.T) {
	type A struct {
		X int
	}
	var a = &A{1}
	v, err := NewReflectValue(a)
	require.NoError(t, err)
	v2 := v.Copy()
	require.Equal(t, a, ToInterface(v2))
	a.X = 3
	require.NotEqual(t, a, ToInterface(v2))
}

func TestReflectStruct_IndexGet(t *testing.T) {
	type a struct {
		X int
	}
	type b struct {
		V1 a
		V2 *a
	}

	tests := []struct {
		name    string
		obj     any
		key     string
		want    Object
		wantErr error
	}{
		{"1", a{}, "X", Int(0), nil},
		{"2", a{2}, "X", Int(2), nil},
		{"3", &a{2}, "X", Int(2), nil},
		{"4", b{}, "V1.X", Int(0), nil},
		{"5", &b{}, "V1.X", Int(0), nil},
		{"6", &b{V2: &a{}}, "V2.X", Int(0), nil},
		{"7", &b{V2: &a{X: 3}}, "V2.X", Int(3), nil},
	}
	vm := (&VM{}).Setup(SetupOpts{})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewReflectValue(tt.obj)
			if !checkError(t, fmt.Sprintf("NewReflectValue(%T)", tt.obj), tt.wantErr, err) {
				return
			}
			var (
				obj Object
				got Object
			)
			obj = r
			for _, key := range strings.Split(tt.key, ".") {
				if got, err = obj.(*ReflectStruct).IndexGet(vm, Str(key)); err == nil {
					obj = got
				} else if !checkError(t, fmt.Sprintf("IndexGet(%v)", key), tt.wantErr, err) {
					return
				}
			}
			assert.Equalf(t, tt.want, got, "IndexGet(%v)", tt.key)
		})
	}
}

func TestReflectSlice_Copy(t *testing.T) {
	var a = []int{1, 2}
	v, err := NewReflectValue(a)
	require.NoError(t, err)
	v2 := v.Copy()
	require.Equal(t, a, ToInterface(v2))
	a[1] = 3
	require.NotEqual(t, a, ToInterface(v2))

	v3, err := NewReflectValue(&a)
	require.NoError(t, err)
	v4 := v3.Copy()
	require.Equal(t, ToInterface(v3), ToInterface(v4))
	a[1] = 10
	require.NotEqual(t, fmt.Sprint(ToInterface(v3)), fmt.Sprint(ToInterface(v4)))
}

func TestReflectSlice_Insert(t *testing.T) {
	var a = []int{3, 4}
	v, err := NewReflectValue(a)
	require.NoError(t, err)
	vm := NewVM(nil).Init()
	s := v.(*ReflectSlice)

	v2, err := s.Insert(vm, 0)
	assert.NoError(t, err)
	assert.Equal(t, v2.ToString(), "‹reflectSlice:slice‹[]int: [3 4]››")

	v2, err = s.Insert(vm, 0, Int(0))
	assert.NoError(t, err)
	assert.Equal(t, v2.ToString(), "‹reflectSlice:slice‹[]int: [0 3 4]››")

	v2, err = s.Insert(vm, -2, Int(0))
	assert.NoError(t, err)
	assert.Equal(t, v2.ToString(), "‹reflectSlice:slice‹[]int: [0 3 4]››")

	v2, err = s.Insert(vm, 0, Int(0), Int(1), Int(2))
	assert.NoError(t, err)
	assert.Equal(t, v2.ToString(), "‹reflectSlice:slice‹[]int: [0 1 2 3 4]››")

	for _, i := range []int{1, -1} {
		v2, err = s.Insert(vm, i)
		assert.NoError(t, err)
		assert.Equal(t, v2.ToString(), "‹reflectSlice:slice‹[]int: [3 4]››")

		v2, err = s.Insert(vm, i, Int(0))
		assert.NoError(t, err)
		assert.Equal(t, v2.ToString(), "‹reflectSlice:slice‹[]int: [3 0 4]››")

		v2, err = s.Insert(vm, i, Int(0), Int(1), Int(2))
		assert.NoError(t, err)
		assert.Equal(t, v2.ToString(), "‹reflectSlice:slice‹[]int: [3 0 1 2 4]››")
	}

	v2, err = s.Insert(vm, 2)
	assert.NoError(t, err)
	assert.Equal(t, v2.ToString(), "‹reflectSlice:slice‹[]int: [3 4]››")

	v2, err = s.Insert(vm, 2, Int(0))
	assert.NoError(t, err)
	assert.Equal(t, v2.ToString(), "‹reflectSlice:slice‹[]int: [3 4 0]››")

	v2, err = s.Insert(vm, 2, Int(0), Int(1), Int(2))
	assert.NoError(t, err)
	assert.Equal(t, v2.ToString(), "‹reflectSlice:slice‹[]int: [3 4 0 1 2]››")

	_, err = s.Insert(vm, -3)
	assert.EqualError(t, err, "InvalidIndexError: negative position is greather then slice length")
}

func TestReflectArray_Copy(t *testing.T) {
	var a = [2]int{1, 2}
	v, err := NewReflectValue(a)
	require.NoError(t, err)
	v2 := v.Copy()
	require.Equal(t, a, ToInterface(v2))
	a[1] = 3
	require.NotEqual(t, a, ToInterface(v2))

	v3, err := NewReflectValue(&a)
	require.NoError(t, err)
	v4 := v3.Copy()
	require.Equal(t, ToInterface(v3), ToInterface(v4))
	a[1] = 10
	require.NotEqual(t, fmt.Sprint(ToInterface(v3)), fmt.Sprint(ToInterface(v4)))
}

func TestReflectMap_Copy(t *testing.T) {
	var a = map[int]int{5: 9}
	v, err := NewReflectValue(a)
	require.NoError(t, err)
	v2 := v.Copy()
	require.Equal(t, a, ToInterface(v2))
	a[1] = 3
	require.NotEqual(t, a, ToInterface(v2))

	v3, err := NewReflectValue(&a)
	require.NoError(t, err)
	v4 := v3.Copy()
	require.Equal(t, ToInterface(v3), ToInterface(v4))
	a[1] = 10
	require.NotEqual(t, fmt.Sprint(ToInterface(v3)), fmt.Sprint(ToInterface(v4)))
}

func TestReflectMap_IndexGet(t *testing.T) {
	tests := []struct {
		name    string
		obj     any
		key     string
		want    Object
		wantErr error
	}{
		{"1", map[string]any{"X": 1}, "X", Int(1), nil},
		{"2", map[string]any{"x": map[string]any{"y": 1}}, "x.y", Int(1), nil},
	}
	vm := (&VM{}).Setup(SetupOpts{})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewReflectValue(tt.obj)
			if !checkError(t, fmt.Sprintf("NewReflectValue(%T)", tt.obj), tt.wantErr, err) {
				return
			}
			var (
				obj Object
				got Object
			)
			obj = r
			for _, key := range strings.Split(tt.key, ".") {
				if got, err = obj.(*ReflectMap).IndexGet(vm, Str(key)); err == nil {
					obj = got
				} else if !checkError(t, fmt.Sprintf("IndexGet(%v)", key), tt.wantErr, err) {
					return
				}
			}
			assert.Equalf(t, tt.want, got, "IndexGet(%v)", tt.key)
		})
	}
}

func TestReflectSlice_IndexGet(t *testing.T) {
	tests := []struct {
		name    string
		obj     any
		key     Object
		want    Object
		wantErr error
	}{
		{"1", []string{"a"}, Int(0), Str("a"), nil},
	}
	vm := (&VM{}).Setup(SetupOpts{})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewReflectValue(tt.obj)
			if !checkError(t, fmt.Sprintf("NewReflectValue(%T)", tt.obj), tt.wantErr, err) {
				return
			}

			got, err := Val(r.(IndexGetter).IndexGet(vm, tt.key))
			if !checkError(t, fmt.Sprintf("IndexGet(%T)", tt.obj), tt.wantErr, err) {
				return
			}

			assert.Equalf(t, tt.want, got, "IndexGet(%v)", tt.key)
		})
	}
}

func TestReflectStruct_IndexSet(t *testing.T) {
	type a struct {
		X int
	}
	type b struct {
		V1 a
		V2 *a
	}

	f := &a{}
	tests := []struct {
		name    string
		obj     any
		key     Object
		value   Object
		wantErr error
	}{
		{"1", a{}, Str("X"), Int(1), nil},
		{"2", &a{}, Str("X"), Int(1), nil},
		{"3", b{}, Str("V2"), MustToObject(f), nil},
		{"4", &b{}, Str("V2"), MustToObject(f), nil},
		{"5", &b{}, Str("V2"), MustToObject(nil), nil},
	}
	vm := (&VM{}).Setup(SetupOpts{})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewReflectValue(tt.obj)
			if !checkError(t, fmt.Sprintf("NewReflectValue(%T)", tt.obj), tt.wantErr, err) {
				return
			}
			obj := r.(*ReflectStruct)

			err = obj.IndexSet(vm, tt.key, tt.value)
			if !checkError(t, fmt.Sprintf("IndexSet(%T)", tt.obj), tt.wantErr, err) {
				return
			}
			got, err := obj.IndexGet(vm, tt.key)
			if !checkError(t, fmt.Sprintf("IndexGet(%T)", tt.obj), tt.wantErr, err) {
				return
			}
			assert.True(t, got.Equal(tt.value), "IndexGet(%v)", tt.key)
		})
	}
}

func TestReflectMap_IndexSet(t *testing.T) {
	tests := []struct {
		name    string
		obj     any
		key     Object
		value   Object
		wantErr error
	}{
		{"6", map[string]any{}, Str("a"), Str("b"), nil},
	}
	vm := (&VM{}).Setup(SetupOpts{})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewReflectValue(tt.obj)
			if !checkError(t, fmt.Sprintf("NewReflectValue(%T)", tt.obj), tt.wantErr, err) {
				return
			}
			obj := r.(*ReflectMap)

			err = obj.IndexSet(vm, tt.key, tt.value)
			if !checkError(t, fmt.Sprintf("IndexSet(%T)", tt.obj), tt.wantErr, err) {
				return
			}
			got, err := obj.IndexGet(vm, tt.key)
			if !checkError(t, fmt.Sprintf("IndexGet(%T)", tt.obj), tt.wantErr, err) {
				return
			}
			assert.True(t, got.Equal(tt.value), "IndexGet(%v)", tt.key)
		})
	}
}

func TestReflectSlice_IndexSet(t *testing.T) {
	tests := []struct {
		name    string
		obj     any
		key     Object
		value   Object
		wantErr error
	}{
		{"7", []string{""}, Int(0), Str("a"), nil},
	}
	vm := (&VM{}).Setup(SetupOpts{})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewReflectValue(tt.obj)
			if !checkError(t, fmt.Sprintf("NewReflectValue(%T)", tt.obj), tt.wantErr, err) {
				return
			}
			obj := r.(*ReflectSlice)

			err = obj.IndexSet(vm, tt.key, tt.value)
			if !checkError(t, fmt.Sprintf("IndexSet(%T)", tt.obj), tt.wantErr, err) {
				return
			}
			got, err := obj.IndexGet(vm, tt.key)
			if !checkError(t, fmt.Sprintf("IndexGet(%T)", tt.obj), tt.wantErr, err) {
				return
			}
			assert.True(t, got.Equal(tt.value), "IndexGet(%v)", tt.key)
		})
	}
}

func checkError(t *testing.T, label string, want, got error) bool {
	t.Helper()
	if want != nil {
		if got == nil {
			t.Errorf("%s: expected error, but not got.", label)
			return false
		}
		if !assert.Equal(t, fmt.Sprintf("%[1]T: %[1]v", want),
			fmt.Sprintf("%[1]T: %[1]v", got),
			label) {
			return false
		}
	} else if got != nil {
		t.Errorf("%s: unexpected expected error: %[1]T: %[1]v.", label, got)
		return false
	}
	return true
}
