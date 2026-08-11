// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

// This file carries the `gad:doc` documentation for the root builtin functions
// (those available without an import). `gaddoc api . samples/builtins_api.gad
// builtins` renders it as the documented Gad API stub `samples/builtins_api.gad`.
//
// It is a work in progress: the type-conversion builtins (str/int/…) and the
// meta/operator builtins (cast/wrap/is/implements/Class/binOp/…) are still to be
// documented from their implementations and VM tests.

// gad:doc
// # builtins module
//
// Gad's **builtin functions** are available in every script without an
// `import`. This page documents the builtins whose signatures are settled; the
// remaining conversion, meta and operator builtins are still being typed.
//
// ## Functions
//
// len(val any; check=no) <int>
// Returns the number of elements of a value that has a length (array, dict,
// str, bytes, …). With `check=yes`, a value that has no length raises
// `ErrNotLengther` instead of returning 0.
//
// cap(o any) <int>
// Returns the capacity of an array or bytes value (0 for values without one).
//
// typeName(o any) <str>
// Returns the type name of a value (e.g. "int", "array", "dict").
//
// typeof(o any) <type>
// Returns the type object of a value.
//
// chars(s any) <array | error>
// Returns the characters of a str/bytes as an array of `char`, or an error for
// an unsupported value.
//
// copy(o any) <any>
// Returns a shallow copy of a value (a new array/dict with the same elements).
//
// dcopy(o any) <any>
// Returns a deep copy of a value, cloning nested arrays and dicts recursively.
//
// repeat(o any, count int) <any | error>
// Returns a value (array/str/bytes) repeated `count` times, or an error for an
// unsupported value.
//
// contains(o any, val any) <bool | error>
// Reports whether a collection/str contains val (a dict key, an array element
// or a substring), or an error for an unsupported value.
//
// repr(o any; indent=no) <str>
// Returns the debug representation of a value; `indent=yes` pretty-prints it.
//
// isNil(o any) <bool>
// Reports whether o is nil.
//
// isInt(o any) <bool>
// Reports whether o is an int.
//
// isUint(o any) <bool>
// Reports whether o is a uint.
//
// isFloat(o any) <bool>
// Reports whether o is a float.
//
// isChar(o any) <bool>
// Reports whether o is a char.
//
// isBool(o any) <bool>
// Reports whether o is a bool.
//
// isStr(o any) <bool>
// Reports whether o is a str.
//
// isRawStr(o any) <bool>
// Reports whether o is a rawStr.
//
// isBytes(o any) <bool>
// Reports whether o is a bytes value.
//
// isArray(o any) <bool>
// Reports whether o is an array.
//
// isDict(o any) <bool>
// Reports whether o is a dict.
//
// isSyncDict(o any) <bool>
// Reports whether o is a syncDict.
//
// isFunction(o any) <bool>
// Reports whether o is a function value.
//
// isCallable(o any) <bool>
// Reports whether o can be called.
//
// isIterable(o any) <bool>
// Reports whether o can be iterated.
//
// isIterator(o any) <bool>
// Reports whether o is an iterator.
//
// isError(o any) <bool>
// Reports whether o is an error value.
//
// filter(iterable any, callback any) <iterator>
// Returns a lazy iterator over the elements of iterable for which callback
// returns a truthy value.
//
// map(iterable any, callback any; update=no, nokey=no) <iterator>
// Returns a lazy iterator applying callback to each element of iterable.
// `update=yes` replaces elements in place; `nokey=yes` passes only the value to
// the callback.
//
// each(iterable any, callback any) <any>
// Calls callback for every element of iterable (for its side effects) and
// returns the iterable.
//
// reduce(iterable any, callback any, initial any) <any>
// Folds the elements of iterable with callback into a single value, starting
// from initial (or the first element when initial is omitted).
//
// keys(o any) <iterator>
// Returns a lazy iterator over the keys of a value.
//
// values(o any) <iterator>
// Returns a lazy iterator over the values of a value.
//
// items(o any) <iterator>
// Returns a lazy iterator over the key/value items of a value.
//
// iterate(o any) <iterator>
// Returns an iterator over a value.
//
// enumerate(o any) <iterator>
// Returns a lazy iterator yielding each element paired with its index.
//
// collect(o any) <array>
// Consumes an iterator or iterable into an array.
//
// toArray(*args) <array>
// Returns its arguments collected into an array.
//
// sort(o any; less=nil) <any | error>
// Returns the collection sorted ascending; `less` is an optional comparator
// function `less(a, b) <bool>`.
//
// sortReverse(o any; less=nil) <any | error>
// Returns the collection sorted descending; `less` is an optional comparator.
//
// print(*args) <int>
// Writes its arguments to standard output and returns the number of bytes
// written.
//
// printf(format str, *args) <int>
// Writes format applied to args to standard output and returns the byte count.
//
// println(*args) <int>
// Writes its arguments and a trailing newline to standard output.
//
// sprintf(format str, *args) <str>
// Returns format applied to args as a str.
