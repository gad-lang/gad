
# `builtins` module

Gad's **builtin functions** are available in every script without an
`import`. This page documents the builtins whose signatures are settled; the
remaining conversion, meta and operator builtins are still being typed.

## Public API

### str

```gad
str(v any) <str>
```

Converts a value to its str form (never fails).

### rawstr

```gad
rawstr(v any) <rawstr>
```

Converts a value to a rawstr (an uninterpreted string).

### int

```gad
int(v any) <int>
```

Converts a str/number/char/bool to an int; throws otherwise.

### uint

```gad
uint(v any) <uint>
```

Converts a value to a uint; throws otherwise.

### float

```gad
float(v any) <float>
```

Converts a str/number to a float; throws otherwise.

### decimal

```gad
decimal(v any) <decimal>
```

Converts a str/number to an arbitrary-precision decimal; throws otherwise.

### char

```gad
char(v any) <char>
```

Converts an int code point or a single-character str to a char; throws
otherwise.

### bool

```gad
bool(v any) <bool>
```

Returns the truthiness of a value (never fails).

### bytes

```gad
bytes(v any) <bytes>
```

Converts a str to its bytes, or builds a bytes of the given int length.

### array

```gad
array(*args) <array>
```

Collects its arguments into a new array.

### dict

```gad
dict(o; **named) <dict>
```

Builds a dict from named arguments, or from a dict-like value `o`.

### len

```gad
len(val any; check)
```

Returns the number of elements of a value that has a length (array, dict,
str, bytes, …). With `check=yes`, a value that has no length raises
`ErrNotLengther` instead of returning 0.

### cap

```gad
cap(o any) <int>
```

Returns the capacity of an array or bytes value (0 for values without one).

### typeName

```gad
typeName(o any) <str>
```

Returns the type name of a value (e.g. "int", "array", "dict").

### typeof

```gad
typeof(o any) <type>
```

Returns the type object of a value.

### chars

```gad
chars(s str|bytes) <array>
```

Returns the characters of a str/bytes as an array of `char`; throws for an
unsupported value.

### copy

```gad
copy(o any) <any>
```

Returns a shallow copy of a value (a new array/dict with the same elements).

### dcopy

```gad
dcopy(o any) <any>
```

Returns a deep copy of a value, cloning nested arrays and dicts recursively.

### repeat

```gad
repeat(o str|bytes|array, count int) <any>
```

Returns a value (array/str/bytes) repeated `count` times; throws for an
unsupported value.

### contains

```gad
contains(o iterable, val any) <bool>
```

Reports whether a collection/str contains val (a dict key, an array element
or a substring); throws for an unsupported value.

### repr

```gad
repr(o any; indent=no) <str>
```

Returns the debug representation of a value; `indent=yes` pretty-prints it.

### isNil

```gad
isNil(o any) <bool>
```

Reports whether o is nil.

### isInt

```gad
isInt(o any) <bool>
```

Reports whether o is an int.

### isUint

```gad
isUint(o any) <bool>
```

Reports whether o is a uint.

### isFloat

```gad
isFloat(o any) <bool>
```

Reports whether o is a float.

### isChar

```gad
isChar(o any) <bool>
```

Reports whether o is a char.

### isBool

```gad
isBool(o any) <bool>
```

Reports whether o is a bool.

### isStr

```gad
isStr(o any) <bool>
```

Reports whether o is a str.

### isRawStr

```gad
isRawStr(o any) <bool>
```

Reports whether o is a rawStr.

### isBytes

```gad
isBytes(o any) <bool>
```

Reports whether o is a bytes value.

### isArray

```gad
isArray(o any) <bool>
```

Reports whether o is an array.

### isDict

```gad
isDict(o any) <bool>
```

Reports whether o is a dict.

### isSyncDict

```gad
isSyncDict(o any) <bool>
```

Reports whether o is a syncDict.

### isFunction

```gad
isFunction(o any) <bool>
```

Reports whether o is a function value.

### isCallable

```gad
isCallable(o any) <bool>
```

Reports whether o can be called.

### isIterable

```gad
isIterable(o any) <bool>
```

Reports whether o can be iterated.

### isIterator

