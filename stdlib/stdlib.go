package stdlib

//go:generate go run ../cmd/mkcallable -export -output zfuncs.go stdlib.go

// time module IsTime
// json module Marshal, Quote, NoQuote, NoEscape
//
//gad:callable func(o gad.Object) (ret gad.Object)

// time module MountString, WeekdayString
//
//gad:callable func(i1 int) (ret gad.Object)

// time module DurationString, DurationHours, DurationMinutes, DurationSeconds
// DurationMilliseconds, DurationMicroseconds, DurationNanoseconds
//
//gad:callable func(i1 int64) (ret gad.Object)

// time module Sleep
//
//gad:callable func(i1 int64)

// time module ParseDuration, LoadLocation
//
//gad:callable func(s string) (ret gad.Object, err error)

// time module FixedZone
// strings module Repeat
//
//gad:callable func(s string, i1 int) (ret gad.Object)

// time module Time, Now
//
//gad:callable func() (ret gad.Object)

// time module DurationRound, DurationTruncate
//
//gad:callable func(i1 int64, i2 int64) (ret gad.Object)

// json module Unmarshal, RawMessage, Valid
//
//gad:callable func(b []byte) (ret gad.Object)

// json module MarshalIndent
//
//gad:callable func(o gad.Object, s1 string, s2 string) (ret gad.Object)

// json module Compact
//
//gad:callable func(p []byte, b bool) (ret gad.Object)

// json module Indent
//
//gad:callable func(p []byte, s1 string, s2 string) (ret gad.Object)

// strings module Contains, ContainsAny, Count, EqualFold, HasPrefix, HasSuffix
// Index, IndexAny, LastIndex, LastIndexAny, Trim, TrimLeft, TrimPrefix,
// TrimRight, TrimSuffix
//
//gad:callable func(s1 string, s2 string) (ret gad.Object)

// strings module Fields, Title, ToLower, ToTitle, ToUpper, TrimSpace
//
//gad:callable func(s string) (ret gad.Object)

// strings module ContainsChar, IndexByte, IndexChar, LastIndexByte
//
//gad:callable func(s string, r rune) (ret gad.Object)

// strings module Join
//
//gad:callable func(arr gad.Array, s string) (ret gad.Object)

// misc. functions
//
//gad:callable func(o gad.Object, i int64) (ret gad.Object, err error)
