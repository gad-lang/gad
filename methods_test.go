package gad

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

	fn, err := NewFuncFunc(Call{Args: Args{Array{f1, f2, f3, f4, f5, f6, f7, f8}}})
	assert.NoError(t, err)

	f := fn.(*Func)

	assert.Equal(t, f1, f.CallerMethodOfArgsTypes(ObjectTypeArray{}))
	assert.Equal(t, f2, f.CallerMethodOfArgsTypes(ObjectTypeArray{TInt}))
	assert.Equal(t, f4, f.CallerMethodOfArgsTypes(ObjectTypeArray{TInt, TInt}))
	assert.Equal(t, f4, f.CallerMethodOfArgsTypes(ObjectTypeArray{TInt, TInt, TInt}))
	assert.Equal(t, f4, f.CallerMethodOfArgsTypes(ObjectTypeArray{TInt, TInt, TInt, TInt}))
	assert.Equal(t, f2, f.CallerMethodOfArgsTypes(ObjectTypeArray{TInt, TInt, TInt, TFloat}))
	assert.Equal(t, f5, f.CallerMethodOfArgsTypes(ObjectTypeArray{TInt, TInt, TFloat}))
	assert.Equal(t, f5, f.CallerMethodOfArgsTypes(ObjectTypeArray{TInt, TInt, TFloat, TFloat}))
	assert.Equal(t, f5, f.CallerMethodOfArgsTypes(ObjectTypeArray{TInt, TInt, TFloat, TFloat, TFloat}))
	assert.Equal(t, f2, f.CallerMethodOfArgsTypes(ObjectTypeArray{TInt, TInt, TFloat, TFloat, TBool}))
	assert.Equal(t, f3, f.CallerMethodOfArgsTypes(ObjectTypeArray{TStr}))
	assert.Equal(t, f6, f.CallerMethodOfArgsTypes(ObjectTypeArray{TStr, TStr}))
	assert.Equal(t, f7, f.CallerMethodOfArgsTypes(ObjectTypeArray{TStr, TStr, TStr, TStr}))
	assert.Equal(t, f8, f.CallerMethodOfArgsTypes(ObjectTypeArray{TStr, TFloat}))
	assert.Equal(t, f1, f.CallerMethodOfArgsTypes(ObjectTypeArray{TBool}))
	assert.Equal(t, f1, f.CallerMethodOfArgsTypes(ObjectTypeArray{TBool, TChar, TFloat}))

	f9 := NewFunction("f9", func(c Call) (_ Object, err error) { return }, FunctionWithParams(func(p func(name string) *ParamBuilder) {
		p("s").Type(TStr)
		p("f").Type(TFloat)
	}))

	assert.ErrorContains(t, f.AddMethod(nil, f9, false, nil), "ErrMethodDuplication: params (str, float): ‹function f9(s str, f float)›. Current method is ‹function f8(s str, s1 float, *floats float)›")
	assert.NoError(t, f.AddMethod(nil, f9, true, nil))
	assert.Equal(t, f9, f.CallerMethodOfArgsTypes(ObjectTypeArray{TStr, TFloat}))
	assert.Equal(t, f8, f.CallerMethodOfArgsTypes(ObjectTypeArray{TStr, TFloat, TFloat}))

	f10 := NewFunction("f10", func(c Call) (_ Object, err error) { return }, FunctionWithParams(func(p func(name string) *ParamBuilder) {
		p("s").Type(TStr)
		p("f").Type(TFloat)
		p("fts").Type(TFloat).Var()
	}))

	assert.ErrorContains(t, f.AddMethod(nil, f10, false, nil), "ErrMethodDuplication: params (str, float): ‹function f10(s str, f float, *fts float)›. Current method is ‹function f9(s str, f float)›")
	assert.NoError(t, f.AddMethod(nil, f10, true, nil))
	assert.Equal(t, f10, f.CallerMethodOfArgsTypes(ObjectTypeArray{TStr, TFloat}))
	assert.Equal(t, f10, f.CallerMethodOfArgsTypes(ObjectTypeArray{TStr, TFloat, TFloat}))
}
