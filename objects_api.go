package gad

import (
	"fmt"
	"io"

	"github.com/gad-lang/gad/token"
)

const (
	// True represents a true value.
	True = Bool(true)

	// False represents a false value.
	False = Bool(false)
)

var (
	// Nil represents nil value.
	Nil Object = &NilType{}
)

// Object represents an object in the VM.
type Object interface {
	// TypeName should return the name of the type.
	Type() ObjectType

	// ToString should return a string of the type's value.
	ToString() string

	// IsFalsy returns true if value is falsy otherwise false.
	IsFalsy() bool

	// Equal checks equality of objects.
	Equal(right Object) bool
}

type ObjectType interface {
	Object
	CallerObject
	Name() string
	Getters() Map
	Setters() Map
	Methods() Map
	Fields() Map
	New(*VM, Map) (Object, error)
	IsChildOf(t ObjectType) bool
}

type Objector interface {
	Object
	Fields() Map
}

type ToStringer interface {
	Object
	Stringer(c Call) (String, error)
}

// Copier wraps the Copy method to create a single copy of the object.
type Copier interface {
	Object
	Copy() Object
}

// DeepCopier wraps the Copy method to create a deep copy of the object.
type DeepCopier interface {
	Object
	DeepCopy() Object
}

// IndexDeleter wraps the IndexDelete method to delete an index of an object.
type IndexDeleter interface {
	Object
	IndexDelete(vm *VM, key Object) error
}

// IndexGetter wraps the IndexGet method to get index value.
type IndexGetter interface {
	Object
	// IndexGet should take an index Object and return a result Object or an
	// error for indexable objects. Indexable is an object that can take an
	// index and return an object. Returned error stops VM execution if not
	// handled with an error handler and VM.Run returns the same error as
	// wrapped. If Object is not indexable, ErrNotIndexable should be returned
	// as error.
	IndexGet(vm *VM, index Object) (value Object, err error)
}

// IndexSetter wraps the IndexSet method to set index value.
type IndexSetter interface {
	Object
	// IndexSet should take an index Object and a value Object for index
	// assignable objects. Index assignable is an object that can take an index
	// and a value on the left-hand side of the assignment statement. If Object
	// is not index assignable, ErrNotIndexAssignable should be returned as
	// error. Returned error stops VM execution if not handled with an error
	// handler and VM.Run returns the same error as wrapped.
	IndexSet(vm *VM, index, value Object) error
}

type IndexGetSetter interface {
	IndexGetter
	IndexSetter
}

type Indexer interface {
	IndexGetter
	IndexSetter
	IndexDeleter
}

// LengthGetter wraps the Len method to get the number of elements of an object.
type LengthGetter interface {
	Object
	Len() int
}

// CallerObject is an interface for objects that can be called with Call
// method.
type CallerObject interface {
	Object
	Call(c Call) (Object, error)
}

// CanCallerObject is an interface for objects that can be objects implements
// this CallerObject interface.
// Note if CallerObject implements this interface, CanCall() is called for check
// if object is callable.
type CanCallerObject interface {
	CallerObject
	// CanCall returns true if type can be called with Call() method.
	// VM returns an error if one tries to call a noncallable object.
	CanCall() bool
}

// NameCallerObject is an interface for objects that can be called with CallName
// method to call a method of an object. Objects implementing this interface can
// reduce allocations by not creating a callable object for each method call.
type NameCallerObject interface {
	Object
	CallName(name string, c Call) (Object, error)
}

type ToArrayAppenderObject interface {
	Object
	AppendToArray(arr *Array)
}

// ItemsGetter is an interface for returns pairs of fields or keys with same values.
type ItemsGetter interface {
	Object
	Items() (arr KeyValueArray)
}

// KeysGetter is an interface for returns keys or fields names.
type KeysGetter interface {
	Object
	Keys() (arr Array)
}

