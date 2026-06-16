// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import (
	"fmt"
	"reflect"
	"strconv"
)

// FmtModule returns the `fmt` builtin namespace (Print/Printf/Sprint/Scan…). It
// is also used by the stdlib `fmt` importable module.
func FmtModule() Dict { return newFmtModule() }

// newFmtModule builds the `fmt` builtin namespace (Go's fmt: print/scan).
func newFmtModule() Dict {
	return Dict{
		// gad:doc
		// # fmt module
		//
		// ## Scan Examples
		//
		// ```go
		// arg1 := fmt.ScanArg("str")
		// arg2 := fmt.ScanArg("int")
		// ret := fmt.Sscanf("abc123", "%3s%d", arg1, arg2)
		// if isError(ret) {
		//   // handle error
		//   fmt.Println(err)
		// } else {
		//   fmt.Println(ret)            // 2, number of scanned items
		//   fmt.Println(arg1.Value)     // abc
		//   fmt.Println(bool(arg1))     // true, reports whether arg1 is scanned
		//   fmt.Println(arg2.Value)     // 123
		//   fmt.Println(bool(arg2))     // true, reports whether arg2 is scanned
		// }
		// ```
		//
		// ```go
		// arg1 = fmt.ScanArg("str")
		// arg2 = fmt.ScanArg("int")
		// arg3 = fmt.ScanArg("float")
		// ret = fmt.Sscanf("abc 123", "%s%d%f", arg1, arg2, arg3)
		// fmt.Println(ret)         // error: EOF
		// fmt.Println(arg1.Value)  // abc
		// fmt.Println(bool(arg1))  // true
		// fmt.Println(arg2.Value)  // 123
		// fmt.Println(bool(arg2))  // true
		// fmt.Println(arg3.Value)  // nil
		// fmt.Println(bool(arg2))  // false, not scanned
		//
		// // Use if statement or a ternary expression to get the scanned value or a default value.
		// v := arg1 ? arg1.Value : "default value"
		// ```

		// gad:doc
		// ## Functions
		// Print(*any) <int>
		// Formats using the default formats for its operands and writes to standard
		// output. Spaces are added between operands when neither is a str.
		// It returns the number of bytes written and any encountered write error
		// throws a runtime error.
		"Print": &BuiltinFunction{
			FuncName: "Print",
			Value:    fmtNewPrint(fmt.Print),
		},
		// gad:doc
		// Printf(format str, *any) <int>
		// Formats according to a format specifier and writes to standard output.
		// It returns the number of bytes written and any encountered write error
		// throws a runtime error.
		"Printf": &BuiltinFunction{
			FuncName: "Printf",
			Value:    fmtNewPrintf(fmt.Printf),
		},
		// gad:doc
		// Println(*any) <int>
		// Formats using the default formats for its operands and writes to standard
		// output. Spaces are always added between operands and a newline
		// is appended. It returns the number of bytes written and any encountered
		// write error throws a runtime error.
		"Println": &BuiltinFunction{
			FuncName: "Println",
			Value:    fmtNewPrint(fmt.Println),
		},
		// gad:doc
		// Sprint(*any) <str>
		// Formats using the default formats for its operands and returns the
		// resulting str. Spaces are added between operands when neither is a
		// str.
		"Sprint": &BuiltinFunction{
			FuncName: "Sprint",
			Value:    fmtNewSprint(fmt.Sprint),
		},
		// gad:doc
		// Sprintf(format str, *any) <str>
		// Formats according to a format specifier and returns the resulting str.
		"Sprintf": &BuiltinFunction{
			FuncName: "Sprintf",
			Value:    fmtNewSprintf(fmt.Sprintf),
		},
		// gad:doc
		// Sprintln(*any) <str>
		// Formats using the default formats for its operands and returns the
		// resulting str. Spaces are always added between operands and a newline
		// is appended.
		"Sprintln": &BuiltinFunction{
			FuncName: "Sprintln",
			Value:    fmtNewSprint(fmt.Sprintln),
		},
		// gad:doc
		// Sscan(str str, ScanArg[, *ScanArg]) <int | error>
		// Scans the argument str, storing successive space-separated values into
		// successive ScanArg arguments. Newlines count as space. If no error is
		// encountered, it returns the number of items successfully scanned. If that
		// is less than the number of arguments, error will report why.
		"Sscan": &BuiltinFunction{
			FuncName: "Sscan",
			Value:    fmtNewSscan(fmt.Sscan),
		},
		// gad:doc
		// Sscanf(str str, format str, ScanArg[, *ScanArg]) <int | error>
		// Scans the argument str, storing successive space-separated values into
		// successive ScanArg arguments as determined by the format. It returns the
		// number of items successfully parsed or an error.
		// Newlines in the input must match newlines in the format.
		"Sscanf": &BuiltinFunction{
			FuncName: "Sscanf",
			Value:    fmtNewSscanf(fmt.Sscanf),
		},
		// Sscanln(str str, ScanArg[, *ScanArg]) <int | error>
		// Sscanln is similar to Sscan, but stops scanning at a newline and after
		// the final item there must be a newline or EOF. It returns the number of
		// items successfully parsed or an error.
		"Sscanln": &BuiltinFunction{
			FuncName: "Sscanln",
			Value:    fmtNewSscan(fmt.Sscanln),
		},
		// gad:doc
		// ScanArg(typeName str) <scanArg>
		// Returns a `scanArg` object to scan a value of given type name in scan
		// functions.
		// Supported type names are `"str", "int", "uint", "float", "char",
		// "bool", "bytes"`.
		// It throws a runtime error if type name is not supported.
		// Alternatively, `str, int, uint, float, char, bool, bytes` builtin
		// functions can be provided to get the type name from the BuiltinFunction's
		// Literal field.
		"ScanArg": &BuiltinFunction{
			FuncName: "ScanArg",
			Value:    fmtNewScanArgFunc,
		},
	}
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
		return fmtPostScan(1, n, err, c), nil
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
		return fmtPostScan(2, n, err, c), nil
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

// args are always of ScanArg interface type.
func fmtPostScan(offset, n int, err error, c Call) Object {
	for i := offset; i < n+offset; i++ {
		c.Args.Get(i).(FmtScanArg).Set(true)
	}
	if err != nil {
		if s := reflect.TypeOf(err).String(); s == "*errors.errorString" {
			return &Error{
				Message: err.Error(),
			}
		}
		return &Error{
			Message: err.Error(),
			Cause:   err,
		}
	}
	return Int(n)
}
