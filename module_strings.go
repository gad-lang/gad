// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import (
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

var stringsReSpaces = regexp.MustCompile(`\s+`)

// stringsModuleSpec is the module spec shared by the builtin `strings` namespace
// members and the importable strings module.
var stringsModuleSpec = NewModuleSpecFromName("strings")

// StringsModule returns the `strings` builtin namespace. It is also used by the
// stdlib `strings` importable module.
func StringsModule() Dict { return newStringsModule() }

// newStringsModule builds the `strings` builtin namespace (Go's strings).
func newStringsModule() Dict {
	return Dict{
		// gad:doc
		// # strings module
		//
		// ## Functions
		// contains(s str, substr str) <bool>
		// Reports whether substr is within s.
		"contains": &BuiltinFunction{
			FuncName: "contains",
			Value:    funcPssRO(stringsContainsFunc),
		},
		// gad:doc
		// containsAny(s str, chars str) <bool>
		// Reports whether any char in chars are within s.
		"containsAny": &BuiltinFunction{
			FuncName: "containsAny",
			Value:    funcPssRO(stringsContainsAnyFunc),
		},
		// gad:doc
		// containsChar(s str, c char) <bool>
		// Reports whether the char c is within s.
		"containsChar": &BuiltinFunction{
			FuncName: "containsChar",
			Value:    funcPsrRO(stringsContainsCharFunc),
		},
		// gad:doc
		// count(s str, substr str) <int>
		// Counts the number of non-overlapping instances of substr in s.
		"count": &BuiltinFunction{
			FuncName: "count",
			Value:    funcPssRO(stringsCountFunc),
		},
		// gad:doc
		// equalFold(s str, t str) <bool>
		// EqualFold reports whether s and t, interpreted as UTF-8 strings,
		// are equal under Unicode case-folding, which is a more general form of
		// case-insensitivity.
		"equalFold": &BuiltinFunction{
			FuncName: "equalFold",
			Value:    funcPssRO(stringsEqualFoldFunc),
		},
		// gad:doc
		// fields(s str) <array>
		// Splits the string s around each instance of one or more consecutive white
		// space characters, returning an array of substrings of s or an empty array
		// if s contains only white space.
		"fields": &BuiltinFunction{
			FuncName: "fields",
			Value:    funcPsRO(stringsFieldsFunc),
		},
		// gad:doc
		// fieldsFunc(s str, f func(char) bool) <array>
		// Splits the string s at each run of Unicode code points c satisfying f(c),
		// and returns an array of slices of s. If all code points in s satisfy
		// f(c) or the string is empty, an empty array is returned.
		"fieldsFunc": &BuiltinFunction{
			FuncName: "fieldsFunc",
			Value:    stringsFieldsFuncInv,
		},
		// gad:doc
		// hasPrefix(s str, prefix str) <bool>
		// Reports whether the string s begins with prefix.
		"hasPrefix": &BuiltinFunction{
			FuncName: "hasPrefix",
			Value:    funcPssRO(stringsHasPrefixFunc),
		},
		// gad:doc
		// hasSuffix(s str, suffix str) <bool>
		// Reports whether the string s ends with prefix.
		"hasSuffix": &BuiltinFunction{
			FuncName: "hasSuffix",
			Value:    funcPssRO(stringsHasSuffixFunc),
		},
		// gad:doc
		// index(s str, substr str) <int>
		// Returns the index of the first instance of substr in s, or -1 if substr
		// is not present in s.
		"index": &BuiltinFunction{
			FuncName: "index",
			Value:    funcPssRO(stringsIndexFunc),
		},
		// gad:doc
		// indexAny(s str, chars str) <int>
		// Returns the index of the first instance of any char from chars in s, or
		// -1 if no char from chars is present in s.
		"indexAny": &BuiltinFunction{
			FuncName: "indexAny",
			Value:    funcPssRO(stringsIndexAnyFunc),
		},
		// gad:doc
		// indexByte(s str, c char|int) <int>
		// Returns the index of the first byte value of c in s, or -1 if byte value
		// of c is not present in s. c's integer value must be between 0 and 255.
		"indexByte": &BuiltinFunction{
			FuncName: "indexByte",
			Value:    funcPsrRO(stringsIndexByteFunc),
		},
		// gad:doc
		// indexChar(s str, c char) <int>
		// Returns the index of the first instance of the char c, or -1 if char is
		// not present in s.
		"indexChar": &BuiltinFunction{
			FuncName: "indexChar",
			Value:    funcPsrRO(stringsIndexCharFunc),
		},
		// gad:doc
		// indexFunc(s str, f func(char) bool) <int>
		// Returns the index into s of the first Unicode code point satisfying f(c),
		// or -1 if none do.
		"indexFunc": &BuiltinFunction{
			FuncName: "indexFunc",
			Value:    stringsNewIndexFuncInv(strings.IndexFunc),
		},
		// gad:doc
		// join(arr array, sep str) <str>
		// Concatenates the string values of array arr elements to create a
		// single string. The separator string sep is placed between elements in the
		// resulting string.
		"join": &BuiltinFunction{
			FuncName: "join",
			Value:    funcPAsRO(stringsJoinFunc),
		},
		// gad:doc
		// joinAnd(arr array, sep, lastSep str) <str>
		// Concatenates the string values of array arr elements to create a
		// single string. The separator string sep is placed between elements
		// and lastSep is placed between non last and last elements in the
		// resulting string.
		"joinAnd": &BuiltinFunction{
			FuncName: "joinAnd",
			Value:    funcPAssRO(stringsJoinAndFunc),
		},
		// gad:doc
		// lastIndex(s str, substr str) <int>
		// Returns the index of the last instance of substr in s, or -1 if substr
		// is not present in s.
		"lastIndex": &BuiltinFunction{
			FuncName: "lastIndex",
			Value:    funcPssRO(stringsLastIndexFunc),
		},
		// gad:doc
		// lastIndexAny(s str, chars str) <int>
		// Returns the index of the last instance of any char from chars in s, or
		// -1 if no char from chars is present in s.
		"lastIndexAny": &BuiltinFunction{
			FuncName: "lastIndexAny",
			Value:    funcPssRO(stringsLastIndexAnyFunc),
		},
		// gad:doc
		// lastIndexByte(s str, c char|int) <int>
		// Returns the index of byte value of the last instance of c in s, or -1
		// if c is not present in s. c's integer value must be between 0 and 255.
		"lastIndexByte": &BuiltinFunction{
			FuncName: "lastIndexByte",
			Value:    funcPsrRO(stringsLastIndexByteFunc),
		},
		// gad:doc
		// lastIndexFunc(s str, f func(char) bool) <int>
		// Returns the index into s of the last Unicode code point satisfying f(c),
		// or -1 if none do.
		"lastIndexFunc": &BuiltinFunction{
			FuncName: "lastIndexFunc",
			Value:    stringsNewIndexFuncInv(strings.LastIndexFunc),
		},
		// gad:doc
		// dict(f func(char) char, s str) <str>
		// Returns a copy of the string s with all its characters modified
		// according to the mapping function f. If f returns a negative value, the
		// character is dropped from the string with no replacement.
		"dict": &BuiltinFunction{
			FuncName: "dict",
			Value:    stringsMapFuncInv,
		},
		// gad:doc
		// padLeft(s str, padLen int[, padWith any]) <str>
		// Returns a string that is padded on the left with the string `padWith` until
		// the `padLen` length is reached. If padWith is not given, a white space is
		// used as default padding.
		"padLeft": &BuiltinFunction{
			FuncName: "padLeft",
			Value: func(c Call) (Object, error) {
				return stringsPad(c, true)
			},
		},
		// gad:doc
		// padRight(s str, padLen int[, padWith any]) <str>
		// Returns a string that is padded on the right with the string `padWith` until
		// the `padLen` length is reached. If padWith is not given, a white space is
		// used as default padding.
		"padRight": &BuiltinFunction{
			FuncName: "padRight",
			Value: func(c Call) (Object, error) {
				return stringsPad(c, false)
			},
		},
		// gad:doc
		// repeat(s str, count int) <str>
		// Returns a new string consisting of count copies of the string s.
		//
		// - If count is a negative int, it returns empty string.
		// - If (len(s) * count) overflows, it panics.
		"repeat": &BuiltinFunction{
			FuncName: "repeat",
			Value:    funcPsiRO(stringsRepeatFunc),
		},
		// gad:doc
		// replace(s str, old str, new str[, n int]) <str>
		// Returns a copy of the string s with the first n non-overlapping instances
		// of old replaced by new. If n is not provided or -1, it replaces all
		// instances.
		"replace": &BuiltinFunction{
			FuncName: "replace",
			Value:    stringsReplaceFunc,
		},
		// gad:doc
		// split(s str, sep str[, n int]) <[str]>
		// Splits s into substrings separated by sep and returns an array of
		// the substrings between those separators.
		//
		// n determines the number of substrings to return:
		//
		// - n < 0: all substrings (default)
		// - n > 0: at most n substrings; the last substring will be the unsplit remainder.
		// - n == 0: the result is empty array
		"split": &BuiltinFunction{
			FuncName: "split",
			Value:    stringsNewSplitFunc(strings.SplitN),
		},
		// gad:doc
		// splitAfter(s str, sep str[, n int]) <[str]>
		// Slices s into substrings after each instance of sep and returns an array
		// of those substrings.
		//
		// n determines the number of substrings to return:
		//
		// - n < 0: all substrings (default)
		// - n > 0: at most n substrings; the last substring will be the unsplit remainder.
		// - n == 0: the result is empty array
		"splitAfter": &BuiltinFunction{
			FuncName: "splitAfter",
			Value:    stringsNewSplitFunc(strings.SplitAfterN),
		},
		// gad:doc
		// title(s str) <str>
		// Deprecated: Returns a copy of the string s with all Unicode letters that
		// begin words mapped to their Unicode title case.
		"title": &BuiltinFunction{
			FuncName: "title",
			Value:    funcPsRO(stringsTitleFunc),
		},
		// gad:doc
		// toLower(s str) <str>
		// Returns s with all Unicode letters mapped to their lower case.
		"toLower": &BuiltinFunction{
			FuncName: "toLower",
			Value:    funcPsRO(stringsToLowerFunc),
		},
		// gad:doc
		// toTitle(s str) <str>
		// Returns a copy of the string s with all Unicode letters mapped to their
		// Unicode title case.
		"toTitle": &BuiltinFunction{
			FuncName: "toTitle",
			Value:    funcPsRO(stringsToTitleFunc),
		},
		// gad:doc
		// toUpper(s str) <str>
		// Returns s with all Unicode letters mapped to their upper case.
		"toUpper": &BuiltinFunction{
			FuncName: "toUpper",
			Value:    funcPsRO(stringsToUpperFunc),
		},
		// gad:doc
		// toValidUTF8(s str[, replacement str]) <str>
		// Returns a copy of the string s with each run of invalid UTF-8 byte
		// sequences replaced by the replacement string, which may be empty.
		"toValidUTF8": &BuiltinFunction{
			FuncName: "toValidUTF8",
			Value:    stringsToValidUTF8Func,
		},
		// gad:doc
		// trim(s str, cutset str) <str>
		// Returns a slice of the string s with all leading and trailing Unicode
		// code points contained in cutset removed.
		"trim": &BuiltinFunction{
			FuncName: "trim",
			Value:    funcPssRO(stringsTrimFunc),
		},
		// gad:doc
		// trimFunc(s str, f func(char) bool) <str>
		// Returns a slice of the string s with all leading and trailing Unicode
		// code points satisfying f removed.
		"trimFunc": &BuiltinFunction{
			FuncName: "trimFunc",
			Value:    stringsNewTrimFuncInv(strings.TrimFunc),
		},
		// gad:doc
		// trimLeft(s str, cutset str) <str>
		// Returns a slice of the string s with all leading Unicode code points
		// contained in cutset removed.
		"trimLeft": &BuiltinFunction{
			FuncName: "trimLeft",
			Value:    funcPssRO(stringsTrimLeftFunc),
		},
		// gad:doc
		// trimLeftFunc(s str, f func(char) bool) <str>
		// Returns a slice of the string s with all leading Unicode code points
		// c satisfying f(c) removed.
		"trimLeftFunc": &BuiltinFunction{
			FuncName: "trimLeftFunc",
			Value:    stringsNewTrimFuncInv(strings.TrimLeftFunc),
		},
		// gad:doc
		// trimPrefix(s str, prefix str) <str>
		// Returns s without the provided leading prefix string. If s doesn't start
		// with prefix, s is returned unchanged.
		"trimPrefix": &BuiltinFunction{
			FuncName: "trimPrefix",
			Value:    funcPssRO(stringsTrimPrefixFunc),
		},
		// gad:doc
		// trimRight(s str, cutset str) <str>
		// Returns a slice of the string s with all trailing Unicode code points
		// contained in cutset removed.
		"trimRight": &BuiltinFunction{
			FuncName: "trimRight",
			Value:    funcPssRO(stringsTrimRightFunc),
		},
		// gad:doc
		// trimRightFunc(s str, f func(char) bool) <str>
		// Returns a slice of the string s with all trailing Unicode code points
		// c satisfying f(c) removed.
		"trimRightFunc": &BuiltinFunction{
			FuncName: "trimRightFunc",
			Value:    stringsNewTrimFuncInv(strings.TrimRightFunc),
		},
		// gad:doc
		// trimSpace(s str) <str>
		// Returns a slice of the string s, with all leading and trailing white
		// space removed, as defined by Unicode.
		"trimSpace": &BuiltinFunction{
			FuncName: "trimSpace",
			Value:    funcPsRO(stringsTrimSpaceFunc),
		},
		// gad:doc
		// trimSuffix(s str, suffix str) <str>
		// Returns s without the provided trailing suffix string. If s doesn't end
		// with suffix, s is returned unchanged.
		"trimSuffix": &BuiltinFunction{
			FuncName: "trimSuffix",
			Value:    funcPssRO(stringsTrimSuffixFunc),
		},

		// gad:doc
		// trunc(s str, maxLen int; emph="...") <str>
		// Truncate s to maxLen concatenated with emph.
		"trunc": &BuiltinFunction{
			FuncName: "trunc",
			Value: func(c Call) (Object, error) {
				if err := c.Args.CheckLen(2); err != nil {
					return Nil, err
				}

				var (
					emph = &NamedArgVar{
						Name:          "emph",
						Value:         Str("..."),
						TypeAssertion: TypeAssertionFromTypes(TStr),
					}
				)
				if err := c.NamedArgs.Get(emph); err != nil {
					return Nil, err
				}

				s1, ok := ToGoString(c.Args.Get(0))
				if !ok {
					return Nil, NewArgumentTypeError("1st", "str", c.Args.Get(0).Type().Name())
				}
				i, ok := ToGoInt(c.Args.Get(1))
				if !ok {
					return Nil, NewArgumentTypeError("2nd", "int", c.Args.Get(1).Type().Name())
				}
				return stringsTruncFunc(s1, i, emph.Value.ToString()), nil
			},
		},

		// gad:doc
		// slitWords(s str|rawstr) <array>
		// Split words by spaces using regex `\s+`.
		// If s is rawstr, returns Array of Rawstr, otherwise, Array of Str.
		"slitWords": &BuiltinFunction{
			FuncName: "trunc",
			Value: func(c Call) (Object, error) {
				if err := c.Args.CheckLen(1); err != nil {
					return Nil, err
				}

				var (
					arg    = c.Args.Get(0)
					_, raw = arg.(RawStr)
					s      string
					ret    Array
				)

				if arg == Nil {
					return ret, nil
				}

				s = arg.ToString()

				words := stringsReSpaces.Split(s, -1)

				if len(words) == 0 {
					return ret, nil
				}

				if words[0] == "" {
					words = words[1:]
				}

				ret = make(Array, len(words))

				if raw {
					for i, word := range words {
						ret[i] = RawStr(word)
					}
				} else {
					for i, word := range words {
						ret[i] = Str(word)
					}
				}

				return ret, nil
			},
		},

		// gad:doc
		// truncWords(s str|rawstr, max int; emph="...", atlimit=off) <str|rawstr>
		// Truncate words in s to maxLen concatenated with emph. If atlimit is Falsy,
		// limits at word count equals to max, otherwise at length of s equals to max.
		"truncWords": &BuiltinFunction{
			FuncName: "trunc",
			Value: func(c Call) (Object, error) {
				if err := c.Args.CheckLen(2); err != nil {
					return Nil, err
				}

				var (
					emph = &NamedArgVar{
						Name:          "emph",
						Value:         Str("..."),
						TypeAssertion: TypeAssertionFromTypes(TStr),
					}
					atlimit = &NamedArgVar{
						Name:  "atlimit",
						Value: No,
					}
				)

				if err := c.NamedArgs.Get(emph, atlimit); err != nil {
					return Nil, err
				}

				var (
					arg    = c.Args.Get(0)
					_, raw = arg.(RawStr)
					s      string
				)

				if arg == Nil {
					return Str(""), nil
				}

				s = arg.ToString()
				limit, ok := ToGoInt(c.Args.Get(1))
				if !ok {
					return Nil, NewArgumentTypeError("2nd", "int", c.Args.Get(1).Type().Name())
				}

				if atlimit.Value.IsFalsy() {
					var (
						words = stringsReSpaces.Split(s, limit+1)
						b     strings.Builder
						emphs = emph.Value.ToString()
						limit = limit - len(emphs)
					)

					for _, word := range words {
						if word == "" {
							continue
						}
						if b.Len()+len(word) > limit {
							break
						}
						b.WriteByte(' ')
						b.WriteString(word)
					}
					b.WriteString(emphs)
					s = strings.TrimSpace(b.String())
					if raw {
						return RawStr(s), nil
					}
					return Str(s), nil
				}

				return stringsTruncFunc(s, limit, emph.Value.ToString()), nil
			},
		},
	}
}