// ValuesGetter is an interface for returns values.
type ValuesGetter interface {
	Object
	Values() (arr Array)
}

// Sorter is an interface for return sorted values.
type Sorter interface {
	Object

	// Sort sorts object. if `update`, sort self and return then, other else sorts a self copy object.
	Sort() (Object, error)
}

// ReverseSorter is an interface for return reverse sorted values.
type ReverseSorter interface {
	Object

	// SortReverse sorts object reversely. if `update`, sort self and return then, other else sorts a self copy object.
	SortReverse() (Object, error)
}

type Iterabler interface {
	// Iterate should return an Iterator for the type.
	Iterate(vm *VM) Iterator
}

type CanIterabler interface {
	// CanIterate should return whether the Object can be Iterated.
	CanIterate() bool
}

type Slicer interface {
	LengthGetter
	Slice(low, high int) Object
}

type ToIterfaceConverter interface {
	ToInterface() any
}

type Niler interface {
	Object
	IsNil() bool
}

type Appender interface {
	Object
	Append(arr ...Object) (Object, error)
}

// ObjectImpl is the basic Object implementation and it does not nothing, and
// helps to implement Object interface by embedding and overriding methods in
// custom implementations. String and TypeName must be implemented otherwise
// calling these methods causes panic.
type ObjectImpl struct{}

var _ Object = ObjectImpl{}

func (ObjectImpl) Type() ObjectType {
	panic(ErrNotImplemented)
}

func (ObjectImpl) ToString() string {
	panic(ErrNotImplemented)
}

// Equal implements Object interface.
func (ObjectImpl) Equal(Object) bool { return false }

// IsFalsy implements Object interface.
func (ObjectImpl) IsFalsy() bool { return true }

// NilType represents the type of global Nil Object. One should use
// the NilType in type switches only.
type NilType struct {
	ObjectImpl
}

func (o *NilType) Type() ObjectType {
	return TNil
}

func (o *NilType) ToString() string {
	return "nil"
}

func (o *NilType) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v':
		f.Write([]byte(o.ToString()))
	}
}

// Equal implements Object interface.
func (o *NilType) Equal(right Object) bool {
	return right == Nil
}

func (o *NilType) IsNil() bool {
	return true
}

// BinaryOp implements Object interface.
func (o *NilType) BinaryOp(tok token.Token, right Object) (Object, error) {
	switch right.(type) {
	case *NilType:
		switch tok {
		case token.Less, token.Greater:
			return False, nil
		case token.LessEq, token.GreaterEq:
			return True, nil
		}
	default:
		switch tok {
		case token.Less, token.LessEq:
			return True, nil
		case token.Greater, token.GreaterEq:
			return False, nil
		}
	}
	return nil, NewOperandTypeError(
		tok.String(),
		Nil.Type().Name(),
		right.Type().Name())
}

func Callable(o Object) (ok bool) {
	if _, ok = o.(CallerObject); ok {
		if cc, _ := o.(CanCallerObject); cc != nil {
			ok = cc.CanCall()
		}
	}
	return
}

// BytesConverter is to bytes converter
type BytesConverter interface {
	Object
	ToBytes() (Bytes, error)
}

func Iterable(obj Object) bool {
	if it, _ := obj.(Iterabler); it != nil {
		if cit, _ := obj.(CanIterabler); cit != nil {
			return cit.CanIterate()
		}
		return true
	}
	return false
}

type BinaryOperatorHandler interface {
	// BinaryOp handles +,-,*,/,%,<<,>>,<=,>=,<,> operators.
	// Returned error stops VM execution if not handled with an error handler
	// and VM.Run returns the same error as wrapped.
	BinaryOp(tok token.Token, right Object) (Object, error)
}

type Writer interface {
	Object
	io.Writer
	GoWriter() io.Writer
}

type Reader interface {
	Object
	io.Reader
	GoReader() io.Reader
}

type ReadWriter interface {
	Writer
	Reader
}
