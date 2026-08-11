// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import (
	"fmt"
	"reflect"
	"strconv"
)

// fmtModuleSpec is the module spec shared by the builtin `fmt` namespace members
// and the importable fmt module.
var fmtModuleSpec = NewModuleSpecFromName("fmt")

// FmtModule returns the `fmt` builtin namespace (Print/Printf/Sprint/Scan…). It
// is also used by the stdlib `fmt` importable module.
func FmtModule() StdModuleData { return newFmtModule() }

// newFmtModule builds the `fmt` builtin namespace (Go's fmt: print/scan).
func newFmtModule() StdModuleData {
	return StdModuleData{Funcs: Dict{
		// gad:doc
		// # fmt module
		// gad:samples [module,auto] stdlib/fmt/samples.gad
		//
		// ## Scan Examples
		//
		// A scan function returns the number of items scanned and **throws** on a
		// scan error, so handle failures with `try`/`catch`. Every arg scanned
		// before the error is still populated (a partial scan).
		//
		// ```gad ignore
		// arg1 := fmt.scanArg("str")
		// arg2 := fmt.scanArg("int")
		// n := fmt.sscanf("abc123", "%3s%d", arg1, arg2)
		// fmt.println(n)              // 2, number of scanned items
		// fmt.println(arg1.Value)     // abc
		// fmt.println(bool(arg1))     // true, reports whether arg1 is scanned
		// fmt.println(arg2.Value)     // 123
		// fmt.println(bool(arg2))     // true, reports whether arg2 is scanned
		// ```
		//
		// ```gad ignore
		// arg1 := fmt.scanArg("str")
		// arg2 := fmt.scanArg("int")
		// arg3 := fmt.scanArg("float")
		// try {
		//   fmt.sscanf("abc 123", "%s%d%f", arg1, arg2, arg3)  // throws: EOF
		// } catch err {
		//   fmt.println(err)            // the scan error
		// }
		// // the partial scan is still readable on the args
		// fmt.println(arg1.Value)  // abc
		// fmt.println(bool(arg1))  // true
		// fmt.println(arg2.Value)  // 123
		// fmt.println(bool(arg2))  // true
		// fmt.println(arg3.Value)  // nil
		// fmt.println(bool(arg3))  // false, not scanned
		// ```

		// gad:doc
		// ## Functions
		// print(*any) <int>
		// Formats using the default formats for its operands and writes to standard
		// output. Spaces are added between operands when neither is a str.
		// It returns the number of bytes written and any encountered write error
		// throws a runtime error.
		"print": &BuiltinFunction{
			FuncName: "print",
			Value:    fmtNewPrint(fmt.Print),
		},
		// gad:doc
		// printf(format str, *any) <int>
		// Formats according to a format specifier and writes to standard output.
		// It returns the number of bytes written and any encountered write error
		// throws a runtime error.
		"printf": &BuiltinFunction{
			FuncName: "printf",
			Value:    fmtNewPrintf(fmt.Printf),
		},
		// gad:doc
		// println(*any) <int>
		// Formats using the default formats for its operands and writes to standard
		// output. Spaces are always added between operands and a newline
		// is appended. It returns the number of bytes written and any encountered
		// write error throws a runtime error.
		"println": &BuiltinFunction{
			FuncName: "println",
			Value:    fmtNewPrint(fmt.Println),
		},
		// gad:doc
		// sprint(*any) <str>
		// Formats using the default formats for its operands and returns the
		// resulting str. Spaces are added between operands when neither is a
		// str.
		"sprint": &BuiltinFunction{
			FuncName: "sprint",
			Value:    fmtNewSprint(fmt.Sprint),
		},
		// gad:doc
		// sprintf(format str, *any) <str>
		// Formats according to a format specifier and returns the resulting str.
		"sprintf": &BuiltinFunction{
			FuncName: "sprintf",
			Value:    fmtNewSprintf(fmt.Sprintf),
		},
		// gad:doc
		// sprintln(*any) <str>
		// Formats using the default formats for its operands and returns the
		// resulting str. Spaces are always added between operands and a newline
		// is appended.
		"sprintln": &BuiltinFunction{
			FuncName: "sprintln",
			Value:    fmtNewSprint(fmt.Sprintln),
		},
		// gad:doc
		// sscan(str str, *scanArg) <int>
		// Scans the argument str, storing successive space-separated values into
		// successive scanArg arguments. Newlines count as space. If no error is
		// encountered, it returns the number of items successfully scanned. If that
		// is less than the number of arguments, a scan error is thrown.
		"sscan": &BuiltinFunction{
			FuncName: "sscan",
			Value:    fmtNewSscan(fmt.Sscan),
		},
		// gad:doc
		// sscanf(str str, format str, *scanArg) <int>
		// Scans the argument str, storing successive space-separated values into
		// successive scanArg arguments as determined by the format. It returns the
		// number of items successfully parsed, and throws on a scan error.
		// Newlines in the input must match newlines in the format.
		"sscanf": &BuiltinFunction{
			FuncName: "sscanf",
			Value:    fmtNewSscanf(fmt.Sscanf),
		},
		// sscanln(str str, *scanArg) <int>
		// Sscanln is similar to Sscan, but stops scanning at a newline and after
		// the final item there must be a newline or EOF. It returns the number of
		// items successfully parsed, and throws on a scan error.
		"sscanln": &BuiltinFunction{
			FuncName: "sscanln",
			Value:    fmtNewSscan(fmt.Sscanln),
		},
		// gad:doc
		// scanArg(typeName str) <scanArg>
		// Returns a `scanArg` object to scan a value of given type name in scan
		// functions.
		// Supported type names are `"str", "int", "uint", "float", "char",
		// "bool", "bytes"`.
		// It throws a runtime error if type name is not supported.
		// Alternatively, `str, int, uint, float, char, bool, bytes` builtin
		// functions can be provided to get the type name from the BuiltinFunction's
		// Literal field.
		"scanArg": &BuiltinFunction{
			FuncName: "scanArg",
			Value:    fmtNewScanArgFunc,
		},
	}}
}