```gad
isIterator(o any) <bool>
```

Reports whether o is an iterator.

### isError

```gad
isError(o any) <bool>
```

Reports whether o is an error value.

### filter

```gad
filter(it iterable, callback callable) <iterator>
```

Returns a lazy iterator over the elements of iterable for which callback
returns a truthy value.

### map

```gad
map(it iterable, callback callable; update=no, nokey=no) <iterator>
```

Returns a lazy iterator applying callback to each element of iterable.
`update=yes` replaces elements in place; `nokey=yes` passes only the value to
the callback.

### each

```gad
each(it iterable, callback callable) <any>
```

Calls callback for every element of iterable (for its side effects) and
returns the iterable.

### reduce

```gad
reduce(it iterable, callback callable, initial any) <any>
```

Folds the elements of iterable with callback into a single value, starting
from initial (or the first element when initial is omitted).

### keys

```gad
keys(it iterable) <iterator>
```

Returns a lazy iterator over the keys of a value.

### values

```gad
values(it iterable) <iterator>
```

Returns a lazy iterator over the values of a value.

### items

```gad
items(it iterable) <iterator>
```

Returns a lazy iterator over the key/value items of a value.

### iterate

```gad
iterate(it iterable) <iterator>
```

Returns an iterator over a value.

### enumerate

```gad
enumerate(it iterable) <iterator>
```

Returns a lazy iterator yielding each element paired with its index.

### collect

```gad
collect(it iterable) <array>
```

Consumes an iterator or iterable into an array.

### toArray

```gad
toArray(*args) <array>
```

Returns its arguments collected into an array.

### sort

```gad
sort(o any; less=nil) <any>
```

Returns the collection sorted ascending; `less` is an optional comparator
function `less(a, b) <bool>`.

### sortReverse

```gad
sortReverse(o any; less=nil) <any>
```

Returns the collection sorted descending; `less` is an optional comparator.

### print

```gad
print(*args) <int>
```

Writes its arguments to standard output and returns the number of bytes
written.

### printf

```gad
printf(format str, *args) <int>
```

Writes format applied to args to standard output and returns the byte count.

### println

```gad
println(*args) <int>
```

Writes its arguments and a trailing newline to standard output.

### sprintf

```gad
sprintf(format str, *args) <str>
```

Returns format applied to args as a str.

### is

```gad
is(type any, *values) <bool>
```

Reports whether every value is of `type`. `type` may be a single type or an
array of types, in which case a value matches when it is any of them.

### implements

```gad
implements(fn callable, mi) <bool>
```

Reports whether the callable `fn` provides every function header required by
the method interface `mi` (a `meti { … }` value).

### wrap

```gad
wrap(caller callable, *args; **named) <function>
```

Returns a new function that calls `caller` with `args`/`named` prepended —
a partial application. Calling the wrapper appends its own arguments.

### cast

```gad
cast(toType type, obj any) <any>
```

Casts `obj` (an object that supports casting — a class instance or a reflected
Go value) to the object type `toType`, throwing when incompatible. For the
general checked cast that also accepts interfaces and unions, use the `::`
operator.

### userData

```gad
userData(o any) <any>
```

Returns the host-attached user data of a value that carries it (a Go value
implementing UserDataStorage); throws otherwise.

### Class

```gad
Class(name str, define callable) <classType>
```

Creates a class named `name`; `define` builds its fields, methods and
properties (see the Classes chapter). Also written with the `class` keyword.
It returns a `classType`; calling that class type — `classType(…)` — yields a
`classInstance`.

### addMethod

```gad
addMethod(target callable, *methods) <any>
```

Attaches typed method overloads to a callable or type, so the VM dispatches on
argument types. Returns the target.

### obstart

```gad
obstart() <buffer>
```

Starts capturing standard output into a fresh buffer, which it returns.
Nested calls stack.

### obend

```gad
obend() <buffer>
```

Stops the most recent output capture and returns its buffer (the captured
output).

### read

```gad
read(r readable) <bytes>
```

Reads all remaining bytes from a `readable` value.

### write

```gad
write(w writable, *data) <int>
```

Writes each data value to a `writable` and returns the number of bytes written.

### close

```gad
close(o)
```

