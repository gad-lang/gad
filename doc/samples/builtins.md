
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
Example:
```gad
str(12)
= "12"
[str([1, "a"]), str(nil)]
= ["[1, \"a\"]", "nil"]
```

### rawstr

```gad
rawstr(v any) <rawstr>
```

Converts a value to a rawstr (an uninterpreted string).
Example:
```gad
isRawStr(rawstr("a"))
= true
```

### int

```gad
int(v any) <int>
```

Converts a str/number/char/bool to an int; throws otherwise.
Example:
```gad
[int("42"), int(4.9), int('A'), int(true)]
= [42, 4, 65, 1]
```

### uint

```gad
uint(v any) <uint>
```

Converts a value to a uint; throws otherwise.
Example:
```gad
uint(3)
= 3
```

### float

```gad
float(v any) <float>
```

Converts a str/number to a float; throws otherwise.
Example:
```gad
float("1.5")
= 1.5
```

### decimal

```gad
decimal(v any) <decimal>
```

Converts a str/number to an arbitrary-precision decimal; throws otherwise.
Example:
```gad
decimal("1.10")
= 1.1
```

### char

```gad
char(v any) <char>
```

Converts an int code point or a single-character str to a char; throws
otherwise.
Example:
```gad
char(65)
= 'A'
```

### bool

```gad
bool(v any) <bool>
```

Returns the truthiness of a value (never fails).
Example:
```gad
[bool(1), bool(0), bool(""), bool("x")]
= [true, false, false, true]
```

### bytes

```gad
bytes(v any) <bytes>
```

Converts a str to its bytes, or builds a bytes of the given int length.
Example:
```gad
str(bytes("hi"))
= "hi"
```

### array

```gad
array(*args) <array>
```

Collects its arguments into a new array.
Example:
```gad
array(1, 2)
= [1, 2]
```

### dict

```gad
dict(o; **named) <dict>
```

Builds a dict from named arguments, or from a dict-like value `o`.
Example:
```gad
[dict(; a=1), dict((;b=2))]
= [{a: 1}, {b: 2}]
```

### len

```gad
len(val any; check)
```

Returns the number of elements of a value that has a length (array, dict,
str, bytes, …). With `check=yes`, a value that has no length raises
`ErrNotLengther` instead of returning 0.
Example:
```gad
[len("abc"), len([1, 2]), len({a: 1})]
= [3, 2, 1]
```

### cap

```gad
cap(o any) <int>
```

Returns the capacity of an array or bytes value (0 for values without one).
Example:
```gad
cap([1, 2]) >= 2
= true
```

### typeName

```gad
typeName(o any) <str>
```

Returns the type name of a value (e.g. "int", "array", "dict").
Example:
```gad
[typeName(1), typeName("a"), typeName([])]
= ["int", "str", "array"]
```

### typeof

```gad
typeof(o any) <type>
```

Returns the type object of a value.
Example:
```gad
[typeof(1) == int, typeof([1]) === array]
= [true, true]
```

### chars

```gad
chars(s str|bytes) <array>
```

Returns the characters of a str/bytes as an array of `char`; throws for an
unsupported value.
Example:
```gad
chars("ab")
= ['a', 'b']
```

### copy

```gad
copy(o any) <any>
```

Returns a shallow copy of a value (a new array/dict with the same elements).
Example:
```gad
a := [[1]]
b := copy(a)
b[0] === a[0]     // shallow: the inner array is shared
= true
```

### dcopy

```gad
dcopy(o any) <any>
```

Returns a deep copy of a value, cloning nested arrays and dicts recursively.
Example:
```gad
a := [[1]]
b := dcopy(a)
b[0] === a[0]     // deep: copied too
= false
```

### repeat

```gad
repeat(o str|bytes|array, count int) <any>
```

Returns a value (array/str/bytes) repeated `count` times; throws for an
unsupported value.
Example:
```gad
[repeat("ab", 2), repeat([1], 3)]
= ["abab", [1, 1, 1]]
```

### contains

```gad
contains(o iterable, val any) <bool>
```

Reports whether a collection/str contains val (a dict key, an array element
or a substring); throws for an unsupported value.
Example:
```gad
[contains([1, 2], 2), contains("hello", "ell"), contains({a: 1}, "a")]
= [true, true, true]
```

### repr

