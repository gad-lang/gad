package gad

import (
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
	TypeName() string

	// String should return a string of the type's value.
	String() string

	// BinaryOp handles +,-,*,/,%,<<,>>,<=,>=,<,> operators.
	// Returned error stops VM execution if not handled with an error handler
	// and VM.Run returns the same error as wrapped.
	BinaryOp(tok token.Token, right Object) (Object, error)

	// IsFalsy returns true if value is falsy otherwise false.
	IsFalsy() bool

	// Equal checks equality of objects.
	Equal(right Object) bool

	// Iterate should return an Iterator for the type.
	Iterate() Iterator

	// CanIterate should return whether the Object can be Iterated.
	CanIterate() bool

	// IndexGet should take an index Object and return a result Object or an
	// error for indexable objects. Indexable is an object that can take an
	// index and return an object. Returned error stops VM execution if not
	// handled with an error handler and VM.Run returns the same error as
	// wrapped. If Object is not indexable, ErrNotIndexable should be returned
	// as error.
	IndexGet(index Object) (value Object, err error)

	// IndexSet should take an index Object and a value Object for index
	// assignable objects. Index assignable is an object that can take an index
	// and a value on the left-hand side of the assignment statement. If Object
	// is not index assignable, ErrNotIndexAssignable should be returned as
	// error. Returned error stops VM execution if not handled with an error
	// handler and VM.Run returns the same error as wrapped.
	IndexSet(index, value Object) error
}

// Copier wraps the Copy method to create a deep copy of the object.
type Copier interface {
	Copy() Object
}

// IndexDeleter wraps the IndexDelete method to delete an index of an object.
type IndexDeleter interface {
	IndexDelete(Object) error
}

// LengthGetter wraps the Len method to get the number of elements of an object.
type LengthGetter interface {
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

// ObjectImpl is the basic Object implementation and it does not nothing, and
// helps to implement Object interface by embedding and overriding methods in
// custom implementations. String and TypeName must be implemented otherwise
// calling these methods causes panic.
type ObjectImpl struct{}

var _ Object = ObjectImpl{}

// TypeName implements Object interface.
func (ObjectImpl) TypeName() string {
	panic(ErrNotImplemented)
}

// String implements Object interface.
func (ObjectImpl) String() string {
	panic(ErrNotImplemented)
}

// Equal implements Object interface.
func (ObjectImpl) Equal(Object) bool { return false }

// IsFalsy implements Object interface.
func (ObjectImpl) IsFalsy() bool { return true }

// CanIterate implements Object interface.
func (ObjectImpl) CanIterate() bool { return false }

// Iterate implements Object interface.
func (ObjectImpl) Iterate() Iterator { return nil }

// IndexGet implements Object interface.
func (ObjectImpl) IndexGet(index Object) (value Object, err error) {
	return nil, ErrNotIndexable
}

// IndexSet implements Object interface.
func (ObjectImpl) IndexSet(index, value Object) error {
	return ErrNotIndexAssignable
}

// BinaryOp implements Object interface.
func (ObjectImpl) BinaryOp(_ token.Token, _ Object) (Object, error) {
	return nil, ErrInvalidOperator
}

// NilType represents the type of global Nil Object. One should use
// the NilType in type switches only.
type NilType struct {
	ObjectImpl
}

// TypeName implements Object interface.
func (o *NilType) TypeName() string {
	return "nil"
}

// String implements Object interface.
func (o *NilType) String() string {
	return "nil"
}

// Equal implements Object interface.
func (o *NilType) Equal(right Object) bool {
	return right == Nil
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
		Nil.TypeName(),
		right.TypeName())
}

// IndexGet implements Object interface.
func (*NilType) IndexGet(Object) (Object, error) {
	return Nil, nil
}

// IndexSet implements Object interface.
func (*NilType) IndexSet(_, _ Object) error {
	return ErrNotIndexAssignable
}

func Callable(o Object) (ok bool) {
	if _, ok = o.(CallerObject); ok {
		if cc, _ := o.(CanCallerObject); cc != nil {
			ok = cc.CanCall()
		}
	}
	return
}
