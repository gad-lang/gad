package strings_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/gad-lang/gad"
	. "github.com/gad-lang/gad/stdlib/strings"
)

func TestModuleStrings(t *testing.T) {
	contains := Module["Contains"]
	ret, err := MustCall(contains, Str("abc"), Str("b"))
	require.NoError(t, err)
	require.EqualValues(t, true, ret)
	ret, err = MustCall(contains, Str("abc"), Str("d"))
	require.NoError(t, err)
	require.EqualValues(t, false, ret)
	_, err = MustCall(contains, Str("abc"), Str("d"), Str("x"))
	require.Error(t, err)
	_, err = MustCall(contains, Str("abc"))
	require.Error(t, err)
	_, err = MustCall(contains)
	require.Error(t, err)

	containsAny := Module["ContainsAny"]
	ret, err = MustCall(containsAny, Str("abc"), Str("ax"))
	require.NoError(t, err)
	require.EqualValues(t, true, ret)
	ret, err = MustCall(containsAny, Str("abc"), Str("d"))
	require.NoError(t, err)
	require.EqualValues(t, false, ret)

	containsChar := Module["ContainsChar"]
	ret, err = MustCall(containsChar, Str("abc"), Char('a'))
	require.NoError(t, err)
	require.EqualValues(t, true, ret)
	ret, err = MustCall(containsChar, Str("abc"), Char('d'))
	require.NoError(t, err)
	require.EqualValues(t, false, ret)

	count := Module["Count"]
	ret, err = MustCall(count, Str("cheese"), Str("e"))
	require.NoError(t, err)
	require.EqualValues(t, 3, ret)
	ret, err = MustCall(count, Str("cheese"), Str("d"))
	require.NoError(t, err)
	require.EqualValues(t, 0, ret)

	equalFold := Module["EqualFold"]
	ret, err = MustCall(equalFold, Str("GAD"), Str("gad"))
	require.NoError(t, err)
	require.EqualValues(t, true, ret)
	ret, err = MustCall(equalFold, Str("GAD"), Str("go"))
	require.NoError(t, err)
	require.EqualValues(t, false, ret)

	fields := Module["Fields"]
	ret, err = MustCall(fields, Str("\tfoo bar\nbaz"))
	require.NoError(t, err)
	require.Equal(t, 3, len(ret.(Array)))
	require.EqualValues(t, "foo", ret.(Array)[0].(Str))
	require.EqualValues(t, "bar", ret.(Array)[1].(Str))
	require.EqualValues(t, "baz", ret.(Array)[2].(Str))

	hasPrefix := Module["HasPrefix"]
	ret, err = MustCall(hasPrefix, Str("foobarbaz"), Str("foo"))
	require.NoError(t, err)
	require.EqualValues(t, true, ret)
	ret, err = MustCall(hasPrefix, Str("foobarbaz"), Str("baz"))
	require.NoError(t, err)
	require.EqualValues(t, false, ret)

	hasSuffix := Module["HasSuffix"]
	ret, err = MustCall(hasSuffix, Str("foobarbaz"), Str("baz"))
	require.NoError(t, err)
	require.EqualValues(t, true, ret)
	ret, err = MustCall(hasSuffix, Str("foobarbaz"), Str("foo"))
	require.NoError(t, err)
	require.EqualValues(t, false, ret)

	index := Module["Index"]
	ret, err = MustCall(index, Str("foobarbaz"), Str("bar"))
	require.NoError(t, err)
	require.EqualValues(t, 3, ret)
	ret, err = MustCall(index, Str("foobarbaz"), Str("x"))
	require.NoError(t, err)
	require.EqualValues(t, -1, ret)

	indexAny := Module["IndexAny"]
	ret, err = MustCall(indexAny, Str("foobarbaz"), Str("xz"))
	require.NoError(t, err)
	require.EqualValues(t, 8, ret)
	ret, err = MustCall(indexAny, Str("foobarbaz"), Str("x"))
	require.NoError(t, err)
	require.EqualValues(t, -1, ret)

	indexByte := Module["IndexByte"]
	ret, err = MustCall(indexByte, Str("foobarbaz"), Char('z'))
	require.NoError(t, err)
	require.EqualValues(t, 8, ret)
	ret, err = MustCall(indexByte, Str("foobarbaz"), Int('z'))
	require.NoError(t, err)
	require.EqualValues(t, 8, ret)
	ret, err = MustCall(indexByte, Str("foobarbaz"), Char('x'))
	require.NoError(t, err)
	require.EqualValues(t, -1, ret)

	indexChar := Module["IndexChar"]
	ret, err = MustCall(indexChar, Str("foobarbaz"), Char('z'))
	require.NoError(t, err)
	require.EqualValues(t, 8, ret)
	ret, err = MustCall(indexChar, Str("foobarbaz"), Char('x'))
	require.NoError(t, err)
	require.EqualValues(t, -1, ret)

	join := Module["Join"]
	ret, err = MustCall(join, Array{Str("foo"), Str("bar")}, Str(";"))
	require.NoError(t, err)
	require.EqualValues(t, "foo;bar", ret)

	lastIndex := Module["LastIndex"]
	ret, err = MustCall(lastIndex, Str("zfoobarbaz"), Str("z"))
	require.NoError(t, err)
	require.EqualValues(t, 9, ret)
	ret, err = MustCall(lastIndex, Str("zfoobarbaz"), Str("x"))
	require.NoError(t, err)
	require.EqualValues(t, -1, ret)

	lastIndexAny := Module["LastIndexAny"]
	ret, err = MustCall(lastIndexAny, Str("zfoobarbaz"), Str("xz"))
	require.NoError(t, err)
	require.EqualValues(t, 9, ret)
	ret, err = MustCall(lastIndexAny, Str("foobarbaz"), Str("o"))
	require.NoError(t, err)
	require.EqualValues(t, 2, ret)
	ret, err = MustCall(lastIndexAny, Str("foobarbaz"), Str("p"))
	require.NoError(t, err)
	require.EqualValues(t, -1, ret)

	lastIndexByte := Module["LastIndexByte"]
	ret, err = MustCall(lastIndexByte, Str("zfoobarbaz"), Char('z'))
	require.NoError(t, err)
	require.EqualValues(t, 9, ret)
	ret, err = MustCall(lastIndexByte, Str("zfoobarbaz"), Int('z'))
	require.NoError(t, err)
	require.EqualValues(t, 9, ret)
	ret, err = MustCall(lastIndexByte, Str("zfoobarbaz"), Char('x'))
	require.NoError(t, err)
	require.EqualValues(t, -1, ret)

	padLeft := Module["PadLeft"]
	ret, err = MustCall(padLeft, Str("abc"), Int(3))
	require.NoError(t, err)
	require.EqualValues(t, "abc", ret)
	ret, err = MustCall(padLeft, Str("abc"), Int(4))
	require.NoError(t, err)
	require.EqualValues(t, " abc", ret)
	ret, err = MustCall(padLeft, Str("abc"), Int(5))
	require.NoError(t, err)
	require.EqualValues(t, "  abc", ret)
	ret, err = MustCall(padLeft, Str("abc"), Int(5), Str("="))
	require.NoError(t, err)
	require.EqualValues(t, "==abc", ret)
	ret, err = MustCall(padLeft, Str(""), Int(6), Str("="))
	require.NoError(t, err)
	require.EqualValues(t, "======", ret)

	padRight := Module["PadRight"]
	ret, err = MustCall(padRight, Str("abc"), Int(3))
	require.NoError(t, err)
	require.EqualValues(t, "abc", ret)
	ret, err = MustCall(padRight, Str("abc"), Int(4))
	require.NoError(t, err)
	require.EqualValues(t, "abc ", ret)
	ret, err = MustCall(padRight, Str("abc"), Int(5))
	require.NoError(t, err)
	require.EqualValues(t, "abc  ", ret)
	ret, err = MustCall(padRight, Str("abc"), Int(5), Str("="))
	require.NoError(t, err)
	require.EqualValues(t, "abc==", ret)
	ret, err = MustCall(padRight, Str(""), Int(6), Str("="))
	require.NoError(t, err)
	require.EqualValues(t, "======", ret)

	repeat := Module["Repeat"]
	ret, err = MustCall(repeat, Str("abc"), Int(3))
	require.NoError(t, err)
	require.EqualValues(t, "abcabcabc", ret)
	ret, err = MustCall(repeat, Str("abc"), Int(-1))
	require.NoError(t, err)
	require.EqualValues(t, "", ret)

	replace := Module["Replace"]
	ret, err = MustCall(replace, Str("abcdefbc"), Str("bc"), Str("(bc)"))
	require.NoError(t, err)
	require.EqualValues(t, "a(bc)def(bc)", ret)
	ret, err = MustCall(replace,
		Str("abcdefbc"), Str("bc"), Str("(bc)"), Int(1))
	require.NoError(t, err)
	require.EqualValues(t, "a(bc)defbc", ret)

	split := Module["Split"]
	ret, err = MustCall(split, Str("abc;def;"), Str(";"))
	require.NoError(t, err)
	require.Equal(t, 3, len(ret.(Array)))
	require.EqualValues(t, "abc", ret.(Array)[0])
	require.EqualValues(t, "def", ret.(Array)[1])
	require.EqualValues(t, "", ret.(Array)[2])
	ret, err = MustCall(split, Str("abc;def;"), Str("!"), Int(0))
	require.NoError(t, err)
	require.Equal(t, 0, len(ret.(Array)))
	ret, err = MustCall(split, Str("abc;def;"), Str(";"), Int(1))
	require.NoError(t, err)
	require.Equal(t, 1, len(ret.(Array)))
	require.EqualValues(t, "abc;def;", ret.(Array)[0])
	ret, err = MustCall(split, Str("abc;def;"), Str(";"), Int(2))
	require.NoError(t, err)
	require.Equal(t, 2, len(ret.(Array)))
	require.EqualValues(t, "abc", ret.(Array)[0])
	require.EqualValues(t, "def;", ret.(Array)[1])

	splitAfter := Module["SplitAfter"]
	ret, err = MustCall(splitAfter, Str("abc;def;"), Str(";"))
	require.NoError(t, err)
	require.Equal(t, 3, len(ret.(Array)))
	require.EqualValues(t, "abc;", ret.(Array)[0])
	require.EqualValues(t, "def;", ret.(Array)[1])
	require.EqualValues(t, "", ret.(Array)[2])
	ret, err = MustCall(splitAfter, Str("abc;def;"), Str("!"), Int(0))
	require.NoError(t, err)
	require.Equal(t, 0, len(ret.(Array)))
	ret, err = MustCall(splitAfter, Str("abc;def;"), Str(";"), Int(1))
	require.NoError(t, err)
	require.Equal(t, 1, len(ret.(Array)))
	require.EqualValues(t, "abc;def;", ret.(Array)[0])
	ret, err = MustCall(splitAfter, Str("abc;def;"), Str(";"), Int(2))
	require.NoError(t, err)
	require.Equal(t, 2, len(ret.(Array)))
	require.EqualValues(t, "abc;", ret.(Array)[0])
	require.EqualValues(t, "def;", ret.(Array)[1])

	title := Module["Title"]
	ret, err = MustCall(title, Str("хлеб"))
	require.NoError(t, err)
	require.EqualValues(t, "Хлеб", ret)

	toLower := Module["ToLower"]
	ret, err = MustCall(toLower, Str("ÇİĞÖŞÜ"))
	require.NoError(t, err)
	require.EqualValues(t, "çiğöşü", ret)

	toTitle := Module["ToTitle"]
	ret, err = MustCall(toTitle, Str("хлеб"))
	require.NoError(t, err)
	require.EqualValues(t, "ХЛЕБ", ret)

	toUpper := Module["ToUpper"]
	ret, err = MustCall(toUpper, Str("çığöşü"))
	require.NoError(t, err)
	require.EqualValues(t, "ÇIĞÖŞÜ", ret)

	trim := Module["Trim"]
	ret, err = MustCall(trim, Str("!!??abc?!"), Str("!?"))
	require.NoError(t, err)
	require.EqualValues(t, "abc", ret)

	trimLeft := Module["TrimLeft"]
	ret, err = MustCall(trimLeft, Str("!!??abc?!"), Str("!?"))
	require.NoError(t, err)
	require.EqualValues(t, "abc?!", ret)

	trimPrefix := Module["TrimPrefix"]
	ret, err = MustCall(trimPrefix, Str("abcdef"), Str("abc"))
	require.NoError(t, err)
	require.EqualValues(t, "def", ret)

	trimRight := Module["TrimRight"]
	ret, err = MustCall(trimRight, Str("!!??abc?!"), Str("!?"))
	require.NoError(t, err)
	require.EqualValues(t, "!!??abc", ret)

	trimSpace := Module["TrimSpace"]
	ret, err = MustCall(trimSpace, Str("\n \tabcdef\t \n"))
	require.NoError(t, err)
	require.EqualValues(t, "abcdef", ret)

	trimSuffix := Module["TrimSuffix"]
	ret, err = MustCall(trimSuffix, Str("abcdef"), Str("def"))
	require.NoError(t, err)
	require.EqualValues(t, "abc", ret)

	trunc := Module["Trunc"]
	ret, err = MustCall(trunc, Str("abcdef"), Int(6))
	require.NoError(t, err)

	require.EqualValues(t, "abcdef", ret)
	ret, err = MustCall(trunc, Str("abcdef"), Int(5))
	require.NoError(t, err)
	require.EqualValues(t, "abcde...", ret)
}