```gad
repr(o any; indent=no) <str>
```

Returns the debug representation of a value; `indent=yes` pretty-prints it.
Example:
```gad
repr("a")
= "‹str: \"a\"›"
```

### isNil

```gad
isNil(o any) <bool>
```

Reports whether o is nil.
Example:
```gad
[isNil(nil), isNil(0)]
= [true, false]
```

### isInt

```gad
isInt(o any) <bool>
```

Reports whether o is an int.
Example:
```gad
[isInt(1), isInt(1u)]
= [true, false]
```

### isUint

```gad
isUint(o any) <bool>
```

Reports whether o is a uint.
Example:
```gad
[isUint(1u), isUint(1)]
= [true, false]
```

### isFloat

```gad
isFloat(o any) <bool>
```

Reports whether o is a float.
Example:
```gad
[isFloat(1.0), isFloat(1)]
= [true, false]
```

### isChar

```gad
isChar(o any) <bool>
```

Reports whether o is a char.
Example:
```gad
[isChar('a'), isChar("a")]
= [true, false]
```

### isBool

```gad
isBool(o any) <bool>
```

Reports whether o is a bool.
Example:
```gad
[isBool(true), isBool(1)]
= [true, false]
```

### isStr

```gad
isStr(o any) <bool>
```

Reports whether o is a str.
Example:
```gad
[isStr("a"), isStr(`a`)]
= [true, false]
```

### isRawStr

```gad
isRawStr(o any) <bool>
```

Reports whether o is a rawStr.
Example:
```gad
[isRawStr(`a`), isRawStr("a")]
= [true, false]
```

### isBytes

```gad
isBytes(o any) <bool>
```

Reports whether o is a bytes value.
Example:
```gad
[isBytes(bytes("a")), isBytes("a")]
= [true, false]
```

### isArray

```gad
isArray(o any) <bool>
```

Reports whether o is an array.
Example:
```gad
[isArray([]), isArray({})]
= [true, false]
```

### isDict

```gad
isDict(o any) <bool>
```

Reports whether o is a dict.
Example:
```gad
[isDict({}), isDict([])]
= [true, false]
```

### isSyncDict

```gad
isSyncDict(o any) <bool>
```

Reports whether o is a syncDict.
Example:
```gad
[isSyncDict(syncDict({})), isSyncDict({})]
= [true, false]
```

### isFunction

```gad
isFunction(o any) <bool>
```

Reports whether o is a function value.
Example:
```gad
[isFunction(len), isFunction(() => 1), isFunction(1)]
= [true, true, false]
```

### isCallable

```gad
isCallable(o any) <bool>
```

Reports whether o can be called.
Example:
```gad
[isCallable(len), isCallable(int), isCallable(1)]
= [true, true, false]
```

### isIterable

```gad
isIterable(o any) <bool>
```

Reports whether o can be iterated.
Example:
```gad
[isIterable([]), isIterable("a"), isIterable(1)]
= [true, true, false]
```

### isIterator

```gad
isIterator(o any) <bool>
```

Reports whether o is an iterator.
Example:
```gad
[isIterator(iterate([])), isIterator([])]
= [true, false]
```

### isError

```gad
isError(o any) <bool>
```

Reports whether o is an error value.
Example:
```gad
[isError(error("x")), isError("x")]
= [true, false]
```

### filter

```gad
filter(it iterable, callback <(v any, k any, it iterable) <bool>>) <iterator>
```

Returns a lazy iterator over the elements of iterable for which callback
returns a truthy value. The callback is called as `callback(value, key,
iterable)` (key: the index or dict key).
Example:
```gad
collect(filter([1, 2, 3], (v) => v > 1))
= [2, 3]
collect(filter({a: 1, b: 2}, (v, k) => k == "b"))
= [2]
```

### map

```gad
map(it iterable, callback <(v any, k any) <any>>; update=no, nokey=no) <iterator>
```

Returns a lazy iterator applying callback to each element of iterable, called
as `callback(value, key)`; `nokey=yes` passes only the value. `update=yes`
also writes each result back into the collection — as the iterator is
consumed (e.g. by `collect`), since it is lazy.
Example:
```gad
collect(map([1, 2], (v) => v * 10))
= [10, 20]
dict(map({a: 1}, (v, k) => k + "=" + str(v)))   // the keys kept
= {a: "a=1"}
collect(map([1, 2], func(*a) => len(a); nokey))
= [1, 1]
```

