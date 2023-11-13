// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

// Package strings provides strings module implementing simple functions to
// manipulate UTF-8 encoded strings for Gad script language. It wraps ToInterface's
// strings package functionalities.
package strings

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/stdlib"
)

// Module represents time module.
var Module = map[string]gad.Object{
	// gad:doc
	// # strings Module
	//
	// ## Functions
	// Contains(s string, substr string) -> bool
	// Reports whether substr is within s.
	"Contains": &gad.Function{
		Name:  "Contains",
		Value: stdlib.FuncPssRO(containsFunc),
	},
	// gad:doc
	// ContainsAny(s string, chars string) -> bool
	// Reports whether any char in chars are within s.
	"ContainsAny": &gad.Function{
		Name:  "ContainsAny",
		Value: stdlib.FuncPssRO(containsAnyFunc),
	},
	// gad:doc
	// ContainsChar(s string, c char) -> bool
	// Reports whether the char c is within s.
	"ContainsChar": &gad.Function{
		Name:  "ContainsChar",
		Value: stdlib.FuncPsrRO(containsCharFunc),
	},
	// gad:doc
	// Count(s string, substr string) -> int
	// Counts the number of non-overlapping instances of substr in s.
	"Count": &gad.Function{
		Name:  "Count",
		Value: stdlib.FuncPssRO(countFunc),
	},
	// gad:doc
	// EqualFold(s string, t string) -> bool
	// EqualFold reports whether s and t, interpreted as UTF-8 strings,
	// are equal under Unicode case-folding, which is a more general form of
	// case-insensitivity.
	"EqualFold": &gad.Function{
		Name:  "EqualFold",
		Value: stdlib.FuncPssRO(equalFoldFunc),
	},
	// gad:doc
	// Fields(s string) -> array
	// Splits the string s around each instance of one or more consecutive white
	// space characters, returning an array of substrings of s or an empty array
	// if s contains only white space.
	"Fields": &gad.Function{
		Name:  "Fields",
		Value: stdlib.FuncPsRO(fieldsFunc),
	},
	// gad:doc
	// FieldsFunc(s string, f func(char) bool) -> array
	// Splits the string s at each run of Unicode code points c satisfying f(c),
	// and returns an array of slices of s. If all code points in s satisfy
	// f(c) or the string is empty, an empty array is returned.
	"FieldsFunc": &gad.Function{
		Name:  "FieldsFunc",
		Value: fieldsFuncInv,
	},
	// gad:doc
	// HasPrefix(s string, prefix string) -> bool
	// Reports whether the string s begins with prefix.
	"HasPrefix": &gad.Function{
		Name:  "HasPrefix",
		Value: stdlib.FuncPssRO(hasPrefixFunc),
	},
	// gad:doc
	// HasSuffix(s string, suffix string) -> bool
	// Reports whether the string s ends with prefix.
	"HasSuffix": &gad.Function{
		Name:  "HasSuffix",
		Value: stdlib.FuncPssRO(hasSuffixFunc),
	},
	// gad:doc
	// Index(s string, substr string) -> int
	// Returns the index of the first instance of substr in s, or -1 if substr
	// is not present in s.
	"Index": &gad.Function{
		Name:  "Index",
		Value: stdlib.FuncPssRO(indexFunc),
	},
	// gad:doc
	// IndexAny(s string, chars string) -> int
	// Returns the index of the first instance of any char from chars in s, or
	// -1 if no char from chars is present in s.
	"IndexAny": &gad.Function{
		Name:  "IndexAny",
		Value: stdlib.FuncPssRO(indexAnyFunc),
	},
	// gad:doc
	// IndexByte(s string, c char|int) -> int
	// Returns the index of the first byte value of c in s, or -1 if byte value
	// of c is not present in s. c's integer value must be between 0 and 255.
	"IndexByte": &gad.Function{
		Name:  "IndexByte",
		Value: stdlib.FuncPsrRO(indexByteFunc),
	},
	// gad:doc
	// IndexChar(s string, c char) -> int
	// Returns the index of the first instance of the char c, or -1 if char is
	// not present in s.
	"IndexChar": &gad.Function{
		Name:  "IndexChar",
		Value: stdlib.FuncPsrRO(indexCharFunc),
	},
	// gad:doc
	// IndexFunc(s string, f func(char) bool) -> int
	// Returns the index into s of the first Unicode code point satisfying f(c),
	// or -1 if none do.
	"IndexFunc": &gad.Function{
		Name:  "IndexFunc",
		Value: newIndexFuncInv(strings.IndexFunc),
	},
	// gad:doc
	// Join(arr array, sep string) -> string
	// Concatenates the string values of array arr elements to create a
	// single string. The separator string sep is placed between elements in the
	// resulting string.
	"Join": &gad.Function{
		Name:  "Join",
		Value: stdlib.FuncPAsRO(joinFunc),
	},
	// gad:doc
	// LastIndex(s string, substr string) -> int
	// Returns the index of the last instance of substr in s, or -1 if substr
	// is not present in s.
	"LastIndex": &gad.Function{
		Name:  "LastIndex",
		Value: stdlib.FuncPssRO(lastIndexFunc),
	},
	// gad:doc
	// LastIndexAny(s string, chars string) -> int
	// Returns the index of the last instance of any char from chars in s, or
	// -1 if no char from chars is present in s.
	"LastIndexAny": &gad.Function{
		Name:  "LastIndexAny",
		Value: stdlib.FuncPssRO(lastIndexAnyFunc),
	},
	// gad:doc
	// LastIndexByte(s string, c char|int) -> int
	// Returns the index of byte value of the last instance of c in s, or -1
	// if c is not present in s. c's integer value must be between 0 and 255.
	"LastIndexByte": &gad.Function{
		Name:  "LastIndexByte",
		Value: stdlib.FuncPsrRO(lastIndexByteFunc),
	},
	// gad:doc
	// LastIndexFunc(s string, f func(char) bool) -> int
	// Returns the index into s of the last Unicode code point satisfying f(c),
	// or -1 if none do.
	"LastIndexFunc": &gad.Function{
		Name:  "LastIndexFunc",
		Value: newIndexFuncInv(strings.LastIndexFunc),
	},
	// gad:doc
	// Dict(f func(char) char, s string) -> string
	// Returns a copy of the string s with all its characters modified
	// according to the mapping function f. If f returns a negative value, the
	// character is dropped from the string with no replacement.
	"Dict": &gad.Function{
		Name:  "Dict",
		Value: mapFuncInv,
	},
	// gad:doc
	// PadLeft(s string, padLen int[, padWith any]) -> string
	// Returns a string that is padded on the left with the string `padWith` until
	// the `padLen` length is reached. If padWith is not given, a white space is
	// used as default padding.
	"PadLeft": &gad.Function{
		Name: "PadLeft",
		Value: func(c gad.Call) (gad.Object, error) {
			return pad(c, true)
		},
	},
	// gad:doc
	// PadRight(s string, padLen int[, padWith any]) -> string
	// Returns a string that is padded on the right with the string `padWith` until
	// the `padLen` length is reached. If padWith is not given, a white space is
	// used as default padding.
	"PadRight": &gad.Function{
		Name: "PadRight",
		Value: func(c gad.Call) (gad.Object, error) {
			return pad(c, false)
		},
	},
	// gad:doc
	// Repeat(s string, count int) -> string
	// Returns a new string consisting of count copies of the string s.
	//
	// - If count is a negative int, it returns empty string.
	// - If (len(s) * count) overflows, it panics.
	"Repeat": &gad.Function{
		Name:  "Repeat",
		Value: stdlib.FuncPsiRO(repeatFunc),
	},
	// gad:doc
	// Replace(s string, old string, new string[, n int]) -> string
	// Returns a copy of the string s with the first n non-overlapping instances
	// of old replaced by new. If n is not provided or -1, it replaces all
	// instances.
	"Replace": &gad.Function{
		Name:  "Replace",
		Value: replaceFunc,
	},
	// gad:doc
	// Split(s string, sep string[, n int]) -> [string]
	// Splits s into substrings separated by sep and returns an array of
	// the substrings between those separators.
	//
	// n determines the number of substrings to return:
	//
	// - n < 0: all substrings (default)
	// - n > 0: at most n substrings; the last substring will be the unsplit remainder.
	// - n == 0: the result is empty array
	"Split": &gad.Function{
		Name:  "Split",
		Value: newSplitFunc(strings.SplitN),
	},
	// gad:doc
	// SplitAfter(s string, sep string[, n int]) -> [string]
	// Slices s into substrings after each instance of sep and returns an array
	// of those substrings.
	//
	// n determines the number of substrings to return:
	//
	// - n < 0: all substrings (default)
	// - n > 0: at most n substrings; the last substring will be the unsplit remainder.
	// - n == 0: the result is empty array
	"SplitAfter": &gad.Function{
		Name:  "SplitAfter",
		Value: newSplitFunc(strings.SplitAfterN),
	},
	// gad:doc
	// Title(s string) -> string
	// Deprecated: Returns a copy of the string s with all Unicode letters that
	// begin words mapped to their Unicode title case.
	"Title": &gad.Function{
		Name:  "Title",
		Value: stdlib.FuncPsRO(titleFunc),
	},
	// gad:doc
	// ToLower(s string) -> string
	// Returns s with all Unicode letters mapped to their lower case.
	"ToLower": &gad.Function{
		Name:  "ToLower",
		Value: stdlib.FuncPsRO(toLowerFunc),
	},
	// gad:doc
	// ToTitle(s string) -> string
	// Returns a copy of the string s with all Unicode letters mapped to their
	// Unicode title case.
	"ToTitle": &gad.Function{
		Name:  "ToTitle",
		Value: stdlib.FuncPsRO(toTitleFunc),
	},
	// gad:doc
	// ToUpper(s string) -> string
	// Returns s with all Unicode letters mapped to their upper case.
	"ToUpper": &gad.Function{
		Name:  "ToUpper",
		Value: stdlib.FuncPsRO(toUpperFunc),
	},
	// gad:doc
	// ToValidUTF8(s string[, replacement string]) -> string
	// Returns a copy of the string s with each run of invalid UTF-8 byte
	// sequences replaced by the replacement string, which may be empty.
	"ToValidUTF8": &gad.Function{
		Name:  "ToValidUTF8",
		Value: toValidUTF8Func,
	},
	// gad:doc
	// Trim(s string, cutset string) -> string
	// Returns a slice of the string s with all leading and trailing Unicode
	// code points contained in cutset removed.
	"Trim": &gad.Function{
		Name:  "Trim",
		Value: stdlib.FuncPssRO(trimFunc),
	},
	// gad:doc
	// TrimFunc(s string, f func(char) bool) -> string
	// Returns a slice of the string s with all leading and trailing Unicode
	// code points satisfying f removed.
	"TrimFunc": &gad.Function{
		Name:  "TrimFunc",
		Value: newTrimFuncInv(strings.TrimFunc),
	},
	// gad:doc
	// TrimLeft(s string, cutset string) -> string
	// Returns a slice of the string s with all leading Unicode code points
	// contained in cutset removed.
	"TrimLeft": &gad.Function{
		Name:  "TrimLeft",
		Value: stdlib.FuncPssRO(trimLeftFunc),
	},
	// gad:doc
	// TrimLeftFunc(s string, f func(char) bool) -> string
	// Returns a slice of the string s with all leading Unicode code points
	// c satisfying f(c) removed.
	"TrimLeftFunc": &gad.Function{
		Name:  "TrimLeftFunc",
		Value: newTrimFuncInv(strings.TrimLeftFunc),
	},
	// gad:doc
	// TrimPrefix(s string, prefix string) -> string
	// Returns s without the provided leading prefix string. If s doesn't start
	// with prefix, s is returned unchanged.
	"TrimPrefix": &gad.Function{
		Name:  "TrimPrefix",
		Value: stdlib.FuncPssRO(trimPrefixFunc),
	},
	// gad:doc
	// TrimRight(s string, cutset string) -> string
	// Returns a slice of the string s with all trailing Unicode code points
	// contained in cutset removed.
	"TrimRight": &gad.Function{
		Name:  "TrimRight",
		Value: stdlib.FuncPssRO(trimRightFunc),
	},
	// gad:doc
	// TrimRightFunc(s string, f func(char) bool) -> string
	// Returns a slice of the string s with all trailing Unicode code points
	// c satisfying f(c) removed.
	"TrimRightFunc": &gad.Function{
		Name:  "TrimRightFunc",
		Value: newTrimFuncInv(strings.TrimRightFunc),
	},
	// gad:doc
	// TrimSpace(s string) -> string
	// Returns a slice of the string s, with all leading and trailing white
	// space removed, as defined by Unicode.
	"TrimSpace": &gad.Function{
		Name:  "TrimSpace",
		Value: stdlib.FuncPsRO(trimSpaceFunc),
	},
	// gad:doc
	// TrimSuffix(s string, suffix string) -> string
	// Returns s without the provided trailing suffix string. If s doesn't end
	// with suffix, s is returned unchanged.
	"TrimSuffix": &gad.Function{
		Name:  "TrimSuffix",
		Value: stdlib.FuncPssRO(trimSuffixFunc),
	},
}

