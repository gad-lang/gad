package gad

import (
	"strings"
	"testing"

	"github.com/gad-lang/gad/parser/node"
	"github.com/stretchr/testify/require"
)

// uuidCompiled is src compiled: the UUIDs among its constants, and whether its
// main code calls something.
func uuidCompiled(t *testing.T, src string) (uuids []UUID, calls bool) {
	t.Helper()
	res, err := Compile(NewSymbolTable(NewBuiltins().NameSet), []byte(src), CompileOptions{})
	require.NoError(t, err)
	for _, c := range res.Bytecode.Constants {
		if u, ok := c.(UUID); ok {
			uuids = append(uuids, u)
		}
	}
	IterateInstructions(res.Bytecode.Main.Instructions, func(_ int, op Opcode, _ []int, _ int) bool {
		if op == OpCall || op == OpCallName {
			calls = true
		}
		return true
	})
	return
}

// uuid("…") — the builtin, a text literal alone — compiles to the UUID
// constant: read when the code compiles, no call as it runs; the same text
// twice, one constant. Its text in any form uuid("…") reads, in any literal.
func TestUUIDConstant(t *testing.T) {
	want, err := ParseUUID("01a0a168-6707-748f-b487-c0ab78e25c11")
	require.NoError(t, err)
	for _, src := range []string{
		`return uuid("01A0A168-6707-748F-B487-C0AB78E25C11")`,
		"return uuid(`01a0a1686707748fb487c0ab78e25c11`)",
		`return uuid("{01a0a168-6707-748f-b487-c0ab78e25c11}")`,
		`return uuid("urn:uuid:01a0a168-6707-748f-b487-c0ab78e25c11")`,
		"return uuid(```\n01a0a168-6707-748f-b487-c0ab78e25c11\n```)",
		`f := func() => uuid("01a0a168-6707-748f-b487-c0ab78e25c11"); return f()`,
	} {
		uuids, _ := uuidCompiled(t, src)
		require.Equal(t, []UUID{want}, uuids, src)
		require.Equal(t, want, runUUIDSrc(t, src), src)
	}
	uuids, calls := uuidCompiled(t, `return uuid("01a0a168-6707-748f-b487-c0ab78e25c11")`)
	require.Len(t, uuids, 1)
	require.False(t, calls, "a constant is no call")

	uuids, _ = uuidCompiled(t, `return [uuid("01a0a168-6707-748f-b487-c0ab78e25c11"), uuid("01A0A1686707748FB487C0AB78E25C11")]`)
	require.Len(t, uuids, 1, "the same UUID, one constant")
}

// Any other uuid(…) is a call: no argument (a new one each time), a value
// that is not a literal, an interpolated text, bytes, a named argument; and a
// uuid that is not the builtin — a name of the code's own.
func TestUUIDConstantOnlyOfTheBuiltin(t *testing.T) {
	for _, src := range []string{
		`return uuid()`,
		`s := "01a0a168-6707-748f-b487-c0ab78e25c11"; return uuid(s)`,
		`x := "01a0a168"; return uuid(#"{x}-6707-748f-b487-c0ab78e25c11")`,
		`return uuid(bytes(16))`,
		`uuid := func(s) => s; return uuid("01a0a168-6707-748f-b487-c0ab78e25c11")`,
		`f := func(uuid) => uuid("01a0a168-6707-748f-b487-c0ab78e25c11"); return f(func(s) => s)`,
	} {
		uuids, calls := uuidCompiled(t, src)
		require.Empty(t, uuids, src)
		require.True(t, calls, src)
	}
	// the code's own uuid: what it answers, not a UUID
	require.Equal(t, Str("01a0a168-6707-748f-b487-c0ab78e25c11"),
		runSrc(t, `uuid := func(s) => s; return uuid("01a0a168-6707-748f-b487-c0ab78e25c11")`))
	// a name an application set in place of the builtin: its own index
	bi := NewBuiltins()
	bi.Set("uuid", NewFunction("uuid", func(c Call) (Object, error) { return Str("app"), nil }))
	res, err := Compile(NewSymbolTable(bi.NameSet), []byte(`return uuid("01a0a168-6707-748f-b487-c0ab78e25c11")`), CompileOptions{})
	require.NoError(t, err)
	for _, c := range res.Bytecode.Constants {
		_, isUUID := c.(UUID)
		require.False(t, isUUID, "folded the application's uuid")
	}
}

// A text that is no UUID stays a call: its error the run's, as ever — a try
// catches it.
func TestUUIDConstantInvalid(t *testing.T) {
	uuids, calls := uuidCompiled(t, `return uuid("nope")`)
	require.Empty(t, uuids)
	require.True(t, calls)
	require.Equal(t, Str("refused"), runSrc(t, `try { uuid("nope"); return "taken" } catch { return "refused" }`))
}

func runSrc(t *testing.T, src string) Object {
	t.Helper()
	res, err := Compile(NewSymbolTable(NewBuiltins().NameSet), []byte(src), CompileOptions{})
	require.NoError(t, err)
	ret, err := NewVM(NewBuiltins().Build(), res.Bytecode).Run()
	require.NoError(t, err)
	return ret
}

func runUUIDSrc(t *testing.T, src string) UUID {
	t.Helper()
	ret := runSrc(t, src)
	u, ok := ret.(UUID)
	require.True(t, ok, "%s: %v (%s)", src, ret, strings.TrimSpace(ret.Type().Name()))
	return u
}

// No syntax of its own: uuid("…") is parsed as the call it is written as —
// the constant is the compiler's (TestUUIDConstant).
func TestUUIDConstantParsedAsCall(t *testing.T) {
	f, err := ParseSource("x.gad", []byte(`return uuid("01a0a168-6707-748f-b487-c0ab78e25c11")`), SourceKindGad)
	require.NoError(t, err)
	ret, ok := f.Stmts[0].(*node.ReturnStmt)
	require.True(t, ok, "%T", f.Stmts[0])
	call, ok := ret.Result.(*node.CallExpr)
	require.True(t, ok, "%T", ret.Result)
	require.Equal(t, "uuid", call.Func.(*node.IdentExpr).Name)
	_, ok = call.CallArgs.Args.Values[0].(*node.StrLit)
	require.True(t, ok)
}
