package gad

import (
	"regexp"
	"strings"

	"github.com/gad-lang/gad/token"
)

var (
	_ Object           = (*Regexp)(nil)
	_ NameCallerObject = (*Regexp)(nil)
	_ Printabler       = (*Regexp)(nil)
)

type Regexp regexp.Regexp

func (o *Regexp) CallName(name string, c Call) (_ Object, err error) {
	switch name {
	case "find":
		if err = c.Args.CheckLen(1); err != nil {
			return
		}
		return o.Find(c.Args.MustGet(0)), nil
	case "findAll":
		if err = c.Args.CheckMaxLen(1); err != nil {
			return
		}

		var count int

		if c.Args.Length() == 1 {
			count = -1
		} else {
			count, _ = ToGoInt(c.Args.MustGet(1))
		}

		return o.FindAll(c.Args.MustGet(0), count), nil
	case "match":
		if err = c.Args.CheckLen(1); err != nil {
			return
		}
		return o.Match(c.Args.MustGet(0)), nil
	case "replace":
		if err = c.Args.CheckLen(2); err != nil {
			return
		}
		return o.Replace(c.VM, c.Args.MustGet(0), c.Args.MustGet(1))
	}

	return nil, ErrInvalidIndex.NewError(name)
}

// Replace replaces all matches of o in subject with repl. repl may be a
// Str/RawStr/Bytes template (Go's $1 / ${name} group expansion applies) or a
// callable returning each replacement. The callable is invoked once per match
// with the matched substring as the positional argument and two named
// arguments: `m`, the full submatch (whole match + capture groups, so groups
// are `m[1]`, `m[2]`, …), and `re`, the regexp itself.
func (o *Regexp) Replace(vm *VM, subject, repl Object) (Object, error) {
	_, subjIsBytes := subject.(Bytes)

	switch r := repl.(type) {
	case Str, RawStr:
		if subjIsBytes {
			return Bytes(o.Go().ReplaceAll(subject.(Bytes), []byte(r.ToString()))), nil
		}
		return Str(o.Go().ReplaceAllString(subject.ToString(), r.ToString())), nil
	case Bytes:
		if subjIsBytes {
			return Bytes(o.Go().ReplaceAll(subject.(Bytes), r)), nil
		}
		return Str(o.Go().ReplaceAllString(subject.ToString(), string(r))), nil
	default:
		if !Callable(repl) {
			return nil, NewOperandTypeError("replace", repl.Type().Name(), o.Type().Name())
		}
		inv := NewInvoker(vm, repl)
		inv.Acquire()
		defer inv.Release()

		var (
			// The callable receives the whole match as the positional argument and
			// the full submatch (whole match + capture groups) as the named
			// argument `m`, so it can reference groups via `m[1]`, `m[2]`, … The
			// NamedArgs is built once and reused read-only across matches; the
			// backing Dict's `m` value is updated in place each iteration.
			subj          = subject.ToString()
			namedArgsDict = Dict{"m": Nil, "re": o}
			namedArgs     = namedArgsDict.ToNamedArgs().WithReadOnly(true)

			callErr error
			sb      strings.Builder
			last    int
		)

		for _, idx := range o.Go().FindAllStringSubmatchIndex(subj, -1) {
			groups := make(RegexpStrsResult, len(idx)/2)
			for i := 0; i < len(idx); i += 2 {
				if idx[i] >= 0 {
					groups[i/2] = subj[idx[i]:idx[i+1]]
				}
			}
			namedArgsDict["m"] = groups

			res, err := inv.Invoke(Args{Array{Str(subj[idx[0]:idx[1]])}}, namedArgs)
			if err != nil {
				callErr = err
				break
			}

			sb.WriteString(subj[last:idx[0]])
			sb.WriteString(res.ToString())
			last = idx[1]
		}
		if callErr != nil {
			return nil, callErr
		}
		sb.WriteString(subj[last:])

		result := sb.String()
		if subjIsBytes {
			return Bytes(result), nil
		}
		return Str(result), nil
	}
}

func (o *Regexp) Match(arg Object) (ret Bool) {
	switch t := arg.(type) {
	case Str, RawStr:
		ret = Bool(o.Go().MatchString(t.ToString()))
	case Bytes:
		ret = Bool(o.Go().Match(t))
	}
	return
}

func (o *Regexp) Find(arg Object) (ret Object) {
	ret = Nil
	// "^a" ~~ "a"
	switch t := arg.(type) {
	case Str, RawStr:
		ret = RegexpStrsResult(o.Go().FindStringSubmatch(t.ToString()))
	case Bytes:
		ret = RegexpBytesResult(o.Go().FindSubmatch(t))
	}
	return
}

func (o *Regexp) FindAll(arg Object, n int) (ret Object) {
	ret = Nil
	// "^a" ~~ "a"
	switch t := arg.(type) {
	case Str, RawStr:
		ret = RegexpStrsSliceResult(o.Go().FindAllStringSubmatch(t.ToString(), n))
	case Bytes:
		ret = RegexpBytesSliceResult(o.Go().FindAllSubmatch(t, n))
	}
	return
}

