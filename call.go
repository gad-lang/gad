package gad

import "fmt"

type Args []Array

// GetDefault returns the nth argument. If n is greater than the number of arguments,
// it returns the nth variadic argument.
// If n is greater than the number of arguments and variadic arguments, return defaul.
func (c Args) GetDefault(n int, defaul Object) Object {
	var at int
	for _, arr := range c {
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
func (c Args) Get(n int) (v Object) {
	v = c.GetDefault(n, nil)
	if v == nil {
		panic(fmt.Sprintf("index out of range [%d] with length %d", n, c.Len()))
	}
	return
}

func (c Args) GetIJ(n int) (i, j int, ok bool) {
	var (
		at  int
		arr Array
	)
	for i, arr = range c {
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
func (c *Args) ShiftOk() (Object, bool) {
	if len(*c) == 0 {
		return Nil, false
	}

	for len((*c)[0]) == 0 {
		*c = (*c)[1:]
		if len(*c) == 0 {
			return Nil, false
		}
	}

	i, j, ok := c.GetIJ(0)
	if ok {
		v := (*c)[i][j]
		arr := (*c)[i][j+1:]
		if len(arr) == 0 {
			*c = (*c)[i+1:]
		} else {
			(*c)[i] = arr
		}
		return v, true
	}
	return Nil, false
}

// Shift returns the first argument and removes it from the arguments.
// If it cannot Shift, it returns nil.
func (c *Args) Shift() (v Object) {
	v, _ = c.ShiftOk()
	return v
}

// Len returns the number of arguments including variadic arguments.
func (c Args) Len() (l int) {
	for _, v := range c {
		l += len(v)
	}
	return l
}

// CheckLen checks the number of arguments and variadic arguments. If the number
// of arguments is not equal to n, it returns an error.
func (c Args) CheckLen(n int) error {
	if n != c.Len() {
		return ErrWrongNumArguments.NewError(
			fmt.Sprintf("want=%d got=%d", n, c.Len()),
		)
	}
	return nil
}

func (c Args) Values() (ret Array) {
	switch len(c) {
	case 0:
		return Array{}
	case 1:
		if c[0] == nil {
			return Array{}
		}
		return c[0]
	default:
		ret = Array{}
		for _, arr := range c {
			ret = append(ret, arr...)
		}
		return
	}
}

// Call is a struct to pass arguments to CallEx and CallName methods.
// It provides VM for various purposes.
//
// Call struct intentionally does not provide access to normal and variadic
// arguments directly. Using Len() and Get() methods is preferred. It is safe to
// create Call with a nil VM as long as VM is not required by the callee.
type Call struct {
	vm   *VM
	Args Args
}

// NewCall creates a new Call struct.
func NewCall(vm *VM, opts ...CallOpt) Call {
	c := Call{vm: vm}
	for _, opt := range opts {
		opt(&c)
	}
	return c
}

// VM returns the VM of the call.
func (c *Call) VM() *VM {
	return c.vm
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
