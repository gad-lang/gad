package gad

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gad-lang/gad/token"
	"github.com/shopspring/decimal"
)

func BuiltinMakeArrayFunc(n int, arg Object) (Object, error) {
	if n <= 0 {
		return arg, nil
	}

	arr, ok := arg.(Array)
	if !ok {
		ret := make(Array, n)
		for i := 1; i < n; i++ {
			ret[i] = Nil
		}
		ret[0] = arg
		return ret, nil
	}

	length := len(arr)
	if n <= length {
		return arr[:n], nil
	}

	ret := make(Array, n)
	x := copy(ret, arr)
	for i := x; i < n; i++ {
		ret[i] = Nil
	}
	return ret, nil
}

func BuiltinAppendFunc(c Call) (Object, error) {
	target, ok := c.Args.ShiftOk()
	if !ok {
		return Nil, ErrWrongNumArguments.NewError("want>=1 got=0")
	}

	switch obj := target.(type) {
	case Array:
		for _, arg := range c.Args {
			arg.AppendToArray(&obj)
		}
		return obj, nil
	case Bytes:
		n := 0
		for _, args := range c.Args {
			for _, v := range args {
				n++
				switch vv := v.(type) {
				case Int:
					obj = append(obj, byte(vv))
				case Uint:
					obj = append(obj, byte(vv))
				case Char:
					obj = append(obj, byte(vv))
				default:
					return Nil, NewArgumentTypeError(
						strconv.Itoa(n),
						"int|uint|char",
						vv.Type().Name(),
					)
				}
			}
		}
		return obj, nil
	case *NilType:
		ret := make(Array, 0, c.Args.Len())
		for _, arg := range c.Args {
			arg.AppendToArray(&ret)
		}
		return ret, nil
	case KeyValueArray:
		var (
			err        error
			i          = 1
			arg, valid = c.Args.ShiftOk()
		)

		for valid {
			if obj, err = obj.AppendObject(arg); err != nil {
				err = NewArgumentTypeError(
					strconv.Itoa(i)+"st",
					err.Error(),
					arg.Type().Name(),
				)
				return nil, err
			}
			arg, valid = c.Args.ShiftOk()
			i++
		}
		return obj, nil
	case Appender:
		return obj.Append(c.Args.Values()...)
	default:
		return Nil, NewArgumentTypeError(
			"1st",
			"array",
			obj.Type().Name(),
		)
	}
}

func BuiltinDeleteFunc(c Call) (_ Object, err error) {
	var (
		target = &Arg{
			Name: "target",
			Accept: func(v Object) string {
				if _, ok := v.(IndexDeleter); !ok {
					return ErrNotIndexDeletable.Name
				}
				return ""
			},
		}
		key = &Arg{}
	)
	if err = c.Args.Destructure(target, key); err != nil {
		return
	}
	return Nil, target.Value.(IndexDeleter).IndexDelete(c.VM, key.Value)
}

func BuiltinCopyFunc(arg Object) Object {
	if v, ok := arg.(Copier); ok {
		return v.Copy()
	}
	return arg
}

func BuiltinDeepCopyFunc(arg Object) Object {
	if v, ok := arg.(DeepCopier); ok {
		return v.DeepCopy()
	} else if v, ok := arg.(Copier); ok {
		return v.Copy()
	}
	return arg
}

func BuiltinRepeatFunc(arg Object, count int) (ret Object, err error) {
	if count < 0 {
		return nil, NewArgumentTypeError(
			"2nd",
			"non-negative integer",
			"negative integer",
		)
	}

	switch v := arg.(type) {
	case Array:
		out := make(Array, 0, len(v)*count)
		for i := 0; i < count; i++ {
			out = append(out, v...)
		}
		ret = out
	case String:
		ret = String(strings.Repeat(string(v), count))
	case Bytes:
		ret = Bytes(bytes.Repeat(v, count))
	default:
		err = NewArgumentTypeError(
			"1st",
			"array|string|bytes",
			arg.Type().Name(),
		)
	}
	return
}

func BuiltinContainsFunc(arg0, arg1 Object) (Object, error) {
	var ok bool
	switch obj := arg0.(type) {
	case Dict:
		_, ok = obj[arg1.ToString()]
	case *SyncMap:
		_, ok = obj.Get(arg1.ToString())
	case Array:
		for _, item := range obj {
			if item.Equal(arg1) {
				ok = true
				break
			}
		}
	case *NamedArgs:
		ok = obj.Contains(arg1.ToString())
	case String:
		ok = strings.Contains(string(obj), arg1.ToString())
	case Bytes:
		switch v := arg1.(type) {
		case Int:
			ok = bytes.Contains(obj, []byte{byte(v)})
		case Uint:
			ok = bytes.Contains(obj, []byte{byte(v)})
		case Char:
			ok = bytes.Contains(obj, []byte{byte(v)})
		case String:
			ok = bytes.Contains(obj, []byte(v))
		case Bytes:
			ok = bytes.Contains(obj, v)
		default:
			return Nil, NewArgumentTypeError(
				"2nd",
				"int|uint|string|char|bytes",
				arg1.Type().Name(),
			)
		}
	case *NilType:
	default:
		return Nil, NewArgumentTypeError(
			"1st",
			"map|array|string|bytes|namedArgs",
			arg0.Type().Name(),
		)
	}
	return Bool(ok), nil
}

