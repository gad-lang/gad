package json

//go:generate go run ../../cmd/mkcallable -output zfuncs_funcs.go zfuncs.go

// builtin Unmarshal
//
//gad:callable func(b []byte,numberAsDecimal=bool,floatAsDecimal=bool,intAsDecimal=bool) (ret gad.Object)
