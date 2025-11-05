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

	"github.com/gad-lang/gad/repr"
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
			obj = arg.AppendToArray(obj)
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
		ret := make(Array, 0, c.Args.Length())
		for _, arg := range c.Args {
			ret = arg.AppendToArray(ret)
		}
		return ret, nil
	case Appender:
		return obj.AppendObjects(c.VM, c.Args.Values()...)
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
			TypeAssertion: NewTypeAssertion(TypeAssertionHandlers{
				"indexDeleter": IsIndexDeleter,
			}),
		}
		key = &Arg{}
	)
	if err = c.Args.Destructure(target, key); err != nil {
		return
	}
	return Nil, target.Value.(IndexDeleter).IndexDelete(c.VM, key.Value)
}

func BuiltinCopyFunc(c Call) (_ Object, err error) {
	switch c.Args.Length() {
	case 2:
		var (
			w = &Arg{
				Name: "writer",
				TypeAssertion: &TypeAssertion{
					Handlers: map[string]TypeAssertionHandler{
						"writer": func(v Object) (ok bool) {
							return WriterFrom(v) != nil
						},
					},
				},
			}
			r = &Arg{
				Name: "reader",
				TypeAssertion: &TypeAssertion{
					Handlers: map[string]TypeAssertionHandler{
						"reader": func(v Object) (ok bool) {
							return ReaderFrom(v) != nil
						},
					},
				},
			}
		)

		if err = c.Args.Destructure(w, r); err != nil {
			return
		}

		var n int64
		n, err = io.Copy(WriterFrom(w.Value).GoWriter(), ReaderFrom(r.Value).GoReader())
		return Int(n), err
	default:
		if err = c.Args.CheckLen(1); err != nil {
			return
		}
	}

	switch t := c.Args.GetOnly(0).(type) {
	case Copier:
		return t.Copy(), nil
	default:
		return t, nil
	}
}

func BuiltinDeepCopyFunc(c Call) (_ Object, err error) {
	if err = c.Args.CheckLen(1); err != nil {
		return
	}

	switch t := c.Args.GetOnly(0).(type) {
	case DeepCopier:
		return t.DeepCopy(c.VM)
	case Copier:
		return t.Copy(), nil
	default:
		return t, nil
	}
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
	case Str:
		ret = Str(strings.Repeat(string(v), count))
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
	case *SyncDict:
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
	case Str:
		ok = strings.Contains(string(obj), arg1.ToString())
	case Bytes:
		switch v := arg1.(type) {
		case Int:
			ok = bytes.Contains(obj, []byte{byte(v)})
		case Uint:
			ok = bytes.Contains(obj, []byte{byte(v)})
		case Char:
			ok = bytes.Contains(obj, []byte{byte(v)})
		case Str:
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
			"dict|array|string|bytes|namedArgs",
			arg0.Type().Name(),
		)
	}
	return Bool(ok), nil
}

