// Code generated by 'go generate'; DO NOT EDIT.

package gad

import ()

// funcPOsRe is a generated function to make CallableFunc.
// Source: func(o Object, k string) (err error)
func funcPOsRe(fn func(Object, string) error) CallableFunc {
	return func(c Call) (ret Object, err error) {
		if err := c.Args.CheckLen(2); err != nil {
			return Nil, err
		}

		o := c.Args.Get(0)
		k, ok := ToGoString(c.Args.Get(1))
		if !ok {
			return Nil, NewArgumentTypeError("2nd", "string", c.Args.Get(1).Type().Name())
		}

		err = fn(o, k)
		ret = Nil
		return
	}
}

// funcPORO is a generated function to make CallableFunc.
// Source: func(o Object) (ret Object)
func funcPORO(fn func(Object) Object) CallableFunc {
	return func(c Call) (ret Object, err error) {
		if err := c.Args.CheckLen(1); err != nil {
			return Nil, err
		}

		o := c.Args.Get(0)

		ret = fn(o)
		return
	}
}

// funcPOiROe is a generated function to make CallableFunc.
// Source: func(o Object, n int) (ret Object, err error)
func funcPOiROe(fn func(Object, int) (Object, error)) CallableFunc {
	return func(c Call) (ret Object, err error) {
		if err := c.Args.CheckLen(2); err != nil {
			return Nil, err
		}

		o := c.Args.Get(0)
		n, ok := ToGoInt(c.Args.Get(1))
		if !ok {
			return Nil, NewArgumentTypeError("2nd", "int", c.Args.Get(1).Type().Name())
		}

		ret, err = fn(o, n)
		return
	}
}

// funcPiOROe is a generated function to make CallableFunc.
// Source: func(n int, o Object) (ret Object, err error)
func funcPiOROe(fn func(int, Object) (Object, error)) CallableFunc {
	return func(c Call) (ret Object, err error) {
		if err := c.Args.CheckLen(2); err != nil {
			return Nil, err
		}

		n, ok := ToGoInt(c.Args.Get(0))
		if !ok {
			return Nil, NewArgumentTypeError("1st", "int", c.Args.Get(0).Type().Name())
		}
		o := c.Args.Get(1)

		ret, err = fn(n, o)
		return
	}
}

// funcPOOROe is a generated function to make CallableFunc.
// Source: func(o Object, v Object) (ret Object, err error)
func funcPOOROe(fn func(Object, Object) (Object, error)) CallableFunc {
	return func(c Call) (ret Object, err error) {
		if err := c.Args.CheckLen(2); err != nil {
			return Nil, err
		}

		o := c.Args.Get(0)
		v := c.Args.Get(1)

		ret, err = fn(o, v)
		return
	}
}

// funcPOROe is a generated function to make CallableFunc.
// Source: func(o Object) (ret Object, err error)
func funcPOROe(fn func(Object) (Object, error)) CallableFunc {
	return func(c Call) (ret Object, err error) {
		if err := c.Args.CheckLen(1); err != nil {
			return Nil, err
		}

		o := c.Args.Get(0)

		ret, err = fn(o)
		return
	}
}

// funcPi64RO is a generated function to make CallableFunc.
// Source: func(v int64) (ret Object)
func funcPi64RO(fn func(int64) Object) CallableFunc {
	return func(c Call) (ret Object, err error) {
		if err := c.Args.CheckLen(1); err != nil {
			return Nil, err
		}

		v, ok := ToGoInt64(c.Args.Get(0))
		if !ok {
			return Nil, NewArgumentTypeError("1st", "int", c.Args.Get(0).Type().Name())
		}

		ret = fn(v)
		return
	}
}

// funcPu64RO is a generated function to make CallableFunc.
// Source: func(v uint64) (ret Object)
func funcPu64RO(fn func(uint64) Object) CallableFunc {
	return func(c Call) (ret Object, err error) {
		if err := c.Args.CheckLen(1); err != nil {
			return Nil, err
		}

		v, ok := ToGoUint64(c.Args.Get(0))
		if !ok {
			return Nil, NewArgumentTypeError("1st", "uint", c.Args.Get(0).Type().Name())
		}

		ret = fn(v)
		return
	}
}

// funcPf64RO is a generated function to make CallableFunc.
// Source: func(v float64) (ret Object)
func funcPf64RO(fn func(float64) Object) CallableFunc {
	return func(c Call) (ret Object, err error) {
		if err := c.Args.CheckLen(1); err != nil {
			return Nil, err
		}

		v, ok := ToGoFloat64(c.Args.Get(0))
		if !ok {
			return Nil, NewArgumentTypeError("1st", "float", c.Args.Get(0).Type().Name())
		}

		ret = fn(v)
		return
	}
}
