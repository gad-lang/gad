package gad

import (
	"reflect"
)

// Pointer is what the `&` operator yields: a reference through which a value is
// read and written in place. In Gad `p.v` reads it (PtrGet) and `p.v = x`
// writes it (PtrSet); a `*T` parameter/field type accepts a Pointer whose
// current value is a T. Every implementation reports the `ptr` type.
//
// The core implementations are VarPtr (a variable: `&x`) and IndexPtr (a member
// of any IndexGetter: `&obj.field`, `&arr[i]`). A Go type may implement
// Pointer too — e.g. to hand a script a reference to host state — and should
// then answer `.v` by calling PtrIndexGet / PtrIndexSet from its IndexGet /
// IndexSet.
type Pointer interface {
	Object
	// PtrGet returns the value pointed to.
	PtrGet(vm *VM) (Object, error)
	// PtrSet replaces the value pointed to.
	PtrSet(vm *VM, value Object) error
}

// Addressable lets a type hand out its own Pointer for a member, e.g. one
// backed by host memory. AddrOf returns (nil, nil) to fall back to the generic
// IndexPtr.
type Addressable interface {
	Object
	AddrOf(vm *VM, key Object) (Pointer, error)
}

// TPtr is the object type of every Pointer (`typeName(&x)` is "ptr").
var TPtr = RegisterBuiltinType(BuiltinPtr, "ptr", (*VarPtr)(nil), nil).TypeKey()

// AddrOf returns a Pointer to target[key] — what `&target.key` and
// `&target[key]` evaluate, and the entry point for Go hosts. An Addressable
// target supplies its own pointer; any other IndexGetter gets an IndexPtr, which
// reads and writes through IndexGet / IndexSet (so a class instance's property
// accessors and typed-field checks apply).
func AddrOf(vm *VM, target Object, key Object) (Pointer, error) {
	if a, ok := target.(Addressable); ok {
		p, err := a.AddrOf(vm, key)
		if p != nil || err != nil {
			return p, err
		}
	}
	if g, ok := target.(IndexGetter); ok {
		return &IndexPtr{Target: g, Key: key}, nil
	}
	return nil, ErrNotIndexable.NewError(target.Type().Name())
}

// PtrIndexGet implements `.v` (the value) for a Pointer's IndexGet.
func PtrIndexGet(vm *VM, p Pointer, index Object) (Object, error) {
	if index.ToString() == "v" {
		return p.PtrGet(vm)
	}
	return nil, ErrInvalidIndex.NewError(index.ToString())
}

// PtrIndexSet implements `.v = x` for a Pointer's IndexSet.
func PtrIndexSet(vm *VM, p Pointer, index, value Object) error {
	if index.ToString() == "v" {
		return p.PtrSet(vm, value)
	}
	return ErrInvalidIndex.NewError(index.ToString())
}

// ptrString renders a pointer as `&value` (or `&?` when it cannot be read).
func ptrString(p Pointer) string {
	v, err := p.PtrGet(nil)
	if err != nil || v == nil {
		return "&?"
	}
	return "&" + v.ToString()
}

// --- VarPtr ---

// VarPtr points to a variable: the stack slot promoted to a shared cell — the
// same cell a closure capturing the variable uses — so writes through the
// pointer change the variable itself, and the pointer stays valid after the
// declaring function returns.
type VarPtr struct {
	Cell *ObjectPtr
}

var (
	_ Pointer        = (*VarPtr)(nil)
	_ IndexGetSetter = (*VarPtr)(nil)
)

// NewVarPtr returns a pointer to the value held by cell (a fresh cell holding
// value when cell is nil).
func NewVarPtr(cell *ObjectPtr, value Object) *VarPtr {
	if cell == nil {
		if value == nil {
			value = Nil
		}
		cell = &ObjectPtr{Value: &value}
	}
	return &VarPtr{Cell: cell}
}

func (p *VarPtr) Type() ObjectType { return TPtr }
func (p *VarPtr) IsFalsy() bool    { return false }
func (p *VarPtr) ToString() string { return ptrString(p) }

// Equal reports whether x points to the same variable.
func (p *VarPtr) Equal(x Object) bool {
	o, ok := x.(*VarPtr)
	return ok && o.Cell == p.Cell
}

func (p *VarPtr) PtrGet(*VM) (Object, error) {
	if p.Cell.Value == nil || *p.Cell.Value == nil {
		return Nil, nil
	}
	return *p.Cell.Value, nil
}

func (p *VarPtr) PtrSet(_ *VM, value Object) error {
	if p.Cell.Value == nil {
		p.Cell.Value = &value
		return nil
	}
	*p.Cell.Value = value
	return nil
}

func (p *VarPtr) IndexGet(vm *VM, index Object) (Object, error) {
	return PtrIndexGet(vm, p, index)
}

func (p *VarPtr) IndexSet(vm *VM, index, value Object) error {
	return PtrIndexSet(vm, p, index, value)
}

// --- IndexPtr ---

