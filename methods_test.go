package gad

import (
	"testing"

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