func BuiltinLenFunc(arg Object) Object {
	var n int
	if v, ok := arg.(LengthGetter); ok {
		n = v.Len()
	}
	return Int(n)
}

func BuiltinCapFunc(arg Object) Object {
	var n int
	switch v := arg.(type) {
	case Array:
		n = cap(v)
	case Bytes:
		n = cap(v)
	}
	return Int(n)
}

func BuiltinSortFunc(arg Object) (ret Object, err error) {
	switch obj := arg.(type) {
	case Sorter:
		ret, err = obj.Sort()
	case String:
		s := []rune(obj)
		sort.Slice(s, func(i, j int) bool {
			return s[i] < s[j]
		})
		ret = String(s)
	case Bytes:
		sort.Slice(obj, func(i, j int) bool {
			return obj[i] < obj[j]
		})
		ret = arg
	case *NilType:
		ret = Nil
	default:
		ret = Nil
		err = NewArgumentTypeError(
			"1st",
			"array|string|bytes",
			arg.Type().Name(),
		)
	}
	return
}

func BuiltinSortReverseFunc(arg Object) (Object, error) {
	switch obj := arg.(type) {
	case ReverseSorter:
		return obj.SortReverse()
	case String:
		s := []rune(obj)
		sort.Slice(s, func(i, j int) bool {
			return s[j] < s[i]
		})
		return String(s), nil
	case Bytes:
		sort.Slice(obj, func(i, j int) bool {
			return obj[j] < obj[i]
		})
		return obj, nil
	case *NilType:
		return Nil, nil
	}

	return Nil, NewArgumentTypeError(
		"1st",
		"array|string|bytes",
		arg.Type().Name(),
	)
}

func BuiltinFilterFunc(c Call) (_ Object, err error) {
	var (
		iterabler = &Arg{
			Name: "iterable",
			Accept: func(v Object) string {
				if !Filterable(v) && !Iterable(v) {
					return "filterable|iterable"
				}
				return ""
			},
		}

		callback = &Arg{
			Name: "callback",
			Accept: func(v Object) string {
				if !Callable(v) {
					return "callable"
				}
				return ""
			},
		}
	)

	if err = c.Args.Destructure(iterabler, callback); err != nil {
		return
	}

	var (
		args   = Array{Nil, Nil, iterabler.Value}
		caller VMCaller
	)

	if caller, err = NewInvoker(c.VM, callback.Value).Caller(Args{args}, &c.NamedArgs); err != nil {
		return
	}

	if Filterable(iterabler.Value) {
		return iterabler.Value.(Filterabler).Filter(c.VM, args, caller)
	}

	var (
		it  = iterabler.Value.(Iterabler).Iterate(c.VM)
		fe  = NewForEach(it, args, 0, caller)
		ret Array
	)

	if itl, _ := it.(LengthIterator); itl != nil {
		ret = make(Array, itl.Length())
		var (
			i  int
			ok Object
		)
		for fe.Next() {
			if ok, err = fe.Call(); err != nil {
				return
			} else if !ok.IsFalsy() {
				ret[i] = fe.Value
			}
			i++
		}
	} else {
		var ok Object
		for fe.Next() {
			if ok, err = fe.Call(); err != nil {
				return
			} else if !ok.IsFalsy() {
				ret = append(ret, fe.Value)
			}
		}
	}

	return ret, nil
}

func BuiltinMapFunc(c Call) (_ Object, err error) {
	var (
		iterabler = &Arg{
			Name: "iterable",
			Accept: func(v Object) string {
				if !Mapable(v) && !Iterable(v) {
					return "mapable|iterable"
				}
				return ""
			},
		}

		callback = &Arg{
			Name: "callback",
			Accept: func(v Object) string {
				if !Callable(v) {
					return "callable"
				}
				return ""
			},
		}
	)

	if err = c.Args.Destructure(iterabler, callback); err != nil {
		return
	}

	var (
		args   = Array{Nil, Nil}
		caller VMCaller
	)

	if caller, err = NewInvoker(c.VM, callback.Value).Caller(Args{args}, &c.NamedArgs); err != nil {
		return
	}

	if Mapable(iterabler.Value) {
		return iterabler.Value.(Mapabler).Map(c.VM, args, caller)
	}

	var (
		it  = iterabler.Value.(Iterabler).Iterate(c.VM)
		fe  = NewForEach(it, args, 0, caller)
		ret Array
	)

	if itl, _ := it.(LengthIterator); itl != nil {
		ret = make(Array, itl.Length())
		var i int
		for fe.Next() {
			if ret[i], err = fe.Call(); err != nil {
				return
			}
			i++
		}
	} else {
		var val Object
		for fe.Next() {
			if val, err = fe.Call(); err != nil {
				return
			}
			ret = append(ret, val)
		}
	}

	return ret, nil
}