// IndexPtr points to target[key] of any IndexGetter. The target object and the
// key are fixed when the pointer is taken; every read/write goes through
// IndexGet / IndexSet, so it is live (a property getter runs on each read) and
// honours the target's rules (typed class fields, read-only members). A target
// that is not an IndexSetter gives a read-only pointer.
//
// It points to the OBJECT, not to the variable holding it: after `q :=
// &xs[1]`, an `xs += …` that reallocates the array leaves q on the old one —
// point to the variable (`&xs`) when it may be replaced.
type IndexPtr struct {
	Target IndexGetter
	Key    Object
}

var (
	_ Pointer        = (*IndexPtr)(nil)
	_ IndexGetSetter = (*IndexPtr)(nil)
)

func (p *IndexPtr) Type() ObjectType { return TPtr }
func (p *IndexPtr) IsFalsy() bool    { return false }
func (p *IndexPtr) ToString() string { return ptrString(p) }

// Equal reports whether x points to the same key of the same object.
func (p *IndexPtr) Equal(x Object) bool {
	o, ok := x.(*IndexPtr)
	return ok && sameObject(o.Target, p.Target) && o.Key.Equal(p.Key)
}

func (p *IndexPtr) PtrGet(vm *VM) (Object, error) {
	return p.Target.IndexGet(vm, p.Key)
}

func (p *IndexPtr) PtrSet(vm *VM, value Object) error {
	s, ok := p.Target.(IndexSetter)
	if !ok {
		return ErrNotIndexAssignable.NewError(p.Target.Type().Name())
	}
	return s.IndexSet(vm, p.Key, value)
}

func (p *IndexPtr) IndexGet(vm *VM, index Object) (Object, error) {
	return PtrIndexGet(vm, p, index)
}

func (p *IndexPtr) IndexSet(vm *VM, index, value Object) error {
	return PtrIndexSet(vm, p, index, value)
}

// sameObject reports whether a and b are the same object: identical pointers,
// or for slice/map-backed values (Array, Dict, …) the same backing storage.
func sameObject(a, b Object) bool {
	ra, rb := reflect.ValueOf(a), reflect.ValueOf(b)
	if ra.Type() != rb.Type() {
		return false
	}
	switch ra.Kind() {
	case reflect.Slice:
		return ra.Len() == rb.Len() && (ra.Len() == 0 || ra.Pointer() == rb.Pointer())
	case reflect.Map, reflect.Ptr, reflect.Func, reflect.Chan, reflect.UnsafePointer:
		return ra.Pointer() == rb.Pointer()
	}
	return ra.Type().Comparable() && a == b
}

// --- GoPtr ---

// GoPtr is a Go pointer to a basic value (`*int`, `*string`, `*float64`,
// `*bool`, …) handed to a script — `vm.RunOpts(…Args{&cfg.Port}…)` or any
// ToObject(&v). It is a `ptr`: `p.v` reads the Go variable and `p.v = x`
// converts x to the variable's Go type and writes it, so the host sees the
// change. (A pointer to a struct, slice or map keeps its reflected form, whose
// members already read and write the Go memory: `&goStruct.Field` points into
// it through an IndexPtr.)
type GoPtr struct {
	ReflectValue
}

var (
	_ Pointer        = (*GoPtr)(nil)
	_ IndexGetSetter = (*GoPtr)(nil)
	_ ReflectValuer  = (*GoPtr)(nil)
)

func (p *GoPtr) Type() ObjectType { return TPtr }
func (p *GoPtr) IsFalsy() bool    { return false }
func (p *GoPtr) ToString() string { return ptrString(p) }

// Copy keeps pointer semantics: a copy points to the same Go variable.
func (p *GoPtr) Copy() Object { return p }

// Print renders the pointer like any other (`&value`), not as a reflected value.
func (p *GoPtr) Print(state *PrinterState) error {
	return state.WriteString(p.ToString())
}

// Equal reports whether x points to the same Go variable.
func (p *GoPtr) Equal(x Object) bool {
	o, ok := x.(*GoPtr)
	return ok && o.RValue.CanAddr() && p.RValue.CanAddr() &&
		o.RValue.Addr().Pointer() == p.RValue.Addr().Pointer()
}

func (p *GoPtr) PtrGet(vm *VM) (Object, error) {
	if vm != nil {
		return vm.ToObject(p.RValue.Interface())
	}
	return ToObject(p.RValue.Interface())
}

func (p *GoPtr) PtrSet(vm *VM, value Object) (err error) {
	var gv any
	if vm != nil {
		gv = vm.ToInterface(value)
	} else {
		gv = ToInterface(value)
	}
	v := reflect.ValueOf(gv)
	if !v.IsValid() {
		p.RValue.Set(reflect.Zero(p.RValue.Type()))
		return nil
	}
	if v, err = prepareArg(v, p.RValue.Type()); err != nil {
		return err
	}
	p.RValue.Set(v)
	return nil
}

func (p *GoPtr) IndexGet(vm *VM, index Object) (Object, error) {
	return PtrIndexGet(vm, p, index)
}

func (p *GoPtr) IndexSet(vm *VM, index, value Object) error {
	return PtrIndexSet(vm, p, index, value)
}

// isBasicKind reports the reflect kinds a GoPtr points to.
func isBasicKind(k reflect.Kind) bool {
	switch k {
	case reflect.Bool, reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}
