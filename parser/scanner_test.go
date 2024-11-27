package parser_test

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/gad-lang/gad/parser/source"
	"github.com/stretchr/testify/require"

	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/token"
)

func TestScanner_ScanMixed(t *testing.T) {
	tr := &tester{
		opts: parser.ScannerOptions{
			Mode: parser.Mixed,
		},
	}

	tr.scanExpect(t, "abc\n{%\na\n+\nb\n`}`\n{\n}\n%}",
		parser.ScanComments|parser.DontInsertSemis, []scanResult{
			{Token: token.MixedText, Literal: "abc\n", Line: 1, Column: 1},
			{Token: token.MixedCodeStart, Literal: "{%", Line: 2, Column: 1},
			{Token: token.Ident, Literal: "a", Line: 3, Column: 1},
			{Token: token.Add, Literal: "", Line: 4, Column: 1},
			{Token: token.Ident, Literal: "b", Line: 5, Column: 1},
			{Token: token.RawString, Literal: "`}`", Line: 6, Column: 1},
			{Token: token.LBrace, Literal: "{", Line: 7, Column: 1},
			{Token: token.RBrace, Literal: "}", Line: 8, Column: 1},
			{Token: token.MixedCodeEnd, Literal: "%}", Line: 9, Column: 1},
		}...,
	)
}

func TestScanner_ScanMixed2(t *testing.T) {
	tr := &tester{
		opts: parser.ScannerOptions{
			Mode: parser.Mixed,
		},
	}
	tr.do(t, []testCase{
		{token.MixedText, "abc"},
		{token.MixedCodeStart, "{%"},
		{token.Ident, "a"},
		{token.Add, "+"},
		{token.Ident, "b"},
		{token.RawString, "`}`"},
		{token.LBrace, "{"},
		{token.RBrace, "}"},
		{token.MixedCodeEnd, "%}"},
	})

	tr.opts.MixedDelimiter = parser.MixedDelimiter{
		Start: []rune("{{"),
		End:   []rune("}}"),
	}
	tr.do(t, []testCase{
		{token.MixedText, "abc"},
		{token.MixedCodeStart, "{{"},
		{token.Ident, "a"},
		{token.Add, "+"},
		{token.Ident, "b"},
		{token.RawString, "`}`"},
		{token.LBrace, "{"},
		{token.RBrace, "}"},
		{token.MixedCodeEnd, "}}"},
	})

	tr.opts.MixedDelimiter = parser.MixedDelimiter{
		Start: []rune("<!--"),
		End:   []rune("-->"),
	}
	tr.do(t, []testCase{
		{token.MixedText, "abc"},
		{token.MixedCodeStart, "<!--"},
		{token.Ident, "a"},
		{token.Add, "+"},
		{token.Ident, "b"},
		{token.MixedCodeEnd, "-->"},
	})

	tr.opts.MixedDelimiter = parser.MixedDelimiter{
		Start: []rune("<"),
		End:   []rune(">"),
	}
	tr.do(t, []testCase{
		{token.MixedText, "abc"},
		{token.MixedCodeStart, "<"},
		{token.Ident, "a"},
		{token.Add, "+"},
		{token.Ident, "b"},
		{token.MixedCodeEnd, ">"},
		{token.MixedText, "x"},
	})

	tr.do(t, []testCase{
		{token.MixedText, "abc"},
		{token.MixedCodeStart, "<"},
		{token.Ident, "a"},
		{token.Add, "+"},
		{token.Ident, "b"},
		{token.MixedCodeEnd, ">"},
		{token.MixedText, "x"},
		{token.MixedCodeStart, "<"},
		{token.Add, "+"},
		{token.MixedCodeEnd, ">"},
	})
	tr.do(t, []testCase{
		{token.MixedText, "abc"},
		{token.MixedCodeStart, "<"},
		{token.Sub, "-"},
		{token.Ident, "b"},
		{token.Sub, "-"},
		{token.MixedCodeEnd, ">"},
		{token.MixedText, "x"},
		{token.MixedCodeStart, "<"},
		{token.Add, "+"},
		{token.MixedCodeEnd, ">"},
	})
}

