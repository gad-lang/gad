// Code generated by 'go generate'; DO NOT EDIT.

package stdlib

import (
	"github.com/gad-lang/gad"
)

// FuncPORO is a generated function to make gad.CallableFunc.
// Source: func(o gad.Object) (ret gad.Object)
func FuncPORO(fn func(gad.Object) gad.Object) gad.CallableFunc {
	return func(c gad.Call) (ret gad.Object, err error) {
		if err := c.Args.CheckLen(1); err != nil {
			return gad.Nil, err
		}

		o := c.Args.Get(0)

		ret = fn(o)
		return
	}
}

// FuncPiRO is a generated function to make gad.CallableFunc.
// Source: func(i1 int) (ret gad.Object)
func FuncPiRO(fn func(int) gad.Object) gad.CallableFunc {
	return func(c gad.Call) (ret gad.Object, err error) {
		if err := c.Args.CheckLen(1); err != nil {
			return gad.Nil, err
		}

		i1, ok := gad.ToGoInt(c.Args.Get(0))
		if !ok {
			return gad.Nil, gad.NewArgumentTypeError("1st", "int", c.Args.Get(0).Type().Name())
		}

		ret = fn(i1)
		return
	}
}

// FuncPi64RO is a generated function to make gad.CallableFunc.
// Source: func(i1 int64) (ret gad.Object)
func FuncPi64RO(fn func(int64) gad.Object) gad.CallableFunc {
	return func(c gad.Call) (ret gad.Object, err error) {
		if err := c.Args.CheckLen(1); err != nil {
			return gad.Nil, err
		}

		i1, ok := gad.ToGoInt64(c.Args.Get(0))
		if !ok {
			return gad.Nil, gad.NewArgumentTypeError("1st", "int", c.Args.Get(0).Type().Name())
		}

		ret = fn(i1)
		return
	}
}

// FuncPi64R is a generated function to make gad.CallableFunc.
// Source: func(i1 int64)
func FuncPi64R(fn func(int64)) gad.CallableFunc {
	return func(c gad.Call) (ret gad.Object, err error) {
		if err := c.Args.CheckLen(1); err != nil {
			return gad.Nil, err
		}

		i1, ok := gad.ToGoInt64(c.Args.Get(0))
		if !ok {
			return gad.Nil, gad.NewArgumentTypeError("1st", "int", c.Args.Get(0).Type().Name())
		}

		fn(i1)
		ret = gad.Nil
		return
	}
}

// FuncPsROe is a generated function to make gad.CallableFunc.
// Source: func(s string) (ret gad.Object, err error)
func FuncPsROe(fn func(string) (gad.Object, error)) gad.CallableFunc {
	return func(c gad.Call) (ret gad.Object, err error) {
		if err := c.Args.CheckLen(1); err != nil {
			return gad.Nil, err
		}

		s, ok := gad.ToGoString(c.Args.Get(0))
		if !ok {
			return gad.Nil, gad.NewArgumentTypeError("1st", "string", c.Args.Get(0).Type().Name())
		}

		ret, err = fn(s)
		return
	}
}

// FuncPsiRO is a generated function to make gad.CallableFunc.
// Source: func(s string, i1 int) (ret gad.Object)
func FuncPsiRO(fn func(string, int) gad.Object) gad.CallableFunc {
	return func(c gad.Call) (ret gad.Object, err error) {
		if err := c.Args.CheckLen(2); err != nil {
			return gad.Nil, err
		}

		s, ok := gad.ToGoString(c.Args.Get(0))
		if !ok {
			return gad.Nil, gad.NewArgumentTypeError("1st", "string", c.Args.Get(0).Type().Name())
		}
		i1, ok := gad.ToGoInt(c.Args.Get(1))
		if !ok {
			return gad.Nil, gad.NewArgumentTypeError("2nd", "int", c.Args.Get(1).Type().Name())
		}

		ret = fn(s, i1)
		return
	}
}

// FuncPRO is a generated function to make gad.CallableFunc.
// Source: func() (ret gad.Object)
func FuncPRO(fn func() gad.Object) gad.CallableFunc {
	return func(c gad.Call) (ret gad.Object, err error) {
		if err := c.Args.CheckLen(0); err != nil {
			return gad.Nil, err
		}

		ret = fn()
		return
	}
}

