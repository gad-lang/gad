// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import "strings"

// TTypeUnion is the meta type of TypeUnion values.
var TTypeUnion = NewBuiltinObjType("typeUnion")

// NumberTypeUnion is the builtin `number` type union: int|uint|float|decimal.
// A value satisfies `number` when it is any of those numeric types. Its members
// are filled in newNumberTypeUnion (called from registration) rather than in a
// var initializer, because the TInt/… type keys are themselves package-level
// vars whose initialization order relative to this one is undefined.
var NumberTypeUnion = &TypeUnion{UName: "number"}

// newNumberTypeUnion populates NumberTypeUnion's members. It is called once the
// numeric type keys are known (see the builtins registration).
func newNumberTypeUnion() *TypeUnion {
	NumberTypeUnion.Types = []Object{TInt, TUint, TFloat, TDecimal}
	return NumberTypeUnion
}

// TypeUnion is a union of types (`int|uint`, `str|number`, `int|interface{…}`):
// a value satisfies it when it satisfies any member. Members may be ObjectTypes,
// interfaces, method interfaces, func-header types or other TypeUnions, so
// unions nest — `str|number` reuses the `number` union. It is usable anywhere a
// type is expected: parameter and return types, and the `::` cast.
type TypeUnion struct {
	// UName is the union's name when it is a named/builtin union (e.g. "number"),
	// or "" for an anonymous `type a|b` union.
	UName string
	// Types are the member types.
	Types []Object
}

var (
	_ Object        = (*TypeUnion)(nil)
	_ vmCanAssigner = (*TypeUnion)(nil)
	_ TypeAssigner  = (*TypeUnion)(nil)
)

// NewTypeUnion returns an (anonymous) union of the given member types, flattening
// any member that is itself a TypeUnion.
func NewTypeUnion(types ...Object) *TypeUnion {
	u := &TypeUnion{}
	for _, t := range types {
		if inner, ok := t.(*TypeUnion); ok {
			u.Types = append(u.Types, inner.Types...)
		} else {
			u.Types = append(u.Types, t)
		}
	}
	return u
}

func (u *TypeUnion) Type() ObjectType { return TTypeUnion }

// ToString renders the union name, or its `a|b|c` member list when anonymous.
func (u *TypeUnion) ToString() string {
	if u.UName != "" {
		return u.UName
	}
	parts := make([]string, len(u.Types))
	for i, t := range u.Types {
		parts[i] = typeDisplayName(t)
	}
	return strings.Join(parts, "|")
}

func (u *TypeUnion) IsFalsy() bool { return len(u.Types) == 0 }

// Equal reports whether two unions have the same members in order.
func (u *TypeUnion) Equal(right Object) bool {
	r, ok := right.(*TypeUnion)
	if !ok || len(u.Types) != len(r.Types) {
		return false
	}
	for i := range u.Types {
		if !u.Types[i].Equal(r.Types[i]) {
			return false
		}
	}
	return true
}

// CanAssignVM reports whether obj satisfies any member of the union, reusing the
// per-type assignability (AssignToType), so nested unions and interfaces resolve
// recursively.
func (u *TypeUnion) CanAssignVM(vm *VM, obj Object) (bool, error) {
	for _, m := range u.Types {
		if _, err := AssignToType(vm, obj, m); err == nil {
			return true, nil
		}
	}
	return false, nil
}

// CanAssign is CanAssignVM without a VM (structural members needing one are
// relaxed).
func (u *TypeUnion) CanAssign(obj Object) (bool, error) { return u.CanAssignVM(nil, obj) }

// AssignTo makes *TypeUnion a TypeAssigner: obj is assignable to the union `to`
// when it satisfies any of its members. Returns obj unchanged on success.
func (u *TypeUnion) AssignTo(vm *VM, obj Object, to TypeAssigner) (Object, error) {
	tu, ok := to.(*TypeUnion)
	if !ok {
		return nil, ErrIncompatibleCast
	}
	if ok, err := tu.CanAssignVM(vm, obj); err != nil {
		return nil, err
	} else if ok {
		return obj, nil
	}
	return nil, ErrIncompatibleCast
}

// typeDisplayName returns a type value's display name for a union body.
func typeDisplayName(t Object) string {
	switch v := t.(type) {
	case ObjectType:
		return v.Name()
	case *Interface:
		return v.Name()
	case *TypeUnion:
		return v.ToString()
	}
	return t.ToString()
}
