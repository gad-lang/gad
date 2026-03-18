package teststrings

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/stretchr/testify/assert"
)

func Indent(prefix, s string) string {
	al := strings.Split(s, "\n")
	if len(al) > 1 {
		for i, s := range al {
			al[i] = prefix + s
		}
		s = strings.Join(al, "\n")
	} else {
		s = prefix + s
	}
	return s
}

func Diff(expected, got string) string {
	diff, _ := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(expected),
		B:        difflib.SplitLines(got),
		FromFile: "Expected",
		FromDate: "",
		ToFile:   "Actual",
		ToDate:   "",
		Context:  1,
	})

	return "\n\nDiff:\n" + diff
}

func EqualStringf(t *testing.T, expected, actual, msg string, args ...interface{}) {
	var withTab = func(s string) string {
		const prefix = "\t"
		el := strings.Split(s, "\n")

		if len(el) > 1 {
			for i, s := range el {
				el[i] = prefix + s
			}
			return strings.Join(el, "\n")
		} else {
			return strconv.Quote(s)
		}
	}

	t.Helper()
	if expected != actual {
		diff := Diff(withTab(expected), withTab(actual))
		assert.Fail(t, fmt.Sprintf("Not equal: \n"+
			"expected: %v\n"+
			"actual  : %v%s", expected, actual, diff), append([]interface{}{msg}, args...)...)
		t.FailNow()
	}
	// t.Fail()
}
