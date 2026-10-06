package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// The forms of a UUID's text: canonical (any case), its 32 hex digits, in
// braces, after `urn:uuid:` — one UUID; anything else refused.
func TestParseUUID(t *testing.T) {
	const want = "01a0a168-6707-748f-b487-c0ab78e25c11"
	for _, s := range []string{
		want, "01A0A168-6707-748F-B487-C0AB78E25C11", "01a0a1686707748fb487c0ab78e25c11",
		"{01a0a168-6707-748f-b487-c0ab78e25c11}", "urn:uuid:01a0a168-6707-748f-b487-c0ab78e25c11",
		"  " + want + "\n",
	} {
		u, err := ParseUUID(s)
		if err != nil || u.ToString() != want {
			t.Errorf("ParseUUID(%q) = %s, %v", s, u, err)
		}
	}
	for _, s := range []string{"", "nope", "01a0a168-6707-748f-b487-c0ab78e25c1", "01a0a168x6707-748f-b487-c0ab78e25c11",
		"01a0a168-6707-748f-b487-c0ab78e25cgg"} {
		if _, err := ParseUUID(s); err == nil {
			t.Errorf("ParseUUID(%q): want an error", s)
		}
	}
}

// A new UUID is random, of version 4 and the RFC's variant.
func TestNewUUIDv4(t *testing.T) {
	a, _ := NewUUIDv4()
	b, _ := NewUUIDv4()
	if a == b || a[6]>>4 != 4 || a[8]&0xc0 != 0x80 {
		t.Errorf("%s %s", a, b)
	}
}

// `uuid` is a global type, as `str` is: its constructor, its values, a type
// of parameters and of fields; equal to a UUID of the same bytes, or to its
// text; the nil UUID falsy.
func TestUUIDType(t *testing.T) {
	testExpectRun(t, `
		u := uuid("01A0A168-6707-748F-B487-C0AB78E25C11")
		f := func(x uuid) => str(x)
		class C { id? uuid }
		return [typeName(u), str(u), u.version, uuid().version, uuid(u.bytes) == u,
			u == "01a0a168-6707-748f-b487-c0ab78e25c11", f(u), str(C(; id=u).id),
			bool(uuid("00000000000000000000000000000000")), bool(u)]`,
		nil, Array{Str("uuid"), Str("01a0a168-6707-748f-b487-c0ab78e25c11"), Int(7), Int(4), True,
			True, Str("01a0a168-6707-748f-b487-c0ab78e25c11"), Str("01a0a168-6707-748f-b487-c0ab78e25c11"),
			False, True})
	testExpectRun(t, `try { uuid("nope"); return "taken" } catch { return "refused" }`, nil, Str("refused"))
	testExpectRun(t, `try { f := func(x uuid) => x; f("x"); return "taken" } catch { return "refused" }`, nil, Str("refused"))
}