func (o *Regexp) BinaryOp(vm *VM, tok token.Token, right Object) (ret Object, err error) {
	switch tok {
	case token.Tilde:
		return o.Match(right), nil
	case token.DoubleTilde:
		return o.Find(right), nil
	case token.TripleTilde:
		return o.FindAll(right, -1), nil
	case token.Or:
		// `re | repl` yields a unary replacer: f(subject) -> replaced value.
		repl := right
		return &Function{
			FuncName: "regexpReplacer",
			Value: func(c Call) (Object, error) {
				if err := c.Args.CheckLen(1); err != nil {
					return nil, err
				}
				return o.Replace(c.VM, c.Args.MustGet(0), repl)
			},
		}, nil
	}
	return nil, NewOperandTypeError(
		tok.String(),
		o.Type().Name(),
		right.Type().Name())
}

func (o *Regexp) IsFalsy() bool {
	return false
}

func (o *Regexp) Type() ObjectType {
	return TRegexp
}

func (o *Regexp) ToInterface() any {
	return o.Go()
}

func (o *Regexp) Go() *regexp.Regexp {
	return (*regexp.Regexp)(o)
}

func (o *Regexp) ToString() string {
	return o.Go().String()
}

func (o *Regexp) Equal(right Object) bool {
	switch t := right.(type) {
	case *Regexp:
		return o == t
	default:
		return false
	}
}

func (o *Regexp) Print(state *PrinterState) error {
	if state.IsRepr {
		defer state.WrapRepr(o)()
	}
	return state.WriteString(o.Go().String())
}

type RegexpStrsResult []string

func (o RegexpStrsResult) IsFalsy() bool {
	return len(o) == 0
}

func (o RegexpStrsResult) Type() ObjectType {
	return TRegexpStrsResult
}

func (o RegexpStrsResult) ToArray() Array {
	var arr = make(Array, len(o))
	for i, value := range o {
		arr[i] = Str(value)
	}
	return arr
}

func (o RegexpStrsResult) ToString() string {
	return o.ToArray().ToString()
}

func (o RegexpStrsResult) Equal(right Object) bool {
	switch t := right.(type) {
	case RegexpStrsResult:
		return o.ToArray().Equal(t.ToArray())
	default:
		return false
	}
}

func (o RegexpStrsResult) Print(state *PrinterState) error {
	return state.WithoutRepr(func(s *PrinterState) error {
		defer state.WrapRepr(o)()
		defer state.options.Backup(PrintStateOptionQuoteStr)
		state.options.WithQuoteStr()
		return o.ToArray().PrintObject(state, o)
	})
}

// regexpResultIndex resolves an index Object against a result of length n,
// supporting negative indices (from the end) for Int, like Array.
func regexpResultIndex(index Object, n int) (int, error) {
	switch v := index.(type) {
	case Int:
		idx := int(v)
		if idx < 0 {
			idx = n + idx
		}
		if idx >= 0 && idx < n {
			return idx, nil
		}
		return 0, ErrIndexOutOfBounds
	case Uint:
		idx := int(v)
		if idx >= 0 && idx < n {
			return idx, nil
		}
		return 0, ErrIndexOutOfBounds
	default:
		return 0, NewIndexTypeError("int|uint", index.Type().Name())
	}
}

// Length returns the number of submatches (the full match plus capture groups).
func (o RegexpStrsResult) Length() int { return len(o) }

// IndexGet returns submatch i (i == 0 is the full match, i >= 1 are the capture
// groups), so groups are accessed like an array.
func (o RegexpStrsResult) IndexGet(_ *VM, index Object) (Object, error) {
	i, err := regexpResultIndex(index, len(o))
	if err != nil {
		return nil, err
	}
	return Str(o[i]), nil
}

func (o RegexpStrsResult) Iterate(_ *VM, na *NamedArgs) Iterator {
	return SliceIteration(TArrayIterator, o, []string(o), func(e *KeyValue, i Int, v string) error {
		e.K = i
		e.V = Str(v)
		return nil
	}).ParseNamedArgs(na)
}

type RegexpStrsSliceResult [][]string

func (o RegexpStrsSliceResult) IsFalsy() bool {
	return len(o) == 0
}

func (o RegexpStrsSliceResult) Type() ObjectType {
	return TRegexpStrsSliceResult
}

func (o RegexpStrsSliceResult) ToArray() Array {
	var arr = make(Array, len(o))
	for i, values := range o {
		arr[i] = RegexpStrsResult(values).ToArray()
	}
	return arr
}

func (o RegexpStrsSliceResult) ToString() string {
	return o.ToArray().ToString()
}

func (o RegexpStrsSliceResult) Equal(right Object) bool {
	switch t := right.(type) {
	case RegexpStrsSliceResult:
		return o.ToArray().Equal(t.ToArray())
	default:
		return false
	}
}