func TestScript(t *testing.T) {
	ret := func(s string) string {
		return fmt.Sprintf(`
		strings := import("strings")
		return %s`, s)
	}
	catch := func(s string) string {
		return fmt.Sprintf(`
		strings := import("strings")
		try {
			return %s
		} catch err {
			return str(err)
		}`, s)
	}
	wrongArgs := func(want, got int) Str {
		return Str(ErrWrongNumArguments.NewError(
			fmt.Sprintf("want=%d got=%d", want, got),
		).ToString())
	}
	nwrongArgs := func(want1, want2, got int) Str {
		return Str(ErrWrongNumArguments.NewError(
			fmt.Sprintf("want=%d..%d got=%d", want1, want2, got),
		).ToString())
	}
	typeErr := func(pos, expected, got string) Str {
		return Str(NewArgumentTypeError(pos, expected, got).ToString())
	}
	testCases := []struct {
		s string
		m func(string) string
		e Object
	}{
		{s: `strings.Contains()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.Contains(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.Contains(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.Contains(1, 2)`, e: False},
		{s: `strings.Contains("", 2)`, e: False},
		{s: `strings.Contains("acbdef", "de")`, e: True},
		{s: `strings.Contains("acbdef", "dex")`, e: False},

		{s: `strings.ContainsAny()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.ContainsAny(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.ContainsAny(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.ContainsAny(1, 2)`, e: False},
		{s: `strings.ContainsAny("", 2)`, e: False},
		{s: `strings.ContainsAny("acbdef", "de")`, e: True},
		{s: `strings.ContainsAny("acbdef", "xw")`, e: False},

		{s: `strings.ContainsChar()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.ContainsChar(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.ContainsChar(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.ContainsChar(1, 2)`, e: False},
		{s: `strings.ContainsChar("", 2)`, e: False},
		{s: `strings.ContainsChar("acbdef", 'd')`, e: True},
		{s: `strings.ContainsChar("acbdef", 'x')`, e: False},

		{s: `strings.Count()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.Count(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.Count(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.Count(1, 2)`, e: Int(0)},
		{s: `strings.Count("", 2)`, e: Int(0)},
		{s: `strings.Count("abcddef", "d")`, e: Int(2)},
		{s: `strings.Count("abcddef", "x")`, e: Int(0)},

		{s: `strings.EqualFold()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.EqualFold(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.EqualFold(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.EqualFold(1, 2)`, e: False},
		{s: `strings.EqualFold("", 2)`, e: False},
		{s: `strings.EqualFold("çğöşü", "ÇĞÖŞÜ")`, e: True},
		{s: `strings.EqualFold("x", "y")`, e: False},

		{s: `strings.Fields()`, m: catch, e: wrongArgs(1, 0)},
		{s: `strings.Fields(1)`, e: Array{Str("1")}},
		{s: `strings.Fields(1, 2)`, m: catch, e: wrongArgs(1, 2)},
		{s: `strings.Fields("a\nb c\td ")`,
			e: Array{Str("a"), Str("b"), Str("c"), Str("d")}},
		{s: `strings.Fields("")`, e: Array{}},

		{s: `strings.FieldsFunc()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.FieldsFunc("")`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.FieldsFunc("axbxcx", func(c){ return c == 'x' })`,
			e: Array{Str("a"), Str("b"), Str("c")}},
		{s: `strings.FieldsFunc("axbxcx", func(c){ return false })`,
			e: Array{Str("axbxcx")}},

		{s: `strings.HasPrefix()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.HasPrefix(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.HasPrefix(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.HasPrefix(1, 2)`, e: False},
		{s: `strings.HasPrefix("", 2)`, e: False},
		{s: `strings.HasPrefix("abcdef", "abcde")`, e: True},
		{s: `strings.HasPrefix("abcdef", "x")`, e: False},

		{s: `strings.HasSuffix()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.HasSuffix(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.HasSuffix(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.HasSuffix(1, 2)`, e: False},
		{s: `strings.HasSuffix("", 2)`, e: False},
		{s: `strings.HasSuffix("abcdef", "ef")`, e: True},
		{s: `strings.HasSuffix("abcdef", "abc")`, e: False},

		{s: `strings.Index()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.Index(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.Index(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.Index(1, 2)`, e: Int(-1)},
		{s: `strings.Index("", 2)`, e: Int(-1)},
		{s: `strings.Index("abcdef", "ef")`, e: Int(4)},
		{s: `strings.Index("abcdef", "x")`, e: Int(-1)},

		{s: `strings.IndexAny()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.IndexAny(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.IndexAny(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.IndexAny(1, 2)`, e: Int(-1)},
		{s: `strings.IndexAny("", 2)`, e: Int(-1)},
		{s: `strings.IndexAny("abcdef", "ef")`, e: Int(4)},
		{s: `strings.IndexAny("abcdef", "x")`, e: Int(-1)},
		{s: `strings.IndexAny("abcdef", "xa")`, e: Int(0)},

		{s: `strings.IndexByte()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.IndexByte(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.IndexByte(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.IndexByte(1, 2)`, e: Int(-1)},
		{s: `strings.IndexByte("", "")`, e: Int(-1)},
		{s: `strings.IndexByte("abcdef", 'b')`, e: Int(1)},
		{s: `strings.IndexByte("abcdef", int('c'))`, e: Int(2)},
		{s: `strings.IndexByte("abcdef", 'g')`, e: Int(-1)},
		{s: `strings.IndexByte("abcdef", int('g'))`, e: Int(-1)},

		{s: `strings.IndexChar()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.IndexChar(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.IndexChar(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.IndexChar(1, 2)`, e: Int(-1)},
		{s: `strings.IndexChar("", 1)`, e: Int(-1)},
		{s: `strings.IndexChar("abcdef", 'c')`, e: Int(2)},

		{s: `strings.IndexFunc()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.IndexFunc("")`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.IndexFunc("abcd", func(c){return c == 'c'})`, e: Int(2)},
		{s: `strings.IndexFunc("abcd", func(c){return c == 'e'})`, e: Int(-1)},

		{s: `strings.Join()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.Join(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.Join(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.Join(1, 2)`, m: catch,
			e: typeErr("1st", "array", "int")},
		{s: `strings.Join([], 1)`, e: Str("")},
		{s: `strings.Join(["a", "b", "c"], "\t")`, e: Str("a\tb\tc")},

		{s: `strings.JoinAnd()`, m: catch, e: wrongArgs(3, 0)},
		{s: `strings.JoinAnd(1)`, m: catch, e: wrongArgs(3, 1)},
		{s: `strings.JoinAnd(1, 2)`, m: catch, e: wrongArgs(3, 2)},
		{s: `strings.JoinAnd(1, 2, 3, 4)`, m: catch, e: wrongArgs(3, 4)},
		{s: `strings.JoinAnd(1, 2, 3)`, m: catch,
			e: typeErr("1st", "array", "int")},
		{s: `strings.JoinAnd([], 1, 2)`, e: Str("")},
		{s: `strings.JoinAnd(["a", "b", "c"], "\t", "\n")`, e: Str("a\tb\nc")},
		{s: `strings.JoinAnd(["a", "b", "c"], ",", "|")`, e: Str("a,b|c")},

		{s: `strings.LastIndex()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.LastIndex(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.LastIndex(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.LastIndex(1, 2)`, e: Int(-1)},
		{s: `strings.LastIndex("", 2)`, e: Int(-1)},
		{s: `strings.LastIndex("efabcdef", "ef")`, e: Int(6)},
		{s: `strings.LastIndex("abcdef", "g")`, e: Int(-1)},

		{s: `strings.LastIndexAny()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.LastIndexAny(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.LastIndexAny(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.LastIndexAny(1, 2)`, e: Int(-1)},
		{s: `strings.LastIndexAny("", 2)`, e: Int(-1)},
		{s: `strings.LastIndexAny("efabcdef", "xf")`, e: Int(7)},
		{s: `strings.LastIndexAny("abcdef", "g")`, e: Int(-1)},

		{s: `strings.LastIndexByte()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.LastIndexByte(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.LastIndexByte(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.LastIndexByte(1, 2)`, e: Int(-1)},
		{s: `strings.LastIndexByte("", "")`, e: Int(-1)},
		{s: `strings.LastIndexByte("efabcdef", 'f')`, e: Int(7)},
		{s: `strings.LastIndexByte("efabcdef", int('f'))`, e: Int(7)},
		{s: `strings.LastIndexByte("abcdef", 'g')`, e: Int(-1)},

		{s: `strings.LastIndexFunc()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.LastIndexFunc("")`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.LastIndexFunc("acbcd", func(c){return c=='c'})`, e: Int(3)},
		{s: `strings.LastIndexFunc("abcd", func(c){return c=='e'})`, e: Int(-1)},

		{s: `strings.Dict()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.Dict(func(){})`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.Dict(
			func(c){
				if c == 't' { return 'I' }
				if c == 'e' { return '❤' }
				if c == 'n' { return 'G' }
				if c == 'g' { return 'a' }
				if c == 'o' { return 'd' }
				return c
			},
			"tengo")`, e: Str("I❤Gad")},
		{s: `strings.Dict(func(c){return c}, "test")`,
			m: catch, e: Str("test")},

		{s: `strings.PadLeft()`, m: catch, e: nwrongArgs(2, 3, 0)},
		{s: `strings.PadLeft(1)`, m: catch, e: nwrongArgs(2, 3, 1)},
		{s: `strings.PadLeft(1, 2, 3, 4)`, m: catch, e: nwrongArgs(2, 3, 4)},
		{s: `strings.PadLeft(1, 2, 3)`, e: Str("31")},
		{s: `strings.PadLeft("", "", "")`, m: catch,
			e: typeErr("2nd", "int", "str")},
		{s: `strings.PadLeft("", 0, "")`, e: Str("")},
		{s: `strings.PadLeft("", -1, "")`, e: Str("")},
		{s: `strings.PadLeft("", 1, "")`, e: Str("")},
		{s: `strings.PadLeft("", 1, "x")`, e: Str("x")},
		{s: `strings.PadLeft("abc", 3)`, e: Str("abc")},
		{s: `strings.PadLeft("abc", 4)`, e: Str(" abc")},
		{s: `strings.PadLeft("abc", 5)`, e: Str("  abc")},
		{s: `strings.PadLeft("abc", 6)`, e: Str("   abc")},
		{s: `strings.PadLeft("abc", -1, "x")`, e: Str("abc")},
		{s: `strings.PadLeft("abc", 0, "x")`, e: Str("abc")},
		{s: `strings.PadLeft("abc", 1, "x")`, e: Str("abc")},
		{s: `strings.PadLeft("abc", 2, "x")`, e: Str("abc")},
		{s: `strings.PadLeft("abc", 3, "x")`, e: Str("abc")},
		{s: `strings.PadLeft("abc", 4, "x")`, e: Str("xabc")},
		{s: `strings.PadLeft("abc", 5, "x")`, e: Str("xxabc")},
		{s: `strings.PadLeft("abc", 5, "xy")`, e: Str("xyabc")},
		{s: `strings.PadLeft("abc", 6, "xy")`, e: Str("xyxabc")},
		{s: `strings.PadLeft("abc", 6, "wxyz")`, e: Str("wxyabc")},

		{s: `strings.PadRight()`, m: catch, e: nwrongArgs(2, 3, 0)},
		{s: `strings.PadRight(1)`, m: catch, e: nwrongArgs(2, 3, 1)},
		{s: `strings.PadRight(1, 2, 3, 4)`, m: catch, e: nwrongArgs(2, 3, 4)},
		{s: `strings.PadRight(1, 2, 3)`, e: Str("13")},
		{s: `strings.PadRight("", "", "")`, m: catch,
			e: typeErr("2nd", "int", "str")},
		{s: `strings.PadRight("", 0, "")`, e: Str("")},
		{s: `strings.PadRight("", -1, "")`, e: Str("")},
		{s: `strings.PadRight("", 1, "")`, e: Str("")},
		{s: `strings.PadRight("", 1, "x")`, e: Str("x")},
		{s: `strings.PadRight("abc", 3)`, e: Str("abc")},
		{s: `strings.PadRight("abc", 4)`, e: Str("abc ")},
		{s: `strings.PadRight("abc", 5)`, e: Str("abc  ")},
		{s: `strings.PadRight("abc", 6)`, e: Str("abc   ")},
		{s: `strings.PadRight("abc", -1, "x")`, e: Str("abc")},
		{s: `strings.PadRight("abc", 0, "x")`, e: Str("abc")},
		{s: `strings.PadRight("abc", 1, "x")`, e: Str("abc")},
		{s: `strings.PadRight("abc", 2, "x")`, e: Str("abc")},
		{s: `strings.PadRight("abc", 3, "x")`, e: Str("abc")},
		{s: `strings.PadRight("abc", 4, "x")`, e: Str("abcx")},
		{s: `strings.PadRight("abc", 5, "x")`, e: Str("abcxx")},
		{s: `strings.PadRight("abc", 5, "xy")`, e: Str("abcxy")},
		{s: `strings.PadRight("abc", 6, "xy")`, e: Str("abcxyx")},
		{s: `strings.PadRight("abc", 6, "wxyz")`, e: Str("abcwxy")},

		{s: `strings.Repeat()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.Repeat(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.Repeat(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.Repeat(1, 2)`, e: Str("11")},
		{s: `strings.Repeat("", "")`, m: catch,
			e: typeErr("2nd", "int", "str")},
		{s: `strings.Repeat("a", -1)`, e: Str("")},
		{s: `strings.Repeat("a", 0)`, e: Str("")},
		{s: `strings.Repeat("a", 2)`, e: Str("aa")},

		{s: `strings.Replace()`, m: catch, e: nwrongArgs(3, 4, 0)},
		{s: `strings.Replace(1)`, m: catch, e: nwrongArgs(3, 4, 1)},
		{s: `strings.Replace(1, 2)`, m: catch, e: nwrongArgs(3, 4, 2)},
		{s: `strings.Replace(1, 2, 3, 4, 5)`, m: catch, e: nwrongArgs(3, 4, 5)},
		{s: `strings.Replace(1, 2, 3)`, e: Str("1")},
		{s: `strings.Replace("", 1, 3)`, e: Str("")},
		{s: `strings.Replace("", "", 1)`, e: Str("1")},
		{s: `strings.Replace("", "", "", "")`, m: catch,
			e: typeErr("4th", "int", "str")},
		{s: `strings.Replace("abc", "s", "ş")`, e: Str("abc")},
		{s: `strings.Replace("abbc", "b", "a")`, e: Str("aaac")},
		{s: `strings.Replace("abbc", "b", "a", 0)`, e: Str("abbc")},
		{s: `strings.Replace("abbc", "b", "a", 1)`, e: Str("aabc")},

		{s: `strings.Split()`, m: catch, e: nwrongArgs(2, 3, 0)},
		{s: `strings.Split(1)`, m: catch, e: nwrongArgs(2, 3, 1)},
		{s: `strings.Split(1, 2, 3, 4)`, m: catch, e: nwrongArgs(2, 3, 4)},
		{s: `strings.Split(1, 2)`, e: Array{Str("1")}},
		{s: `strings.Split("", 1, 3)`, e: Array{Str("")}},
		{s: `strings.Split("", "", "")`, m: catch,
			e: typeErr("3rd", "int", "str")},
		{s: `strings.Split("a.b.c", ".", 0)`, e: Array{}},
		{s: `strings.Split("a.b.c", ".", 1)`, e: Array{Str("a.b.c")}},
		{s: `strings.Split("a.b.c", ".", -1)`,
			e: Array{Str("a"), Str("b"), Str("c")}},
		{s: `strings.Split("a.b.c", ".")`,
			e: Array{Str("a"), Str("b"), Str("c")}},
		{s: `strings.Split("a.b.c.", ".")`,
			e: Array{Str("a"), Str("b"), Str("c"), Str("")}},
		{s: `strings.Split("a.b.c.", ".", 5)`,
			e: Array{Str("a"), Str("b"), Str("c"), Str("")}},

		{s: `strings.SplitAfter()`, m: catch, e: nwrongArgs(2, 3, 0)},
		{s: `strings.SplitAfter(1)`, m: catch, e: nwrongArgs(2, 3, 1)},
		{s: `strings.SplitAfter(1, 2, 3, 4)`, m: catch, e: nwrongArgs(2, 3, 4)},
		{s: `strings.SplitAfter(1, 2)`, e: Array{Str("1")}},
		{s: `strings.SplitAfter("", 1, 3)`, e: Array{Str("")}},
		{s: `strings.SplitAfter("", "", "")`, m: catch,
			e: typeErr("3rd", "int", "str")},
		{s: `strings.SplitAfter("a.b.c", ".", 0)`, e: Array{}},
		{s: `strings.SplitAfter("a.b.c", ".", 1)`, e: Array{Str("a.b.c")}},
		{s: `strings.SplitAfter("a.b.c", ".", -1)`,
			e: Array{Str("a."), Str("b."), Str("c")}},
		{s: `strings.SplitAfter("a.b.c", ".")`,
			e: Array{Str("a."), Str("b."), Str("c")}},
		{s: `strings.SplitAfter("a.b.c.", ".")`,
			e: Array{Str("a."), Str("b."), Str("c."), Str("")}},
		{s: `strings.SplitAfter("a.b.c.", ".", 5)`,
			e: Array{Str("a."), Str("b."), Str("c."), Str("")}},

		{s: `strings.Title()`, m: catch, e: wrongArgs(1, 0)},
		{s: `strings.Title(1, 2)`, m: catch, e: wrongArgs(1, 2)},
		{s: `strings.Title(1)`, e: Str("1")},
		{s: `strings.Title("")`, e: Str("")},
		{s: `strings.Title("abc def")`, e: Str("Abc Def")},

		{s: `strings.ToLower()`, m: catch, e: wrongArgs(1, 0)},
		{s: `strings.ToLower(1, 2)`, m: catch, e: wrongArgs(1, 2)},
		{s: `strings.ToLower(1)`, e: Str("1")},
		{s: `strings.ToLower("")`, e: Str("")},
		{s: `strings.ToLower("XYZ")`, e: Str("xyz")},

		{s: `strings.ToTitle()`, m: catch, e: wrongArgs(1, 0)},
		{s: `strings.ToTitle(1, 2)`, m: catch, e: wrongArgs(1, 2)},
		{s: `strings.ToTitle(1)`, e: Str("1")},
		{s: `strings.ToTitle("")`, e: Str("")},
		{s: `strings.ToTitle("çğ öşü")`, e: Str("ÇĞ ÖŞÜ")},

		{s: `strings.ToUpper()`, m: catch, e: wrongArgs(1, 0)},
		{s: `strings.ToUpper(1, 2)`, m: catch, e: wrongArgs(1, 2)},
		{s: `strings.ToUpper(1)`, e: Str("1")},
		{s: `strings.ToUpper("")`, e: Str("")},
		{s: `strings.ToUpper("çğ öşü")`, e: Str("ÇĞ ÖŞÜ")},

		{s: `strings.ToValidUTF8()`, m: catch, e: nwrongArgs(1, 2, 0)},
		{s: `strings.ToValidUTF8(1, 2, 2)`, m: catch, e: nwrongArgs(1, 2, 3)},
		{s: `strings.ToValidUTF8("a")`, e: Str("a")},
		{s: `strings.ToValidUTF8("a☺\xffb☺\xC0\xAFc☺\xff", "日本語")`, e: Str("a☺日本語b☺日本語c☺日本語")},
		{s: `strings.ToValidUTF8("a☺\xffb☺\xC0\xAFc☺\xff", "")`, e: Str("a☺b☺c☺")},

		{s: `strings.Trim()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.Trim(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.Trim(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.Trim(1, 2)`, e: Str("1")},
		{s: `strings.Trim("", 2)`, e: Str("")},
		{s: `strings.Trim("!!xyz!!", "")`, e: Str("!!xyz!!")},
		{s: `strings.Trim("!!xyz!!", "!")`, e: Str("xyz")},
		{s: `strings.Trim("!!xyz!!", "!?")`, e: Str("xyz")},

		{s: `strings.TrimFunc()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.TrimFunc("")`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.TrimFunc("xabcxx",
			func(c){return c=='x'})`, e: Str("abc")},

		{s: `strings.TrimLeft()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.TrimLeft(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.TrimLeft(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.TrimLeft(1, 2)`, e: Str("1")},
		{s: `strings.TrimLeft("", 2)`, e: Str("")},
		{s: `strings.TrimLeft("!!xyz!!", "")`, e: Str("!!xyz!!")},
		{s: `strings.TrimLeft("!!xyz!!", "!")`, e: Str("xyz!!")},
		{s: `strings.TrimLeft("!!?xyz!!", "!?")`, e: Str("xyz!!")},

		{s: `strings.TrimLeftFunc()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.TrimLeftFunc("")`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.TrimLeftFunc("xxabcxx",
			func(c){return c=='x'})`, e: Str("abcxx")},
		{s: `strings.TrimLeftFunc("abcxx",
			func(c){return c=='x'})`, e: Str("abcxx")},

		{s: `strings.TrimPrefix()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.TrimPrefix(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.TrimPrefix(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.TrimPrefix(1, 2)`, e: Str("1")},
		{s: `strings.TrimPrefix("", 2)`, e: Str("")},
		{s: `strings.TrimPrefix("!!xyz!!", "")`, e: Str("!!xyz!!")},
		{s: `strings.TrimPrefix("!!xyz!!", "!")`, e: Str("!xyz!!")},
		{s: `strings.TrimPrefix("!!xyz!!", "!!")`, e: Str("xyz!!")},
		{s: `strings.TrimPrefix("!!xyz!!", "!!x")`, e: Str("yz!!")},

		{s: `strings.TrimRight()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.TrimRight(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.TrimRight(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.TrimRight(1, 2)`, e: Str("1")},
		{s: `strings.TrimRight("", 2)`, e: Str("")},
		{s: `strings.TrimRight("!!xyz!!", "")`, e: Str("!!xyz!!")},
		{s: `strings.TrimRight("!!xyz!!", "!")`, e: Str("!!xyz")},
		{s: `strings.TrimRight("!!xyz?!!", "!?")`, e: Str("!!xyz")},

		{s: `strings.TrimRightFunc()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.TrimRightFunc("")`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.TrimRightFunc("xxabcxx",
			func(c){return c=='x'})`, e: Str("xxabc")},
		{s: `strings.TrimRightFunc("xxabc",
			func(c){return c=='x'})`, e: Str("xxabc")},

		{s: `strings.TrimSpace()`, m: catch, e: wrongArgs(1, 0)},
		{s: `strings.TrimSpace(1, 2)`, m: catch, e: wrongArgs(1, 2)},
		{s: `strings.TrimSpace(1)`, e: Str("1")},
		{s: `strings.TrimSpace(" \txyz\n\r")`, e: Str("xyz")},
		{s: `strings.TrimSpace("xyz")`, e: Str("xyz")},

		{s: `strings.TrimSuffix()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.TrimSuffix(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.TrimSuffix(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.TrimSuffix(1, 2)`, e: Str("1")},
		{s: `strings.TrimSuffix("", 2)`, e: Str("")},
		{s: `strings.TrimSuffix("!!xyz!!", "")`, e: Str("!!xyz!!")},
		{s: `strings.TrimSuffix("!!xyz!!", "!")`, e: Str("!!xyz!")},
		{s: `strings.TrimSuffix("!!xyz!!", "!!")`, e: Str("!!xyz")},
		{s: `strings.TrimSuffix("!!xyz!!", "z!!")`, e: Str("!!xy")},
	}
	for _, tt := range testCases {
		var s string
		if tt.m == nil {
			s = ret(tt.s)
		} else {
			s = catch(tt.s)
		}
		t.Run(tt.s, func(t *testing.T) {
			expectRun(t, s, tt.e)
		})
	}
}

func expectRun(t *testing.T, script string, expected Object) {
	t.Helper()
	mm := NewModuleMap()
	mm.AddBuiltinModule("strings", Module)
	c := CompileOptions{CompilerOptions: DefaultCompilerOptions}
	c.ModuleMap = mm
	bc, err := Compile([]byte(script), c)
	require.NoError(t, err)
	ret, err := NewVM(bc).Run(nil)
	require.NoError(t, err)
	require.Equal(t, expected, ret)
}
