package gad

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestMethodArgs(t *testing.T) {
	co := &Function{Name: "test"}
	var args MethodArgType
	assert.NoError(t, args.Add(MultipleObjectTypes{{TStr}}, &CallerMethod{
		CallerObject: co,
	}, false))
	assert.NoError(t, args.Add(MultipleObjectTypes{{TStr}, {TInt}, {TFloat}}, &CallerMethod{
		CallerObject: co,
	}, false))
	assert.Error(t, args.Add(MultipleObjectTypes{{TStr}, {TInt}, {TFloat}}, &CallerMethod{
		CallerObject: co,
	}, false))
	assert.NotNil(t, args.GetMethod([]ObjectType{TStr, TInt, TFloat}))
	assert.Nil(t, args.GetMethod([]ObjectType{TStr, TBool}))
	assert.Nil(t, args.GetMethod([]ObjectType{TStr, TInt, TFloat, TRawStr}))
	assert.NotNil(t, args.GetMethod([]ObjectType{TStr}))
	assert.Nil(t, args.GetMethod([]ObjectType{}))
}

func TestMethodArgsMixed(t *testing.T) {
	typeName := &Function{
		Name: "type_name",
		Value: func(c Call) (Object, error) {
			return Str("type_name_result:" + c.Args.Get(0).Type().String()), nil
		},
	}

	f := NewCallerObjectWithMethods(&Function{
		Name: "fn",
		Value: func(c Call) (Object, error) {
			return Str("fn_result:" + c.Args.Get(0).Type().String()), nil
		},
	})

	assert.NoError(t, f.AddCallerMethod(nil, MultipleObjectTypes{{TDecimal, TInt, TFloat}}, &CallerMethod{
		CallerObject: typeName,
	}, false))

	assert.Equal(t, `‹function:fn› with 3 methods:
	1. ‹function:type_name›(decimal)
	2. ‹function:type_name›(float)
	3. ‹function:type_name›(int)`, f.ToString())

	var tests = []struct {
		name string
		v    any
		s    string
		ret  string
	}{
		{"", "", "‹function:fn›", "fn_result:‹builtinType:str›"},
		{"", false, "‹function:fn›", "fn_result:‹builtinType:bool›"},
		{"", 0, "‹function:type_name›", "type_name_result:‹builtinType:int›"},
		{"", 12.2, "‹function:type_name›", "type_name_result:‹builtinType:float›"},
		{"", decimal.Zero, "‹function:type_name›", "type_name_result:‹builtinType:decimal›"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := MustToObject(tt.v)
			m := f.CallerMethodOfArgs(Args{{o}})
			if m == nil {
				m = f.Caller()
			}
			assert.Equal(t, tt.s, m.ToString())
			ret, err := m.Call(Call{Args: Args{{o}}})
			assert.NoError(t, err)
			assert.NotNil(t, ret)
			assert.Equal(t, tt.ret, ret.ToString())
		})
	}
}
