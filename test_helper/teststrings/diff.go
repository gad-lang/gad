package teststrings

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/stretchr/testify/assert"

	"github.com/davecgh/go-spew/spew"
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
			return s
		}
	}

	t.Helper()
	if expected != actual {
		e, a, diff := Diff(withTab(expected), withTab(actual))
		assert.Fail(t, fmt.Sprintf("Not equal: \n"+
			"expected: %v\n"+
			"actual  : %v%s", e, a, diff), append([]interface{}{msg}, args...)...)
		t.FailNow()
	}
	// t.Fail()
}

// Diff returns a diff of both values as long as both are of the same type and
// are a struct, map, slice, array or string. Otherwise it returns an empty string.
func Diff(expected interface{}, actual interface{}) (e, a, diff string) {
	return DiffWithLinePrefix("", expected, actual)
}

// DiffWithLinePrefix returns a diff of both values as long as both are of the same type and
// are a struct, map, slice, array or string. Otherwise it returns an empty string.
func DiffWithLinePrefix(linePrefix string, expected interface{}, actual interface{}) (e, a, diff string) {
	if expected == nil || actual == nil {
		return
	}

	et, ek := typeAndKind(expected)
	at, _ := typeAndKind(actual)

	if et != at {
		return
	}

	if ek != reflect.Struct && ek != reflect.Map && ek != reflect.Slice && ek != reflect.Array && ek != reflect.String {
		return
	}

	switch et {
	case reflect.TypeOf(""):
		e = reflect.ValueOf(expected).String()
		a = reflect.ValueOf(actual).String()
	case reflect.TypeOf(time.Time{}):
		e = spewConfigStringerEnabled.Sdump(expected)
		a = spewConfigStringerEnabled.Sdump(actual)
	default:
		e = spewConfig.Sdump(expected)
		a = spewConfig.Sdump(actual)
	}

	if len(linePrefix) > 0 {
		var withPrefix = func(s string) string {
			el := difflib.SplitLines(s)

			if len(el) > 1 {
				for i, s := range el {
					el[i] = linePrefix + s
				}
				return strings.Join(el, "")
			} else {
				return strconv.Quote(s)
			}
		}
		e = withPrefix(e)
		a = withPrefix(a)
	}

	el := difflib.SplitLines(e)
	al := difflib.SplitLines(a)

	diff, _ = difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        el,
		B:        al,
		FromFile: "Expected",
		FromDate: "",
		ToFile:   "Actual",
		ToDate:   "",
	})

	for i, s := range el {
		el[i] = fmt.Sprintf("Expected %s%04d | %s", linePrefix, i+1, s)
	}

	for i, s := range al {
		al[i] = fmt.Sprintf("Actual    %s%04d | %s", linePrefix, i+1, s)
	}

	e = "\n" + linePrefix + strings.Join(el, "")
	a = "\n" + linePrefix + strings.Join(al, "")

	if len(diff) == 0 {
		e = ""
		a = ""
		return
	}

	diff = "\n\nDiff:\n" + diff
	return
}

func typeAndKind(v interface{}) (reflect.Type, reflect.Kind) {
	t := reflect.TypeOf(v)
	k := t.Kind()

	if k == reflect.Ptr {
		t = t.Elem()
		k = t.Kind()
	}
	return t, k
}

var spewConfig = spew.ConfigState{
	Indent:                  "\t",
	DisablePointerAddresses: true,
	DisableCapacities:       true,
	SortKeys:                true,
	DisableMethods:          true,
	MaxDepth:                10,
}

var spewConfigStringerEnabled = spew.ConfigState{
	Indent:                  "\t",
	DisablePointerAddresses: true,
	DisableCapacities:       true,
	SortKeys:                true,
	MaxDepth:                10,
}
