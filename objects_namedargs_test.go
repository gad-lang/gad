package gad

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/gad-lang/gad/tests"
	"github.com/stretchr/testify/assert"
)

func TestNamedArgs_All(t *testing.T) {
	tests := []struct {
		args    Dict
		vargs   Dict
		wantRet Dict
	}{
		{Dict{}, Dict{}, Dict{}},
		{Dict{"a": True}, Dict{}, Dict{"a": True}},
		{Dict{"a": True}, Dict{"b": False}, Dict{"a": True, "b": False}},
	}
	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			a1, err := ConvertToKeyValueArray(nil, tt.args)
			assert.NoError(t, err)
			a2, err := ConvertToKeyValueArray(nil, tt.vargs)
			assert.NoError(t, err)
			n := NewNamedArgs(a1, a2)
			assert.Equalf(t, tt.wantRet, n.Dict(), "All()")
		})
	}
}

func TestNamedArgs_CheckNames(t *testing.T) {
	tests := []struct {
		args    Dict
		vargs   Dict
		accept  []string
		wantErr bool
	}{
		{Dict{}, Dict{}, nil, false},
		{Dict{"a": True}, Dict{}, nil, true},
		{Dict{"a": True}, Dict{}, []string{"a"}, false},
		{Dict{"a": True}, Dict{"b": False}, []string{"a"}, true},
	}
	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			a1, err := ConvertToKeyValueArray(nil, tt.args)
			assert.NoError(t, err)
			a2, err := ConvertToKeyValueArray(nil, tt.vargs)
			assert.NoError(t, err)
			n := NewNamedArgs(a1, a2)
			if err := n.CheckNames(tt.accept...); err == nil {
				if tt.wantErr {
					t.Error("want error, but not got")
					t.Failed()
				}
			} else if !tt.wantErr {
				t.Error("not want error, but got=" + err.Error())
				t.Failed()
			}
		})
	}
}

func TestNamedArgs_CheckNamesFromSet(t *testing.T) {
	tests := []struct {
		args    Dict
		vargs   Dict
		accept  []string
		wantErr bool
	}{
		{Dict{}, Dict{}, nil, false},
		{Dict{"a": True}, Dict{}, nil, true},
		{Dict{"a": True}, Dict{}, []string{"a"}, false},
		{Dict{"a": True}, Dict{"b": False}, []string{"a"}, true},
	}
	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			a1, err := ConvertToKeyValueArray(nil, tt.args)
			assert.NoError(t, err)
			a2, err := ConvertToKeyValueArray(nil, tt.vargs)
			assert.NoError(t, err)
			n := NewNamedArgs(a1, a2)
			set := make(map[string]int, len(tt.accept))
			for _, v := range tt.accept {
				set[v] = 0
			}
			if err := n.CheckNamesFromSet(set); err == nil {
				if tt.wantErr {
					t.Error("want error, but not got")
					t.Failed()
				}
			} else if !tt.wantErr {
				t.Error("not want error, but got=" + err.Error())
				t.Failed()
			}
		})
	}
}

func TestNamedArgs_Get(t *testing.T) {
	tests := []struct {
		args    Dict
		vargs   Dict
		dst     []*NamedArgVar
		wantErr bool
	}{
		{Dict{}, Dict{}, nil, false},
		{Dict{"a": True}, Dict{}, nil, true},
		{Dict{"a": True}, Dict{}, []*NamedArgVar{{Name: "a"}}, false},
		{Dict{"a": True}, Dict{}, []*NamedArgVar{{Name: "a", TypeAssertion: TypeAssertionFromTypes(TInt)}}, true},
		{Dict{"a": True}, Dict{}, []*NamedArgVar{{Name: "a", TypeAssertion: TypeAssertionFromTypes(TBool)}}, false},
		{Dict{"a": True}, Dict{"b": False}, []*NamedArgVar{{Name: "a"}}, true},
	}
	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			a1, err := ConvertToKeyValueArray(nil, tt.args)
			assert.NoError(t, err)
			a2, err := ConvertToKeyValueArray(nil, tt.vargs)
			assert.NoError(t, err)
			n := NewNamedArgs(a1, a2)
			if err := n.Get(tt.dst...); err == nil {
				if tt.wantErr {
					t.Error("want error, but not got")
					t.Failed()
				} else {
					for _, dst := range tt.dst {
						if dst.Value != n.GetValue(dst.Name) {
							t.Errorf("bad value of %q: want=%v, got=%v", dst.Name, dst.Value, n.GetValue(dst.Name))
							t.Failed()
						}
					}
				}
			} else if !tt.wantErr {
				t.Error("not want error, but got=" + err.Error())
				t.Failed()
			}
		})
	}
}

func TestNamedArgs_GetVar(t *testing.T) {
	tests_ := []struct {
		args    Dict
		vargs   Dict
		dst     []*NamedArgVar
		other   Dict
		wantErr bool
	}{
		{Dict{}, Dict{}, nil, Dict{}, false},
		{Dict{"a": True}, Dict{}, nil, Dict{"a": True}, false},
		{Dict{"a": True}, Dict{}, []*NamedArgVar{{Name: "a"}}, Dict{}, false},
		{Dict{"a": True}, Dict{}, []*NamedArgVar{{Name: "a", TypeAssertion: TypeAssertionFromTypes(TInt)}}, Dict{}, true},
		{Dict{"a": True}, Dict{}, []*NamedArgVar{{Name: "a", TypeAssertion: TypeAssertionFromTypes(TBool)}}, Dict{}, false},
		{Dict{"a": True}, Dict{"b": False}, []*NamedArgVar{{Name: "a"}}, Dict{"b": False}, false},
		{Dict{"a": True, "c": Int(1)}, Dict{"b": False}, []*NamedArgVar{{Name: "a"}}, Dict{"c": Int(1), "b": False}, false},
	}
	for i, tt := range tests_ {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			a1, err := ConvertToKeyValueArray(nil, tt.args)
			assert.NoError(t, err)
			a2, err := ConvertToKeyValueArray(nil, tt.vargs)
			assert.NoError(t, err)
			n := NewNamedArgs(a1, a2)
			if other, err := n.GetVar(tt.dst...); err == nil {
				if tt.wantErr {
					t.Error("want error, but not got")
					t.Failed()
				} else {
					for _, dst := range tt.dst {
						if dst.Value != n.GetPassedValue(dst.Name) {
							t.Errorf("bad value of %q: want=%v, got=%v", dst.Name, dst.Value, n.GetValue(dst.Name))
							t.Failed()
						}
					}

					if !reflect.DeepEqual(other, tt.other) {
						t.Fatalf("Objects not equal:\nExpected:\n%s\nGot:\n%s\n",
							tests.Sdump(tt.other), tests.Sdump(other))
					}
				}
			} else if !tt.wantErr {
				t.Error("not want error, but got=" + err.Error())
				t.Failed()
			}
		})
	}
}
