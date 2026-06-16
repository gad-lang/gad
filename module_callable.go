// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

// Callable wrappers used by the builtin stdlib module namespaces (module_strings.go,
// module_time.go). Package-local equivalents of the stdlib FuncPxxx helpers.

func funcPsRO(fn func(string) Object) CallableFunc {
	return func(c Call) (ret Object, err error) {
		if err := c.Args.CheckLen(1); err != nil {
			return Nil, err
		}

		s, ok := ToGoString(c.Args.Get(0))
		if !ok {
			return Nil, NewArgumentTypeError("1st", "str", c.Args.Get(0).Type().Name())
		}

		ret = fn(s)
		return
	}
}

func funcPssRO(fn func(string, string) Object) CallableFunc {
	return func(c Call) (ret Object, err error) {
		if err := c.Args.CheckLen(2); err != nil {
			return Nil, err
		}

		s1, ok := ToGoString(c.Args.Get(0))
		if !ok {
			return Nil, NewArgumentTypeError("1st", "str", c.Args.Get(0).Type().Name())
		}
		s2, ok := ToGoString(c.Args.Get(1))
		if !ok {
			return Nil, NewArgumentTypeError("2nd", "str", c.Args.Get(1).Type().Name())
		}

		ret = fn(s1, s2)
		return
	}
}

func funcPsrRO(fn func(string, rune) Object) CallableFunc {
	return func(c Call) (ret Object, err error) {
		if err := c.Args.CheckLen(2); err != nil {
			return Nil, err
		}

		s, ok := ToGoString(c.Args.Get(0))
		if !ok {
			return Nil, NewArgumentTypeError("1st", "str", c.Args.Get(0).Type().Name())
		}
		r, ok := ToGoRune(c.Args.Get(1))
		if !ok {
			return Nil, NewArgumentTypeError("2nd", "char", c.Args.Get(1).Type().Name())
		}

		ret = fn(s, r)
		return
	}
}

func funcPsiRO(fn func(string, int) Object) CallableFunc {
	return func(c Call) (ret Object, err error) {
		if err := c.Args.CheckLen(2); err != nil {
			return Nil, err
		}

		s, ok := ToGoString(c.Args.Get(0))
		if !ok {
			return Nil, NewArgumentTypeError("1st", "str", c.Args.Get(0).Type().Name())
		}
		i1, ok := ToGoInt(c.Args.Get(1))
		if !ok {
			return Nil, NewArgumentTypeError("2nd", "int", c.Args.Get(1).Type().Name())
		}

		ret = fn(s, i1)
		return
	}
}

func funcPAsRO(fn func(Array, string) Object) CallableFunc {
	return func(c Call) (ret Object, err error) {
		if err := c.Args.CheckLen(2); err != nil {
			return Nil, err
		}

		arr, ok := ToArray(c.Args.Get(0))
		if !ok {
			return Nil, NewArgumentTypeError("1st", "array", c.Args.Get(0).Type().Name())
		}
		s, ok := ToGoString(c.Args.Get(1))
		if !ok {
			return Nil, NewArgumentTypeError("2nd", "str", c.Args.Get(1).Type().Name())
		}

		ret = fn(arr, s)
		return
	}
}

func funcPAssRO(fn func(Array, string, string) Object) CallableFunc {
	return func(c Call) (ret Object, err error) {
		if err := c.Args.CheckLen(3); err != nil {
			return Nil, err
		}

		arr, ok := ToArray(c.Args.Get(0))
		if !ok {
			return Nil, NewArgumentTypeError("1st", "array", c.Args.Get(0).Type().Name())
		}
		s1, ok := ToGoString(c.Args.Get(1))
		if !ok {
			return Nil, NewArgumentTypeError("2nd", "str", c.Args.Get(1).Type().Name())
		}
		s2, ok := ToGoString(c.Args.Get(2))
		if !ok {
			return Nil, NewArgumentTypeError("3rd", "str", c.Args.Get(2).Type().Name())
		}

		ret = fn(arr, s1, s2)
		return
	}
}

func funcPRO(fn func() Object) CallableFunc {
	return func(c Call) (ret Object, err error) {
		if err := c.Args.CheckLen(0); err != nil {
			return Nil, err
		}

		ret = fn()
		return
	}
}

func funcPiRO(fn func(int) Object) CallableFunc {
	return func(c Call) (ret Object, err error) {
		if err := c.Args.CheckLen(1); err != nil {
			return Nil, err
		}

		i1, ok := ToGoInt(c.Args.Get(0))
		if !ok {
			return Nil, NewArgumentTypeError("1st", "int", c.Args.Get(0).Type().Name())
		}

		ret = fn(i1)
		return
	}
}

func funcPsROe(fn func(string) (Object, error)) CallableFunc {
	return func(c Call) (ret Object, err error) {
		if err := c.Args.CheckLen(1); err != nil {
			return Nil, err
		}

		s, ok := ToGoString(c.Args.Get(0))
		if !ok {
			return Nil, NewArgumentTypeError("1st", "str", c.Args.Get(0).Type().Name())
		}

		ret, err = fn(s)
		return
	}
}

func funcPi64i64RO(fn func(int64, int64) Object) CallableFunc {
	return func(c Call) (ret Object, err error) {
		if err := c.Args.CheckLen(2); err != nil {
			return Nil, err
		}

		i1, ok := ToGoInt64(c.Args.Get(0))
		if !ok {
			return Nil, NewArgumentTypeError("1st", "int", c.Args.Get(0).Type().Name())
		}
		i2, ok := ToGoInt64(c.Args.Get(1))
		if !ok {
			return Nil, NewArgumentTypeError("2nd", "int", c.Args.Get(1).Type().Name())
		}

		ret = fn(i1, i2)
		return
	}
}
