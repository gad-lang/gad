package gad

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gad-lang/gad/token"
)

type Args []Array

var (
	_ Object       = Args{}
	_ IndexGetter  = Args{}
	_ ValuesGetter = Args{}
)

func (o Args) Type() ObjectType {
	return TArgs
}

func (o *Args) Prepend(items ...Object) {
	*o = append(Args{items}, *o...)
}

func (o Args) ToString() string {
	var sb strings.Builder
	sb.WriteString("[")

	for _, v := range o {
		if len(v) > 0 {
			sb.WriteString(v.ToString())
			sb.WriteString(", ")
		}
	}

	return strings.TrimSuffix(sb.String(), ", ") + "]"
}

// DeepCopy implements DeepCopier interface.
func (o Args) DeepCopy() Object {
	cp := make(Args, len(o))
	for i, v := range o {
		cp[i] = v.DeepCopy().(Array)
	}
	return cp
}

// Copy implements Copier interface.
func (o Args) Copy() Object {
	cp := make(Args, len(o))
	for i := range o {
		cp[i] = o[i].Copy().(Array)
	}
	return cp
}

func (o Args) BinaryOp(tok token.Token, right Object) (Object, error) {
	switch tok {
	case token.Less, token.LessEq:
		if right == Nil {
			return False, nil
		}
		if other, ok := right.(Args); ok {
			if tok == token.LessEq {
				return Bool(o.Len() <= other.Len()), nil
			}
			return Bool(o.Len() < other.Len()), nil
		}
	case token.Greater, token.GreaterEq:
		if right == Nil {
			return True, nil
		}
		if other, ok := right.(Args); ok {
			if tok == token.GreaterEq {
				return Bool(o.Len() >= other.Len()), nil
			}
			return Bool(o.Len() > other.Len()), nil
		}
	}
	return nil, NewOperandTypeError(
		tok.String(),
		o.Type().Name(),
		right.Type().Name())
}

func (o Args) IsFalsy() bool {
	return o.Len() == 0
}

func (o Args) Equal(right Object) (ok bool) {
	switch t := right.(type) {
	case Args:
		if t.Len() == o.Len() {
			o.Walk(func(i int, arg Object) (continueLoop bool) {
				ok = arg.Equal(t.Get(i))
				return ok
			})
		}
	}
	return
}

func (o Args) Iterate(*VM) Iterator {
	return &ArgsIterator{o, o.Len(), 0}
}

func (o Args) CanIterate() bool {
	return true
}

// IndexGet implements Object interface.
func (o Args) IndexGet(_ *VM, index Object) (Object, error) {
	switch v := index.(type) {
	case Int:
		idx := int(v)
		if idx >= 0 && idx < o.Len() {
			return o.Get(idx), nil
		}
		return nil, ErrIndexOutOfBounds
	case Uint:
		idx := int(v)
		if idx >= 0 && idx < o.Len() {
			return o.Get(idx), nil
		}
		return nil, ErrIndexOutOfBounds
	case String:
		switch v {
		case "values":
			return o.Values(), nil
		case "array":
			arr := make(Array, len(o))
			for i := range o {
				arr[i] = o[i]
			}
			return arr, nil
		default:
			return nil, ErrInvalidIndex.NewError(string(v))
		}
	}
	return nil, NewIndexTypeError("int|uint|string", index.Type().Name())
}

// Walk iterates over all values and call callback function.
func (o Args) Walk(cb func(i int, arg Object) (continueLoop bool)) {
	var i int
	for _, arr := range o {
		for _, arg := range arr {
			if !cb(i, arg) {
				return
			}
			i++
		}
	}
}

// GetDefault returns the nth argument. If n is greater than the number of arguments,
// it returns the nth variadic argument.
// If n is greater than the number of arguments and variadic arguments, return defaul.
func (o Args) GetDefault(n int, defaul Object) Object {
	var at int
	for _, arr := range o {
		if len(arr) == 0 {
			continue
		}
		if at == n {
			return arr[0]
		}
		at += len(arr)
		if at > n {
			at -= len(arr)
			i := n - at
			return arr[i]
		}
	}
	return defaul
}

// Get returns the nth argument. If n is greater than the number of arguments,
// it returns the nth variadic argument.
// If n is greater than the number of arguments and variadic arguments, it
// panics!
func (o Args) Get(n int) (v Object) {
	v = o.GetDefault(n, nil)
	if v == nil {
		panic(fmt.Sprintf("index out of range [%d] with length %d", n, o.Len()))
	}
	return
}