### each

```gad
each(it iterable, callback <(k any, v any)>) <any>
```

Calls callback for every element of iterable (for its side effects) and
returns the iterable. The callback is called as `callback(key, value)` — key
first, unlike map/filter.
Example:
```gad
each({a: 1}, (k, v) => println(k, v))
<<< a 1
```

### reduce

```gad
reduce(it iterable, callback <(acc any, v any, k any) <any>>, initial any) <any>
```

Folds the elements of iterable with callback into a single value, starting
from initial (or the first element when initial is omitted). The callback is
called as `callback(accumulator, value, key)` and returns the next
accumulator.
Example:
```gad
reduce([1, 2, 3], (acc, v) => acc + v, 10)
= 16
reduce([1, 2, 3], (acc, v) => acc * v)   // the first element the initial
= 6
```

### keys

```gad
keys(it iterable) <iterator>
```

Returns a lazy iterator over the keys of a value.
Example:
```gad
collect(keys({a: 1}))
= ["a"]
```

### values

```gad
values(it iterable) <iterator>
```

Returns a lazy iterator over the values of a value.
Example:
```gad
collect(values({a: 1}))
= [1]
```

### items

```gad
items(it iterable) <iterator>
```

Returns a lazy iterator over the key/value items of a value.
Example:
```gad
collect(items({a: 1}))
= [[a=1]]
```

### iterate

```gad
iterate(it iterable) <iterator>
```

Returns an iterator over a value.
Example:
```gad
collect(iterate([1, 2]))
= [1, 2]
```

### enumerate

```gad
enumerate(it iterable) <iterator>
```

Returns a lazy iterator yielding each element paired with its index, as a
key-value `[(index)=value]` (read `.k` / `.v`).
Example:
```gad
[kv.k for kv in collect(enumerate(["a", "b"]))]
= [0, 1]
```

### collect

```gad
collect(it iterable) <array>
```

Consumes an iterator or iterable into an array.
Example:
```gad
collect(map([1, 2], (v) => v + 1))
= [2, 3]
```

### toArray

```gad
toArray(*its) <array>
```

Concatenates its arguments into one array: an array contributes its elements,
any other iterable its entries as key-value pairs `[(key)=value]`.
Example:
```gad
toArray([1], [2, 3])
= [1, 2, 3]
```

### sort

```gad
sort(o any; less <(a any, b any) <bool>>=nil) <any>
```

Returns the collection sorted ascending; `less` is an optional comparator
function `less(a, b) <bool>`.
Example:
```gad
sort([3, 1, 2])
= [1, 2, 3]
sort(["bb", "a"]; less=(a, b) => len(a) < len(b))
= ["a", "bb"]
```

### sortReverse

```gad
sortReverse(o any; less <(a any, b any) <bool>>=nil) <any>
```

Returns the collection sorted descending; `less` is an optional comparator.
Example:
```gad
sortReverse([1, 3, 2])
= [3, 2, 1]
```

### print

```gad
print(*args) <int>
```

Writes its arguments to standard output and returns the number of bytes
written.
Example:
```gad
print("a", 1)
= 3
<<< a 1
```

### printf

```gad
printf(format str, *args) <int>
```

Writes format applied to args to standard output and returns the byte count.
Example:
```gad
printf("%d!", 5)
= 2
<<< 5!
```

### println

```gad
println(*args) <int>
```

Writes its arguments and a trailing newline to standard output.
Example:
```gad
println("hi", 1)
= 5
<<< hi 1
```

### sprintf

```gad
sprintf(format str, *args) <str>
```

Returns format applied to args as a str.
Example:
```gad
sprintf("%s=%d", "n", 42)
= "n=42"
```

### is

```gad
is(type any, *values) <bool>
```

Reports whether every value is of `type`. `type` may be a single type or an
array of types, in which case a value matches when it is any of them.
Example:
```gad
[is(int, 1, 2), is(int, 1, "a"), is([int, str], 1, "a")]
= [true, false, true]
```

### implements

```gad
implements(fn callable, mi) <bool>
```

Reports whether the callable `fn` provides every function header required by
the method interface `mi` (a `meti { … }` value).
Example:
```gad
implements(func(x int) => x, meti { (x int) })
= true
```

### wrap