func BuiltinReduceFunc(c Call) (_ Object, err error) {
	var (
		iterabler = &Arg{
			Name: "iterable",
			Accept: func(v Object) string {
				if !Reducable(v) && !Iterable(v) {
					return "reducable|iterable"
				}
				return ""
			},
		}

		callback = &Arg{
			Name: "callback",
			Accept: func(v Object) string {
				if !Callable(v) {
					return "callable"
				}
				return ""
			},
		}

		val = Nil
	)

	if c.Args.Len() == 3 {
		initialArg := &Arg{}
		if err = c.Args.Destructure(iterabler, callback, initialArg); err != nil {
			return
		}
		val = initialArg.Value
	} else {
		if err = c.Args.Destructure(iterabler, callback); err != nil {
			return
		}
	}

	var (
		args   = Array{Nil, Nil, Nil}
		caller VMCaller
	)

	if caller, err = NewInvoker(c.VM, callback.Value).Caller(Args{args}, &c.NamedArgs); err != nil {
		return
	}

	if Reducable(iterabler.Value) {
		return iterabler.Value.(Reducer).Reduce(c.VM, val, args, caller)
	}

	var (
		it = iterabler.Value.(Iterabler).Iterate(c.VM)
		fe = NewForEach(it, args, 1, caller)
	)

	if itl, _ := it.(LengthIterator); itl != nil {
		if val == Nil {
			if fe.Next() {
				args[0] = fe.Value
				if val, err = fe.Call(); err != nil {
					return
				}
			}
		}

		args[0] = val

		for fe.Next() {
			if val, err = fe.Call(); err != nil {
				return
			}
			args[0] = val
		}
	} else {
		if val == Nil {
			if fe.Next() {
				val = fe.Value
				args[0] = val
				if val, err = fe.Call(); err != nil {
					return
				}
			}
		}

		args[0] = val
		for fe.Next() {
			if val, err = fe.Call(); err != nil {
				return
			}
			args[0] = val
		}
	}

	return val, nil
}

func BuiltinForEachFunc(c Call) (_ Object, err error) {
	var (
		iterabler = &Arg{
			Name: "iterable",
			Accept: func(v Object) string {
				if !Iterable(v) {
					return "iterable"
				}
				return ""
			},
		}

		callback = &Arg{
			Name: "callback",
			Accept: func(v Object) string {
				if !Callable(v) {
					return "callable"
				}
				return ""
			},
		}
	)

	if err = c.Args.Destructure(iterabler, callback); err != nil {
		return
	}

	var (
		args   = Array{Nil, Nil}
		caller VMCaller
	)

	if caller, err = NewInvoker(c.VM, callback.Value).Caller(Args{args}, &c.NamedArgs); err != nil {
		return
	}

	var (
		it = iterabler.Value.(Iterabler).Iterate(c.VM)
		fe = NewForEach(it, args, 0, caller)
	)

	var val Object
	for fe.Next() {
		if val, err = fe.Call(); err != nil {
			return
		}
		if val != Nil && val.IsFalsy() {
			break
		}
	}

	return iterabler.Value, nil
}

func BuiltinErrorFunc(arg Object) Object {
	return &Error{Name: "error", Message: arg.ToString()}
}

func BuiltinTypeNameFunc(arg Object) Object { return String(arg.Type().Name()) }

func BuiltinBoolFunc(arg Object) Object { return Bool(!arg.IsFalsy()) }

func BuiltinIntFunc(v int64) Object { return Int(v) }

func BuiltinUintFunc(v uint64) Object { return Uint(v) }

func BuiltinFloatFunc(v float64) Object { return Float(v) }

func BuiltinDecimalFunc(v Object) (Object, error) {
	return Decimal(decimal.Zero).BinaryOp(token.Add, v)
}

func BuiltinCharFunc(arg Object) (Object, error) {
	v, ok := ToChar(arg)
	if ok && v != utf8.RuneError {
		return v, nil
	}
	if v == utf8.RuneError || arg == Nil {
		return Nil, nil
	}
	return Nil, NewArgumentTypeError(
		"1st",
		"numeric|string|bool",
		arg.Type().Name(),
	)
}

func BuiltinTextFunc(c Call) (ret Object, err error) {
	if err := c.Args.CheckLen(1); err != nil {
		return Nil, err
	}

	o := c.Args.Get(0)

	if ts, _ := o.(ToStringer); ts != nil {
		var s String
		s, err = ts.Stringer(c)
		ret = Text(s)
	} else {
		ret = Text(o.ToString())
	}
	return
}

