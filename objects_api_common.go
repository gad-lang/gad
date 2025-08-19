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

	// Yes represents a flag on.
	Yes = Flag(true)

	// Yes represents a flag off.
	No = Flag(false)
)

var (
	// Nil represents nil value.
	Nil Object = &NilType{}
)

// ObjectImpl is the basic Object implementation and it does not nothing, and
// helps to implement Object interface by embedding and overriding methods in
// custom implementations. Str and OpDotName must be implemented otherwise
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
	return right == nil || right == Nil
}

func (o *NilType) IsNil() bool {
	return true
}

// BinaryOp implements Object interface.
func (o *NilType) BinaryOp(_ *VM, tok token.Token, right Object) (Object, error) {
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
