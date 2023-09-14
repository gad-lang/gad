// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package fmt

import (
	"fmt"
	"strconv"

	"github.com/gad-lang/gad"
)

// Module represents fmt module.
var Module = map[string]gad.Object{
	// gad:doc
	// # fmt Module
	//
	// ## Scan Examples
	//
	// ```go
	// arg1 := fmt.ScanArg("string")
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
	// arg1 = fmt.ScanArg("string")
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
	// Print(...any) -> int
	// Formats using the default formats for its operands and writes to standard
	// output. Spaces are added between operands when neither is a string.
	// It returns the number of bytes written and any encountered write error
	// throws a runtime error.
	"Print": &gad.Function{
		Name:  "Print",
		Value: newPrint(fmt.Print),
	},
	// gad:doc
	// Printf(format string, ...any) -> int
	// Formats according to a format specifier and writes to standard output.
	// It returns the number of bytes written and any encountered write error
	// throws a runtime error.
	"Printf": &gad.Function{
		Name:  "Printf",
		Value: newPrintf(fmt.Printf),
	},
	// gad:doc
	// Println(...any) -> int
	// Formats using the default formats for its operands and writes to standard
	// output. Spaces are always added between operands and a newline
	// is appended. It returns the number of bytes written and any encountered
	// write error throws a runtime error.
	"Println": &gad.Function{
		Name:  "Println",
		Value: newPrint(fmt.Println),
	},
	// gad:doc
	// Sprint(...any) -> string
	// Formats using the default formats for its operands and returns the
	// resulting string. Spaces are added between operands when neither is a
	// string.
	"Sprint": &gad.Function{
		Name:  "Sprint",
		Value: newSprint(fmt.Sprint),
	},
	// gad:doc
	// Sprintf(format string, ...any) -> string
	// Formats according to a format specifier and returns the resulting string.
	"Sprintf": &gad.Function{
		Name:  "Sprintf",
		Value: newSprintf(fmt.Sprintf),
	},
	// gad:doc
	// Sprintln(...any) -> string
	// Formats using the default formats for its operands and returns the
	// resulting string. Spaces are always added between operands and a newline
	// is appended.
	"Sprintln": &gad.Function{
		Name:  "Sprintln",
		Value: newSprint(fmt.Sprintln),
	},
	// gad:doc
	// Sscan(str string, ScanArg[, ...ScanArg]) -> int | error
	// Scans the argument string, storing successive space-separated values into
	// successive ScanArg arguments. Newlines count as space. If no error is
	// encountered, it returns the number of items successfully scanned. If that
	// is less than the number of arguments, error will report why.
	"Sscan": &gad.Function{
		Name:  "Sscan",
		Value: newSscan(fmt.Sscan),
	},
	// gad:doc
	// Sscanf(str string, format string, ScanArg[, ...ScanArg]) -> int | error
	// Scans the argument string, storing successive space-separated values into
	// successive ScanArg arguments as determined by the format. It returns the
	// number of items successfully parsed or an error.
	// Newlines in the input must match newlines in the format.
	"Sscanf": &gad.Function{
		Name:  "Sscanf",
		Value: newSscanf(fmt.Sscanf),
	},
	// Sscanln(str string, ScanArg[, ...ScanArg]) -> int | error
	// Sscanln is similar to Sscan, but stops scanning at a newline and after
	// the final item there must be a newline or EOF. It returns the number of
	// items successfully parsed or an error.
	"Sscanln": &gad.Function{
		Name:  "Sscanln",
		Value: newSscan(fmt.Sscanln),
	},
	// gad:doc
	// ScanArg(typeName string) -> scanArg
	// Returns a `scanArg` object to scan a value of given type name in scan
	// functions.
	// Supported type names are `"string", "int", "uint", "float", "char",
	// "bool", "bytes"`.
	// It throws a runtime error if type name is not supported.
	// Alternatively, `string, int, uint, float, char, bool, bytes` builtin
	// functions can be provided to get the type name from the BuiltinFunction's
	// Name field.
	"ScanArg": &gad.Function{
		Name:  "ScanArg",
		Value: newScanArgFunc,
	},
}

