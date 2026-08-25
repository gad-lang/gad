
# `strings` module

## Public API

### contains

```gad
contains(s str, substr str) <bool>
```

Reports whether substr is within s.

## Example

```gad
strings.contains("hello", "ell")
>>> true
```

### containsAny

```gad
containsAny(s str, chars str) <bool>
```

Reports whether any char in chars are within s.

## Example

```gad
strings.containsAny("hello", "aeiou")
>>> true
```

### containsChar

```gad
containsChar(s str, c char) <bool>
```

Reports whether the char c is within s.

## Example

```gad
strings.containsChar("hello", "e"[0])
>>> true
```

### count

```gad
count(s str, substr str) <int>
```

Counts the number of non-overlapping instances of substr in s.

## Example

```gad
strings.count("banana", "a")
>>> 3
```

### equalFold

```gad
equalFold(s str, t str) <bool>
```

EqualFold reports whether s and t, interpreted as UTF-8 strings,
are equal under Unicode case-folding, which is a more general form of
case-insensitivity.

## Example

```gad
strings.equalFold("Go", "GO")
>>> true
```

### fields

```gad
fields(s str) <array>
```

Splits the string s around each instance of one or more consecutive white
space characters, returning an array of substrings of s or an empty array
if s contains only white space.

## Example

```gad
strings.fields("  a b   c ")
>>> ["a", "b", "c"]
```

### fieldsFunc

```gad
fieldsFunc(s str, f) <array>
```

Splits the string s at each run of Unicode code points c satisfying f(c),
and returns an array of slices of s. If all code points in s satisfy
f(c) or the string is empty, an empty array is returned.

## Example

```gad
strings.fieldsFunc("a1b2c", func(c) => c >= "0"[0] && c <= "9"[0])
>>> ["a", "b", "c"]
```

### hasPrefix

```gad
hasPrefix(s str, prefix str) <bool>
```

Reports whether the string s begins with prefix.

## Example

```gad
strings.hasPrefix("gopher", "go")
>>> true
```

### hasSuffix

```gad
hasSuffix(s str, suffix str) <bool>
```

Reports whether the string s ends with prefix.

## Example

```gad
strings.hasSuffix("gopher", "her")
>>> true
```

### index

```gad
index(s str, substr str) <int>
```

Returns the index of the first instance of substr in s, or -1 if substr
is not present in s.

## Example

```gad
strings.index("chicken", "ken")
>>> 4
```

### indexAny

```gad
indexAny(s str, chars str) <int>
```

Returns the index of the first instance of any char from chars in s, or
-1 if no char from chars is present in s.

## Example

```gad
strings.indexAny("golang", "ln")
>>> 2
```

### indexByte

```gad
indexByte(s str, c char|int) <int>
```

Returns the index of the first byte value of c in s, or -1 if byte value
of c is not present in s. c's integer value must be between 0 and 255.

## Example

```gad
strings.indexByte("golang", "g"[0])
>>> 0
```

### indexChar

```gad
indexChar(s str, c char) <int>
```

Returns the index of the first instance of the char c, or -1 if char is
not present in s.

## Example

```gad
strings.indexChar("golang", "a"[0])
>>> 3
```

### indexFunc

```gad
indexFunc(s str, f) <int>
```

Returns the index into s of the first Unicode code point satisfying f(c),
or -1 if none do.

## Example

```gad
strings.indexFunc("go123", func(c) => c >= "0"[0] && c <= "9"[0])
>>> 2
```

### join

```gad
join(arr array, sep str) <str>
```

Concatenates the string values of array arr elements to create a
single string. The separator string sep is placed between elements in the
resulting string.

## Example

```gad
strings.join(["a", "b", "c"], "-")
>>> "a-b-c"
```

### joinAnd

```gad
joinAnd(arr array, sep, lastSep str) <str>
```

Concatenates the string values of array arr elements to create a
single string. The separator string sep is placed between elements
and lastSep is placed between non last and last elements in the
resulting string.