func containsFunc(s, substr string) gad.Object {
	return gad.Bool(strings.Contains(s, substr))
}

func containsAnyFunc(s, chars string) gad.Object {
	return gad.Bool(strings.ContainsAny(s, chars))
}

func containsCharFunc(s string, c rune) gad.Object {
	return gad.Bool(strings.ContainsRune(s, c))
}

func countFunc(s, substr string) gad.Object {
	return gad.Int(strings.Count(s, substr))
}

func equalFoldFunc(s, t string) gad.Object {
	return gad.Bool(strings.EqualFold(s, t))
}

func fieldsFunc(s string) gad.Object {
	fields := strings.Fields(s)
	out := make(gad.Array, 0, len(fields))
	for _, s := range fields {
		out = append(out, gad.String(s))
	}
	return out
}

func fieldsFuncInv(c gad.Call) (gad.Object, error) {
	return stringInvoke(c, 0, 1,
		func(s string, inv *gad.Invoker) (gad.Object, error) {
			var err error
			fields := strings.FieldsFunc(s, func(r rune) bool {
				if err != nil {
					return false
				}
				var ret gad.Object
				ret, err = inv.Invoke(gad.Args{{gad.Char(r)}}, nil)
				if err != nil {
					return false
				}
				return !ret.IsFalsy()
			})
			if err != nil {
				return gad.Nil, err
			}
			out := make(gad.Array, 0, len(fields))
			for _, s := range fields {
				out = append(out, gad.String(s))
			}
			return out, nil
		},
	)
}