func TestScanner_Scan(t *testing.T) {
	tr := &tester{addLines: true}
	tr.do(t, []testCase{
		{token.Comment, "/* a comment */"},
		{token.Comment, "// a comment \n"},
		{token.Comment, "/*\n*/"},
		{token.Comment, "/**\n/*/"},
		{token.Comment, "/**\n\n/*/"},
		{token.Comment, "//\n"},
		{token.Ident, "foobar"},
		{token.Ident, "a۰۱۸"},
		{token.Ident, "foo६४"},
		{token.Ident, "bar９８７６"},
		{token.Ident, "ŝ"},
		{token.Ident, "ŝfoo"},
		{token.Ident, "$"},
		{token.Ident, "$$$"},
		{token.Ident, "_"},
		{token.Ident, "__"},
		{token.Ident, "$_"},
		{token.Int, "0"},
		{token.Int, "1"},
		{token.Int, "123456789012345678890"},
		{token.Int, "01234567"},
		{token.Int, "0xcafebabe"},
		{token.Uint, "0u"},
		{token.Uint, "1u"},
		{token.Uint, "123456789012345678890u"},
		{token.Uint, "01234567u"},
		{token.Float, "0."},
		{token.Float, ".0"},
		{token.Float, "3.14159265"},
		{token.Float, "1e0"},
		{token.Float, "1e+100"},
		{token.Float, "1e-100"},
		{token.Float, "2.71828e-1000"},
		{token.Decimal, "123456789012345678890d"},
		{token.Decimal, "01234567d"},
		{token.Decimal, "123456789012345678890d"},
		{token.Decimal, "01234567d"},
		{token.Decimal, "0d"},
		{token.Decimal, "3.14159265d"},
		{token.Decimal, "1e0d"},
		{token.Decimal, "1e+100d"},
		{token.Decimal, "1e-100d"},
		{token.Decimal, "2.71828e-1000d"},
		{token.Char, "'a'"},
		{token.Char, "'\\000'"},
		{token.Char, "'\\xFF'"},
		{token.Char, "'\\uff16'"},
		{token.Char, "'\\U0000ff16'"},
		{token.String, `""`},
		{token.String, `"foobar"`},
		{token.String, `"foo` + "\n\n" + `bar"`},
		{token.RawString, "``"},
		{token.RawString, "`foobar`"},
		{token.RawString, "`" + `foo
	                        bar` +
			"`",
		},
		{token.RawString, "`\n`"},
		{token.RawString, "`foo\nbar`"},
		{token.RawHeredoc, "```\n  a\n  bc\n```"},
		{token.RawHeredoc, "```\nabc\n```"},
		{token.RawHeredoc, "```abc```"},
		{token.RawHeredoc, "```a``bc```"},
		{token.RawHeredoc, "`````a``b```c`````"},
		{token.Add, "+"},
		{token.Sub, "-"},
		{token.Mul, "*"},
		{token.Quo, "/"},
		{token.Rem, "%"},
		{token.And, "&"},
		{token.Or, "|"},
		{token.Xor, "^"},
		{token.Shl, "<<"},
		{token.Shr, ">>"},
		{token.AndNot, "&^"},
		{token.AddAssign, "+="},
		{token.SubAssign, "-="},
		{token.MulAssign, "*="},
		{token.QuoAssign, "/="},
		{token.RemAssign, "%="},
		{token.AndAssign, "&="},
		{token.OrAssign, "|="},
		{token.XorAssign, "^="},
		{token.ShlAssign, "<<="},
		{token.ShrAssign, ">>="},
		{token.AndNotAssign, "&^="},
		{token.LOrAssign, "||="},
		{token.NullichAssign, "??="},
		{token.LAnd, "&&"},
		{token.LOr, "||"},
		{token.NullichCoalesce, "??"},
		{token.Inc, "++"},
		{token.Dec, "--"},
		{token.Equal, "=="},
		{token.Less, "<"},
		{token.Greater, ">"},
		{token.Assign, "="},
		{token.Not, "!"},
		{token.NotEqual, "!="},
		{token.LessEq, "<="},
		{token.GreaterEq, ">="},
		{token.Define, ":="},
		{token.Question, "?"},
		{token.Tilde, "~"},
		{token.DoubleTilde, "~~"},
		{token.Pipe, ".|"},
		{token.LParen, "("},
		{token.LBrack, "["},
		{token.LBrace, "{"},
		{token.Comma, ","},
		{token.Period, "."},
		{token.RParen, ")"},
		{token.RBrack, "]"},
		{token.RBrace, "}"},
		{token.Semicolon, ";"},
		{token.Colon, ":"},
		{token.Break, "break"},
		{token.Continue, "continue"},
		{token.Else, "else"},
		{token.For, "for"},
		{token.Func, "func"},
		{token.If, "if"},
		{token.Return, "return"},
		{token.True, "true"},
		{token.False, "false"},
		{token.Yes, "yes"},
		{token.No, "no"},
		{token.In, "in"},
		{token.Nil, "nil"},
		{token.Import, "import"},
		{token.Param, "param"},
		{token.Global, "global"},
		{token.Var, "var"},
		{token.Const, "const"},
		{token.Try, "try"},
		{token.Catch, "catch"},
		{token.Finally, "finally"},
		{token.Throw, "throw"},
		{token.NullishSelector, "?."},
		{token.Callee, "__callee__"},
		{token.Args, "__args__"},
		{token.NamedArgs, "__named_args__"},
		{token.StdIn, "STDIN"},
		{token.StdOut, "STDOUT"},
		{token.StdErr, "STDERR"},
		{token.RBrace, "end"},
		{token.LBrace, "then"},
		{token.LBrace, "do"},
		{token.DotName, "__name__"},
		{token.DotFile, "__file__"},
		{token.IsModule, "__is_module__"},
	})
}