```gad
wrap(caller callable, *args; **named) <function>
```

Returns a new function that calls `caller` with `args`/`named` prepended —
a partial application. Calling the wrapper appends its own arguments.
Example:
```gad
add := func(a, b) => a + b
inc := wrap(add, 1)
inc(2)
= 3
```

### cast

```gad
cast(toType type, obj any) <any>
```

Casts `obj` (an object that supports casting — a class instance or a reflected
Go value) to the object type `toType`, throwing when incompatible. For the
general checked cast that also accepts interfaces and unions, use the `::`
operator.
Example:
```gad
// a host value or a class instance; anything else throws
r := func() { try { return cast(str, 1) } catch { return "not castable" } }
r()
= "not castable"
```

### userData

```gad
userData(o any) <any>
```

Returns the host-attached user data of a value that carries it (a Go value
implementing UserDataStorage); throws otherwise.
Example:
```gad
// a host value carrying user data; anything else throws
r := func() { try { return userData(1) } catch { return "no user data" } }
r()
= "no user data"
```

### stdio

```gad
stdio(which any) <any>
```

Returns a standard stream of the running VM selected by `which` — "IN"/"OUT"/
"ERR" or 0/1/2. "IN" is a `readable`; "OUT" and "ERR" are `writable`.
Example:
```gad
typeName(stdio("OUT"))
= "writer"
```

### Class

```gad
Class(name str, define <(cls classType, define callable) <any>>) <classType>
```

Creates a class named `name`; `define` builds its fields, methods and
properties (see the Classes chapter). Also written with the `class` keyword.
It returns a `classType`; calling that class type — `classType(…)` — yields a
`classInstance`.
Example:
```gad
P := Class("P", (cls, define) => define(; fields = (; x = (= 1))))
P().x
= 1
```

### addMethod

```gad
addMethod(target callable, *methods) <any>
```

Attaches typed method overloads to a callable or type, so the VM dispatches on
argument types. Returns the target.
Example:
```gad
kind := func { (x int) => "int" }
addMethod(kind, func(x str) => "str")
kind("a")
= "str"
```

### obstart

```gad
obstart() <buffer>
```

Starts capturing standard output into a fresh buffer, which it returns.
Nested calls stack.
Example:
```gad
obstart()
print("hidden")
str(obend())
= "hidden"
```

### obend

```gad
obend() <buffer>
```

Stops the most recent output capture and returns its buffer (the captured
output).
Example:
```gad
obstart()
println("a")
str(obend())
= "a\n"
```

### read

```gad
read(r readable) <bytes>
```

Reads all remaining bytes from a `readable` value.
Example:
```gad
str(read(buffer("xy")))
= "xy"
```

### write

```gad
write(w writable, *data) <int>
```

Writes each data value to a `writable` and returns the number of bytes written.
Example:
```gad
w := buffer()
write(w, "ab", 1)
= 3
str(w)
= "ab1"
```

### close

```gad
close(o)
```

Closes a closable value (e.g. a reader/writer).
Example:
```gad
w := buffer()
close(w)
isError(w)
= false
```

### flush

```gad
flush(w writable)
```

Flushes any buffered output of a `writable`.
Example:
```gad
w := buffer()
write(w, "a")
flush(w)
str(w)
= "a"
```

### iterator

```gad
iterator(it iterable) <iterator>
```

Builds an iterator over any iterable value (the type constructor form of
`iterate`).
Example:
```gad
collect(iterator([1, 2]))
= [1, 2]
```

### zip

```gad
zip(*iterables) <iterator>
```

Returns an iterator that chains the given iterables end to end: it yields
every element of the first, then of the second, and so on.
Example:
```gad
collect(zip([1], [2, 3]))
= [1, 2, 3]
```

### keyValue

```gad
keyValue(key any, value any) <keyValue>
```

Builds a single key/value pair value. A dict takes one with `+`:
`d += keyValue(k, v)` sets that entry. `++` is for a sequence of entries.
Example:
```gad
kv := keyValue("a", 1)
[kv.k, kv.v]
= ["a", 1]
```

### keyValueArray

```gad
keyValueArray(*pairs) <keyValueArray>
```

Builds an ordered key/value collection from `keyValue` pairs.
Example:
```gad
str(keyValueArray(keyValue("a", 1), keyValue("b", 2)))
= "(;a=1, b=2)"
```