func BuiltinStringFunc(c Call) (ret Object, err error) {
	if err := c.Args.CheckLen(1); err != nil {
		return Nil, err
	}

	o := c.Args.Get(0)

	if ts, _ := o.(ToStringer); ts != nil {
		return ts.Stringer(c)
	}

	ret = String(o.ToString())
	return
}

func BuiltinBytesFunc(c Call) (Object, error) {
	size := c.Args.Len()

	switch size {
	case 0:
		return Bytes{}, nil
	case 1:
		if v, ok := ToBytes(c.Args.Get(0)); ok {
			return v, nil
		}
	}

	out := make(Bytes, 0, size)
	for _, args := range c.Args {
		for i, obj := range args {
			switch v := obj.(type) {
			case Int:
				out = append(out, byte(v))
			case Uint:
				out = append(out, byte(v))
			case Char:
				out = append(out, byte(v))
			default:
				return Nil, NewArgumentTypeError(
					strconv.Itoa(i+1),
					"int|uint|char",
					args[i].Type().Name(),
				)
			}
		}
	}
	return out, nil
}

func BuiltinCharsFunc(arg Object) (ret Object, err error) {
	switch obj := arg.(type) {
	case String:
		s := string(obj)
		ret = make(Array, 0, utf8.RuneCountInString(s))
		sz := len(obj)
		i := 0

		for i < sz {
			r, w := utf8.DecodeRuneInString(s[i:])
			if r == utf8.RuneError {
				return Nil, nil
			}
			ret = append(ret.(Array), Char(r))
			i += w
		}
	case Bytes:
		ret = make(Array, 0, utf8.RuneCount(obj))
		sz := len(obj)
		i := 0

		for i < sz {
			r, w := utf8.DecodeRune(obj[i:])
			if r == utf8.RuneError {
				return Nil, nil
			}
			ret = append(ret.(Array), Char(r))
			i += w
		}
	default:
		ret = Nil
		err = NewArgumentTypeError(
			"1st",
			"string|bytes",
			arg.Type().Name(),
		)
	}
	return
}

func BuiltinPrintfFunc(c Call) (_ Object, err error) {
	var (
		out = &NamedArgVar{Value: c.VM.StdOut, AcceptTypes: []ObjectType{TWriter}}
		n   int
	)

	if err = c.NamedArgs.Get(out); err != nil {
		return
	}

	w := out.Value.(Writer)

	switch size := c.Args.Len(); size {
	case 0:
		err = ErrWrongNumArguments.NewError("want>=1 got=0")
	case 1:
		n, err = fmt.Fprint(w, c.Args.Get(0).ToString())
	default:
		format, _ := c.Args.ShiftOk()
		vargs := make([]any, 0, size-1)
		for i := 0; i < size-1; i++ {
			vargs = append(vargs, c.Args.Get(i))
		}
		n, err = fmt.Fprintf(w, format.ToString(), vargs...)
	}
	return Int(n), err
}

func BuiltinWriteFunc(c Call) (ret Object, err error) {
	var (
		w     io.Writer = c.VM.StdOut
		total Int
		n     int
		write = func(w io.Writer, obj Object) (total Int, err error) {
			var n int
			switch t := obj.(type) {
			case Text:
				n, err = w.Write([]byte(t))
			case String:
				n, err = w.Write([]byte(t))
			case Bytes:
				n, err = w.Write(t)
			case BytesConverter:
				var b Bytes
				if b, err = t.ToBytes(); err == nil {
					n, err = w.Write(b)
				}
			case io.WriterTo:
				var i64 int64
				i64, err = t.WriteTo(w)
				total = Int(i64)
			default:
				n, err = fmt.Fprint(w, t)
			}
			total += Int(n)
			return
		}
		convert CallerObject
	)

	if err = c.Args.CheckMinLen(1); err != nil {
		return
	}

	arg := c.Args.Get(0)
	if w2, ok := arg.(Writer); ok {
		w = w2
		c.Args.Shift()
	}

	if convertValue := c.NamedArgs.GetValueOrNil("convert"); convertValue != nil {
		convert = convertValue.(CallerObject)
	}

	if convert == nil {
		c.Args.Walk(func(i int, arg Object) (continueLoop bool) {
			switch t := arg.(type) {
			case Text:
				n, err = w.Write([]byte(t))
				total += Int(n)
			default:
				total, err = write(w, arg)
			}
			return err == nil
		})
	} else {
		var (
			convertCallArgs = Array{
				NewWriter(w),
				&Function{
					Value: func(c Call) (_ Object, err error) {
						var i Int
						i, err = write(c.Args.MustGet(0).(Writer), c.Args.MustGet(1))
						return i, err
					},
				},
				nil,
			}
			caller VMCaller
		)
		if caller, err = NewInvoker(c.VM, convert).Caller(Args{convertCallArgs}, nil); err != nil {
			return
		}

		c.Args.Walk(func(i int, arg Object) (continueLoop bool) {
			switch t := arg.(type) {
			case Text:
				n, err = w.Write([]byte(t))
				total += Int(n)
			default:
				var iO Object
				convertCallArgs[2] = t
				iO, err = caller.Call()
				if i, ok := iO.(Int); ok {
					total += i
				}
			}
			return err == nil
		})
	}

	return total, err
}