func TestStripCR(t *testing.T) {
	for _, tc := range []struct {
		input  string
		expect string
	}{
		{"//\n", "//\n"},
		{"//\r\n", "//\n"},
		{"//\r\r\r\n", "//\n"},
		{"//\r*\r/\r\n", "//*/\n"},
		{"/**/", "/**/"},
		{"/*\r/*/", "/*/*/"},
		{"/*\r*/", "/**/"},
		{"/**\r/*/", "/**\r/*/"},
		{"/*\r/\r*\r/*/", "/*/*\r/*/"},
		{"/*\r\r\r\r*/", "/**/"},
	} {
		actual := string(parser.StripCR([]byte(tc.input),
			len(tc.input) >= 2 && tc.input[1] == '*'))
		require.Equal(t, tc.expect, actual)
	}
}

func countLines(s string) int {
	if s == "" {
		return 0
	}
	n := 1
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			n++
		}
	}
	return n
}

var testFileSet = source.NewFileSet()

type scanResult struct {
	Token   token.Token
	Literal string
	Line    int
	Column  int
}

type testCase struct {
	token   token.Token
	literal string
}

type tester struct {
	addLines bool
	lineSep  string
	opts     parser.ScannerOptions
}

func (tr tester) do(t *testing.T, testCases []testCase) {
	t.Helper()
	if tr.addLines {
		tr.lineSep = "\r\n"
		tr.doI(t, testCases)
	}
	tr.doI(t, testCases)
}