func hasPrefixFunc(s, prefix string) gad.Object {
	return gad.Bool(strings.HasPrefix(s, prefix))
}

func hasSuffixFunc(s, suffix string) gad.Object {
	return gad.Bool(strings.HasSuffix(s, suffix))
}

func indexFunc(s, substr string) gad.Object {
	return gad.Int(strings.Index(s, substr))
}

func indexAnyFunc(s, chars string) gad.Object {
	return gad.Int(strings.IndexAny(s, chars))
}

func indexByteFunc(s string, c rune) gad.Object {
	if c > 255 || c < 0 {
		return gad.Int(-1)
	}
	return gad.Int(strings.IndexByte(s, byte(c)))
}

func indexCharFunc(s string, c rune) gad.Object {
	return gad.Int(strings.IndexRune(s, c))
}

func joinFunc(arr gad.Array, sep string) gad.Object {
	elems := make([]string, len(arr))
	for i := range arr {
		elems[i] = arr[i].ToString()
	}
	return gad.String(strings.Join(elems, sep))
}

func lastIndexFunc(s, substr string) gad.Object {
	return gad.Int(strings.LastIndex(s, substr))
}

func lastIndexAnyFunc(s, chars string) gad.Object {
	return gad.Int(strings.LastIndexAny(s, chars))
}

