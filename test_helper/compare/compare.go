package compare

import (
	"fmt"
	_ "unsafe"

	"github.com/gad-lang/gad/test_helper/teststrings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestingT is an interface wrapper around *testing.T
type TestingT interface {
	Errorf(format string, args ...interface{})
	FailNow()
}

type tHelper interface {
	Helper()
}

// Equalf asserts that two objects are equal.
//
//	assert.Equalf(t, 123, 123, "error message %s", "formatted")
//
// Pointer variable equality is determined based on the equality of the
// referenced values (as opposed to the memory addresses). Function equality
// cannot be determined and will always fail.
func Equalf(t TestingT, expected interface{}, actual interface{}, msg string, args ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if equalf(t, expected, actual, msg, args...) {
		return
	}
	t.FailNow()
}

// Equalf asserts that two objects are equal.
//
//	assert.Equalf(t, 123, 123, "error message %s", "formatted")
//
// Pointer variable equality is determined based on the equality of the
// referenced values (as opposed to the memory addresses). Function equality
// cannot be determined and will always fail.
func equalf(t TestingT, expected interface{}, actual interface{}, msg string, args ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	return Equal(t, expected, actual, append([]interface{}{msg}, args...)...)
}

// Equal asserts that two objects are equal.
//
//	assert.Equal(t, 123, 123)
//
// Pointer variable equality is determined based on the equality of the
// referenced values (as opposed to the memory addresses). Function equality
// cannot be determined and will always fail.
func Equal(t require.TestingT, expected, actual interface{}, msgAndArgs ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if err := validateEqualArgs(expected, actual); err != nil {
		return assert.Fail(t, fmt.Sprintf("Invalid operation: %#v == %#v (%s)",
			expected, actual, err), msgAndArgs...)
	}

	if !assert.ObjectsAreEqual(expected, actual) {
		e, a, diff := teststrings.Diff(expected, actual)
		return assert.Fail(t, fmt.Sprintf("Not equal: \n"+
			"expected: %s\n"+
			"actual  : %s%s", e, a, diff), msgAndArgs...)
	}

	return true

}

// validateEqualArgs checks whether provided arguments can be safely used in the
// Equal/NotEqual functions.
//
//go:linkname validateEqualArgs github.com/stretchr/testify/assert.validateEqualArgs
func validateEqualArgs(expected, actual interface{}) error