func (tr tester) doI(t *testing.T, testCases []testCase) {
	t.Helper()
	if tr.lineSep == "" {
		tr.lineSep = "\n"
	}
	var lines []string
	var lineSum int
	lineNos := make([]int, len(testCases))
	columnNos := make([]int, len(testCases))
	for i, tc := range testCases {
		// add 0-2 lines before each test case
		var emptyLines, emptyColumns int
		if tr.addLines {
			emptyLines = rand.Intn(3)
			for j := 0; j < emptyLines; j++ {
				lines = append(lines, strings.Repeat(" ", rand.Intn(10)))
			}
		}

		if tr.addLines {
			// add test case line with some whitespaces around it
			emptyColumns = rand.Intn(10)
			lines = append(lines, fmt.Sprintf("%s%s%s",
				strings.Repeat(" ", emptyColumns),
				tc.literal,
				strings.Repeat(" ", rand.Intn(10))))
		} else {
			lines = append(lines, tc.literal)
		}

		lineNos[i] = lineSum + emptyLines + 1
		columnNos[i] = emptyColumns + 1

		if tc.token == token.MixedText {
			if i > 0 {
				lineNos[i]--
				columnNos[i] = emptyColumns + 2
			}
		}

		lineSum += emptyLines + countLines(tc.literal)
	}

	// expected results
	var expected []scanResult
	var expectedSkipComments []scanResult
	for i, tc := range testCases {
		// expected literal
		var expectedLiteral string
		switch tc.token {
		case token.Comment:
			// strip CRs in comments
			expectedLiteral = string(parser.StripCR([]byte(tc.literal),
				tc.literal[1] == '*'))

			// -style comment literal doesn't contain newline
			if expectedLiteral[1] == '/' {
				expectedLiteral = expectedLiteral[:len(expectedLiteral)-1]
			}
		case token.Ident, token.MixedCodeStart, token.MixedCodeEnd:
			expectedLiteral = tc.literal
		case token.MixedText:
			expectedLiteral = tc.literal
			if i < len(testCases)-1 {
				// remove last \n
				expectedLiteral += "\n"

			}
			if i > 0 {
				expectedLiteral = "\n" + expectedLiteral
			}
		case token.String, token.RawString:
			expectedLiteral = tc.literal
		case token.LParen:
			expectedLiteral = tc.literal
			if expectedLiteral == "" {
				expectedLiteral = "("
			}
		case token.RParen:
			expectedLiteral = tc.literal
			if expectedLiteral == "" {
				expectedLiteral = ")"
			}
		case token.LBrack:
			expectedLiteral = tc.literal
			if expectedLiteral == "" {
				expectedLiteral = "["
			}
		case token.RBrack:
			expectedLiteral = tc.literal
			if expectedLiteral == "" {
				expectedLiteral = "]"
			}
		case token.LBrace:
			expectedLiteral = tc.literal
			if expectedLiteral == "" {
				expectedLiteral = "{"
			}
		case token.RBrace:
			expectedLiteral = tc.literal
			if expectedLiteral == "" {
				expectedLiteral = "}"
			}
		case token.Semicolon:
			if tc.literal == "\n" {
				expectedLiteral = tc.literal
			} else {
				expectedLiteral = ";"
			}
		default:
			if tc.token.IsLiteral() {
				// strip CRs in raw string
				expectedLiteral = tc.literal
				if expectedLiteral[0] == '`' {
					expectedLiteral = string(parser.StripCR(
						[]byte(expectedLiteral), false))
				}
			} else if tc.token.IsKeyword() {
				expectedLiteral = tc.literal
			}
		}

		res := scanResult{
			Token:   tc.token,
			Literal: expectedLiteral,
			Line:    lineNos[i],
			Column:  columnNos[i],
		}

		expected = append(expected, res)
		if tc.token != token.Comment {
			expectedSkipComments = append(expectedSkipComments, res)
		}
	}

	tr.scanExpect(t, strings.Join(lines, tr.lineSep),
		parser.ScanComments|parser.DontInsertSemis, expected...)
	tr.scanExpect(t, strings.Join(lines, tr.lineSep),
		parser.DontInsertSemis, expectedSkipComments...)
}

func (tr *tester) scanExpect(
	t *testing.T,
	input string,
	mode parser.ScanMode,
	expected ...scanResult,
) {
	t.Helper()
	testFile := testFileSet.AddFileData("test", -1, []byte(input))
	opts := tr.opts
	opts.Mode |= mode

	s := parser.NewScanner(
		testFile,
		&opts)
	s.ErrorHandler(func(_ source.SourceFilePos, msg string) { require.Fail(t, msg) })

	for idx, e := range expected {
		tok := s.Scan()

		filePos := testFile.Position(tok.Pos)

		es := fmt.Sprintf("[%s %d:%d] %s", e.Token, e.Line, e.Column, e.Literal)
		gs := fmt.Sprintf("[%s %d:%d] %s", tok.Token, filePos.Line, filePos.Column, tok.Literal)

		require.Equalf(t, es, gs, "[test %d] input: \n%s",
			idx, input)
	}

	tok := s.Scan()
	require.Equal(t, token.EOF, tok.Token, "more tokens left")
	require.Equal(t, 0, s.ErrorCount())
}
