
# `builtins` module

Gad's **builtin functions** are available in every script without an
`import`. This page documents the builtins whose signatures are settled; the
remaining conversion, meta and operator builtins are still being typed.

## Public API

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
chars(s any) <_ array|error>
```

Returns the characters of a str/bytes as an array of `char`, or an error for
an unsupported value.

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
repeat(o any, count int) <_ any|error>
```

Returns a value (array/str/bytes) repeated `count` times, or an error for an
unsupported value.

### contains

```gad
contains(o any, val any) <_ bool|error>
```

Reports whether a collection/str contains val (a dict key, an array element
or a substring), or an error for an unsupported value.

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
sort(o any; less=nil) <_ any|error>
```

Returns the collection sorted ascending; `less` is an optional comparator
function `less(a, b) <bool>`.

### sortReverse

```gad
sortReverse(o any; less=nil) <_ any|error>
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

## Example — `builtins_api.gad`

```gad
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
Returns the characters of a str/bytes as an array of `char`, or an error for
an unsupported value.
**/
export chars(s any) <_ array|error> => nil

/**
Returns a shallow copy of a value (a new array/dict with the same elements).
**/
export copy(o any) <any> => nil

/**
Returns a deep copy of a value, cloning nested arrays and dicts recursively.
**/
export dcopy(o any) <any> => nil

/**
Returns a value (array/str/bytes) repeated `count` times, or an error for an
unsupported value.
**/
export repeat(o any, count int) <_ any|error> => nil

/**
Reports whether a collection/str contains val (a dict key, an array element
or a substring), or an error for an unsupported value.
**/
export contains(o any, val any) <_ bool|error> => nil

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
export sort(o any; less=nil) <_ any|error> => nil

/**
Returns the collection sorted descending; `less` is an optional comparator.
**/
export sortReverse(o any; less=nil) <_ any|error> => nil

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
```
