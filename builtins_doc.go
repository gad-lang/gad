// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

// This file carries the `gad:doc` documentation for the root builtin functions
// (those available without an import). `gaddoc api . samples/builtins.gad builtins`
// renders it as the documented Gad API stub `samples/builtins.gad`.
//
// Documented: the type constructors (str/int/…), the container/iteration/I/O
// builtins, the type predicates (is*), and the meta builtins (is/cast/wrap/
// implements/Class/…). Left out: the user-operator dispatch builtins
// (binOp/unOp/selfAssignOp — see the User Operators chapter), `stdio`, and the
// internal builtins (namedParamTypeCheck/methodFromArgs/rawCaller/vmPushWriter/
// vmPopWriter/iteratorInput). `enter`/`exit` live in the `gad` namespace.

// gad:doc
// # builtins module
//
// Gad's **builtin functions** are available in every script without an
// `import`. This page documents the builtins whose signatures are settled; the
// remaining conversion, meta and operator builtins are still being typed.
//
// ## Functions
//
// Type constructors — each named type is also a conversion function: it coerces
// a value to that type, throwing when the value cannot be converted.
//
// str(v any) <str>
// Converts a value to its str form (never fails).
//
// rawstr(v any) <rawstr>
// Converts a value to a rawstr (an uninterpreted string).
//
// int(v any) <int>
// Converts a str/number/char/bool to an int; throws otherwise.
//
// uint(v any) <uint>
// Converts a value to a uint; throws otherwise.
//
// float(v any) <float>
// Converts a str/number to a float; throws otherwise.
//
// decimal(v any) <decimal>
// Converts a str/number to an arbitrary-precision decimal; throws otherwise.
//
// char(v any) <char>
// Converts an int code point or a single-character str to a char; throws
// otherwise.
//
// bool(v any) <bool>
// Returns the truthiness of a value (never fails).
//
// bytes(v any) <bytes>
// Converts a str to its bytes, or builds a bytes of the given int length.
//
// array(*args) <array>
// Collects its arguments into a new array.
//
// dict(o; **named) <dict>
// Builds a dict from named arguments, or from a dict-like value `o`.
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
// chars(s str|bytes) <array>
// Returns the characters of a str/bytes as an array of `char`; throws for an
// unsupported value.
//
// copy(o any) <any>
// Returns a shallow copy of a value (a new array/dict with the same elements).
//
// dcopy(o any) <any>
// Returns a deep copy of a value, cloning nested arrays and dicts recursively.
//
// repeat(o str|bytes|array, count int) <any>
// Returns a value (array/str/bytes) repeated `count` times; throws for an
// unsupported value.
//
// contains(o iterable, val any) <bool>
// Reports whether a collection/str contains val (a dict key, an array element
// or a substring); throws for an unsupported value.
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
// filter(it iterable, callback callable) <iterator>
// Returns a lazy iterator over the elements of iterable for which callback
// returns a truthy value.
//
// map(it iterable, callback callable; update=no, nokey=no) <iterator>
// Returns a lazy iterator applying callback to each element of iterable.
// `update=yes` replaces elements in place; `nokey=yes` passes only the value to
// the callback.
//
// each(it iterable, callback callable) <any>
// Calls callback for every element of iterable (for its side effects) and
// returns the iterable.
//
// reduce(it iterable, callback callable, initial any) <any>
// Folds the elements of iterable with callback into a single value, starting
// from initial (or the first element when initial is omitted).
//
// keys(it iterable) <iterator>
// Returns a lazy iterator over the keys of a value.
//
// values(it iterable) <iterator>
// Returns a lazy iterator over the values of a value.
//
// items(it iterable) <iterator>
// Returns a lazy iterator over the key/value items of a value.
//
// iterate(it iterable) <iterator>
// Returns an iterator over a value.
//
// enumerate(it iterable) <iterator>
// Returns a lazy iterator yielding each element paired with its index.
//
// collect(it iterable) <array>
// Consumes an iterator or iterable into an array.
//
// toArray(*args) <array>
// Returns its arguments collected into an array.
//
// sort(o any; less=nil) <any>
// Returns the collection sorted ascending; `less` is an optional comparator
// function `less(a, b) <bool>`.
//
// sortReverse(o any; less=nil) <any>
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
//
// is(type any, *values) <bool>
// Reports whether every value is of `type`. `type` may be a single type or an
// array of types, in which case a value matches when it is any of them.
//
// implements(fn callable, mi) <bool>
// Reports whether the callable `fn` provides every function header required by
// the method interface `mi` (a `meti { … }` value).
//
// wrap(caller callable, *args; **named) <function>
// Returns a new function that calls `caller` with `args`/`named` prepended —
// a partial application. Calling the wrapper appends its own arguments.
//
// cast(toType type, obj any) <any>
// Casts `obj` (an object that supports casting — a class instance or a reflected
// Go value) to the object type `toType`, throwing when incompatible. For the
// general checked cast that also accepts interfaces and unions, use the `::`
// operator.
//
// userData(o any) <any>
// Returns the host-attached user data of a value that carries it (a Go value
// implementing UserDataStorage); throws otherwise.
//
// Class(name str, define callable) <classType>
// Creates a class named `name`; `define` builds its fields, methods and
// properties (see the Classes chapter). Also written with the `class` keyword.
// It returns a `classType`; calling that class type — `classType(…)` — yields a
// `classInstance`.
//
// addMethod(target callable, *methods) <any>
// Attaches typed method overloads to a callable or type, so the VM dispatches on
// argument types. Returns the target.
//
// obstart() <buffer>
// Starts capturing standard output into a fresh buffer, which it returns.
// Nested calls stack.
//
// obend() <buffer>
// Stops the most recent output capture and returns its buffer (the captured
// output).
//
// read(r readable) <bytes>
// Reads all remaining bytes from a `readable` value.
//
// write(w writable, *data) <int>
// Writes each data value to a `writable` and returns the number of bytes written.
//
// close(o) <nil>
// Closes a closable value (e.g. a reader/writer).
//
// flush(w writable) <nil>
// Flushes any buffered output of a `writable`.
