// Code generated by 'go generate'; DO NOT EDIT.

package time

import (
	"strconv"

	"github.com/ozanh/ugo"
)

// funcPTRO is a generated function to make ugo.CallableFunc.
// Source: funcPTRO(t *Time) (ret ugo.Object)
func funcPTRO(fn func(*Time) ugo.Object) ugo.CallableFunc {
	return func(args ...ugo.Object) (ret ugo.Object, err error) {
		if len(args) != 1 {
			return ugo.Undefined, ugo.ErrWrongNumArguments.NewError("want=1 got=" + strconv.Itoa(len(args)))
		}

		t, ok := ToTime(args[0])
		if !ok {
			return ugo.Undefined, ugo.NewArgumentTypeError("1st", "time", args[0].TypeName())
		}

		ret = fn(t)
		return
	}
}

// funcPTi64RO is a generated function to make ugo.CallableFunc.
// Source: funcPTi64RO(t *Time, d int64) (ret ugo.Object)
func funcPTi64RO(fn func(*Time, int64) ugo.Object) ugo.CallableFunc {
	return func(args ...ugo.Object) (ret ugo.Object, err error) {
		if len(args) != 2 {
			return ugo.Undefined, ugo.ErrWrongNumArguments.NewError("want=2 got=" + strconv.Itoa(len(args)))
		}

		t, ok := ToTime(args[0])
		if !ok {
			return ugo.Undefined, ugo.NewArgumentTypeError("1st", "time", args[0].TypeName())
		}
		d, ok := ugo.ToGoInt64(args[1])
		if !ok {
			return ugo.Undefined, ugo.NewArgumentTypeError("2nd", "int", args[1].TypeName())
		}

		ret = fn(t, d)
		return
	}
}

// funcPTTRO is a generated function to make ugo.CallableFunc.
// Source: funcPTTRO(t1 *Time, t2 *Time) (ret ugo.Object)
func funcPTTRO(fn func(*Time, *Time) ugo.Object) ugo.CallableFunc {
	return func(args ...ugo.Object) (ret ugo.Object, err error) {
		if len(args) != 2 {
			return ugo.Undefined, ugo.ErrWrongNumArguments.NewError("want=2 got=" + strconv.Itoa(len(args)))
		}

		t1, ok := ToTime(args[0])
		if !ok {
			return ugo.Undefined, ugo.NewArgumentTypeError("1st", "time", args[0].TypeName())
		}
		t2, ok := ToTime(args[1])
		if !ok {
			return ugo.Undefined, ugo.NewArgumentTypeError("2nd", "time", args[1].TypeName())
		}

		ret = fn(t1, t2)
		return
	}
}

// funcPTiiiRO is a generated function to make ugo.CallableFunc.
// Source: funcPTiiiRO(t *Time, i1 int, i2 int, i3 int) (ret ugo.Object)
func funcPTiiiRO(fn func(*Time, int, int, int) ugo.Object) ugo.CallableFunc {
	return func(args ...ugo.Object) (ret ugo.Object, err error) {
		if len(args) != 4 {
			return ugo.Undefined, ugo.ErrWrongNumArguments.NewError("want=4 got=" + strconv.Itoa(len(args)))
		}

		t, ok := ToTime(args[0])
		if !ok {
			return ugo.Undefined, ugo.NewArgumentTypeError("1st", "time", args[0].TypeName())
		}
		i1, ok := ugo.ToGoInt(args[1])
		if !ok {
			return ugo.Undefined, ugo.NewArgumentTypeError("2nd", "int", args[1].TypeName())
		}
		i2, ok := ugo.ToGoInt(args[2])
		if !ok {
			return ugo.Undefined, ugo.NewArgumentTypeError("3rd", "int", args[2].TypeName())
		}
		i3, ok := ugo.ToGoInt(args[3])
		if !ok {
			return ugo.Undefined, ugo.NewArgumentTypeError("4th", "int", args[3].TypeName())
		}

		ret = fn(t, i1, i2, i3)
		return
	}
}

// funcPTsRO is a generated function to make ugo.CallableFunc.
// Source: funcPTsRO(t *Time, s string) (ret ugo.Object)
func funcPTsRO(fn func(*Time, string) ugo.Object) ugo.CallableFunc {
	return func(args ...ugo.Object) (ret ugo.Object, err error) {
		if len(args) != 2 {
			return ugo.Undefined, ugo.ErrWrongNumArguments.NewError("want=2 got=" + strconv.Itoa(len(args)))
		}

		t, ok := ToTime(args[0])
		if !ok {
			return ugo.Undefined, ugo.NewArgumentTypeError("1st", "time", args[0].TypeName())
		}
		s, ok := ugo.ToGoString(args[1])
		if !ok {
			return ugo.Undefined, ugo.NewArgumentTypeError("2nd", "string", args[1].TypeName())
		}

		ret = fn(t, s)
		return
	}
}

// funcPTb2sRO is a generated function to make ugo.CallableFunc.
// Source: funcPTb2sRO(t *Time, b []byte, s string) (ret ugo.Object)
func funcPTb2sRO(fn func(*Time, []byte, string) ugo.Object) ugo.CallableFunc {
	return func(args ...ugo.Object) (ret ugo.Object, err error) {
		if len(args) != 3 {
			return ugo.Undefined, ugo.ErrWrongNumArguments.NewError("want=3 got=" + strconv.Itoa(len(args)))
		}

		t, ok := ToTime(args[0])
		if !ok {
			return ugo.Undefined, ugo.NewArgumentTypeError("1st", "time", args[0].TypeName())
		}
		b, ok := ugo.ToGoByteSlice(args[1])
		if !ok {
			return ugo.Undefined, ugo.NewArgumentTypeError("2nd", "bytes", args[1].TypeName())
		}
		s, ok := ugo.ToGoString(args[2])
		if !ok {
			return ugo.Undefined, ugo.NewArgumentTypeError("3rd", "string", args[2].TypeName())
		}

		ret = fn(t, b, s)
		return
	}
}

// funcPTLRO is a generated function to make ugo.CallableFunc.
// Source: funcPTLRO(t *Time, loc *Location) (ret ugo.Object)
func funcPTLRO(fn func(*Time, *Location) ugo.Object) ugo.CallableFunc {
	return func(args ...ugo.Object) (ret ugo.Object, err error) {
		if len(args) != 2 {
			return ugo.Undefined, ugo.ErrWrongNumArguments.NewError("want=2 got=" + strconv.Itoa(len(args)))
		}

		t, ok := ToTime(args[0])
		if !ok {
			return ugo.Undefined, ugo.NewArgumentTypeError("1st", "time", args[0].TypeName())
		}
		loc, ok := ToLocation(args[1])
		if !ok {
			return ugo.Undefined, ugo.NewArgumentTypeError("2nd", "location", args[1].TypeName())
		}

		ret = fn(t, loc)
		return
	}
}