## Example — `builtins.gad`

````gad
/**
Converts a value to its str form (never fails).
Example:
```gad
str(12)
= "12"
[str([1, "a"]), str(nil)]
= ["[1, \"a\"]", "nil"]
```
**/
export str(v any) <str> => nil

/**
Converts a value to a rawstr (an uninterpreted string).
Example:
```gad
isRawStr(rawstr("a"))
= true
```
**/
export rawstr(v any) <rawstr> => nil

/**
Converts a str/number/char/bool to an int; throws otherwise.
Example:
```gad
[int("42"), int(4.9), int('A'), int(true)]
= [42, 4, 65, 1]
```
**/
export int(v any) <int> => nil

/**
Converts a value to a uint; throws otherwise.
Example:
```gad
uint(3)
= 3
```
**/
export uint(v any) <uint> => nil

/**
Converts a str/number to a float; throws otherwise.
Example:
```gad
float("1.5")
= 1.5
```
**/
export float(v any) <float> => nil

/**
Converts a str/number to an arbitrary-precision decimal; throws otherwise.
Example:
```gad
decimal("1.10")
= 1.1
```
**/
export decimal(v any) <decimal> => nil

/**
Converts an int code point or a single-character str to a char; throws
otherwise.
Example:
```gad
char(65)
= 'A'
```
**/
export char(v any) <char> => nil

/**
Returns the truthiness of a value (never fails).
Example:
```gad
[bool(1), bool(0), bool(""), bool("x")]
= [true, false, false, true]
```
**/
export bool(v any) <bool> => nil

/**
Converts a str to its bytes, or builds a bytes of the given int length.
Example:
```gad
str(bytes("hi"))
= "hi"
```
**/
export bytes(v any) <bytes> => nil

/**
Collects its arguments into a new array.
Example:
```gad
array(1, 2)
= [1, 2]
```
**/
export array(*args) <array> => nil

/**
Builds a dict from named arguments, or from a dict-like value `o`.
Example:
```gad
[dict(; a=1), dict((;b=2))]
= [{a: 1}, {b: 2}]
```
**/
export dict(o; **named) <dict> => nil

/**
Returns the number of elements of a value that has a length (array, dict,
str, bytes, …). With `check=yes`, a value that has no length raises
`ErrNotLengther` instead of returning 0.
Example:
```gad
[len("abc"), len([1, 2]), len({a: 1})]
= [3, 2, 1]
```
**/
export len(val any; check) => nil

/**
Returns the capacity of an array or bytes value (0 for values without one).
Example:
```gad
cap([1, 2]) >= 2
= true
```
**/
export cap(o any) <int> => nil

/**
Returns the type name of a value (e.g. "int", "array", "dict").
Example:
```gad
[typeName(1), typeName("a"), typeName([])]
= ["int", "str", "array"]
```
**/
export typeName(o any) <str> => nil

/**
Returns the type object of a value.
Example:
```gad
[typeof(1) == int, typeof([1]) === array]
= [true, true]
```
**/
export typeof(o any) <type> => nil

/**
Returns the characters of a str/bytes as an array of `char`; throws for an
unsupported value.
Example:
```gad
chars("ab")
= ['a', 'b']
```
**/
export chars(s str|bytes) <array> => nil

/**
Returns a shallow copy of a value (a new array/dict with the same elements).
Example:
```gad
a := [[1]]
b := copy(a)
b[0] === a[0]     // shallow: the inner array is shared
= true
```
**/
export copy(o any) <any> => nil

/**
Returns a deep copy of a value, cloning nested arrays and dicts recursively.
Example:
```gad
a := [[1]]
b := dcopy(a)
b[0] === a[0]     // deep: copied too
= false
```
**/
export dcopy(o any) <any> => nil

/**
Returns a value (array/str/bytes) repeated `count` times; throws for an
unsupported value.
Example:
```gad
[repeat("ab", 2), repeat([1], 3)]
= ["abab", [1, 1, 1]]
```
**/
export repeat(o str|bytes|array, count int) <any> => nil

/**
Reports whether a collection/str contains val (a dict key, an array element
or a substring); throws for an unsupported value.
Example:
```gad
[contains([1, 2], 2), contains("hello", "ell"), contains({a: 1}, "a")]
= [true, true, true]
```
**/
export contains(o iterable, val any) <bool> => nil

