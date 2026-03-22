package gad

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMethodArgs(t *testing.T) {
	co := &Function{FuncName: "test"}
	var args MethodArgType
	assert.NoError(t, args.Add(ParamsTypes{ObjectTypes{TStr}}, NewCallerMethod(nil, co), false, nil))
	assert.NoError(t, args.Add(ParamsTypes{ObjectTypes{TStr}, ObjectTypes{TInt}, ObjectTypes{TFloat}}, NewCallerMethod(nil, co), false, nil))
	assert.Error(t, args.Add(ParamsTypes{ObjectTypes{TStr}, ObjectTypes{TInt}, ObjectTypes{TFloat}}, NewCallerMethod(nil, co), false, nil))
	assert.NotNil(t, args.GetMethod([]ObjectType{TStr, TInt, TFloat}))
	assert.Nil(t, args.GetMethod([]ObjectType{TStr, TBool}))
	assert.Nil(t, args.GetMethod([]ObjectType{TStr, TInt, TFloat, TRawStr}))
	assert.NotNil(t, args.GetMethod([]ObjectType{TStr}))
	assert.Nil(t, args.GetMethod([]ObjectType{}))
}

func TestMethodArgsMixed(t *testing.T) {
	f1 := NewFunction("f1", func(c Call) (_ Object, err error) { return }, FunctionWithParams(func(p func(name string) *ParamBuilder) {
		p("args").Var()
	}))
	f2 := NewFunction("f2", func(c Call) (_ Object, err error) { return }, FunctionWithParams(func(p func(name string) *ParamBuilder) {
		p("i").Type(TInt)
		p("args").Var()
	}))
	f3 := NewFunction("f3", func(c Call) (_ Object, err error) { return }, FunctionWithParams(func(p func(name string) *ParamBuilder) {
		p("s").Type(TStr)
	}))
	f4 := NewFunction("f4", func(c Call) (_ Object, err error) { return }, FunctionWithParams(func(p func(name string) *ParamBuilder) {
		p("i").Type(TInt)
		p("i2").Type(TInt)
		p("args").Type(TInt).Var()
	}))
	f5 := NewFunction("f5", func(c Call) (_ Object, err error) { return }, FunctionWithParams(func(p func(name string) *ParamBuilder) {
		p("v1").Type(TInt)
		p("v2").Type(TInt)
		p("v3").Type(TFloat)
		p("args").Type(TFloat).Var()
	}))
	f6 := NewFunction("f6", func(c Call) (_ Object, err error) { return }, FunctionWithParams(func(p func(name string) *ParamBuilder) {
		p("s").Type(TStr)
		p("s2").Type(TStr)
		p("args").Type(TStr).Var()
	}))
	f7 := NewFunction("f7", func(c Call) (_ Object, err error) { return }, FunctionWithParams(func(p func(name string) *ParamBuilder) {
		p("s").Type(TStr)
		p("s1").Type(TStr)
		p("s2").Type(TStr)
		p("args").Var()
	}))
	f8 := NewFunction("f8", func(c Call) (_ Object, err error) { return }, FunctionWithParams(func(p func(name string) *ParamBuilder) {
		p("s").Type(TStr)
		p("s1").Type(TFloat)
		p("floats").Type(TFloat).Var()
	}))

	vm := NewVM(nil, nil)
	vm.curFrame = &frame{fn: &CompiledFunction{module: NewModule(ModuleInfo{Name: "test"})}}

	fn, err := NewFuncFunc(Call{VM: vm, Args: Args{Array{f1, f2, f3, f4, f5, f6, f7, f8}}})
	assert.NoError(t, err)

	f := fn.(*Func)

	method := func(types ObjectTypeArray) CallerObject {
		m := f.CallerMethodOfArgsTypes(types)
		if m, _ := m.(*TypedCallerMethod); m != nil {
			return m.CallerObject
		}
		return m
	}

	require.Equal(t, f1, method(ObjectTypeArray{}))
	require.Equal(t, f2, method(ObjectTypeArray{TInt}))
	require.Equal(t, f4, method(ObjectTypeArray{TInt, TInt}))
	require.Equal(t, f4, method(ObjectTypeArray{TInt, TInt, TInt}))
	require.Equal(t, f4, method(ObjectTypeArray{TInt, TInt, TInt, TInt}))
	require.Equal(t, f2, method(ObjectTypeArray{TInt, TInt, TInt, TFloat}))
	require.Equal(t, f5, method(ObjectTypeArray{TInt, TInt, TFloat}))
	require.Equal(t, f5, method(ObjectTypeArray{TInt, TInt, TFloat, TFloat}))
	require.Equal(t, f5, method(ObjectTypeArray{TInt, TInt, TFloat, TFloat, TFloat}))
	require.Equal(t, f2, method(ObjectTypeArray{TInt, TInt, TFloat, TFloat, TBool}))
	require.Equal(t, f3, method(ObjectTypeArray{TStr}))
	require.Equal(t, f6, method(ObjectTypeArray{TStr, TStr}))
	require.Equal(t, f7, method(ObjectTypeArray{TStr, TStr, TStr, TStr}))
	require.Equal(t, f8, method(ObjectTypeArray{TStr, TFloat}))
	require.Equal(t, f1, method(ObjectTypeArray{TBool}))
	require.Equal(t, f1, method(ObjectTypeArray{TBool, TChar, TFloat}))

	f9 := NewFunction("f9", func(c Call) (_ Object, err error) { return }, FunctionWithParams(func(p func(name string) *ParamBuilder) {
		p("s").Type(TStr)
		p("f").Type(TFloat)
	}))

	require.ErrorContains(t, f.AddMethod(nil, f9, false, nil), "ErrMethodDuplication: params (str, float): ‹function f9(s str, f float)›. Current method is ‹function f8(s str, s1 float, *floats float)›")
	require.NoError(t, f.AddMethod(nil, f9, true, nil))
	require.Equal(t, f9, method(ObjectTypeArray{TStr, TFloat}))
	require.Equal(t, f8, method(ObjectTypeArray{TStr, TFloat, TFloat}))

	f10 := NewFunction("f10", func(c Call) (_ Object, err error) { return }, FunctionWithParams(func(p func(name string) *ParamBuilder) {
		p("s").Type(TStr)
		p("f").Type(TFloat)
		p("fts").Type(TFloat).Var()
	}))

	require.ErrorContains(t, f.AddMethod(nil, f10, false, nil), "ErrMethodDuplication: params (str, float): ‹function f10(s str, f float, *fts float)›. Current method is ‹function f9(s str, f float)›")
	require.NoError(t, f.AddMethod(nil, f10, true, nil))
	require.Equal(t, f10, method(ObjectTypeArray{TStr, TFloat}))
	require.Equal(t, f10, method(ObjectTypeArray{TStr, TFloat, TFloat}))
}
