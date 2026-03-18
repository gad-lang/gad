package json

//go:generate go run ../../cmd/mkcallable -output zfuncs_funcs.go zfuncs.go

// builtin Unmarshal
//
//gad:callable func(b []byte,numberAsDecimal=bool,floatAsDecimal=bool,intAsDecimal=bool) (ret gad.Object, err error)

// builtin Marshal
//
//gad:callable func(vm *gad.VM, o gad.Object) (ret gad.Object, err error)

// builtin Compact
//
//gad:callable func(p []byte, b bool) (ret gad.Object, err error)

// builtin QuoteE
//
//gad:callable func(o gad.Object) (ret gad.Object, err error)

// builtin Quote, NoQuote, NoEscape
//
//gad:callable func(o gad.Object) (ret gad.Object)

// builtin MarshalIndent
//
//gad:callable func(vm *gad.VM, o gad.Object, s1 string, s2 string) (ret gad.Object, err error)

// builtin IndentCount
//
//gad:callable func(p []byte, s1 string, s2 string) (ret gad.Object, err error)
