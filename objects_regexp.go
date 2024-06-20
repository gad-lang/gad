package gad

import (
	"regexp"

	"github.com/gad-lang/gad/token"
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
	}

	return nil, ErrInvalidIndex.NewError(name)
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