func (o RegexpStrsSliceResult) Print(state *PrinterState) error {
	return state.WithoutRepr(func(s *PrinterState) error {
		defer state.WrapRepr(o)()
		defer state.options.Backup(PrintStateOptionQuoteStr)
		state.options.WithQuoteStr()
		return o.ToArray().PrintObject(state, o)
	})
}

// Length returns the number of matches.
func (o RegexpStrsSliceResult) Length() int { return len(o) }

// IndexGet returns match i, itself a submatch list (full match + groups).
func (o RegexpStrsSliceResult) IndexGet(_ *VM, index Object) (Object, error) {
	i, err := regexpResultIndex(index, len(o))
	if err != nil {
		return nil, err
	}
	return RegexpStrsResult(o[i]), nil
}

func (o RegexpStrsSliceResult) Iterate(_ *VM, na *NamedArgs) Iterator {
	return SliceIteration(TArrayIterator, o, [][]string(o), func(e *KeyValue, i Int, v []string) error {
		e.K = i
		e.V = RegexpStrsResult(v)
		return nil
	}).ParseNamedArgs(na)
}

type RegexpBytesResult [][]byte

func (o RegexpBytesResult) IsFalsy() bool {
	return len(o) == 0
}

func (o RegexpBytesResult) Type() ObjectType {
	return TRegexpBytesResult
}

func (o RegexpBytesResult) ToArray() Array {
	var arr = make(Array, len(o))
	for i, value := range o {
		arr[i] = Bytes(value)
	}
	return arr
}

func (o RegexpBytesResult) ToString() string {
	return o.ToArray().ToString()
}

func (o RegexpBytesResult) Equal(right Object) bool {
	switch t := right.(type) {
	case RegexpBytesResult:
		return o.ToArray().Equal(t.ToArray())
	default:
		return false
	}
}

func (o RegexpBytesResult) Print(state *PrinterState) error {
	return state.WithoutRepr(func(s *PrinterState) error {
		defer state.WrapRepr(o)()
		defer state.options.Backup(PrintStateOptionBytesToHex)
		state.options.WithBytesToHex()
		return o.ToArray().PrintObject(state, o)
	})
}

// Length returns the number of submatches (the full match plus capture groups).
func (o RegexpBytesResult) Length() int { return len(o) }

// IndexGet returns submatch i as bytes (i == 0 is the full match).
func (o RegexpBytesResult) IndexGet(_ *VM, index Object) (Object, error) {
	i, err := regexpResultIndex(index, len(o))
	if err != nil {
		return nil, err
	}
	return Bytes(o[i]), nil
}

func (o RegexpBytesResult) Iterate(_ *VM, na *NamedArgs) Iterator {
	return SliceIteration(TArrayIterator, o, [][]byte(o), func(e *KeyValue, i Int, v []byte) error {
		e.K = i
		e.V = Bytes(v)
		return nil
	}).ParseNamedArgs(na)
}

type RegexpBytesSliceResult [][][]byte

func (o RegexpBytesSliceResult) IsFalsy() bool {
	return len(o) == 0
}

func (o RegexpBytesSliceResult) Type() ObjectType {
	return TRegexpBytesSliceResult
}

func (o RegexpBytesSliceResult) ToArray() Array {
	var arr = make(Array, len(o))
	for i, values := range o {
		arr[i] = RegexpBytesResult(values).ToArray()
	}
	return arr
}

func (o RegexpBytesSliceResult) ToString() string {
	return o.ToArray().ToString()
}

func (o RegexpBytesSliceResult) Equal(right Object) bool {
	switch t := right.(type) {
	case RegexpBytesResult:
		return o.ToArray().Equal(t.ToArray())
	default:
		return false
	}
}

func (o RegexpBytesSliceResult) Print(state *PrinterState) error {
	return state.WithoutRepr(func(s *PrinterState) error {
		defer state.WrapRepr(o)()
		defer state.options.Backup(PrintStateOptionBytesToHex)
		state.options.WithBytesToHex()
		return o.ToArray().PrintObject(state, o)
	})
}

// Length returns the number of matches.
func (o RegexpBytesSliceResult) Length() int { return len(o) }

// IndexGet returns match i, itself a submatch list (full match + groups).
func (o RegexpBytesSliceResult) IndexGet(_ *VM, index Object) (Object, error) {
	i, err := regexpResultIndex(index, len(o))
	if err != nil {
		return nil, err
	}
	return RegexpBytesResult(o[i]), nil
}

func (o RegexpBytesSliceResult) Iterate(_ *VM, na *NamedArgs) Iterator {
	return SliceIteration(TArrayIterator, o, [][][]byte(o), func(e *KeyValue, i Int, v [][]byte) error {
		e.K = i
		e.V = RegexpBytesResult(v)
		return nil
	}).ParseNamedArgs(na)
}
