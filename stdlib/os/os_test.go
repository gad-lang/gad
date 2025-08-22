package os

import (
	"reflect"
	"testing"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/test_helper"
	"github.com/stretchr/testify/assert"
)

func TestNewFileMode(t *testing.T) {
	tests := []struct {
		name    string
		args    gad.Array
		want    gad.Object
		wantErr error
	}{
		{"int", gad.Array{gad.Int(OTrunc)}, OTrunc, nil},
		{"int2", gad.Array{gad.Int(OTrunc | OSync)}, OTrunc | OSync, nil},
		{"uint", gad.Array{gad.Uint(OTrunc)}, OTrunc, nil},
		{"uint2", gad.Array{gad.Uint(OTrunc | OSync)}, OTrunc | OSync, nil},
		{"str", gad.Array{gad.Str(OTrunc.String())}, OTrunc, nil},
		{"str2", gad.Array{gad.Str((OTrunc | OSync).String())}, OTrunc | OSync, nil},
		{"rawstr", gad.Array{gad.RawStr(OTrunc.String())}, nil, gad.NewArgumentTypeErrorT("0st", gad.TRawStr, gad.TInt, gad.TUint, gad.TStr)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewFileMode(gad.Call{Args: gad.Args{tt.args}})

			if (err != nil) != (tt.wantErr != nil) {
				t.Errorf("NewFileMode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewFileMode() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRunNewFileMode(t *testing.T) {
	expectRun(t, "return os.FileFlag", nil, TFileFlag)
	expectRun(t, "return os.FileFlag.RO", nil, ORo)
	expectRun(t, "return os.FileFlag.RO|os.FileFlag.WO", nil, ORo|OWo)
	expectRun(t, "return os.FileFlag()", nil, ORo)
	expectRun(t, "return os.FileFlag(0)", nil, ORo)
	expectRun(t, "return os.FileFlag(1|1052672)", nil, ORo|OWo|OSync)
	expectRun(t, "return os.FileFlag(0u)", nil, ORo)
	expectRun(t, `return os.FileFlag(1u|1052672u)`, nil, ORo|OWo|OSync)
	expectRun(t, `return os.FileFlag("")`, nil, ORo)
	expectRun(t, `return os.FileFlag("ro|wo|sync")`, nil, ORo|OWo|OSync)
}

func expectRun(t *testing.T, script string, opts *test_helper.VMTestOpts, expect gad.Object) {
	if opts == nil {
		opts = test_helper.NewVMTestOpts()
	}
	opts = opts.Module("os", Module)
	script = `const os = import("os");` + script
	test_helper.VMTestExpectRun(t, script, opts, expect)
}
