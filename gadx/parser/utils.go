package parser

import (
	"strings"

	gadparser "github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
)

// noBase marks a fragment whose absolute position in the original gadx source
// is unknown, so gad assigns fragment-local positions instead.
const noBase source.Pos = source.NoPos

// parseGad parses a GAD expression string using gad's parser.
// If textMode is true, uses mixed mode with {...} delimiters.
func parseGad(s string, file *source.File, textMode bool) (node.Stmts, error) {
	return parseGadAt(s, noBase, textMode)
}

// parseGadTextAt parses mixed text/expression content where a bare `{ expr }` is
// a CONTROL statement (runs, emits nothing) and only `{= expr }` emits — the
// pug-style tag body / `| text` rule. It is parseGadAt without
// ParseMixedExprAsValue.
func parseGadTextAt(s string, base source.Pos) (node.Stmts, error) {
	return parseGadModeAt(s, base, true, false)
}

// parseGadAt parses a GAD fragment that is a verbatim slice of the original gadx
// source beginning at absolute position base (a source.Pos in the enclosing
// FileSet). Setting the fragment file's base to that offset makes gad assign
// every node a position of base+localOffset, i.e. in the original file's
// coordinate space, so error traces and node positions resolve to the correct
// .gadx line and column. Because a fragment's byte layout matches the original
// (verbatim), a constant text prefix such as "return " only shifts base
// uniformly and is handled by the caller.
//
// When base is noBase (position unknown) the fragment is parsed with an
// automatic base, preserving the previous fragment-local behavior.
func parseGadAt(s string, base source.Pos, textMode bool) (_ node.Stmts, err error) {
	// Default text mode keeps a bare `{ expr }` emitting its value (used by the
	// literal-text blocks @text/@p/@md, which render through goldmark and re-parse
	// interpolations). The pug tag body path uses parseGadTextAt (exprAsValue=false)
	// so a bare `{ expr }` is a no-output control statement there.
	return parseGadModeAt(s, base, textMode, true)
}

func parseGadModeAt(s string, base source.Pos, textMode, exprAsValue bool) (_ node.Stmts, err error) {
	po := &gadparser.ParserOptions{
		Mode: gadparser.ParseConfigDisabled,
	}
	so := &gadparser.ScannerOptions{}

	if textMode {
		po.Mode |= gadparser.ParseMixed
		if exprAsValue {
			po.Mode |= gadparser.ParseMixedExprAsValue
		}
		so.MixedDelimiter.Start = []rune("{")
		so.MixedDelimiter.End = []rune("}")
	}

	fileSet := source.NewFileSet()
	fbase := -1
	// The fragment base must be >= the file set base (1). Imported gadx files
	// live at large offsets, so this holds in practice; guard the standalone
	// case where an early fragment could map below the set base.
	if base != noBase && int(base) >= fileSet.Base {
		fbase = int(base)
	}
	srcFile := fileSet.AddFileData("(main)", fbase, []byte(s))
	p := gadparser.NewParserWithOptions(srcFile, po, so)

	var f *gadparser.File
	if f, err = p.ParseFile(); err != nil {
		return
	}
	return f.Stmts, err
}

func parseGadFirstStmtAt(s string, base source.Pos, mixed bool) (_ node.Stmt, err error) {
	var stmts node.Stmts
	if stmts, err = parseGadAt(s, base, mixed); err != nil {
		return
	}
	return stmts[0], nil
}

// parseTextGadRawAt is like parseTextGadAt but keeps leading and trailing
// whitespace verbatim (no trimming). It is used for `@text`/`@p`/`@md` body lines
// where indentation and surrounding spaces are significant content. base points
// at the first character of s so `{ … }` interpolation keeps its source position.
func parseTextGadRawAt(s string, base source.Pos) (node.Stmts, error) {
	return parseGadAt(s, base, true)
}

// parseTextGadAt parses mixed text/expression content (with {...} interpolation
// delimiters) beginning at absolute position base, so embedded expressions map
// back onto the original source. Leading whitespace trimmed from s shifts base
// accordingly.
func parseTextGadAt(s string, base source.Pos) (node.Stmts, error) {
	trimmed := strings.TrimSpace(s)
	if base != noBase {
		lead := source.Pos(len(s) - len(strings.TrimLeft(s, " \t\r\n")))
		base += lead
	}
	// Pug-style tag body / `| text`: a bare `{ expr }` is control (no output).
	return parseGadTextAt(trimmed, base)
}

func parseCallArgsString(s string) (args *node.CallArgs, err error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return &node.CallArgs{}, nil
	}
	parts := splitTopLevelArgs(s)
	args = &node.CallArgs{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.HasPrefix(part, "**") {
			name := strings.TrimSpace(part[2:])
			args.NamedArgs.Append(&node.NamedArgExpr{Ident: node.EIdent(name, 0), Var: true}, nil)
			continue
		}
		if idx := topLevelAssignIndex(part); idx >= 0 {
			name := strings.TrimSpace(part[:idx])
			value := strings.TrimSpace(part[idx+1:])
			args.NamedArgs.AppendS(name, parseExprStr(value, 0))
			continue
		}
		args.Args.Values = append(args.Args.Values, parseExprStr(part, 0))
	}
	return args, nil
}

func parseFuncParamsString(s string) (params *node.FuncParams, err error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return &node.FuncParams{}, nil
	}
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		s = strings.TrimSpace(s[1 : len(s)-1])
	}
	return parseGadxFuncParamsString(s), nil
}

func parseGadxFuncParamsString(s string) *node.FuncParams {
	s = strings.TrimSpace(s)
	if s == "" {
		return &node.FuncParams{}
	}

	params := &node.FuncParams{}
	parts := splitTopLevelArgs(s)
	seenNamed := false
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.HasPrefix(part, "**") {
			name := strings.TrimSpace(part[2:])
			params.NamedArgs.Var = node.EIdent(name, 0)
			seenNamed = true
			continue
		}
		if strings.HasPrefix(part, "*") {
			name := strings.TrimSpace(part[1:])
			params.Args.Var = node.ETypedIdent(node.EIdent(name, 0))
			continue
		}
		if idx := topLevelAssignIndex(part); idx >= 0 {
			seenNamed = true
			name := strings.TrimSpace(part[:idx])
			value := strings.TrimSpace(part[idx+1:])
			params.NamedArgs.Add(node.ETypedIdent(node.EIdent(name, 0)), parseExprStr(value, 0))
			continue
		}
		if seenNamed {
			params.NamedArgs.Add(node.ETypedIdent(node.EIdent(strings.TrimSpace(part), 0)), nil)
			continue
		}
		params.Args.Values = append(params.Args.Values, node.ETypedIdent(node.EIdent(part, 0)))
	}
	return params
}

func ParseCallArgsStringForTest(s string) (*node.CallArgs, error) {
	return parseCallArgsString(s)
}

func ParseFuncParamsStringForTest(s string) (*node.FuncParams, error) {
	return parseFuncParamsString(s)
}