func lastIndexByteFunc(s string, c rune) gad.Object {
	if c > 255 || c < 0 {
		return gad.Int(-1)
	}
	return gad.Int(strings.LastIndexByte(s, byte(c)))
}

func mapFuncInv(c gad.Call) (gad.Object, error) {
	return stringInvoke(c, 1, 0,
		func(s string, inv *gad.Invoker) (gad.Object, error) {
			var err error
			out := strings.Map(func(r rune) rune {
				if err != nil {
					return utf8.RuneError
				}
				var ret gad.Object
				ret, err = inv.Invoke(gad.Args{{gad.Char(r)}}, nil)
				if err != nil {
					return 0
				}
				r, ok := gad.ToGoRune(ret)
				if !ok {
					return utf8.RuneError
				}
				return r
			}, s)
			return gad.String(out), err
		},
	)
}

func pad(c gad.Call, left bool) (gad.Object, error) {
	size := c.Args.Len()
	if size != 2 && size != 3 {
		return gad.Nil,
			gad.ErrWrongNumArguments.NewError("want=2..3 got=" + strconv.Itoa(size))
	}
	s := c.Args.Get(0).ToString()
	padLen, ok := gad.ToGoInt(c.Args.Get(1))
	if !ok {
		return gad.Nil,
			gad.NewArgumentTypeError("2nd", "int", c.Args.Get(1).Type().Name())
	}
	diff := padLen - len(s)
	if diff <= 0 {
		return gad.String(s), nil
	}
	padWith := " "
	if size > 2 {
		if padWith = c.Args.Get(2).ToString(); len(padWith) == 0 {
			return gad.String(s), nil
		}
	}
	r := (diff-len(padWith))/len(padWith) + 2
	if r <= 0 {
		return gad.String(s), nil
	}
	var sb strings.Builder
	sb.Grow(padLen)
	if left {
		sb.WriteString(strings.Repeat(padWith, r)[:diff])
		sb.WriteString(s)
	} else {
		sb.WriteString(s)
		sb.WriteString(strings.Repeat(padWith, r)[:diff])
	}
	return gad.String(sb.String()), nil
}

