package gad_test

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/gad-lang/gad"
)

func TestToInterface(t *testing.T) {
	testCases := []struct {
		object Object
		want   any
	}{
		{object: nil, want: nil},
		{object: Nil, want: nil},
		{object: Int(1), want: int64(1)},
		{object: Str(""), want: ""},
		{object: Str("a"), want: "a"},
		{object: Bytes(nil), want: []byte(nil)},
		{object: Bytes(""), want: []byte{}},
		{object: Bytes("a"), want: []byte{'a'}},
		{object: Array(nil), want: []any{}},
		{object: Array{}, want: []any{}},
		{object: Array{Int(1)}, want: []any{int64(1)}},
		{object: Array{Nil}, want: []any{nil}},
		{object: Dict(nil), want: map[string]any{}},
		{object: Dict{}, want: map[string]any{}},
		{object: Dict{"a": Nil}, want: map[string]any{"a": nil}},
		{object: Dict{"a": Int(1)}, want: map[string]any{"a": int64(1)}},
		{object: Uint(1), want: uint64(1)},
		{object: Char(1), want: rune(1)},
		{object: Float(1), want: float64(1)},
		{object: True, want: true},
		{object: False, want: false},
		{object: (*SyncDict)(nil), want: map[string]any{}},
		{
			object: &SyncDict{Value: Dict{"a": Int(1)}},
			want:   map[string]any{"a": int64(1)},
		},
	}
	for _, tC := range testCases {
		t.Run(fmt.Sprintf("%T", tC.object), func(t *testing.T) {
			if got := ToInterface(tC.object); !reflect.DeepEqual(got, tC.want) {
				t.Errorf("ToInterface() = %v, want %v", got, tC.want)
			}
		})
	}
}

func TestToObject(t *testing.T) {
	err := errors.New("test error")
	fn := func(Call) (Object, error) { return nil, nil }

	testCases := []struct {
		iface   any
		want    Object
		wantErr bool
	}{
		{iface: nil, want: Nil},
		{iface: "a", want: Str("a")},
		{iface: int64(-1), want: Int(-1)},
		{iface: int32(-1), want: Int(-1)},
		{iface: int16(-1), want: Int(-1)},
		{iface: int8(-1), want: Int(-1)},
		{iface: int(1), want: Int(1)},
		{iface: uint(1), want: Uint(1)},
		{iface: uint64(1), want: Uint(1)},
		{iface: uint32(1), want: Uint(1)},
		{iface: uint16(1), want: Uint(1)},
		{iface: uint8(1), want: Uint(1)},
		{iface: uintptr(1), want: Uint(1)},
		{iface: true, want: True},
		{iface: false, want: False},
		{iface: rune(1), want: Int(1)},
		{iface: byte(2), want: Uint(2)},
		{iface: float64(1), want: Float(1)},
		{iface: float32(1), want: Float(1)},
		{iface: []byte(nil), want: Bytes{}},
		{iface: []byte("a"), want: Bytes{'a'}},
		{iface: map[string]Object(nil), want: Dict{}},
		{iface: map[string]Object{"a": Int(1)}, want: Dict{"a": Int(1)}},
		{iface: []Object(nil), want: Array{}},
		{iface: []Object{Int(1), Char('a')}, want: Array{Int(1), Char('a')}},
		{iface: Object(nil), want: Nil},
		{iface: Str("a"), want: Str("a")},
		{iface: CallableFunc(nil), want: Nil},
		{iface: fn, want: &Function{Value: fn}},
		{iface: err, want: &Error{Message: err.Error(), Cause: err}},
		{iface: error(nil), want: Nil},
	}

	for _, tC := range testCases {
		t.Run(fmt.Sprintf("%[1]T:%[1]v", tC.iface), func(t *testing.T) {
			got, err := ToObject(tC.iface)
			if (err != nil) != tC.wantErr {
				t.Errorf("ToObject() error = %v, wantErr %v", err, tC.wantErr)
				return
			}
			if fn, ok := tC.iface.(CallableFunc); ok && fn != nil {
				require.NotNil(t, tC.want.(*Function).Value)
				return
			}
			if !reflect.DeepEqual(got, tC.want) {
				t.Errorf("ToObject() = %[1]v (%[1]T), want %[2]v (%[2]T)", got, tC.want)
			}
		})
	}
}
