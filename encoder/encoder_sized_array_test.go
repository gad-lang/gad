package encoder_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/gad-lang/gad"
	"github.com/stretchr/testify/require"

	. "github.com/gad-lang/gad/encoder"
)

// An array type with sizes survives the bytecode: its levels, and the sizes
// naming variables.
func TestEncDecSizedArrayTypes(t *testing.T) {
	// (an overloaded func is left out: its dispatch caches, on the compiled
	// functions, what the decoded ones have not computed yet)
	testEncDecBytecode(t, `
	n := 2
	f := (a [n]str) => len(a)
	g := (a [2..3][2]int) => len(a)
	const t = type [2..3][2]str
	refused := func() { try { [1] :: [n]int; return false } catch { return true } }
	return [f(["a", "b"]), g([[1, 2], [3, 4]]), t.@size[0] == [nil, 2, 3], str(t), refused()]`, nil,
		gad.Array{gad.Int(2), gad.Int(2), gad.True, gad.Str("[2..3][2]str"), gad.True})
}

// The V1 format (no sizes) is still read.
func TestDecArrayTypeV1(t *testing.T) {
	at := &gad.ArrayType{Depth: 2, Elem: gad.ParamType{{Name: "int", Scope: gad.ScopeBuiltin, Index: 5}}, ElemNames: []string{"int"}}
	var buf bytes.Buffer
	ctx := NewWriteContext(context.Background(), NewWriter(&buf))
	require.NoError(t, ArrayTypeV1.Encode(ctx, at))
	v, err := ArrayTypeV1.Decode(NewReadContext(NewReader(bytes.NewReader(buf.Bytes()))))
	require.NoError(t, err)
	got := v.(*gad.ArrayType)
	require.Equal(t, "[][]int", got.Name())
	require.Nil(t, got.Levels)
}

// The default of an interface field survives the encoding (InterfaceFieldV2),
// and V1 (no default) is still read.
func TestEncDecInterfaceFieldDefault(t *testing.T) {
	for _, def := range []gad.Object{gad.Str("table"), gad.Nil, gad.Int(3), nil} {
		f := &gad.InterfaceField{Name: "layout", Nullable: true, Default: def}
		var buf bytes.Buffer
		require.NoError(t, InterfaceFieldV2.Encode(NewWriteContext(context.Background(), NewWriter(&buf)), f))
		v, err := InterfaceFieldV2.Decode(NewReadContext(NewReader(bytes.NewReader(buf.Bytes()))))
		require.NoError(t, err)
		got := v.(*gad.InterfaceField)
		require.Equal(t, "layout", got.Name)
		require.Equal(t, def, got.Default)
	}
	var buf bytes.Buffer
	require.NoError(t, InterfaceFieldV1.Encode(NewWriteContext(context.Background(), NewWriter(&buf)), &gad.InterfaceField{Name: "x"}))
	v, err := InterfaceFieldV1.Decode(NewReadContext(NewReader(bytes.NewReader(buf.Bytes()))))
	require.NoError(t, err)
	require.Nil(t, v.(*gad.InterfaceField).Default)
}