func repeatFunc(s string, count int) gad.Object {
	// if n is negative strings.Repeat function panics
	if count < 0 {
		return gad.String("")
	}
	return gad.String(strings.Repeat(s, count))
}

func replaceFunc(c gad.Call) (gad.Object, error) {
	size := c.Args.Len()
	if size != 3 && size != 4 {
		return gad.Nil,
			gad.ErrWrongNumArguments.NewError("want=3..4 got=" + strconv.Itoa(size))
	}
	s := c.Args.Get(0).ToString()
	old := c.Args.Get(1).ToString()
	news := c.Args.Get(2).ToString()
	n := -1
	if size == 4 {
		v, ok := gad.ToGoInt(c.Args.Get(3))
		if !ok {
			return gad.Nil,
				gad.NewArgumentTypeError("4th", "int", c.Args.Get(3).Type().Name())
		}
		n = v
	}
	return gad.String(strings.Replace(s, old, news, n)), nil
}

func titleFunc(s string) gad.Object {
	//lint:ignore SA1019 Keep it for backward compatibility.
	return gad.String(strings.Title(s)) // nolint staticcheck Keep it for backward compatibility
}

func toLowerFunc(s string) gad.Object { return gad.String(strings.ToLower(s)) }

func toTitleFunc(s string) gad.Object { return gad.String(strings.ToTitle(s)) }

func toUpperFunc(s string) gad.Object { return gad.String(strings.ToUpper(s)) }

