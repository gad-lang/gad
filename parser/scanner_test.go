package parser_test

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/token"
)

var testFileSet = parser.NewFileSet()

type scanResult struct {
	Token   token.Token
	Literal string
	Line    int
	Column  int
}

func TestScanner_ScanMixed(t *testing.T) {
	testScan(t, parser.Mixed, false, []struct {
		token   token.Token
		literal string
	}{
		{token.Text, "abc"},
		{token.CodeBegin, "#{"},
		{token.Ident, "a"},
		{token.Add, "+"},
		{token.Ident, "b"},
		{token.Begin, "begin"},
		{token.End, "end"},
		{token.CodeEnd, "}"},
	})
}

func TestScanner_Scan(t *testing.T) {
	testScan(t, 0, true, []struct {
		token   token.Token
		literal string
	}{
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
		{token.String, "`foobar`"},
		{token.String, "`" + `foo
	                        bar` +
			"`",
		},
		{token.String, "`\n`"},
		{token.String, "`foo\nbar`"},
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
		{token.Ellipsis, "..."},
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
		{token.Begin, "begin"},
		{token.End, "end"},
	})
}

func testScan(t *testing.T, mode parser.ScanMode, addLines bool, testCases []struct {
	token   token.Token
	literal string
}) {
	// combine
	var lines []string
	var lineSum int
	lineNos := make([]int, len(testCases))
	columnNos := make([]int, len(testCases))
	for i, tc := range testCases {
		// add 0-2 lines before each test case
		var emptyLines, emptyColumns int
		if addLines {
			emptyLines = rand.Intn(3)
			for j := 0; j < emptyLines; j++ {
				lines = append(lines, strings.Repeat(" ", rand.Intn(10)))
			}
		}

		if addLines {
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
		lineSum += emptyLines + countLines(tc.literal)
		columnNos[i] = emptyColumns + 1
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
		case token.Ident, token.CodeBegin, token.CodeEnd:
			expectedLiteral = tc.literal
		case token.Text:
			expectedLiteral = tc.literal
			if i < len(testCases)-1 {
				// remove last \n
				expectedLiteral += "\n"
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

	scanExpect(t, strings.Join(lines, "\n"),
		parser.ScanComments|parser.DontInsertSemis|mode, expected...)
	scanExpect(t, strings.Join(lines, "\n"),
		parser.DontInsertSemis|mode, expectedSkipComments...)
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

func scanExpect(
	t *testing.T,
	input string,
	mode parser.ScanMode,
	expected ...scanResult,
) {
	testFile := testFileSet.AddFile("test", -1, len(input))

	s := parser.NewScanner(
		testFile,
		[]byte(input),
		mode)
	s.ErrorHandler(func(_ parser.SourceFilePos, msg string) { require.Fail(t, msg) })

	for idx, e := range expected {
		tok := s.Scan()

		filePos := testFile.Position(tok.Pos)

		require.Equalf(t, e.Token, tok.Token, "[%d] expected: %s, actual: %s",
			idx, e.Token.String(), tok.Token.String())
		require.Equal(t, e.Literal, tok.Literal)
		require.Equal(t, e.Line, filePos.Line)
		require.Equal(t, e.Column, filePos.Column)
	}

	tok := s.Scan()
	require.Equal(t, token.EOF, tok.Token, "more tokens left")
	require.Equal(t, 0, s.ErrorCount())
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
