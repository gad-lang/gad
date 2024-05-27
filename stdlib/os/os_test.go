package os

import (
	"reflect"
	"testing"

	"github.com/gad-lang/gad"
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
	expectRun(t, "return os.FileFlag.ReadOnly", nil, OReadOnly)
	expectRun(t, "return os.FileFlag.ReadOnly|os.FileFlag.WriteOnly", nil, OReadOnly|OWriteOnly)
	expectRun(t, "return os.FileFlag()", nil, OReadOnly)
	expectRun(t, "return os.FileFlag(0)", nil, OReadOnly)
	expectRun(t, "return os.FileFlag(1|1052672)", nil, OReadOnly|OWriteOnly|OSync)
	expectRun(t, "return os.FileFlag(0u)", nil, OReadOnly)
	expectRun(t, `return os.FileFlag(1u|1052672u)`, nil, OReadOnly|OWriteOnly|OSync)
	expectRun(t, `return os.FileFlag("")`, nil, OReadOnly)
	expectRun(t, `return os.FileFlag("read_only|write_only|sync")`, nil, OReadOnly|OWriteOnly|OSync)
}

func expectRun(t *testing.T, script string, opts *gad.TestOpts, expect gad.Object) {
	if opts == nil {
		opts = gad.NewTestOpts()
	}
	opts = opts.Module("os", Module)
	script = `const os = import("os");` + script
	gad.TestExpectRun(t, script, opts, expect)
}