// FuncPi64i64RO is a generated function to make gad.CallableFunc.
// Source: func(i1 int64, i2 int64) (ret gad.Object)
func FuncPi64i64RO(fn func(int64, int64) gad.Object) gad.CallableFunc {
	return func(c gad.Call) (ret gad.Object, err error) {
		if err := c.Args.CheckLen(2); err != nil {
			return gad.Nil, err
		}

		i1, ok := gad.ToGoInt64(c.Args.Get(0))
		if !ok {
			return gad.Nil, gad.NewArgumentTypeError("1st", "int", c.Args.Get(0).Type().Name())
		}
		i2, ok := gad.ToGoInt64(c.Args.Get(1))
		if !ok {
			return gad.Nil, gad.NewArgumentTypeError("2nd", "int", c.Args.Get(1).Type().Name())
		}

		ret = fn(i1, i2)
		return
	}
}

// FuncPb2RO is a generated function to make gad.CallableFunc.
// Source: func(b []byte) (ret gad.Object)
func FuncPb2RO(fn func([]byte) gad.Object) gad.CallableFunc {
	return func(c gad.Call) (ret gad.Object, err error) {
		if err := c.Args.CheckLen(1); err != nil {
			return gad.Nil, err
		}

		b, ok := gad.ToGoByteSlice(c.Args.Get(0))
		if !ok {
			return gad.Nil, gad.NewArgumentTypeError("1st", "bytes", c.Args.Get(0).Type().Name())
		}

		ret = fn(b)
		return
	}
}

// FuncPOssRO is a generated function to make gad.CallableFunc.
// Source: func(o gad.Object, s1 string, s2 string) (ret gad.Object)
func FuncPOssRO(fn func(gad.Object, string, string) gad.Object) gad.CallableFunc {
	return func(c gad.Call) (ret gad.Object, err error) {
		if err := c.Args.CheckLen(3); err != nil {
			return gad.Nil, err
		}

		o := c.Args.Get(0)
		s1, ok := gad.ToGoString(c.Args.Get(1))
		if !ok {
			return gad.Nil, gad.NewArgumentTypeError("2nd", "string", c.Args.Get(1).Type().Name())
		}
		s2, ok := gad.ToGoString(c.Args.Get(2))
		if !ok {
			return gad.Nil, gad.NewArgumentTypeError("3rd", "string", c.Args.Get(2).Type().Name())
		}

		ret = fn(o, s1, s2)
		return
	}
}

// FuncPb2bRO is a generated function to make gad.CallableFunc.
// Source: func(p []byte, b bool) (ret gad.Object)
func FuncPb2bRO(fn func([]byte, bool) gad.Object) gad.CallableFunc {
	return func(c gad.Call) (ret gad.Object, err error) {
		if err := c.Args.CheckLen(2); err != nil {
			return gad.Nil, err
		}

		p, ok := gad.ToGoByteSlice(c.Args.Get(0))
		if !ok {
			return gad.Nil, gad.NewArgumentTypeError("1st", "bytes", c.Args.Get(0).Type().Name())
		}
		b, ok := gad.ToGoBool(c.Args.Get(1))
		if !ok {
			return gad.Nil, gad.NewArgumentTypeError("2nd", "bool", c.Args.Get(1).Type().Name())
		}

		ret = fn(p, b)
		return
	}
}

// FuncPb2ssRO is a generated function to make gad.CallableFunc.
// Source: func(p []byte, s1 string, s2 string) (ret gad.Object)
func FuncPb2ssRO(fn func([]byte, string, string) gad.Object) gad.CallableFunc {
	return func(c gad.Call) (ret gad.Object, err error) {
		if err := c.Args.CheckLen(3); err != nil {
			return gad.Nil, err
		}

		p, ok := gad.ToGoByteSlice(c.Args.Get(0))
		if !ok {
			return gad.Nil, gad.NewArgumentTypeError("1st", "bytes", c.Args.Get(0).Type().Name())
		}
		s1, ok := gad.ToGoString(c.Args.Get(1))
		if !ok {
			return gad.Nil, gad.NewArgumentTypeError("2nd", "string", c.Args.Get(1).Type().Name())
		}
		s2, ok := gad.ToGoString(c.Args.Get(2))
		if !ok {
			return gad.Nil, gad.NewArgumentTypeError("3rd", "string", c.Args.Get(2).Type().Name())
		}

		ret = fn(p, s1, s2)
		return
	}
}

