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

	// ErrNotLengther is an error where an Object is not lengther.
	ErrNotLengther = &Error{Name: "NotLengther"}

	// ErrNotIndexable is an error where an Object is not indexable.
	ErrNotIndexable = &Error{Name: "NotIndexableError"}

	// ErrNotIndexAssignable is an error where an Object is not index assignable.
	ErrNotIndexAssignable = &Error{Name: "NotIndexAssignableError"}

	// ErrNotIndexDeletable is an error where an Object is not index deletable.
	ErrNotIndexDeletable = &Error{Name: "NotIndexDeletableError"}

	// ErrNotCallable is an error where Object is not callable.
	ErrNotCallable = &Error{Name: "NotCallableError"}

	// ErrCall is an error where call Object
	ErrCall = &Error{Name: "ErrCall"}

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

	// ErrNoMethodFound is an error where no method found.
	ErrNoMethodFound = &Error{Name: "ErrNoMethodFound"}

	// ErrConstructorRecursiveCall is an error where call recursive constructor.
	ErrConstructorRecursiveCall = &Error{Name: "ErrConstructorRecursiveCall"}

	// ErrConstructorMethodFound is an error where no constructor method found.
	ErrConstructorMethodFound = &Error{Name: "ErrConstructorMethodFound"}

	// ErrType represents a type error.
	ErrType = &Error{Name: "TypeError"}

	// ErrArgument represents a argument error.
	ErrArgument = &Error{Name: "ArgumentError"}

	// ErrNotInitializable represents a not initializable type error.
	ErrNotInitializable = &Error{Name: "ErrNotInitializable"}

	// ErrNotWriteable represents a not writeable type error.
	ErrNotWriteable = &Error{Name: "ErrNotWriteable"}

	// ErrDefineClass represents an error for define class.
	ErrDefineClass = &Error{Name: "ErrDefineClass"}

	// ErrNewClassInstance represents an error for create new ClassInstance.
	ErrNewClassInstance = &Error{Name: "ErrNewClassInstance"}

	// ErrClassInstanceInitialized represents an error for initialized ClassInstance.
	ErrClassInstanceInitialized = &Error{Name: "ErrClassInstanceInitialized"}

	// ErrClassInstanceProperty represents an error of ClassInstance property.
	ErrClassInstanceProperty = &Error{Name: "ErrClassInstanceProperty"}

	// ErrClassPropertyChange represents an error on change ClassProperty.
	ErrClassPropertyChange = &Error{Name: "ErrClassPropertyChange"}

	// ErrClassPropertyRegister represents an error on change ClassProperty.
	ErrClassPropertyRegister = &Error{Name: "ErrClassPropertyRegister"}

	// ErrClassMethodRegister represents an error on register method.
	ErrClassMethodRegister = &Error{Name: "ErrClassMethodRegister"}
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
func NewIndexValueTypeError(index string, expectType, foundType string) *Error {
	return ErrType.NewError(
		fmt.Sprintf("index %s value type expected %s, found %s", index, expectType, foundType))
}

// NewArgumentTypeErrorT creates a new Error from ErrType.
func NewArgumentTypeErrorT(pos string, foundType ObjectType, expectType ...ObjectType) *Error {
	var et = make([]string, len(expectType))
	for i, t := range expectType {
		et[i] = t.ToString()
	}
	return NewArgumentTypeError(pos, strings.Join(et, "|"), foundType.FullName())
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

// NewStructPropertyInstanceError creates a new Error from ErrClassInstanceProperty.
func NewStructPropertyInstanceError(propertyName, message string) *Error {
	return ErrClassInstanceProperty.NewError(fmt.Sprintf("property '%s': %s", propertyName, message))
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
