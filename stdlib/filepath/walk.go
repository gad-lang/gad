package filepath

import (
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/gad-lang/gad"
)

const (
	WalkSkipNone WalkSkip = iota
	WalkSkipDir
	WalkSkipAll
)

type WalkSkip uint64

func (f WalkSkip) IsFalsy() bool {
	return f == 0
}

func (f WalkSkip) Type() gad.ObjectType {
	return TWalkSkip
}

func (f WalkSkip) ToString() string {
	return f.String()
}

func (f WalkSkip) Equal(right gad.Object) bool {
	if r, ok := right.(WalkSkip); ok {
		return f == r
	}
	return false
}

func (f *WalkSkip) Set(flag WalkSkip) *WalkSkip    { *f = *f | flag; return f }
func (f *WalkSkip) Clear(flag WalkSkip) *WalkSkip  { *f = *f &^ flag; return f }
func (f *WalkSkip) Toggle(flag WalkSkip) *WalkSkip { *f = *f ^ flag; return f }
func (f WalkSkip) Has(flag WalkSkip) bool          { return f&flag != 0 }
func (f WalkSkip) String() string {
	switch f {
	case WalkSkipDir:
		return "dir"
	case WalkSkipAll:
		return "all"
	default:
		return "none"
	}
}

func (f *WalkSkip) Parse(str string) {
	switch str {
	case "dir":
		*f = WalkSkipDir
	case "all":
		*f = WalkSkipAll
	default:
		*f = 0
	}
}

func Walk(c gad.Call) (_ gad.Object, err error) {
	var (
		pth = &gad.Arg{
			Name:          "path",
			TypeAssertion: gad.TypeAssertionFromTypes(gad.TStr),
		}

		callback = &gad.Arg{
			Name: "callback",
			TypeAssertion: gad.NewTypeAssertion(gad.TypeAssertionHandlers{
				"callable": gad.Callable,
			}),
		}

		relative = &gad.NamedArgVar{
			Name:  "relative",
			Value: gad.No,
		}

		dotSkip = &gad.NamedArgVar{
			Name:  "dotSkip",
			Value: gad.No,
		}
	)

	if err = c.Args.Destructure(pth, callback); err != nil {
		return
	}

	if err = c.NamedArgs.Get(relative, dotSkip); err != nil {
		return
	}

	var (
		basePath   = filepath.Clean(string(pth.Value.(gad.Str)))
		args       = gad.Array{gad.Nil, gad.Nil, gad.Nil}
		caller     gad.VMCaller
		ret        gad.Object
		formatPath = func(pth *string) {}
	)

	if caller, err = gad.NewInvoker(c.VM, callback.Value).Caller(gad.Args{args}, &c.NamedArgs); err != nil {
		return
	}

	if !relative.Value.IsFalsy() {
		formatPath = func(pth *string) {
			*pth = strings.TrimPrefix(*pth, basePath)
			if *pth == "" {
				*pth = "."
			} else {
				*pth = (*pth)[1:]
			}
		}
	}

	first := true

	return gad.Nil, filepath.Walk(basePath, func(path string, info fs.FileInfo, err error) (err2 error) {
		if first {
			first = false
			if !dotSkip.Value.IsFalsy() {
				return nil
			}
		}

		formatPath(&path)

		args[0], args[1], args[2] = gad.Str(path), gad.MustNewReflectValue(info), gad.Nil
		if err != nil {
			args[2] = gad.WrapError(err)
		}
		if ret, err2 = caller.Call(); err2 != nil {
			return
		}
		if mode, ok := ret.(WalkSkip); ok {
			switch mode {
			case WalkSkipDir:
				return filepath.SkipDir
			case WalkSkipAll:
				return filepath.SkipAll
			}
		}
		return
	})
}

func NewSkipMode(c gad.Call) (_ gad.Object, err error) {
	if err = c.Args.CheckMaxLen(1); err != nil {
		return
	}
	if c.Args.Length() == 0 {
		return WalkSkip(0), nil
	}
	switch arg := c.Args.GetOnly(0).(type) {
	case gad.Int:
		return WalkSkip(arg), nil
	case gad.Uint:
		return WalkSkip(arg), nil
	case gad.Str:
		var m WalkSkip
		m.Parse(arg.ToString())
		return m, nil
	default:
		return nil, gad.NewArgumentTypeErrorT("0st", arg.Type(), gad.TInt, gad.TUint, gad.TStr)
	}
}

var TWalkSkip = &gad.Type{
	Parent:   gad.TInt,
	TypeName: "WalkSkip",
	Constructor: &gad.Function{
		Value: NewSkipMode,
	},
	Static: gad.Dict{
		"None": WalkSkipNone,
		"Dir":  WalkSkipDir,
		"All":  WalkSkipAll,
	},
}