## Example

```gad
strings.joinAnd(["a", "b", "c"], ", ", " and ")
>>> "a, b and c"
```

### lastIndex

```gad
lastIndex(s str, substr str) <int>
```

Returns the index of the last instance of substr in s, or -1 if substr
is not present in s.

## Example

```gad
strings.lastIndex("go gopher", "go")
>>> 3
```

### lastIndexAny

```gad
lastIndexAny(s str, chars str) <int>
```

Returns the index of the last instance of any char from chars in s, or
-1 if no char from chars is present in s.

## Example

```gad
strings.lastIndexAny("golang", "ln")
>>> 4
```

### lastIndexByte

```gad
lastIndexByte(s str, c char|int) <int>
```

Returns the index of byte value of the last instance of c in s, or -1
if c is not present in s. c's integer value must be between 0 and 255.

## Example

```gad
strings.lastIndexByte("golang", "g"[0])
>>> 5
```

### lastIndexFunc

```gad
lastIndexFunc(s str, f) <int>
```

Returns the index into s of the last Unicode code point satisfying f(c),
or -1 if none do.

## Example

```gad
strings.lastIndexFunc("1a2b", func(c) => c >= "0"[0] && c <= "9"[0])
>>> 2
```

### dict

```gad
dict(f, s str) <str>
```

Returns a copy of the string s with all its characters modified
according to the mapping function f. If f returns a negative value, the
character is dropped from the string with no replacement.

### padLeft

```gad
padLeft(s str, padLen int, padWith any) <str>
```

Returns a string that is padded on the left with the string `padWith` until
the `padLen` length is reached. If padWith is not given, a white space is
used as default padding.

## Example

```gad
strings.padLeft("7", 3, "0")
>>> "007"
```

### padRight

```gad
padRight(s str, padLen int, padWith any) <str>
```

Returns a string that is padded on the right with the string `padWith` until
the `padLen` length is reached. If padWith is not given, a white space is
used as default padding.

## Example

```gad
strings.padRight("7", 3, "0")
>>> "700"
```

### repeat

```gad
repeat(s str, count int) <str>
```

Returns a new string consisting of count copies of the string s.

- If count is a negative int, it returns empty string.
- If (len(s) * count) overflows, it panics.

## Example

```gad
strings.repeat("ab", 3)
>>> "ababab"
```

### replace

```gad
replace(s str, old str, new str, n int) <str>
```

Returns a copy of the string s with the first n non-overlapping instances
of old replaced by new. If n is not provided or -1, it replaces all
instances.

## Example

```gad
strings.replace("oink oink", "k", "ky", 2)
>>> "oinky oinky"
```

### split

```gad
split(s str, sep str, n int) <array>
```

Splits s into substrings separated by sep and returns an array of
the substrings between those separators.

n determines the number of substrings to return:

- n < 0: all substrings (default)
- n > 0: at most n substrings; the last substring will be the unsplit remainder.
- n == 0: the result is empty array

## Example

```gad
strings.split("a,b,c", ",", -1)
>>> ["a", "b", "c"]
```

### splitAfter

```gad
splitAfter(s str, sep str, n int) <array>
```

Slices s into substrings after each instance of sep and returns an array
of those substrings.

n determines the number of substrings to return:

- n < 0: all substrings (default)
- n > 0: at most n substrings; the last substring will be the unsplit remainder.
- n == 0: the result is empty array

## Example

```gad
strings.splitAfter("a,b,c", ",", -1)
>>> ["a,", "b,", "c"]
```

### title

```gad
title(s str) <str>
```

Deprecated: Returns a copy of the string s with all Unicode letters that
begin words mapped to their Unicode title case.

## Example

```gad
strings.title("hello world")
>>> "Hello World"
```

### toLower

```gad
toLower(s str) <str>
```

Returns s with all Unicode letters mapped to their lower case.

## Example