func BuiltinLenFunc(arg Object) Object {
	var n int
	if v, ok := arg.(LengthGetter); ok {
		n = v.Length()
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

func BuiltinSortFunc(vm *VM, arg Object, less CallerObject) (ret Object, err error) {
	switch obj := arg.(type) {
	case Sorter:
		ret, err = obj.Sort(vm, less)
	case Str:
		s := []rune(obj)
		sort.Slice(s, func(i, j int) bool {
			return s[i] < s[j]
		})
		ret = Str(s)
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

func BuiltinSortReverseFunc(vm *VM, arg Object, less CallerObject) (Object, error) {
	switch obj := arg.(type) {
	case ReverseSorter:
		return obj.SortReverse(vm)
	case Str:
		s := []rune(obj)
		sort.Slice(s, func(i, j int) bool {
			return s[j] < s[i]
		})
		return Str(s), nil
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
			TypeAssertion: NewTypeAssertion(TypeAssertionHandlers{
				"iterable": func(v Object) bool {
					return Iterable(c.VM, v)
				},
				"filterable": Filterable,
			}),
		}

		callback = &Arg{
			Name: "callback",
			TypeAssertion: NewTypeAssertion(TypeAssertionHandlers{
				"callable": Callable,
			}),
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

	var it Iterator
	if _, it, err = ToIterator(c.VM, iterabler.Value, &c.NamedArgs); err != nil {
		return
	}
	return IteratorObject(NewPipedInvokeIterator(it, args, 0, caller).
		SetType(TFilterIterator).
		SetPostCall(func(state *IteratorState, ret Object) error {
			if ret.IsFalsy() {
				state.Mode = IteratorStateModeContinue
			}
			return nil
		})), nil
}

func BuiltinMapFunc(c Call) (_ Object, err error) {
	var (
		iterabler = &Arg{
			Name: "iterable",
			TypeAssertion: NewTypeAssertion(TypeAssertionHandlers{
				"mapable": Mapable,
				"iterable": func(v Object) bool {
					return Iterable(c.VM, v)
				},
			}),
		}

		callback = &Arg{
			Name: "callback",
			TypeAssertion: NewTypeAssertion(TypeAssertionHandlers{
				"callable": Callable,
			}),
		}

		update = &NamedArgVar{
			Name:  "update",
			Value: False,
		}

		nokey = &NamedArgVar{
			Name:  "nokey",
			Value: False,
		}

		canUpdate bool
	)

	if err = c.NamedArgs.Get(update, nokey); err != nil {
		return
	}

	if canUpdate = !update.Value.IsFalsy(); canUpdate {
		iterabler.AcceptHandler("IndexSetter", func(v Object) bool {
			_, ok := v.(IndexSetter)
			return ok
		})
	}

	if err = c.Args.Destructure(iterabler, callback); err != nil {
		return
	}

	var (
		args   = Array{Nil}
		caller VMCaller
	)

	if nokey.Value.IsFalsy() {
		args = append(args, Nil)
	}

	if caller, err = NewInvoker(c.VM, callback.Value).Caller(Args{args}, &c.NamedArgs); err != nil {
		return
	}

	if Mapable(iterabler.Value) {
		return iterabler.Value.(Mapabler).Map(c, bool(update.Value.(Bool)), args, caller)
	}

	var it Iterator
	if _, it, err = ToIterator(c.VM, iterabler.Value, &c.NamedArgs); err != nil {
		return
	}

	fe := NewPipedInvokeIterator(it, args, 0, caller).
		SetType(TMapIterator)

	if canUpdate {
		var indexSetter IndexSetter
		switch t := iterabler.Value.(type) {
		case Array:
			indexSetter = &t
		case IndexSetter:
			indexSetter = t
		}
		fe.SetPostCall(func(state *IteratorState, ret Object) (err error) {
			err = indexSetter.IndexSet(c.VM, state.Entry.K, ret)
			state.Entry.V = ret
			return
		})
		iterabler.Value = indexSetter
	}

	return IteratorObject(fe), nil
}

func BuiltinEnumerateFunc(c Call) (_ Object, err error) {
	if err := c.Args.CheckLen(1); err != nil {
		return nil, err
	}
	var (
		v  = c.Args.Get(0)
		it Iterator
		i  Int
	)
	if _, it, err = ToIterator(c.VM, v, &c.NamedArgs); err != nil {
		return
	}
	if values := c.NamedArgs.MustGetValue("values"); !values.IsFalsy() {
		return TypedIteratorObject(TEnumerateIterator, WrapIterator(it, func(state *IteratorState) error {
			state.Entry.K = i
			i++
			return nil
		})), nil
	} else if keys := c.NamedArgs.MustGetValue("keys"); !keys.IsFalsy() {
		return TypedIteratorObject(TEnumerateIterator, WrapIterator(it, func(state *IteratorState) error {
			state.Entry.V = state.Entry.K
			state.Entry.K = i
			i++
			return nil
		})), nil
	}
	return TypedIteratorObject(TEnumerateIterator, WrapIterator(it, func(state *IteratorState) error {
		kv := state.Entry
		state.Entry.K = i
		state.Entry.V = &kv
		i++
		return nil
	})), nil
}

func BuiltinEachFunc(c Call) (_ Object, err error) {
	var (
		iterabler = &Arg{
			Name: "iterable",
			TypeAssertion: NewTypeAssertion(TypeAssertionHandlers{
				"iterable": func(v Object) bool {
					return Iterable(c.VM, v)
				},
			}),
		}

		callback = &Arg{
			Name: "callback",
			TypeAssertion: NewTypeAssertion(TypeAssertionHandlers{
				"callable": Callable,
			}),
		}
	)

	if c.Args.Length() == 1 {
		if err = c.Args.Destructure(iterabler); err != nil {
			return
		}

		err = IterateObject(c.VM, iterabler.Value, &c.NamedArgs, nil, func(e *KeyValue) (err error) {
			return
		})
		return iterabler.Value, err
	} else if err = c.Args.Destructure(iterabler, callback); err != nil {
		return
	}

	var (
		args   = Array{Nil, Nil}
		caller VMCaller
	)

	if caller, err = NewInvoker(c.VM, callback.Value).Caller(Args{args}, &c.NamedArgs); err != nil {
		return
	}

	err = IterateObject(c.VM, iterabler.Value, &c.NamedArgs, nil, func(e *KeyValue) (err error) {
		args[0] = e.K
		args[1] = e.V
		_, err = caller.Call()
		return
	})

	return iterabler.Value, err
}

func BuiltinReduceFunc(c Call) (_ Object, err error) {
	var (
		iterabler = &Arg{
			Name: "iterable",
			TypeAssertion: NewTypeAssertion(TypeAssertionHandlers{
				"reducable": Reducable,
				"iterable": func(v Object) bool {
					return Iterable(c.VM, v)
				},
			}),
		}

		callback = &Arg{
			Name: "callback",
			TypeAssertion: NewTypeAssertion(TypeAssertionHandlers{
				"callable": Callable,
			}),
		}

		args   = Array{Nil, Nil, Nil}
		caller VMCaller
	)

	if c.Args.Length() == 3 {
		initialArg := &Arg{}
		if err = c.Args.Destructure(iterabler, callback, initialArg); err != nil {
			return
		}
		args[0] = initialArg.Value
	} else {
		if err = c.Args.Destructure(iterabler, callback); err != nil {
			return
		}
	}

	if caller, err = NewInvoker(c.VM, callback.Value).Caller(Args{args}, &c.NamedArgs); err != nil {
		return
	}

	if Reducable(iterabler.Value) {
		return iterabler.Value.(Reducer).Reduce(c.VM, args[0], args, caller)
	}

	var it Iterator
	if _, it, err = ToIterator(c.VM, iterabler.Value, &c.NamedArgs); err != nil {
		return
	}

	fe := NewPipedInvokeIterator(it, args, 1, caller)

	if args[0] == Nil {
		fe.preCall = func(k, v Object) (Object, error) {
			args[0] = v
			return v, nil
		}
	}

	err = Iterate(c.VM, fe, nil, func(e *KeyValue) error {
		args[0] = e.V
		return nil
	})
	return args[0], err
}

func BuiltinTypeNameFunc(arg Object) Object { return Str(arg.Type().Name()) }

func BuiltinCharsFunc(arg Object) (ret Object, err error) {
	switch obj := arg.(type) {
	case Str:
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
		out = &NamedArgVar{
			Value:         c.VM.StdOut,
			TypeAssertion: TypeAssertionFromTypes(TWriter),
		}
		n int
	)

	if err = c.NamedArgs.Get(out); err != nil {
		return
	}

	w := out.Value.(Writer)

	switch size := c.Args.Length(); size {
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

func BuiltinCloseFunc(c Call) (ret Object, err error) {
	if err = c.Args.CheckMinLen(1); err != nil {
		return
	}
	if l := c.Args.Length(); l == 1 {
		ret = c.Args.GetOnly(0)
		if c := CloserFrom(ret); c != nil {
			err = c.Close()
		}
		return
	}

	c.Args.Walk(func(i int, arg Object) any {
		if c := CloserFrom(arg); c != nil {
			if err = c.Close(); err != nil {
				return err
			}
		}
		return nil
	})

	return
}

func BuiltinReadFunc(c Call) (ret Object, err error) {
	var (
		reader = &Arg{
			Name: "reader",
			TypeAssertion: &TypeAssertion{
				Handlers: map[string]TypeAssertionHandler{
					"reader": func(v Object) (ok bool) {
						return ReaderFrom(v) != nil
					},
				},
			},
		}

		limit = &NamedArgVar{
			Name:          "limit",
			Value:         Int(0),
			TypeAssertion: TypeAssertionFromTypes(TInt),
		}
		close = &NamedArgVar{
			Name:  "close",
			Value: No,
		}
		b        []byte
		buffered bool
	)

	switch c.Args.Length() {
	case 0:
		reader.Value = c.VM.StdIn
	case 2:
		buffered = true
		var buffer = &Arg{
			Name:          "buffer",
			TypeAssertion: TypeAssertionFromTypes(TBytes),
		}

		if err = c.Args.Destructure(reader, buffer); err != nil {
			return
		}

		b = buffer.Value.(Bytes)
	default:
		if err = c.Args.Destructure(reader); err != nil {
			return
		}
	}

	if err = c.NamedArgs.Get(limit, close); err != nil {
		return
	}

	var l, s int

	if buffered {
		l = len(b)
	} else {
		if l = int(limit.Value.(Int)); l < 0 {
			l = 0
		}
	}

	if l == 0 {
		if buffered {
			return Bytes{}, nil
		}
		b, err = io.ReadAll(ReaderFrom(reader.Value).GoReader())
		s = len(b)
	} else {
		if len(b) == 0 {
			b = make([]byte, l)
		}
		s, err = ReaderFrom(reader.Value).Read(b)
	}

	if !close.Value.IsFalsy() {
		_, err = c.VM.Builtins.Call(BuiltinClose, Call{VM: c.VM, Args: Args{Array{reader.Value}}})
	}

	if err != nil {
		return
	}

	return Bytes(b[:s]), nil
}

func BuiltinWriteFunc(c Call) (ret Object, err error) {
	var (
		w     io.Writer = c.VM.StdOut
		total Int
		n     int
		write = func(w io.Writer, obj Object) (i int64, err error) {
			_, i, err = c.VM.ObjectToWriter.WriteTo(c.VM, w, obj)
			return
		}
		convert CallerObject
	)

	if err = c.Args.CheckMinLen(1); err != nil {
		return
	}

	arg := c.Args.Get(0)
	if w2 := WriterFrom(arg); w2 != nil {
		w = w2
		c.Args.Shift()
	}

	if convertValue := c.NamedArgs.GetValueOrNil("convert"); convertValue != nil {
		convert = convertValue.(CallerObject)
	}

	if convert == nil {
		c.Args.Walk(func(i int, arg Object) any {
			switch t := arg.(type) {
			case RawStr:
				n, err = w.Write([]byte(t))
				total += Int(n)
			default:
				var n2 int64
				n2, err = write(w, arg)
				total += Int(n2)
			}
			return err
		})
	} else {
		var (
			convertCallArgs = Array{
				NewWriter(w),
				&Function{
					Value: func(c Call) (_ Object, err error) {
						var i int64
						i, err = write(c.Args.MustGet(0).(Writer), c.Args.MustGet(1))
						return Int(i), err
					},
				},
				nil,
			}
			caller VMCaller
		)
		if caller, err = NewInvoker(c.VM, convert).Caller(Args{convertCallArgs}, nil); err != nil {
			return
		}

		c.Args.Walk(func(i int, arg Object) any {
			switch t := arg.(type) {
			case RawStr:
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
			return err
		})
	}

	if !c.NamedArgs.GetValue("close").IsFalsy() {
		_, err = c.VM.Builtins.Call(BuiltinClose, Call{VM: c.VM, Args: Args{Array{arg}}})
	}

	return total, err
}

func BuiltinMultiValueDictFunc(c Call) (ret Object, err error) {
	var (
		d  = Dict{}
		cb = func(kv *KeyValue) error {
			k := kv.K.ToString()
			if arr, ok := d[k].(Array); ok {
				d[k] = append(arr, kv.V)
			} else {
				d[k] = Array{kv.V}
			}
			return nil
		}
	)

	if c.Args.IsFalsy() {
		err = ItemsOfCb(c.VM, nil, cb, &c.NamedArgs)
	} else {
		err = c.Args.Items(c.VM, func(_ int, item *KeyValue) (err error) {
			return ItemsOfCb(c.VM, &c.NamedArgs, cb, item.V)
		})
	}
	ret = d
	return
}

func BuiltinPrintFunc(c Call) (bytesWritten Object, err error) {
	if err = c.Args.CheckMinLen(1); err != nil {
		return
	}

	var (
		state             = PrinterStateFromCall(&c)
		startBytesWritten = state.bytesWriten
	)

	defer func() {
		bytesWritten = Int(state.bytesWriten - startBytesWritten)
	}()

	err = state.PrintFromArgs([]byte{' '}, c.Args)
	return
}

func BuiltinPrintlnFunc(c Call) (bytesWritten Object, err error) {
	if c.Args.Length() == 0 {
		_, err = c.VM.Write([]byte{'\n'})
		return Int(1), err
	}

	var (
		state             = PrinterStateFromCall(&c)
		startBytesWritten = state.bytesWriten
	)

	defer func() {
		bytesWritten = Int(state.bytesWriten - startBytesWritten)
	}()

	if err = state.PrintFromArgs([]byte{' '}, c.Args); err != nil {
		return
	}

	_, err = state.Write([]byte{'\n'})
	return
}

func BuiltinSprintfFunc(c Call) (ret Object, err error) {
	ret = Nil
	switch size := c.Args.Length(); size {
	case 0:
		err = ErrWrongNumArguments.NewError("want>=1 got=0")
	case 1:
		ret = Str(c.Args.Get(0).ToString())
	default:
		format, _ := c.Args.ShiftOk()
		vargs := make([]any, 0, size-1)
		for i := 0; i < size-1; i++ {
			vargs = append(vargs, c.Args.Get(i))
		}
		ret = Str(fmt.Sprintf(format.ToString(), vargs...))
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
		types     []ObjectType
		t         = c.Args.Shift()
		argt      ObjectType
		assertion *TypeAssertion
	)

	if arr, ok_ := t.(Array); ok_ {
		types = make([]ObjectType, len(arr))
		for i, t := range arr {
		try:
			switch t2 := t.(type) {
			case ObjectType:
				types[i] = t2
			case *CallerObjectWithMethods:
				t = t2.CallerObject
				goto try
			default:
				return nil, NewArgumentTypeError(fmt.Sprintf("1st [%d]", i), "type", "object")
			}
		}

		assertion = TypeAssertionFromTypes(types...)
	} else {
		if com, ok := t.(*CallerObjectWithMethods); ok {
			t = com.CallerObject
		}
		if t, ok := t.(ObjectType); !ok {
			return nil, NewArgumentTypeError("1st", "type|array of types", "object")
		} else {
			assertion = TypeAssertionFromTypes(t)
		}
	}

	c.Args.Walk(func(i int, arg Object) any {
		argt = arg.Type()
		if expectedNames := assertion.AcceptType(argt); expectedNames != "" {
			ok = False
		}
		return ok
	})

	return
}

func BuiltinNamedParamTypeCheckFunc(c Call) (val Object, err error) {
	var (
		nameArg = &Arg{
			Name:          "NamedParam",
			TypeAssertion: TypeAssertionFromTypes(TStr),
		}

		typesArg = &Arg{
			Name: "types",
		}

		valueArg = &Arg{
			Name: "value",
		}
	)
	if err = c.Args.Destructure(nameArg, typesArg, valueArg); err != nil {
		return
	}

	val = Nil
	var badTypes string
	if badTypes, err = NamedParamTypeCheck(string(nameArg.Value.(Str)), typesArg.Value, valueArg.Value); err != nil {
		return
	} else if badTypes != "" {
		err = NewArgumentTypeError(
			"2st (types)",
			badTypes,
			typesArg.Value.Type().Name(),
		)
	}
	return
}

func NamedParamTypeCheck(name string, typeso, value Object) (badTypes string, err error) {
	var (
		types     []ObjectType
		assertion = &TypeAssertion{
			Handlers: map[string]TypeAssertionHandler{
				"ObjectType|[]ObjectType": func(v Object) bool {
					switch t := v.(type) {
					case ObjectType:
						types = append(types, t)
						return true
					case Array:
						for _, object := range t {
						try:
							switch t2 := object.(type) {
							case ObjectType:
								types = append(types, t2)
							case *CallerObjectWithMethods:
								object = t2.CallerObject
								goto try
							default:
								return false
							}
						}
						return true
					default:
					try2:
						switch t2 := t.(type) {
						case ObjectType:
							types = append(types, t2)
							return true
						case *CallerObjectWithMethods:
							t = t2.CallerObject
							goto try2
						default:
							return false
						}
					}
				},
			},
		}
	)

	if badTypes = assertion.Accept(typeso); badTypes != "" {
		return
	}

	err = NamedParamTypeCheckAssertion(name, TypeAssertionFromTypes(types...), value)
	return
}

func NamedParamTypeCheckAssertion(name string, assertion *TypeAssertion, value Object) (err error) {
	if expectedNames := assertion.Accept(value); expectedNames != "" {
		err = NewNamedArgumentTypeError(
			string(name),
			expectedNames,
			value.Type().Name(),
		)
		return
	}
	return
}

func BuiltinIsErrorFunc(c Call) (ret Object, err error) {
	ret = False
	switch c.Args.Length() {
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
			"want=1..2 got=", strconv.Itoa(c.Args.Length()))
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

func BuiltinIsStrFunc(arg Object) Object {
	_, ok := arg.(Str)
	return Bool(ok)
}

func BuiltinIsRawStrFunc(arg Object) Object {
	_, ok := arg.(RawStr)
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
	_, ok := arg.(*SyncDict)
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
	if com, _ := arg.(*CallerObjectWithMethods); com != nil {
		arg = com.CallerObject
	}

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

func BuiltinIsIterableFunc(vm *VM, arg Object) Object {
	switch arg.(type) {
	case Iterator:
		return True
	case Iterabler:
		if cit, _ := arg.(CanIterabler); cit != nil {
			return Bool(cit.CanIterate())
		}
		return True
	}

	m := vm.Builtins.Get(BuiltinIterator).(MethodCaller).CallerMethodOfArgsTypes(ObjectTypes{arg.Type()})
	return Bool(m != nil)
}
func BuiltinIsIteratorFunc(arg Object) Object {
	return Bool(IsIterator(arg))
}

func BuiltinIterateFunc(c Call) (_ Object, err error) {
	if err := c.Args.CheckLen(1); err != nil {
		return nil, err
	}

	var (
		v  = c.Args.Get(0)
		it Iterator
	)

	if _, it, err = ToIterator(c.VM, v, &c.NamedArgs); err != nil {
		return
	}

	return IteratorObject(it), nil
}

func BuiltinIterationDoneFunc(c Call) (_ Object, err error) {
	if err = c.Args.CheckLen(1); err != nil {
		return
	}
	if ite := ToIterationDoner(c.Args.GetOnly(1)); ite != nil {
		err = ite.IterationDone(c.VM)
	}
	return
}

func BuiltinKeysFunc(c Call) (_ Object, err error) {
	if err := c.Args.CheckLen(1); err != nil {
		return nil, err
	}
	var (
		v  = c.Args.Get(0)
		it Iterator
	)
	if _, it, err = ToIterator(c.VM, v, &c.NamedArgs); err != nil {
		return
	}
	return TypedIteratorObject(TKeysIterator, CollectModeIterator(it, IteratorStateCollectModeKeys)), nil
}

func BuiltinValuesFunc(c Call) (_ Object, err error) {
	if err := c.Args.CheckLen(1); err != nil {
		return nil, err
	}
	var (
		v  = c.Args.Get(0)
		it Iterator
	)
	if _, it, err = ToIterator(c.VM, v, &c.NamedArgs); err != nil {
		return
	}
	return TypedIteratorObject(TValuesIterator, CollectModeIterator(it, IteratorStateCollectModeValues)), nil
}

func BuiltinItemsFunc(c Call) (_ Object, err error) {
	if err := c.Args.CheckLen(1); err != nil {
		return nil, err
	}
	var (
		v  = c.Args.Get(0)
		it Iterator
	)
	if _, it, err = ToIterator(c.VM, v, &c.NamedArgs); err != nil {
		return
	}
	return TypedIteratorObject(TItemsIterator, CollectModeIterator(it, IteratorStateCollectModePair)), nil
}

func BuiltinCollectFunc(c Call) (_ Object, err error) {
	if err := c.Args.CheckLen(1); err != nil {
		return nil, err
	}

	var (
		o   = c.Args.Get(0)
		dst = Array{}
		h   func(e *KeyValue) Object
	)

	if oi, _ := o.(ObjectIterator); oi != nil {
		if itc, _ := oi.GetIterator().(CollectableIterator); itc != nil {
			return itc.Collect(c.VM)
		}
	}

	err = IterateObject(c.VM, o, &c.NamedArgs, func(state *IteratorState) error {
		switch state.CollectMode {
		case IteratorStateCollectModeKeys:
			h = func(e *KeyValue) Object {
				return e.K
			}
		case IteratorStateCollectModePair:
			h = func(e *KeyValue) Object {
				kv := *e
				return &kv
			}
		default:
			h = func(e *KeyValue) Object {
				return e.V
			}
		}
		return nil
	}, func(e *KeyValue) error {
		return dst.Append(c.VM, h(e))
	})
	return dst, err
}

func BuiltinIteratorInputFunc(o Object) Object {
	if IsIterator(o) {
		return o.(Iterator).Input()
	}
	return Nil
}

func BuiltinStdIOFunc(c Call) (ret Object, err error) {
	ret = Nil
	l := c.Args.Length()
	identifier := Arg{
		Name:          "indentifier",
		TypeAssertion: TypeAssertionFromTypes(TStr, TInt, TUint),
	}
	switch l {
	case 1:
		// get
		var arg = identifier

		if err = c.Args.Destructure(&arg); err != nil {
			return
		}
		switch t := arg.Value.(type) {
		case Str:
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
		var codeArg = identifier
		if err = c.Args.DestructureValue(&codeArg); err != nil {
			return
		}
		switch t := codeArg.Value.(type) {
		case Str:
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
			var v = &Arg{
				Name:          "in",
				TypeAssertion: TypeAssertionFromTypes(TReader),
			}
			if err = c.Args.DestructureValue(v); err != nil {
				return
			}
			c.VM.StdIn = NewStackReader(v.Value.(Reader))
		case 1, 2:
			var v = &Arg{
				Name:          "out",
				TypeAssertion: TypeAssertionFromTypes(TWriter),
			}
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
	if c.Args.Length() == 0 {
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

func BuiltinStructFunc(c Call) (ret Object, err error) {
	var (
		name = &Arg{
			Name:          "name",
			TypeAssertion: TypeAssertionFromTypes(TStr),
		}
	)
	if err = c.Args.Destructure(name); err != nil {
		return
	}
	t := NewObjType(string(name.Value.(Str)))
	var (
		get = &NamedArgVar{
			Name:          "get",
			TypeAssertion: TypeAssertionFromTypes(TDict),
		}
		set = &NamedArgVar{
			Name:          "set",
			TypeAssertion: get.TypeAssertion,
		}
		fields = &NamedArgVar{
			Name:          "fields",
			TypeAssertion: get.TypeAssertion,
		}
		methods = &NamedArgVar{
			Name:          "methods",
			TypeAssertion: get.TypeAssertion,
		}
		init = &NamedArgVar{
			Name: "init",
			TypeAssertion: NewTypeAssertion(TypeAssertionHandlers{
				"callable": Callable,
			}),
		}
		toString = &NamedArgVar{
			Name: "toString",
			TypeAssertion: NewTypeAssertion(TypeAssertionHandlers{
				"callable": Callable,
			}),
		}
		extends = &NamedArgVar{
			Name:          "extends",
			TypeAssertion: TypeAssertionFromTypes(TArray),
		}
	)

	if err = c.NamedArgs.Get(init, get, set, fields, methods, toString, extends); err != nil {
		return
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
	return t, nil
}

func BuiltinNewFunc(c Call) (ret Object, err error) {
	if err = c.Args.CheckLen(1); err != nil {
		return
	}
	var arg = c.Args.GetOnly(0)
	if typ, _ := arg.(ObjectType); typ != nil {
		if c.NamedArgs.IsFalsy() {
			return typ.New(c.VM, nil)
		}
		return typ.New(c.VM, c.NamedArgs.Dict())
	}
	if c.NamedArgs.IsFalsy() {
		return arg, nil
	}

	if is, ok := arg.(IndexSetter); ok {
		for k, v := range c.NamedArgs.Dict() {
			if err = is.IndexSet(c.VM, Str(k), v); err != nil {
				return
			}
		}
		return arg, nil
	}
	return NewArgumentTypeError("1st", "IndexSetter|ObjectType", arg.Type().Name()), nil
}

func BuiltinTypeOfFunc(c Call) (_ Object, err error) {
	if err = c.Args.CheckLen(1); err != nil {
		return
	}

	return TypeOf(c.Args.Get(0)), nil
}

func BuiltinBinaryOpFunc(c Call) (ret Object, err error) {
	var (
		op = &Arg{
			Name: "Op",
			TypeAssertion: new(TypeAssertion).
				AcceptHandler("BinaryOperatorType", func(v Object) (ok bool) {
					_, ok = v.(*BinaryOperatorType)
					return
				}),
		}
		left = &Arg{
			Name: "left",
		}
		right = &Arg{
			Name: "right",
		}
	)

	if err = c.Args.Destructure(op, left, right); err != nil {
		return
	}

	switch left := left.Value.(type) {
	case BinaryOperatorHandler:
		ret, err = left.BinaryOp(c.VM, op.Value.(*BinaryOperatorType).Token, right.Value)
	default:
		err = ErrInvalidOperator.NewError(op.Value.(*BinaryOperatorType).Name())
	}
	return
}

func BuiltinCastFunc(c Call) (ret Object, err error) {
	if err = c.Args.CheckLen(2); err != nil {
		return
	}

	var (
		typ = &Arg{
			Name: "toType",
			TypeAssertion: NewTypeAssertion(TypeAssertionHandlers{
				"objectType": IsType,
			}),
		}
		obj = &Arg{
			TypeAssertion: NewTypeAssertion(TypeAssertionHandlers{
				"objector": IsObjector,
			}),
		}
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

func BuiltinAddCallMethodFunc(vm *VM, fn CallerObject, handler CallerObject, override bool) (err error) {
	if fn, _ := fn.(MethodCaller); fn != nil {
		var types MultipleObjectTypes

		if cwm, _ := handler.(*CallerObjectWithMethods); cwm != nil {
			cwm.MethodWalk(func(m *CallerMethod) any {
				var handler = m.CallerObject
				if types, err = handler.(CallerObjectWithParamTypes).ParamTypes(vm); err != nil {
					return err
				}

				if err = fn.AddCallerMethod(vm, types, handler, override); err != nil {
					if mde := IsError(err, ErrMethodDuplication); mde != nil {
						mde.Message = fn.Caller().ToString() + ": " + mde.Message
					}
				}
				return err
			})
		} else {
			if types, err = handler.(CallerObjectWithParamTypes).ParamTypes(vm); err != nil {
				return
			}

			if cf, _ := handler.(*CompiledFunction); cf != nil {
				if l := len(cf.Params); l > 0 && !cf.Params.Typed() {
					types = make(MultipleObjectTypes, l)
					for i := range types {
						types[i] = ObjectTypes{nil}
					}
				}
			}

			if err = fn.AddCallerMethod(vm, types, handler, override); err != nil {
				if mde := IsError(err, ErrMethodDuplication); mde != nil {
					mde.Message = fn.Caller().ToString() + ": " + mde.Message
				}
			}
		}
	}
	return
}

func BuiltinRawCallerFunc(c Call) (ret Object, err error) {
	if err = c.Args.CheckLen(1); err != nil {
		return
	}

	var (
		obj = &Arg{
			Name: "caller",
			TypeAssertion: NewTypeAssertion(TypeAssertionHandlers{
				"caller": Callable,
			}),
		}
	)

	if err = c.Args.Destructure(obj); err != nil {
		return
	}
	co := obj.Value.(CallerObject)
	if cowm, _ := co.(*CallerObjectWithMethods); cowm != nil {
		return cowm.CallerObject, nil
	}
	return co, nil
}

func BuiltinReprFunc(c Call) (_ Object, err error) {
	if err = c.Args.CheckLen(1); err != nil {
		return
	}

	var (
		arg = c.Args.Get(0)
		s   string
	)

	switch t := arg.(type) {
	case ObjectRepresenter:
		s, err = t.Repr(c.VM)
		return Str(s), err
	}

	typ := arg.Type()

	if arg, err = Val(c.VM.Builtins.Call(BuiltinStr, c)); err != nil {
		return
	}
	return Str(repr.Quote(typ.Name() + ":" + arg.ToString())), nil
}

func BuiltinUserDataFunc(c Call) (_ Object, err error) {
	if err = c.Args.CheckLen(1); err != nil {
		return
	}

	var arg = c.Args.Get(0)
	if ud, _ := arg.(UserDataStorage); ud == nil {
		return Nil, NewArgumentTypeError(
			strconv.Itoa(1),
			"UserDataStorage",
			arg.Type().Name(),
		)
	} else {
		return ud.UserData(), nil
	}
}

func BuiltinToArrayFunc(c Call) (_ Object, err error) {
	var arr Array
	err = c.Args.WalkE(func(i int, arg Object) (err error) {
		switch t := arg.(type) {
		case Array:
			if i == 0 {
				arr = t
			} else {
				arr = append(arr, t...)
			}
		default:
			err = ItemsOfCb(c.VM, &c.NamedArgs, func(kv *KeyValue) error {
				arr = append(arr, kv)
				return nil
			}, arg)
		}
		return
	})
	return arr, err
}