func stringsContainsFunc(s, substr string) Object {
	return Bool(strings.Contains(s, substr))
}

func stringsContainsAnyFunc(s, chars string) Object {
	return Bool(strings.ContainsAny(s, chars))
}

func stringsContainsCharFunc(s string, c rune) Object {
	return Bool(strings.ContainsRune(s, c))
}

func stringsCountFunc(s, substr string) Object {
	return Int(strings.Count(s, substr))
}

func stringsEqualFoldFunc(s, t string) Object {
	return Bool(strings.EqualFold(s, t))
}

func stringsFieldsFunc(s string) Object {
	fields := strings.Fields(s)
	out := make(Array, 0, len(fields))
	for _, s := range fields {
		out = append(out, Str(s))
	}
	return out
}

func stringsFieldsFuncInv(c Call) (Object, error) {
	return stringsStringInvoke(c, 0, 1,
		func(s string, inv *Invoker) (Object, error) {
			var err error
			fields := strings.FieldsFunc(s, func(r rune) bool {
				if err != nil {
					return false
				}
				var ret Object
				ret, err = inv.Invoke(Args{{Char(r)}}, nil)
				if err != nil {
					return false
				}
				return !ret.IsFalsy()
			})
			if err != nil {
				return Nil, err
			}
			out := make(Array, 0, len(fields))
			for _, s := range fields {
				out = append(out, Str(s))
			}
			return out, nil
		},
	)
}