```gad
strings.toLower("Go")
>>> "go"
```

### toTitle

```gad
toTitle(s str) <str>
```

Returns a copy of the string s with all Unicode letters mapped to their
Unicode title case.

## Example

```gad
strings.toTitle("hello")
>>> "HELLO"
```

### toUpper

```gad
toUpper(s str) <str>
```

Returns s with all Unicode letters mapped to their upper case.

## Example

```gad
strings.toUpper("go")
>>> "GO"
```

### toValidUTF8

```gad
toValidUTF8(s str, replacement str) <str>
```

Returns a copy of the string s with each run of invalid UTF-8 byte
sequences replaced by the replacement string, which may be empty.

## Example

```gad
strings.toValidUTF8("abc", "?")
>>> "abc"
```

### trim

```gad
trim(s str, cutset str) <str>
```

Returns a slice of the string s with all leading and trailing Unicode
code points contained in cutset removed.

## Example

```gad
strings.trim("xxhixx", "x")
>>> "hi"
```

### trimFunc

```gad
trimFunc(s str, f) <str>
```

Returns a slice of the string s with all leading and trailing Unicode
code points satisfying f removed.

## Example

```gad
strings.trimFunc("12hi34", func(c) => c >= "0"[0] && c <= "9"[0])
>>> "hi"
```

### trimLeft

```gad
trimLeft(s str, cutset str) <str>
```

Returns a slice of the string s with all leading Unicode code points
contained in cutset removed.

## Example

```gad
strings.trimLeft("xxhi", "x")
>>> "hi"
```

### trimLeftFunc

```gad
trimLeftFunc(s str, f) <str>
```

Returns a slice of the string s with all leading Unicode code points
c satisfying f(c) removed.

## Example

```gad
strings.trimLeftFunc("12hi", func(c) => c >= "0"[0] && c <= "9"[0])
>>> "hi"
```

### trimPrefix

```gad
trimPrefix(s str, prefix str) <str>
```

Returns s without the provided leading prefix string. If s doesn't start
with prefix, s is returned unchanged.

## Example

```gad
strings.trimPrefix("gopher", "go")
>>> "pher"
```

### trimRight

```gad
trimRight(s str, cutset str) <str>
```

Returns a slice of the string s with all trailing Unicode code points
contained in cutset removed.

## Example

```gad
strings.trimRight("hixx", "x")
>>> "hi"
```

### trimRightFunc

```gad
trimRightFunc(s str, f) <str>
```

Returns a slice of the string s with all trailing Unicode code points
c satisfying f(c) removed.

## Example

```gad
strings.trimRightFunc("hi12", func(c) => c >= "0"[0] && c <= "9"[0])
>>> "hi"
```

### trimSpace

```gad
trimSpace(s str) <str>
```

Returns a slice of the string s, with all leading and trailing white
space removed, as defined by Unicode.

## Example

```gad
strings.trimSpace("  hi  ")
>>> "hi"
```

### trimSuffix

```gad
trimSuffix(s str, suffix str) <str>
```

Returns s without the provided trailing suffix string. If s doesn't end
with suffix, s is returned unchanged.

## Example

```gad
strings.trimSuffix("gopher", "her")
>>> "gop"
```

### trunc

```gad
trunc(s str, maxLen int; emph="...") <str>
```

Truncate s to maxLen concatenated with emph.

## Example

```gad
strings.trunc("hello world", 8)
>>> "hello wo..."
```

### slitWords

```gad
slitWords(s str|rawstr) <array>
```

Split words by spaces using regex `\s+`.
If s is rawstr, returns Array of Rawstr, otherwise, Array of Str.

## Example

```gad
strings.slitWords("foo bar baz")
>>> ["foo", "bar", "baz"]
```

### truncWords

```gad
truncWords(s str|rawstr, max int; emph="...", atlimit=false) <_ str|rawstr>
```

Truncate words in s to maxLen concatenated with emph. If atlimit is Falsy,
limits at word count equals to max, otherwise at length of s equals to max.