func (o Args) GetIJ(n int) (i, j int, ok bool) {
	var (
		at  int
		arr Array
	)
	for i, arr = range o {
		if len(arr) == 0 {
			continue
		}
		if at == n {
			return i, 0, true
		}
		at += len(arr)
		if at > n {
			at -= len(arr)
			return i, n - at, true
		}
	}
	return 0, 0, false
}

// ShiftOk returns the first argument and removes it from the arguments.
// It updates the arguments and variadic arguments accordingly.
// If it cannot ShiftOk, it returns nil and false.
func (o *Args) ShiftOk() (Object, bool) {
	if len(*o) == 0 {
		return Nil, false
	}

	for len((*o)[0]) == 0 {
		*o = (*o)[1:]
		if len(*o) == 0 {
			return Nil, false
		}
	}

	i, j, ok := o.GetIJ(0)
	if ok {
		v := (*o)[i][j]
		arr := (*o)[i][j+1:]
		if len(arr) == 0 {
			*o = (*o)[i+1:]
		} else {
			(*o)[i] = arr
		}
		return v, true
	}
	return Nil, false
}

// Shift returns the first argument and removes it from the arguments.
// If it cannot Shift, it returns nil.
func (o *Args) Shift() (v Object) {
	v, _ = o.ShiftOk()
	return v
}

// Len returns the number of arguments including variadic arguments.
func (o Args) Len() (l int) {
	for _, v := range o {
		l += len(v)
	}
	return l
}

// CheckLen checks the number of arguments and variadic arguments. If the number
// of arguments is not equal to n, it returns an error.
func (o Args) CheckLen(n int) error {
	if n != o.Len() {
		return ErrWrongNumArguments.NewError(
			fmt.Sprintf("want=%d got=%d", n, o.Len()),
		)
	}
	return nil
}

// CheckMinLen checks the number of arguments and variadic arguments. If the number
// of arguments is less then to n, it returns an error.
func (o Args) CheckMinLen(n int) error {
	if o.Len() < n {
		return ErrWrongNumArguments.NewError(
			fmt.Sprintf("want>=%d got=%d", n, o.Len()),
		)
	}
	return nil
}

// CheckMaxLen checks the number of arguments and variadic arguments. If the number
// of arguments is greather then to n, it returns an error.
func (o Args) CheckMaxLen(n int) error {
	if o.Len() > n {
		return ErrWrongNumArguments.NewError(
			fmt.Sprintf("want<=%d got=%d", n, o.Len()),
		)
	}
	return nil
}

// CheckRangeLen checks the number of arguments and variadic arguments. If the number
// of arguments is less then to min or greather then to max, it returns an error.
func (o Args) CheckRangeLen(min, max int) error {
	if l := o.Len(); l < min || l > max {
		return ErrWrongNumArguments.NewError(
			fmt.Sprintf("want[%d...%d] got=%d", min, max, l),
		)
	}
	return nil
}

func (o Args) Values() (ret Array) {
	switch len(o) {
	case 0:
		return Array{}
	case 1:
		if o[0] == nil {
			return Array{}
		}
		return o[0]
	default:
		l := o.Len()
		ret = make(Array, l)
		for i := 0; i < l; i++ {
			ret[i] = o.Get(i)
		}
		return
	}
}

// ShiftArg shifts argument and set value to dst.
// If is empty, retun ok as false.
// If type check of arg is fails, returns ArgumentTypeError.
func (o Args) ShiftArg(shifts *int, dst *Arg) (ok bool, err error) {
	if dst.Value, ok = o.ShiftOk(); !ok {
		return
	}

	*shifts++

	if len(dst.AcceptTypes) == 0 {
		return
	}

	for _, t := range dst.AcceptTypes {
		if dst.Value.Type().Equal(t) {
			return
		}
	}

	var s = make([]string, len(dst.AcceptTypes))
	for i, acceptType := range dst.AcceptTypes {
		s[i] = acceptType.ToString()
	}

	return false, NewArgumentTypeError(
		strconv.Itoa(*shifts)+"st",
		strings.Join(s, "|"),
		dst.Value.Type().Name(),
	)
}