/**
Returns the debug representation of a value; `indent=yes` pretty-prints it.
Example:
```gad
repr("a")
= "‹str: \"a\"›"
```
**/
export repr(o any; indent=no) <str> => nil

/**
Reports whether o is nil.
Example:
```gad
[isNil(nil), isNil(0)]
= [true, false]
```
**/
export isNil(o any) <bool> => nil

/**
Reports whether o is an int.
Example:
```gad
[isInt(1), isInt(1u)]
= [true, false]
```
**/
export isInt(o any) <bool> => nil

/**
Reports whether o is a uint.
Example:
```gad
[isUint(1u), isUint(1)]
= [true, false]
```
**/
export isUint(o any) <bool> => nil

/**
Reports whether o is a float.
Example:
```gad
[isFloat(1.0), isFloat(1)]
= [true, false]
```
**/
export isFloat(o any) <bool> => nil

/**
Reports whether o is a char.
Example:
```gad
[isChar('a'), isChar("a")]
= [true, false]
```
**/
export isChar(o any) <bool> => nil

/**
Reports whether o is a bool.
Example:
```gad
[isBool(true), isBool(1)]
= [true, false]
```
**/
export isBool(o any) <bool> => nil

/**
Reports whether o is a str.
Example:
```gad
[isStr("a"), isStr(`a`)]
= [true, false]
```
**/
export isStr(o any) <bool> => nil

/**
Reports whether o is a rawStr.
Example:
```gad
[isRawStr(`a`), isRawStr("a")]
= [true, false]
```
**/
export isRawStr(o any) <bool> => nil

/**
Reports whether o is a bytes value.
Example:
```gad
[isBytes(bytes("a")), isBytes("a")]
= [true, false]
```
**/
export isBytes(o any) <bool> => nil

/**
Reports whether o is an array.
Example:
```gad
[isArray([]), isArray({})]
= [true, false]
```
**/
export isArray(o any) <bool> => nil

/**
Reports whether o is a dict.
Example:
```gad
[isDict({}), isDict([])]
= [true, false]
```
**/
export isDict(o any) <bool> => nil

/**
Reports whether o is a syncDict.
Example:
```gad
[isSyncDict(syncDict({})), isSyncDict({})]
= [true, false]
```
**/
export isSyncDict(o any) <bool> => nil

/**
Reports whether o is a function value.
Example:
```gad
[isFunction(len), isFunction(() => 1), isFunction(1)]
= [true, true, false]
```
**/
export isFunction(o any) <bool> => nil

/**
Reports whether o can be called.
Example:
```gad
[isCallable(len), isCallable(int), isCallable(1)]
= [true, true, false]
```
**/
export isCallable(o any) <bool> => nil

/**
Reports whether o can be iterated.
Example:
```gad
[isIterable([]), isIterable("a"), isIterable(1)]
= [true, true, false]
```
**/
export isIterable(o any) <bool> => nil

/**
Reports whether o is an iterator.
Example:
```gad
[isIterator(iterate([])), isIterator([])]
= [true, false]
```
**/
export isIterator(o any) <bool> => nil

/**
Reports whether o is an error value.
Example:
```gad
[isError(error("x")), isError("x")]
= [true, false]
```
**/
export isError(o any) <bool> => nil

/**
Returns a lazy iterator over the elements of iterable for which callback
returns a truthy value. The callback is called as `callback(value, key,
iterable)` (key: the index or dict key).
Example:
```gad
collect(filter([1, 2, 3], (v) => v > 1))
= [2, 3]
collect(filter({a: 1, b: 2}, (v, k) => k == "b"))
= [2]
```
**/
export filter(it iterable, callback <(v any, k any, it iterable) <bool>>) <iterator> => nil

/**
Returns a lazy iterator applying callback to each element of iterable, called
as `callback(value, key)`; `nokey=yes` passes only the value. `update=yes`
also writes each result back into the collection — as the iterator is
consumed (e.g. by `collect`), since it is lazy.
Example:
```gad
collect(map([1, 2], (v) => v * 10))
= [10, 20]
dict(map({a: 1}, (v, k) => k + "=" + str(v)))   // the keys kept
= {a: "a=1"}
collect(map([1, 2], func(*a) => len(a); nokey))
= [1, 1]
```
**/
export map(it iterable, callback <(v any, k any) <any>>; update=no, nokey=no) <iterator> => nil

