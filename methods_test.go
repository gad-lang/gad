package gad

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMethodArgs(t *testing.T) {
	co := &Function{Name: "test"}
	var args MethodArgType
	assert.NoError(t, args.Add(MultipleObjectTypes{{TString}}, &CallerMethod{
		CallerObject: co,
	}, false))
	assert.NoError(t, args.Add(MultipleObjectTypes{{TString}, {TInt}, {TFloat}}, &CallerMethod{
		CallerObject: co,
	}, false))
	assert.Error(t, args.Add(MultipleObjectTypes{{TString}, {TInt}, {TFloat}}, &CallerMethod{
		CallerObject: co,
	}, false))
	assert.NotNil(t, args.GetMethod([]ObjectType{TString, TInt, TFloat}))
	assert.Nil(t, args.GetMethod([]ObjectType{TString, TBool}))
	assert.Nil(t, args.GetMethod([]ObjectType{TString, TInt, TFloat, TText}))
	assert.NotNil(t, args.GetMethod([]ObjectType{TString}))
	assert.Nil(t, args.GetMethod([]ObjectType{}))
}