func BuiltinBufferFunc(c Call) (ret Object, err error) {
	var (
		w = &Buffer{}
	)

	c.Args.Walk(func(i int, arg Object) (continueLoop bool) {
		switch t := arg.(type) {
		case String:
			_, err = w.Write([]byte(t))
		case Bytes:
			_, err = w.Write(t)
		case BytesConverter:
			var b Bytes
			if b, err = t.ToBytes(); err == nil {
				_, err = w.Write(b)
			}
		case ToWriter:
			_, err = t.WriteTo(w)
		default:
			_, err = fmt.Fprint(w, arg)
		}
		return err == nil
	})

	return w, err
}

func BuiltinPrintFunc(c Call) (_ Object, err error) {
	var (
		w     io.Writer = c.VM.StdOut
		total Int
		n     int
	)

	if err = c.Args.CheckMinLen(1); err != nil {
		return
	}

	arg := c.Args.Get(0)
	if w2, ok := arg.(Writer); ok {
		w = w2
		c.Args.Shift()
	}

	switch size := c.Args.Len(); size {
	case 0:
	default:
		vargs := make([]any, 0, size)
		for i := 0; i < size; i++ {
			vargs = append(vargs, c.Args.Get(i))
		}
		n, err = fmt.Fprint(w, vargs...)
		return Int(n), err
	}

	return total, err
}

func BuiltinPrintlnFunc(c Call) (ret Object, err error) {
	var (
		w io.Writer = c.VM.StdOut
		n int
	)

	switch size := c.Args.Len(); size {
	case 0:
		n, err = w.Write([]byte("\n"))
	case 1:
		arg := c.Args.Get(0)
		if w2, ok := arg.(Writer); ok {
			n, err = w2.Write([]byte("\n"))
		} else {
			n, err = fmt.Fprintln(w, c.Args.Get(0))
		}
	default:
		arg := c.Args.Get(0)
		if w2, ok := arg.(Writer); ok {
			w = w2
			c.Args.Shift()
			size--
		}

		vargs := make([]any, 0, size)
		for i := 0; i < size; i++ {
			vargs = append(vargs, c.Args.Get(i))
		}
		n, err = fmt.Fprintln(w, vargs...)
	}
	return Int(n), err
}

func BuiltinSprintfFunc(c Call) (ret Object, err error) {
	ret = Nil
	switch size := c.Args.Len(); size {
	case 0:
		err = ErrWrongNumArguments.NewError("want>=1 got=0")
	case 1:
		ret = String(c.Args.Get(0).ToString())
	default:
		format, _ := c.Args.ShiftOk()
		vargs := make([]any, 0, size-1)
		for i := 0; i < size-1; i++ {
			vargs = append(vargs, c.Args.Get(i))
		}
		ret = String(fmt.Sprintf(format.ToString(), vargs...))
	}
	return
}

func BuiltinGlobalsFunc(c Call) (Object, error) {
	return c.VM.GetGlobals(), nil
}

func BuiltinIsFunc(c Call) (ok Object, err error) {
	if err = c.Args.CheckMinLen(2); err != nil {
		return
	}
	ok = True
	var (
		types []ObjectType
		t     = c.Args.Shift()
		argt  ObjectType
	)

	if arr, ok_ := t.(Array); ok_ {
		types = make([]ObjectType, len(arr))
		for i, t := range arr {
			if ot, _ := t.(ObjectType); ok_ {
				types[i] = ot
			} else {
				return nil, NewArgumentTypeError(fmt.Sprintf("1st [%d]", i), "type", "object")
			}
		}

		c.Args.Walk(func(i int, arg Object) bool {
			argt = arg.Type()
			for _, t := range types {
				if t.Equal(argt) {
					return true
				}
			}
			ok = False
			return false
		})
	} else {
		c.Args.Walk(func(i int, arg Object) bool {
			if !t.Equal(arg.Type()) {
				ok = False
				return false
			}
			return true
		})
	}
	return
}

func BuiltinIsErrorFunc(c Call) (ret Object, err error) {
	ret = False
	switch c.Args.Len() {
	case 1:
		// We have Error, BuiltinError and also user defined error types.
		if _, ok := c.Args.Get(0).(error); ok {
			ret = True
		}
	case 2:
		if err, ok := c.Args.Get(0).(error); ok {
			if target, ok := c.Args.Get(1).(error); ok {
				ret = Bool(errors.Is(err, target))
			}
		}
	default:
		err = ErrWrongNumArguments.NewError(
			"want=1..2 got=", strconv.Itoa(c.Args.Len()))
	}
	return
}

