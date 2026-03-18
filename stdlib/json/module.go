// Copyright (c) 2022-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package json

import (
	"bytes"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/stdlib"
)

// ModuleInit represents json module.
var ModuleInit gad.ModuleInitFunc = func(module *gad.Module, c gad.Call) (data gad.ModuleData, err error) {
	return gad.Dict{
		// gad:doc
		// # json module
		//
		// ## Functions
		// Marshal(v any) -> bytes
		// Returns the JSON encoding v or error.
		"Marshal": &gad.Function{
			Module:   module,
			FuncName: "Marshal",
			Value:    funcPpVM_OROe(marshalFunc),
		},
		// gad:doc
		// MarshalIndent(v any, prefix string, indent string) -> bytes
		// MarshalIndent is like Marshal but applies IndentCount to format the output.
		"MarshalIndent": &gad.Function{
			Module:   module,
			FuncName: "MarshalIndent",
			Value:    funcPpVM_OssROe(marshalIndentFunc),
		},
		// gad:doc
		// IndentCount(src bytes, prefix string, indent string) -> bytes
		// Returns indented form of the JSON-encoded src or error.
		"IndentCount": &gad.Function{
			Module:   module,
			FuncName: "IndentCount",
			Value:    funcPb2ssROe(indentFunc),
		},
		// gad:doc
		// RawMessage(v bytes) -> rawMessage
		// Returns a wrapped bytes to provide raw encoded JSON value to Marshal
		// functions.
		"RawMessage": &gad.Function{
			Module:   module,
			FuncName: "RawMessage",
			Value:    stdlib.FuncPb2RO(rawMessageFunc),
		},
		// gad:doc
		// Compact(data bytes, escape bool) -> bytes
		// Returns elided insignificant space characters from data or error.
		"Compact": &gad.Function{
			Module:   module,
			FuncName: "Compact",
			Value:    funcPb2bROe(compactFunc),
		},
		// gad:doc
		// Quote(v any) -> encoderOptions
		// Returns a wrapped object to provide Marshal functions to quote v.
		"Quote": &gad.Function{
			Module:   module,
			FuncName: "Quote",
			Value:    funcPORO(quoteFunc),
		},
		// gad:doc
		// NoQuote(v any) -> encoderOptions
		// Returns a wrapped object to provide Marshal functions not to quote while
		// encoding.
		// This can be used not to quote all array or map items.
		"NoQuote": &gad.Function{
			Module:   module,
			FuncName: "NoQuote",
			Value:    funcPORO(noQuoteFunc),
		},
		// gad:doc
		// NoEscape(v any) -> encoderOptions
		// Returns a wrapped object to provide Marshal functions not to escape html
		// while encoding.
		"NoEscape": &gad.Function{
			Module:   module,
			FuncName: "NoEscape",
			Value:    funcPORO(noEscapeFunc),
		},
		// gad:doc
		// Unmarshal(p bytes,numericAsDecimal=false,floatsAsDecimal=false,intAsDecimal=false) -> any
		// if numericAsDecimal is true, set floatsAsDecimal to true and intAsDecimal to true
		// if floatsAsDecimal is true, parses float values as decimal
		// if intAsDecimal is true, parses int values as decimal
		// Unmarshal parses the JSON-encoded p and returns the result or error.
		"Unmarshal": &gad.Function{
			Module:   module,
			FuncName: "Unmarshal",
			Value:    funcPb2b_numberAsDecimal_b_floatAsDecimal_b_intAsDecimal_ROe(unmarshalFunc),
		},
		// gad:doc
		// Valid(p bytes) -> bool
		// Reports whether p is a valid JSON encoding.
		"Valid": &gad.Function{
			Module:   module,
			FuncName: "Valid",
			Value:    stdlib.FuncPb2RO(validFunc),
		},
	}, nil
}

func marshalFunc(vm *gad.VM, o gad.Object) (gad.Object, error) {
	b, err := Marshal(vm, o)
	if err != nil {
		return nil, gad.WrapError(err)
	}
	return gad.Bytes(b), nil
}

func marshalIndentFunc(vm *gad.VM, o gad.Object, prefix, indent string) (gad.Object, error) {
	b, err := MarshalIndent(vm, o, prefix, indent)
	if err != nil {
		return nil, gad.WrapError(err)
	}
	return gad.Bytes(b), nil
}

func indentFunc(src []byte, prefix, indent string) (gad.Object, error) {
	var buf bytes.Buffer
	err := indentBuffer(&buf, src, prefix, indent)
	if err != nil {
		return nil, gad.WrapError(err)
	}
	return gad.Bytes(buf.Bytes()), nil
}

func rawMessageFunc(b []byte) gad.Object { return &RawMessage{Value: b} }

func compactFunc(data []byte, escape bool) (gad.Object, error) {
	var buf bytes.Buffer
	err := compact(&buf, data, escape)
	if err != nil {
		return nil, gad.WrapError(err)
	}
	return gad.Bytes(buf.Bytes()), nil
}

func quoteFunc(o gad.Object) gad.Object {
	if v, ok := o.(*EncoderOptions); ok {
		v.Quote = true
		return v
	}
	return &EncoderOptions{Value: o, Quote: true, EscapeHTML: true}
}

func noQuoteFunc(o gad.Object) gad.Object {
	if v, ok := o.(*EncoderOptions); ok {
		v.Quote = false
		return v
	}
	return &EncoderOptions{Value: o, Quote: false, EscapeHTML: true}
}

func noEscapeFunc(o gad.Object) gad.Object {
	if v, ok := o.(*EncoderOptions); ok {
		v.EscapeHTML = false
		return v
	}
	return &EncoderOptions{Value: o}
}

func toDecimal(s string) (gad.Object, error) {
	return gad.DecimalFromString(gad.Str(s))
}

func toInt(s string) (o gad.Object, _ error) {
	o, _ = gad.ToInt(gad.Str(s))
	return
}

func toFloat(s string) (o gad.Object, _ error) {
	o, _ = gad.ToFloat(gad.Str(s))
	return
}

func unmarshalFunc(b []byte, numericAsDecimal, floatsAsDecimal, intAsDecimal bool) (gad.Object, error) {
	opts := NewDecodeOptions()
	if numericAsDecimal {
		floatsAsDecimal = true
		intAsDecimal = true
	}
	if intAsDecimal {
		opts.IntFunc = toDecimal
	}
	if floatsAsDecimal {
		opts.FloatFunc = toDecimal
	}
	v, err := Unmarshal(b, opts)
	if err != nil {
		return nil, gad.WrapError(err)
	}
	return v, nil
}

func validFunc(b []byte) gad.Object { return gad.Bool(valid(b)) }