## Example

```gad
strings.truncWords("one two three", 10)
>>> "one two..."
```

## Example — `strings.gad`

````gad
/**
Reports whether substr is within s.

## Example

```gad
strings.contains("hello", "ell")
>>> true
```
**/
export contains(s str, substr str) <bool> => nil

/**
Reports whether any char in chars are within s.

## Example

```gad
strings.containsAny("hello", "aeiou")
>>> true
```
**/
export containsAny(s str, chars str) <bool> => nil

/**
Reports whether the char c is within s.

## Example

```gad
strings.containsChar("hello", "e"[0])
>>> true
```
**/
export containsChar(s str, c char) <bool> => nil

/**
Counts the number of non-overlapping instances of substr in s.

## Example

```gad
strings.count("banana", "a")
>>> 3
```
**/
export count(s str, substr str) <int> => nil

/**
EqualFold reports whether s and t, interpreted as UTF-8 strings,
are equal under Unicode case-folding, which is a more general form of
case-insensitivity.

## Example

```gad
strings.equalFold("Go", "GO")
>>> true
```
**/
export equalFold(s str, t str) <bool> => nil

/**
Splits the string s around each instance of one or more consecutive white
space characters, returning an array of substrings of s or an empty array
if s contains only white space.

## Example

```gad
strings.fields("  a b   c ")
>>> ["a", "b", "c"]
```
**/
export fields(s str) <array> => nil

/**
Splits the string s at each run of Unicode code points c satisfying f(c),
and returns an array of slices of s. If all code points in s satisfy
f(c) or the string is empty, an empty array is returned.

## Example

```gad
strings.fieldsFunc("a1b2c", func(c) => c >= "0"[0] && c <= "9"[0])
>>> ["a", "b", "c"]
```
**/
export fieldsFunc(s str, f) <array> => nil

/**
Reports whether the string s begins with prefix.

## Example

```gad
strings.hasPrefix("gopher", "go")
>>> true
```
**/
export hasPrefix(s str, prefix str) <bool> => nil

/**
Reports whether the string s ends with prefix.

## Example

```gad
strings.hasSuffix("gopher", "her")
>>> true
```
**/
export hasSuffix(s str, suffix str) <bool> => nil

/**
Returns the index of the first instance of substr in s, or -1 if substr
is not present in s.

## Example

```gad
strings.index("chicken", "ken")
>>> 4
```
**/
export index(s str, substr str) <int> => nil

/**
Returns the index of the first instance of any char from chars in s, or
-1 if no char from chars is present in s.

## Example

```gad
strings.indexAny("golang", "ln")
>>> 2
```
**/
export indexAny(s str, chars str) <int> => nil

/**
Returns the index of the first byte value of c in s, or -1 if byte value
of c is not present in s. c's integer value must be between 0 and 255.

## Example

```gad
strings.indexByte("golang", "g"[0])
>>> 0
```
**/
export indexByte(s str, c char|int) <int> => nil

/**
Returns the index of the first instance of the char c, or -1 if char is
not present in s.

## Example

```gad
strings.indexChar("golang", "a"[0])
>>> 3
```
**/
export indexChar(s str, c char) <int> => nil

/**
Returns the index into s of the first Unicode code point satisfying f(c),
or -1 if none do.

## Example

```gad
strings.indexFunc("go123", func(c) => c >= "0"[0] && c <= "9"[0])
>>> 2
```
**/
export indexFunc(s str, f) <int> => nil

/**
Concatenates the string values of array arr elements to create a
single string. The separator string sep is placed between elements in the
resulting string.

## Example

```gad
strings.join(["a", "b", "c"], "-")
>>> "a-b-c"
```
**/
export join(arr array, sep str) <str> => nil

