package gad

import (
	"fmt"

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

	// Call is called from VM if CanCall() returns true. Check the number of
	// arguments provided and their types in the method. Returned error stops VM
	// execution if not handled with an error handler and VM.Run returns the
	// same error as wrapped.
	Call(args ...Object) (Object, error)

	// CanCall returns true if type can be called with Call() method.
	// VM returns an error if one tries to call a noncallable object.
	CanCall() bool

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

// ExCallerObject is an interface for objects that can be called with CallEx
// method. It is an extended version of the Call method that can be used to
// call an object with a Call struct. Objects implementing this interface is
// called with CallEx method instead of Call method.
// Note that CanCall() should return true for objects implementing this
// interface.
type ExCallerObject interface {
	Object
	CallEx(c Call) (Object, error)
}

// NameCallerObject is an interface for objects that can be called with CallName
// method to call a method of an object. Objects implementing this interface can
// reduce allocations by not creating a callable object for each method call.
type NameCallerObject interface {
	Object
	CallName(name string, c Call) (Object, error)
}

// Call is a struct to pass arguments to CallEx and CallName methods.
// It provides VM for various purposes.
//
// Call struct intentionally does not provide access to normal and variadic
// arguments directly. Using Len() and Get() methods is preferred. It is safe to
// create Call with a nil VM as long as VM is not required by the callee.
type Call struct {
	vm    *VM
	args  []Object
	vargs []Object
}

// NewCall creates a new Call struct with the given arguments.
func NewCall(vm *VM, args []Object, vargs ...Object) Call {
	return Call{
		vm:    vm,
		args:  args,
		vargs: vargs,
	}
}

// VM returns the VM of the call.
func (c *Call) VM() *VM {
	return c.vm
}

// Get returns the nth argument. If n is greater than the number of arguments,
// it returns the nth variadic argument.
// If n is greater than the number of arguments and variadic arguments, it
// panics!
func (c *Call) Get(n int) Object {
	if n < len(c.args) {
		return c.args[n]
	}
	return c.vargs[n-len(c.args)]
}

// Len returns the number of arguments including variadic arguments.
func (c *Call) Len() int {
	return len(c.args) + len(c.vargs)
}

// CheckLen checks the number of arguments and variadic arguments. If the number
// of arguments is not equal to n, it returns an error.
func (c *Call) CheckLen(n int) error {
	if n != c.Len() {
		return ErrWrongNumArguments.NewError(
			fmt.Sprintf("want=%d got=%d", n, c.Len()),
		)
	}
	return nil
}

// shift returns the first argument and removes it from the arguments.
// It updates the arguments and variadic arguments accordingly.
// If it cannot shift, it returns nil and false.
func (c *Call) shift() (Object, bool) {
	if len(c.args) == 0 {
		if len(c.vargs) == 0 {
			return nil, false
		}
		v := c.vargs[0]
		c.vargs = c.vargs[1:]
		return v, true
	}
	v := c.args[0]
	c.args = c.args[1:]
	return v, true
}

func (c *Call) callArgs() []Object {
	if len(c.args) == 0 {
		return c.vargs
	}
	args := make([]Object, 0, c.Len())
	args = append(args, c.args...)
	args = append(args, c.vargs...)
	return args
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

// CanCall implements Object interface.
func (ObjectImpl) CanCall() bool { return false }

// Call implements Object interface.
func (ObjectImpl) Call(_ ...Object) (Object, error) {
	return nil, ErrNotCallable
}

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

// Call implements Object interface.
func (*NilType) Call(_ ...Object) (Object, error) {
	return nil, ErrNotCallable
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