func fmtNewPrint(fn func(...any) (int, error)) CallableFunc {
	return func(c Call) (ret Object, err error) {
		vargs := fmtToPrintArgs(0, c)
		n, err := fn(vargs...)
		return Int(n), err
	}
}

func fmtNewPrintf(fn func(string, ...any) (int, error)) CallableFunc {
	return func(c Call) (ret Object, err error) {
		if c.Args.Length() < 1 {
			return Nil, ErrWrongNumArguments.NewError(
				"want>=1 got=" + strconv.Itoa(c.Args.Length()))
		}
		vargs := fmtToPrintArgs(1, c)
		n, err := fn(c.Args.Get(0).ToString(), vargs...)
		return Int(n), err
	}
}

func fmtNewSprint(fn func(...any) string) CallableFunc {
	return func(c Call) (ret Object, err error) {
		vargs := fmtToPrintArgs(0, c)
		return Str(fn(vargs...)), nil
	}
}

func fmtNewSprintf(fn func(string, ...any) string) CallableFunc {
	return func(c Call) (ret Object, err error) {
		if c.Args.Length() < 1 {
			return Nil, ErrWrongNumArguments.NewError(
				"want>=1 got=" + strconv.Itoa(c.Args.Length()))
		}
		vargs := fmtToPrintArgs(1, c)
		return Str(fn(c.Args.Get(0).ToString(), vargs...)), nil
	}
}

func fmtNewSscan(fn func(string, ...any) (int, error)) CallableFunc {
	return func(c Call) (ret Object, err error) {
		if c.Args.Length() < 2 {
			return Nil, ErrWrongNumArguments.NewError(
				"want>=2 got=" + strconv.Itoa(c.Args.Length()))
		}
		vargs, err := fmtToScanArgs(1, c)
		if err != nil {
			return Nil, err
		}
		n, err := fn(c.Args.Get(0).ToString(), vargs...)
		return fmtPostScan(1, n, err, c)
	}
}

func fmtNewSscanf(
	fn func(string, string, ...any) (int, error),
) CallableFunc {
	return func(c Call) (ret Object, err error) {
		if c.Args.Length() < 3 {
			return Nil, ErrWrongNumArguments.NewError(
				"want>=3 got=" + strconv.Itoa(c.Args.Length()))
		}
		vargs, err := fmtToScanArgs(2, c)
		if err != nil {
			return Nil, err
		}
		n, err := fn(c.Args.Get(0).ToString(), c.Args.Get(1).ToString(), vargs...)
		return fmtPostScan(2, n, err, c)
	}
}

func fmtToScanArgs(offset int, c Call) ([]any, error) {
	size := c.Args.Length()
	vargs := make([]any, 0, size-offset)
	for i := offset; i < size; i++ {
		v, ok := c.Args.Get(i).(FmtScanArg)
		if !ok {
			return nil, NewArgumentTypeError(strconv.Itoa(i),
				"ScanArg interface", c.Args.Get(i).Type().Name())
		}
		v.Set(false)
		vargs = append(vargs, v.Arg())
	}
	return vargs, nil
}

func fmtToPrintArgs(offset int, c Call) []any {
	size := c.Args.Length()
	vargs := make([]any, 0, size-offset)
	for i := offset; i < size; i++ {
		vargs = append(vargs, c.Args.Get(i))
	}
	return vargs
}

// args are always of scanArg interface type.
// fmtPostScan marks the successfully-scanned args (so their .Value/bool reflect
// the partial scan) and returns the number scanned. A scan error is propagated
// (thrown in the VM) rather than returned as a value, so callers handle it with
// try/catch; the partial results remain readable on the scan args.
func fmtPostScan(offset, n int, err error, c Call) (Object, error) {
	for i := offset; i < n+offset; i++ {
		c.Args.Get(i).(FmtScanArg).Set(true)
	}
	if err != nil {
		if s := reflect.TypeOf(err).String(); s == "*errors.errorString" {
			return Nil, &Error{Message: err.Error()}
		}
		return Nil, &Error{Message: err.Error(), Cause: err}
	}
	return Int(n), nil
}