/**
Concatenates the string values of array arr elements to create a
single string. The separator string sep is placed between elements
and lastSep is placed between non last and last elements in the
resulting string.

## Example

```gad
strings.joinAnd(["a", "b", "c"], ", ", " and ")
>>> "a, b and c"
```
**/
export joinAnd(arr array, sep, lastSep str) <str> => nil

/**
Returns the index of the last instance of substr in s, or -1 if substr
is not present in s.

## Example

```gad
strings.lastIndex("go gopher", "go")
>>> 3
```
**/
export lastIndex(s str, substr str) <int> => nil

/**
Returns the index of the last instance of any char from chars in s, or
-1 if no char from chars is present in s.

## Example

```gad
strings.lastIndexAny("golang", "ln")
>>> 4
```
**/
export lastIndexAny(s str, chars str) <int> => nil

/**
Returns the index of byte value of the last instance of c in s, or -1
if c is not present in s. c's integer value must be between 0 and 255.

## Example

```gad
strings.lastIndexByte("golang", "g"[0])
>>> 5
```
**/
export lastIndexByte(s str, c char|int) <int> => nil

/**
Returns the index into s of the last Unicode code point satisfying f(c),
or -1 if none do.

## Example

```gad
strings.lastIndexFunc("1a2b", func(c) => c >= "0"[0] && c <= "9"[0])
>>> 2
```
**/
export lastIndexFunc(s str, f) <int> => nil

/**
Returns a copy of the string s with all its characters modified
according to the mapping function f. If f returns a negative value, the
character is dropped from the string with no replacement.
**/
export dict(f, s str) <str> => nil

/**
Returns a string that is padded on the left with the string `padWith` until
the `padLen` length is reached. If padWith is not given, a white space is
used as default padding.

## Example

```gad
strings.padLeft("7", 3, "0")
>>> "007"
```
**/
export padLeft(s str, padLen int, padWith any) <str> => nil

/**
Returns a string that is padded on the right with the string `padWith` until
the `padLen` length is reached. If padWith is not given, a white space is
used as default padding.

## Example

```gad
strings.padRight("7", 3, "0")
>>> "700"
```
**/
export padRight(s str, padLen int, padWith any) <str> => nil

/**
Returns a new string consisting of count copies of the string s.

- If count is a negative int, it returns empty string.
- If (len(s) * count) overflows, it panics.

## Example

```gad
strings.repeat("ab", 3)
>>> "ababab"
```
**/
export repeat(s str, count int) <str> => nil

/**
Returns a copy of the string s with the first n non-overlapping instances
of old replaced by new. If n is not provided or -1, it replaces all
instances.

## Example

```gad
strings.replace("oink oink", "k", "ky", 2)
>>> "oinky oinky"
```
**/
export replace(s str, old str, new str, n int) <str> => nil

/**
Splits s into substrings separated by sep and returns an array of
the substrings between those separators.

n determines the number of substrings to return:

- n < 0: all substrings (default)
- n > 0: at most n substrings; the last substring will be the unsplit remainder.
- n == 0: the result is empty array

## Example

```gad
strings.split("a,b,c", ",", -1)
>>> ["a", "b", "c"]
```
**/
export split(s str, sep str, n int) <array> => nil

/**
Slices s into substrings after each instance of sep and returns an array
of those substrings.

n determines the number of substrings to return:

- n < 0: all substrings (default)
- n > 0: at most n substrings; the last substring will be the unsplit remainder.
- n == 0: the result is empty array

## Example

```gad
strings.splitAfter("a,b,c", ",", -1)
>>> ["a,", "b,", "c"]
```
**/
export splitAfter(s str, sep str, n int) <array> => nil

/**
Deprecated: Returns a copy of the string s with all Unicode letters that
begin words mapped to their Unicode title case.

## Example

```gad
strings.title("hello world")
>>> "Hello World"
```
**/
export title(s str) <str> => nil

/**
Returns s with all Unicode letters mapped to their lower case.

## Example

```gad
strings.toLower("Go")
>>> "go"
```
**/
export toLower(s str) <str> => nil

