// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import (
	"fmt"
	"strings"
)

var (
	// ErrSymbolLimit represents a symbol limit error which is returned by
	// Compiler when number of local symbols exceeds the symbo limit for
	// a function that is 256.
	ErrSymbolLimit = &Error{
		Name:    "SymbolLimitError",
		Message: "number of local symbols exceeds the limit",
	}

	// ErrStackOverflow represents a stack overflow error.
	ErrStackOverflow = &Error{Name: "StackOverflowError"}

	// ErrVMAborted represents a VM aborted error.
	ErrVMAborted = &Error{Name: "VMAbortedError"}

	// ErrWrongNumArguments represents a wrong number of arguments error.
	ErrWrongNumArguments = &Error{Name: "WrongNumberOfArgumentsError"}

	// ErrInvalidOperator represents an error for invalid operator usage.
	ErrInvalidOperator = &Error{Name: "InvalidOperatorError"}

	// ErrIndexOutOfBounds represents an out of bounds index error.
	ErrIndexOutOfBounds = &Error{Name: "IndexOutOfBoundsError"}

	// ErrInvalidIndex represents an invalid index error.
	ErrInvalidIndex = &Error{Name: "InvalidIndexError"}

	// ErrNotIterable is an error where an Object is not iterable.
	ErrNotIterable = &Error{Name: "NotIterableError"}

	// ErrNotIndexable is an error where an Object is not indexable.
	ErrNotIndexable = &Error{Name: "NotIndexableError"}

	// ErrNotIndexAssignable is an error where an Object is not index assignable.
	ErrNotIndexAssignable = &Error{Name: "NotIndexAssignableError"}

	// ErrNotIndexDeletable is an error where an Object is not index deletable.
	ErrNotIndexDeletable = &Error{Name: "NotIndexDeletableError"}

	// ErrNotCallable is an error where Object is not callable.
	ErrNotCallable = &Error{Name: "NotCallableError"}

	// ErrNotImplemented is an error where an Object has not implemented a required method.
	ErrNotImplemented = &Error{Name: "NotImplementedError"}

	// ErrZeroDivision is an error where divisor is zero.
	ErrZeroDivision = &Error{Name: "ZeroDivisionError"}

	// ErrUnexpectedNamedArg is an error where unexpected kwarg.
	ErrUnexpectedNamedArg = &Error{Name: "ErrUnexpectedNamedArg"}

	// ErrUnexpectedArgValue is an error where unexpected argument value.
	ErrUnexpectedArgValue = &Error{Name: "ErrUnexpectedArgValue"}

	// ErrIncompatibleCast is an error where incompatible cast.
	ErrIncompatibleCast = &Error{Name: "ErrIncompatibleCast"}

	// ErrIncompatibleReflectFuncType is an error where incompatible reflect func type.
	ErrIncompatibleReflectFuncType = &Error{Name: "ErrIncompatibleReflectFuncType"}

	// ErrReflectCallPanicsType is an error where call reflect function panics.
	ErrReflectCallPanicsType = &Error{Name: "ErrReflectCallPanicsType"}

	// ErrMethodDuplication is an error where method was duplication.
	ErrMethodDuplication = &Error{Name: "ErrMethodDuplication"}

	// ErrMethodNotAppendable is an error where method append is disabled.
	ErrMethodNotAppendable = &Error{Name: "ErrMethodNotAppendable"}

	// ErrType represents a type error.
	ErrType = &Error{Name: "TypeError"}

	// ErrNotInitializable represents a not initializable type error.
	ErrNotInitializable = &Error{Name: "ErrNotInitializable"}

	// ErrNotWriteable represents a not writeable type error.
	ErrNotWriteable = &Error{Name: "ErrNotWriteable"}
)

// NewOperandTypeError creates a new Error from ErrType.
func NewOperandTypeError(token, leftType, rightType string) *Error {
	return ErrType.NewError(
		fmt.Sprintf("unsupported operand types for '%s': '%s' and '%s'",
			token, leftType, rightType))
}

// NewArgumentTypeError creates a new Error from ErrType.
func NewArgumentTypeError(pos, expectType, foundType string) *Error {
	return ErrType.NewError(
		fmt.Sprintf("invalid type for argument '%s': expected %s, found %s",
			pos, expectType, foundType))
}

// NewNamedArgumentTypeError creates a new Error from ErrType.
func NewNamedArgumentTypeError(name, expectType, foundType string) *Error {
	return ErrType.NewError(
		fmt.Sprintf("invalid type for named argument '%s': expected %s, found %s",
			name, expectType, foundType))
}

// NewIndexTypeError creates a new Error from ErrType.
func NewIndexTypeError(expectType, foundType string) *Error {
	return ErrType.NewError(
		fmt.Sprintf("index type expected %s, found %s", expectType, foundType))
}

// NewIndexValueTypeError creates a new Error from ErrType.
func NewIndexValueTypeError(expectType, foundType string) *Error {
	return ErrType.NewError(
		fmt.Sprintf("index value type expected %s, found %s", expectType, foundType))
}

// NewArgumentTypeErrorT creates a new Error from ErrType.
func NewArgumentTypeErrorT(pos string, foundType ObjectType, expectType ...ObjectType) *Error {
	var et = make([]string, len(expectType))
	for i, t := range expectType {
		et[i] = t.ToString()
	}
	return ErrType.NewError(
		fmt.Sprintf("invalid type for argument '%s': expected %s, found %s",
			pos, strings.Join(et, "|"), foundType))
}

// NewIndexTypeErrorT creates a new Error from ErrType.
func NewIndexTypeErrorT(foundType ObjectType, expectType ...ObjectType) *Error {
	var et = make([]string, len(expectType))
	for i, t := range expectType {
		et[i] = t.ToString()
	}
	return ErrType.NewError(
		fmt.Sprintf("index type expected %s, found %s", strings.Join(et, "|"), foundType))
}

// NewIndexValueTypeErrorT creates a new Error from ErrType.
func NewIndexValueTypeErrorT(foundType ObjectType, expectType ...ObjectType) *Error {
	var et = make([]string, len(expectType))
	for i, t := range expectType {
		et[i] = t.ToString()
	}
	return ErrType.NewError(
		fmt.Sprintf("index value type expected %s, found %s", strings.Join(et, "|"), foundType))
}

func IsError(a, b error) *Error {
	if age, _ := a.(*Error); age != nil {
		if bge, _ := b.(*Error); bge != nil {
			if age.Name == bge.Name {
				return age
			}
		}
	}
	return nil
}