func BuiltinIsIntFunc(arg Object) Object {
	_, ok := arg.(Int)
	return Bool(ok)
}

func BuiltinIsUintFunc(arg Object) Object {
	_, ok := arg.(Uint)
	return Bool(ok)
}

func BuiltinIsFloatFunc(arg Object) Object {
	_, ok := arg.(Float)
	return Bool(ok)
}

func BuiltinIsCharFunc(arg Object) Object {
	_, ok := arg.(Char)
	return Bool(ok)
}

func BuiltinIsBoolFunc(arg Object) Object {
	_, ok := arg.(Bool)
	return Bool(ok)
}

func BuiltinIsStringFunc(arg Object) Object {
	_, ok := arg.(String)
	return Bool(ok)
}

func BuiltinIsBytesFunc(arg Object) Object {
	_, ok := arg.(Bytes)
	return Bool(ok)
}

func BuiltinIsDictFunc(arg Object) Object {
	_, ok := arg.(Dict)
	return Bool(ok)
}

func BuiltinIsSyncDictFunc(arg Object) Object {
	_, ok := arg.(*SyncMap)
	return Bool(ok)
}

func BuiltinIsArrayFunc(arg Object) Object {
	_, ok := arg.(Array)
	return Bool(ok)
}

func BuiltinIsNilFunc(arg Object) Object {
	_, ok := arg.(*NilType)
	return Bool(ok)
}

func BuiltinIsFunctionFunc(arg Object) Object {
	_, ok := arg.(*CompiledFunction)
	if ok {
		return True
	}

	_, ok = arg.(*BuiltinFunction)
	if ok {
		return True
	}

	_, ok = arg.(*Function)
	return Bool(ok)
}

func BuiltinIsCallableFunc(arg Object) Object {
	return Bool(Callable(arg))
}

func BuiltinIsIterableFunc(arg Object) Object { return Bool(Iterable(arg)) }

func BuiltinKeysFunc(c Call) (_ Object, err error) {
	if err := c.Args.CheckLen(1); err != nil {
		return nil, err
	}
	var arr Array
	switch v := c.Args.Get(0).(type) {
	case KeysGetter:
		arr = v.Keys()
	default:
		if Iterable(v) {
			it := v.(Iterabler).Iterate(c.VM)
			for it.Next() {
				arr = append(arr, it.Key())
			}
		}
	}
	return arr, nil
}

func BuiltinValuesFunc(c Call) (_ Object, err error) {
	if err := c.Args.CheckLen(1); err != nil {
		return nil, err
	}
	var arr Array
	switch v := c.Args.Get(0).(type) {
	case Array:
		arr = v
	case ValuesGetter:
		arr = v.Values()
	default:
		if Iterable(v) {
			var (
				it = v.(Iterabler).Iterate(c.VM)
				v  Object
			)
			for it.Next() {
				if v, err = it.Value(); err != nil {
					return nil, err
				}
				arr = append(arr, v)
			}
		}
	}
	return arr, nil
}

func BuiltinItemsFunc(c Call) (_ Object, err error) {
	if err := c.Args.CheckLen(1); err != nil {
		return nil, err
	}
	var arr KeyValueArray
	switch v := c.Args.Get(0).(type) {
	case ItemsGetter:
		arr = v.Items()
	default:
		if Iterable(v) {
			var (
				it = v.(Iterabler).Iterate(c.VM)
				v  Object
			)
			for it.Next() {
				if v, err = it.Value(); err != nil {
					return nil, err
				}
				arr = append(arr, KeyValue{it.Key(), v})
			}
		}
	}
	return arr, nil
}

func BuiltinKeyValueFunc(c Call) (ret Object, err error) {
	if err := c.Args.CheckLen(2); err != nil {
		return nil, err
	}
	return KeyValue{c.Args.Get(0), c.Args.Get(1)}, nil
}

func BuiltinKeyValueArrayFunc(c Call) (Object, error) {
	var (
		arr        KeyValueArray
		arg, valid = c.Args.ShiftOk()
	)

	for valid {
		switch t := arg.(type) {
		case KeyValueArray:
			arr = append(arr, t...)
		case KeyValue:
			arr = append(arr, t)
		case Array:
			if len(t) == 2 {
				arr = append(arr, KeyValue{t[0], t[1]})
			}
		case ItemsGetter:
			arr = append(arr, t.Items()...)
		}
		arg, valid = c.Args.ShiftOk()
	}
	return arr, nil
}

