package os

import (
	"io"
	"os"
	"reflect"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/helper"
	cmdu "github.com/unapu-go/cmd-utils"
)

func NewFileMode(c gad.Call) (_ gad.Object, err error) {
	if err = c.Args.CheckMaxLen(1); err != nil {
		return
	}
	if c.Args.Length() == 0 {
		return FileFlag(0), nil
	}
	switch arg := c.Args.GetOnly(0).(type) {
	case gad.Int:
		return FileFlag(arg), nil
	case gad.Uint:
		return FileFlag(arg), nil
	case gad.Str:
		var m FileFlag
		m.Parse(arg.ToString())
		return m, nil
	default:
		return nil, gad.NewArgumentTypeErrorT("0st", arg.Type(), gad.TInt, gad.TUint, gad.TStr)
	}
}

func Exec(c gad.Call) (o gad.Object, err error) {
	var (
		naio = c.NamedArgs.GetValueOrNil("io")
		typ  = gad.NewReflectType(reflect.TypeOf(cmdu.CmdBuilder{}))
	)

	if o, err = typ.Call(c); err != nil {
		return
	}

	var (
		Cmd     *cmdu.Cmd
		builder = o.(gad.ReflectValuer).ToInterface().(cmdu.CmdBuilder)
	)

	o = gad.Nil

	if Cmd, err = builder.Build(nil); err != nil {
		return
	}

	if naio != nil {
		var values gad.KeyValueArray

		if values, err = values.AppendObject(naio); err != nil {
			return
		}

		var (
			na     = gad.NewNamedArgs(values)
			stdin  = helper.NamedArgOfReader("stdin")
			stdout = helper.NamedArgOfWriter("stdout")
			stderr = helper.NamedArgOfWriter("stderr")
		)

		if err = na.Get(stdin, stdout, stderr); err != nil {
			return
		}

		if stdin.Value != nil {
			Cmd.Stdin = stdin.Value.(gad.Reader).GoReader()
		}
		if stdout.Value != nil {
			Cmd.Stdout = stdout.Value.(gad.Writer).GoWriter()
		}
		if stdout.Value != nil {
			Cmd.Stderr = stdout.Value.(gad.Writer).GoWriter()
		}
	}

	if err = Cmd.StartContext(c.VM.Context); err != nil {
		return
	}
	o, _ = gad.NewReflectValue(Cmd)
	err = Cmd.Wait()
	return
}

func Exists(c gad.Call) (o gad.Object, err error) {
	pth := &gad.Arg{
		Name:          "path",
		TypeAssertion: gad.TypeAssertionFromTypes(gad.TStr),
	}
	if err = c.Args.Destructure(pth); err != nil {
		return
	}

	if _, err = os.Stat(pth.Value.ToString()); err != nil {
		if os.IsNotExist(err) {
			return gad.False, nil
		}
		return
	}
	return gad.True, nil
}

func CreateFile(c gad.Call) (o gad.Object, err error) {
	var (
		pth = &gad.Arg{
			Name:          "path",
			TypeAssertion: gad.TypeAssertionFromTypes(gad.TStr),
		}

		perm = &gad.NamedArgVar{
			Name:          "perm",
			TypeAssertion: gad.TypeAssertionFromTypes(gad.TInt),
			Value:         gad.Int(0),
		}

		data = helper.NamedArgOfReader("data")

		closes = &gad.NamedArgVar{
			Name:          "close",
			TypeAssertion: gad.TypeAssertionFlag(),
			Value:         gad.No,
		}

		f *os.File
	)

	if err = c.Args.Destructure(pth); err != nil {
		return
	}
	if err = c.NamedArgs.Get(perm, data, closes); err != nil {
		return
	}

	if mode := perm.Value.(gad.Int); mode > 0 {
		f, err = os.OpenFile(pth.Value.ToString(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(mode))
	} else {
		f, err = os.Create(pth.Value.ToString())
	}

	if err != nil {
		return
	}

	if !closes.Value.IsFalsy() {
		defer f.Close()
	}

	o = gad.MustNewReflectValue(f)

	if data.Value != nil {
		if _, err = c.VM.Builtins.Call(gad.BuiltinCopy, gad.Call{
			VM:   c.VM,
			Args: gad.Args{gad.Array{o, data.Value}},
		}); err != nil {
			return
		}
	}

	return
}

func OpenFile(c gad.Call) (o gad.Object, err error) {
	var (
		pth = &gad.Arg{
			Name:          "path",
			TypeAssertion: gad.TypeAssertionFromTypes(gad.TStr),
		}

		flag = &gad.NamedArgVar{
			Name:          "flag",
			TypeAssertion: gad.TypeAssertionFromTypes(TFileFlag),
			Value:         OReadOnly,
		}

		perm = &gad.NamedArgVar{
			Name:          "perm",
			TypeAssertion: gad.TypeAssertionFromTypes(gad.TInt),
			Value:         gad.Int(0),
		}

		f *os.File
	)

	if err = c.Args.Destructure(pth); err != nil {
		return
	}
	if err = c.NamedArgs.Get(flag, perm); err != nil {
		return
	}

	f, err = os.OpenFile(pth.Value.ToString(), int(flag.Value.(FileFlag)), os.FileMode(perm.Value.(gad.Int)))

	if err != nil {
		return
	}

	o = gad.MustNewReflectValue(f)

	return
}

func ReadFile(c gad.Call) (o gad.Object, err error) {
	var (
		pth = &gad.Arg{
			Name:          "path",
			TypeAssertion: gad.TypeAssertionFromTypes(gad.TStr),
		}

		f    *os.File
		data []byte
	)

	if err = c.Args.Destructure(pth); err != nil {
		return
	}

	if f, err = os.Open(pth.Value.ToString()); err != nil {
		return
	}

	defer f.Close()

	if data, err = io.ReadAll(f); err != nil {
		return
	}

	o = gad.Bytes(data)
	return
}
