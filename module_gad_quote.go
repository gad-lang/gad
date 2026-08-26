package gad

import "github.com/gad-lang/gad/quote"

// gadQuoteFn / gadUnquoteFn are the callable objects exposed as `gad.quote` and
// `gad.unquote`. Each carries a typed `str` and a `rawstr` overload (added with
// AddMethod): the argument type selects the cooked (`"…"`) or raw (“ `…` “)
// string flavour.
var (
	gadQuoteFn   Object
	gadUnquoteFn Object
)

// buildGadQuoteFuncs builds gad.quote / gad.unquote. Call it once at init, after
// the builtin type values (TStr, TRawStr, TInt) are registered.
func buildGadQuoteFuncs() {
	// gad.quote(s str; maxCols=120) -> str : encode s as a cooked "…" literal,
	// switching to a """…""" heredoc when a single line would exceed maxCols.
	quoteStr := NewFunction("quote", gadQuoteStr,
		FunctionWithModule(gadModuleSpec),
		FunctionWithUsage("encode a str as a Gad \"…\" string literal (heredoc past maxCols)"),
		FunctionWithParams(func(p func(name string) *ParamBuilder) {
			p("s").Type(TStr).Usage("value to quote")
		}),
		FunctionWithNamedParams(func(np func(name string) *NamedParamBuilder) {
			np("maxCols").Type(TInt).Usage("max chars per line before going multiline (default 120)")
			np("fence").Type(TInt).Usage("heredoc start/end delimiter count, odd >= 3 (default 3)")
		}),
		FunctionWithReturnVars(func(ret func(name string, typ ...TypeAssigner)) { ret("_", TStr) }),
	)
	// gad.quote(s rawstr; maxCols=120) -> str : encode s as a raw `…` literal
	// (or a ```…``` heredoc when it contains backticks).
	quoteRaw := NewFunction("quote", gadQuoteRaw,
		FunctionWithModule(gadModuleSpec),
		FunctionWithUsage("encode a rawstr as a Gad `…` raw string literal"),
		FunctionWithParams(func(p func(name string) *ParamBuilder) {
			p("s").Type(TRawStr).Usage("value to quote")
		}),
		FunctionWithNamedParams(func(np func(name string) *NamedParamBuilder) {
			np("maxCols").Type(TInt).Usage("max chars per line before going multiline (default 120)")
			np("fence").Type(TInt).Usage("heredoc start/end delimiter count, odd >= 3 (default 3)")
		}),
		FunctionWithReturnVars(func(ret func(name string, typ ...TypeAssigner)) { ret("_", TStr) }),
	)
	gadQuoteFn = AddMethod(quoteStr, quoteRaw)

	// gad.unquote(lit str) -> str : decode any string/heredoc literal to its value.
	unquoteStr := NewFunction("unquote", gadUnquoteStr,
		FunctionWithModule(gadModuleSpec),
		FunctionWithUsage("decode a Gad string literal (\"…\", `…`, heredoc) to its str value"),
		FunctionWithParams(func(p func(name string) *ParamBuilder) {
			p("lit").Type(TStr).Usage("string literal to decode")
		}),
		FunctionWithReturnVars(func(ret func(name string, typ ...TypeAssigner)) { ret("_", TStr) }),
	)
	// gad.unquote(lit rawstr) -> rawstr : decode any string/heredoc literal,
	// returning the value as a rawstr.
	unquoteRaw := NewFunction("unquote", gadUnquoteRaw,
		FunctionWithModule(gadModuleSpec),
		FunctionWithUsage("decode a Gad string literal, returning the value as a rawstr"),
		FunctionWithParams(func(p func(name string) *ParamBuilder) {
			p("lit").Type(TRawStr).Usage("string literal to decode")
		}),
		FunctionWithReturnVars(func(ret func(name string, typ ...TypeAssigner)) { ret("_", TRawStr) }),
	)
	gadUnquoteFn = AddMethod(unquoteStr, unquoteRaw)
}

// quoteOptions reads the `maxCols` and `fence` named arguments, defaulting to
// 120 columns and a fence of 3.
func quoteOptions(c Call, raw bool) quote.Options {
	o := quote.Options{MaxLineWidth: quote.DefaultMaxLineWidth, Raw: raw}
	if v, ok := c.NamedArgs.GetValueOrNil("maxCols").(Int); ok {
		o.MaxLineWidth = int(v)
	}
	if v, ok := c.NamedArgs.GetValueOrNil("fence").(Int); ok {
		o.Fence = int(v)
	}
	return o
}

func gadQuoteStr(c Call) (Object, error) {
	s, _ := c.Args.Get(0).(Str)
	return Str(quote.Quote(string(s), quoteOptions(c, false))), nil
}

func gadQuoteRaw(c Call) (Object, error) {
	s, _ := c.Args.Get(0).(RawStr)
	return Str(quote.Quote(string(s), quoteOptions(c, true))), nil
}

func gadUnquoteStr(c Call) (Object, error) {
	s, _ := c.Args.Get(0).(Str)
	v, err := quote.Unquote(string(s))
	if err != nil {
		return nil, err
	}
	return Str(v), nil
}

func gadUnquoteRaw(c Call) (Object, error) {
	s, _ := c.Args.Get(0).(RawStr)
	v, err := quote.Unquote(string(s))
	if err != nil {
		return nil, err
	}
	return RawStr(v), nil
}