func BuiltinStdIOFunc(c Call) (ret Object, err error) {
	ret = Nil
	l := c.Args.Len()
	switch l {
	case 1:
		// get
		var arg = &Arg{AcceptTypes: []ObjectType{TString, TInt, TUint}}
		if err = c.Args.Destructure(arg); err != nil {
			return
		}
		switch t := arg.Value.(type) {
		case String:
			switch t {
			case "IN":
				ret = c.VM.StdIn
			case "OUT":
				ret = c.VM.StdOut
			case "ERR":
				ret = c.VM.StdErr
			default:
				err = ErrUnexpectedArgValue.NewError("string(" + string(t) + ")")
			}
		case Int:
			switch t {
			case 0:
				ret = c.VM.StdIn
			case 1:
				ret = c.VM.StdOut
			case 2:
				ret = c.VM.StdErr
			default:
				err = ErrUnexpectedArgValue.NewError("int(" + t.ToString() + ")")
			}
		case Uint:
			switch t {
			case 0:
				ret = c.VM.StdIn
			case 1:
				ret = c.VM.StdOut
			case 2:
				ret = c.VM.StdErr
			default:
				err = ErrUnexpectedArgValue.NewError("uint(" + t.ToString() + ")")
			}
		}
	case 2:
		var code = -1
		var codeArg = &Arg{AcceptTypes: []ObjectType{TString, TInt, TUint}}
		if err = c.Args.DestructureValue(codeArg); err != nil {
			return
		}
		switch t := codeArg.Value.(type) {
		case String:
			switch t {
			case "IN":
				code = 0
			case "OUT":
				code = 1
			case "ERR":
				code = 2
			default:
				err = ErrUnexpectedArgValue.NewError("string(" + string(t) + ")")
			}
		case Int:
			switch t {
			case 0, 1, 2:
				code = int(t)
			default:
				err = ErrUnexpectedArgValue.NewError("int(" + t.ToString() + ")")
			}
		case Uint:
			switch t {
			case 0, 1, 2:
				code = int(t)
			default:
				err = ErrUnexpectedArgValue.NewError("uint(" + t.ToString() + ")")
			}
		}

		switch code {
		case 0:
			var v = &Arg{AcceptTypes: []ObjectType{TReader}}
			if err = c.Args.DestructureValue(v); err != nil {
				return
			}
			c.VM.StdIn = NewStackReader(v.Value.(Reader))
		case 1, 2:
			var v = &Arg{AcceptTypes: []ObjectType{TWriter}}
			if err = c.Args.DestructureValue(v); err != nil {
				return
			}
			if code == 1 {
				c.VM.StdOut = NewStackWriter(v.Value.(Writer))
			} else {
				c.VM.StdErr = NewStackWriter(v.Value.(Writer))
			}
		}
	// set
	default:
		err = ErrWrongNumArguments.NewError(fmt.Sprintf("want=1|2 got=%d", l))
	}
	return
}

func BuiltinPushWriterFunc(c Call) (ret Object, err error) {
	if c.Args.Len() == 0 {
		buf := &Buffer{}
		c.VM.StdOut.Push(buf)
		return buf, nil
	}

	if err := c.Args.CheckMaxLen(1); err != nil {
		return nil, err
	}

	arg := c.Args.Get(0)
	if arg == Nil {
		arg = DiscardWriter
	}
	if w, ok := arg.(Writer); ok {
		c.VM.StdOut.Push(w)
		return w, nil
	}

	return nil, NewArgumentTypeError(
		"1st",
		"writer",
		arg.Type().Name(),
	)
}

func BuiltinPopWriterFunc(c Call) (ret Object, err error) {
	return c.VM.StdOut.Pop(), nil
}

func BuiltinOBStartFunc(c Call) (ret Object, err error) {
	return BuiltinPushWriterFunc(Call{VM: c.VM, Args: Args{Array{&Buffer{}}}})
}

func BuiltinOBEndFunc(c Call) (ret Object, err error) {
	return c.VM.StdOut.Pop(), nil
}

func BuiltinFlushFunc(c Call) (Object, error) {
	return c.VM.StdOut.Flush()
}

func BuiltinWrapFunc(c Call) (ret Object, err error) {
	if err = c.Args.CheckMinLen(1); err != nil {
		return
	}
	caller := c.Args.Shift()
	if !Callable(caller) {
		err = ErrNotCallable.NewError("1st arg")
	}
	return &CallWrapper{
		Caller:    caller.(CallerObject),
		Args:      c.Args.Copy().(Args),
		NamedArgs: c.NamedArgs.UnreadPairs(),
	}, nil
}