/**
Calls callback for every element of iterable (for its side effects) and
returns the iterable. The callback is called as `callback(key, value)` — key
first, unlike map/filter.
Example:
```gad
each({a: 1}, (k, v) => println(k, v))
<<< a 1
```
**/
export each(it iterable, callback <(k any, v any)>) <any> => nil

/**
Folds the elements of iterable with callback into a single value, starting
from initial (or the first element when initial is omitted). The callback is
called as `callback(accumulator, value, key)` and returns the next
accumulator.
Example:
```gad
reduce([1, 2, 3], (acc, v) => acc + v, 10)
= 16
reduce([1, 2, 3], (acc, v) => acc * v)   // the first element the initial
= 6
```
**/
export reduce(it iterable, callback <(acc any, v any, k any) <any>>, initial any) <any> => nil

/**
Returns a lazy iterator over the keys of a value.
Example:
```gad
collect(keys({a: 1}))
= ["a"]
```
**/
export keys(it iterable) <iterator> => nil

/**
Returns a lazy iterator over the values of a value.
Example:
```gad
collect(values({a: 1}))
= [1]
```
**/
export values(it iterable) <iterator> => nil

/**
Returns a lazy iterator over the key/value items of a value.
Example:
```gad
collect(items({a: 1}))
= [[a=1]]
```
**/
export items(it iterable) <iterator> => nil

/**
Returns an iterator over a value.
Example:
```gad
collect(iterate([1, 2]))
= [1, 2]
```
**/
export iterate(it iterable) <iterator> => nil

/**
Returns a lazy iterator yielding each element paired with its index, as a
key-value `[(index)=value]` (read `.k` / `.v`).
Example:
```gad
[kv.k for kv in collect(enumerate(["a", "b"]))]
= [0, 1]
```
**/
export enumerate(it iterable) <iterator> => nil

/**
Consumes an iterator or iterable into an array.
Example:
```gad
collect(map([1, 2], (v) => v + 1))
= [2, 3]
```
**/
export collect(it iterable) <array> => nil

/**
Concatenates its arguments into one array: an array contributes its elements,
any other iterable its entries as key-value pairs `[(key)=value]`.
Example:
```gad
toArray([1], [2, 3])
= [1, 2, 3]
```
**/
export toArray(*its) <array> => nil

/**
Returns the collection sorted ascending; `less` is an optional comparator
function `less(a, b) <bool>`.
Example:
```gad
sort([3, 1, 2])
= [1, 2, 3]
sort(["bb", "a"]; less=(a, b) => len(a) < len(b))
= ["a", "bb"]
```
**/
export sort(o any; less <(a any, b any) <bool>> = nil) <any> => nil

/**
Returns the collection sorted descending; `less` is an optional comparator.
Example:
```gad
sortReverse([1, 3, 2])
= [3, 2, 1]
```
**/
export sortReverse(o any; less <(a any, b any) <bool>> = nil) <any> => nil

/**
Writes its arguments to standard output and returns the number of bytes
written.
Example:
```gad
print("a", 1)
= 3
<<< a 1
```
**/
export print(*args) <int> => nil

/**
Writes format applied to args to standard output and returns the byte count.
Example:
```gad
printf("%d!", 5)
= 2
<<< 5!
```
**/
export printf(format str, *args) <int> => nil

/**
Writes its arguments and a trailing newline to standard output.
Example:
```gad
println("hi", 1)
= 5
<<< hi 1
```
**/
export println(*args) <int> => nil

/**
Returns format applied to args as a str.
Example:
```gad
sprintf("%s=%d", "n", 42)
= "n=42"
```
**/
export sprintf(format str, *args) <str> => nil

/**
Reports whether every value is of `type`. `type` may be a single type or an
array of types, in which case a value matches when it is any of them.
Example:
```gad
[is(int, 1, 2), is(int, 1, "a"), is([int, str], 1, "a")]
= [true, false, true]
```
**/
export is(type any, *values) <bool> => nil

/**
Reports whether the callable `fn` provides every function header required by
the method interface `mi` (a `meti { … }` value).
Example:
```gad
implements(func(x int) => x, meti { (x int) })
= true
```
**/
export implements(fn callable, mi) <bool> => nil