// Destructure shifts argument and set value to dst.
// If the number of arguments not equals to called args length, it returns an error.
// If type check of arg is fails, returns ArgumentTypeError.
func (o Args) Destructure(dst ...*Arg) (err error) {
	if err = o.CheckLen(len(dst)); err != nil {
		return
	}

args:
	for i, d := range dst {
		d.Value = o.Shift()

		if d.Accept != nil {
			if err = d.Accept(d.Value); err != nil {
				pos := strconv.Itoa(i) + "st"
				if d.Name != "" {
					pos += " (" + d.Name + ")"
				}
				return NewArgumentTypeError(
					pos,
					err.Error(),
					d.Value.Type().Name(),
				)
			}
		} else if len(d.AcceptTypes) > 0 {
			for _, t := range d.AcceptTypes {
				if t.Equal(d.Value.Type()) {
					continue args
				}
			}

			pos := strconv.Itoa(i+1) + "st"
			if d.Name != "" {
				pos += " (" + d.Name + ")"
			}

			var s = make([]string, len(d.AcceptTypes))
			for i, acceptType := range d.AcceptTypes {
				s[i] = acceptType.ToString()
			}
			return NewArgumentTypeError(
				pos,
				strings.Join(s, "|"),
				d.Value.Type().Name(),
			)
		}
	}
	return
}

// DestructureValue shifts argument and set value to dst.
// If type check of arg is fails, returns ArgumentTypeError.
func (o Args) DestructureValue(dst ...*Arg) (err error) {
args:
	for i, d := range dst {
		d.Value = o.Shift()

		if len(d.AcceptTypes) == 0 {
			continue
		}

		for _, t := range d.AcceptTypes {
			if t.Equal(d.Value.Type()) {
				continue args
			}
		}

		var s = make([]string, len(d.AcceptTypes))
		for i, acceptType := range d.AcceptTypes {
			s[i] = acceptType.ToString()
		}
		return NewArgumentTypeError(
			strconv.Itoa(i)+"st",
			strings.Join(s, "|"),
			d.Value.Type().Name(),
		)
	}
	return
}

// DestructureVar shifts argument and set value to dst, and returns left arguments.
// If the number of arguments is less then to called args length, it returns an error.
// If type check of arg is fails, returns ArgumentTypeError.
func (o Args) DestructureVar(dst ...*Arg) (other Array, err error) {
	if err = o.CheckMinLen(len(dst)); err != nil {
		return
	}

args:
	for i, d := range dst {
		d.Value = o.Shift()

		if len(d.AcceptTypes) == 0 {
			continue
		}

		for _, t := range d.AcceptTypes {
			if t.Equal(d.Value.Type()) {
				continue args
			}
		}

		var s = make([]string, len(d.AcceptTypes))
		for i, acceptType := range d.AcceptTypes {
			s[i] = acceptType.ToString()
		}
		return nil, NewArgumentTypeError(
			strconv.Itoa(i)+"st",
			strings.Join(s, "|"),
			d.Value.Type().Name(),
		)
	}
	other = o.Values()
	return
}

var _ Iterator = (*ArgsIterator)(nil)

type ArgsIterator struct {
	V   Args
	len int
	i   int
}

func (it *ArgsIterator) Length() int {
	return it.len
}

func (it *ArgsIterator) Next() bool {
	it.i++
	return it.i-1 < it.len
}

func (it *ArgsIterator) Key() Object {
	return Int(it.i)
}

func (it *ArgsIterator) Value() (Object, error) {
	i := it.i - 1
	if i > -1 && i < it.len {
		return it.V.Get(i), nil
	}
	return Nil, nil
}

// Call is a struct to pass arguments to CallEx and CallName methods.
// It provides VM for various purposes.
//
// Call struct intentionally does not provide access to normal and variadic
// arguments directly. Using Len() and Get() methods is preferred. It is safe to
// create Call with a nil VM as long as VM is not required by the callee.
type Call struct {
	VM        *VM
	Args      Args
	NamedArgs NamedArgs
}

// NewCall creates a new Call struct.
func NewCall(vm *VM, opts ...CallOpt) Call {
	c := Call{VM: vm}
	for _, opt := range opts {
		opt(&c)
	}
	return c
}

type CallOpt func(c *Call)

func WithArgs(args ...Object) func(c *Call) {
	return func(c *Call) {
		c.Args = Args{args, nil}
	}
}

func WithArgsV(args []Object, vargs ...Object) func(c *Call) {
	return func(c *Call) {
		c.Args = Args{args, vargs}
	}
}

func WithNamedArgs(na *NamedArgs) func(c *Call) {
	return func(c *Call) {
		c.NamedArgs = *na
	}
}

func MustCall(callee Object, args ...Object) (Object, error) {
	if !Callable(callee) {
		return nil, ErrNotCallable
	}
	return callee.(CallerObject).Call(NewCall(nil, WithArgs(args...)))
}

func MustCallVargs(callee Object, args []Object, vargs ...Object) (Object, error) {
	if !Callable(callee) {
		return nil, ErrNotCallable
	}
	return callee.(CallerObject).Call(NewCall(nil, WithArgsV(args, vargs...)))
}