Closes a closable value (e.g. a reader/writer).

### flush

```gad
flush(w writable)
```

Flushes any buffered output of a `writable`.

## Example — `builtins.gad`

```gad
/**
Converts a value to its str form (never fails).
**/
export str(v any) <str> => nil

/**
Converts a value to a rawstr (an uninterpreted string).
**/
export rawstr(v any) <rawstr> => nil

/**
Converts a str/number/char/bool to an int; throws otherwise.
**/
export int(v any) <int> => nil

/**
Converts a value to a uint; throws otherwise.
**/
export uint(v any) <uint> => nil

/**
Converts a str/number to a float; throws otherwise.
**/
export float(v any) <float> => nil

/**
Converts a str/number to an arbitrary-precision decimal; throws otherwise.
**/
export decimal(v any) <decimal> => nil

/**
Converts an int code point or a single-character str to a char; throws
otherwise.
**/
export char(v any) <char> => nil

/**
Returns the truthiness of a value (never fails).
**/
export bool(v any) <bool> => nil

/**
Converts a str to its bytes, or builds a bytes of the given int length.
**/
export bytes(v any) <bytes> => nil

/**
Collects its arguments into a new array.
**/
export array(*args) <array> => nil

/**
Builds a dict from named arguments, or from a dict-like value `o`.
**/
export dict(o; **named) <dict> => nil

/**
Returns the number of elements of a value that has a length (array, dict,
str, bytes, …). With `check=yes`, a value that has no length raises
`ErrNotLengther` instead of returning 0.
**/
export len(val any; check) => nil

/**
Returns the capacity of an array or bytes value (0 for values without one).
**/
export cap(o any) <int> => nil

/**
Returns the type name of a value (e.g. "int", "array", "dict").
**/
export typeName(o any) <str> => nil

/**
Returns the type object of a value.
**/
export typeof(o any) <type> => nil

/**
Returns the characters of a str/bytes as an array of `char`; throws for an
unsupported value.
**/
export chars(s str|bytes) <array> => nil

/**
Returns a shallow copy of a value (a new array/dict with the same elements).
**/
export copy(o any) <any> => nil

/**
Returns a deep copy of a value, cloning nested arrays and dicts recursively.
**/
export dcopy(o any) <any> => nil

/**
Returns a value (array/str/bytes) repeated `count` times; throws for an
unsupported value.
**/
export repeat(o str|bytes|array, count int) <any> => nil

/**
Reports whether a collection/str contains val (a dict key, an array element
or a substring); throws for an unsupported value.
**/
export contains(o iterable, val any) <bool> => nil

/**
Returns the debug representation of a value; `indent=yes` pretty-prints it.
**/
export repr(o any; indent=no) <str> => nil

/**
Reports whether o is nil.
**/
export isNil(o any) <bool> => nil

/**
Reports whether o is an int.
**/
export isInt(o any) <bool> => nil

/**
Reports whether o is a uint.
**/
export isUint(o any) <bool> => nil

/**
Reports whether o is a float.
**/
export isFloat(o any) <bool> => nil

/**
Reports whether o is a char.
**/
export isChar(o any) <bool> => nil

/**
Reports whether o is a bool.
**/
export isBool(o any) <bool> => nil

/**
Reports whether o is a str.
**/
export isStr(o any) <bool> => nil

/**
Reports whether o is a rawStr.
**/
export isRawStr(o any) <bool> => nil

/**
Reports whether o is a bytes value.
**/
export isBytes(o any) <bool> => nil

/**
Reports whether o is an array.
**/
export isArray(o any) <bool> => nil

/**
Reports whether o is a dict.
**/
export isDict(o any) <bool> => nil

/**
Reports whether o is a syncDict.
**/
export isSyncDict(o any) <bool> => nil

/**
Reports whether o is a function value.
**/
export isFunction(o any) <bool> => nil

/**
Reports whether o can be called.
**/
export isCallable(o any) <bool> => nil

/**
Reports whether o can be iterated.
**/
export isIterable(o any) <bool> => nil

/**
Reports whether o is an iterator.
**/
export isIterator(o any) <bool> => nil

/**
Reports whether o is an error value.
**/
export isError(o any) <bool> => nil

/**
Returns a lazy iterator over the elements of iterable for which callback
returns a truthy value.
**/
export filter(it iterable, callback callable) <iterator> => nil

/**
Returns a lazy iterator applying callback to each element of iterable.
`update=yes` replaces elements in place; `nokey=yes` passes only the value to
the callback.
**/
export map(it iterable, callback callable; update=no, nokey=no) <iterator> => nil

/**
Calls callback for every element of iterable (for its side effects) and
returns the iterable.
**/
export each(it iterable, callback callable) <any> => nil

/**
Folds the elements of iterable with callback into a single value, starting
from initial (or the first element when initial is omitted).
**/
export reduce(it iterable, callback callable, initial any) <any> => nil

/**
Returns a lazy iterator over the keys of a value.
**/
export keys(it iterable) <iterator> => nil

/**
Returns a lazy iterator over the values of a value.
**/
export values(it iterable) <iterator> => nil

/**
Returns a lazy iterator over the key/value items of a value.
**/
export items(it iterable) <iterator> => nil

/**
Returns an iterator over a value.
**/
export iterate(it iterable) <iterator> => nil

/**
Returns a lazy iterator yielding each element paired with its index.
**/
export enumerate(it iterable) <iterator> => nil

/**
Consumes an iterator or iterable into an array.
**/
export collect(it iterable) <array> => nil

/**
Returns its arguments collected into an array.
**/
export toArray(*args) <array> => nil

/**
Returns the collection sorted ascending; `less` is an optional comparator
function `less(a, b) <bool>`.
**/
export sort(o any; less=nil) <any> => nil

/**
Returns the collection sorted descending; `less` is an optional comparator.
**/
export sortReverse(o any; less=nil) <any> => nil

/**
Writes its arguments to standard output and returns the number of bytes
written.
**/
export print(*args) <int> => nil

/**
Writes format applied to args to standard output and returns the byte count.
**/
export printf(format str, *args) <int> => nil

/**
Writes its arguments and a trailing newline to standard output.
**/
export println(*args) <int> => nil

/**
Returns format applied to args as a str.
**/
export sprintf(format str, *args) <str> => nil

/**
Reports whether every value is of `type`. `type` may be a single type or an
array of types, in which case a value matches when it is any of them.
**/
export is(type any, *values) <bool> => nil

/**
Reports whether the callable `fn` provides every function header required by
the method interface `mi` (a `meti { … }` value).
**/
export implements(fn callable, mi) <bool> => nil

/**
Returns a new function that calls `caller` with `args`/`named` prepended —
a partial application. Calling the wrapper appends its own arguments.
**/
export wrap(caller callable, *args; **named) <function> => nil

/**
Casts `obj` (an object that supports casting — a class instance or a reflected
Go value) to the object type `toType`, throwing when incompatible. For the
general checked cast that also accepts interfaces and unions, use the `::`
operator.
**/
export cast(toType type, obj any) <any> => nil

/**
Returns the host-attached user data of a value that carries it (a Go value
implementing UserDataStorage); throws otherwise.
**/
export userData(o any) <any> => nil

/**
Creates a class named `name`; `define` builds its fields, methods and
properties (see the Classes chapter). Also written with the `class` keyword.
It returns a `classType`; calling that class type — `classType(…)` — yields a
`classInstance`.
**/
export Class(name str, define callable) <classType> => nil

/**
Attaches typed method overloads to a callable or type, so the VM dispatches on
argument types. Returns the target.
**/
export addMethod(target callable, *methods) <any> => nil

/**
Starts capturing standard output into a fresh buffer, which it returns.
Nested calls stack.
**/
export obstart() <buffer> => nil

/**
Stops the most recent output capture and returns its buffer (the captured
output).
**/
export obend() <buffer> => nil

/**
Reads all remaining bytes from a `readable` value.
**/
export read(r readable) <bytes> => nil

/**
Writes each data value to a `writable` and returns the number of bytes written.
**/
export write(w writable, *data) <int> => nil

/**
Closes a closable value (e.g. a reader/writer).
**/
export close(o) => nil

/**
Flushes any buffered output of a `writable`.
**/
export flush(w writable) => nil
```