/**
Returns a copy of the string s with all Unicode letters mapped to their
Unicode title case.

## Example

```gad
strings.toTitle("hello")
>>> "HELLO"
```
**/
export toTitle(s str) <str> => nil

/**
Returns s with all Unicode letters mapped to their upper case.

## Example

```gad
strings.toUpper("go")
>>> "GO"
```
**/
export toUpper(s str) <str> => nil

/**
Returns a copy of the string s with each run of invalid UTF-8 byte
sequences replaced by the replacement string, which may be empty.

## Example

```gad
strings.toValidUTF8("abc", "?")
>>> "abc"
```
**/
export toValidUTF8(s str, replacement str) <str> => nil

/**
Returns a slice of the string s with all leading and trailing Unicode
code points contained in cutset removed.

## Example

```gad
strings.trim("xxhixx", "x")
>>> "hi"
```
**/
export trim(s str, cutset str) <str> => nil

/**
Returns a slice of the string s with all leading and trailing Unicode
code points satisfying f removed.

## Example

```gad
strings.trimFunc("12hi34", func(c) => c >= "0"[0] && c <= "9"[0])
>>> "hi"
```
**/
export trimFunc(s str, f) <str> => nil

/**
Returns a slice of the string s with all leading Unicode code points
contained in cutset removed.

## Example

```gad
strings.trimLeft("xxhi", "x")
>>> "hi"
```
**/
export trimLeft(s str, cutset str) <str> => nil

/**
Returns a slice of the string s with all leading Unicode code points
c satisfying f(c) removed.

## Example

```gad
strings.trimLeftFunc("12hi", func(c) => c >= "0"[0] && c <= "9"[0])
>>> "hi"
```
**/
export trimLeftFunc(s str, f) <str> => nil

/**
Returns s without the provided leading prefix string. If s doesn't start
with prefix, s is returned unchanged.

## Example

```gad
strings.trimPrefix("gopher", "go")
>>> "pher"
```
**/
export trimPrefix(s str, prefix str) <str> => nil

/**
Returns a slice of the string s with all trailing Unicode code points
contained in cutset removed.

## Example

```gad
strings.trimRight("hixx", "x")
>>> "hi"
```
**/
export trimRight(s str, cutset str) <str> => nil

/**
Returns a slice of the string s with all trailing Unicode code points
c satisfying f(c) removed.

## Example

```gad
strings.trimRightFunc("hi12", func(c) => c >= "0"[0] && c <= "9"[0])
>>> "hi"
```
**/
export trimRightFunc(s str, f) <str> => nil

/**
Returns a slice of the string s, with all leading and trailing white
space removed, as defined by Unicode.

## Example

```gad
strings.trimSpace("  hi  ")
>>> "hi"
```
**/
export trimSpace(s str) <str> => nil

/**
Returns s without the provided trailing suffix string. If s doesn't end
with suffix, s is returned unchanged.

## Example

```gad
strings.trimSuffix("gopher", "her")
>>> "gop"
```
**/
export trimSuffix(s str, suffix str) <str> => nil

/**
Truncate s to maxLen concatenated with emph.

## Example

```gad
strings.trunc("hello world", 8)
>>> "hello wo..."
```
**/
export trunc(s str, maxLen int; emph="...") <str> => nil

/**
Split words by spaces using regex `\s+`.
If s is rawstr, returns Array of Rawstr, otherwise, Array of Str.

## Example

```gad
strings.slitWords("foo bar baz")
>>> ["foo", "bar", "baz"]
```
**/
export slitWords(s str|rawstr) <array> => nil

/**
Truncate words in s to maxLen concatenated with emph. If atlimit is Falsy,
limits at word count equals to max, otherwise at length of s equals to max.

## Example

```gad
strings.truncWords("one two three", 10)
>>> "one two..."
```
**/
export truncWords(s str|rawstr, max int; emph="...", atlimit=false) <_ str|rawstr> => nil
````