/**
Returns a new function that calls `caller` with `args`/`named` prepended —
a partial application. Calling the wrapper appends its own arguments.
Example:
```gad
add := func(a, b) => a + b
inc := wrap(add, 1)
inc(2)
= 3
```
**/
export wrap(caller callable, *args; **named) <function> => nil

/**
Casts `obj` (an object that supports casting — a class instance or a reflected
Go value) to the object type `toType`, throwing when incompatible. For the
general checked cast that also accepts interfaces and unions, use the `::`
operator.
Example:
```gad
// a host value or a class instance; anything else throws
r := func() { try { return cast(str, 1) } catch { return "not castable" } }
r()
= "not castable"
```
**/
export cast(toType type, obj any) <any> => nil

/**
Returns the host-attached user data of a value that carries it (a Go value
implementing UserDataStorage); throws otherwise.
Example:
```gad
// a host value carrying user data; anything else throws
r := func() { try { return userData(1) } catch { return "no user data" } }
r()
= "no user data"
```
**/
export userData(o any) <any> => nil

/**
Returns a standard stream of the running VM selected by `which` — "IN"/"OUT"/
"ERR" or 0/1/2. "IN" is a `readable`; "OUT" and "ERR" are `writable`.
Example:
```gad
typeName(stdio("OUT"))
= "writer"
```
**/
export stdio(which any) <any> => nil

/**
Creates a class named `name`; `define` builds its fields, methods and
properties (see the Classes chapter). Also written with the `class` keyword.
It returns a `classType`; calling that class type — `classType(…)` — yields a
`classInstance`.
Example:
```gad
P := Class("P", (cls, define) => define(; fields = (; x = (= 1))))
P().x
= 1
```
**/
export Class(name str, define <(cls classType, define callable) <any>>) <classType> => nil

/**
Attaches typed method overloads to a callable or type, so the VM dispatches on
argument types. Returns the target.
Example:
```gad
kind := func { (x int) => "int" }
addMethod(kind, func(x str) => "str")
kind("a")
= "str"
```
**/
export addMethod(target callable, *methods) <any> => nil

/**
Starts capturing standard output into a fresh buffer, which it returns.
Nested calls stack.
Example:
```gad
obstart()
print("hidden")
str(obend())
= "hidden"
```
**/
export obstart() <buffer> => nil

/**
Stops the most recent output capture and returns its buffer (the captured
output).
Example:
```gad
obstart()
println("a")
str(obend())
= "a\n"
```
**/
export obend() <buffer> => nil

/**
Reads all remaining bytes from a `readable` value.
Example:
```gad
str(read(buffer("xy")))
= "xy"
```
**/
export read(r readable) <bytes> => nil

/**
Writes each data value to a `writable` and returns the number of bytes written.
Example:
```gad
w := buffer()
write(w, "ab", 1)
= 3
str(w)
= "ab1"
```
**/
export write(w writable, *data) <int> => nil

/**
Closes a closable value (e.g. a reader/writer).
Example:
```gad
w := buffer()
close(w)
isError(w)
= false
```
**/
export close(o) => nil

/**
Flushes any buffered output of a `writable`.
Example:
```gad
w := buffer()
write(w, "a")
flush(w)
str(w)
= "a"
```
**/
export flush(w writable) => nil

/**
Builds an iterator over any iterable value (the type constructor form of
`iterate`).
Example:
```gad
collect(iterator([1, 2]))
= [1, 2]
```
**/
export iterator(it iterable) <iterator> => nil

/**
Returns an iterator that chains the given iterables end to end: it yields
every element of the first, then of the second, and so on.
Example:
```gad
collect(zip([1], [2, 3]))
= [1, 2, 3]
```
**/
export zip(*iterables) <iterator> => nil

/**
Builds a single key/value pair value. A dict takes one with `+`:
`d += keyValue(k, v)` sets that entry. `++` is for a sequence of entries.
Example:
```gad
kv := keyValue("a", 1)
[kv.k, kv.v]
= ["a", 1]
```
**/
export keyValue(key any, value any) <keyValue> => nil

/**
Builds an ordered key/value collection from `keyValue` pairs.
Example:
```gad
str(keyValueArray(keyValue("a", 1), keyValue("b", 2)))
= "(;a=1, b=2)"
```
**/
export keyValueArray(*pairs) <keyValueArray> => nil
````