// FuncPssRO is a generated function to make gad.CallableFunc.
// Source: func(s1 string, s2 string) (ret gad.Object)
func FuncPssRO(fn func(string, string) gad.Object) gad.CallableFunc {
	return func(c gad.Call) (ret gad.Object, err error) {
		if err := c.Args.CheckLen(2); err != nil {
			return gad.Nil, err
		}

		s1, ok := gad.ToGoString(c.Args.Get(0))
		if !ok {
			return gad.Nil, gad.NewArgumentTypeError("1st", "string", c.Args.Get(0).Type().Name())
		}
		s2, ok := gad.ToGoString(c.Args.Get(1))
		if !ok {
			return gad.Nil, gad.NewArgumentTypeError("2nd", "string", c.Args.Get(1).Type().Name())
		}

		ret = fn(s1, s2)
		return
	}
}

// FuncPsRO is a generated function to make gad.CallableFunc.
// Source: func(s string) (ret gad.Object)
func FuncPsRO(fn func(string) gad.Object) gad.CallableFunc {
	return func(c gad.Call) (ret gad.Object, err error) {
		if err := c.Args.CheckLen(1); err != nil {
			return gad.Nil, err
		}

		s, ok := gad.ToGoString(c.Args.Get(0))
		if !ok {
			return gad.Nil, gad.NewArgumentTypeError("1st", "string", c.Args.Get(0).Type().Name())
		}

		ret = fn(s)
		return
	}
}

// FuncPsrRO is a generated function to make gad.CallableFunc.
// Source: func(s string, r rune) (ret gad.Object)
func FuncPsrRO(fn func(string, rune) gad.Object) gad.CallableFunc {
	return func(c gad.Call) (ret gad.Object, err error) {
		if err := c.Args.CheckLen(2); err != nil {
			return gad.Nil, err
		}

		s, ok := gad.ToGoString(c.Args.Get(0))
		if !ok {
			return gad.Nil, gad.NewArgumentTypeError("1st", "string", c.Args.Get(0).Type().Name())
		}
		r, ok := gad.ToGoRune(c.Args.Get(1))
		if !ok {
			return gad.Nil, gad.NewArgumentTypeError("2nd", "char", c.Args.Get(1).Type().Name())
		}

		ret = fn(s, r)
		return
	}
}

// FuncPAsRO is a generated function to make gad.CallableFunc.
// Source: func(arr gad.Array, s string) (ret gad.Object)
func FuncPAsRO(fn func(gad.Array, string) gad.Object) gad.CallableFunc {
	return func(c gad.Call) (ret gad.Object, err error) {
		if err := c.Args.CheckLen(2); err != nil {
			return gad.Nil, err
		}

		arr, ok := gad.ToArray(c.Args.Get(0))
		if !ok {
			return gad.Nil, gad.NewArgumentTypeError("1st", "array", c.Args.Get(0).Type().Name())
		}
		s, ok := gad.ToGoString(c.Args.Get(1))
		if !ok {
			return gad.Nil, gad.NewArgumentTypeError("2nd", "string", c.Args.Get(1).Type().Name())
		}

		ret = fn(arr, s)
		return
	}
}

// FuncPOi64ROe is a generated function to make gad.CallableFunc.
// Source: func(o gad.Object, i int64) (ret gad.Object, err error)
func FuncPOi64ROe(fn func(gad.Object, int64) (gad.Object, error)) gad.CallableFunc {
	return func(c gad.Call) (ret gad.Object, err error) {
		if err := c.Args.CheckLen(2); err != nil {
			return gad.Nil, err
		}

		o := c.Args.Get(0)
		i, ok := gad.ToGoInt64(c.Args.Get(1))
		if !ok {
			return gad.Nil, gad.NewArgumentTypeError("2nd", "int", c.Args.Get(1).Type().Name())
		}

		ret, err = fn(o, i)
		return
	}
}
