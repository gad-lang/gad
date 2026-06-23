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

// nil orders before every non-nil value and equals itself, so `nil < x` and
// `nil <= x` are true (false only against nil for `<`), while `nil > x` is false
// and `nil >= x` is true only against nil. These implement the comparison
// ObjectWith{Op}BinOperator interfaces.
func (o *NilType) BinOpLess(_ *VM, right Object) (Object, error) {
	if _, ok := right.(*NilType); ok {
		return False, nil
	}
	return True, nil
}

func (o *NilType) BinOpLessEq(_ *VM, _ Object) (Object, error) { return True, nil }

func (o *NilType) BinOpGreater(_ *VM, _ Object) (Object, error) { return False, nil }

func (o *NilType) BinOpGreaterEq(_ *VM, right Object) (Object, error) {
	if _, ok := right.(*NilType); ok {
		return True, nil
	}
	return False, nil
}

// binCmpAfterNil implements one comparison operator for a type that sorts after
// nil and is otherwise not comparable: `< nil` / `<= nil` are false,
// `> nil` / `>= nil` are true; any other operand is an unsupported-operand
// error. self is the left operand (for the error message).
func binCmpAfterNil(tok token.Token, self, right Object) (Object, error) {
	if right == Nil {
		switch tok {
		case token.Less, token.LessEq:
			return False, nil
		case token.Greater, token.GreaterEq:
			return True, nil
		}
	}
	return nil, NewOperandTypeError(tok.String(), self.Type().Name(), right.Type().Name())
}

func (o *NilType) Print(state *PrinterState) error {
	if state.IsRepr {
		return state.WriteString(ReprQuote("nil"))
	}
	return state.WriteString("nil")
}
