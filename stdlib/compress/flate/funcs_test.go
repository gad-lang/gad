package flate

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/gad-lang/gad"
	"github.com/stretchr/testify/assert"
)

func TestEncode(t *testing.T) {
	var (
		buf    bytes.Buffer
		r      = gad.NewReader(strings.NewReader("abc"))
		w      = gad.NewWriter(&buf)
		_, err = Encode(gad.Call{Args: gad.Args{gad.Array{w, r}}})
	)

	assert.NoError(t, err)
	assert.Equal(t, "4a4c4a06040000ffff", hex.EncodeToString(buf.Bytes()))
}

func TestDecode(t *testing.T) {
	var (
		compressed, _ = hex.DecodeString("4a4c4a06040000ffff")
		buf           bytes.Buffer
		r             = gad.NewReader(bytes.NewReader(compressed))
		w             = gad.NewWriter(&buf)
		_, err        = Decode(gad.Call{Args: gad.Args{gad.Array{w, r}}})
	)
	assert.NoError(t, err)
	assert.Equal(t, "abc", buf.String())
}
