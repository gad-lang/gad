package gad

func ExprToTextOverride(name string, f func(vm *VM, w Writer, old func(w Writer, expr Object) (n Int, err error), expr Object) (n Int, err error)) CallerObject {
	return &Function{
		Name: name,
		Value: func(c Call) (_ Object, err error) {
			var n Int
			n, err = f(c.VM, c.Args.MustGet(0).(Writer), func(w Writer, expr Object) (_ Int, err error) {
				var n Object
				n, err = Val(c.Args.MustGet(1).(CallerObject).Call(Call{Args: Args{Array{w, expr}}}))
				return n.(Int), err
			}, c.Args.MustGet(2))
			return n, err
		},
	}
}
