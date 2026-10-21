package gad

import "fmt"

// FmtArg is o as an argument of Go's fmt (sprintf, the fmt module): itself
// when it formats itself (a fmt.Formatter: int, uint, float, decimal, char,
// str, bool), else its gad text — `%v`/`%s` its str, `%q` the str quoted,
// `%#v` its repr —, never the Go value inside it (`&{1 3 <nil>}`).
func FmtArg(vm *VM, o Object) any {
	if _, ok := o.(fmt.Formatter); ok {
		return o
	}
	return fmtObject{vm: vm, o: o}
}

// FmtArgs are the objects as arguments of Go's fmt (FmtArg).
func FmtArgs(vm *VM, objects ...Object) []any {
	out := make([]any, len(objects))
	for i, o := range objects {
		out[i] = FmtArg(vm, o)
	}
	return out
}

// fmtObject formats an object by its gad text.
type fmtObject struct {
	vm *VM
	o  Object
}

func (f fmtObject) Format(s fmt.State, verb rune) {
	var (
		text Str
		err  error
	)
	if verb == 'v' && s.Flag('#') {
		text, err = ToRepr(f.vm, f.o)
		verb = 's'
	} else {
		text, err = ToStr(f.vm, f.o)
	}
	if err != nil {
		fmt.Fprintf(s, "%%!%c(%s)", verb, err)
		return
	}
	if verb == 'v' {
		verb = 's'
	}
	// the flags and the width of the verb apply to the text (`%-10v`, `%q`);
	// a verb of no text (`%d`) is Go's error for a string
	fmt.Fprintf(s, fmt.FormatString(s, verb), string(text))
}
