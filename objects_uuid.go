package gad

import (
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"strings"
)

// TUUID is the type `uuid`: a global builtin — no namespace —, as `str` is.
// Called, it is the constructor (NewUUIDFunc).
var TUUID BuiltinObjTypeKey

func init() {
	typ := RegisterBuiltinType(BuiltinUUID, "uuid", UUID{}, NewUUIDFunc)
	TUUID = typ.TypeKey()
	// the constructor's forms, typed: uuid(v str), uuid(v rawstr),
	// uuid(v bytes), uuid(v uuid) — as calendarDate's
	addTypeCtors(typ, "uuid", NewUUIDFunc, TStr, TRawStr, TBytes, TUUID)
}

// UUID is a universally unique identifier, the 16 bytes of RFC 9562; its
// text is the canonical lower-case `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`.
type UUID [16]byte

// NilUUID is the zero UUID, all of its bits 0: falsy.
var NilUUID UUID

// NewUUIDFunc is the uuid(...) constructor:
//
//	uuid()          a new random UUID (version 4)
//	uuid("…")       the UUID of a text: canonical, 32 hex digits, `{…}`,
//	                `urn:uuid:…`
//	uuid(bytes)     the UUID of 16 bytes
//	uuid(u uuid)    u
func NewUUIDFunc(c Call) (Object, error) {
	if err := c.Args.CheckMaxLen(1); err != nil {
		return Nil, err
	}
	if c.Args.Length() == 0 {
		return NewUUIDv4()
	}
	switch v := c.Args.Get(0).(type) {
	case UUID:
		return v, nil
	case Str:
		return ParseUUID(string(v))
	case RawStr:
		return ParseUUID(string(v))
	case Bytes:
		if len(v) != 16 {
			return Nil, ErrType.NewError("uuid: " + strconv.Itoa(len(v)) + " bytes, want 16")
		}
		var u UUID
		copy(u[:], v)
		return u, nil
	}
	return Nil, NewArgumentTypeError("1st", "str|bytes|uuid", c.Args.Get(0).Type().Name())
}

// NewUUIDv4 is a new random UUID (version 4, variant RFC 9562).
func NewUUIDv4() (UUID, error) {
	var u UUID
	if _, err := rand.Read(u[:]); err != nil {
		return NilUUID, err
	}
	u[6] = u[6]&0x0f | 0x40
	u[8] = u[8]&0x3f | 0x80
	return u, nil
}

// ParseUUID is the UUID of s: the canonical text (any case), its 32 hex
// digits alone, in braces (`{…}`) or after `urn:uuid:`.
func ParseUUID(s string) (UUID, error) {
	var u UUID
	t := strings.TrimSpace(s)
	if len(t) > 9 && strings.EqualFold(t[:9], "urn:uuid:") {
		t = t[9:]
	}
	if len(t) == 38 && t[0] == '{' && t[37] == '}' {
		t = t[1:37]
	}
	switch len(t) {
	case 36:
		if t[8] != '-' || t[13] != '-' || t[18] != '-' || t[23] != '-' {
			return NilUUID, ErrType.NewError("uuid: invalid text " + strconv.Quote(s))
		}
		t = t[:8] + t[9:13] + t[14:18] + t[19:23] + t[24:]
	case 32:
	default:
		return NilUUID, ErrType.NewError("uuid: invalid text " + strconv.Quote(s))
	}
	if _, err := hex.Decode(u[:], []byte(t)); err != nil {
		return NilUUID, ErrType.NewError("uuid: invalid text " + strconv.Quote(s))
	}
	return u, nil
}

func (UUID) Type() ObjectType { return TUUID }

// ToString is its canonical text, lower case.
func (u UUID) ToString() string {
	var b [36]byte
	hex.Encode(b[0:8], u[0:4])
	b[8] = '-'
	hex.Encode(b[9:13], u[4:6])
	b[13] = '-'
	hex.Encode(b[14:18], u[6:8])
	b[18] = '-'
	hex.Encode(b[19:23], u[8:10])
	b[23] = '-'
	hex.Encode(b[24:], u[10:])
	return string(b[:])
}

func (u UUID) String() string { return u.ToString() }

// IsFalsy reports whether it is the nil UUID.
func (u UUID) IsFalsy() bool { return u == NilUUID }

// Equal is a UUID of the same bytes, or a text of it.
func (u UUID) Equal(right Object) bool {
	switch v := right.(type) {
	case UUID:
		return u == v
	case Str:
		o, err := ParseUUID(string(v))
		return err == nil && o == u
	}
	return false
}

// Print writes its text (Printabler).
func (u UUID) Print(s *PrinterState) error {
	if s.IsRepr {
		defer s.WrapRepr(u)()
	}
	return s.WriteString(u.ToString())
}

// MarshalJSON encodes it as its text, a JSON string.
func (u UUID) MarshalJSON() ([]byte, error) { return []byte(strconv.Quote(u.ToString())), nil }

// IndexGet: `version`, the version its bits say (4 random, 7 by time…);
// `bytes`, its 16 bytes.
func (u UUID) IndexGet(_ *VM, index Object) (Object, error) {
	switch index.ToString() {
	case "version":
		return Int(u[6] >> 4), nil
	case "bytes":
		return Bytes(append([]byte(nil), u[:]...)), nil
	}
	return nil, ErrInvalidIndex.NewError(index.ToString())
}