func toValidUTF8Func(c gad.Call) (gad.Object, error) {
	size := c.Args.Len()
	if size != 1 && size != 2 {
		return gad.Nil,
			gad.ErrWrongNumArguments.NewError("want=1..2 got=" + strconv.Itoa(size))
	}
	s := c.Args.Get(0).ToString()
	var repl string
	if size == 2 {
		repl = c.Args.Get(1).ToString()
	}
	return gad.String(strings.ToValidUTF8(s, repl)), nil
}

func trimFunc(s, cutset string) gad.Object {
	return gad.String(strings.Trim(s, cutset))
}

func trimLeftFunc(s, cutset string) gad.Object {
	return gad.String(strings.TrimLeft(s, cutset))
}

func trimPrefixFunc(s, prefix string) gad.Object {
	return gad.String(strings.TrimPrefix(s, prefix))
}

func trimRightFunc(s, cutset string) gad.Object {
	return gad.String(strings.TrimRight(s, cutset))
}

func trimSpaceFunc(s string) gad.Object {
	return gad.String(strings.TrimSpace(s))
}

func trimSuffixFunc(s, suffix string) gad.Object {
	return gad.String(strings.TrimSuffix(s, suffix))
}

func newSplitFunc(fn func(string, string, int) []string) gad.CallableFunc {
	return func(c gad.Call) (gad.Object, error) {
		size := c.Args.Len()
		if size != 2 && size != 3 {
			return gad.Nil,
				gad.ErrWrongNumArguments.NewError("want=2..3 got=" + strconv.Itoa(size))
		}
		s := c.Args.Get(0).ToString()
		sep := c.Args.Get(1).ToString()
		n := -1
		if size == 3 {
			v, ok := gad.ToGoInt(c.Args.Get(2))
			if !ok {
				return gad.Nil,
					gad.NewArgumentTypeError("3rd", "int", c.Args.Get(2).Type().Name())
			}
			n = v
		}
		strs := fn(s, sep, n)
		out := make(gad.Array, 0, len(strs))
		for _, s := range strs {
			out = append(out, gad.String(s))
		}
		return out, nil
	}
}

func newIndexFuncInv(fn func(string, func(rune) bool) int) gad.CallableFunc {
	return func(c gad.Call) (gad.Object, error) {
		return stringInvoke(c, 0, 1,
			func(s string, inv *gad.Invoker) (gad.Object, error) {
				var err error
				out := fn(s, func(r rune) bool {
					if err != nil {
						return false
					}
					var ret gad.Object
					ret, err = inv.Invoke(gad.Args{{gad.Char(r)}}, nil)
					if err != nil {
						return false
					}
					return !ret.IsFalsy()
				})
				return gad.Int(out), err
			},
		)
	}
}

func newTrimFuncInv(fn func(string, func(rune) bool) string) gad.CallableFunc {
	return func(c gad.Call) (gad.Object, error) {
		return stringInvoke(c, 0, 1,
			func(s string, inv *gad.Invoker) (gad.Object, error) {
				var err error
				out := fn(s, func(r rune) bool {
					if err != nil {
						return false
					}
					var ret gad.Object
					ret, err = inv.Invoke(gad.Args{{gad.Char(r)}}, nil)
					if err != nil {
						return false
					}
					return !ret.IsFalsy()
				})
				return gad.String(out), err
			},
		)
	}
}

func stringInvoke(
	c gad.Call,
	sidx int,
	cidx int,
	fn func(string, *gad.Invoker) (gad.Object, error),
) (gad.Object, error) {
	err := c.Args.CheckLen(2)
	if err != nil {
		return gad.Nil, err
	}

	str := c.Args.Get(sidx).ToString()
	callee := c.Args.Get(cidx)
	if !gad.Callable(callee) {
		return gad.Nil, gad.ErrNotCallable
	}
	if c.VM == nil {
		if _, ok := callee.(*gad.CompiledFunction); ok {
			return gad.Nil, gad.ErrNotCallable
		}
	}

	inv := gad.NewInvoker(c.VM, callee)
	inv.Acquire()
	defer inv.Release()
	return fn(str, inv)
}