func BuiltinNewTypeFunc(c Call) (ret Object, err error) {
	var t ObjType
	var name = &Arg{
		Name:        "name",
		AcceptTypes: []ObjectType{TString},
	}
	if err = c.Args.Destructure(name); err != nil {
		return
	}
	t.TypeName = string(name.Value.(String))
	var (
		get = &NamedArgVar{
			Name:        "get",
			AcceptTypes: []ObjectType{TDict},
		}
		set = &NamedArgVar{
			Name:        "set",
			AcceptTypes: []ObjectType{TDict},
		}
		fields = &NamedArgVar{
			Name:        "fields",
			AcceptTypes: []ObjectType{TDict},
		}
		methods = &NamedArgVar{
			Name:        "methods",
			AcceptTypes: []ObjectType{TDict},
		}
		init = &NamedArgVar{
			Name: "init",
			Accept: func(v Object) error {
				if !Callable(v) {
					return ErrNotCallable
				}
				return nil
			},
		}
		toString = &NamedArgVar{
			Name: "toString",
			Accept: func(v Object) error {
				if !Callable(v) {
					return ErrNotCallable
				}
				return nil
			},
		}
		extends = &NamedArgVar{
			Name:        "extends",
			AcceptTypes: []ObjectType{TArray},
		}
	)

	if err = c.NamedArgs.Get(init, get, set, fields, methods, toString, extends); err != nil {
		return
	}

	if init.Value != nil {
		t.Init = init.Value.(CallerObject)
	}

	if toString.Value != nil {
		t.Stringer = toString.Value.(CallerObject)
	}

	if fields.Value != nil {
		t.FieldsDict = fields.Value.(Dict)
	}

	if get.Value != nil {
		t.GettersDict = Dict{}
		for name, v := range get.Value.(Dict) {
			if !Callable(v) {
				return nil, NewArgumentTypeError(
					"get["+name+"]st",
					"callable",
					v.Type().Name(),
				)
			}
			t.GettersDict[name] = v
		}
	}

	if set.Value != nil {
		t.SettersDict = Dict{}
		for name, v := range set.Value.(Dict) {
			if !Callable(v) {
				return nil, NewArgumentTypeError(
					"set["+name+"]st",
					"callable",
					v.Type().Name(),
				)
			}
			t.SettersDict[name] = v
		}
	}
	if methods.Value != nil {
		t.MethodsDict = Dict{}
		for name, v := range methods.Value.(Dict) {
			if !Callable(v) {
				return nil, NewArgumentTypeError(
					"method["+name+"]st",
					"callable",
					v.Type().Name(),
				)
			}
			t.MethodsDict[name] = v
		}
	}
	if extends.Value != nil {
		arr := methods.Value.(Array)
		t.Inherits = make(ObjectTypeArray, len(arr))
		for i, v := range arr {
			if ot, _ := v.(ObjectType); ot == nil {
				return nil, NewArgumentTypeError(
					"extends["+strconv.Itoa(i)+"]st",
					"ObjectType",
					v.Type().Name(),
				)
			} else {
				t.Inherits = append(t.Inherits, ot)
				for name, f := range ot.Fields() {
					if _, ok := t.FieldsDict[name]; !ok {
						t.FieldsDict[name] = f
					}
				}
				for name, f := range ot.Getters() {
					if _, ok := t.GettersDict[name]; !ok {
						t.GettersDict[name] = f
					}
				}
				for name, f := range ot.Setters() {
					if _, ok := t.SettersDict[name]; !ok {
						t.SettersDict[name] = f
					}
				}
				for name, f := range ot.Methods() {
					if _, ok := t.MethodsDict[name]; !ok {
						t.MethodsDict[name] = f
					}
				}
			}
		}
	}
	return &t, nil
}

func BuiltinTypeOfFunc(c Call) (_ Object, err error) {
	if err = c.Args.CheckLen(1); err != nil {
		return
	}

	return TypeOf(c.Args.Get(0)), nil
}

func BuiltinSyncMapFunc(c Call) (ret Object, err error) {
	if err = c.Args.CheckMaxLen(1); err != nil {
		return
	}
	if c.Args.Len() == 0 {
		return &SyncMap{Value: map[string]Object{}}, nil
	}
	arg := c.Args.Get(0)
	switch t := arg.(type) {
	case Dict:
		return &SyncMap{Value: t}, nil
	case *SyncMap:
		return t, nil
	default:
		err = NewArgumentTypeError(
			"0st",
			"map|syncMap",
			arg.Type().Name(),
		)
	}
	return
}

func BuiltinCastFunc(c Call) (ret Object, err error) {
	if err = c.Args.CheckLen(2); err != nil {
		return
	}

	var (
		typ = &Arg{Accept: func(v Object) string {
			if ot, _ := v.(ObjectType); ot == nil {
				return "objectType"
			}
			return ""
		}}
		obj = &Arg{Accept: func(v Object) string {
			if ot, _ := v.(Objector); ot == nil {
				return "objector"
			}
			return ""
		}}
	)
	if err = c.Args.Destructure(typ, obj); err != nil {
		return
	}
	curFields := obj.Value.(Objector).Fields()
	ot2 := typ.Value.(ObjectType)
	for f := range ot2.Fields() {
		if curFields[f] == nil {
			err = ErrIncompatibleCast.NewError(fmt.Sprintf("field %q not found in %s", f, ot2.ToString()))
			return
		}
	}
	return ot2.New(c.VM, obj.Value.(Objector).Fields())
}