func stringsHasPrefixFunc(s, prefix string) Object {
	return Bool(strings.HasPrefix(s, prefix))
}

func stringsHasSuffixFunc(s, suffix string) Object {
	return Bool(strings.HasSuffix(s, suffix))
}

func stringsIndexFunc(s, substr string) Object {
	return Int(strings.Index(s, substr))
}

func stringsIndexAnyFunc(s, chars string) Object {
	return Int(strings.IndexAny(s, chars))
}

func stringsIndexByteFunc(s string, c rune) Object {
	if c > 255 || c < 0 {
		return Int(-1)
	}
	return Int(strings.IndexByte(s, byte(c)))
}

func stringsIndexCharFunc(s string, c rune) Object {
	return Int(strings.IndexRune(s, c))
}

func stringsJoinFunc(arr Array, sep string) Object {
	elems := make([]string, len(arr))
	for i := range arr {
		elems[i] = arr[i].ToString()
	}
	return Str(strings.Join(elems, sep))
}

func stringsJoinAndFunc(arr Array, sep, lastSep string) Object {
	switch len(arr) {
	case 0:
		return Str("")
	case 1:
		return Str(arr[0].ToString())
	default:
		last := len(arr) - 1
		elems := make([]string, last)
		for i := range elems {
			elems[i] = arr[i].ToString()
		}

		return Str(strings.Join(elems, sep) + lastSep + arr[last].ToString())
	}
}