func newPrint(fn func(...any) (int, error)) gad.CallableFunc {
	return func(c gad.Call) (ret gad.Object, err error) {
		vargs := toPrintArgs(0, c)
		n, err := fn(vargs...)
		return gad.Int(n), err
	}
}

func newPrintf(fn func(string, ...any) (int, error)) gad.CallableFunc {
	return func(c gad.Call) (ret gad.Object, err error) {
		if c.Args.Len() < 1 {
			return gad.Nil, gad.ErrWrongNumArguments.NewError(
				"want>=1 got=" + strconv.Itoa(c.Args.Len()))
		}
		vargs := toPrintArgs(1, c)
		n, err := fn(c.Args.Get(0).String(), vargs...)
		return gad.Int(n), err
	}
}

func newSprint(fn func(...any) string) gad.CallableFunc {
	return func(c gad.Call) (ret gad.Object, err error) {
		vargs := toPrintArgs(0, c)
		return gad.String(fn(vargs...)), nil
	}
}

func newSprintf(fn func(string, ...any) string) gad.CallableFunc {
	return func(c gad.Call) (ret gad.Object, err error) {
		if c.Args.Len() < 1 {
			return gad.Nil, gad.ErrWrongNumArguments.NewError(
				"want>=1 got=" + strconv.Itoa(c.Args.Len()))
		}
		vargs := toPrintArgs(1, c)
		return gad.String(fn(c.Args.Get(0).String(), vargs...)), nil
	}
}

func newSscan(fn func(string, ...any) (int, error)) gad.CallableFunc {
	return func(c gad.Call) (ret gad.Object, err error) {
		if c.Args.Len() < 2 {
			return gad.Nil, gad.ErrWrongNumArguments.NewError(
				"want>=2 got=" + strconv.Itoa(c.Args.Len()))
		}
		vargs, err := toScanArgs(1, c)
		if err != nil {
			return gad.Nil, err
		}
		n, err := fn(c.Args.Get(0).String(), vargs...)
		return postScan(1, n, err, c), nil
	}
}

func newSscanf(
	fn func(string, string, ...any) (int, error),
) gad.CallableFunc {
	return func(c gad.Call) (ret gad.Object, err error) {
		if c.Args.Len() < 3 {
			return gad.Nil, gad.ErrWrongNumArguments.NewError(
				"want>=3 got=" + strconv.Itoa(c.Args.Len()))
		}
		vargs, err := toScanArgs(2, c)
		if err != nil {
			return gad.Nil, err
		}
		n, err := fn(c.Args.Get(0).String(), c.Args.Get(1).String(), vargs...)
		return postScan(2, n, err, c), nil
	}
}

func toScanArgs(offset int, c gad.Call) ([]any, error) {
	size := c.Args.Len()
	vargs := make([]any, 0, size-offset)
	for i := offset; i < size; i++ {
		v, ok := c.Args.Get(i).(ScanArg)
		if !ok {
			return nil, gad.NewArgumentTypeError(strconv.Itoa(i),
				"ScanArg interface", c.Args.Get(i).TypeName())
		}
		v.Set(false)
		vargs = append(vargs, v.Arg())
	}
	return vargs, nil
}

func toPrintArgs(offset int, c gad.Call) []any {
	size := c.Args.Len()
	vargs := make([]any, 0, size-offset)
	for i := offset; i < size; i++ {
		vargs = append(vargs, c.Args.Get(i))
	}
	return vargs
}

// args are always of ScanArg interface type.
func postScan(offset, n int, err error, c gad.Call) gad.Object {
	for i := offset; i < n+offset; i++ {
		c.Args.Get(i).(ScanArg).Set(true)
	}
	if err != nil {
		return &gad.Error{
			Message: err.Error(),
			Cause:   err,
		}
	}
	return gad.Int(n)
}
