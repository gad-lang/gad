package strings_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/gad-lang/gad"
	. "github.com/gad-lang/gad/stdlib/strings"
)

func TestModuleStrings(t *testing.T) {
	mod := NewModule(NewModuleSpecFromName("test"))
	ModuleInit(mod, Call{})
	module := mod.ToDict()

	contains := module["contains"]
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

	containsAny := module["containsAny"]
	ret, err = MustCall(containsAny, Str("abc"), Str("ax"))
	require.NoError(t, err)
	require.EqualValues(t, true, ret)
	ret, err = MustCall(containsAny, Str("abc"), Str("d"))
	require.NoError(t, err)
	require.EqualValues(t, false, ret)

	containsChar := module["containsChar"]
	ret, err = MustCall(containsChar, Str("abc"), Char('a'))
	require.NoError(t, err)
	require.EqualValues(t, true, ret)
	ret, err = MustCall(containsChar, Str("abc"), Char('d'))
	require.NoError(t, err)
	require.EqualValues(t, false, ret)

	count := module["count"]
	ret, err = MustCall(count, Str("cheese"), Str("e"))
	require.NoError(t, err)
	require.EqualValues(t, 3, ret)
	ret, err = MustCall(count, Str("cheese"), Str("d"))
	require.NoError(t, err)
	require.EqualValues(t, 0, ret)

	equalFold := module["equalFold"]
	ret, err = MustCall(equalFold, Str("GAD"), Str("gad"))
	require.NoError(t, err)
	require.EqualValues(t, true, ret)
	ret, err = MustCall(equalFold, Str("GAD"), Str("go"))
	require.NoError(t, err)
	require.EqualValues(t, false, ret)

	fields := module["fields"]
	ret, err = MustCall(fields, Str("\tfoo bar\nbaz"))
	require.NoError(t, err)
	require.Equal(t, 3, len(ret.(Array)))
	require.EqualValues(t, "foo", ret.(Array)[0].(Str))
	require.EqualValues(t, "bar", ret.(Array)[1].(Str))
	require.EqualValues(t, "baz", ret.(Array)[2].(Str))

	hasPrefix := module["hasPrefix"]
	ret, err = MustCall(hasPrefix, Str("foobarbaz"), Str("foo"))
	require.NoError(t, err)
	require.EqualValues(t, true, ret)
	ret, err = MustCall(hasPrefix, Str("foobarbaz"), Str("baz"))
	require.NoError(t, err)
	require.EqualValues(t, false, ret)

	hasSuffix := module["hasSuffix"]
	ret, err = MustCall(hasSuffix, Str("foobarbaz"), Str("baz"))
	require.NoError(t, err)
	require.EqualValues(t, true, ret)
	ret, err = MustCall(hasSuffix, Str("foobarbaz"), Str("foo"))
	require.NoError(t, err)
	require.EqualValues(t, false, ret)

	index := module["index"]
	ret, err = MustCall(index, Str("foobarbaz"), Str("bar"))
	require.NoError(t, err)
	require.EqualValues(t, 3, ret)
	ret, err = MustCall(index, Str("foobarbaz"), Str("x"))
	require.NoError(t, err)
	require.EqualValues(t, -1, ret)

	indexAny := module["indexAny"]
	ret, err = MustCall(indexAny, Str("foobarbaz"), Str("xz"))
	require.NoError(t, err)
	require.EqualValues(t, 8, ret)
	ret, err = MustCall(indexAny, Str("foobarbaz"), Str("x"))
	require.NoError(t, err)
	require.EqualValues(t, -1, ret)

	indexByte := module["indexByte"]
	ret, err = MustCall(indexByte, Str("foobarbaz"), Char('z'))
	require.NoError(t, err)
	require.EqualValues(t, 8, ret)
	ret, err = MustCall(indexByte, Str("foobarbaz"), Int('z'))
	require.NoError(t, err)
	require.EqualValues(t, 8, ret)
	ret, err = MustCall(indexByte, Str("foobarbaz"), Char('x'))
	require.NoError(t, err)
	require.EqualValues(t, -1, ret)

	indexChar := module["indexChar"]
	ret, err = MustCall(indexChar, Str("foobarbaz"), Char('z'))
	require.NoError(t, err)
	require.EqualValues(t, 8, ret)
	ret, err = MustCall(indexChar, Str("foobarbaz"), Char('x'))
	require.NoError(t, err)
	require.EqualValues(t, -1, ret)

	join := module["join"]
	ret, err = MustCall(join, Array{Str("foo"), Str("bar")}, Str(";"))
	require.NoError(t, err)
	require.EqualValues(t, "foo;bar", ret)

	lastIndex := module["lastIndex"]
	ret, err = MustCall(lastIndex, Str("zfoobarbaz"), Str("z"))
	require.NoError(t, err)
	require.EqualValues(t, 9, ret)
	ret, err = MustCall(lastIndex, Str("zfoobarbaz"), Str("x"))
	require.NoError(t, err)
	require.EqualValues(t, -1, ret)

	lastIndexAny := module["lastIndexAny"]
	ret, err = MustCall(lastIndexAny, Str("zfoobarbaz"), Str("xz"))
	require.NoError(t, err)
	require.EqualValues(t, 9, ret)
	ret, err = MustCall(lastIndexAny, Str("foobarbaz"), Str("o"))
	require.NoError(t, err)
	require.EqualValues(t, 2, ret)
	ret, err = MustCall(lastIndexAny, Str("foobarbaz"), Str("p"))
	require.NoError(t, err)
	require.EqualValues(t, -1, ret)

	lastIndexByte := module["lastIndexByte"]
	ret, err = MustCall(lastIndexByte, Str("zfoobarbaz"), Char('z'))
	require.NoError(t, err)
	require.EqualValues(t, 9, ret)
	ret, err = MustCall(lastIndexByte, Str("zfoobarbaz"), Int('z'))
	require.NoError(t, err)
	require.EqualValues(t, 9, ret)
	ret, err = MustCall(lastIndexByte, Str("zfoobarbaz"), Char('x'))
	require.NoError(t, err)
	require.EqualValues(t, -1, ret)

	padLeft := module["padLeft"]
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

	padRight := module["padRight"]
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

	repeat := module["repeat"]
	ret, err = MustCall(repeat, Str("abc"), Int(3))
	require.NoError(t, err)
	require.EqualValues(t, "abcabcabc", ret)
	ret, err = MustCall(repeat, Str("abc"), Int(-1))
	require.NoError(t, err)
	require.EqualValues(t, "", ret)

	replace := module["replace"]
	ret, err = MustCall(replace, Str("abcdefbc"), Str("bc"), Str("(bc)"))
	require.NoError(t, err)
	require.EqualValues(t, "a(bc)def(bc)", ret)
	ret, err = MustCall(replace,
		Str("abcdefbc"), Str("bc"), Str("(bc)"), Int(1))
	require.NoError(t, err)
	require.EqualValues(t, "a(bc)defbc", ret)

	split := module["split"]
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

	splitAfter := module["splitAfter"]
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

	title := module["title"]
	ret, err = MustCall(title, Str("хлеб"))
	require.NoError(t, err)
	require.EqualValues(t, "Хлеб", ret)

	toLower := module["toLower"]
	ret, err = MustCall(toLower, Str("ÇİĞÖŞÜ"))
	require.NoError(t, err)
	require.EqualValues(t, "çiğöşü", ret)

	toTitle := module["toTitle"]
	ret, err = MustCall(toTitle, Str("хлеб"))
	require.NoError(t, err)
	require.EqualValues(t, "ХЛЕБ", ret)

	toUpper := module["toUpper"]
	ret, err = MustCall(toUpper, Str("çığöşü"))
	require.NoError(t, err)
	require.EqualValues(t, "ÇIĞÖŞÜ", ret)

	trim := module["trim"]
	ret, err = MustCall(trim, Str("!!??abc?!"), Str("!?"))
	require.NoError(t, err)
	require.EqualValues(t, "abc", ret)

	trimLeft := module["trimLeft"]
	ret, err = MustCall(trimLeft, Str("!!??abc?!"), Str("!?"))
	require.NoError(t, err)
	require.EqualValues(t, "abc?!", ret)

	trimPrefix := module["trimPrefix"]
	ret, err = MustCall(trimPrefix, Str("abcdef"), Str("abc"))
	require.NoError(t, err)
	require.EqualValues(t, "def", ret)

	trimRight := module["trimRight"]
	ret, err = MustCall(trimRight, Str("!!??abc?!"), Str("!?"))
	require.NoError(t, err)
	require.EqualValues(t, "!!??abc", ret)

	trimSpace := module["trimSpace"]
	ret, err = MustCall(trimSpace, Str("\n \tabcdef\t \n"))
	require.NoError(t, err)
	require.EqualValues(t, "abcdef", ret)

	trimSuffix := module["trimSuffix"]
	ret, err = MustCall(trimSuffix, Str("abcdef"), Str("def"))
	require.NoError(t, err)
	require.EqualValues(t, "abc", ret)

	trunc := module["trunc"]
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
			return str(err.cause)
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
		{s: `strings.contains()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.contains(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.contains(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.contains(1, 2)`, e: False},
		{s: `strings.contains("", 2)`, e: False},
		{s: `strings.contains("acbdef", "de")`, e: True},
		{s: `strings.contains("acbdef", "dex")`, e: False},

		{s: `strings.containsAny()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.containsAny(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.containsAny(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.containsAny(1, 2)`, e: False},
		{s: `strings.containsAny("", 2)`, e: False},
		{s: `strings.containsAny("acbdef", "de")`, e: True},
		{s: `strings.containsAny("acbdef", "xw")`, e: False},

		{s: `strings.containsChar()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.containsChar(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.containsChar(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.containsChar(1, 2)`, e: False},
		{s: `strings.containsChar("", 2)`, e: False},
		{s: `strings.containsChar("acbdef", 'd')`, e: True},
		{s: `strings.containsChar("acbdef", 'x')`, e: False},

		{s: `strings.count()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.count(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.count(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.count(1, 2)`, e: Int(0)},
		{s: `strings.count("", 2)`, e: Int(0)},
		{s: `strings.count("abcddef", "d")`, e: Int(2)},
		{s: `strings.count("abcddef", "x")`, e: Int(0)},

		{s: `strings.equalFold()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.equalFold(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.equalFold(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.equalFold(1, 2)`, e: False},
		{s: `strings.equalFold("", 2)`, e: False},
		{s: `strings.equalFold("çğöşü", "ÇĞÖŞÜ")`, e: True},
		{s: `strings.equalFold("x", "y")`, e: False},

		{s: `strings.fields()`, m: catch, e: wrongArgs(1, 0)},
		{s: `strings.fields(1)`, e: Array{Str("1")}},
		{s: `strings.fields(1, 2)`, m: catch, e: wrongArgs(1, 2)},
		{s: `strings.fields("a\nb c\td ")`,
			e: Array{Str("a"), Str("b"), Str("c"), Str("d")}},
		{s: `strings.fields("")`, e: Array{}},

		{s: `strings.fieldsFunc()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.fieldsFunc("")`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.fieldsFunc("axbxcx", func(c){ return c == 'x' })`,
			e: Array{Str("a"), Str("b"), Str("c")}},
		{s: `strings.fieldsFunc("axbxcx", func(c){ return false })`,
			e: Array{Str("axbxcx")}},

		{s: `strings.hasPrefix()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.hasPrefix(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.hasPrefix(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.hasPrefix(1, 2)`, e: False},
		{s: `strings.hasPrefix("", 2)`, e: False},
		{s: `strings.hasPrefix("abcdef", "abcde")`, e: True},
		{s: `strings.hasPrefix("abcdef", "x")`, e: False},

		{s: `strings.hasSuffix()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.hasSuffix(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.hasSuffix(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.hasSuffix(1, 2)`, e: False},
		{s: `strings.hasSuffix("", 2)`, e: False},
		{s: `strings.hasSuffix("abcdef", "ef")`, e: True},
		{s: `strings.hasSuffix("abcdef", "abc")`, e: False},

		{s: `strings.index()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.index(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.index(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.index(1, 2)`, e: Int(-1)},
		{s: `strings.index("", 2)`, e: Int(-1)},
		{s: `strings.index("abcdef", "ef")`, e: Int(4)},
		{s: `strings.index("abcdef", "x")`, e: Int(-1)},

		{s: `strings.indexAny()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.indexAny(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.indexAny(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.indexAny(1, 2)`, e: Int(-1)},
		{s: `strings.indexAny("", 2)`, e: Int(-1)},
		{s: `strings.indexAny("abcdef", "ef")`, e: Int(4)},
		{s: `strings.indexAny("abcdef", "x")`, e: Int(-1)},
		{s: `strings.indexAny("abcdef", "xa")`, e: Int(0)},

		{s: `strings.indexByte()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.indexByte(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.indexByte(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.indexByte(1, 2)`, e: Int(-1)},
		{s: `strings.indexByte("", "")`, e: Int(-1)},
		{s: `strings.indexByte("abcdef", 'b')`, e: Int(1)},
		{s: `strings.indexByte("abcdef", int('c'))`, e: Int(2)},
		{s: `strings.indexByte("abcdef", 'g')`, e: Int(-1)},
		{s: `strings.indexByte("abcdef", int('g'))`, e: Int(-1)},

		{s: `strings.indexChar()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.indexChar(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.indexChar(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.indexChar(1, 2)`, e: Int(-1)},
		{s: `strings.indexChar("", 1)`, e: Int(-1)},
		{s: `strings.indexChar("abcdef", 'c')`, e: Int(2)},

		{s: `strings.indexFunc()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.indexFunc("")`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.indexFunc("abcd", func(c){return c == 'c'})`, e: Int(2)},
		{s: `strings.indexFunc("abcd", func(c){return c == 'e'})`, e: Int(-1)},

		{s: `strings.join()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.join(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.join(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.join(1, 2)`, m: catch,
			e: typeErr("1st", "array", "int")},
		{s: `strings.join([], 1)`, e: Str("")},
		{s: `strings.join(["a", "b", "c"], "\t")`, e: Str("a\tb\tc")},

		{s: `strings.joinAnd()`, m: catch, e: wrongArgs(3, 0)},
		{s: `strings.joinAnd(1)`, m: catch, e: wrongArgs(3, 1)},
		{s: `strings.joinAnd(1, 2)`, m: catch, e: wrongArgs(3, 2)},
		{s: `strings.joinAnd(1, 2, 3, 4)`, m: catch, e: wrongArgs(3, 4)},
		{s: `strings.joinAnd(1, 2, 3)`, m: catch,
			e: typeErr("1st", "array", "int")},
		{s: `strings.joinAnd([], 1, 2)`, e: Str("")},
		{s: `strings.joinAnd(["a", "b", "c"], "\t", "\n")`, e: Str("a\tb\nc")},
		{s: `strings.joinAnd(["a", "b", "c"], ",", "|")`, e: Str("a,b|c")},

		{s: `strings.lastIndex()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.lastIndex(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.lastIndex(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.lastIndex(1, 2)`, e: Int(-1)},
		{s: `strings.lastIndex("", 2)`, e: Int(-1)},
		{s: `strings.lastIndex("efabcdef", "ef")`, e: Int(6)},
		{s: `strings.lastIndex("abcdef", "g")`, e: Int(-1)},

		{s: `strings.lastIndexAny()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.lastIndexAny(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.lastIndexAny(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.lastIndexAny(1, 2)`, e: Int(-1)},
		{s: `strings.lastIndexAny("", 2)`, e: Int(-1)},
		{s: `strings.lastIndexAny("efabcdef", "xf")`, e: Int(7)},
		{s: `strings.lastIndexAny("abcdef", "g")`, e: Int(-1)},

		{s: `strings.lastIndexByte()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.lastIndexByte(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.lastIndexByte(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.lastIndexByte(1, 2)`, e: Int(-1)},
		{s: `strings.lastIndexByte("", "")`, e: Int(-1)},
		{s: `strings.lastIndexByte("efabcdef", 'f')`, e: Int(7)},
		{s: `strings.lastIndexByte("efabcdef", int('f'))`, e: Int(7)},
		{s: `strings.lastIndexByte("abcdef", 'g')`, e: Int(-1)},

		{s: `strings.lastIndexFunc()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.lastIndexFunc("")`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.lastIndexFunc("acbcd", func(c){return c=='c'})`, e: Int(3)},
		{s: `strings.lastIndexFunc("abcd", func(c){return c=='e'})`, e: Int(-1)},

		{s: `strings.dict()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.dict(func(){})`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.dict(
			func(c){
				if c == 't' { return 'I' }
				if c == 'e' { return '❤' }
				if c == 'n' { return 'G' }
				if c == 'g' { return 'a' }
				if c == 'o' { return 'd' }
				return c
			},
			"tengo")`, e: Str("I❤Gad")},
		{s: `strings.dict(func(c){return c}, "test")`,
			m: catch, e: Str("test")},

		{s: `strings.padLeft()`, m: catch, e: nwrongArgs(2, 3, 0)},
		{s: `strings.padLeft(1)`, m: catch, e: nwrongArgs(2, 3, 1)},
		{s: `strings.padLeft(1, 2, 3, 4)`, m: catch, e: nwrongArgs(2, 3, 4)},
		{s: `strings.padLeft(1, 2, 3)`, e: Str("31")},
		{s: `strings.padLeft("", "", "")`, m: catch,
			e: typeErr("2nd", "int", "str")},
		{s: `strings.padLeft("", 0, "")`, e: Str("")},
		{s: `strings.padLeft("", -1, "")`, e: Str("")},
		{s: `strings.padLeft("", 1, "")`, e: Str("")},
		{s: `strings.padLeft("", 1, "x")`, e: Str("x")},
		{s: `strings.padLeft("abc", 3)`, e: Str("abc")},
		{s: `strings.padLeft("abc", 4)`, e: Str(" abc")},
		{s: `strings.padLeft("abc", 5)`, e: Str("  abc")},
		{s: `strings.padLeft("abc", 6)`, e: Str("   abc")},
		{s: `strings.padLeft("abc", -1, "x")`, e: Str("abc")},
		{s: `strings.padLeft("abc", 0, "x")`, e: Str("abc")},
		{s: `strings.padLeft("abc", 1, "x")`, e: Str("abc")},
		{s: `strings.padLeft("abc", 2, "x")`, e: Str("abc")},
		{s: `strings.padLeft("abc", 3, "x")`, e: Str("abc")},
		{s: `strings.padLeft("abc", 4, "x")`, e: Str("xabc")},
		{s: `strings.padLeft("abc", 5, "x")`, e: Str("xxabc")},
		{s: `strings.padLeft("abc", 5, "xy")`, e: Str("xyabc")},
		{s: `strings.padLeft("abc", 6, "xy")`, e: Str("xyxabc")},
		{s: `strings.padLeft("abc", 6, "wxyz")`, e: Str("wxyabc")},

		{s: `strings.padRight()`, m: catch, e: nwrongArgs(2, 3, 0)},
		{s: `strings.padRight(1)`, m: catch, e: nwrongArgs(2, 3, 1)},
		{s: `strings.padRight(1, 2, 3, 4)`, m: catch, e: nwrongArgs(2, 3, 4)},
		{s: `strings.padRight(1, 2, 3)`, e: Str("13")},
		{s: `strings.padRight("", "", "")`, m: catch,
			e: typeErr("2nd", "int", "str")},
		{s: `strings.padRight("", 0, "")`, e: Str("")},
		{s: `strings.padRight("", -1, "")`, e: Str("")},
		{s: `strings.padRight("", 1, "")`, e: Str("")},
		{s: `strings.padRight("", 1, "x")`, e: Str("x")},
		{s: `strings.padRight("abc", 3)`, e: Str("abc")},
		{s: `strings.padRight("abc", 4)`, e: Str("abc ")},
		{s: `strings.padRight("abc", 5)`, e: Str("abc  ")},
		{s: `strings.padRight("abc", 6)`, e: Str("abc   ")},
		{s: `strings.padRight("abc", -1, "x")`, e: Str("abc")},
		{s: `strings.padRight("abc", 0, "x")`, e: Str("abc")},
		{s: `strings.padRight("abc", 1, "x")`, e: Str("abc")},
		{s: `strings.padRight("abc", 2, "x")`, e: Str("abc")},
		{s: `strings.padRight("abc", 3, "x")`, e: Str("abc")},
		{s: `strings.padRight("abc", 4, "x")`, e: Str("abcx")},
		{s: `strings.padRight("abc", 5, "x")`, e: Str("abcxx")},
		{s: `strings.padRight("abc", 5, "xy")`, e: Str("abcxy")},
		{s: `strings.padRight("abc", 6, "xy")`, e: Str("abcxyx")},
		{s: `strings.padRight("abc", 6, "wxyz")`, e: Str("abcwxy")},

		{s: `strings.repeat()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.repeat(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.repeat(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.repeat(1, 2)`, e: Str("11")},
		{s: `strings.repeat("", "")`, m: catch,
			e: typeErr("2nd", "int", "str")},
		{s: `strings.repeat("a", -1)`, e: Str("")},
		{s: `strings.repeat("a", 0)`, e: Str("")},
		{s: `strings.repeat("a", 2)`, e: Str("aa")},

		{s: `strings.replace()`, m: catch, e: nwrongArgs(3, 4, 0)},
		{s: `strings.replace(1)`, m: catch, e: nwrongArgs(3, 4, 1)},
		{s: `strings.replace(1, 2)`, m: catch, e: nwrongArgs(3, 4, 2)},
		{s: `strings.replace(1, 2, 3, 4, 5)`, m: catch, e: nwrongArgs(3, 4, 5)},
		{s: `strings.replace(1, 2, 3)`, e: Str("1")},
		{s: `strings.replace("", 1, 3)`, e: Str("")},
		{s: `strings.replace("", "", 1)`, e: Str("1")},
		{s: `strings.replace("", "", "", "")`, m: catch,
			e: typeErr("4th", "int", "str")},
		{s: `strings.replace("abc", "s", "ş")`, e: Str("abc")},
		{s: `strings.replace("abbc", "b", "a")`, e: Str("aaac")},
		{s: `strings.replace("abbc", "b", "a", 0)`, e: Str("abbc")},
		{s: `strings.replace("abbc", "b", "a", 1)`, e: Str("aabc")},

		{s: `strings.split()`, m: catch, e: nwrongArgs(2, 3, 0)},
		{s: `strings.split(1)`, m: catch, e: nwrongArgs(2, 3, 1)},
		{s: `strings.split(1, 2, 3, 4)`, m: catch, e: nwrongArgs(2, 3, 4)},
		{s: `strings.split(1, 2)`, e: Array{Str("1")}},
		{s: `strings.split("", 1, 3)`, e: Array{Str("")}},
		{s: `strings.split("", "", "")`, m: catch,
			e: typeErr("3rd", "int", "str")},
		{s: `strings.split("a.b.c", ".", 0)`, e: Array{}},
		{s: `strings.split("a.b.c", ".", 1)`, e: Array{Str("a.b.c")}},
		{s: `strings.split("a.b.c", ".", -1)`,
			e: Array{Str("a"), Str("b"), Str("c")}},
		{s: `strings.split("a.b.c", ".")`,
			e: Array{Str("a"), Str("b"), Str("c")}},
		{s: `strings.split("a.b.c.", ".")`,
			e: Array{Str("a"), Str("b"), Str("c"), Str("")}},
		{s: `strings.split("a.b.c.", ".", 5)`,
			e: Array{Str("a"), Str("b"), Str("c"), Str("")}},

		{s: `strings.splitAfter()`, m: catch, e: nwrongArgs(2, 3, 0)},
		{s: `strings.splitAfter(1)`, m: catch, e: nwrongArgs(2, 3, 1)},
		{s: `strings.splitAfter(1, 2, 3, 4)`, m: catch, e: nwrongArgs(2, 3, 4)},
		{s: `strings.splitAfter(1, 2)`, e: Array{Str("1")}},
		{s: `strings.splitAfter("", 1, 3)`, e: Array{Str("")}},
		{s: `strings.splitAfter("", "", "")`, m: catch,
			e: typeErr("3rd", "int", "str")},
		{s: `strings.splitAfter("a.b.c", ".", 0)`, e: Array{}},
		{s: `strings.splitAfter("a.b.c", ".", 1)`, e: Array{Str("a.b.c")}},
		{s: `strings.splitAfter("a.b.c", ".", -1)`,
			e: Array{Str("a."), Str("b."), Str("c")}},
		{s: `strings.splitAfter("a.b.c", ".")`,
			e: Array{Str("a."), Str("b."), Str("c")}},
		{s: `strings.splitAfter("a.b.c.", ".")`,
			e: Array{Str("a."), Str("b."), Str("c."), Str("")}},
		{s: `strings.splitAfter("a.b.c.", ".", 5)`,
			e: Array{Str("a."), Str("b."), Str("c."), Str("")}},

		{s: `strings.title()`, m: catch, e: wrongArgs(1, 0)},
		{s: `strings.title(1, 2)`, m: catch, e: wrongArgs(1, 2)},
		{s: `strings.title(1)`, e: Str("1")},
		{s: `strings.title("")`, e: Str("")},
		{s: `strings.title("abc def")`, e: Str("Abc Def")},

		{s: `strings.toLower()`, m: catch, e: wrongArgs(1, 0)},
		{s: `strings.toLower(1, 2)`, m: catch, e: wrongArgs(1, 2)},
		{s: `strings.toLower(1)`, e: Str("1")},
		{s: `strings.toLower("")`, e: Str("")},
		{s: `strings.toLower("XYZ")`, e: Str("xyz")},

		{s: `strings.toTitle()`, m: catch, e: wrongArgs(1, 0)},
		{s: `strings.toTitle(1, 2)`, m: catch, e: wrongArgs(1, 2)},
		{s: `strings.toTitle(1)`, e: Str("1")},
		{s: `strings.toTitle("")`, e: Str("")},
		{s: `strings.toTitle("çğ öşü")`, e: Str("ÇĞ ÖŞÜ")},

		{s: `strings.toUpper()`, m: catch, e: wrongArgs(1, 0)},
		{s: `strings.toUpper(1, 2)`, m: catch, e: wrongArgs(1, 2)},
		{s: `strings.toUpper(1)`, e: Str("1")},
		{s: `strings.toUpper("")`, e: Str("")},
		{s: `strings.toUpper("çğ öşü")`, e: Str("ÇĞ ÖŞÜ")},

		{s: `strings.toValidUTF8()`, m: catch, e: nwrongArgs(1, 2, 0)},
		{s: `strings.toValidUTF8(1, 2, 2)`, m: catch, e: nwrongArgs(1, 2, 3)},
		{s: `strings.toValidUTF8("a")`, e: Str("a")},
		{s: `strings.toValidUTF8("a☺\xffb☺\xC0\xAFc☺\xff", "日本語")`, e: Str("a☺日本語b☺日本語c☺日本語")},
		{s: `strings.toValidUTF8("a☺\xffb☺\xC0\xAFc☺\xff", "")`, e: Str("a☺b☺c☺")},

		{s: `strings.trim()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.trim(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.trim(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.trim(1, 2)`, e: Str("1")},
		{s: `strings.trim("", 2)`, e: Str("")},
		{s: `strings.trim("!!xyz!!", "")`, e: Str("!!xyz!!")},
		{s: `strings.trim("!!xyz!!", "!")`, e: Str("xyz")},
		{s: `strings.trim("!!xyz!!", "!?")`, e: Str("xyz")},

		{s: `strings.trimFunc()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.trimFunc("")`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.trimFunc("xabcxx",
			func(c){return c=='x'})`, e: Str("abc")},

		{s: `strings.trimLeft()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.trimLeft(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.trimLeft(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.trimLeft(1, 2)`, e: Str("1")},
		{s: `strings.trimLeft("", 2)`, e: Str("")},
		{s: `strings.trimLeft("!!xyz!!", "")`, e: Str("!!xyz!!")},
		{s: `strings.trimLeft("!!xyz!!", "!")`, e: Str("xyz!!")},
		{s: `strings.trimLeft("!!?xyz!!", "!?")`, e: Str("xyz!!")},

		{s: `strings.trimLeftFunc()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.trimLeftFunc("")`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.trimLeftFunc("xxabcxx",
			func(c){return c=='x'})`, e: Str("abcxx")},
		{s: `strings.trimLeftFunc("abcxx",
			func(c){return c=='x'})`, e: Str("abcxx")},

		{s: `strings.trimPrefix()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.trimPrefix(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.trimPrefix(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.trimPrefix(1, 2)`, e: Str("1")},
		{s: `strings.trimPrefix("", 2)`, e: Str("")},
		{s: `strings.trimPrefix("!!xyz!!", "")`, e: Str("!!xyz!!")},
		{s: `strings.trimPrefix("!!xyz!!", "!")`, e: Str("!xyz!!")},
		{s: `strings.trimPrefix("!!xyz!!", "!!")`, e: Str("xyz!!")},
		{s: `strings.trimPrefix("!!xyz!!", "!!x")`, e: Str("yz!!")},

		{s: `strings.trimRight()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.trimRight(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.trimRight(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.trimRight(1, 2)`, e: Str("1")},
		{s: `strings.trimRight("", 2)`, e: Str("")},
		{s: `strings.trimRight("!!xyz!!", "")`, e: Str("!!xyz!!")},
		{s: `strings.trimRight("!!xyz!!", "!")`, e: Str("!!xyz")},
		{s: `strings.trimRight("!!xyz?!!", "!?")`, e: Str("!!xyz")},

		{s: `strings.trimRightFunc()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.trimRightFunc("")`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.trimRightFunc("xxabcxx",
			func(c){return c=='x'})`, e: Str("xxabc")},
		{s: `strings.trimRightFunc("xxabc",
			func(c){return c=='x'})`, e: Str("xxabc")},

		{s: `strings.trimSpace()`, m: catch, e: wrongArgs(1, 0)},
		{s: `strings.trimSpace(1, 2)`, m: catch, e: wrongArgs(1, 2)},
		{s: `strings.trimSpace(1)`, e: Str("1")},
		{s: `strings.trimSpace(" \txyz\n\r")`, e: Str("xyz")},
		{s: `strings.trimSpace("xyz")`, e: Str("xyz")},

		{s: `strings.trimSuffix()`, m: catch, e: wrongArgs(2, 0)},
		{s: `strings.trimSuffix(1)`, m: catch, e: wrongArgs(2, 1)},
		{s: `strings.trimSuffix(1, 2, 3)`, m: catch, e: wrongArgs(2, 3)},
		{s: `strings.trimSuffix(1, 2)`, e: Str("1")},
		{s: `strings.trimSuffix("", 2)`, e: Str("")},
		{s: `strings.trimSuffix("!!xyz!!", "")`, e: Str("!!xyz!!")},
		{s: `strings.trimSuffix("!!xyz!!", "!")`, e: Str("!!xyz!")},
		{s: `strings.trimSuffix("!!xyz!!", "!!")`, e: Str("!!xyz")},
		{s: `strings.trimSuffix("!!xyz!!", "z!!")`, e: Str("!!xy")},
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
	ret, err := run(script)
	require.NoError(t, err)
	require.Equal(t, expected, ret)
}

func run(script string) (ret Object, err error) {
	mm := NewModuleMap()
	mm.AddBuiltinModuleInit("strings", ModuleInit)
	c := CompileOptions{CompilerOptions: DefaultCompilerOptions}
	c.ModuleMap = mm

	builtins := NewBuiltins().Build()
	_, bc, err := Compile(NewSymbolTable(builtins.Builtins().NameSet), []byte(script), c)
	if err != nil {
		return
	}
	return NewVM(builtins, bc).Run()
}