func stringsLastIndexFunc(s, substr string) Object {
	return Int(strings.LastIndex(s, substr))
}

func stringsLastIndexAnyFunc(s, chars string) Object {
	return Int(strings.LastIndexAny(s, chars))
}

func stringsLastIndexByteFunc(s string, c rune) Object {
	if c > 255 || c < 0 {
		return Int(-1)
	}
	return Int(strings.LastIndexByte(s, byte(c)))
}

func stringsMapFuncInv(c Call) (Object, error) {
	return stringsStringInvoke(c, 1, 0,
		func(s string, inv *Invoker) (Object, error) {
			var err error
			out := strings.Map(func(r rune) rune {
				if err != nil {
					return utf8.RuneError
				}
				var ret Object
				ret, err = inv.Invoke(Args{{Char(r)}}, nil)
				if err != nil {
					return 0
				}
				r, ok := ToGoRune(ret)
				if !ok {
					return utf8.RuneError
				}
				return r
			}, s)
			return Str(out), err
		},
	)
}

func stringsPad(c Call, left bool) (Object, error) {
	size := c.Args.Length()
	if size != 2 && size != 3 {
		return Nil,
			ErrWrongNumArguments.NewError("want=2..3 got=" + strconv.Itoa(size))
	}
	s := c.Args.Get(0).ToString()
	padLen, ok := ToGoInt(c.Args.Get(1))
	if !ok {
		return Nil,
			NewArgumentTypeError("2nd", "int", c.Args.Get(1).Type().Name())
	}
	diff := padLen - len(s)
	if diff <= 0 {
		return Str(s), nil
	}
	padWith := " "
	if size > 2 {
		if padWith = c.Args.Get(2).ToString(); len(padWith) == 0 {
			return Str(s), nil
		}
	}
	r := (diff-len(padWith))/len(padWith) + 2
	if r <= 0 {
		return Str(s), nil
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
	return Str(sb.String()), nil
}

func stringsRepeatFunc(s string, count int) Object {
	// if n is negative strings.repeat function panics
	if count < 0 {
		return Str("")
	}
	return Str(strings.Repeat(s, count))
}

func stringsReplaceFunc(c Call) (Object, error) {
	size := c.Args.Length()
	if size != 3 && size != 4 {
		return Nil,
			ErrWrongNumArguments.NewError("want=3..4 got=" + strconv.Itoa(size))
	}
	s := c.Args.Get(0).ToString()
	old := c.Args.Get(1).ToString()
	news := c.Args.Get(2).ToString()
	n := -1
	if size == 4 {
		v, ok := ToGoInt(c.Args.Get(3))
		if !ok {
			return Nil,
				NewArgumentTypeError("4th", "int", c.Args.Get(3).Type().Name())
		}
		n = v
	}
	return Str(strings.Replace(s, old, news, n)), nil
}

func stringsTitleFunc(s string) Object {
	//lint:ignore SA1019 Keep it for backward compatibility.
	return Str(strings.Title(s)) // nolint staticcheck Keep it for backward compatibility
}

func stringsToLowerFunc(s string) Object { return Str(strings.ToLower(s)) }

func stringsToTitleFunc(s string) Object { return Str(strings.ToTitle(s)) }

func stringsToUpperFunc(s string) Object { return Str(strings.ToUpper(s)) }

func stringsToValidUTF8Func(c Call) (Object, error) {
	size := c.Args.Length()
	if size != 1 && size != 2 {
		return Nil,
			ErrWrongNumArguments.NewError("want=1..2 got=" + strconv.Itoa(size))
	}
	s := c.Args.Get(0).ToString()
	var repl string
	if size == 2 {
		repl = c.Args.Get(1).ToString()
	}
	return Str(strings.ToValidUTF8(s, repl)), nil
}

func stringsTrimFunc(s, cutset string) Object {
	return Str(strings.Trim(s, cutset))
}

func stringsTrimLeftFunc(s, cutset string) Object {
	return Str(strings.TrimLeft(s, cutset))
}

func stringsTrimPrefixFunc(s, prefix string) Object {
	return Str(strings.TrimPrefix(s, prefix))
}

func stringsTrimRightFunc(s, cutset string) Object {
	return Str(strings.TrimRight(s, cutset))
}

func stringsTrimSpaceFunc(s string) Object {
	return Str(strings.TrimSpace(s))
}

func stringsTrimSuffixFunc(s, suffix string) Object {
	return Str(strings.TrimSuffix(s, suffix))
}

func stringsTruncFunc(s string, max int, emph string) Object {
	if s == "" || len(s) <= max {
		return Str(s)
	}

	return Str(string([]rune(s)[:max]) + emph)
}

func stringsNewSplitFunc(fn func(string, string, int) []string) CallableFunc {
	return func(c Call) (Object, error) {
		size := c.Args.Length()
		if size != 2 && size != 3 {
			return Nil,
				ErrWrongNumArguments.NewError("want=2..3 got=" + strconv.Itoa(size))
		}
		s := c.Args.Get(0).ToString()
		sep := c.Args.Get(1).ToString()
		n := -1
		if size == 3 {
			v, ok := ToGoInt(c.Args.Get(2))
			if !ok {
				return Nil,
					NewArgumentTypeError("3rd", "int", c.Args.Get(2).Type().Name())
			}
			n = v
		}
		strs := fn(s, sep, n)
		out := make(Array, 0, len(strs))
		for _, s := range strs {
			out = append(out, Str(s))
		}
		return out, nil
	}
}

func stringsNewIndexFuncInv(fn func(string, func(rune) bool) int) CallableFunc {
	return func(c Call) (Object, error) {
		return stringsStringInvoke(c, 0, 1,
			func(s string, inv *Invoker) (Object, error) {
				var err error
				out := fn(s, func(r rune) bool {
					if err != nil {
						return false
					}
					var ret Object
					ret, err = inv.Invoke(Args{{Char(r)}}, nil)
					if err != nil {
						return false
					}
					return !ret.IsFalsy()
				})
				return Int(out), err
			},
		)
	}
}

func stringsNewTrimFuncInv(fn func(string, func(rune) bool) string) CallableFunc {
	return func(c Call) (Object, error) {
		return stringsStringInvoke(c, 0, 1,
			func(s string, inv *Invoker) (Object, error) {
				var err error
				out := fn(s, func(r rune) bool {
					if err != nil {
						return false
					}
					var ret Object
					ret, err = inv.Invoke(Args{{Char(r)}}, nil)
					if err != nil {
						return false
					}
					return !ret.IsFalsy()
				})
				return Str(out), err
			},
		)
	}
}

func stringsStringInvoke(
	c Call,
	sidx int,
	cidx int,
	fn func(string, *Invoker) (Object, error),
) (Object, error) {
	err := c.Args.CheckLen(2)
	if err != nil {
		return Nil, err
	}

	str := c.Args.Get(sidx).ToString()
	callee := c.Args.Get(cidx)
	if !Callable(callee) {
		return Nil, ErrNotCallable
	}
	if c.VM == nil {
		if _, ok := callee.(*CompiledFunction); ok {
			return Nil, ErrNotCallable
		}
	}

	inv := NewInvoker(c.VM, callee)
	inv.Acquire()
	defer inv.Release()
	return fn(str, inv)
}
